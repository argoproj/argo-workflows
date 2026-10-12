# 本地参数化能力试用

<!-- cspell:words labkubectl codegen -->

> 已部署的兼容性构建基于官方 v4.0.1；社区分支分别对齐 main 和 release-4.0。增强版仅部署独立本地 k3s，不连接生产或预发布。

此版本是基于 Argo Workflows v4.0.1 的实验实现。部署目标仅为独立的
`k3d-argo-parameterization`，namespace 为 `argo-parameterization-lab`。
所有演示使用独立本地集群和合成数据。

入口：<http://127.0.0.1:2747/parameters/argo-parameterization-lab>。
需要保持本机指向该实验集群的 2747 端口转发运行。

可按文末章节测试 artifact 映射与内容核验、Vault 版本与动态数据库凭据、循环拓扑和局部重试。
以下三个旧绑定也继续保留，分别创建新的 Parameter set。

| 绑定 | 主要观察点 |
| --- | --- |
| `action-calls-demo` | 同一共享模板的两次调用只替换 east；新参数重试保留 west 与原审批 |
| `secret-versions-demo` | 索引从 v1 切到 v2 后，原参数重试保持 v1，明确参数重试才采用 v2 |
| `steps-retry-demo` | Steps 与嵌套调用局部恢复，Workflow UID、无关成功 Pod 和审批不变 |

所有演示只使用本地合成数据。

## 建议的手工验收顺序

这一节对应当前支持范围，后面的 v12–v47 章节保留实现过程和补充操作。
试用前先确认页面左侧版本包含 `parameterization`，namespace 为 `argo-parameterization-lab`。
若本机已配置 Traefik，可使用 `https://argoworkflows.local/parameters/argo-parameterization-lab`；未配置时沿用上方的本机端口转发入口。
每个场景使用不同的 Parameter set 名称，例如 `manual-actions-20261014-a`，再次试验时更换后缀。
不要加载或修改其他人的参数集、等待审批的 Workflow 或演示定义。
绑定下拉列表只显示有上限的首批条目；可在参数页面 URL 后加 `?binding=<下方绑定名>` 直接打开已安装的演示。

### 1. 查看参数、表单、动作与绑定的 CR

1. 打开 **Resource definitions and records**。
2. 依次选择 `ParameterTemplate`、`WorkflowForm`、`WorkflowParameterBinding`、`ActionDefinition`、`ActionResultTemplate`、`WorkflowActionBinding`。
3. 查看 `topology-v1`、`topology-form-v1`、`dynamic-delivery` 及 `action-calls-demo` 对应的定义。
4. 展开 CR YAML，核对类型字段、表单布局、参数映射与 `callPath`，确认它们是独立 CR 引用。
5. 需要试验定义编辑时，先 **Clone as new definition** 成自己的名称，再编辑和发布，避免改动公共演示。
6. 对自己的定义执行 **Publish named version**；从 **View published versions** 比较，再克隆旧版本为新的定义。

预期：查看、编辑草稿、发布和比较定义都不会创建 Workflow；已发布版本不能原地改写。

### 2. 动态表单、版本与 Diff

1. 选择 `dynamic-delivery`，Parameter set 填 `manual-topology-20261014-a`。
2. 点击 **YAML**，填入下方数据，再切回 **Form**。
3. 展开 zones/subnets/roles，确认两层数组都能编辑；点击 **Save revision** 保存版本 A。
4. 增加一个子网，或移动数组项，再次 **Save revision** 保存版本 B。
5. 在 **Revision history** 中给 A 点 **Compare from**，给 B 点 **Compare to**，然后点 **Compare revisions**。
6. 给旧版本点 **Restore into editor**，再保存，得到新的版本 C；A/B 仍保留。
7. **Validate and preview**，核对输入、定义版本、生成的 Workflow；然后 **Submit this revision**。
8. 点击执行链接，在图中选择 `approve`，使用原生 **Resume** 继续。

```yaml
region: demo-north
environment: test
zones:
- name: zone-a
  subnets:
  - id: one
    cidr: 10.10.1.0/24
  - id: two
    cidr: 10.10.2.0/24
- name: zone-b
  subnets:
  - id: one
    cidr: 10.20.1.0/24
roles: [admin, dns]
fail: false
```

预期：每个子网包含 2 个角色，原始输入的 3 个子网共生成 6 份本地 `ConfigMap`；数量来自参数。
保存和预览不执行；审批后才开始创建演示资源。
该示例不创建实际云子网或云集群。

### 3. 动作调用映射与新参数局部重试

1. 选择 `action-calls-demo`，新建 `manual-actions-20261014-a`。
2. 保存 `message: invalid`，预览、提交、恢复 `approve`。
3. 等待结束：`west/emit` 成功，`east/emit` 故意失败。
4. 在 Workflow 的 YAML 中记下 `metadata.name` 和 `metadata.uid`，并在节点详情记下 west 的 Pod、完成时间以及审批完成时间。
5. 展开 **Parameter revision and execution inputs**，点击 **Edit a parameter version**。
6. 仅改成 `message: fixed`，点击 **Save revision**。
7. 点击 **Preview partial retry**，核对受影响的 east 和下游 report，以及保留的 west、approve。
8. 点击 **Apply revision and retry this Workflow**，不要点 **Submit as a new Workflow**。
9. 回到执行详情，核对 UID、west 的 Pod 和审批记录。

预期：Workflow UID 不变，原审批与 west 成功结果保留；east 输出 `action:fixed`，report 输出 `base:action:fixed`。
若输入校验或影响预览失败，界面应展示原因，而不是暗中新建整条 Workflow。
同类 Steps 场景使用 `steps-retry-demo`，初始 `value: invalid`、`items: [one, two]`，之后只把 value 改成 `valid`。

### 4. 循环拓扑变化

1. 选择 `topology-retry-demo`，保存 `items: [a, b]`，预览、提交并恢复 `approve`。
2. 成功后从执行记录返回参数编辑，把 items 改成 `[b, c, d]`。
3. 保存 → **Preview partial retry** → **Apply revision and retry this Workflow**。
4. 再分别使用 `items: []` 和 `items: [restored]`，重复上述过程。

预期：同一 Workflow 中，仅作者允许的 `expand` 整个循环及 report 重新执行，approve 和 stable 保留。
这不是任意修改 DAG 的能力，也不复用变化循环中旧下标对应的结果，不会自动删除先前的外部资源。

### 5. Artifact 映射、原参数重试与结果复用

先验证新参数重试：

1. 选择 `artifact-retry-demo`，保存 `mode: bad`，预览、提交并恢复 approve。
2. producer 成功，consume 故意失败；记下 producer 的 Pod 和 checksum 输出。
3. 从执行记录返回参数编辑，改为 `mode: good`，保存并预览局部重试。
4. 应用到同一 Workflow。

