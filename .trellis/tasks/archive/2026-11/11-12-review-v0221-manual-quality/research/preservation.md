# Preservation verification

Pinned upstream v0.2.21 commit remains an ancestor of HEAD. Fresh byte comparisons against the pre-upgrade fork confirm all four identity/fingerprint/account/native-transform files unchanged:

```json
[
  {
    "file": "backend/internal/service/openai_codex_identity.go",
    "sameAsPreUpgrade": true,
    "sha256": "cb431eda0f4379a46e298d3748e69c7bbacadb62836c9f248cb4b6be05b42445"
  },
  {
    "file": "backend/internal/service/openai_codex_fingerprint.go",
    "sameAsPreUpgrade": true,
    "sha256": "fc0136e4c87da6a7c7f3a2a7ecec67c140b7d5945114a704fac421adc8902fcf"
  },
  {
    "file": "backend/internal/service/openai_codex_account_identity.go",
    "sameAsPreUpgrade": true,
    "sha256": "21d4f82742c1ce5aa226df7494dd73ea5f39cc95012624c701810eca642c52a5"
  },
  {
    "file": "backend/internal/service/openai_codex_transform.go",
    "sameAsPreUpgrade": true,
    "sha256": "0c3bf9e9c7a6d39827005650aeba6871258f93732518f8708af482efab4f7336"
  }
]
```

No published migration, dependency manifest or lockfile is changed by this repair. Existing final transport identity/branding tests are included in the backend final regression selection; detailed test results live in backend-implementation.md. Manual probe credential lifecycle changes are isolated by Pelican context; normal forwarding policies remain covered. No production environment or live account call was used.
