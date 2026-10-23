# 实施计划

## 步骤

- [x] 1. 新增 `backend/internal/service/openai_invalid_encrypted_content.go`
  - 实现 `isOpenAIInvalidEncryptedContentUpstreamError(code, msg string) bool`
  - 注释写明：为什么不能只认 `error.code`（生产形态 code=null）、与 WS 路径共用的原因
- [x] 2. 改 `backend/internal/service/openai_gateway_forward.go:1113`
  - 用共享判定替换现有两分支
  - `previous_response_id` 保持不动（HTTP 路径必须保留续链锚点）
  - 补一条日志，说明本次重试的原因与动作
- [x] 3. 改 `backend/internal/service/openai_ws_forwarder_support.go:761`
  - `classifyOpenAIWSErrorEventFromRaw` 改用共享判定，删除本地重复文案判定
- [x] 4. 新增测试 `backend/internal/service/openai_invalid_encrypted_content_test.go`
  - 判定函数表驱动用例
  - HTTP 路径 `code=null` 生产形态 → 自愈重试成功
  - `previous_response_id` 两种场景
  - 无关 400 不重试
  - WS 分类与 HTTP 判定一致
- [x] 5. 跑测试
  - `cd backend && go test ./internal/service/...`
  - 重点：`-run 'EncryptedContent|InvalidEncrypted|ThinkingSignature'`
  - `go build ./...` + `go vet ./internal/service/`
- [x] 6. 更新 spec（code-spec 深度 7 节）
  - 新增 `.trellis/spec/backend/openai-encrypted-reasoning-recovery.md`
  - 更新 `.trellis/spec/backend/index.md` 索引与 Quality Check
- [ ] 7. 收尾
  - `task.py finish` / 归档
  - 记录会话到 `.trellis/workspace/.../journal`
  - 在 vutl 仓库 `docs/Sub2API-GPT-5.6适配记录.md` 记一条「待发版」条目

## 验证命令

```bash
cd /Users/bingooyong/Code/01Code/github.com/bingooyong/sub2api/backend
go build ./...
go vet ./internal/service/
go test ./internal/service/ -run 'EncryptedContent|InvalidEncrypted|ThinkingSignature|EncryptedReasoning' -count=1
go test ./internal/service/ -count=1        # 全量
```

## 回滚点

改动集中在 3 个文件，无迁移、无数据变更。回滚 = `git revert` 本次提交后重新发版。
发版回滚边界见 vutl 仓库 `.trellis/spec/ops/sub2api-160-recovery.md`。