预期：producer 不重复执行，consume 的 digest 与 producer 的 checksum 一致，审批和 Workflow UID 保留。

再验证原参数重试：

1. 选择 `artifact-mapping-demo`，保存 `message: artifact-content`，提交并恢复 approve。
2. transform 成功后，gate 因缺少本次专属的 ConfigMap 故意失败。
3. 用本节末尾的 `labkubectl` 创建 `<Workflow 名>-ready` ConfigMap，数据为 `ready=true`。
4. 在原生 Workflow 详情选择 **Retry**，不修改参数。

```sh
labkubectl create configmap '<Workflow 名>-ready' --from-literal=ready=true
```

预期：同一 Workflow 继续，transform 的原结果被保留，consume 下载原文件并输出摘要。
只创建自己本次 Workflow 的 ready 标记。
演示文件仓库是临时存储，删除它的 Pod 会丢文件；不在这次手工验收中重启公共文件仓库。

### 6. 执行中表单和机密输入

1. 选择 `intermediate-demo`，新建参数集，保存 `{}` 并提交。
2. 等待原生输入节点，在 Workflow 详情里的参数输入区域填写 release、replicas，再提交。
3. 查看节点继续执行，以及提交人、时间和已冻结输入的记录。
4. 选择 `secret-input-demo`，用另一份参数集保存 `{}` 并提交。
5. 在执行中的授权表单里填写合成值 `synthetic-manual-token-20261014`，点击 **Store new encrypted secret version**，再提交表单。
6. 查看参数修订、Workflow 和输入回执中的引用；它们应保存 Secret 身份，而不是刚输入的文本。

数组机密使用 `secret-array-demo`；给每行稳定的 id，并分别存储新的合成机密版本。
仅使用测试字符串，不输入生产凭据。
加密输入要求端到端受验证的 TLS，或显式开启的本机 loopback 开发入口；任意 `X-Forwarded-Proto` 不能代替 TLS。

### 7. 最新机密版本与外部 Vault

1. 选择 `secret-versions-demo`，保存下方索引引用，预览并提交，恢复 approve。
2. 初始演示 v1 会令 consume 故意失败；保存预览中的实际 Secret 版本身份。
3. 使用本地 kubectl 将 `secret-versions-demo` ConfigMap 的 current 指向 `secret-versions-demo-v2` 的 token 键。
4. 原参数 **Retry** 仍使用冻结的 v1；它可能再次出现同样的演示失败，这是预期。
5. 从执行记录编辑参数，重新保存并 **Preview partial retry**，核对新的固定机密版本，再应用重试。

```yaml
credential:
  latestSecretRef:
    configMap: secret-versions-demo
    key: current
```

```sh
labkubectl patch configmap secret-versions-demo --type merge \
  -p '{"data":{"current":"{\"name\":\"secret-versions-demo-v2\",\"key\":\"token\"}"}}'
```

公共演示索引可能已被上一位测试者切到 v2；开始前只读检查它，测试结束后恢复开始时的值。
外部后端用 `external-secret-demo`，把 `configMap` 换成 `external-vault-demo`，操作见下文“外部机密”。
动态租约用 `dynamic-credential-demo`，保存 `credential.secretRef: {name: dynamic-credential-demo, key: token}`，审批后约 36 秒完成数据库读取，输出仅包含不透明的租约编号。
该演示 bootstrap token 有效期 24 小时；过期后按英文文档明确重新安装本地演示，不自动切换身份。

### 8. 动作回执与恢复

1. 选择 `operation-demo-report`，新建参数集，保存 `mode: normal`、`delay: "2"`。
2. 预览并提交，查看原生 Resource 节点及对应的 ResourceOperation。
3. 在 **Resource definitions and records → ResourceOperation** 展开本次记录，核对 Workflow UID、操作身份、状态和输出。
4. 等待下游收到 reportURI；记录与 Workflow 的执行信息应能相互关联。

故障恢复的可复现入口为 `test/parameterization/operation_e2e.py`。
它为本次测试创建独立的 adapter 和合成服务，覆盖响应丢失、控制器恢复、同 Workflow 重试、取消、detach 与未知结果不重复提交，并用服务端计数验证没有重复创建。
不要为手工测试重启公共的 operation-demo-controller，避免影响其他等待中的演示。

### 本地核对命令

把 KUBECONFIG 指向实验集群的独立 kubeconfig 文件，不能使用生产或预发布 kubeconfig。
以下函数固定 context 和 namespace，不修改 kubectl 默认 context。

```sh
export KUBECONFIG=/absolute/path/to/local-k3d.kubeconfig
labkubectl() {
  kubectl --kubeconfig "$KUBECONFIG" --context k3d-argo-parameterization \
    -n argo-parameterization-lab "$@"
}
labkubectl get workflows
labkubectl get workflow '<本次 Workflow 名>' -o jsonpath='{.metadata.uid}{"\n"}'
labkubectl get pods -l 'workflows.argoproj.io/workflow=<本次 Workflow 名>'
labkubectl get parameterrevisions,parameterretries,resourceoperations
```

验收记录至少包含：参数集名称、原/新修订、Workflow 名称及 UID、受影响节点、保留的 Pod/审批、预期与实际结果。
未支持的范围以本页“当前验收边界”和 `implementation-decisions.md` 为准；本地试用通过不等于社区已经接受或合并提案。

## 查看定义和动态绑定

打开 **Resource definitions and records**，选择资源类型：

| 资源 | 当前用途 |
| --- | --- |
| ParameterTemplate | 类型、默认值、数组身份、敏感字段契约 |
| WorkflowForm | 字段展示和动态选项来源 |
| WorkflowParameterBinding | WorkflowTemplate 与参数模型、表单的绑定 |
| ActionDefinition | 原生动作实现及声明的输入输出契约 |
| WorkflowActionBinding | 模板调用位置与动作定义及结果模板的绑定 |
| `ActionResultTemplate` | 独立结果来源、格式、字段投影和视图 |
| ParameterSet / ParameterRevision | 草稿及不可变参数版本 |
| ResourceOperation | 单次外部操作的持久回执；可选控制器负责提交、回读与恢复 |
| WorkflowSubmission / ParameterRetry / ParameterNodeInput | 提交、原生重试及执行中输入记录 |

列表每页最多 50 条，仅加载元数据，点击资源名称才读取实际 CR 的完整 YAML，可折叠。定义可通过页面或 `examples/parameterization/demo.yaml` 一类 Kubernetes 清单管理；执行回执只读。

原生 WorkflowTemplate 详情和 Submit 窗口也会显示对应的结构化绑定。
点击绑定名称进入该表单；存在多个绑定时逐项选择。打开表单不会启动执行。

## 发布、比较与恢复定义版本

