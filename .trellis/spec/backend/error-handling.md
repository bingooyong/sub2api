# Error Handling

> How errors are handled in this project.

---

## Overview

<!--
Document your project's error handling conventions here.

Questions to answer:
- What error types do you define?
- How are errors propagated?
- How are errors logged?
- How are errors returned to clients?
-->

(To be filled by the team)

---

## Error Types

<!-- Custom error classes/types -->

(To be filled by the team)

---

## Error Handling Patterns

<!-- Try-catch patterns, error propagation -->

Upstream (OpenAI-compatible) error classification, retry budgets, recovery
actions and ops reporting are specified in
[OpenAI Upstream Error Handling](./openai-upstream-error-handling.md).

Key points that generalize beyond the OpenAI path:

- Upstream error bodies are **multi-signal**: the same failure may appear in
  `error.code`, `error.type`, `error.param`, or only in free-form text. Match
  on more than one signal.
- A 4xx that is deterministic for a given request body must not be retried
  unchanged; recovery has to change the request.
- Every recovery branch is budgeted (one attempt). A second failure is
  terminal.
- Ops error events are **one row per upstream attempt**, so the row count is
  the evidence for whether a recovery fired.

(To be filled by the team)

---

## API Error Responses

<!-- Standard error response format -->

(To be filled by the team)

---

## Common Mistakes

<!-- Error handling mistakes your team has made -->

- **Classifying on a single field.** Matching only `error.code` misses every
  error shape where the code is `null` and the meaning lives in the message.
- **Retrying a deterministic 4xx unchanged.** A different account does not
  make an undecryptable ciphertext decryptable.
- **Reading the error code from an ops column that is not the classifier's
  output.** See the note in
  [OpenAI Upstream Error Handling](./openai-upstream-error-handling.md).

(To be filled by the team)
