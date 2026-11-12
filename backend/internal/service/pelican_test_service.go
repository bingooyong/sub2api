package service

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
)

type pelicanTestContextKey struct{}

type pelicanTestOptions struct {
	prompt          string
	reasoningEffort string
	manual          bool
}

func withPelicanTestOptions(ctx context.Context, options pelicanTestOptions) context.Context {
	return context.WithValue(ctx, pelicanTestContextKey{}, options)
}

func pelicanTestOptionsFromContext(ctx context.Context) (pelicanTestOptions, bool) {
	options, ok := ctx.Value(pelicanTestContextKey{}).(pelicanTestOptions)
	return options, ok
}

// TestPelicanAccountConnection is the dedicated account test path for the
// Pelican UI. The existing /test endpoint deliberately keeps its historical
// probe payload; only this endpoint opts into the user prompt and reasoning.
func (s *AccountTestService) TestPelicanAccountConnection(c *gin.Context, accountID int64, modelID, prompt, reasoningEffort string) error {
	return s.testPelicanAccountConnection(c, accountID, modelID, prompt, reasoningEffort, false)
}

// TestManualPelicanAccountConnection limits ad hoc comparison to verified text
// transports. Scheduled Pelican plans retain their existing provider routing.
func (s *AccountTestService) TestManualPelicanAccountConnection(c *gin.Context, accountID int64, modelID, prompt, reasoningEffort string) error {
	if err := ValidatePelicanTestInput(modelID, prompt, reasoningEffort); err != nil {
		return s.sendErrorAndEnd(c, err.Error())
	}
	return s.testPelicanAccountConnection(c, accountID, modelID, prompt, reasoningEffort, true)
}

// ValidatePelicanTestInput also runs at the HTTP boundary, before streaming.
func ValidatePelicanTestInput(modelID, prompt, reasoningEffort string) error {
	if model := strings.TrimSpace(modelID); model == "" || utf8.RuneCountInString(model) > 100 || strings.ContainsAny(model, "\r\n\x00") {
		return errors.New("model_id must contain 1–100 characters")
	}
	if text := strings.TrimSpace(prompt); text == "" || utf8.RuneCountInString(text) > 32000 {
		return errors.New("prompt must contain 1–32000 characters")
	}
	if strings.TrimSpace(reasoningEffort) != "" && normalizePelicanReasoningEffort(reasoningEffort) == "" {
		return errors.New("reasoning_effort must be minimal, low, medium, high or xhigh")
	}
	return nil
}

func (s *AccountTestService) testPelicanAccountConnection(c *gin.Context, accountID int64, modelID, prompt, reasoningEffort string, manual bool) error {
	options := pelicanTestOptions{
		prompt:          strings.TrimSpace(prompt),
		reasoningEffort: normalizePelicanReasoningEffort(reasoningEffort),
		manual:          manual,
	}
	ctx := withPelicanTestOptions(c.Request.Context(), options)
	c.Request = c.Request.WithContext(ctx)
	return s.TestAccountConnection(c, accountID, strings.TrimSpace(modelID), options.prompt, AccountTestModeDefault)
}

func isPelicanProbe(ctx context.Context) bool {
	_, ok := pelicanTestOptionsFromContext(ctx)
	return ok
}

func isPelicanTextModel(model string) bool {
	model = strings.ToLower(codexProviderQualifiedModelID(model))
	// Inspect mapped provider model families, not arbitrary substrings in a
	// user-defined model name (for example, an image-analysis text model).
	for _, prefix := range []string{"gpt-image-", "dall-e-", "gpt-audio", "gpt-4o-audio", "gpt-4o-mini-audio", "gpt-realtime", "gpt-4o-realtime", "gpt-4o-mini-realtime", "gpt-4o-transcribe", "gpt-4o-mini-transcribe", "gpt-4o-mini-tts", "text-embedding-", "omni-moderation-", "text-moderation-", "tts-", "whisper-", "sora-"} {
		if strings.HasPrefix(model, prefix) {
			return false
		}
	}
	return true
}

func normalizePelicanReasoningEffort(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "minimal", "low", "medium", "high", "xhigh":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return ""
	}
}

func createPelicanClaudePayload(modelID, prompt string) (map[string]any, error) {
	payload, err := createTestPayload(modelID)
	if err != nil {
		return nil, err
	}
	messages, ok := payload["messages"].([]map[string]any)
	if !ok || len(messages) == 0 {
		return payload, nil
	}
	content, ok := messages[0]["content"].([]map[string]any)
	if ok && len(content) > 0 {
		content[0]["text"] = promptOrDefault(prompt)
	}
	return payload, nil
}

func createPelicanOpenAIPayload(modelID string, isOAuth bool, prompt, reasoningEffort string) map[string]any {
	payload := createOpenAITestPayload(modelID, isOAuth)
	input, ok := payload["input"].([]map[string]any)
	if ok && len(input) > 0 {
		content, ok := input[0]["content"].([]map[string]any)
		if ok && len(content) > 0 {
			content[0]["text"] = promptOrDefault(prompt)
		}
	}
	if effort := normalizePelicanReasoningEffort(reasoningEffort); effort != "" {
		payload["reasoning"] = map[string]any{"effort": effort}
	}
	return payload
}

func promptOrDefault(prompt string) string {
	if value := strings.TrimSpace(prompt); value != "" {
		return value
	}
	return "hi"
}