1. 在 **Resource definitions and records** 中打开一个已保存的定义，选择 **Publish named version**。
2. 填写新版本名称和说明，点击 **Confirm publish definition**；发布会创建同类型的新 CR，不修改源定义，也不会提交 Workflow。
3. **View published versions** 按定义家族分页加载版本；点击 **Compare … with selected definition** 查看字段差异。
4. 打开旧版本后选择 **Clone as new definition**，保存成新的可编辑定义，再显式把绑定指向它。
5. 返回参数表单，保存参数版本、预览并提交，原生 Argo 才会执行。

已发布定义的 spec 和版本来源由 API Server 的准入策略保护，不能通过普通编辑或直接 Kubernetes PATCH 覆盖。
源草稿删除后，发布版本仍可打开、比较和克隆；重复提交同一个发布名称和内容会返回原对象。
一个发布版本固定的是该定义自身；需要固定多种定义时应逐项发布并引用具名版本。
实际执行仍会独立冻结完整定义及原生模板引用闭包，不把一次定义发布冒充为整套依赖的原子发布。
运行 `test/parameterization/definition_publications_e2e.py` 验证六类定义发布、直接改写拦截、源删除后的历史、浏览器恢复与绑定、原生 Resource 输出。

## 具体 CR API 结构

Go API 定义位于 `pkg/apis/parameterization/v1alpha1/`：`resources.go` 定义
Kind、List、metadata、spec、status；`types.go` 定义每类资源的具体字段；
`register.go` 注册 Scheme；DeepCopy、clientset、informer、lister 由原生 `make codegen` 生成。
结构化 CRD 由同版本 controller-gen 生成，`hack/parameterization-codegen.sh` 单独重建实验 CRD 清单。

`manifests/parameterization/crds/` 包含每类独立 CRD，`crds.yaml` 是合并安装文件。
定义外层的字段、必填项和枚举有明确结构；仅业务参数 JSON、JSON Schema 和
嵌入的原生 Argo 实现保留开放结构。后者继续接受原生 Argo 校验。

实验安装要求 Kubernetes 1.30 或以上，并安装 `immutable-snapshots.yaml`。
它通过原生 ValidatingAdmissionPolicy 比较实际快照 JSON，防止任意嵌套参数被改写，
同时允许元数据更新。CRD 的 CEL 无法直接访问未知类型的业务 JSON，因此不能只依赖
一个 CRD `self == oldSelf` 规则。缺少该策略时安装不完整。

可在独立实验集群查看：

```sh
kubectl --context k3d-argo-parameterization explain workflowparameterbinding.spec --recursive
kubectl --context k3d-argo-parameterization explain workflowform.spec --recursive
kubectl --context k3d-argo-parameterization explain resourceoperation.spec --recursive
```

ResourceOperation 的可选参考控制器已安装在实验 namespace，用合成 HTTP 服务演示提交、回读、恢复和取消。
它校验原生 Resource Pod 和冻结动作身份，不调度其他节点。status/conditions 已通过子资源隔离；
五类定义的静态校验状态由可选控制器协调，具体边界见 v34 说明。
声明式布局与字段交互已接入初始化和运行中输入页面，支持范围见下文；这不代表整个设计文档已完成。

## 动态拓扑、版本和执行

1. 选择 `dynamic-delivery`，填写一个新的 Parameter set 名称。
2. 切换 YAML，粘贴 `examples/parameterization/topology-values.yaml`。
3. 切回 Form，展开可用区、子网和角色。修改数量或顺序后保存。
4. 修改值再次保存，选择历史版本比较。历史按 25 条摘要分页，点击恢复后才加载完整参数。恢复历史版本后再次保存会生成新版本，
   不覆盖历史。重复保存相同内容不会追加版本。
5. 执行预览，核对来源、绑定版本和生成的原生 Workflow，然后明确提交。
6. 打开 Workflow，继续其原生审批暂停节点。示例会按可用区、子网和角色的组合
   创建本地 ConfigMap；默认数据为六个资源，不创建任何真实云资源。

普通保存只更新草稿，不更新正在执行的快照。明确选择“Submit as a new Workflow”才会新建 Workflow。
支持范围内的新参数可以先分析影响，再局部重试同一 Workflow，操作步骤见下文。

## 其他通用场景

- `training`：另一套类型模型与普通容器执行，验证能力可用于不同业务领域。
- `intermediate-demo`：初始空参数提交后，在 Workflow 详情中填写 `release`、
  `replicas`，提交到原生 suspend 节点。完成后可查看提交人、时间和值。
- `secret-demo`：仅使用合成测试值。通过加密输入控件保存为 Secret 引用，
  历史和 Workflow 参数不展示明文。实验 k3s 已开启 Secretbox 静态加密。

原参数重试使用原生 Argo retry 并保留 Workflow UID。完整自动用例会模拟依赖修复，
确认七个已成功的具体执行及审批节点保留。原生 retry 也可能恢复其他失败节点，
不承诺任意单节点与整条工作流无关的重试语义。

## 当前验收边界

UT、前端测试和真实本地 k3s 完整路径覆盖参数保存、版本、差异、动态资源执行、
原参数重试、机密引用、执行中输入。构建校验同时检查实际页面资源与本地构建一致。

当前已支持 DAG、Steps、静态嵌套调用与固定身份复合循环的新参数局部重试，以及具名动作调用路径和机密最新版本策略。
本批另外支持 artifact 接口映射、Vault KV-v2 后端，以及作者明确允许的整个循环展开替换。
支持经过内容核验的 HTTP artifact 复用、原生叶任务 retryStrategy，以及配置明确的 Vault Agent 动态数据库租约示例。
仍不支持任意模板图结构变化、循环项单独复用、非 HTTP 或未校验的 artifact 复用、复合 retry groups、动态模板引用及任意执行通道的自动机密注入。
真实云适配器需要独立实现；社区 CI、API 设计评审和合并状态以两份 PR 为准。
不能将此实验版当作文档的全部功能已交付。
英文 `local-testing.md` 包含可复现命令；逐项边界见 `implementation-decisions.md` 的 Feature acceptance ledger。

## 从执行记录返回编辑

在原生 Workflow 详情中展开“Parameter revision and execution inputs”，点击“Edit a parameter version”。页面自动加载本次执行原有的绑定、参数集和执行修订，并展示它与当前已保存修订的差异。保存时使用当前草稿的 resourceVersion，遇到并发修改会拒绝覆盖。

修改并保存后可以分别选择新建关联执行，或预览局部重试。入口使用当前绑定与表单；绑定已更换参数模型或原资源被删除时会明确报错。

## 新参数局部重试

1. 从已完成 Workflow 返回参数编辑，修改并保存参数版本。
2. 点击 **Preview partial retry**，核对受影响任务和保留结果数量。预览不会启动执行。
3. 点击 **Apply revision and retry this Workflow**。新版本只更新同一 Workflow 的参数；不会新建 Workflow。
4. 查看 Workflow 的 UID、原审批和无关成功任务的完成时间及输出，应保持不变。
5. 在 ParameterRetry 记录中查看冻结的重试前 Workflow、原/新参数版本与影响计划。

