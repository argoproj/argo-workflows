# 结构化参数使用指南

结构化参数适合需要嵌套配置、共享表单、版本比较和修改参数后局部重试的 WorkflowTemplate。
参数扩展负责校验输入并生成原生 Workflow；Argo 原生控制器继续调度任务和维护执行状态。
此功能默认关闭，`parameters.workflows.example.io/v1alpha1` 仍是实验 API。

## 安装与入口

按 [英文安装说明](getting-started.md#install-an-isolated-example) 在独立的本地 k3d/k3s 集群安装。
使用专用 kubeconfig、上下文 `k3d-argo-parameterization` 和命名空间 `argo-parameterization-lab`。
示例只使用合成数据与本地资源。
不要用它覆盖已有 Argo 的 CRD 或部署。

端口转发后，打开 `http://127.0.0.1:2747/parameters/argo-parameterization-lab`。
原生 WorkflowTemplate 的详情和提交入口也会列出可用的参数绑定。
仅查看定义不会提交 Workflow。

## 参数、表单和执行

1. 选择 `dynamic-delivery` 绑定，填写新的参数集名称。
2. 切换到 YAML，粘贴 `examples/parameterization/topology-values.yaml`。
3. 切回表单，展开 zones、subnets、roles，添加或删除数组项。
   表单和 YAML 编辑的是同一份参数。
4. 保存一版，修改值后再保存一版，比较两个版本。
   恢复旧版本会先把内容带回编辑器，再显式保存为新版本。
5. 校验并预览编译后的原生 Workflow，然后提交。
6. 在 Workflow 页面恢复等待中的审批节点，查看生成的 ConfigMap 和任务输出。
7. 从执行记录打开参数快照，核对当时使用的值、表单、绑定和动作定义。

`training` 使用另一份参数模型和容器动作。
`intermediate-demo` 可以用空参数提交，在执行中的原生 suspend 节点填写表单，再由下游任务读取输出。

## 查看资源和发布定义

进入 **Structured parameters → Resource definitions and records** 查看、创建和编辑定义。
`ParameterTemplate` 定义参数结构，`WorkflowForm` 定义表单联动，`WorkflowParameterBinding` 将它们映射到原生模板。
`ActionDefinition` 定义动作实现及输入输出，`WorkflowActionBinding` 将动作和结果契约绑定到具体调用路径。
`ActionResultTemplate` 定义结果来源和展示格式。
完整关系见 [资源与引用](contracts.md#resources-and-references)。

**Publish named version** 将单个定义发布为不可变版本。
**View published versions** 可以浏览和比较版本。
**Clone as new definition** 创建可编辑副本；随后需要在绑定中显式选择新名称。
发布定义不提交 Workflow。

## 原参数和新参数重试

将 `fail` 设为 `true` 后提交，示例会在资源任务完成后故意失败。
在示例命名空间创建名为 `<workflow-name>-recovery`、包含 `data.allow: "true"` 的 ConfigMap，再使用原参数重试。
Workflow UID、已保留的成功结果和审批不变。

新参数重试需要保存新版本，选择 **Preview partial retry** 查看受影响的调用，再显式应用。
无关的成功节点和审批应保留，受影响的调用使用新输入。
保存草稿和查看 Diff 都不会提交重试。
选择新建提交则会创建另一条 Workflow。

模板作者需要声明可安全重试和可复用结果的契约。
无法验证的拓扑变化、存储形式或执行形态会在修改 Workflow 前报错。
契约声明不能让本来不具备幂等性的外部操作自动具备幂等性。

## 文件、机密和结果

`artifact-mappings.yaml` 演示原生 artifact 映射；`artifact-retry.yaml` 演示校验文件内容后的局部重试。
`secret-demo`、`secret-array-demo` 和 `secret-input-demo` 分别演示机密引用、数组和执行中输入。
机密示例要求 Kubernetes 加密存储及不记录 Secret 正文的审计策略。
使用合成测试值，参数历史中只保留引用。

外部 Vault 和动态数据库凭据示例需要单独安装本地测试后端。
具体步骤、有效期和清理边界见 [英文指南](getting-started.md#sensitive-values)。
不能把示例后端用于共享或生产环境。

[动作结果说明](execution-results.md) 展示输出快照和由执行适配器维护的实时观察。
刷新只读取状态，不重新执行动作。
不同调用可复用同一动作并绑定不同的结果视图。
完整路径测试及其核对点见 [测试清单](testing.md)。
