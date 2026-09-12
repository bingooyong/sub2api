package service

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestOpenAIGatewayService_Forward_PreservesNativeCallIDs(t *testing.T) {
	for _, tt := range []struct {
		name  string
		input string
	}{
		{
			name: "distinct function calls with outputs in reverse order",
			input: `[
				{"type":"function_call","id":"fc_item_first","call_id":"call_abc","name":"first","arguments":"{}"},
				{"type":"function_call","id":"fc_item_second","call_id":"fc_abc","name":"second","arguments":"{}"},
				{"type":"function_call_output","call_id":"fc_abc","output":"second output"},
				{"type":"function_call_output","call_id":"call_abc","output":"first output"},
				{"type":"item_reference","id":"fc_item_first"},
				{"type":"item_reference","id":"fc_item_second"}
			]`,
		},
		{
			name: "custom call and native item reference",
			input: `[
				{"type":"custom_tool_call","id":"ctc_item","call_id":"call_custom","name":"apply_patch","input":"patch contents"},
				{"type":"custom_tool_call_output","call_id":"call_custom","output":"done"},
				{"type":"item_reference","id":"ctc_item"}
			]`,
		},
		{
			name: "tool search call and native item reference",
			input: `[
				{"type":"tool_search_call","id":"tsc_item","call_id":"call_search","arguments":{}},
				{"type":"tool_search_output","call_id":"call_search","tools":[]},
				{"type":"item_reference","id":"tsc_item"}
			]`,
		},
		{
			name: "isolated reference from an earlier turn",
			input: `[
				{"type":"item_reference","id":"call_previous_turn"},
				{"type":"item_reference","id":"fc_remote_item"}
			]`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			body := []byte(`{"model":"gpt-5.5","stream":false,"instructions":"Keep the caller instructions.\n\n","input":` + tt.input + `}`)
			sent := captureNativeCallIDForward(t, body)

			require.JSONEq(t, tt.input, gjson.GetBytes(sent, "input").Raw)
			require.Equal(t, "Keep the caller instructions.\n\n", gjson.GetBytes(sent, "instructions").String())
		})
	}
}

func TestOpenAIGatewayService_Forward_PreservesCallIDBoundaryAndPairing(t *testing.T) {
	for _, tt := range []struct {
		name   string
		callID string
	}{
		{name: "64 bytes", callID: "opaque_" + strings.Repeat("a", 57)},
		{name: "65 bytes", callID: "opaque_" + strings.Repeat("a", 58)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			body := []byte(fmt.Sprintf(`{
				"model":"gpt-5.5","stream":false,"instructions":"Use the tool output.",
				"input":[
					{"type":"function_call","id":"fc_item","call_id":%q,"name":"shell","arguments":"{}"},
					{"type":"function_call_output","call_id":%q,"output":"done"},
					{"type":"item_reference","id":"fc_item"}
				]
			}`, tt.callID, tt.callID))
			sent := captureNativeCallIDForward(t, body)
			callID := gjson.GetBytes(sent, "input.0.call_id").String()

			if len(tt.callID) == 64 {
				require.Equal(t, tt.callID, callID)
			} else {
				require.NotEqual(t, tt.callID, callID)
				require.Len(t, callID, 64)
			}
			require.Equal(t, callID, gjson.GetBytes(sent, "input.1.call_id").String())
			require.Equal(t, "fc_item", gjson.GetBytes(sent, "input.0.id").String())
			require.Equal(t, "fc_item", gjson.GetBytes(sent, "input.2.id").String())

			replayed := captureNativeCallIDForward(t, body)
			require.JSONEq(t, gjson.GetBytes(sent, "input").Raw, gjson.GetBytes(replayed, "input").Raw)
		})
	}
}

func captureNativeCallIDForward(t *testing.T, body []byte) []byte {
	t.Helper()
	upstream := &httpUpstreamRecorder{resp: newOpenAIRejectedFieldTestResponse(
		http.StatusOK,
		`{"id":"resp_call_ids","object":"response","status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"ok"}]}],"usage":{"input_tokens":1,"output_tokens":1}}`,
	)}
	svc := newOpenAIRejectedFieldTestService(upstream)
	c := newOpenAIRejectedFieldTestContext(body)
	c.Request.Header.Set("User-Agent", "codex_cli_rs/0.144.1")
	SetOpenAIClientTransport(c, OpenAIClientTransportHTTP)

	result, err := svc.Forward(context.Background(), c, newOpenAIOAuthNamespaceTestAccount(), body)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, upstream.bodies, 1)
	return upstream.bodies[0]
}