当前支持已结束的 DAG/Steps、具名静态嵌套调用、固定身份复合循环及输入未变化的循环结果复用。
叶任务必须显式声明 `parameters.workflows.example.io/result-reuse: same-inputs` 才能保留既有结果；
会重新执行的非审批任务必须声明 `parameters.workflows.example.io/retry-safe: idempotent`，并由动作实现真正保证幂等。
影响分析以改变的任务输入及其下游依赖为范围，实际重试复用 Argo 原生逻辑。
任意模板结构和条件变化、未经声明的循环变化、未知副作用、未校验的 artifact 复用、递归、memoization 和复合 retry groups 等不支持的恢复情况会明确拒绝。
机密引用作为参数传递时，显式参数重试可以解析最新版本；模板内固定 Secret 环境配置的变化仍视为结构变化。
原生审批节点可以保留；不能把改变了输入的任务旧结果伪装成新结果。
可运行 `test/parameterization/partial_retry_e2e.py`，验证实际保存→失败→修改→Diff→预览→同 Workflow 局部重试和幂等重放。

## 嵌套模板引用（v12）

安装 `examples/parameterization/nested-refs.yaml` 后，选择 `nested-library-demo`。
保存 items 数组，预览并提交，再在原生 Workflow 中通过 approve 暂停节点。
根 DAG 调用命名空间 WorkflowTemplate 的 steps，steps 再调用 ClusterWorkflowTemplate；
items 数量决定原生循环次数，结果通过原生输出参数返回。

提交快照把静态 templateRef 的实际引用闭包内联为独立原生模板，保留循环、依赖、
条件、输入输出和退出 hook，不再在运行途中读取最新模板。预览后引用变化会拒绝
旧预览；提交后引用被修改或删除，原运行及原参数重试仍沿用冻结内容。
引用读取使用当前调用者权限，命名空间模板限本 namespace，集群模板需显式 GET 权限。

动态计算的 templateRef 名称尚不支持，预览会明确拒绝。跨命名空间引用没有开放。引用闭包最多 64 个资源、
256 个外部模板，inline 嵌套最多 64 层。这些是资源保护上限，不是业务拓扑限制。

新增完整路径用例：`test/parameterization/template_refs_e2e.py`。它创建每次独立命名的
合成模板，在提交后删除自己的源引用，确认三个循环输出正确，再制造可回读依赖失败，
修复依赖后原参数重试；断言 Workflow UID 与已成功节点和审批结果保持。

## 顶层 ClusterWorkflowTemplate 绑定（v13）

绑定的 `workflowTemplate` 指定原生模板名，`clusterScope: true` 选择 ClusterWorkflowTemplate。
省略或 false 继续选择本 namespace 的 WorkflowTemplate；同名资源不会相互替代。
WorkflowActionBinding 使用相同字段，调用目标的名称和 scope 必须一致。
参数集、表单、绑定、版本和生成的 Workflow 仍在提交 namespace 内；读取集群模板需要调用者的 Kubernetes GET 权限。

1. 在独立实验集群安装 `examples/parameterization/cluster-template.yaml`（先安装 demo.yaml）。
2. 打开 Cluster Workflow Templates → cluster-training-demo，点击 `cluster-training` 绑定；Submit 面板也显示此入口。
3. 填写 dataset、epochs、shuffle，保存版本并预览，Definition versions 中可见 ClusterWorkflowTemplate 的 UID/版本。
4. 提交后查看原生 Workflow。提交前模板变化会拒绝旧预览；提交后的源模板变化不会改写已经冻结的执行。

完整路径回归设置 `ARGO_PARAMETERIZATION_CLUSTER_ROOT=true`，运行 `test/parameterization/template_refs_e2e.py`。
用例覆盖集群根模板 → namespace steps → 集群叶子模板、三项循环、预览过期拒绝、提交后删除源模板，以及依赖修复后的原参数重试；Workflow UID 和成功节点及审批保留。

## 动作输入输出契约

ActionDefinition 的 inputSchema/outputSchema 会与原生实现一起冻结到本次模板。
字符串参数使用原生文本；integer/number/boolean/object/array/null 参数使用 JSON 编码。
每个顶层字段需声明具体 type，不支持的编码或 Plugin 等组合会在预览时拒绝。
原生 artifact 通过单独的接口映射传递文件引用，不作为 JSON 参数内容校验。
容器、Resource 与 HTTP 动作在原生执行前校验输入，在原生结果回写后校验输出。
HTTP/Script 的原生 outputs.result 可以通过 outputSchema 的 result 字段约束。
无动作契约注解的原生模板保持原有行为；必须一起安装增强 Server 和 Controller。
执行错误只显示字段路径，不回显参数值。
`test/parameterization/action_contract_e2e.py` 覆盖错误输入不创建 Pod、错误输出阻断下游、正确容器及 HTTP 输出继续执行。
可选 ResourceOperation 控制器的本地完整路径见下节；普通原生动作不会自动获得外部幂等能力。

## 外部动作恢复示例

选择 `operation-demo-report` 绑定，填写新的 Parameter set 名称；`mode` 使用 `normal`，
`delay` 使用字符串 `2`，保存、预览并提交。原生 Resource 节点创建 ResourceOperation，
参考控制器向本地合成服务提交，轮询外部状态；成功后输出 `reportURI` 给下游节点。
这不是云资源调用，也不依赖生产 runtime。

`test/parameterization/operation_e2e.py` 覆盖提交响应丢失、控制器重启、同 Workflow 原参数重试、
取消确认、脱离后继续运行、未知结果不重复提交、非法输出阻断下游，以及伪造调用身份被拒绝。
在无法证明外部结果时会保持 Unknown，而不会为了继续流程重复创建资源。

## 版本 Diff

保存多个版本后，在 Revision history 选择 **Compare from**、**Compare to**，再点 **Compare revisions**。
参数值、参数模型、值来源分组显示；可按路径或数组项身份筛选。
数组按声明的稳定身份比较移动；身份规则改变时会明确提示，并整体比较数组。
缺失、null、false 和 0 分别展示。敏感字段只展示引用和版本信息。
`test/parameterization/diff_e2e.py` 验证实际保存、模型变更、历史回读到差异返回，并断言不创建 Workflow。

## 原生提交与 admission

可选校验 webhook 使用 `server/parameterization/admission.go` 与 Argo Server 相同的校验函数。
用 `test/parameterization/install_admission.py` 在独立本地 k3s 安装；只匹配明确携带增强快照注解的
Workflow。普通原生 Workflow 不受影响。证书只用于实验 webhook，不加入系统信任库。

采用增强能力的直接 Kubernetes 提交需要有效的冻结 WorkflowSubmission 和参数版本，且实际 spec
与解析结果一致。改写执行输入需要经过局部重试计划；不能直接 PATCH 参数来冒充已验证的执行。
原生状态回写、暂停、继续和终止沿用冻结契约，不依赖最新模板仍存在。
webhook 对每次依赖读取检查实际调用者权限，不获得任意用户 impersonate 权限。

