# Reproducible upstream OpenAI diff

The research agent read the exact upstream v0.2.19→v0.2.21 diff for the seven
paths below. Both immutable revisions remain in this repository. The raw
research copy is retained in ignored `output/trellis/upstream-openai-diff.patch`;
reproduce it without storing a large nested patch in the task archive.

```bash
git diff dece26153621959ccd7635814f4b2d08304e681a 10ba8f9b4dc2e6023d4a9879ab77b21841babd10 -- \
  backend/internal/service/account_test_service.go \
  backend/internal/service/openai_codex_metadata_events_test.go \
  backend/internal/service/openai_codex_model_metadata.go \
  backend/internal/service/openai_codex_models_service.go \
  backend/internal/service/openai_gateway_forward.go \
  backend/internal/service/openai_gateway_request_body.go \
  backend/internal/service/openai_ws_pool.go
```

SHA-256 of the exact output: `17938f6454443dace72ed384e39aca3640b62ed8462e9ed34949ec86edadbed6`.

The reproduced bytes were compared with the original research input before
replacing the archived dump with this recipe.
