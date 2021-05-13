# Changelog

## v3.1.0-rc3 (2021-05-13)

 * d2bd1def Merge branch 'master' into release-3.1
 * 37f8c7cd ci: Auto-generate changelog (#5903)
 * ba700eed test: Fix bug in port-forward.sh (#5895)
 * e71d33c5 fix(controller): Fix pod spec jumbling. Fixes #5897 (#5899)
 * 9a10bd47 fix: workflow-controller: use parentId (#5831)

### Contributors

 * Alex Collins
 * Jan Heylen

## v3.1.0-rc2 (2021-05-12)

 * df66c5af build: use fetch-depth=0

### Contributors

 * Alex Collins

## v3.1.0-rc1 (2021-05-12)

 * 3fff791e build!: Automatically add manifests to `v*` tags (#5880)
 * be207a97 test: 3m faster and more robust e2e (#5883)
 * 23175b6b docs: Fix typo (#5884)
 * 6487afb4 docs: Added a period to the end of the sentence (#5869)
 * d86d458b build: Draft releases automatically (#5867)
 * 2687e240 fix(controller): Fix active pods count in node pending status with pod deleted. (#5836)
 * 3428b832 fix(controller): Error template ref exit handlers. Fixes #5835 (#5837)
 * d3eb5bc6 docs: Add FAQs in memoization doc and add link in the err message (#5847)
 * 1a539359 fix(controller): Remove un-safe Sprig funcs. Fixes #5286 (#5850)
 * 206aff00 docs: Added organization (#5856)
 * c6825acc fix(executor): Enable PNS executor to better kill sidecars. Fixes #5779 (#5794)
 * 2b3396fa feat: configurable windows os version (#5816)
 * d66954f5 feat(controller): Add config for potential CPU hogs (#5853)
 * 7ec262a5 feat(cli): Support input from device files for lint command (#5851)
 * ab786ecb fix: Reset started time for each node to current when retrying workflow (#5801)
 * 8f59af53 ci: Run Snyk on `go.mod` (#5787)
 * e332be5e fix(ui): dont show cluster workflows in namespaced mode. Closes #5841 (#5846)
 * c59f59ad feat: Support Arguments in Exit Handler (#5455)
 * 5ff48bbc feat: Allow to toggle GZip implementations in docker executor (#5698)
 * 86545f63 5739 (#5797)
 * a4d275af docs: Fix emptyDir pattern example. Fixes #5825 (#5826)
 * 461b0b3c fix(executor): Fix artifactory saving files. Fixes #5733 (#5775)
 * 5ba08ed8 docs: wrong parent name of 'wait-approval' step (#5815)
 * 507b92cf feat(cli): resubmit workflows by label and field selector (#5807)
 * 3f808661 chore: Surface parse duration error in operator (#5814)
 * a65cc3e2 docs: fix typo in async-pattern page (#5810)
 * bdd44c72 fix: Add note about hyphenated variables (#5805)
 * b9a79e06 feat(cli): Retry workflows by label selector and field selector (#5795)
 * 8f2acee3 fix: Node set updating global output parameter updates global. #5699 (#5700)
 * 076ff18a feat(controller): Add validation for ContainerSet (#5758)
 * 4b3a30f4 fix: Reset workflow started time to current when retrying workflow. Fixes #5796 (#5798)
 * 418395b7 build: fix codegen failure on master branch (#5799)
 * 4af01131 fix: change log level to warn level (#5790)
 * 0283fafd test: Refactoring test JSON marshalling. (#5785)
 * 7e974dcd fix(docs): Fix yaml snippet (#5788)
 * 01f37b81 docs: fix typo (#5781)
 * 4a55e6f0 feat: Support bucket lifecycle for OSS artifact driver (#5731)
 * 3cdb22a1 feat: Emit WorkflowNodeRunning Event (#5531)
 * 66c77099 upgrade github.com/gogo/protobuf (#5780)
 * 26f08c10 chore: Prevent free-port.sh from killing browsers (#5791)
 * 4aff34c9 docs: Add training doc (#5777)
 * cb55cba0 fix(ui): Fix an UI dropdown flickering issue (#5772)
 * 9dd3c6a4 docs: Update managed-namespace.md (#5768)
 * 60a64c82 feat(cli): Stop workflows by label selector and field selector (#5752)
 * 0efb57d0 docs: Redirect Argo community documents to argoproj/argoproj (#5734)
 * 05af5edf fix(ui): Fix the UI crashing issue (#5751)
 * 40774004 fix(ui): Remove the ability to change namespaces via the UI in Managed Namespace Mode. Closes #5577 (#5729)
 * 2a050348 fix(ui): Fix workflow summary page unscrollable issue (#5743)
 * 11e045dc chore: Fix expr array equality operator (#5744)
 * 500d9338 fix(ui): Fix greediness in regex for auth token replacement (#5746)
 * 284adfe1 fix(server): Fix the issue where GetArtifact didn't look for input artifacts (#5705)
 * 511bbed2 fix(ui): Fix workflow list table column width mismatch (#5736)
 * 0a1bff19 chore(url): move edge case paths to /argo-workflows/ (#5730)
 * 2b874094 fix(executor): Remove unnecessary check on resource group (#5723)
 * e387b41f docs: Added clarifying points for multi-namespace usage (#5725)
 * 0ebf6c0a ci: Fix argoexec-dev target (#5714)
 * dba2c044 fix: Only save memoization cache when node succeeded (#5711)
 * 8e9e6d67 fix(controller): Fix cron timezone support. Fixes #5653 (#5712)
 * 0a6f2fc3 fix(ui): Fix `showWorkflows` button. Fixes #5645 (#5693)
 * f9635563 fix(ui): Fix YAML/JSON toggle. Fixes #5690 (#5694)
 * b267e3cf fix(cli): Validate cron on update. Fixes #5691 (#5692)
 * 9a872de1 fix(executor): Ignore not existing metadata. Fixes #5656 (#5695)
 * e93ee4fe ci: Separate `make wait` (#5696)
 * 91c08cdd fix(executor): More logs for PNS sidecar termination. #5627 (#5683)
 * f6be5691 fix(controller): Correct bug for repository ref without default key. Fixes #5646 (#5660)
 * 5a397226 build: Specify Dockerfile version (#5689)
 * e3d1d1e8 feat(controller): Allow to disable leader election (#5638) (#5648)
 * 3c8f5617 docs: Revert "docs: Update link to the new CNCF Slack channel (#5681)" (#5684)
 * cad916ef docs(tls): 3.0 defaults to tls enabled (#5686)
 * 86073914 feat(cli): Add offline linting (#5569)
 * a0185236 feat(ui): Support expression evaluation in links (#5666)
 * 24ac7252 fix(executor): Correctly surface error when resource is deleted during status checking (#5675)
 * eab122bb docs: Update link to the new CNCF Slack channel (#5681)
 * 3fab1e5d docs(cron): add dst description (#5679)
 * 1d367ddf fix(ui): strip inner quotes from argoToken (#5677)
 * bf5d7bfa fix: Increase Name width to 3 and decrease NameSpace width to 1 (#5678)
 * 71dfc797 feat(ui): support any yaml reference in link (#5667)
 * fce82bfd ci: Fix test-cli: STATIC_FILES=false (#5673)
 * e80177a6 chore: Update link to k8s API conventions (#5674)
 * ec3b82d9 fix: git clone on non-default branch fails (Fixes #5629) (#5630)
 * d5e492c2 fix(executor):Failure node failed to get archived log (#5671)
 * b7d69053 fix(artifacts): only retry on transient S3 errors (#5579)
 * defbd600 fix: Default ARGO_SECURE=true. Fixes #5607 (#5626)
 * 9c942d59 docs: manifests for SSO using ArgoCD Dex, to be used with Kustomize (#5647)
 * 46ec3028 fix: Make task/step name extractor robust (#5672)
 * ded95bc3 chore: Correctly log version string and remove unnecessary error check (#5664)
 * 54f4c262 ci: Try to make CI more robust (#5633)
 * cc7e310c chore: Support substitute global variable in Spec level elements (#5565)
 * 88917cbd fix: Surface error during wait timeout for OSS artifact driver API calls (#5601)
 * ec4c662c docs: Document SSO expiry option (#5552)
 * b76fac75 fix(ui): Fix editor. Fixes #5613 Fixes #5617 (#5620)
 * 9d175cf9 fix(ui): various ui fixes (#5606)
 * b4ce78bb feat: Identifiable user agents in various Argo commands (#5624)
 * 13fa6524 docs: server runs over https (#5604)
 * 22a8e93c feat(executor): Support accessing output parameters by PNS executor running as non-root (#5564)
 * 2baae1dc add -o short option for argo cli get command (#5533)
 * 0edd32b5 fix(controller): Workflow hangs indefinitely during ContainerCreating if the Pod or Node unexpectedly dies (#5585)
 * d0a0289e feat(ui): let workflow dag and node info scroll independently (#5603)
 * 2651bd61 fix: Improve error message when missing required fields in resource manifest (#5578)
 * 4f3bbdcb feat: Support security token for OSS artifact driver (#5491)
 * 5f51e6d7 docs: Add badge to @argoproj Twitter handle (#5590)
 * 9b6c8b45 fix: parse username from git url when using SSH key auth (#5156)
 * 7276bc39 fix(controller): Consider nested expanded task in reference (#5594)
 * 79eb50b4 docs: Add ByteDance as user (#5593)
 * c941ef8b docs: Add Robinhood as user (#5589)
 * 1dbae739 build: Add make nuke; speed-up make start (#5583)
 * 4e450e25 fix: Switch InsecureSkipVerify to true (#5575)
 * ed54f158 fix(controller): clean up before insert into argo_archived_workflows_labels (#5568)
 * 2b3655ec fix: Remove invalid label value for last hit timestamp from caches (#5528)
 * 2ba0a436 fix(executor): GODEBUG=x509ignoreCN=0 (#5562)
 * 17ff2c17 docs: Updated README.md (#5560)
 * 3df62791 docs: added Intralinks (#5561)
 * 3c3754f9 fix: Build static files in engineering builds (#5559)
 * 23ccd9cf fix(cli): exit when calling subcommand node without args (#5556)
 * aa049485 fix: Reference new argo-workflows url in in-app links (#5553)
 * 26ca3fab docs: Add Zhihu as user (#5555)
 * 20f00470 fix(server): Disable CN check (Go 15 does not support). Fixes #5539 (#5550)
 * 872897ff fix: allow mountPaths with traling slash (#5521)
 * 4c3b0ac5 fix(controller): Enable metrics server on stand-by  controller (#5540)
 * eb6f3a14 chore: Lint master (#5527)
 * a443c53d docs: Add instruction on configuring env vars for controller and executor (#5524)
 * 76b6a0ef feat(controller): Add last hit timestamp to memoization caches (#5487)
 * a61d84cc fix: Default to insecure mode when no certs are present (#5511)
 * 4a1caca1 fix: add softonic as user (#5522)
 * dcc51665 docs: Add demo link to README.md (#5512)
 * bbdf651b fix: Spelling Mistake (#5507)
 * b8af3411 fix: avoid short names in deployment manifests (#5475)
 * 24ccdf40 docs: Conditionals example improved with complex syntax (#5467)
 * d964fe44 fix(controller): Use node.Name instead of node.DisplayName for onExit nodes (#5486)
 * 80cea6a3 fix(ui): Correct Argo Events swagger (#5490)
 * 865b1fe8 fix(executor): Always poll for Docker injected sidecars. Resolves #5448 (#5479)
 * c13755b1 fix: avoid short names in Dockerfiles (#5474)
 * 95bd1ccb ci: Set E2E_TIMEOUT=1m in CI (#5499)
 * beb0f26b fix: Add logging to aid troubleshooting (#5501)
 * 30659416 fix: Run controller as un-privileged (#5460)
 * c8645fc4 chore: Handling panic in go routines (#5489)
 * 2a099f8a fix: certs in non-root (#5476)
 * d246bcf4 docs: add limitation about windows versions (#5462)
 * 4eb351cb fix(ui): Multiple UI fixes (#5498)
 * dfe6ceb4 fix(controller): Fix workflows with retryStrategy left Running after completion (#5497)
 * be44ce9d test: Make cron test suite robust (#5453)
 * 93901907 ci: Add codecov.yaml to silence red-herring errors (#5482)
 * ea26a964 fix(cli): Linting improvements (#5224)
 * 513756eb fix(controller): Only set global parameters after workflow validation succeeded to avoid panics (#5477)
 * 5bd7ce81 docs: fixed the typo, The submit ... doesn't make sense (#5470)
 * d10696d3 chore: Remove unused work queue in Cron controller (#5461)
 * 9a1c046e fix(controller): Enhance output capture (#5450)
 * 46aaa700 feat(server): Disable Basic auth if server is not running in client mode (#5401)
 * e638981b fix(controller): Add permissions to create/update configmaps for memoization in quick-start manifests (#5447)
 * b01ca3a1 fix(controller): Fix the issue of {{retries}} in PodSpecPatch not being updated (#5389)
 * 72ee1cce fix: Set daemoned nodes to Succeeded when boudary ends (#5421)
 * d9f20100 fix(executor): Ignore non-running Docker kill errors (#5451)
 * 7e4e1b78 feat: Template defaults (#5410)
 * 3b129a8f ci: More e2e test work (#5458)
 * 440a6897 fix: Fix getStepOrDAGTaskName (#5454)
 * 8d200618 fix: Various UI fixes (#5449)
 * fc8ecc45 build: `make start` tails logs (#5416)
 * 2371a6d3 fix(executor): PNS support artifacts for short-running containers (#5427)
 * 07ef0e6b fix(test): TestWorkflowTemplateRefWithShutdownAndSuspend flakiness (#5441)
 * c16a471c fix(cli): Only append parse result when not nil to avoid panic (#5424)
 * 8f03970b fix(ui): Fix link button. Fixes #5429 (#5430)
 * f4432043 fix(executor): Surface error when wait container fails to establish pod watch (#5372)
 * d7178657 feat(executor): Move exit code capture to controller. See #5251 (#5328)
 * 525ddecb docs: add documentation about network security (#5420)
 * 04f3a957 fix(test): Fix TestWorkflowTemplateRefWithShutdownAndSuspend flakyness (#5418)
 * ed957dd9 feat(executor): Switch to use SDK and poll-based resource status checking (#5364)
 * d3eeddb1 feat(executor) Add injected sidecar support to Emissary (#5383)
 * d3079b31 docs: Document releases. Closes #5379 (#5402)
 * 189b6a8e fix: Do not allow cron workflow names with more than 52 chars (#5407)
 * 8e137582 feat(executor): Reduce poll time 3s to 1s for PNS and Emissary executors (#5386)
 * 4c4cbd2e docs: add polarpoint.io (#5387)
 * 26eb8a95 docs: Update code of conduct (#5404)
 * b24aaeaf feat: Allow transient errors in StopWorkflow() (#5396)
 * 1ec7ac0f fix(controller): Fixed wrong error message (#5390)
 * 4b7e3513 fix(ui): typo (#5391)
 * 982e5e9d fix(test): TestWorkflowTemplateRefWithShutdownAndSuspend flaky (#5381)
 * 57c05dfa feat(controller): Add failFast flag to DAG and Step templates (#5315)
 * fcb09899 fix(executor): Kill injected sidecars. Fixes #5337 (#5345)
 * 1f7cf1e3 feat: Add helper functions to expr when parsing metadata. Fixes #5351 (#5374)
 * d828717c fix(controller): Fix `podSpecPatch` (#5360)
 * 2d331f3a fix: Fix S3 file loading (#5353)
 * 9faae18a fix(executor): Make docker executor more robust. (#5363)
 * e0f71f3a fix(executor): Fix resource patch when not providing flags. Fixes #5310 (#5311)
 * 94e155b0 fix(controller): Correctly log pods/exec call (#5359)
 * 2320bdc7 docs: Add data doc to website (#5358)
 * 80b5ab9b fix(ui): Fix container-set log viewing in UI (#5348)
 * bde9f217 fix: More Makefile fixes (#5347)
 * 36f672fc test: Make tests more robust, again (#5346)
 * 93e4d4fe docs: Add @terrytangyuan as reviewer (#5191)
 * a7903879 docs: copy information about what is argo to the docs landing page (#5336)
 * 849a5f9a fix: Ensure release images are 'clean' (#5344)
 * 23b8c031 fix: Ensure DEV_BRANCH is correct (#5343)
 * ad7653b4 ci: Fix argoexec build (#5335)
 * 814944cb ci: Github actions refactor (#5280)
 * ba949c3a fix(executor): Fix container set bugs (#5317)
 * 9d2e9615 feat: Support structured JSON logging for controller, executor, and server (#5158)
 * 7fc1f2f2 fix(test): Flaky TestWorkflowShutdownStrategy  (#5331)
 * 3dce211c fix: Only retry on transient errors for OSS artifact driver (#5322)
 * 8309fd83 fix: Minor UI fixes (#5325)
 * 67f8ca27 fix: Disallow object names with more than 63 chars (#5324)
 * b048875d fix(executor): Delegate PNS wait to K8SAPI executor. (#5307)
 * a5d1accf fix(controller): shutdownstrategy on running workflow (#5289)
 * 112378fc fix: Backward compatible workflowTemplateRef from 2.11.x to  2.12.x (#5314)
 * 103bf2bc feat(executor): Configurable retry backoff settings for workflow executor (#5309)
 * 2e857f09 fix: Makefile target (#5313)
 * 1c6775a0 feat: Track nodeView tab in URL (#5300)
 * b4cb24e4 test: Fix tests (#5302)
 * dc5bb12e fix: Use ScopedLocalStorage instead of direct localStorage (#5301)
 * e396ea0e build: Decrease `make codegen` time by (at least) 4 min (#5284)
 * a31fd445 feat: Improve OSS artifact driver usability when load/save directories (#5293)
 * 757e0be1 fix(executor): Enhance PNS executor. Resolves #5251 (#5296)
 * 78ec644c feat: Conditional Artifacts and Parameters (#4987)
 * 1a8ce1f1 fix(executor): Fix emissary resource template bug (#5295)
 * 288ab927 docs: Add Onepanel to projects (#5287)
 * 8729587e feat(controller): Container set template. Closes #2551 (#5099)
 * 21a026b5 chore: Remove uses of deprecated fields in docs and manifests (#5200)
 * e56da57a fix: Use bucket.ListObjects() for OSS ListObjects() implementation (#5283)
 * eaf25ac3 test: Make e2e more robust (#5277)
 * b6961ce6 fix: Fixes around archiving workflows (#5278)
 * ab68ea4c fix: Correctly log sub-resource Kubernetes API requests (#5276)
 * 66fa8da0 feat: Support ListObjects() for Alibaba OSS artifact driver (#5261)
 * b062cbf0 fix: Fix swapped artifact repository key and ref in error message (#5272)
 * 69c40c09 fix(executor): Fix concurrency error in PNS executor. Fixes #5250 (#5258)
 * 9b538d92 fix(executor): Fix docker "created" issue. Fixes #5252 (#5249)
 * 38b0f551 ci: Enable emissary on CI :-o (#5259)
 * 07283cda fix(controller): Take labels change into account in SignificantPodChange() (#5253)
 * c4bcabd7 fix(controller): Work-around Golang bug. Fixes #5192 (#5230)
 * 14542098 test: Rename 'whitelist' to 'allowed' (#5246)
 * b59c8ebd chore: Pass cron schedule to workflow (#5138)
 * e6fa41a1 feat(controller): Expression template tags. Resolves #4548 & #1293 (#5115)
 * 54b04c64 docs: Add example on using label selector in pod GC (#5237)
 * af5abc8e docs: Add link to community calendar (#5233)
 * bd4b46cd feat(controller): Allow to modify time related configurations in leader election (#5234)
 * cb9676e8 feat(controller): Reused existing workflow informer. Resolves #5202 (#5204)
 * e5f3dc12 chore: Remove un-used code (#5232)
 * d7dc48c1 fix(controller): Leader lease shared name improvments (#5218)
 * 2d2fba30 fix(server): Enable HTTPS probe for TLS by default. See #5205 (#5228)
 * fb19af1c fix: Flakey TestDataTransformationArtifactRepositoryRef (#5226)
 * 6412bc68 fix: Do not display pagination warning when there is no pagination (#5221)
 * 0c226ca4 feat: Support for data sourcing and transformation with `data` template (#4958)
 * 7a91ade8 chore(server): Enable TLS by default. Resolves #5205 (#5212)
 * 01d31023 chore(server)!: Required authentication by default. Resolves #5206 (#5211)
 * 694690b0 fix: Checkbox is not clickable (#5213)
 * f0e8df07 fix(controller): Leader Lease Shared Name (#5214)
 * d83cf219 chore: Tidy up test/verification (#5209)
 * 47ac3237 fix(controller): Support emissary on Windows (#5203)
 * 8acdb1ba fix(controller): More emissary minor bugs (#5193)
 * e282fcc8 docs: workflow-controller-configmap.md entry about argo-server. (#5195)
 * 48811117 feat(cli): Add cost optimization nudges for Argo CLI (#5168)
 * 4635d352 docs: Document the new OnTransientError retry policy (#5196)
 * 26ce0c09 fix: Ensure whitespaces is allowed between name and bracket (#5176)
 * 2abf08eb fix: Consder templateRef when filtering by tag (#5190)
 * 23415b2c fix(executor): Fix emissary bugs (#5187)
 * f5dcd1bd fix: Propagate URL changes to react state (#5188)
 * e5a5f039 fix(controller): Fix timezone support. Fixes #5181  (#5182)
 * 199016a6 feat(server): Enforce TLS >= v1.2 (#5172)
 * 8a8759f3 test: Add additional resource template test (#5166)
 * a46ff824 docs: document the WorkflowTemplate workflowMetadata feature (#5177)
 * 71bf06c3 docs: typo (#5183)
 * 7cee66c8 docs: Grafana dashboard link added to metrics.md (#5165)
 * ab361667 feat(controller) Emissary executor.  (#4925)

### Contributors

 * AIKAWA
 * Alex Collins
 * BOOK
 * Bogdan Luput
 * Brandon
 * Caue Augusto dos Santos
 * Christophe Blin
 * Dan Garfield
 * Iven
 * Jesse Suen
 * Jiaxin Shan
 * Kevin Hwang
 * Kishore Chitrapu
 * Luciano Sá
 * Markus Lippert
 * Michael Crenshaw
 * Michael Ruoss
 * Michael Weibel
 * Nicolas Michel
 * Nicoló Lino
 * Niklas Hansson
 * Peixuan Ding
 * Pruthvi Papasani
 * Radolumbo
 * Rand Xie
 * Reijer Copier
 * Riccardo Piccoli
 * Roi Kramer
 * Rush Tehrani
 * Saravanan Balasubramanian
 * Saïfane FARFAR
 * Shoubhik Bose
 * Simon Behar
 * Stephan van Maris
 * Tianchu Zhao
 * Tim Collins
 * Vivek Kumar
 * Vlad Losev
 * Vladimir Ivanov
 * Wen-Chih (Ryan) Lo
 * Yuan Tang
 * Zach Aller
 * Zhong Dai
 * alexey
 * descrepes
 * dherman
 * dinever
 * jsato8094
 * kennytrytek
 * markterm
 * sa-
 * surj-bains
 * tczhao
 * tobisinghania
 * uucloud
 * wanglong001

## v3.0.3 (2021-05-11)

 * 02071057 test: make lint + fix functional tests
 * e450ea7f Update manifests to v3.0.3
 * 80142b12 fix(controller): Error template ref exit handlers. Fixes #5835 (#5837)
 * 8a4a3729 fix(executor): Enable PNS executor to better kill sidecars. Fixes #5779 (#5794)
 * cb8a5479 feat(controller): Add config for potential CPU hogs (#5853)
 * 702bfb24 5739 (#5797)
 * a4c246b2 fix(ui): dont show cluster workflows in namespaced mode. Closes #5841 (#5846)
 * abc0fdf5 chore: `make lint`
 * 4afbcca9 test: Change CLI tests to insecure
 * a8a784d0 ci: Do not run emissary test
 * 4b01cd36 build: Correct image tag in `make codegen`
 * 95a7dec1 ci: fix argexec build
 * 910f552d fix: certs in non-root (#5476)
 * f6493ac3 fix(executor): Fix artifactory saving files. Fixes #5733 (#5775)
 * 6c16cec6 fix(controller): Enable metrics server on stand-by  controller (#5540)
 * b6d70347 feat(controller): Allow to disable leader election (#5638) (#5648)
 * 0ae8061c fix: Node set updating global output parameter updates global. #5699 (#5700)
 * 0d3ad801 fix: Reset workflow started time to current when retrying workflow. Fixes #5796 (#5798)
 * e67cb424 fix: change log level to warn level (#5790)
 * cfd0fad0 fix(ui): Remove the ability to change namespaces via the UI in Managed Namespace Mode. Closes #5577
 * d2f53eae fix(ui): Fix greediness in regex for auth token replacement (#5746)

### Contributors

 * Alex Collins
 * Michael Ruoss
 * Radolumbo
 * Saravanan Balasubramanian
 * Shoubhik Bose
 * Wen-Chih (Ryan) Lo
 * Yuan Tang
 * alexey
 * markterm
 * tobisinghania

## v3.0.2 (2021-04-20)

 * 38fff9c0 Update manifests to v3.0.2
 * a43caa57 fix binary build
 * ca848998 fix: Build argosay binary if it doesn't exist
 * 9492e12b fix(executor): More logs for PNS sidecar termination. #5627 (#5683)
 * 239991f7 build: Specify Dockerfile version (#5689)
 * c8c7ce3b fix: Only save memoization cache when node succeeded (#5711)
 * 1ba1d61f fix(controller): Fix cron timezone support. Fixes #5653 (#5712)
 * 408d31a5 fix(ui): Fix `showWorkflows` button. Fixes #5645 (#5693)
 * b7b4b3f7 fix(ui): Fix YAML/JSON toggle. Fixes #5690 (#5694)
 * 279b78b4 fix(cli): Validate cron on update. Fixes #5691 (#5692)
 * f7200402 fix(executor): Ignore not existing metadata. Fixes #5656 (#5695)
 * 193f8751 fix(controller): Correct bug for repository ref without default key. Fixes #5646 (#5660)
 * e2081330 fix(ui): strip inner quotes from argoToken (#5677)
 * 493e5d65 fix: git clone on non-default branch fails (Fixes #5629) (#5630)
 * f8ab29b4 fix: Default ARGO_SECURE=true. Fixes #5607 (#5626)
 * 49a4926d fix: Make task/step name extractor robust (#5672)
 * 0cea6125 fix: Surface error during wait timeout for OSS artifact driver API calls (#5601)
 * 026c1279 fix(ui): Fix editor. Fixes #5613 Fixes #5617 (#5620)
 * dafa9832 fix(ui): various ui fixes (#5606)
 * c17e72e8 fix(controller): Workflow hangs indefinitely during ContainerCreating if the Pod or Node unexpectedly dies (#5585)
 * 3472b4f5 feat(ui): let workflow dag and node info scroll independently (#5603)
 * f6c47e4b fix: parse username from git url when using SSH key auth (#5156)
 * 5bc28dee fix(controller): Consider nested expanded task in reference (#5594)

### Contributors

 * Alex Collins
 * Iven
 * Michael Ruoss
 * Saravanan Balasubramanian
 * Simon Behar
 * Vladimir Ivanov
 * Yuan Tang
 * kennytrytek
 * tczhao

## v3.0.1 (2021-04-01)

 * a8c7d54c Update manifests to v3.0.1
 * 65250dd6 fix: Switch InsecureSkipVerify to true (#5575)
 * 0de125ac fix(controller): clean up before insert into argo_archived_workflows_labels (#5568)
 * f0578945 fix(executor): GODEBUG=x509ignoreCN=0 (#5562)
 * bda3af2e fix: Reference new argo-workflows url in in-app links (#5553)
 * 46628c88 Update manifests to v3.0.0
 * 3089d8a2 fix: Add 'ToBeFailed'
 * 5771c60e fix: Default to insecure mode when no certs are present (#5511)
 * c77f1ece fix(controller): Use node.Name instead of node.DisplayName for onExit nodes (#5486)
 * 0e91e5f1 fix(ui): Correct Argo Events swagger (#5490)
 * aa07d93a fix(executor): Always poll for Docker injected sidecars. Resolves #5448 (#5479)
 * 7f908af1 chore: Handling panic in go routines (#5489)

### Contributors

 * Alex Collins
 * BOOK
 * Saravanan Balasubramanian
 * Simon Behar
 * Tim Collins

## v3.0.0-rc9 (2021-03-23)

 * 02b87aa7 Update manifests to v3.0.0-rc9
 * 0f5a9ad1 fix(ui): Multiple UI fixes (#5498)
 * ac5f1714 fix(controller): Fix workflows with retryStrategy left Running after completion (#5497)
 * 3e81ed4c fix(controller): Only set global parameters after workflow validation succeeded to avoid panics (#5477)
 * 6d70f9cc fix: Set daemoned nodes to Succeeded when boudary ends (#5421)
 * de31db41 fix(executor): Ignore non-running Docker kill errors (#5451)
 * f6ada612 fix: Fix getStepOrDAGTaskName (#5454)
 * 586a04c1 fix: Various UI fixes (#5449)
 * 78939009 fix(executor): PNS support artifacts for short-running containers (#5427)
 * 8f0235a0 fix(test): TestWorkflowTemplateRefWithShutdownAndSuspend flakiness (#5441)
 * 6f1027a1 fix(cli): Only append parse result when not nil to avoid panic (#5424)
 * 5b871adb fix(ui): Fix link button. Fixes #5429 (#5430)
 * 41eaa357 fix(executor): Surface error when wait container fails to establish pod watch (#5372)
 * f55d41ac fix(test): Fix TestWorkflowTemplateRefWithShutdownAndSuspend flakyness (#5418)

### Contributors

 * Alex Collins
 * Saravanan Balasubramanian
 * Simon Behar
 * Yuan Tang

## v3.0.0-rc8 (2021-03-17)

 * ff504001 Update manifests to v3.0.0-rc8
 * 50fe7970 fix(server): Enable HTTPS probe for TLS by default. See #5205 (#5228)

### Contributors

 * Alex Collins
 * Simon Behar

## v3.0.0-rc7 (2021-03-16)

 * 8049ed82 Update manifests to v3.0.0-rc7
 * fb676b41 test: remove PNS
 * 870dc46e test: remove emissary - not supported in v3.0
 * 6dbb94a5 chore: lint
 * 0c969842 chore: Skip TestOutputArtifactS3BucketCreationEnabled - this feat is not in v3.0
 * 11825c51 chore: Fix post-merge conflicts
 * c2c44102 fix(executor): Kill injected sidecars. Fixes #5337 (#5345)
 * c9d7bfc6 chore(server): Enable TLS by default. Resolves #5205 (#5212)
 * 701623f7 fix(executor): Fix resource patch when not providing flags. Fixes #5310 (#5311)
 * ae34e4d7 fix: Do not allow cron workflow names with more than 52 chars (#5407)
 * 4468c26f fix(test): TestWorkflowTemplateRefWithShutdownAndSuspend flaky (#5381)
 * 1ce011e4 fix(controller): Fix `podSpecPatch` (#5360)
 * a4dacde8 fix: Fix S3 file loading (#5353)
 * 452b3708 fix(executor): Make docker executor more robust. (#5363)
 * 83fc1c38 fix(test): Flaky TestWorkflowShutdownStrategy  (#5331)

### Contributors

 * Alex Collins
 * Saravanan Balasubramanian
 * Simon Behar
 * Yuan Tang

## v3.0.0-rc6 (2021-03-09)

 * ab611694 Update manifests to v3.0.0-rc6
 * 309fd111 fix: More Makefile fixes (#5347)
 * f7734050 fix: Ensure release images are 'clean' (#5344)
 * ce915f57 fix: Ensure DEV_BRANCH is correct (#5343)

### Contributors

 * Simon Behar

## v3.0.0-rc5 (2021-03-08)

 * 3b422776 Update manifests to v3.0.0-rc5
 * 145847d7 cherry-picked fix(controller): shutdownstrategy on running workflow (#5289)
 * 29723f49 codegen
 * ec130465 fix: Makefile target (#5313)
 * 8c69f4fa add json/fix.go
 * 4233d0b7 fix: Minor UI fixes (#5325)
 * 87b62c08 fix: Disallow object names with more than 63 chars (#5324)
 * e16bd95b fix(executor): Delegate PNS wait to K8SAPI executor. (#5307)
 * 62956be0 fix: Backward compatible workflowTemplateRef from 2.11.x to  2.12.x (#5314)
 * 95dd7f4b feat: Track nodeView tab in URL (#5300)
 * a3c12df5 fix: Use ScopedLocalStorage instead of direct localStorage (#5301)
 * 301aacb9 build: Decrease `make codegen` time by (at least) 4 min (#5284)
 * f368c32f fix(executor): Enhance PNS executor. Resolves #5251 (#5296)
 * 4b2fd9f7 fix: Fixes around archiving workflows (#5278)
 * afe2cdb6 fix: Correctly log sub-resource Kubernetes API requests (#5276)
 * 27956b71 fix(executor): Fix concurrency error in PNS executor. Fixes #5250 (#5258)
 * 0a8b8f71 fix(executor): Fix docker "created" issue. Fixes #5252 (#5249)
 * 71d1130d fix(controller): Take labels change into account in SignificantPodChange() (#5253)
 * 39adcd5f fix(controller): Work-around Golang bug. Fixes #5192 (#5230)
 * a77d91e5 chore: Pass cron schedule to workflow (#5138)

### Contributors

 * Alex Collins
 * Saravanan Balasubramanian
 * Simon Behar
 * Yuan Tang

## v3.0.0-rc4 (2021-03-02)

 * ae5587e9 Update manifests to v3.0.0-rc4
 * a7ecfc08 feat(controller): Allow to modify time related configurations in leader election (#5234)
 * 9b9043a6 feat(controller): Reused existing workflow informer. Resolves #5202 (#5204)
 * 4e9f6350 fix(controller): Leader lease shared name improvments (#5218)
 * 94211393 fix: Do not display pagination warning when there is no pagination (#5221)
 * 0891dc2f fix: Checkbox is not clickable (#5213)
 * 9a1971ef fix(controller): Leader Lease Shared Name (#5214)
 * 339bf4e8 fix: Ensure whitespaces is allowed between name and bracket (#5176)
 * df032f62 fix: Consder templateRef when filtering by tag (#5190)
 * d9d831ca fix: Propagate URL changes to react state (#5188)
 * db657758 fix(controller): Fix timezone support. Fixes #5181  (#5182)

### Contributors

 * Alex Collins
 * Simon Behar
 * Yuan Tang
 * Zach Aller

## v3.0.0-rc3 (2021-02-23)

 * c0c364c2 Update manifests to v3.0.0-rc3
 * 57ed7d3d Merge branch 'master' into release-3.0
 * 8af35954 chore: Add missing CLI short docstrings (#5175)
 * a9c42006 fix: Specify where in YAML validation error occurred (#5174)
 * 4b78a7ee fix: Fix node filters in UI (#5162)
 * d9fb0c30 feat(controller): Support pod GC strategy based on label selector on pods (#5090)
 * 6618f475 docs: Add DLR to USERS.md (#5157)
 * 91528cc8 fix(executor): Do not make unneeded `get pod` when no sidecars (#5161)
 * bec80c86 fix: Better message formating for nodes (#5160)
 * d33b5cc0 fix: send periodic keepalive packets on eventstream connections (#5101)
 * 0f9b22b6 fix: Append the error message prior to offloading node status (#5043)
 * b54f13d6 build: Fix build of `argocli` (#5150)
 * 4611a167 feat: Support automatically create OSS bucket if not exists (#5133)
 * 687479fa feat(controller): Use different container runtime executors for each workflow. Close #4254 (#4998)
 * 590df1dc feat: Add `argo submit --verify` hidden flag. Closes #5136 (#5141)
 * 377c5f84 feat: added lint from stdin (#5095)
 * 633da258 feat(server): Write an audit log entry for SSO users (#5145)
 * 2ab02d95 fix: Revert the unwanted change in example  (#5139)
 * 1c792129 fix: Multiple UI fixes (#5140)
 * 46538d95 feat(ui): Surface result and exit-code outputs (#5137)
 * ccf4c612 test: Enhanced e2e tests (#5128)
 * 5c5c9f1c build: Fix path to golangci-lint binary (#5131)
 * 5e64ec40 feat: Build dev-* branches as engineering builds (#5129)
 * 4aa9847e fix(ui): add a tooltip for commonly truncated fields in the events pane (#5062)
 * b1535e53 feat: Support pgzip as an alternative (de)compression implementation (#5108)
 * fb3cab21 docs: Update workflow-controller-configmap workflowRestrictions example doc (#5109)
 * e2c360d2 chore: Add SHA256 checksums to release (#5122)

### Contributors

 * Alex Collins
 * Florian
 * Ken Kaizu
 * Roi Kramer
 * Saravanan Balasubramanian
 * Simon Behar
 * Yuan Tang
 * dherman

## v3.0.0-rc2 (2021-02-16)

 * ea3439c9 Update manifests to v3.0.0-rc2
 * 97471672 Merge branch 'master' into release-3.0
 * b0685bdd fix(executor): Fix S3 policy based auth. Fixes #5110 (#5111)
 * 4b9b658a Merge branch 'master' into release-3.0
 * fcf4e992 fix: Invalid OpenAPI Spec (Issue 4817) (#4831)
 * a50ddb20 chore: More opinionated linting (#5072)
 * 19b22f25 feat: Add checker to ensure that env variable doc is up to date (#5091)
 * 210080a0 feat(controller): Logs Kubernetes API requests (#5084)
 * 2f7c9087 build: Fix path to openapi-gen binary (#5089)
 * 2ff4db11 feat(executor): Minimize the number of Kubernetes API requests made by executors (#4954)
 * 68979f6e fix: Do not create pods under shutdown strategy (#5055)
 * 75d09b0f fix: Synchronization lock handling in Step/DAG Template level (#5081)
 * cda5dc2e docs: Add document for environment variables (#5080)
 * 57b38282 docs: Add Jungle to USERS.md (#5096)
 * 3b7e373e feat(ui): Display pretty cron schedule (#5088)
 * 1a0889cf fix: Revert "fix(controller): keep special characters in json string when … … 19da392 …use withItems (#4814)" (#5076)
 * 893e9c9f fix: Prefer to break labels by '-' in UI (#5083)
 * 75f08e2e docs: Add community video to README (#5087)
 * 77b23098 fix(controller): Fix creator dashes (#5082)
 * f461b040 feat(controller): Add podMetadata field to workflow spec. Resolves #4985 (#5031)
 * 3b63e7d8 feat(controller): Add retry policy to support retry only on transient errors (#4999)
 * 1578c618 chore: Update links to argo-workflows documentation (#5070)
 * 34f29c8e docs: Add Vispera to USERS.md (#5047)
 * b18b9920 build: Simpler Docker build (#5057)
 * 21e137ba fix(executor): Correct usage of time.Duration. Fixes #5046 (#5049)
 * 19a34b1f feat(executor): Add user agent to workflow executor (#5014)
 * f31e0c6f chore!: Remove deprecated fields (#5035)
 * f59d4622 fix: Invalid URL for API Docs (#5063)
 * daf1a71b feat: Allow to specify grace period for pod GC (#5033)
 * 65fb530e chore: Move paths to /argo-workflows/ (#5059)
 * 26f48a9d fix: Use React state to avoid new page load in Workflow view (#5058)
 * a0669b5d fix: Don't allow graph container to have its own scroll (#5056)

### Contributors

 * Alex Collins
 * Dylan Hellems
 * Kaan C. Fidan
 * Nelson Rodrigues
 * Saravanan Balasubramanian
 * Simon Behar
 * Viktor Farcic
 * Yuan Tang
 * drannenberg
 * kennytrytek

## v3.0.0-rc1 (2021-02-08)


### Contributors


## v3.0.0 (2021-03-29)

 * 46628c88 Update manifests to v3.0.0
 * 3089d8a2 fix: Add 'ToBeFailed'
 * 5771c60e fix: Default to insecure mode when no certs are present (#5511)
 * c77f1ece fix(controller): Use node.Name instead of node.DisplayName for onExit nodes (#5486)
 * 0e91e5f1 fix(ui): Correct Argo Events swagger (#5490)
 * aa07d93a fix(executor): Always poll for Docker injected sidecars. Resolves #5448 (#5479)
 * 7f908af1 chore: Handling panic in go routines (#5489)
 * 02b87aa7 Update manifests to v3.0.0-rc9
 * 0f5a9ad1 fix(ui): Multiple UI fixes (#5498)
 * ac5f1714 fix(controller): Fix workflows with retryStrategy left Running after completion (#5497)
 * 3e81ed4c fix(controller): Only set global parameters after workflow validation succeeded to avoid panics (#5477)
 * 6d70f9cc fix: Set daemoned nodes to Succeeded when boudary ends (#5421)
 * de31db41 fix(executor): Ignore non-running Docker kill errors (#5451)
 * f6ada612 fix: Fix getStepOrDAGTaskName (#5454)
 * 586a04c1 fix: Various UI fixes (#5449)
 * 78939009 fix(executor): PNS support artifacts for short-running containers (#5427)
 * 8f0235a0 fix(test): TestWorkflowTemplateRefWithShutdownAndSuspend flakiness (#5441)
 * 6f1027a1 fix(cli): Only append parse result when not nil to avoid panic (#5424)
 * 5b871adb fix(ui): Fix link button. Fixes #5429 (#5430)
 * 41eaa357 fix(executor): Surface error when wait container fails to establish pod watch (#5372)
 * f55d41ac fix(test): Fix TestWorkflowTemplateRefWithShutdownAndSuspend flakyness (#5418)
 * ff504001 Update manifests to v3.0.0-rc8
 * 50fe7970 fix(server): Enable HTTPS probe for TLS by default. See #5205 (#5228)
 * 8049ed82 Update manifests to v3.0.0-rc7
 * fb676b41 test: remove PNS
 * 870dc46e test: remove emissary - not supported in v3.0
 * 6dbb94a5 chore: lint
 * 0c969842 chore: Skip TestOutputArtifactS3BucketCreationEnabled - this feat is not in v3.0
 * 11825c51 chore: Fix post-merge conflicts
 * c2c44102 fix(executor): Kill injected sidecars. Fixes #5337 (#5345)
 * c9d7bfc6 chore(server): Enable TLS by default. Resolves #5205 (#5212)
 * 701623f7 fix(executor): Fix resource patch when not providing flags. Fixes #5310 (#5311)
 * ae34e4d7 fix: Do not allow cron workflow names with more than 52 chars (#5407)
 * 4468c26f fix(test): TestWorkflowTemplateRefWithShutdownAndSuspend flaky (#5381)
 * 1ce011e4 fix(controller): Fix `podSpecPatch` (#5360)
 * a4dacde8 fix: Fix S3 file loading (#5353)
 * 452b3708 fix(executor): Make docker executor more robust. (#5363)
 * 83fc1c38 fix(test): Flaky TestWorkflowShutdownStrategy  (#5331)
 * ab611694 Update manifests to v3.0.0-rc6
 * 309fd111 fix: More Makefile fixes (#5347)
 * f7734050 fix: Ensure release images are 'clean' (#5344)
 * ce915f57 fix: Ensure DEV_BRANCH is correct (#5343)
 * 3b422776 Update manifests to v3.0.0-rc5
 * 145847d7 cherry-picked fix(controller): shutdownstrategy on running workflow (#5289)
 * 29723f49 codegen
 * ec130465 fix: Makefile target (#5313)
 * 8c69f4fa add json/fix.go
 * 4233d0b7 fix: Minor UI fixes (#5325)
 * 87b62c08 fix: Disallow object names with more than 63 chars (#5324)
 * e16bd95b fix(executor): Delegate PNS wait to K8SAPI executor. (#5307)
 * 62956be0 fix: Backward compatible workflowTemplateRef from 2.11.x to  2.12.x (#5314)
 * 95dd7f4b feat: Track nodeView tab in URL (#5300)
 * a3c12df5 fix: Use ScopedLocalStorage instead of direct localStorage (#5301)
 * 301aacb9 build: Decrease `make codegen` time by (at least) 4 min (#5284)
 * f368c32f fix(executor): Enhance PNS executor. Resolves #5251 (#5296)
 * 4b2fd9f7 fix: Fixes around archiving workflows (#5278)
 * afe2cdb6 fix: Correctly log sub-resource Kubernetes API requests (#5276)
 * 27956b71 fix(executor): Fix concurrency error in PNS executor. Fixes #5250 (#5258)
 * 0a8b8f71 fix(executor): Fix docker "created" issue. Fixes #5252 (#5249)
 * 71d1130d fix(controller): Take labels change into account in SignificantPodChange() (#5253)
 * 39adcd5f fix(controller): Work-around Golang bug. Fixes #5192 (#5230)
 * a77d91e5 chore: Pass cron schedule to workflow (#5138)
 * ae5587e9 Update manifests to v3.0.0-rc4
 * a7ecfc08 feat(controller): Allow to modify time related configurations in leader election (#5234)
 * 9b9043a6 feat(controller): Reused existing workflow informer. Resolves #5202 (#5204)
 * 4e9f6350 fix(controller): Leader lease shared name improvments (#5218)
 * 94211393 fix: Do not display pagination warning when there is no pagination (#5221)
 * 0891dc2f fix: Checkbox is not clickable (#5213)
 * 9a1971ef fix(controller): Leader Lease Shared Name (#5214)
 * 339bf4e8 fix: Ensure whitespaces is allowed between name and bracket (#5176)
 * df032f62 fix: Consder templateRef when filtering by tag (#5190)
 * d9d831ca fix: Propagate URL changes to react state (#5188)
 * db657758 fix(controller): Fix timezone support. Fixes #5181  (#5182)
 * c0c364c2 Update manifests to v3.0.0-rc3
 * 57ed7d3d Merge branch 'master' into release-3.0
 * 8af35954 chore: Add missing CLI short docstrings (#5175)
 * a9c42006 fix: Specify where in YAML validation error occurred (#5174)
 * 4b78a7ee fix: Fix node filters in UI (#5162)
 * d9fb0c30 feat(controller): Support pod GC strategy based on label selector on pods (#5090)
 * 6618f475 docs: Add DLR to USERS.md (#5157)
 * 91528cc8 fix(executor): Do not make unneeded `get pod` when no sidecars (#5161)
 * bec80c86 fix: Better message formating for nodes (#5160)
 * d33b5cc0 fix: send periodic keepalive packets on eventstream connections (#5101)
 * 0f9b22b6 fix: Append the error message prior to offloading node status (#5043)
 * b54f13d6 build: Fix build of `argocli` (#5150)
 * 4611a167 feat: Support automatically create OSS bucket if not exists (#5133)
 * 687479fa feat(controller): Use different container runtime executors for each workflow. Close #4254 (#4998)
 * 590df1dc feat: Add `argo submit --verify` hidden flag. Closes #5136 (#5141)
 * 377c5f84 feat: added lint from stdin (#5095)
 * 633da258 feat(server): Write an audit log entry for SSO users (#5145)
 * 2ab02d95 fix: Revert the unwanted change in example  (#5139)
 * 1c792129 fix: Multiple UI fixes (#5140)
 * 46538d95 feat(ui): Surface result and exit-code outputs (#5137)
 * ccf4c612 test: Enhanced e2e tests (#5128)
 * 5c5c9f1c build: Fix path to golangci-lint binary (#5131)
 * 5e64ec40 feat: Build dev-* branches as engineering builds (#5129)
 * 4aa9847e fix(ui): add a tooltip for commonly truncated fields in the events pane (#5062)
 * b1535e53 feat: Support pgzip as an alternative (de)compression implementation (#5108)
 * fb3cab21 docs: Update workflow-controller-configmap workflowRestrictions example doc (#5109)
 * e2c360d2 chore: Add SHA256 checksums to release (#5122)
 * ea3439c9 Update manifests to v3.0.0-rc2
 * 97471672 Merge branch 'master' into release-3.0
 * b0685bdd fix(executor): Fix S3 policy based auth. Fixes #5110 (#5111)
 * 4b9b658a Merge branch 'master' into release-3.0
 * fcf4e992 fix: Invalid OpenAPI Spec (Issue 4817) (#4831)
 * a50ddb20 chore: More opinionated linting (#5072)
 * 19b22f25 feat: Add checker to ensure that env variable doc is up to date (#5091)
 * 210080a0 feat(controller): Logs Kubernetes API requests (#5084)
 * 2f7c9087 build: Fix path to openapi-gen binary (#5089)
 * 2ff4db11 feat(executor): Minimize the number of Kubernetes API requests made by executors (#4954)
 * 68979f6e fix: Do not create pods under shutdown strategy (#5055)
 * 75d09b0f fix: Synchronization lock handling in Step/DAG Template level (#5081)
 * cda5dc2e docs: Add document for environment variables (#5080)
 * 57b38282 docs: Add Jungle to USERS.md (#5096)
 * 3b7e373e feat(ui): Display pretty cron schedule (#5088)
 * 1a0889cf fix: Revert "fix(controller): keep special characters in json string when … … 19da392 …use withItems (#4814)" (#5076)
 * 893e9c9f fix: Prefer to break labels by '-' in UI (#5083)
 * 75f08e2e docs: Add community video to README (#5087)
 * 77b23098 fix(controller): Fix creator dashes (#5082)
 * f461b040 feat(controller): Add podMetadata field to workflow spec. Resolves #4985 (#5031)
 * 3b63e7d8 feat(controller): Add retry policy to support retry only on transient errors (#4999)
 * 1578c618 chore: Update links to argo-workflows documentation (#5070)
 * 34f29c8e docs: Add Vispera to USERS.md (#5047)
 * b18b9920 build: Simpler Docker build (#5057)
 * 21e137ba fix(executor): Correct usage of time.Duration. Fixes #5046 (#5049)
 * 19a34b1f feat(executor): Add user agent to workflow executor (#5014)
 * f31e0c6f chore!: Remove deprecated fields (#5035)
 * f59d4622 fix: Invalid URL for API Docs (#5063)
 * daf1a71b feat: Allow to specify grace period for pod GC (#5033)
 * 65fb530e chore: Move paths to /argo-workflows/ (#5059)
 * 26f48a9d fix: Use React state to avoid new page load in Workflow view (#5058)
 * a0669b5d fix: Don't allow graph container to have its own scroll (#5056)
 * 9d0be908 Update manifests to v3.0.0-rc1
 * 425173a2 fix(cli): Add insecure-skip-verify for HTTP1. Fixes #5008 (#5015)
 * 3edb55c1 chore: Reuse LookupEnvDurationOr to parse duration from environment variables (#5032)
 * 48b669cc M is demonstrably not less than 1 in the examples (#5021)
 * 5915a216 feat(controller): configurable terminationGracePeriodSeconds (#4940)
 * 5824fc6b Fix golang build (#5039)
 * ef76f729 feat: DAG render options panel float through scrolling (#5036)
 * b4ea47e0 fix: Skip the Workflow not found error in Concurrency policy (#5030)
 * edbe5bc9 fix(ui): Display all node inputs/output in one tab. Resolves #5027 (#5029)
 * c4e8d1cf feat(executor): Log `verb kind statusCode` for executor Kubernetes API requests (#4989)
 * d1abcb05 fix: Unmark daemoned nodes after stopping them (#5005)
 * 38e98f7e Video (#5019)
 * d40c117c docs: Add a note for docker executors regarding outputs.result (#5017)
 * 342caeff fix(ui): Fix event-flow hidden nodes (#5013)
 * 1b77ec80 build: Upgrade Golang to v1.15 (#5009)
 * c232d237 build: Upgrade argoexec base image to debian:10.7-slim (#5010)
 * d5ccc8e0 feat(executor): Upgrade kubectl to v1.19 (#5011)
 * d3b7adb1 test: Delete TestDeletingRunningPod (#5012)
 * 8f5e17ac feat: Set CORS headers (#4990)
 * 99c049bd feat(ui): Node search tool in UI Workflow viewer (#5000)
 * 5047f073 fix: Fail DAG templates with variables with invalid dependencies (#4992)
 * a730b4f4 chore: Set Go mod to v3 (#4978)
 * ccd669e4 fix: Coalesce UI filtering menus (#4972)
 * 7710a2ca docs: Add Couler, SQLFlow, and Kubeflow to USERS.md (#4964)
 * ce508c89 feat: Configurable retry backoff settings when retrying API calls (#4979)
 * 61e8b963 test: Fix TestDeletingRunningPod (#4996)
 * 44a4f7e1 fix(controller): Revert prepending ExecutorScriptSourcePath which brought a breaking change in args handling (#4884)
 * b68d63eb fix(controller): Adds PNS_PRIVILEGED, fixed termination bug (#4983)
 * d324b43c fix: Use button in side panel links (#4977)
 * 2672a933 docs: Fix broken docs (#4967)
 * 655c7e25 fix: Surface the underlying error on wait timeout. (#4966)
 * a00aa325 fix: Correct usage of wait.ExponentialBackoff (#4962)
 * faa3363f docs: Specify states to ensure clean state when releaseing (#4963)
 * 64147238 docs: Update running-locally.md (#4961)
 * e00623d6 fix(server): Fix missing logs bug (#4960)
 * 1d8b9549 ci: Run all executors on CI (#4957)
 * eabe9637 feat(server): add ServiceAccount info to api/v1/userinfo and ui user tab (#4944)
 * 15156d19 Added Astraea (#4855)
 * 7404b1f8 fix(controller): report OOM when wait container OOM (#4930)
 * 6166e80c feat: Support retry on transient errors during executor status checking (#4946)
 * f06550d7 docs: Remove miscreant line in argo.md (#4821)
 * 727e27a9 docs: Update security docs (#4743)
 * b6efd00c chore: Remove self-referential replace directive in go.mod (#4959)
 * 2ebcb9f9 docs: Improve Documents Around Inputs (#4900)
 * 6e116e46 feat(crds): Update CRDs to apiextensions.k8s.io/v1 (#4918)
 * 26162532 feat(server): Add Prometheus metrics. Closes #4751 (#4952)
 * 0bffade3 chore: Set Go mod to /v2 (#4916)
 * 7c69898e fix(cli): Allow full node name in node-field-selector (#4913)
 * c7293062 fix(cli): Update the map-reduce example, fix bug. (#4948)
 * e7e51d08 feat: Check the workflow is not being deleted for Synchronization workflow (#4935)
 * 2254ac90 docs: add Devtron labs to USERS.md (#4939)
 * 9d4edaef fix(ui): v3 UI tweaks (#4933)
 * 8f2f4276 build: Faster `make start`, faster smoke jobs (#4926)
 * 2d73d58a fix(ui): fix object-editor text render issue (#4921)
 * 6e961ec9 feat: support K8S json patch (#4908)
 * 957ef677 chore: Introduce `WorkflowPhase` (#4856)
 * f872366f fix(controller): Report reconciliation errors better (#4877)
 * c8215f97 feat(controller)!: Key-only artifacts. Fixes #3184 (#4618)
 * cd7c16b2 fix(ui): objecteditor only runs onChange when values are modified (#4911)
 * 8c353d94 build: Inline Argo Events CRD URLs to reduce build errors (#4902)
 * ee1f8276 fix(ui): Fix workflow refresh bug (#4906)
 * 929cd50e fix: Mutex not being released on step completion (#4847)
 * c1f9280a fix(ui): UI bug fixes (#4895)
 * 25abd1a0 feat: Support specifying the pattern for transient and retryable errors (#4889)
 * 0c5ebbba build: on a dev branch automatically start the UI (#4896)
 * 16f25ba0 Revert "feat(cli): add selector and field-selector option to the stop command. (#4853)"
 * 53f7998e feat(cli): add selector and field-selector option to the stop command. (#4853)
 * 1f13241f fix(workflow-event-bindings): removing unneeded ':' in protocol (#4893)
 * ecbca6ce fix(ui): Show non-pod nodes (#4890)
 * 4a5db1b7 fix(controller): Consider processed retry node in metrics. Fixes #4846 (#4872)
 * dd8c1ba0 feat(controller): optional database migration (#4869)
 * a8e93482 feat(ui): Argo Events API and UI. Fixes #888 (#4470)
 * 17e79e8a fix(controller): make creator label DNS compliant. Fixes #4880 (#4881)
 * 2ff11cc9 fix(controller): Fix node status when daemon pod deleted but its children nodes are still running (#4683)
 * 955a4bb1 fix: Do not error on duplicate workflow creation by cron (#4871)
 * 239272a1 build: sudoless `make codegen -B` (#4866)
 * 622624e8 fix(controller): Add matrix tests for node offload disabled. Resolves #2333 (#4864)
 * f38c9a2d feat: Expose exitCode to step level metrics (#4861)
 * 45c792a5 feat(controller): `k8s_request_total` and `workflow_condition` metrics (#4811)
 * e3320d36 feat: Publish images on Quay.io (#4860)
 * b674aa30 feat: Publish images to Quay.io (#4854)
 * a6301d7c refactor: upgrade kube client version to v0.19.6. Fixes #4425, #4791 (#4810)
 * 6b3ce504 feat: Worker busy and active pod metrics (#4823)
 * 53110b61 fix: Preserve the original slice when removing string (#4835)
 * adfa988f fix(controller): keep special characters in json string when use withItems (#4814)
 * 6e158780 feat(controller): Retry pod creation on API timeout (#4820)
 * 3c33ffb3 build: Fix lint (#4837)
 * 945106e3 chore: Bump OSS Go SDK to v2.1.5 (#4834)
 * 01e6c9d5 feat(controller): Add retry on different host (#4679)
 * 2243d349 fix: Metrics documentation (#4829)
 * 5f8a83a0 chore: add missing phase option to node cli command (#4825)
 * f0a315cf fix(crds): Inline WorkflowSteps schema to generate valid OpenAPI spec (#4828)
 * f037fd2b feat(controller): Adding Eventrecorder on LeaderElection
 * a0024d0d fix(controller): Various v2.12 fixes. Fixes #4798, #4801, #4806 (#4808)
 * ee59d49d fix: Memoize Example (Issue 4626) (#4818)
 * b73bd2b6 feat: Customize workfow metadata from event data (#4783)
 * 4eaae251 chore: Remove redundant "from" in log message (#4813)
 * 7e6c799a fix: load all supported authentication plugins for k8s client-go (#4802)
 * 78b0bffd fix(executor): Do not delete local artifacts after upload. Fixes #4676 (#4697)
 * 764f118c docs: Clarify PNS security (#4789)
 * e86b377f test: Fix TestDeletingRunningPod (#4779)
 * af03a74f refactor(ui): replace node-sass with sass (#4780)
 * 15ec9f5e chore(example): Add watch timeout and print out workflow status message (#4740)
 * 4ac436d5 fix(server): Do not silently ignore sso secret creation error (#4775)
 * 442d367b feat(controller): unix timestamp support on creationTimestamp var (#4763)
 * 9f67b28c feat(controller): Rate-limit workflows. Closes #4718 (#4726)
 * aed25fef Change argo-server crt/key owner (#4750)
 * fbb4e8d4 fix(controller): Support default database port. Fixes #4756 (#4757)
 * 69ce2acf refactor(controller): Enhanced pod clean-up scalability (#4728)
 * 549585f1 test: Improved e2e test robustness (#4758)
 * 9c4d735a feat: Add a minimal prometheus server manifest (#4687)
 * 625e3ce2 fix(ui): Remove unused Heebo files. Fixes #4730 (#4739)
 * 2e278b01 fix(controller): Fixes resource version misuse. Fixes #4714 (#4741)
 * 300db5e6 fix(controller): Requeue when the pod was deleted. Fixes #4719 (#4742)
 * a1f7aedb fix(controller): Fixed workflow stuck with mutex lock (#4744)
 * 1a7ed734 feat(controller): Enhanced TTL controller scalability (#4736)
 * 7437f429 fix(executor): Always check if resource has been deleted in checkResourceState() (#4738)
 * 122c5fd2 fix(executor): Copy main/executor container resources from controller by value instead of reference (#4737)
 * 86fdd74a docs: Update SSO docs to explain how to use the Argo CD Dex instance with Argo Workflows Server (#4729)
 * 440d732d fix(ui): Fix YAML for workflows with storedWorkflowTemplateSpec. Fixes #4691 (#4695)
 * ed853eb0 fix: Allow Bearer token in server mode (#4735)
 * 1f421df6 fix(executor): Deal with the pod watch API call timing out (#4734)
 * 724fd80c feat(controller): Pod deletion grace period. Fixes #4719 (#4725)
 * c52cdd19 docs: minor typo fix (#4727)
 * 38026894 feat(controller): Add Prometheus metric: `workflow_ttl_queue` (#4722)
 * 9f5c9102 chore: Update stress test files. (#4721)
 * 55019c6e fix(controller): Fix incorrect main container customization precedence and isResourcesSpecified check (#4681)
 * 625189d8 fix(ui): Fix "Using Your Login". Fixes #4707 (#4708)
 * 433dc5b9 feat(server): Support email for SSO+RBAC. Closes #4612 (#4644)
 * ae0c0bb8 fix(controller): Fixed RBAC on leases (#4715)
 * cd4adda1 fix(controller): Fixed Leader election name (#4709)
 * aec22189 fix(test): Fixed Flaky e2e tests TestSynchronizationWfLevelMutex and TestResourceTemplateStopAndTerminate/ResourceTemplateStop (#4688)
 * ab837753 fix(controller): Fix the RBAC for leader-election (#4706)
 * 1850b1f9 ci: Pin kustomize version (#4704)
 * 06f51422 chore: direct users to ask questions on GitHub discussions (#4701)
 * 9669aa52 fix(controller): Increate default EventSpamBurst in Eventrecorder (#4698)
 * 96a55ce5 feat(controller): HA Leader election support on Workflow-controller (#4622)
 * ad1b6de4 fix: Consider optional artifact arguments (#4672)
 * d9d5f5fb feat(controller): Use deterministic name for cron workflow children (#4638)
 * f47fc222 fix(controller): Only patch status.active in cron workflows when syncing (#4659)
 * 6b68b1f6 test: Add test for aggrgate parameters  (#4682)
 * 9becf303 fix(ui): Fixed reconnection hot-loop. Fixes #4580 (#4663)
 * e8cc2fbb feat: Support per-output parameter aggregation (#4374)
 * b1e2c207 feat(controller): Allow to configure main container resources (#4656)
 * 4f9fab98 fix(controller): Cleanup the synchronize  pending queue once Workflow deleted (#4664)
 * 70554205 feat(ui): Make it easy to use SSO login with CLI. Resolves #4630 (#4645)
 * 76bcaecd feat(ui): add countdown to cronWorkflowList Closes #4636 (#4641)
 * 5614700b feat(ui): Add parameter value enum support to the UI. Fixes #4192 (#4365)
 * 7d0f1139 docs: adding more information about terminate and stop process. Fixes: #4454. (#4653)
 * 95ad3349 feat: Add shorthanded option -A for --all-namespaces (#4658)
 * 3b66f74c fix(ui): DataLoaderDropdown fix input type from promise to function that (#4655)
 * c4d986ab feat(ui): Replace 3 buttons with drop-down (#4648)
 * fafde1d6 fix(controller): Deal with hyphen in creator. Fixes #4058 (#4643)
 * 30e172d5 fix(manifests): Drop capabilities, add CNCF badge. Fixes #2614 (#4633)
 * f726b9f8 feat(ui): Add links to init and wait logs (#4642)
 * 94be7da3 feat(executor): Auto create s3 bucket if not present. Closes #3586  (#4574)
 * 1212df4d feat(controller): Support .AnySucceeded / .AllFailed for TaskGroup in depends logic. Closes #3405 (#3964)
 * 6175458a fix: Count Workflows with no phase as Pending for metrics (#4628)
 * a2566b95 feat(executor): More informative log when executors do not support output param from base image layer (#4620)
 * e1919c86 fix(ui): Fix Snyk issues (#4631)
 * 454f3ae3 fix(ui): Reference secrets in EnvVars. Fixes #3973  (#4419)
 * 1f039207 fix: derive jsonschema and fix up issues, validate examples dir… (#4611)
 * 92a28327 fix(argo-server): fix global variable validation error with reversed dag.tasks (#4369)
 * 79ca27f3 fix: Fix TestCleanFieldsExclude (#4625)
 * b3336e73 feat(ui): Add columns--narrower-height to AttributeRow (#4371)
 * 91bce257 fix(server): Correct webhook event payload marshalling. Fixes #4572 (#4594)
 * 39c805fa fix: Perform fields filtering server side (#4595)
 * 3af8195b fix: Null check pagination variable (#4617)
 * c84d56b6 feat(controller): Enhanced artifact repository ref. See #3184 (#4458)
 * 5c538d7a fix(executor): Fixed waitMainContainerStart returning prematurely. Closes #4599 (#4601)
 * b92d889a fix(docs): Bring minio chart instructions up to date (#4586)
 * 6c46aab7 fix(controller): Prevent tasks with names starting with digit to use either 'depends' or 'dependencies' (#4598)
 * 5bf7044b docs: Minor typo fix (#4610)
 * 4531d793 refactor: Use polling model for workflow phase metric (#4557)
 * 99240ada docs: Add JSON schema for IDE validation (#4581)
 * ef779bbf fix(executor): Handle sidecar killing in a process-namespace-shared pod (#4575)
 * 9ee4d446 fix(server): serve artifacts directly from disk to support large artifacts (#4589)
 * e3aaf2fb fix(server): use the correct name when downloading artifacts (#4579)
 * 1c62586e feat(controller): Retry transient offload errors. Resolves #4464 (#4482)
 * 2a3ab1ac docs: Fix a typo in example (#4590)
 * 15fd5794 feat(controller): Make MAX_OPERATION_TIME configurable. Close #4239 (#4562)
 * da4ff259 docs: Updated kubectl apply command in manifests README (#4577)
 * fdafc4ba chore: Updated stress test YAML (#4569)
 * 916b4549 feat(ui): Add Template/Cron workflow filter to workflow page. Closes #4532 (#4543)
 * 6e2b0cf3 docs: Clean-up examples. Fixes #4124 (#4128)
 * 4998b2d6 chore: Remove unused image build and push hooks (#4539)
 * 48af0244 fix: executor/pns containerid prefix fix (#4555)
 * 53195ed5 fix: Respect continueOn for leaf tasks (#4455)
 * e3d59f08 docs: Added CloudSeeds as one of the users for argo (#4553)
 * ad11180e docs: Update cost optimisation  to include information about cleaning up workflows and archiving (#4549)
 * 7e121509 fix(controller): Correct default port logic (#4547)
 * a712e535 fix: Validate metric key names (#4540)
 * c469b053 fix: Missing arg lines caused files not to copy into containers (#4542)
 * 0980ead3 fix(test): fix TestWFDefaultWithWFTAndWf flakiness (#4538)
 * 564e69f3 fix(ui): Do not auto-reload doc.location. Fixes #4530 (#4535)
 * eebcb8b8 docs: Add "Argo Workflows in 5 min" link to README (#4533)
 * 176d890c fix(controller): support float for param value (#4490)
 * 4bacbc12 feat(controller): make sso timeout configurable via cm (#4494)
 * 02e1f0e0 fix(server): Add `list sa` and `create secret` to `argo-server` roles. Closes #4526 (#4514)
 * d0082e8f fix: link templates not replacing multiple templates with same name (#4516)
 * 411bde37 feat: adds millisecond-level timestamps to argo and workflow-controller (#4518)
 * 2c54ca3f add bonprix to argo users (#4520)
 * 754a201d build: Reuse IMAGE_OS instead of hard-coded linux (#4519)
 * 2dab2d15 fix(test):  fix TestWFDefaultWithWFTAndWf flakiness (#4507)
 * 64ae3303 fix(controller): prepend script path to the script template args. Resolves #4481 (#4492)
 * 0931baf5 feat: Redirect to requested URL after SSO login (#4495)
 * 465447c0 fix: Ensure ContainerStatus in PNS is terminated before continuing (#4469)
 * f7287687 fix(ui): Check node children before counting them. (#4498)
 * bfc13c3f fix: Ensure redirect to login when using empty auth token (#4496)
 * d56ce890 feat(cli): add selector and field-selector option to terminate (#4448)
 * e501fcca fix(controller): Refactor the Merge Workflow, WorkflowTemplate and WorkflowDefaults (#4354)
 * 2ee3f5a7 fix(ui): fix the `all` option in the workflow archive list (#4486)
 * a441a97b refactor(server): Use patch instead of update to resume/suspend (#4468)
 * 9ecf0499 fix(controller): When semaphore lock config gets updated, enqueue the waiting workflows (#4421)
 * c31d1722 feat(cli): Support ARGO_HTTP1 for HTTP/1 CLI requests. Fixes #4394 (#4416)
 * b8fb2a8b chore(docs): Fix docgen (#4459)
 * 6c5ab780 feat: Add the --no-utf8 parameter to `argo get` command (#4449)
 * 933a4db0 refactor: Simplify grpcutil.TranslateError (#4465)
 * 24015065 docs: Add DDEV to USERS.md (#4456)
 * d752e2fa feat: Add resume/suspend endpoints for CronWorkflows (#4457)
 * 42d06050 fix: localhost not being resolved. Resolves #4460, #3564 (#4461)
 * 59843e1f fix(controller): Trigger no of workflows based on available lock (#4413)
 * 1be03db7 fix: Return copy of stored templates to ensure they are not modified (#4452)
 * 854883bd fix(controller): Fix throttler. Fixes #1554 and #4081 (#4132)
 * b956bc1a chore(controller): Refactor and tidy up (#4453)
 * 3e451114 fix(docs): timezone DST note on Cronworkflow (#4429)
 * f4f68a74 fix: Resolve inconsistent CronWorkflow persistence (#4440)
 * 76887cfd chore: Update pull request template (#4437)
 * da93545f feat(server): Add WorkflowLogs API. See #4394 (#4450)
 * 3960a0ee fix: Fix validation with Argo Variable in activeDeadlineSeconds (#4451)
 * dedf0521 feat(ui): Visualisation of the suspended CronWorkflows in the list. Fixes #4264 (#4446)
 * 6016ebdd ci: Speed up e2e tests (#4436)
 * 0d13f40d fix(controller): Tolerate int64 parameters. Fixes #4359 (#4401)
 * 2628be91 fix(server): Only try to use auth-mode if enabled. Fixes #4400 (#4412)
 * 7f2ff80f fix: Assume controller is in UTC when calculating NextScheduledRuntime (#4417)
 * 45fbc951 fix(controller): Design-out event errors. Fixes #4364 (#4383)
 * 5a18c674 fix(docs): update link to container spec (#4424)
 * 8006da12 fix: Add x-frame config option (#4420)
 * 46f0ca0f docs: Add Acquia to USERS.md (#4415)
 * 462e55e9 fix: Ensure resourceDuration variables in metrics are always in seconds (#4411)
 * 3aeb1741 fix(executor): artifact chmod should only if err != nil (#4409)
 * 2821e4e8 fix: Use correct template when processing metrics (#4399)
 * b34b91f9 ci: Split e2e tests to make them faster (#4404)
 * e8f82614 fix(validate): Local parameters should be validated locally. Fixes #4326 (#4358)
 * ddd45b6e fix(ui): Reconnect to DAG. Fixes #4301 (#4378)
 * 252c4633 feat(ui): Sign-post examples and the catalog. Fixes #4360 (#4382)
 * 334d1340 feat(server): Enable RBAC for SSO. Closes #3525 (#4198)
 * e409164b fix(ui): correct log viewer only showing first log line (#4389)
 * 28bdb6ff fix(ui): Ignore running workflows in report. Fixes #4387 (#4397)
 * 7ace8f85 fix(controller): Fix estimation bug. Fixes #4386 (#4396)
 * bdac65b0 fix(ui): correct typing errors in workflow-drawer (#4373)
 * cc9c0580 docs: Update events.md (#4391)
 * db5e28ed fix: Use DeletionHandlingMetaNamespaceKeyFunc in cron controller (#4379)
 * 99d33eed fix(server): Download artifacts from UI. Fixes #4338 (#4350)
 * db8a6d0b fix(controller): Enqueue the front workflow if semaphore lock is available (#4380)
 * 933ba834 fix: Fix intstr nil dereference (#4376)
 * 52520aac build: Fix clean-up of vendor (#4363)
 * f206b83b chore: match code comments to the generated documentation (#4375)
 * 220ac736 fix(controller): Only warn if cron job missing. Fixes #4351 (#4352)
 * dbbe95cc Use '[[:blank:]]' instead of ' ' to match spaces/tabs (#4361)
 * b03bd12a fix: Do not allow tasks using 'depends' to begin with a digit (#4218)
 * c237f3f5 docs: Fix a copy-paste typo in the docs (#4357)
 * b76246e2 fix(executor): Increase pod patch backoff. Fixes #4339 (#4340)
 * 7bfe303c docs: Update pvc example with correct resource indentation. Fixes #4330 (#4346)
 * ec671ddc feat(executor): Wait for termination using pod watch for PNS and K8SAPI executors. (#4253)
 * 3156559b fix: ui/package.json & ui/yarn.lock to reduce vulnerabilities (#4342)
 * f5e23f79 refactor: De-couple config (#4307)
 * 36002a26 docs: Fix typos (/priviledged/privileged/) (#4335)
 * 37a2ae06 fix(ui): correct typing errors in events-panel (#4334)
 * 03ef9d61 fix(ui): correct typing errors in workflows-toolbar (#4333)
 * 4de64c61 fix(ui): correct typing errors in cron-workflow-details (#4332)
 * 595aaa55 docs: Update releasing guide (#4284)
 * 939d8c30 feat(controller): add enum support in parameters (fixes #4192) (#4314)
 * e14f4f92 fix(executor): Fix the artifacts option in k8sapi and PNS executor Fixes#4244 (#4279)
 * ea9db436 fix(cli): Return exit code on Argo template lint command (#4292)
 * aa4a435b fix(cli): Fix panic on argo template lint without argument (#4300)
 * 640998e3 docs: Fix incorrect link to static code analysis document (#4329)
 * 04586fdb docs: Add diagram (#4295)
 * f7116fc7 build: Refactor `codegen` to make target dependencies clearer (#4315)
 * 20b3b1ba fix: merge artifact arguments from workflow template. Fixes #4296 (#4316)
 * cae8d1dc docs: Ye olde whitespace fix (#4323)
 * 3c63c3c4 chore(controller): Refactor the CronWorkflow schedule logic with sync.Map (#4320)
 * 1db63e52 docs: Fix typo in docs for artifact repository
 * 40648bcf Update USERS.md (#4322)
 * 07b2ef62 fix(executor): Retrieve containerId from cri-containerd /proc/{pid}/cgroup. Fixes #4302 (#4309)
 * e6b02490 feat(controller): Allow whitespace in variable substitution. Fixes #4286 (#4310)
 * 9119682b fix(build): Some minor Makefile fixes (#4311)
 * db20b4f2 feat(ui): Submit resources without namespace to current namespace. Fixes #4293 (#4298)
 * 240cd792 docs: improve ingress documentation for argo-server (#4306)
 * 26f39b6d fix(ci): add non-root user to Dockerfile (#4305)
 * 1cc68d89 fix(ui): undefined namespace in constructors (#4303)
 * d04f68fd docs: Docker TLS env config (#4299)
 * e54bf815 fix(controller): Patch rather than update cron workflows. (#4294)
 * 9157ef2a fix: TestMutexInDAG failure in master (#4283)
 * 2d6f4e66 fix: WorkflowEventBinding typo in aggregated roles (#4287)
 * c02bb7f0 fix(controller): Fix argo retry with PVCs. Fixes #4275 (#4277)
 * c0423a22 fix(ui): Ignore missing nodes in DAG. Fixes #4232 (#4280)
 * 58144290 fix(controller): Fix cron-workflow re-apply error. (#4278)
 * c605c6d7 fix(controller): Synchronization lock didn't release on DAG call flow Fixes #4046 (#4263)
 * 3cefc147 feat(ui): Add a nudge for users who have not set their security context. Closes #4233  (#4255)
 * e07abe27 test: Tidy up E2E tests (#4276)
 * 8ed799fa docs: daemon example: clarify prefix usage (#4258)
 * a461b076 feat(cli): add `--field-selector` option for `delete` command (#4274)
 * d7fac63e chore(controller): N/W progress fixes (#4269)
 * d377b7c0 docs: Add docs for `securityContext` and `emptyDir`. Closes #2239 (#4251)
 * 4c423453 feat(controller): Track N/M progress. See #2717 (#4194)
 * afbb957a fix: Add WorkflowEventBinding to aggregated roles (#4268)
 * 9bc675eb docs: Minor formatting fixes (#4259)
 * 4dfe7551 docs: correct a typo (#4261)
 * 6ce6bf49 fix(controller): Make the delay before the first workflow reconciliation configurable. Fixes #4107 (#4224)
 * 42b797b8 chore(api): Update swagger.json with Kubernetes v1.17.5 types. Closes #4204 (#4226)
 * 346292b1 feat(controller): Reduce reconcilliation time by exiting earlier. (#4225)
 * 407ac349 fix(ui): Revert bad part of commit (#4248)
 * eaae2309 fix(ui): Fix bugs with DAG view. Fixes #4232 & #4236 (#4241)
 * 04f7488a feat(ui): Adds a report page which shows basic historical workflow metrics. Closes #3557 (#3558)
 * a545a53f fix(controller): Check the correct object for Cronworkflow reapply error log (#4243)
 * ec7a5a40 fix(Makefile): removed deprecated k3d cmds. Fixes #4206 (#4228)
 * 1706a395 fix: Increase deafult number of CronWorkflow workers (#4215)
 * 270e6925 docs: Correct formatting of variables example YAML (#4237)
 * c5ff60b3 docs: Reinstate swagger.md so docs can build. (#4227)
 * 50f23181 feat(cli): Print 'no resource msg' when `argo list` returns zero workflows (#4166)
 * 2143a501 fix(controller): Support workflowDefaults on TTLController for WorkflowTemplateRef Fixes #4188 (#4195)
 * cac10f13 fix(controller): Support int64 for param value. Fixes #4169 (#4202)
 * e910b701 feat: Controller/server runAsNonRoot. Closes #1824 (#4184)
 * f3d1e9f8 chore: Enahnce logging around pod failures (#4220)
 * 4bd5fe10 fix(controller): Apply Workflow default on normal workflow scenario Fixes #4208 (#4213)
 * f9b65c52 chore(build): Update `make codegen` to only run on changes (#4205)
 * 0879067a chore(build): re-add #4127 and steps to verify image pull (#4219)
 * b17b569e fix(controller): reduce withItem/withParams memory usage. Fixes #3907 (#4207)
 * 524049f0 fix: Revert "chore: try out pre-pushing linux/amd64 images and updating ma… Fixes #4216 (#4217)
 * 9c08433f feat(executor): Decompress zip file input artifacts. Fixes #3585 (#4068)
 * 300634af docs: add Mixpanel to argo/USERS.md (#4137)
 * 14650339 fix(executor): Update executor retry config for ExponentialBackoff. (#4196)
 * 2b127625 fix(executor): Remove IsTransientErr check for ExponentialBackoff. Fixes #4144 (#4149)
 * f7e85f04 feat(server): Make Argo Server issue own JWE for SSO. Fixes #4027 & #3873 (#4095)
 * 951d38f8 refactor: Refactor Synchronization code (#4114)
 * 9319c074 fix(ui): handle logging disconnects gracefully (#4150)
 * 88ee7e13 docs: Add map-reduce example. Closes #4165  (#4175)
 * 6265c709 fix: Ensure CronWorkflows are persisted once per operation (#4172)
 * 2a992aee fix: Provide helpful hint when creating workflow with existing name (#4156)
 * de3a90dd refactor: upgrade argo-ui library version (#4178)
 * b7523369 feat(controller): Estimate workflow & node duration. Closes #2717 (#4091)
 * b3db0f5f chore: Update Github issue templates (#4161)
 * c468b34d fix(controller): Correct unstructured API version. Caused by #3719 (#4148)
 * d298706f chore: Deprecate unused RuntimeResolution field (#4155)
 * de81242e fix: Render full tree of onExit nodes in UI (#4109)
 * 109876e6 fix: Changing DeletePropagation to background in TTL Controller and Argo CLI (#4133)
 * c8eed1f2 chore: try out pre-pushing linux/amd64 images and updating manifest later (#4127)
 * 1e10e0cc Documentation (#4122)
 * 3a508a39 docs: Add Onepanel to USERS.md (#4147)
 * b3682d4f fix(cli): add validate args in delete command (#4142)
 * 45fe5173 test: Fix retry (attempt 3). Fixes #4101 (#4146)
 * 373543d1 feat(controller): Sum resources duration for DAGs and steps (#4089)
 * 4829e9ab feat: Add MaxAge to memoization (#4060)
 * af53a4b0 fix(docs): Update k3d command for running argo locally (#4139)
 * 554d6616 fix(ui): Ignore referenced nodes that don't exist in UI. Fixes #4079 (#4099)
 * e8b79921 fix(executor): race condition in docker kill (#4097)
 * 1af88d68 docs: Adding reserved.ai to users list (#4116)
 * 3bb0c2a1 feat(artifacts): Allow HTTP artifact load to set request headers (#4010)
 * 63b41375 fix(cli): Add retry to retry, again. Fixes #4101 (#4118)
 * bd4289ec docs: Added PDOK to USERS.md (#4110)
 * 76cbfa9d fix(ui): Show "waiting" msg while waiting for pod logs. Fixes #3916 (#4119)
 * 196c5eed fix(controller): Process workflows at least once every 20m (#4103)
 * 4825b7ec fix(server): argo-server-role to allow submitting cronworkflows from UI (#4112)
 * 29aba3d1 fix(controller): Treat annotation and conditions changes as significant (#4104)
 * befcbbce feat(ui): Improve error recovery. Close #4087 (#4094)
 * 5cb99a43 fix(ui): No longer redirect to `undefined` namespace. See #4084 (#4115)
 * fafc5a90 fix(cli): Reinstate --gloglevel flag. Fixes #4093 (#4100)
 * c4d91023 fix(cli): Add retry to retry ;). Fixes #4101 (#4105)
 * ff195f2e chore: use build matrix and cache (#4111)
 * 6b350b09 fix(controller): Correct the order merging the fields in WorkflowTemplateRef scenario. Fixes #4044 (#4063)
 * 764b56ba fix(executor): windows output artifacts. Fixes #4082 (#4083)
 * 7c92b3a5 fix(server): Optional timestamp inclusion when retrieving workflow logs. Closes #4033 (#4075)
 * 1bf651b5 feat(controller): Write back workflow to informer to prevent conflict errors. Fixes #3719 (#4025)
 * fdf0b056 feat(controller): Workflow-level `retryStrategy`/resubmit pending pods by default. Closes #3918 (#3965)
 * d7a297c0 feat(controller): Use pod informer for performance. (#4024)
 * d8d0ecbb fix(ui): [Snyk] Fix for 1 vulnerabilities (#4031)
 * ed59408f fix: Improve better handling on Pod deletion scenario  (#4064)
 * e2f4966b fix: make cross-plattform compatible filepaths/keys (#4040)
 * e94b9649 docs: Update release guide (#4054)
 * 4b673a51 chore: Fix merge error (#4073)
 * dcc8c23a docs: Update SSO docs to clarify the user must create K8S secrets for holding the OAuth2 values (#4070)
 * 5461d541 feat(controller): Retry archiving later on error. Fixes #3786 (#3862)
 * 4e085226 fix: Fix unintended inf recursion (#4067)
 * f1083f39 fix: Tolerate malformed workflows when retrying (#4028)
 * a0753951 chore(executor): upgrade `kubectl` to 1.18.8. Closes #3996 (#3999) (#3999)
 * 513ed9fd docs: Add `buildkit` example. Closes #2325 (#4008)
 * fc77beec fix(ui): Tiny modal DAG tweaks. Fixes #4039 (#4043)
 * 3f1792ab docs: Update USERS.md (#4041)
 * 74da0672 docs(Windows): Add more information on artifacts and limitations (#4032)
 * 81c0e427 docs: Add Data4Risk to users list (#4037)
 * ef0ce47e feat(controller): Support different volume GC strategies. Fixes #3095 (#3938)
 * d61a91c4 chore: Add stress testing code. Closes #2899 (#3934)
 * 9f120624 fix: Don't save label filter in local storage (#4022)
 * 0123c9a8 fix(controller): use interpolated values for mutexes and semaphores #3955 (#3957)
 * 25f12441 docs: Update pages to GA. Closes #4000 (#4007)
 * 5be25442 feat(controller): Panic or error on mis-matched resource version (#3949)
 * ae779599 fix: Delete realtime metrics of running Workflows that are deleted (#3993)
 * 4557c713 fix(controller): Script Output didn't set if template has RetryStrategy (#4002)
 * a013609c fix(ui): Do not save undefined namespace. Fixes #4019 (#4021)
 * f8145f83 fix(ui): Correctly show pod events. Fixes #4016 (#4018)
 * 2d722f1f fix(ui): Allow you to view timeline tab. Fixes #4005 (#4006)
 * f36ad2bb fix(ui): Report errors when uploading files. Fixes #3994 (#3995)
 * b5f31919 feat(ui): Introduce modal DAG renderer. Fixes: #3595 (#3967)
 * ad607469 fix(controller): Revert `resubmitPendingPods` mistake. Fixes #4001 (#4004)
 * fd1465c9 fix(controller): Revert parameter value to `*string`. Fixes #3960 (#3963)
 * 13879341 fix: argo-cluster-role pvc get (#3986)
 * f09babdb fix: Default PDB example typo (#3914)
 * f81b006a fix: Step and Task level timeout examples (#3997)
 * 91c49c14 fix: Consider WorkflowTemplate metadata during validation (#3988)
 * 7b1d17a0 fix(server): Remove XSS vulnerability. Fixes #3942 (#3975)
 * 20c518ca fix(controller): End DAG execution on deadline exceeded error. Fixes #3905 (#3921)
 * 74a68d47 feat(ui): Add `startedAt` and `finishedAt` variables to configurable links. Fixes #3898 (#3946)
 * 8e89617b fix: typo of argo server cli (#3984) (#3985)
 * 557531c7 docs: Adding Stillwater Supercomputing (#3987)
 * 1def65b1 fix: Create global scope before workflow-level realtime metrics (#3979)
 * df816958 build: Allow build with older `docker`. Fixes #3977 (#3978)
 * ca55c835 docs: fixed typo (#3972)
 * 29363b6b docs: Correct indentation to display correctly docs in the website (#3969)
 * 402fc0bf fix(executor): set artifact mode recursively. Fixes #3444 (#3832)
 * ff5ed7e4 fix(cli): Allow `argo version` without KUBECONFIG. Fixes #3943 (#3945)
 * d4210ff3 fix(server): Adds missing webhook permissions. Fixes #3927 (#3929)
 * 184884af fix(swagger): Correct item type. Fixes #3926 (#3932)
 * 97764ba9 fix: Fix UI selection issues (#3928)
 * b4329afd fix: Fix children is not defined error (#3950)
 * 3b16a023 chore(doc): fixed java client project link (#3947)
 * 946da359 chore: Fixed  TestTemplateTimeoutDuration testcase (#3940)
 * c977aa27 docs: Adds roadmap. Closes #3835 (#3863)
 * 5a0c515c feat: Step and Task Level Global Timeout (#3686)
 * a8d10340 docs: add Nikkei to user list (#3935)
 * 24c77838 fix: Custom metrics are not recorded for DAG tasks Fixes #3872 (#3886)
 * d4cf0d26 docs: Update workflow-controller-configmap.yaml with SSL options (#3924)
 * 7e6a8910 docs: update docs for upgrading readiness probe to HTTPS. Closes #3859 (#3877)
 * de2185c8 feat(controller): Set retry factor to 2. Closes #3911 (#3919)
 * be91d762 fix: Workflow should fail on Pod failure before container starts Fixes #3879 (#3890)
 * c4c80069 test: Fix TestRetryOmit and TestStopBehavior (#3910)
 * 650869fd feat(server): Display events involved in the workflow. Closes #3673 (#3726)
 * 5b5d2359 fix(controller): Cron re-apply update (#3883)
 * fd3fca80 feat(artifacts): retrieve subpath from unarchived ref artifact. Closes #3061 (#3063)
 * 6a452ccd test: Fix flaky e2e tests (#3909)
 * 6e82bf38 feat(controller): Emit events for malformed cron workflows. See #3881 (#3889)
 * f04bdd6a Update workflow-controller-configmap.yaml (#3901)
 * bb79e3f5 fix(executor): Replace default retry in executor with an increased value retryer (#3891)
 * b681c113 fix(ui): use absolute URL to redirect from autocomplete list. Closes #3903 (#3906)
 * 712c77f5 chore(users): Add Fynd Trak to the list of Users (#3900)
 * d55402db ci: Fix broken Multiplatform builds (#3908)
 * 9681a4e2 fix(ui): Improve error recovery. Fixes #3867 (#3869)
 * b926f8c0 chore: Remove unused imports (#3892)
 * 4c18a06b feat(controller): Always retry when `IsTransientErr` to tolerate transient errors. Fixes #3217 (#3853)
 * 0cf7709f fix(controller): Failure tolerant workflow archiving and offloading. Fixes #3786 and #3837 (#3787)
 * 359ee8db fix: Corrects CRD and Swagger types. Fixes #3578 (#3809)
 * 58ac52b8 chore(ui): correct a typo (#3876)
 * dae0f2df feat(controller): Do not try to create pods we know exists to prevent `exceeded quota` errors. Fixes #3791 (#3851)
 * 9781a1de ci: Create manifest from images again (#3871)
 * c6b51362 test: E2E test refactoring (#3849)
 * 04898fee chore: Added unittest for PVC exceed quota Closes #3561 (#3860)
 * 4e42208c ci: Changed tagging and amend multi-arch manifest. (#3854)
 * c352f69d chore: Reduce the 2x workflow save on Semaphore scenario (#3846)
 * a24bc944 feat(controller): Mutexes. Closes #2677 (#3631)
 * a821c6d4 ci: Fix build by providing clean repo inside Docker (#3848)
 * 99fe11a7 feat: Show next scheduled cron run in UI/CLI (#3847)
 * 6aaceeb9 fix: Treat collapsed nodes as their siblings (#3808)
 * 7f5acd6f docs: Add 23mofang to USERS.md
 * 10cb447a docs: Update example README for duration (#3844)
 * 1678e58c ci: Remove external build dependency (#3831)
 * 743ec536 fix(ui): crash when workflow node has no memoization info (#3839)
 * a2f54da1 fix(docs): Amend link to the Workflow CRD (#3828)
 * ca8ab468 fix: Carry over ownerReferences from resubmitted workflow. Fixes #3818 (#3820)
 * da43086a fix(docs): Add Entrypoint Cron Backfill example  Fixes #3807 (#3814)
 * ed749a55 test: Skip TestStopBehavior and TestRetryOmit (#3822)
 * 9292ae1e ci: static files not being built with Homebrew and dirty binary. Fixes #3769 (#3801)
 * c840adb2 docs: memory base amount denominator documentation
 * 8e1a3db5 feat(ui): add node memoization information to node summary view (#3741)
 * 9de49e2e ci: Change workflow for pushing images. Fixes #2080
 * d235c7d5 fix: Consider all children of TaskGroups in DAGs (#3740)
 * 3540d152 Add SYS_PTRACE to ease the setup of non-root deployments with PNS executor. (#3785)
 * 2f654971 chore: add New Relic to USERS.md (#3810)
 * ce5da590 docs: Add section on CronWorkflow crash recovery (#3804)
 * 0ca83924 feat: Github Workflow multi arch. Fixes #2080 (#3744)
 * bee0e040 docs: Remove confusing namespace (#3772)
 * 7ad6eb84 fix(ui): Remove outdated download links. Fixes #3762 (#3783)
 * 22636782 fix(ui): Correctly load and store namespace. Fixes #3773 and #3775 (#3778)
 * a9577ab9 test: Increase cron test timeout to 7m (#3799)
 * ed90d403 fix(controller): Support exit handler on workflow templates.  Fixes #3737 (#3782)
 * dc75ee81 test: Simplify E2E test tear-down (#3749)
 * 821e40a2 build: Retry downloading Kustomize (#3792)
 * f15a8f77 fix: workflow template ref does not work in other namespace (#3795)
 * ef44a03d fix: Increase the requeue duration on checkForbiddenErrorAndResubmitAllowed (#3794)
 * 0125ab53 fix(server): Trucate creator label at 63 chars. Fixes #3756 (#3758)
 * a38101f4 feat(ui): Sign-post IDE set-up. Closes #3720 (#3723)
 * 21dc23db chore: Format test code (#3777)
 * ee910b55 feat(server): Emit audit events for workflow event binding errors (#3704)
 * e9b29e8c fix: TestWorkflowLevelSemaphore flakiness (#3764)
 * fadd6d82 fix: Fix workflow onExit nodes not being displayed in UI (#3765)
 * df06e901 docs: Correct typo in `--instanceid`
 * 82a671c0 build: Lint e2e test files (#3752)
 * 513675bc fix(executor): Add retry on pods watch to handle timeout. (#3675)
 * e35a86ff feat: Allow parametrizable int fields (#3610)
 * da115f9d fix(controller): Tolerate malformed resources. Fixes #3677 (#3680)
 * 407f9e63 docs: Remove misleading argument in workflow template dag examples. (#3735) (#3736)
 * f8053ae3 feat(operator): Add scope params for step startedAt and finishedAt (#3724)
 * 54c2134f fix: Couldn't Terminate/Stop the ResourceTemplate Workflow (#3679)
 * 12ddc1f6 fix: Argo linting does not respect namespace of declared resource (#3671)
 * acfda260 feat(controller): controller logs to be structured #2308 (#3727)
 * cc2e42a6 fix(controller): Tolerate PDB delete race. Fixes #3706 (#3717)
 * 5eda8b86 fix: Ensure target task's onExit handlers are run (#3716)
 * 811a4419 docs(windows): Add note about artifacts on windows (#3714)
 * 5e5865fb fix: Ingress docs (#3713)
 * eeb3c9d1 fix: Fix bug with 'argo delete --older' (#3699)
 * 6134a565 chore: Introduce convenience methods for intstr. (#3702)
 * 7aa536ed feat: Upgrade Minio v7 with support IRSA (#3700)
 * 4065f265 docs: Correct version. Fixes #3697 (#3701)
 * 71d61281 feat(server): Trigger workflows from webhooks. Closes #2667  (#3488)
 * a5d995dc fix(controller): Adds ALL_POD_CHANGES_SIGNIFICANT (#3689)
 * 9f00cdc9 fix: Fixed workflow queue duration if PVC creation is forbidden (#3691)
 * 2baaf914 chore: Update issue templates (#3681)
 * 41ebbe8e fix: Re-introduce 1 second sleep to reconcile informer (#3684)
 * 6e3c5bef feat(ui): Make UI errors recoverable. Fixes #3666 (#3674)
 * 27fea1bb chore(ui): Add label to 'from' section in Workflow Drawer (#3685)
 * 32d6f752 feat(ui): Add links to wft, cwf, or cwft to workflow list and details. Closes #3621 (#3662)
 * 1c95a985 fix: Fix collapsible nodes rendering (#3669)
 * 87b62bbb build: Use http in dev server (#3670)
 * dbb39368 feat: Add submit options to 'argo cron create' (#3660)
 * 2b6db45b fix(controller): Fix nested maps. Fixes #3653 (#3661)
 * 3f293a4d fix: interface{} values should be expanded with '%v' (#3659)
 * f08ab972 docs: Fix type in default-workflow-specs.md (#3654)
 * a8f4da00 fix(server): Report v1.Status errors. Fixes #3608 (#3652)
 * a3a4ea0a fix: Avoid overriding the Workflow parameter when it is merging with WorkflowTemplate parameter (#3651)
 * 9ce1d824 fix: Enforce metric Help must be the same for each metric Name (#3613)
 * 4eca0481 docs: Update link to examples so works in raw github.com view (#3623)
 * f77780f5 fix(controller): Carry-over labels for re-submitted workflows. Fixes #3622 (#3638)
 * 3f3a4c91 docs: Add comment to config map about SSO auth-mode (#3634)
 * d9090c99 build: Disable TLS for dev mode. Fixes #3617 (#3618)
 * bcc6e1f7 fix: Fixed flaky unit test TestFailSuspendedAndPendingNodesAfterDeadline (#3640)
 * 8f70d224 fix: Don't panic on invalid template creation (#3643)
 * 5b0210dc fix: Simplify the WorkflowTemplateRef field validation to support all fields in WorkflowSpec except `Templates` (#3632)
 * 2375878a fix: Fix 'malformed request: field selector' error (#3636)
 * 87ea54c9 docs: Add documentation for configuring OSS artifact storage (#3639)
 * 861afd36 docs: Correct indentation for codeblocks within bullet-points for "workflow-templates" (#3627)
 * 0f37e81a fix: DAG level Output Artifacts on K8S and Kubelet executor (#3624)
 * a89261bf build(cli)!: Zip binaries binaries. Closes #3576 (#3614)
 * 7f844473 fix(controller): Panic when outputs in a cache entry are nil (#3615)
 * 86f03a3f fix(controller): Treat TooManyError same as Forbidden (i.e. try again). Fixes #3606 (#3607)
 * 2e299df3 build: Increase timeout (#3616)
 * e0a4f13d fix(server): Re-establish watch on v1.Status errors. Fixes #3608 (#3609)
 * cdbb5711 docs: Memoization Documentation (#3598)
 * 7abead2a docs: fix typo - replace "workfow" with "workflow" (#3612)
 * f7be20c1 fix: Fix panic and provide better error message on watch endpoint (#3605)
 * 491f4f74 fix: Argo Workflows does not honour global timeout if step/pod is not able to schedule (#3581)
 * 5d8f85d5 feat(ui): Enhanced workflow submission. Closes #3498 (#3580)
 * a4a26414 build: Initialize npm before installing swagger-markdown (#3602)
 * ad3441dc feat: Add 'argo node set' command (#3277)
 * a43bf129 docs: Migrate to homebrew-core (#3567) (#3568)
 * 17b46bdb fix(controller): Fix bug in util/RecoverWorkflowNameFromSelectorString. Add error handling (#3596)
 * c968877c docs: Document ingress set-up. Closes #3080 (#3592)
 * 8b6e43f6 fix(ui): Fix multiple UI issues (#3573)
 * cdc935ae feat(cli): Support deleting resubmitted workflows (#3554)
 * 1b757ea9 feat(ui): Change default language for Resource Editor to YAML and store preference in localStorage. Fixes #3543 (#3560)
 * c583bc04 fix(server): Ignore not-JWT server tokens. Fixes #3562 (#3579)
 * 5afbc131 fix(controller): Do not panic on nil output value. Fixes #3505 (#3509)
 * c409624b docs: Synchronization documentation (#3537)
 * 0bca0769 docs: Workflow of workflows pattern (#3536)
 * 827106de fix: Skip TestStorageQuotaLimit (#3566)
 * 13b1d3c1 feat(controller): Step level memoization. Closes #944 (#3356)
 * 96e520eb fix: Exceeding quota with volumeClaimTemplates (#3490)
 * 144c9b65 fix(ui): cannot push to nil when filtering by label (#3555)
 * 7e4a7808 feat: Collapse children in UI Workflow viewer (#3526)
 * 7536982a fix: Fix flakey TestRetryOmitted (#3552)
 * 05d573d7 docs: Change formatting to put content into code block (#3553)
 * dcee3484 fix: Fix links in fields doc (#3539)
 * fb67c1be Fix issue #3546 (#3547)
 * d07a0e74 ci: Make builds marginally faster. Fixes #3515 (#3519)
 * 4cb6aa04 chore: Enable no-response bot (#3510)
 * 31afa92a fix(artifacts): support optional input artifacts, Fixes #3491 (#3512)
 * 977beb46 fix: Fix when retrying Workflows with Omitted nodes (#3528)
 * ab4ef5c5 fix: Panic on CLI Watch command (#3532)
 * b901b279 fix(controller): Backoff exponent is off by one. Fixes #3513 (#3514)
 * 49ef5c0f fix: String interpreted as boolean in labels (#3518)
 * 19e700a3 fix(cli): Check mutual exclusivity for argo CLI flags (#3493)
 * 7d45ff7f fix: Panic on releaseAllWorkflowLocks if Object is not Unstructured type (#3504)
 * 1b68a5a1 fix: ui/package.json & ui/yarn.lock to reduce vulnerabilities (#3501)
 * 7f262fd8 fix(cli)!: Enable CLI to work without kube config. Closes #3383, #2793 (#3385)
 * 2976e7ac build: Clear cmd docs before generating them (#3499)
 * 27528ba3 feat: Support completions for more resources (#3494)
 * 5bd2ad7a fix: Merge WorkflowTemplateRef with defaults workflow spec (#3480)
 * e244337b chore: Added examples for exit handler for step and dag level (#3495)
 * bcb32547 build: Use `git rev-parse` to accomodate older gits (#3497)
 * 3eb6e2f9 docs: Add link to GitHub Actions in the badge (#3492)
 * 69179e72 fix: link to server auth mode docs, adds Tulip as official user (#3486)
 * 7a8e2b34 docs: Add comments to NodePhase definition. Closes #1117. (#3467)
 * 24d1e529 build: Simplify builds (#3478)
 * acf56f9f feat(server): Label workflows with creator. Closes #2437 (#3440)
 * 3b8ac065 fix: Pass resolved arguments to onExit handler (#3477)
 * 58097a9e docs: Add controller-level metrics (#3464)
 * f6f1844b feat: Attempt to resolve nested tags (#3339)
 * 48e15d6f feat(cli): List only resubmitted workflows option (#3357)
 * 25e9c0cd docs, quick-start. Use http, not https for link (#3476)
 * 7a2d7642 fix: Metric emission with retryStrategy (#3470)
 * f5876e04 test(controller): Ensure resubmitted workflows have correct labels (#3473)
 * aa92ec03 fix(controller): Correct fail workflow when pod is deleted with --force. Fixes #3097 (#3469)
 * a1945d63 fix(controller): Respect the volumes of a workflowTemplateRef. Fixes … (#3451)
 * 847ba530 test(controller): Add memoization tests. See #3214 (#3455) (#3466)
 * f5183aed docs: Fix CLI docs (#3465)
 * 1e42813a test(controller): Add memoization tests. See #3214 (#3455)
 * abe768c4 feat(cli): Allow to view previously terminated container logs (#3423)
 * 7581025f fix: Allow ints for sequence start/end/count. Fixes #3420 (#3425)
 * b82f900a Fixed typos (#3456)
 * 23760119 feat: Workflow Semaphore Support (#3141)
 * 81cba832 feat: Support WorkflowMetadata in WorkflowTemplate and ClusterWorkflowTemplate (#3364)
 * 568c032b chore: update aws-sdk-go version (#3376)
 * bd27d9f3 chore: Upgrade node-sass (#3450)
 * b1e601e5 docs: typo in argo stop --help (#3439)
 * 308c7083 fix(controller): Prevent panic on nil node. Fixes #3436 (#3437)
 * 8ab06f53 feat(controller): Add log message count as metrics. (#3362)
 * 5d0c436d chore: Fix GitHub Actions Docker Image build  (#3442)
 * e54b4ab5 docs: Add Sohu as official Argo user (#3430)
 * ee6c8760 fix: Ensure task dependencies run after onExit handler is fulfilled (#3435)
 * 6dc04b39 chore: Use GitHub Actions to build Docker Images to allow publishing Windows Images (#3291)
 * 05b3590b feat(controller): Add support for Docker workflow executor for Windows nodes (#3301)
 * 676868f3 fix(docs): Update kubectl proxy URL (#3433)
 * 3507c3e6 docs: Make https://argoproj.github.io/argo/  (#3369)
 * 733e95f7 fix: Add struct-wide RWMutext to metrics (#3421)
 * 0463f241 fix: Use a unique queue to visit nodes (#3418)
 * eddcac63 fix: Script steps fail with exceededQuota (#3407)
 * c631a545 feat(ui): Add Swagger UI (#3358)
 * 910f636d fix: No panic on watch. Fixes #3411 (#3426)
 * b4da1bcc fix(sso): Remove unused `groups` claim. Fixes #3411 (#3427)
 * 330d4a0a fix: panic on wait command if event is null (#3424)
 * 7c439424 docs: Include timezone name reference (#3414)
 * 03cbb8cf fix(ui): Render DAG with exit node (#3408)
 * 3d50f985 feat: Expose certain queue metrics (#3371)
 * c7b35e05 fix: Ensure non-leaf DAG tasks have their onExit handler's run (#3403)
 * 70111600 fix: Fix concurrency issues with metrics (#3401)
 * d307f96f docs: Update config example to include useSDKCreds (#3398)
 * 637d50bc chore: maybe -> may be (#3392)
 * e70a8863 chore: Added CWFT WorkflowTemplateRef example (#3386)
 * bc4faf5f fix: Fix bug parsing parmeters (#3372)
 * 4934ad22 fix: Running pods are garaged in PodGC onSuccess
 * 0541cfda chore(ui): Remove unused interfaces for artifacts (#3377)
 * 20382cab docs: Fix incorrect example of global parameter (#3375)
 * 1db93c06 perf: Optimize time-based filtering on large number of workflows (#3340)
 * 2ab9495f fix: Don't double-count metric events (#3350)
 * 7bd3e720 fix(ui): Confirmation of workflow actions (#3370)
 * 488790b2 Wellcome is using Argo in our Data Labs division (#3365)
 * 63e71192 chore: Remove unused code (#3367)
 * a64ceb03 build: Enable Stale Bot (#3363)
 * e4b08abb fix(server): Remove `context cancelled` error. Fixes #3073 (#3359)
 * 74ba5162 fix: Fix UI bug in DAGs (#3368)
 * 5e60decf feat(crds)!: Adds CRD generation and enhanced UI resource editor. Closes #859 (#3075)
 * c2347f35 chore: Simplify deps by removing YAML (#3353)
 * 1323f9f4 test: Add e2e tags (#3354)
 * 731a1b4a fix(controller): Allow events to be sent to non-argo namespace. Fixes #3342 (#3345)
 * 916e0db2 Adding InVision to Users (#3352)
 * 6caf10fa fix: Ensure child pods respect maxDuration (#3280)
 * 8f4945f5 docs: Field fix (ParallelSteps -> WorkflowStep) (#3338)
 * 2b4b7340 fix: Remove broken SSO from quick-starts (#3327)
 * 26570fd5 fix(controller)!: Support nested items. Fixes #3288 (#3290)
 * c3d85716 chore: Avoid variable name collision with imported package name (#3335)
 * ca822af0 build: Fix path to go-to-protobuf binary (#3308)
 * 769a964f feat(controller): Label workflows with their source workflow template (#3328)
 * 0785be24 fix(ui): runtime error from null savedOptions props (#3330)
 * 200be0e1 feat: Save pagination limit and selected phases/labels to local storage (#3322)
 * b5ed90fe feat: Allow to change priority when resubmitting workflows (#3293)
 * 60c86c84 fix(ui): Compiler error from workflows toolbar (#3317)
 * 3fe6ecc4 docs: Document access token creation and usage (#3316)
 * ab3c081e docs: Rename Ant Financial to Ant Group (#3304)
 * baad42ea feat(ui): Add ability to select multiple workflows from list and perform actions on them. Fixes #3185 (#3234)
 * b6118939 fix(controller): Fix panic logging. (#3315)
 * 633ea71e build: Pin `goimports` to working version (#3311)
 * 436c1259 ci: Remove CircleCI (#3302)
 * 8e340229 build: Remove generated Swagger files. (#3297)
 * e021d7c5 Clean up unused constants (#3298)
 * 48d86f03 build: Upload E2E diagnostics after failure (#3294)
 * 8b12f433 feat(cli): Add --logs to `argo [submit|resubmit|retry]. Closes #3183 (#3279)
 * 07b450e8 fix: Reapply Update if CronWorkflow resource changed (#3272)
 * 8af01491 docs: ArchiveLabelSelector document (#3284)
 * 38c908a2 docs: Add example for handling large output resutls (#3276)
 * d44d264c Fixes validation of overridden ref template parameters. (#3286)
 * 62e54fb6 fix: Fix delete --complete (#3278)
 * a3c379bb docs: Updated WorkflowTemplateRef  on WFT and CWFT (#3137)
 * 824de95b fix(git): Fixes Git when using auth or fetch. Fixes #2343 (#3248)
 * 018fcc23 Update releasing.md (#3283)

### Contributors

 * 0x1D-1983
 * Aayush Rangwala
 * Alastair Maw
 * Alex Capras
 * Alex Collins
 * Alexander Matyushentsev
 * Alexander Mikhailian
 * Alexander Zigelski
 * Alexey Volkov
 * Amim Knabben
 * Ang Gao
 * Antoine Dao
 * Arghya Sadhu
 * Arthur Outhenin-Chalandre
 * BOOK
 * Bailey Hayes
 * Basanth Jenu H B
 * Bikramdeep Singh
 * Boolman
 * Byungjin Park (BJ)
 * Carlos Montemuino
 * Chris Hepner
 * Daisuke Taniwaki
 * David Gibbons
 * Douglas Lehr
 * Dylan Hellems
 * Elli Ludwigson
 * Elvis Jakupovic
 * Espen Finnesand
 * Fischer Jemison
 * Florian
 * Floris Van den Abeele
 * Francesco Murdaca
 * Galen Han
 * Greg Roodt
 * Guillaume Hormiere
 * Huan-Cheng Chang
 * Hussein Awala
 * Ids van der Molen
 * Igor Stepura
 * InvictusMB
 * Isaac Gaskin
 * J.P. Zivalich
 * James Laverack
 * Jared Welch
 * Jeff Uren
 * Jesse Suen
 * Jie Zhang
 * Jonny
 * Juan C. Müller
 * Justen Walker
 * Kaan C. Fidan
 * Kaushik B
 * Ken Kaizu
 * Kristoffer Johansson
 * Lennart Kindermann
 * Lucas Theisen
 * Ludovic Cléroux
 * Lénaïc Huard
 * Marcin Gucki
 * Markus Lippert
 * Martin Suchanek
 * Matt Campbell
 * Maximilian Roos
 * Michael Albers
 * Michael Crenshaw
 * Michael Ruoss
 * Michael Weibel
 * Michal Cwienczek
 * Mike Chau
 * Naisisor
 * Nelson Rodrigues
 * Nicwalle
 * Niklas Vest
 * Nirav Patel
 * Noah Hanjun Lee
 * Noj Vek
 * Noorain Panjwani
 * Oleg Borodai
 * Paavo Pokkinen
 * Paul Brabban
 * Pavel Čižinský
 * Pranaye Karnati
 * Remington Breeze
 * Roi Kramer
 * RossyWhite
 * Rush Tehrani
 * Saravanan Balasubramanian
 * Sebastian Ortan
 * Shannon
 * Simeon H.K. Fitch
 * Simon Behar
 * Simon Frey
 * Snyk bot
 * Song Juchao
 * Stefan Gloutnikov
 * Stéphane Este-Gracias
 * Takahiro Tsuruda
 * Takayoshi Nishida
 * Theodore Omtzigt
 * Tomáš Coufal
 * Trevor Foster
 * Trevor Wood
 * Viktor Farcic
 * Vlad Losev
 * Weston Platter
 * Wouter Remijn
 * Wylie Hobbs
 * Yuan Tang
 * Zach
 * Zach Aller
 * Zach Himsel
 * Zadjad Rezai
 * aletepe
 * bei-re
 * bellevuerails
 * boundless-thread
 * candonov
 * cocotyty
 * conanoc
 * dgiebert
 * dherman
 * drannenberg
 * duluong
 * ermeaney
 * fsiegmund
 * haibingzhao
 * hermanhobnob
 * ivancili
 * jacky
 * joe
 * joyciep
 * juliusvonkohout
 * kennytrytek
 * lonsdale8734
 * maguowei
 * makocchi
 * markterm
 * nishant-d
 * omerfsen
 * saranyaeu2987
 * sh-tatsuno
 * tczhao
 * tianfeiyu
 * tomgoren
 * yonirab
 * zhengchenyu

## v2.9.5 (2020-08-06)

 * 5759a0e1 Update manifests to v2.9.5
 * 53d20462 codegen
 * c0382fd9 remove line
 * 18cf4ea6 fix: Enforce metric Help must be the same for each metric Name (#3613)
 * 7b4e98a8 fix: Fix 'malformed request: field selector' error (#3636)
 * 0fceb627 fix: Fix panic and provide better error message on watch endpoint (#3605)
 * 8a7e9d3d fix(controller): Fix bug in util/RecoverWorkflowNameFromSelectorString. Add error handling (#3596)
 * 2ba24334 fix: Re-introduce 1 second sleep to reconcile informer (#3684)
 * dca3b6ce fix(controller): Adds ALL_POD_CHANGES_SIGNIFICANT (#3689)
 * 819bfdb6 fix: Avoid overriding the Workflow parameter when it is merging with WorkflowTemplate parameter (#3651)
 * 89e05bdb fix: Don't panic on invalid template creation (#3643)
 * 0b8d78e1 fix: Simplify the WorkflowTemplateRef field validation to support all fields in WorkflowSpec except `Templates` (#3632)

### Contributors

 * Alex Collins
 * Remington Breeze
 * Saravanan Balasubramanian
 * Simon Behar

## v2.9.4 (2020-07-24)

 * 20d2ace3 Update manifests to v2.9.4
 * 41db5525 Fix build
 * 58778559 Fix build
 * f047ddf3 fix: Fix flakey TestRetryOmitted (#3552)
 * b6ad88e2 fix: Fix when retrying Workflows with Omitted nodes (#3528)
 * 79599820 fix: Panic on CLI Watch command (#3532)
 * eaa815f1 Fixed Packer and Hydrator test
 * 71c7f64e Fixed test failure
 * f0e8a332 fix: Merge WorkflowTemplateRef with defaults workflow spec (#3480)
 * b03498be chore: Fix GitHub Actions Docker Image build  (#3442)
 * bf138662 chore: Use GitHub Actions to build Docker Images to allow publishing Windows Images (#3291)

### Contributors

 * Markus Lippert
 * Saravanan Balasubramanian
 * Simon Behar

## v2.9.3 (2020-07-14)

 * 9407e19b Merge branch 'release-2.9' of github.com:argoproj/argo into release-2.9
 * d597af5c Update manifests to v2.9.3
 * d1a2ffd9 fix: Pass resolved arguments to onExit handler (#3482)
 * a52c371c test: Removed flakey image tag check
 * 06a695e9 ci: v2.9.2
 * 1818b0a9 ci: make codegen fetch tags
 * 7d49f6a4 ci: Fix
 * 2376707c ci: Checkout tags
 * 54b24ff7 ci: Checkout tags
 * 2b706247 Revert "fix: Pass resolved arguments to onExit handler (#3477)"
 * a431f93c fix: Pass resolved arguments to onExit handler (#3477)
 * f394d571 ci: Fix
 * 052efb47 ci: Fix
 * 52bb1471 fix: Metric emission with retryStrategy (#3470)
 * 1766a019 build: Better VERSION logic
 * 7c6d8aa6 build: Fix
 * cf811286 ci: Attempt to fix
 * e10a5bd7 ci: Capture logs
 * 07292ebe build: mkdir -p dist
 * 505a6478 build: Sync Github Action from master
 * 675ce293 fix(controller): Correct fail workflow when pod is deleted with --force. Fixes #3097 (#3469)
 * 194a2139 fix(controller): Respect the volumes of a workflowTemplateRef. Fixes … (#3451)
 * 584cb402 fix(controller): Port master fix for #3214
 * 065d9b65 test(controller): Add memoization tests. See #3214 (#3455) (#3466)
 * b252b408 test(controller): Add memoization tests. See #3214 (#3455)
 * e3a8319b fix(controller): Prevent panic on nil node. Fixes #3436 (#3437)

### Contributors

 * Alex Collins
 * Simon Behar

## v2.9.2 (2020-07-08)

 * 65c2bd44 merge Dockerfile from master
 * 14942f2f Update manifests to v2.9.2
 * 823f9c54 Fix botched conflict resolution
 * 2b3ccd3a fix: Add struct-wide RWMutext to metrics (#3421)
 * 8e9ba494 fix: Use a unique queue to visit nodes (#3418)
 * 28f76572 conflict resolved
 * dcc09c98 fix: No panic on watch. Fixes #3411 (#3426)
 * 4a48e25f fix(sso): Remove unused `groups` claim. Fixes #3411 (#3427)
 * 1e736b23 fix: panic on wait command if event is null (#3424)
 * c10da5ec fix: Ensure non-leaf DAG tasks have their onExit handler's run (#3403)
 * 25b150aa fix(ui): Render DAG with exit node (#3408)
 * 6378a587 fix: Fix concurrency issues with metrics (#3401)

### Contributors

 * Alex Collins
 * Saravanan Balasubramanian
 * Simon Behar

## v2.9.1 (2020-07-03)

 * 6b967d08 Update manifests to v2.9.1
 * 6bf5fb3c fix: Running pods are garaged in PodGC onSuccess
 * d67d3b1d Update manifests to v2.9.0
 * 9c52c1be fix: Don't double-count metric events (#3350)
 * 813122f7 fix: Fix UI bug in DAGs (#3368)
 * 248643d3 fix: Ensure child pods respect maxDuration (#3280)
 * 71d29584 fix(controller): Allow events to be sent to non-argo namespace. Fixes #3342 (#3345)
 * 52be71bc fix: Remove broken SSO from quick-starts (#3327)

### Contributors

 * Alex Collins
 * Saravanan Balasubramanian
 * Simon Behar

## v2.9.0-rc4 (2020-06-26)

 * 5b109bcb Update manifests to v2.9.0-rc4
 * 36cbcfb6 docs: Document access token creation and usage (#3316)
 * 00ad29bd docs: Rename Ant Financial to Ant Group (#3304)
 * 011f1368 fix(controller): Fix panic logging. (#3315)
 * 9f934496 build: Pin `goimports` to working version (#3311)
 * 888c9b8a ci: Remove CircleCI (#3302)
 * f7e13a3d build: Remove generated Swagger files. (#3297)
 * 5395ad3f Clean up unused constants (#3298)
 * 2bb2cdf0 build: Upload E2E diagnostics after failure (#3294)
 * a2a1fba8 fix: Reapply Update if CronWorkflow resource changed (#3272)
 * 9af98a5b Fixes validation of overridden ref template parameters. (#3286)
 * dc65faa9 docs: Updated WorkflowTemplateRef  on WFT and CWFT (#3137)
 * 86401ba4 docs: ArchiveLabelSelector document (#3284)
 * cce666b7 docs: Add example for handling large output resutls (#3276)
 * a91cea5f fix: Fix delete --complete (#3278)
 * d5a4807a Update releasing.md (#3283)

### Contributors

 * Alex Collins
 * Michael Crenshaw
 * Saravanan Balasubramanian
 * Simon Behar
 * Vlad Losev
 * Yuan Tang

## v2.9.0-rc3 (2020-06-23)

 * 2e95ff48 Update manifests to v2.9.0-rc3
 * 61ca07e2 Merge branch 'master' into release-2.9
 * acee573b docs: Update CI Badges (#3282)
 * 9eb182c0 build: Allow to change k8s namespace for installation (#3281)
 * 2bcfafb5 fix: Add {{workflow.status}} to workflow-metrics (#3271)
 * e6aab605 fix(jqFilter)!: remove extra quotes around output parameter value (#3251)
 * f4580163 fix(ui): Allow render of templates without entrypoint. Fixes #2891 (#3274)
 * f30c05c7 build: Add warning to ensure 'v' is present on release versions (#3273)
 * d1cb1992 fixed archiveLabelSelector nil (#3270)
 * c7e4c180 fix(ui): Update workflow drawer with new duration format (#3256)
 * f2381a54 fix(controller): More structured logging. Fixes #3260 (#3262)
 * acba084a fix: Avoid unnecessary nil check for annotations of resubmitted workflow (#3268)
 * 55e13705 feat: Append previous workflow name as label to resubmitted workflow (#3261)
 * 2dae7244 feat: Add mode to require Workflows to use workflowTemplateRef (#3149)
 * 56694abe Fixed onexit on workflowtempalteRef (#3263)
 * 54dd72c2 update mysql yaml port (#3258)
 * fb502632 feat: Configure ArchiveLabelSelector for Workflow Archive (#3249)
 * 5467c899 fix(controller): set pod finish timestamp when it is deleted (#3230)
 * 04bc5492 build: Disable Circle CI and Sonar (#3253)
 * 23ca07a7 chore: Covered steps.<STEPNAME>.outputs.parameters in variables document (#3245)
 * 4bd33c6c chore(cli): Add examples of @latest alias for relevant commands. Fixes #3225 (#3242)
 * 17108df1 fix: Ensure subscription is closed in log viewer (#3247)
 * 495dc89b docs: Correct available fields in {{workflow.failures}} (#3238)

### Contributors

 * Alex Collins
 * Ben Ye
 * Jie Zhang
 * Pierre Houssin
 * Remington Breeze
 * Saravanan Balasubramanian
 * Simon Behar
 * Yuan Tang
 * mark9white

## v2.9.0-rc2 (2020-06-16)

 * abf02c3b Update manifests to v2.9.0-rc2
 * 8fdbfe58 Merge branch 'master' into release-2.9
 * 4db1c4c8 fix: Support the TTLStrategy for WorkflowTemplateRef (#3239)
 * 47f50693 feat(logging): Made more controller err/warn logging structured (#3240)
 * c25e2880 build: Migrate to Github Actions (#3233)
 * ef159f9a feat: Tick CLI Workflow watch even if there are no new events (#3219)
 * ff1627b7 fix(events): Adds config flag. Reduce number of dupe events emitted. (#3205)
 * eae8f681 feat: Validate CronWorkflows before execution (#3223)
 * 4470a8a2 fix(ui/server): Fix broken label filter functionality on UI due to bug on server. Fix #3226 (#3228)
 * e5e6456b feat(cli): Add --latest flag for argo get command as per #3128 (#3179)
 * 34608594 fix(ui): Correctly update workflow list when workflow are modified/deleted (#3220)
 * a7d8546c feat(controller): Improve throughput of many workflows. Fixes #2908 (#2921)
 * a37d0a72 build: Change "DB=..." to "PROFILE=..." (#3216)
 * 15885d3e feat(sso): Allow reading SSO clientID from a secret. (#3207)
 * 723e9d5f fix: Ensrue image name is present in containers (#3215)

### Contributors

 * Alex Collins
 * Remington Breeze
 * Saravanan Balasubramanian
 * Simon Behar
 * Vlad Losev

## v2.9.0-rc1 (2020-06-10)


### Contributors


## v2.9.0 (2020-07-01)

 * d67d3b1d Update manifests to v2.9.0
 * 9c52c1be fix: Don't double-count metric events (#3350)
 * 813122f7 fix: Fix UI bug in DAGs (#3368)
 * 248643d3 fix: Ensure child pods respect maxDuration (#3280)
 * 71d29584 fix(controller): Allow events to be sent to non-argo namespace. Fixes #3342 (#3345)
 * 52be71bc fix: Remove broken SSO from quick-starts (#3327)
 * 5b109bcb Update manifests to v2.9.0-rc4
 * 36cbcfb6 docs: Document access token creation and usage (#3316)
 * 00ad29bd docs: Rename Ant Financial to Ant Group (#3304)
 * 011f1368 fix(controller): Fix panic logging. (#3315)
 * 9f934496 build: Pin `goimports` to working version (#3311)
 * 888c9b8a ci: Remove CircleCI (#3302)
 * f7e13a3d build: Remove generated Swagger files. (#3297)
 * 5395ad3f Clean up unused constants (#3298)
 * 2bb2cdf0 build: Upload E2E diagnostics after failure (#3294)
 * a2a1fba8 fix: Reapply Update if CronWorkflow resource changed (#3272)
 * 9af98a5b Fixes validation of overridden ref template parameters. (#3286)
 * dc65faa9 docs: Updated WorkflowTemplateRef  on WFT and CWFT (#3137)
 * 86401ba4 docs: ArchiveLabelSelector document (#3284)
 * cce666b7 docs: Add example for handling large output resutls (#3276)
 * a91cea5f fix: Fix delete --complete (#3278)
 * d5a4807a Update releasing.md (#3283)
 * 2e95ff48 Update manifests to v2.9.0-rc3
 * 61ca07e2 Merge branch 'master' into release-2.9
 * acee573b docs: Update CI Badges (#3282)
 * 9eb182c0 build: Allow to change k8s namespace for installation (#3281)
 * 2bcfafb5 fix: Add {{workflow.status}} to workflow-metrics (#3271)
 * e6aab605 fix(jqFilter)!: remove extra quotes around output parameter value (#3251)
 * f4580163 fix(ui): Allow render of templates without entrypoint. Fixes #2891 (#3274)
 * f30c05c7 build: Add warning to ensure 'v' is present on release versions (#3273)
 * d1cb1992 fixed archiveLabelSelector nil (#3270)
 * c7e4c180 fix(ui): Update workflow drawer with new duration format (#3256)
 * f2381a54 fix(controller): More structured logging. Fixes #3260 (#3262)
 * acba084a fix: Avoid unnecessary nil check for annotations of resubmitted workflow (#3268)
 * 55e13705 feat: Append previous workflow name as label to resubmitted workflow (#3261)
 * 2dae7244 feat: Add mode to require Workflows to use workflowTemplateRef (#3149)
 * 56694abe Fixed onexit on workflowtempalteRef (#3263)
 * 54dd72c2 update mysql yaml port (#3258)
 * fb502632 feat: Configure ArchiveLabelSelector for Workflow Archive (#3249)
 * 5467c899 fix(controller): set pod finish timestamp when it is deleted (#3230)
 * 04bc5492 build: Disable Circle CI and Sonar (#3253)
 * 23ca07a7 chore: Covered steps.<STEPNAME>.outputs.parameters in variables document (#3245)
 * 4bd33c6c chore(cli): Add examples of @latest alias for relevant commands. Fixes #3225 (#3242)
 * 17108df1 fix: Ensure subscription is closed in log viewer (#3247)
 * 495dc89b docs: Correct available fields in {{workflow.failures}} (#3238)
 * abf02c3b Update manifests to v2.9.0-rc2
 * 8fdbfe58 Merge branch 'master' into release-2.9
 * 4db1c4c8 fix: Support the TTLStrategy for WorkflowTemplateRef (#3239)
 * 47f50693 feat(logging): Made more controller err/warn logging structured (#3240)
 * c25e2880 build: Migrate to Github Actions (#3233)
 * ef159f9a feat: Tick CLI Workflow watch even if there are no new events (#3219)
 * ff1627b7 fix(events): Adds config flag. Reduce number of dupe events emitted. (#3205)
 * eae8f681 feat: Validate CronWorkflows before execution (#3223)
 * 4470a8a2 fix(ui/server): Fix broken label filter functionality on UI due to bug on server. Fix #3226 (#3228)
 * e5e6456b feat(cli): Add --latest flag for argo get command as per #3128 (#3179)
 * 34608594 fix(ui): Correctly update workflow list when workflow are modified/deleted (#3220)
 * a7d8546c feat(controller): Improve throughput of many workflows. Fixes #2908 (#2921)
 * a37d0a72 build: Change "DB=..." to "PROFILE=..." (#3216)
 * 15885d3e feat(sso): Allow reading SSO clientID from a secret. (#3207)
 * 723e9d5f fix: Ensrue image name is present in containers (#3215)
 * c930d2ec Update manifests to v2.9.0-rc1
 * 0ee5e112 feat: Only process significant pod changes (#3181)
 * c89a81f3 feat: Add '--schedule' flag to 'argo cron create' (#3199)
 * 591f649a refactor: Refactor assesDAGPhase logic (#3035)
 * 285eda6b chore: Remove unused pod in addArchiveLocation() (#3200)
 * 8e1d56cb feat(controller): Add default name for artifact repository ref. (#3060)
 * f1cdba18 feat(controller): Add `--qps` and `--burst` flags to controller (#3180)
 * b86949f0 fix: Ensure stable desc/hash for metrics (#3196)
 * e26d2f08 docs: Update Getting Started (#3099)
 * 47bfea5d docs: Add Graviti as official Argo user (#3187)
 * 04c77f49 fix(server): Allow field selection for workflow-event endpoint (fixes #3163) (#3165)
 * 0c38e66e chore: Update Community Meeting link and specify Go@v1.13 (#3178)
 * 81846d41 build: Only check Dex in hosts file when SSO is enabled (#3177)
 * a130d488 feat(ui): Add drawer with more details for each workflow in Workflow List (#3151)
 * fa84e203 fix: Do not use alphabetical order if index exists (#3174)
 * 138af597 fix(cli): Sort expanded nodes by index. Closes #3145 (#3146)
 * a9ec4d08 docs: Fix api swagger file path in docs (#3167)
 * c42e4d3a feat(metrics): Add node-level resources duration as Argo variable for metrics. Closes #3110 (#3161)
 * e36fe66e docs: Add instructions on using Minikube as an alternative to K3D (#3162)
 * edfa5b93 feat(metrics): Report controller error counters via metrics. Closes #3034 (#3144)
 * 8831e4ea feat(argo-server): Add support for SSO. See #1813 (#2745)
 * b62184c2 feat(cli): More `argo list` and `argo delete` options (#3117)
 * c6565d7c fix(controller): Maybe bug with nil woc.wfSpec. Fixes #3121 (#3160)
 * 06ca71d7 build: Fix path to staticfiles and goreman binaries (#3159)
 * cad84cab chore: Remove unused nodeType in initializeNodeOrMarkError() (#3153)
 * be425513 chore: Master needs lint (#3152)
 * 70b56f25 enhancement(ui): Add workflow labels column to workflow list. Fixes #2782 (#3143)
 * 3318c115 chore: Move default metrics server port/path to consts (#3135)
 * a0062adf feat(ui): Add Alibaba Cloud OSS related models in UI (#3140)
 * 1469991c fix: Update container delete grace period to match Kubernetes default (#3064)
 * df725bbd fix(ui): Input artifacts labelled in UI. Fixes #3098 (#3131)
 * c0d59cc2 feat: Persist DAG rendering options in local storage (#3126)
 * 8715050b fix(ui): Fix label error (#3130)
 * 1814ea2e fix(item): Support ItemValue.Type == List. Fixes #2660 (#3129)
 * 12b72546 fix: Panic on invalid WorkflowTemplateRef (#3127)
 * 09092147 fix(ui): Display error message instead of DAG when DAG cannot be rendered. Fixes #3091 (#3125)
 * 2d9a74de docs: Document cost optimizations. Fixes #1139 (#2972)
 * 69c9e5f0 fix: Remove unnecessary panic (#3123)
 * 2f3aca89 add AppDirect to the list of users (#3124)
 * 257355e4 feat: Add 'submit --from' to CronWorkflow and WorkflowTemplate in UI. Closes #3112 (#3116)
 * 6e5dd2e1 Add Alibaba OSS to the list of supported artifacts (#3108)
 * 1967b45b support sso (#3079)
 * 9229165f feat(ui): Add cost optimisation nudges. (#3089)
 * e88124db fix(controller): Do not panic of woc.orig in not hydrated. Fixes #3118 (#3119)
 * 132b947a fix: Differentiate between Fulfilled and Completed (#3083)
 * a93968ff docs: Document how to backfill a cron workflow (#3094)
 * 4de99746 feat: Added Label selector and Field selector in Argo list  (#3088)
 * 6229353b chore: goimports (#3107)
 * 8491e00f docs: Add link to USERS.md in PR template (#3086)
 * bb2ce9f7 fix: Graceful error handling of malformatted log lines in watch (#3071)
 * 4fd27c31 build(swagger): Fix Swagger build problems (#3084)
 * e4e0dfb6 test: fix TestContinueOnFailDag (#3101)
 * fa69c1bb feat: Add CronWorkflowConditions to report errors (#3055)
 * 50ad3cec adds millisecond-level timestamps to argoexec (#2950)
 * 6464bd19 fix(controller): Implement offloading for workflow updates that are re-applied. Fixes #2856 (#2941)
 * 6c369e61 chore: Rename files that include 'top-level' terminology (#3076)
 * bd40b80b docs: Document work avoidance. (#3066)
 * 6df0b2d3 feat: Support Top level workflow template reference  (#2912)
 * 0709ad28 feat: Enhanced filters for argo {watch,get,submit} (#2450)
 * 784c1385 build: Use goreman for starting locally. (#3074)
 * 5b5bae9a docs: Add Isbank to users.md (#3068)
 * 2b038ed2 feat: Enhanced depends logic (#2673)
 * 4c3387b2 fix: Linters should error if nothing was validated (#3011)
 * 51dd05b5 fix(artifacts): Explicit archive strategy. Fixes #2140 (#3052)
 * ada2209e Revert "fix(artifacts): Allow tar check to be ignored. Fixes #2140 (#3024)" (#3047)
 * b7ff9f09 chore: Add ability to configure maximum DB connection lifetime (#3032)
 * 38a995b7 fix(executor): Properly handle empty resource results, like for a missing get (#3037)
 * a1ac8bcf fix(artifacts): Allow tar check to be ignored. Fixes #2140 (#3024)
 * f12d79ca fix(controller)!: Correctly format workflow.creationTimepstamp as RFC3339. Fixes #2974 (#3023)
 * d10e949a fix: Consider metric nodes that were created and completed in the same operation (#3033)
 * 202d4ab3 fix(executor): Optional input artifacts. Fixes #2990 (#3019)
 * f17e946c fix(executor): Save script results before artifacts in case of error. Fixes #1472 (#3025)
 * 3d216ae6 fix: Consider missing optional input/output artifacts with same name (#3029)
 * 3717dd63 fix: Improve robustness of releases. Fixes #3004 (#3009)
 * 9f86a4e9 feat(ui): Enable CSP, HSTS, X-Frame-Options. Fixes #2760, #1376, #2761 (#2971)
 * cb71d585 refactor(metrics)!: Refactor Metric interface (#2979)
 * c0ee1eb2 docs: Add Ravelin as a user of Argo (#3020)
 * 052e6c51 Fix isTarball to handle the small gzipped file (#3014)
 * cdcba3c4 fix(ui): Displays command args correctl pre-formatted. (#3018)
 * b5160988 build: Mockery v1.1.1 (#3015)
 * a04d8f28 docs: Add StatefulSet and Service doc (#3008)
 * 8412526c docs: Fix Deprecated formatting (#3010)
 * cc0fe433 fix(events): Correct event API Version. Fixes #2994 (#2999)
 * d5d6f750 feat(controller)!: Updates the resource duration calculation. Fixes #2934 (#2937)
 * fa3801a5 feat(ui): Render 2000+ nodes DAG acceptably. (#2959)
 * f952df51 fix(executor/pns): remove sleep before sigkill (#2995)
 * 2a9ee21f feat(ui): Add Suspend and Resume to CronWorkflows in UI (#2982)
 * eefe120f test: Upgrade to argosay:v2 (#3001)
 * 47472f73 chore: Update Mockery (#3000)
 * 46b11e1e docs: Use keyFormat instead of keyPrefix in docs (#2997)
 * 60d5fdc7 fix: Begin counting maxDuration from first child start (#2976)
 * 76aca493 build: Fix Docker build. Fixes #2983 (#2984)
 * d8cb66e7 feat: Add Argo variable {{retries}} to track retry attempt (#2911)
 * 14b7a459 docs: Fix typo with WorkflowTemplates link (#2977)
 * 3c442232 fix: Remove duplicate node event. Fixes #2961 (#2964)
 * d8ab13f2 fix: Consider Shutdown when assesing DAG Phase for incomplete Retry node (#2966)
 * 8a511e10 fix: Nodes with pods deleted out-of-band should be Errored, not Failed (#2855)
 * ca4e08f7 build: Build dev images from cache (#2968)
 * 5f01c4a5 Upgraded to Node 14.0.0 (#2816)
 * 849d876c Fixes error with unknown flag: --show-all (#2960)
 * 93bf6609 fix: Don't update backoff message to save operations (#2951)
 * 3413a5df fix(cli): Remove info logging from watches. Fixes #2955 (#2958)
 * fe9f9019 fix: Display Workflow finish time in UI (#2896)
 * f281199a docs: Update README with new features (#2807)
 * c8bd0bb8 fix(ui): Change default pagination to all and sort workflows (#2943)
 * e3ed686e fix(cli): Re-establish watch on EOF (#2944)
 * 67355372 fix(swagger)!: Fixes invalid K8S definitions in `swagger.json`. Fixes #2888 (#2907)
 * 023f2338 fix(argo-server)!: Implement missing instanceID code. Fixes #2780 (#2786)
 * 7b0739e0 Fix typo (#2939)
 * 20d69c75 Detect ctrl key when a link is clicked (#2935)
 * f32cec31 fix default null value for timestamp column - MySQL 5.7 (#2933)
 * 9773cfeb docs: Add docs/scaling.md (#2918)
 * 99858ea5 feat(controller): Remove the excessive logging of node data (#2925)
 * 03ad694c feat(cli): Refactor `argo list --chunk-size` and add `argo archive list --chunk-size`. Fixes #2820 (#2854)
 * 1c45d5ea test: Use argoproj/argosay:v1 (#2917)
 * f311a5a7 build: Fix Darwin build (#2920)
 * a06cb5e0 fix: remove doubled entry in server cluster role deployment (#2904)
 * c71116dd feat: Windows Container Support. Fixes #1507 and #1383 (#2747)
 * 3afa7b2f fix(ui): Use LogsViewer for container logs (#2825)
 * 9ecd5226 docs: Document node field selector. Closes #2860 (#2882)
 * 7d8818ca fix(controller): Workflow stop and resume by node didn't properly support offloaded nodes. Fixes #2543 (#2548)
 * e013f29d ci: Remove context to stop unauthozied errors on test jobs (#2910)
 * db52e7ba fix(controller): Add mutex to nameEntryIDMap in cron controller. Fix #2638 (#2851)
 * 9a33aa2d docs(users): Adding Habx to the users list (#2781)
 * 9e4ac9b3 feat(cli): Tolerate deleted workflow when running `argo delete`. Fixes #2821 (#2877)
 * a0035dd5 fix: ConfigMap syntax (#2889)
 * c05c3859 ci: Build less and therefore faster (#2839)
 * 56143eb1 feat(ui): Add pagination to workflow list. Fixes #1080 and #976 (#2863)
 * e0ad7de9 test: Fixes various tests (#2874)
 * e378ca47 fix: Cannot create WorkflowTemplate with un-supplied inputs (#2869)
 * c3e30c50 fix(swagger): Generate correct Swagger for inline objects. Fixes #2835 (#2837)
 * c0143d34 feat: Add metric retention policy (#2836)
 * f03cda61 Update getting-started.md (#2872)

### Contributors

 * Adam Gilat
 * Alex Collins
 * Alex Stein
 * Ben Ye
 * Caden
 * Caglar Gulseni
 * Daisuke Taniwaki
 * Daniel Sutton
 * Florent Clairambault
 * Grant Stephens
 * Huan-Cheng Chang
 * Jie Zhang
 * Kannappan Sirchabesan
 * Leonardo Luz
 * Markus Lippert
 * Matt Brant
 * Michael Crenshaw
 * Mike Seddon
 * Pierre Houssin
 * Pradip Caulagi
 * Remington Breeze
 * Romain GUICHARD
 * Saravanan Balasubramanian
 * Sascha Grunert
 * Simon Behar
 * Stephen Steiner
 * Tomas Valasek
 * Vardan Manucharyan
 * Vlad Losev
 * William
 * Youngjoon Lee
 * Yuan Tang
 * Yunhai Luo
 * dmayle
 * maguowei
 * mark9white
 * shibataka000

## v2.8.2 (2020-06-22)

 * c15e817b Update manifests to v2.8.2
 * 8a151aec Update manifests to 2.8.2
 * 123e94ac fix(controller): set pod finish timestamp when it is deleted (#3230)
 * 68a60661 fix: Begin counting maxDuration from first child start (#2976)

### Contributors

 * Jie Zhang
 * Simon Behar

## v2.8.1 (2020-05-28)

 * 0fff4b21 Update manifests to v2.8.1
 * 05dd7862 fix(item): Support ItemValue.Type == List. Fixes #2660 (#3129)
 * def3b97a test: Re-generate mocks
 * 888b2154 test: Add argoproj/argosay:v2 to whitelisted test images
 * baf30725 build: Fix version
 * ac02aa2c build: Fix Makefile `git git tag` typo
 * 3b840201 Fix test
 * 41689c55 fix: Graceful error handling of malformatted log lines in watch (#3071)
 * 79aeca1f fix: Linters should error if nothing was validated (#3011)
 * c977d8bb fix(executor): Properly handle empty resource results, like for a missing get (#3037)
 * 1a01c804 fix: Consider metric nodes that were created and completed in the same operation (#3033)
 * 6065b7ed fix: Consider missing optional input/output artifacts with same name (#3029)
 * acb0f1c1 fix: Cannot create WorkflowTemplate with un-supplied inputs (#2869)
 * 5b04ccce fix(controller)!: Correctly format workflow.creationTimepstamp as RFC3339. Fixes #2974 (#3023)
 * 319ee46d fix(events): Correct event API Version. Fixes #2994 (#2999)
 * be32b207 build: Correct version
 * 8f696174 Update manifests to v2.8.0

### Contributors

 * Alex Collins
 * Saravanan Balasubramanian
 * Simon Behar
 * dmayle

## v2.8.0-rc4 (2020-05-06)

 * ee0dc575 Update manifests to v2.8.0-rc4
 * 3a85610a fix(cli): Remove info logging from watches. Fixes #2955 (#2958)
 * 29c7780d make codegen
 * 265666bf fix(cli): Re-establish watch on EOF (#2944)
 * fef4e968 fix(swagger)!: Fixes invalid K8S definitions in `swagger.json`. Fixes #2888 (#2907)
 * 0f7f9c45 test: Use argoproj/argosay:v1 (#2917)
 * 1ea37284 build: Fix Darwin build (#2920)
 * 54610011 ci: Build less and therefore faster (#2839)
 * 249309aa fix(swagger): Generate correct Swagger for inline objects. Fixes #2835 (#2837)
 * f668c789 docs: Document node field selector. Closes #2860 (#2882)
 * ad28a9c9 fix(controller): Workflow stop and resume by node didn't properly support offloaded nodes. Fixes #2543 (#2548)
 * db4cfc75 ci: Remove context to stop unauthozied errors on test jobs (#2910)
 * d9fca8f0 fix(controller): Add mutex to nameEntryIDMap in cron controller. Fix #2638 (#2851)

### Contributors

 * Alex Collins
 * mark9white
 * shibataka000

## v2.8.0-rc3 (2020-04-28)

 * 2f153b21 Update manifests to v2.8.0-rc3
 * dc7e9984 Merge branch 'master' into release-2.8
 * d66224e1 fix: Don't error when deleting already-deleted WFs (#2866)
 * e84acb50 chore: Display wf.Status.Conditions in CLI (#2858)
 * 3c7f3a07 docs: Fix typo ".yam" -> ".yaml" (#2862)
 * db9e14cf Merge branch 'master' into release-2.8
 * d7f8e0c4 fix(CLI): Re-establish workflow watch on disconnect. Fixes #2796 (#2830)
 * 31358d6e feat(CLI): Add -v and --verbose to Argo CLI (#2814)
 * 1d30f708 ci: Don't configure Sonar on CI for release branches (#2811)
 * d9c54075 docs: Fix exit code example and docs (#2853)
 * 90743353 feat: Expose workflow.serviceAccountName as global variable (#2838)
 * f07f7bf6 note that tar.gz'ing output artifacts is optional (#2797)
 * 3fd3fc6c docs: Document how to label creator (#2827)
 * b956ec65 fix: Add Step node outputs to global scope (#2826)
 * bac339af chore: Configure webpack dev server to proxy using HTTPS (#2812)
 * cc136f9c test: Skip TestStopBehavior. See #2833 (#2834)
 * 52ff43b5 fix: Artifact panic on unknown artifact. Fixes #2824 (#2829)
 * 554fd06c fix: Enforce metric naming validation (#2819)
 * dd223669 docs: Add Microba as official Argo user (#2822)
 * 8151f0c4 docs: Update tls.md (#2813)

### Contributors

 * Alex Collins
 * Fabio Rigato
 * Michael Crenshaw
 * Mike Seddon
 * Simon Behar

## v2.8.0-rc2 (2020-04-23)

 * 4126d22b Update manifests to v2.8.0-rc2
 * ce6b23e9 revert
 * c0cfab52 Merge branch 'master' into release-2.8
 * 0dbd78ff feat: Add TLS support. Closes #2764 (#2766)
 * 510e11b6 fix: Allow empty strings in valueFrom.default (#2805)
 * 399591c9 fix: Don't configure Sonar on CI for release branches
 * d7f41ac8 fix: Print correct version in logs. (#2806)
 * e9c21120 chore: Add GCS native example for output artifact (#2789)
 * e0f2697e fix(controller): Include global params when using withParam (#2757)
 * 3441b11a docs: Fix typo in CronWorkflow doc (#2804)
 * a2d2b848 docs: Add example of recursive for loop (#2801)
 * 29d39e29 docs: Update the contributing docs  (#2791)
 * 1ea286eb fix: ClusterWorkflowTemplate RBAC for  argo server (#2753)
 * 1f14f2a5 feat(archive): Implement data retention. Closes #2273 (#2312)
 * d0cc7764 feat: Display argo-server version in `argo version` and in UI. (#2740)
 * 8de57281 feat(controller): adds Kubernetes node name to workflow node detail in web UI and CLI output. Implements #2540 (#2732)
 * 52fa5fde MySQL config fix (#2681)
 * 43d9eebb fix: Rename Submittable API endpoint to `submit` (#2778)
 * 69333a87 Fix template scope tests (#2779)
 * bb1abf7f chore: Add CODEOWNERS file (#2776)
 * 905e0b99 fix: Naming error in Makefile (#2774)
 * 7cb2fd17 fix: allow non path output params (#2680)

### Contributors

 * Alex Collins
 * Alex Stein
 * Daisuke Taniwaki
 * Fabio Rigato
 * Kannappan Sirchabesan
 * Michael Crenshaw
 * Saravanan Balasubramanian
 * Simon Behar

## v2.8.0-rc1 (2020-04-20)


### Contributors


## v2.8.0 (2020-05-11)

 * 8f696174 Update manifests to v2.8.0
 * ee0dc575 Update manifests to v2.8.0-rc4
 * 3a85610a fix(cli): Remove info logging from watches. Fixes #2955 (#2958)
 * 29c7780d make codegen
 * 265666bf fix(cli): Re-establish watch on EOF (#2944)
 * fef4e968 fix(swagger)!: Fixes invalid K8S definitions in `swagger.json`. Fixes #2888 (#2907)
 * 0f7f9c45 test: Use argoproj/argosay:v1 (#2917)
 * 1ea37284 build: Fix Darwin build (#2920)
 * 54610011 ci: Build less and therefore faster (#2839)
 * 249309aa fix(swagger): Generate correct Swagger for inline objects. Fixes #2835 (#2837)
 * f668c789 docs: Document node field selector. Closes #2860 (#2882)
 * ad28a9c9 fix(controller): Workflow stop and resume by node didn't properly support offloaded nodes. Fixes #2543 (#2548)
 * db4cfc75 ci: Remove context to stop unauthozied errors on test jobs (#2910)
 * d9fca8f0 fix(controller): Add mutex to nameEntryIDMap in cron controller. Fix #2638 (#2851)
 * 2f153b21 Update manifests to v2.8.0-rc3
 * dc7e9984 Merge branch 'master' into release-2.8
 * d66224e1 fix: Don't error when deleting already-deleted WFs (#2866)
 * e84acb50 chore: Display wf.Status.Conditions in CLI (#2858)
 * 3c7f3a07 docs: Fix typo ".yam" -> ".yaml" (#2862)
 * db9e14cf Merge branch 'master' into release-2.8
 * d7f8e0c4 fix(CLI): Re-establish workflow watch on disconnect. Fixes #2796 (#2830)
 * 31358d6e feat(CLI): Add -v and --verbose to Argo CLI (#2814)
 * 1d30f708 ci: Don't configure Sonar on CI for release branches (#2811)
 * d9c54075 docs: Fix exit code example and docs (#2853)
 * 90743353 feat: Expose workflow.serviceAccountName as global variable (#2838)
 * f07f7bf6 note that tar.gz'ing output artifacts is optional (#2797)
 * 3fd3fc6c docs: Document how to label creator (#2827)
 * b956ec65 fix: Add Step node outputs to global scope (#2826)
 * bac339af chore: Configure webpack dev server to proxy using HTTPS (#2812)
 * cc136f9c test: Skip TestStopBehavior. See #2833 (#2834)
 * 52ff43b5 fix: Artifact panic on unknown artifact. Fixes #2824 (#2829)
 * 554fd06c fix: Enforce metric naming validation (#2819)
 * dd223669 docs: Add Microba as official Argo user (#2822)
 * 8151f0c4 docs: Update tls.md (#2813)
 * 4126d22b Update manifests to v2.8.0-rc2
 * ce6b23e9 revert
 * c0cfab52 Merge branch 'master' into release-2.8
 * 0dbd78ff feat: Add TLS support. Closes #2764 (#2766)
 * 510e11b6 fix: Allow empty strings in valueFrom.default (#2805)
 * 399591c9 fix: Don't configure Sonar on CI for release branches
 * d7f41ac8 fix: Print correct version in logs. (#2806)
 * e9c21120 chore: Add GCS native example for output artifact (#2789)
 * e0f2697e fix(controller): Include global params when using withParam (#2757)
 * 3441b11a docs: Fix typo in CronWorkflow doc (#2804)
 * a2d2b848 docs: Add example of recursive for loop (#2801)
 * 29d39e29 docs: Update the contributing docs  (#2791)
 * 1ea286eb fix: ClusterWorkflowTemplate RBAC for  argo server (#2753)
 * 1f14f2a5 feat(archive): Implement data retention. Closes #2273 (#2312)
 * d0cc7764 feat: Display argo-server version in `argo version` and in UI. (#2740)
 * 8de57281 feat(controller): adds Kubernetes node name to workflow node detail in web UI and CLI output. Implements #2540 (#2732)
 * 52fa5fde MySQL config fix (#2681)
 * 43d9eebb fix: Rename Submittable API endpoint to `submit` (#2778)
 * 69333a87 Fix template scope tests (#2779)
 * bb1abf7f chore: Add CODEOWNERS file (#2776)
 * 905e0b99 fix: Naming error in Makefile (#2774)
 * 7cb2fd17 fix: allow non path output params (#2680)
 * 4a73f45c Update manifests to v2.8.0-rc1
 * af9f61ea ci: Recurl (#2769)
 * ef08e642 build: Retry curl 3x (#2768)
 * dedec906 test: Get tests running on release branches (#2767)
 * 1c8318eb fix: Add compatiblity mode to templateReference (#2765)
 * 7975952b fix: Consider expanded tasks in getTaskFromNode (#2756)
 * bc421380 fix: Fix template resolution in UI (#2754)
 * 391c0f78 Make phase and templateRef available for unsuspend and retry selectors (#2723)
 * a6fa3f71 fix: Improve cookie security. Fixes #2759 (#2763)
 * 57f0183c Fix typo on the documentation. It causes error unmarshaling JSON: while (#2730)
 * c6ef1ff1 feat(manifests): add name on workflow-controller-metrics service port (#2744)
 * af5cd1ae docs: Update OWNERS (#2750)
 * 06c4bd60 fix: Make ClusterWorkflowTemplate optional for namespaced Installation (#2670)
 * 25c62463 docs: Update README (#2752)
 * 908e1685 docs: Update README.md (#2751)
 * 4ea43e2d fix: Children of onExit nodes are also onExit nodes (#2722)
 * 3f1b6667 feat: Add Kustomize as supported install option. Closes #2715 (#2724)
 * 691459ed fix: Error pending nodes w/o Pods unless resubmitPendingPods is set (#2721)
 * 874d8776 test: Longer timeout for deletion (#2737)
 * 3c8149fa Fix typo (#2741)
 * 98f60e79 feat: Added Workflow SubmitFromResource API (#2544)
 * 6253997a fix: Reset all conditions when resubmitting (#2702)
 * e7c67de3 fix: Maybe fix watch. Fixes #2678 (#2719)
 * cef6dfb6 fix: Print correct version string. (#2713)
 * e9589d28 feat: Increase pod workers and workflow workers both to 32 by default. (#2705)
 * 3a1990e0 test: Fix Goroutine leak that was making controller unit tests slow. (#2701)
 * 9894c29f ci: Fix Sonar analysis on master. (#2709)
 * 54f5be36 style: Camelcase "clusterScope" (#2720)
 * db6d1416 fix: Flakey TestNestedClusterWorkflowTemplate testcase failure (#2613)
 * b4fd4475 feat(ui): Add a YAML panel to view the workflow manifest. (#2700)
 * 65d413e5 build(ui): Fix compression of UI package. (#2704)
 * 4129528d fix: Don't use docker cache when building release images (#2707)
 * 8d0956c9 test: Increase runCli timeout to 1m (#2703)
 * 9d93e971 Update getting-started.md (#2697)
 * ee644a35 docs: Fix CONTRIBUTING.md and running-locally.md. Fixes #2682 (#2699)
 * 2737c0ab feat: Allow to pass optional flags to resource template (#1779)
 * c1a2fc7c Update running-locally.md - fixing incorrect protoc install (#2689)
 * a1226c46 fix: Enhanced WorkflowTemplate and ClusterWorkflowTemplate validation to support Global Variables   (#2644)
 * c21cc2f3 fix a typo (#2669)
 * 9430a513 fix: Namespace-related validation in UI (#2686)
 * f3eeca6e feat: Add exit code as output variable (#2111)
 * 9f95e23a fix: Report metric emission errors via Conditions (#2676)
 * c67f5ff5 fix: Leaf task with continueOn should not fail DAG (#2668)
 * 3c20d4c0 ci: Migrate to use Sonar instead of CodeCov for analysis (#2666)
 * 9c6351fa feat: Allow step restart on workflow retry. Closes #2334 (#2431)
 * cf277eb5 docs: Updates docs for CII. See #2641 (#2643)
 * e2d0aa23 fix: Consider offloaded and compressed node in retry and resume (#2645)
 * a25c6a20 build: Fix codegen for releases (#2662)
 * 4a3ca930 fix: Correctly emit events. Fixes #2626 (#2629)
 * 4a7d4bdb test: Fix flakey DeleteCompleted test (#2659)
 * 41f91e18 fix: Add DAG as default in UI filter and reorder (#2661)
 * f138ada6 fix: DAG should not fail if its tasks have continueOn (#2656)
 * e5cbdf6a ci: Only run CI jobs if needed (#2655)
 * 4c452d5f fix: Don't attempt to resolve artifacts if task is going to be skipped (#2657)
 * 2caf570a chore: Add newline to fields.md (#2654)
 * 2cb596da Storage region should be specified (#2538)
 * 271e4551 chore: Fix-up Yarn deps (#2649)
 * 4c1b0777 fix: Sort log entries. (#2647)
 * 268fc461  docs: Added doc generator code (#2632)
 * d58b7fc3 fix: Add input paremeters to metric scope (#2646)
 * cc3af0b8 fix: Validating Item Param in Steps Template (#2608)
 * 6c685c5b fix: allow onExit to run if wf exceeds activeDeadlineSeconds. Fixes #2603 (#2605)
 * ffc43ce9 feat: Added Client validation on Workflow/WFT/CronWF/CWFT (#2612)
 * 24655cd9 feat(UI): Move Workflow parameters to top of submit (#2640)
 * 0a3b159a Use error equals (#2636)
 * 8c29e05c ci: Fix codegen job (#2648)
 * a78ecb7f docs(users): Add CoreWeave and ConciergeRender (#2641)
 * 14be4670 fix: Fix logs part 2 (#2639)
 * 4da6f4f3 feat: Add 'outputs.result' to Container templates (#2584)
 * 51bc876d test: Fixes TestCreateWorkflowDryRun. Closes #2618 (#2628)
 * 212c6d75 fix: Support minimal mysql version 5.7.8 (#2633)
 * 8facacee refactor: Refactor Template context interfaces (#2573)
 * 812813a2 fix: fix test cases (#2631)
 * ed028b25 fix: Fix logging problems. See #2589 (#2595)
 * d4e81238 test: Fix teething problems (#2630)
 * 4aad6d55 chore: Add comments to issues (#2627)
 * 54f7a013 test: Enhancements and repairs to e2e test framework (#2609)
 * d95926fe fix: Fix WorkflowTemplate icons to be more cohesive (#2607)
 * 0130e1fd docs: Add fields and core concepts doc (#2610)
 * 5a1ac203 fix: Fixes panic in toWorkflow method (#2604)
 * 51910292 chore: Lint UI on CI, test diagnostics, skip bad test (#2587)
 * 232bb115 fix(error handling): use Errorf instead of New when throwing errors with formatted text (#2598)
 * eeb2f97b fix(controller): dag continue on failed. Fixes #2596 (#2597)
 * 99c35129 docs: Fix inaccurate field name in docs (#2591)
 * 21c73779 fix: Fixes lint errors (#2594)
 * 38aca5fa chore: Added ClusterWorkflowTemplate RBAC on quick-start manifests (#2576)
 * 59f746e1 feat: UI enhancement for Cluster Workflow Template (#2525)
 * 0801a428 fix(cli): Show lint errors of all files (#2552)
 * c3535ba5 docs: Fix wrong Configuring Your Artifact Repository document. (#2586)
 * 79217bc8 feat(archive): allow specifying a compression level (#2575)
 * 88d261d7 fix: Use outputs of last child instead of retry node itslef (#2565)
 * 5c08292e style: Correct the confused logic (#2577)
 * 3d146144 fix: Fix bug in deleting pods. Fixes #2571 (#2572)
 * cb739a68 feat: Cluster scoped workflow template (#2451)
 * c63e3d40 feat: Show workflow duration in the index UI page (#2568)
 * 1520452a chore: Error -> Warn when Parent CronWf no longer exists (#2566)
 * ffbb3b89 fix: Fixes empty/missing CM. Fixes #2285 (#2562)
 * d0fba6f4 chore: fix typos in the workflow template docs (#2563)
 * 49801e32 chore(docker): upgrade base image for executor image (#2561)
 * c4efb8f8 Add Riskified to the user list (#2558)
 * 8b92d33e feat: Create K8S events on node completion. Closes #2274 (#2521)

### Contributors

 * Adam Gilat
 * Alex Collins
 * Alex Stein
 * CWen
 * Daisuke Taniwaki
 * Derek Wang
 * Dustin Specker
 * Ed Lee
 * Ejiah
 * Fabio Rigato
 * Gabriele Santomaggio
 * Heikki Kesa
 * Kannappan Sirchabesan
 * Marek Čermák
 * Michael Crenshaw
 * Mike Seddon
 * Niklas Hansson
 * Omer Kahani
 * Peng Li
 * Peter Salanki
 * Romain Di Giorgio
 * Saradhi Sreegiriraju
 * Saravanan Balasubramanian
 * Simon Behar
 * Song Juchao
 * Vardan Manucharyan
 * Wei Yan
 * dherman
 * lueenavarro
 * mark9white
 * shibataka000
 * tunoat

## v2.7.7 (2020-05-06)

 * 54154c61 Update manifests to v2.7.7
 * 1254dd44 fix(cli): Re-establish watch on EOF (#2944)
 * 42d622b6 fix(controller): Add mutex to nameEntryIDMap in cron controller. Fix #2638 (#2851)
 * 51ce1063 fix: Print correct version in logs. (#2806)

### Contributors

 * Alex Collins
 * shibataka000

## v2.7.6 (2020-04-28)

 * 70facdb6 Update manifests to v2.7.6
 * 15f0d741 Fix TestGlobalScope
 * 3a906e65 Fix build
 * b6022a9b fix(controller): Include global params when using withParam (#2757)
 * 728287e8 fix: allow non path output params (#2680)
 * 83fa9406 fix: Add Step node outputs to global scope (#2826)
 * 462f6af0 fix: Enforce metric naming validation (#2819)
 * ed9f87c5 fix: Allow empty strings in valueFrom.default (#2805)
 * 4d1690c4 fix: Children of onExit nodes are also onExit nodes (#2722)
 * 7452c868 ci: Recurl (#2769)
 * d40036c3 fix(CLI): Re-establish workflow watch on disconnect. Fixes #2796 (#2830)
 * 01196831 test: Skip TestStopBehavior. See #2833 (#2834)
 * f1a331a1 fix: Artifact panic on unknown artifact. Fixes #2824 (#2829)

### Contributors

 * Alex Collins
 * Daisuke Taniwaki
 * Simon Behar

## v2.7.5 (2020-04-20)

 * ede163e1 Update manifests to v2.7.5
 * ab18ab4c Hard-code build opts
 * ca77a5e6 Resolve conflicts
 * dacfa20f fix: Error pending nodes w/o Pods unless resubmitPendingPods is set (#2721)
 * 2880f294 build: Retry curl 3x (#2768)
 * e014c6e0 Run make manifests
 * 42f49582 test: Get tests running on release branches (#2767)
 * ee107969 fix: Improve cookie security. Fixes #2759 (#2763)
 * e8cd8d77 fix: Consider expanded tasks in getTaskFromNode (#2756)
 * ca5cdc47 fix: Reset all conditions when resubmitting (#2702)
 * 80dd96af feat: Add Kustomize as supported install option. Closes #2715 (#2724)
 * 306a1189 fix: Maybe fix watch. Fixes #2678 (#2719)
 * 5b05519d fix: Print correct version string. (#2713)

### Contributors

 * Alex Collins
 * Simon Behar

## v2.7.4 (2020-04-16)

 * 50b209ca Update manifests to v2.7.4
 * a8ecd513 chore(docker): upgrade base image for executor image (#2561)

### Contributors

 * Dustin Specker
 * Simon Behar

## v2.7.3 (2020-04-15)

 * 66bd0425 go mod tidy
 * a8cd8b83 Update manifests to v2.7.3
 * b879f5c6 fix: Don't use docker cache when building release images (#2707)
 * 60fe5bd3 fix: Report metric emission errors via Conditions (#2676)
 * 04f79f2b fix: Leaf task with continueOn should not fail DAG (#2668)

### Contributors

 * Alex Collins
 * Simon Behar

## v2.7.2 (2020-04-10)

 * c52a65aa Update manifests to v2.7.2
 * 180f9e4d fix: Consider offloaded and compressed node in retry and resume (#2645)
 * a28fc4fb fix: allow onExit to run if wf exceeds activeDeadlineSeconds. Fixes #2603 (#2605)
 * 6983e56b fix: Support minimal mysql version 5.7.8 (#2633)
 * 4ab86db5 build: Fix codegen for releases (#2662)
 * f99fa50f fix: Add DAG as default in UI filter and reorder (#2661)
 * 0a2c0d1a fix: DAG should not fail if its tasks have continueOn (#2656)
 * b7a8f6e6 fix: Don't attempt to resolve artifacts if task is going to be skipped (#2657)
 * 910db665 fix: Add input paremeters to metric scope (#2646)
 * c17aeda0 test: Fix flakey DeleteCompleted test (#2659)
 * b19753e3 ci: Fix codegen job (#2648)
 * 05e5ce6d fix: Sort log entries. (#2647)
 * b35f2337 fix: Fix logs part 2 (#2639)
 * 733ace4d fix: Fix logging problems. See #2589 (#2595)
 * bdef4e6e test: Fix teething problems (#2630)
 * e99309b8 remove file
 * 83cca179 chore: Add comments to issues (#2627)
 * d608ff7d test: Enhancements and repairs to e2e test framework (#2609)

### Contributors

 * Alex Collins
 * Derek Wang
 * Simon Behar
 * mark9white

## v2.7.1 (2020-04-07)

 * 2a3f59c1 Update manifests to v2.7.1
 * 25f673df fix: Fixes panic in toWorkflow method (#2604)
 * 8c799b1f make codegen
 * 8488033f chore: Lint UI on CI, test diagnostics, skip bad test (#2587)
 * d02c4620 fix(error handling): use Errorf instead of New when throwing errors with formatted text (#2598)
 * c0d50ca2 fix(controller): dag continue on failed. Fixes #2596 (#2597)
 * 9ebe026a docs: Fix inaccurate field name in docs (#2591)
 * 12ac3387 fix: Fixes lint errors (#2594)
 * fd49ef2d fix(cli): Show lint errors of all files (#2552)
 * e697dbc5 fix: Use outputs of last child instead of retry node itslef (#2565)
 * 7623a4f3 style: Correct the confused logic (#2577)
 * f619f8ff fix: Fix bug in deleting pods. Fixes #2571 (#2572)
 * 4c623bee feat: Show workflow duration in the index UI page (#2568)
 * 670b1653 chore: Error -> Warn when Parent CronWf no longer exists (#2566)
 * 49c00156 Merge commit '2902e144ddba2f8c5a93cdfc8e2437c04705065b' into release-2.7
 * f97be738 fix: Fixes empty/missing CM. Fixes #2285 (#2562)
 * 4d1175eb Update manifests to v2.7.0
 * 2902e144 feat: Add node type and phase filter to UI (#2555)
 * fb74ba1c fix: Separate global scope processing from local scope building (#2528)
 * be91ea01 Merge branch 'master' into release-2.7
 * 618b6dee fix: Fixes --kubeconfig flag. Fixes #2492 (#2553)

### Contributors

 * Alex Collins
 * Heikki Kesa
 * Niklas Hansson
 * Peng Li
 * Simon Behar
 * Vardan Manucharyan
 * Wei Yan

## v2.7.0-rc4 (2020-03-30)

 * 479fa48a Update manifests to v2.7.0-rc4
 * 7431b7ff Merge branch 'master' into release-2.7
 * 79dc969f test: Increase timeout for flaky test (#2543)
 * 687cb73a Merge branch 'master' into release-2.7
 * 15a3c990 feat: Report SpecWarnings in status.conditions (#2541)
 * f142f30a docs: Add example of template-level volume declaration. (#2542)
 * 33156a9c Merge branch 'master' into release-2.7
 * 93b6be61 fix(archive): Fix bug that prevents listing archive workflows. Fixes … (#2523)
 * b4c9c54f fix: Omit config key in configure artifact document. (#2539)
 * 864bf1e5 fix: Show template on its own field in CLI (#2535)
 * 555aaf06 test: fix master (#2534)
 * 94862b98 chore: Remove deprecated example (#2533)
 * 5e1e7829 fix: Validate CronWorkflow before creation (#2532)
 * c9241339 fix: Fix wrong assertions (#2531)
 * 67fe04bb Revert "fix: fix template scope tests (#2498)" (#2526)
 * ddfa1ad0 docs: couple of examples for REST API usage of argo-server (#2519)
 * 30542be7 chore(docs): Update docs for useSDKCreds (#2518)
 * e2cc6988 feat: More control over resuming suspended nodes Fixes #1893 (#1904)
 * b2771249 chore: minor fix and refactory (#2517)
 * b1ad163a fix: fix template scope tests (#2498)

### Contributors

 * Alex Collins
 * Daisuke Taniwaki
 * Daniel Moran
 * Derek Wang
 * Ejiah
 * Simon Behar
 * Zach Aller
 * mark9white
 * maryoush

## v2.7.0-rc3 (2020-03-25)

 * 2bb0a7a4 Update manifests to v2.7.0-rc3
 * d2a50750 Merge branch 'master' into release-2.7
 * 661d1b67 Increase client gRPC max size to match server (#2514)
 * d8aa477f fix: Fix potential panic (#2516)
 * 1afb692e fix: Allow runtime resolution for workflow parameter names (#2501)
 * 243ea338 fix(controller): Ensure we copy any executor securityContext when creating wait containers; fixes #2512 (#2510)
 * 6e8c7bad feat: Extend workflowDefaults to full Workflow and clean up docs and code (#2508)
 * 06cfc129 feat: Native Google Cloud Storage support for artifact. Closes #1911 (#2484)
 * 999b1e1d  fix: Read ConfigMap before starting servers  (#2507)
 * 3d6e9b61 docs: Add separate ConfigMap doc for 2.7+ (#2505)
 * e5bd6a7e fix(controller): Updates GetTaskAncestry to skip visited nod. Fixes #1907 (#1908)
 * 183a29e4 docs: add official user (#2499)
 * e636000b feat: Updated arm64 support patch (#2491)
 * 559cb005 feat(ui): Report resources duration in UI. Closes #2460 (#2489)
 * 09291d9d feat: Add default field in parameters.valueFrom (#2500)
 * 33cd4f2b feat(config): Make configuration mangement easier. Closes #2463 (#2464)

### Contributors

 * Alex Collins
 * Derek Wang
 * Rafael Rodrigues
 * Simon Behar
 * StoneHuang
 * Xin Wang
 * mark9white
 * vatine

## v2.7.0-rc2 (2020-03-23)

 * 240d7ad9 Update manifests to v2.7.0-rc2
 * 3bd1c4fc Merge branch 'master' into release-2.7
 * f3df9660 test: Fix test (#2490)
 * bfaf1c21 chore: Move quickstart Prometheus port to 9090 (#2487)
 * 487ed425 feat: Logging the Pod Spec in controller log (#2476)
 * 96c80e3e fix(cli): Rearrange the order of chunk size argument in list command. Closes #2420 (#2485)
 * 47bd70a0 chore: Fix Swagger for PDB to support Java client (#2483)
 * 53a10564 feat(usage): Report resource duration. Closes #1066 (#2219)
 * 063d9bc6 Revert "feat: Add support for arm64 platform (#2364)" (#2482)
 * 735d25e9 fix: Build image with SHA tag when a git tag is not available (#2479)
 * c55bb3b2 ci: Run lint on CI and fix GolangCI (#2470)
 * e1c9f7af fix ParallelSteps child type so replacements happen correctly; fixes argoproj-labs/argo-client-gen#5 (#2478)
 * 55c315db feat: Add support for IRSA and aws default provider chain. (#2468)
 * c724c7c1 feat: Add support for arm64 platform (#2364)
 * 315dc164 feat: search archived wf by startat. Closes #2436 (#2473)

### Contributors

 * Alex Collins
 * Derek Wang
 * Huan-Cheng Chang
 * Michael Crenshaw
 * Saravanan Balasubramanian
 * Simon Behar
 * Xin Wang
 * Zach Aller

## v2.7.0-rc1 (2020-03-18)


### Contributors


## v2.7.0 (2020-03-31)

 * 4d1175eb Update manifests to v2.7.0
 * be91ea01 Merge branch 'master' into release-2.7
 * 618b6dee fix: Fixes --kubeconfig flag. Fixes #2492 (#2553)
 * 479fa48a Update manifests to v2.7.0-rc4
 * 7431b7ff Merge branch 'master' into release-2.7
 * 79dc969f test: Increase timeout for flaky test (#2543)
 * 687cb73a Merge branch 'master' into release-2.7
 * 15a3c990 feat: Report SpecWarnings in status.conditions (#2541)
 * f142f30a docs: Add example of template-level volume declaration. (#2542)
 * 33156a9c Merge branch 'master' into release-2.7
 * 93b6be61 fix(archive): Fix bug that prevents listing archive workflows. Fixes … (#2523)
 * b4c9c54f fix: Omit config key in configure artifact document. (#2539)
 * 864bf1e5 fix: Show template on its own field in CLI (#2535)
 * 555aaf06 test: fix master (#2534)
 * 94862b98 chore: Remove deprecated example (#2533)
 * 5e1e7829 fix: Validate CronWorkflow before creation (#2532)
 * c9241339 fix: Fix wrong assertions (#2531)
 * 67fe04bb Revert "fix: fix template scope tests (#2498)" (#2526)
 * ddfa1ad0 docs: couple of examples for REST API usage of argo-server (#2519)
 * 30542be7 chore(docs): Update docs for useSDKCreds (#2518)
 * e2cc6988 feat: More control over resuming suspended nodes Fixes #1893 (#1904)
 * b2771249 chore: minor fix and refactory (#2517)
 * 2bb0a7a4 Update manifests to v2.7.0-rc3
 * b1ad163a fix: fix template scope tests (#2498)
 * d2a50750 Merge branch 'master' into release-2.7
 * 661d1b67 Increase client gRPC max size to match server (#2514)
 * d8aa477f fix: Fix potential panic (#2516)
 * 1afb692e fix: Allow runtime resolution for workflow parameter names (#2501)
 * 243ea338 fix(controller): Ensure we copy any executor securityContext when creating wait containers; fixes #2512 (#2510)
 * 6e8c7bad feat: Extend workflowDefaults to full Workflow and clean up docs and code (#2508)
 * 06cfc129 feat: Native Google Cloud Storage support for artifact. Closes #1911 (#2484)
 * 999b1e1d  fix: Read ConfigMap before starting servers  (#2507)
 * 3d6e9b61 docs: Add separate ConfigMap doc for 2.7+ (#2505)
 * e5bd6a7e fix(controller): Updates GetTaskAncestry to skip visited nod. Fixes #1907 (#1908)
 * 183a29e4 docs: add official user (#2499)
 * e636000b feat: Updated arm64 support patch (#2491)
 * 559cb005 feat(ui): Report resources duration in UI. Closes #2460 (#2489)
 * 09291d9d feat: Add default field in parameters.valueFrom (#2500)
 * 33cd4f2b feat(config): Make configuration mangement easier. Closes #2463 (#2464)
 * 240d7ad9 Update manifests to v2.7.0-rc2
 * 3bd1c4fc Merge branch 'master' into release-2.7
 * f3df9660 test: Fix test (#2490)
 * bfaf1c21 chore: Move quickstart Prometheus port to 9090 (#2487)
 * 487ed425 feat: Logging the Pod Spec in controller log (#2476)
 * 96c80e3e fix(cli): Rearrange the order of chunk size argument in list command. Closes #2420 (#2485)
 * 47bd70a0 chore: Fix Swagger for PDB to support Java client (#2483)
 * 53a10564 feat(usage): Report resource duration. Closes #1066 (#2219)
 * 063d9bc6 Revert "feat: Add support for arm64 platform (#2364)" (#2482)
 * 735d25e9 fix: Build image with SHA tag when a git tag is not available (#2479)
 * c55bb3b2 ci: Run lint on CI and fix GolangCI (#2470)
 * e1c9f7af fix ParallelSteps child type so replacements happen correctly; fixes argoproj-labs/argo-client-gen#5 (#2478)
 * 55c315db feat: Add support for IRSA and aws default provider chain. (#2468)
 * c724c7c1 feat: Add support for arm64 platform (#2364)
 * 315dc164 feat: search archived wf by startat. Closes #2436 (#2473)
 * 55702224 Update manifests to v2.7.0-rc1
 * 23d230bd feat(ui): add Env to Node Container Info pane. Closes #2471 (#2472)
 * 10a0789b fix: ParallelSteps swagger.json (#2459)
 * a59428e7 fix: Duration must be a string. Make it a string. (#2467)
 * 47bc6f3b feat: Add `argo stop` command (#2352)
 * 14478bc0 feat(ui): Add the ability to have links to logging facility in UI. Closes #2438 (#2443)
 * 2864c745 chore: make codegen + make start (#2465)
 * a85f62c5 feat: Custom, step-level, and usage metrics (#2254)
 * 64ac0298 fix: Deprecate template.{template,templateRef,arguments} (#2447)
 * 6cb79e4e fix: Postgres persistence SSL Mode (#1866) (#1867)
 * 2205c0e1 fix(controller): Updates to add condition to workflow status. Fixes #2421 (#2453)
 * 9d96ab2f fix: make dir if needed (#2455)
 * 5346609e test: Maybe fix TestPendingRetryWorkflowWithRetryStrategy. Fixes #2446 (#2456)
 * 3448ccf9 fix: Delete PVCs unless WF Failed/Errored (#2449)
 * 782bc8e7 fix: Don't error when optional artifacts are not found (#2445)
 * fc18f3cf chore: Master needs codegen (#2448)
 * 32fc2f78 feat: Support workflow templates submission. Closes #2007 (#2222)
 * 050a143d fix(archive): Fix edge-cast error for archiving. Fixes #2427 (#2434)
 * 9455c1b8 doc: update CHANGELOG.md (#2425)
 * 1baa7ee4 feat(ui): cache namespace selection. Closes #2439 (#2441)
 * 91d29881 feat: Retry pending nodes (#2385)
 * 7094433e test: Skip flakey tests in operator_template_scope_test.go. See #2432 (#2433)
 * 30332b14 fix: Allow numbers in steps.args.params.value (#2414)
 * e9a06dde feat: instanceID support for argo server. Closes #2004 (#2365)
 * 3f8be0cd fix "Unable to retry workflow" on argo-server (#2409)
 * dd3029ab docs: Example showing how to use default settings for workflow spec. Related to ##2388 (#2411)
 * 13508828 fix: Check child node status before backoff in retry (#2407)
 * b59419c9 fix: Build with the correct version if you check out a specific version (#2423)
 * 6d834d54 chore: document BASE_HREF (#2418)
 * 184c3653 fix: Remove lazy workflow template (#2417)
 * 918d0d17 docs: Added Survey Results (#2416)
 * 20d6e27b Update CONTRIBUTING.md (#2410)
 * f2ca045e feat: Allow WF metadata spec on Cron WF (#2400)
 * 068a4336 fix: Correctly report version. Fixes #2374 (#2402)
 * e19a398c Update pull_request_template.md (#2401)
 * 7c99c109 chore: Fix typo (#2405)
 * 175b164c Change font family for class yaml (#2394)
 * d1194755 fix: Don't display Retry Nodes in UI if len(children) == 1 (#2390)
 * b8623ec7 docs: Create USERS.md (#2389)
 * 1d21d3f5 fix(doc strings): Fix bug related documentation/clean up of default configurations #2331 (#2388)
 * 77e11fc4 chore: add noindex meta tag to solve #2381; add kustomize to build docs (#2383)
 * 42200fad fix(controller): Mount volumes defined in script templates. Closes #1722 (#2377)
 * 96af36d8 fix: duration must be a string (#2380)
 * 7bf08192 fix: Say no logs were outputted when pod is done (#2373)
 * 847c3507 fix(ui): Removed tailLines from EventSource (#2330)
 * 3890a124 feat: Allow for setting default configurations for workflows, Fixes #1923, #2044 (#2331)
 * 81ab5385 Update readme (#2379)
 * 91810273 feat: Log version (structured) on component start-up (#2375)
 * d0572a74 docs: Make Getting Started agnostic to version (#2371)
 * d3a3f6b1 docs: Add Prudential to the users list (#2353)
 * 4714c880 chore: Master needs codegen (#2369)
 * 5b6b8257 fix(docker): fix streaming of combined stdout/stderr (#2368)
 * 97438313 fix: Restart server ConfigMap watch when closed (#2360)
 * 64d0cec0 chore: Master needs make lint (#2361)
 * 12386fc6 fix: rerun codegen after merging OSS artifact support (#2357)
 * 40586ed5 fix: Always validate templates (#2342)
 * 897db894 feat: Add support for Alibaba Cloud OSS artifact (#1919)
 * 7e2dba03 feat(ui): Circles for nodes (#2349)
 * e85f6169 chore: update getting started guide to use 2.6.0 (#2350)
 * 7ae4ec78 docker: remove NopCloser from the executor. (#2345)
 * 5895b364 feat: Expose workflow.paramteres with JSON string of all params (#2341)
 * a9850b43 Fix the default (#2346)
 * c3763d34 fix: Simplify completion detection logic in DAGs (#2344)
 * d8a9ea09 fix(auth): Fixed returning  expired  Auth token for GKE (#2327)
 * 6fef0454 fix: Add timezone support to startingDeadlineSeconds (#2335)
 * c28731b9 chore: Add go mod tidy to codegen (#2332)
 * a66c8802 feat: Allow Worfklows to be submitted as files from UI (#2340)
 * a9c1d547 docs: Update Argo Rollouts description (#2336)
 * 8672b97f fix(Dockerfile): Using `--no-install-recommends` (Optimization) (#2329)
 * c3fe1ae1 fix(ui): fixed worflow UI refresh. Fixes ##2337 (#2338)
 * d7690e32 feat(ui): Adds ability zoom and hide successful steps. POC (#2319)
 * e9e13d4c feat: Allow retry strategy on non-leaf nodes, eg for step groups. Fixes #1891 (#1892)
 * 62e6db82 feat: Ability to include or exclude fields in the response (#2326)
 * 52ba89ad fix(swagger): Fix the broken swagger. (#2317)
 * efb8a1ac docs: Update CODE_OF_CONDUCT.md (#2323)
 * 1c77e864 fix(swagger): Fix the broken swagger. (#2317)
 * aa052346 feat: Support workflow level poddisruptionbudge for workflow pods #1728 (#2286)
 * 8da88d7e chore: update getting-started guide for 2.5.2 and apply other tweaks (#2311)
 * 2f97c261 build: Improve reliability of release. (#2309)
 * 5dcb84bb chore(cli): Clean-up code. Closes #2117 (#2303)
 * e49dd8c4 chore(cli): Migrate `argo logs` to use API client. See #2116 (#2177)
 * 5c3d9cf9 chore(cli): Migrate `argo wait` to use API client. See #2116 (#2282)
 * baf03f67 fix(ui): Provide a link to archived logs. Fixes #2300 (#2301)

### Contributors

 * Aaron Curtis
 * Alex Collins
 * Antoine Dao
 * Antonio Macías Ojeda
 * Daisuke Taniwaki
 * Daniel Moran
 * Derek Wang
 * EDGsheryl
 * Ejiah
 * Huan-Cheng Chang
 * Michael Crenshaw
 * Mingjie Tang
 * Mukulikak
 * Niklas Hansson
 * Pascal VanDerSwalmen
 * Pratik Raj
 * Rafael Rodrigues
 * Roman Galeev
 * Saradhi Sreegiriraju
 * Saravanan Balasubramanian
 * Simon Behar
 * StoneHuang
 * Theodore Messinezis
 * Tristan Colgate-McFarlane
 * Xin Wang
 * Zach Aller
 * fsiegmund
 * mark9white
 * maryoush
 * tkilpela
 * vatine
 * モハメド

## v2.6.4 (2020-04-15)

 * e6caf984 Update manifests to v2.6.4
 * 5aeb3ecf fix: Don't use docker cache when building release images (#2707)

### Contributors

 * Alex Collins
 * Simon Behar

## v2.6.3 (2020-03-16)

 * 2e8ac609 Update manifests to v2.6.3
 * 9633bad1 fix: Delete PVCs unless WF Failed/Errored (#2449)
 * a0b933a0 fix: Don't error when optional artifacts are not found (#2445)
 * d1513e68 fix: Allow numbers in steps.args.params.value (#2414)
 * 9c608e50 fix: Check child node status before backoff in retry (#2407)
 * 8ad643c4 fix: Say no logs were outputted when pod is done (#2373)
 * 60fcfe90 fix(ui): Removed tailLines from EventSource (#2330)
 * 6ec81d35 fix "Unable to retry workflow" on argo-server (#2409)
 * 642ccca2 fix: Build with the correct version if you check out a specific version (#2423)

### Contributors

 * Alex Collins
 * EDGsheryl
 * Simon Behar
 * tkilpela

## v2.6.2 (2020-03-12)

 * be0a0bb4 Update manifests to v2.6.2
 * 09ec9a0d fix(docker): fix streaming of combined stdout/stderr (#2368)
 * 64b6f3a4 fix: Correctly report version. Fixes #2374 (#2402)

### Contributors

 * Alex Collins

## v2.6.1 (2020-03-04)

 * 842739d7 Update manifests to v2.6.1
 * 64c6aa43 fix: Restart server ConfigMap watch when closed (#2360)
 * 9ff429aa fix: Always validate templates (#2342)
 * 51c3ad33 fix: Simplify completion detection logic in DAGs (#2344)
 * 3de7e513 fix(auth): Fixed returning  expired  Auth token for GKE (#2327)
 * fa2a3023 fix: Add timezone support to startingDeadlineSeconds (#2335)
 * a9b6a254 fix(ui): fixed worflow UI refresh. Fixes ##2337 (#2338)
 * 793c072e docker: remove NopCloser from the executor. (#2345)
 * 5d3bdd56 Update manifests to v2.6.0

### Contributors

 * Alex Collins
 * Derek Wang
 * Saravanan Balasubramanian
 * Simon Behar
 * Tristan Colgate-McFarlane
 * fsiegmund

## v2.6.0-rc3 (2020-02-25)

 * fc24de46 Update manifests to v2.6.0-rc3
 * ef297d4c Merge branch 'rel-enhance' into release-2.6
 * bc9de69f Merge branch 'master' into release-2.6
 * 39ab8046 build: Improve reliability of release.
 * b5947165 feat: Create API clients (#2218)
 * 214c4515 fix(controller): Get correct Step or DAG name. Fixes #2244 (#2304)
 * 19460276 Merge branch 'master' into release-2.6
 * c4d26466 fix: Remove active wf from Cron when deleted (#2299)
 * 0eff938d fix: Skip empty withParam steps (#2284)
 * 636ea443 chore(cli): Migrate `argo terminate` to use API client. See #2116 (#2280)
 * d0a9b528 chore(cli): Migrate `argo template` to use API client. Closes #2115 (#2296)
 * f69a6c5f chore(cli): Migrate `argo cron` to use API client. Closes #2114 (#2295)
 * 80b9b590 chore(cli): Migrate `argo retry` to use API client. See #2116 (#2277)

### Contributors

 * Alex Collins
 * Derek Wang
 * Simon Behar

## v2.6.0-rc2 (2020-02-21)

 * 9f7ef614 Update manifests to v2.6.0-rc2
 * 21163db3 Merge branch 'master' into release-2.6
 * cdbc6194 fix(sequence): broken in 2.5. Fixes #2248 (#2263)
 * 0d3955a7 refactor(cli): 2x simplify migration to API client. See #2116 (#2290)
 * df8493a1 fix: Start Argo server with out Configmap #2285 (#2293)
 * 51cdf95b doc: More detail for namespaced installation (#2292)
 * a7302697 build(swagger): Fix argo-server swagger so version does not change. (#2291)
 * 47b4fc28 fix(cli): Reinstate `argo wait`. Fixes #2281 (#2283)
 * 1793887b chore(cli): Migrate `argo suspend` and `argo resume` to use API client. See #2116 (#2275)
 * 1f3d2f5a chore(cli): Update `argo resubmit` to support client API. See #2116 (#2276)
 * c33f6cda fix(archive): Fix bug in migrating cluster name. Fixes #2272 (#2279)
 * fb0acbbf fix: Fixes double logging in UI. Fixes #2270 (#2271)
 * acf37c2d fix: Correctly report version. Fixes #2264 (#2268)
 * b30f1af6 fix: Removes Template.Arguments as this is never used. Fixes #2046 (#2267)

### Contributors

 * Alex Collins
 * Derek Wang
 * Saravanan Balasubramanian
 * mark9white

## v2.6.0-rc1 (2020-02-19)


### Contributors


## v2.6.0 (2020-02-28)

 * 5d3bdd56 Update manifests to v2.6.0
 * fc24de46 Update manifests to v2.6.0-rc3
 * ef297d4c Merge branch 'rel-enhance' into release-2.6
 * bc9de69f Merge branch 'master' into release-2.6
 * 39ab8046 build: Improve reliability of release.
 * b5947165 feat: Create API clients (#2218)
 * 214c4515 fix(controller): Get correct Step or DAG name. Fixes #2244 (#2304)
 * 19460276 Merge branch 'master' into release-2.6
 * c4d26466 fix: Remove active wf from Cron when deleted (#2299)
 * 0eff938d fix: Skip empty withParam steps (#2284)
 * 636ea443 chore(cli): Migrate `argo terminate` to use API client. See #2116 (#2280)
 * d0a9b528 chore(cli): Migrate `argo template` to use API client. Closes #2115 (#2296)
 * f69a6c5f chore(cli): Migrate `argo cron` to use API client. Closes #2114 (#2295)
 * 80b9b590 chore(cli): Migrate `argo retry` to use API client. See #2116 (#2277)
 * 9f7ef614 Update manifests to v2.6.0-rc2
 * 21163db3 Merge branch 'master' into release-2.6
 * cdbc6194 fix(sequence): broken in 2.5. Fixes #2248 (#2263)
 * 0d3955a7 refactor(cli): 2x simplify migration to API client. See #2116 (#2290)
 * df8493a1 fix: Start Argo server with out Configmap #2285 (#2293)
 * 51cdf95b doc: More detail for namespaced installation (#2292)
 * a7302697 build(swagger): Fix argo-server swagger so version does not change. (#2291)
 * 47b4fc28 fix(cli): Reinstate `argo wait`. Fixes #2281 (#2283)
 * 1793887b chore(cli): Migrate `argo suspend` and `argo resume` to use API client. See #2116 (#2275)
 * 1f3d2f5a chore(cli): Update `argo resubmit` to support client API. See #2116 (#2276)
 * c33f6cda fix(archive): Fix bug in migrating cluster name. Fixes #2272 (#2279)
 * fb0acbbf fix: Fixes double logging in UI. Fixes #2270 (#2271)
 * acf37c2d fix: Correctly report version. Fixes #2264 (#2268)
 * b30f1af6 fix: Removes Template.Arguments as this is never used. Fixes #2046 (#2267)
 * bd89f9cb Update manifests to v2.6.0-rc1
 * 79b09ed4 fix: Removed duplicate Watch Command (#2262)
 * b5c47266 feat(ui): Add filters for archived workflows (#2257)
 * d30aa335 fix(archive): Return correct next page info. Fixes #2255 (#2256)
 * 8c97689e fix: Ignore bookmark events for restart. Fixes #2249 (#2253)
 * 63858eaa fix(offloading): Change offloaded nodes datatype to JSON to support 1GB. Fixes #2246 (#2250)
 * 4d88374b Add Cartrack into officially using Argo (#2251)
 * d309d5c1 feat(archive): Add support to filter list by labels. Closes #2171 (#2205)
 * 79f13373 feat: Add a new symbol for suspended nodes. Closes #1896 (#2240)
 * 82b48821 Fix presumed typo (#2243)
 * af94352f feat: Reduce API calls when changing filters. Closes #2231 (#2232)
 * a58cbc7d BasisAI uses Argo (#2241)
 * 68e3c9fd feat: Add Pod Name to UI (#2227)
 * eef85072 fix(offload): Fix bug which deleted completed workflows. Fixes #2233 (#2234)
 * 4e4565cd feat: Label workflow-created pvc with workflow name (#1890)
 * 8bd5ecbc fix: display error message when deleting archived workflow fails. (#2235)
 * ae381ae5 feat: This add support to enable debug logging for all CLI commands (#2212)
 * 1b1927fc feat(swagger): Adds a make api/argo-server/swagger.json (#2216)
 * 5d7b4c8c Update README.md (#2226)
 * 170abfa5 chore: Run `go mod tidy` (#2225)
 * 2981e6ff fix: Enforce UnknownField requirement in WorkflowStep (#2210)
 * affc235c feat: Add failed node info to exit handler (#2166)
 * af1f6d60 fix: UI Responsive design on filter box (#2221)
 * a445049c fix: Fixed race condition in kill container method. Fixes #1884 (#2208)
 * 2672857f feat: upgrade to Go 1.13. Closes #1375 (#2097)
 * 7466efa9 feat: ArtifactRepositoryRef ConfigMap is now taken from the workflow namespace (#1821)
 * 50f331d0 build: Fix ARGO_TOKEN (#2215)
 * 7f090351 test: Correctly report diagnostics (#2214)
 * f2bd74bc fix: Remove quotes from UI (#2213)
 * 62f46680 fix(offloading): Correctly deleted offloaded data. Fixes #2206 (#2207)
 * e30b77fc feat(ui): Add label filter to workflow list page. Fixes #802 (#2196)
 * 930ced39 fix(ui): fixed workflow filtering and ordering. Fixes #2201 (#2202)
 * 88112312 fix: Correct login instructions. (#2198)
 * d6f5953d Update ReadMe for EBSCO (#2195)
 * b024c46c feat: Add ability to submit CronWorkflow from CLI (#2003)
 * c97527ce test: Invoke tests using s.T() (#2194)
 * 72a54fe1 chore: Move info.proto et al to correct package (#2193)
 * f6600fa4 fix: Namespace and phase selection in UI (#2191)
 * c4a24dca fix(k8sapi-executor): Fix KillContainer impl (#2160)
 * d22a5fe6 Update cli_with_server_test.go (#2189)
 * ff18180f test: Remove podGC (#2187)
 * 78245305 chore: Improved error handling and refactor (#2184)
 * b9c828ad fix(archive): Only delete offloaded data we do not need. Fixes #2170 and #2156 (#2172)
 * 73cb5418 feat: Allow CronWorkflows to have instanceId (#2081)
 * 9efea660 Sort list and add Greenhouse (#2182)
 * cae399ba fix: Fixed the Exec Provider token bug (#2181)
 * fc476b2a fix(ui): Retry workflow event stream on connection loss. Fixes #2179 (#2180)
 * 65058a27 fix: Correctly create code from changed protos. (#2178)
 * 585d1eef chore: Update lint command to use apiclient. See #2116 (#2131)
 * 299d467c build: Update release process and docs (#2128)
 * fcfe1d43 feat: Implemented open default browser in local mode (#2122)
 * f6cee552 fix: Specify download .tgz extension (#2164)
 * 8a1e611a feat: Update archived workdflow column to be JSON. Closes #2133 (#2152)
 * f591c471 fix!: Change `argo token` to `argo auth token`. Closes #2149 (#2150)
 * 519c9434 chore: Add Mock gen to make codegen (#2148)
 * 409a5154 fix: Add certs to argocli image. Fixes #2129 (#2143)
 * b094802a fix: Allow download of artifacs in server auth-mode. Fixes #2129 (#2147)
 * 520fa540 fix: Correct SQL syntax. (#2141)
 * 059cb9b1 fix: logs UI should fall back to archive (#2139)
 * 4cda9a05 fix: route all unknown web content requests to index.html (#2134)
 * 14d8b5d3 fix: archiveLogs needs to copy stderr (#2136)
 * 91319ee4 fixed ui navigation issues with basehref (#2130)
 * 7881b980 docs: Add CronWorkflow usage docs (#2124)
 * badfd183 feat: Add support to delete by using labels. Depended on by #2116 (#2123)
 * 706d0f23 test: Try and make e2e more robust. Fixes #2125 (#2127)
 * a75ac1b4 fix: mark CLI common.go vars and funcs as DEPRECATED (#2119)
 * be21a0f1 feat(server): Restart server when config changes. Fixes #2090 (#2092)
 * b5cd72b0 test: Parallelize Cron tests (#2118)
 * b2bd25bc fix: Disable webpack dot rule (#2112)
 * 865b4f3a addcompany (#2109)
 * 213e3a9d fix: Fix Resource Deletion Bug (#2084)
 * ab1de233 refactor(cli): Introduce v1.Interface for CLI. Closes #2107 (#2048)
 * 7a19f85c feat: Implemented Basic Auth scheme (#2093)
 * 7611b9f6 fix(ui): Add support for bash href. Fixes ##2100 (#2105)
 * 516d05f8  fix: Namespace redirects no longer error and are snappier (#2106)
 * 16aed5c8 fix: Skip running --token testing if it is not on CI (#2104)
 * aece7e6e Parse container ID in correct way on CRI-O. Fixes #2095 (#2096)
 * b6a2be89 feat: support arg --token when talking to argo-server (#2027) (#2089)
 * 01d8cae1 build: adds `make env` to make testing easier (#2087)
 * 492842aa docs(README): Add Capital One to user list (#2094)
 * d56a0e12 fix(controller): Fix template resolution for step groups. Fixes #1868  (#1920)
 * b97044d2 fix(security): Fixes an issue that allowed you to list archived workf… (#2079)

### Contributors

 * Aaron Curtis
 * Alex Collins
 * Alexey Volkov
 * Daisuke Taniwaki
 * Derek Wang
 * Dineshmohan Rajaveeran
 * Huan-Cheng Chang
 * Jialu Zhu
 * Juan C. Muller
 * Nasrudin Bin Salim
 * Nick Groszewski
 * Rafał Bigaj
 * Roman Galeev
 * Saravanan Balasubramanian
 * Simon Behar
 * Tom Wieczorek
 * Tristan Colgate-McFarlane
 * fsiegmund
 * mark9white
 * mdvorakramboll
 * tkilpela

## v2.5.3-rc4 (2020-01-27)


### Contributors


## v2.5.2 (2020-02-24)

 * 4b25e2ac Update manifests to v2.5.2
 * 6092885c fix(archive): Fix bug in migrating cluster name. Fixes #2272 (#2279)

### Contributors

 * Alex Collins

## v2.5.1 (2020-02-20)

 * fb496a24 Update manifests to v2.5.1
 * 61114d62 fix: Fixes double logging in UI. Fixes #2270 (#2271)
 * 4737c8a2 fix: Correctly report version. Fixes #2264 (#2268)
 * e096feaf fix: Removed duplicate Watch Command (#2262)
 * 11d2232e Update manifests to v2.5.0
 * 661f8a11 fix: Ignore bookmark events for restart. Fixes #2249 (#2253)
 * 6c1a6601 fix(offloading): Change offloaded nodes datatype to JSON to support 1GB. Fixes #2246 (#2250)
 * befd3594 Update manifests to v2.5.0-rc12
 * 4670c99e fix(offload): Fix bug which deleted completed workflows. Fixes #2233 (#2234)
 * 47d9a41a Update manifests to v2.5.0-rc11
 * 04917cde fix: Remove quotes from UI (#2213)
 * 2705a114 fix(offloading): Correctly deleted offloaded data. Fixes #2206 (#2207)
 * cee3a771 Merge commit '930ced39241b427a521b609c403e7a39f6cc8c48' into release-2.5
 * 930ced39 fix(ui): fixed workflow filtering and ordering. Fixes #2201 (#2202)
 * 88112312 fix: Correct login instructions. (#2198)
 * b557eeb9 Update manifests to v2.5.0-rc10
 * f8b8efc7 Merge branch 'master' into release-2.5
 * d6f5953d Update ReadMe for EBSCO (#2195)
 * b024c46c feat: Add ability to submit CronWorkflow from CLI (#2003)
 * c97527ce test: Invoke tests using s.T() (#2194)
 * 073216c8 Merge branch 'master' into release-2.5
 * 72a54fe1 chore: Move info.proto et al to correct package (#2193)
 * cbfbb98b Merge branch 'master' into release-2.5
 * f6600fa4 fix: Namespace and phase selection in UI (#2191)
 * c4a24dca fix(k8sapi-executor): Fix KillContainer impl (#2160)
 * d22a5fe6 Update cli_with_server_test.go (#2189)
 * ff18180f test: Remove podGC (#2187)

### Contributors

 * Alex Collins
 * Dineshmohan Rajaveeran
 * Saravanan Balasubramanian
 * Simon Behar
 * Tom Wieczorek
 * fsiegmund
 * tkilpela

## v2.5.0-rc9 (2020-02-06)

 * bea41b49 Update manifests to v2.5.0-rc9
 * 58f38ba7 Merge branch 'master' into release-2.5
 * 78245305 chore: Improved error handling and refactor (#2184)
 * b9c828ad fix(archive): Only delete offloaded data we do not need. Fixes #2170 and #2156 (#2172)
 * 73cb5418 feat: Allow CronWorkflows to have instanceId (#2081)
 * 3fed36f0 Merge branch 'master' into release-2.5
 * 9efea660 Sort list and add Greenhouse (#2182)
 * cae399ba fix: Fixed the Exec Provider token bug (#2181)
 * fc476b2a fix(ui): Retry workflow event stream on connection loss. Fixes #2179 (#2180)
 * 4d53cc47 Merge branch 'master' into release-2.5
 * 65058a27 fix: Correctly create code from changed protos. (#2178)
 * 585d1eef chore: Update lint command to use apiclient. See #2116 (#2131)
 * 299d467c build: Update release process and docs (#2128)
 * fcfe1d43 feat: Implemented open default browser in local mode (#2122)
 * f6cee552 fix: Specify download .tgz extension (#2164)
 * 8a1e611a feat: Update archived workdflow column to be JSON. Closes #2133 (#2152)
 * f591c471 fix!: Change `argo token` to `argo auth token`. Closes #2149 (#2150)
 * 519c9434 chore: Add Mock gen to make codegen (#2148)

### Contributors

 * Alex Collins
 * Juan C. Muller
 * Saravanan Balasubramanian
 * Simon Behar
 * fsiegmund

## v2.5.0-rc8 (2020-02-03)

 * 392de814 Update manifests to v2.5.0-rc8
 * 3d84a935 Merge branch 'master' into release-2.5
 * 409a5154 fix: Add certs to argocli image. Fixes #2129 (#2143)
 * 9b327658 Merge branch 'master' into release-2.5
 * b094802a fix: Allow download of artifacs in server auth-mode. Fixes #2129 (#2147)
 * a61f1457 Merge branch 'master' into release-2.5
 * 520fa540 fix: Correct SQL syntax. (#2141)
 * 059cb9b1 fix: logs UI should fall back to archive (#2139)
 * 1d0a5234 Merge branch 'master' into release-2.5
 * 4cda9a05 fix: route all unknown web content requests to index.html (#2134)
 * 14d8b5d3 fix: archiveLogs needs to copy stderr (#2136)
 * 91319ee4 fixed ui navigation issues with basehref (#2130)
 * 7881b980 docs: Add CronWorkflow usage docs (#2124)
 * adbe67c0 Merge branch 'master' into release-2.5
 * badfd183 feat: Add support to delete by using labels. Depended on by #2116 (#2123)
 * 706d0f23 test: Try and make e2e more robust. Fixes #2125 (#2127)
 * 7bee02d8 build: correct version

### Contributors

 * Alex Collins
 * Simon Behar
 * Tristan Colgate-McFarlane
 * fsiegmund

## v2.5.0-rc7 (2020-01-31)

 * 40e7ca37 Update manifests to v2.5.0-rc7
 * 98702cae Merge branch 'master' into release-2.5
 * a75ac1b4 fix: mark CLI common.go vars and funcs as DEPRECATED (#2119)
 * be21a0f1 feat(server): Restart server when config changes. Fixes #2090 (#2092)
 * b5cd72b0 test: Parallelize Cron tests (#2118)
 * 26f9bcad Merge branch 'master' into release-2.5
 * b2bd25bc fix: Disable webpack dot rule (#2112)
 * 865b4f3a addcompany (#2109)
 * 213e3a9d fix: Fix Resource Deletion Bug (#2084)
 * ab1de233 refactor(cli): Introduce v1.Interface for CLI. Closes #2107 (#2048)
 * 3c8eced6 Merge branch 'master' into release-2.5
 * 7a19f85c feat: Implemented Basic Auth scheme (#2093)

### Contributors

 * Alex Collins
 * Jialu Zhu
 * Saravanan Balasubramanian
 * Simon Behar

## v2.5.0-rc6 (2020-01-30)

 * 7b7fcf01 Update manifests to v2.5.0-rc6
 * c1cb415e Merge branch 'master' into release-2.5
 * 7611b9f6 fix(ui): Add support for bash href. Fixes ##2100 (#2105)
 * 516d05f8  fix: Namespace redirects no longer error and are snappier (#2106)
 * 16aed5c8 fix: Skip running --token testing if it is not on CI (#2104)
 * aece7e6e Parse container ID in correct way on CRI-O. Fixes #2095 (#2096)

### Contributors

 * Alex Collins
 * Derek Wang
 * Rafał Bigaj
 * Simon Behar

## v2.5.0-rc5 (2020-01-29)

 * 4609f3d7 Update manifests to v2.5.0-rc5
 * af14e452 Merge branch 'master' into release-2.5
 * b6a2be89 feat: support arg --token when talking to argo-server (#2027) (#2089)
 * 01d8cae1 build: adds `make env` to make testing easier (#2087)
 * 492842aa docs(README): Add Capital One to user list (#2094)
 * ed223e9d build: use the latest tag for codegen
 * 2db76d24 Merge branch 'master' into release-2.5
 * d56a0e12 fix(controller): Fix template resolution for step groups. Fixes #1868  (#1920)
 * b97044d2 fix(security): Fixes an issue that allowed you to list archived workf… (#2079)

### Contributors

 * Alex Collins
 * Daisuke Taniwaki
 * Derek Wang
 * Nick Groszewski

## v2.5.0-rc4 (2020-01-27)

 * 2afcb0f2 Update manifests to v2.5.0-rc4
 * e2f4d640 Merge branch 'master' into release-2.5
 * c4f49cf0 refactor: Move server code (cmd/server/ -> server/) (#2071)
 * 2542454c fix(controller): Do not crash if cm is empty. Fixes #2069 (#2070)

### Contributors

 * Alex Collins
 * Simon Behar

## v2.5.0-rc3 (2020-01-27)

 * 091c2f7e lint
 * 67a51858 build: simplify release
 * 3572d4df build: force tag
 * f1c03bb3 build: only tag if committing changes
 * 30775fac Update manifests to v2.5.0-rc3
 * 9afa6783 Merge branch 'master' into release-2.5
 * 85fa9aaf fix: Do not expect workflowChange to always be defined (#2068)
 * 6f65bc2b fix: "base64 -d" not always available, using "base64 --decode" (#2067)
 * 5e755c61 build: correct image tag
 * e7cf886b ci: install kustomize
 * 5328389a adds "verify-manifests" target
 * ef1c403e fix: generate no-db manifests
 * 6f2c8802 feat(ui): Use cookies in the UI. Closes #1949 (#2058)
 * 4592aec6 fix(api): Change `CronWorkflowName` to `Name`. Fixes #1982 (#2033)
 * a37562a0 ci: DEV_IMAGE=true
 * 4676a946 try and improve the release tasks
 * fe64ae11 Merge branch 'master' into release-2.5
 * e26c11af fix: only run archived wf testing when persistence is enabled (#2059)
 * b3cab5df fix: Fix permission test cases (#2035)

### Contributors

 * Alex Collins
 * Derek Wang
 * Simon Behar

## v2.5.0-rc2 (2020-01-24)


### Contributors


## v2.5.0-rc12 (2020-02-13)

 * befd3594 Update manifests to v2.5.0-rc12
 * 4670c99e fix(offload): Fix bug which deleted completed workflows. Fixes #2233 (#2234)

### Contributors

 * Alex Collins

## v2.5.0-rc11 (2020-02-11)

 * 47d9a41a Update manifests to v2.5.0-rc11
 * 04917cde fix: Remove quotes from UI (#2213)
 * 2705a114 fix(offloading): Correctly deleted offloaded data. Fixes #2206 (#2207)
 * cee3a771 Merge commit '930ced39241b427a521b609c403e7a39f6cc8c48' into release-2.5
 * 930ced39 fix(ui): fixed workflow filtering and ordering. Fixes #2201 (#2202)
 * 88112312 fix: Correct login instructions. (#2198)

### Contributors

 * Alex Collins
 * fsiegmund

## v2.5.0-rc10 (2020-02-07)

 * b557eeb9 Update manifests to v2.5.0-rc10
 * f8b8efc7 Merge branch 'master' into release-2.5
 * d6f5953d Update ReadMe for EBSCO (#2195)
 * b024c46c feat: Add ability to submit CronWorkflow from CLI (#2003)
 * c97527ce test: Invoke tests using s.T() (#2194)
 * 073216c8 Merge branch 'master' into release-2.5
 * 72a54fe1 chore: Move info.proto et al to correct package (#2193)
 * cbfbb98b Merge branch 'master' into release-2.5
 * f6600fa4 fix: Namespace and phase selection in UI (#2191)
 * c4a24dca fix(k8sapi-executor): Fix KillContainer impl (#2160)
 * d22a5fe6 Update cli_with_server_test.go (#2189)
 * ff18180f test: Remove podGC (#2187)
 * bea41b49 Update manifests to v2.5.0-rc9
 * 58f38ba7 Merge branch 'master' into release-2.5
 * 78245305 chore: Improved error handling and refactor (#2184)
 * b9c828ad fix(archive): Only delete offloaded data we do not need. Fixes #2170 and #2156 (#2172)
 * 73cb5418 feat: Allow CronWorkflows to have instanceId (#2081)
 * 3fed36f0 Merge branch 'master' into release-2.5
 * 9efea660 Sort list and add Greenhouse (#2182)
 * cae399ba fix: Fixed the Exec Provider token bug (#2181)
 * fc476b2a fix(ui): Retry workflow event stream on connection loss. Fixes #2179 (#2180)
 * 4d53cc47 Merge branch 'master' into release-2.5
 * 65058a27 fix: Correctly create code from changed protos. (#2178)
 * 585d1eef chore: Update lint command to use apiclient. See #2116 (#2131)
 * 299d467c build: Update release process and docs (#2128)
 * fcfe1d43 feat: Implemented open default browser in local mode (#2122)
 * f6cee552 fix: Specify download .tgz extension (#2164)
 * 8a1e611a feat: Update archived workdflow column to be JSON. Closes #2133 (#2152)
 * f591c471 fix!: Change `argo token` to `argo auth token`. Closes #2149 (#2150)
 * 392de814 Update manifests to v2.5.0-rc8
 * 519c9434 chore: Add Mock gen to make codegen (#2148)
 * 3d84a935 Merge branch 'master' into release-2.5
 * 409a5154 fix: Add certs to argocli image. Fixes #2129 (#2143)
 * 9b327658 Merge branch 'master' into release-2.5
 * b094802a fix: Allow download of artifacs in server auth-mode. Fixes #2129 (#2147)
 * a61f1457 Merge branch 'master' into release-2.5
 * 520fa540 fix: Correct SQL syntax. (#2141)
 * 059cb9b1 fix: logs UI should fall back to archive (#2139)
 * 1d0a5234 Merge branch 'master' into release-2.5
 * 4cda9a05 fix: route all unknown web content requests to index.html (#2134)
 * 14d8b5d3 fix: archiveLogs needs to copy stderr (#2136)
 * 91319ee4 fixed ui navigation issues with basehref (#2130)
 * 7881b980 docs: Add CronWorkflow usage docs (#2124)
 * adbe67c0 Merge branch 'master' into release-2.5
 * badfd183 feat: Add support to delete by using labels. Depended on by #2116 (#2123)
 * 706d0f23 test: Try and make e2e more robust. Fixes #2125 (#2127)
 * 7bee02d8 build: correct version
 * 40e7ca37 Update manifests to v2.5.0-rc7
 * 98702cae Merge branch 'master' into release-2.5
 * a75ac1b4 fix: mark CLI common.go vars and funcs as DEPRECATED (#2119)
 * be21a0f1 feat(server): Restart server when config changes. Fixes #2090 (#2092)
 * b5cd72b0 test: Parallelize Cron tests (#2118)
 * 26f9bcad Merge branch 'master' into release-2.5
 * b2bd25bc fix: Disable webpack dot rule (#2112)
 * 865b4f3a addcompany (#2109)
 * 213e3a9d fix: Fix Resource Deletion Bug (#2084)
 * ab1de233 refactor(cli): Introduce v1.Interface for CLI. Closes #2107 (#2048)
 * 3c8eced6 Merge branch 'master' into release-2.5
 * 7a19f85c feat: Implemented Basic Auth scheme (#2093)
 * 7b7fcf01 Update manifests to v2.5.0-rc6
 * c1cb415e Merge branch 'master' into release-2.5
 * 7611b9f6 fix(ui): Add support for bash href. Fixes ##2100 (#2105)
 * 516d05f8  fix: Namespace redirects no longer error and are snappier (#2106)
 * 16aed5c8 fix: Skip running --token testing if it is not on CI (#2104)
 * aece7e6e Parse container ID in correct way on CRI-O. Fixes #2095 (#2096)
 * 4609f3d7 Update manifests to v2.5.0-rc5
 * af14e452 Merge branch 'master' into release-2.5
 * b6a2be89 feat: support arg --token when talking to argo-server (#2027) (#2089)
 * 01d8cae1 build: adds `make env` to make testing easier (#2087)
 * 492842aa docs(README): Add Capital One to user list (#2094)
 * ed223e9d build: use the latest tag for codegen
 * 2db76d24 Merge branch 'master' into release-2.5
 * d56a0e12 fix(controller): Fix template resolution for step groups. Fixes #1868  (#1920)
 * b97044d2 fix(security): Fixes an issue that allowed you to list archived workf… (#2079)
 * 2afcb0f2 Update manifests to v2.5.0-rc4
 * e2f4d640 Merge branch 'master' into release-2.5
 * 091c2f7e lint
 * 67a51858 build: simplify release
 * c4f49cf0 refactor: Move server code (cmd/server/ -> server/) (#2071)
 * 2542454c fix(controller): Do not crash if cm is empty. Fixes #2069 (#2070)
 * 3572d4df build: force tag
 * f1c03bb3 build: only tag if committing changes
 * 30775fac Update manifests to v2.5.0-rc3
 * 9afa6783 Merge branch 'master' into release-2.5
 * 85fa9aaf fix: Do not expect workflowChange to always be defined (#2068)
 * 6f65bc2b fix: "base64 -d" not always available, using "base64 --decode" (#2067)
 * 5e755c61 build: correct image tag
 * e7cf886b ci: install kustomize
 * 5328389a adds "verify-manifests" target
 * ef1c403e fix: generate no-db manifests
 * 6f2c8802 feat(ui): Use cookies in the UI. Closes #1949 (#2058)
 * 4592aec6 fix(api): Change `CronWorkflowName` to `Name`. Fixes #1982 (#2033)
 * a37562a0 ci: DEV_IMAGE=true
 * 4676a946 try and improve the release tasks
 * fe64ae11 Merge branch 'master' into release-2.5
 * e26c11af fix: only run archived wf testing when persistence is enabled (#2059)
 * 243eeceb make manifests
 * c74d0f40 build: use GIT_TAG as VERSION
 * b1fd43f7 Merge branch 'release-2.5' of https://github.com/argoproj/argo into release-2.5
 * 8663652a make manifesets
 * c8abcb92 Merge branch 'master' into release-2.5
 * b3cab5df fix: Fix permission test cases (#2035)
 * 6cf64a21 Update Makefile
 * 216d14ad fixed makefile
 * ba2f7891 merge conflict
 * 8752f026 merge conflict
 * 50777ed8 fix: nil pointer in GC (#2055)
 * b408e7cd fix: nil pointer in GC (#2055)
 * 7ed058c3 fix: offload Node Status in Get and List api call (#2051)
 * 4ac11560 fix: offload Node Status in Get and List api call (#2051)
 * dfdde1d0 ci: Run using our own cowsay image (#2047)
 * aa6a536d fix(persistence): Allow `argo server` to run without persistence (#2050)
 * 71ba8238 Update README.md (#2045)
 * c7953052 fix(persistence): Allow `argo server` to run without persistence (#2050)

### Contributors

 * Alex Collins
 * Daisuke Taniwaki
 * Derek Wang
 * Dineshmohan Rajaveeran
 * Ed Lee
 * Jialu Zhu
 * Juan C. Muller
 * Nick Groszewski
 * Rafał Bigaj
 * Saravanan Balasubramanian
 * Simon Behar
 * Tom Wieczorek
 * Tristan Colgate-McFarlane
 * fsiegmund

## v2.5.0-rc1 (2020-01-23)


### Contributors


## v2.5.0 (2020-02-18)

 * 11d2232e Update manifests to v2.5.0
 * 661f8a11 fix: Ignore bookmark events for restart. Fixes #2249 (#2253)
 * 6c1a6601 fix(offloading): Change offloaded nodes datatype to JSON to support 1GB. Fixes #2246 (#2250)
 * befd3594 Update manifests to v2.5.0-rc12
 * 4670c99e fix(offload): Fix bug which deleted completed workflows. Fixes #2233 (#2234)
 * 47d9a41a Update manifests to v2.5.0-rc11
 * 04917cde fix: Remove quotes from UI (#2213)
 * 2705a114 fix(offloading): Correctly deleted offloaded data. Fixes #2206 (#2207)
 * cee3a771 Merge commit '930ced39241b427a521b609c403e7a39f6cc8c48' into release-2.5
 * 930ced39 fix(ui): fixed workflow filtering and ordering. Fixes #2201 (#2202)
 * 88112312 fix: Correct login instructions. (#2198)
 * b557eeb9 Update manifests to v2.5.0-rc10
 * f8b8efc7 Merge branch 'master' into release-2.5
 * d6f5953d Update ReadMe for EBSCO (#2195)
 * b024c46c feat: Add ability to submit CronWorkflow from CLI (#2003)
 * c97527ce test: Invoke tests using s.T() (#2194)
 * 073216c8 Merge branch 'master' into release-2.5
 * 72a54fe1 chore: Move info.proto et al to correct package (#2193)
 * cbfbb98b Merge branch 'master' into release-2.5
 * f6600fa4 fix: Namespace and phase selection in UI (#2191)
 * c4a24dca fix(k8sapi-executor): Fix KillContainer impl (#2160)
 * d22a5fe6 Update cli_with_server_test.go (#2189)
 * ff18180f test: Remove podGC (#2187)
 * bea41b49 Update manifests to v2.5.0-rc9
 * 58f38ba7 Merge branch 'master' into release-2.5
 * 78245305 chore: Improved error handling and refactor (#2184)
 * b9c828ad fix(archive): Only delete offloaded data we do not need. Fixes #2170 and #2156 (#2172)
 * 73cb5418 feat: Allow CronWorkflows to have instanceId (#2081)
 * 3fed36f0 Merge branch 'master' into release-2.5
 * 9efea660 Sort list and add Greenhouse (#2182)
 * cae399ba fix: Fixed the Exec Provider token bug (#2181)
 * fc476b2a fix(ui): Retry workflow event stream on connection loss. Fixes #2179 (#2180)
 * 4d53cc47 Merge branch 'master' into release-2.5
 * 65058a27 fix: Correctly create code from changed protos. (#2178)
 * 585d1eef chore: Update lint command to use apiclient. See #2116 (#2131)
 * 299d467c build: Update release process and docs (#2128)
 * fcfe1d43 feat: Implemented open default browser in local mode (#2122)
 * f6cee552 fix: Specify download .tgz extension (#2164)
 * 8a1e611a feat: Update archived workdflow column to be JSON. Closes #2133 (#2152)
 * f591c471 fix!: Change `argo token` to `argo auth token`. Closes #2149 (#2150)
 * 392de814 Update manifests to v2.5.0-rc8
 * 519c9434 chore: Add Mock gen to make codegen (#2148)
 * 3d84a935 Merge branch 'master' into release-2.5
 * 409a5154 fix: Add certs to argocli image. Fixes #2129 (#2143)
 * 9b327658 Merge branch 'master' into release-2.5
 * b094802a fix: Allow download of artifacs in server auth-mode. Fixes #2129 (#2147)
 * a61f1457 Merge branch 'master' into release-2.5
 * 520fa540 fix: Correct SQL syntax. (#2141)
 * 059cb9b1 fix: logs UI should fall back to archive (#2139)
 * 1d0a5234 Merge branch 'master' into release-2.5
 * 4cda9a05 fix: route all unknown web content requests to index.html (#2134)
 * 14d8b5d3 fix: archiveLogs needs to copy stderr (#2136)
 * 91319ee4 fixed ui navigation issues with basehref (#2130)
 * 7881b980 docs: Add CronWorkflow usage docs (#2124)
 * adbe67c0 Merge branch 'master' into release-2.5
 * badfd183 feat: Add support to delete by using labels. Depended on by #2116 (#2123)
 * 706d0f23 test: Try and make e2e more robust. Fixes #2125 (#2127)
 * 7bee02d8 build: correct version
 * 40e7ca37 Update manifests to v2.5.0-rc7
 * 98702cae Merge branch 'master' into release-2.5
 * a75ac1b4 fix: mark CLI common.go vars and funcs as DEPRECATED (#2119)
 * be21a0f1 feat(server): Restart server when config changes. Fixes #2090 (#2092)
 * b5cd72b0 test: Parallelize Cron tests (#2118)
 * 26f9bcad Merge branch 'master' into release-2.5
 * b2bd25bc fix: Disable webpack dot rule (#2112)
 * 865b4f3a addcompany (#2109)
 * 213e3a9d fix: Fix Resource Deletion Bug (#2084)
 * ab1de233 refactor(cli): Introduce v1.Interface for CLI. Closes #2107 (#2048)
 * 3c8eced6 Merge branch 'master' into release-2.5
 * 7a19f85c feat: Implemented Basic Auth scheme (#2093)
 * 7b7fcf01 Update manifests to v2.5.0-rc6
 * c1cb415e Merge branch 'master' into release-2.5
 * 7611b9f6 fix(ui): Add support for bash href. Fixes ##2100 (#2105)
 * 516d05f8  fix: Namespace redirects no longer error and are snappier (#2106)
 * 16aed5c8 fix: Skip running --token testing if it is not on CI (#2104)
 * aece7e6e Parse container ID in correct way on CRI-O. Fixes #2095 (#2096)
 * 4609f3d7 Update manifests to v2.5.0-rc5
 * af14e452 Merge branch 'master' into release-2.5
 * b6a2be89 feat: support arg --token when talking to argo-server (#2027) (#2089)
 * 01d8cae1 build: adds `make env` to make testing easier (#2087)
 * 492842aa docs(README): Add Capital One to user list (#2094)
 * ed223e9d build: use the latest tag for codegen
 * 2db76d24 Merge branch 'master' into release-2.5
 * d56a0e12 fix(controller): Fix template resolution for step groups. Fixes #1868  (#1920)
 * b97044d2 fix(security): Fixes an issue that allowed you to list archived workf… (#2079)
 * 2afcb0f2 Update manifests to v2.5.0-rc4
 * e2f4d640 Merge branch 'master' into release-2.5
 * 091c2f7e lint
 * 67a51858 build: simplify release
 * c4f49cf0 refactor: Move server code (cmd/server/ -> server/) (#2071)
 * 2542454c fix(controller): Do not crash if cm is empty. Fixes #2069 (#2070)
 * 3572d4df build: force tag
 * f1c03bb3 build: only tag if committing changes
 * 30775fac Update manifests to v2.5.0-rc3
 * 9afa6783 Merge branch 'master' into release-2.5
 * 85fa9aaf fix: Do not expect workflowChange to always be defined (#2068)
 * 6f65bc2b fix: "base64 -d" not always available, using "base64 --decode" (#2067)
 * 5e755c61 build: correct image tag
 * e7cf886b ci: install kustomize
 * 5328389a adds "verify-manifests" target
 * ef1c403e fix: generate no-db manifests
 * 6f2c8802 feat(ui): Use cookies in the UI. Closes #1949 (#2058)
 * 4592aec6 fix(api): Change `CronWorkflowName` to `Name`. Fixes #1982 (#2033)
 * a37562a0 ci: DEV_IMAGE=true
 * 4676a946 try and improve the release tasks
 * fe64ae11 Merge branch 'master' into release-2.5
 * e26c11af fix: only run archived wf testing when persistence is enabled (#2059)
 * 243eeceb make manifests
 * c74d0f40 build: use GIT_TAG as VERSION
 * b1fd43f7 Merge branch 'release-2.5' of https://github.com/argoproj/argo into release-2.5
 * 8663652a make manifesets
 * c8abcb92 Merge branch 'master' into release-2.5
 * b3cab5df fix: Fix permission test cases (#2035)
 * 6cf64a21 Update Makefile
 * 216d14ad fixed makefile
 * ba2f7891 merge conflict
 * 8752f026 merge conflict
 * 50777ed8 fix: nil pointer in GC (#2055)
 * b408e7cd fix: nil pointer in GC (#2055)
 * 7ed058c3 fix: offload Node Status in Get and List api call (#2051)
 * 4ac11560 fix: offload Node Status in Get and List api call (#2051)
 * dfdde1d0 ci: Run using our own cowsay image (#2047)
 * aa6a536d fix(persistence): Allow `argo server` to run without persistence (#2050)
 * 71ba8238 Update README.md (#2045)
 * c7953052 fix(persistence): Allow `argo server` to run without persistence (#2050)
 * b0ee44ac fixed git push
 * e4cfefee revert cmd/server/static/files.go
 * ecdb8b09 v2.5.0-rc1
 * 6638936d Update manifests to 2.5.0-rc1
 * c3e02d81 Update Makefile
 * 43656c6e Update Makefile
 * b49d82d7 Update manifests to v2.5.0-rc1
 * 38bc90ac Update Makefile
 * 1db74e1a fix(archive): upsert archive + ci: Pin images on CI, add readiness probes, clean-up logging and other tweaks (#2038)
 * c46c6836 feat: Allow workflow-level parameters to be modified in the UI when submitting a workflow (#2030)
 * faa9dbb5 fix(Makefile): Rename staticfiles make target. Fixes #2010 (#2040)
 * 79a42d48 docs: Update link to configure-artifact-repository.md (#2041)
 * 1a96007f fix: Redirect to correct page when using managed namespace. Fixes #2015 (#2029)
 * 78726314 fix(api): Updates proto message naming (#2034)
 * 4a1307c8 feat: Adds support for MySQL. Fixes #1945 (#2013)
 * d843e608 chore: Smoke tests are timing out, give them more time (#2032)
 * 5c98a14e feat(controller): Add audit logs to workflows. Fixes #1769 (#1930)
 * 2982c1a8 fix(validate): Allow placeholder in values taken from inputs. Fixes #1984 (#2028)
 * 3293c83f feat: Add version to offload nodes. Fixes #1944 and #1946 (#1974)
 * 283bbf8d build: `make clean` now only deletes dist directories (#2019)
 * 72fa88c9 build: Enable linting for tests. Closes #1971 (#2025)
 * f8569ae9 feat: Auth refactoring to support single version token (#1998)
 * eb360d60 Fix README (#2023)
 * ef1bd3a3 fix typo (#2021)
 * f25a45de feat(controller): Exposes container runtime executor as CLI option. (#2014)
 * 3b26af7d Enable s3 trace support. Bump version to v2.5.0. Tweak proto id to match Workflow (#2009)
 * 5eb15bb5 fix: Fix workflow level timeouts (#1369)
 * 5868982b fix: Fixes the `test` job on master (#2008)
 * 29c85072 fix: Fixed grammar on TTLStrategy (#2006)
 * 2f58d202 fix: v2 token bug (#1991)
 * ed36d92f feat: Add quick start manifests to Git. Change auth-mode to default to server. Fixes #1990 (#1993)
 * d1965c93 docs: Encourage users to upvote issues relevant to them (#1996)
 * 91331a89 fix: No longer delete the argo ns as this is dangerous (#1995)
 * 1a777cc6 feat(cron): Added timezone support to cron workflows. Closes #1931 (#1986)
 * 48b85e57 fix: WorkflowTempalteTest fix (#1992)
 * 51dab8a4 feat: Adds `argo server` command. Fixes #1966 (#1972)
 * 732e03bb chore: Added WorkflowTemplate test (#1989)
 * 27387d4b chore: Fix UI TODOs (#1987)
 * dd704dd6 feat: Renders namespace in UI. Fixes #1952 and #1959 (#1965)
 * 14d58036 feat(server): Argo Server. Closes #1331 (#1882)
 * f69655a0 fix: Added decompress in retry, resubmit and resume. (#1934)
 * 1e7ccb53 updated jq version to 1.6 (#1937)
 * c51c1302 feat: Enhancement for namespace installation mode configuration (#1939)
 * 6af100d5 feat: Add suspend and resume to CronWorkflows CLI (#1925)
 * 232a465d feat: Added onExit handlers to Step and DAG (#1716)
 * 071eb112 docs: Update PR template to demand tests. (#1929)
 * ae58527e docs: Add CyberAgent to the list of Argo users (#1926)
 * 02022e4b docs: Minor formatting fix (#1922)
 * e4107bb8 Updated Readme.md for companies using Argo: (#1916)
 * 7e9b2b58 feat: Support for scheduled Workflows with CronWorkflow CRD (#1758)
 * 5d7e9185 feat: Provide values of withItems maps as JSON in {{item}}. Fixes #1905 (#1906)
 * de3ffd78  feat: Enhanced Different TTLSecondsAfterFinished depending on if job is in Succeeded, Failed or Error, Fixes (#1883)
 * 94449876 docs: Add question option to issue templates (#1910)
 * 83ae2df4 fix: Decrease docker build time by ignoring node_modules (#1909)
 * 59a19069 feat: support iam roles for service accounts in artifact storage (#1899)
 * 6526b6cc fix: Revert node creation logic (#1818)
 * 160a7940 fix: Update Gopkg.lock with dep ensure -update (#1898)
 * ce78227a fix: quick fail after pod termination (#1865)
 * cd3bd235 refactor: Format Argo UI using prettier (#1878)
 * b48446e0 fix: Fix support for continueOn failed for DAG. Fixes #1817 (#1855)
 * 48256961 fix: Fix template scope (#1836)
 * eb585ef7 fix: Use dynamically generated placeholders (#1844)
 * c821cfcc test: Adds 'test' and 'ui' jobs to CI (#1869)
 * 54f44909 feat: Always archive logs if in config. Closes #1790 (#1860)
 * 1e25d6cf docs: Fix e2e testing link (#1873)
 * f5f40728 fix: Minor comment fix (#1872)
 * 72fad7ec Update docs (#1870)
 * 90352865 docs: Update doc based on helm 3.x changes (#1843)
 * 78889895 Move Workflows UI from https://github.com/argoproj/argo-ui (#1859)
 * 4b96172f docs: Refactored and cleaned up docs (#1856)
 * 6ba4598f test: Adds core e2e test infra. Fixes #678 (#1854)
 * 87f26c8d fix: Move ISSUE_TEMPLATE/ under .github/ (#1858)
 * bd78d159 fix: Ensure timer channel is empty after stop (#1829)
 * afc63024 Code duplication (#1482)
 * 5b136713 docs: biobox analytics (#1830)
 * 68b72a8f add CCRi to list of users in README (#1845)
 * 941f30aa Add Sidecar Technologies to list of who uses Argo (#1850)
 * a08048b6 Adding Wavefront to the users list (#1852)
 * 1cb68c98 docs: Updates issue and PR templates. (#1848)
 * cb0598ea Fixed Panic if DB context has issue (#1851)
 * e5fb8848 fix: Fix a couple of nil derefs (#1847)
 * b3d45850 Add HOVER to the list of who uses Argo (#1825)
 * 99db30d6 InsideBoard uses Argo (#1835)
 * ac8efcf4 Red Hat uses Argo (#1828)
 * 41ed3acf Adding Fairwinds to the list of companies that use Argo (#1820)
 * 5274afb9 Add exponential back-off to retryStrategy (#1782)
 * e522e30a Handle operation level errors PVC in Retry (#1762)
 * f2e6054e Do not resolve remote templates in lint (#1787)
 * 3852bc3f SSL enabled database connection for workflow repository (#1712) (#1756)
 * f2676c87 Fix retry node name issue on error (#1732)
 * d38a107c Refactoring Template Resolution Logic (#1744)
 * 23e94604 Error occurred on pod watch should result in an error on the wait container (#1776)
 * 57d051b5 Added hint when using certain tokens in when expressions (#1810)
 * 0e79edff Make kubectl print status and start/finished time (#1766)
 * 723b3c15 Fix code-gen docs (#1811)
 * 711bb114 Fix withParam node naming issue (#1800)
 * 4351a336 Minor doc fix (#1808)
 * efb748fe Fix some issues in examples (#1804)
 * a3e31289 Add documentation for executors (#1778)
 * 1ac75b39 Add  to linter (#1777)
 * 3bead0db Add ability to retry nodes after errors (#1696)
 * b50845e2 Support no-headers flag (#1760)
 * 7ea2b2f8 Minor rework of suspened node (#1752)
 * 9ab1bc88 Update README.md (#1768)
 * e66fa328 Fixed lint issues (#1739)
 * 63e12d09 binary up version (#1748)
 * 1b7f9bec Minor typo fix (#1754)
 * 4c002677 fix blank lines (#1753)
 * fae73826 Fail suspended steps after deadline (#1704)
 * b2d7ee62 Fix typo in docs (#1745)
 * f2592448 Removed uneccessary debug Println (#1741)
 * 846d01ed Filter workflows in list  based on name prefix (#1721)
 * 8ae688c6 Added ability to auto-resume from suspended state (#1715)
 * fb617b63 unquote strings from parameter-file (#1733)
 * 34120341 example for pod spec from output of previous step (#1724)
 * 12b983f4 Add gonum.org/v1/gonum/graph to Gopkg.toml (#1726)
 * 327fcb24 Added  Protobuf extension  (#1601)
 * 602e5ad8 Fix invitation link. (#1710)
 * eb29ae4c Fixes bugs in demo (#1700)
 * ebb25b86 `restartPolicy` -> `retryStrategy` in examples (#1702)
 * 167d65b1 Fixed incorrect `pod.name` in retry pods (#1699)
 * e0818029 fixed broke metrics endpoint per #1634 (#1695)
 * 36fd09a1 Apply Strategic merge patch against the pod spec (#1687)
 * d3546467 Fix retry node processing (#1694)
 * dd517e4c Print multiple workflows in one command (#1650)
 * 09a6cb4e Added status of previous steps as variables (#1681)
 * ad3dd4d4 Fix issue that workflow.priority substitution didn't pass validation (#1690)
 * 095d67f8 Store locally referenced template properly (#1670)
 * 30a91ef0 Handle retried node properly (#1669)
 * 263cb703 Update README.md  Argo Ansible role: Provisioning Argo Workflows on Kubernetes/OpenShift (#1673)
 * 867f5ff7 Handle sidecar killing properly (#1675)
 * f0ab9df9 Fix typo (#1679)
 * 502db42d Don't provision VM for empty artifacts (#1660)
 * b5dcac81 Resolve WorkflowTemplate lazily (#1655)
 * d15994bb [User] Update Argo users list (#1661)
 * 4a654ca6 Stop failing if artifact file exists, but empty (#1653)
 * c6cddafe Bug fixes in getting started (#1656)
 * ec788373 Update workflow_level_host_aliases.yaml (#1657)
 * 7e5af474 Fix child node template handling (#1654)
 * 7f385a6b Use stored templates to raggregate step outputs (#1651)
 * cd6f3627 Fix dag output aggregation correctly (#1649)
 * 706075a5 Fix DAG output aggregation (#1648)
 * fa32dabd Fix missing merged changes in validate.go (#1647)
 * 45716027 fixed example wrong comment (#1643)
 * 69fd8a58 Delay killing sidecars until artifacts are saved (#1645)
 * ec5f9860 pin colinmarc/hdfs to the next commit, which no longer has vendored deps (#1622)
 * 4b84f975 Fix global lint issue (#1641)
 * bb579138 Fix regression where global outputs were unresolveable in DAGs (#1640)
 * cbf99682 Fix regression where parallelism could cause workflow to fail (#1639)

### Contributors

 * Adam Thornton
 * Aditya Sundaramurthy
 * Akshay Chitneni
 * Alessandro Marrella
 * Alex Collins
 * Alexander Matyushentsev
 * Alexey Volkov
 * Anastasia Satonina
 * Andrew Suderman
 * Antoine Dao
 * Avi Weit
 * Daisuke Taniwaki
 * David Seapy
 * Deepen Mehta
 * Derek Wang
 * Dineshmohan Rajaveeran
 * Ed Lee
 * Elton
 * Erik Parmann
 * Huan-Cheng Chang
 * Jesse Suen
 * Jialu Zhu
 * Jonathan Steele
 * Jonathon Belotti
 * Juan C. Muller
 * Julian Fahrer
 * Julian Mazzitelli
 * Marek Čermák
 * MengZeLee
 * Michael Crenshaw
 * Neutron Soutmun
 * Nick Groszewski
 * Niklas Hansson
 * Patryk Jeziorowski
 * Pavel Kravchenko
 * Per Buer
 * Praneet Chandra
 * Rafał Bigaj
 * Rick Avendaño
 * Saravanan Balasubramanian
 * Shubham Koli (FaultyCarry)
 * Simon Behar
 * Takashi Abe
 * Tobias Bradtke
 * Tom Wieczorek
 * Tristan Colgate-McFarlane
 * Vincent Boulineau
 * Wei Yan
 * William Reed
 * Zhipeng Wang
 * descrepes
 * dherman
 * fsiegmund
 * gerdos82
 * mark9white
 * nglinh
 * sang
 * vdinesh2461990
 * zhujl1991

## v2.4.3 (2019-12-05)

 * cfe5f377 Update version to v2.4.3
 * 256e9a2a Update version to v2.4.3
 * b99e6a0e Error occurred on pod watch should result in an error on the wait container (#1776)
 * b00fea14 SSL enabled database connection for workflow repository (#1712) (#1756)
 * 400274f4 Added hint when using certain tokens in when expressions (#1810)
 * 15a0aa7a Handle operation level errors PVC in Retry (#1762)
 * 81c7f5bd Do not resolve remote templates in lint (#1787)
 * 20cec1d9 Fix retry node name issue on error (#1732)
 * 468cb8fe Refactoring Template Resolution Logic (#1744)
 * 67369fb3 Support no-headers flag (#1760)
 * 340ab073 Filter workflows in list  based on name prefix (#1721)
 * e9581273 Added ability to auto-resume from suspended state (#1715)
 * a0a1b6fb Fixed incorrect `pod.name` in retry pods (#1699)

### Contributors

 * Antoine Dao
 * Daisuke Taniwaki
 * Saravanan Balasubramanian
 * Simon Behar
 * gerdos82
 * sang

## v2.4.2 (2019-10-21)

 * 675c6626 fixed broke metrics endpoint per #1634 (#1695)
 * 1a9310c6 Apply Strategic merge patch against the pod spec (#1687)
 * 0d0562aa Fix retry node processing (#1694)
 * 08f49d01 Print multiple workflows in one command (#1650)
 * defbc297 Added status of previous steps as variables (#1681)
 * 6ac44330 Fix issue that workflow.priority substitution didn't pass validation (#1690)
 * ab9d710a Update version to v2.4.2
 * 338af3e7 Store locally referenced template properly (#1670)
 * be0929dc Handle retried node properly (#1669)
 * 88e210de Update README.md  Argo Ansible role: Provisioning Argo Workflows on Kubernetes/OpenShift (#1673)
 * 946b0fa2 Handle sidecar killing properly (#1675)
 * 4ce972bd Fix typo (#1679)

### Contributors

 * Daisuke Taniwaki
 * Marek Čermák
 * Rick Avendaño
 * Saravanan Balasubramanian
 * Simon Behar
 * Tobias Bradtke
 * mark9white

## v2.4.1 (2019-10-08)

 * d7f09999 Update version to v2.4.1
 * 6b876b20 Don't provision VM for empty artifacts (#1660)
 * 0d00a52e Resolve WorkflowTemplate lazily (#1655)
 * effd7c33 Stop failing if artifact file exists, but empty (#1653)
 * a6576314 Fix child node template handling (#1654)
 * 982c7c55 Use stored templates to raggregate step outputs (#1651)
 * a8305ed7 Fix dag output aggregation correctly (#1649)
 * f14dd56d Fix DAG output aggregation (#1648)
 * 30c3b937 Fix missing merged changes in validate.go (#1647)
 * 85f50e30 fixed example wrong comment (#1643)
 * 09e22fb2 Delay killing sidecars until artifacts are saved (#1645)
 * 99e28f1c pin colinmarc/hdfs to the next commit, which no longer has vendored deps (#1622)
 * 885aae40 Fix global lint issue (#1641)
 * d9c4d236 Merge branch 'release-2.4' of https://github.com/argoproj/argo into release-2.4
 * 972abdd6 Fix regression where global outputs were unresolveable in DAGs (#1640)
 * 7272bec4 Fix regression where parallelism could cause workflow to fail (#1639)
 * 6b77abb2 Add back SetGlogLevel calls
 * e7544f3d Update version to v2.4.0
 * 35eae441 Merge branch 'master' into release-2.4
 * 45d65889 Merge branch 'master' into release-2.4
 * 76461f92 Update CHANGELOG for v2.4.0 (#1636)
 * c75a0861 Regenerate installation manifests (#1638)
 * e20cb28c Grant get secret role to controller to support persistence (#1615)
 * 644946e4 Save stored template ID in nodes (#1631)
 * 5d530bec Fix retry workflow state (#1632)
 * 2f0af522 Update operator.go (#1630)
 * 6acea0c1 Store resolved templates (#1552)
 * df8260d6 Increase timeout of golangci-lint (#1623)
 * 138f89f6 updated invite link (#1621)
 * d027188d Updated the API Rule Violations list (#1618)
 * a317fbf1 Prevent controller from crashing due to glog writing to /tmp (#1613)
 * 20e91ea5 Added WorkflowStatus and NodeStatus types to the Open API Spec. (#1614)
 * ffb281a5 Small code cleanup and add tests (#1562)
 * 1cb8345d Add merge keys to Workflow objects to allow for StrategicMergePatches (#1611)
 * c855a66a Increased Lint timeout (#1612)
 * 4bf83fc3 Fix DAG enable failFast will hang in some case (#1595)
 * e9f3d9cb Do not relocate the mounted docker.sock (#1607)
 * 1bd50fa2 Added retry around RuntimeExecutor.Wait call when waiting for main container completion (#1597)
 * 0393427b Issue1571  Support ability to assume IAM roles in S3 Artifacts  (#1587)
 * ffc0c84f Update Gopkg.toml and Gopkg.lock (#1596)
 * aa3a8f1c Update from github.com/ghodss/yaml to sigs.k8s.io/yaml (#1572)
 * 07a26f16 Regard resource templates as leaf nodes (#1593)
 * 89e959e7 Fix workflow template in namespaced controller (#1580)
 * cd04ab8b remove redundant codes (#1582)
 * 5bba8449 Add entrypoint label to workflow default labels (#1550)
 * 9685d7b6 Fix inputs and arguments during template resolution (#1545)
 * 19210ba6 added DataStax as an organization that uses Argo (#1576)
 * b5f2fdef Support AutomountServiceAccountToken and executor specific service account(#1480)
 * 8808726c Fix issue saving outputs which overlap paths with inputs (#1567)
 * ba7a1ed6 Add coverage make target (#1557)
 * ced0ee96 Document workflow controller dockerSockPath config (#1555)
 * 3e95f2da Optimize argo binary install documentation (#1563)
 * e2ebb166 docs(readme): fix workflow types link (#1560)
 * 6d150a15 Initialize the wfClientset before using it (#1548)
 * 5331fc02 Remove GLog config from argo executor (#1537)
 * ed4ac6d0 Update main.go (#1536)

### Contributors

 * Alexander Matyushentsev
 * Alexey Volkov
 * Anastasia Satonina
 * Anes Benmerzoug
 * Brian Mericle
 * Daisuke Taniwaki
 * David Seapy
 * Ed Lee
 * Erik Parmann
 * Ian Howell
 * Jesse Suen
 * John Wass
 * Jonathon Belotti
 * Mostapha Sadeghipour Roudsari
 * Pablo Osinaga
 * Premkumar Masilamani
 * Saravanan Balasubramanian
 * Simon Behar
 * Takayuki Kasai
 * Xianlu Bird
 * Xie.CS
 * mark9white

## v2.4.0-rc1 (2019-08-08)


### Contributors


## v2.4.0 (2019-10-08)

 * a6576314 Fix child node template handling (#1654)
 * 982c7c55 Use stored templates to raggregate step outputs (#1651)
 * a8305ed7 Fix dag output aggregation correctly (#1649)
 * f14dd56d Fix DAG output aggregation (#1648)
 * 30c3b937 Fix missing merged changes in validate.go (#1647)
 * 85f50e30 fixed example wrong comment (#1643)
 * 09e22fb2 Delay killing sidecars until artifacts are saved (#1645)
 * 99e28f1c pin colinmarc/hdfs to the next commit, which no longer has vendored deps (#1622)
 * 885aae40 Fix global lint issue (#1641)
 * d9c4d236 Merge branch 'release-2.4' of https://github.com/argoproj/argo into release-2.4
 * 972abdd6 Fix regression where global outputs were unresolveable in DAGs (#1640)
 * 7272bec4 Fix regression where parallelism could cause workflow to fail (#1639)
 * 6b77abb2 Add back SetGlogLevel calls
 * e7544f3d Update version to v2.4.0
 * 35eae441 Merge branch 'master' into release-2.4
 * 45d65889 Merge branch 'master' into release-2.4
 * 76461f92 Update CHANGELOG for v2.4.0 (#1636)
 * c75a0861 Regenerate installation manifests (#1638)
 * e20cb28c Grant get secret role to controller to support persistence (#1615)
 * 644946e4 Save stored template ID in nodes (#1631)
 * 5d530bec Fix retry workflow state (#1632)
 * 2f0af522 Update operator.go (#1630)
 * 6acea0c1 Store resolved templates (#1552)
 * df8260d6 Increase timeout of golangci-lint (#1623)
 * 138f89f6 updated invite link (#1621)
 * d027188d Updated the API Rule Violations list (#1618)
 * a317fbf1 Prevent controller from crashing due to glog writing to /tmp (#1613)
 * 20e91ea5 Added WorkflowStatus and NodeStatus types to the Open API Spec. (#1614)
 * ffb281a5 Small code cleanup and add tests (#1562)
 * 1cb8345d Add merge keys to Workflow objects to allow for StrategicMergePatches (#1611)
 * c855a66a Increased Lint timeout (#1612)
 * 4bf83fc3 Fix DAG enable failFast will hang in some case (#1595)
 * e9f3d9cb Do not relocate the mounted docker.sock (#1607)
 * 1bd50fa2 Added retry around RuntimeExecutor.Wait call when waiting for main container completion (#1597)
 * 0393427b Issue1571  Support ability to assume IAM roles in S3 Artifacts  (#1587)
 * ffc0c84f Update Gopkg.toml and Gopkg.lock (#1596)
 * aa3a8f1c Update from github.com/ghodss/yaml to sigs.k8s.io/yaml (#1572)
 * 07a26f16 Regard resource templates as leaf nodes (#1593)
 * 89e959e7 Fix workflow template in namespaced controller (#1580)
 * cd04ab8b remove redundant codes (#1582)
 * 5bba8449 Add entrypoint label to workflow default labels (#1550)
 * 9685d7b6 Fix inputs and arguments during template resolution (#1545)
 * 19210ba6 added DataStax as an organization that uses Argo (#1576)
 * b5f2fdef Support AutomountServiceAccountToken and executor specific service account(#1480)
 * 8808726c Fix issue saving outputs which overlap paths with inputs (#1567)
 * ba7a1ed6 Add coverage make target (#1557)
 * ced0ee96 Document workflow controller dockerSockPath config (#1555)
 * 3e95f2da Optimize argo binary install documentation (#1563)
 * e2ebb166 docs(readme): fix workflow types link (#1560)
 * 6d150a15 Initialize the wfClientset before using it (#1548)
 * 6131721f Remove GLog config from argo executor (#1537)
 * 5331fc02 Remove GLog config from argo executor (#1537)
 * 8e94ca37 Update main.go (#1536)
 * ed4ac6d0 Update main.go (#1536)
 * dfb06b6d Update version to v2.4.0-rc1
 * 9fca1441 Update argo dependencies to kubernetes v1.14 (#1530)
 * 0246d184 Use cache to retrieve WorkflowTemplates (#1534)
 * 4864c32f Update README.md (#1533)
 * 4df114fa Update CHANGELOG for v2.4 (#1531)
 * c7e5cba1 Introduce podGC strategy for deleting completed/successful pods (#1234)
 * bb0d14af Update ISSUE_TEMPLATE.md (#1528)
 * b5702d8a Format sources and order imports with the help of goimports (#1504)
 * d3ff77bf Added Architecture doc (#1515)
 * fc1ec1a5 WorkflowTemplate CRD (#1312)
 * f99d3266 Expose all input parameters to template as JSON (#1488)
 * bea60526 Fix argo logs empty content when workflow run in virtual kubelet env (#1201)
 * d82de881 Implemented support for WorkflowSpec.ArtifactRepositoryRef (#1350)
 * 0fa20c7b Fix validation (#1508)
 * 87e2cb60 Add --dry-run option to `argo submit` (#1506)
 * e7e50af6 Support git shallow clones and additional ref fetches (#1521)
 * 605489cd Allow overriding workflow labels in 'argo submit' (#1475)
 * 47eba519 Fix issue [Documentation] kubectl get service argo-artifacts -o wide (#1516)
 * 02f38262 Fixed #1287 Executor kubectl version is obsolete (#1513)
 * f62105e6 Allow Makefile variables to be set from the command line (#1501)
 * e62be65b Fix a compiler error in a unit test (#1514)
 * 5c5c29af Fix the lint target (#1505)
 * e03287bf Allow output parameters with .value, not only .valueFrom (#1336)
 * 781d3b8a Implemented Conditionally annotate outputs of script template only when consumed #1359 (#1462)
 * b028e61d change 'continue-on-fail' example to better reflect its description (#1494)
 * 97e824c9 Readme update to add argo and airflow comparison (#1502)
 * 414d6ce7 Fix a compiler error (#1500)
 * ca1d5e67 Fix: Support the List within List type in withParam #1471 (#1473)
 * 75cb8b9c Fix #1366 unpredictable global artifact behavior (#1461)
 * 082e5c4f Exposed workflow priority as a variable (#1476)
 * 38c4def7 Fix: Argo CLI should show warning if there is no workflow definition in file #1486
 * af7e496d Add Commodus Tech as official user (#1484)
 * 8c559642 Update OWNERS (#1485)
 * 007d1f88 Fix: 1008 `argo wait` and `argo submit --wait` should exit 1 if workflow fails  (#1467)
 * 3ab7bc94 Document the insecureIgnoreHostKey git flag (#1483)
 * 7d9bb51a Fix failFast bug:   When a node in the middle fails, the entire workflow will hang (#1468)
 * 42adbf32 Add --no-color flag to logs (#1479)
 * 67fc29c5 fix typo: symboloic > symbolic (#1478)
 * 7c3e1901 Added Codec to the Argo community list (#1477)
 * 0a9cf9d3 Add doc about failFast feature (#1453)
 * 6a590300 Support PodSecurityContext (#1463)
 * e392d854 issue-1445: changing temp directory for output artifacts from root to tmp (#1458)
 * 7a21adfe New Feature:  provide failFast flag, allow a DAG to run all branches of the DAG (either success or failure) (#1443)
 * b9b87b7f Centralized Longterm workflow persistence storage  (#1344)
 * cb09609b mention sidecar in failure message for sidecar containers (#1430)
 * 373bbe6e Fix demo's doc issue of install minio chart (#1450)
 * 83552334 Add threekit to user list (#1444)
 * 83f82ad1 Improve bash completion (#1437)
 * ee0ec78a Update documentation for workflow.outputs.artifacts (#1439)
 * 9e30c06e Revert "Update demo.md (#1396)" (#1433)
 * c08de630 Add paging function for list command (#1420)
 * bba2f9cb Fixed:  Implemented Template level service account (#1354)
 * d635c1de Ability to configure hostPath mount for `/var/run/docker.sock` (#1419)
 * d2f7162a Terminate all containers within pod after main container completes (#1423)
 * 1607d74a PNS executor intermitently failed to capture entire log of script templates (#1406)
 * 5e47256c Fix typo (#1431)
 * 5635c33a Update demo.md (#1396)
 * 83425455 Add OVH as official user (#1417)
 * 82e5f63d Typo fix in ARTIFACT_REPO.md (#1425)
 * 15fa6f52 Update OWNERS (#1429)
 * 96b9a40e Orders uses alphabetically (#1411)
 * 6550e2cb chore: add IBM to official users section in README.md (#1409)
 * bc81fe28 Fiixed: persistentvolumeclaims already exists #1130 (#1363)
 * 6a042d1f Update README.md (#1404)
 * aa811fbd Update README.md (#1402)
 * abe3c99f Add Mirantis as an official user (#1401)
 * 18ab750a Added Argo Rollouts to README (#1388)
 * 67714f99 Make locating kubeconfig in example os independent (#1393)
 * 672dc04f Fixed: withParam parsing of JSON/YAML lists #1389 (#1397)
 * b9aec5f9 Fixed: make verify-codegen is failing on the master branch (#1399) (#1400)
 * 270aabf1 Fixed:  failed to save outputs: verify serviceaccount default:default has necessary privileges (#1362)
 * 163f4a5d Fixed: Support hostAliases in WorkflowSpec #1265 (#1365)
 * abb17478 Add Max Kelsen to USERS in README.md (#1374)
 * dc549193 Update docs for the v2.3.0 release and to use the stable tag
 * 4001c964 Update README.md (#1372)
 * 6c18039b Fix issue where a DAG with exhausted retries would get stuck Running (#1364)
 * d7e74fe3 Validate action for resource templates (#1346)
 * 810949d5 Fixed :  CLI Does Not Honor metadata.namespace #1288 (#1352)
 * e58859d7 [Fix #1242] Failed DAG nodes are now kept and set to running on RetryWorkflow. (#1250)
 * d5fe5f98 Use golangci-lint instead of deprecated gometalinter (#1335)
 * 26744d10 Support an easy way to set owner reference (#1333)
 * 8bf7578e Add --status filter for get command (#1325)
 * 3f6ac9c9 Update release instructions

### Contributors

 * Aisuko
 * Alex Capras
 * Alex Collins
 * Alexander Matyushentsev
 * Alexey Volkov
 * Anastasia Satonina
 * Anes Benmerzoug
 * Ben Wells
 * Brandon Steinman
 * Brian Mericle
 * Christian Muehlhaeuser
 * Cristian Pop
 * Daisuke Taniwaki
 * Daniel Duvall
 * David Seapy
 * Ed Lee
 * Edwin Jacques
 * Erik Parmann
 * Ian Howell
 * Jacob O'Farrell
 * Jaime
 * Jean-Louis Queguiner
 * Jesse Suen
 * John Wass
 * Jonathon Belotti
 * Mostapha Sadeghipour Roudsari
 * Mukulikak
 * Orion Delwaterman
 * Pablo Osinaga
 * Paul Brit
 * Premkumar Masilamani
 * Saravanan Balasubramanian
 * Semjon Kopp
 * Stephen Steiner
 * Takayuki Kasai
 * Tim Schrodi
 * Xianlu Bird
 * Xie.CS
 * Ziyang Wang
 * alex weidner
 * commodus-sebastien
 * hidekuro
 * ianCambrio
 * jacky
 * mark9white
 * tralexa

## v2.3.0-rc3 (2019-05-07)

 * 2274130d Update version to v2.3.0-rc3
 * b024b3d8 Fix: # 1328 argo submit --wait and argo wait quits while workflow is running (#1347)
 * 24680b7f Fixed : Validate the secret credentials name and key (#1358)
 * f641d84e Fix input artifacts with multiple ssh keys (#1338)
 * e680bd21 add / test (#1240)
 * ee788a8a Fix #1340 parameter substitution bug (#1345)
 * 60b65190 Fix missing template local volumes, Handle volumes only used in init containers (#1342)
 * 4e37a444 Add documentation on releasing

### Contributors

 * Daisuke Taniwaki
 * Hideto Inamura
 * Ilias Katsakioris
 * Jesse Suen
 * Saravanan Balasubramanian
 * almariah

## v2.3.0-rc2 (2019-04-21)

 * bb1bfdd9 Update version to v2.3.0-rc2. Update changelog
 * 49a6b6d7 wait will conditionally become privileged if main/sidecar privileged (resolves #1323)
 * 34af5a06 Fix regression where argoexec wait would not return when podname was too long
 * bd8d5cb4 `argo list` was not displaying non-zero priorities correctly
 * 64370a2d Support parameter substitution in the volumes attribute (#1238)
 * 6607dca9 Issue1316 Pod creation with secret volumemount  (#1318)
 * a5a2bcf2 Update README.md (#1321)
 * 950de1b9 Export the methods of `KubernetesClientInterface` (#1294)
 * 1c729a72 Update v2.3.0 CHANGELOG.md

### Contributors

 * Chris Chambers
 * Ed Lee
 * Ilias Katsakioris
 * Jesse Suen
 * Saravanan Balasubramanian

## v2.3.0-rc1 (2019-04-10)


### Contributors


## v2.3.0 (2019-05-20)

 * 88fcc70d Update VERSION to v2.3.0, changelog, and manifests
 * 1731cd7c Fix issue where a DAG with exhausted retries would get stuck Running (#1364)
 * 3f6ac9c9 Update release instructions
 * 2274130d Update version to v2.3.0-rc3
 * b024b3d8 Fix: # 1328 argo submit --wait and argo wait quits while workflow is running (#1347)
 * 24680b7f Fixed : Validate the secret credentials name and key (#1358)
 * f641d84e Fix input artifacts with multiple ssh keys (#1338)
 * e680bd21 add / test (#1240)
 * ee788a8a Fix #1340 parameter substitution bug (#1345)
 * 60b65190 Fix missing template local volumes, Handle volumes only used in init containers (#1342)
 * 4e37a444 Add documentation on releasing
 * bb1bfdd9 Update version to v2.3.0-rc2. Update changelog
 * 49a6b6d7 wait will conditionally become privileged if main/sidecar privileged (resolves #1323)
 * 34af5a06 Fix regression where argoexec wait would not return when podname was too long
 * bd8d5cb4 `argo list` was not displaying non-zero priorities correctly
 * 64370a2d Support parameter substitution in the volumes attribute (#1238)
 * 6607dca9 Issue1316 Pod creation with secret volumemount  (#1318)
 * a5a2bcf2 Update README.md (#1321)
 * 950de1b9 Export the methods of `KubernetesClientInterface` (#1294)
 * 1c729a72 Update v2.3.0 CHANGELOG.md
 * 40f9a875 Reorganize manifests to kustomize 2 and update version to v2.3.0-rc1
 * 75b28a37 Implement support for PNS (Process Namespace Sharing) executor (#1214)
 * b4edfd30 Fix SIGSEGV in watch/CheckAndDecompress. Consolidate duplicate code (resolves #1315)
 * 02550be3 Archive location should conditionally be added to template only when needed
 * c60010da Fix nil pointer dereference with secret volumes (#1314)
 * db89c477 Fix formatting issues in examples documentation (#1310)
 * 0d400f2c Refactor checkandEstimate to optimize podReconciliation (#1311)
 * bbdf2e2c Add alibaba cloud to officially using argo list (#1313)
 * abb77062 CheckandEstimate implementation to optimize podReconciliation (#1308)
 * 1a028d54 Secrets should be passed to pods using volumes instead of API calls (#1302)
 * e34024a3 Add support for init containers (#1183)
 * 4591e44f Added support for artifact path references (#1300)
 * 928e4df8 Add Karius to users in README.md (#1305)
 * de779f36 Add community meeting notes link (#1304)
 * a8a55579 Speed up podReconciliation using parallel goroutine (#1286)
 * 93451119 Add dns config support (#1301)
 * 850f3f15 Admiralty: add link to blog post, add user (#1295)
 * d5f4b428 Fix for Resource creation where template has same parameter templating (#1283)
 * 9b555cdb Issue#896 Workflow steps with non-existant output artifact path will succeed (#1277)
 * adab9ed6 Argo CI is current inactive (#1285)
 * 59fcc5cc Add workflow labels and annotations global vars (#1280)
 * 1e111caa Fix bug with DockerExecutor's CopyFile (#1275)
 * 73a37f2b Add the `mergeStrategy` option to resource patching (#1269)
 * e6105243 Reduce redundancy pod label action (#1271)
 * 4bfbb20b Error running 1000s of tasks: "etcdserver: request is too large" #1186 (#1264)
 * b2743f30 Proxy Priority and PriorityClassName to pods (#1179)
 * 70c130ae Update versions (#1218)
 * b0384129 Git cloning via SSH was not verifying host public key (#1261)
 * 3f06385b Issue#1165 fake outputs don't notify and task completes successfully (#1247)
 * fa042aa2 typo, executo -> executor (#1243)
 * 1cb88bae Fixed Issue#1223 Kubernetes Resource action: patch is not supported (#1245)
 * 2b0b8f1c Fix the Prometheus address references (#1237)
 * 94cda3d5 Add feature to continue workflow on failed/error steps/tasks (#1205)
 * 3f1fb9d5 Add Gardener to "Who uses Argo" (#1228)
 * cde5cd32 Include stderr when retrieving docker logs (#1225)
 * 2b1d56e7 Update README.md (#1224)
 * eeac5a0e Remove extra quotes around output parameter value (#1232)
 * 8b67e1bf Update README.md (#1236)
 * baa3e622 Update README with typo fixes (#1220)
 * f6b0c8f2 Executor can access the k8s apiserver with a out-of-cluster config file (#1134)
 * 0bda53c7 fix dag retries (#1221)
 * 8aae2931 Issue #1190 - Fix incorrect retry node handling (#1208)
 * f1797f78 Add schedulerName to workflow and template spec (#1184)
 * 2ddae161 Set executor image pull policy for resource template (#1174)
 * edcb5629 Dockerfile: argoexec base image correction (fixes #1209) (#1213)
 * f92284d7 Minor spelling, formatting, and style updates. (#1193)
 * bd249a83 Issue #1128 - Use polling instead of fs notify to get annotation changes (#1194)
 * 14a432e7 Update community/README (#1197)
 * eda7e084 Updated OWNERS (#1198)
 * 73504a24 Fischerjulian adds ruby to rest docs (#1196)
 * 311ad86f Fix missing docker binary in argoexec image. Improve reuse of image layers
 * 831e2198 Issue #988 - Submit should not print logs to stdout unless output is 'wide' (#1192)
 * 17250f3a Add documentation how to use parameter-file's (#1191)
 * 01ce5c3b Add Docker Hub build hooks
 * 93289b42 Refactor Makefile/Dockerfile to remove volume binding in favor of build stages (#1189)
 * 8eb4c666 Issue #1123 - Fix 'kubectl get' failure if resource namespace is different from workflow namespace (#1171)
 * eaaad7d4 Increased S3 artifact retry time and added log (#1138)
 * f07b5afe Issue #1113 - Wait for daemon pods completion to handle annotations (#1177)
 * 2b2651b0 Do not mount unnecessary docker socket (#1178)
 * 1fc03144 Argo users: Equinor (#1175)
 * e381653b Update README. (#1173) (#1176)
 * 5a917140 Update README and preview notice in CLA.
 * 521eb25a Validate ArchiveLocation artifacts (#1167)
 * 528e8f80 Add missing patch in namespace kustomization.yaml (#1170)
 * 0b41ca0a Add Preferred Networks to users in README.md (#1172)
 * 649d64d1 Add GitHub to users in README.md (#1151)
 * 864c7090 Update codegen for network config (#1168)
 * c3cc51be Support HDFS Artifact (#1159)
 * 8db00066 add support for hostNetwork & dnsPolicy config (#1161)
 * 149d176f Replace exponential retry with poll (#1166)
 * 31e5f63c Fix tests compilation error (#1157)
 * 6726d9a9 Fix failing TestAddGlobalArtifactToScope unit test
 * 4fd758c3 Add slack badge to README (#1164)
 * 3561bff7 Issue #1136 - Fix metadata for DAG with loops (#1149)
 * c7fec9d4 Reflect minio chart changes in documentation (#1147)
 * f6ce7833 add support for other archs (#1137)
 * cb538489 Fix issue where steps with exhausted retires would not complete (#1148)
 * e400b65c Fix global artifact overwriting in nested workflow (#1086)
 * 174eb20a Issue #1040 - Kill daemoned step if workflow consist of single daemoned step (#1144)
 * e078032e Issue #1132 - Fix panic in ttl controller (#1143)
 * e09d9ade Issue #1104 - Remove container wait timeout from 'argo logs --follow' (#1142)
 * 0f84e514 Allow owner reference to be set in submit util (#1120)
 * 3484099c Update generated swagger to fix verify-codegen (#1131)
 * 587ab1a0 Fix output artifact and parameter conflict (#1125)
 * 6bb3adbc Adding Quantibio in Who uses Argo (#1111)
 * 1ae3696c Install mime-support in argoexec to set proper mime types for S3 artifacts (resolves #1119)
 * 515a9005 add support for ppc64le and s390x (#1102)
 * 78142837 Remove docker_lib mount volume which is not needed anymore (#1115)
 * e59398ad Fix examples docs of parameters. (#1110)
 * ec20d94b Issue #1114 - Set FORCE_NAMESPACE_ISOLATION env variable in namespace install manifests (#1116)
 * 49c1fa4f Update docs with examples using the K8s REST API
 * bb8a6a58 Update ROADMAP.md
 * 46855dcd adding logo to be used by the OS Site (#1099)
 * 438330c3 #1081 added retry logic to s3 load and save function (#1082)
 * cb8b036b Initialize child node before marking phase. Fixes panic on invalid `When` (#1075)
 * 60b508dd Drop reference to removed `argo install` command. (#1074)
 * 62b24368 Fix typo in demo.md (#1089)
 * b5dfa021 Use relative links on README file (#1087)
 * 95b72f38 Update docs to outline bare minimum set of privileges for a workflow
 * d4ef6e94 Add new article and minor edits. (#1083)
 * afdac9bb Issue #740 - System level workflow parallelism limits & priorities (#1065)
 * a53a76e9 fix #1078 Azure AKS authentication issues (#1079)
 * 79b3e307 Fix string format arguments in workflow utilities. (#1070)
 * 76b14f54 Auto-complete workflow names (#1061)
 * f2914d63 Support nested steps workflow parallelism (#1046)
 * eb48c23a Raise not implemented error when artifact saving is unsupported (#1062)
 * 036969c0 Add Cratejoy to list of users (#1063)
 * a07bbe43 Adding SAP Hybris in Who uses Argo (#1064)
 * 7ef1cea6 Update dependencies to K8s v1.12 and client-go 9.0
 * 23d733ba Add namespace explicitly to pod metadata (#1059)
 * 79ed7665 Parameter and Argument names should support snake case (#1048)
 * 6e6c59f1 Submodules are dirty after checkout -- need to update (#1052)
 * f18716b7 Support for K8s API based Executor (#1010)
 * e297d195 Updated examples/README.md (#1051)
 * 19d6cee8 Updated ARTIFACT_REPO.md (#1049)

### Contributors

 * Adrien Trouillaud
 * Alexander Matyushentsev
 * Alexey Volkov
 * Andrei Miulescu
 * Anna Winkler
 * Bastian Echterhölter
 * Chen Zhiwei
 * Chris Chambers
 * Clemens Lange
 * Daisuke Taniwaki
 * Dan Norris
 * Divya Vavili
 * Ed Lee
 * Edward Lee
 * Erik Parmann
 * Fred Dubois
 * Greg Roodt
 * Hamel Husain
 * Hideto Inamura
 * Howie Benefiel
 * Ian Howell
 * Ilias K
 * Ilias Katsakioris
 * Ismail Alidzhikov
 * Jesse Suen
 * Johannes 'fish' Ziemke
 * Joshua Carp
 * Julian Fischer
 * Konstantin Zadorozhny
 * Marcin Karkocha
 * Matthew Coleman
 * Miyamae Yuuya
 * Naoto Migita
 * Naresh Kumar Amrutham
 * Nick Stott
 * Pengfei Zhao
 * Rocio Montes
 * Saravanan Balasubramanian
 * Tang Lee
 * Tim Schrodi
 * Val Sichkovskyi
 * WeiYan
 * Xianlu Bird
 * almariah
 * gerardaus
 * houz
 * jacky
 * jdfalko
 * kshamajain99
 * shahin
 * xubofei1983

## v2.2.1 (2018-10-11)

 * 0a928e93 Update installation manifests to use v2.2.1
 * 3b52b261 Fix linter warnings and update swagger
 * 7d0e77ba Update changelog and bump version to 2.2.1
 * b402e12f Issue #1033 - Workflow executor panic: workflows.argoproj.io/template workflows.argoproj.io/template not found in annotation file (#1034)
 * 3f2e986e fix typo in examples/README.md (#1025)
 * 9c5e056a Replace tabs with spaces (#1027)
 * 091f1407 Update README.md (#1030)
 * 159fe09c Fix format issues to resolve build errors (#1023)
 * 363bd97b Fix error in env syntax (#1014)
 * ae7bf0a5 Issue #1018 - Workflow controller should save information about archived logs in step outputs (#1019)
 * 15d006c5 Add example of workflow using imagePullSecrets (resolves #1013)
 * 2388294f Fix RBAC roles to include workflow delete for GC to work properly (resolves #1004)
 * 6f611cb9 Fix issue where resubmission of a terminated workflow creates a terminated workflow (issue #1011)
 * 4a7748f4 Disable Persistence in the demo example (#997)
 * 55ae0cb2 Fix example pod name (#1002)
 * c275e7ac Add imagePullPolicy config for executors (#995)
 * b1eed124 `tar -tf` will detect compressed tars correctly. (#998)
 * 03a7137c Add new organization using argo (#994)
 * 83884528 Update argoproj/pkg to trim leading/trailing whitespace in S3 credentials (resolves #981)
 * 978b4938 Add syntax highlighting for all YAML snippets and most shell snippets (#980)
 * 60d5dc11 Give control to decide whether or not to archive logs at a template level
 * 8fab73b1 Detect and indicate when container was OOMKilled
 * 47a9e556 Update config map doc with instructions to enable log archiving
 * 79dbbaa1 Add instructions to match git URL format to auth type in git example (issue #979)
 * 429f03f5 Add feature list to README.md. Tweaks to getting started.
 * 36fd1948 Update getting started guide with v2.2.0 instructions

### Contributors

 * Alexander Matyushentsev
 * Appréderisse Benjamin
 * Daisuke Taniwaki
 * David Bernard
 * Feynman Liang
 * Ilya Sotkov
 * Jesse Suen
 * Marco Sanvido
 * Matt Hillsdon
 * Sean Fern
 * WeiYan

## v2.2.0 (2018-08-30)


### Contributors


## v2.12.9 (2021-02-16)

 * 73790534 Update manifests to v2.12.9
 * 20e4c401 chore: Add SHA256 checksums to release (#5122)
 * 81c07344 codegen
 * 26d2ec0a cherry-picked 5081
 * 92ad730a fix: Revert "fix(controller): keep special characters in json string when … … 19da392 …use withItems (#4814)" (#5076)
 * 1e868ec1 fix(controller): Fix creator dashes (#5082)

### Contributors

 * Alex Collins
 * Saravanan Balasubramanian
 * Simon Behar

## v2.12.8 (2021-02-08)

 * d19d4eee Update manifests to v2.12.8
 * cf3b1980 fix: Fix build
 * a8d0b67e fix(cli): Add insecure-skip-verify for HTTP1. Fixes #5008 (#5015)
 * a3134de9 fix: Skip the Workflow not found error in Concurrency policy (#5030)
 * a60e4105 fix: Unmark daemoned nodes after stopping them (#5005)

### Contributors

 * Alex Collins
 * Saravanan Balasubramanian
 * Simon Behar

## v2.12.7 (2021-02-01)

 * 5f515073 Update manifests to v2.12.7
 * 637154d0 feat: Support retry on transient errors during executor status checking (#4946)
 * 8e7ed235 feat(server): Add Prometheus metrics. Closes #4751 (#4952)
 * 372cef03 chore: Remove self-referential replace directive in go.mod (#4959)
 * 77b34e3c chore: Set Go mod to /v2 (#4916)

### Contributors

 * Alex Collins
 * Simon Behar
 * Yuan Tang

## v2.12.6 (2021-01-25)

 * 4cb5b7eb Update manifests to v2.12.6
 * 2696898b fix: Mutex not being released on step completion (#4847)
 * 067b6036 feat(server): Support email for SSO+RBAC. Closes #4612 (#4644)

### Contributors

 * Alex Collins
 * Saravanan Balasubramanian
 * Simon Behar

## v2.12.5 (2021-01-19)

 * 53f022c3 Update manifests to v2.12.5
 * 86d7b3b6 fix tests
 * 63390940 fix tests
 * 0c7aa149 fix: Mutex not being released on step completion (#4847)
 * b3742193 fix(controller): Consider processed retry node in metrics. Fixes #4846 (#4872)
 * 9063a94d fix(controller): make creator label DNS compliant. Fixes #4880 (#4881)
 * 84b44cfd fix(controller): Fix node status when daemon pod deleted but its children nodes are still running (#4683)
 * 8cd96352 fix: Do not error on duplicate workflow creation by cron (#4871)

### Contributors

 * Saravanan Balasubramanian
 * Simon Behar
 * ermeaney
 * lonsdale8734

## v2.12.4 (2021-01-12)

 * f97bef5d Update manifests to v2.12.4
 * daba40ea build: Fix lint (#4837)
 * c521b27e feat: Publish images on Quay.io (#4860)
 * 1cd2570c feat: Publish images to Quay.io (#4854)
 * 7eb16e61 fix: Preserve the original slice when removing string (#4835)
 * e64183db fix(controller): keep special characters in json string when use withItems (#4814)
 * cd1b8b27 chore: add missing phase option to node cli command (#4825)

### Contributors

 * Alex Collins
 * Simon Behar
 * Song Juchao
 * cocotyty
 * markterm

## v2.12.3 (2021-01-04)

 * 93ee5301 Update manifests to v2.12.3
 * 3ce298e2 fix tests
 * 9bb1168e test: Fix TestDeletingRunningPod (#4779)
 * 8177b53c fix(controller): Various v2.12 fixes. Fixes #4798, #4801, #4806 (#4808)
 * 19c7bdab fix: load all supported authentication plugins for k8s client-go (#4802)
 * 331aa4ee fix(server): Do not silently ignore sso secret creation error (#4775)
 * 0bbc082c feat(controller): Rate-limit workflows. Closes #4718 (#4726)
 * a6027982 fix(controller): Support default database port. Fixes #4756 (#4757)
 * 5d857358 feat(controller): Enhanced TTL controller scalability (#4736)

### Contributors

 * Alex Collins
 * Kristoffer Johansson
 * Simon Behar

## v2.12.2 (2020-12-18)


### Contributors


## v2.12.11 (2021-04-05)

 * 71d00c78 Update manifests to v2.12.11
 * d5e0823f fix: InsecureSkipVerify true
 * 3b6c53af fix(executor): GODEBUG=x509ignoreCN=0 (#5562)
 * 051413e0 chore: Handling panic in go routines (#5489)
 * 631e55d0 feat(server): Enforce TLS >= v1.2 (#5172)

### Contributors

 * Alex Collins
 * Saravanan Balasubramanian
 * Simon Behar

## v2.12.10 (2021-03-08)

 * f1e0c617 Update manifests to v2.12.10
 * 1ecc5c00 fix(test): Flaky TestWorkflowShutdownStrategy  (#5331)
 * fa8f63c6 Cherry-pick 5289
 * d56c420b fix: Disallow object names with more than 63 chars (#5324)
 * 6ccfe46d fix: Backward compatible workflowTemplateRef from 2.11.x to  2.12.x (#5314)
 * 0ad73462 fix: Ensure whitespaces is allowed between name and bracket (#5176)
 * 73790534 Update manifests to v2.12.9
 * 20e4c401 chore: Add SHA256 checksums to release (#5122)
 * 81c07344 codegen
 * 26d2ec0a cherry-picked 5081
 * 92ad730a fix: Revert "fix(controller): keep special characters in json string when … … 19da392 …use withItems (#4814)" (#5076)
 * 1e868ec1 fix(controller): Fix creator dashes (#5082)
 * d19d4eee Update manifests to v2.12.8
 * cf3b1980 fix: Fix build
 * a8d0b67e fix(cli): Add insecure-skip-verify for HTTP1. Fixes #5008 (#5015)
 * a3134de9 fix: Skip the Workflow not found error in Concurrency policy (#5030)
 * a60e4105 fix: Unmark daemoned nodes after stopping them (#5005)
 * 5f515073 Update manifests to v2.12.7
 * 637154d0 feat: Support retry on transient errors during executor status checking (#4946)
 * 8e7ed235 feat(server): Add Prometheus metrics. Closes #4751 (#4952)
 * 372cef03 chore: Remove self-referential replace directive in go.mod (#4959)
 * 77b34e3c chore: Set Go mod to /v2 (#4916)
 * 4cb5b7eb Update manifests to v2.12.6
 * 2696898b fix: Mutex not being released on step completion (#4847)
 * 067b6036 feat(server): Support email for SSO+RBAC. Closes #4612 (#4644)
 * 53f022c3 Update manifests to v2.12.5
 * 86d7b3b6 fix tests
 * 63390940 fix tests
 * 0c7aa149 fix: Mutex not being released on step completion (#4847)
 * b3742193 fix(controller): Consider processed retry node in metrics. Fixes #4846 (#4872)
 * 9063a94d fix(controller): make creator label DNS compliant. Fixes #4880 (#4881)
 * 84b44cfd fix(controller): Fix node status when daemon pod deleted but its children nodes are still running (#4683)
 * 8cd96352 fix: Do not error on duplicate workflow creation by cron (#4871)
 * f97bef5d Update manifests to v2.12.4
 * daba40ea build: Fix lint (#4837)
 * c521b27e feat: Publish images on Quay.io (#4860)
 * 1cd2570c feat: Publish images to Quay.io (#4854)
 * 7eb16e61 fix: Preserve the original slice when removing string (#4835)
 * e64183db fix(controller): keep special characters in json string when use withItems (#4814)
 * cd1b8b27 chore: add missing phase option to node cli command (#4825)
 * 93ee5301 Update manifests to v2.12.3
 * 3ce298e2 fix tests
 * 9bb1168e test: Fix TestDeletingRunningPod (#4779)
 * 8177b53c fix(controller): Various v2.12 fixes. Fixes #4798, #4801, #4806 (#4808)
 * 19c7bdab fix: load all supported authentication plugins for k8s client-go (#4802)
 * 331aa4ee fix(server): Do not silently ignore sso secret creation error (#4775)
 * 0bbc082c feat(controller): Rate-limit workflows. Closes #4718 (#4726)
 * a6027982 fix(controller): Support default database port. Fixes #4756 (#4757)
 * 5d857358 feat(controller): Enhanced TTL controller scalability (#4736)
 * 7868e723 Update manifests to v2.12.2
 * e8c4aa4a fix(controller): Requeue when the pod was deleted. Fixes #4719 (#4742)
 * 11bc9c41 feat(controller): Pod deletion grace period. Fixes #4719 (#4725)

### Contributors

 * Alex Collins
 * Kristoffer Johansson
 * Saravanan Balasubramanian
 * Simon Behar
 * Song Juchao
 * Yuan Tang
 * cocotyty
 * ermeaney
 * lonsdale8734
 * markterm

## v2.12.1 (2020-12-17)

 * 9a7e044e Update manifests to v2.12.1
 * d21c4528 Change argo-server crt/key owner (#4750)
 * 53029017 Update manifests to v2.12.0
 * 43458066 fix(controller): Fixes resource version misuse. Fixes #4714 (#4741)
 * e192fb15 fix(executor): Copy main/executor container resources from controller by value instead of reference (#4737)
 * 4fb0d96d fix(controller): Fix incorrect main container customization precedence and isResourcesSpecified check (#4681)
 * 1aac79e9 feat(controller): Allow to configure main container resources (#4656)

### Contributors

 * Alex Collins
 * Daisuke Taniwaki
 * Simon Behar
 * Yuan Tang

## v2.12.0-rc6 (2020-12-15)

 * e55b886e Update manifests to v2.12.0-rc6
 * 1fb0d8b9 fix(controller): Fixed workflow stuck with mutex lock (#4744)
 * 4059820e fix(executor): Always check if resource has been deleted in checkResourceState() (#4738)
 * 739af45b fix(ui): Fix YAML for workflows with storedWorkflowTemplateSpec. Fixes #4691 (#4695)
 * 35980343 fix: Allow Bearer token in server mode (#4735)
 * bf589b01 fix(executor): Deal with the pod watch API call timing out (#4734)
 * 71edcfd4 ci: Pin kustomize version (#4704)
 * fabf20b5 fix(controller): Increate default EventSpamBurst in Eventrecorder (#4698)

### Contributors

 * Alex Collins
 * Saravanan Balasubramanian
 * Simon Behar
 * Yuan Tang
 * hermanhobnob

## v2.12.0-rc5 (2020-12-10)

 * 3aa86fff Update manifests to v2.12.0-rc5
 * 3581a1e7 fix: Consider optional artifact arguments (#4672)
 * 50210fc3 feat(controller): Use deterministic name for cron workflow children (#4638)
 * 3a4e974c fix(controller): Only patch status.active in cron workflows when syncing (#4659)
 * 2aaad26f fix(ui): DataLoaderDropdown fix input type from promise to function that (#4655)
 * 72ca92cb fix: Count Workflows with no phase as Pending for metrics (#4628)
 * 8ea219b8 fix(ui): Reference secrets in EnvVars. Fixes #3973  (#4419)
 * 3b35ba2b fix: derive jsonschema and fix up issues, validate examples dir… (#4611)
 * 20f59de9 docs: Add JSON schema for IDE validation (#4581)
 * 2f49720a fix(ui): Fixed reconnection hot-loop. Fixes #4580 (#4663)
 * 4f8e4a51 fix(controller): Cleanup the synchronize  pending queue once Workflow deleted (#4664)
 * 12859847 fix(controller): Deal with hyphen in creator. Fixes #4058 (#4643)
 * 2d05d56e feat(controller): Make MAX_OPERATION_TIME configurable. Close #4239 (#4562)
 * c00ff714 fix: Fix TestCleanFieldsExclude (#4625)

### Contributors

 * Alex Collins
 * Paul Brabban
 * Saravanan Balasubramanian
 * Simon Behar
 * aletepe
 * tczhao

## v2.12.0-rc4 (2020-12-02)

 * e34bc3b7 Update manifests to v2.12.0-rc4
 * feea63f0 feat(executor): More informative log when executors do not support output param from base image layer (#4620)
 * 65f5aefe fix(argo-server): fix global variable validation error with reversed dag.tasks (#4369)
 * e6870664 fix(server): Correct webhook event payload marshalling. Fixes #4572 (#4594)
 * b1d682e7 fix: Perform fields filtering server side (#4595)
 * 61b67048 fix: Null check pagination variable (#4617)
 * ace0ee1b fix(executor): Fixed waitMainContainerStart returning prematurely. Closes #4599 (#4601)
 * f03f99ef refactor: Use polling model for workflow phase metric (#4557)
 * 8e887e73 fix(executor): Handle sidecar killing in a process-namespace-shared pod (#4575)
 * 991fa674 fix(server): serve artifacts directly from disk to support large artifacts (#4589)
 * 2eeb1fce fix(server): use the correct name when downloading artifacts (#4579)
 * d1a37d5f feat(controller): Retry transient offload errors. Resolves #4464 (#4482)

### Contributors

 * Alex Collins
 * Daisuke Taniwaki
 * Simon Behar
 * Yuan Tang
 * dherman
 * fsiegmund
 * zhengchenyu

## v2.12.0-rc3 (2020-11-23)

 * 85cafe6e Update manifests to v2.12.0-rc3
 * 42e3af7a Merge branch 'master' into release-2.12
 * fdafc4ba chore: Updated stress test YAML (#4569)
 * 916b4549 feat(ui): Add Template/Cron workflow filter to workflow page. Closes #4532 (#4543)
 * 6e2b0cf3 docs: Clean-up examples. Fixes #4124 (#4128)
 * 4998b2d6 chore: Remove unused image build and push hooks (#4539)
 * 48af0244 fix: executor/pns containerid prefix fix (#4555)
 * 53195ed5 fix: Respect continueOn for leaf tasks (#4455)
 * e3d59f08 docs: Added CloudSeeds as one of the users for argo (#4553)
 * ad11180e docs: Update cost optimisation  to include information about cleaning up workflows and archiving (#4549)
 * 7e121509 fix(controller): Correct default port logic (#4547)
 * a712e535 fix: Validate metric key names (#4540)
 * c469b053 fix: Missing arg lines caused files not to copy into containers (#4542)
 * 0980ead3 fix(test): fix TestWFDefaultWithWFTAndWf flakiness (#4538)
 * 564e69f3 fix(ui): Do not auto-reload doc.location. Fixes #4530 (#4535)
 * eebcb8b8 docs: Add "Argo Workflows in 5 min" link to README (#4533)
 * 5147f0a1 Merge branch 'master' into release-2.12
 * 176d890c fix(controller): support float for param value (#4490)
 * 4bacbc12 feat(controller): make sso timeout configurable via cm (#4494)
 * 02e1f0e0 fix(server): Add `list sa` and `create secret` to `argo-server` roles. Closes #4526 (#4514)
 * d0082e8f fix: link templates not replacing multiple templates with same name (#4516)
 * 411bde37 feat: adds millisecond-level timestamps to argo and workflow-controller (#4518)
 * 2c54ca3f add bonprix to argo users (#4520)
 * 754a201d build: Reuse IMAGE_OS instead of hard-coded linux (#4519)

### Contributors

 * Alex Collins
 * Alexander Mikhailian
 * Arghya Sadhu
 * Boolman
 * David Gibbons
 * Espen Finnesand
 * Lennart Kindermann
 * Ludovic Cléroux
 * Oleg Borodai
 * Saravanan Balasubramanian
 * Simon Behar
 * Yuan Tang
 * Zach
 * tczhao

## v2.12.0-rc2 (2020-11-12)

 * f509fa55 Update manifests to v2.12.0-rc2
 * c297b2aa Merge branch 'master' into release-2.12
 * 2dab2d15 fix(test):  fix TestWFDefaultWithWFTAndWf flakiness (#4507)
 * 64ae3303 fix(controller): prepend script path to the script template args. Resolves #4481 (#4492)
 * 0931baf5 feat: Redirect to requested URL after SSO login (#4495)
 * 465447c0 fix: Ensure ContainerStatus in PNS is terminated before continuing (#4469)
 * f7287687 fix(ui): Check node children before counting them. (#4498)
 * bfc13c3f fix: Ensure redirect to login when using empty auth token (#4496)
 * d56ce890 feat(cli): add selector and field-selector option to terminate (#4448)
 * e501fcca fix(controller): Refactor the Merge Workflow, WorkflowTemplate and WorkflowDefaults (#4354)
 * 2ee3f5a7 fix(ui): fix the `all` option in the workflow archive list (#4486)

### Contributors

 * Noah Hanjun Lee
 * Saravanan Balasubramanian
 * Simon Behar
 * Vlad Losev
 * dherman
 * ivancili

## v2.12.0-rc1 (2020-11-06)

 * 5e2c656b chore: Correct manifests

### Contributors

 * Alex Collins

## v2.12.0 (2020-12-17)

 * 53029017 Update manifests to v2.12.0
 * 43458066 fix(controller): Fixes resource version misuse. Fixes #4714 (#4741)
 * e192fb15 fix(executor): Copy main/executor container resources from controller by value instead of reference (#4737)
 * 4fb0d96d fix(controller): Fix incorrect main container customization precedence and isResourcesSpecified check (#4681)
 * 1aac79e9 feat(controller): Allow to configure main container resources (#4656)
 * e55b886e Update manifests to v2.12.0-rc6
 * 1fb0d8b9 fix(controller): Fixed workflow stuck with mutex lock (#4744)
 * 4059820e fix(executor): Always check if resource has been deleted in checkResourceState() (#4738)
 * 739af45b fix(ui): Fix YAML for workflows with storedWorkflowTemplateSpec. Fixes #4691 (#4695)
 * 35980343 fix: Allow Bearer token in server mode (#4735)
 * bf589b01 fix(executor): Deal with the pod watch API call timing out (#4734)
 * 71edcfd4 ci: Pin kustomize version (#4704)
 * fabf20b5 fix(controller): Increate default EventSpamBurst in Eventrecorder (#4698)
 * 3aa86fff Update manifests to v2.12.0-rc5
 * 3581a1e7 fix: Consider optional artifact arguments (#4672)
 * 50210fc3 feat(controller): Use deterministic name for cron workflow children (#4638)
 * 3a4e974c fix(controller): Only patch status.active in cron workflows when syncing (#4659)
 * 2aaad26f fix(ui): DataLoaderDropdown fix input type from promise to function that (#4655)
 * 72ca92cb fix: Count Workflows with no phase as Pending for metrics (#4628)
 * 8ea219b8 fix(ui): Reference secrets in EnvVars. Fixes #3973  (#4419)
 * 3b35ba2b fix: derive jsonschema and fix up issues, validate examples dir… (#4611)
 * 20f59de9 docs: Add JSON schema for IDE validation (#4581)
 * 2f49720a fix(ui): Fixed reconnection hot-loop. Fixes #4580 (#4663)
 * 4f8e4a51 fix(controller): Cleanup the synchronize  pending queue once Workflow deleted (#4664)
 * 12859847 fix(controller): Deal with hyphen in creator. Fixes #4058 (#4643)
 * 2d05d56e feat(controller): Make MAX_OPERATION_TIME configurable. Close #4239 (#4562)
 * c00ff714 fix: Fix TestCleanFieldsExclude (#4625)
 * e34bc3b7 Update manifests to v2.12.0-rc4
 * feea63f0 feat(executor): More informative log when executors do not support output param from base image layer (#4620)
 * 65f5aefe fix(argo-server): fix global variable validation error with reversed dag.tasks (#4369)
 * e6870664 fix(server): Correct webhook event payload marshalling. Fixes #4572 (#4594)
 * b1d682e7 fix: Perform fields filtering server side (#4595)
 * 61b67048 fix: Null check pagination variable (#4617)
 * ace0ee1b fix(executor): Fixed waitMainContainerStart returning prematurely. Closes #4599 (#4601)
 * f03f99ef refactor: Use polling model for workflow phase metric (#4557)
 * 8e887e73 fix(executor): Handle sidecar killing in a process-namespace-shared pod (#4575)
 * 991fa674 fix(server): serve artifacts directly from disk to support large artifacts (#4589)
 * 2eeb1fce fix(server): use the correct name when downloading artifacts (#4579)
 * d1a37d5f feat(controller): Retry transient offload errors. Resolves #4464 (#4482)
 * 85cafe6e Update manifests to v2.12.0-rc3
 * 42e3af7a Merge branch 'master' into release-2.12
 * fdafc4ba chore: Updated stress test YAML (#4569)
 * 916b4549 feat(ui): Add Template/Cron workflow filter to workflow page. Closes #4532 (#4543)
 * 6e2b0cf3 docs: Clean-up examples. Fixes #4124 (#4128)
 * 4998b2d6 chore: Remove unused image build and push hooks (#4539)
 * 48af0244 fix: executor/pns containerid prefix fix (#4555)
 * 53195ed5 fix: Respect continueOn for leaf tasks (#4455)
 * e3d59f08 docs: Added CloudSeeds as one of the users for argo (#4553)
 * ad11180e docs: Update cost optimisation  to include information about cleaning up workflows and archiving (#4549)
 * 7e121509 fix(controller): Correct default port logic (#4547)
 * a712e535 fix: Validate metric key names (#4540)
 * c469b053 fix: Missing arg lines caused files not to copy into containers (#4542)
 * 0980ead3 fix(test): fix TestWFDefaultWithWFTAndWf flakiness (#4538)
 * 564e69f3 fix(ui): Do not auto-reload doc.location. Fixes #4530 (#4535)
 * eebcb8b8 docs: Add "Argo Workflows in 5 min" link to README (#4533)
 * 5147f0a1 Merge branch 'master' into release-2.12
 * 176d890c fix(controller): support float for param value (#4490)
 * 4bacbc12 feat(controller): make sso timeout configurable via cm (#4494)
 * 02e1f0e0 fix(server): Add `list sa` and `create secret` to `argo-server` roles. Closes #4526 (#4514)
 * d0082e8f fix: link templates not replacing multiple templates with same name (#4516)
 * 411bde37 feat: adds millisecond-level timestamps to argo and workflow-controller (#4518)
 * 2c54ca3f add bonprix to argo users (#4520)
 * 754a201d build: Reuse IMAGE_OS instead of hard-coded linux (#4519)
 * f509fa55 Update manifests to v2.12.0-rc2
 * c297b2aa Merge branch 'master' into release-2.12
 * 2dab2d15 fix(test):  fix TestWFDefaultWithWFTAndWf flakiness (#4507)
 * 64ae3303 fix(controller): prepend script path to the script template args. Resolves #4481 (#4492)
 * 0931baf5 feat: Redirect to requested URL after SSO login (#4495)
 * 465447c0 fix: Ensure ContainerStatus in PNS is terminated before continuing (#4469)
 * f7287687 fix(ui): Check node children before counting them. (#4498)
 * bfc13c3f fix: Ensure redirect to login when using empty auth token (#4496)
 * d56ce890 feat(cli): add selector and field-selector option to terminate (#4448)
 * e501fcca fix(controller): Refactor the Merge Workflow, WorkflowTemplate and WorkflowDefaults (#4354)
 * 2ee3f5a7 fix(ui): fix the `all` option in the workflow archive list (#4486)
 * 98be709d Update manifests to v2.12.0-rc1
 * a441a97b refactor(server): Use patch instead of update to resume/suspend (#4468)
 * 9ecf0499 fix(controller): When semaphore lock config gets updated, enqueue the waiting workflows (#4421)
 * c31d1722 feat(cli): Support ARGO_HTTP1 for HTTP/1 CLI requests. Fixes #4394 (#4416)
 * b8fb2a8b chore(docs): Fix docgen (#4459)
 * 6c5ab780 feat: Add the --no-utf8 parameter to `argo get` command (#4449)
 * 933a4db0 refactor: Simplify grpcutil.TranslateError (#4465)
 * 24015065 docs: Add DDEV to USERS.md (#4456)
 * d752e2fa feat: Add resume/suspend endpoints for CronWorkflows (#4457)
 * 42d06050 fix: localhost not being resolved. Resolves #4460, #3564 (#4461)
 * 59843e1f fix(controller): Trigger no of workflows based on available lock (#4413)
 * 1be03db7 fix: Return copy of stored templates to ensure they are not modified (#4452)
 * 854883bd fix(controller): Fix throttler. Fixes #1554 and #4081 (#4132)
 * b956bc1a chore(controller): Refactor and tidy up (#4453)
 * 3e451114 fix(docs): timezone DST note on Cronworkflow (#4429)
 * f4f68a74 fix: Resolve inconsistent CronWorkflow persistence (#4440)
 * 76887cfd chore: Update pull request template (#4437)
 * da93545f feat(server): Add WorkflowLogs API. See #4394 (#4450)
 * 3960a0ee fix: Fix validation with Argo Variable in activeDeadlineSeconds (#4451)
 * dedf0521 feat(ui): Visualisation of the suspended CronWorkflows in the list. Fixes #4264 (#4446)
 * 6016ebdd ci: Speed up e2e tests (#4436)
 * 0d13f40d fix(controller): Tolerate int64 parameters. Fixes #4359 (#4401)
 * 2628be91 fix(server): Only try to use auth-mode if enabled. Fixes #4400 (#4412)
 * 7f2ff80f fix: Assume controller is in UTC when calculating NextScheduledRuntime (#4417)
 * 45fbc951 fix(controller): Design-out event errors. Fixes #4364 (#4383)
 * 5a18c674 fix(docs): update link to container spec (#4424)
 * 8006da12 fix: Add x-frame config option (#4420)
 * 46f0ca0f docs: Add Acquia to USERS.md (#4415)
 * 462e55e9 fix: Ensure resourceDuration variables in metrics are always in seconds (#4411)
 * 3aeb1741 fix(executor): artifact chmod should only if err != nil (#4409)
 * 2821e4e8 fix: Use correct template when processing metrics (#4399)
 * b34b91f9 ci: Split e2e tests to make them faster (#4404)
 * e8f82614 fix(validate): Local parameters should be validated locally. Fixes #4326 (#4358)
 * ddd45b6e fix(ui): Reconnect to DAG. Fixes #4301 (#4378)
 * 252c4633 feat(ui): Sign-post examples and the catalog. Fixes #4360 (#4382)
 * 334d1340 feat(server): Enable RBAC for SSO. Closes #3525 (#4198)
 * e409164b fix(ui): correct log viewer only showing first log line (#4389)
 * 28bdb6ff fix(ui): Ignore running workflows in report. Fixes #4387 (#4397)
 * 7ace8f85 fix(controller): Fix estimation bug. Fixes #4386 (#4396)
 * bdac65b0 fix(ui): correct typing errors in workflow-drawer (#4373)
 * cc9c0580 docs: Update events.md (#4391)
 * db5e28ed fix: Use DeletionHandlingMetaNamespaceKeyFunc in cron controller (#4379)
 * 99d33eed fix(server): Download artifacts from UI. Fixes #4338 (#4350)
 * db8a6d0b fix(controller): Enqueue the front workflow if semaphore lock is available (#4380)
 * 933ba834 fix: Fix intstr nil dereference (#4376)
 * 52520aac build: Fix clean-up of vendor (#4363)
 * f206b83b chore: match code comments to the generated documentation (#4375)
 * 220ac736 fix(controller): Only warn if cron job missing. Fixes #4351 (#4352)
 * dbbe95cc Use '[[:blank:]]' instead of ' ' to match spaces/tabs (#4361)
 * b03bd12a fix: Do not allow tasks using 'depends' to begin with a digit (#4218)
 * c237f3f5 docs: Fix a copy-paste typo in the docs (#4357)
 * b76246e2 fix(executor): Increase pod patch backoff. Fixes #4339 (#4340)
 * 7bfe303c docs: Update pvc example with correct resource indentation. Fixes #4330 (#4346)
 * ec671ddc feat(executor): Wait for termination using pod watch for PNS and K8SAPI executors. (#4253)
 * 3156559b fix: ui/package.json & ui/yarn.lock to reduce vulnerabilities (#4342)
 * f5e23f79 refactor: De-couple config (#4307)
 * 36002a26 docs: Fix typos (/priviledged/privileged/) (#4335)
 * 37a2ae06 fix(ui): correct typing errors in events-panel (#4334)
 * 03ef9d61 fix(ui): correct typing errors in workflows-toolbar (#4333)
 * 4de64c61 fix(ui): correct typing errors in cron-workflow-details (#4332)
 * 595aaa55 docs: Update releasing guide (#4284)
 * 939d8c30 feat(controller): add enum support in parameters (fixes #4192) (#4314)
 * e14f4f92 fix(executor): Fix the artifacts option in k8sapi and PNS executor Fixes#4244 (#4279)
 * ea9db436 fix(cli): Return exit code on Argo template lint command (#4292)
 * aa4a435b fix(cli): Fix panic on argo template lint without argument (#4300)
 * 640998e3 docs: Fix incorrect link to static code analysis document (#4329)
 * 04586fdb docs: Add diagram (#4295)
 * f7116fc7 build: Refactor `codegen` to make target dependencies clearer (#4315)
 * 20b3b1ba fix: merge artifact arguments from workflow template. Fixes #4296 (#4316)
 * cae8d1dc docs: Ye olde whitespace fix (#4323)
 * 3c63c3c4 chore(controller): Refactor the CronWorkflow schedule logic with sync.Map (#4320)
 * 1db63e52 docs: Fix typo in docs for artifact repository
 * 40648bcf Update USERS.md (#4322)
 * 07b2ef62 fix(executor): Retrieve containerId from cri-containerd /proc/{pid}/cgroup. Fixes #4302 (#4309)
 * e6b02490 feat(controller): Allow whitespace in variable substitution. Fixes #4286 (#4310)
 * 9119682b fix(build): Some minor Makefile fixes (#4311)
 * db20b4f2 feat(ui): Submit resources without namespace to current namespace. Fixes #4293 (#4298)
 * 240cd792 docs: improve ingress documentation for argo-server (#4306)
 * 26f39b6d fix(ci): add non-root user to Dockerfile (#4305)
 * 1cc68d89 fix(ui): undefined namespace in constructors (#4303)
 * d04f68fd docs: Docker TLS env config (#4299)
 * e54bf815 fix(controller): Patch rather than update cron workflows. (#4294)
 * 9157ef2a fix: TestMutexInDAG failure in master (#4283)
 * 2d6f4e66 fix: WorkflowEventBinding typo in aggregated roles (#4287)
 * c02bb7f0 fix(controller): Fix argo retry with PVCs. Fixes #4275 (#4277)
 * c0423a22 fix(ui): Ignore missing nodes in DAG. Fixes #4232 (#4280)
 * 58144290 fix(controller): Fix cron-workflow re-apply error. (#4278)
 * c605c6d7 fix(controller): Synchronization lock didn't release on DAG call flow Fixes #4046 (#4263)
 * 3cefc147 feat(ui): Add a nudge for users who have not set their security context. Closes #4233  (#4255)
 * e07abe27 test: Tidy up E2E tests (#4276)
 * 8ed799fa docs: daemon example: clarify prefix usage (#4258)
 * a461b076 feat(cli): add `--field-selector` option for `delete` command (#4274)
 * d7fac63e chore(controller): N/W progress fixes (#4269)
 * d377b7c0 docs: Add docs for `securityContext` and `emptyDir`. Closes #2239 (#4251)
 * 4c423453 feat(controller): Track N/M progress. See #2717 (#4194)
 * afbb957a fix: Add WorkflowEventBinding to aggregated roles (#4268)
 * 9bc675eb docs: Minor formatting fixes (#4259)
 * 4dfe7551 docs: correct a typo (#4261)
 * 6ce6bf49 fix(controller): Make the delay before the first workflow reconciliation configurable. Fixes #4107 (#4224)
 * 42b797b8 chore(api): Update swagger.json with Kubernetes v1.17.5 types. Closes #4204 (#4226)
 * 346292b1 feat(controller): Reduce reconcilliation time by exiting earlier. (#4225)
 * 407ac349 fix(ui): Revert bad part of commit (#4248)
 * eaae2309 fix(ui): Fix bugs with DAG view. Fixes #4232 & #4236 (#4241)
 * 04f7488a feat(ui): Adds a report page which shows basic historical workflow metrics. Closes #3557 (#3558)
 * a545a53f fix(controller): Check the correct object for Cronworkflow reapply error log (#4243)
 * ec7a5a40 fix(Makefile): removed deprecated k3d cmds. Fixes #4206 (#4228)
 * 1706a395 fix: Increase deafult number of CronWorkflow workers (#4215)
 * 270e6925 docs: Correct formatting of variables example YAML (#4237)
 * c5ff60b3 docs: Reinstate swagger.md so docs can build. (#4227)
 * 50f23181 feat(cli): Print 'no resource msg' when `argo list` returns zero workflows (#4166)
 * 2143a501 fix(controller): Support workflowDefaults on TTLController for WorkflowTemplateRef Fixes #4188 (#4195)
 * cac10f13 fix(controller): Support int64 for param value. Fixes #4169 (#4202)
 * e910b701 feat: Controller/server runAsNonRoot. Closes #1824 (#4184)
 * f3d1e9f8 chore: Enahnce logging around pod failures (#4220)
 * 4bd5fe10 fix(controller): Apply Workflow default on normal workflow scenario Fixes #4208 (#4213)
 * f9b65c52 chore(build): Update `make codegen` to only run on changes (#4205)
 * 0879067a chore(build): re-add #4127 and steps to verify image pull (#4219)
 * b17b569e fix(controller): reduce withItem/withParams memory usage. Fixes #3907 (#4207)
 * 524049f0 fix: Revert "chore: try out pre-pushing linux/amd64 images and updating ma… Fixes #4216 (#4217)
 * 9c08433f feat(executor): Decompress zip file input artifacts. Fixes #3585 (#4068)
 * 300634af docs: add Mixpanel to argo/USERS.md (#4137)
 * 14650339 fix(executor): Update executor retry config for ExponentialBackoff. (#4196)
 * 2b127625 fix(executor): Remove IsTransientErr check for ExponentialBackoff. Fixes #4144 (#4149)
 * f7e85f04 feat(server): Make Argo Server issue own JWE for SSO. Fixes #4027 & #3873 (#4095)
 * 951d38f8 refactor: Refactor Synchronization code (#4114)
 * 9319c074 fix(ui): handle logging disconnects gracefully (#4150)
 * 88ee7e13 docs: Add map-reduce example. Closes #4165  (#4175)
 * 6265c709 fix: Ensure CronWorkflows are persisted once per operation (#4172)
 * 2a992aee fix: Provide helpful hint when creating workflow with existing name (#4156)
 * de3a90dd refactor: upgrade argo-ui library version (#4178)
 * b7523369 feat(controller): Estimate workflow & node duration. Closes #2717 (#4091)
 * b3db0f5f chore: Update Github issue templates (#4161)
 * c468b34d fix(controller): Correct unstructured API version. Caused by #3719 (#4148)
 * d298706f chore: Deprecate unused RuntimeResolution field (#4155)
 * de81242e fix: Render full tree of onExit nodes in UI (#4109)
 * 109876e6 fix: Changing DeletePropagation to background in TTL Controller and Argo CLI (#4133)
 * c8eed1f2 chore: try out pre-pushing linux/amd64 images and updating manifest later (#4127)
 * 1e10e0cc Documentation (#4122)
 * 3a508a39 docs: Add Onepanel to USERS.md (#4147)
 * b3682d4f fix(cli): add validate args in delete command (#4142)
 * 45fe5173 test: Fix retry (attempt 3). Fixes #4101 (#4146)
 * 373543d1 feat(controller): Sum resources duration for DAGs and steps (#4089)
 * 4829e9ab feat: Add MaxAge to memoization (#4060)
 * af53a4b0 fix(docs): Update k3d command for running argo locally (#4139)
 * 554d6616 fix(ui): Ignore referenced nodes that don't exist in UI. Fixes #4079 (#4099)
 * e8b79921 fix(executor): race condition in docker kill (#4097)
 * 1af88d68 docs: Adding reserved.ai to users list (#4116)
 * 3bb0c2a1 feat(artifacts): Allow HTTP artifact load to set request headers (#4010)
 * 63b41375 fix(cli): Add retry to retry, again. Fixes #4101 (#4118)
 * bd4289ec docs: Added PDOK to USERS.md (#4110)
 * 76cbfa9d fix(ui): Show "waiting" msg while waiting for pod logs. Fixes #3916 (#4119)
 * 196c5eed fix(controller): Process workflows at least once every 20m (#4103)
 * 4825b7ec fix(server): argo-server-role to allow submitting cronworkflows from UI (#4112)
 * 29aba3d1 fix(controller): Treat annotation and conditions changes as significant (#4104)
 * befcbbce feat(ui): Improve error recovery. Close #4087 (#4094)
 * 5cb99a43 fix(ui): No longer redirect to `undefined` namespace. See #4084 (#4115)
 * fafc5a90 fix(cli): Reinstate --gloglevel flag. Fixes #4093 (#4100)
 * c4d91023 fix(cli): Add retry to retry ;). Fixes #4101 (#4105)
 * ff195f2e chore: use build matrix and cache (#4111)
 * 6b350b09 fix(controller): Correct the order merging the fields in WorkflowTemplateRef scenario. Fixes #4044 (#4063)
 * 764b56ba fix(executor): windows output artifacts. Fixes #4082 (#4083)
 * 7c92b3a5 fix(server): Optional timestamp inclusion when retrieving workflow logs. Closes #4033 (#4075)
 * 1bf651b5 feat(controller): Write back workflow to informer to prevent conflict errors. Fixes #3719 (#4025)
 * fdf0b056 feat(controller): Workflow-level `retryStrategy`/resubmit pending pods by default. Closes #3918 (#3965)
 * d7a297c0 feat(controller): Use pod informer for performance. (#4024)
 * d8d0ecbb fix(ui): [Snyk] Fix for 1 vulnerabilities (#4031)
 * ed59408f fix: Improve better handling on Pod deletion scenario  (#4064)
 * e2f4966b fix: make cross-plattform compatible filepaths/keys (#4040)
 * e94b9649 docs: Update release guide (#4054)
 * 4b673a51 chore: Fix merge error (#4073)
 * dcc8c23a docs: Update SSO docs to clarify the user must create K8S secrets for holding the OAuth2 values (#4070)
 * 5461d541 feat(controller): Retry archiving later on error. Fixes #3786 (#3862)
 * 4e085226 fix: Fix unintended inf recursion (#4067)
 * f1083f39 fix: Tolerate malformed workflows when retrying (#4028)
 * a0753951 chore(executor): upgrade `kubectl` to 1.18.8. Closes #3996 (#3999) (#3999)
 * 513ed9fd docs: Add `buildkit` example. Closes #2325 (#4008)
 * fc77beec fix(ui): Tiny modal DAG tweaks. Fixes #4039 (#4043)
 * 3f1792ab docs: Update USERS.md (#4041)
 * 74da0672 docs(Windows): Add more information on artifacts and limitations (#4032)
 * 81c0e427 docs: Add Data4Risk to users list (#4037)
 * ef0ce47e feat(controller): Support different volume GC strategies. Fixes #3095 (#3938)
 * d61a91c4 chore: Add stress testing code. Closes #2899 (#3934)
 * 9f120624 fix: Don't save label filter in local storage (#4022)
 * 0123c9a8 fix(controller): use interpolated values for mutexes and semaphores #3955 (#3957)
 * 25f12441 docs: Update pages to GA. Closes #4000 (#4007)
 * 5be25442 feat(controller): Panic or error on mis-matched resource version (#3949)
 * ae779599 fix: Delete realtime metrics of running Workflows that are deleted (#3993)
 * 4557c713 fix(controller): Script Output didn't set if template has RetryStrategy (#4002)
 * a013609c fix(ui): Do not save undefined namespace. Fixes #4019 (#4021)
 * f8145f83 fix(ui): Correctly show pod events. Fixes #4016 (#4018)
 * 2d722f1f fix(ui): Allow you to view timeline tab. Fixes #4005 (#4006)
 * f36ad2bb fix(ui): Report errors when uploading files. Fixes #3994 (#3995)
 * b5f31919 feat(ui): Introduce modal DAG renderer. Fixes: #3595 (#3967)
 * ad607469 fix(controller): Revert `resubmitPendingPods` mistake. Fixes #4001 (#4004)
 * fd1465c9 fix(controller): Revert parameter value to `*string`. Fixes #3960 (#3963)
 * 13879341 fix: argo-cluster-role pvc get (#3986)
 * f09babdb fix: Default PDB example typo (#3914)
 * f81b006a fix: Step and Task level timeout examples (#3997)
 * 91c49c14 fix: Consider WorkflowTemplate metadata during validation (#3988)
 * 7b1d17a0 fix(server): Remove XSS vulnerability. Fixes #3942 (#3975)
 * 20c518ca fix(controller): End DAG execution on deadline exceeded error. Fixes #3905 (#3921)
 * 74a68d47 feat(ui): Add `startedAt` and `finishedAt` variables to configurable links. Fixes #3898 (#3946)
 * 8e89617b fix: typo of argo server cli (#3984) (#3985)
 * 557531c7 docs: Adding Stillwater Supercomputing (#3987)
 * 1def65b1 fix: Create global scope before workflow-level realtime metrics (#3979)
 * df816958 build: Allow build with older `docker`. Fixes #3977 (#3978)
 * ca55c835 docs: fixed typo (#3972)
 * 29363b6b docs: Correct indentation to display correctly docs in the website (#3969)
 * 402fc0bf fix(executor): set artifact mode recursively. Fixes #3444 (#3832)
 * ff5ed7e4 fix(cli): Allow `argo version` without KUBECONFIG. Fixes #3943 (#3945)
 * d4210ff3 fix(server): Adds missing webhook permissions. Fixes #3927 (#3929)
 * 184884af fix(swagger): Correct item type. Fixes #3926 (#3932)
 * 97764ba9 fix: Fix UI selection issues (#3928)
 * b4329afd fix: Fix children is not defined error (#3950)
 * 3b16a023 chore(doc): fixed java client project link (#3947)
 * 946da359 chore: Fixed  TestTemplateTimeoutDuration testcase (#3940)
 * c977aa27 docs: Adds roadmap. Closes #3835 (#3863)
 * 5a0c515c feat: Step and Task Level Global Timeout (#3686)
 * a8d10340 docs: add Nikkei to user list (#3935)
 * 24c77838 fix: Custom metrics are not recorded for DAG tasks Fixes #3872 (#3886)
 * d4cf0d26 docs: Update workflow-controller-configmap.yaml with SSL options (#3924)
 * 7e6a8910 docs: update docs for upgrading readiness probe to HTTPS. Closes #3859 (#3877)
 * de2185c8 feat(controller): Set retry factor to 2. Closes #3911 (#3919)
 * be91d762 fix: Workflow should fail on Pod failure before container starts Fixes #3879 (#3890)
 * c4c80069 test: Fix TestRetryOmit and TestStopBehavior (#3910)
 * 650869fd feat(server): Display events involved in the workflow. Closes #3673 (#3726)
 * 5b5d2359 fix(controller): Cron re-apply update (#3883)
 * fd3fca80 feat(artifacts): retrieve subpath from unarchived ref artifact. Closes #3061 (#3063)
 * 6a452ccd test: Fix flaky e2e tests (#3909)
 * 6e82bf38 feat(controller): Emit events for malformed cron workflows. See #3881 (#3889)
 * f04bdd6a Update workflow-controller-configmap.yaml (#3901)
 * bb79e3f5 fix(executor): Replace default retry in executor with an increased value retryer (#3891)
 * b681c113 fix(ui): use absolute URL to redirect from autocomplete list. Closes #3903 (#3906)
 * 712c77f5 chore(users): Add Fynd Trak to the list of Users (#3900)
 * d55402db ci: Fix broken Multiplatform builds (#3908)
 * 9681a4e2 fix(ui): Improve error recovery. Fixes #3867 (#3869)
 * b926f8c0 chore: Remove unused imports (#3892)
 * 4c18a06b feat(controller): Always retry when `IsTransientErr` to tolerate transient errors. Fixes #3217 (#3853)
 * 0cf7709f fix(controller): Failure tolerant workflow archiving and offloading. Fixes #3786 and #3837 (#3787)
 * 359ee8db fix: Corrects CRD and Swagger types. Fixes #3578 (#3809)
 * 58ac52b8 chore(ui): correct a typo (#3876)
 * dae0f2df feat(controller): Do not try to create pods we know exists to prevent `exceeded quota` errors. Fixes #3791 (#3851)
 * 9781a1de ci: Create manifest from images again (#3871)
 * c6b51362 test: E2E test refactoring (#3849)
 * 04898fee chore: Added unittest for PVC exceed quota Closes #3561 (#3860)
 * 4e42208c ci: Changed tagging and amend multi-arch manifest. (#3854)
 * c352f69d chore: Reduce the 2x workflow save on Semaphore scenario (#3846)
 * a24bc944 feat(controller): Mutexes. Closes #2677 (#3631)
 * a821c6d4 ci: Fix build by providing clean repo inside Docker (#3848)
 * 99fe11a7 feat: Show next scheduled cron run in UI/CLI (#3847)
 * 6aaceeb9 fix: Treat collapsed nodes as their siblings (#3808)
 * 7f5acd6f docs: Add 23mofang to USERS.md
 * 10cb447a docs: Update example README for duration (#3844)
 * 1678e58c ci: Remove external build dependency (#3831)
 * 743ec536 fix(ui): crash when workflow node has no memoization info (#3839)
 * a2f54da1 fix(docs): Amend link to the Workflow CRD (#3828)
 * ca8ab468 fix: Carry over ownerReferences from resubmitted workflow. Fixes #3818 (#3820)
 * da43086a fix(docs): Add Entrypoint Cron Backfill example  Fixes #3807 (#3814)
 * ed749a55 test: Skip TestStopBehavior and TestRetryOmit (#3822)
 * 9292ae1e ci: static files not being built with Homebrew and dirty binary. Fixes #3769 (#3801)
 * c840adb2 docs: memory base amount denominator documentation
 * 8e1a3db5 feat(ui): add node memoization information to node summary view (#3741)
 * 9de49e2e ci: Change workflow for pushing images. Fixes #2080
 * d235c7d5 fix: Consider all children of TaskGroups in DAGs (#3740)
 * 3540d152 Add SYS_PTRACE to ease the setup of non-root deployments with PNS executor. (#3785)
 * 2f654971 chore: add New Relic to USERS.md (#3810)
 * ce5da590 docs: Add section on CronWorkflow crash recovery (#3804)
 * 0ca83924 feat: Github Workflow multi arch. Fixes #2080 (#3744)
 * bee0e040 docs: Remove confusing namespace (#3772)
 * 7ad6eb84 fix(ui): Remove outdated download links. Fixes #3762 (#3783)
 * 22636782 fix(ui): Correctly load and store namespace. Fixes #3773 and #3775 (#3778)
 * a9577ab9 test: Increase cron test timeout to 7m (#3799)
 * ed90d403 fix(controller): Support exit handler on workflow templates.  Fixes #3737 (#3782)
 * dc75ee81 test: Simplify E2E test tear-down (#3749)
 * 821e40a2 build: Retry downloading Kustomize (#3792)
 * f15a8f77 fix: workflow template ref does not work in other namespace (#3795)
 * ef44a03d fix: Increase the requeue duration on checkForbiddenErrorAndResubmitAllowed (#3794)
 * 0125ab53 fix(server): Trucate creator label at 63 chars. Fixes #3756 (#3758)
 * a38101f4 feat(ui): Sign-post IDE set-up. Closes #3720 (#3723)
 * 21dc23db chore: Format test code (#3777)
 * ee910b55 feat(server): Emit audit events for workflow event binding errors (#3704)
 * e9b29e8c fix: TestWorkflowLevelSemaphore flakiness (#3764)
 * fadd6d82 fix: Fix workflow onExit nodes not being displayed in UI (#3765)
 * df06e901 docs: Correct typo in `--instanceid`
 * 82a671c0 build: Lint e2e test files (#3752)
 * 513675bc fix(executor): Add retry on pods watch to handle timeout. (#3675)
 * e35a86ff feat: Allow parametrizable int fields (#3610)
 * da115f9d fix(controller): Tolerate malformed resources. Fixes #3677 (#3680)
 * 407f9e63 docs: Remove misleading argument in workflow template dag examples. (#3735) (#3736)
 * f8053ae3 feat(operator): Add scope params for step startedAt and finishedAt (#3724)
 * 54c2134f fix: Couldn't Terminate/Stop the ResourceTemplate Workflow (#3679)
 * 12ddc1f6 fix: Argo linting does not respect namespace of declared resource (#3671)
 * acfda260 feat(controller): controller logs to be structured #2308 (#3727)
 * cc2e42a6 fix(controller): Tolerate PDB delete race. Fixes #3706 (#3717)
 * 5eda8b86 fix: Ensure target task's onExit handlers are run (#3716)
 * 811a4419 docs(windows): Add note about artifacts on windows (#3714)
 * 5e5865fb fix: Ingress docs (#3713)
 * eeb3c9d1 fix: Fix bug with 'argo delete --older' (#3699)
 * 6134a565 chore: Introduce convenience methods for intstr. (#3702)
 * 7aa536ed feat: Upgrade Minio v7 with support IRSA (#3700)
 * 4065f265 docs: Correct version. Fixes #3697 (#3701)
 * 71d61281 feat(server): Trigger workflows from webhooks. Closes #2667  (#3488)
 * a5d995dc fix(controller): Adds ALL_POD_CHANGES_SIGNIFICANT (#3689)
 * 9f00cdc9 fix: Fixed workflow queue duration if PVC creation is forbidden (#3691)
 * 2baaf914 chore: Update issue templates (#3681)
 * 41ebbe8e fix: Re-introduce 1 second sleep to reconcile informer (#3684)
 * 6e3c5bef feat(ui): Make UI errors recoverable. Fixes #3666 (#3674)
 * 27fea1bb chore(ui): Add label to 'from' section in Workflow Drawer (#3685)
 * 32d6f752 feat(ui): Add links to wft, cwf, or cwft to workflow list and details. Closes #3621 (#3662)
 * 1c95a985 fix: Fix collapsible nodes rendering (#3669)
 * 87b62bbb build: Use http in dev server (#3670)
 * dbb39368 feat: Add submit options to 'argo cron create' (#3660)
 * 2b6db45b fix(controller): Fix nested maps. Fixes #3653 (#3661)
 * 3f293a4d fix: interface{} values should be expanded with '%v' (#3659)
 * f08ab972 docs: Fix type in default-workflow-specs.md (#3654)
 * a8f4da00 fix(server): Report v1.Status errors. Fixes #3608 (#3652)
 * a3a4ea0a fix: Avoid overriding the Workflow parameter when it is merging with WorkflowTemplate parameter (#3651)
 * 9ce1d824 fix: Enforce metric Help must be the same for each metric Name (#3613)
 * 4eca0481 docs: Update link to examples so works in raw github.com view (#3623)
 * f77780f5 fix(controller): Carry-over labels for re-submitted workflows. Fixes #3622 (#3638)
 * 3f3a4c91 docs: Add comment to config map about SSO auth-mode (#3634)
 * d9090c99 build: Disable TLS for dev mode. Fixes #3617 (#3618)
 * bcc6e1f7 fix: Fixed flaky unit test TestFailSuspendedAndPendingNodesAfterDeadline (#3640)
 * 8f70d224 fix: Don't panic on invalid template creation (#3643)
 * 5b0210dc fix: Simplify the WorkflowTemplateRef field validation to support all fields in WorkflowSpec except `Templates` (#3632)
 * 2375878a fix: Fix 'malformed request: field selector' error (#3636)
 * 87ea54c9 docs: Add documentation for configuring OSS artifact storage (#3639)
 * 861afd36 docs: Correct indentation for codeblocks within bullet-points for "workflow-templates" (#3627)
 * 0f37e81a fix: DAG level Output Artifacts on K8S and Kubelet executor (#3624)
 * a89261bf build(cli)!: Zip binaries binaries. Closes #3576 (#3614)
 * 7f844473 fix(controller): Panic when outputs in a cache entry are nil (#3615)
 * 86f03a3f fix(controller): Treat TooManyError same as Forbidden (i.e. try again). Fixes #3606 (#3607)
 * 2e299df3 build: Increase timeout (#3616)
 * e0a4f13d fix(server): Re-establish watch on v1.Status errors. Fixes #3608 (#3609)
 * cdbb5711 docs: Memoization Documentation (#3598)
 * 7abead2a docs: fix typo - replace "workfow" with "workflow" (#3612)
 * f7be20c1 fix: Fix panic and provide better error message on watch endpoint (#3605)
 * 491f4f74 fix: Argo Workflows does not honour global timeout if step/pod is not able to schedule (#3581)
 * 5d8f85d5 feat(ui): Enhanced workflow submission. Closes #3498 (#3580)
 * a4a26414 build: Initialize npm before installing swagger-markdown (#3602)
 * ad3441dc feat: Add 'argo node set' command (#3277)
 * a43bf129 docs: Migrate to homebrew-core (#3567) (#3568)
 * 17b46bdb fix(controller): Fix bug in util/RecoverWorkflowNameFromSelectorString. Add error handling (#3596)
 * c968877c docs: Document ingress set-up. Closes #3080 (#3592)
 * 8b6e43f6 fix(ui): Fix multiple UI issues (#3573)
 * cdc935ae feat(cli): Support deleting resubmitted workflows (#3554)
 * 1b757ea9 feat(ui): Change default language for Resource Editor to YAML and store preference in localStorage. Fixes #3543 (#3560)
 * c583bc04 fix(server): Ignore not-JWT server tokens. Fixes #3562 (#3579)
 * 5afbc131 fix(controller): Do not panic on nil output value. Fixes #3505 (#3509)
 * c409624b docs: Synchronization documentation (#3537)
 * 0bca0769 docs: Workflow of workflows pattern (#3536)
 * 827106de fix: Skip TestStorageQuotaLimit (#3566)
 * 13b1d3c1 feat(controller): Step level memoization. Closes #944 (#3356)
 * 96e520eb fix: Exceeding quota with volumeClaimTemplates (#3490)
 * 144c9b65 fix(ui): cannot push to nil when filtering by label (#3555)
 * 7e4a7808 feat: Collapse children in UI Workflow viewer (#3526)
 * 7536982a fix: Fix flakey TestRetryOmitted (#3552)
 * 05d573d7 docs: Change formatting to put content into code block (#3553)
 * dcee3484 fix: Fix links in fields doc (#3539)
 * fb67c1be Fix issue #3546 (#3547)
 * d07a0e74 ci: Make builds marginally faster. Fixes #3515 (#3519)
 * 4cb6aa04 chore: Enable no-response bot (#3510)
 * 31afa92a fix(artifacts): support optional input artifacts, Fixes #3491 (#3512)
 * 977beb46 fix: Fix when retrying Workflows with Omitted nodes (#3528)
 * ab4ef5c5 fix: Panic on CLI Watch command (#3532)
 * b901b279 fix(controller): Backoff exponent is off by one. Fixes #3513 (#3514)
 * 49ef5c0f fix: String interpreted as boolean in labels (#3518)
 * 19e700a3 fix(cli): Check mutual exclusivity for argo CLI flags (#3493)
 * 7d45ff7f fix: Panic on releaseAllWorkflowLocks if Object is not Unstructured type (#3504)
 * 1b68a5a1 fix: ui/package.json & ui/yarn.lock to reduce vulnerabilities (#3501)
 * 7f262fd8 fix(cli)!: Enable CLI to work without kube config. Closes #3383, #2793 (#3385)
 * 2976e7ac build: Clear cmd docs before generating them (#3499)
 * 27528ba3 feat: Support completions for more resources (#3494)
 * 5bd2ad7a fix: Merge WorkflowTemplateRef with defaults workflow spec (#3480)
 * e244337b chore: Added examples for exit handler for step and dag level (#3495)
 * bcb32547 build: Use `git rev-parse` to accomodate older gits (#3497)
 * 3eb6e2f9 docs: Add link to GitHub Actions in the badge (#3492)
 * 69179e72 fix: link to server auth mode docs, adds Tulip as official user (#3486)
 * 7a8e2b34 docs: Add comments to NodePhase definition. Closes #1117. (#3467)
 * 24d1e529 build: Simplify builds (#3478)
 * acf56f9f feat(server): Label workflows with creator. Closes #2437 (#3440)
 * 3b8ac065 fix: Pass resolved arguments to onExit handler (#3477)
 * 58097a9e docs: Add controller-level metrics (#3464)
 * f6f1844b feat: Attempt to resolve nested tags (#3339)
 * 48e15d6f feat(cli): List only resubmitted workflows option (#3357)
 * 25e9c0cd docs, quick-start. Use http, not https for link (#3476)
 * 7a2d7642 fix: Metric emission with retryStrategy (#3470)
 * f5876e04 test(controller): Ensure resubmitted workflows have correct labels (#3473)
 * aa92ec03 fix(controller): Correct fail workflow when pod is deleted with --force. Fixes #3097 (#3469)
 * a1945d63 fix(controller): Respect the volumes of a workflowTemplateRef. Fixes … (#3451)
 * 847ba530 test(controller): Add memoization tests. See #3214 (#3455) (#3466)
 * f5183aed docs: Fix CLI docs (#3465)
 * 1e42813a test(controller): Add memoization tests. See #3214 (#3455)
 * abe768c4 feat(cli): Allow to view previously terminated container logs (#3423)
 * 7581025f fix: Allow ints for sequence start/end/count. Fixes #3420 (#3425)
 * b82f900a Fixed typos (#3456)
 * 23760119 feat: Workflow Semaphore Support (#3141)
 * 81cba832 feat: Support WorkflowMetadata in WorkflowTemplate and ClusterWorkflowTemplate (#3364)
 * 568c032b chore: update aws-sdk-go version (#3376)
 * bd27d9f3 chore: Upgrade node-sass (#3450)
 * b1e601e5 docs: typo in argo stop --help (#3439)
 * 308c7083 fix(controller): Prevent panic on nil node. Fixes #3436 (#3437)
 * 8ab06f53 feat(controller): Add log message count as metrics. (#3362)
 * 5d0c436d chore: Fix GitHub Actions Docker Image build  (#3442)
 * e54b4ab5 docs: Add Sohu as official Argo user (#3430)
 * ee6c8760 fix: Ensure task dependencies run after onExit handler is fulfilled (#3435)
 * 6dc04b39 chore: Use GitHub Actions to build Docker Images to allow publishing Windows Images (#3291)
 * 05b3590b feat(controller): Add support for Docker workflow executor for Windows nodes (#3301)
 * 676868f3 fix(docs): Update kubectl proxy URL (#3433)
 * 3507c3e6 docs: Make https://argoproj.github.io/argo/  (#3369)
 * 733e95f7 fix: Add struct-wide RWMutext to metrics (#3421)
 * 0463f241 fix: Use a unique queue to visit nodes (#3418)
 * eddcac63 fix: Script steps fail with exceededQuota (#3407)
 * c631a545 feat(ui): Add Swagger UI (#3358)
 * 910f636d fix: No panic on watch. Fixes #3411 (#3426)
 * b4da1bcc fix(sso): Remove unused `groups` claim. Fixes #3411 (#3427)
 * 330d4a0a fix: panic on wait command if event is null (#3424)
 * 7c439424 docs: Include timezone name reference (#3414)
 * 03cbb8cf fix(ui): Render DAG with exit node (#3408)
 * 3d50f985 feat: Expose certain queue metrics (#3371)
 * c7b35e05 fix: Ensure non-leaf DAG tasks have their onExit handler's run (#3403)
 * 70111600 fix: Fix concurrency issues with metrics (#3401)
 * d307f96f docs: Update config example to include useSDKCreds (#3398)
 * 637d50bc chore: maybe -> may be (#3392)
 * e70a8863 chore: Added CWFT WorkflowTemplateRef example (#3386)
 * bc4faf5f fix: Fix bug parsing parmeters (#3372)
 * 4934ad22 fix: Running pods are garaged in PodGC onSuccess
 * 0541cfda chore(ui): Remove unused interfaces for artifacts (#3377)
 * 20382cab docs: Fix incorrect example of global parameter (#3375)
 * 1db93c06 perf: Optimize time-based filtering on large number of workflows (#3340)
 * 2ab9495f fix: Don't double-count metric events (#3350)
 * 7bd3e720 fix(ui): Confirmation of workflow actions (#3370)
 * 488790b2 Wellcome is using Argo in our Data Labs division (#3365)
 * 63e71192 chore: Remove unused code (#3367)
 * a64ceb03 build: Enable Stale Bot (#3363)
 * e4b08abb fix(server): Remove `context cancelled` error. Fixes #3073 (#3359)
 * 74ba5162 fix: Fix UI bug in DAGs (#3368)
 * 5e60decf feat(crds)!: Adds CRD generation and enhanced UI resource editor. Closes #859 (#3075)
 * c2347f35 chore: Simplify deps by removing YAML (#3353)
 * 1323f9f4 test: Add e2e tags (#3354)
 * 731a1b4a fix(controller): Allow events to be sent to non-argo namespace. Fixes #3342 (#3345)
 * 916e0db2 Adding InVision to Users (#3352)
 * 6caf10fa fix: Ensure child pods respect maxDuration (#3280)
 * 8f4945f5 docs: Field fix (ParallelSteps -> WorkflowStep) (#3338)
 * 2b4b7340 fix: Remove broken SSO from quick-starts (#3327)
 * 26570fd5 fix(controller)!: Support nested items. Fixes #3288 (#3290)
 * c3d85716 chore: Avoid variable name collision with imported package name (#3335)
 * ca822af0 build: Fix path to go-to-protobuf binary (#3308)
 * 769a964f feat(controller): Label workflows with their source workflow template (#3328)
 * 0785be24 fix(ui): runtime error from null savedOptions props (#3330)
 * 200be0e1 feat: Save pagination limit and selected phases/labels to local storage (#3322)
 * b5ed90fe feat: Allow to change priority when resubmitting workflows (#3293)
 * 60c86c84 fix(ui): Compiler error from workflows toolbar (#3317)
 * 3fe6ecc4 docs: Document access token creation and usage (#3316)
 * ab3c081e docs: Rename Ant Financial to Ant Group (#3304)
 * baad42ea feat(ui): Add ability to select multiple workflows from list and perform actions on them. Fixes #3185 (#3234)
 * b6118939 fix(controller): Fix panic logging. (#3315)
 * 633ea71e build: Pin `goimports` to working version (#3311)
 * 436c1259 ci: Remove CircleCI (#3302)
 * 8e340229 build: Remove generated Swagger files. (#3297)
 * e021d7c5 Clean up unused constants (#3298)
 * 48d86f03 build: Upload E2E diagnostics after failure (#3294)
 * 8b12f433 feat(cli): Add --logs to `argo [submit|resubmit|retry]. Closes #3183 (#3279)
 * 07b450e8 fix: Reapply Update if CronWorkflow resource changed (#3272)
 * 8af01491 docs: ArchiveLabelSelector document (#3284)
 * 38c908a2 docs: Add example for handling large output resutls (#3276)
 * d44d264c Fixes validation of overridden ref template parameters. (#3286)
 * 62e54fb6 fix: Fix delete --complete (#3278)
 * a3c379bb docs: Updated WorkflowTemplateRef  on WFT and CWFT (#3137)
 * 824de95b fix(git): Fixes Git when using auth or fetch. Fixes #2343 (#3248)
 * 018fcc23 Update releasing.md (#3283)
 * acee573b docs: Update CI Badges (#3282)
 * 9eb182c0 build: Allow to change k8s namespace for installation (#3281)
 * 2bcfafb5 fix: Add {{workflow.status}} to workflow-metrics (#3271)
 * e6aab605 fix(jqFilter)!: remove extra quotes around output parameter value (#3251)
 * f4580163 fix(ui): Allow render of templates without entrypoint. Fixes #2891 (#3274)
 * f30c05c7 build: Add warning to ensure 'v' is present on release versions (#3273)
 * d1cb1992 fixed archiveLabelSelector nil (#3270)
 * c7e4c180 fix(ui): Update workflow drawer with new duration format (#3256)
 * f2381a54 fix(controller): More structured logging. Fixes #3260 (#3262)
 * acba084a fix: Avoid unnecessary nil check for annotations of resubmitted workflow (#3268)
 * 55e13705 feat: Append previous workflow name as label to resubmitted workflow (#3261)
 * 2dae7244 feat: Add mode to require Workflows to use workflowTemplateRef (#3149)
 * 56694abe Fixed onexit on workflowtempalteRef (#3263)
 * 54dd72c2 update mysql yaml port (#3258)
 * fb502632 feat: Configure ArchiveLabelSelector for Workflow Archive (#3249)
 * 5467c899 fix(controller): set pod finish timestamp when it is deleted (#3230)
 * 04bc5492 build: Disable Circle CI and Sonar (#3253)
 * 23ca07a7 chore: Covered steps.<STEPNAME>.outputs.parameters in variables document (#3245)
 * 4bd33c6c chore(cli): Add examples of @latest alias for relevant commands. Fixes #3225 (#3242)
 * 17108df1 fix: Ensure subscription is closed in log viewer (#3247)
 * 495dc89b docs: Correct available fields in {{workflow.failures}} (#3238)
 * 4db1c4c8 fix: Support the TTLStrategy for WorkflowTemplateRef (#3239)
 * 47f50693 feat(logging): Made more controller err/warn logging structured (#3240)
 * c25e2880 build: Migrate to Github Actions (#3233)
 * ef159f9a feat: Tick CLI Workflow watch even if there are no new events (#3219)
 * ff1627b7 fix(events): Adds config flag. Reduce number of dupe events emitted. (#3205)
 * eae8f681 feat: Validate CronWorkflows before execution (#3223)
 * 4470a8a2 fix(ui/server): Fix broken label filter functionality on UI due to bug on server. Fix #3226 (#3228)
 * e5e6456b feat(cli): Add --latest flag for argo get command as per #3128 (#3179)
 * 34608594 fix(ui): Correctly update workflow list when workflow are modified/deleted (#3220)
 * a7d8546c feat(controller): Improve throughput of many workflows. Fixes #2908 (#2921)
 * a37d0a72 build: Change "DB=..." to "PROFILE=..." (#3216)
 * 15885d3e feat(sso): Allow reading SSO clientID from a secret. (#3207)
 * 723e9d5f fix: Ensrue image name is present in containers (#3215)
 * 0ee5e112 feat: Only process significant pod changes (#3181)
 * c89a81f3 feat: Add '--schedule' flag to 'argo cron create' (#3199)
 * 591f649a refactor: Refactor assesDAGPhase logic (#3035)
 * 285eda6b chore: Remove unused pod in addArchiveLocation() (#3200)
 * 8e1d56cb feat(controller): Add default name for artifact repository ref. (#3060)
 * f1cdba18 feat(controller): Add `--qps` and `--burst` flags to controller (#3180)
 * b86949f0 fix: Ensure stable desc/hash for metrics (#3196)
 * e26d2f08 docs: Update Getting Started (#3099)
 * 47bfea5d docs: Add Graviti as official Argo user (#3187)
 * 04c77f49 fix(server): Allow field selection for workflow-event endpoint (fixes #3163) (#3165)
 * 0c38e66e chore: Update Community Meeting link and specify Go@v1.13 (#3178)
 * 81846d41 build: Only check Dex in hosts file when SSO is enabled (#3177)
 * a130d488 feat(ui): Add drawer with more details for each workflow in Workflow List (#3151)
 * fa84e203 fix: Do not use alphabetical order if index exists (#3174)
 * 138af597 fix(cli): Sort expanded nodes by index. Closes #3145 (#3146)
 * a9ec4d08 docs: Fix api swagger file path in docs (#3167)
 * c42e4d3a feat(metrics): Add node-level resources duration as Argo variable for metrics. Closes #3110 (#3161)
 * e36fe66e docs: Add instructions on using Minikube as an alternative to K3D (#3162)
 * edfa5b93 feat(metrics): Report controller error counters via metrics. Closes #3034 (#3144)
 * 8831e4ea feat(argo-server): Add support for SSO. See #1813 (#2745)
 * b62184c2 feat(cli): More `argo list` and `argo delete` options (#3117)
 * c6565d7c fix(controller): Maybe bug with nil woc.wfSpec. Fixes #3121 (#3160)
 * 06ca71d7 build: Fix path to staticfiles and goreman binaries (#3159)
 * cad84cab chore: Remove unused nodeType in initializeNodeOrMarkError() (#3153)
 * be425513 chore: Master needs lint (#3152)
 * 70b56f25 enhancement(ui): Add workflow labels column to workflow list. Fixes #2782 (#3143)
 * 3318c115 chore: Move default metrics server port/path to consts (#3135)
 * a0062adf feat(ui): Add Alibaba Cloud OSS related models in UI (#3140)
 * 1469991c fix: Update container delete grace period to match Kubernetes default (#3064)
 * df725bbd fix(ui): Input artifacts labelled in UI. Fixes #3098 (#3131)
 * c0d59cc2 feat: Persist DAG rendering options in local storage (#3126)
 * 8715050b fix(ui): Fix label error (#3130)
 * 1814ea2e fix(item): Support ItemValue.Type == List. Fixes #2660 (#3129)
 * 12b72546 fix: Panic on invalid WorkflowTemplateRef (#3127)
 * 09092147 fix(ui): Display error message instead of DAG when DAG cannot be rendered. Fixes #3091 (#3125)
 * 2d9a74de docs: Document cost optimizations. Fixes #1139 (#2972)
 * 69c9e5f0 fix: Remove unnecessary panic (#3123)
 * 2f3aca89 add AppDirect to the list of users (#3124)
 * 257355e4 feat: Add 'submit --from' to CronWorkflow and WorkflowTemplate in UI. Closes #3112 (#3116)
 * 6e5dd2e1 Add Alibaba OSS to the list of supported artifacts (#3108)
 * 1967b45b support sso (#3079)
 * 9229165f feat(ui): Add cost optimisation nudges. (#3089)
 * e88124db fix(controller): Do not panic of woc.orig in not hydrated. Fixes #3118 (#3119)
 * 132b947a fix: Differentiate between Fulfilled and Completed (#3083)
 * a93968ff docs: Document how to backfill a cron workflow (#3094)
 * 4de99746 feat: Added Label selector and Field selector in Argo list  (#3088)
 * 6229353b chore: goimports (#3107)
 * 8491e00f docs: Add link to USERS.md in PR template (#3086)
 * bb2ce9f7 fix: Graceful error handling of malformatted log lines in watch (#3071)
 * 4fd27c31 build(swagger): Fix Swagger build problems (#3084)
 * e4e0dfb6 test: fix TestContinueOnFailDag (#3101)
 * fa69c1bb feat: Add CronWorkflowConditions to report errors (#3055)
 * 50ad3cec adds millisecond-level timestamps to argoexec (#2950)
 * 6464bd19 fix(controller): Implement offloading for workflow updates that are re-applied. Fixes #2856 (#2941)
 * 6c369e61 chore: Rename files that include 'top-level' terminology (#3076)
 * bd40b80b docs: Document work avoidance. (#3066)
 * 6df0b2d3 feat: Support Top level workflow template reference  (#2912)
 * 0709ad28 feat: Enhanced filters for argo {watch,get,submit} (#2450)
 * 784c1385 build: Use goreman for starting locally. (#3074)
 * 5b5bae9a docs: Add Isbank to users.md (#3068)
 * 2b038ed2 feat: Enhanced depends logic (#2673)
 * 4c3387b2 fix: Linters should error if nothing was validated (#3011)
 * 51dd05b5 fix(artifacts): Explicit archive strategy. Fixes #2140 (#3052)
 * ada2209e Revert "fix(artifacts): Allow tar check to be ignored. Fixes #2140 (#3024)" (#3047)
 * b7ff9f09 chore: Add ability to configure maximum DB connection lifetime (#3032)
 * 38a995b7 fix(executor): Properly handle empty resource results, like for a missing get (#3037)
 * a1ac8bcf fix(artifacts): Allow tar check to be ignored. Fixes #2140 (#3024)
 * f12d79ca fix(controller)!: Correctly format workflow.creationTimepstamp as RFC3339. Fixes #2974 (#3023)
 * d10e949a fix: Consider metric nodes that were created and completed in the same operation (#3033)
 * 202d4ab3 fix(executor): Optional input artifacts. Fixes #2990 (#3019)
 * f17e946c fix(executor): Save script results before artifacts in case of error. Fixes #1472 (#3025)
 * 3d216ae6 fix: Consider missing optional input/output artifacts with same name (#3029)
 * 3717dd63 fix: Improve robustness of releases. Fixes #3004 (#3009)
 * 9f86a4e9 feat(ui): Enable CSP, HSTS, X-Frame-Options. Fixes #2760, #1376, #2761 (#2971)
 * cb71d585 refactor(metrics)!: Refactor Metric interface (#2979)
 * c0ee1eb2 docs: Add Ravelin as a user of Argo (#3020)
 * 052e6c51 Fix isTarball to handle the small gzipped file (#3014)
 * cdcba3c4 fix(ui): Displays command args correctl pre-formatted. (#3018)
 * b5160988 build: Mockery v1.1.1 (#3015)
 * a04d8f28 docs: Add StatefulSet and Service doc (#3008)
 * 8412526c docs: Fix Deprecated formatting (#3010)
 * cc0fe433 fix(events): Correct event API Version. Fixes #2994 (#2999)
 * d5d6f750 feat(controller)!: Updates the resource duration calculation. Fixes #2934 (#2937)
 * fa3801a5 feat(ui): Render 2000+ nodes DAG acceptably. (#2959)
 * f952df51 fix(executor/pns): remove sleep before sigkill (#2995)
 * 2a9ee21f feat(ui): Add Suspend and Resume to CronWorkflows in UI (#2982)
 * eefe120f test: Upgrade to argosay:v2 (#3001)
 * 47472f73 chore: Update Mockery (#3000)
 * 46b11e1e docs: Use keyFormat instead of keyPrefix in docs (#2997)
 * 60d5fdc7 fix: Begin counting maxDuration from first child start (#2976)
 * 76aca493 build: Fix Docker build. Fixes #2983 (#2984)
 * d8cb66e7 feat: Add Argo variable {{retries}} to track retry attempt (#2911)
 * 14b7a459 docs: Fix typo with WorkflowTemplates link (#2977)
 * 3c442232 fix: Remove duplicate node event. Fixes #2961 (#2964)
 * d8ab13f2 fix: Consider Shutdown when assesing DAG Phase for incomplete Retry node (#2966)
 * 8a511e10 fix: Nodes with pods deleted out-of-band should be Errored, not Failed (#2855)
 * ca4e08f7 build: Build dev images from cache (#2968)
 * 5f01c4a5 Upgraded to Node 14.0.0 (#2816)
 * 849d876c Fixes error with unknown flag: --show-all (#2960)
 * 93bf6609 fix: Don't update backoff message to save operations (#2951)
 * 3413a5df fix(cli): Remove info logging from watches. Fixes #2955 (#2958)
 * fe9f9019 fix: Display Workflow finish time in UI (#2896)
 * f281199a docs: Update README with new features (#2807)
 * c8bd0bb8 fix(ui): Change default pagination to all and sort workflows (#2943)
 * e3ed686e fix(cli): Re-establish watch on EOF (#2944)
 * 67355372 fix(swagger)!: Fixes invalid K8S definitions in `swagger.json`. Fixes #2888 (#2907)
 * 023f2338 fix(argo-server)!: Implement missing instanceID code. Fixes #2780 (#2786)
 * 7b0739e0 Fix typo (#2939)
 * 20d69c75 Detect ctrl key when a link is clicked (#2935)
 * f32cec31 fix default null value for timestamp column - MySQL 5.7 (#2933)
 * 9773cfeb docs: Add docs/scaling.md (#2918)
 * 99858ea5 feat(controller): Remove the excessive logging of node data (#2925)
 * 03ad694c feat(cli): Refactor `argo list --chunk-size` and add `argo archive list --chunk-size`. Fixes #2820 (#2854)
 * 1c45d5ea test: Use argoproj/argosay:v1 (#2917)
 * f311a5a7 build: Fix Darwin build (#2920)
 * a06cb5e0 fix: remove doubled entry in server cluster role deployment (#2904)
 * c71116dd feat: Windows Container Support. Fixes #1507 and #1383 (#2747)
 * 3afa7b2f fix(ui): Use LogsViewer for container logs (#2825)
 * 9ecd5226 docs: Document node field selector. Closes #2860 (#2882)
 * 7d8818ca fix(controller): Workflow stop and resume by node didn't properly support offloaded nodes. Fixes #2543 (#2548)
 * e013f29d ci: Remove context to stop unauthozied errors on test jobs (#2910)
 * db52e7ba fix(controller): Add mutex to nameEntryIDMap in cron controller. Fix #2638 (#2851)
 * 9a33aa2d docs(users): Adding Habx to the users list (#2781)
 * 9e4ac9b3 feat(cli): Tolerate deleted workflow when running `argo delete`. Fixes #2821 (#2877)
 * a0035dd5 fix: ConfigMap syntax (#2889)
 * c05c3859 ci: Build less and therefore faster (#2839)
 * 56143eb1 feat(ui): Add pagination to workflow list. Fixes #1080 and #976 (#2863)
 * e0ad7de9 test: Fixes various tests (#2874)
 * e378ca47 fix: Cannot create WorkflowTemplate with un-supplied inputs (#2869)
 * c3e30c50 fix(swagger): Generate correct Swagger for inline objects. Fixes #2835 (#2837)
 * c0143d34 feat: Add metric retention policy (#2836)
 * f03cda61 Update getting-started.md (#2872)
 * d66224e1 fix: Don't error when deleting already-deleted WFs (#2866)
 * e84acb50 chore: Display wf.Status.Conditions in CLI (#2858)
 * 3c7f3a07 docs: Fix typo ".yam" -> ".yaml" (#2862)
 * d7f8e0c4 fix(CLI): Re-establish workflow watch on disconnect. Fixes #2796 (#2830)
 * 31358d6e feat(CLI): Add -v and --verbose to Argo CLI (#2814)
 * 1d30f708 ci: Don't configure Sonar on CI for release branches (#2811)
 * d9c54075 docs: Fix exit code example and docs (#2853)
 * 90743353 feat: Expose workflow.serviceAccountName as global variable (#2838)
 * f07f7bf6 note that tar.gz'ing output artifacts is optional (#2797)
 * 3fd3fc6c docs: Document how to label creator (#2827)
 * b956ec65 fix: Add Step node outputs to global scope (#2826)
 * bac339af chore: Configure webpack dev server to proxy using HTTPS (#2812)
 * cc136f9c test: Skip TestStopBehavior. See #2833 (#2834)
 * 52ff43b5 fix: Artifact panic on unknown artifact. Fixes #2824 (#2829)
 * 554fd06c fix: Enforce metric naming validation (#2819)
 * dd223669 docs: Add Microba as official Argo user (#2822)
 * 8151f0c4 docs: Update tls.md (#2813)
 * 0dbd78ff feat: Add TLS support. Closes #2764 (#2766)
 * 510e11b6 fix: Allow empty strings in valueFrom.default (#2805)
 * d7f41ac8 fix: Print correct version in logs. (#2806)
 * e9c21120 chore: Add GCS native example for output artifact (#2789)
 * e0f2697e fix(controller): Include global params when using withParam (#2757)
 * 3441b11a docs: Fix typo in CronWorkflow doc (#2804)
 * a2d2b848 docs: Add example of recursive for loop (#2801)
 * 29d39e29 docs: Update the contributing docs  (#2791)
 * 1ea286eb fix: ClusterWorkflowTemplate RBAC for  argo server (#2753)
 * 1f14f2a5 feat(archive): Implement data retention. Closes #2273 (#2312)
 * d0cc7764 feat: Display argo-server version in `argo version` and in UI. (#2740)
 * 8de57281 feat(controller): adds Kubernetes node name to workflow node detail in web UI and CLI output. Implements #2540 (#2732)
 * 52fa5fde MySQL config fix (#2681)
 * 43d9eebb fix: Rename Submittable API endpoint to `submit` (#2778)
 * 69333a87 Fix template scope tests (#2779)
 * bb1abf7f chore: Add CODEOWNERS file (#2776)
 * 905e0b99 fix: Naming error in Makefile (#2774)
 * 7cb2fd17 fix: allow non path output params (#2680)
 * af9f61ea ci: Recurl (#2769)
 * ef08e642 build: Retry curl 3x (#2768)
 * dedec906 test: Get tests running on release branches (#2767)
 * 1c8318eb fix: Add compatiblity mode to templateReference (#2765)
 * 7975952b fix: Consider expanded tasks in getTaskFromNode (#2756)
 * bc421380 fix: Fix template resolution in UI (#2754)
 * 391c0f78 Make phase and templateRef available for unsuspend and retry selectors (#2723)
 * a6fa3f71 fix: Improve cookie security. Fixes #2759 (#2763)
 * 57f0183c Fix typo on the documentation. It causes error unmarshaling JSON: while (#2730)
 * c6ef1ff1 feat(manifests): add name on workflow-controller-metrics service port (#2744)
 * af5cd1ae docs: Update OWNERS (#2750)
 * 06c4bd60 fix: Make ClusterWorkflowTemplate optional for namespaced Installation (#2670)
 * 25c62463 docs: Update README (#2752)
 * 908e1685 docs: Update README.md (#2751)
 * 4ea43e2d fix: Children of onExit nodes are also onExit nodes (#2722)
 * 3f1b6667 feat: Add Kustomize as supported install option. Closes #2715 (#2724)
 * 691459ed fix: Error pending nodes w/o Pods unless resubmitPendingPods is set (#2721)
 * 874d8776 test: Longer timeout for deletion (#2737)
 * 3c8149fa Fix typo (#2741)
 * 98f60e79 feat: Added Workflow SubmitFromResource API (#2544)
 * 6253997a fix: Reset all conditions when resubmitting (#2702)
 * e7c67de3 fix: Maybe fix watch. Fixes #2678 (#2719)
 * cef6dfb6 fix: Print correct version string. (#2713)
 * e9589d28 feat: Increase pod workers and workflow workers both to 32 by default. (#2705)
 * 3a1990e0 test: Fix Goroutine leak that was making controller unit tests slow. (#2701)
 * 9894c29f ci: Fix Sonar analysis on master. (#2709)
 * 54f5be36 style: Camelcase "clusterScope" (#2720)
 * db6d1416 fix: Flakey TestNestedClusterWorkflowTemplate testcase failure (#2613)
 * b4fd4475 feat(ui): Add a YAML panel to view the workflow manifest. (#2700)
 * 65d413e5 build(ui): Fix compression of UI package. (#2704)
 * 4129528d fix: Don't use docker cache when building release images (#2707)
 * 8d0956c9 test: Increase runCli timeout to 1m (#2703)
 * 9d93e971 Update getting-started.md (#2697)
 * ee644a35 docs: Fix CONTRIBUTING.md and running-locally.md. Fixes #2682 (#2699)
 * 2737c0ab feat: Allow to pass optional flags to resource template (#1779)
 * c1a2fc7c Update running-locally.md - fixing incorrect protoc install (#2689)
 * a1226c46 fix: Enhanced WorkflowTemplate and ClusterWorkflowTemplate validation to support Global Variables   (#2644)
 * c21cc2f3 fix a typo (#2669)
 * 9430a513 fix: Namespace-related validation in UI (#2686)
 * f3eeca6e feat: Add exit code as output variable (#2111)
 * 9f95e23a fix: Report metric emission errors via Conditions (#2676)
 * c67f5ff5 fix: Leaf task with continueOn should not fail DAG (#2668)
 * 3c20d4c0 ci: Migrate to use Sonar instead of CodeCov for analysis (#2666)
 * 9c6351fa feat: Allow step restart on workflow retry. Closes #2334 (#2431)
 * cf277eb5 docs: Updates docs for CII. See #2641 (#2643)
 * e2d0aa23 fix: Consider offloaded and compressed node in retry and resume (#2645)
 * a25c6a20 build: Fix codegen for releases (#2662)
 * 4a3ca930 fix: Correctly emit events. Fixes #2626 (#2629)
 * 4a7d4bdb test: Fix flakey DeleteCompleted test (#2659)
 * 41f91e18 fix: Add DAG as default in UI filter and reorder (#2661)
 * f138ada6 fix: DAG should not fail if its tasks have continueOn (#2656)
 * e5cbdf6a ci: Only run CI jobs if needed (#2655)
 * 4c452d5f fix: Don't attempt to resolve artifacts if task is going to be skipped (#2657)
 * 2caf570a chore: Add newline to fields.md (#2654)
 * 2cb596da Storage region should be specified (#2538)
 * 271e4551 chore: Fix-up Yarn deps (#2649)
 * 4c1b0777 fix: Sort log entries. (#2647)
 * 268fc461  docs: Added doc generator code (#2632)
 * d58b7fc3 fix: Add input paremeters to metric scope (#2646)
 * cc3af0b8 fix: Validating Item Param in Steps Template (#2608)
 * 6c685c5b fix: allow onExit to run if wf exceeds activeDeadlineSeconds. Fixes #2603 (#2605)
 * ffc43ce9 feat: Added Client validation on Workflow/WFT/CronWF/CWFT (#2612)
 * 24655cd9 feat(UI): Move Workflow parameters to top of submit (#2640)
 * 0a3b159a Use error equals (#2636)
 * 8c29e05c ci: Fix codegen job (#2648)
 * a78ecb7f docs(users): Add CoreWeave and ConciergeRender (#2641)
 * 14be4670 fix: Fix logs part 2 (#2639)
 * 4da6f4f3 feat: Add 'outputs.result' to Container templates (#2584)
 * 51bc876d test: Fixes TestCreateWorkflowDryRun. Closes #2618 (#2628)
 * 212c6d75 fix: Support minimal mysql version 5.7.8 (#2633)
 * 8facacee refactor: Refactor Template context interfaces (#2573)
 * 812813a2 fix: fix test cases (#2631)
 * ed028b25 fix: Fix logging problems. See #2589 (#2595)
 * d4e81238 test: Fix teething problems (#2630)
 * 4aad6d55 chore: Add comments to issues (#2627)
 * 54f7a013 test: Enhancements and repairs to e2e test framework (#2609)
 * d95926fe fix: Fix WorkflowTemplate icons to be more cohesive (#2607)
 * 0130e1fd docs: Add fields and core concepts doc (#2610)
 * 5a1ac203 fix: Fixes panic in toWorkflow method (#2604)
 * 51910292 chore: Lint UI on CI, test diagnostics, skip bad test (#2587)
 * 232bb115 fix(error handling): use Errorf instead of New when throwing errors with formatted text (#2598)
 * eeb2f97b fix(controller): dag continue on failed. Fixes #2596 (#2597)
 * 99c35129 docs: Fix inaccurate field name in docs (#2591)
 * 21c73779 fix: Fixes lint errors (#2594)
 * 38aca5fa chore: Added ClusterWorkflowTemplate RBAC on quick-start manifests (#2576)
 * 59f746e1 feat: UI enhancement for Cluster Workflow Template (#2525)
 * 0801a428 fix(cli): Show lint errors of all files (#2552)
 * c3535ba5 docs: Fix wrong Configuring Your Artifact Repository document. (#2586)
 * 79217bc8 feat(archive): allow specifying a compression level (#2575)
 * 88d261d7 fix: Use outputs of last child instead of retry node itslef (#2565)
 * 5c08292e style: Correct the confused logic (#2577)
 * 3d146144 fix: Fix bug in deleting pods. Fixes #2571 (#2572)
 * cb739a68 feat: Cluster scoped workflow template (#2451)
 * c63e3d40 feat: Show workflow duration in the index UI page (#2568)
 * 1520452a chore: Error -> Warn when Parent CronWf no longer exists (#2566)
 * ffbb3b89 fix: Fixes empty/missing CM. Fixes #2285 (#2562)
 * d0fba6f4 chore: fix typos in the workflow template docs (#2563)
 * 49801e32 chore(docker): upgrade base image for executor image (#2561)
 * c4efb8f8 Add Riskified to the user list (#2558)
 * 8b92d33e feat: Create K8S events on node completion. Closes #2274 (#2521)
 * 2902e144 feat: Add node type and phase filter to UI (#2555)
 * fb74ba1c fix: Separate global scope processing from local scope building (#2528)
 * 618b6dee fix: Fixes --kubeconfig flag. Fixes #2492 (#2553)
 * 79dc969f test: Increase timeout for flaky test (#2543)
 * 15a3c990 feat: Report SpecWarnings in status.conditions (#2541)
 * f142f30a docs: Add example of template-level volume declaration. (#2542)
 * 93b6be61 fix(archive): Fix bug that prevents listing archive workflows. Fixes … (#2523)
 * b4c9c54f fix: Omit config key in configure artifact document. (#2539)
 * 864bf1e5 fix: Show template on its own field in CLI (#2535)
 * 555aaf06 test: fix master (#2534)
 * 94862b98 chore: Remove deprecated example (#2533)
 * 5e1e7829 fix: Validate CronWorkflow before creation (#2532)
 * c9241339 fix: Fix wrong assertions (#2531)
 * 67fe04bb Revert "fix: fix template scope tests (#2498)" (#2526)
 * ddfa1ad0 docs: couple of examples for REST API usage of argo-server (#2519)
 * 30542be7 chore(docs): Update docs for useSDKCreds (#2518)
 * e2cc6988 feat: More control over resuming suspended nodes Fixes #1893 (#1904)
 * b2771249 chore: minor fix and refactory (#2517)
 * b1ad163a fix: fix template scope tests (#2498)
 * 661d1b67 Increase client gRPC max size to match server (#2514)
 * d8aa477f fix: Fix potential panic (#2516)
 * 1afb692e fix: Allow runtime resolution for workflow parameter names (#2501)
 * 243ea338 fix(controller): Ensure we copy any executor securityContext when creating wait containers; fixes #2512 (#2510)
 * 6e8c7bad feat: Extend workflowDefaults to full Workflow and clean up docs and code (#2508)
 * 06cfc129 feat: Native Google Cloud Storage support for artifact. Closes #1911 (#2484)
 * 999b1e1d  fix: Read ConfigMap before starting servers  (#2507)
 * 3d6e9b61 docs: Add separate ConfigMap doc for 2.7+ (#2505)
 * e5bd6a7e fix(controller): Updates GetTaskAncestry to skip visited nod. Fixes #1907 (#1908)
 * 183a29e4 docs: add official user (#2499)
 * e636000b feat: Updated arm64 support patch (#2491)
 * 559cb005 feat(ui): Report resources duration in UI. Closes #2460 (#2489)
 * 09291d9d feat: Add default field in parameters.valueFrom (#2500)
 * 33cd4f2b feat(config): Make configuration mangement easier. Closes #2463 (#2464)
 * f3df9660 test: Fix test (#2490)
 * bfaf1c21 chore: Move quickstart Prometheus port to 9090 (#2487)
 * 487ed425 feat: Logging the Pod Spec in controller log (#2476)
 * 96c80e3e fix(cli): Rearrange the order of chunk size argument in list command. Closes #2420 (#2485)
 * 47bd70a0 chore: Fix Swagger for PDB to support Java client (#2483)
 * 53a10564 feat(usage): Report resource duration. Closes #1066 (#2219)
 * 063d9bc6 Revert "feat: Add support for arm64 platform (#2364)" (#2482)
 * 735d25e9 fix: Build image with SHA tag when a git tag is not available (#2479)
 * c55bb3b2 ci: Run lint on CI and fix GolangCI (#2470)
 * e1c9f7af fix ParallelSteps child type so replacements happen correctly; fixes argoproj-labs/argo-client-gen#5 (#2478)
 * 55c315db feat: Add support for IRSA and aws default provider chain. (#2468)
 * c724c7c1 feat: Add support for arm64 platform (#2364)
 * 315dc164 feat: search archived wf by startat. Closes #2436 (#2473)
 * 23d230bd feat(ui): add Env to Node Container Info pane. Closes #2471 (#2472)
 * 10a0789b fix: ParallelSteps swagger.json (#2459)
 * a59428e7 fix: Duration must be a string. Make it a string. (#2467)
 * 47bc6f3b feat: Add `argo stop` command (#2352)
 * 14478bc0 feat(ui): Add the ability to have links to logging facility in UI. Closes #2438 (#2443)
 * 2864c745 chore: make codegen + make start (#2465)
 * a85f62c5 feat: Custom, step-level, and usage metrics (#2254)
 * 64ac0298 fix: Deprecate template.{template,templateRef,arguments} (#2447)
 * 6cb79e4e fix: Postgres persistence SSL Mode (#1866) (#1867)
 * 2205c0e1 fix(controller): Updates to add condition to workflow status. Fixes #2421 (#2453)
 * 9d96ab2f fix: make dir if needed (#2455)
 * 5346609e test: Maybe fix TestPendingRetryWorkflowWithRetryStrategy. Fixes #2446 (#2456)
 * 3448ccf9 fix: Delete PVCs unless WF Failed/Errored (#2449)
 * 782bc8e7 fix: Don't error when optional artifacts are not found (#2445)
 * fc18f3cf chore: Master needs codegen (#2448)
 * 32fc2f78 feat: Support workflow templates submission. Closes #2007 (#2222)
 * 050a143d fix(archive): Fix edge-cast error for archiving. Fixes #2427 (#2434)
 * 9455c1b8 doc: update CHANGELOG.md (#2425)
 * 1baa7ee4 feat(ui): cache namespace selection. Closes #2439 (#2441)
 * 91d29881 feat: Retry pending nodes (#2385)
 * 7094433e test: Skip flakey tests in operator_template_scope_test.go. See #2432 (#2433)
 * 30332b14 fix: Allow numbers in steps.args.params.value (#2414)
 * e9a06dde feat: instanceID support for argo server. Closes #2004 (#2365)
 * 3f8be0cd fix "Unable to retry workflow" on argo-server (#2409)
 * dd3029ab docs: Example showing how to use default settings for workflow spec. Related to ##2388 (#2411)
 * 13508828 fix: Check child node status before backoff in retry (#2407)
 * b59419c9 fix: Build with the correct version if you check out a specific version (#2423)
 * 6d834d54 chore: document BASE_HREF (#2418)
 * 184c3653 fix: Remove lazy workflow template (#2417)
 * 918d0d17 docs: Added Survey Results (#2416)
 * 20d6e27b Update CONTRIBUTING.md (#2410)
 * f2ca045e feat: Allow WF metadata spec on Cron WF (#2400)
 * 068a4336 fix: Correctly report version. Fixes #2374 (#2402)
 * e19a398c Update pull_request_template.md (#2401)
 * 7c99c109 chore: Fix typo (#2405)
 * 175b164c Change font family for class yaml (#2394)
 * d1194755 fix: Don't display Retry Nodes in UI if len(children) == 1 (#2390)
 * b8623ec7 docs: Create USERS.md (#2389)
 * 1d21d3f5 fix(doc strings): Fix bug related documentation/clean up of default configurations #2331 (#2388)
 * 77e11fc4 chore: add noindex meta tag to solve #2381; add kustomize to build docs (#2383)
 * 42200fad fix(controller): Mount volumes defined in script templates. Closes #1722 (#2377)
 * 96af36d8 fix: duration must be a string (#2380)
 * 7bf08192 fix: Say no logs were outputted when pod is done (#2373)
 * 847c3507 fix(ui): Removed tailLines from EventSource (#2330)
 * 3890a124 feat: Allow for setting default configurations for workflows, Fixes #1923, #2044 (#2331)
 * 81ab5385 Update readme (#2379)
 * 91810273 feat: Log version (structured) on component start-up (#2375)
 * d0572a74 docs: Make Getting Started agnostic to version (#2371)
 * d3a3f6b1 docs: Add Prudential to the users list (#2353)
 * 4714c880 chore: Master needs codegen (#2369)
 * 5b6b8257 fix(docker): fix streaming of combined stdout/stderr (#2368)
 * 97438313 fix: Restart server ConfigMap watch when closed (#2360)
 * 64d0cec0 chore: Master needs make lint (#2361)
 * 12386fc6 fix: rerun codegen after merging OSS artifact support (#2357)
 * 40586ed5 fix: Always validate templates (#2342)
 * 897db894 feat: Add support for Alibaba Cloud OSS artifact (#1919)
 * 7e2dba03 feat(ui): Circles for nodes (#2349)
 * e85f6169 chore: update getting started guide to use 2.6.0 (#2350)
 * 7ae4ec78 docker: remove NopCloser from the executor. (#2345)
 * 5895b364 feat: Expose workflow.paramteres with JSON string of all params (#2341)
 * a9850b43 Fix the default (#2346)
 * c3763d34 fix: Simplify completion detection logic in DAGs (#2344)
 * d8a9ea09 fix(auth): Fixed returning  expired  Auth token for GKE (#2327)
 * 6fef0454 fix: Add timezone support to startingDeadlineSeconds (#2335)
 * c28731b9 chore: Add go mod tidy to codegen (#2332)
 * a66c8802 feat: Allow Worfklows to be submitted as files from UI (#2340)
 * a9c1d547 docs: Update Argo Rollouts description (#2336)
 * 8672b97f fix(Dockerfile): Using `--no-install-recommends` (Optimization) (#2329)
 * c3fe1ae1 fix(ui): fixed worflow UI refresh. Fixes ##2337 (#2338)
 * d7690e32 feat(ui): Adds ability zoom and hide successful steps. POC (#2319)
 * e9e13d4c feat: Allow retry strategy on non-leaf nodes, eg for step groups. Fixes #1891 (#1892)
 * 62e6db82 feat: Ability to include or exclude fields in the response (#2326)
 * 52ba89ad fix(swagger): Fix the broken swagger. (#2317)
 * efb8a1ac docs: Update CODE_OF_CONDUCT.md (#2323)
 * 1c77e864 fix(swagger): Fix the broken swagger. (#2317)
 * aa052346 feat: Support workflow level poddisruptionbudge for workflow pods #1728 (#2286)
 * 8da88d7e chore: update getting-started guide for 2.5.2 and apply other tweaks (#2311)
 * 2f97c261 build: Improve reliability of release. (#2309)
 * 5dcb84bb chore(cli): Clean-up code. Closes #2117 (#2303)
 * e49dd8c4 chore(cli): Migrate `argo logs` to use API client. See #2116 (#2177)
 * 5c3d9cf9 chore(cli): Migrate `argo wait` to use API client. See #2116 (#2282)
 * baf03f67 fix(ui): Provide a link to archived logs. Fixes #2300 (#2301)
 * b5947165 feat: Create API clients (#2218)
 * 214c4515 fix(controller): Get correct Step or DAG name. Fixes #2244 (#2304)
 * c4d26466 fix: Remove active wf from Cron when deleted (#2299)
 * 0eff938d fix: Skip empty withParam steps (#2284)
 * 636ea443 chore(cli): Migrate `argo terminate` to use API client. See #2116 (#2280)
 * d0a9b528 chore(cli): Migrate `argo template` to use API client. Closes #2115 (#2296)
 * f69a6c5f chore(cli): Migrate `argo cron` to use API client. Closes #2114 (#2295)
 * 80b9b590 chore(cli): Migrate `argo retry` to use API client. See #2116 (#2277)
 * cdbc6194 fix(sequence): broken in 2.5. Fixes #2248 (#2263)
 * 0d3955a7 refactor(cli): 2x simplify migration to API client. See #2116 (#2290)
 * df8493a1 fix: Start Argo server with out Configmap #2285 (#2293)
 * 51cdf95b doc: More detail for namespaced installation (#2292)
 * a7302697 build(swagger): Fix argo-server swagger so version does not change. (#2291)
 * 47b4fc28 fix(cli): Reinstate `argo wait`. Fixes #2281 (#2283)
 * 1793887b chore(cli): Migrate `argo suspend` and `argo resume` to use API client. See #2116 (#2275)
 * 1f3d2f5a chore(cli): Update `argo resubmit` to support client API. See #2116 (#2276)
 * c33f6cda fix(archive): Fix bug in migrating cluster name. Fixes #2272 (#2279)
 * fb0acbbf fix: Fixes double logging in UI. Fixes #2270 (#2271)
 * acf37c2d fix: Correctly report version. Fixes #2264 (#2268)
 * b30f1af6 fix: Removes Template.Arguments as this is never used. Fixes #2046 (#2267)
 * 79b09ed4 fix: Removed duplicate Watch Command (#2262)
 * b5c47266 feat(ui): Add filters for archived workflows (#2257)
 * d30aa335 fix(archive): Return correct next page info. Fixes #2255 (#2256)
 * 8c97689e fix: Ignore bookmark events for restart. Fixes #2249 (#2253)
 * 63858eaa fix(offloading): Change offloaded nodes datatype to JSON to support 1GB. Fixes #2246 (#2250)
 * 4d88374b Add Cartrack into officially using Argo (#2251)
 * d309d5c1 feat(archive): Add support to filter list by labels. Closes #2171 (#2205)
 * 79f13373 feat: Add a new symbol for suspended nodes. Closes #1896 (#2240)
 * 82b48821 Fix presumed typo (#2243)
 * af94352f feat: Reduce API calls when changing filters. Closes #2231 (#2232)
 * a58cbc7d BasisAI uses Argo (#2241)
 * 68e3c9fd feat: Add Pod Name to UI (#2227)
 * eef85072 fix(offload): Fix bug which deleted completed workflows. Fixes #2233 (#2234)
 * 4e4565cd feat: Label workflow-created pvc with workflow name (#1890)
 * 8bd5ecbc fix: display error message when deleting archived workflow fails. (#2235)
 * ae381ae5 feat: This add support to enable debug logging for all CLI commands (#2212)
 * 1b1927fc feat(swagger): Adds a make api/argo-server/swagger.json (#2216)
 * 5d7b4c8c Update README.md (#2226)
 * 170abfa5 chore: Run `go mod tidy` (#2225)
 * 2981e6ff fix: Enforce UnknownField requirement in WorkflowStep (#2210)
 * affc235c feat: Add failed node info to exit handler (#2166)
 * af1f6d60 fix: UI Responsive design on filter box (#2221)
 * a445049c fix: Fixed race condition in kill container method. Fixes #1884 (#2208)
 * 2672857f feat: upgrade to Go 1.13. Closes #1375 (#2097)
 * 7466efa9 feat: ArtifactRepositoryRef ConfigMap is now taken from the workflow namespace (#1821)
 * 50f331d0 build: Fix ARGO_TOKEN (#2215)
 * 7f090351 test: Correctly report diagnostics (#2214)
 * f2bd74bc fix: Remove quotes from UI (#2213)
 * 62f46680 fix(offloading): Correctly deleted offloaded data. Fixes #2206 (#2207)
 * e30b77fc feat(ui): Add label filter to workflow list page. Fixes #802 (#2196)
 * 930ced39 fix(ui): fixed workflow filtering and ordering. Fixes #2201 (#2202)
 * 88112312 fix: Correct login instructions. (#2198)
 * d6f5953d Update ReadMe for EBSCO (#2195)
 * b024c46c feat: Add ability to submit CronWorkflow from CLI (#2003)
 * c97527ce test: Invoke tests using s.T() (#2194)
 * 72a54fe1 chore: Move info.proto et al to correct package (#2193)
 * f6600fa4 fix: Namespace and phase selection in UI (#2191)
 * c4a24dca fix(k8sapi-executor): Fix KillContainer impl (#2160)
 * d22a5fe6 Update cli_with_server_test.go (#2189)
 * ff18180f test: Remove podGC (#2187)
 * 78245305 chore: Improved error handling and refactor (#2184)
 * b9c828ad fix(archive): Only delete offloaded data we do not need. Fixes #2170 and #2156 (#2172)
 * 73cb5418 feat: Allow CronWorkflows to have instanceId (#2081)
 * 9efea660 Sort list and add Greenhouse (#2182)
 * cae399ba fix: Fixed the Exec Provider token bug (#2181)
 * fc476b2a fix(ui): Retry workflow event stream on connection loss. Fixes #2179 (#2180)
 * 65058a27 fix: Correctly create code from changed protos. (#2178)
 * 585d1eef chore: Update lint command to use apiclient. See #2116 (#2131)
 * 299d467c build: Update release process and docs (#2128)
 * fcfe1d43 feat: Implemented open default browser in local mode (#2122)
 * f6cee552 fix: Specify download .tgz extension (#2164)
 * 8a1e611a feat: Update archived workdflow column to be JSON. Closes #2133 (#2152)
 * f591c471 fix!: Change `argo token` to `argo auth token`. Closes #2149 (#2150)
 * 519c9434 chore: Add Mock gen to make codegen (#2148)
 * 409a5154 fix: Add certs to argocli image. Fixes #2129 (#2143)
 * b094802a fix: Allow download of artifacs in server auth-mode. Fixes #2129 (#2147)
 * 520fa540 fix: Correct SQL syntax. (#2141)
 * 059cb9b1 fix: logs UI should fall back to archive (#2139)
 * 4cda9a05 fix: route all unknown web content requests to index.html (#2134)
 * 14d8b5d3 fix: archiveLogs needs to copy stderr (#2136)
 * 91319ee4 fixed ui navigation issues with basehref (#2130)
 * 7881b980 docs: Add CronWorkflow usage docs (#2124)
 * badfd183 feat: Add support to delete by using labels. Depended on by #2116 (#2123)
 * 706d0f23 test: Try and make e2e more robust. Fixes #2125 (#2127)
 * a75ac1b4 fix: mark CLI common.go vars and funcs as DEPRECATED (#2119)
 * be21a0f1 feat(server): Restart server when config changes. Fixes #2090 (#2092)
 * b5cd72b0 test: Parallelize Cron tests (#2118)
 * b2bd25bc fix: Disable webpack dot rule (#2112)
 * 865b4f3a addcompany (#2109)
 * 213e3a9d fix: Fix Resource Deletion Bug (#2084)
 * ab1de233 refactor(cli): Introduce v1.Interface for CLI. Closes #2107 (#2048)
 * 7a19f85c feat: Implemented Basic Auth scheme (#2093)
 * 7611b9f6 fix(ui): Add support for bash href. Fixes ##2100 (#2105)
 * 516d05f8  fix: Namespace redirects no longer error and are snappier (#2106)
 * 16aed5c8 fix: Skip running --token testing if it is not on CI (#2104)
 * aece7e6e Parse container ID in correct way on CRI-O. Fixes #2095 (#2096)
 * b6a2be89 feat: support arg --token when talking to argo-server (#2027) (#2089)
 * 01d8cae1 build: adds `make env` to make testing easier (#2087)
 * 492842aa docs(README): Add Capital One to user list (#2094)
 * d56a0e12 fix(controller): Fix template resolution for step groups. Fixes #1868  (#1920)
 * b97044d2 fix(security): Fixes an issue that allowed you to list archived workf… (#2079)
 * c4f49cf0 refactor: Move server code (cmd/server/ -> server/) (#2071)
 * 2542454c fix(controller): Do not crash if cm is empty. Fixes #2069 (#2070)
 * 85fa9aaf fix: Do not expect workflowChange to always be defined (#2068)
 * 6f65bc2b fix: "base64 -d" not always available, using "base64 --decode" (#2067)
 * 6f2c8802 feat(ui): Use cookies in the UI. Closes #1949 (#2058)
 * 4592aec6 fix(api): Change `CronWorkflowName` to `Name`. Fixes #1982 (#2033)
 * e26c11af fix: only run archived wf testing when persistence is enabled (#2059)
 * b3cab5df fix: Fix permission test cases (#2035)
 * b408e7cd fix: nil pointer in GC (#2055)
 * 4ac11560 fix: offload Node Status in Get and List api call (#2051)
 * dfdde1d0 ci: Run using our own cowsay image (#2047)
 * 71ba8238 Update README.md (#2045)
 * c7953052 fix(persistence): Allow `argo server` to run without persistence (#2050)
 * 1db74e1a fix(archive): upsert archive + ci: Pin images on CI, add readiness probes, clean-up logging and other tweaks (#2038)
 * c46c6836 feat: Allow workflow-level parameters to be modified in the UI when submitting a workflow (#2030)
 * faa9dbb5 fix(Makefile): Rename staticfiles make target. Fixes #2010 (#2040)
 * 79a42d48 docs: Update link to configure-artifact-repository.md (#2041)
 * 1a96007f fix: Redirect to correct page when using managed namespace. Fixes #2015 (#2029)
 * 78726314 fix(api): Updates proto message naming (#2034)
 * 4a1307c8 feat: Adds support for MySQL. Fixes #1945 (#2013)
 * d843e608 chore: Smoke tests are timing out, give them more time (#2032)
 * 5c98a14e feat(controller): Add audit logs to workflows. Fixes #1769 (#1930)
 * 2982c1a8 fix(validate): Allow placeholder in values taken from inputs. Fixes #1984 (#2028)
 * 3293c83f feat: Add version to offload nodes. Fixes #1944 and #1946 (#1974)
 * 283bbf8d build: `make clean` now only deletes dist directories (#2019)
 * 72fa88c9 build: Enable linting for tests. Closes #1971 (#2025)
 * f8569ae9 feat: Auth refactoring to support single version token (#1998)
 * eb360d60 Fix README (#2023)
 * ef1bd3a3 fix typo (#2021)
 * f25a45de feat(controller): Exposes container runtime executor as CLI option. (#2014)
 * 3b26af7d Enable s3 trace support. Bump version to v2.5.0. Tweak proto id to match Workflow (#2009)
 * 5eb15bb5 fix: Fix workflow level timeouts (#1369)
 * 5868982b fix: Fixes the `test` job on master (#2008)
 * 29c85072 fix: Fixed grammar on TTLStrategy (#2006)
 * 2f58d202 fix: v2 token bug (#1991)
 * ed36d92f feat: Add quick start manifests to Git. Change auth-mode to default to server. Fixes #1990 (#1993)
 * d1965c93 docs: Encourage users to upvote issues relevant to them (#1996)
 * 91331a89 fix: No longer delete the argo ns as this is dangerous (#1995)
 * 1a777cc6 feat(cron): Added timezone support to cron workflows. Closes #1931 (#1986)
 * 48b85e57 fix: WorkflowTempalteTest fix (#1992)
 * 51dab8a4 feat: Adds `argo server` command. Fixes #1966 (#1972)
 * 732e03bb chore: Added WorkflowTemplate test (#1989)
 * 27387d4b chore: Fix UI TODOs (#1987)
 * dd704dd6 feat: Renders namespace in UI. Fixes #1952 and #1959 (#1965)
 * 14d58036 feat(server): Argo Server. Closes #1331 (#1882)
 * f69655a0 fix: Added decompress in retry, resubmit and resume. (#1934)
 * 1e7ccb53 updated jq version to 1.6 (#1937)
 * c51c1302 feat: Enhancement for namespace installation mode configuration (#1939)
 * 6af100d5 feat: Add suspend and resume to CronWorkflows CLI (#1925)
 * 232a465d feat: Added onExit handlers to Step and DAG (#1716)
 * 071eb112 docs: Update PR template to demand tests. (#1929)
 * ae58527e docs: Add CyberAgent to the list of Argo users (#1926)
 * 02022e4b docs: Minor formatting fix (#1922)
 * e4107bb8 Updated Readme.md for companies using Argo: (#1916)
 * 7e9b2b58 feat: Support for scheduled Workflows with CronWorkflow CRD (#1758)
 * 5d7e9185 feat: Provide values of withItems maps as JSON in {{item}}. Fixes #1905 (#1906)
 * de3ffd78  feat: Enhanced Different TTLSecondsAfterFinished depending on if job is in Succeeded, Failed or Error, Fixes (#1883)
 * 94449876 docs: Add question option to issue templates (#1910)
 * 83ae2df4 fix: Decrease docker build time by ignoring node_modules (#1909)
 * 59a19069 feat: support iam roles for service accounts in artifact storage (#1899)
 * 6526b6cc fix: Revert node creation logic (#1818)
 * 160a7940 fix: Update Gopkg.lock with dep ensure -update (#1898)
 * ce78227a fix: quick fail after pod termination (#1865)
 * cd3bd235 refactor: Format Argo UI using prettier (#1878)
 * b48446e0 fix: Fix support for continueOn failed for DAG. Fixes #1817 (#1855)
 * 48256961 fix: Fix template scope (#1836)
 * eb585ef7 fix: Use dynamically generated placeholders (#1844)
 * c821cfcc test: Adds 'test' and 'ui' jobs to CI (#1869)
 * 54f44909 feat: Always archive logs if in config. Closes #1790 (#1860)
 * 1e25d6cf docs: Fix e2e testing link (#1873)
 * f5f40728 fix: Minor comment fix (#1872)
 * 72fad7ec Update docs (#1870)
 * 90352865 docs: Update doc based on helm 3.x changes (#1843)
 * 78889895 Move Workflows UI from https://github.com/argoproj/argo-ui (#1859)
 * 4b96172f docs: Refactored and cleaned up docs (#1856)
 * 6ba4598f test: Adds core e2e test infra. Fixes #678 (#1854)
 * 87f26c8d fix: Move ISSUE_TEMPLATE/ under .github/ (#1858)
 * bd78d159 fix: Ensure timer channel is empty after stop (#1829)
 * afc63024 Code duplication (#1482)
 * 5b136713 docs: biobox analytics (#1830)
 * 68b72a8f add CCRi to list of users in README (#1845)
 * 941f30aa Add Sidecar Technologies to list of who uses Argo (#1850)
 * a08048b6 Adding Wavefront to the users list (#1852)
 * 1cb68c98 docs: Updates issue and PR templates. (#1848)
 * cb0598ea Fixed Panic if DB context has issue (#1851)
 * e5fb8848 fix: Fix a couple of nil derefs (#1847)
 * b3d45850 Add HOVER to the list of who uses Argo (#1825)
 * 99db30d6 InsideBoard uses Argo (#1835)
 * ac8efcf4 Red Hat uses Argo (#1828)
 * 41ed3acf Adding Fairwinds to the list of companies that use Argo (#1820)
 * 5274afb9 Add exponential back-off to retryStrategy (#1782)
 * e522e30a Handle operation level errors PVC in Retry (#1762)
 * f2e6054e Do not resolve remote templates in lint (#1787)
 * 3852bc3f SSL enabled database connection for workflow repository (#1712) (#1756)
 * f2676c87 Fix retry node name issue on error (#1732)
 * d38a107c Refactoring Template Resolution Logic (#1744)
 * 23e94604 Error occurred on pod watch should result in an error on the wait container (#1776)
 * 57d051b5 Added hint when using certain tokens in when expressions (#1810)
 * 0e79edff Make kubectl print status and start/finished time (#1766)
 * 723b3c15 Fix code-gen docs (#1811)
 * 711bb114 Fix withParam node naming issue (#1800)
 * 4351a336 Minor doc fix (#1808)
 * efb748fe Fix some issues in examples (#1804)
 * a3e31289 Add documentation for executors (#1778)
 * 1ac75b39 Add  to linter (#1777)
 * 3bead0db Add ability to retry nodes after errors (#1696)
 * b50845e2 Support no-headers flag (#1760)
 * 7ea2b2f8 Minor rework of suspened node (#1752)
 * 9ab1bc88 Update README.md (#1768)
 * e66fa328 Fixed lint issues (#1739)
 * 63e12d09 binary up version (#1748)
 * 1b7f9bec Minor typo fix (#1754)
 * 4c002677 fix blank lines (#1753)
 * fae73826 Fail suspended steps after deadline (#1704)
 * b2d7ee62 Fix typo in docs (#1745)
 * f2592448 Removed uneccessary debug Println (#1741)
 * 846d01ed Filter workflows in list  based on name prefix (#1721)
 * 8ae688c6 Added ability to auto-resume from suspended state (#1715)
 * fb617b63 unquote strings from parameter-file (#1733)
 * 34120341 example for pod spec from output of previous step (#1724)
 * 12b983f4 Add gonum.org/v1/gonum/graph to Gopkg.toml (#1726)
 * 327fcb24 Added  Protobuf extension  (#1601)
 * 602e5ad8 Fix invitation link. (#1710)
 * eb29ae4c Fixes bugs in demo (#1700)
 * ebb25b86 `restartPolicy` -> `retryStrategy` in examples (#1702)
 * 167d65b1 Fixed incorrect `pod.name` in retry pods (#1699)
 * e0818029 fixed broke metrics endpoint per #1634 (#1695)
 * 36fd09a1 Apply Strategic merge patch against the pod spec (#1687)
 * d3546467 Fix retry node processing (#1694)
 * dd517e4c Print multiple workflows in one command (#1650)
 * 09a6cb4e Added status of previous steps as variables (#1681)
 * ad3dd4d4 Fix issue that workflow.priority substitution didn't pass validation (#1690)
 * 095d67f8 Store locally referenced template properly (#1670)
 * 30a91ef0 Handle retried node properly (#1669)
 * 263cb703 Update README.md  Argo Ansible role: Provisioning Argo Workflows on Kubernetes/OpenShift (#1673)
 * 867f5ff7 Handle sidecar killing properly (#1675)
 * f0ab9df9 Fix typo (#1679)
 * 502db42d Don't provision VM for empty artifacts (#1660)
 * b5dcac81 Resolve WorkflowTemplate lazily (#1655)
 * d15994bb [User] Update Argo users list (#1661)
 * 4a654ca6 Stop failing if artifact file exists, but empty (#1653)
 * c6cddafe Bug fixes in getting started (#1656)
 * ec788373 Update workflow_level_host_aliases.yaml (#1657)
 * 7e5af474 Fix child node template handling (#1654)
 * 7f385a6b Use stored templates to raggregate step outputs (#1651)
 * cd6f3627 Fix dag output aggregation correctly (#1649)
 * 706075a5 Fix DAG output aggregation (#1648)
 * fa32dabd Fix missing merged changes in validate.go (#1647)
 * 45716027 fixed example wrong comment (#1643)
 * 69fd8a58 Delay killing sidecars until artifacts are saved (#1645)
 * ec5f9860 pin colinmarc/hdfs to the next commit, which no longer has vendored deps (#1622)
 * 4b84f975 Fix global lint issue (#1641)
 * bb579138 Fix regression where global outputs were unresolveable in DAGs (#1640)
 * cbf99682 Fix regression where parallelism could cause workflow to fail (#1639)
 * 76461f92 Update CHANGELOG for v2.4.0 (#1636)
 * c75a0861 Regenerate installation manifests (#1638)
 * e20cb28c Grant get secret role to controller to support persistence (#1615)
 * 644946e4 Save stored template ID in nodes (#1631)
 * 5d530bec Fix retry workflow state (#1632)
 * 2f0af522 Update operator.go (#1630)
 * 6acea0c1 Store resolved templates (#1552)
 * df8260d6 Increase timeout of golangci-lint (#1623)
 * 138f89f6 updated invite link (#1621)
 * d027188d Updated the API Rule Violations list (#1618)
 * a317fbf1 Prevent controller from crashing due to glog writing to /tmp (#1613)
 * 20e91ea5 Added WorkflowStatus and NodeStatus types to the Open API Spec. (#1614)
 * ffb281a5 Small code cleanup and add tests (#1562)
 * 1cb8345d Add merge keys to Workflow objects to allow for StrategicMergePatches (#1611)
 * c855a66a Increased Lint timeout (#1612)
 * 4bf83fc3 Fix DAG enable failFast will hang in some case (#1595)
 * e9f3d9cb Do not relocate the mounted docker.sock (#1607)
 * 1bd50fa2 Added retry around RuntimeExecutor.Wait call when waiting for main container completion (#1597)
 * 0393427b Issue1571  Support ability to assume IAM roles in S3 Artifacts  (#1587)
 * ffc0c84f Update Gopkg.toml and Gopkg.lock (#1596)
 * aa3a8f1c Update from github.com/ghodss/yaml to sigs.k8s.io/yaml (#1572)
 * 07a26f16 Regard resource templates as leaf nodes (#1593)
 * 89e959e7 Fix workflow template in namespaced controller (#1580)
 * cd04ab8b remove redundant codes (#1582)
 * 5bba8449 Add entrypoint label to workflow default labels (#1550)
 * 9685d7b6 Fix inputs and arguments during template resolution (#1545)
 * 19210ba6 added DataStax as an organization that uses Argo (#1576)
 * b5f2fdef Support AutomountServiceAccountToken and executor specific service account(#1480)
 * 8808726c Fix issue saving outputs which overlap paths with inputs (#1567)
 * ba7a1ed6 Add coverage make target (#1557)
 * ced0ee96 Document workflow controller dockerSockPath config (#1555)
 * 3e95f2da Optimize argo binary install documentation (#1563)
 * e2ebb166 docs(readme): fix workflow types link (#1560)
 * 6d150a15 Initialize the wfClientset before using it (#1548)
 * 5331fc02 Remove GLog config from argo executor (#1537)
 * ed4ac6d0 Update main.go (#1536)
 * 9fca1441 Update argo dependencies to kubernetes v1.14 (#1530)
 * 0246d184 Use cache to retrieve WorkflowTemplates (#1534)
 * 4864c32f Update README.md (#1533)
 * 4df114fa Update CHANGELOG for v2.4 (#1531)
 * c7e5cba1 Introduce podGC strategy for deleting completed/successful pods (#1234)
 * bb0d14af Update ISSUE_TEMPLATE.md (#1528)
 * b5702d8a Format sources and order imports with the help of goimports (#1504)
 * d3ff77bf Added Architecture doc (#1515)
 * fc1ec1a5 WorkflowTemplate CRD (#1312)
 * f99d3266 Expose all input parameters to template as JSON (#1488)
 * bea60526 Fix argo logs empty content when workflow run in virtual kubelet env (#1201)
 * d82de881 Implemented support for WorkflowSpec.ArtifactRepositoryRef (#1350)
 * 0fa20c7b Fix validation (#1508)
 * 87e2cb60 Add --dry-run option to `argo submit` (#1506)
 * e7e50af6 Support git shallow clones and additional ref fetches (#1521)
 * 605489cd Allow overriding workflow labels in 'argo submit' (#1475)
 * 47eba519 Fix issue [Documentation] kubectl get service argo-artifacts -o wide (#1516)
 * 02f38262 Fixed #1287 Executor kubectl version is obsolete (#1513)
 * f62105e6 Allow Makefile variables to be set from the command line (#1501)
 * e62be65b Fix a compiler error in a unit test (#1514)
 * 5c5c29af Fix the lint target (#1505)
 * e03287bf Allow output parameters with .value, not only .valueFrom (#1336)
 * 781d3b8a Implemented Conditionally annotate outputs of script template only when consumed #1359 (#1462)
 * b028e61d change 'continue-on-fail' example to better reflect its description (#1494)
 * 97e824c9 Readme update to add argo and airflow comparison (#1502)
 * 414d6ce7 Fix a compiler error (#1500)
 * ca1d5e67 Fix: Support the List within List type in withParam #1471 (#1473)
 * 75cb8b9c Fix #1366 unpredictable global artifact behavior (#1461)
 * 082e5c4f Exposed workflow priority as a variable (#1476)
 * 38c4def7 Fix: Argo CLI should show warning if there is no workflow definition in file #1486
 * af7e496d Add Commodus Tech as official user (#1484)
 * 8c559642 Update OWNERS (#1485)
 * 007d1f88 Fix: 1008 `argo wait` and `argo submit --wait` should exit 1 if workflow fails  (#1467)
 * 3ab7bc94 Document the insecureIgnoreHostKey git flag (#1483)
 * 7d9bb51a Fix failFast bug:   When a node in the middle fails, the entire workflow will hang (#1468)
 * 42adbf32 Add --no-color flag to logs (#1479)
 * 67fc29c5 fix typo: symboloic > symbolic (#1478)
 * 7c3e1901 Added Codec to the Argo community list (#1477)
 * 0a9cf9d3 Add doc about failFast feature (#1453)
 * 6a590300 Support PodSecurityContext (#1463)
 * e392d854 issue-1445: changing temp directory for output artifacts from root to tmp (#1458)
 * 7a21adfe New Feature:  provide failFast flag, allow a DAG to run all branches of the DAG (either success or failure) (#1443)
 * b9b87b7f Centralized Longterm workflow persistence storage  (#1344)
 * cb09609b mention sidecar in failure message for sidecar containers (#1430)
 * 373bbe6e Fix demo's doc issue of install minio chart (#1450)
 * 83552334 Add threekit to user list (#1444)
 * 83f82ad1 Improve bash completion (#1437)
 * ee0ec78a Update documentation for workflow.outputs.artifacts (#1439)
 * 9e30c06e Revert "Update demo.md (#1396)" (#1433)
 * c08de630 Add paging function for list command (#1420)
 * bba2f9cb Fixed:  Implemented Template level service account (#1354)
 * d635c1de Ability to configure hostPath mount for `/var/run/docker.sock` (#1419)
 * d2f7162a Terminate all containers within pod after main container completes (#1423)
 * 1607d74a PNS executor intermitently failed to capture entire log of script templates (#1406)
 * 5e47256c Fix typo (#1431)
 * 5635c33a Update demo.md (#1396)
 * 83425455 Add OVH as official user (#1417)
 * 82e5f63d Typo fix in ARTIFACT_REPO.md (#1425)
 * 15fa6f52 Update OWNERS (#1429)
 * 96b9a40e Orders uses alphabetically (#1411)
 * 6550e2cb chore: add IBM to official users section in README.md (#1409)
 * bc81fe28 Fiixed: persistentvolumeclaims already exists #1130 (#1363)
 * 6a042d1f Update README.md (#1404)
 * aa811fbd Update README.md (#1402)
 * abe3c99f Add Mirantis as an official user (#1401)
 * 18ab750a Added Argo Rollouts to README (#1388)
 * 67714f99 Make locating kubeconfig in example os independent (#1393)
 * 672dc04f Fixed: withParam parsing of JSON/YAML lists #1389 (#1397)
 * b9aec5f9 Fixed: make verify-codegen is failing on the master branch (#1399) (#1400)
 * 270aabf1 Fixed:  failed to save outputs: verify serviceaccount default:default has necessary privileges (#1362)
 * 163f4a5d Fixed: Support hostAliases in WorkflowSpec #1265 (#1365)
 * abb17478 Add Max Kelsen to USERS in README.md (#1374)
 * dc549193 Update docs for the v2.3.0 release and to use the stable tag
 * 4001c964 Update README.md (#1372)
 * 6c18039b Fix issue where a DAG with exhausted retries would get stuck Running (#1364)
 * d7e74fe3 Validate action for resource templates (#1346)
 * 810949d5 Fixed :  CLI Does Not Honor metadata.namespace #1288 (#1352)
 * e58859d7 [Fix #1242] Failed DAG nodes are now kept and set to running on RetryWorkflow. (#1250)
 * d5fe5f98 Use golangci-lint instead of deprecated gometalinter (#1335)
 * 26744d10 Support an easy way to set owner reference (#1333)
 * 8bf7578e Add --status filter for get command (#1325)
 * 3f6ac9c9 Update release instructions
 * 2274130d Update version to v2.3.0-rc3
 * b024b3d8 Fix: # 1328 argo submit --wait and argo wait quits while workflow is running (#1347)
 * 24680b7f Fixed : Validate the secret credentials name and key (#1358)
 * f641d84e Fix input artifacts with multiple ssh keys (#1338)
 * e680bd21 add / test (#1240)
 * ee788a8a Fix #1340 parameter substitution bug (#1345)
 * 60b65190 Fix missing template local volumes, Handle volumes only used in init containers (#1342)
 * 4e37a444 Add documentation on releasing
 * bb1bfdd9 Update version to v2.3.0-rc2. Update changelog
 * 49a6b6d7 wait will conditionally become privileged if main/sidecar privileged (resolves #1323)
 * 34af5a06 Fix regression where argoexec wait would not return when podname was too long
 * bd8d5cb4 `argo list` was not displaying non-zero priorities correctly
 * 64370a2d Support parameter substitution in the volumes attribute (#1238)
 * 6607dca9 Issue1316 Pod creation with secret volumemount  (#1318)
 * a5a2bcf2 Update README.md (#1321)
 * 950de1b9 Export the methods of `KubernetesClientInterface` (#1294)
 * 1c729a72 Update v2.3.0 CHANGELOG.md
 * 40f9a875 Reorganize manifests to kustomize 2 and update version to v2.3.0-rc1
 * 75b28a37 Implement support for PNS (Process Namespace Sharing) executor (#1214)
 * b4edfd30 Fix SIGSEGV in watch/CheckAndDecompress. Consolidate duplicate code (resolves #1315)
 * 02550be3 Archive location should conditionally be added to template only when needed
 * c60010da Fix nil pointer dereference with secret volumes (#1314)
 * db89c477 Fix formatting issues in examples documentation (#1310)
 * 0d400f2c Refactor checkandEstimate to optimize podReconciliation (#1311)
 * bbdf2e2c Add alibaba cloud to officially using argo list (#1313)
 * abb77062 CheckandEstimate implementation to optimize podReconciliation (#1308)
 * 1a028d54 Secrets should be passed to pods using volumes instead of API calls (#1302)
 * e34024a3 Add support for init containers (#1183)
 * 4591e44f Added support for artifact path references (#1300)
 * 928e4df8 Add Karius to users in README.md (#1305)
 * de779f36 Add community meeting notes link (#1304)
 * a8a55579 Speed up podReconciliation using parallel goroutine (#1286)
 * 93451119 Add dns config support (#1301)
 * 850f3f15 Admiralty: add link to blog post, add user (#1295)
 * d5f4b428 Fix for Resource creation where template has same parameter templating (#1283)
 * 9b555cdb Issue#896 Workflow steps with non-existant output artifact path will succeed (#1277)
 * adab9ed6 Argo CI is current inactive (#1285)
 * 59fcc5cc Add workflow labels and annotations global vars (#1280)
 * 1e111caa Fix bug with DockerExecutor's CopyFile (#1275)
 * 73a37f2b Add the `mergeStrategy` option to resource patching (#1269)
 * e6105243 Reduce redundancy pod label action (#1271)
 * 4bfbb20b Error running 1000s of tasks: "etcdserver: request is too large" #1186 (#1264)
 * b2743f30 Proxy Priority and PriorityClassName to pods (#1179)
 * 70c130ae Update versions (#1218)
 * b0384129 Git cloning via SSH was not verifying host public key (#1261)
 * 3f06385b Issue#1165 fake outputs don't notify and task completes successfully (#1247)
 * fa042aa2 typo, executo -> executor (#1243)
 * 1cb88bae Fixed Issue#1223 Kubernetes Resource action: patch is not supported (#1245)
 * 2b0b8f1c Fix the Prometheus address references (#1237)
 * 94cda3d5 Add feature to continue workflow on failed/error steps/tasks (#1205)
 * 3f1fb9d5 Add Gardener to "Who uses Argo" (#1228)
 * cde5cd32 Include stderr when retrieving docker logs (#1225)
 * 2b1d56e7 Update README.md (#1224)
 * eeac5a0e Remove extra quotes around output parameter value (#1232)
 * 8b67e1bf Update README.md (#1236)
 * baa3e622 Update README with typo fixes (#1220)
 * f6b0c8f2 Executor can access the k8s apiserver with a out-of-cluster config file (#1134)
 * 0bda53c7 fix dag retries (#1221)
 * 8aae2931 Issue #1190 - Fix incorrect retry node handling (#1208)
 * f1797f78 Add schedulerName to workflow and template spec (#1184)
 * 2ddae161 Set executor image pull policy for resource template (#1174)
 * edcb5629 Dockerfile: argoexec base image correction (fixes #1209) (#1213)
 * f92284d7 Minor spelling, formatting, and style updates. (#1193)
 * bd249a83 Issue #1128 - Use polling instead of fs notify to get annotation changes (#1194)
 * 14a432e7 Update community/README (#1197)
 * eda7e084 Updated OWNERS (#1198)
 * 73504a24 Fischerjulian adds ruby to rest docs (#1196)
 * 311ad86f Fix missing docker binary in argoexec image. Improve reuse of image layers
 * 831e2198 Issue #988 - Submit should not print logs to stdout unless output is 'wide' (#1192)
 * 17250f3a Add documentation how to use parameter-file's (#1191)
 * 01ce5c3b Add Docker Hub build hooks
 * 93289b42 Refactor Makefile/Dockerfile to remove volume binding in favor of build stages (#1189)
 * 8eb4c666 Issue #1123 - Fix 'kubectl get' failure if resource namespace is different from workflow namespace (#1171)
 * eaaad7d4 Increased S3 artifact retry time and added log (#1138)
 * f07b5afe Issue #1113 - Wait for daemon pods completion to handle annotations (#1177)
 * 2b2651b0 Do not mount unnecessary docker socket (#1178)
 * 1fc03144 Argo users: Equinor (#1175)
 * e381653b Update README. (#1173) (#1176)
 * 5a917140 Update README and preview notice in CLA.
 * 521eb25a Validate ArchiveLocation artifacts (#1167)
 * 528e8f80 Add missing patch in namespace kustomization.yaml (#1170)
 * 0b41ca0a Add Preferred Networks to users in README.md (#1172)
 * 649d64d1 Add GitHub to users in README.md (#1151)
 * 864c7090 Update codegen for network config (#1168)
 * c3cc51be Support HDFS Artifact (#1159)
 * 8db00066 add support for hostNetwork & dnsPolicy config (#1161)
 * 149d176f Replace exponential retry with poll (#1166)
 * 31e5f63c Fix tests compilation error (#1157)
 * 6726d9a9 Fix failing TestAddGlobalArtifactToScope unit test
 * 4fd758c3 Add slack badge to README (#1164)
 * 3561bff7 Issue #1136 - Fix metadata for DAG with loops (#1149)
 * c7fec9d4 Reflect minio chart changes in documentation (#1147)
 * f6ce7833 add support for other archs (#1137)
 * cb538489 Fix issue where steps with exhausted retires would not complete (#1148)
 * e400b65c Fix global artifact overwriting in nested workflow (#1086)
 * 174eb20a Issue #1040 - Kill daemoned step if workflow consist of single daemoned step (#1144)
 * e078032e Issue #1132 - Fix panic in ttl controller (#1143)
 * e09d9ade Issue #1104 - Remove container wait timeout from 'argo logs --follow' (#1142)
 * 0f84e514 Allow owner reference to be set in submit util (#1120)
 * 3484099c Update generated swagger to fix verify-codegen (#1131)
 * 587ab1a0 Fix output artifact and parameter conflict (#1125)
 * 6bb3adbc Adding Quantibio in Who uses Argo (#1111)
 * 1ae3696c Install mime-support in argoexec to set proper mime types for S3 artifacts (resolves #1119)
 * 515a9005 add support for ppc64le and s390x (#1102)
 * 78142837 Remove docker_lib mount volume which is not needed anymore (#1115)
 * e59398ad Fix examples docs of parameters. (#1110)
 * ec20d94b Issue #1114 - Set FORCE_NAMESPACE_ISOLATION env variable in namespace install manifests (#1116)
 * 49c1fa4f Update docs with examples using the K8s REST API
 * bb8a6a58 Update ROADMAP.md
 * 46855dcd adding logo to be used by the OS Site (#1099)
 * 438330c3 #1081 added retry logic to s3 load and save function (#1082)
 * cb8b036b Initialize child node before marking phase. Fixes panic on invalid `When` (#1075)
 * 60b508dd Drop reference to removed `argo install` command. (#1074)
 * 62b24368 Fix typo in demo.md (#1089)
 * b5dfa021 Use relative links on README file (#1087)
 * 95b72f38 Update docs to outline bare minimum set of privileges for a workflow
 * d4ef6e94 Add new article and minor edits. (#1083)
 * afdac9bb Issue #740 - System level workflow parallelism limits & priorities (#1065)
 * a53a76e9 fix #1078 Azure AKS authentication issues (#1079)
 * 79b3e307 Fix string format arguments in workflow utilities. (#1070)
 * 76b14f54 Auto-complete workflow names (#1061)
 * f2914d63 Support nested steps workflow parallelism (#1046)
 * eb48c23a Raise not implemented error when artifact saving is unsupported (#1062)
 * 036969c0 Add Cratejoy to list of users (#1063)
 * a07bbe43 Adding SAP Hybris in Who uses Argo (#1064)
 * 7ef1cea6 Update dependencies to K8s v1.12 and client-go 9.0
 * 23d733ba Add namespace explicitly to pod metadata (#1059)
 * 79ed7665 Parameter and Argument names should support snake case (#1048)
 * 6e6c59f1 Submodules are dirty after checkout -- need to update (#1052)
 * f18716b7 Support for K8s API based Executor (#1010)
 * e297d195 Updated examples/README.md (#1051)
 * 19d6cee8 Updated ARTIFACT_REPO.md (#1049)
 * 0a928e93 Update installation manifests to use v2.2.1
 * 3b52b261 Fix linter warnings and update swagger
 * 7d0e77ba Update changelog and bump version to 2.2.1
 * b402e12f Issue #1033 - Workflow executor panic: workflows.argoproj.io/template workflows.argoproj.io/template not found in annotation file (#1034)
 * 3f2e986e fix typo in examples/README.md (#1025)
 * 9c5e056a Replace tabs with spaces (#1027)
 * 091f1407 Update README.md (#1030)
 * 159fe09c Fix format issues to resolve build errors (#1023)
 * 363bd97b Fix error in env syntax (#1014)
 * ae7bf0a5 Issue #1018 - Workflow controller should save information about archived logs in step outputs (#1019)
 * 15d006c5 Add example of workflow using imagePullSecrets (resolves #1013)
 * 2388294f Fix RBAC roles to include workflow delete for GC to work properly (resolves #1004)
 * 6f611cb9 Fix issue where resubmission of a terminated workflow creates a terminated workflow (issue #1011)
 * 4a7748f4 Disable Persistence in the demo example (#997)
 * 55ae0cb2 Fix example pod name (#1002)
 * c275e7ac Add imagePullPolicy config for executors (#995)
 * b1eed124 `tar -tf` will detect compressed tars correctly. (#998)
 * 03a7137c Add new organization using argo (#994)
 * 83884528 Update argoproj/pkg to trim leading/trailing whitespace in S3 credentials (resolves #981)
 * 978b4938 Add syntax highlighting for all YAML snippets and most shell snippets (#980)
 * 60d5dc11 Give control to decide whether or not to archive logs at a template level
 * 8fab73b1 Detect and indicate when container was OOMKilled
 * 47a9e556 Update config map doc with instructions to enable log archiving
 * 79dbbaa1 Add instructions to match git URL format to auth type in git example (issue #979)
 * 429f03f5 Add feature list to README.md. Tweaks to getting started.
 * 36fd1948 Update getting started guide with v2.2.0 instructions
 * af636ddd Update installation manifests to use v2.2.0
 * 7864ad36 Introduce `withSequence` to iterate a range of numbers in a loop (resolves #527)
 * 99e9977e Introduce `argo terminate` to terminate a workflow without deleting it (resolves #527)
 * f52c0450 Reorganize codebase to make CLI functionality available as a library
 * 311169f7 Fix issue where sidecars and daemons were not reliably killed (resolves #879)
 * 67ffb6eb Add a reason/message for Unschedulable Pending pods
 * 69c390f2 Support for workflow level timeouts (resolves #848)
 * f88732ec Update docs to use keyFormat field
 * 0df022e7 Rename keyPattern to keyFormat. Remove pending pod query during pod reconciliation
 * 75a9983b Fix potential panic in `argo watch`
 * 9cb46449 Add TTLSecondsAfterFinished field and controller to garbage collect completed workflows (resolves #911)
 * 7540714a Add ability to archive container logs to the artifact repository (resolves #454)
 * 11e57f4d Introduce archive strategies with ability to disable tar.gz archiving (resolves #784)
 * e180b547 Update CHANGELOG.md
 * 5670bf5a Introduce `argo watch` command to watch live workflows from terminal (resolves #969)
 * 57394361 Support additional container runtimes through kubelet executor (#952)
 * a9c84c97 Error workflows which hit k8s/etcd 1M resource size limit (resolves #913)
 * 67792eb8 Add parameter-file support (#966)
 * 841832a3 Aggregate workflow RBAC roles to built-in admin/edit/view clusterroles (resolves #960)
 * 35bb7093 Allow scaling of workflow and pod workers via controller CLI flags (resolves #962)
 * b479fa10 Improve workflow configmap documentation for keyPattern
 * f1802f91 Introduce `keyPattern` workflow config to enable flexibility in archive location path (issue #953)
 * a5648a96 Fix kubectl proxy link for argo-ui Service (#963)
 * 09f05912 Introduce Pending node state to highlight failures when start workflow pods
 * a3ff464f Speed up CI job
 * 88627e84 Update base images to debian:9.5-slim. Use stable metalinter
 * 753c5945 Update argo-ci-builder image with new dependencies
 * 674b61bb Remove unnecessary dependency on argo-cd and obsolete RBAC constants
 * 60658de0 Refactor linting/validation into standalone package. Support linting of .json files
 * f55d579a Detect and fail upon unknown fields during argo submit & lint (resolves #892)
 * edf6a574 Migrate to using argoproj.io/pkg packages
 * 5ee1e0c7 Update artifact config docs (#957)
 * faca49c0 Updated README
 * 936c6df7 Add table of content to examples (#956)
 * d2c03f67 Correct image used in install manifests
 * ec3b7be0 Remove CLI installer/uninstaller. Executor image configured via CLI argument (issue #928) Remove redundant/unused downward API metadata
 * 3a85e242 Rely on `git checkout` instead of go-git checkout for more reliable revision resolution
 * ecef0e3d Rename Prometheus metrics (#948)
 * b9cffe9c Issue #896 - Prometheus metrics and telemetry (#935)
 * 290dee52 Support parameter aggregation of map results in scripts
 * fc20f5d7 Fix errors when submodules are from different URL (#939)
 * b4f1a00a Add documentation about workflow variables
 * 4a242518 Update readme.md (#943)
 * a5baca60 Support referencing of global workflow artifacts (issue #900)
 * 9b5c8563 Support for sophisticated expressions in `when` conditionals (issue #860)
 * ecc0f027 Resolve revision added ability to specify shorthand revision and other things like HEAD~2 etc (#936)
 * 11024318 Support conditions with DAG tasks. Support aggregated outputs from scripts (issue #921)
 * d07c1d2f Support withItems/withParam and parameter aggregation with DAG templates (issue #801)
 * 94c195cb Bump VERSION to v2.2.0
 * 9168c59d Fix outbound node metadata with retry nodes causing disconnected nodes to be rendered in UI (issue #880)
 * c6ce48d0 Fix outbound node metadata issue with steps template causing incorrect edges to be rendered in UI
 * 520b33d5 Add ability to aggregate and reference output parameters expanded by loops (issue #861)
 * ece1eef8 Support submission of workflows as json, and from stdin (resolves #926)
 * 4c31d61d Add `argo delete --older` to delete completed workflows older than specified duration (resolves #930)
 * c87cd33c Update golang version to v1.10.3
 * 618b7eb8 Proper fix for assessing overall DAG phase. Add some DAG unit tests (resolves #885)
 * f223e5ad Fix issue where a DAG would fail even if retry was successful (resolves #885)
 * 143477f3 Start use of argoproj/pkg shared libraries
 * 1220d080 Update argo-cluster-role to work with OpenShift (resolves #922)
 * 4744f45a Added SSH clone and proper git clone using go-git (#919)
 * d657abf4 Regenerate code and address OpenAPI rule validation errors (resolves #923)
 * c5ec4cf6 Upgrade k8s dependencies to v1.10 (resolves #908)
 * ba8061ab Redundant verifyResolvedVariables check in controller precluded the ability to use {{ }} in other circumstances
 * 05a61449 Added link to community meetings (#912)
 * f33624d6 Add an example on how to submit and wait on a workflow
 * aeed7f9d Added new members
 * 288e4fc8 Added Argo Events link.
 * 3322506e Updated README
 * 3ce640a2 Issue #889 - Support retryStrategy for scripts (#890)
 * 91c6afb2 adding BlackRock as corporate contributor/user (#886)
 * c8667b5c Fix issue where `argo lint` required spec level arguments to be supplied
 * ed7dedde Update influx-ci example to choose a stable InfluxDB branch
 * 135813e1 Add datadog to the argo users (#882)
 * f1038948 Fix `make verify-codegen` build target when run in CI
 * 785f2cbd Update references to v2.1.1. Add better checks in release Makefile target
 * d65e1cd3 readme: add Interline Technologies to user list (#867)
 * c903168e Add documentation on global parameters (#871)

### Contributors

 * 0x1D-1983
 * Aaron Curtis
 * Aayush Rangwala
 * Adam Gilat
 * Adam Thornton
 * Aditya Sundaramurthy
 * Adrien Trouillaud
 * Aisuko
 * Akshay Chitneni
 * Alessandro Marrella
 * Alex Capras
 * Alex Collins
 * Alex Stein
 * Alexander Matyushentsev
 * Alexander Mikhailian
 * Alexander Zigelski
 * Alexey Volkov
 * Amim Knabben
 * Anastasia Satonina
 * Andrei Miulescu
 * Andrew Suderman
 * Anes Benmerzoug
 * Ang Gao
 * Anna Winkler
 * Antoine Dao
 * Antonio Macías Ojeda
 * Appréderisse Benjamin
 * Arghya Sadhu
 * Avi Weit
 * Bailey Hayes
 * Basanth Jenu H B
 * Bastian Echterhölter
 * Ben Wells
 * Ben Ye
 * Bikramdeep Singh
 * Boolman
 * Brandon Steinman
 * Brian Mericle
 * Byungjin Park (BJ)
 * CWen
 * Caden
 * Caglar Gulseni
 * Carlos Montemuino
 * Chen Zhiwei
 * Chris Chambers
 * Chris Hepner
 * Christian Muehlhaeuser
 * Clemens Lange
 * Cristian Pop
 * Daisuke Taniwaki
 * Dan Norris
 * Daniel Duvall
 * Daniel Moran
 * Daniel Sutton
 * David Bernard
 * David Gibbons
 * David Seapy
 * David Van Loon
 * Deepen Mehta
 * Derek Wang
 * Dineshmohan Rajaveeran
 * Divya Vavili
 * Douglas Lehr
 * Drew Dara-Abrams
 * Dustin Specker
 * EDGsheryl
 * Ed Lee
 * Edward Lee
 * Edwin Jacques
 * Ejiah
 * Elli Ludwigson
 * Elton
 * Elvis Jakupovic
 * Erik Parmann
 * Espen Finnesand
 * Fabio Rigato
 * Feynman Liang
 * Fischer Jemison
 * Florent Clairambault
 * Florian
 * Floris Van den Abeele
 * Francesco Murdaca
 * Fred Dubois
 * Gabriele Santomaggio
 * Galen Han
 * Grant Stephens
 * Greg Roodt
 * Guillaume Hormiere
 * Hamel Husain
 * Heikki Kesa
 * Hideto Inamura
 * Howie Benefiel
 * Huan-Cheng Chang
 * Hussein Awala
 * Ian Howell
 * Ids van der Molen
 * Igor Stepura
 * Ilias K
 * Ilias Katsakioris
 * Ilya Sotkov
 * InvictusMB
 * Ismail Alidzhikov
 * Jacob O'Farrell
 * Jaime
 * James Laverack
 * Jared Welch
 * Jean-Louis Queguiner
 * Jeff Uren
 * Jesse Suen
 * Jialu Zhu
 * Jie Zhang
 * Johannes 'fish' Ziemke
 * John Wass
 * Jonathan Steele
 * Jonathon Belotti
 * Jonny
 * Joshua Carp
 * Juan C. Muller
 * Juan C. Müller
 * Julian Fahrer
 * Julian Fischer
 * Julian Mazzitelli
 * Julien Balestra
 * Justen Walker
 * Kannappan Sirchabesan
 * Kaushik B
 * Konstantin Zadorozhny
 * Lennart Kindermann
 * Leonardo Luz
 * Lucas Theisen
 * Ludovic Cléroux
 * Lénaïc Huard
 * Marcin Karkocha
 * Marco Sanvido
 * Marek Čermák
 * Markus Lippert
 * Martin Suchanek
 * Matt Brant
 * Matt Campbell
 * Matt Hillsdon
 * Matthew Coleman
 * Matthew Magaldi
 * MengZeLee
 * Michael Crenshaw
 * Michael Ruoss
 * Michael Weibel
 * Michal Cwienczek
 * Mike Chau
 * Mike Seddon
 * Mingjie Tang
 * Miyamae Yuuya
 * Mostapha Sadeghipour Roudsari
 * Mukulikak
 * Naisisor
 * Naoto Migita
 * Naresh Kumar Amrutham
 * Nasrudin Bin Salim
 * Neutron Soutmun
 * Nick Groszewski
 * Nick Stott
 * Nicwalle
 * Niklas Hansson
 * Niklas Vest
 * Nirav Patel
 * Noah Hanjun Lee
 * Noj Vek
 * Noorain Panjwani
 * Nándor István Krácser
 * Oleg Borodai
 * Omer Kahani
 * Orion Delwaterman
 * Pablo Osinaga
 * Pascal VanDerSwalmen
 * Patryk Jeziorowski
 * Paul Brabban
 * Paul Brit
 * Pavel Kravchenko
 * Pavel Čižinský
 * Peng Li
 * Pengfei Zhao
 * Per Buer
 * Peter Salanki
 * Pierre Houssin
 * Pradip Caulagi
 * Pranaye Karnati
 * Praneet Chandra
 * Pratik Raj
 * Premkumar Masilamani
 * Rafael Rodrigues
 * Rafał Bigaj
 * Remington Breeze
 * Rick Avendaño
 * Rocio Montes
 * Romain Di Giorgio
 * Romain GUICHARD
 * Roman Galeev
 * Rush Tehrani
 * Saradhi Sreegiriraju
 * Saravanan Balasubramanian
 * Sascha Grunert
 * Sean Fern
 * Sebastian Ortan
 * Semjon Kopp
 * Shannon
 * Shubham Koli (FaultyCarry)
 * Simon Behar
 * Simon Frey
 * Snyk bot
 * Song Juchao
 * Stephen Steiner
 * StoneHuang
 * Takahiro Tsuruda
 * Takashi Abe
 * Takayuki Kasai
 * Tang Lee
 * Theodore Messinezis
 * Theodore Omtzigt
 * Tim Schrodi
 * Tobias Bradtke
 * Tom Wieczorek
 * Tomas Valasek
 * Tomáš Coufal
 * Trevor Foster
 * Tristan Colgate-McFarlane
 * Val Sichkovskyi
 * Vardan Manucharyan
 * Vincent Boulineau
 * Vincent Smith
 * Vlad Losev
 * Wei Yan
 * WeiYan
 * Weston Platter
 * William
 * William Reed
 * Wouter Remijn
 * Xianlu Bird
 * Xie.CS
 * Xin Wang
 * Youngjoon Lee
 * Yuan Tang
 * Yunhai Luo
 * Zach
 * Zach Aller
 * Zach Himsel
 * Zadjad Rezai
 * Zhipeng Wang
 * Ziyang Wang
 * aletepe
 * alex weidner
 * almariah
 * boundless-thread
 * candonov
 * commodus-sebastien
 * conanoc
 * descrepes
 * dgiebert
 * dherman
 * dmayle
 * dthomson25
 * duluong
 * fsiegmund
 * gerardaus
 * gerdos82
 * haibingzhao
 * hermanhobnob
 * hidekuro
 * houz
 * ianCambrio
 * ivancili
 * jacky
 * jdfalko
 * joe
 * juliusvonkohout
 * kshamajain99
 * lueenavarro
 * maguowei
 * mark9white
 * maryoush
 * mdvorakramboll
 * nglinh
 * omerfsen
 * sang
 * saranyaeu2987
 * sh-tatsuno
 * shahin
 * shibataka000
 * tczhao
 * tianfeiyu
 * tkilpela
 * tomgoren
 * tralexa
 * tunoat
 * vatine
 * vdinesh2461990
 * xubofei1983
 * yonirab
 * zhengchenyu
 * zhujl1991
 * モハメド

## v2.1.2 (2018-10-11)

 * b82ce5b0 Update version to 2.1.2
 * 01a1214e Issue #1033 - Workflow executor panic: workflows.argoproj.io/template workflows.argoproj.io/template not found in annotation file (#1034)

### Contributors

 * Alexander Matyushentsev

## v2.11.8 (2020-11-20)

 * 310e099f Update manifests to v2.11.8
 * 0f82b7f6 ci: Picked :add-env lines from master
 * 0a9e69b0 chore: Removed unused parameter
 * e8ba1ed8 feat(controller): Make MAX_OPERATION_TIME configurable. Close #4239 (#4562)
 * 66f2306b feat(controller): Allow whitespace in variable substitution. Fixes #4286 (#4310)

### Contributors

 * Alex Collins
 * Ids van der Molen

## v2.11.7 (2020-11-02)

 * bf3fec17 Update manifests to v2.11.7
 * 0f18ab1f fix: Assume controller is in UTC when calculating NextScheduledRuntime (#4417)
 * 6026ba5f fix: Ensure resourceDuration variables in metrics are always in seconds (#4411)
 * ca5adbc0 fix: Use correct template when processing metrics (#4399)
 * 0a0255a7 fix(ui): Reconnect to DAG. Fixes #4301 (#4378)
 * 8dd7d3ba fix: Use DeletionHandlingMetaNamespaceKeyFunc in cron controller (#4379)
 * 47f58008 fix(server): Download artifacts from UI. Fixes #4338 (#4350)
 * 0416aba5 fix(controller): Enqueue the front workflow if semaphore lock is available (#4380)
 * a2073d58 fix: Fix intstr nil dereference (#4376)
 * 89080cf8 fix(controller): Only warn if cron job missing. Fixes #4351 (#4352)
 * a4186dfd fix(executor): Increase pod patch backoff. Fixes #4339 (#4340)

### Contributors

 * Alex Collins
 * Saravanan Balasubramanian
 * Simon Behar

## v2.11.6 (2020-10-19)

 * 5eebce9a Update manifests to v2.11.6
 * 38a4a2e3 chore(controller): Refactor the CronWorkflow schedule logic with sync.Map (#4320)
 * 79e7a12a fix(executor): Remove IsTransientErr check for ExponentialBackoff. Fixes #4144 (#4149)

### Contributors

 * Alex Collins
 * Ang Gao
 * Saravanan Balasubramanian

## v2.11.5 (2020-10-15)

 * 076bf89c Update manifests to v2.11.5
 * 7919768d test: Fix test
 * b9d8c96b fix(controller): Patch rather than update cron workflows. (#4294)
 * 3d122426 fix: TestMutexInDAG failure in master (#4283)
 * 05519427 fix(controller): Synchronization lock didn't release on DAG call flow Fixes #4046 (#4263)
 * 74b905f6 Merge branch 'release-2.11' of https://github.com/argoproj/argo into release-2.11
 * ff2abd63 fix: Increase deafult number of CronWorkflow workers (#4215)

### Contributors

 * Alex Collins
 * Saravanan Balasubramanian
 * Simon Behar

## v2.11.4 (2020-10-14)

 * 571bff1f Update manifests to v2.11.4
 * 05a6078d fix(controller): Fix argo retry with PVCs. Fixes #4275 (#4277)
 * 08216ec7 fix(ui): Ignore missing nodes in DAG. Fixes #4232 (#4280)
 * 476ea70f fix(controller): Fix cron-workflow re-apply error. (#4278)
 * 448ae113 fix(controller): Check the correct object for Cronworkflow reapply error log (#4243)
 * e3dfd788 fix(ui): Revert bad part of commit (#4248)
 * 249e8329 fix(ui): Fix bugs with DAG view. Fixes #4232 & #4236 (#4241)

### Contributors

 * Alex Collins
 * Juan C. Müller
 * Saravanan Balasubramanian

## v2.11.3 (2020-10-07)

 * a00a8f14 Update manifests to v2.11.3
 * e48fe222 fixed merge conflict
 * 3f8ebfdd chore: Fix merge issues
 * 51068f72 fix(controller): Support int64 for param value. Fixes #4169 (#4202)

### Contributors

 * Alex Collins
 * Saravanan Balasubramanian

## v2.11.2 (2020-10-05)

 * 0dfeb8e5 Update manifests to v2.11.2
 * 5b69ab42 chore: Enahnce logging around pod failures (#4220)
 * 461a36a1 fix(controller): Apply Workflow default on normal workflow scenario Fixes #4208 (#4213)
 * 4b9cf5d2 fix(controller): reduce withItem/withParams memory usage. Fixes #3907 (#4207)
 * 8fea7bf6 Revert "Revert "chore: use build matrix and cache (#4111)""
 * efb20eea Revert "chore: use build matrix and cache (#4111)"
 * de1c9e52 refactor: Refactor Synchronization code (#4114)
 * 605d0895 fix: Ensure CronWorkflows are persisted once per operation (#4172)
 * 6f738db0 Revert "chore: Update Go module to argo/v2"
 * bcebdf00 chore: Update Go module to argo/v2

### Contributors

 * Alex Collins
 * Saravanan Balasubramanian
 * Simon Behar

## v2.11.1 (2020-09-29)

 * 13b51d56 Update manifests to v2.11.1
 * 3f88216e fix: Render full tree of onExit nodes in UI (#4109)
 * d6c2a57b fix: Fix unintended inf recursion (#4067)
 * 4fda60f4 fix: Tolerate malformed workflows when retrying (#4028)
 * 995d59cc fix: Changing DeletePropagation to background in TTL Controller and Argo CLI (#4133)
 * aaef0a28 fix(ui): Ignore referenced nodes that don't exist in UI. Fixes #4079 (#4099)
 * fedae45a fix(controller): Process workflows at least once every 20m (#4103)
 * 6de464e8 fix(server): argo-server-role to allow submitting cronworkflows from UI (#4112)
 * ce3b90e2 fix(controller): Treat annotation and conditions changes as significant (#4104)
 * bf98b977 fix(ui): No longer redirect to `undefined` namespace. See #4084 (#4115)
 * af60b37d fix(cli): Reinstate --gloglevel flag. Fixes #4093 (#4100)
 * 38b63008 chore: use build matrix and cache (#4111)
 * 2cd6a967 fix(server): Optional timestamp inclusion when retrieving workflow logs. Closes #4033 (#4075)
 * 2f7c4035 fix(controller): Correct the order merging the fields in WorkflowTemplateRef scenario. Fixes #4044 (#4063)
 * f8e750de Update manifests to v2.11.0
 * c06db575 fix(ui): Tiny modal DAG tweaks. Fixes #4039 (#4043)

### Contributors

 * Alex Collins
 * Markus Lippert
 * Saravanan Balasubramanian
 * Simon Behar
 * Tomáš Coufal
 * ivancili

## v2.11.0-rc3 (2020-09-14)

 * 1b4cf3f1 Update manifests to v2.11.0-rc3
 * e2594eca fix: Fix children is not defined error (#3950)
 * 2ed8025e fix: Fix UI selection issues (#3928)
 * 8dc0e94e fix: Create global scope before workflow-level realtime metrics (#3979)
 * cdeabab7 fix(controller): Script Output didn't set if template has RetryStrategy (#4002)
 * 9c83fac8 fix(ui): Do not save undefined namespace. Fixes #4019 (#4021)
 * 7fd2ecb1 fix(ui): Correctly show pod events. Fixes #4016 (#4018)
 * 11242c8b fix(ui): Allow you to view timeline tab. Fixes #4005 (#4006)
 * 3770f618 fix(ui): Report errors when uploading files. Fixes #3994 (#3995)
 * 0fed28ce fix: Custom metrics are not recorded for DAG tasks Fixes #3872 (#3886)
 * 9146636e feat(ui): Introduce modal DAG renderer. Fixes: #3595 (#3967)
 * 4b7a4694 fix(controller): Revert `resubmitPendingPods` mistake. Fixes #4001 (#4004)
 * 49752fb5 fix(controller): Revert parameter value to `*string`. Fixes #3960 (#3963)
 * ddf850b1 fix: Consider WorkflowTemplate metadata during validation (#3988)
 * a8ba447e fix(server): Remove XSS vulnerability. Fixes #3942 (#3975)

### Contributors

 * Alex Collins
 * Saravanan Balasubramanian
 * Simon Behar

## v2.11.0-rc2 (2020-09-09)

 * f930c029 Update manifests to v2.11.0-rc2
 * 21cceb0f build: Allow build with older `docker`. Fixes #3977
 * b6890adb fix(cli): Allow `argo version` without KUBECONFIG. Fixes #3943 (#3945)
 * 354733e7 fix(swagger): Correct item type. Fixes #3926 (#3932)
 * 1e461766 fix(server): Adds missing webhook permissions. Fixes #3927 (#3929)
 * 88486192 feat: Step and Task Level Global Timeout (#3686)

### Contributors

 * Alex Collins
 * Saravanan Balasubramanian

## v2.11.0-rc1 (2020-09-01)


### Contributors


## v2.11.0 (2020-09-17)

 * f8e750de Update manifests to v2.11.0
 * c06db575 fix(ui): Tiny modal DAG tweaks. Fixes #4039 (#4043)
 * 1b4cf3f1 Update manifests to v2.11.0-rc3
 * e2594eca fix: Fix children is not defined error (#3950)
 * 2ed8025e fix: Fix UI selection issues (#3928)
 * 8dc0e94e fix: Create global scope before workflow-level realtime metrics (#3979)
 * cdeabab7 fix(controller): Script Output didn't set if template has RetryStrategy (#4002)
 * 9c83fac8 fix(ui): Do not save undefined namespace. Fixes #4019 (#4021)
 * 7fd2ecb1 fix(ui): Correctly show pod events. Fixes #4016 (#4018)
 * 11242c8b fix(ui): Allow you to view timeline tab. Fixes #4005 (#4006)
 * 3770f618 fix(ui): Report errors when uploading files. Fixes #3994 (#3995)
 * 0fed28ce fix: Custom metrics are not recorded for DAG tasks Fixes #3872 (#3886)
 * 9146636e feat(ui): Introduce modal DAG renderer. Fixes: #3595 (#3967)
 * 4b7a4694 fix(controller): Revert `resubmitPendingPods` mistake. Fixes #4001 (#4004)
 * 49752fb5 fix(controller): Revert parameter value to `*string`. Fixes #3960 (#3963)
 * ddf850b1 fix: Consider WorkflowTemplate metadata during validation (#3988)
 * a8ba447e fix(server): Remove XSS vulnerability. Fixes #3942 (#3975)
 * f930c029 Update manifests to v2.11.0-rc2
 * 21cceb0f build: Allow build with older `docker`. Fixes #3977
 * b6890adb fix(cli): Allow `argo version` without KUBECONFIG. Fixes #3943 (#3945)
 * 354733e7 fix(swagger): Correct item type. Fixes #3926 (#3932)
 * 1e461766 fix(server): Adds missing webhook permissions. Fixes #3927 (#3929)
 * 88486192 feat: Step and Task Level Global Timeout (#3686)
 * f446f735 Update manifests to v2.11.0-rc1
 * de2185c8 feat(controller): Set retry factor to 2. Closes #3911 (#3919)
 * be91d762 fix: Workflow should fail on Pod failure before container starts Fixes #3879 (#3890)
 * c4c80069 test: Fix TestRetryOmit and TestStopBehavior (#3910)
 * 650869fd feat(server): Display events involved in the workflow. Closes #3673 (#3726)
 * 5b5d2359 fix(controller): Cron re-apply update (#3883)
 * fd3fca80 feat(artifacts): retrieve subpath from unarchived ref artifact. Closes #3061 (#3063)
 * 6a452ccd test: Fix flaky e2e tests (#3909)
 * 6e82bf38 feat(controller): Emit events for malformed cron workflows. See #3881 (#3889)
 * f04bdd6a Update workflow-controller-configmap.yaml (#3901)
 * bb79e3f5 fix(executor): Replace default retry in executor with an increased value retryer (#3891)
 * b681c113 fix(ui): use absolute URL to redirect from autocomplete list. Closes #3903 (#3906)
 * 712c77f5 chore(users): Add Fynd Trak to the list of Users (#3900)
 * d55402db ci: Fix broken Multiplatform builds (#3908)
 * 9681a4e2 fix(ui): Improve error recovery. Fixes #3867 (#3869)
 * b926f8c0 chore: Remove unused imports (#3892)
 * 4c18a06b feat(controller): Always retry when `IsTransientErr` to tolerate transient errors. Fixes #3217 (#3853)
 * 0cf7709f fix(controller): Failure tolerant workflow archiving and offloading. Fixes #3786 and #3837 (#3787)
 * 359ee8db fix: Corrects CRD and Swagger types. Fixes #3578 (#3809)
 * 58ac52b8 chore(ui): correct a typo (#3876)
 * dae0f2df feat(controller): Do not try to create pods we know exists to prevent `exceeded quota` errors. Fixes #3791 (#3851)
 * 9781a1de ci: Create manifest from images again (#3871)
 * c6b51362 test: E2E test refactoring (#3849)
 * 04898fee chore: Added unittest for PVC exceed quota Closes #3561 (#3860)
 * 4e42208c ci: Changed tagging and amend multi-arch manifest. (#3854)
 * c352f69d chore: Reduce the 2x workflow save on Semaphore scenario (#3846)
 * a24bc944 feat(controller): Mutexes. Closes #2677 (#3631)
 * a821c6d4 ci: Fix build by providing clean repo inside Docker (#3848)
 * 99fe11a7 feat: Show next scheduled cron run in UI/CLI (#3847)
 * 6aaceeb9 fix: Treat collapsed nodes as their siblings (#3808)
 * 7f5acd6f docs: Add 23mofang to USERS.md
 * 10cb447a docs: Update example README for duration (#3844)
 * 1678e58c ci: Remove external build dependency (#3831)
 * 743ec536 fix(ui): crash when workflow node has no memoization info (#3839)
 * a2f54da1 fix(docs): Amend link to the Workflow CRD (#3828)
 * ca8ab468 fix: Carry over ownerReferences from resubmitted workflow. Fixes #3818 (#3820)
 * da43086a fix(docs): Add Entrypoint Cron Backfill example  Fixes #3807 (#3814)
 * ed749a55 test: Skip TestStopBehavior and TestRetryOmit (#3822)
 * 9292ae1e ci: static files not being built with Homebrew and dirty binary. Fixes #3769 (#3801)
 * c840adb2 docs: memory base amount denominator documentation
 * 8e1a3db5 feat(ui): add node memoization information to node summary view (#3741)
 * 9de49e2e ci: Change workflow for pushing images. Fixes #2080
 * d235c7d5 fix: Consider all children of TaskGroups in DAGs (#3740)
 * 3540d152 Add SYS_PTRACE to ease the setup of non-root deployments with PNS executor. (#3785)
 * 2f654971 chore: add New Relic to USERS.md (#3810)
 * ce5da590 docs: Add section on CronWorkflow crash recovery (#3804)
 * 0ca83924 feat: Github Workflow multi arch. Fixes #2080 (#3744)
 * bee0e040 docs: Remove confusing namespace (#3772)
 * 7ad6eb84 fix(ui): Remove outdated download links. Fixes #3762 (#3783)
 * 22636782 fix(ui): Correctly load and store namespace. Fixes #3773 and #3775 (#3778)
 * a9577ab9 test: Increase cron test timeout to 7m (#3799)
 * ed90d403 fix(controller): Support exit handler on workflow templates.  Fixes #3737 (#3782)
 * dc75ee81 test: Simplify E2E test tear-down (#3749)
 * 821e40a2 build: Retry downloading Kustomize (#3792)
 * f15a8f77 fix: workflow template ref does not work in other namespace (#3795)
 * ef44a03d fix: Increase the requeue duration on checkForbiddenErrorAndResubmitAllowed (#3794)
 * 0125ab53 fix(server): Trucate creator label at 63 chars. Fixes #3756 (#3758)
 * a38101f4 feat(ui): Sign-post IDE set-up. Closes #3720 (#3723)
 * 21dc23db chore: Format test code (#3777)
 * ee910b55 feat(server): Emit audit events for workflow event binding errors (#3704)
 * e9b29e8c fix: TestWorkflowLevelSemaphore flakiness (#3764)
 * fadd6d82 fix: Fix workflow onExit nodes not being displayed in UI (#3765)
 * df06e901 docs: Correct typo in `--instanceid`
 * 82a671c0 build: Lint e2e test files (#3752)
 * 513675bc fix(executor): Add retry on pods watch to handle timeout. (#3675)
 * e35a86ff feat: Allow parametrizable int fields (#3610)
 * da115f9d fix(controller): Tolerate malformed resources. Fixes #3677 (#3680)
 * 407f9e63 docs: Remove misleading argument in workflow template dag examples. (#3735) (#3736)
 * f8053ae3 feat(operator): Add scope params for step startedAt and finishedAt (#3724)
 * 54c2134f fix: Couldn't Terminate/Stop the ResourceTemplate Workflow (#3679)
 * 12ddc1f6 fix: Argo linting does not respect namespace of declared resource (#3671)
 * acfda260 feat(controller): controller logs to be structured #2308 (#3727)
 * cc2e42a6 fix(controller): Tolerate PDB delete race. Fixes #3706 (#3717)
 * 5eda8b86 fix: Ensure target task's onExit handlers are run (#3716)
 * 811a4419 docs(windows): Add note about artifacts on windows (#3714)
 * 5e5865fb fix: Ingress docs (#3713)
 * eeb3c9d1 fix: Fix bug with 'argo delete --older' (#3699)
 * 6134a565 chore: Introduce convenience methods for intstr. (#3702)
 * 7aa536ed feat: Upgrade Minio v7 with support IRSA (#3700)
 * 4065f265 docs: Correct version. Fixes #3697 (#3701)
 * 71d61281 feat(server): Trigger workflows from webhooks. Closes #2667  (#3488)
 * a5d995dc fix(controller): Adds ALL_POD_CHANGES_SIGNIFICANT (#3689)
 * 9f00cdc9 fix: Fixed workflow queue duration if PVC creation is forbidden (#3691)
 * 2baaf914 chore: Update issue templates (#3681)
 * 41ebbe8e fix: Re-introduce 1 second sleep to reconcile informer (#3684)
 * 6e3c5bef feat(ui): Make UI errors recoverable. Fixes #3666 (#3674)
 * 27fea1bb chore(ui): Add label to 'from' section in Workflow Drawer (#3685)
 * 32d6f752 feat(ui): Add links to wft, cwf, or cwft to workflow list and details. Closes #3621 (#3662)
 * 1c95a985 fix: Fix collapsible nodes rendering (#3669)
 * 87b62bbb build: Use http in dev server (#3670)
 * dbb39368 feat: Add submit options to 'argo cron create' (#3660)
 * 2b6db45b fix(controller): Fix nested maps. Fixes #3653 (#3661)
 * 3f293a4d fix: interface{} values should be expanded with '%v' (#3659)
 * f08ab972 docs: Fix type in default-workflow-specs.md (#3654)
 * a8f4da00 fix(server): Report v1.Status errors. Fixes #3608 (#3652)
 * a3a4ea0a fix: Avoid overriding the Workflow parameter when it is merging with WorkflowTemplate parameter (#3651)
 * 9ce1d824 fix: Enforce metric Help must be the same for each metric Name (#3613)
 * 4eca0481 docs: Update link to examples so works in raw github.com view (#3623)
 * f77780f5 fix(controller): Carry-over labels for re-submitted workflows. Fixes #3622 (#3638)
 * 3f3a4c91 docs: Add comment to config map about SSO auth-mode (#3634)
 * d9090c99 build: Disable TLS for dev mode. Fixes #3617 (#3618)
 * bcc6e1f7 fix: Fixed flaky unit test TestFailSuspendedAndPendingNodesAfterDeadline (#3640)
 * 8f70d224 fix: Don't panic on invalid template creation (#3643)
 * 5b0210dc fix: Simplify the WorkflowTemplateRef field validation to support all fields in WorkflowSpec except `Templates` (#3632)
 * 2375878a fix: Fix 'malformed request: field selector' error (#3636)
 * 87ea54c9 docs: Add documentation for configuring OSS artifact storage (#3639)
 * 861afd36 docs: Correct indentation for codeblocks within bullet-points for "workflow-templates" (#3627)
 * 0f37e81a fix: DAG level Output Artifacts on K8S and Kubelet executor (#3624)
 * a89261bf build(cli)!: Zip binaries binaries. Closes #3576 (#3614)
 * 7f844473 fix(controller): Panic when outputs in a cache entry are nil (#3615)
 * 86f03a3f fix(controller): Treat TooManyError same as Forbidden (i.e. try again). Fixes #3606 (#3607)
 * 2e299df3 build: Increase timeout (#3616)
 * e0a4f13d fix(server): Re-establish watch on v1.Status errors. Fixes #3608 (#3609)
 * cdbb5711 docs: Memoization Documentation (#3598)
 * 7abead2a docs: fix typo - replace "workfow" with "workflow" (#3612)
 * f7be20c1 fix: Fix panic and provide better error message on watch endpoint (#3605)
 * 491f4f74 fix: Argo Workflows does not honour global timeout if step/pod is not able to schedule (#3581)
 * 5d8f85d5 feat(ui): Enhanced workflow submission. Closes #3498 (#3580)
 * a4a26414 build: Initialize npm before installing swagger-markdown (#3602)
 * ad3441dc feat: Add 'argo node set' command (#3277)
 * a43bf129 docs: Migrate to homebrew-core (#3567) (#3568)
 * 17b46bdb fix(controller): Fix bug in util/RecoverWorkflowNameFromSelectorString. Add error handling (#3596)
 * c968877c docs: Document ingress set-up. Closes #3080 (#3592)
 * 8b6e43f6 fix(ui): Fix multiple UI issues (#3573)
 * cdc935ae feat(cli): Support deleting resubmitted workflows (#3554)
 * 1b757ea9 feat(ui): Change default language for Resource Editor to YAML and store preference in localStorage. Fixes #3543 (#3560)
 * c583bc04 fix(server): Ignore not-JWT server tokens. Fixes #3562 (#3579)
 * 5afbc131 fix(controller): Do not panic on nil output value. Fixes #3505 (#3509)
 * c409624b docs: Synchronization documentation (#3537)
 * 0bca0769 docs: Workflow of workflows pattern (#3536)
 * 827106de fix: Skip TestStorageQuotaLimit (#3566)
 * 13b1d3c1 feat(controller): Step level memoization. Closes #944 (#3356)
 * 96e520eb fix: Exceeding quota with volumeClaimTemplates (#3490)
 * 144c9b65 fix(ui): cannot push to nil when filtering by label (#3555)
 * 7e4a7808 feat: Collapse children in UI Workflow viewer (#3526)
 * 7536982a fix: Fix flakey TestRetryOmitted (#3552)
 * 05d573d7 docs: Change formatting to put content into code block (#3553)
 * dcee3484 fix: Fix links in fields doc (#3539)
 * fb67c1be Fix issue #3546 (#3547)
 * d07a0e74 ci: Make builds marginally faster. Fixes #3515 (#3519)
 * 4cb6aa04 chore: Enable no-response bot (#3510)
 * 31afa92a fix(artifacts): support optional input artifacts, Fixes #3491 (#3512)
 * 977beb46 fix: Fix when retrying Workflows with Omitted nodes (#3528)
 * ab4ef5c5 fix: Panic on CLI Watch command (#3532)
 * b901b279 fix(controller): Backoff exponent is off by one. Fixes #3513 (#3514)
 * 49ef5c0f fix: String interpreted as boolean in labels (#3518)
 * 19e700a3 fix(cli): Check mutual exclusivity for argo CLI flags (#3493)
 * 7d45ff7f fix: Panic on releaseAllWorkflowLocks if Object is not Unstructured type (#3504)
 * 1b68a5a1 fix: ui/package.json & ui/yarn.lock to reduce vulnerabilities (#3501)
 * 7f262fd8 fix(cli)!: Enable CLI to work without kube config. Closes #3383, #2793 (#3385)
 * 2976e7ac build: Clear cmd docs before generating them (#3499)
 * 27528ba3 feat: Support completions for more resources (#3494)
 * 5bd2ad7a fix: Merge WorkflowTemplateRef with defaults workflow spec (#3480)
 * e244337b chore: Added examples for exit handler for step and dag level (#3495)
 * bcb32547 build: Use `git rev-parse` to accomodate older gits (#3497)
 * 3eb6e2f9 docs: Add link to GitHub Actions in the badge (#3492)
 * 69179e72 fix: link to server auth mode docs, adds Tulip as official user (#3486)
 * 7a8e2b34 docs: Add comments to NodePhase definition. Closes #1117. (#3467)
 * 24d1e529 build: Simplify builds (#3478)
 * acf56f9f feat(server): Label workflows with creator. Closes #2437 (#3440)
 * 3b8ac065 fix: Pass resolved arguments to onExit handler (#3477)
 * 58097a9e docs: Add controller-level metrics (#3464)
 * f6f1844b feat: Attempt to resolve nested tags (#3339)
 * 48e15d6f feat(cli): List only resubmitted workflows option (#3357)
 * 25e9c0cd docs, quick-start. Use http, not https for link (#3476)
 * 7a2d7642 fix: Metric emission with retryStrategy (#3470)
 * f5876e04 test(controller): Ensure resubmitted workflows have correct labels (#3473)
 * aa92ec03 fix(controller): Correct fail workflow when pod is deleted with --force. Fixes #3097 (#3469)
 * a1945d63 fix(controller): Respect the volumes of a workflowTemplateRef. Fixes … (#3451)
 * 847ba530 test(controller): Add memoization tests. See #3214 (#3455) (#3466)
 * f5183aed docs: Fix CLI docs (#3465)
 * 1e42813a test(controller): Add memoization tests. See #3214 (#3455)
 * abe768c4 feat(cli): Allow to view previously terminated container logs (#3423)
 * 7581025f fix: Allow ints for sequence start/end/count. Fixes #3420 (#3425)
 * b82f900a Fixed typos (#3456)
 * 23760119 feat: Workflow Semaphore Support (#3141)
 * 81cba832 feat: Support WorkflowMetadata in WorkflowTemplate and ClusterWorkflowTemplate (#3364)
 * 568c032b chore: update aws-sdk-go version (#3376)
 * bd27d9f3 chore: Upgrade node-sass (#3450)
 * b1e601e5 docs: typo in argo stop --help (#3439)
 * 308c7083 fix(controller): Prevent panic on nil node. Fixes #3436 (#3437)
 * 8ab06f53 feat(controller): Add log message count as metrics. (#3362)
 * 5d0c436d chore: Fix GitHub Actions Docker Image build  (#3442)
 * e54b4ab5 docs: Add Sohu as official Argo user (#3430)
 * ee6c8760 fix: Ensure task dependencies run after onExit handler is fulfilled (#3435)
 * 6dc04b39 chore: Use GitHub Actions to build Docker Images to allow publishing Windows Images (#3291)
 * 05b3590b feat(controller): Add support for Docker workflow executor for Windows nodes (#3301)
 * 676868f3 fix(docs): Update kubectl proxy URL (#3433)
 * 3507c3e6 docs: Make https://argoproj.github.io/argo/  (#3369)
 * 733e95f7 fix: Add struct-wide RWMutext to metrics (#3421)
 * 0463f241 fix: Use a unique queue to visit nodes (#3418)
 * eddcac63 fix: Script steps fail with exceededQuota (#3407)
 * c631a545 feat(ui): Add Swagger UI (#3358)
 * 910f636d fix: No panic on watch. Fixes #3411 (#3426)
 * b4da1bcc fix(sso): Remove unused `groups` claim. Fixes #3411 (#3427)
 * 330d4a0a fix: panic on wait command if event is null (#3424)
 * 7c439424 docs: Include timezone name reference (#3414)
 * 03cbb8cf fix(ui): Render DAG with exit node (#3408)
 * 3d50f985 feat: Expose certain queue metrics (#3371)
 * c7b35e05 fix: Ensure non-leaf DAG tasks have their onExit handler's run (#3403)
 * 70111600 fix: Fix concurrency issues with metrics (#3401)
 * d307f96f docs: Update config example to include useSDKCreds (#3398)
 * 637d50bc chore: maybe -> may be (#3392)
 * e70a8863 chore: Added CWFT WorkflowTemplateRef example (#3386)
 * bc4faf5f fix: Fix bug parsing parmeters (#3372)
 * 4934ad22 fix: Running pods are garaged in PodGC onSuccess
 * 0541cfda chore(ui): Remove unused interfaces for artifacts (#3377)
 * 20382cab docs: Fix incorrect example of global parameter (#3375)
 * 1db93c06 perf: Optimize time-based filtering on large number of workflows (#3340)
 * 2ab9495f fix: Don't double-count metric events (#3350)
 * 7bd3e720 fix(ui): Confirmation of workflow actions (#3370)
 * 488790b2 Wellcome is using Argo in our Data Labs division (#3365)
 * 63e71192 chore: Remove unused code (#3367)
 * a64ceb03 build: Enable Stale Bot (#3363)
 * e4b08abb fix(server): Remove `context cancelled` error. Fixes #3073 (#3359)
 * 74ba5162 fix: Fix UI bug in DAGs (#3368)
 * 5e60decf feat(crds)!: Adds CRD generation and enhanced UI resource editor. Closes #859 (#3075)
 * c2347f35 chore: Simplify deps by removing YAML (#3353)
 * 1323f9f4 test: Add e2e tags (#3354)
 * 731a1b4a fix(controller): Allow events to be sent to non-argo namespace. Fixes #3342 (#3345)
 * 916e0db2 Adding InVision to Users (#3352)
 * 6caf10fa fix: Ensure child pods respect maxDuration (#3280)
 * 8f4945f5 docs: Field fix (ParallelSteps -> WorkflowStep) (#3338)
 * 2b4b7340 fix: Remove broken SSO from quick-starts (#3327)
 * 26570fd5 fix(controller)!: Support nested items. Fixes #3288 (#3290)
 * c3d85716 chore: Avoid variable name collision with imported package name (#3335)
 * ca822af0 build: Fix path to go-to-protobuf binary (#3308)
 * 769a964f feat(controller): Label workflows with their source workflow template (#3328)
 * 0785be24 fix(ui): runtime error from null savedOptions props (#3330)
 * 200be0e1 feat: Save pagination limit and selected phases/labels to local storage (#3322)
 * b5ed90fe feat: Allow to change priority when resubmitting workflows (#3293)
 * 60c86c84 fix(ui): Compiler error from workflows toolbar (#3317)
 * 3fe6ecc4 docs: Document access token creation and usage (#3316)
 * ab3c081e docs: Rename Ant Financial to Ant Group (#3304)
 * baad42ea feat(ui): Add ability to select multiple workflows from list and perform actions on them. Fixes #3185 (#3234)
 * b6118939 fix(controller): Fix panic logging. (#3315)
 * 633ea71e build: Pin `goimports` to working version (#3311)
 * 436c1259 ci: Remove CircleCI (#3302)
 * 8e340229 build: Remove generated Swagger files. (#3297)
 * e021d7c5 Clean up unused constants (#3298)
 * 48d86f03 build: Upload E2E diagnostics after failure (#3294)
 * 8b12f433 feat(cli): Add --logs to `argo [submit|resubmit|retry]. Closes #3183 (#3279)
 * 07b450e8 fix: Reapply Update if CronWorkflow resource changed (#3272)
 * 8af01491 docs: ArchiveLabelSelector document (#3284)
 * 38c908a2 docs: Add example for handling large output resutls (#3276)
 * d44d264c Fixes validation of overridden ref template parameters. (#3286)
 * 62e54fb6 fix: Fix delete --complete (#3278)
 * a3c379bb docs: Updated WorkflowTemplateRef  on WFT and CWFT (#3137)
 * 824de95b fix(git): Fixes Git when using auth or fetch. Fixes #2343 (#3248)
 * 018fcc23 Update releasing.md (#3283)
 * acee573b docs: Update CI Badges (#3282)
 * 9eb182c0 build: Allow to change k8s namespace for installation (#3281)
 * 2bcfafb5 fix: Add {{workflow.status}} to workflow-metrics (#3271)
 * e6aab605 fix(jqFilter)!: remove extra quotes around output parameter value (#3251)
 * f4580163 fix(ui): Allow render of templates without entrypoint. Fixes #2891 (#3274)
 * f30c05c7 build: Add warning to ensure 'v' is present on release versions (#3273)
 * d1cb1992 fixed archiveLabelSelector nil (#3270)
 * c7e4c180 fix(ui): Update workflow drawer with new duration format (#3256)
 * f2381a54 fix(controller): More structured logging. Fixes #3260 (#3262)
 * acba084a fix: Avoid unnecessary nil check for annotations of resubmitted workflow (#3268)
 * 55e13705 feat: Append previous workflow name as label to resubmitted workflow (#3261)
 * 2dae7244 feat: Add mode to require Workflows to use workflowTemplateRef (#3149)
 * 56694abe Fixed onexit on workflowtempalteRef (#3263)
 * 54dd72c2 update mysql yaml port (#3258)
 * fb502632 feat: Configure ArchiveLabelSelector for Workflow Archive (#3249)
 * 5467c899 fix(controller): set pod finish timestamp when it is deleted (#3230)
 * 04bc5492 build: Disable Circle CI and Sonar (#3253)
 * 23ca07a7 chore: Covered steps.<STEPNAME>.outputs.parameters in variables document (#3245)
 * 4bd33c6c chore(cli): Add examples of @latest alias for relevant commands. Fixes #3225 (#3242)
 * 17108df1 fix: Ensure subscription is closed in log viewer (#3247)
 * 495dc89b docs: Correct available fields in {{workflow.failures}} (#3238)
 * 4db1c4c8 fix: Support the TTLStrategy for WorkflowTemplateRef (#3239)
 * 47f50693 feat(logging): Made more controller err/warn logging structured (#3240)
 * c25e2880 build: Migrate to Github Actions (#3233)
 * ef159f9a feat: Tick CLI Workflow watch even if there are no new events (#3219)
 * ff1627b7 fix(events): Adds config flag. Reduce number of dupe events emitted. (#3205)
 * eae8f681 feat: Validate CronWorkflows before execution (#3223)
 * 4470a8a2 fix(ui/server): Fix broken label filter functionality on UI due to bug on server. Fix #3226 (#3228)
 * e5e6456b feat(cli): Add --latest flag for argo get command as per #3128 (#3179)
 * 34608594 fix(ui): Correctly update workflow list when workflow are modified/deleted (#3220)
 * a7d8546c feat(controller): Improve throughput of many workflows. Fixes #2908 (#2921)
 * a37d0a72 build: Change "DB=..." to "PROFILE=..." (#3216)
 * 15885d3e feat(sso): Allow reading SSO clientID from a secret. (#3207)
 * 723e9d5f fix: Ensrue image name is present in containers (#3215)
 * 0ee5e112 feat: Only process significant pod changes (#3181)
 * c89a81f3 feat: Add '--schedule' flag to 'argo cron create' (#3199)
 * 591f649a refactor: Refactor assesDAGPhase logic (#3035)
 * 285eda6b chore: Remove unused pod in addArchiveLocation() (#3200)
 * 8e1d56cb feat(controller): Add default name for artifact repository ref. (#3060)
 * f1cdba18 feat(controller): Add `--qps` and `--burst` flags to controller (#3180)
 * b86949f0 fix: Ensure stable desc/hash for metrics (#3196)
 * e26d2f08 docs: Update Getting Started (#3099)
 * 47bfea5d docs: Add Graviti as official Argo user (#3187)
 * 04c77f49 fix(server): Allow field selection for workflow-event endpoint (fixes #3163) (#3165)
 * 0c38e66e chore: Update Community Meeting link and specify Go@v1.13 (#3178)
 * 81846d41 build: Only check Dex in hosts file when SSO is enabled (#3177)
 * a130d488 feat(ui): Add drawer with more details for each workflow in Workflow List (#3151)
 * fa84e203 fix: Do not use alphabetical order if index exists (#3174)
 * 138af597 fix(cli): Sort expanded nodes by index. Closes #3145 (#3146)
 * a9ec4d08 docs: Fix api swagger file path in docs (#3167)
 * c42e4d3a feat(metrics): Add node-level resources duration as Argo variable for metrics. Closes #3110 (#3161)
 * e36fe66e docs: Add instructions on using Minikube as an alternative to K3D (#3162)
 * edfa5b93 feat(metrics): Report controller error counters via metrics. Closes #3034 (#3144)
 * 8831e4ea feat(argo-server): Add support for SSO. See #1813 (#2745)
 * b62184c2 feat(cli): More `argo list` and `argo delete` options (#3117)
 * c6565d7c fix(controller): Maybe bug with nil woc.wfSpec. Fixes #3121 (#3160)
 * 06ca71d7 build: Fix path to staticfiles and goreman binaries (#3159)
 * cad84cab chore: Remove unused nodeType in initializeNodeOrMarkError() (#3153)
 * be425513 chore: Master needs lint (#3152)
 * 70b56f25 enhancement(ui): Add workflow labels column to workflow list. Fixes #2782 (#3143)
 * 3318c115 chore: Move default metrics server port/path to consts (#3135)
 * a0062adf feat(ui): Add Alibaba Cloud OSS related models in UI (#3140)
 * 1469991c fix: Update container delete grace period to match Kubernetes default (#3064)
 * df725bbd fix(ui): Input artifacts labelled in UI. Fixes #3098 (#3131)
 * c0d59cc2 feat: Persist DAG rendering options in local storage (#3126)
 * 8715050b fix(ui): Fix label error (#3130)
 * 1814ea2e fix(item): Support ItemValue.Type == List. Fixes #2660 (#3129)
 * 12b72546 fix: Panic on invalid WorkflowTemplateRef (#3127)
 * 09092147 fix(ui): Display error message instead of DAG when DAG cannot be rendered. Fixes #3091 (#3125)
 * 2d9a74de docs: Document cost optimizations. Fixes #1139 (#2972)
 * 69c9e5f0 fix: Remove unnecessary panic (#3123)
 * 2f3aca89 add AppDirect to the list of users (#3124)
 * 257355e4 feat: Add 'submit --from' to CronWorkflow and WorkflowTemplate in UI. Closes #3112 (#3116)
 * 6e5dd2e1 Add Alibaba OSS to the list of supported artifacts (#3108)
 * 1967b45b support sso (#3079)
 * 9229165f feat(ui): Add cost optimisation nudges. (#3089)
 * e88124db fix(controller): Do not panic of woc.orig in not hydrated. Fixes #3118 (#3119)
 * 132b947a fix: Differentiate between Fulfilled and Completed (#3083)
 * a93968ff docs: Document how to backfill a cron workflow (#3094)
 * 4de99746 feat: Added Label selector and Field selector in Argo list  (#3088)
 * 6229353b chore: goimports (#3107)
 * 8491e00f docs: Add link to USERS.md in PR template (#3086)
 * bb2ce9f7 fix: Graceful error handling of malformatted log lines in watch (#3071)
 * 4fd27c31 build(swagger): Fix Swagger build problems (#3084)
 * e4e0dfb6 test: fix TestContinueOnFailDag (#3101)
 * fa69c1bb feat: Add CronWorkflowConditions to report errors (#3055)
 * 50ad3cec adds millisecond-level timestamps to argoexec (#2950)
 * 6464bd19 fix(controller): Implement offloading for workflow updates that are re-applied. Fixes #2856 (#2941)
 * 6c369e61 chore: Rename files that include 'top-level' terminology (#3076)
 * bd40b80b docs: Document work avoidance. (#3066)
 * 6df0b2d3 feat: Support Top level workflow template reference  (#2912)
 * 0709ad28 feat: Enhanced filters for argo {watch,get,submit} (#2450)
 * 784c1385 build: Use goreman for starting locally. (#3074)
 * 5b5bae9a docs: Add Isbank to users.md (#3068)
 * 2b038ed2 feat: Enhanced depends logic (#2673)
 * 4c3387b2 fix: Linters should error if nothing was validated (#3011)
 * 51dd05b5 fix(artifacts): Explicit archive strategy. Fixes #2140 (#3052)
 * ada2209e Revert "fix(artifacts): Allow tar check to be ignored. Fixes #2140 (#3024)" (#3047)
 * b7ff9f09 chore: Add ability to configure maximum DB connection lifetime (#3032)
 * 38a995b7 fix(executor): Properly handle empty resource results, like for a missing get (#3037)
 * a1ac8bcf fix(artifacts): Allow tar check to be ignored. Fixes #2140 (#3024)
 * f12d79ca fix(controller)!: Correctly format workflow.creationTimepstamp as RFC3339. Fixes #2974 (#3023)
 * d10e949a fix: Consider metric nodes that were created and completed in the same operation (#3033)
 * 202d4ab3 fix(executor): Optional input artifacts. Fixes #2990 (#3019)
 * f17e946c fix(executor): Save script results before artifacts in case of error. Fixes #1472 (#3025)
 * 3d216ae6 fix: Consider missing optional input/output artifacts with same name (#3029)
 * 3717dd63 fix: Improve robustness of releases. Fixes #3004 (#3009)
 * 9f86a4e9 feat(ui): Enable CSP, HSTS, X-Frame-Options. Fixes #2760, #1376, #2761 (#2971)
 * cb71d585 refactor(metrics)!: Refactor Metric interface (#2979)
 * c0ee1eb2 docs: Add Ravelin as a user of Argo (#3020)
 * 052e6c51 Fix isTarball to handle the small gzipped file (#3014)
 * cdcba3c4 fix(ui): Displays command args correctl pre-formatted. (#3018)
 * b5160988 build: Mockery v1.1.1 (#3015)
 * a04d8f28 docs: Add StatefulSet and Service doc (#3008)
 * 8412526c docs: Fix Deprecated formatting (#3010)
 * cc0fe433 fix(events): Correct event API Version. Fixes #2994 (#2999)
 * d5d6f750 feat(controller)!: Updates the resource duration calculation. Fixes #2934 (#2937)
 * fa3801a5 feat(ui): Render 2000+ nodes DAG acceptably. (#2959)
 * f952df51 fix(executor/pns): remove sleep before sigkill (#2995)
 * 2a9ee21f feat(ui): Add Suspend and Resume to CronWorkflows in UI (#2982)
 * eefe120f test: Upgrade to argosay:v2 (#3001)
 * 47472f73 chore: Update Mockery (#3000)
 * 46b11e1e docs: Use keyFormat instead of keyPrefix in docs (#2997)
 * 60d5fdc7 fix: Begin counting maxDuration from first child start (#2976)
 * 76aca493 build: Fix Docker build. Fixes #2983 (#2984)
 * d8cb66e7 feat: Add Argo variable {{retries}} to track retry attempt (#2911)
 * 14b7a459 docs: Fix typo with WorkflowTemplates link (#2977)
 * 3c442232 fix: Remove duplicate node event. Fixes #2961 (#2964)
 * d8ab13f2 fix: Consider Shutdown when assesing DAG Phase for incomplete Retry node (#2966)
 * 8a511e10 fix: Nodes with pods deleted out-of-band should be Errored, not Failed (#2855)
 * ca4e08f7 build: Build dev images from cache (#2968)
 * 5f01c4a5 Upgraded to Node 14.0.0 (#2816)
 * 849d876c Fixes error with unknown flag: --show-all (#2960)
 * 93bf6609 fix: Don't update backoff message to save operations (#2951)
 * 3413a5df fix(cli): Remove info logging from watches. Fixes #2955 (#2958)
 * fe9f9019 fix: Display Workflow finish time in UI (#2896)
 * f281199a docs: Update README with new features (#2807)
 * c8bd0bb8 fix(ui): Change default pagination to all and sort workflows (#2943)
 * e3ed686e fix(cli): Re-establish watch on EOF (#2944)
 * 67355372 fix(swagger)!: Fixes invalid K8S definitions in `swagger.json`. Fixes #2888 (#2907)
 * 023f2338 fix(argo-server)!: Implement missing instanceID code. Fixes #2780 (#2786)
 * 7b0739e0 Fix typo (#2939)
 * 20d69c75 Detect ctrl key when a link is clicked (#2935)
 * f32cec31 fix default null value for timestamp column - MySQL 5.7 (#2933)
 * 9773cfeb docs: Add docs/scaling.md (#2918)
 * 99858ea5 feat(controller): Remove the excessive logging of node data (#2925)
 * 03ad694c feat(cli): Refactor `argo list --chunk-size` and add `argo archive list --chunk-size`. Fixes #2820 (#2854)
 * 1c45d5ea test: Use argoproj/argosay:v1 (#2917)
 * f311a5a7 build: Fix Darwin build (#2920)
 * a06cb5e0 fix: remove doubled entry in server cluster role deployment (#2904)
 * c71116dd feat: Windows Container Support. Fixes #1507 and #1383 (#2747)
 * 3afa7b2f fix(ui): Use LogsViewer for container logs (#2825)
 * 9ecd5226 docs: Document node field selector. Closes #2860 (#2882)
 * 7d8818ca fix(controller): Workflow stop and resume by node didn't properly support offloaded nodes. Fixes #2543 (#2548)
 * e013f29d ci: Remove context to stop unauthozied errors on test jobs (#2910)
 * db52e7ba fix(controller): Add mutex to nameEntryIDMap in cron controller. Fix #2638 (#2851)
 * 9a33aa2d docs(users): Adding Habx to the users list (#2781)
 * 9e4ac9b3 feat(cli): Tolerate deleted workflow when running `argo delete`. Fixes #2821 (#2877)
 * a0035dd5 fix: ConfigMap syntax (#2889)
 * c05c3859 ci: Build less and therefore faster (#2839)
 * 56143eb1 feat(ui): Add pagination to workflow list. Fixes #1080 and #976 (#2863)
 * e0ad7de9 test: Fixes various tests (#2874)
 * e378ca47 fix: Cannot create WorkflowTemplate with un-supplied inputs (#2869)
 * c3e30c50 fix(swagger): Generate correct Swagger for inline objects. Fixes #2835 (#2837)
 * c0143d34 feat: Add metric retention policy (#2836)
 * f03cda61 Update getting-started.md (#2872)
 * d66224e1 fix: Don't error when deleting already-deleted WFs (#2866)
 * e84acb50 chore: Display wf.Status.Conditions in CLI (#2858)
 * 3c7f3a07 docs: Fix typo ".yam" -> ".yaml" (#2862)
 * d7f8e0c4 fix(CLI): Re-establish workflow watch on disconnect. Fixes #2796 (#2830)
 * 31358d6e feat(CLI): Add -v and --verbose to Argo CLI (#2814)
 * 1d30f708 ci: Don't configure Sonar on CI for release branches (#2811)
 * d9c54075 docs: Fix exit code example and docs (#2853)
 * 90743353 feat: Expose workflow.serviceAccountName as global variable (#2838)
 * f07f7bf6 note that tar.gz'ing output artifacts is optional (#2797)
 * 3fd3fc6c docs: Document how to label creator (#2827)
 * b956ec65 fix: Add Step node outputs to global scope (#2826)
 * bac339af chore: Configure webpack dev server to proxy using HTTPS (#2812)
 * cc136f9c test: Skip TestStopBehavior. See #2833 (#2834)
 * 52ff43b5 fix: Artifact panic on unknown artifact. Fixes #2824 (#2829)
 * 554fd06c fix: Enforce metric naming validation (#2819)
 * dd223669 docs: Add Microba as official Argo user (#2822)
 * 8151f0c4 docs: Update tls.md (#2813)
 * 0dbd78ff feat: Add TLS support. Closes #2764 (#2766)
 * 510e11b6 fix: Allow empty strings in valueFrom.default (#2805)
 * d7f41ac8 fix: Print correct version in logs. (#2806)
 * e9c21120 chore: Add GCS native example for output artifact (#2789)
 * e0f2697e fix(controller): Include global params when using withParam (#2757)
 * 3441b11a docs: Fix typo in CronWorkflow doc (#2804)
 * a2d2b848 docs: Add example of recursive for loop (#2801)
 * 29d39e29 docs: Update the contributing docs  (#2791)
 * 1ea286eb fix: ClusterWorkflowTemplate RBAC for  argo server (#2753)
 * 1f14f2a5 feat(archive): Implement data retention. Closes #2273 (#2312)
 * d0cc7764 feat: Display argo-server version in `argo version` and in UI. (#2740)
 * 8de57281 feat(controller): adds Kubernetes node name to workflow node detail in web UI and CLI output. Implements #2540 (#2732)
 * 52fa5fde MySQL config fix (#2681)
 * 43d9eebb fix: Rename Submittable API endpoint to `submit` (#2778)
 * 69333a87 Fix template scope tests (#2779)
 * bb1abf7f chore: Add CODEOWNERS file (#2776)
 * 905e0b99 fix: Naming error in Makefile (#2774)
 * 7cb2fd17 fix: allow non path output params (#2680)
 * af9f61ea ci: Recurl (#2769)
 * ef08e642 build: Retry curl 3x (#2768)
 * dedec906 test: Get tests running on release branches (#2767)
 * 1c8318eb fix: Add compatiblity mode to templateReference (#2765)
 * 7975952b fix: Consider expanded tasks in getTaskFromNode (#2756)
 * bc421380 fix: Fix template resolution in UI (#2754)
 * 391c0f78 Make phase and templateRef available for unsuspend and retry selectors (#2723)
 * a6fa3f71 fix: Improve cookie security. Fixes #2759 (#2763)
 * 57f0183c Fix typo on the documentation. It causes error unmarshaling JSON: while (#2730)
 * c6ef1ff1 feat(manifests): add name on workflow-controller-metrics service port (#2744)
 * af5cd1ae docs: Update OWNERS (#2750)
 * 06c4bd60 fix: Make ClusterWorkflowTemplate optional for namespaced Installation (#2670)
 * 25c62463 docs: Update README (#2752)
 * 908e1685 docs: Update README.md (#2751)
 * 4ea43e2d fix: Children of onExit nodes are also onExit nodes (#2722)
 * 3f1b6667 feat: Add Kustomize as supported install option. Closes #2715 (#2724)
 * 691459ed fix: Error pending nodes w/o Pods unless resubmitPendingPods is set (#2721)
 * 874d8776 test: Longer timeout for deletion (#2737)
 * 3c8149fa Fix typo (#2741)
 * 98f60e79 feat: Added Workflow SubmitFromResource API (#2544)
 * 6253997a fix: Reset all conditions when resubmitting (#2702)
 * e7c67de3 fix: Maybe fix watch. Fixes #2678 (#2719)
 * cef6dfb6 fix: Print correct version string. (#2713)
 * e9589d28 feat: Increase pod workers and workflow workers both to 32 by default. (#2705)
 * 3a1990e0 test: Fix Goroutine leak that was making controller unit tests slow. (#2701)
 * 9894c29f ci: Fix Sonar analysis on master. (#2709)
 * 54f5be36 style: Camelcase "clusterScope" (#2720)
 * db6d1416 fix: Flakey TestNestedClusterWorkflowTemplate testcase failure (#2613)
 * b4fd4475 feat(ui): Add a YAML panel to view the workflow manifest. (#2700)
 * 65d413e5 build(ui): Fix compression of UI package. (#2704)
 * 4129528d fix: Don't use docker cache when building release images (#2707)
 * 8d0956c9 test: Increase runCli timeout to 1m (#2703)
 * 9d93e971 Update getting-started.md (#2697)
 * ee644a35 docs: Fix CONTRIBUTING.md and running-locally.md. Fixes #2682 (#2699)
 * 2737c0ab feat: Allow to pass optional flags to resource template (#1779)
 * c1a2fc7c Update running-locally.md - fixing incorrect protoc install (#2689)
 * a1226c46 fix: Enhanced WorkflowTemplate and ClusterWorkflowTemplate validation to support Global Variables   (#2644)
 * c21cc2f3 fix a typo (#2669)
 * 9430a513 fix: Namespace-related validation in UI (#2686)
 * f3eeca6e feat: Add exit code as output variable (#2111)
 * 9f95e23a fix: Report metric emission errors via Conditions (#2676)
 * c67f5ff5 fix: Leaf task with continueOn should not fail DAG (#2668)
 * 3c20d4c0 ci: Migrate to use Sonar instead of CodeCov for analysis (#2666)
 * 9c6351fa feat: Allow step restart on workflow retry. Closes #2334 (#2431)
 * cf277eb5 docs: Updates docs for CII. See #2641 (#2643)
 * e2d0aa23 fix: Consider offloaded and compressed node in retry and resume (#2645)
 * a25c6a20 build: Fix codegen for releases (#2662)
 * 4a3ca930 fix: Correctly emit events. Fixes #2626 (#2629)
 * 4a7d4bdb test: Fix flakey DeleteCompleted test (#2659)
 * 41f91e18 fix: Add DAG as default in UI filter and reorder (#2661)
 * f138ada6 fix: DAG should not fail if its tasks have continueOn (#2656)
 * e5cbdf6a ci: Only run CI jobs if needed (#2655)
 * 4c452d5f fix: Don't attempt to resolve artifacts if task is going to be skipped (#2657)
 * 2caf570a chore: Add newline to fields.md (#2654)
 * 2cb596da Storage region should be specified (#2538)
 * 271e4551 chore: Fix-up Yarn deps (#2649)
 * 4c1b0777 fix: Sort log entries. (#2647)
 * 268fc461  docs: Added doc generator code (#2632)
 * d58b7fc3 fix: Add input paremeters to metric scope (#2646)
 * cc3af0b8 fix: Validating Item Param in Steps Template (#2608)
 * 6c685c5b fix: allow onExit to run if wf exceeds activeDeadlineSeconds. Fixes #2603 (#2605)
 * ffc43ce9 feat: Added Client validation on Workflow/WFT/CronWF/CWFT (#2612)
 * 24655cd9 feat(UI): Move Workflow parameters to top of submit (#2640)
 * 0a3b159a Use error equals (#2636)
 * 8c29e05c ci: Fix codegen job (#2648)
 * a78ecb7f docs(users): Add CoreWeave and ConciergeRender (#2641)
 * 14be4670 fix: Fix logs part 2 (#2639)
 * 4da6f4f3 feat: Add 'outputs.result' to Container templates (#2584)
 * 51bc876d test: Fixes TestCreateWorkflowDryRun. Closes #2618 (#2628)
 * 212c6d75 fix: Support minimal mysql version 5.7.8 (#2633)
 * 8facacee refactor: Refactor Template context interfaces (#2573)
 * 812813a2 fix: fix test cases (#2631)
 * ed028b25 fix: Fix logging problems. See #2589 (#2595)
 * d4e81238 test: Fix teething problems (#2630)
 * 4aad6d55 chore: Add comments to issues (#2627)
 * 54f7a013 test: Enhancements and repairs to e2e test framework (#2609)
 * d95926fe fix: Fix WorkflowTemplate icons to be more cohesive (#2607)
 * 0130e1fd docs: Add fields and core concepts doc (#2610)
 * 5a1ac203 fix: Fixes panic in toWorkflow method (#2604)
 * 51910292 chore: Lint UI on CI, test diagnostics, skip bad test (#2587)
 * 232bb115 fix(error handling): use Errorf instead of New when throwing errors with formatted text (#2598)
 * eeb2f97b fix(controller): dag continue on failed. Fixes #2596 (#2597)
 * 99c35129 docs: Fix inaccurate field name in docs (#2591)
 * 21c73779 fix: Fixes lint errors (#2594)
 * 38aca5fa chore: Added ClusterWorkflowTemplate RBAC on quick-start manifests (#2576)
 * 59f746e1 feat: UI enhancement for Cluster Workflow Template (#2525)
 * 0801a428 fix(cli): Show lint errors of all files (#2552)
 * c3535ba5 docs: Fix wrong Configuring Your Artifact Repository document. (#2586)
 * 79217bc8 feat(archive): allow specifying a compression level (#2575)
 * 88d261d7 fix: Use outputs of last child instead of retry node itslef (#2565)
 * 5c08292e style: Correct the confused logic (#2577)
 * 3d146144 fix: Fix bug in deleting pods. Fixes #2571 (#2572)
 * cb739a68 feat: Cluster scoped workflow template (#2451)
 * c63e3d40 feat: Show workflow duration in the index UI page (#2568)
 * 1520452a chore: Error -> Warn when Parent CronWf no longer exists (#2566)
 * ffbb3b89 fix: Fixes empty/missing CM. Fixes #2285 (#2562)
 * d0fba6f4 chore: fix typos in the workflow template docs (#2563)
 * 49801e32 chore(docker): upgrade base image for executor image (#2561)
 * c4efb8f8 Add Riskified to the user list (#2558)
 * 8b92d33e feat: Create K8S events on node completion. Closes #2274 (#2521)
 * 2902e144 feat: Add node type and phase filter to UI (#2555)
 * fb74ba1c fix: Separate global scope processing from local scope building (#2528)
 * 618b6dee fix: Fixes --kubeconfig flag. Fixes #2492 (#2553)
 * 79dc969f test: Increase timeout for flaky test (#2543)
 * 15a3c990 feat: Report SpecWarnings in status.conditions (#2541)
 * f142f30a docs: Add example of template-level volume declaration. (#2542)
 * 93b6be61 fix(archive): Fix bug that prevents listing archive workflows. Fixes … (#2523)
 * b4c9c54f fix: Omit config key in configure artifact document. (#2539)
 * 864bf1e5 fix: Show template on its own field in CLI (#2535)
 * 555aaf06 test: fix master (#2534)
 * 94862b98 chore: Remove deprecated example (#2533)
 * 5e1e7829 fix: Validate CronWorkflow before creation (#2532)
 * c9241339 fix: Fix wrong assertions (#2531)
 * 67fe04bb Revert "fix: fix template scope tests (#2498)" (#2526)
 * ddfa1ad0 docs: couple of examples for REST API usage of argo-server (#2519)
 * 30542be7 chore(docs): Update docs for useSDKCreds (#2518)
 * e2cc6988 feat: More control over resuming suspended nodes Fixes #1893 (#1904)
 * b2771249 chore: minor fix and refactory (#2517)
 * b1ad163a fix: fix template scope tests (#2498)
 * 661d1b67 Increase client gRPC max size to match server (#2514)
 * d8aa477f fix: Fix potential panic (#2516)
 * 1afb692e fix: Allow runtime resolution for workflow parameter names (#2501)
 * 243ea338 fix(controller): Ensure we copy any executor securityContext when creating wait containers; fixes #2512 (#2510)
 * 6e8c7bad feat: Extend workflowDefaults to full Workflow and clean up docs and code (#2508)
 * 06cfc129 feat: Native Google Cloud Storage support for artifact. Closes #1911 (#2484)
 * 999b1e1d  fix: Read ConfigMap before starting servers  (#2507)
 * 3d6e9b61 docs: Add separate ConfigMap doc for 2.7+ (#2505)
 * e5bd6a7e fix(controller): Updates GetTaskAncestry to skip visited nod. Fixes #1907 (#1908)
 * 183a29e4 docs: add official user (#2499)
 * e636000b feat: Updated arm64 support patch (#2491)
 * 559cb005 feat(ui): Report resources duration in UI. Closes #2460 (#2489)
 * 09291d9d feat: Add default field in parameters.valueFrom (#2500)
 * 33cd4f2b feat(config): Make configuration mangement easier. Closes #2463 (#2464)
 * f3df9660 test: Fix test (#2490)
 * bfaf1c21 chore: Move quickstart Prometheus port to 9090 (#2487)
 * 487ed425 feat: Logging the Pod Spec in controller log (#2476)
 * 96c80e3e fix(cli): Rearrange the order of chunk size argument in list command. Closes #2420 (#2485)
 * 47bd70a0 chore: Fix Swagger for PDB to support Java client (#2483)
 * 53a10564 feat(usage): Report resource duration. Closes #1066 (#2219)
 * 063d9bc6 Revert "feat: Add support for arm64 platform (#2364)" (#2482)
 * 735d25e9 fix: Build image with SHA tag when a git tag is not available (#2479)
 * c55bb3b2 ci: Run lint on CI and fix GolangCI (#2470)
 * e1c9f7af fix ParallelSteps child type so replacements happen correctly; fixes argoproj-labs/argo-client-gen#5 (#2478)
 * 55c315db feat: Add support for IRSA and aws default provider chain. (#2468)
 * c724c7c1 feat: Add support for arm64 platform (#2364)
 * 315dc164 feat: search archived wf by startat. Closes #2436 (#2473)
 * 23d230bd feat(ui): add Env to Node Container Info pane. Closes #2471 (#2472)
 * 10a0789b fix: ParallelSteps swagger.json (#2459)
 * a59428e7 fix: Duration must be a string. Make it a string. (#2467)
 * 47bc6f3b feat: Add `argo stop` command (#2352)
 * 14478bc0 feat(ui): Add the ability to have links to logging facility in UI. Closes #2438 (#2443)
 * 2864c745 chore: make codegen + make start (#2465)
 * a85f62c5 feat: Custom, step-level, and usage metrics (#2254)
 * 64ac0298 fix: Deprecate template.{template,templateRef,arguments} (#2447)
 * 6cb79e4e fix: Postgres persistence SSL Mode (#1866) (#1867)
 * 2205c0e1 fix(controller): Updates to add condition to workflow status. Fixes #2421 (#2453)
 * 9d96ab2f fix: make dir if needed (#2455)
 * 5346609e test: Maybe fix TestPendingRetryWorkflowWithRetryStrategy. Fixes #2446 (#2456)
 * 3448ccf9 fix: Delete PVCs unless WF Failed/Errored (#2449)
 * 782bc8e7 fix: Don't error when optional artifacts are not found (#2445)
 * fc18f3cf chore: Master needs codegen (#2448)
 * 32fc2f78 feat: Support workflow templates submission. Closes #2007 (#2222)
 * 050a143d fix(archive): Fix edge-cast error for archiving. Fixes #2427 (#2434)
 * 9455c1b8 doc: update CHANGELOG.md (#2425)
 * 1baa7ee4 feat(ui): cache namespace selection. Closes #2439 (#2441)
 * 91d29881 feat: Retry pending nodes (#2385)
 * 7094433e test: Skip flakey tests in operator_template_scope_test.go. See #2432 (#2433)
 * 30332b14 fix: Allow numbers in steps.args.params.value (#2414)
 * e9a06dde feat: instanceID support for argo server. Closes #2004 (#2365)
 * 3f8be0cd fix "Unable to retry workflow" on argo-server (#2409)
 * dd3029ab docs: Example showing how to use default settings for workflow spec. Related to ##2388 (#2411)
 * 13508828 fix: Check child node status before backoff in retry (#2407)
 * b59419c9 fix: Build with the correct version if you check out a specific version (#2423)
 * 6d834d54 chore: document BASE_HREF (#2418)
 * 184c3653 fix: Remove lazy workflow template (#2417)
 * 918d0d17 docs: Added Survey Results (#2416)
 * 20d6e27b Update CONTRIBUTING.md (#2410)
 * f2ca045e feat: Allow WF metadata spec on Cron WF (#2400)
 * 068a4336 fix: Correctly report version. Fixes #2374 (#2402)
 * e19a398c Update pull_request_template.md (#2401)
 * 7c99c109 chore: Fix typo (#2405)
 * 175b164c Change font family for class yaml (#2394)
 * d1194755 fix: Don't display Retry Nodes in UI if len(children) == 1 (#2390)
 * b8623ec7 docs: Create USERS.md (#2389)
 * 1d21d3f5 fix(doc strings): Fix bug related documentation/clean up of default configurations #2331 (#2388)
 * 77e11fc4 chore: add noindex meta tag to solve #2381; add kustomize to build docs (#2383)
 * 42200fad fix(controller): Mount volumes defined in script templates. Closes #1722 (#2377)
 * 96af36d8 fix: duration must be a string (#2380)
 * 7bf08192 fix: Say no logs were outputted when pod is done (#2373)
 * 847c3507 fix(ui): Removed tailLines from EventSource (#2330)
 * 3890a124 feat: Allow for setting default configurations for workflows, Fixes #1923, #2044 (#2331)
 * 81ab5385 Update readme (#2379)
 * 91810273 feat: Log version (structured) on component start-up (#2375)
 * d0572a74 docs: Make Getting Started agnostic to version (#2371)
 * d3a3f6b1 docs: Add Prudential to the users list (#2353)
 * 4714c880 chore: Master needs codegen (#2369)
 * 5b6b8257 fix(docker): fix streaming of combined stdout/stderr (#2368)
 * 97438313 fix: Restart server ConfigMap watch when closed (#2360)
 * 64d0cec0 chore: Master needs make lint (#2361)
 * 12386fc6 fix: rerun codegen after merging OSS artifact support (#2357)
 * 40586ed5 fix: Always validate templates (#2342)
 * 897db894 feat: Add support for Alibaba Cloud OSS artifact (#1919)
 * 7e2dba03 feat(ui): Circles for nodes (#2349)
 * e85f6169 chore: update getting started guide to use 2.6.0 (#2350)
 * 7ae4ec78 docker: remove NopCloser from the executor. (#2345)
 * 5895b364 feat: Expose workflow.paramteres with JSON string of all params (#2341)
 * a9850b43 Fix the default (#2346)
 * c3763d34 fix: Simplify completion detection logic in DAGs (#2344)
 * d8a9ea09 fix(auth): Fixed returning  expired  Auth token for GKE (#2327)
 * 6fef0454 fix: Add timezone support to startingDeadlineSeconds (#2335)
 * c28731b9 chore: Add go mod tidy to codegen (#2332)
 * a66c8802 feat: Allow Worfklows to be submitted as files from UI (#2340)
 * a9c1d547 docs: Update Argo Rollouts description (#2336)
 * 8672b97f fix(Dockerfile): Using `--no-install-recommends` (Optimization) (#2329)
 * c3fe1ae1 fix(ui): fixed worflow UI refresh. Fixes ##2337 (#2338)
 * d7690e32 feat(ui): Adds ability zoom and hide successful steps. POC (#2319)
 * e9e13d4c feat: Allow retry strategy on non-leaf nodes, eg for step groups. Fixes #1891 (#1892)
 * 62e6db82 feat: Ability to include or exclude fields in the response (#2326)
 * 52ba89ad fix(swagger): Fix the broken swagger. (#2317)
 * efb8a1ac docs: Update CODE_OF_CONDUCT.md (#2323)
 * 1c77e864 fix(swagger): Fix the broken swagger. (#2317)
 * aa052346 feat: Support workflow level poddisruptionbudge for workflow pods #1728 (#2286)
 * 8da88d7e chore: update getting-started guide for 2.5.2 and apply other tweaks (#2311)
 * 2f97c261 build: Improve reliability of release. (#2309)
 * 5dcb84bb chore(cli): Clean-up code. Closes #2117 (#2303)
 * e49dd8c4 chore(cli): Migrate `argo logs` to use API client. See #2116 (#2177)
 * 5c3d9cf9 chore(cli): Migrate `argo wait` to use API client. See #2116 (#2282)
 * baf03f67 fix(ui): Provide a link to archived logs. Fixes #2300 (#2301)
 * b5947165 feat: Create API clients (#2218)
 * 214c4515 fix(controller): Get correct Step or DAG name. Fixes #2244 (#2304)
 * c4d26466 fix: Remove active wf from Cron when deleted (#2299)
 * 0eff938d fix: Skip empty withParam steps (#2284)
 * 636ea443 chore(cli): Migrate `argo terminate` to use API client. See #2116 (#2280)
 * d0a9b528 chore(cli): Migrate `argo template` to use API client. Closes #2115 (#2296)
 * f69a6c5f chore(cli): Migrate `argo cron` to use API client. Closes #2114 (#2295)
 * 80b9b590 chore(cli): Migrate `argo retry` to use API client. See #2116 (#2277)
 * cdbc6194 fix(sequence): broken in 2.5. Fixes #2248 (#2263)
 * 0d3955a7 refactor(cli): 2x simplify migration to API client. See #2116 (#2290)
 * df8493a1 fix: Start Argo server with out Configmap #2285 (#2293)
 * 51cdf95b doc: More detail for namespaced installation (#2292)
 * a7302697 build(swagger): Fix argo-server swagger so version does not change. (#2291)
 * 47b4fc28 fix(cli): Reinstate `argo wait`. Fixes #2281 (#2283)
 * 1793887b chore(cli): Migrate `argo suspend` and `argo resume` to use API client. See #2116 (#2275)
 * 1f3d2f5a chore(cli): Update `argo resubmit` to support client API. See #2116 (#2276)
 * c33f6cda fix(archive): Fix bug in migrating cluster name. Fixes #2272 (#2279)
 * fb0acbbf fix: Fixes double logging in UI. Fixes #2270 (#2271)
 * acf37c2d fix: Correctly report version. Fixes #2264 (#2268)
 * b30f1af6 fix: Removes Template.Arguments as this is never used. Fixes #2046 (#2267)
 * 79b09ed4 fix: Removed duplicate Watch Command (#2262)
 * b5c47266 feat(ui): Add filters for archived workflows (#2257)
 * d30aa335 fix(archive): Return correct next page info. Fixes #2255 (#2256)
 * 8c97689e fix: Ignore bookmark events for restart. Fixes #2249 (#2253)
 * 63858eaa fix(offloading): Change offloaded nodes datatype to JSON to support 1GB. Fixes #2246 (#2250)
 * 4d88374b Add Cartrack into officially using Argo (#2251)
 * d309d5c1 feat(archive): Add support to filter list by labels. Closes #2171 (#2205)
 * 79f13373 feat: Add a new symbol for suspended nodes. Closes #1896 (#2240)
 * 82b48821 Fix presumed typo (#2243)
 * af94352f feat: Reduce API calls when changing filters. Closes #2231 (#2232)
 * a58cbc7d BasisAI uses Argo (#2241)
 * 68e3c9fd feat: Add Pod Name to UI (#2227)
 * eef85072 fix(offload): Fix bug which deleted completed workflows. Fixes #2233 (#2234)
 * 4e4565cd feat: Label workflow-created pvc with workflow name (#1890)
 * 8bd5ecbc fix: display error message when deleting archived workflow fails. (#2235)
 * ae381ae5 feat: This add support to enable debug logging for all CLI commands (#2212)
 * 1b1927fc feat(swagger): Adds a make api/argo-server/swagger.json (#2216)
 * 5d7b4c8c Update README.md (#2226)
 * 170abfa5 chore: Run `go mod tidy` (#2225)
 * 2981e6ff fix: Enforce UnknownField requirement in WorkflowStep (#2210)
 * affc235c feat: Add failed node info to exit handler (#2166)
 * af1f6d60 fix: UI Responsive design on filter box (#2221)
 * a445049c fix: Fixed race condition in kill container method. Fixes #1884 (#2208)
 * 2672857f feat: upgrade to Go 1.13. Closes #1375 (#2097)
 * 7466efa9 feat: ArtifactRepositoryRef ConfigMap is now taken from the workflow namespace (#1821)
 * 50f331d0 build: Fix ARGO_TOKEN (#2215)
 * 7f090351 test: Correctly report diagnostics (#2214)
 * f2bd74bc fix: Remove quotes from UI (#2213)
 * 62f46680 fix(offloading): Correctly deleted offloaded data. Fixes #2206 (#2207)
 * e30b77fc feat(ui): Add label filter to workflow list page. Fixes #802 (#2196)
 * 930ced39 fix(ui): fixed workflow filtering and ordering. Fixes #2201 (#2202)
 * 88112312 fix: Correct login instructions. (#2198)
 * d6f5953d Update ReadMe for EBSCO (#2195)
 * b024c46c feat: Add ability to submit CronWorkflow from CLI (#2003)
 * c97527ce test: Invoke tests using s.T() (#2194)
 * 72a54fe1 chore: Move info.proto et al to correct package (#2193)
 * f6600fa4 fix: Namespace and phase selection in UI (#2191)
 * c4a24dca fix(k8sapi-executor): Fix KillContainer impl (#2160)
 * d22a5fe6 Update cli_with_server_test.go (#2189)
 * ff18180f test: Remove podGC (#2187)
 * 78245305 chore: Improved error handling and refactor (#2184)
 * b9c828ad fix(archive): Only delete offloaded data we do not need. Fixes #2170 and #2156 (#2172)
 * 73cb5418 feat: Allow CronWorkflows to have instanceId (#2081)
 * 9efea660 Sort list and add Greenhouse (#2182)
 * cae399ba fix: Fixed the Exec Provider token bug (#2181)
 * fc476b2a fix(ui): Retry workflow event stream on connection loss. Fixes #2179 (#2180)
 * 65058a27 fix: Correctly create code from changed protos. (#2178)
 * 585d1eef chore: Update lint command to use apiclient. See #2116 (#2131)
 * 299d467c build: Update release process and docs (#2128)
 * fcfe1d43 feat: Implemented open default browser in local mode (#2122)
 * f6cee552 fix: Specify download .tgz extension (#2164)
 * 8a1e611a feat: Update archived workdflow column to be JSON. Closes #2133 (#2152)
 * f591c471 fix!: Change `argo token` to `argo auth token`. Closes #2149 (#2150)
 * 519c9434 chore: Add Mock gen to make codegen (#2148)
 * 409a5154 fix: Add certs to argocli image. Fixes #2129 (#2143)
 * b094802a fix: Allow download of artifacs in server auth-mode. Fixes #2129 (#2147)
 * 520fa540 fix: Correct SQL syntax. (#2141)
 * 059cb9b1 fix: logs UI should fall back to archive (#2139)
 * 4cda9a05 fix: route all unknown web content requests to index.html (#2134)
 * 14d8b5d3 fix: archiveLogs needs to copy stderr (#2136)
 * 91319ee4 fixed ui navigation issues with basehref (#2130)
 * 7881b980 docs: Add CronWorkflow usage docs (#2124)
 * badfd183 feat: Add support to delete by using labels. Depended on by #2116 (#2123)
 * 706d0f23 test: Try and make e2e more robust. Fixes #2125 (#2127)
 * a75ac1b4 fix: mark CLI common.go vars and funcs as DEPRECATED (#2119)
 * be21a0f1 feat(server): Restart server when config changes. Fixes #2090 (#2092)
 * b5cd72b0 test: Parallelize Cron tests (#2118)
 * b2bd25bc fix: Disable webpack dot rule (#2112)
 * 865b4f3a addcompany (#2109)
 * 213e3a9d fix: Fix Resource Deletion Bug (#2084)
 * ab1de233 refactor(cli): Introduce v1.Interface for CLI. Closes #2107 (#2048)
 * 7a19f85c feat: Implemented Basic Auth scheme (#2093)
 * 7611b9f6 fix(ui): Add support for bash href. Fixes ##2100 (#2105)
 * 516d05f8  fix: Namespace redirects no longer error and are snappier (#2106)
 * 16aed5c8 fix: Skip running --token testing if it is not on CI (#2104)
 * aece7e6e Parse container ID in correct way on CRI-O. Fixes #2095 (#2096)
 * b6a2be89 feat: support arg --token when talking to argo-server (#2027) (#2089)
 * 01d8cae1 build: adds `make env` to make testing easier (#2087)
 * 492842aa docs(README): Add Capital One to user list (#2094)
 * d56a0e12 fix(controller): Fix template resolution for step groups. Fixes #1868  (#1920)
 * b97044d2 fix(security): Fixes an issue that allowed you to list archived workf… (#2079)
 * c4f49cf0 refactor: Move server code (cmd/server/ -> server/) (#2071)
 * 2542454c fix(controller): Do not crash if cm is empty. Fixes #2069 (#2070)
 * 85fa9aaf fix: Do not expect workflowChange to always be defined (#2068)
 * 6f65bc2b fix: "base64 -d" not always available, using "base64 --decode" (#2067)
 * 6f2c8802 feat(ui): Use cookies in the UI. Closes #1949 (#2058)
 * 4592aec6 fix(api): Change `CronWorkflowName` to `Name`. Fixes #1982 (#2033)
 * e26c11af fix: only run archived wf testing when persistence is enabled (#2059)
 * b3cab5df fix: Fix permission test cases (#2035)
 * b408e7cd fix: nil pointer in GC (#2055)
 * 4ac11560 fix: offload Node Status in Get and List api call (#2051)
 * dfdde1d0 ci: Run using our own cowsay image (#2047)
 * 71ba8238 Update README.md (#2045)
 * c7953052 fix(persistence): Allow `argo server` to run without persistence (#2050)
 * 1db74e1a fix(archive): upsert archive + ci: Pin images on CI, add readiness probes, clean-up logging and other tweaks (#2038)
 * c46c6836 feat: Allow workflow-level parameters to be modified in the UI when submitting a workflow (#2030)
 * faa9dbb5 fix(Makefile): Rename staticfiles make target. Fixes #2010 (#2040)
 * 79a42d48 docs: Update link to configure-artifact-repository.md (#2041)
 * 1a96007f fix: Redirect to correct page when using managed namespace. Fixes #2015 (#2029)
 * 78726314 fix(api): Updates proto message naming (#2034)
 * 4a1307c8 feat: Adds support for MySQL. Fixes #1945 (#2013)
 * d843e608 chore: Smoke tests are timing out, give them more time (#2032)
 * 5c98a14e feat(controller): Add audit logs to workflows. Fixes #1769 (#1930)
 * 2982c1a8 fix(validate): Allow placeholder in values taken from inputs. Fixes #1984 (#2028)
 * 3293c83f feat: Add version to offload nodes. Fixes #1944 and #1946 (#1974)
 * 283bbf8d build: `make clean` now only deletes dist directories (#2019)
 * 72fa88c9 build: Enable linting for tests. Closes #1971 (#2025)
 * f8569ae9 feat: Auth refactoring to support single version token (#1998)
 * eb360d60 Fix README (#2023)
 * ef1bd3a3 fix typo (#2021)
 * f25a45de feat(controller): Exposes container runtime executor as CLI option. (#2014)
 * 3b26af7d Enable s3 trace support. Bump version to v2.5.0. Tweak proto id to match Workflow (#2009)
 * 5eb15bb5 fix: Fix workflow level timeouts (#1369)
 * 5868982b fix: Fixes the `test` job on master (#2008)
 * 29c85072 fix: Fixed grammar on TTLStrategy (#2006)
 * 2f58d202 fix: v2 token bug (#1991)
 * ed36d92f feat: Add quick start manifests to Git. Change auth-mode to default to server. Fixes #1990 (#1993)
 * d1965c93 docs: Encourage users to upvote issues relevant to them (#1996)
 * 91331a89 fix: No longer delete the argo ns as this is dangerous (#1995)
 * 1a777cc6 feat(cron): Added timezone support to cron workflows. Closes #1931 (#1986)
 * 48b85e57 fix: WorkflowTempalteTest fix (#1992)
 * 51dab8a4 feat: Adds `argo server` command. Fixes #1966 (#1972)
 * 732e03bb chore: Added WorkflowTemplate test (#1989)
 * 27387d4b chore: Fix UI TODOs (#1987)
 * dd704dd6 feat: Renders namespace in UI. Fixes #1952 and #1959 (#1965)
 * 14d58036 feat(server): Argo Server. Closes #1331 (#1882)
 * f69655a0 fix: Added decompress in retry, resubmit and resume. (#1934)
 * 1e7ccb53 updated jq version to 1.6 (#1937)
 * c51c1302 feat: Enhancement for namespace installation mode configuration (#1939)
 * 6af100d5 feat: Add suspend and resume to CronWorkflows CLI (#1925)
 * 232a465d feat: Added onExit handlers to Step and DAG (#1716)
 * 071eb112 docs: Update PR template to demand tests. (#1929)
 * ae58527e docs: Add CyberAgent to the list of Argo users (#1926)
 * 02022e4b docs: Minor formatting fix (#1922)
 * e4107bb8 Updated Readme.md for companies using Argo: (#1916)
 * 7e9b2b58 feat: Support for scheduled Workflows with CronWorkflow CRD (#1758)
 * 5d7e9185 feat: Provide values of withItems maps as JSON in {{item}}. Fixes #1905 (#1906)
 * de3ffd78  feat: Enhanced Different TTLSecondsAfterFinished depending on if job is in Succeeded, Failed or Error, Fixes (#1883)
 * 94449876 docs: Add question option to issue templates (#1910)
 * 83ae2df4 fix: Decrease docker build time by ignoring node_modules (#1909)
 * 59a19069 feat: support iam roles for service accounts in artifact storage (#1899)
 * 6526b6cc fix: Revert node creation logic (#1818)
 * 160a7940 fix: Update Gopkg.lock with dep ensure -update (#1898)
 * ce78227a fix: quick fail after pod termination (#1865)
 * cd3bd235 refactor: Format Argo UI using prettier (#1878)
 * b48446e0 fix: Fix support for continueOn failed for DAG. Fixes #1817 (#1855)
 * 48256961 fix: Fix template scope (#1836)
 * eb585ef7 fix: Use dynamically generated placeholders (#1844)
 * c821cfcc test: Adds 'test' and 'ui' jobs to CI (#1869)
 * 54f44909 feat: Always archive logs if in config. Closes #1790 (#1860)
 * 1e25d6cf docs: Fix e2e testing link (#1873)
 * f5f40728 fix: Minor comment fix (#1872)
 * 72fad7ec Update docs (#1870)
 * 90352865 docs: Update doc based on helm 3.x changes (#1843)
 * 78889895 Move Workflows UI from https://github.com/argoproj/argo-ui (#1859)
 * 4b96172f docs: Refactored and cleaned up docs (#1856)
 * 6ba4598f test: Adds core e2e test infra. Fixes #678 (#1854)
 * 87f26c8d fix: Move ISSUE_TEMPLATE/ under .github/ (#1858)
 * bd78d159 fix: Ensure timer channel is empty after stop (#1829)
 * afc63024 Code duplication (#1482)
 * 5b136713 docs: biobox analytics (#1830)
 * 68b72a8f add CCRi to list of users in README (#1845)
 * 941f30aa Add Sidecar Technologies to list of who uses Argo (#1850)
 * a08048b6 Adding Wavefront to the users list (#1852)
 * 1cb68c98 docs: Updates issue and PR templates. (#1848)
 * cb0598ea Fixed Panic if DB context has issue (#1851)
 * e5fb8848 fix: Fix a couple of nil derefs (#1847)
 * b3d45850 Add HOVER to the list of who uses Argo (#1825)
 * 99db30d6 InsideBoard uses Argo (#1835)
 * ac8efcf4 Red Hat uses Argo (#1828)
 * 41ed3acf Adding Fairwinds to the list of companies that use Argo (#1820)
 * 5274afb9 Add exponential back-off to retryStrategy (#1782)
 * e522e30a Handle operation level errors PVC in Retry (#1762)
 * f2e6054e Do not resolve remote templates in lint (#1787)
 * 3852bc3f SSL enabled database connection for workflow repository (#1712) (#1756)
 * f2676c87 Fix retry node name issue on error (#1732)
 * d38a107c Refactoring Template Resolution Logic (#1744)
 * 23e94604 Error occurred on pod watch should result in an error on the wait container (#1776)
 * 57d051b5 Added hint when using certain tokens in when expressions (#1810)
 * 0e79edff Make kubectl print status and start/finished time (#1766)
 * 723b3c15 Fix code-gen docs (#1811)
 * 711bb114 Fix withParam node naming issue (#1800)
 * 4351a336 Minor doc fix (#1808)
 * efb748fe Fix some issues in examples (#1804)
 * a3e31289 Add documentation for executors (#1778)
 * 1ac75b39 Add  to linter (#1777)
 * 3bead0db Add ability to retry nodes after errors (#1696)
 * b50845e2 Support no-headers flag (#1760)
 * 7ea2b2f8 Minor rework of suspened node (#1752)
 * 9ab1bc88 Update README.md (#1768)
 * e66fa328 Fixed lint issues (#1739)
 * 63e12d09 binary up version (#1748)
 * 1b7f9bec Minor typo fix (#1754)
 * 4c002677 fix blank lines (#1753)
 * fae73826 Fail suspended steps after deadline (#1704)
 * b2d7ee62 Fix typo in docs (#1745)
 * f2592448 Removed uneccessary debug Println (#1741)
 * 846d01ed Filter workflows in list  based on name prefix (#1721)
 * 8ae688c6 Added ability to auto-resume from suspended state (#1715)
 * fb617b63 unquote strings from parameter-file (#1733)
 * 34120341 example for pod spec from output of previous step (#1724)
 * 12b983f4 Add gonum.org/v1/gonum/graph to Gopkg.toml (#1726)
 * 327fcb24 Added  Protobuf extension  (#1601)
 * 602e5ad8 Fix invitation link. (#1710)
 * eb29ae4c Fixes bugs in demo (#1700)
 * ebb25b86 `restartPolicy` -> `retryStrategy` in examples (#1702)
 * 167d65b1 Fixed incorrect `pod.name` in retry pods (#1699)
 * e0818029 fixed broke metrics endpoint per #1634 (#1695)
 * 36fd09a1 Apply Strategic merge patch against the pod spec (#1687)
 * d3546467 Fix retry node processing (#1694)
 * dd517e4c Print multiple workflows in one command (#1650)
 * 09a6cb4e Added status of previous steps as variables (#1681)
 * ad3dd4d4 Fix issue that workflow.priority substitution didn't pass validation (#1690)
 * 095d67f8 Store locally referenced template properly (#1670)
 * 30a91ef0 Handle retried node properly (#1669)
 * 263cb703 Update README.md  Argo Ansible role: Provisioning Argo Workflows on Kubernetes/OpenShift (#1673)
 * 867f5ff7 Handle sidecar killing properly (#1675)
 * f0ab9df9 Fix typo (#1679)
 * 502db42d Don't provision VM for empty artifacts (#1660)
 * b5dcac81 Resolve WorkflowTemplate lazily (#1655)
 * d15994bb [User] Update Argo users list (#1661)
 * 4a654ca6 Stop failing if artifact file exists, but empty (#1653)
 * c6cddafe Bug fixes in getting started (#1656)
 * ec788373 Update workflow_level_host_aliases.yaml (#1657)
 * 7e5af474 Fix child node template handling (#1654)
 * 7f385a6b Use stored templates to raggregate step outputs (#1651)
 * cd6f3627 Fix dag output aggregation correctly (#1649)
 * 706075a5 Fix DAG output aggregation (#1648)
 * fa32dabd Fix missing merged changes in validate.go (#1647)
 * 45716027 fixed example wrong comment (#1643)
 * 69fd8a58 Delay killing sidecars until artifacts are saved (#1645)
 * ec5f9860 pin colinmarc/hdfs to the next commit, which no longer has vendored deps (#1622)
 * 4b84f975 Fix global lint issue (#1641)
 * bb579138 Fix regression where global outputs were unresolveable in DAGs (#1640)
 * cbf99682 Fix regression where parallelism could cause workflow to fail (#1639)
 * 76461f92 Update CHANGELOG for v2.4.0 (#1636)
 * c75a0861 Regenerate installation manifests (#1638)
 * e20cb28c Grant get secret role to controller to support persistence (#1615)
 * 644946e4 Save stored template ID in nodes (#1631)
 * 5d530bec Fix retry workflow state (#1632)
 * 2f0af522 Update operator.go (#1630)
 * 6acea0c1 Store resolved templates (#1552)
 * df8260d6 Increase timeout of golangci-lint (#1623)
 * 138f89f6 updated invite link (#1621)
 * d027188d Updated the API Rule Violations list (#1618)
 * a317fbf1 Prevent controller from crashing due to glog writing to /tmp (#1613)
 * 20e91ea5 Added WorkflowStatus and NodeStatus types to the Open API Spec. (#1614)
 * ffb281a5 Small code cleanup and add tests (#1562)
 * 1cb8345d Add merge keys to Workflow objects to allow for StrategicMergePatches (#1611)
 * c855a66a Increased Lint timeout (#1612)
 * 4bf83fc3 Fix DAG enable failFast will hang in some case (#1595)
 * e9f3d9cb Do not relocate the mounted docker.sock (#1607)
 * 1bd50fa2 Added retry around RuntimeExecutor.Wait call when waiting for main container completion (#1597)
 * 0393427b Issue1571  Support ability to assume IAM roles in S3 Artifacts  (#1587)
 * ffc0c84f Update Gopkg.toml and Gopkg.lock (#1596)
 * aa3a8f1c Update from github.com/ghodss/yaml to sigs.k8s.io/yaml (#1572)
 * 07a26f16 Regard resource templates as leaf nodes (#1593)
 * 89e959e7 Fix workflow template in namespaced controller (#1580)
 * cd04ab8b remove redundant codes (#1582)
 * 5bba8449 Add entrypoint label to workflow default labels (#1550)
 * 9685d7b6 Fix inputs and arguments during template resolution (#1545)
 * 19210ba6 added DataStax as an organization that uses Argo (#1576)
 * b5f2fdef Support AutomountServiceAccountToken and executor specific service account(#1480)
 * 8808726c Fix issue saving outputs which overlap paths with inputs (#1567)
 * ba7a1ed6 Add coverage make target (#1557)
 * ced0ee96 Document workflow controller dockerSockPath config (#1555)
 * 3e95f2da Optimize argo binary install documentation (#1563)
 * e2ebb166 docs(readme): fix workflow types link (#1560)
 * 6d150a15 Initialize the wfClientset before using it (#1548)
 * 5331fc02 Remove GLog config from argo executor (#1537)
 * ed4ac6d0 Update main.go (#1536)
 * 9fca1441 Update argo dependencies to kubernetes v1.14 (#1530)
 * 0246d184 Use cache to retrieve WorkflowTemplates (#1534)
 * 4864c32f Update README.md (#1533)
 * 4df114fa Update CHANGELOG for v2.4 (#1531)
 * c7e5cba1 Introduce podGC strategy for deleting completed/successful pods (#1234)
 * bb0d14af Update ISSUE_TEMPLATE.md (#1528)
 * b5702d8a Format sources and order imports with the help of goimports (#1504)
 * d3ff77bf Added Architecture doc (#1515)
 * fc1ec1a5 WorkflowTemplate CRD (#1312)
 * f99d3266 Expose all input parameters to template as JSON (#1488)
 * bea60526 Fix argo logs empty content when workflow run in virtual kubelet env (#1201)
 * d82de881 Implemented support for WorkflowSpec.ArtifactRepositoryRef (#1350)
 * 0fa20c7b Fix validation (#1508)
 * 87e2cb60 Add --dry-run option to `argo submit` (#1506)
 * e7e50af6 Support git shallow clones and additional ref fetches (#1521)
 * 605489cd Allow overriding workflow labels in 'argo submit' (#1475)
 * 47eba519 Fix issue [Documentation] kubectl get service argo-artifacts -o wide (#1516)
 * 02f38262 Fixed #1287 Executor kubectl version is obsolete (#1513)
 * f62105e6 Allow Makefile variables to be set from the command line (#1501)
 * e62be65b Fix a compiler error in a unit test (#1514)
 * 5c5c29af Fix the lint target (#1505)
 * e03287bf Allow output parameters with .value, not only .valueFrom (#1336)
 * 781d3b8a Implemented Conditionally annotate outputs of script template only when consumed #1359 (#1462)
 * b028e61d change 'continue-on-fail' example to better reflect its description (#1494)
 * 97e824c9 Readme update to add argo and airflow comparison (#1502)
 * 414d6ce7 Fix a compiler error (#1500)
 * ca1d5e67 Fix: Support the List within List type in withParam #1471 (#1473)
 * 75cb8b9c Fix #1366 unpredictable global artifact behavior (#1461)
 * 082e5c4f Exposed workflow priority as a variable (#1476)
 * 38c4def7 Fix: Argo CLI should show warning if there is no workflow definition in file #1486
 * af7e496d Add Commodus Tech as official user (#1484)
 * 8c559642 Update OWNERS (#1485)
 * 007d1f88 Fix: 1008 `argo wait` and `argo submit --wait` should exit 1 if workflow fails  (#1467)
 * 3ab7bc94 Document the insecureIgnoreHostKey git flag (#1483)
 * 7d9bb51a Fix failFast bug:   When a node in the middle fails, the entire workflow will hang (#1468)
 * 42adbf32 Add --no-color flag to logs (#1479)
 * 67fc29c5 fix typo: symboloic > symbolic (#1478)
 * 7c3e1901 Added Codec to the Argo community list (#1477)
 * 0a9cf9d3 Add doc about failFast feature (#1453)
 * 6a590300 Support PodSecurityContext (#1463)
 * e392d854 issue-1445: changing temp directory for output artifacts from root to tmp (#1458)
 * 7a21adfe New Feature:  provide failFast flag, allow a DAG to run all branches of the DAG (either success or failure) (#1443)
 * b9b87b7f Centralized Longterm workflow persistence storage  (#1344)
 * cb09609b mention sidecar in failure message for sidecar containers (#1430)
 * 373bbe6e Fix demo's doc issue of install minio chart (#1450)
 * 83552334 Add threekit to user list (#1444)
 * 83f82ad1 Improve bash completion (#1437)
 * ee0ec78a Update documentation for workflow.outputs.artifacts (#1439)
 * 9e30c06e Revert "Update demo.md (#1396)" (#1433)
 * c08de630 Add paging function for list command (#1420)
 * bba2f9cb Fixed:  Implemented Template level service account (#1354)
 * d635c1de Ability to configure hostPath mount for `/var/run/docker.sock` (#1419)
 * d2f7162a Terminate all containers within pod after main container completes (#1423)
 * 1607d74a PNS executor intermitently failed to capture entire log of script templates (#1406)
 * 5e47256c Fix typo (#1431)
 * 5635c33a Update demo.md (#1396)
 * 83425455 Add OVH as official user (#1417)
 * 82e5f63d Typo fix in ARTIFACT_REPO.md (#1425)
 * 15fa6f52 Update OWNERS (#1429)
 * 96b9a40e Orders uses alphabetically (#1411)
 * 6550e2cb chore: add IBM to official users section in README.md (#1409)
 * bc81fe28 Fiixed: persistentvolumeclaims already exists #1130 (#1363)
 * 6a042d1f Update README.md (#1404)
 * aa811fbd Update README.md (#1402)
 * abe3c99f Add Mirantis as an official user (#1401)
 * 18ab750a Added Argo Rollouts to README (#1388)
 * 67714f99 Make locating kubeconfig in example os independent (#1393)
 * 672dc04f Fixed: withParam parsing of JSON/YAML lists #1389 (#1397)
 * b9aec5f9 Fixed: make verify-codegen is failing on the master branch (#1399) (#1400)
 * 270aabf1 Fixed:  failed to save outputs: verify serviceaccount default:default has necessary privileges (#1362)
 * 163f4a5d Fixed: Support hostAliases in WorkflowSpec #1265 (#1365)
 * abb17478 Add Max Kelsen to USERS in README.md (#1374)
 * dc549193 Update docs for the v2.3.0 release and to use the stable tag
 * 4001c964 Update README.md (#1372)
 * 6c18039b Fix issue where a DAG with exhausted retries would get stuck Running (#1364)
 * d7e74fe3 Validate action for resource templates (#1346)
 * 810949d5 Fixed :  CLI Does Not Honor metadata.namespace #1288 (#1352)
 * e58859d7 [Fix #1242] Failed DAG nodes are now kept and set to running on RetryWorkflow. (#1250)
 * d5fe5f98 Use golangci-lint instead of deprecated gometalinter (#1335)
 * 26744d10 Support an easy way to set owner reference (#1333)
 * 8bf7578e Add --status filter for get command (#1325)
 * 3f6ac9c9 Update release instructions
 * 2274130d Update version to v2.3.0-rc3
 * b024b3d8 Fix: # 1328 argo submit --wait and argo wait quits while workflow is running (#1347)
 * 24680b7f Fixed : Validate the secret credentials name and key (#1358)
 * f641d84e Fix input artifacts with multiple ssh keys (#1338)
 * e680bd21 add / test (#1240)
 * ee788a8a Fix #1340 parameter substitution bug (#1345)
 * 60b65190 Fix missing template local volumes, Handle volumes only used in init containers (#1342)
 * 4e37a444 Add documentation on releasing
 * bb1bfdd9 Update version to v2.3.0-rc2. Update changelog
 * 49a6b6d7 wait will conditionally become privileged if main/sidecar privileged (resolves #1323)
 * 34af5a06 Fix regression where argoexec wait would not return when podname was too long
 * bd8d5cb4 `argo list` was not displaying non-zero priorities correctly
 * 64370a2d Support parameter substitution in the volumes attribute (#1238)
 * 6607dca9 Issue1316 Pod creation with secret volumemount  (#1318)
 * a5a2bcf2 Update README.md (#1321)
 * 950de1b9 Export the methods of `KubernetesClientInterface` (#1294)
 * 1c729a72 Update v2.3.0 CHANGELOG.md
 * 40f9a875 Reorganize manifests to kustomize 2 and update version to v2.3.0-rc1
 * 75b28a37 Implement support for PNS (Process Namespace Sharing) executor (#1214)
 * b4edfd30 Fix SIGSEGV in watch/CheckAndDecompress. Consolidate duplicate code (resolves #1315)
 * 02550be3 Archive location should conditionally be added to template only when needed
 * c60010da Fix nil pointer dereference with secret volumes (#1314)
 * db89c477 Fix formatting issues in examples documentation (#1310)
 * 0d400f2c Refactor checkandEstimate to optimize podReconciliation (#1311)
 * bbdf2e2c Add alibaba cloud to officially using argo list (#1313)
 * abb77062 CheckandEstimate implementation to optimize podReconciliation (#1308)
 * 1a028d54 Secrets should be passed to pods using volumes instead of API calls (#1302)
 * e34024a3 Add support for init containers (#1183)
 * 4591e44f Added support for artifact path references (#1300)
 * 928e4df8 Add Karius to users in README.md (#1305)
 * de779f36 Add community meeting notes link (#1304)
 * a8a55579 Speed up podReconciliation using parallel goroutine (#1286)
 * 93451119 Add dns config support (#1301)
 * 850f3f15 Admiralty: add link to blog post, add user (#1295)
 * d5f4b428 Fix for Resource creation where template has same parameter templating (#1283)
 * 9b555cdb Issue#896 Workflow steps with non-existant output artifact path will succeed (#1277)
 * adab9ed6 Argo CI is current inactive (#1285)
 * 59fcc5cc Add workflow labels and annotations global vars (#1280)
 * 1e111caa Fix bug with DockerExecutor's CopyFile (#1275)
 * 73a37f2b Add the `mergeStrategy` option to resource patching (#1269)
 * e6105243 Reduce redundancy pod label action (#1271)
 * 4bfbb20b Error running 1000s of tasks: "etcdserver: request is too large" #1186 (#1264)
 * b2743f30 Proxy Priority and PriorityClassName to pods (#1179)
 * 70c130ae Update versions (#1218)
 * b0384129 Git cloning via SSH was not verifying host public key (#1261)
 * 3f06385b Issue#1165 fake outputs don't notify and task completes successfully (#1247)
 * fa042aa2 typo, executo -> executor (#1243)
 * 1cb88bae Fixed Issue#1223 Kubernetes Resource action: patch is not supported (#1245)
 * 2b0b8f1c Fix the Prometheus address references (#1237)
 * 94cda3d5 Add feature to continue workflow on failed/error steps/tasks (#1205)
 * 3f1fb9d5 Add Gardener to "Who uses Argo" (#1228)
 * cde5cd32 Include stderr when retrieving docker logs (#1225)
 * 2b1d56e7 Update README.md (#1224)
 * eeac5a0e Remove extra quotes around output parameter value (#1232)
 * 8b67e1bf Update README.md (#1236)
 * baa3e622 Update README with typo fixes (#1220)
 * f6b0c8f2 Executor can access the k8s apiserver with a out-of-cluster config file (#1134)
 * 0bda53c7 fix dag retries (#1221)
 * 8aae2931 Issue #1190 - Fix incorrect retry node handling (#1208)
 * f1797f78 Add schedulerName to workflow and template spec (#1184)
 * 2ddae161 Set executor image pull policy for resource template (#1174)
 * edcb5629 Dockerfile: argoexec base image correction (fixes #1209) (#1213)
 * f92284d7 Minor spelling, formatting, and style updates. (#1193)
 * bd249a83 Issue #1128 - Use polling instead of fs notify to get annotation changes (#1194)
 * 14a432e7 Update community/README (#1197)
 * eda7e084 Updated OWNERS (#1198)
 * 73504a24 Fischerjulian adds ruby to rest docs (#1196)
 * 311ad86f Fix missing docker binary in argoexec image. Improve reuse of image layers
 * 831e2198 Issue #988 - Submit should not print logs to stdout unless output is 'wide' (#1192)
 * 17250f3a Add documentation how to use parameter-file's (#1191)
 * 01ce5c3b Add Docker Hub build hooks
 * 93289b42 Refactor Makefile/Dockerfile to remove volume binding in favor of build stages (#1189)
 * 8eb4c666 Issue #1123 - Fix 'kubectl get' failure if resource namespace is different from workflow namespace (#1171)
 * eaaad7d4 Increased S3 artifact retry time and added log (#1138)
 * f07b5afe Issue #1113 - Wait for daemon pods completion to handle annotations (#1177)
 * 2b2651b0 Do not mount unnecessary docker socket (#1178)
 * 1fc03144 Argo users: Equinor (#1175)
 * e381653b Update README. (#1173) (#1176)
 * 5a917140 Update README and preview notice in CLA.
 * 521eb25a Validate ArchiveLocation artifacts (#1167)
 * 528e8f80 Add missing patch in namespace kustomization.yaml (#1170)
 * 0b41ca0a Add Preferred Networks to users in README.md (#1172)
 * 649d64d1 Add GitHub to users in README.md (#1151)
 * 864c7090 Update codegen for network config (#1168)
 * c3cc51be Support HDFS Artifact (#1159)
 * 8db00066 add support for hostNetwork & dnsPolicy config (#1161)
 * 149d176f Replace exponential retry with poll (#1166)
 * 31e5f63c Fix tests compilation error (#1157)
 * 6726d9a9 Fix failing TestAddGlobalArtifactToScope unit test
 * 4fd758c3 Add slack badge to README (#1164)
 * 3561bff7 Issue #1136 - Fix metadata for DAG with loops (#1149)
 * c7fec9d4 Reflect minio chart changes in documentation (#1147)
 * f6ce7833 add support for other archs (#1137)
 * cb538489 Fix issue where steps with exhausted retires would not complete (#1148)
 * e400b65c Fix global artifact overwriting in nested workflow (#1086)
 * 174eb20a Issue #1040 - Kill daemoned step if workflow consist of single daemoned step (#1144)
 * e078032e Issue #1132 - Fix panic in ttl controller (#1143)
 * e09d9ade Issue #1104 - Remove container wait timeout from 'argo logs --follow' (#1142)
 * 0f84e514 Allow owner reference to be set in submit util (#1120)
 * 3484099c Update generated swagger to fix verify-codegen (#1131)
 * 587ab1a0 Fix output artifact and parameter conflict (#1125)
 * 6bb3adbc Adding Quantibio in Who uses Argo (#1111)
 * 1ae3696c Install mime-support in argoexec to set proper mime types for S3 artifacts (resolves #1119)
 * 515a9005 add support for ppc64le and s390x (#1102)
 * 78142837 Remove docker_lib mount volume which is not needed anymore (#1115)
 * e59398ad Fix examples docs of parameters. (#1110)
 * ec20d94b Issue #1114 - Set FORCE_NAMESPACE_ISOLATION env variable in namespace install manifests (#1116)
 * 49c1fa4f Update docs with examples using the K8s REST API
 * bb8a6a58 Update ROADMAP.md
 * 46855dcd adding logo to be used by the OS Site (#1099)
 * 438330c3 #1081 added retry logic to s3 load and save function (#1082)
 * cb8b036b Initialize child node before marking phase. Fixes panic on invalid `When` (#1075)
 * 60b508dd Drop reference to removed `argo install` command. (#1074)
 * 62b24368 Fix typo in demo.md (#1089)
 * b5dfa021 Use relative links on README file (#1087)
 * 95b72f38 Update docs to outline bare minimum set of privileges for a workflow
 * d4ef6e94 Add new article and minor edits. (#1083)
 * afdac9bb Issue #740 - System level workflow parallelism limits & priorities (#1065)
 * a53a76e9 fix #1078 Azure AKS authentication issues (#1079)
 * 79b3e307 Fix string format arguments in workflow utilities. (#1070)
 * 76b14f54 Auto-complete workflow names (#1061)
 * f2914d63 Support nested steps workflow parallelism (#1046)
 * eb48c23a Raise not implemented error when artifact saving is unsupported (#1062)
 * 036969c0 Add Cratejoy to list of users (#1063)
 * a07bbe43 Adding SAP Hybris in Who uses Argo (#1064)
 * 7ef1cea6 Update dependencies to K8s v1.12 and client-go 9.0
 * 23d733ba Add namespace explicitly to pod metadata (#1059)
 * 79ed7665 Parameter and Argument names should support snake case (#1048)
 * 6e6c59f1 Submodules are dirty after checkout -- need to update (#1052)
 * f18716b7 Support for K8s API based Executor (#1010)
 * e297d195 Updated examples/README.md (#1051)
 * 19d6cee8 Updated ARTIFACT_REPO.md (#1049)
 * 0a928e93 Update installation manifests to use v2.2.1
 * 3b52b261 Fix linter warnings and update swagger
 * 7d0e77ba Update changelog and bump version to 2.2.1
 * b402e12f Issue #1033 - Workflow executor panic: workflows.argoproj.io/template workflows.argoproj.io/template not found in annotation file (#1034)
 * 3f2e986e fix typo in examples/README.md (#1025)
 * 9c5e056a Replace tabs with spaces (#1027)
 * 091f1407 Update README.md (#1030)
 * 159fe09c Fix format issues to resolve build errors (#1023)
 * 363bd97b Fix error in env syntax (#1014)
 * ae7bf0a5 Issue #1018 - Workflow controller should save information about archived logs in step outputs (#1019)
 * 15d006c5 Add example of workflow using imagePullSecrets (resolves #1013)
 * 2388294f Fix RBAC roles to include workflow delete for GC to work properly (resolves #1004)
 * 6f611cb9 Fix issue where resubmission of a terminated workflow creates a terminated workflow (issue #1011)
 * 4a7748f4 Disable Persistence in the demo example (#997)
 * 55ae0cb2 Fix example pod name (#1002)
 * c275e7ac Add imagePullPolicy config for executors (#995)
 * b1eed124 `tar -tf` will detect compressed tars correctly. (#998)
 * 03a7137c Add new organization using argo (#994)
 * 83884528 Update argoproj/pkg to trim leading/trailing whitespace in S3 credentials (resolves #981)
 * 978b4938 Add syntax highlighting for all YAML snippets and most shell snippets (#980)
 * 60d5dc11 Give control to decide whether or not to archive logs at a template level
 * 8fab73b1 Detect and indicate when container was OOMKilled
 * 47a9e556 Update config map doc with instructions to enable log archiving
 * 79dbbaa1 Add instructions to match git URL format to auth type in git example (issue #979)
 * 429f03f5 Add feature list to README.md. Tweaks to getting started.
 * 36fd1948 Update getting started guide with v2.2.0 instructions
 * af636ddd Update installation manifests to use v2.2.0
 * 7864ad36 Introduce `withSequence` to iterate a range of numbers in a loop (resolves #527)
 * 99e9977e Introduce `argo terminate` to terminate a workflow without deleting it (resolves #527)
 * f52c0450 Reorganize codebase to make CLI functionality available as a library
 * 311169f7 Fix issue where sidecars and daemons were not reliably killed (resolves #879)
 * 67ffb6eb Add a reason/message for Unschedulable Pending pods
 * 69c390f2 Support for workflow level timeouts (resolves #848)
 * f88732ec Update docs to use keyFormat field
 * 0df022e7 Rename keyPattern to keyFormat. Remove pending pod query during pod reconciliation
 * 75a9983b Fix potential panic in `argo watch`
 * 9cb46449 Add TTLSecondsAfterFinished field and controller to garbage collect completed workflows (resolves #911)
 * 7540714a Add ability to archive container logs to the artifact repository (resolves #454)
 * 11e57f4d Introduce archive strategies with ability to disable tar.gz archiving (resolves #784)
 * e180b547 Update CHANGELOG.md
 * 5670bf5a Introduce `argo watch` command to watch live workflows from terminal (resolves #969)
 * 57394361 Support additional container runtimes through kubelet executor (#952)
 * a9c84c97 Error workflows which hit k8s/etcd 1M resource size limit (resolves #913)
 * 67792eb8 Add parameter-file support (#966)
 * 841832a3 Aggregate workflow RBAC roles to built-in admin/edit/view clusterroles (resolves #960)
 * 35bb7093 Allow scaling of workflow and pod workers via controller CLI flags (resolves #962)
 * b479fa10 Improve workflow configmap documentation for keyPattern
 * f1802f91 Introduce `keyPattern` workflow config to enable flexibility in archive location path (issue #953)
 * a5648a96 Fix kubectl proxy link for argo-ui Service (#963)
 * 09f05912 Introduce Pending node state to highlight failures when start workflow pods
 * a3ff464f Speed up CI job
 * 88627e84 Update base images to debian:9.5-slim. Use stable metalinter
 * 753c5945 Update argo-ci-builder image with new dependencies
 * 674b61bb Remove unnecessary dependency on argo-cd and obsolete RBAC constants
 * 60658de0 Refactor linting/validation into standalone package. Support linting of .json files
 * f55d579a Detect and fail upon unknown fields during argo submit & lint (resolves #892)
 * edf6a574 Migrate to using argoproj.io/pkg packages
 * 5ee1e0c7 Update artifact config docs (#957)
 * faca49c0 Updated README
 * 936c6df7 Add table of content to examples (#956)
 * d2c03f67 Correct image used in install manifests
 * ec3b7be0 Remove CLI installer/uninstaller. Executor image configured via CLI argument (issue #928) Remove redundant/unused downward API metadata
 * 3a85e242 Rely on `git checkout` instead of go-git checkout for more reliable revision resolution
 * ecef0e3d Rename Prometheus metrics (#948)
 * b9cffe9c Issue #896 - Prometheus metrics and telemetry (#935)
 * 290dee52 Support parameter aggregation of map results in scripts
 * fc20f5d7 Fix errors when submodules are from different URL (#939)
 * b4f1a00a Add documentation about workflow variables
 * 4a242518 Update readme.md (#943)
 * a5baca60 Support referencing of global workflow artifacts (issue #900)
 * 9b5c8563 Support for sophisticated expressions in `when` conditionals (issue #860)
 * ecc0f027 Resolve revision added ability to specify shorthand revision and other things like HEAD~2 etc (#936)
 * 11024318 Support conditions with DAG tasks. Support aggregated outputs from scripts (issue #921)
 * d07c1d2f Support withItems/withParam and parameter aggregation with DAG templates (issue #801)
 * 94c195cb Bump VERSION to v2.2.0
 * 9168c59d Fix outbound node metadata with retry nodes causing disconnected nodes to be rendered in UI (issue #880)
 * c6ce48d0 Fix outbound node metadata issue with steps template causing incorrect edges to be rendered in UI
 * 520b33d5 Add ability to aggregate and reference output parameters expanded by loops (issue #861)
 * ece1eef8 Support submission of workflows as json, and from stdin (resolves #926)
 * 4c31d61d Add `argo delete --older` to delete completed workflows older than specified duration (resolves #930)
 * c87cd33c Update golang version to v1.10.3
 * 618b7eb8 Proper fix for assessing overall DAG phase. Add some DAG unit tests (resolves #885)
 * f223e5ad Fix issue where a DAG would fail even if retry was successful (resolves #885)
 * 143477f3 Start use of argoproj/pkg shared libraries
 * 1220d080 Update argo-cluster-role to work with OpenShift (resolves #922)
 * 4744f45a Added SSH clone and proper git clone using go-git (#919)
 * d657abf4 Regenerate code and address OpenAPI rule validation errors (resolves #923)
 * c5ec4cf6 Upgrade k8s dependencies to v1.10 (resolves #908)
 * ba8061ab Redundant verifyResolvedVariables check in controller precluded the ability to use {{ }} in other circumstances
 * 05a61449 Added link to community meetings (#912)
 * f33624d6 Add an example on how to submit and wait on a workflow
 * aeed7f9d Added new members
 * 288e4fc8 Added Argo Events link.
 * 3322506e Updated README
 * 3ce640a2 Issue #889 - Support retryStrategy for scripts (#890)
 * 91c6afb2 adding BlackRock as corporate contributor/user (#886)
 * c8667b5c Fix issue where `argo lint` required spec level arguments to be supplied
 * ed7dedde Update influx-ci example to choose a stable InfluxDB branch
 * 135813e1 Add datadog to the argo users (#882)
 * f1038948 Fix `make verify-codegen` build target when run in CI
 * 785f2cbd Update references to v2.1.1. Add better checks in release Makefile target
 * d65e1cd3 readme: add Interline Technologies to user list (#867)
 * c903168e Add documentation on global parameters (#871)

### Contributors

 * 0x1D-1983
 * Aaron Curtis
 * Adam Gilat
 * Adam Thornton
 * Aditya Sundaramurthy
 * Adrien Trouillaud
 * Aisuko
 * Akshay Chitneni
 * Alessandro Marrella
 * Alex Capras
 * Alex Collins
 * Alex Stein
 * Alexander Matyushentsev
 * Alexander Zigelski
 * Alexey Volkov
 * Anastasia Satonina
 * Andrei Miulescu
 * Andrew Suderman
 * Anes Benmerzoug
 * Ang Gao
 * Anna Winkler
 * Antoine Dao
 * Antonio Macías Ojeda
 * Appréderisse Benjamin
 * Avi Weit
 * Bastian Echterhölter
 * Ben Wells
 * Ben Ye
 * Brandon Steinman
 * Brian Mericle
 * CWen
 * Caden
 * Caglar Gulseni
 * Carlos Montemuino
 * Chen Zhiwei
 * Chris Chambers
 * Chris Hepner
 * Christian Muehlhaeuser
 * Clemens Lange
 * Cristian Pop
 * Daisuke Taniwaki
 * Dan Norris
 * Daniel Duvall
 * Daniel Moran
 * Daniel Sutton
 * David Bernard
 * David Seapy
 * David Van Loon
 * Deepen Mehta
 * Derek Wang
 * Dineshmohan Rajaveeran
 * Divya Vavili
 * Drew Dara-Abrams
 * Dustin Specker
 * EDGsheryl
 * Ed Lee
 * Edward Lee
 * Edwin Jacques
 * Ejiah
 * Elton
 * Erik Parmann
 * Fabio Rigato
 * Feynman Liang
 * Florent Clairambault
 * Fred Dubois
 * Gabriele Santomaggio
 * Galen Han
 * Grant Stephens
 * Greg Roodt
 * Guillaume Hormiere
 * Hamel Husain
 * Heikki Kesa
 * Hideto Inamura
 * Howie Benefiel
 * Huan-Cheng Chang
 * Ian Howell
 * Ilias K
 * Ilias Katsakioris
 * Ilya Sotkov
 * Ismail Alidzhikov
 * Jacob O'Farrell
 * Jaime
 * Jared Welch
 * Jean-Louis Queguiner
 * Jeff Uren
 * Jesse Suen
 * Jialu Zhu
 * Jie Zhang
 * Johannes 'fish' Ziemke
 * John Wass
 * Jonathan Steele
 * Jonathon Belotti
 * Jonny
 * Joshua Carp
 * Juan C. Muller
 * Julian Fahrer
 * Julian Fischer
 * Julian Mazzitelli
 * Julien Balestra
 * Kannappan Sirchabesan
 * Kaushik B
 * Konstantin Zadorozhny
 * Leonardo Luz
 * Lucas Theisen
 * Marcin Karkocha
 * Marco Sanvido
 * Marek Čermák
 * Markus Lippert
 * Matt Brant
 * Matt Hillsdon
 * Matthew Coleman
 * Matthew Magaldi
 * MengZeLee
 * Michael Crenshaw
 * Michael Weibel
 * Mike Seddon
 * Mingjie Tang
 * Miyamae Yuuya
 * Mostapha Sadeghipour Roudsari
 * Mukulikak
 * Naoto Migita
 * Naresh Kumar Amrutham
 * Nasrudin Bin Salim
 * Neutron Soutmun
 * Nick Groszewski
 * Nick Stott
 * Niklas Hansson
 * Niklas Vest
 * Nirav Patel
 * Noorain Panjwani
 * Nándor István Krácser
 * Omer Kahani
 * Orion Delwaterman
 * Pablo Osinaga
 * Pascal VanDerSwalmen
 * Patryk Jeziorowski
 * Paul Brit
 * Pavel Kravchenko
 * Peng Li
 * Pengfei Zhao
 * Per Buer
 * Peter Salanki
 * Pierre Houssin
 * Pradip Caulagi
 * Praneet Chandra
 * Pratik Raj
 * Premkumar Masilamani
 * Rafael Rodrigues
 * Rafał Bigaj
 * Remington Breeze
 * Rick Avendaño
 * Rocio Montes
 * Romain Di Giorgio
 * Romain GUICHARD
 * Roman Galeev
 * Saradhi Sreegiriraju
 * Saravanan Balasubramanian
 * Sascha Grunert
 * Sean Fern
 * Sebastian Ortan
 * Semjon Kopp
 * Shannon
 * Shubham Koli (FaultyCarry)
 * Simon Behar
 * Simon Frey
 * Snyk bot
 * Song Juchao
 * Stephen Steiner
 * StoneHuang
 * Takashi Abe
 * Takayuki Kasai
 * Tang Lee
 * Theodore Messinezis
 * Tim Schrodi
 * Tobias Bradtke
 * Tom Wieczorek
 * Tomas Valasek
 * Trevor Foster
 * Tristan Colgate-McFarlane
 * Val Sichkovskyi
 * Vardan Manucharyan
 * Vincent Boulineau
 * Vincent Smith
 * Vlad Losev
 * Wei Yan
 * WeiYan
 * Weston Platter
 * William
 * William Reed
 * Xianlu Bird
 * Xie.CS
 * Xin Wang
 * Youngjoon Lee
 * Yuan Tang
 * Yunhai Luo
 * Zach Aller
 * Zach Himsel
 * Zadjad Rezai
 * Zhipeng Wang
 * Ziyang Wang
 * alex weidner
 * almariah
 * candonov
 * commodus-sebastien
 * descrepes
 * dgiebert
 * dherman
 * dmayle
 * dthomson25
 * fsiegmund
 * gerardaus
 * gerdos82
 * haibingzhao
 * hidekuro
 * houz
 * ianCambrio
 * jacky
 * jdfalko
 * joe
 * juliusvonkohout
 * kshamajain99
 * lueenavarro
 * maguowei
 * mark9white
 * maryoush
 * mdvorakramboll
 * nglinh
 * sang
 * sh-tatsuno
 * shahin
 * shibataka000
 * tkilpela
 * tralexa
 * tunoat
 * vatine
 * vdinesh2461990
 * xubofei1983
 * yonirab
 * zhujl1991
 * モハメド

## v2.1.1 (2018-05-29)

 * ac241c95 Update CHANGELOG for v2.1.1
 * 468e0760 Retrying failed steps templates could potentially result in disconnected children
 * 8d96ea7b Switch to an UnstructuredInformer to guard against malformed workflow manifests (resolves #632)
 * 5bef6cae Suspend templates were not properly being connected to their children (resolves #869)
 * 543e9392 Fix issue where a failed step in a template with parallelism would not complete (resolves #868)
 * 289000ca Update argocli Dockerfile and make cli-image part of release
 * d35a1e69 Bump version to v2.1.1
 * bbcff0c9 Fix issue where `argo list` age column maxed out at 1d (resolves #857)
 * d68cfb7e Fix issue where volumes were not supported in script templates (resolves #852)
 * fa72b6db Fix implementation of DAG task targets (resolves #865)
 * dc003f43 Children of nested DAG templates were not correctly being connected to its parent
 * b8065797 Simplify some examples for readability and clarity
 * 7b02c050 Add CoreFiling to "Who uses Argo?" section. (#864)
 * 4f2fde50 Add windows support for argo-cli (#856)
 * 703241e6 Updated ROADMAP.md for v2.2
 * 54f2138e Spell check the examples README (#855)
 * f23feff5 Mkbranch (#851)
 * 628b5408 DAG docs. (#850)
 * 22f62439 Small edit to README
 * edc09afc Added OWNERS file
 * 530e7244 Update release notes and documentation for v2.1.0
 * 93796381 Avoid `println` which outputs to stderr. (#844)
 * 30e472e9 Add gladly as an official argo user (#843)
 * cb4c1a13 Add ability to override metadata.name and/or metadata.generateName during submission (resolves #836)
 * 834468a5 Command print the logs for a container in a workflow
 * 1cf13f9b Issue #825 - fix locating outbound nodes for skipped node (#842)
 * 30034d42 Bump from debian:9.1 to debian:9.4. (#841)
 * f3c41717 Owner reference example (#839)
 * 191f7aff Minor edit to README
 * c8a2e25f Fixed typo (#835)
 * cf13bf0b Added users section to README
 * e4d76329 Updated News in README
 * b631d0af added community meeting (#834)
 * e34728c6 Fix issue where daemoned steps were not terminated properly in DAG templates (resolves #832)
 * 2e9e113f Update docs to work with latest minio chart
 * ea95f191 Use octal syntax for mode values (#833)
 * 5fc67d2b Updated community docs
 * 8fa4f006 Added community docs
 * 423c8d14 Issue #830 - retain Step node children references
 * 73990c78 Moved cricket gifs to a different s3 bucket
 * ca1858ca edit Argo license info so that GitHub recognizes it (#823)
 * 206451f0 Fix influxdb-ci.yml example
 * da582a51 Avoid nil pointer for 2.0 workflows. (#820)
 * 0f225cef ClusterRoleBinding was using incorrect service account namespace reference when overriding install namespace (resolves #814)
 * 66ea711a Issue #816 - fix updating outboundNodes field of failed step group node (#817)
 * 00ceef6a install & uninstall commands use --namespace flag (#813)

### Contributors

 * Adam Pearse
 * Alexander Matyushentsev
 * Andrea Kao
 * Edward Lee
 * Eric
 * Javier Castellanos
 * Jesse Suen
 * Jonas Fonseca
 * Lukasz Lempart
 * Matt Hillsdon
 * Mukulikak
 * Sean Fitzgerald
 * Sebastien Doido

## v2.1.0-beta2 (2018-03-29)

 * fe23c2f6 Issue #810 - `argo install`does not install argo ui (#811)
 * 28673ed2 Update release date in change log

### Contributors

 * Alexander Matyushentsev

## v2.1.0-beta1 (2018-03-29)

 * 05e8a983 Update change log for 2.1.0-beta1 release
 * bf38b6b5 Use socket type for hostPath to mount docker.sock (#804) (#809)
 * 37680ef2 Minimal shell completion support (#807)
 * c83ad24a Omit empty status fields. (#806)
 * d7291a3e Issue #660 - Support rendering logs from all steps using 'argo logs' command (#792)
 * 7d3f1e83 Minor edits to README
 * 7a4c9c1f Added a review to README
 * 383276f3 Inlined LICENSE file. Renamed old license to COPYRIGHT
 * 91d0f47f Build argo cli image (#800)
 * 3b2c426e Add ability to pass pod annotations and labels at the template level (#798)
 * d8be0287 Add ability to use IAM role from EC2 instance for AWS S3 credentials
 * 624f0f48 Update CHANGELOG.md for v2.1.0-beta1 release
 * e96a09a3 Allow spec.arguments to be not supplied during linting. Global parameters were not referencable from artifact arguments (resolves #791)
 * 018e663a Fix for https://github.com/argoproj/argo/issues/739 Nested stepgroups render correctly (#790)
 * 5c5b35ba Fix install issue where service account was not being created
 * 88e9e5ec packr needs to run compiled in order to cross compile darwin binaries
 * dcdf9acf Fix install tests and build failure
 * 06c0d324 Rewrite the installer such that manifests are maintainable
 * a45bf1b7 Introduce support for exported global output parameters and artifacts
 * 60c48a9a Introduce `argo retry` to retry a failed workflow with the same name (resolves #762) onExit and related nodes should never be executed during resubmit/retry (resolves #780)
 * 90c08bff Refactor command structure
 * 101509d6 Abstract the container runtime as an interface to support mocking and future runtimes Trim a trailing newline from path-based output parameters (resolves #758)
 * a3441d38 Add ability to reference global parameters in spec level fields (resolves #749)
 * cd73a9ce Fix template.parallelism limiting parallelism of entire workflow (resolves #772) Refactor operator to make template execution method signatures consistent
 * 7d7b74fa Make {{pod.name}} available as a parameter in pod templates (resolves #744)
 * 3cf4bb13 parse the artifactory URL before appending the artifact to the path (#774)
 * ea1257f7 examples: use alpine python image
 * 2114078c fix typo
 * 9f605589 Fix retry-container-to-completion example
 * 07422f26 Update CHANGELOG release date. Remove ui-image from release target

### Contributors

 * Alexander Matyushentsev
 * Dmitry Monakhov
 * Edward Lee
 * Jesse Suen
 * Johannes 'fish' Ziemke
 * Lukasz Lempart
 * Matt Hillsdon
 * Yang Pan
 * dougsc
 * gaganapplatix

## v2.1.0-alpha1 (2018-02-21)


### Contributors


## v2.10.2 (2020-09-14)

 * ed79a540 Update manifests to v2.10.2
 * d27bf2d2 fix: Fix UI selection issues (#3928)
 * 51220389 fix: Create global scope before workflow-level realtime metrics (#3979)
 * 857ef750 fix: Custom metrics are not recorded for DAG tasks Fixes #3872 (#3886)
 * b9a0bb00 fix: Consider WorkflowTemplate metadata during validation (#3988)
 * 089e1862 fix(server): Remove XSS vulnerability. Fixes #3942 (#3975)
 * 1215d9e1 fix(cli): Allow `argo version` without KUBECONFIG. Fixes #3943 (#3945)

### Contributors

 * Alex Collins
 * Saravanan Balasubramanian
 * Simon Behar

## v2.10.1 (2020-09-02)

 * 854444e4 Update manifests to v2.10.1
 * 69861fc9 fix: Workflow should fail on Pod failure before container starts Fixes #3879 (#3890)
 * 670fc618 fix(controller): Cron re-apply update (#3883)
 * 4b30fa4e fix(executor): Replace default retry in executor with an increased value retryer (#3891)
 * ae537cd7 fix(ui): use absolute URL to redirect from autocomplete list. Closes #3903 (#3906)
 * 0e4b3902 chore: Added unittest for PVC exceed quota Closes #3561 (#3860)
 * 56dc9f7a fix: Consider all children of TaskGroups in DAGs (#3740)
 * 8ac7369b fix(controller): Support exit handler on workflow templates.  Fixes #3737 (#3782)
 * ee848921 fix(controller): Failure tolerant workflow archiving and offloading. Fixes #3786 and #3837 (#3787)
 * c367d93e build: Copy Dockerfile lines from master
 * f491bbc6 chore: Merge .gitignore from master to prevent dirty
 * 195c6d83 Update manifests to v2.10.0
 * 08117f0c fix: Increase the requeue duration on checkForbiddenErrorAndResubmitAllowed (#3794)
 * 5ea2ed0d fix(server): Trucate creator label at 63 chars. Fixes #3756 (#3758)

### Contributors

 * Alex Collins
 * Ang Gao
 * Nirav Patel
 * Saravanan Balasubramanian
 * Simon Behar

## v2.10.0-rc7 (2020-08-13)

 * 267da535 Update manifests to v2.10.0-rc7
 * baeb0fed fix: Revert merge error
 * 66bae22f fix(executor): Add retry on pods watch to handle timeout. (#3675)
 * 971f1153 removed unused test-report files
 * 8c0b9f0a fix: Couldn't Terminate/Stop the ResourceTemplate Workflow (#3679)
 * a04d72f9 fix(controller): Tolerate PDB delete race. Fixes #3706 (#3717)
 * a7635763 fix: Fix bug with 'argo delete --older' (#3699)
 * fe8129cf fix(controller): Carry-over labels for re-submitted workflows. Fixes #3622 (#3638)
 * e12d26e5 fix(controller): Treat TooManyError same as Forbidden (i.e. try again). Fixes #3606 (#3607)
 * 9a5febec fix: Ensure target task's onExit handlers are run (#3716)
 * c3a58e36 fix: Enforce metric Help must be the same for each metric Name (#3613)

### Contributors

 * Alex Collins
 * Guillaume Hormiere
 * Saravanan Balasubramanian
 * Simon Behar

## v2.10.0-rc6 (2020-08-06)

 * cb3536f9 Update manifests to v2.10.0-rc6
 * 6e004ace lint
 * b31fc1f8 fix(controller): Adds ALL_POD_CHANGES_SIGNIFICANT (#3689)
 * 0b7cd5b3 fix: Fixed workflow queue duration if PVC creation is forbidden (#3691)
 * 03b84162 fix: Re-introduce 1 second sleep to reconcile informer (#3684)

### Contributors

 * Alex Collins
 * Saravanan Balasubramanian
 * Simon Behar

## v2.10.0-rc5 (2020-08-03)

 * e9ca55ec Update manifests to v2.10.0-rc5
 * 85ddda05 lint
 * fb367f5e fix(controller): Fix nested maps. Fixes #3653 (#3661)
 * 2385cca5 fix: interface{} values should be expanded with '%v' (#3659)
 * 263e4bad fix(server): Report v1.Status errors. Fixes #3608 (#3652)
 * 718f802b fix: Avoid overriding the Workflow parameter when it is merging with WorkflowTemplate parameter (#3651)
 * 9735df32 fix: Fixed flaky unit test TestFailSuspendedAndPendingNodesAfterDeadline (#3640)
 * 662d22e4 fix: Don't panic on invalid template creation (#3643)
 * 854aaefa fix: Fix 'malformed request: field selector' error (#3636)
 * 9d56eb29 fix: DAG level Output Artifacts on K8S and Kubelet executor (#3624)
 * c7512b6c fix: Simplify the WorkflowTemplateRef field validation to support all fields in WorkflowSpec except `Templates` (#3632)

### Contributors

 * Alex Collins
 * Saravanan Balasubramanian
 * Simon Behar

## v2.10.0-rc4 (2020-07-28)

 * 8d6dae61 Update manifests to v2.10.0-rc4
 * a4b1dde5 build(cli)!: Zip binaries binaries. Closes #3576 (#3614)
 * dea03a9c fix(server): Re-establish watch on v1.Status errors. Fixes #3608 (#3609)
 * c063f9f1 fix: Fix panic and provide better error message on watch endpoint (#3605)
 * 35a00498 fix: Argo Workflows does not honour global timeout if step/pod is not able to schedule (#3581)
 * 3879827c fix(controller): Fix bug in util/RecoverWorkflowNameFromSelectorString. Add error handling (#3596)
 * 5f4dec75 fix(ui): Fix multiple UI issues (#3573)
 * e94cf8a2 fix(ui): cannot push to nil when filtering by label (#3555)
 * 61b5bd93 fix: Fix flakey TestRetryOmitted (#3552)
 * d53c883b fix: Fix links in fields doc (#3539)
 * d2bd5879 fix(artifacts): support optional input artifacts, Fixes #3491 (#3512)
 * 652956e0 fix: Fix when retrying Workflows with Omitted nodes (#3528)
 * 32c36d78 fix(controller): Backoff exponent is off by one. Fixes #3513 (#3514)
 * 75d29574 fix: String interpreted as boolean in labels (#3518)

### Contributors

 * Alex Collins
 * Jie Zhang
 * Jonny
 * Remington Breeze
 * Saravanan Balasubramanian
 * Simon Behar
 * haibingzhao

## v2.10.0-rc3 (2020-07-23)

 * 37f4f9da Update manifests to v2.10.0-rc3
 * 37297af7 Update manifests to v2.10.0-rc2
 * cbf27edf fix: Panic on CLI Watch command (#3532)
 * a3666482 fix: Skip TestStorageQuotaLimit (#3566)
 * 802c18ed fix: Exceeding quota with volumeClaimTemplates (#3490)
 * 94b20124 ci: Make builds marginally faster. Fixes #3515 (#3519)
 * 38caab7c chore: `make lint`
 * bbee82a0 fix(server): Ignore not-JWT server tokens. Fixes #3562 (#3579)
 * f72ae881 fix(controller): Do not panic on nil output value. Fixes #3505 (#3509)
 * 18c7440f build: Fix version and do not push images. Fixes #3515
 * 66d6f964 build: Fix version and do not push images. Fixes #3515

### Contributors

 * Alex Collins
 * Saravanan Balasubramanian

## v2.10.0-rc2 (2020-07-18)

 * 4bba17f3 Update manifests to v2.10.0-rc2
 * 953f50e4 build: Fix version and do not push images. Fixes #3515
 * 616c79df Update manifests to v2.10.0-rc1

### Contributors

 * Alex Collins

## v2.10.0-rc1 (2020-07-17)


### Contributors


## v2.10.0 (2020-08-18)

 * 195c6d83 Update manifests to v2.10.0
 * 08117f0c fix: Increase the requeue duration on checkForbiddenErrorAndResubmitAllowed (#3794)
 * 5ea2ed0d fix(server): Trucate creator label at 63 chars. Fixes #3756 (#3758)
 * 267da535 Update manifests to v2.10.0-rc7
 * baeb0fed fix: Revert merge error
 * 66bae22f fix(executor): Add retry on pods watch to handle timeout. (#3675)
 * 971f1153 removed unused test-report files
 * 8c0b9f0a fix: Couldn't Terminate/Stop the ResourceTemplate Workflow (#3679)
 * a04d72f9 fix(controller): Tolerate PDB delete race. Fixes #3706 (#3717)
 * a7635763 fix: Fix bug with 'argo delete --older' (#3699)
 * fe8129cf fix(controller): Carry-over labels for re-submitted workflows. Fixes #3622 (#3638)
 * e12d26e5 fix(controller): Treat TooManyError same as Forbidden (i.e. try again). Fixes #3606 (#3607)
 * 9a5febec fix: Ensure target task's onExit handlers are run (#3716)
 * c3a58e36 fix: Enforce metric Help must be the same for each metric Name (#3613)
 * cb3536f9 Update manifests to v2.10.0-rc6
 * 6e004ace lint
 * b31fc1f8 fix(controller): Adds ALL_POD_CHANGES_SIGNIFICANT (#3689)
 * 0b7cd5b3 fix: Fixed workflow queue duration if PVC creation is forbidden (#3691)
 * 03b84162 fix: Re-introduce 1 second sleep to reconcile informer (#3684)
 * e9ca55ec Update manifests to v2.10.0-rc5
 * 85ddda05 lint
 * fb367f5e fix(controller): Fix nested maps. Fixes #3653 (#3661)
 * 2385cca5 fix: interface{} values should be expanded with '%v' (#3659)
 * 263e4bad fix(server): Report v1.Status errors. Fixes #3608 (#3652)
 * 718f802b fix: Avoid overriding the Workflow parameter when it is merging with WorkflowTemplate parameter (#3651)
 * 9735df32 fix: Fixed flaky unit test TestFailSuspendedAndPendingNodesAfterDeadline (#3640)
 * 662d22e4 fix: Don't panic on invalid template creation (#3643)
 * 854aaefa fix: Fix 'malformed request: field selector' error (#3636)
 * 9d56eb29 fix: DAG level Output Artifacts on K8S and Kubelet executor (#3624)
 * c7512b6c fix: Simplify the WorkflowTemplateRef field validation to support all fields in WorkflowSpec except `Templates` (#3632)
 * 8d6dae61 Update manifests to v2.10.0-rc4
 * a4b1dde5 build(cli)!: Zip binaries binaries. Closes #3576 (#3614)
 * dea03a9c fix(server): Re-establish watch on v1.Status errors. Fixes #3608 (#3609)
 * c063f9f1 fix: Fix panic and provide better error message on watch endpoint (#3605)
 * 35a00498 fix: Argo Workflows does not honour global timeout if step/pod is not able to schedule (#3581)
 * 3879827c fix(controller): Fix bug in util/RecoverWorkflowNameFromSelectorString. Add error handling (#3596)
 * 5f4dec75 fix(ui): Fix multiple UI issues (#3573)
 * e94cf8a2 fix(ui): cannot push to nil when filtering by label (#3555)
 * 61b5bd93 fix: Fix flakey TestRetryOmitted (#3552)
 * d53c883b fix: Fix links in fields doc (#3539)
 * d2bd5879 fix(artifacts): support optional input artifacts, Fixes #3491 (#3512)
 * 652956e0 fix: Fix when retrying Workflows with Omitted nodes (#3528)
 * 32c36d78 fix(controller): Backoff exponent is off by one. Fixes #3513 (#3514)
 * 75d29574 fix: String interpreted as boolean in labels (#3518)
 * 37f4f9da Update manifests to v2.10.0-rc3
 * 37297af7 Update manifests to v2.10.0-rc2
 * cbf27edf fix: Panic on CLI Watch command (#3532)
 * a3666482 fix: Skip TestStorageQuotaLimit (#3566)
 * 802c18ed fix: Exceeding quota with volumeClaimTemplates (#3490)
 * 94b20124 ci: Make builds marginally faster. Fixes #3515 (#3519)
 * 38caab7c chore: `make lint`
 * bbee82a0 fix(server): Ignore not-JWT server tokens. Fixes #3562 (#3579)
 * f72ae881 fix(controller): Do not panic on nil output value. Fixes #3505 (#3509)
 * 18c7440f build: Fix version and do not push images. Fixes #3515
 * 66d6f964 build: Fix version and do not push images. Fixes #3515
 * 4bba17f3 Update manifests to v2.10.0-rc2
 * 953f50e4 build: Fix version and do not push images. Fixes #3515
 * 616c79df Update manifests to v2.10.0-rc1
 * 19e700a3 fix(cli): Check mutual exclusivity for argo CLI flags (#3493)
 * 7d45ff7f fix: Panic on releaseAllWorkflowLocks if Object is not Unstructured type (#3504)
 * 1b68a5a1 fix: ui/package.json & ui/yarn.lock to reduce vulnerabilities (#3501)
 * 7f262fd8 fix(cli)!: Enable CLI to work without kube config. Closes #3383, #2793 (#3385)
 * 2976e7ac build: Clear cmd docs before generating them (#3499)
 * 27528ba3 feat: Support completions for more resources (#3494)
 * 5bd2ad7a fix: Merge WorkflowTemplateRef with defaults workflow spec (#3480)
 * e244337b chore: Added examples for exit handler for step and dag level (#3495)
 * bcb32547 build: Use `git rev-parse` to accomodate older gits (#3497)
 * 3eb6e2f9 docs: Add link to GitHub Actions in the badge (#3492)
 * 69179e72 fix: link to server auth mode docs, adds Tulip as official user (#3486)
 * 7a8e2b34 docs: Add comments to NodePhase definition. Closes #1117. (#3467)
 * 24d1e529 build: Simplify builds (#3478)
 * acf56f9f feat(server): Label workflows with creator. Closes #2437 (#3440)
 * 3b8ac065 fix: Pass resolved arguments to onExit handler (#3477)
 * 58097a9e docs: Add controller-level metrics (#3464)
 * f6f1844b feat: Attempt to resolve nested tags (#3339)
 * 48e15d6f feat(cli): List only resubmitted workflows option (#3357)
 * 25e9c0cd docs, quick-start. Use http, not https for link (#3476)
 * 7a2d7642 fix: Metric emission with retryStrategy (#3470)
 * f5876e04 test(controller): Ensure resubmitted workflows have correct labels (#3473)
 * aa92ec03 fix(controller): Correct fail workflow when pod is deleted with --force. Fixes #3097 (#3469)
 * a1945d63 fix(controller): Respect the volumes of a workflowTemplateRef. Fixes … (#3451)
 * 847ba530 test(controller): Add memoization tests. See #3214 (#3455) (#3466)
 * f5183aed docs: Fix CLI docs (#3465)
 * 1e42813a test(controller): Add memoization tests. See #3214 (#3455)
 * abe768c4 feat(cli): Allow to view previously terminated container logs (#3423)
 * 7581025f fix: Allow ints for sequence start/end/count. Fixes #3420 (#3425)
 * b82f900a Fixed typos (#3456)
 * 23760119 feat: Workflow Semaphore Support (#3141)
 * 81cba832 feat: Support WorkflowMetadata in WorkflowTemplate and ClusterWorkflowTemplate (#3364)
 * 568c032b chore: update aws-sdk-go version (#3376)
 * bd27d9f3 chore: Upgrade node-sass (#3450)
 * b1e601e5 docs: typo in argo stop --help (#3439)
 * 308c7083 fix(controller): Prevent panic on nil node. Fixes #3436 (#3437)
 * 8ab06f53 feat(controller): Add log message count as metrics. (#3362)
 * 5d0c436d chore: Fix GitHub Actions Docker Image build  (#3442)
 * e54b4ab5 docs: Add Sohu as official Argo user (#3430)
 * ee6c8760 fix: Ensure task dependencies run after onExit handler is fulfilled (#3435)
 * 6dc04b39 chore: Use GitHub Actions to build Docker Images to allow publishing Windows Images (#3291)
 * 05b3590b feat(controller): Add support for Docker workflow executor for Windows nodes (#3301)
 * 676868f3 fix(docs): Update kubectl proxy URL (#3433)
 * 3507c3e6 docs: Make https://argoproj.github.io/argo/  (#3369)
 * 733e95f7 fix: Add struct-wide RWMutext to metrics (#3421)
 * 0463f241 fix: Use a unique queue to visit nodes (#3418)
 * eddcac63 fix: Script steps fail with exceededQuota (#3407)
 * c631a545 feat(ui): Add Swagger UI (#3358)
 * 910f636d fix: No panic on watch. Fixes #3411 (#3426)
 * b4da1bcc fix(sso): Remove unused `groups` claim. Fixes #3411 (#3427)
 * 330d4a0a fix: panic on wait command if event is null (#3424)
 * 7c439424 docs: Include timezone name reference (#3414)
 * 03cbb8cf fix(ui): Render DAG with exit node (#3408)
 * 3d50f985 feat: Expose certain queue metrics (#3371)
 * c7b35e05 fix: Ensure non-leaf DAG tasks have their onExit handler's run (#3403)
 * 70111600 fix: Fix concurrency issues with metrics (#3401)
 * d307f96f docs: Update config example to include useSDKCreds (#3398)
 * 637d50bc chore: maybe -> may be (#3392)
 * e70a8863 chore: Added CWFT WorkflowTemplateRef example (#3386)
 * bc4faf5f fix: Fix bug parsing parmeters (#3372)
 * 4934ad22 fix: Running pods are garaged in PodGC onSuccess
 * 0541cfda chore(ui): Remove unused interfaces for artifacts (#3377)
 * 20382cab docs: Fix incorrect example of global parameter (#3375)
 * 1db93c06 perf: Optimize time-based filtering on large number of workflows (#3340)
 * 2ab9495f fix: Don't double-count metric events (#3350)
 * 7bd3e720 fix(ui): Confirmation of workflow actions (#3370)
 * 488790b2 Wellcome is using Argo in our Data Labs division (#3365)
 * 63e71192 chore: Remove unused code (#3367)
 * a64ceb03 build: Enable Stale Bot (#3363)
 * e4b08abb fix(server): Remove `context cancelled` error. Fixes #3073 (#3359)
 * 74ba5162 fix: Fix UI bug in DAGs (#3368)
 * 5e60decf feat(crds)!: Adds CRD generation and enhanced UI resource editor. Closes #859 (#3075)
 * c2347f35 chore: Simplify deps by removing YAML (#3353)
 * 1323f9f4 test: Add e2e tags (#3354)
 * 731a1b4a fix(controller): Allow events to be sent to non-argo namespace. Fixes #3342 (#3345)
 * 916e0db2 Adding InVision to Users (#3352)
 * 6caf10fa fix: Ensure child pods respect maxDuration (#3280)
 * 8f4945f5 docs: Field fix (ParallelSteps -> WorkflowStep) (#3338)
 * 2b4b7340 fix: Remove broken SSO from quick-starts (#3327)
 * 26570fd5 fix(controller)!: Support nested items. Fixes #3288 (#3290)
 * c3d85716 chore: Avoid variable name collision with imported package name (#3335)
 * ca822af0 build: Fix path to go-to-protobuf binary (#3308)
 * 769a964f feat(controller): Label workflows with their source workflow template (#3328)
 * 0785be24 fix(ui): runtime error from null savedOptions props (#3330)
 * 200be0e1 feat: Save pagination limit and selected phases/labels to local storage (#3322)
 * b5ed90fe feat: Allow to change priority when resubmitting workflows (#3293)
 * 60c86c84 fix(ui): Compiler error from workflows toolbar (#3317)
 * 3fe6ecc4 docs: Document access token creation and usage (#3316)
 * ab3c081e docs: Rename Ant Financial to Ant Group (#3304)
 * baad42ea feat(ui): Add ability to select multiple workflows from list and perform actions on them. Fixes #3185 (#3234)
 * b6118939 fix(controller): Fix panic logging. (#3315)
 * 633ea71e build: Pin `goimports` to working version (#3311)
 * 436c1259 ci: Remove CircleCI (#3302)
 * 8e340229 build: Remove generated Swagger files. (#3297)
 * e021d7c5 Clean up unused constants (#3298)
 * 48d86f03 build: Upload E2E diagnostics after failure (#3294)
 * 8b12f433 feat(cli): Add --logs to `argo [submit|resubmit|retry]. Closes #3183 (#3279)
 * 07b450e8 fix: Reapply Update if CronWorkflow resource changed (#3272)
 * 8af01491 docs: ArchiveLabelSelector document (#3284)
 * 38c908a2 docs: Add example for handling large output resutls (#3276)
 * d44d264c Fixes validation of overridden ref template parameters. (#3286)
 * 62e54fb6 fix: Fix delete --complete (#3278)
 * a3c379bb docs: Updated WorkflowTemplateRef  on WFT and CWFT (#3137)
 * 824de95b fix(git): Fixes Git when using auth or fetch. Fixes #2343 (#3248)
 * 018fcc23 Update releasing.md (#3283)
 * acee573b docs: Update CI Badges (#3282)
 * 9eb182c0 build: Allow to change k8s namespace for installation (#3281)
 * 2bcfafb5 fix: Add {{workflow.status}} to workflow-metrics (#3271)
 * e6aab605 fix(jqFilter)!: remove extra quotes around output parameter value (#3251)
 * f4580163 fix(ui): Allow render of templates without entrypoint. Fixes #2891 (#3274)
 * f30c05c7 build: Add warning to ensure 'v' is present on release versions (#3273)
 * d1cb1992 fixed archiveLabelSelector nil (#3270)
 * c7e4c180 fix(ui): Update workflow drawer with new duration format (#3256)
 * f2381a54 fix(controller): More structured logging. Fixes #3260 (#3262)
 * acba084a fix: Avoid unnecessary nil check for annotations of resubmitted workflow (#3268)
 * 55e13705 feat: Append previous workflow name as label to resubmitted workflow (#3261)
 * 2dae7244 feat: Add mode to require Workflows to use workflowTemplateRef (#3149)
 * 56694abe Fixed onexit on workflowtempalteRef (#3263)
 * 54dd72c2 update mysql yaml port (#3258)
 * fb502632 feat: Configure ArchiveLabelSelector for Workflow Archive (#3249)
 * 5467c899 fix(controller): set pod finish timestamp when it is deleted (#3230)
 * 04bc5492 build: Disable Circle CI and Sonar (#3253)
 * 23ca07a7 chore: Covered steps.<STEPNAME>.outputs.parameters in variables document (#3245)
 * 4bd33c6c chore(cli): Add examples of @latest alias for relevant commands. Fixes #3225 (#3242)
 * 17108df1 fix: Ensure subscription is closed in log viewer (#3247)
 * 495dc89b docs: Correct available fields in {{workflow.failures}} (#3238)
 * 4db1c4c8 fix: Support the TTLStrategy for WorkflowTemplateRef (#3239)
 * 47f50693 feat(logging): Made more controller err/warn logging structured (#3240)
 * c25e2880 build: Migrate to Github Actions (#3233)
 * ef159f9a feat: Tick CLI Workflow watch even if there are no new events (#3219)
 * ff1627b7 fix(events): Adds config flag. Reduce number of dupe events emitted. (#3205)
 * eae8f681 feat: Validate CronWorkflows before execution (#3223)
 * 4470a8a2 fix(ui/server): Fix broken label filter functionality on UI due to bug on server. Fix #3226 (#3228)
 * e5e6456b feat(cli): Add --latest flag for argo get command as per #3128 (#3179)
 * 34608594 fix(ui): Correctly update workflow list when workflow are modified/deleted (#3220)
 * a7d8546c feat(controller): Improve throughput of many workflows. Fixes #2908 (#2921)
 * a37d0a72 build: Change "DB=..." to "PROFILE=..." (#3216)
 * 15885d3e feat(sso): Allow reading SSO clientID from a secret. (#3207)
 * 723e9d5f fix: Ensrue image name is present in containers (#3215)
 * 0ee5e112 feat: Only process significant pod changes (#3181)
 * c89a81f3 feat: Add '--schedule' flag to 'argo cron create' (#3199)
 * 591f649a refactor: Refactor assesDAGPhase logic (#3035)
 * 285eda6b chore: Remove unused pod in addArchiveLocation() (#3200)
 * 8e1d56cb feat(controller): Add default name for artifact repository ref. (#3060)
 * f1cdba18 feat(controller): Add `--qps` and `--burst` flags to controller (#3180)
 * b86949f0 fix: Ensure stable desc/hash for metrics (#3196)
 * e26d2f08 docs: Update Getting Started (#3099)
 * 47bfea5d docs: Add Graviti as official Argo user (#3187)
 * 04c77f49 fix(server): Allow field selection for workflow-event endpoint (fixes #3163) (#3165)
 * 0c38e66e chore: Update Community Meeting link and specify Go@v1.13 (#3178)
 * 81846d41 build: Only check Dex in hosts file when SSO is enabled (#3177)
 * a130d488 feat(ui): Add drawer with more details for each workflow in Workflow List (#3151)
 * fa84e203 fix: Do not use alphabetical order if index exists (#3174)
 * 138af597 fix(cli): Sort expanded nodes by index. Closes #3145 (#3146)
 * a9ec4d08 docs: Fix api swagger file path in docs (#3167)
 * c42e4d3a feat(metrics): Add node-level resources duration as Argo variable for metrics. Closes #3110 (#3161)
 * e36fe66e docs: Add instructions on using Minikube as an alternative to K3D (#3162)
 * edfa5b93 feat(metrics): Report controller error counters via metrics. Closes #3034 (#3144)
 * 8831e4ea feat(argo-server): Add support for SSO. See #1813 (#2745)
 * b62184c2 feat(cli): More `argo list` and `argo delete` options (#3117)
 * c6565d7c fix(controller): Maybe bug with nil woc.wfSpec. Fixes #3121 (#3160)
 * 06ca71d7 build: Fix path to staticfiles and goreman binaries (#3159)
 * cad84cab chore: Remove unused nodeType in initializeNodeOrMarkError() (#3153)
 * be425513 chore: Master needs lint (#3152)
 * 70b56f25 enhancement(ui): Add workflow labels column to workflow list. Fixes #2782 (#3143)
 * 3318c115 chore: Move default metrics server port/path to consts (#3135)
 * a0062adf feat(ui): Add Alibaba Cloud OSS related models in UI (#3140)
 * 1469991c fix: Update container delete grace period to match Kubernetes default (#3064)
 * df725bbd fix(ui): Input artifacts labelled in UI. Fixes #3098 (#3131)
 * c0d59cc2 feat: Persist DAG rendering options in local storage (#3126)
 * 8715050b fix(ui): Fix label error (#3130)
 * 1814ea2e fix(item): Support ItemValue.Type == List. Fixes #2660 (#3129)
 * 12b72546 fix: Panic on invalid WorkflowTemplateRef (#3127)
 * 09092147 fix(ui): Display error message instead of DAG when DAG cannot be rendered. Fixes #3091 (#3125)
 * 2d9a74de docs: Document cost optimizations. Fixes #1139 (#2972)
 * 69c9e5f0 fix: Remove unnecessary panic (#3123)
 * 2f3aca89 add AppDirect to the list of users (#3124)
 * 257355e4 feat: Add 'submit --from' to CronWorkflow and WorkflowTemplate in UI. Closes #3112 (#3116)
 * 6e5dd2e1 Add Alibaba OSS to the list of supported artifacts (#3108)
 * 1967b45b support sso (#3079)
 * 9229165f feat(ui): Add cost optimisation nudges. (#3089)
 * e88124db fix(controller): Do not panic of woc.orig in not hydrated. Fixes #3118 (#3119)
 * 132b947a fix: Differentiate between Fulfilled and Completed (#3083)
 * a93968ff docs: Document how to backfill a cron workflow (#3094)
 * 4de99746 feat: Added Label selector and Field selector in Argo list  (#3088)
 * 6229353b chore: goimports (#3107)
 * 8491e00f docs: Add link to USERS.md in PR template (#3086)
 * bb2ce9f7 fix: Graceful error handling of malformatted log lines in watch (#3071)
 * 4fd27c31 build(swagger): Fix Swagger build problems (#3084)
 * e4e0dfb6 test: fix TestContinueOnFailDag (#3101)
 * fa69c1bb feat: Add CronWorkflowConditions to report errors (#3055)
 * 50ad3cec adds millisecond-level timestamps to argoexec (#2950)
 * 6464bd19 fix(controller): Implement offloading for workflow updates that are re-applied. Fixes #2856 (#2941)
 * 6c369e61 chore: Rename files that include 'top-level' terminology (#3076)
 * bd40b80b docs: Document work avoidance. (#3066)
 * 6df0b2d3 feat: Support Top level workflow template reference  (#2912)
 * 0709ad28 feat: Enhanced filters for argo {watch,get,submit} (#2450)
 * 784c1385 build: Use goreman for starting locally. (#3074)
 * 5b5bae9a docs: Add Isbank to users.md (#3068)
 * 2b038ed2 feat: Enhanced depends logic (#2673)
 * 4c3387b2 fix: Linters should error if nothing was validated (#3011)
 * 51dd05b5 fix(artifacts): Explicit archive strategy. Fixes #2140 (#3052)
 * ada2209e Revert "fix(artifacts): Allow tar check to be ignored. Fixes #2140 (#3024)" (#3047)
 * b7ff9f09 chore: Add ability to configure maximum DB connection lifetime (#3032)
 * 38a995b7 fix(executor): Properly handle empty resource results, like for a missing get (#3037)
 * a1ac8bcf fix(artifacts): Allow tar check to be ignored. Fixes #2140 (#3024)
 * f12d79ca fix(controller)!: Correctly format workflow.creationTimepstamp as RFC3339. Fixes #2974 (#3023)
 * d10e949a fix: Consider metric nodes that were created and completed in the same operation (#3033)
 * 202d4ab3 fix(executor): Optional input artifacts. Fixes #2990 (#3019)
 * f17e946c fix(executor): Save script results before artifacts in case of error. Fixes #1472 (#3025)
 * 3d216ae6 fix: Consider missing optional input/output artifacts with same name (#3029)
 * 3717dd63 fix: Improve robustness of releases. Fixes #3004 (#3009)
 * 9f86a4e9 feat(ui): Enable CSP, HSTS, X-Frame-Options. Fixes #2760, #1376, #2761 (#2971)
 * cb71d585 refactor(metrics)!: Refactor Metric interface (#2979)
 * c0ee1eb2 docs: Add Ravelin as a user of Argo (#3020)
 * 052e6c51 Fix isTarball to handle the small gzipped file (#3014)
 * cdcba3c4 fix(ui): Displays command args correctl pre-formatted. (#3018)
 * b5160988 build: Mockery v1.1.1 (#3015)
 * a04d8f28 docs: Add StatefulSet and Service doc (#3008)
 * 8412526c docs: Fix Deprecated formatting (#3010)
 * cc0fe433 fix(events): Correct event API Version. Fixes #2994 (#2999)
 * d5d6f750 feat(controller)!: Updates the resource duration calculation. Fixes #2934 (#2937)
 * fa3801a5 feat(ui): Render 2000+ nodes DAG acceptably. (#2959)
 * f952df51 fix(executor/pns): remove sleep before sigkill (#2995)
 * 2a9ee21f feat(ui): Add Suspend and Resume to CronWorkflows in UI (#2982)
 * eefe120f test: Upgrade to argosay:v2 (#3001)
 * 47472f73 chore: Update Mockery (#3000)
 * 46b11e1e docs: Use keyFormat instead of keyPrefix in docs (#2997)
 * 60d5fdc7 fix: Begin counting maxDuration from first child start (#2976)
 * 76aca493 build: Fix Docker build. Fixes #2983 (#2984)
 * d8cb66e7 feat: Add Argo variable {{retries}} to track retry attempt (#2911)
 * 14b7a459 docs: Fix typo with WorkflowTemplates link (#2977)
 * 3c442232 fix: Remove duplicate node event. Fixes #2961 (#2964)
 * d8ab13f2 fix: Consider Shutdown when assesing DAG Phase for incomplete Retry node (#2966)
 * 8a511e10 fix: Nodes with pods deleted out-of-band should be Errored, not Failed (#2855)
 * ca4e08f7 build: Build dev images from cache (#2968)
 * 5f01c4a5 Upgraded to Node 14.0.0 (#2816)
 * 849d876c Fixes error with unknown flag: --show-all (#2960)
 * 93bf6609 fix: Don't update backoff message to save operations (#2951)
 * 3413a5df fix(cli): Remove info logging from watches. Fixes #2955 (#2958)
 * fe9f9019 fix: Display Workflow finish time in UI (#2896)
 * f281199a docs: Update README with new features (#2807)
 * c8bd0bb8 fix(ui): Change default pagination to all and sort workflows (#2943)
 * e3ed686e fix(cli): Re-establish watch on EOF (#2944)
 * 67355372 fix(swagger)!: Fixes invalid K8S definitions in `swagger.json`. Fixes #2888 (#2907)
 * 023f2338 fix(argo-server)!: Implement missing instanceID code. Fixes #2780 (#2786)
 * 7b0739e0 Fix typo (#2939)
 * 20d69c75 Detect ctrl key when a link is clicked (#2935)
 * f32cec31 fix default null value for timestamp column - MySQL 5.7 (#2933)
 * 9773cfeb docs: Add docs/scaling.md (#2918)
 * 99858ea5 feat(controller): Remove the excessive logging of node data (#2925)
 * 03ad694c feat(cli): Refactor `argo list --chunk-size` and add `argo archive list --chunk-size`. Fixes #2820 (#2854)
 * 1c45d5ea test: Use argoproj/argosay:v1 (#2917)
 * f311a5a7 build: Fix Darwin build (#2920)
 * a06cb5e0 fix: remove doubled entry in server cluster role deployment (#2904)
 * c71116dd feat: Windows Container Support. Fixes #1507 and #1383 (#2747)
 * 3afa7b2f fix(ui): Use LogsViewer for container logs (#2825)
 * 9ecd5226 docs: Document node field selector. Closes #2860 (#2882)
 * 7d8818ca fix(controller): Workflow stop and resume by node didn't properly support offloaded nodes. Fixes #2543 (#2548)
 * e013f29d ci: Remove context to stop unauthozied errors on test jobs (#2910)
 * db52e7ba fix(controller): Add mutex to nameEntryIDMap in cron controller. Fix #2638 (#2851)
 * 9a33aa2d docs(users): Adding Habx to the users list (#2781)
 * 9e4ac9b3 feat(cli): Tolerate deleted workflow when running `argo delete`. Fixes #2821 (#2877)
 * a0035dd5 fix: ConfigMap syntax (#2889)
 * c05c3859 ci: Build less and therefore faster (#2839)
 * 56143eb1 feat(ui): Add pagination to workflow list. Fixes #1080 and #976 (#2863)
 * e0ad7de9 test: Fixes various tests (#2874)
 * e378ca47 fix: Cannot create WorkflowTemplate with un-supplied inputs (#2869)
 * c3e30c50 fix(swagger): Generate correct Swagger for inline objects. Fixes #2835 (#2837)
 * c0143d34 feat: Add metric retention policy (#2836)
 * f03cda61 Update getting-started.md (#2872)
 * d66224e1 fix: Don't error when deleting already-deleted WFs (#2866)
 * e84acb50 chore: Display wf.Status.Conditions in CLI (#2858)
 * 3c7f3a07 docs: Fix typo ".yam" -> ".yaml" (#2862)
 * d7f8e0c4 fix(CLI): Re-establish workflow watch on disconnect. Fixes #2796 (#2830)
 * 31358d6e feat(CLI): Add -v and --verbose to Argo CLI (#2814)
 * 1d30f708 ci: Don't configure Sonar on CI for release branches (#2811)
 * d9c54075 docs: Fix exit code example and docs (#2853)
 * 90743353 feat: Expose workflow.serviceAccountName as global variable (#2838)
 * f07f7bf6 note that tar.gz'ing output artifacts is optional (#2797)
 * 3fd3fc6c docs: Document how to label creator (#2827)
 * b956ec65 fix: Add Step node outputs to global scope (#2826)
 * bac339af chore: Configure webpack dev server to proxy using HTTPS (#2812)
 * cc136f9c test: Skip TestStopBehavior. See #2833 (#2834)
 * 52ff43b5 fix: Artifact panic on unknown artifact. Fixes #2824 (#2829)
 * 554fd06c fix: Enforce metric naming validation (#2819)
 * dd223669 docs: Add Microba as official Argo user (#2822)
 * 8151f0c4 docs: Update tls.md (#2813)
 * 0dbd78ff feat: Add TLS support. Closes #2764 (#2766)
 * 510e11b6 fix: Allow empty strings in valueFrom.default (#2805)
 * d7f41ac8 fix: Print correct version in logs. (#2806)
 * e9c21120 chore: Add GCS native example for output artifact (#2789)
 * e0f2697e fix(controller): Include global params when using withParam (#2757)
 * 3441b11a docs: Fix typo in CronWorkflow doc (#2804)
 * a2d2b848 docs: Add example of recursive for loop (#2801)
 * 29d39e29 docs: Update the contributing docs  (#2791)
 * 1ea286eb fix: ClusterWorkflowTemplate RBAC for  argo server (#2753)
 * 1f14f2a5 feat(archive): Implement data retention. Closes #2273 (#2312)
 * d0cc7764 feat: Display argo-server version in `argo version` and in UI. (#2740)
 * 8de57281 feat(controller): adds Kubernetes node name to workflow node detail in web UI and CLI output. Implements #2540 (#2732)
 * 52fa5fde MySQL config fix (#2681)
 * 43d9eebb fix: Rename Submittable API endpoint to `submit` (#2778)
 * 69333a87 Fix template scope tests (#2779)
 * bb1abf7f chore: Add CODEOWNERS file (#2776)
 * 905e0b99 fix: Naming error in Makefile (#2774)
 * 7cb2fd17 fix: allow non path output params (#2680)
 * af9f61ea ci: Recurl (#2769)
 * ef08e642 build: Retry curl 3x (#2768)
 * dedec906 test: Get tests running on release branches (#2767)
 * 1c8318eb fix: Add compatiblity mode to templateReference (#2765)
 * 7975952b fix: Consider expanded tasks in getTaskFromNode (#2756)
 * bc421380 fix: Fix template resolution in UI (#2754)
 * 391c0f78 Make phase and templateRef available for unsuspend and retry selectors (#2723)
 * a6fa3f71 fix: Improve cookie security. Fixes #2759 (#2763)
 * 57f0183c Fix typo on the documentation. It causes error unmarshaling JSON: while (#2730)
 * c6ef1ff1 feat(manifests): add name on workflow-controller-metrics service port (#2744)
 * af5cd1ae docs: Update OWNERS (#2750)
 * 06c4bd60 fix: Make ClusterWorkflowTemplate optional for namespaced Installation (#2670)
 * 25c62463 docs: Update README (#2752)
 * 908e1685 docs: Update README.md (#2751)
 * 4ea43e2d fix: Children of onExit nodes are also onExit nodes (#2722)
 * 3f1b6667 feat: Add Kustomize as supported install option. Closes #2715 (#2724)
 * 691459ed fix: Error pending nodes w/o Pods unless resubmitPendingPods is set (#2721)
 * 874d8776 test: Longer timeout for deletion (#2737)
 * 3c8149fa Fix typo (#2741)
 * 98f60e79 feat: Added Workflow SubmitFromResource API (#2544)
 * 6253997a fix: Reset all conditions when resubmitting (#2702)
 * e7c67de3 fix: Maybe fix watch. Fixes #2678 (#2719)
 * cef6dfb6 fix: Print correct version string. (#2713)
 * e9589d28 feat: Increase pod workers and workflow workers both to 32 by default. (#2705)
 * 3a1990e0 test: Fix Goroutine leak that was making controller unit tests slow. (#2701)
 * 9894c29f ci: Fix Sonar analysis on master. (#2709)
 * 54f5be36 style: Camelcase "clusterScope" (#2720)
 * db6d1416 fix: Flakey TestNestedClusterWorkflowTemplate testcase failure (#2613)
 * b4fd4475 feat(ui): Add a YAML panel to view the workflow manifest. (#2700)
 * 65d413e5 build(ui): Fix compression of UI package. (#2704)
 * 4129528d fix: Don't use docker cache when building release images (#2707)
 * 8d0956c9 test: Increase runCli timeout to 1m (#2703)
 * 9d93e971 Update getting-started.md (#2697)
 * ee644a35 docs: Fix CONTRIBUTING.md and running-locally.md. Fixes #2682 (#2699)
 * 2737c0ab feat: Allow to pass optional flags to resource template (#1779)
 * c1a2fc7c Update running-locally.md - fixing incorrect protoc install (#2689)
 * a1226c46 fix: Enhanced WorkflowTemplate and ClusterWorkflowTemplate validation to support Global Variables   (#2644)
 * c21cc2f3 fix a typo (#2669)
 * 9430a513 fix: Namespace-related validation in UI (#2686)
 * f3eeca6e feat: Add exit code as output variable (#2111)
 * 9f95e23a fix: Report metric emission errors via Conditions (#2676)
 * c67f5ff5 fix: Leaf task with continueOn should not fail DAG (#2668)
 * 3c20d4c0 ci: Migrate to use Sonar instead of CodeCov for analysis (#2666)
 * 9c6351fa feat: Allow step restart on workflow retry. Closes #2334 (#2431)
 * cf277eb5 docs: Updates docs for CII. See #2641 (#2643)
 * e2d0aa23 fix: Consider offloaded and compressed node in retry and resume (#2645)
 * a25c6a20 build: Fix codegen for releases (#2662)
 * 4a3ca930 fix: Correctly emit events. Fixes #2626 (#2629)
 * 4a7d4bdb test: Fix flakey DeleteCompleted test (#2659)
 * 41f91e18 fix: Add DAG as default in UI filter and reorder (#2661)
 * f138ada6 fix: DAG should not fail if its tasks have continueOn (#2656)
 * e5cbdf6a ci: Only run CI jobs if needed (#2655)
 * 4c452d5f fix: Don't attempt to resolve artifacts if task is going to be skipped (#2657)
 * 2caf570a chore: Add newline to fields.md (#2654)
 * 2cb596da Storage region should be specified (#2538)
 * 271e4551 chore: Fix-up Yarn deps (#2649)
 * 4c1b0777 fix: Sort log entries. (#2647)
 * 268fc461  docs: Added doc generator code (#2632)
 * d58b7fc3 fix: Add input paremeters to metric scope (#2646)
 * cc3af0b8 fix: Validating Item Param in Steps Template (#2608)
 * 6c685c5b fix: allow onExit to run if wf exceeds activeDeadlineSeconds. Fixes #2603 (#2605)
 * ffc43ce9 feat: Added Client validation on Workflow/WFT/CronWF/CWFT (#2612)
 * 24655cd9 feat(UI): Move Workflow parameters to top of submit (#2640)
 * 0a3b159a Use error equals (#2636)
 * 8c29e05c ci: Fix codegen job (#2648)
 * a78ecb7f docs(users): Add CoreWeave and ConciergeRender (#2641)
 * 14be4670 fix: Fix logs part 2 (#2639)
 * 4da6f4f3 feat: Add 'outputs.result' to Container templates (#2584)
 * 51bc876d test: Fixes TestCreateWorkflowDryRun. Closes #2618 (#2628)
 * 212c6d75 fix: Support minimal mysql version 5.7.8 (#2633)
 * 8facacee refactor: Refactor Template context interfaces (#2573)
 * 812813a2 fix: fix test cases (#2631)
 * ed028b25 fix: Fix logging problems. See #2589 (#2595)
 * d4e81238 test: Fix teething problems (#2630)
 * 4aad6d55 chore: Add comments to issues (#2627)
 * 54f7a013 test: Enhancements and repairs to e2e test framework (#2609)
 * d95926fe fix: Fix WorkflowTemplate icons to be more cohesive (#2607)
 * 0130e1fd docs: Add fields and core concepts doc (#2610)
 * 5a1ac203 fix: Fixes panic in toWorkflow method (#2604)
 * 51910292 chore: Lint UI on CI, test diagnostics, skip bad test (#2587)
 * 232bb115 fix(error handling): use Errorf instead of New when throwing errors with formatted text (#2598)
 * eeb2f97b fix(controller): dag continue on failed. Fixes #2596 (#2597)
 * 99c35129 docs: Fix inaccurate field name in docs (#2591)
 * 21c73779 fix: Fixes lint errors (#2594)
 * 38aca5fa chore: Added ClusterWorkflowTemplate RBAC on quick-start manifests (#2576)
 * 59f746e1 feat: UI enhancement for Cluster Workflow Template (#2525)
 * 0801a428 fix(cli): Show lint errors of all files (#2552)
 * c3535ba5 docs: Fix wrong Configuring Your Artifact Repository document. (#2586)
 * 79217bc8 feat(archive): allow specifying a compression level (#2575)
 * 88d261d7 fix: Use outputs of last child instead of retry node itslef (#2565)
 * 5c08292e style: Correct the confused logic (#2577)
 * 3d146144 fix: Fix bug in deleting pods. Fixes #2571 (#2572)
 * cb739a68 feat: Cluster scoped workflow template (#2451)
 * c63e3d40 feat: Show workflow duration in the index UI page (#2568)
 * 1520452a chore: Error -> Warn when Parent CronWf no longer exists (#2566)
 * ffbb3b89 fix: Fixes empty/missing CM. Fixes #2285 (#2562)
 * d0fba6f4 chore: fix typos in the workflow template docs (#2563)
 * 49801e32 chore(docker): upgrade base image for executor image (#2561)
 * c4efb8f8 Add Riskified to the user list (#2558)
 * 8b92d33e feat: Create K8S events on node completion. Closes #2274 (#2521)
 * 2902e144 feat: Add node type and phase filter to UI (#2555)
 * fb74ba1c fix: Separate global scope processing from local scope building (#2528)
 * 618b6dee fix: Fixes --kubeconfig flag. Fixes #2492 (#2553)
 * 79dc969f test: Increase timeout for flaky test (#2543)
 * 15a3c990 feat: Report SpecWarnings in status.conditions (#2541)
 * f142f30a docs: Add example of template-level volume declaration. (#2542)
 * 93b6be61 fix(archive): Fix bug that prevents listing archive workflows. Fixes … (#2523)
 * b4c9c54f fix: Omit config key in configure artifact document. (#2539)
 * 864bf1e5 fix: Show template on its own field in CLI (#2535)
 * 555aaf06 test: fix master (#2534)
 * 94862b98 chore: Remove deprecated example (#2533)
 * 5e1e7829 fix: Validate CronWorkflow before creation (#2532)
 * c9241339 fix: Fix wrong assertions (#2531)
 * 67fe04bb Revert "fix: fix template scope tests (#2498)" (#2526)
 * ddfa1ad0 docs: couple of examples for REST API usage of argo-server (#2519)
 * 30542be7 chore(docs): Update docs for useSDKCreds (#2518)
 * e2cc6988 feat: More control over resuming suspended nodes Fixes #1893 (#1904)
 * b2771249 chore: minor fix and refactory (#2517)
 * b1ad163a fix: fix template scope tests (#2498)
 * 661d1b67 Increase client gRPC max size to match server (#2514)
 * d8aa477f fix: Fix potential panic (#2516)
 * 1afb692e fix: Allow runtime resolution for workflow parameter names (#2501)
 * 243ea338 fix(controller): Ensure we copy any executor securityContext when creating wait containers; fixes #2512 (#2510)
 * 6e8c7bad feat: Extend workflowDefaults to full Workflow and clean up docs and code (#2508)
 * 06cfc129 feat: Native Google Cloud Storage support for artifact. Closes #1911 (#2484)
 * 999b1e1d  fix: Read ConfigMap before starting servers  (#2507)
 * 3d6e9b61 docs: Add separate ConfigMap doc for 2.7+ (#2505)
 * e5bd6a7e fix(controller): Updates GetTaskAncestry to skip visited nod. Fixes #1907 (#1908)
 * 183a29e4 docs: add official user (#2499)
 * e636000b feat: Updated arm64 support patch (#2491)
 * 559cb005 feat(ui): Report resources duration in UI. Closes #2460 (#2489)
 * 09291d9d feat: Add default field in parameters.valueFrom (#2500)
 * 33cd4f2b feat(config): Make configuration mangement easier. Closes #2463 (#2464)
 * f3df9660 test: Fix test (#2490)
 * bfaf1c21 chore: Move quickstart Prometheus port to 9090 (#2487)
 * 487ed425 feat: Logging the Pod Spec in controller log (#2476)
 * 96c80e3e fix(cli): Rearrange the order of chunk size argument in list command. Closes #2420 (#2485)
 * 47bd70a0 chore: Fix Swagger for PDB to support Java client (#2483)
 * 53a10564 feat(usage): Report resource duration. Closes #1066 (#2219)
 * 063d9bc6 Revert "feat: Add support for arm64 platform (#2364)" (#2482)
 * 735d25e9 fix: Build image with SHA tag when a git tag is not available (#2479)
 * c55bb3b2 ci: Run lint on CI and fix GolangCI (#2470)
 * e1c9f7af fix ParallelSteps child type so replacements happen correctly; fixes argoproj-labs/argo-client-gen#5 (#2478)
 * 55c315db feat: Add support for IRSA and aws default provider chain. (#2468)
 * c724c7c1 feat: Add support for arm64 platform (#2364)
 * 315dc164 feat: search archived wf by startat. Closes #2436 (#2473)
 * 23d230bd feat(ui): add Env to Node Container Info pane. Closes #2471 (#2472)
 * 10a0789b fix: ParallelSteps swagger.json (#2459)
 * a59428e7 fix: Duration must be a string. Make it a string. (#2467)
 * 47bc6f3b feat: Add `argo stop` command (#2352)
 * 14478bc0 feat(ui): Add the ability to have links to logging facility in UI. Closes #2438 (#2443)
 * 2864c745 chore: make codegen + make start (#2465)
 * a85f62c5 feat: Custom, step-level, and usage metrics (#2254)
 * 64ac0298 fix: Deprecate template.{template,templateRef,arguments} (#2447)
 * 6cb79e4e fix: Postgres persistence SSL Mode (#1866) (#1867)
 * 2205c0e1 fix(controller): Updates to add condition to workflow status. Fixes #2421 (#2453)
 * 9d96ab2f fix: make dir if needed (#2455)
 * 5346609e test: Maybe fix TestPendingRetryWorkflowWithRetryStrategy. Fixes #2446 (#2456)
 * 3448ccf9 fix: Delete PVCs unless WF Failed/Errored (#2449)
 * 782bc8e7 fix: Don't error when optional artifacts are not found (#2445)
 * fc18f3cf chore: Master needs codegen (#2448)
 * 32fc2f78 feat: Support workflow templates submission. Closes #2007 (#2222)
 * 050a143d fix(archive): Fix edge-cast error for archiving. Fixes #2427 (#2434)
 * 9455c1b8 doc: update CHANGELOG.md (#2425)
 * 1baa7ee4 feat(ui): cache namespace selection. Closes #2439 (#2441)
 * 91d29881 feat: Retry pending nodes (#2385)
 * 7094433e test: Skip flakey tests in operator_template_scope_test.go. See #2432 (#2433)
 * 30332b14 fix: Allow numbers in steps.args.params.value (#2414)
 * e9a06dde feat: instanceID support for argo server. Closes #2004 (#2365)
 * 3f8be0cd fix "Unable to retry workflow" on argo-server (#2409)
 * dd3029ab docs: Example showing how to use default settings for workflow spec. Related to ##2388 (#2411)
 * 13508828 fix: Check child node status before backoff in retry (#2407)
 * b59419c9 fix: Build with the correct version if you check out a specific version (#2423)
 * 6d834d54 chore: document BASE_HREF (#2418)
 * 184c3653 fix: Remove lazy workflow template (#2417)
 * 918d0d17 docs: Added Survey Results (#2416)
 * 20d6e27b Update CONTRIBUTING.md (#2410)
 * f2ca045e feat: Allow WF metadata spec on Cron WF (#2400)
 * 068a4336 fix: Correctly report version. Fixes #2374 (#2402)
 * e19a398c Update pull_request_template.md (#2401)
 * 7c99c109 chore: Fix typo (#2405)
 * 175b164c Change font family for class yaml (#2394)
 * d1194755 fix: Don't display Retry Nodes in UI if len(children) == 1 (#2390)
 * b8623ec7 docs: Create USERS.md (#2389)
 * 1d21d3f5 fix(doc strings): Fix bug related documentation/clean up of default configurations #2331 (#2388)
 * 77e11fc4 chore: add noindex meta tag to solve #2381; add kustomize to build docs (#2383)
 * 42200fad fix(controller): Mount volumes defined in script templates. Closes #1722 (#2377)
 * 96af36d8 fix: duration must be a string (#2380)
 * 7bf08192 fix: Say no logs were outputted when pod is done (#2373)
 * 847c3507 fix(ui): Removed tailLines from EventSource (#2330)
 * 3890a124 feat: Allow for setting default configurations for workflows, Fixes #1923, #2044 (#2331)
 * 81ab5385 Update readme (#2379)
 * 91810273 feat: Log version (structured) on component start-up (#2375)
 * d0572a74 docs: Make Getting Started agnostic to version (#2371)
 * d3a3f6b1 docs: Add Prudential to the users list (#2353)
 * 4714c880 chore: Master needs codegen (#2369)
 * 5b6b8257 fix(docker): fix streaming of combined stdout/stderr (#2368)
 * 97438313 fix: Restart server ConfigMap watch when closed (#2360)
 * 64d0cec0 chore: Master needs make lint (#2361)
 * 12386fc6 fix: rerun codegen after merging OSS artifact support (#2357)
 * 40586ed5 fix: Always validate templates (#2342)
 * 897db894 feat: Add support for Alibaba Cloud OSS artifact (#1919)
 * 7e2dba03 feat(ui): Circles for nodes (#2349)
 * e85f6169 chore: update getting started guide to use 2.6.0 (#2350)
 * 7ae4ec78 docker: remove NopCloser from the executor. (#2345)
 * 5895b364 feat: Expose workflow.paramteres with JSON string of all params (#2341)
 * a9850b43 Fix the default (#2346)
 * c3763d34 fix: Simplify completion detection logic in DAGs (#2344)
 * d8a9ea09 fix(auth): Fixed returning  expired  Auth token for GKE (#2327)
 * 6fef0454 fix: Add timezone support to startingDeadlineSeconds (#2335)
 * c28731b9 chore: Add go mod tidy to codegen (#2332)
 * a66c8802 feat: Allow Worfklows to be submitted as files from UI (#2340)
 * a9c1d547 docs: Update Argo Rollouts description (#2336)
 * 8672b97f fix(Dockerfile): Using `--no-install-recommends` (Optimization) (#2329)
 * c3fe1ae1 fix(ui): fixed worflow UI refresh. Fixes ##2337 (#2338)
 * d7690e32 feat(ui): Adds ability zoom and hide successful steps. POC (#2319)
 * e9e13d4c feat: Allow retry strategy on non-leaf nodes, eg for step groups. Fixes #1891 (#1892)
 * 62e6db82 feat: Ability to include or exclude fields in the response (#2326)
 * 52ba89ad fix(swagger): Fix the broken swagger. (#2317)
 * efb8a1ac docs: Update CODE_OF_CONDUCT.md (#2323)
 * 1c77e864 fix(swagger): Fix the broken swagger. (#2317)
 * aa052346 feat: Support workflow level poddisruptionbudge for workflow pods #1728 (#2286)
 * 8da88d7e chore: update getting-started guide for 2.5.2 and apply other tweaks (#2311)
 * 2f97c261 build: Improve reliability of release. (#2309)
 * 5dcb84bb chore(cli): Clean-up code. Closes #2117 (#2303)
 * e49dd8c4 chore(cli): Migrate `argo logs` to use API client. See #2116 (#2177)
 * 5c3d9cf9 chore(cli): Migrate `argo wait` to use API client. See #2116 (#2282)
 * baf03f67 fix(ui): Provide a link to archived logs. Fixes #2300 (#2301)
 * b5947165 feat: Create API clients (#2218)
 * 214c4515 fix(controller): Get correct Step or DAG name. Fixes #2244 (#2304)
 * c4d26466 fix: Remove active wf from Cron when deleted (#2299)
 * 0eff938d fix: Skip empty withParam steps (#2284)
 * 636ea443 chore(cli): Migrate `argo terminate` to use API client. See #2116 (#2280)
 * d0a9b528 chore(cli): Migrate `argo template` to use API client. Closes #2115 (#2296)
 * f69a6c5f chore(cli): Migrate `argo cron` to use API client. Closes #2114 (#2295)
 * 80b9b590 chore(cli): Migrate `argo retry` to use API client. See #2116 (#2277)
 * cdbc6194 fix(sequence): broken in 2.5. Fixes #2248 (#2263)
 * 0d3955a7 refactor(cli): 2x simplify migration to API client. See #2116 (#2290)
 * df8493a1 fix: Start Argo server with out Configmap #2285 (#2293)
 * 51cdf95b doc: More detail for namespaced installation (#2292)
 * a7302697 build(swagger): Fix argo-server swagger so version does not change. (#2291)
 * 47b4fc28 fix(cli): Reinstate `argo wait`. Fixes #2281 (#2283)
 * 1793887b chore(cli): Migrate `argo suspend` and `argo resume` to use API client. See #2116 (#2275)
 * 1f3d2f5a chore(cli): Update `argo resubmit` to support client API. See #2116 (#2276)
 * c33f6cda fix(archive): Fix bug in migrating cluster name. Fixes #2272 (#2279)
 * fb0acbbf fix: Fixes double logging in UI. Fixes #2270 (#2271)
 * acf37c2d fix: Correctly report version. Fixes #2264 (#2268)
 * b30f1af6 fix: Removes Template.Arguments as this is never used. Fixes #2046 (#2267)
 * 79b09ed4 fix: Removed duplicate Watch Command (#2262)
 * b5c47266 feat(ui): Add filters for archived workflows (#2257)
 * d30aa335 fix(archive): Return correct next page info. Fixes #2255 (#2256)
 * 8c97689e fix: Ignore bookmark events for restart. Fixes #2249 (#2253)
 * 63858eaa fix(offloading): Change offloaded nodes datatype to JSON to support 1GB. Fixes #2246 (#2250)
 * 4d88374b Add Cartrack into officially using Argo (#2251)
 * d309d5c1 feat(archive): Add support to filter list by labels. Closes #2171 (#2205)
 * 79f13373 feat: Add a new symbol for suspended nodes. Closes #1896 (#2240)
 * 82b48821 Fix presumed typo (#2243)
 * af94352f feat: Reduce API calls when changing filters. Closes #2231 (#2232)
 * a58cbc7d BasisAI uses Argo (#2241)
 * 68e3c9fd feat: Add Pod Name to UI (#2227)
 * eef85072 fix(offload): Fix bug which deleted completed workflows. Fixes #2233 (#2234)
 * 4e4565cd feat: Label workflow-created pvc with workflow name (#1890)
 * 8bd5ecbc fix: display error message when deleting archived workflow fails. (#2235)
 * ae381ae5 feat: This add support to enable debug logging for all CLI commands (#2212)
 * 1b1927fc feat(swagger): Adds a make api/argo-server/swagger.json (#2216)
 * 5d7b4c8c Update README.md (#2226)
 * 170abfa5 chore: Run `go mod tidy` (#2225)
 * 2981e6ff fix: Enforce UnknownField requirement in WorkflowStep (#2210)
 * affc235c feat: Add failed node info to exit handler (#2166)
 * af1f6d60 fix: UI Responsive design on filter box (#2221)
 * a445049c fix: Fixed race condition in kill container method. Fixes #1884 (#2208)
 * 2672857f feat: upgrade to Go 1.13. Closes #1375 (#2097)
 * 7466efa9 feat: ArtifactRepositoryRef ConfigMap is now taken from the workflow namespace (#1821)
 * 50f331d0 build: Fix ARGO_TOKEN (#2215)
 * 7f090351 test: Correctly report diagnostics (#2214)
 * f2bd74bc fix: Remove quotes from UI (#2213)
 * 62f46680 fix(offloading): Correctly deleted offloaded data. Fixes #2206 (#2207)
 * e30b77fc feat(ui): Add label filter to workflow list page. Fixes #802 (#2196)
 * 930ced39 fix(ui): fixed workflow filtering and ordering. Fixes #2201 (#2202)
 * 88112312 fix: Correct login instructions. (#2198)
 * d6f5953d Update ReadMe for EBSCO (#2195)
 * b024c46c feat: Add ability to submit CronWorkflow from CLI (#2003)
 * c97527ce test: Invoke tests using s.T() (#2194)
 * 72a54fe1 chore: Move info.proto et al to correct package (#2193)
 * f6600fa4 fix: Namespace and phase selection in UI (#2191)
 * c4a24dca fix(k8sapi-executor): Fix KillContainer impl (#2160)
 * d22a5fe6 Update cli_with_server_test.go (#2189)
 * ff18180f test: Remove podGC (#2187)
 * 78245305 chore: Improved error handling and refactor (#2184)
 * b9c828ad fix(archive): Only delete offloaded data we do not need. Fixes #2170 and #2156 (#2172)
 * 73cb5418 feat: Allow CronWorkflows to have instanceId (#2081)
 * 9efea660 Sort list and add Greenhouse (#2182)
 * cae399ba fix: Fixed the Exec Provider token bug (#2181)
 * fc476b2a fix(ui): Retry workflow event stream on connection loss. Fixes #2179 (#2180)
 * 65058a27 fix: Correctly create code from changed protos. (#2178)
 * 585d1eef chore: Update lint command to use apiclient. See #2116 (#2131)
 * 299d467c build: Update release process and docs (#2128)
 * fcfe1d43 feat: Implemented open default browser in local mode (#2122)
 * f6cee552 fix: Specify download .tgz extension (#2164)
 * 8a1e611a feat: Update archived workdflow column to be JSON. Closes #2133 (#2152)
 * f591c471 fix!: Change `argo token` to `argo auth token`. Closes #2149 (#2150)
 * 519c9434 chore: Add Mock gen to make codegen (#2148)
 * 409a5154 fix: Add certs to argocli image. Fixes #2129 (#2143)
 * b094802a fix: Allow download of artifacs in server auth-mode. Fixes #2129 (#2147)
 * 520fa540 fix: Correct SQL syntax. (#2141)
 * 059cb9b1 fix: logs UI should fall back to archive (#2139)
 * 4cda9a05 fix: route all unknown web content requests to index.html (#2134)
 * 14d8b5d3 fix: archiveLogs needs to copy stderr (#2136)
 * 91319ee4 fixed ui navigation issues with basehref (#2130)
 * 7881b980 docs: Add CronWorkflow usage docs (#2124)
 * badfd183 feat: Add support to delete by using labels. Depended on by #2116 (#2123)
 * 706d0f23 test: Try and make e2e more robust. Fixes #2125 (#2127)
 * a75ac1b4 fix: mark CLI common.go vars and funcs as DEPRECATED (#2119)
 * be21a0f1 feat(server): Restart server when config changes. Fixes #2090 (#2092)
 * b5cd72b0 test: Parallelize Cron tests (#2118)
 * b2bd25bc fix: Disable webpack dot rule (#2112)
 * 865b4f3a addcompany (#2109)
 * 213e3a9d fix: Fix Resource Deletion Bug (#2084)
 * ab1de233 refactor(cli): Introduce v1.Interface for CLI. Closes #2107 (#2048)
 * 7a19f85c feat: Implemented Basic Auth scheme (#2093)
 * 7611b9f6 fix(ui): Add support for bash href. Fixes ##2100 (#2105)
 * 516d05f8  fix: Namespace redirects no longer error and are snappier (#2106)
 * 16aed5c8 fix: Skip running --token testing if it is not on CI (#2104)
 * aece7e6e Parse container ID in correct way on CRI-O. Fixes #2095 (#2096)
 * b6a2be89 feat: support arg --token when talking to argo-server (#2027) (#2089)
 * 01d8cae1 build: adds `make env` to make testing easier (#2087)
 * 492842aa docs(README): Add Capital One to user list (#2094)
 * d56a0e12 fix(controller): Fix template resolution for step groups. Fixes #1868  (#1920)
 * b97044d2 fix(security): Fixes an issue that allowed you to list archived workf… (#2079)
 * c4f49cf0 refactor: Move server code (cmd/server/ -> server/) (#2071)
 * 2542454c fix(controller): Do not crash if cm is empty. Fixes #2069 (#2070)
 * 85fa9aaf fix: Do not expect workflowChange to always be defined (#2068)
 * 6f65bc2b fix: "base64 -d" not always available, using "base64 --decode" (#2067)
 * 6f2c8802 feat(ui): Use cookies in the UI. Closes #1949 (#2058)
 * 4592aec6 fix(api): Change `CronWorkflowName` to `Name`. Fixes #1982 (#2033)
 * e26c11af fix: only run archived wf testing when persistence is enabled (#2059)
 * b3cab5df fix: Fix permission test cases (#2035)
 * b408e7cd fix: nil pointer in GC (#2055)
 * 4ac11560 fix: offload Node Status in Get and List api call (#2051)
 * dfdde1d0 ci: Run using our own cowsay image (#2047)
 * 71ba8238 Update README.md (#2045)
 * c7953052 fix(persistence): Allow `argo server` to run without persistence (#2050)
 * 1db74e1a fix(archive): upsert archive + ci: Pin images on CI, add readiness probes, clean-up logging and other tweaks (#2038)
 * c46c6836 feat: Allow workflow-level parameters to be modified in the UI when submitting a workflow (#2030)
 * faa9dbb5 fix(Makefile): Rename staticfiles make target. Fixes #2010 (#2040)
 * 79a42d48 docs: Update link to configure-artifact-repository.md (#2041)
 * 1a96007f fix: Redirect to correct page when using managed namespace. Fixes #2015 (#2029)
 * 78726314 fix(api): Updates proto message naming (#2034)
 * 4a1307c8 feat: Adds support for MySQL. Fixes #1945 (#2013)
 * d843e608 chore: Smoke tests are timing out, give them more time (#2032)
 * 5c98a14e feat(controller): Add audit logs to workflows. Fixes #1769 (#1930)
 * 2982c1a8 fix(validate): Allow placeholder in values taken from inputs. Fixes #1984 (#2028)
 * 3293c83f feat: Add version to offload nodes. Fixes #1944 and #1946 (#1974)
 * 283bbf8d build: `make clean` now only deletes dist directories (#2019)
 * 72fa88c9 build: Enable linting for tests. Closes #1971 (#2025)
 * f8569ae9 feat: Auth refactoring to support single version token (#1998)
 * eb360d60 Fix README (#2023)
 * ef1bd3a3 fix typo (#2021)
 * f25a45de feat(controller): Exposes container runtime executor as CLI option. (#2014)
 * 3b26af7d Enable s3 trace support. Bump version to v2.5.0. Tweak proto id to match Workflow (#2009)
 * 5eb15bb5 fix: Fix workflow level timeouts (#1369)
 * 5868982b fix: Fixes the `test` job on master (#2008)
 * 29c85072 fix: Fixed grammar on TTLStrategy (#2006)
 * 2f58d202 fix: v2 token bug (#1991)
 * ed36d92f feat: Add quick start manifests to Git. Change auth-mode to default to server. Fixes #1990 (#1993)
 * d1965c93 docs: Encourage users to upvote issues relevant to them (#1996)
 * 91331a89 fix: No longer delete the argo ns as this is dangerous (#1995)
 * 1a777cc6 feat(cron): Added timezone support to cron workflows. Closes #1931 (#1986)
 * 48b85e57 fix: WorkflowTempalteTest fix (#1992)
 * 51dab8a4 feat: Adds `argo server` command. Fixes #1966 (#1972)
 * 732e03bb chore: Added WorkflowTemplate test (#1989)
 * 27387d4b chore: Fix UI TODOs (#1987)
 * dd704dd6 feat: Renders namespace in UI. Fixes #1952 and #1959 (#1965)
 * 14d58036 feat(server): Argo Server. Closes #1331 (#1882)
 * f69655a0 fix: Added decompress in retry, resubmit and resume. (#1934)
 * 1e7ccb53 updated jq version to 1.6 (#1937)
 * c51c1302 feat: Enhancement for namespace installation mode configuration (#1939)
 * 6af100d5 feat: Add suspend and resume to CronWorkflows CLI (#1925)
 * 232a465d feat: Added onExit handlers to Step and DAG (#1716)
 * 071eb112 docs: Update PR template to demand tests. (#1929)
 * ae58527e docs: Add CyberAgent to the list of Argo users (#1926)
 * 02022e4b docs: Minor formatting fix (#1922)
 * e4107bb8 Updated Readme.md for companies using Argo: (#1916)
 * 7e9b2b58 feat: Support for scheduled Workflows with CronWorkflow CRD (#1758)
 * 5d7e9185 feat: Provide values of withItems maps as JSON in {{item}}. Fixes #1905 (#1906)
 * de3ffd78  feat: Enhanced Different TTLSecondsAfterFinished depending on if job is in Succeeded, Failed or Error, Fixes (#1883)
 * 94449876 docs: Add question option to issue templates (#1910)
 * 83ae2df4 fix: Decrease docker build time by ignoring node_modules (#1909)
 * 59a19069 feat: support iam roles for service accounts in artifact storage (#1899)
 * 6526b6cc fix: Revert node creation logic (#1818)
 * 160a7940 fix: Update Gopkg.lock with dep ensure -update (#1898)
 * ce78227a fix: quick fail after pod termination (#1865)
 * cd3bd235 refactor: Format Argo UI using prettier (#1878)
 * b48446e0 fix: Fix support for continueOn failed for DAG. Fixes #1817 (#1855)
 * 48256961 fix: Fix template scope (#1836)
 * eb585ef7 fix: Use dynamically generated placeholders (#1844)
 * c821cfcc test: Adds 'test' and 'ui' jobs to CI (#1869)
 * 54f44909 feat: Always archive logs if in config. Closes #1790 (#1860)
 * 1e25d6cf docs: Fix e2e testing link (#1873)
 * f5f40728 fix: Minor comment fix (#1872)
 * 72fad7ec Update docs (#1870)
 * 90352865 docs: Update doc based on helm 3.x changes (#1843)
 * 78889895 Move Workflows UI from https://github.com/argoproj/argo-ui (#1859)
 * 4b96172f docs: Refactored and cleaned up docs (#1856)
 * 6ba4598f test: Adds core e2e test infra. Fixes #678 (#1854)
 * 87f26c8d fix: Move ISSUE_TEMPLATE/ under .github/ (#1858)
 * bd78d159 fix: Ensure timer channel is empty after stop (#1829)
 * afc63024 Code duplication (#1482)
 * 5b136713 docs: biobox analytics (#1830)
 * 68b72a8f add CCRi to list of users in README (#1845)
 * 941f30aa Add Sidecar Technologies to list of who uses Argo (#1850)
 * a08048b6 Adding Wavefront to the users list (#1852)
 * 1cb68c98 docs: Updates issue and PR templates. (#1848)
 * cb0598ea Fixed Panic if DB context has issue (#1851)
 * e5fb8848 fix: Fix a couple of nil derefs (#1847)
 * b3d45850 Add HOVER to the list of who uses Argo (#1825)
 * 99db30d6 InsideBoard uses Argo (#1835)
 * ac8efcf4 Red Hat uses Argo (#1828)
 * 41ed3acf Adding Fairwinds to the list of companies that use Argo (#1820)
 * 5274afb9 Add exponential back-off to retryStrategy (#1782)
 * e522e30a Handle operation level errors PVC in Retry (#1762)
 * f2e6054e Do not resolve remote templates in lint (#1787)
 * 3852bc3f SSL enabled database connection for workflow repository (#1712) (#1756)
 * f2676c87 Fix retry node name issue on error (#1732)
 * d38a107c Refactoring Template Resolution Logic (#1744)
 * 23e94604 Error occurred on pod watch should result in an error on the wait container (#1776)
 * 57d051b5 Added hint when using certain tokens in when expressions (#1810)
 * 0e79edff Make kubectl print status and start/finished time (#1766)
 * 723b3c15 Fix code-gen docs (#1811)
 * 711bb114 Fix withParam node naming issue (#1800)
 * 4351a336 Minor doc fix (#1808)
 * efb748fe Fix some issues in examples (#1804)
 * a3e31289 Add documentation for executors (#1778)
 * 1ac75b39 Add  to linter (#1777)
 * 3bead0db Add ability to retry nodes after errors (#1696)
 * b50845e2 Support no-headers flag (#1760)
 * 7ea2b2f8 Minor rework of suspened node (#1752)
 * 9ab1bc88 Update README.md (#1768)
 * e66fa328 Fixed lint issues (#1739)
 * 63e12d09 binary up version (#1748)
 * 1b7f9bec Minor typo fix (#1754)
 * 4c002677 fix blank lines (#1753)
 * fae73826 Fail suspended steps after deadline (#1704)
 * b2d7ee62 Fix typo in docs (#1745)
 * f2592448 Removed uneccessary debug Println (#1741)
 * 846d01ed Filter workflows in list  based on name prefix (#1721)
 * 8ae688c6 Added ability to auto-resume from suspended state (#1715)
 * fb617b63 unquote strings from parameter-file (#1733)
 * 34120341 example for pod spec from output of previous step (#1724)
 * 12b983f4 Add gonum.org/v1/gonum/graph to Gopkg.toml (#1726)
 * 327fcb24 Added  Protobuf extension  (#1601)
 * 602e5ad8 Fix invitation link. (#1710)
 * eb29ae4c Fixes bugs in demo (#1700)
 * ebb25b86 `restartPolicy` -> `retryStrategy` in examples (#1702)
 * 167d65b1 Fixed incorrect `pod.name` in retry pods (#1699)
 * e0818029 fixed broke metrics endpoint per #1634 (#1695)
 * 36fd09a1 Apply Strategic merge patch against the pod spec (#1687)
 * d3546467 Fix retry node processing (#1694)
 * dd517e4c Print multiple workflows in one command (#1650)
 * 09a6cb4e Added status of previous steps as variables (#1681)
 * ad3dd4d4 Fix issue that workflow.priority substitution didn't pass validation (#1690)
 * 095d67f8 Store locally referenced template properly (#1670)
 * 30a91ef0 Handle retried node properly (#1669)
 * 263cb703 Update README.md  Argo Ansible role: Provisioning Argo Workflows on Kubernetes/OpenShift (#1673)
 * 867f5ff7 Handle sidecar killing properly (#1675)
 * f0ab9df9 Fix typo (#1679)
 * 502db42d Don't provision VM for empty artifacts (#1660)
 * b5dcac81 Resolve WorkflowTemplate lazily (#1655)
 * d15994bb [User] Update Argo users list (#1661)
 * 4a654ca6 Stop failing if artifact file exists, but empty (#1653)
 * c6cddafe Bug fixes in getting started (#1656)
 * ec788373 Update workflow_level_host_aliases.yaml (#1657)
 * 7e5af474 Fix child node template handling (#1654)
 * 7f385a6b Use stored templates to raggregate step outputs (#1651)
 * cd6f3627 Fix dag output aggregation correctly (#1649)
 * 706075a5 Fix DAG output aggregation (#1648)
 * fa32dabd Fix missing merged changes in validate.go (#1647)
 * 45716027 fixed example wrong comment (#1643)
 * 69fd8a58 Delay killing sidecars until artifacts are saved (#1645)
 * ec5f9860 pin colinmarc/hdfs to the next commit, which no longer has vendored deps (#1622)
 * 4b84f975 Fix global lint issue (#1641)
 * bb579138 Fix regression where global outputs were unresolveable in DAGs (#1640)
 * cbf99682 Fix regression where parallelism could cause workflow to fail (#1639)
 * 76461f92 Update CHANGELOG for v2.4.0 (#1636)
 * c75a0861 Regenerate installation manifests (#1638)
 * e20cb28c Grant get secret role to controller to support persistence (#1615)
 * 644946e4 Save stored template ID in nodes (#1631)
 * 5d530bec Fix retry workflow state (#1632)
 * 2f0af522 Update operator.go (#1630)
 * 6acea0c1 Store resolved templates (#1552)
 * df8260d6 Increase timeout of golangci-lint (#1623)
 * 138f89f6 updated invite link (#1621)
 * d027188d Updated the API Rule Violations list (#1618)
 * a317fbf1 Prevent controller from crashing due to glog writing to /tmp (#1613)
 * 20e91ea5 Added WorkflowStatus and NodeStatus types to the Open API Spec. (#1614)
 * ffb281a5 Small code cleanup and add tests (#1562)
 * 1cb8345d Add merge keys to Workflow objects to allow for StrategicMergePatches (#1611)
 * c855a66a Increased Lint timeout (#1612)
 * 4bf83fc3 Fix DAG enable failFast will hang in some case (#1595)
 * e9f3d9cb Do not relocate the mounted docker.sock (#1607)
 * 1bd50fa2 Added retry around RuntimeExecutor.Wait call when waiting for main container completion (#1597)
 * 0393427b Issue1571  Support ability to assume IAM roles in S3 Artifacts  (#1587)
 * ffc0c84f Update Gopkg.toml and Gopkg.lock (#1596)
 * aa3a8f1c Update from github.com/ghodss/yaml to sigs.k8s.io/yaml (#1572)
 * 07a26f16 Regard resource templates as leaf nodes (#1593)
 * 89e959e7 Fix workflow template in namespaced controller (#1580)
 * cd04ab8b remove redundant codes (#1582)
 * 5bba8449 Add entrypoint label to workflow default labels (#1550)
 * 9685d7b6 Fix inputs and arguments during template resolution (#1545)
 * 19210ba6 added DataStax as an organization that uses Argo (#1576)
 * b5f2fdef Support AutomountServiceAccountToken and executor specific service account(#1480)
 * 8808726c Fix issue saving outputs which overlap paths with inputs (#1567)
 * ba7a1ed6 Add coverage make target (#1557)
 * ced0ee96 Document workflow controller dockerSockPath config (#1555)
 * 3e95f2da Optimize argo binary install documentation (#1563)
 * e2ebb166 docs(readme): fix workflow types link (#1560)
 * 6d150a15 Initialize the wfClientset before using it (#1548)
 * 5331fc02 Remove GLog config from argo executor (#1537)
 * ed4ac6d0 Update main.go (#1536)
 * 9fca1441 Update argo dependencies to kubernetes v1.14 (#1530)
 * 0246d184 Use cache to retrieve WorkflowTemplates (#1534)
 * 4864c32f Update README.md (#1533)
 * 4df114fa Update CHANGELOG for v2.4 (#1531)
 * c7e5cba1 Introduce podGC strategy for deleting completed/successful pods (#1234)
 * bb0d14af Update ISSUE_TEMPLATE.md (#1528)
 * b5702d8a Format sources and order imports with the help of goimports (#1504)
 * d3ff77bf Added Architecture doc (#1515)
 * fc1ec1a5 WorkflowTemplate CRD (#1312)
 * f99d3266 Expose all input parameters to template as JSON (#1488)
 * bea60526 Fix argo logs empty content when workflow run in virtual kubelet env (#1201)
 * d82de881 Implemented support for WorkflowSpec.ArtifactRepositoryRef (#1350)
 * 0fa20c7b Fix validation (#1508)
 * 87e2cb60 Add --dry-run option to `argo submit` (#1506)
 * e7e50af6 Support git shallow clones and additional ref fetches (#1521)
 * 605489cd Allow overriding workflow labels in 'argo submit' (#1475)
 * 47eba519 Fix issue [Documentation] kubectl get service argo-artifacts -o wide (#1516)
 * 02f38262 Fixed #1287 Executor kubectl version is obsolete (#1513)
 * f62105e6 Allow Makefile variables to be set from the command line (#1501)
 * e62be65b Fix a compiler error in a unit test (#1514)
 * 5c5c29af Fix the lint target (#1505)
 * e03287bf Allow output parameters with .value, not only .valueFrom (#1336)
 * 781d3b8a Implemented Conditionally annotate outputs of script template only when consumed #1359 (#1462)
 * b028e61d change 'continue-on-fail' example to better reflect its description (#1494)
 * 97e824c9 Readme update to add argo and airflow comparison (#1502)
 * 414d6ce7 Fix a compiler error (#1500)
 * ca1d5e67 Fix: Support the List within List type in withParam #1471 (#1473)
 * 75cb8b9c Fix #1366 unpredictable global artifact behavior (#1461)
 * 082e5c4f Exposed workflow priority as a variable (#1476)
 * 38c4def7 Fix: Argo CLI should show warning if there is no workflow definition in file #1486
 * af7e496d Add Commodus Tech as official user (#1484)
 * 8c559642 Update OWNERS (#1485)
 * 007d1f88 Fix: 1008 `argo wait` and `argo submit --wait` should exit 1 if workflow fails  (#1467)
 * 3ab7bc94 Document the insecureIgnoreHostKey git flag (#1483)
 * 7d9bb51a Fix failFast bug:   When a node in the middle fails, the entire workflow will hang (#1468)
 * 42adbf32 Add --no-color flag to logs (#1479)
 * 67fc29c5 fix typo: symboloic > symbolic (#1478)
 * 7c3e1901 Added Codec to the Argo community list (#1477)
 * 0a9cf9d3 Add doc about failFast feature (#1453)
 * 6a590300 Support PodSecurityContext (#1463)
 * e392d854 issue-1445: changing temp directory for output artifacts from root to tmp (#1458)
 * 7a21adfe New Feature:  provide failFast flag, allow a DAG to run all branches of the DAG (either success or failure) (#1443)
 * b9b87b7f Centralized Longterm workflow persistence storage  (#1344)
 * cb09609b mention sidecar in failure message for sidecar containers (#1430)
 * 373bbe6e Fix demo's doc issue of install minio chart (#1450)
 * 83552334 Add threekit to user list (#1444)
 * 83f82ad1 Improve bash completion (#1437)
 * ee0ec78a Update documentation for workflow.outputs.artifacts (#1439)
 * 9e30c06e Revert "Update demo.md (#1396)" (#1433)
 * c08de630 Add paging function for list command (#1420)
 * bba2f9cb Fixed:  Implemented Template level service account (#1354)
 * d635c1de Ability to configure hostPath mount for `/var/run/docker.sock` (#1419)
 * d2f7162a Terminate all containers within pod after main container completes (#1423)
 * 1607d74a PNS executor intermitently failed to capture entire log of script templates (#1406)
 * 5e47256c Fix typo (#1431)
 * 5635c33a Update demo.md (#1396)
 * 83425455 Add OVH as official user (#1417)
 * 82e5f63d Typo fix in ARTIFACT_REPO.md (#1425)
 * 15fa6f52 Update OWNERS (#1429)
 * 96b9a40e Orders uses alphabetically (#1411)
 * 6550e2cb chore: add IBM to official users section in README.md (#1409)
 * bc81fe28 Fiixed: persistentvolumeclaims already exists #1130 (#1363)
 * 6a042d1f Update README.md (#1404)
 * aa811fbd Update README.md (#1402)
 * abe3c99f Add Mirantis as an official user (#1401)
 * 18ab750a Added Argo Rollouts to README (#1388)
 * 67714f99 Make locating kubeconfig in example os independent (#1393)
 * 672dc04f Fixed: withParam parsing of JSON/YAML lists #1389 (#1397)
 * b9aec5f9 Fixed: make verify-codegen is failing on the master branch (#1399) (#1400)
 * 270aabf1 Fixed:  failed to save outputs: verify serviceaccount default:default has necessary privileges (#1362)
 * 163f4a5d Fixed: Support hostAliases in WorkflowSpec #1265 (#1365)
 * abb17478 Add Max Kelsen to USERS in README.md (#1374)
 * dc549193 Update docs for the v2.3.0 release and to use the stable tag
 * 4001c964 Update README.md (#1372)
 * 6c18039b Fix issue where a DAG with exhausted retries would get stuck Running (#1364)
 * d7e74fe3 Validate action for resource templates (#1346)
 * 810949d5 Fixed :  CLI Does Not Honor metadata.namespace #1288 (#1352)
 * e58859d7 [Fix #1242] Failed DAG nodes are now kept and set to running on RetryWorkflow. (#1250)
 * d5fe5f98 Use golangci-lint instead of deprecated gometalinter (#1335)
 * 26744d10 Support an easy way to set owner reference (#1333)
 * 8bf7578e Add --status filter for get command (#1325)
 * 3f6ac9c9 Update release instructions
 * 2274130d Update version to v2.3.0-rc3
 * b024b3d8 Fix: # 1328 argo submit --wait and argo wait quits while workflow is running (#1347)
 * 24680b7f Fixed : Validate the secret credentials name and key (#1358)
 * f641d84e Fix input artifacts with multiple ssh keys (#1338)
 * e680bd21 add / test (#1240)
 * ee788a8a Fix #1340 parameter substitution bug (#1345)
 * 60b65190 Fix missing template local volumes, Handle volumes only used in init containers (#1342)
 * 4e37a444 Add documentation on releasing
 * bb1bfdd9 Update version to v2.3.0-rc2. Update changelog
 * 49a6b6d7 wait will conditionally become privileged if main/sidecar privileged (resolves #1323)
 * 34af5a06 Fix regression where argoexec wait would not return when podname was too long
 * bd8d5cb4 `argo list` was not displaying non-zero priorities correctly
 * 64370a2d Support parameter substitution in the volumes attribute (#1238)
 * 6607dca9 Issue1316 Pod creation with secret volumemount  (#1318)
 * a5a2bcf2 Update README.md (#1321)
 * 950de1b9 Export the methods of `KubernetesClientInterface` (#1294)
 * 1c729a72 Update v2.3.0 CHANGELOG.md
 * 40f9a875 Reorganize manifests to kustomize 2 and update version to v2.3.0-rc1
 * 75b28a37 Implement support for PNS (Process Namespace Sharing) executor (#1214)
 * b4edfd30 Fix SIGSEGV in watch/CheckAndDecompress. Consolidate duplicate code (resolves #1315)
 * 02550be3 Archive location should conditionally be added to template only when needed
 * c60010da Fix nil pointer dereference with secret volumes (#1314)
 * db89c477 Fix formatting issues in examples documentation (#1310)
 * 0d400f2c Refactor checkandEstimate to optimize podReconciliation (#1311)
 * bbdf2e2c Add alibaba cloud to officially using argo list (#1313)
 * abb77062 CheckandEstimate implementation to optimize podReconciliation (#1308)
 * 1a028d54 Secrets should be passed to pods using volumes instead of API calls (#1302)
 * e34024a3 Add support for init containers (#1183)
 * 4591e44f Added support for artifact path references (#1300)
 * 928e4df8 Add Karius to users in README.md (#1305)
 * de779f36 Add community meeting notes link (#1304)
 * a8a55579 Speed up podReconciliation using parallel goroutine (#1286)
 * 93451119 Add dns config support (#1301)
 * 850f3f15 Admiralty: add link to blog post, add user (#1295)
 * d5f4b428 Fix for Resource creation where template has same parameter templating (#1283)
 * 9b555cdb Issue#896 Workflow steps with non-existant output artifact path will succeed (#1277)
 * adab9ed6 Argo CI is current inactive (#1285)
 * 59fcc5cc Add workflow labels and annotations global vars (#1280)
 * 1e111caa Fix bug with DockerExecutor's CopyFile (#1275)
 * 73a37f2b Add the `mergeStrategy` option to resource patching (#1269)
 * e6105243 Reduce redundancy pod label action (#1271)
 * 4bfbb20b Error running 1000s of tasks: "etcdserver: request is too large" #1186 (#1264)
 * b2743f30 Proxy Priority and PriorityClassName to pods (#1179)
 * 70c130ae Update versions (#1218)
 * b0384129 Git cloning via SSH was not verifying host public key (#1261)
 * 3f06385b Issue#1165 fake outputs don't notify and task completes successfully (#1247)
 * fa042aa2 typo, executo -> executor (#1243)
 * 1cb88bae Fixed Issue#1223 Kubernetes Resource action: patch is not supported (#1245)
 * 2b0b8f1c Fix the Prometheus address references (#1237)
 * 94cda3d5 Add feature to continue workflow on failed/error steps/tasks (#1205)
 * 3f1fb9d5 Add Gardener to "Who uses Argo" (#1228)
 * cde5cd32 Include stderr when retrieving docker logs (#1225)
 * 2b1d56e7 Update README.md (#1224)
 * eeac5a0e Remove extra quotes around output parameter value (#1232)
 * 8b67e1bf Update README.md (#1236)
 * baa3e622 Update README with typo fixes (#1220)
 * f6b0c8f2 Executor can access the k8s apiserver with a out-of-cluster config file (#1134)
 * 0bda53c7 fix dag retries (#1221)
 * 8aae2931 Issue #1190 - Fix incorrect retry node handling (#1208)
 * f1797f78 Add schedulerName to workflow and template spec (#1184)
 * 2ddae161 Set executor image pull policy for resource template (#1174)
 * edcb5629 Dockerfile: argoexec base image correction (fixes #1209) (#1213)
 * f92284d7 Minor spelling, formatting, and style updates. (#1193)
 * bd249a83 Issue #1128 - Use polling instead of fs notify to get annotation changes (#1194)
 * 14a432e7 Update community/README (#1197)
 * eda7e084 Updated OWNERS (#1198)
 * 73504a24 Fischerjulian adds ruby to rest docs (#1196)
 * 311ad86f Fix missing docker binary in argoexec image. Improve reuse of image layers
 * 831e2198 Issue #988 - Submit should not print logs to stdout unless output is 'wide' (#1192)
 * 17250f3a Add documentation how to use parameter-file's (#1191)
 * 01ce5c3b Add Docker Hub build hooks
 * 93289b42 Refactor Makefile/Dockerfile to remove volume binding in favor of build stages (#1189)
 * 8eb4c666 Issue #1123 - Fix 'kubectl get' failure if resource namespace is different from workflow namespace (#1171)
 * eaaad7d4 Increased S3 artifact retry time and added log (#1138)
 * f07b5afe Issue #1113 - Wait for daemon pods completion to handle annotations (#1177)
 * 2b2651b0 Do not mount unnecessary docker socket (#1178)
 * 1fc03144 Argo users: Equinor (#1175)
 * e381653b Update README. (#1173) (#1176)
 * 5a917140 Update README and preview notice in CLA.
 * 521eb25a Validate ArchiveLocation artifacts (#1167)
 * 528e8f80 Add missing patch in namespace kustomization.yaml (#1170)
 * 0b41ca0a Add Preferred Networks to users in README.md (#1172)
 * 649d64d1 Add GitHub to users in README.md (#1151)
 * 864c7090 Update codegen for network config (#1168)
 * c3cc51be Support HDFS Artifact (#1159)
 * 8db00066 add support for hostNetwork & dnsPolicy config (#1161)
 * 149d176f Replace exponential retry with poll (#1166)
 * 31e5f63c Fix tests compilation error (#1157)
 * 6726d9a9 Fix failing TestAddGlobalArtifactToScope unit test
 * 4fd758c3 Add slack badge to README (#1164)
 * 3561bff7 Issue #1136 - Fix metadata for DAG with loops (#1149)
 * c7fec9d4 Reflect minio chart changes in documentation (#1147)
 * f6ce7833 add support for other archs (#1137)
 * cb538489 Fix issue where steps with exhausted retires would not complete (#1148)
 * e400b65c Fix global artifact overwriting in nested workflow (#1086)
 * 174eb20a Issue #1040 - Kill daemoned step if workflow consist of single daemoned step (#1144)
 * e078032e Issue #1132 - Fix panic in ttl controller (#1143)
 * e09d9ade Issue #1104 - Remove container wait timeout from 'argo logs --follow' (#1142)
 * 0f84e514 Allow owner reference to be set in submit util (#1120)
 * 3484099c Update generated swagger to fix verify-codegen (#1131)
 * 587ab1a0 Fix output artifact and parameter conflict (#1125)
 * 6bb3adbc Adding Quantibio in Who uses Argo (#1111)
 * 1ae3696c Install mime-support in argoexec to set proper mime types for S3 artifacts (resolves #1119)
 * 515a9005 add support for ppc64le and s390x (#1102)
 * 78142837 Remove docker_lib mount volume which is not needed anymore (#1115)
 * e59398ad Fix examples docs of parameters. (#1110)
 * ec20d94b Issue #1114 - Set FORCE_NAMESPACE_ISOLATION env variable in namespace install manifests (#1116)
 * 49c1fa4f Update docs with examples using the K8s REST API
 * bb8a6a58 Update ROADMAP.md
 * 46855dcd adding logo to be used by the OS Site (#1099)
 * 438330c3 #1081 added retry logic to s3 load and save function (#1082)
 * cb8b036b Initialize child node before marking phase. Fixes panic on invalid `When` (#1075)
 * 60b508dd Drop reference to removed `argo install` command. (#1074)
 * 62b24368 Fix typo in demo.md (#1089)
 * b5dfa021 Use relative links on README file (#1087)
 * 95b72f38 Update docs to outline bare minimum set of privileges for a workflow
 * d4ef6e94 Add new article and minor edits. (#1083)
 * afdac9bb Issue #740 - System level workflow parallelism limits & priorities (#1065)
 * a53a76e9 fix #1078 Azure AKS authentication issues (#1079)
 * 79b3e307 Fix string format arguments in workflow utilities. (#1070)
 * 76b14f54 Auto-complete workflow names (#1061)
 * f2914d63 Support nested steps workflow parallelism (#1046)
 * eb48c23a Raise not implemented error when artifact saving is unsupported (#1062)
 * 036969c0 Add Cratejoy to list of users (#1063)
 * a07bbe43 Adding SAP Hybris in Who uses Argo (#1064)
 * 7ef1cea6 Update dependencies to K8s v1.12 and client-go 9.0
 * 23d733ba Add namespace explicitly to pod metadata (#1059)
 * 79ed7665 Parameter and Argument names should support snake case (#1048)
 * 6e6c59f1 Submodules are dirty after checkout -- need to update (#1052)
 * f18716b7 Support for K8s API based Executor (#1010)
 * e297d195 Updated examples/README.md (#1051)
 * 19d6cee8 Updated ARTIFACT_REPO.md (#1049)
 * 0a928e93 Update installation manifests to use v2.2.1
 * 3b52b261 Fix linter warnings and update swagger
 * 7d0e77ba Update changelog and bump version to 2.2.1
 * b402e12f Issue #1033 - Workflow executor panic: workflows.argoproj.io/template workflows.argoproj.io/template not found in annotation file (#1034)
 * 3f2e986e fix typo in examples/README.md (#1025)
 * 9c5e056a Replace tabs with spaces (#1027)
 * 091f1407 Update README.md (#1030)
 * 159fe09c Fix format issues to resolve build errors (#1023)
 * 363bd97b Fix error in env syntax (#1014)
 * ae7bf0a5 Issue #1018 - Workflow controller should save information about archived logs in step outputs (#1019)
 * 15d006c5 Add example of workflow using imagePullSecrets (resolves #1013)
 * 2388294f Fix RBAC roles to include workflow delete for GC to work properly (resolves #1004)
 * 6f611cb9 Fix issue where resubmission of a terminated workflow creates a terminated workflow (issue #1011)
 * 4a7748f4 Disable Persistence in the demo example (#997)
 * 55ae0cb2 Fix example pod name (#1002)
 * c275e7ac Add imagePullPolicy config for executors (#995)
 * b1eed124 `tar -tf` will detect compressed tars correctly. (#998)
 * 03a7137c Add new organization using argo (#994)
 * 83884528 Update argoproj/pkg to trim leading/trailing whitespace in S3 credentials (resolves #981)
 * 978b4938 Add syntax highlighting for all YAML snippets and most shell snippets (#980)
 * 60d5dc11 Give control to decide whether or not to archive logs at a template level
 * 8fab73b1 Detect and indicate when container was OOMKilled
 * 47a9e556 Update config map doc with instructions to enable log archiving
 * 79dbbaa1 Add instructions to match git URL format to auth type in git example (issue #979)
 * 429f03f5 Add feature list to README.md. Tweaks to getting started.
 * 36fd1948 Update getting started guide with v2.2.0 instructions
 * af636ddd Update installation manifests to use v2.2.0
 * 7864ad36 Introduce `withSequence` to iterate a range of numbers in a loop (resolves #527)
 * 99e9977e Introduce `argo terminate` to terminate a workflow without deleting it (resolves #527)
 * f52c0450 Reorganize codebase to make CLI functionality available as a library
 * 311169f7 Fix issue where sidecars and daemons were not reliably killed (resolves #879)
 * 67ffb6eb Add a reason/message for Unschedulable Pending pods
 * 69c390f2 Support for workflow level timeouts (resolves #848)
 * f88732ec Update docs to use keyFormat field
 * 0df022e7 Rename keyPattern to keyFormat. Remove pending pod query during pod reconciliation
 * 75a9983b Fix potential panic in `argo watch`
 * 9cb46449 Add TTLSecondsAfterFinished field and controller to garbage collect completed workflows (resolves #911)
 * 7540714a Add ability to archive container logs to the artifact repository (resolves #454)
 * 11e57f4d Introduce archive strategies with ability to disable tar.gz archiving (resolves #784)
 * e180b547 Update CHANGELOG.md
 * 5670bf5a Introduce `argo watch` command to watch live workflows from terminal (resolves #969)
 * 57394361 Support additional container runtimes through kubelet executor (#952)
 * a9c84c97 Error workflows which hit k8s/etcd 1M resource size limit (resolves #913)
 * 67792eb8 Add parameter-file support (#966)
 * 841832a3 Aggregate workflow RBAC roles to built-in admin/edit/view clusterroles (resolves #960)
 * 35bb7093 Allow scaling of workflow and pod workers via controller CLI flags (resolves #962)
 * b479fa10 Improve workflow configmap documentation for keyPattern
 * f1802f91 Introduce `keyPattern` workflow config to enable flexibility in archive location path (issue #953)
 * a5648a96 Fix kubectl proxy link for argo-ui Service (#963)
 * 09f05912 Introduce Pending node state to highlight failures when start workflow pods
 * a3ff464f Speed up CI job
 * 88627e84 Update base images to debian:9.5-slim. Use stable metalinter
 * 753c5945 Update argo-ci-builder image with new dependencies
 * 674b61bb Remove unnecessary dependency on argo-cd and obsolete RBAC constants
 * 60658de0 Refactor linting/validation into standalone package. Support linting of .json files
 * f55d579a Detect and fail upon unknown fields during argo submit & lint (resolves #892)
 * edf6a574 Migrate to using argoproj.io/pkg packages
 * 5ee1e0c7 Update artifact config docs (#957)
 * faca49c0 Updated README
 * 936c6df7 Add table of content to examples (#956)
 * d2c03f67 Correct image used in install manifests
 * ec3b7be0 Remove CLI installer/uninstaller. Executor image configured via CLI argument (issue #928) Remove redundant/unused downward API metadata
 * 3a85e242 Rely on `git checkout` instead of go-git checkout for more reliable revision resolution
 * ecef0e3d Rename Prometheus metrics (#948)
 * b9cffe9c Issue #896 - Prometheus metrics and telemetry (#935)
 * 290dee52 Support parameter aggregation of map results in scripts
 * fc20f5d7 Fix errors when submodules are from different URL (#939)
 * b4f1a00a Add documentation about workflow variables
 * 4a242518 Update readme.md (#943)
 * a5baca60 Support referencing of global workflow artifacts (issue #900)
 * 9b5c8563 Support for sophisticated expressions in `when` conditionals (issue #860)
 * ecc0f027 Resolve revision added ability to specify shorthand revision and other things like HEAD~2 etc (#936)
 * 11024318 Support conditions with DAG tasks. Support aggregated outputs from scripts (issue #921)
 * d07c1d2f Support withItems/withParam and parameter aggregation with DAG templates (issue #801)
 * 94c195cb Bump VERSION to v2.2.0
 * 9168c59d Fix outbound node metadata with retry nodes causing disconnected nodes to be rendered in UI (issue #880)
 * c6ce48d0 Fix outbound node metadata issue with steps template causing incorrect edges to be rendered in UI
 * 520b33d5 Add ability to aggregate and reference output parameters expanded by loops (issue #861)
 * ece1eef8 Support submission of workflows as json, and from stdin (resolves #926)
 * 4c31d61d Add `argo delete --older` to delete completed workflows older than specified duration (resolves #930)
 * c87cd33c Update golang version to v1.10.3
 * 618b7eb8 Proper fix for assessing overall DAG phase. Add some DAG unit tests (resolves #885)
 * f223e5ad Fix issue where a DAG would fail even if retry was successful (resolves #885)
 * 143477f3 Start use of argoproj/pkg shared libraries
 * 1220d080 Update argo-cluster-role to work with OpenShift (resolves #922)
 * 4744f45a Added SSH clone and proper git clone using go-git (#919)
 * d657abf4 Regenerate code and address OpenAPI rule validation errors (resolves #923)
 * c5ec4cf6 Upgrade k8s dependencies to v1.10 (resolves #908)
 * ba8061ab Redundant verifyResolvedVariables check in controller precluded the ability to use {{ }} in other circumstances
 * 05a61449 Added link to community meetings (#912)
 * f33624d6 Add an example on how to submit and wait on a workflow
 * aeed7f9d Added new members
 * 288e4fc8 Added Argo Events link.
 * 3322506e Updated README
 * 3ce640a2 Issue #889 - Support retryStrategy for scripts (#890)
 * 91c6afb2 adding BlackRock as corporate contributor/user (#886)
 * c8667b5c Fix issue where `argo lint` required spec level arguments to be supplied
 * ed7dedde Update influx-ci example to choose a stable InfluxDB branch
 * 135813e1 Add datadog to the argo users (#882)
 * f1038948 Fix `make verify-codegen` build target when run in CI
 * 785f2cbd Update references to v2.1.1. Add better checks in release Makefile target
 * d65e1cd3 readme: add Interline Technologies to user list (#867)
 * c903168e Add documentation on global parameters (#871)
 * ac241c95 Update CHANGELOG for v2.1.1
 * 468e0760 Retrying failed steps templates could potentially result in disconnected children
 * 8d96ea7b Switch to an UnstructuredInformer to guard against malformed workflow manifests (resolves #632)
 * 5bef6cae Suspend templates were not properly being connected to their children (resolves #869)
 * 543e9392 Fix issue where a failed step in a template with parallelism would not complete (resolves #868)
 * 289000ca Update argocli Dockerfile and make cli-image part of release
 * d35a1e69 Bump version to v2.1.1
 * bbcff0c9 Fix issue where `argo list` age column maxed out at 1d (resolves #857)
 * d68cfb7e Fix issue where volumes were not supported in script templates (resolves #852)
 * fa72b6db Fix implementation of DAG task targets (resolves #865)
 * dc003f43 Children of nested DAG templates were not correctly being connected to its parent
 * b8065797 Simplify some examples for readability and clarity
 * 7b02c050 Add CoreFiling to "Who uses Argo?" section. (#864)
 * 4f2fde50 Add windows support for argo-cli (#856)
 * 703241e6 Updated ROADMAP.md for v2.2
 * 54f2138e Spell check the examples README (#855)
 * f23feff5 Mkbranch (#851)
 * 628b5408 DAG docs. (#850)
 * 22f62439 Small edit to README
 * edc09afc Added OWNERS file
 * 530e7244 Update release notes and documentation for v2.1.0

### Contributors

 * 0x1D-1983
 * Aaron Curtis
 * Adam Gilat
 * Adam Thornton
 * Aditya Sundaramurthy
 * Adrien Trouillaud
 * Aisuko
 * Akshay Chitneni
 * Alessandro Marrella
 * Alex Capras
 * Alex Collins
 * Alex Stein
 * Alexander Matyushentsev
 * Alexey Volkov
 * Anastasia Satonina
 * Andrei Miulescu
 * Andrew Suderman
 * Anes Benmerzoug
 * Anna Winkler
 * Antoine Dao
 * Antonio Macías Ojeda
 * Appréderisse Benjamin
 * Avi Weit
 * Bastian Echterhölter
 * Ben Wells
 * Ben Ye
 * Brandon Steinman
 * Brian Mericle
 * CWen
 * Caden
 * Caglar Gulseni
 * Chen Zhiwei
 * Chris Chambers
 * Christian Muehlhaeuser
 * Clemens Lange
 * Cristian Pop
 * Daisuke Taniwaki
 * Dan Norris
 * Daniel Duvall
 * Daniel Moran
 * Daniel Sutton
 * David Bernard
 * David Seapy
 * David Van Loon
 * Deepen Mehta
 * Derek Wang
 * Dineshmohan Rajaveeran
 * Divya Vavili
 * Drew Dara-Abrams
 * Dustin Specker
 * EDGsheryl
 * Ed Lee
 * Edward Lee
 * Edwin Jacques
 * Ejiah
 * Elton
 * Eric
 * Erik Parmann
 * Fabio Rigato
 * Feynman Liang
 * Florent Clairambault
 * Fred Dubois
 * Gabriele Santomaggio
 * Galen Han
 * Grant Stephens
 * Greg Roodt
 * Guillaume Hormiere
 * Hamel Husain
 * Heikki Kesa
 * Hideto Inamura
 * Howie Benefiel
 * Huan-Cheng Chang
 * Ian Howell
 * Ilias K
 * Ilias Katsakioris
 * Ilya Sotkov
 * Ismail Alidzhikov
 * Jacob O'Farrell
 * Jaime
 * Jean-Louis Queguiner
 * Jeff Uren
 * Jesse Suen
 * Jialu Zhu
 * Jie Zhang
 * Johannes 'fish' Ziemke
 * John Wass
 * Jonas Fonseca
 * Jonathan Steele
 * Jonathon Belotti
 * Jonny
 * Joshua Carp
 * Juan C. Muller
 * Julian Fahrer
 * Julian Fischer
 * Julian Mazzitelli
 * Julien Balestra
 * Kannappan Sirchabesan
 * Konstantin Zadorozhny
 * Leonardo Luz
 * Marcin Karkocha
 * Marco Sanvido
 * Marek Čermák
 * Markus Lippert
 * Matt Brant
 * Matt Hillsdon
 * Matthew Coleman
 * Matthew Magaldi
 * MengZeLee
 * Michael Crenshaw
 * Mike Seddon
 * Mingjie Tang
 * Miyamae Yuuya
 * Mostapha Sadeghipour Roudsari
 * Mukulikak
 * Naoto Migita
 * Naresh Kumar Amrutham
 * Nasrudin Bin Salim
 * Neutron Soutmun
 * Nick Groszewski
 * Nick Stott
 * Niklas Hansson
 * Niklas Vest
 * Nándor István Krácser
 * Omer Kahani
 * Orion Delwaterman
 * Pablo Osinaga
 * Pascal VanDerSwalmen
 * Patryk Jeziorowski
 * Paul Brit
 * Pavel Kravchenko
 * Peng Li
 * Pengfei Zhao
 * Per Buer
 * Peter Salanki
 * Pierre Houssin
 * Pradip Caulagi
 * Praneet Chandra
 * Pratik Raj
 * Premkumar Masilamani
 * Rafael Rodrigues
 * Rafał Bigaj
 * Remington Breeze
 * Rick Avendaño
 * Rocio Montes
 * Romain Di Giorgio
 * Romain GUICHARD
 * Roman Galeev
 * Saradhi Sreegiriraju
 * Saravanan Balasubramanian
 * Sascha Grunert
 * Sean Fern
 * Semjon Kopp
 * Shubham Koli (FaultyCarry)
 * Simon Behar
 * Snyk bot
 * Song Juchao
 * Stephen Steiner
 * StoneHuang
 * Takashi Abe
 * Takayuki Kasai
 * Tang Lee
 * Theodore Messinezis
 * Tim Schrodi
 * Tobias Bradtke
 * Tom Wieczorek
 * Tomas Valasek
 * Trevor Foster
 * Tristan Colgate-McFarlane
 * Val Sichkovskyi
 * Vardan Manucharyan
 * Vincent Boulineau
 * Vincent Smith
 * Vlad Losev
 * Wei Yan
 * WeiYan
 * Weston Platter
 * William
 * William Reed
 * Xianlu Bird
 * Xie.CS
 * Xin Wang
 * Youngjoon Lee
 * Yuan Tang
 * Yunhai Luo
 * Zach Aller
 * Zach Himsel
 * Zhipeng Wang
 * Ziyang Wang
 * alex weidner
 * almariah
 * candonov
 * commodus-sebastien
 * descrepes
 * dherman
 * dmayle
 * dthomson25
 * fsiegmund
 * gerardaus
 * gerdos82
 * haibingzhao
 * hidekuro
 * houz
 * ianCambrio
 * jacky
 * jdfalko
 * kshamajain99
 * lueenavarro
 * maguowei
 * mark9white
 * maryoush
 * mdvorakramboll
 * nglinh
 * sang
 * shahin
 * shibataka000
 * tkilpela
 * tralexa
 * tunoat
 * vatine
 * vdinesh2461990
 * xubofei1983
 * zhujl1991
 * モハメド

## v2.1.0 (2018-04-30)

 * 93796381 Avoid `println` which outputs to stderr. (#844)
 * 30e472e9 Add gladly as an official argo user (#843)
 * cb4c1a13 Add ability to override metadata.name and/or metadata.generateName during submission (resolves #836)
 * 834468a5 Command print the logs for a container in a workflow
 * 1cf13f9b Issue #825 - fix locating outbound nodes for skipped node (#842)
 * 30034d42 Bump from debian:9.1 to debian:9.4. (#841)
 * f3c41717 Owner reference example (#839)
 * 191f7aff Minor edit to README
 * c8a2e25f Fixed typo (#835)
 * cf13bf0b Added users section to README
 * e4d76329 Updated News in README
 * b631d0af added community meeting (#834)
 * e34728c6 Fix issue where daemoned steps were not terminated properly in DAG templates (resolves #832)
 * 2e9e113f Update docs to work with latest minio chart
 * ea95f191 Use octal syntax for mode values (#833)
 * 5fc67d2b Updated community docs
 * 8fa4f006 Added community docs
 * 423c8d14 Issue #830 - retain Step node children references
 * 73990c78 Moved cricket gifs to a different s3 bucket
 * ca1858ca edit Argo license info so that GitHub recognizes it (#823)
 * 206451f0 Fix influxdb-ci.yml example
 * da582a51 Avoid nil pointer for 2.0 workflows. (#820)
 * 0f225cef ClusterRoleBinding was using incorrect service account namespace reference when overriding install namespace (resolves #814)
 * 66ea711a Issue #816 - fix updating outboundNodes field of failed step group node (#817)
 * 00ceef6a install & uninstall commands use --namespace flag (#813)
 * fe23c2f6 Issue #810 - `argo install`does not install argo ui (#811)
 * 28673ed2 Update release date in change log
 * 05e8a983 Update change log for 2.1.0-beta1 release
 * bf38b6b5 Use socket type for hostPath to mount docker.sock (#804) (#809)
 * 37680ef2 Minimal shell completion support (#807)
 * c83ad24a Omit empty status fields. (#806)
 * d7291a3e Issue #660 - Support rendering logs from all steps using 'argo logs' command (#792)
 * 7d3f1e83 Minor edits to README
 * 7a4c9c1f Added a review to README
 * 383276f3 Inlined LICENSE file. Renamed old license to COPYRIGHT
 * 91d0f47f Build argo cli image (#800)
 * 3b2c426e Add ability to pass pod annotations and labels at the template level (#798)
 * d8be0287 Add ability to use IAM role from EC2 instance for AWS S3 credentials
 * 624f0f48 Update CHANGELOG.md for v2.1.0-beta1 release
 * e96a09a3 Allow spec.arguments to be not supplied during linting. Global parameters were not referencable from artifact arguments (resolves #791)
 * 018e663a Fix for https://github.com/argoproj/argo/issues/739 Nested stepgroups render correctly (#790)
 * 5c5b35ba Fix install issue where service account was not being created
 * 88e9e5ec packr needs to run compiled in order to cross compile darwin binaries
 * dcdf9acf Fix install tests and build failure
 * 06c0d324 Rewrite the installer such that manifests are maintainable
 * a45bf1b7 Introduce support for exported global output parameters and artifacts
 * 60c48a9a Introduce `argo retry` to retry a failed workflow with the same name (resolves #762) onExit and related nodes should never be executed during resubmit/retry (resolves #780)
 * 90c08bff Refactor command structure
 * 101509d6 Abstract the container runtime as an interface to support mocking and future runtimes Trim a trailing newline from path-based output parameters (resolves #758)
 * a3441d38 Add ability to reference global parameters in spec level fields (resolves #749)
 * cd73a9ce Fix template.parallelism limiting parallelism of entire workflow (resolves #772) Refactor operator to make template execution method signatures consistent
 * 7d7b74fa Make {{pod.name}} available as a parameter in pod templates (resolves #744)
 * 3cf4bb13 parse the artifactory URL before appending the artifact to the path (#774)
 * ea1257f7 examples: use alpine python image
 * 2114078c fix typo
 * 9f605589 Fix retry-container-to-completion example
 * 07422f26 Update CHANGELOG release date. Remove ui-image from release target
 * 5d60d073 Fix make release target
 * a013fb38 Fix inability to override LDFLAGS when env variables were supplied to make
 * f63e552b Minor spell fix for parallelism
 * 88d2ff3a Add UI changes description for 2.1.0-alpha1 release (#761)
 * ce4edb8d Add contributor credits
 * cc8f35b6 Add note about region discovery.
 * 9c691a7c Trim spaces from aws keys
 * 17e24481 add keyPrefix option to ARTIFACT_REPO.md
 * 57a568bf Issue #747 - Support --instanceId parameter in submit a workflow (#748)
 * 81a6cd36 Move UI code to separate repository (#742)
 * 10c7de57 Fix rbac resource versions in install
 * 2756e83d Support workflow pod tolerations
 * 9bdab63f Add workflow.namespace to global parameters
 * 8bf7a1ad Statically link argo linux binary (resolves #735)
 * 813cf8ed Add NodeStatus.DisplayName to remove CLI/UI guesswork from displaying node names (resolves #731)
 * e783ccbd Rename some internal template type names for consistency
 * 19dd406c Introduce suspend templates for suspending a workflow at a predetermined step (resolves #702). Make suspend part of the workflow spec instead of infering parallism in status.
 * d6489e12 Rename pause to suspend
 * f1e2f63d Change definition of WorkflowStep.Item to a struct instead of interface{} (resolves #723) Add better withItems unit testing and validation
 * cd18afae Missed handling of a error during workflow resubmission
 * a7ca59be Support resubmission of failed workflows with ability to re-use successful steps (resolves #694)
 * 76b41877 Include inputs as part of NodeStatus (resolves #730)
 * ba683c1b Support for manual pausing and resuming of workflows via Argo CLI (resolves #729)
 * 5a806f93 Add DAG gif for argo wiki (#728)
 * 62a3fba1 Implement support for DAG templates to have output parameters/artifacts
 * 989e8ed2 Support parameter and artifact passing between DAG tasks. Improved template validation
 * 03d409a3 Switch back to Updating CRDs (from Patch) to enable better unit testing
 * 2da685d9 Fixed typos in examples/README.md
 * 6cf94b1b Added output parameter example to examples/README.md
 * 0517096c Add templateName as part of NodeStatus for UI consumption Simplify and centralize parallelism check into executeTemplate() Improved template validation
 * deae4c65 Add parallelism control at the steps template level
 * c788484e Remove hard-wired executor limits and make it configurable in the controller (resolves #724)
 * f27c7ffd Fix linting issues (ineffassign, errcheck)
 * 98a44c99 Embed container type into the script template instead of cherry-picking fields (resolves #711)
 * c0a8f949 Bump VERSION to 2.1.0
 * 207de824 Add parallism field to limit concurrent pod execution at a workflow level (issue #666)
 * 460c9555 Do not initialize DAG task nodes if they did not execute
 * 13a60936 Merge branch 'feature-dag'
 * 931d7723 Update docs to refer to v2.0.0
 * 0978b9c6 Support setting UI base Url  (#722)
 * b75cd98f updated argo-user slack link
 * b3598d84 Add examples as functional and expected failure e2e tests
 * 83966e60 Fix regression where executor did not annotate errors correctly
 * 751fd270 Update UI references to v2.0.0. Update changelog
 * 75caa877 Initial work for dag based cli for everything. get now works (#714)
 * 8420deb3 Skipped steps were being re-initialized causing a controller panic
 * 491ed08f Check-in the OpenAPI spec. Automate generation as part of `make update-codegen`
 * 8b7e2e24 Check-in the OpenAPI spec. Automate generation as part of `make update-codegen`
 * 17241165 Merge branch 'master' into feature-dag
 * 563bda75 Fix update-openapigen.sh script to presume bash. Tweak documentation
 * 5b9a602b Add documentation to types. Add program to generate OpenAPI spec
 * 42726910 Fix retry in dag branch (#709)
 * 97204e64 Merge branch 'master' into feature-dag
 * d929e79f Generate OpenAPI models for the workflow spec (issue #707)
 * 1d5afee6 Shortened url
 * 617d848d Added news to README
 * ae36b22b Fix typo s/Customer/Custom/ (#704)
 * 5a589fcd Add ability to specify imagePullSecrets in the workflow.spec (resolves #699)
 * 2f77bc1e Add ability to specify affinity rules at both the workflow and template level (resolves #701)
 * c2dd9b63 Fix unit test breakages
 * d38324b4 Add boundaryID field in NodeStatus to group nodes by template boundaries
 * 639ad1e1 Introduce Type field in NodeStatus to to assist with visualization
 * 17601c61 Merge branch 'master' into feature-dag
 * fdafbe27 Sidecars unable to reference volume claim templates (resolves #697)
 * 0b0b52c3 Referencing output artifacts from a container with retries was not functioning (resolves #698)
 * 9597f82c Initial support for DAG based workflows (#693)
 * bf2b376a Update doc references to point to v2.0.0-beta1. Fix secrets example

### Contributors

 * Adam Pearse
 * Alexander Matyushentsev
 * Andrea Kao
 * Dan Bode
 * David Kale
 * Divya Vavili
 * Dmitry Monakhov
 * Edward Lee
 * Javier Castellanos
 * Jesse Dubay
 * Jesse Suen
 * Johannes 'fish' Ziemke
 * Lukasz Lempart
 * Matt Hillsdon
 * Mukulikak
 * Sean Fitzgerald
 * Sebastien Doido
 * Yang Pan
 * dougsc
 * gaganapplatix

## v2.0.0-beta1 (2018-01-18)

 * 549870c1 Fix argo-ui download links to point to v2.0.0-beta1
 * a202049d Update CHANGELOG for v2.0.0-beta1
 * a3739035 Remove dind requirement from argo-ci test steps
 * 1bdd0c03 Include completed pods when attempting to reconcile deleted pods Switch back to Patch (from Update) for persisting workflow changes
 * a4a43892 Sleep 1s after persisting workflow to give informer cache a chance to sync (resolves #686)
 * 5bf49531 Updated demo.md with link to ARTIFACT_REPO.md
 * 863d547a Rely on controller generated timestamps for node.StartedAt instad of pod.CreationTimestamp
 * 672542d1 Re-apply workflow changes and reattempt update on resource conflicts. Make completed pod labeling asynchronous
 * 81bd6d3d Resource state retry (#690)
 * 44dba889 Tune controller to 20 QPS, 30 Burst, 8 wf workers, 8 pod workers
 * 178b9d37 Show running/completed pod counts in `argo list -o wide`
 * 0c565f5f Switch to Updating workflow resources instead of Patching (resolves #686)
 * a571f592 Ensure sidecars get killed unequivocally. Final argoexec stats were not getting printed
 * a0b2d78c Show duration by default in `argo get`. --since flag should always include Running
 * 10110313 Executor hardening: add retries and memoization for executor k8s API calls Recover from unexpected panics and annotate the error.
 * f2b8f248 Regenerate deepcopy code after type changes for raw input artifacts
 * 322e0e3a renamed file as per review comment
 * 0a386cca changes from the review - renamed "contents" to "data" - lint issue
 * d9ebbdc1 support for raw input as artifact
 * a1f821d5 Introduce communication channel from workflow-controller to executor through pod annotations
 * b324f9f5 Artifactory repository was not using correct casing for repoURL field
 * 3d45d25a Add `argo list --since` to filter workflows newer than a relative duration
 * cc2efdec Add ability to set loglevel of controller via CLI flag
 * 60c124e5 Remove hack.go and use dep to install code-generators
 * d14755a7 `argo list` was not handling the default case correctly
 * 472f5604 Improvements to `argo list` * sort workflows by running vs. completed, then by finished time * add --running, --completed, --status XXX filters * add -o wide option to show parameters and -o name to show only the names
 * b063f938 Use minimal ClusterRoles for workflow-controller and argo-ui deployments
 * 21bc2bd0 Added link to configuring artifact repo from main README
 * b54bc067 Added link to configuring artifact repo from main README
 * 58ec5169 Updated ARTIFACT_REPO.md
 * 1057d087 Added detailed instructions on configuring AWS and GCP artifact rpos
 * b0a7f0da Issue 680 - Argo UI is failing to render workflow which has not been picked up by workflow controller (#681)
 * e91c227a Document and clarify artifact passing (#676)
 * 290f6799 Allow containers to be retried. (#661)
 * 80f9b1b6 Improve the error message when insufficent RBAC privileges is detected (resolves #659)
 * 3cf67df4 Regenerate autogenerated code after changes to types
 * baf37052 Add support for resource template outputs. Remove output.parameters.path in favor of valueFrom
 * dc1256c2 Fix expected file name for issue template
 * a492ad14 Add a GitHub issues template
 * 55be93a6 Add a --dry-run option to `argo install`. Remove CRD creation from controller startup
 * fddc052d Fix README.md to contain influxdb-client in the example (#669)
 * 67236a59 Update getting started doc to use `brew install` and better instructions for RBAC clusters (resolves #654, #530)
 * 5ac19753 Support rendering retry steps (#670)
 * 3cca0984 OpenID Connect auth support (#663)
 * c222cb53 Clarify where the Minio secret comes from.
 * a78e2e8d Remove parallel steps that use volumes.
 * 35517385 Prevent a potential k8s scheduler panic from incomplete setting of pod ownership reference (resolves #656)
 * 1a8bc26d Updated README
 * 9721fca0 Updated README
 * e3177606 Fix typos in READMEs
 * 555d50b0 Simplify some getting started instructions. Correct some usages of container resources field
 * 4abc9c40 Updated READMEs
 * a0add24f Switch to k8s-codegen generated workflow client and informer
 * 9b08b6e9 Added link for argoproj slack channel
 * 682bbdc0 Update references to point to latest argo release

### Contributors

 * Alexander Matyushentsev
 * Ed Lee
 * Jesse Suen
 * Matt Hillsdon
 * Rhys Parry
 * Sandeep Bhojwani
 * Shri Javadekar
 * gaganapplatix

## v2.0.0-alpha3 (2018-01-02)

 * 940dd56d Fix artifactory unit test and linting issues
 * e7ba2b44 Update help page links (#651)
 * 53dac4c7 Add artifactory and UI fixes to 2.0.0-alpha3 CHANGELOG
 * 4b4eff43 Allow disabling web console feature (#649)
 * 90b7f2e6 Added support for artifactory
 * 849e916e Adjusted styles for logs stream (#614)
 * a8a96030 Update CHANGELOG for 2.0.0-alpha3
 * e7c7678c Fix issue preventing ability to pass JSON as a command line param (resolves #646)
 * 7f5e2b96 Add validation checks for volumeMount/artifact path collision and activeDeadlineSeconds (#620)
 * dc4a9463 Add the ability to specify the service account used by pods in the workflow (resolves #634) Also add argo CLI support for supplying/overriding spec.serviceAccountName from command line.
 * 16f7000a Workflow operator will recover from unexpected panics and mark the workflow with error (resolves #633)
 * 18dca7fe Issue #629 - Add namespace to workflow list and workflow details page (#639)
 * e656bace Issue #637 -  Implement Workflow list and workflow details page live update (#638)
 * 1503ce3a Issue #636 - Upgrade to ui-lib 2.0.3 to fix xterm incompatibility (#642)
 * f9170e8a Remove manifest-passing.yaml example now that we have resource templates
 * 25be5fd6 Implementation for resource templates and resource success/failure conditions
 * 402ad565 Updated examples/README
 * 8536c7fc added secret example to examples/README
 * e5002b82 Add '--wait' to argo submit.
 * 9646e55f Installer was not update configmap correctly with new executor image during upgrade
 * 69d72913 Support private git repos using secret selector fields in the git artifact (resolves #626)
 * 64e17244 Add argo ci workflow (#619)
 * e8998435 Resolve controller panic when a script template with an input artifact was submitted (resolves #617). Utilize the kubernetes.Interface and fake.Clientset to support unit test mocking. Added a unit test to reproduce the panic. Add an e2e test to verify functionality works.
 * 52075b45 Introduce controller instance IDs to support multiple workflow controllers in a cluster (resolves #508)
 * 133a23ce Add ability to timeout a container/script using activeDeadlineSeconds
 * b5b16e55 Support for workflow exit handlers
 * 906b3e7c Update ROADMAP.md
 * 5047422a Update CHANGELOG.md
 * 2b6583df Add `argo wait` for waiting on workflows to complete. (#618)
 * cfc9801c Add option to print output of submit in json.
 * c20c0f99 Comply with semantic versioning. Include build metadata in `argo version` (resolves #594)
 * bb5ac7db minor change
 * 91845d49 Added more documentation
 * 4e8d69f6 fixed install instructions
 * 0557147d Removed empty toolbar (#600)
 * bb2b29ff Added limit for number of steps in workflows list (#602)
 * 3f57cc1d fixed typo in examples/README
 * ebba6031 Updated examples/README.md with how to override entrypoint and parameters
 * 81834db3 Example with using an emptyDir volume.
 * 4cd949d3 Remove apiserver
 * 6a916ca4 `argo lint` did not split yaml files. `argo submit` was not ignoring non-workflow manifests
 * bf7d9979 Include `make lint` and `make test` as part of CI
 * d1639ecf Create example workflow using kubernetes secrets (resolves #592)
 * 31c54af4 Toolbar and filters on workflows list (#565)
 * bb4520a6 Add and improve the inlined comments in example YAMLs
 * a0470728 Fixed typo.
 * 13366e32 Fix some wrong GOPATH assumptions in Makefile. Add `make test` target. Fix unit tests
 * 9f4f1ee7 Add 'misspell' to linters. Fix misspellings caught by linter
 * 1b918aff Address all issues in code caught by golang linting tools (resolves #584)
 * 903326d9 Add manifest passing to do kubectl create with dynamic manifests (#588)
 * b1ec3a3f Create the argo-ui service with type ClusterIP as part of installation (resolves #582)
 * 5b6271bc Add validate names for various workflow specific fields and tests for them (#586)
 * b6e67131 Implementation for allowing access to global parameters in workflow (#571)
 * c5ac5bfb Fix error message when key does not exist in secret (resolves #574). Improve s3 example and documentation.
 * 4825c43b Increate UI build memory limit (#580)
 * 87a20c6b Update input-artifacts-s3.yaml example to explain concepts and usage better
 * c16a9c87 Rahuldhide patch 2 (#579)
 * f5d0e340 Issue #549 - Prepare argo v1 build config (#575)
 * 3b3a4c87 Argo logo
 * d1967443 Skip e2e tests if Kubeconfig is not provided.
 * 1ec231b6 Create separate namespaces for tests.
 * 5ea20d7e Add a deadline for workflow operation to prevent workqueue starvation and to enable state resync (#531) Tested with 6 x 1000 pod workflows.
 * 346bafe6 Multiple scalability improvements to controller (resolves #531)
 * bbc56b59 Improve argo ui build performance and reduce image size (#572)
 * cdb1ce82 Upgrade ui-lib (#556)
 * 0605ad7b Adjusted tabs content size to see horizontal and vertical scrolls. (#569)
 * a3316236 Fix rendering 'Error' node status (#564)
 * 8c3a7a93 Issue #548  - UI terminal window  (#563)
 * 5ec6cc85 Implement API to ssh into pod (#561)
 * beeb65dd Don't mess the controller's arguments.
 * 01f5db5a Parameterize Install() and related methods.
 * 85a2e271 Fix tests.
 * 56f666e1 Basic E2e tests.
 * 9eafb9dd Issue #547 - Support filtering by status in API GET /workflows (#550)
 * 37f41eb7 Update demo.md
 * ea8d5c11 Update README.md
 * 373f0710 Add support for making a no_ui build. Base all build on no_ui build (#553)
 * ae65c57e Update demo.md
 * f6f8334b V2 style adjustments and small fixes (#544)
 * 12d5b7ca Document argo ui service creation (#545)
 * 3202d4fa Support all namespaces (#543)
 * b553c1bd Update demo.md to qualify the minio endpoint with the default namespace
 * 4df7617c Fix artifacts downloading (#541)
 * 12732200 Update demo.md with references to latest release

### Contributors

 * Alexander Matyushentsev
 * Anshuman Bhartiya
 * Ed Lee
 * Javier Castellanos
 * Jesse Suen
 * Rafal
 * Rahul Dhide
 * Sandeep Bhojwani
 * Shri Javadekar
 * Wojciech Kalemba
 * gaganapplatix
 * mukulikak

## v2.0.0-alpha2 (2017-12-04)

 * 0e67b861 Add 'release' make target. Improve CLI help and set version from git tag. Uninstaller for UI
 * 8ab1d2e9 Install argo ui along with argo workflow controller (#540)
 * f4af881e Add make command to build argo ui (#539)
 * 5bb85814 Add example description in YAML.
 * fc23fcda edit example README
 * 8dd294aa Add example of GIF processing using ImageMagick
 * ef8e9d5c Implement loader (#537)
 * 2ac37361 Allow specifying CRD version (#536)
 * 15b5542d Installer was not using the argo serviceaccount with the workflow-controller deployment. Make progress messages consistent
 * f1471347 Add Yaml viewer (#535)
 * 685a576b Fix Gopkg.lock file following rewrite of git history at github.com/minio/go-homedir
 * 01ab3076 Delete clusterRoleBinding and serviceAccount.
 * 7bb99ae7 Rename references from v1 to v1alpha1 in YAML
 * 32343913 Implement step artifacts tab (#534)
 * b2a58dad Workflow list (#533)
 * 5dd1754b Guard controller from informer sending non workflow/pod objects (#505)
 * 59e31c60 Enable resync period in workflow/pod informers (resolves #532)
 * d5b06dcd Significantly increase efficiency of workflow control loop (resolves #505)
 * 4b2098ee finished walkthrough sections
 * eb7292b0 walkthrough
 * 82b1c7d9 Add -o wide option to `argo get` to display artifacts and durations (resolves #526)
 * 3427955d Use PATCH api from k8s go SDK for annotating/labeling pods
 * 4842bbbc Add support for nodeSelector at both the workflow and step level (resolves #458)
 * 424fba5d Rename apiVersion of workflows from v1 to v1alpha1 (resolves #517)
 * 5286728a Propogate executor errors back to controller. Add error column in `argo get` (#522)
 * 32b5e99b Simplify executor commands to just 'init' and 'wait'. Improve volumes examples
 * e2bfbc12 Update controller config automatically on configmap updates resolves #461
 * c09b13f2 Workflow validation detects when arguments were not supplied (#515)
 * 705193d0 Proper message for non-zero exits from main container. Indicate an Error phase/message when failing to load/save artifacts
 * e69b7510 Update page title and favicon (#519)
 * 4330232f Render workflow steps on workflow list page (#518)
 * 87c447ea Implement kube api proxy. Add workflow node logs tab (#511)
 * 0ab26883 Rework/rename Makefile targets. Bake in image namespace/tag set during build, as part of argo install
 * 3f13f5ca Support for overriding/supplying entrypoint and parameters via argo CLI. Update examples
 * 6f9f2adc Support ListOptions in the WorkflowClient. Add flag to delete completed workflows
 * 30d7fba1 Check Kubernetes version.
 * a3909273 Give proper error for unamed steps
 * eed54f57 Harden the IsURL check
 * bfa62afd Add phase,completed fields to workflow labels. Add startedAt,finishedAt,phase,message to workflow.status
 * 9347619c Create serviceAccount & roleBinding if necessary.
 * 205e5cbc Introduce 'completed' pod label and label selector so controller can ignore completed pods
 * 199dbcbf 476 jobs list page (#501)
 * 05879294 Implement workflow tree tab draft (#494)
 * a2f034a0 Proper error reporting. Add message, startedAt, finishedAt to NodeStatus. Rename status to phase
 * 645fedca Support loop step expansion from input parameters and previous step results
 * 75c1c482 Help page v2 (#492)
 * a4af6702 Basic state of  navigation, top-bar, tooltip for UI v2 (#491)
 * 726e9fa0 moved the service acct note
 * 3a4cd9c4 477 job details page (#488)
 * 8ba7b55c Edited the instructions
 * 1e9dbdba Added influxdb-ci example
 * bd5c0baa Added comment for entrypoint field
 * 2fbecdf0 Argo V2 UI initial commit (#474)
 * da4ef06e added artifact to ci example
 * 9ce20123 added artifacts
 * caaa32a6 Minor edit
 * ae72b583 added more argo/kubectl examples
 * 1db21612 merged
 * 8df393ed added 2.0
 * 9e3a51b1 Update demo.md to have better instructions to restart controller after configuration changes
 * ba9f9277 Add demo markdown file. Delete old demo.txt
 * d8de40bb added 2.0
 * 6c617599 added 2.0
 * 32af692e added 2.0
 * 802940be added 2.0
 * 1d443415 added new png

### Contributors

 * Alexander Matyushentsev
 * Ed Lee
 * Jesse Suen
 * Rafal
 * Sandeep Bhojwani
 * Shri Javadekar
 * Wojciech Kalemba
 * cyee88
 * mukulikak

## v2.0.0-alpha1 (2017-11-16)


### Contributors


## v2.0.0 (2018-02-06)

 * 0978b9c6 Support setting UI base Url  (#722)
 * b75cd98f updated argo-user slack link
 * b3598d84 Add examples as functional and expected failure e2e tests
 * 83966e60 Fix regression where executor did not annotate errors correctly
 * 751fd270 Update UI references to v2.0.0. Update changelog
 * 8b7e2e24 Check-in the OpenAPI spec. Automate generation as part of `make update-codegen`
 * 563bda75 Fix update-openapigen.sh script to presume bash. Tweak documentation
 * 5b9a602b Add documentation to types. Add program to generate OpenAPI spec
 * d929e79f Generate OpenAPI models for the workflow spec (issue #707)
 * 1d5afee6 Shortened url
 * 617d848d Added news to README
 * ae36b22b Fix typo s/Customer/Custom/ (#704)
 * 5a589fcd Add ability to specify imagePullSecrets in the workflow.spec (resolves #699)
 * 2f77bc1e Add ability to specify affinity rules at both the workflow and template level (resolves #701)
 * fdafbe27 Sidecars unable to reference volume claim templates (resolves #697)
 * 0b0b52c3 Referencing output artifacts from a container with retries was not functioning (resolves #698)
 * bf2b376a Update doc references to point to v2.0.0-beta1. Fix secrets example
 * 549870c1 Fix argo-ui download links to point to v2.0.0-beta1
 * a202049d Update CHANGELOG for v2.0.0-beta1
 * a3739035 Remove dind requirement from argo-ci test steps
 * 1bdd0c03 Include completed pods when attempting to reconcile deleted pods Switch back to Patch (from Update) for persisting workflow changes
 * a4a43892 Sleep 1s after persisting workflow to give informer cache a chance to sync (resolves #686)
 * 5bf49531 Updated demo.md with link to ARTIFACT_REPO.md
 * 863d547a Rely on controller generated timestamps for node.StartedAt instad of pod.CreationTimestamp
 * 672542d1 Re-apply workflow changes and reattempt update on resource conflicts. Make completed pod labeling asynchronous
 * 81bd6d3d Resource state retry (#690)
 * 44dba889 Tune controller to 20 QPS, 30 Burst, 8 wf workers, 8 pod workers
 * 178b9d37 Show running/completed pod counts in `argo list -o wide`
 * 0c565f5f Switch to Updating workflow resources instead of Patching (resolves #686)
 * a571f592 Ensure sidecars get killed unequivocally. Final argoexec stats were not getting printed
 * a0b2d78c Show duration by default in `argo get`. --since flag should always include Running
 * 10110313 Executor hardening: add retries and memoization for executor k8s API calls Recover from unexpected panics and annotate the error.
 * f2b8f248 Regenerate deepcopy code after type changes for raw input artifacts
 * 322e0e3a renamed file as per review comment
 * 0a386cca changes from the review - renamed "contents" to "data" - lint issue
 * d9ebbdc1 support for raw input as artifact
 * a1f821d5 Introduce communication channel from workflow-controller to executor through pod annotations
 * b324f9f5 Artifactory repository was not using correct casing for repoURL field
 * 3d45d25a Add `argo list --since` to filter workflows newer than a relative duration
 * cc2efdec Add ability to set loglevel of controller via CLI flag
 * 60c124e5 Remove hack.go and use dep to install code-generators
 * d14755a7 `argo list` was not handling the default case correctly
 * 472f5604 Improvements to `argo list` * sort workflows by running vs. completed, then by finished time * add --running, --completed, --status XXX filters * add -o wide option to show parameters and -o name to show only the names
 * b063f938 Use minimal ClusterRoles for workflow-controller and argo-ui deployments
 * 21bc2bd0 Added link to configuring artifact repo from main README
 * b54bc067 Added link to configuring artifact repo from main README
 * 58ec5169 Updated ARTIFACT_REPO.md
 * 1057d087 Added detailed instructions on configuring AWS and GCP artifact rpos
 * b0a7f0da Issue 680 - Argo UI is failing to render workflow which has not been picked up by workflow controller (#681)
 * e91c227a Document and clarify artifact passing (#676)
 * 290f6799 Allow containers to be retried. (#661)
 * 80f9b1b6 Improve the error message when insufficent RBAC privileges is detected (resolves #659)
 * 3cf67df4 Regenerate autogenerated code after changes to types
 * baf37052 Add support for resource template outputs. Remove output.parameters.path in favor of valueFrom
 * dc1256c2 Fix expected file name for issue template
 * a492ad14 Add a GitHub issues template
 * 55be93a6 Add a --dry-run option to `argo install`. Remove CRD creation from controller startup
 * fddc052d Fix README.md to contain influxdb-client in the example (#669)
 * 67236a59 Update getting started doc to use `brew install` and better instructions for RBAC clusters (resolves #654, #530)
 * 5ac19753 Support rendering retry steps (#670)
 * 3cca0984 OpenID Connect auth support (#663)
 * c222cb53 Clarify where the Minio secret comes from.
 * a78e2e8d Remove parallel steps that use volumes.
 * 35517385 Prevent a potential k8s scheduler panic from incomplete setting of pod ownership reference (resolves #656)
 * 1a8bc26d Updated README
 * 9721fca0 Updated README
 * e3177606 Fix typos in READMEs
 * 555d50b0 Simplify some getting started instructions. Correct some usages of container resources field
 * 4abc9c40 Updated READMEs
 * a0add24f Switch to k8s-codegen generated workflow client and informer
 * 9b08b6e9 Added link for argoproj slack channel
 * 682bbdc0 Update references to point to latest argo release
 * 940dd56d Fix artifactory unit test and linting issues
 * e7ba2b44 Update help page links (#651)
 * 53dac4c7 Add artifactory and UI fixes to 2.0.0-alpha3 CHANGELOG
 * 4b4eff43 Allow disabling web console feature (#649)
 * 90b7f2e6 Added support for artifactory
 * 849e916e Adjusted styles for logs stream (#614)
 * a8a96030 Update CHANGELOG for 2.0.0-alpha3
 * e7c7678c Fix issue preventing ability to pass JSON as a command line param (resolves #646)
 * 7f5e2b96 Add validation checks for volumeMount/artifact path collision and activeDeadlineSeconds (#620)
 * dc4a9463 Add the ability to specify the service account used by pods in the workflow (resolves #634) Also add argo CLI support for supplying/overriding spec.serviceAccountName from command line.
 * 16f7000a Workflow operator will recover from unexpected panics and mark the workflow with error (resolves #633)
 * 18dca7fe Issue #629 - Add namespace to workflow list and workflow details page (#639)
 * e656bace Issue #637 -  Implement Workflow list and workflow details page live update (#638)
 * 1503ce3a Issue #636 - Upgrade to ui-lib 2.0.3 to fix xterm incompatibility (#642)
 * f9170e8a Remove manifest-passing.yaml example now that we have resource templates
 * 25be5fd6 Implementation for resource templates and resource success/failure conditions
 * 402ad565 Updated examples/README
 * 8536c7fc added secret example to examples/README
 * e5002b82 Add '--wait' to argo submit.
 * 9646e55f Installer was not update configmap correctly with new executor image during upgrade
 * 69d72913 Support private git repos using secret selector fields in the git artifact (resolves #626)
 * 64e17244 Add argo ci workflow (#619)
 * e8998435 Resolve controller panic when a script template with an input artifact was submitted (resolves #617). Utilize the kubernetes.Interface and fake.Clientset to support unit test mocking. Added a unit test to reproduce the panic. Add an e2e test to verify functionality works.
 * 52075b45 Introduce controller instance IDs to support multiple workflow controllers in a cluster (resolves #508)
 * 133a23ce Add ability to timeout a container/script using activeDeadlineSeconds
 * b5b16e55 Support for workflow exit handlers
 * 906b3e7c Update ROADMAP.md
 * 5047422a Update CHANGELOG.md
 * 2b6583df Add `argo wait` for waiting on workflows to complete. (#618)
 * cfc9801c Add option to print output of submit in json.
 * c20c0f99 Comply with semantic versioning. Include build metadata in `argo version` (resolves #594)
 * bb5ac7db minor change
 * 91845d49 Added more documentation
 * 4e8d69f6 fixed install instructions
 * 0557147d Removed empty toolbar (#600)
 * bb2b29ff Added limit for number of steps in workflows list (#602)
 * 3f57cc1d fixed typo in examples/README
 * ebba6031 Updated examples/README.md with how to override entrypoint and parameters
 * 81834db3 Example with using an emptyDir volume.
 * 4cd949d3 Remove apiserver
 * 6a916ca4 `argo lint` did not split yaml files. `argo submit` was not ignoring non-workflow manifests
 * bf7d9979 Include `make lint` and `make test` as part of CI
 * d1639ecf Create example workflow using kubernetes secrets (resolves #592)
 * 31c54af4 Toolbar and filters on workflows list (#565)
 * bb4520a6 Add and improve the inlined comments in example YAMLs
 * a0470728 Fixed typo.
 * 13366e32 Fix some wrong GOPATH assumptions in Makefile. Add `make test` target. Fix unit tests
 * 9f4f1ee7 Add 'misspell' to linters. Fix misspellings caught by linter
 * 1b918aff Address all issues in code caught by golang linting tools (resolves #584)
 * 903326d9 Add manifest passing to do kubectl create with dynamic manifests (#588)
 * b1ec3a3f Create the argo-ui service with type ClusterIP as part of installation (resolves #582)
 * 5b6271bc Add validate names for various workflow specific fields and tests for them (#586)
 * b6e67131 Implementation for allowing access to global parameters in workflow (#571)
 * c5ac5bfb Fix error message when key does not exist in secret (resolves #574). Improve s3 example and documentation.
 * 4825c43b Increate UI build memory limit (#580)
 * 87a20c6b Update input-artifacts-s3.yaml example to explain concepts and usage better
 * c16a9c87 Rahuldhide patch 2 (#579)
 * f5d0e340 Issue #549 - Prepare argo v1 build config (#575)
 * 3b3a4c87 Argo logo
 * d1967443 Skip e2e tests if Kubeconfig is not provided.
 * 1ec231b6 Create separate namespaces for tests.
 * 5ea20d7e Add a deadline for workflow operation to prevent workqueue starvation and to enable state resync (#531) Tested with 6 x 1000 pod workflows.
 * 346bafe6 Multiple scalability improvements to controller (resolves #531)
 * bbc56b59 Improve argo ui build performance and reduce image size (#572)
 * cdb1ce82 Upgrade ui-lib (#556)
 * 0605ad7b Adjusted tabs content size to see horizontal and vertical scrolls. (#569)
 * a3316236 Fix rendering 'Error' node status (#564)
 * 8c3a7a93 Issue #548  - UI terminal window  (#563)
 * 5ec6cc85 Implement API to ssh into pod (#561)
 * beeb65dd Don't mess the controller's arguments.
 * 01f5db5a Parameterize Install() and related methods.
 * 85a2e271 Fix tests.
 * 56f666e1 Basic E2e tests.
 * 9eafb9dd Issue #547 - Support filtering by status in API GET /workflows (#550)
 * 37f41eb7 Update demo.md
 * ea8d5c11 Update README.md
 * 373f0710 Add support for making a no_ui build. Base all build on no_ui build (#553)
 * ae65c57e Update demo.md
 * f6f8334b V2 style adjustments and small fixes (#544)
 * 12d5b7ca Document argo ui service creation (#545)
 * 3202d4fa Support all namespaces (#543)
 * b553c1bd Update demo.md to qualify the minio endpoint with the default namespace
 * 4df7617c Fix artifacts downloading (#541)
 * 12732200 Update demo.md with references to latest release
 * 0e67b861 Add 'release' make target. Improve CLI help and set version from git tag. Uninstaller for UI
 * 8ab1d2e9 Install argo ui along with argo workflow controller (#540)
 * f4af881e Add make command to build argo ui (#539)
 * 5bb85814 Add example description in YAML.
 * fc23fcda edit example README
 * 8dd294aa Add example of GIF processing using ImageMagick
 * ef8e9d5c Implement loader (#537)
 * 2ac37361 Allow specifying CRD version (#536)
 * 15b5542d Installer was not using the argo serviceaccount with the workflow-controller deployment. Make progress messages consistent
 * f1471347 Add Yaml viewer (#535)
 * 685a576b Fix Gopkg.lock file following rewrite of git history at github.com/minio/go-homedir
 * 01ab3076 Delete clusterRoleBinding and serviceAccount.
 * 7bb99ae7 Rename references from v1 to v1alpha1 in YAML
 * 32343913 Implement step artifacts tab (#534)
 * b2a58dad Workflow list (#533)
 * 5dd1754b Guard controller from informer sending non workflow/pod objects (#505)
 * 59e31c60 Enable resync period in workflow/pod informers (resolves #532)
 * d5b06dcd Significantly increase efficiency of workflow control loop (resolves #505)
 * 4b2098ee finished walkthrough sections
 * eb7292b0 walkthrough
 * 82b1c7d9 Add -o wide option to `argo get` to display artifacts and durations (resolves #526)
 * 3427955d Use PATCH api from k8s go SDK for annotating/labeling pods
 * 4842bbbc Add support for nodeSelector at both the workflow and step level (resolves #458)
 * 424fba5d Rename apiVersion of workflows from v1 to v1alpha1 (resolves #517)
 * 5286728a Propogate executor errors back to controller. Add error column in `argo get` (#522)
 * 32b5e99b Simplify executor commands to just 'init' and 'wait'. Improve volumes examples
 * e2bfbc12 Update controller config automatically on configmap updates resolves #461
 * c09b13f2 Workflow validation detects when arguments were not supplied (#515)
 * 705193d0 Proper message for non-zero exits from main container. Indicate an Error phase/message when failing to load/save artifacts
 * e69b7510 Update page title and favicon (#519)
 * 4330232f Render workflow steps on workflow list page (#518)
 * 87c447ea Implement kube api proxy. Add workflow node logs tab (#511)
 * 0ab26883 Rework/rename Makefile targets. Bake in image namespace/tag set during build, as part of argo install
 * 3f13f5ca Support for overriding/supplying entrypoint and parameters via argo CLI. Update examples
 * 6f9f2adc Support ListOptions in the WorkflowClient. Add flag to delete completed workflows
 * 30d7fba1 Check Kubernetes version.
 * a3909273 Give proper error for unamed steps
 * eed54f57 Harden the IsURL check
 * bfa62afd Add phase,completed fields to workflow labels. Add startedAt,finishedAt,phase,message to workflow.status
 * 9347619c Create serviceAccount & roleBinding if necessary.
 * 205e5cbc Introduce 'completed' pod label and label selector so controller can ignore completed pods
 * 199dbcbf 476 jobs list page (#501)
 * 05879294 Implement workflow tree tab draft (#494)
 * a2f034a0 Proper error reporting. Add message, startedAt, finishedAt to NodeStatus. Rename status to phase
 * 645fedca Support loop step expansion from input parameters and previous step results
 * 75c1c482 Help page v2 (#492)
 * a4af6702 Basic state of  navigation, top-bar, tooltip for UI v2 (#491)
 * 726e9fa0 moved the service acct note
 * 3a4cd9c4 477 job details page (#488)
 * 8ba7b55c Edited the instructions
 * 1e9dbdba Added influxdb-ci example
 * bd5c0baa Added comment for entrypoint field
 * 2fbecdf0 Argo V2 UI initial commit (#474)
 * da4ef06e added artifact to ci example
 * 9ce20123 added artifacts
 * caaa32a6 Minor edit
 * ae72b583 added more argo/kubectl examples
 * 1db21612 merged
 * 8df393ed added 2.0
 * 9e3a51b1 Update demo.md to have better instructions to restart controller after configuration changes
 * ba9f9277 Add demo markdown file. Delete old demo.txt
 * d8de40bb added 2.0
 * 6c617599 added 2.0
 * 32af692e added 2.0
 * 802940be added 2.0
 * 1d443415 added new png
 * 1069af4f Support submission of manifests via URL
 * cc1f0caf Add recursive coinflip example
 * 90f37ad6 Support configuration of the controller to match specified labels
 * f9c9673a Filter non-workflow related pods in the controller's pod watch
 * 9555a472 Add install notes to support cluster with legacy authentication disabled. Add option to specify service account
 * 837e0a2b Propogate deletion of controller replicaset/pod during uninstall
 * 5a7fcec0 Add parameter passing example yaml
 * 2a34709d Passthrough --namespace flag to `kubectl logs`
 * 3fc6af00 Adding passing parameter example yaml
 * e275bd5a Add support for output as parameter
 * 5ee1819c Restore and formalize sidecar kill functionality as part of executor
 * dec97891 Proper workflow manifest validation during `argo lint` and `argo submit`
 * 6ab0b610 Uninstall support via `argo uninstall`
 * 3ba84082 Adding sidecar container
 * dba29bd9 Support GCP
 * f3049105 Proper controller support for running workflows in arbitrary namespaces. Install controller into kube-system namespace by default
 * ffb3d128 Add support for controller installation via `argo install` and demo instructions
 * dcfb2752 Add `argo delete` command to delete workflows
 * 8e583afb Add `argo logs` command as a convenience wrapper around `kubectl logs`
 * 368193d5 Add argo `submit`, `list`, `get`, `lint` commands
 * 8ef7a131 Executor to load script source code as an artifact to main. Remove controller hack
 * 736c5ec6 Annotate pod with outputs. Properly handle tar/gz of artifacts
 * cd415c9d Introduce Template.ArchiveLocation to store all related step level artifacts to a job, understood by executor
 * 4241cace Support for saving s3 output artifacts
 * cd3a3f1e Bind mount docker.sock to wait container to use `docker wait` and `docker cp`
 * 77d64a66 Support the case where an input artifact path overlaps with a container volume mount
 * 6a54b31f Support for automatic termination for daemoned workflow steps
 * 2435e6f7 Track children node IDs in workflow status nodes
 * 227c1961 Initial support for daemon workflow steps (no termination yet)
 * 738b02d2 Support for git/http input artifacts. hack together wait container logic as a shell script
 * de71cb5b Change according to jesse's comments
 * 621d7ca9 Argo Executor init container
 * efe43927 Switch representation of parallel steps as a list instead of a map. update examples
 * 56ca947b Introduce ability to add sidecars to run adjacent to workflow steps
 * b4d77701 Controller support for overlapping artifact path to user specified volume mountPaths
 * 3782bade Get coinflip example to function
 * 065a8f77 Get python script example to function
 * 8973204a Move to list style of inputs and arguments (vs. maps). Reuse artifact datastructure
 * d9838749 Improve example yamls
 * f83b2620 Support for volumeClaimTemplates (ephemeral volumes) in workflows
 * be3ad92e Support for using volumes within multiple steps in a workflow
 * 4b4dc4a3 Copy outputs from pod metadata to workflow node outputs
 * 07f2c965 Initial support for conditionals as 'when' field in workflow step
 * fe639edd Controller support for "script" templates (workflow step as code)
 * a896f03e Add example yamls for proposal for scripting steps
 * c782e2e1 Support looping with item maps
 * 7dc58fce Initial withItems loop support
 * f3010c1d Support for argument passing and substitution in templates
 * 5e8ba870 Split individual workflow operation logic from controller
 * 63a2c20c Introduce sirupsen/logrus logging package
 * 2058342f Annotate the template used by executor to include destination artifact information
 * 52f8db21 Sync workflow controller configuration from a configmap. Add config validation
 * d0a1748a Upgrade to golang 1.9.1. Get `make lint` target to function
 * ac58d832 Speed up rebuilds from within build container by bind mounting $GOPATH/pkg:/root/go/pkg
 * 71445675 Add installation manifests. Initial stubs for controller configuration
 * 10372091 Introduce s3, git, http artifact sources in inputs.artifacts
 * a68001d3 Add debug tools to argoexec image. Remove privileged mode from sidekick. Disable linting
 * dc530232 Create shared 'input-artifacts' volume and mount between init/main container
 * 6ba84eb5 Expose various pod metadata to argoexec via K8s downward API
 * 1fc079de Add `argo yaml validate` command and `argoexec artifacts load` stub
 * 9125058d Include scheduling of argoexec (init and sidekick) containers to the user's main
 * 67f8353a Initial workflow operator logic
 * 8137021a Reorganize all CLIs into a separate dir. Add stubs for executor and apiserver
 * 74baac71 Introduce Argo errors package
 * 37b7de80 Add apiserver skeleton
 * 3ed1dfeb Initial project structure. CLI and Workflow CRD skeleton

### Contributors

 * Alexander Matyushentsev
 * Anshuman Bhartiya
 * David Kale
 * Ed Lee
 * Edward Lee
 * Javier Castellanos
 * Jesse Suen
 * Matt Hillsdon
 * Rafal
 * Rahul Dhide
 * Rhys Parry
 * Sandeep Bhojwani
 * Shri Javadekar
 * Tianhe Zhang
 * Wojciech Kalemba
 * cyee88
 * gaganapplatix
 * mukulikak

## v0.4.7 (2018-06-07)

 * e4d0bd39 Take into account number of unavailable replicas to decided if deployment is healthy or not (#270)
 * 18dc82d1 Remove hard requirement of initializing OIDC app during server startup (resolves #272)
 * e720abb5 Bump version to v0.4.7
 * a2e9a9ee Repo names containing underscores were not being accepted (resolves #258)

### Contributors

 * Alexander Matyushentsev
 * Jesse Suen

## v0.4.6 (2018-06-06)

 * cf377690 Retry `argocd app wait` connection errors from EOF watch. Show detailed state changes

### Contributors

 * Jesse Suen

## v0.4.5 (2018-05-31)

 * 3acca509 Add `argocd app unset` command to unset parameter overrides. Bump version to v0.4.5
 * 5a622861 Cookie token was not parsed properly when mixed with other site cookies

### Contributors

 * Jesse Suen

## v0.4.4 (2018-05-30)

 * 5452aff0 Add ability to show parameters and overrides in CLI (resolves #240) (#247)
 * 0f4f1262 Add Events API endpoint (#237)
 * 4e7f68cc Update version to 0.4.4
 * 96c05bab Issue #238 - add upsert flag to 'argocd app create' command (#245)
 * 6b78cddb Add repo browsing endpoint (#229)
 * 12596ff9 Issue #233 - Controller does not persist rollback operation result (#234)
 * a240f1b2 Bump version to 0.5.0
 * f6da1967 Support subscribing to settings updates and auto-restart of dex and API server (resolves #174) (#227)
 * e81d30be Update getting_started.md to point to v0.4.3
 * 13b090e3 Issue #147 - App sync frequently fails due to concurrent app modification (#226)
 * d0479e6d Issue # 223 - Remove app finalizers during e2e fixture teardown (#225)
 * 14328270 Add error fields to cluster/repo, shell output (#200)

### Contributors

 * Alexander Matyushentsev
 * Andrew Merenbach
 * Jesse Suen

## v0.4.3 (2018-05-21)

 * 89bf4eac Bump version to 0.4.3
 * 07aac0bd Move local branch deletion as part of git Reset() (resolves #185) (#222)
 * 61220b8d Fix exit code for app wait (#219)

### Contributors

 * Andrew Merenbach
 * Jesse Suen

## v0.4.2 (2018-05-21)

 * 4e470aaf Remove context name prompt during login. (#218)
 * 76922b62 Update version to 0.4.2

### Contributors

 * Jesse Suen

## v0.4.1 (2018-05-18)

 * ac0f623e Add `argocd app wait` command (#216)
 * afd54508 Bump version to v0.4.1
 * c17266fc Add documentation on how to configure SSO and Webhooks
 * f62c8254 Manifest endpoint (#207)
 * 45f44dd4 Add v0.4.0 changelog
 * 9c0daebf Fix diff falsely reporting OutOfSync due to namespace/annotation defaulting
 * f2a0ca56 Add intelligence in diff libray to perform three-way diff from last-applied-configuration annotation (resolves #199)
 * e04d3158 Issue #118 - app delete should be done through controller using finalizers (#206)
 * daec6976 Update ksonnet to v0.10.2 (resolves #208)
 * 7ad56707 Make sure api server started during fixture setup (#209)
 * 80364233 Implement App management and repo management e2e tests (#205)
 * 8039228a Add last update time to operation status, fix operation status patching (#204)
 * b1103af4 Rename recent deployments to history (#201)
 * d67ad5ac Add connect timeouts when interacting with SSH git repos (resolves #131) (#203)
 * c9df9c17 Default Spec.Source.TargetRevision to HEAD server-side if unspecified (issue #190)
 * 8fa46b02 Remove SyncPolicy (issue #190)
 * 92c48133 App creation was not defaulting to server and namespace defined in app.yaml
 * 2664db3e Refactor application controller sync/apply loop (#202)
 * 6b554e5f Add 0.3.0 to 0.4.0 migration utility (#186)
 * 2bc0dff1 Issue #146 - Render health status information in 'app list' and 'app get' commands (#198)
 * c61795f7 Add 'database' library for CRUD operations against repos and clusters. Redact sensitive information (#196)
 * a8a7491b Handle potential panic when `argo install settings` run against an empty configmap

### Contributors

 * Alexander Matyushentsev
 * Andrew Merenbach
 * Jesse Suen

## v0.4.0-alpha1 (2018-05-11)


### Contributors


## v0.4.0 (2018-05-17)

 * 9c0daebf Fix diff falsely reporting OutOfSync due to namespace/annotation defaulting
 * f2a0ca56 Add intelligence in diff libray to perform three-way diff from last-applied-configuration annotation (resolves #199)
 * e04d3158 Issue #118 - app delete should be done through controller using finalizers (#206)
 * daec6976 Update ksonnet to v0.10.2 (resolves #208)
 * 7ad56707 Make sure api server started during fixture setup (#209)
 * 80364233 Implement App management and repo management e2e tests (#205)
 * 8039228a Add last update time to operation status, fix operation status patching (#204)
 * b1103af4 Rename recent deployments to history (#201)
 * d67ad5ac Add connect timeouts when interacting with SSH git repos (resolves #131) (#203)
 * c9df9c17 Default Spec.Source.TargetRevision to HEAD server-side if unspecified (issue #190)
 * 8fa46b02 Remove SyncPolicy (issue #190)
 * 92c48133 App creation was not defaulting to server and namespace defined in app.yaml
 * 2664db3e Refactor application controller sync/apply loop (#202)
 * 6b554e5f Add 0.3.0 to 0.4.0 migration utility (#186)
 * 2bc0dff1 Issue #146 - Render health status information in 'app list' and 'app get' commands (#198)
 * c61795f7 Add 'database' library for CRUD operations against repos and clusters. Redact sensitive information (#196)
 * a8a7491b Handle potential panic when `argo install settings` run against an empty configmap
 * d1c7c4fc Issue #187 - implement `argo settings install` command (#193)
 * 3dbbcf89 Move sync logic to contoller (#180)
 * 0cfd1ad0 Update feature list with SSO and Webhook integration
 * bfa4e233 cli will look to spec.destination.server and namespace when displaying apps
 * dc662da3 Support OAuth2 login flow from CLI (resolves #172) (#181)
 * 4107d242 Fix linting errors
 * b83eac5d Make ApplicationSpec.Destination non-optional, non-pointer (#177)
 * bb51837c Do not delete namespace or CRD during uninstall unless explicitly stated (resolves #167) (#173)
 * 5bbb4fe1 Cache kubernetes API resource discovery (resolves #170) (#176)
 * b5c20e9b Trim spaces server-side in GitHub usernames (#171)
 * 1e1ab636 Don't fail when new app has same spec as old (#168)
 * 73485538 Improve CI build stability
 * 5f65a512 Introduce caching layer to repo server to improve query response times (#165)
 * d9c12e72 Issue #146 - ArgoCD applications should have a rolled up health status (#164)
 * fb2d6b4a Refactor repo server and git client (#163)
 * 3f4ec0ab Expand Git repo URL normalization (#162)
 * ac938fe8 Add GitHub webhook handling to fast-track controller application reprocessing (#160)
 * dc1e8796 Disable authentication for settings service
 * 8c5d59c6 Issue #157 - If argocd token is expired server should return 401 instead of 500 (#158)

### Contributors

 * Alexander Matyushentsev
 * Andrew Merenbach
 * Jesse Suen

## v0.3.3 (2018-05-03)

 * 13558b7c Revert change to redact credentials since logic is reused by controller
 * 3b2b3dac Update version
 * 1b2f8999 Issue #155 - Application update failes due to concurrent access (#156)
 * 0479fcdf Add settings endpoint so frontend can show/hide SSO login button. Rename config to settings (#153)
 * a0446546 Add workflow for blue-green deployments (#148)
 * 670921df SSO Support (#152)
 * 18f7e17d Added OWNERS file
 * a2aede04 Redact sensitive repo/cluster information upon retrieval (#150)

### Contributors

 * Alexander Matyushentsev
 * Andrew Merenbach
 * Edward Lee
 * Jesse Suen

## v0.3.2 (2018-05-01)

 * 1d876c77 Fix compilation error
 * 70465a05 Issue #147 - Use patch to update recentDeployments field (#149)
 * 3c984571  Issue #139 - Application sync should delete 'unexpected' resources (#144)
 * a36cc894 Issue #136 - Use custom formatter to get desired state of deployment and service (#145)
 * 9567b539 Improve comparator to fall back to looking up a resource by name
 * fdf9515d Refactor git library: * store credentials in files (instead of encoded in URL) to prevent leakage during git errors * fix issue where HEAD would not track updates from origin/HEAD (resolves #133) * refactor git library to promote code reuse, and remove shell invocations
 * b3202384 ksonnet util was not locating a ksonnet app dir correctly
 * 7872a604 Update ksonnet to v0.10.1
 * 5fea3846 Adding clusters should always go through argocd-manager service account creation
 * 86a4e0ba RoleBindings do not need to specify service account namespace in subject
 * 917f1df2 Populated 'unexpected' resources while comparing target and live states (#137)
 * 11260f24 Don't ask for user credentials if username and password are specified as arguments (#129)
 * 38d20d0f Add `argocd ctx` command for switching between contexts. Better CLI descriptions (resolves #103)
 * 938f40e8 Prompting for repo credentials was accepting newline as username
 * 5f9c8b86 Error properly when server address is unspecified (resolves #128)
 * d96d67bb Generate a temporary kubeconfig instead of using kubectl flags when applying resources
 * 19c3b876 Bump version to 0.4.0. `argocd app sync --dry-run` was incorrectly appending items to history (resolves #127)

### Contributors

 * Alexander Matyushentsev
 * Jesse Suen

## v0.3.1 (2018-04-24)

 * 7d08ab4e Bump version to v0.3.1
 * efea09d2 Fix linting issue in `app rollback`
 * 2adaef54 Introduce `argocd app history` and `argocd app rollback` CLI commands (resolves #125)
 * d71bbf0d Allow overriding server or namespace separately (#126)
 * 36b3b2b8 Switch to gogo/protobuf for golang code generation in order to use gogo extensions
 * 63dafa08 Issue #110 - Rollback ignores parameter overrides (#117)
 * afddbbe8 Issue #123 - Create .argocd directory before saving config file (#124)
 * 34811caf Update download instructions to getting started

### Contributors

 * Alexander Matyushentsev
 * Jesse Suen

## v0.3.0 (2018-04-23)

 * 8a285116 Enable auth by default. Decrease app resync period from 10m to 3m
 * 1a85a2d8 Bump version file to 0.3.0. Add release target and cli-linux/darwin targets
 * cf2d00e1 Add ability to set a parameter override from the CLI (`argo app set -p`)
 * 266c948a Add documentation about ArgoCD tracking strategies
 * dd564ee9 Introduce `app set` command for updating an app (resolves #116)
 * b9d48cab Add ability to set the tracking revision during app creation
 * 276e0674 Deployment of resources is performed using `kubectl apply` (resolves #106)
 * f3c4a693 Add watch verb to controller role
 * 1c60a698 Rename `argocd app add/rm` to `argocd app create/delete` (resolves #114)
 * 050f937a Update ksonnet to v0.10.0-alpha.3
 * b24e4782 Add application validation
 * e34380ed Expose port 443 to proxy to port 8080 (#113)
 * 338a1b82 `argo login` was not able to properly update boolean connection flags (insecure/plaintext)
 * b87c63c8 Re-add workaround for ksonnet bug
 * f6ed150b Issue #108 - App controller incorrectly report that app is out of sync (#109)
 * d5c683bc Add syncPolicy field to application CRD (#107)
 * 3ac95f3f Fix null pointer error in controller (#105)
 * 3be872ad Rework local config to support multiple servers/credentials
 * 80964a79 Set session cookies, errors appropriately (#100)
 * e719035e Allow ignoring recource deletion related errors while deleting application (#98)
 * f2bcf63b Fix linting breakage in session mock from recent changes to session interface
 * 2c9843f1 Update ksonnet to v0.10.0-alpha.2
 * 0560406d Add server auth cookies (#96)
 * db8083c6 Lowercase repo names before using in secret (#94)
 * fcc9f50b Fix issue preventing uppercased repo and cluster URLs (resolves #81)
 * c1ffbad8 Support manual token use for CLI commands (#90)
 * d7cdb1a5 Convert Kubernetes errors to gRPC errors (#89)
 * 6c41ce5e Add session gateway (#84)
 * 685a814f Add `argocd login` command (#82)
 * 06b64047 Issue #69 - Auto-sync option in application CRD instance (#83)
 * 8a90b324 Show more relevant information in `argocd cluster add`
 * 7e47b1eb TLS support. HTTP/HTTPS/gRPC all serving on single port
 * 150b51a3 Fix linter warning
 * 0002f8db Issue #75 - Implement delete pod API
 * 59ed50d2 Issue #74 - Implement stream logs API
 * 820b4bac Remove obsolete pods api
 * 19c5ecdb Check app label on client side before deleting app resource
 * 66b0702c Issue #65 - Delete all the kube object once app is removed
 * 5b5dc0ef Issue #67 - Application controller should persist ksonnet app parameters in app CRD (#73)
 * 0febf051 Issue #67 - Persist resources tree in application CRD (#68)
 * ee924bda Update ksonnet binary in image to ks tip. Begin using ksonnet as library instead of parsing stdout
 * ecfe571e update ksonnet dependency to tip. override some of ksonnet's dependencies
 * 173ecd93 Installer and settings management refactoring:
 * ba3db35b Add authentication endpoints (#61)
 * 074053da Update go-grpc-middleware version (#62)
 * 6bc98f91 Add JWT support (#60)

### Contributors

 * Alexander Matyushentsev
 * Andrew Merenbach
 * Jesse Suen

## v0.2.0 (2018-03-28)

 * 59dbe8d7 Maintain list of recent deployments in app CRD (#59)
 * 6d793617 Issue #57 - Add configmaps into argocd server role (#58)
 * e1c7f9d6 Fix deleting resources which do not support 'deletecollection' method but support 'delete' (#56)
 * 5febea22 Argo server should not fail if configmap name is not provided or config map does not exist (#55)
 * d093c8c3 Add password hashing (#51)
 * 10a8d521 Add application source and component parameters into recentDeployment field of application CRD (#53)
 * 234ace17 Replace ephemeral environments with override parameters (#52)
 * 817b13cc Add license and copyright. #49
 * b1682cc4 Add install configmap override flag (#47)
 * 74797a2a Delete child dependents while deleting app resources (#48)
 * ca570c7a Use ksonnet release version and fix app copy command (#46)
 * 92b7c6b5 Disable strict host key checking while cloning repo in repo-server (#45)
 * 4884c20d Issue #43 - Don't setup RBAC resources for clusters with basic authentication (#44)
 * 363b9b35 Don't overwrite application status in tryRefreshAppStatus (#42)
 * 5c062bd3 Support deploying/destroying ephemeral environments (#40)
 * 98754c7f Persist parameters during deployment (Sync) (#39)
 * 3927cc07 Add new dependency to CONTRIBUTING.md (#38)
 * 611b0e48 Issue #34 - Support ssh git URLs and ssh key authentication (#37)
 * 0368c2ea Allow use of public repos without prior registration (#36)
 * e7e3c509 Support -f/--file flag in `argocd app add` (#35)
 * d256256d Update CONTRIBUTING.md (#32)

### Contributors

 * Alexander Matyushentsev
 * Andrew Merenbach
 * Edward Lee