执行 `test/parameterization/admission_e2e.py` 验证 API 与 kubectl 两条入口；
`test/parameterization/partial_retry_e2e.py` 同时覆盖新参数重试失败后，修复外部依赖再用原参数重试。
发布安装必须配置当前部署范围、读权限、证书续期及快照 CR 的写权限，不能将实验清单直接用于生产。

## v4.0.1 本地构建与验收边界

Server 使用 `argo-parameterization:local-v37-v4.0.1`，
Controller 使用 `argo-parameterization:local-v31-amd64-v4.0.1`，
校验服务使用 `argo-parameterization-admission:local-v34-v4.0.1`，
示例动作控制器使用 `argo-operation-example:local-v35.1-v4.0.1`。
示例服务只创建合成任务或本地 ConfigMap，不连接生产云服务。

本次完整路径用例为 `test/parameterization/` 下的 `e2e.py`、
`partial_retry_e2e.py`、`template_refs_e2e.py`、`secrets_e2e.py`、
`inputs_e2e.py`、`action_contract_e2e.py`、`admission_e2e.py`、
`operation_e2e.py`，原生互斥锁、终止、恢复及退出动作由 `controls_e2e.py` 覆盖；Diff 另由 `diff_e2e.py` 覆盖。
使用上述独立 kubeconfig，并设置 `ARGO_PARAMETERIZATION_URL=http://127.0.0.1:2747`。
只复测失败用例或后续代码修改影响的路径，不把远端页面测速列为本地验收。

仍未完成：Vault 等其他机密后端、动态敏感数组与机密中间输入、
复杂嵌套/拓扑变化的局部重试以及通用外部动作适配器。
当前版本可以试用已列明的范围，不能视作全部设计验收通过。

## 分层执行差异（本地 v26）

在原生 Workflow 详情的 **Parameter revision and execution inputs** 中，点击
**Compare execution with current draft and definitions**。左侧来自该运行的执行快照，
右侧来自当前草稿和当前绑定定义。页面分别显示参数值、模型、来源、参数绑定、动作绑定、
表单、动作契约、实现及 Workflow 结构、编译后的输入。仅比较不会提交、恢复或重试 Workflow。

新提交和新参数局部重试将完整绑定、动作和表单定义保存在不可变收据内；原参数重试继承
当前已采用的定义快照。缺少这些内容的旧收据明确显示历史不可用，不会读取最新定义替代历史。
原生模板和实际编译输入仍可从既有 Workflow 比较。新参数重试的计划摘要包含解析后的执行摘要，
预览后定义变化需要重新预览。旧版重试收据继续兼容。

编译输入中的上游输出表达式保持原生语义；该预览不预测未来任务输出，也不代表所有运行期
输入已经解析。Sensitive 参数仅展示已保存的 Secret 引用；比较接口不读取 Secret 明文用于展示。

完整路径用例：

```bash
ARGO_PARAMETERIZATION_URL=http://127.0.0.1:2747 python3 test/parameterization/execution_diff_e2e.py
ARGO_PARAMETERIZATION_URL=http://127.0.0.1:2747 python3 test/parameterization/partial_retry_e2e.py
```

第一个用例提交一条冻结运行，更新动作/绑定/表单并删除旧源定义，比较差异后继续原运行，
核验执行的仍是原参数和原实现。第二个用例核验新参数局部重试后再用原参数重试，保留
同一 Workflow、无关的成功结果、审批和已采用的定义快照。两者只创建唯一命名的本地模拟资源。

## 分页读取验收

`test/parameterization/paged_documents_e2e.py` 覆盖目录跨页、27 个大正文版本、分页期间并发保存、未提交孤立版本排除、按需恢复、Diff 与原生执行。列表不返回正文或 kubectl 的 last-applied 注解。完整文档仍通过当前调用者的 Kubernetes 权限读取。

摘要读取和长历史续读是独立能力；长历史分页及 Diff 分页见下文。

## 保存版本 Diff 分页（v28）

在独立本地环境中打开 `http://127.0.0.1:2747/parameters/argo-parameterization-lab`。
选择 `diff-pages-` 开头的验收绑定，填写同名参数集并加载已保存内容，展开版本历史，
选择最初两个版本进行比较。260 个字段变化分为三页，使用 Previous/Next diff page 翻页。
界面只渲染当前页，路径筛选也只作用于当前页；其他页有变化的分组不会显示为“无变化”。
历史恢复仍须显式恢复到编辑器、保存，并在预览后显式提交；翻页不会修改或执行参数。

接口每页重新读取两份版本并核对权限和敏感字段，游标只能用于同一比较。
此项覆盖保存版本比较，长历史续读见下文。

## 执行 Diff 分页（v29）

从一条增强绑定创建的 Workflow 打开“Parameter revision and execution inputs”，点击“Compare execution with current draft and definitions”。
参数值、参数模型、来源、绑定、表单、动作和编译后输入按层次分页，共用每页 100 项的上限。
翻页始终比较同一执行 UID、参数版本及定义内容；定义中途变化时需要重新比较，不会自动混入新配置。
Workflow 只更新运行状态不会使比较失效。
页面筛选仅作用于当前页；历史缺失的定义仍明确标为不可用。
比较不会提交或重试运行。

完整路径用例 `test/parameterization/paged_execution_diff_e2e.py` 冻结原运行，修改 130 个参数和 130 个表单标签后分页比较，再继续原运行并核对 ConfigMap 仍保存原值。

## 长版本历史续读（v30）

历史分页不再每次从最新版本重新扫描，也不再受 1000 项上限影响。
每页使用经过签名校验的续读位置，仅查询参数集、最初的版本身份和本页记录；仍按当前调用者检查 Kubernetes 权限。
翻页期间新增的版本在重新加载历史后显示，不会插入正在浏览的旧快照。
并发保存失败留下的孤立版本不会出现在已保存历史里。

多副本服务器必须挂载同一份随机 32 字节的 Secret 文件，通过 `ARGO_PARAMETERIZATION_HISTORY_KEY_FILE` 指定。
本地单副本默认使用进程随机密钥；重启、密钥轮换或游标 15 分钟未使用会返回过期提示，点击 **Reload history** 即可，不需要重新保存参数。
当前支持顺序翻页，不是任意页码跳转；单次查询有 100 万项显式上限。

完整路径 `test/parameterization/long_history_e2e.py` 覆盖 1006 次实际保存、跨页读取、孤立版本排除、并发更新、恢复最初版本和原生执行。

## 完成状态写入恢复

