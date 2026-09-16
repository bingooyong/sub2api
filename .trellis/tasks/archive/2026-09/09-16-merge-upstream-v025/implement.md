# Implementation plan

1. 建立 v0.2.5、dev、本地指纹提交的提交/文件差异清单。
2. 选择合并策略并执行上游合并或等价提交重放，保留本地工作区文件。
3. 逐个解决冲突；重点复核 Codex identity、fingerprint、gateway forwarding、transform 与审计文档。
4. 检查未跟踪文件，排除临时文件和凭据，纳入应提交项目配置/文档。
5. 运行 Go 测试、lint/vet、前端测试/typecheck 和 diff 检查。
6. 审查最终 diff 与指纹回归点，创建单个清晰提交。
