# 通用动作结果、观察与展示绑定

实现契约见 [execution-results.md](execution-results.md)。结果查询由执行适配器实现，Argo 核心不包含业务查询 SDK。

## 资源关系

- `ActionDefinition` 定义动作输入输出及原生执行实现，支持默认 `results`。
- `ActionResultTemplate` 独立定义结果来源、JSON/text 格式、字段投影、Schema 和视图。
- `WorkflowActionBinding.calls[].resultBindings` 在具体 `callPath` 上选择结果模板，把动作的实际输出映射到结果来源槽位。同一个动作的不同调用可选择不同模板。
- 提交和重试将解析后的定义、版本和映射冻结在 `WorkflowSubmission`、`ParameterRetry` 中；运行时按 Workflow UID、节点、尝试及实际资源名称／UID 读取。
- 新参数局部重试保留的成功节点继续使用原结果契约。刷新详情不重跑动作，不修改 Workflow 状态。
- 执行前已分配的结果对象可通过 `nameInput`、`uidInput` 绑定本次原生输入，从而在 Running 或 Failed 状态读取进度；`inputMappings` 支持复用结果模板的输入槽位。输入与输出身份只能选择一组完整且不同名的参数，不从当前草稿重新解析历史身份。

## 执行侧分工

获取方法和输入由原生动作实现编排。Pod 可以调用适配程序并输出参数或 artifact；Resource 模板可以创建查询 CR，由独立控制器访问外部服务并维护公开 status。Argo 只负责原生编排、绑定校验和读取展示。实际认证、外部协议、分页和业务字段转换留在适配器。

详情展示区分执行快照与最新观察。后者展示 `observedAt`、过期和错误信息，不能覆盖原执行结果。资源来源校验命名空间、Workflow 所属关系及实际 UID；同名替换不能串用。输出参数按冻结快照读取；大报告沿用原生 artifact 下载能力。

## 本次验收清单

- [x] 独立结果 CR、每个动作调用的动态绑定、字段映射及静态校验。
- [x] 定义版本冻结、Diff、声明式视图和只读结果 API。
- [x] 同一动作不同调用隔离、保留成功节点的历史结果契约相关 UT。
- [x] 独立本地 k3s 完整路径和浏览器展示；schema、动作调用、原参数重试、定义发布、执行 Diff 共六项相关 E2E 全部通过。
- [x] 本地测试入口和操作说明。
- 社区目标为 main 与 release-4.0；贡献分支已加入相同契约。提交与 CI 状态以对应 PR 为准，不能将本地验收等同于上游合入。

边界：本次结果来源为原生 output 参数及受控的 CR status／ConfigMap data。不会由详情刷新启动新 Pod；独立控制器负责刷新频率。归档后已删除 Workflow 的通用结果面板、任意远端 artifact 解析不在本增量内；原生历史与 artifact 能力继续保留。尚未完成的验收不得描述为已完成或已部署。

## 本地人工验证

1. 在本地测试环境打开增强版 Argo 的 Workflows 页面，选择 `argo-parameterization-lab`。
2. 打开 `result_bindings_e2e.py` 输出的 Workflow 名称，选择 `east` 下的实际 `invoke` 节点，在 Summary 下方查看 Action results：标题为 New summary，Target 为 fixed。
3. 选择 `west` 下的 invoke：同一个动作保留 Original tree 的结果契约，未被 east 的新参数重试改写。
4. 选择 `observe` 下的 invoke：树形结果包含 dynamic-target-child，并注明 Current observation、观察时间和过期状态。
5. 点击 Refresh results，只回读结果；Workflow UID、节点尝试和审批不变。该按钮不会提交查询动作，也不强制适配器立即轮询。
6. 测试结束会停止示例观察控制器以节省资源，所以最新观察随后会显示过期。需要继续验证动态观察时，按下述命令仅启动本次示例控制器。

命令必须指定独立本地集群的 kubeconfig，不能使用生产或预发布 context：

```bash
kubectl --kubeconfig "$LOCAL_KUBECONFIG" --context k3d-argo-parameterization   -n argo-parameterization-lab scale deployment/<results-示例名称> --replicas=1
```

示例名称可从本次 Workflow 关联的 `WorkflowSubmission.spec.binding` 回读；它与测试创建的 Deployment 同名。
完整复跑命令、环境变量及约束沿用 [本地测试文档](local-testing.zh.md)，新增用例为 `test/parameterization/result_bindings_e2e.py`。
此示例访问合成 provider，不使用业务凭据或真实服务。