`test/parameterization/status_recovery_e2e.py` 在独立本地命名空间创建合成参数、绑定与原生 Resource Workflow。
用例临时拦截这一个 Workflow 的成功状态写入，在看到真实拒绝后移除故障。
控制器应自动回读并写回成功，不创建新 Workflow、不重试任务、不重启控制器。
断言保留同一个 Workflow UID、Pod UID 和结果 ConfigMap UID，避免只验证页面状态。
该用例的临时 Webhook 只匹配合成 Workflow 的名称和命名空间，结束时自动移除。

## 声明式表单联动（v32）

`WorkflowForm.spec.layout` 支持分组、标签页和嵌套树，字段通过 `group` 引用布局。
`fields` 支持文本、数字、复选框、选择、对象、数组、对象数组表格、YAML 和已声明的机密引用控件。
字段的 `readOnly`、`visibleWhen`、`enabledWhen` 作用于展示交互；隐藏字段保留原值，服务端仍校验完整参数。
数组字段使用 `/zones/*/count` 一类路径，同一行的条件可使用 `/zones/*/active`；数量由实际参数决定。
对象数组的语义身份仍来自 ParameterTemplate，不由表格控件重新定义。

选择本地 `form-ui-` 开头的绑定，Parameter set 填写同名后点击 **Load saved set**。
在 Basics 页切换 Enable changes 与 environment，观察字段启用、隐藏和 region 候选项变化；在 Network 页增删表格行、编辑 YAML。
无效 YAML 会阻止保存，即使切到另一个标签页仍会保留错误；改正后保存并预览，再提交才会执行。
已保存的选项若被来源删除，服务端预览也会拒绝。
这份合成示例只写入实验命名空间的 ConfigMap。

`test/parameterization/form_ui_e2e.py` 通过 Playwright 驱动真实页面，覆盖布局、条件、动态选项、数组增删、隐藏值保留、保存修订、Diff、原生执行和实际结果回读。
它也提交原生 suspend 输入节点，在修改源表单后核验运行仍采用冻结表单，输入输出实际传给下游容器。
运行需 Python Playwright、Chromium 和本地端口转发，可用 `ARGO_PARAMETERIZATION_CHROMIUM` 指定浏览器路径。
运行中动态选项与机密引用已在后续版本补齐，见 v37、v39 及机密最新版本说明；无效布局、未知路径及不兼容控件仍会报错。

## 定义编辑入口（v33）

打开 **Resource definitions and records**，选择 ParameterTemplate、WorkflowForm、WorkflowParameterBinding、ActionDefinition 或 WorkflowActionBinding。
点击 **Create definition** 创建定义；选择已有资源后，可修改 spec 或通过 **Clone as new definition** 复制新定义。
已发布动作定义和动作绑定保持不可变，只允许复制为新版本，再明确更新参数绑定。
**Validate definition** 使用真实 Kubernetes dry-run，按已安装的 CRD 结构校验，不写入资源；实际引用与输入值仍在执行预览阶段校验。
保存更新要求读取时的 UID 与 resourceVersion 同时匹配，冲突时保留编辑内容，由用户重新读取比较。
页面不能修改 status、metadata、参数版本或执行回执，也不会提交或重试 Workflow。

`definitions_ui_e2e.py` 从真实浏览器创建五种定义，验证 dry-run 不写入、并发修改冲突、未知字段拒绝，再通过这些定义填写参数、保存修订、预览动作、原生执行并核对 ConfigMap 结果与冻结表单。
具名不可变发布、历史比较与克隆恢复见上文；定义状态见 v34 说明。
此入口的结构校验不代表定义已完成业务输入或执行权限验收。

## 定义校验状态（v34）

独立本地 k3s 增加 `parameterization-definitions`，仅负责五类定义的校验状态。
它通过专用 RBAC 只能更新定义的 `/status`，不能修改定义内容、Workflow、执行收据，也不能读取 Secret。
原生 Workflow Controller 仍负责所有调度、重试和节点状态。

在参数资源目录选择已保存定义，可以查看静态校验状态、具体错误和引用对象的 UID、内容摘要。
修改模型后，引用它的表单与绑定会自动重新校验；删除并重新创建模型也会识别新的 UID。
点击刷新目录后可查看最新结果；`observedGeneration` 落后于当前代数时显示待校验。
没有固定模型的通用表单显示需要绑定上下文，不会被错误标为已校验成功。

`test/parameterization/definition_status_e2e.py` 从缺少引用、修复引用、参数保存、预览、执行到页面状态展示走完整路径。
还检查模型变更传播、删除重建、原生模板变更、权限隔离、无状态回写循环，以及旧 Workflow 成功节点与冻结执行收据保持原样。
状态刷新不再改变执行内容摘要，编辑保存仍保留并发版本保护。
此能力不代表所有执行参数、动态数据源及运行权限已经获准；这些仍在预览和提交时重新校验。

## 数组机密输入（v36）

打开本地参数页，选择 `secret-array-demo`，填写新的 Parameter set 名称。
数组行的 `id` 是稳定身份；在每行的密码输入框填写至少 8 个字符的测试值，点击 **Store new encrypted secret version**。
写入期间其他表单操作暂时禁用；普通参数状态、YAML 和保存版本只显示引用。
可以增删或移动数组行，保存后通过 Diff 查看移动及引用变化。
点击 **Validate and preview**，核对后点击 **Submit this revision**，原生循环容器逐项通过 SecretKeyRef 消费输入，只输出行 ID 和消费成功标记。
示例仅操作独立实验命名空间，勿输入生产凭据。
新增编码 `secret-reference-json` 只传引用；普通 JSON 映射仍拒绝机密字段。
原参数重试会重新核对被冻结的 Secret 身份，已删除或替换的凭据会明确报错。

## 运行中机密输入（v37）

在本地参数页选择 `secret-input-demo`，填写新的参数集名称，保存空参数、预览并提交。
打开生成的 Workflow，在 `authorize` 等待节点下的 “Operation authorization” 表单输入至少 8 个字符的虚拟凭据和说明。
先点击 “Store new encrypted secret version”，确认输入框清空、表单只保留引用，然后点击 “Validate input and continue this node”。
原生 suspend 节点完成后，后续容器通过 SecretKeyRef 消费凭据，日志只输出 `runtime-credential-consumed`。
`ParameterNodeInput` 可查看冻结契约和 Secret UID 版本，不包含机密明文。
完整路径回归 `test/parameterization/sensitive_inputs_e2e.py` 另行覆盖源定义删除、响应丢失恢复、原 Workflow 原生重试及撤销/替换凭据后的拒绝行为。
测试只使用独立本地 k3s 和虚拟凭据。

## 参数引用与局部重试（v38）

`partial_retry_e2e.py` 覆盖保存、Diff、影响预览、新参数重试和原参数重试的完整路径。
其中 `value` 与 `valueExtra` 分别处理，表达式支持静态方括号引用，普通说明文本不作为参数依赖。
三项无关循环结果和一次人工审批保持原状态，整个过程只有一个 Workflow UID。
这是 v38 时的范围记录；当前已支持下述静态嵌套和明确允许的整个循环展开替换，其他变化仍不会通过重新创建 Workflow 绕过校验。

## 运行中动态选项（v39）

选择 `input-options-demo`，保存空参数并提交，在原生 Workflow 的等待输入节点选择环境和地域。
选择 `test` 时可选 `Test East`，切换 `production` 后原选择会提示不可用，此时选择 `Production West` 并提交。
节点会把所选值作为原生输出交给后续容器，仍由 Argo 调度。
完整路径测试另行覆盖源表单删除、候选值在提交前失效、来源不可访问、来源版本记录，以及成功后读取历史不依赖候选源。
演示的 ConfigMap 只包含本地合成数据，不访问真实云账号。

## 嵌套 DAG 的新参数局部重试（v40）

`nested-retry-demo` 两次调用同一个嵌套 DAG，各自包含审批、循环任务和参数消费任务。
首次保存 `value: invalid`、`items: [one, two]` 并提交，完成两处原生审批后，vary 分支会失败。
从该 Workflow 返回参数编辑，只把 `value` 改成 `valid`，保存后预览局部重试。
受影响调用应为 `vary/change` 与 `report`；确认后沿用同一个 Workflow，之前的两次审批及无关结果保留。
节点详情中 vary 边界显示新输入，ParameterRetry 回执保留旧输入。
完整路径用例为 `test/parameterization/nested_retry_e2e.py`。

## 引用已有原生动作实现

安装 `examples/parameterization/action-reference.yaml` 后，在参数页面选择 `action-reference-demo`，创建参数集并填写 `message`。
保存、预览后提交，在原生 Workflow 页面恢复 `approve` 节点；`emit` 的输出参数应与输入相同。
ActionDefinition 的 `implementationRef` 指向已有 WorkflowTemplate 中的 `emit`，不是第二份执行器或业务 DAG。
支持同命名空间 WorkflowTemplate 和具备读取权限的 ClusterWorkflowTemplate；引用模板必须是受支持的原生叶子动作。
`template` 与 `implementationRef` 二选一，提交前校验契约并固定引用版本，源模板后续修改不会改变已提交运行。
`test/parameterization/action_reference_e2e.py` 覆盖两种作用域、参数保存、绑定、过期预览拒绝、输入输出、源删除及同 Workflow 重试，并验证原审批和成功 Pod UID 保留。
此演示保留旧的模板名绑定；下面的新演示使用具名调用路径，v47 增加独立的 artifact 接口示例；Plugin 实现仍未开放。

## 可选动态字段与依赖联动

`optional-options-demo` 用于验证初始表单和运行中输入表单的相同行为。
创建参数集并设置 `count: 2`，可以保持可选的 `environment`、`region` 未填写；预览和提交不要求这些未使用字段的选项提供方在线。
进入原生 `choose` 输入节点后，勾选 `approved` 并继续；最终 Pod 输出应为 `2:true`。
依赖值尚未填写时，选项框显示依赖提示且不发起查询，已有选择不会被自动清空。
只跳过不存在的可选值；必填约束、默认值及显式填写的 false、0、空字符串、null 都仍须通过服务端校验。
`test/parameterization/optional_options_e2e.py` 覆盖两个真实浏览器入口、参数保存、预览、冻结运行中输入和实际容器输出。

## 按调用路径绑定动作

选择 `action-calls-demo`，保存 `message: invalid` 并提交，恢复原生 `approve` 审批。
`east/emit` 使用绑定动作并失败，`west/emit` 仍使用同一共享模板原有实现并成功。
返回参数编辑，只改为 `message: fixed`，保存并预览局部重试。
确认后应保留 Workflow UID、原审批与 west 的成功 Pod，east 返回 `action:fixed`，report 返回 `base:action:fixed`。
查看 WorkflowActionBinding 可见 `callPath: [main, east, emit]` 和输入输出映射。
源 WorkflowTemplate 不会被编译器修改。

## 机密最新版本策略

选择 `secret-versions-demo`，在机密控件选择 `Latest at submission`，索引 ConfigMap 填 `secret-versions-demo`，键填 `current`。
保存的参数只包含索引引用；预览显示解析后的不可变 Secret 版本身份。
初始演示索引指向合成凭据 v1，提交并审批后 consume 故意失败。
把本地演示 ConfigMap 的 current 更新为 `{"name":"secret-versions-demo-v2","key":"token"}`。
选择原参数重试仍使用 v1；选择参数重试并核对差异后才使用 v2。
最新版本策略不会偷偷改变原参数重试；凭据撤销会明确阻断。
完整路径用例为 `test/parameterization/secret_versions_e2e.py`，不使用真实凭据。

## Steps 与嵌套循环局部重试

`steps-retry-demo` 以原生 Steps 组织两处嵌套调用。
保存 `value: invalid`、`items: [one, two]`，审批后 vary/change 失败。
仅将 value 修改为 valid，参数重试应保留稳定分支、已有审批和成功 Pod，仅刷新 vary/change 及 report。
`test/parameterization/steps_retry_e2e.py` 同时覆盖固定循环项的嵌套 DAG 调用。
此稳定循环用例之外，下面的拓扑用例支持经作者允许的整个循环展开替换；任意模板结构修改和未经校验的 artifact 复用仍受限制。

## v47：Artifact、Vault 与循环拓扑

入口仍为 <http://127.0.0.1:2747/parameters/argo-parameterization-lab>。
在参数页面的 Binding 列表选择以下示例；均使用本地合成数据。

| Binding | 输入 | 验证目标 |
| --- | --- | --- |
| `artifact-mapping-demo` | `message: artifact-content` | 文件输入映射、原生上传、原参数重试、下游下载与摘要 |
| `topology-retry-demo` | `items: [a, b]` | 同一个 Workflow 的循环增加、删除、排序和空数组 |
| `external-secret-demo` | 下方的 latest 引用 | Vault 固定版本、轮换和撤销 |

### 文件映射

保存、预览并提交 `artifact-mapping-demo`，在原生 Workflow 详情恢复 `approve`。
`transform` 会生成文件，随后 `gate` 因测试用的 ConfigMap 尚不存在而失败。
用本地 kubectl 创建 `<本次 workflow 名>-ready` ConfigMap，再选择原参数重试。
`consume` 下载原来的文件并输出 SHA-256 摘要；`transform` 的完成时间和输出引用应保持不变。
示例文件存储仅用于本地测试，重启存储 Pod 会丢失其中的数据。

```sh
kubectl --context k3d-argo-parameterization -n argo-parameterization-lab \
  create configmap '<本次 workflow 名>-ready' --from-literal=ready=true
```

### 循环拓扑变化

提交 `topology-retry-demo` 的 `items: [a, b]`，恢复 `approve`，等待成功。
编辑同一参数集为 `items: [b, c, d]`，预览新参数重试，再确认。
影响范围应为 `expand` 和 `report`，Workflow UID、`approve` 及 `stable` 的执行记录应保留。
继续分别改为 `[]`、`[restored]`，验证缩减到零再扩展的路径。
这是整个受影响循环的重新执行，不会把旧数组下标的结果套用到新的数组项；它也不会自动删除先前创建的外部资源。

### 外部机密

`external-vault-demo` 是本地 Vault 开发模式实例，使用明确标为合成的测试凭据。
`external-vault-demo-bridge` 通过只读版本接口把已登记条目写成不可变 Secret，并维护 latest 索引。
`external-secret-demo` 的输入如下；也可在已有 Secret 控件里选择最新版本索引并填入名称和键。

```yaml
credential:
  latestSecretRef:
    configMap: external-vault-demo
    key: current
```

示例初始化时已写入足够长度的合成值，提交后恢复 `approve` 即可执行。
执行快照里只能看到固定 Secret 引用，不能看到 Vault 值。
下面的命令只操作独立本地 Vault，可用于生成下一版合成凭据，然后使用同一参数修订预览新参数重试。

```sh
kubectl --context k3d-argo-parameterization -n argo-parameterization-lab \
  exec deployment/external-vault-demo -- \
  vault kv put secret/demo token=synthetic-local-rotated-value-long-enough
```

外部版本撤销和后端不可用由 `external_secrets_e2e.py` 验证；测试还检查原参数固定版本、审批继承、历史及日志不含明文。
此实现会将值复制到 Kubernetes Secret，仍要求 Secret 静态加密和仅记录元数据的审计策略。
它不提供逐次执行签发动态凭据，也不能清除已经注入运行中进程环境的值。
过期 Secret 清理由 bridge 持续运行完成；参数化 Workflow 在新建任务 Pod 前也会检查外部验证有效期。
因此，等待审批期间过期的引用会阻止新建任务 Pod；同一不可变 Secret 续期后，可用原参数重试，保留原审批及 Workflow UID。
已有 Pod 和已完成节点不会因之后过期而被倒改失败。
此检查发生于 Pod 创建前，无法保证排队后 kubelet 实际注入时仍然有效，也不覆盖已注入进程或 HTTP/Plugin agent 的逐次请求；严格注入与续租策略仍是后续任务。

## 动作替换审核

新增完整路径脚本 `test/parameterization/action_replacement_e2e.py`，沿用上述独立本地 kubeconfig、context 和服务 URL。
脚本覆盖定义创建、缺少审核、目标范围变化、引用实现变化、提交前拒绝、实际原生执行，以及保留原审批和无关成功 Pod 的同 Workflow 新参数重试。
运行结果中的 `action-contract-*` 定义和 Workflow 保留在本地 UI，可查看契约、绑定、参数修订和实际输出。

动作的 `spec.executionContract` 声明副作用、目标范围、权限范围、输出含义及能力适用条件。
通过 `GET /api/v1/parameterization/{namespace}/action-reviews/{name}` 读取解析后的实现和摘要，检查后由工作流作者在原始叶子模板上设置 `parameters.workflows.example.io/action-replacement-policy`。
策略包含 `contract`、`implementationDigests` 和 `interface`，后者明确列出绑定采用的参数及 artifact 映射。
读取审核材料不会批准或执行动作；源码和声明一致也不等于自动证明程序或远程服务的行为，实际权限继续由原生执行身份及目标系统执行。

## 等待期间机密过期

运行 `test/parameterization/secret_dispatch_e2e.py`，脚本使用本地独立资源验证保存、预览、提交、等待审批、凭据过期、阻止创建 Pod、续期和原参数重试整条路径。
成功后在本地 UI 查看 `secret-dispatch-*` 参数集及关联 Workflow，可确认审批只完成一次，任务只创建一个 Pod，重试没有替换 Workflow。
原生执行只输出消费成功标记，参数历史、错误和操作记录中不保存机密明文。

## 原生重试组的新参数局部重试

运行 `test/parameterization/retry_groups_e2e.py`，脚本分别验证 DAG 和 Steps。
测试中的动作第一次尝试故意失败；无关动作在第二次成功后，修改另一动作的参数并局部重试。
在本地 `retry-groups-*` 参数集及原生 Workflow 中，原审批、无关动作的两个 Pod UID，以及它第一次失败的记录应全部保留。
变化动作和输出消费动作使用新参数重新执行，Workflow UID 保持不变。
这一能力覆盖 container、script、resource 叶子动作的原生 retryStrategy；组合 DAG/Steps 重试组仍明确拒绝，不能据此视为任意拓扑重试已完成。

## 新参数局部重试复用 HTTP 文件

先部署 `examples/parameterization/artifact-mappings.yaml` 中的本地文件服务，再部署 `examples/parameterization/artifact-retry.yaml`。
在参数编辑器选择 `artifact-retry-demo`，保存 `{"mode":"bad"}`，预览、提交并恢复原生审批节点。
文件生产者应成功，消费节点会故意失败。
把参数保存为 `{"mode":"good"}`，预览并应用新参数局部重试。
预览应只选择 `consume`，完成后其 `digest` 应等于生产者的 `checksum`；生产者 Pod UID、原审批和 Workflow UID 保持不变。

`test/parameterization/artifact_retry_e2e.py` 覆盖 DAG 和 Steps 的保存、预览、执行、文件篡改、文件缺失、拒绝重试、恢复文件以及下游真实下载路径。
预览通过后，提交重试仍会重新核对文件；缺失或变化不会清理原执行记录。
当前仅支持明确 HTTP 地址、未归档文件及作者声明的 SHA-256 输出参数，最多 32 个文件、共 64 MiB、10 秒。
该校验不保证文件此后不再变化；实际下载继续由原生执行器完成。
文件服务仅用于本地演示，Pod 删除后文件消失；不要用于存放机密。

## 动态凭据本地演示

`dynamic-credential-demo` 通过原生动作容器中的 Vault Agent 获取和续租数据库凭据，参数只保存 bootstrap Secret 引用。
每次尝试的公开租约编号写入原生输出，数据库用户名和密码不进入参数、历史或日志。
本地完整路径已验证续租、撤销失败、拒绝授权、原 Workflow 重试的新租约和原审批保留，以及完成和取消后的 TTL 回收。
安装及操作步骤见 [Native dynamic credential leases](local-testing.md#native-dynamic-credential-leases)。
这是一条明确配置的原生动作集成，未增加另一个凭据调度器，也不代表所有 HTTP 或 Plugin 执行渠道自动支持动态凭据。

## Per-call result bindings

See [bound action results](execution-results.md) for the reusable result CR, output mappings, executor-owned observations and frozen retry semantics.
The complete path is `test/parameterization/result_bindings_e2e.py` and includes browser verification.
For step-by-step instructions in Chinese, see [result binding acceptance](execution-results.zh.md).
