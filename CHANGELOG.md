# Changelog

## v3.6.2 (2024-12-02)

Full Changelog: [v3.6.0...v3.6.2](https://github.com/argoproj/argo-workflows/compare/v3.6.0...v3.6.2)

### Selected Changes

* [741ab0ef7](https://github.com/argoproj/argo-workflows/commit/741ab0ef7b6432925e49882cb4294adccf5912ec) Merge commit from fork
* [6d87a90c0](https://github.com/argoproj/argo-workflows/commit/6d87a90c0fed24614e5e97135beee0a387f8432c) fix(ui): handle parsing errors properly in object editor (#13931)
* [ebed7f998](https://github.com/argoproj/argo-workflows/commit/ebed7f9983ad22fa06275ad64cb5588812dd0d36) refactor(deps): remove `moment` dep and usage (#12611)
* [8a94f2ef0](https://github.com/argoproj/argo-workflows/commit/8a94f2ef0cd5efa4635bccd6ccfd6cebeea5be2c) fix: Set default value to output parameters if suspend node timeout. Fixes #12230 (#12960)
* [1a3a5c233](https://github.com/argoproj/argo-workflows/commit/1a3a5c2335f66c487fe47d0797ae501b8f445ee0) fix: bump minio-go to version that supports eks pod identity #13800 (#13854)
* [e721cfef2](https://github.com/argoproj/argo-workflows/commit/e721cfef2ea1b8b9fd43c5955c9183825fe98b80) fix: consistently set executor log options  (#12979)
* [6371f9bfa](https://github.com/argoproj/argo-workflows/commit/6371f9bfade2ce3da4ea2a27a23855bd3435b387) chore(deps): bump github.com/golang-jwt/jwt/v4 from 4.5.0 to 4.5.1 in the go_modules group (#13865)
* [591928b8c](https://github.com/argoproj/argo-workflows/commit/591928b8c836e0c323c67ccb1bd505df1508c14c) fix(ui): improve editor performance and fix Submit button. Fixes #13892 (#13915)
* [8dd747317](https://github.com/argoproj/argo-workflows/commit/8dd7473170d87d8e24d9954df635615a24f742ad) fix(ui): Clickable URLs are messing up formatting in the UI (#13923)
* [f85d05595](https://github.com/argoproj/argo-workflows/commit/f85d05595d6247de4887a90b99bddb27b50a342c) fix(ui): fix broken workflowtemplate submit button. Fixes #13892 (#13913)

<details><summary><h3>Contributors</h3></summary>

* Adrien Delannoy
* Alan Clucas
* Anton Gilgur
* Blair Drummond
* Carlos R.F.
* Mason Malone
* dependabot[bot]
* instauro
* jswxstw

</details>

## v3.6.0 (2024-10-31)

Full Changelog: [v3.6.0-rc4...v3.6.0](https://github.com/argoproj/argo-workflows/compare/v3.6.0-rc4...v3.6.0)

### Selected Changes

<details><summary><h3>Contributors</h3></summary>

</details>

## v3.6.0-rc4 (2024-10-31)

Full Changelog: [v3.6.0-rc3...v3.6.0-rc4](https://github.com/argoproj/argo-workflows/compare/v3.6.0-rc3...v3.6.0-rc4)

### Selected Changes

* [b26ed4aa4](https://github.com/argoproj/argo-workflows/commit/b26ed4aa4dee395844531efa4a76a022183bec22) fix: use templateName where  possible else nothing (#13836)
* [3bbec4ec6](https://github.com/argoproj/argo-workflows/commit/3bbec4ec6a7d78ef924a9a60256a411a35634bc0) refactor(deps)!: remove `swagger-ui-react` (#13818)
* [3df51839e](https://github.com/argoproj/argo-workflows/commit/3df51839ef22acacf6712ae0c3775f2ea8a47012) refactor(ui): replace deprecated usage of `defaultProps` (#13822)
* [39154fd42](https://github.com/argoproj/argo-workflows/commit/39154fd42f814d3d5745937cbfbe13ff2b6003c1) fix: don't print help for non-validation errors. Fixes #13826 (#13831)
* [030afbcc8](https://github.com/argoproj/argo-workflows/commit/030afbcc85a70db9b669dcae96e17765160ee7b5) fix(ui): disable new graph filter options by default (#13835)
* [eb23eb6b8](https://github.com/argoproj/argo-workflows/commit/eb23eb6b84a1ecf0826a4e16d6a2032fd0cb08b7) fix: mark taskresult complete when failed or error. Fixes #12993, Fixes #13533 (#13798)
* [9f158ae0d](https://github.com/argoproj/argo-workflows/commit/9f158ae0d1ad93d1e8bede31086c742cfddbfc1e) refactor(ui): flatten `ui/src/app` dir  (#13815)
* [3dfea6d5a](https://github.com/argoproj/argo-workflows/commit/3dfea6d5a9572c312b4479bd321c05fdad3d21a6) fix: don't mount SA token when `automountServiceAccountToken: false`. Fixes #12848 (#13820)
* [1017c1dbe](https://github.com/argoproj/argo-workflows/commit/1017c1dbe2ccf3f0a59ae68f04d954dd92e69673) fix: correct nil pointer when listing wf archive without list options. Fixes #13804 (#13807)
* [49ff7a44b](https://github.com/argoproj/argo-workflows/commit/49ff7a44ba307416282a1f5cd3b844d19bce7f88) fix: better error message for multiple workflow controllers running (#13760)
* [b49e88e31](https://github.com/argoproj/argo-workflows/commit/b49e88e3178fc8183e5de7459ef6ae46985b0497) refactor: remove `util/slice` and use standard `slices` library (#13775)

<details><summary><h3>Contributors</h3></summary>

* Adrien Delannoy
* Alan Clucas
* Anton Gilgur
* Darko Janjic
* Greg Sheremeta
* Isitha Subasinghe
* Mason Malone
* MinyiZ
* dependabot[bot]
* github-actions[bot]

</details>

## v3.6.0-rc3 (2024-10-24)

Full Changelog: [v3.6.0-rc2...v3.6.0-rc3](https://github.com/argoproj/argo-workflows/compare/v3.6.0-rc2...v3.6.0-rc3)

### Selected Changes

* [92bf975ac](https://github.com/argoproj/argo-workflows/commit/92bf975ac198f8d17d4061c11fe5b08262aa59d6) fix!: migrate `argo_archived_workflows.workflow` to `jsonb` (#13779)
* [bb5130e84](https://github.com/argoproj/argo-workflows/commit/bb5130e841ab02cfe5b65f19c214fe309ca88bce) feat(artifacts): add git https insecure option. Fixes #10762 (#13797)
* [e11e664d9](https://github.com/argoproj/argo-workflows/commit/e11e664d92677b8addffae90b3238f867b091024) fix: optimise pod finalizers with merge patch and resourceVersion (#13776)
* [7d0d34ef9](https://github.com/argoproj/argo-workflows/commit/7d0d34ef92b76ea245e874cdde0da74dd3cd9e98) fix(cron): correctly run when `startingDeadlineSeconds` and `timezone` are set (#13795)
* [5e02f6eec](https://github.com/argoproj/argo-workflows/commit/5e02f6eece56db4b9c7da6bd227fc6258b11c88f) feat(ui): Group nodes based on `templateRef` and show invoking template name. Fixes #11106 (#13511)
* [6fa5365c9](https://github.com/argoproj/argo-workflows/commit/6fa5365c9bb7512a5e6c95ac8e29a7ec0402d433) feat(ui): Make URLs clickable in workflow node info (#13494)
* [ea6dae9a3](https://github.com/argoproj/argo-workflows/commit/ea6dae9a30893632d863ff664803e17f3e9965f6) fix(ui): clarify log deletion in log-viewer. Fixes #10993 (#13788)
* [e09d2f112](https://github.com/argoproj/argo-workflows/commit/e09d2f1121c142bc47cc12f543b1790a92baf8d1) fix: remove JSON cast when querying archived workflows (#13777)
* [c9b1477fd](https://github.com/argoproj/argo-workflows/commit/c9b1477fd575bf06bed43ca2139f74aa3af4285c) refactor: Only set `ARGO_TEMPLATE` env for init container. (#13761)
* [a88cddc0a](https://github.com/argoproj/argo-workflows/commit/a88cddc0aa4242783a36a8dc74c7269a103dd789) fix!: consolidate cronworkflow variables (#13702)
* [d8f2d858a](https://github.com/argoproj/argo-workflows/commit/d8f2d858ac198b924edbaab1775ca60df9863eed) feat: Add name filters for workflow list (exact/prefix/pattern) (#13160)
* [d05cf6459](https://github.com/argoproj/argo-workflows/commit/d05cf6459cdd956cb621a6552f009e21db0a3789) feat: add config to skip sending workflow events. Fixes #13042 (#13746)
* [a8c360945](https://github.com/argoproj/argo-workflows/commit/a8c3609456738adb55de4056aca9d8c0c1d72a0d) fix: only set `ARGO_PROGRESS_FILE` when needed. Partial fix for #13089 (#13743)
* [98dd46c6e](https://github.com/argoproj/argo-workflows/commit/98dd46c6e1c64c75b30269972abec4e463813bc1) fix(controller): Add configmap already exists judgment on offload Env… (#13756)
* [bf7760a46](https://github.com/argoproj/argo-workflows/commit/bf7760a46e29bf2571bd3e9d852220b44339bb60) feat: `ARGO_TEMPLATE` env var without `inputs.parameters` (#13742)
* [948127c79](https://github.com/argoproj/argo-workflows/commit/948127c7938b32a9a56ff74e31c2367d694d876d) fix(controller): retry transient error on agent pod creation (#13655)
* [6d9e0f566](https://github.com/argoproj/argo-workflows/commit/6d9e0f566cbcf55ee9bdba2c5511a24c172b5338) fix(ui): allow `links` to metadata with dots. Fixes #11741 (#13752)
* [0dfecd6e3](https://github.com/argoproj/argo-workflows/commit/0dfecd6e3a18c7bb884000e1e98d8305440d8d49) fix(ui): switch from js-yaml to yaml. Fixes #12205 (#13750)
* [ad114b047](https://github.com/argoproj/argo-workflows/commit/ad114b0472c0079cb7a982f7f577bc1e965b310b) feat: `SKIP_WORKFLOW_DURATION_ESTIMATION`. Fixes #7271 (#13745)
* [91765fad2](https://github.com/argoproj/argo-workflows/commit/91765fad2f2444b23fb128eda6e8e37948f4b4c9) feat: deprecations metric (#13735)
* [7007cba39](https://github.com/argoproj/argo-workflows/commit/7007cba39359ab870e42198291f9a4ebbf165cf7) fix(controller): node message should be updated when lock msg changed (#13648)
* [f1347b64f](https://github.com/argoproj/argo-workflows/commit/f1347b64fd8fa7ddaa30ab1f5ea5d0bb64d30905) feat(cli): validate `--output` flag and refactor (#13695)
* [2735f6bb1](https://github.com/argoproj/argo-workflows/commit/2735f6bb1eab4439822b9897f1dce38d3f64edca) fix(test): fix http-template test (#13737)
* [8100f9987](https://github.com/argoproj/argo-workflows/commit/8100f99872198a55f8ad90021d01a0bb9f658900) fix(controller): handle race when starting metrics server. Fixes #10807 (#13731)
* [dfcaca4b9](https://github.com/argoproj/argo-workflows/commit/dfcaca4b92e5d020b22aa65b7826ea8f37163b02) fix(cli): handle multi-resource yaml in offline lint. Fixes #12137 (#13531)
* [7a3372039](https://github.com/argoproj/argo-workflows/commit/7a33720398c8152b6da99e006bbe4724ad0bde1c) fix(emissary): signal SIGINT/SIGTERM in windows correctly (#13693)
* [ceaabf12c](https://github.com/argoproj/argo-workflows/commit/ceaabf12cb86f185eb89c3e69880f20d61dfe6af) fix(ui): fix build failures due to conflict (#13730)
* [07703ab1e](https://github.com/argoproj/argo-workflows/commit/07703ab1e5e61f1735008bf79847af49f01af817) feat(ui): Retry a single workflow step manually (#13343)
* [15f2170a1](https://github.com/argoproj/argo-workflows/commit/15f2170a14d357c4804f8451fba4cdd1759ca872) fix(test): windows tests fixes. Fixes #11994 (#12071)
* [734b5b6e9](https://github.com/argoproj/argo-workflows/commit/734b5b6e9f18ebe385df3e41117e67a1e8764939) refactor(cli): improve CLI error handling. Fixes #1935 (#13656)
* [ba75efb22](https://github.com/argoproj/argo-workflows/commit/ba75efb228fa3fbab91e2b1c7689f6db3f3c3b62) fix(ui): ignore `@every` cron expression for FE pretty printing and update `cronstrue` dependency. Fixes #13489 (#13586)
* [f1fbe09cb](https://github.com/argoproj/argo-workflows/commit/f1fbe09cbfe8505a9804a1c044f2a79d0de2ff28) fix: skip clear message when node transition from pending to fail. Fixes #13200 (#13201)

<details><summary><h3>Contributors</h3></summary>

* Adrien Delannoy
* Alan Clucas
* Anton Gilgur
* Ashley Manraj
* Darko Janjic
* Julie Vogelman
* Mason Malone
* Michael Weibel
* Tianchu Zhao
* Yuping Fan
* Zadkiel AHARONIAN
* chengjoey
* github-actions[bot]
* jswxstw
* tooptoop4
* wayne

</details>

## v3.6.0-rc2 (2024-10-01)

Full Changelog: [v3.6.0-rc1...v3.6.0-rc2](https://github.com/argoproj/argo-workflows/compare/v3.6.0-rc1...v3.6.0-rc2)

### Selected Changes

* [68adbcc0c](https://github.com/argoproj/argo-workflows/commit/68adbcc0cce29e4daa37b521f02ca2fccec1ca2c) fix(ui): handle React 18 batching for "Submit" button on details page. Fixes #13453 (#13593)
* [ca6c4144c](https://github.com/argoproj/argo-workflows/commit/ca6c4144ced98e111d77111fc696984739607694) fix: all written artifacts should be saved and garbage collected (#13678)
* [a0ba3c70d](https://github.com/argoproj/argo-workflows/commit/a0ba3c70d18227e74bed1d77aa86c4ad4212d159) fix: add `cronWorkflowWorkers` log. Fixes: #13681 (#13688)
* [54621cc60](https://github.com/argoproj/argo-workflows/commit/54621cc60117cf68183be24322119d85a80bb650) feat(cli): add version header + warning on client-server mismatch. Fixes #9212 (#13635)
* [fc7b21009](https://github.com/argoproj/argo-workflows/commit/fc7b210097181a9e74540e93cee16bffc9f1a682) fix(cron): allow unresolved variables outside of `when` (#13680)
* [ef09d9ffe](https://github.com/argoproj/argo-workflows/commit/ef09d9ffed0fb3c0f3e672db588d6a7c6cf2a5f5) refactor(deps): drop `pkg/errors` as a direct dependency (#13673)
* [e0ca7ffd1](https://github.com/argoproj/argo-workflows/commit/e0ca7ffd1c0d94d744ebb79d0fd28a56121ab0ee) fix: add retry for invalid connection. Fixes #13578 (#13580)
* [524406451](https://github.com/argoproj/argo-workflows/commit/524406451f4dfa57bf3371fb85becdb56a2b309a) fix: Prevent data race from global metrics round-tripper (#13641)
* [2dac1266b](https://github.com/argoproj/argo-workflows/commit/2dac1266b446d059ac6df78afe3d09a6ddb3af4b) fix(cron): various follow-ups for multiple schedules (#13369)
* [dc731d04a](https://github.com/argoproj/argo-workflows/commit/dc731d04ab14deea20ccca221f33e0df1143ad42) fix(deps): Upgrade swagger-ui-react to v5.17.12. Fixes CVE-2024-45801 (#13626)
* [4e887521b](https://github.com/argoproj/argo-workflows/commit/4e887521befd5cc0e490f09d04cd3fb887a00ea9) feat: multiple mutexes and semaphores (#13358)

<details><summary><h3>Contributors</h3></summary>

* Alan Clucas
* Andrew Melnick
* Anton Gilgur
* Eduardo Rodrigues
* Isitha Subasinghe
* Julie Vogelman
* Mason Malone
* Philipp Pfeil
* Tim Collins
* William Van Hevelingen
* Yuan Tang
* github-actions[bot]
* l2h
* shuangkun tian

</details>

## v3.6.0-rc1 (2024-09-18)

Full Changelog: [v3.5.13...v3.6.0-rc1](https://github.com/argoproj/argo-workflows/compare/v3.5.13...v3.6.0-rc1)

### Selected Changes

* [440f1ccc9](https://github.com/argoproj/argo-workflows/commit/440f1ccc986c01ff0f76d161cb8c9389eadce3e2) fix(semaphore): ensure `holderKey`s carry all information needed. Fixes #8684 (#13553)
* [0604fda01](https://github.com/argoproj/argo-workflows/commit/0604fda018e6fe5b78fd55edb9ad981d14ef730d) chore(deps): update go to 1.23 (#13613)
* [5b8ae35f1](https://github.com/argoproj/argo-workflows/commit/5b8ae35f150fd2db2281bd804bc28f56a1a6ecc3) fix(controller): handle `nil` `processedTmpl` in DAGs (#13548)
* [889a9d24b](https://github.com/argoproj/argo-workflows/commit/889a9d24be072c5d04e853db3d4c40a04f939d28) chore(deps)!: bump k8s dependencies to 1.30 (#13413)
* [e593450c4](https://github.com/argoproj/argo-workflows/commit/e593450c47a907fb9baa7b5e09011f6d516c16a2) feat: concurrency policy triggered counter metric (#13497)
* [196aeaba6](https://github.com/argoproj/argo-workflows/commit/196aeaba6c26cb725d0d1406f0e3af11ef3c03c3) feat(cron): cronworkflows `when` clause (#13474)
* [29ea3518a](https://github.com/argoproj/argo-workflows/commit/29ea3518a30b2b605290d0ba38f95bfb75a2901f) fix(test): it is possible to get 1Tb of RAM on a node (#13606)
* [860c862c0](https://github.com/argoproj/argo-workflows/commit/860c862c0b4b1a67444d1263ddcfcc00909e8725) refactor(ui): reorganize `shared/utils.ts` (#13339)
* [19b2322e2](https://github.com/argoproj/argo-workflows/commit/19b2322e26127145a895ec46931efebfd1b0f3b5) feat(artifacts): support ephemeral credentials for S3. Fixes #5446 (#12467)
* [31493d3c3](https://github.com/argoproj/argo-workflows/commit/31493d3c3f8acbf5dd4202f181f36596773cd35b) chore!: remove legacy patch pods fallback (#13100)
* [8065edb34](https://github.com/argoproj/argo-workflows/commit/8065edb34df0e188fd88181b4660c705997e6dcf) fix: remove non-transient logs on missing `artifact-repositories` configmap (#13516)
* [729ac1746](https://github.com/argoproj/argo-workflows/commit/729ac174618696e35324e16a739088a2f06aa0ec) fix(api): optimise archived list query. Fixes #13295 (#13566)
* [1b0f00803](https://github.com/argoproj/argo-workflows/commit/1b0f00803d513d10b0f6f1bc73e99c8d2440c576) feat: support data transfer protection in HDFS artifacts (#13482) (#13483)
* [ed3b8ce77](https://github.com/argoproj/argo-workflows/commit/ed3b8ce778c23e3b3c0d54f87f790e966b15fdb3) fix(ci): revert `\*_PKG_FILES` on windows `argoexec` build (#13568)
* [c698990a0](https://github.com/argoproj/argo-workflows/commit/c698990a0283e98b6695bb8f7dd3651eac17331f) fix(api): `deleteDelayDuration` should be a string (#13543)
* [0d9534140](https://github.com/argoproj/argo-workflows/commit/0d95341404b82f6d8ada619cbac48623ce2c8803) refactor(ci): move api knowledge to the matrix (#13569)
* [b3756472a](https://github.com/argoproj/argo-workflows/commit/b3756472a635e3162761b49aceebec8ac6ef6ae5) fix: ignore error when input artifacts optional. Fixes:#13564 (#13567)
* [7173a271b](https://github.com/argoproj/argo-workflows/commit/7173a271bb9c59ca67df7a06965eb80afd37c0cb) refactor: separate metrics code for reuse (#13514)
* [c29c63006](https://github.com/argoproj/argo-workflows/commit/c29c63006fa404ab374d52c103d7bef4cf3fcc9a) fix: aggregate JSON output parameters correctly (#13513)
* [8142d1990](https://github.com/argoproj/argo-workflows/commit/8142d1990e5d0317b0ff716ea8d691b320090d11) fix(executor): add executable permission to staged `script` (#12787)
* [8a67009c8](https://github.com/argoproj/argo-workflows/commit/8a67009c80e6c842836281872ab9ebdc1c2fb8a3) fix: Mark task result as completed if pod has been deleted for a while. Fixes #13533 (#13537)
* [a35917e49](https://github.com/argoproj/argo-workflows/commit/a35917e493d67004d12ed653dea408acfdae4407) chore(deps): bump `docker` from 26.1.5 to 27.1.1 (#13524)
* [b026a0f1f](https://github.com/argoproj/argo-workflows/commit/b026a0f1ff01b2266f7797b7205717995a6973cb) fix: don't clean up old offloaded records during save. Fixes: #13220 (#13286)
* [3d41fb232](https://github.com/argoproj/argo-workflows/commit/3d41fb2329ab5ed63db05cf2cbc373c082a6a7aa) fix: Mark taskResult completed if wait container terminated not gracefully. Fixes #13373 (#13491)
* [d254ffd21](https://github.com/argoproj/argo-workflows/commit/d254ffd2155a0bf80ea64608ea0114d2a55ea703) feat(ui): add full workflow name in list details. Fixes #11949 (#13519)
* [9da27d11d](https://github.com/argoproj/argo-workflows/commit/9da27d11db0c451c1cd700e94484075462ac1100) refactor(tests): `if assert.\*` -> `require.\*`. Fixes #13465 (#13495)
* [fed83ca9b](https://github.com/argoproj/argo-workflows/commit/fed83ca9b9678d86e57aea55cf0f73dee879c86f) feat(ui): Add start/end workflows ISO display switch (#13284)
* [983c6ca5f](https://github.com/argoproj/argo-workflows/commit/983c6ca5f489d1b314d930e2fe7b510b89552973) fix(docs): remove unused spaces (#13487)
* [6d426882b](https://github.com/argoproj/argo-workflows/commit/6d426882b9a42a732b1be3b2bfcd327165546206) fix(docs): use `sh` instead of `bash` with `busybox` (#13508)
* [f8f189379](https://github.com/argoproj/argo-workflows/commit/f8f1893799b67446a833897af7f860fa08e71b1b) fix: Do not reset the root node by default. Fixes #13196 (#13198)
* [735739f16](https://github.com/argoproj/argo-workflows/commit/735739f16822bb8c869662595a186875b3525307) fix(controller): remove ArtifactGC `finalizer` when no artifacts. Fixes #13499 (#13500)
* [ddbb3c7ad](https://github.com/argoproj/argo-workflows/commit/ddbb3c7ad5b498d50514b3c1158ded56e333d75b) fix: Task result comparison is incorrect, leading to inconsistent comparison results. (#13211)
* [b8bf82104](https://github.com/argoproj/argo-workflows/commit/b8bf82104bb6829f75a4e26198041ed47265bb1e) feat: new template counting and phase counting metrics (#13275)
* [dcd943618](https://github.com/argoproj/argo-workflows/commit/dcd943618e4e85443c75944f5f2c333f07c89805) feat: new pod pending counter metric (#13273)
* [219234443](https://github.com/argoproj/argo-workflows/commit/2192344439980227430162c8e2e53c180f10ac34) feat: new cron workflow trigger counter metric (#13274)
* [f8e125a29](https://github.com/argoproj/argo-workflows/commit/f8e125a29ee86709629a4fe4818aa98afe545d61) feat: new pod phase counter metric (#13272)
* [51951aac2](https://github.com/argoproj/argo-workflows/commit/51951aac2c5bd7e05326d02b67afad9102f0e208) feat: new k8s request duration histogram metric (#13271)
* [bc22fb5bb](https://github.com/argoproj/argo-workflows/commit/bc22fb5bbb8b47756557e405fce69167d796d030) chore(deps): bump `golangci-lint` from 1.55.1 to 1.59.1 (#13473)
* [b0ad007d2](https://github.com/argoproj/argo-workflows/commit/b0ad007d2cde946257191853e79824f9fa2a4042)  ci: Remove Synk ignore for vulnerability for jackc/pgx/v4 (#13481)
* [adf98ceb6](https://github.com/argoproj/argo-workflows/commit/adf98ceb688a444adc32e6039c111ad7461f991f) feat: new leader metric (#13270)
* [cd822031c](https://github.com/argoproj/argo-workflows/commit/cd822031c90dbab992c1047649850f91bcbe5f53) feat: new version metric (#13269)
* [312c0d06a](https://github.com/argoproj/argo-workflows/commit/312c0d06a3ff7be24d5305c8dd8cc4771b86e2a3) feat: enable configuration of individual metrics (#13268)
* [b88a23a6f](https://github.com/argoproj/argo-workflows/commit/b88a23a6fcf873e47733c30994e260a360548d81) fix(build): correct `make` target deps for Go binaries (#13145)
* [c143e3e78](https://github.com/argoproj/argo-workflows/commit/c143e3e78c1b6d585d64a2d9363bf0c9c64caf6c) feat(controller): optimize memory with queue when archiving is rate-limited. Fixes #13418 (#13419)
* [c0f879cc9](https://github.com/argoproj/argo-workflows/commit/c0f879cc922ba77c63e13154a85be04285aff194) feat: enable configuration of temporality in opentelemetry metrics (#13267)
* [9b2dc8dc7](https://github.com/argoproj/argo-workflows/commit/9b2dc8dc7f570529d9e8248577427c3ad34a5dec) fix: avoid exit handler nil pointer when missing outputs. Fixes #13445 (#13448)
* [5842c5cfa](https://github.com/argoproj/argo-workflows/commit/5842c5cfa1f1e415540cd791256ef17eac74fe7c) feat!: enable Prometheus TLS by default (#13266)
* [892c131b2](https://github.com/argoproj/argo-workflows/commit/892c131b28ceec53666ccde2eba81b5b5bced22b) chore(deps): bump axios from 1.6.5 to 1.7.4 in /ui in the deps group (#13472)
* [36b7a72d5](https://github.com/argoproj/argo-workflows/commit/36b7a72d56025ace31de336258c5dbb7543c4c3c) fix: mark node failed if pod absent. Fixes #12993 (#13454)
* [9756babd0](https://github.com/argoproj/argo-workflows/commit/9756babd0ed589d1cd24592f05725f748f74130b) feat!: OpenTelemetry metrics (#13265)
* [282a0d38d](https://github.com/argoproj/argo-workflows/commit/282a0d38df8f52879f04071017dc24d320e1e4d8) feat(ui): add history to WorkflowTemplate & ClusterWorkflowTemplate details (#13452)
* [3f161cb12](https://github.com/argoproj/argo-workflows/commit/3f161cb125ce11ea438ef48acbbd283a77ea248e) refactor(tests): `assert.Error`->`require.Error` (remainder) (#13462)
* [9490db72b](https://github.com/argoproj/argo-workflows/commit/9490db72b602d089344e7011a8a060e7d55d9b00) refactor(tests): testifylint don't use Equal for floating point tests (#13459)
* [2d7e2b559](https://github.com/argoproj/argo-workflows/commit/2d7e2b55914a89653613f75d2d8b19ccb9c01c39) fix: Set initial progress from pod metadata if exists. Fixes #13057 (#13260)
* [a572e7120](https://github.com/argoproj/argo-workflows/commit/a572e7120938e0da493d3f79b2c089bf30222298) fix(docs): Provide versioned links to documentation (#13455)
* [16f0a8ea3](https://github.com/argoproj/argo-workflows/commit/16f0a8ea3cc0690d32994d309ac6b6e26a9409ca) fix: Only apply execution control to nodes that are not part of exit handler. (#13016)
* [eaf84460d](https://github.com/argoproj/argo-workflows/commit/eaf84460d5bceb39b5c3acd6fbe47b802262b9d9) refactor(tests): `assert.Error` -> `require.Error` (`workflow/controller/`) (#13401)
* [24de3c63c](https://github.com/argoproj/argo-workflows/commit/24de3c63c907e6899baf09c24ba3b215073964d7) refactor(tests): `assert.Error` -> `require.Error` (other `workflow/` dirs) (#13402)
* [df66dbe0b](https://github.com/argoproj/argo-workflows/commit/df66dbe0b7b1b1f83911c440176fb97e6f683e5c) chore(deps): bump github.com/docker/docker from 26.1.4+incompatible to 26.1.5+incompatible in the go_modules group (#13446)
* [902ddd4a8](https://github.com/argoproj/argo-workflows/commit/902ddd4a8a88a646e4903197b7d353a38d63d8c9) fix(docs): replace outdated `whalesay` image with `busybox` (#13429)
* [37cfbb54d](https://github.com/argoproj/argo-workflows/commit/37cfbb54dba1f25f2c1ba2f281dd193bcabdcd5a) fix(swagger): no spaces in `archived-workflows`. Fixes #10609 (#13417)
* [500822ae6](https://github.com/argoproj/argo-workflows/commit/500822ae6ceb19e935692f0a896cb72da66d49a2) chore(deps): bump github.com/docker/docker from 24.0.9+incompatible to 26.1.4+incompatible in the go_modules group (#13416)
* [7cc20bbfa](https://github.com/argoproj/argo-workflows/commit/7cc20bbfa5a4b871b757413a76cc3259894baaea) fix: improve get archived workflow query performance during controller estimation. Fixes #13382 (#13394)
* [1239fd022](https://github.com/argoproj/argo-workflows/commit/1239fd022b8444c692d724ab41548b37d47bc0a4) fix(server): don't return `undefined` SA NS (#13347)
* [2cb91187f](https://github.com/argoproj/argo-workflows/commit/2cb91187f6b3cb8b341cb9a3509847efeef898fd) refactor(tests): `assert.Error` -> `require.Error` (other unit tests) (#13399)
* [64a134468](https://github.com/argoproj/argo-workflows/commit/64a13446828c5ca92a0c0a074ab89f34800cde24) refactor(tests): `assert.Error` -> `require.Error` (`cmd/`) (#13397)
* [7d1da63fe](https://github.com/argoproj/argo-workflows/commit/7d1da63fe0cf8bbdd0e6b536142ce422e2a3476b) refactor(tests): `assert.Error` -> `require.Error` (`pkg/`) (#13398)
* [904683cb2](https://github.com/argoproj/argo-workflows/commit/904683cb2429a58d6c92f367c548952622dfc9ac) refactor(tests): `assert.Error` -> `require.Error` (`util/`) (#13400)
* [5d8ee22b2](https://github.com/argoproj/argo-workflows/commit/5d8ee22b249e1702894709ae3aaeb1995d6028a4) fix(resource): don't use `-f` when patch file is provided (#13317)
* [da3cf7423](https://github.com/argoproj/argo-workflows/commit/da3cf742396309474ea2d858212fdc0a1198270b) refactor(tests): testifylint automated fixes (#13396)
* [637ffce64](https://github.com/argoproj/argo-workflows/commit/637ffce641efae16b5168075a5518dd9f24d649e) fix(test): load to stream tmp file counting (#13366)
* [092a43b23](https://github.com/argoproj/argo-workflows/commit/092a43b2344017ef1185b2e9d2b381618e4acf5d) fix(build): slightly optimize UI tasks in `kit` `tasks.yaml` (#13350)
* [c16c5e4ad](https://github.com/argoproj/argo-workflows/commit/c16c5e4adfbc1e321f9e9293d8364904c6db4bf9) fix(ui): display Bitbucket Server event source icon in event flow. Fixes #13386 (#13387)
* [c55130451](https://github.com/argoproj/argo-workflows/commit/c5513045110684e565ad7f98aff7293dbed07fba) refactor: remove unnecessary `PriorityMutex` implementation (#13337)
* [1ed136869](https://github.com/argoproj/argo-workflows/commit/1ed1368694b660eeb5a527fd23695263ef0e4fde) fix: Mark non-fulfilled taskSetNodes error when agent pod failed. Fixes #12703 (#12723)
* [7357a1bbd](https://github.com/argoproj/argo-workflows/commit/7357a1bbd06dbef29ad6fa757194e95562206a49) fix(devcontainer): `chown` regression for `make codegen` (#13375)
* [a27fed4d8](https://github.com/argoproj/argo-workflows/commit/a27fed4d8401328750db109da26e85d2e8f22c37) feat: Support authentication via Shared Access Signatures (SAS) for Azure artifacts (Fixes #10297)  (#13360)
* [52cca7e07](https://github.com/argoproj/argo-workflows/commit/52cca7e079a4f6d76db303ac550b1876e51b3865) fix(cron): `schedules` with `timezone` (#13363)
* [14038a3b3](https://github.com/argoproj/argo-workflows/commit/14038a3b3c3dc50508143bbce2c119ac8bff2bb1) feat(cli): support `-l` flag for `template list`. Fixes #13309 (#13364)
* [a154a93c9](https://github.com/argoproj/argo-workflows/commit/a154a93c978de3dc1fc2e4a6ca19a3b39d62ee5e) fix: constraint containerType outboundnode boundary. Fixes #12997 (#13048)
* [1f0adca09](https://github.com/argoproj/argo-workflows/commit/1f0adca09374362d47cf5fd9f19b4b4b2e104d03) fix(devcontainer): remove `-R` in `chown` (#13348)
* [c834ef5e6](https://github.com/argoproj/argo-workflows/commit/c834ef5e6919b08e9a761b342f8b8961bbb2ad19) fix(cli): `argo lint` with strict should report case-sensitive errors. Fixes #13006 (#13250)
* [d7495b83b](https://github.com/argoproj/argo-workflows/commit/d7495b83b519e0c39b49fe692485e95286ce6665) fix(docs): correct headings in 3.4 upgrade notes (#13351)
* [61f3fd416](https://github.com/argoproj/argo-workflows/commit/61f3fd416af4bc648c1252987a3b74e77cd62041) refactor: use templateScope var instead of repeatedly retrieving it from function (#13345)
* [8604148c9](https://github.com/argoproj/argo-workflows/commit/8604148c97db65af423c34b0ae4cef73ffb89db4) fix(devcontainer): expose ports for all services (#13349)
* [efb52e50c](https://github.com/argoproj/argo-workflows/commit/efb52e50c5b4816599d60f22e33e9d4506a90a34) fix(ui): hide `Workflow gone` message when workflow is archived (#13308)
* [0d706d002](https://github.com/argoproj/argo-workflows/commit/0d706d0027e92a0abe9a58a4aae841465f1c99ec) refactor: reorganize some `Makefile` variables (#13340)
* [a4f1417d5](https://github.com/argoproj/argo-workflows/commit/a4f1417d5351caaefb91bbf8e3316e070db5bffd) chore(deps): update go to 1.22 (#13258)
* [c94e21e16](https://github.com/argoproj/argo-workflows/commit/c94e21e16ed9fc1d09a7e37f6b7810c7f215e91d) fix: allow nodes without `taskResultCompletionStatus` (#13332)
* [c59ff53a3](https://github.com/argoproj/argo-workflows/commit/c59ff53a37a7e7ac41f73b23e621f601eab9d604) fix(cli): Ensure `--dry-run` and `--server-dry-run` flags do not create workflows. fixes #12944 (#13183)
* [fc0f34ecb](https://github.com/argoproj/argo-workflows/commit/fc0f34ecba45744500aa78e01c43d42414141613) fix: Update modification timestamps on untar. Fixes #12885 (#13172)
* [8d8f95472](https://github.com/argoproj/argo-workflows/commit/8d8f95472a1dea98d57d9d77be74107978cb009a) fix(resource): catch fatal `kubectl` errors (#13321)
* [3ece3b30f](https://github.com/argoproj/argo-workflows/commit/3ece3b30f0c445204fec468fd437e77283cab913) fix(build): bump golang to 1.21.12 in builder image to fix CVEs (#13311)
* [a929c8f4d](https://github.com/argoproj/argo-workflows/commit/a929c8f4dd03460d0f1e5ab2937dd2990a659d81) fix: allow artifact gc to delete directory. Fixes #12857 (#13091)
* [d1f2f017f](https://github.com/argoproj/argo-workflows/commit/d1f2f017f96a8dde542bf5efc7a11dae7349e25f) refactor(cli): simplify counter in printer (#13291)
* [825aacf57](https://github.com/argoproj/argo-workflows/commit/825aacf574e975149ea7cbac0054c9095e5bac59) fix: Only cleanup agent pod if exists. Fixes #12659 (#13294)
* [3f9b992cd](https://github.com/argoproj/argo-workflows/commit/3f9b992cdddb09633ef329c13eab1823ca6e5f3e) fix: correct pod names for inline templates. Fixes #12895 (#13261)
* [9c5dd5b13](https://github.com/argoproj/argo-workflows/commit/9c5dd5b1325f270a76ac16311d8328f626a36ff4) fix(docs): clarify CronWorkflow `startingDeadlineSeconds`. Fixes #12971 (#13280)
* [77b573298](https://github.com/argoproj/argo-workflows/commit/77b573298faa7d8705f51f41e11e4f5b16fe67d8) fix: oss internal error should retry. Fixes #13262 (#13263)
* [06da23e86](https://github.com/argoproj/argo-workflows/commit/06da23e8660f7841592070b1a372ace0750c2364) fix(ui): parameter descriptions shouldn't disappear on input (#13244)
* [ac9cb3ae9](https://github.com/argoproj/argo-workflows/commit/ac9cb3ae93bb441bb7ce09b74e99789adf599eae) fix(server): switch to `JSON_EXTRACT` and `JSON_UNQUOTE` for MySQL/MariaDB. Fixes #13202 (#13203)
* [10e3a6dc1](https://github.com/argoproj/argo-workflows/commit/10e3a6dc1a8bd5d250ac3423e52341d28ff6b654) fix(ui): Use proper podname for containersets. Fixes #13038 (#13039)
* [6c33cbbdb](https://github.com/argoproj/argo-workflows/commit/6c33cbbdb3258d53095f8334375f4a18a4c48044) fix: Mark `Pending` pod nodes as `Failed` when shutting down. Fixes #13210 (#13214)
* [b49faaf21](https://github.com/argoproj/argo-workflows/commit/b49faaf21f0aa5bb40b722a08288b34981751da1) feat(deps): upgrade `expr` from 1.16.0 to 1.16.9 to support `concat` function. Fixes #13175 (#13194)
* [6201d759e](https://github.com/argoproj/argo-workflows/commit/6201d759ebb32ba543c01665c142d04b0428f227) fix: process metrics later in `executeTemplate`. Fixes #13162 (#13163)
* [9c57c376c](https://github.com/argoproj/argo-workflows/commit/9c57c376cc500056246d51dcd190f09473820aeb) chore(deps): bump ws from 7.5.9 to 7.5.10 in /ui in the deps group (#13217)
* [824cf991e](https://github.com/argoproj/argo-workflows/commit/824cf991ec1deb0198663be93f543a5a9bd94ab6) fix(deps): bump `github.com/Azure/azure-sdk-for-go/sdk/azidentity` from 1.5.1 to 1.6.0 to fix CVE (#13197)
* [2ca48415b](https://github.com/argoproj/argo-workflows/commit/2ca48415bde03e8c6bd1650ed42f1883176c7cd0) fix: don't necessarily include all artifacts from templates in node outputs (#13066)
* [8f3860d76](https://github.com/argoproj/argo-workflows/commit/8f3860d7683068ab4f93bc725eb24dbbaee78e11) fix(server): don't use cluster scope list + watch in namespaced mode. Fixes #13177 (#13189)
* [dc6a18dc5](https://github.com/argoproj/argo-workflows/commit/dc6a18dc545bbf35103ffddb81a014eefcbb403d) fix(ui): upgrade `argo-ui` for visible filter dropdown bar arrow. Fixes #7789 (#13169)
* [0ca0c0f72](https://github.com/argoproj/argo-workflows/commit/0ca0c0f721c9bf05c63dea98168cfa87441656c7) fix(server): mutex calls to sqlitex (#13166)
* [e00757bb4](https://github.com/argoproj/argo-workflows/commit/e00757bb41df6306fd1fb06bf295a7c8d2d74267) fix(templating): return json, not go structs (#12909)
* [b28486cdc](https://github.com/argoproj/argo-workflows/commit/b28486cdc0a2174d68a0d0d04ba788847552b826) fix: only evaluate retry expression for fail/error node. Fixes #13058 (#13165)
* [465c7b6d6](https://github.com/argoproj/argo-workflows/commit/465c7b6d6abd06a36165955d7fd01d9db2b6a2d4) fix: Merge templateDefaults into dag task tmpl. Fixes #12821 (#12833)
* [6c8393fa3](https://github.com/argoproj/argo-workflows/commit/6c8393fa392557637e9a23faa9a38696acaba207) fix: Apply podSpecPatch  in `woc.execWf.Spec` and template to pod sequentially (#12476)
* [b560ad383](https://github.com/argoproj/argo-workflows/commit/b560ad383d3a7227964c4e4a30e7b69ce6bc9b03) chore!: remove duplicate Server env vars, plus `--basehref` -> `--base-href` (#12653)
* [cc604e4fe](https://github.com/argoproj/argo-workflows/commit/cc604e4fe1e0b9a493ae5f5c7b6f3550ea7b83cc) fix: don't fail workflow if PDB creation fails (#13102)
* [effcab77b](https://github.com/argoproj/argo-workflows/commit/effcab77b2984056f721018b854d5c065844961c) fix(release): set `$DOCKER_CONFIG` if unset (#13155)
* [bda02806c](https://github.com/argoproj/argo-workflows/commit/bda02806cdc27216cd0ce33418997bf5a4215ae1) fix: Allow termination of workflow to update on exit handler nodes. fixes #13052 (#13120)
* [64850e021](https://github.com/argoproj/argo-workflows/commit/64850e0216d27911c266febb1f3fea806f70ab01) chore(deps): use `docker/login-action` consistently instead of `Azure/docker-login` (#12791)
* [58dbcd0b4](https://github.com/argoproj/argo-workflows/commit/58dbcd0b4d9900fc0074884e5a28ede62e6cd9db) fix(build): `uname` handling for mac M1 arch. Fixes #13112 (#13113)
* [b212d5fb1](https://github.com/argoproj/argo-workflows/commit/b212d5fb147e09edeed3766a717846610808096d) fix: load missing fields for archived workflows (#13136)
* [ae0c76773](https://github.com/argoproj/argo-workflows/commit/ae0c767733a4b588fb7ebc05d10c5ad681f14adf) fix(ui): `package.json#license` should be Apache (#13040)
* [c11246469](https://github.com/argoproj/argo-workflows/commit/c1124646963981f5589af15943e708cd998b8491) feat: argo cli support fish completion (#13128)
* [fce51dfd0](https://github.com/argoproj/argo-workflows/commit/fce51dfd0d49c521102a2c1a8d661fab742578a5) fix: silence devcontainer `Permission denied` error. Fixes #13109 (#13111)
* [f480eb9d5](https://github.com/argoproj/argo-workflows/commit/f480eb9d5ccbc2e5e07637a6bd8bef9f01141aa8) fix(docs): Fix `gcloud` typo (#13101)
* [0076cc2bf](https://github.com/argoproj/argo-workflows/commit/0076cc2bf4014c2c8a649dd70852c360333f78f2) refactor(ui): heavily simplify `WorkflowCreator` effect (#13094)
* [4d8f97290](https://github.com/argoproj/argo-workflows/commit/4d8f97290d10305c4ba96e442286b64223515af7) refactor(ui): code-split out large `xterm` dep (#12158)
* [ae2ad227a](https://github.com/argoproj/argo-workflows/commit/ae2ad227a948f132b9cf98f5ace4e6a57223af82) chore(deps): update nixpkgs to nixos-24.05 (#13080)
* [920d965bb](https://github.com/argoproj/argo-workflows/commit/920d965bb6742b215f2a0658fe3a9cae73887d3e) fix(ui): show container logs when using `templateRef` (#12973)
* [b5ed73017](https://github.com/argoproj/argo-workflows/commit/b5ed730172fffa5307e5add28084bbd980d35286) fix: Enable realtime metric gc after its workflow is completed. Fixes #12790 (#12830)
* [d4b9327b9](https://github.com/argoproj/argo-workflows/commit/d4b9327b93511164d4f4401df6e867cd08faa2f7) fix(deps): upgrade swagger-ui-react v5.17.3, react-dom v18.3.1, and react v18.3.1. Fixes CVEs (#13069)
* [670c51a79](https://github.com/argoproj/argo-workflows/commit/670c51a7997d22bd9b337b58b90274859a75f16e) fix: delete skipped node when resubmit with memoized.Fixes: #12936 (#12940)
* [e490d4815](https://github.com/argoproj/argo-workflows/commit/e490d4815dd1a93331a141ed02a7dacd46a4fb5b) fix: nodeAntiAffinity is not working as expected when boundaryID is empty. Fixes: #9193 (#12701)
* [71f1d860b](https://github.com/argoproj/argo-workflows/commit/71f1d860b2c665ad87e8c313c4f865b829d07626) fix: ignore retry node when check succeeded descendant nodes. Fixes: #13003 (#13004)
* [adef075c2](https://github.com/argoproj/argo-workflows/commit/adef075c2a936f0b9edc4513f8bc4490bb551536) fix: add timeout for executor signal (#13012)
* [9f620d784](https://github.com/argoproj/argo-workflows/commit/9f620d7843c7ac5f2b283e900c922e1e4b5bfac8) fix(docs): Clarify quick start installation. Fixes #13032  (#13047)
* [9cacef302](https://github.com/argoproj/argo-workflows/commit/9cacef302bca869da1700b72bede1afad0e9526d) refactor(test): use `t.Setenv` (#13036)
* [5b7d1472f](https://github.com/argoproj/argo-workflows/commit/5b7d1472f393d60d884d3809b57bde7d4d7bd538) refactor(ui): optimize state in logs viewer  (#13046)
* [f4419d04c](https://github.com/argoproj/argo-workflows/commit/f4419d04c22766ec7029bc66f9358f96e3673c48) feat(executor): set seccomp profile to runtimedefault (#12984)
* [0a096e66b](https://github.com/argoproj/argo-workflows/commit/0a096e66ba185d02caa5172fa677087dd4aba065) feat: add sqlite-based memory store for live workflows. Fixes #12025 (#13021)
* [e3b0bb6b2](https://github.com/argoproj/argo-workflows/commit/e3b0bb6b289eafcbe5ecea9c1e3b920ca2bc31a8) fix: don't rebuild `ui/dist/app/index.html` in `argocli-build` stage (#13023)
* [2f1570995](https://github.com/argoproj/argo-workflows/commit/2f15709952075ba208cf45ec459e66a14fa37d02) fix: use argocli image from pull request in CI (#13018)
* [a4fc31887](https://github.com/argoproj/argo-workflows/commit/a4fc31887041e80f7f8f51836bbf692a6ed83c98) fix: setBucketLifecycleRule error in OSS Artifact Driver.  Fixes #12925 (#12926)
* [b58118769](https://github.com/argoproj/argo-workflows/commit/b581187691b4d8ad82a90be15f9b56b1ccaaa1ec) fix(ci): Add missing quote in retest workflow. Fixes #12864 (#13007)
* [ccb71bdac](https://github.com/argoproj/argo-workflows/commit/ccb71bdacd13ead899a9005f3078225b5e388715) feat: implement OpenStream func of OSS artifactdriver. Part of #8489 (#12908)
* [b4af68bdd](https://github.com/argoproj/argo-workflows/commit/b4af68bdd48c41bcf75ba433357c9b9b01a0e0c1) feat(ci): Add retest workflow. Fixes #12864 (#13000)
* [6ea442051](https://github.com/argoproj/argo-workflows/commit/6ea442051f7284a06ae33a1ae57b42493f0d4c25) ci(deps): group Dependabot updates by devDeps vs prod deps (#12890)
* [f1ab5aa32](https://github.com/argoproj/argo-workflows/commit/f1ab5aa32f766090998bcaf5a2706c7e3a6cc608) feat: add sqlite-based memory store for live workflows. Fixes #12025 (#12736)
* [618238644](https://github.com/argoproj/argo-workflows/commit/6182386445f162cabdf8c58312702537cc146006) fix: retry large archived wf. Fixes #12740 (#12741)
* [30ca3692b](https://github.com/argoproj/argo-workflows/commit/30ca3692b4964bca18e8197e737c6ff4f429700f) refactor(controller): optimize PDB creation request (#12974)
* [a2480cb9c](https://github.com/argoproj/argo-workflows/commit/a2480cb9cb8dfbe7c41b124c1a7a25944ac969a8) chore(docs)!: remove references to no longer supported executors (#12975)
* [1b414a3d9](https://github.com/argoproj/argo-workflows/commit/1b414a3d9cff3ee00eb659b716558aefb1e975b6) fix: `insecureSkipVerify` for `GetUserInfoGroups` (#12982)
* [d578d2cb0](https://github.com/argoproj/argo-workflows/commit/d578d2cb095903fe0f7b3731b77bbcb5aa7bfa4d) fix: use GetTemplateFromNode to determine template name (#12970)
* [7c0465354](https://github.com/argoproj/argo-workflows/commit/7c04653543cdfd48e6f2ec96fc4bb87779ed2773) feat(cli): add riscv64 support (#12977)
* [9e1432a8b](https://github.com/argoproj/argo-workflows/commit/9e1432a8b439eb74e901ceed460d298ca3633e2f) fix(ui): try retrieving live workflow first then archived (#12972)
* [671320d4f](https://github.com/argoproj/argo-workflows/commit/671320d4f52bdb948e5cfeef27a90c3c97e9157d) refactor: simplify `getPodName` and make consistent with back-end (#12964)
* [b1c37686c](https://github.com/argoproj/argo-workflows/commit/b1c37686c35b78fc93a025f37469591ef5de0c1b) fix(ui): remove unnecessary hard reload after delete (#12930)
* [a5370334d](https://github.com/argoproj/argo-workflows/commit/a5370334df9e69c47a169ddb70d458faaa95b920) fix(ui): use router navigation instead of page load after submit (#12950)
* [08772fb0a](https://github.com/argoproj/argo-workflows/commit/08772fb0ae49bf1cc77d0170f26a0ebddf9f632d) fix(ui): handle non-existent labels in `isWorkflowInCluster` (#12898)
* [04b51beb6](https://github.com/argoproj/argo-workflows/commit/04b51beb6afd0818657b076b5410acb4be6b89f3) fix(ui): properly get archive logs when no node was chosen (#12932)
* [13d553c45](https://github.com/argoproj/argo-workflows/commit/13d553c45d4d9e7d3a27674e0e25f515e78629fe) feat(cli): Add `--no-color` support to `argo lint`. Fixes #12913 (#12953)
* [6b16f24c9](https://github.com/argoproj/argo-workflows/commit/6b16f24c97e4b9c792e18835304e517f2d0ddd0f) fix(ui): default to `main` container name in event source logs API call (#12939)
* [f0a867b63](https://github.com/argoproj/argo-workflows/commit/f0a867b6325ec3f20369bf6c92518dc5bb2ebf59) fix(build): close `pkg/apiclient/_.secondary.swagger.json` (#12942)
* [f80b9e888](https://github.com/argoproj/argo-workflows/commit/f80b9e8886091742b436613f09ece01a740c9fe5) fix: don't load entire archived workflow into memory in list APIs (#12912)
* [431960843](https://github.com/argoproj/argo-workflows/commit/43196084339df2adecaaeb945514d472d7b0a0ad) refactor(ui): simplify `hasArtifactLogs` with optional chaining (#12933)
* [ea5410aab](https://github.com/argoproj/argo-workflows/commit/ea5410aab7c13146df6bb2523c966c6a98538bbd) fix: correct order in artifactGC error log message (#12935)
* [20956219f](https://github.com/argoproj/argo-workflows/commit/20956219fa0e774d39b19b2560bb253ef1d8a54f) fix: workflows that are retrying should not be deleted (Fixes #12636) (#12905)
* [ec5b5d5f4](https://github.com/argoproj/argo-workflows/commit/ec5b5d5f49d33172cd5dd018b4c5434551d93970) fix: change fatal to panic.  (#12931)
* [7eaf3054b](https://github.com/argoproj/argo-workflows/commit/7eaf3054bb1658e445852a7e1eecd364c50ff88a) fix: Correct log level for agent containers (#12929)
* [76b2a3f5f](https://github.com/argoproj/argo-workflows/commit/76b2a3f5f50c3c214931b1c56d185b1a940a2811) feat: support dynamic templateref naming. Fixes: #10542 (#12842)
* [2eb241586](https://github.com/argoproj/argo-workflows/commit/2eb241586746326f0fa6e2ee4dd3b45d77910551) fix: DAG with continueOn in error after retry. Fixes: #11395 (#12817)
* [87a2041ba](https://github.com/argoproj/argo-workflows/commit/87a2041ba7507c15070a3855b924c1075c50c268) refactor: remove unnecessary `AddEventHandler` error handling (#12917)
* [561beb2ce](https://github.com/argoproj/argo-workflows/commit/561beb2ce1d3ce5fbf068950299b9ae5212093ca) feat: allow custom http client and use ctx for http1 apiclient. Fixes #12827 (#12867)
* [e3bfce5dd](https://github.com/argoproj/argo-workflows/commit/e3bfce5dd4ddc4b9db76688f62d7d5e2d2eaaa02) fix(deps): upgrade x/net to v0.23.0. Fixes CVE-2023-45288 (#12921)
* [fc30b5a2b](https://github.com/argoproj/argo-workflows/commit/fc30b5a2b790c99b8c5addf67d14dbbefea7418d) fix(deps): upgrade `http2` to v0.24. Fixes CVE-2023-45288 (#12901)
* [00101c6f0](https://github.com/argoproj/argo-workflows/commit/00101c6f0bed9022d216d0ad678ec00c40fdaed3) feat: implement Delete func of OSS ArtifactDriver. Fixes #9349 (#12907)
* [3b4551b6f](https://github.com/argoproj/argo-workflows/commit/3b4551b6f33bd22519cc98e2d8c157a92720b5d7) fix(deps): upgrade `crypto` from v0.20 to v0.22. Fixes CVE-2023-42818 (#12900)
* [026848cec](https://github.com/argoproj/argo-workflows/commit/026848cec4605725d31cd39bff254eaea1a78eb1) chore(deps): bump `undici` from 5.28.3 to 5.28.4 in /ui (#12891)
* [40eb51ec7](https://github.com/argoproj/argo-workflows/commit/40eb51ec72c1cdac25ed91d381375704fd84b90d) chore(deps): upgrade `mkdocs-material` from 8.2.6 to 9.x (#12894)
* [e4ce3327b](https://github.com/argoproj/argo-workflows/commit/e4ce3327b585ca7ef1afaa615f4ed60a1ebc4d28) ci(deps): remove auto `yarn-deduplicate` on dependabot PRs (#12892)
* [d2369c977](https://github.com/argoproj/argo-workflows/commit/d2369c977d1c500d3aa9a4dae8352b7008b35f79) fix: use multipart upload method to put files larger than 5Gi to OSS. Fixes #12877 (#12897)
* [66d8351b5](https://github.com/argoproj/argo-workflows/commit/66d8351b51a89157eff99e9bd8178913a44d62b7) chore(deps): bump `express`, `follow-redirects`, and `webpack-dev-middleware` (#12880)
* [ab107e599](https://github.com/argoproj/argo-workflows/commit/ab107e599811541badd07ccb70660b6420571d58) ci(deps): fix typo in `--allow-empty` (#12889)
* [7e4a0dbec](https://github.com/argoproj/argo-workflows/commit/7e4a0dbec71c89e9c20c6bdb4a22b7c035f944d2) ci(deps): fix auto deduplication of `yarn.lock` for Dependabot PRs (#12882)
* [adb6d5d31](https://github.com/argoproj/argo-workflows/commit/adb6d5d31c46e8cb45107decf4c9f95d323fc495) build(deps): bump github.com/go-jose/go-jose/v3 from 3.0.1 to 3.0.3 (#12879)
* [23927c5a6](https://github.com/argoproj/argo-workflows/commit/23927c5a64d06373b51024f6e3e0aeaeeedbafbb) build(deps): bump github.com/docker/docker from 24.0.0+incompatible to 24.0.9+incompatible (#12878)
* [74eb72253](https://github.com/argoproj/argo-workflows/commit/74eb722539869a5e32c2f31e52e6fd16730aca70) feat(ui): display line numbers in object-editor. Fixes #12807. (#12873)
* [cd0c58e05](https://github.com/argoproj/argo-workflows/commit/cd0c58e05a088946d0e01e0275b27e43a23ba080) fix: remove completed taskset status before update workflow. Fixes: #12832 (#12835)
* [fb6c3d0b8](https://github.com/argoproj/argo-workflows/commit/fb6c3d0b801063851561ae5ae61501fba40169b0) fix: make sure Finalizers has chance to be removed. Fixes: #12836 (#12831)
* [db84d5141](https://github.com/argoproj/argo-workflows/commit/db84d5141f1dac5926d5077be1f41f9b84836cbb) chore(deps): bump github.com/argoproj/argo-events from 1.7.3 to 1.9.1 (#12860)
* [24e5ff83b](https://github.com/argoproj/argo-workflows/commit/24e5ff83b71b355de0bde8827c30a63f9563713f) refactor: use context from `processNextPodCleanupItem` (#12858)
* [daa5f7e4a](https://github.com/argoproj/argo-workflows/commit/daa5f7e4a699e235ae567fda4e456f00f6f57be5) fix(test): wait enough time to Trigger Running Hook. Fixes: #12844 (#12855)
* [54106f722](https://github.com/argoproj/argo-workflows/commit/54106f72212442a6ed4d0d111d5313a354a833fd) fix: filter hook node to find the correct lastNode. Fixes: #12109 (#12815)
* [1b095392c](https://github.com/argoproj/argo-workflows/commit/1b095392cf0e4c77a2f4a01845704fad93771597) chore(deps): upgrade Cosign to v2.2.3 (#12850)
* [748ae475d](https://github.com/argoproj/argo-workflows/commit/748ae475d3400f2aa51800c2337607811550d989) chore(deps): bump k8s dependencies to 1.26 (#12847)
* [a82a68903](https://github.com/argoproj/argo-workflows/commit/a82a68903b5063cb29abc52fa7c90b0b2eff3df8) fix: terminate workflow should not get throttled Fixes #12778 (#12792)
* [8b304489e](https://github.com/argoproj/argo-workflows/commit/8b304489eb7cba154d48020b0082a570f5b31624) chore(deps): upgrade `actions/cache` and `create-pull-request` to Node v20 (#12775)
* [e10f695f9](https://github.com/argoproj/argo-workflows/commit/e10f695f9e431db80cc8942aca5b3af4d7dc1a3c) chore(deps): bump k8s dependencies to 1.25 (#12822)
* [cfe2bb791](https://github.com/argoproj/argo-workflows/commit/cfe2bb791d0e769fbe0536bc7042b4c7ff33f987) fix(containerSet): mark container deleted when pod deleted. Fixes: #12210 (#12756)
* [842c613fd](https://github.com/argoproj/argo-workflows/commit/842c613fd1412ce95eaada3ee0d10a1a9cc2a375) refactor(cli): move common functions to `util.go` (#12839)
* [2e5fb3b24](https://github.com/argoproj/argo-workflows/commit/2e5fb3b249535b0b92f36aa14caa059d5369ab61) feat: Add update command for cron,template,cluster-template. Fixes: #5464 #7344 (#12803)
* [bcc483ea5](https://github.com/argoproj/argo-workflows/commit/bcc483ea574748843868862f9e1493a743b8c4ed) chore(deps): bump github.com/vektra/mockery from v2.26.0 to v2.42.0 (#12713)
* [a719d9409](https://github.com/argoproj/argo-workflows/commit/a719d94098a1be1f336c8273e7b404357912004a) fix: return itself when getOutboundNodes from memoization Hit steps/DAG. Fixes: #7873 (#12780)
* [dbff027ff](https://github.com/argoproj/argo-workflows/commit/dbff027ff02b696d0293c74e26308e51172611d7) fix: pass dnsconfig to agent pod. Fixes: #12824 (#12825)
* [87899e5dd](https://github.com/argoproj/argo-workflows/commit/87899e5dd66c73d14ef3f4acbfa25573d8cc3d4c) fix(deps): upgrade `undici` from 5.28.2 to 5.28.3 due to CVE (#12763)
* [6e4bc8333](https://github.com/argoproj/argo-workflows/commit/6e4bc833380dfb214f9220accf40dbbe630180f5) fix(deps): upgrade `pgx` from 4.18.1 to 4.18.2 due to CVE (#12753)
* [d5a4f7ef5](https://github.com/argoproj/argo-workflows/commit/d5a4f7ef52a3022f9b16fb8093705ced0dd897d8) fix: inline template loops should receive more than the first item. Fixes: #12594 (#12628)
* [a67829491](https://github.com/argoproj/argo-workflows/commit/a67829491722d6218f5bde2e192fc3d093e59240) feat: support dag and steps level scheduling constraints. Fixes: #12568 (#12700)
* [16cfef9d4](https://github.com/argoproj/argo-workflows/commit/16cfef9d41aadb8e34936920bc33f1bbd5ed9e8e) fix: workflow stuck in running state when using activeDeadlineSeconds on template level. Fixes: #12329 (#12761)
* [20ece9cb6](https://github.com/argoproj/argo-workflows/commit/20ece9cb6c4dfa1a6900a30d58304f32bf8e93b6) fix(ui): show correct podGC message for deleteDelayDuration. Fixes: #12395 (#12784)
* [0270a0faa](https://github.com/argoproj/argo-workflows/commit/0270a0faad550363fe3c5f1a9ae565694eeb0829) chore(deps): Bump sigs.k8s.io/controller-tools/cmd/controller-gen from v0.4.1 to v0.14.0 (#12719)
* [ebce8ef7a](https://github.com/argoproj/argo-workflows/commit/ebce8ef7a083c392881aa87e9a330c7ecb0bcd92) fix: ensure workflowtaskresults complete before mark workflow completed status. Fixes: #12615 (#12574)
* [0a7559807](https://github.com/argoproj/argo-workflows/commit/0a755980762c087570e123917acd6e319eb0cc42) fix: patch report outputs completed if task result not exists. (#12748)
* [d805b7fa6](https://github.com/argoproj/argo-workflows/commit/d805b7fa6c36eac63d3dc24a84f2348a48f66be7) fix(log): change task set to task result. (#12749)
* [7ba20fea0](https://github.com/argoproj/argo-workflows/commit/7ba20fea08e40a5792a1d25011ee8c12889960c5) fix(hack): various fixes & improvements to cherry-pick script (#12714)
* [33ad82f68](https://github.com/argoproj/argo-workflows/commit/33ad82f689b83875a3b864298221ec3872379a4a) chore(deps): bump prometheus dependencies (#12702)
* [88304568a](https://github.com/argoproj/argo-workflows/commit/88304568af5b8222344cde8e5b96d1a8484a50f8) fix: use optimistic concurrency strategy when updating pod status (#12632)
* [d4ca8d9d0](https://github.com/argoproj/argo-workflows/commit/d4ca8d9d0c35891ee70e8a407d7b1c6b245c9dd3) fix(ui): code-split markdown title + desc, fix row linking, etc (#12580)
* [9bec11438](https://github.com/argoproj/argo-workflows/commit/9bec11438cc14758f363e36be444986b9fd7782b) fix: make WF global parameters available in retries (#12698)
* [a927379f7](https://github.com/argoproj/argo-workflows/commit/a927379f761fbffe733857e7a23d69511fba90de) fix: find correct retry node when using `templateRef`. Fixes: #12633 (#12683)
* [5c8062e55](https://github.com/argoproj/argo-workflows/commit/5c8062e55d975aab117e37c7592c0a648a9e9860) fix: Add limit to number of Workflows in CronWorkflow history (#12681)
* [d3433c610](https://github.com/argoproj/argo-workflows/commit/d3433c6102f2e62e900b835f91cb31de11f74f04) refactor(ui): flatten directory structures (#12539)
* [986b069f3](https://github.com/argoproj/argo-workflows/commit/986b069f34ed5336b6f754cfc6709c30b302c768) feat: allow multiple schedules in a cron workflow (#12616)
* [33c51ed17](https://github.com/argoproj/argo-workflows/commit/33c51ed17250d2a7b66bfd02badbd012ee33dfb3) feat: CronWorkflow/WorkflowTemplate title/description in list view (#12674)
* [4ad0db97a](https://github.com/argoproj/argo-workflows/commit/4ad0db97a1b7e8fa55594ce259a07a94eb6a8166) fix: Patch taskset with subresources to delete completed node status.… (#12620)
* [57a078f8b](https://github.com/argoproj/argo-workflows/commit/57a078f8bc51f058c66d58229e0678534ebd03df) fix(typo): fix some typo (#12673)
* [030d581d7](https://github.com/argoproj/argo-workflows/commit/030d581d7e45e2f41c2f33efd6a5fb19af047027) fix(ui): `ListWatch` should not _both_ set and depend on `nextOffset` (#12672)
* [23f8c3527](https://github.com/argoproj/argo-workflows/commit/23f8c3527eb8a3e837537f9afdc795808149dcd4) fix(controller): re-allow changing executor `args` (#12609)
* [148252a8a](https://github.com/argoproj/argo-workflows/commit/148252a8a1777b3a9001dcaeb9b10c2345b6daf6) fix(controller): add missing namespace index from workflow informer (#12666)
* [0acb4356d](https://github.com/argoproj/argo-workflows/commit/0acb4356d01813193b6f38195ba0c551698e7fde) fix: retry node with expression status Running -> Pending (#12637)
* [66680f1c9](https://github.com/argoproj/argo-workflows/commit/66680f1c9bca8b47c40ce918b5d16714058647cb) fix(build): check for env vars in all dirs (#12652)
* [130417b6c](https://github.com/argoproj/argo-workflows/commit/130417b6c7bbdfa20c3e1b3482f736cf91210f65) fix(docs): remove `workflow-controller-configmap.yaml` self reference (#12654)
* [7d70fe264](https://github.com/argoproj/argo-workflows/commit/7d70fe264fb0dbf42fbdf64cf94539408806a76d) chore(deps): upgrade argoproj/pkg version (#12651)
* [ae0973aed](https://github.com/argoproj/argo-workflows/commit/ae0973aed72ceac9b08b646998fd5f508e210e54) fix: pass through burst and qps for auth.kubeclient (#12575)
* [5f4b2350b](https://github.com/argoproj/argo-workflows/commit/5f4b2350b5047a85b26f7832e5772a8482bef36d) fix: controller option to not watch configmap (#12622)
* [6c8a7157f](https://github.com/argoproj/argo-workflows/commit/6c8a7157f6f551c88113929f41c9ebba6c1b6f9f) fix: artifact subdir error when using volumeMount (#12638)
* [873d3de4c](https://github.com/argoproj/argo-workflows/commit/873d3de4c8bc7b3b6283c5add9919ac90694ad5f) chore(deps): fixed medium CVE in github.com/docker/docker v24.0.0+incompatible (#12635)
* [fbd70aac1](https://github.com/argoproj/argo-workflows/commit/fbd70aac1f8c0cb32a97414027377deb2ca42b1d) fix: Allow valueFrom in dag arguments parameters. Fixes #11900 (#11902)
* [09edbf76d](https://github.com/argoproj/argo-workflows/commit/09edbf76d2ad1d44967ef055814bf6e0f3d25b4c) fix(resources): improve ressource accounting. Fixes #12468 (#12492)
* [1c3179085](https://github.com/argoproj/argo-workflows/commit/1c3179085e9ed4de8d2a9dff28b87acd8590242b) refactor(build): simplify `mkdocs build` scripts (#12463)
* [e771bde9e](https://github.com/argoproj/argo-workflows/commit/e771bde9e09e2f5d772faee331cca36f8196cfb0) fix: make the process of patching pods exclusive (#12596)
* [13444e663](https://github.com/argoproj/argo-workflows/commit/13444e663387e3b5b331c278cd9e79fc88968d7e) refactor(ui): use `import type` syntax where possible (#12514)
* [42262690f](https://github.com/argoproj/argo-workflows/commit/42262690fcbab0ed6106d3f9069e128655b3339f) fix: upgrade expr-lang. Fixes #12037 (#12573)
* [6abe8a950](https://github.com/argoproj/argo-workflows/commit/6abe8a9503dcde3de7057922c6d7688c68ae4957) feat: Add finalizer to workflow pod to prevent 'pod deleted'. Fixes #8783 Continuing Work of #9058 (#12413)
* [8f2746a98](https://github.com/argoproj/argo-workflows/commit/8f2746a98cf93dbfa42f466d124346d3aeef3a70) fix: make sure taskresult completed when mark node succeed when it has outputs (#12537)
* [a15587755](https://github.com/argoproj/argo-workflows/commit/a15587755e250fbbe3b9538b1c15ae581a108f0c) fix: Mark resource && data template report-outputs-completed true (#12544)
* [8d27a9f12](https://github.com/argoproj/argo-workflows/commit/8d27a9f12a78f6a81998023044872ec1006e2f78) fix: make etcd errors transient (#12567)
* [af2cacb36](https://github.com/argoproj/argo-workflows/commit/af2cacb365a6cc03cc35ed9749976e095f9a03f7) fix(ui): clone the `ListWatch` callback array in `WorkflowsList` (#12562)
* [c46986e0d](https://github.com/argoproj/argo-workflows/commit/c46986e0d9c89f43272bb6686380fae4a41c82c5) fix: Global Artifact Passing. Fixes #12554 (#12559)
* [baef4856f](https://github.com/argoproj/argo-workflows/commit/baef4856ff2603c76dbe277c825eaa3f9788fc91) chore(deps): bump github.com/cloudflare/circl to 1.3.7 to fix GHSA-9763-4f94-gfch (#12556)
* [1dbc856e5](https://github.com/argoproj/argo-workflows/commit/1dbc856e51967feb58066a4087a8679b08b87be3) fix: update minio chart repo (#12552)
* [46c1324dc](https://github.com/argoproj/argo-workflows/commit/46c1324dc6d292d5cc7a55ac89f4c2be78615e9b) fix: cache configmap don't create with workflow has retrystrategy. Fixes: #12490 #10426 (#12491)
* [1ab7cd207](https://github.com/argoproj/argo-workflows/commit/1ab7cd2071c6bfc335a89340d736138b20caf421) fix: add resource quota evaluation timed out to transient (#12536)
* [b734b660e](https://github.com/argoproj/argo-workflows/commit/b734b660e90f40d58fdb8e34087e41b4bae0f2e5) chore(deps): upgrade `swagger-ui-react` to v5 (#12540)
* [1202ae473](https://github.com/argoproj/argo-workflows/commit/1202ae473a9f047cccef820a80ecb97c59a02b92) fix: prevent update race in workflow cache (Fixes #9574) (#12233)
* [e20f31226](https://github.com/argoproj/argo-workflows/commit/e20f312263491df8f2fb6a78716fc0c152f3a7f6) fix: Fixed mutex with withSequence in http template broken. Fixes #12018 (#12176)
* [4f4d31582](https://github.com/argoproj/argo-workflows/commit/4f4d31582cbdfa1b340c84f846dba2e5612c5cb7) feat: add stopStrategy to cron workflows (#12305)
* [3931e59c8](https://github.com/argoproj/argo-workflows/commit/3931e59c81c390e10ef0a0a1caf953617acc8326) fix: SSO with Jumpcloud "email_verified" field #12257 (#12318)
* [1e7b2392c](https://github.com/argoproj/argo-workflows/commit/1e7b2392caafdfe7e843f46357932e3e5df3fe93) feat: speed up resolve reference (#12328)
* [2bdd7f39d](https://github.com/argoproj/argo-workflows/commit/2bdd7f39daabe337c169d34e075958ea7c30020b) fix: Switch to upstream go-git. Fixes CVE-2023-49569 (#12515)
* [b290e518d](https://github.com/argoproj/argo-workflows/commit/b290e518da97a16c05387f993ae554f62c3edf04) fix: wrong values are assigned to input parameters of workflowtemplat… (#12412)
* [85d1c79da](https://github.com/argoproj/argo-workflows/commit/85d1c79dafe50ed0e581a4311d76750bd2b69170) feat: support long arguments (#12325)
* [19c729289](https://github.com/argoproj/argo-workflows/commit/19c729289724879f33f5e5a2da1b0d476c91b712) chore(deps): bump google.golang.org/api from 0.155.0 to 0.156.0 (#12507)
* [5dd8002e8](https://github.com/argoproj/argo-workflows/commit/5dd8002e87ae543fb4d13c4b5b5bf02c675435af) chore(deps): bump golang.org/x/crypto from 0.17.0 to 0.18.0 (#12506)
* [4a9ef2ae8](https://github.com/argoproj/argo-workflows/commit/4a9ef2ae8801f31df1ddc29e04466f4712f76a54) chore(deps): bump github.com/evanphx/json-patch from 5.7.0+incompatible to 5.8.0+incompatible (#12504)
* [b447951e1](https://github.com/argoproj/argo-workflows/commit/b447951e11d965f05766674f8ef28c16d68af5ab) fix: Add missing 'archived' prop for ArtifactPanel component. Fixes #12331 (#12397)
* [44b33fadf](https://github.com/argoproj/argo-workflows/commit/44b33fadf2d2d73cde2b5965c4a05946e45b6e75) fix: merge env bug in workflow-controller-configmap and container. Fixes #12424 (#12426)
* [2bb770e89](https://github.com/argoproj/argo-workflows/commit/2bb770e8913a490548efeda1620689fbfbfce420) feat: delete pods in parallel to speed up retryworkflow (#12419)
* [98b578ee3](https://github.com/argoproj/argo-workflows/commit/98b578ee3c23b4f5ed4698db0e258e12e714df0b) fix: http template host header rewrite(#12385) (#12386)
* [93914261c](https://github.com/argoproj/argo-workflows/commit/93914261cff4216561c89c1f5f6123e7ad0d5f61) chore(deps): bump google.golang.org/api from 0.154.0 to 0.155.0 (#12479)
* [6b2d775dd](https://github.com/argoproj/argo-workflows/commit/6b2d775dd5ff185a21208d9058e2eb4f34916aba) chore(deps): bump golang.org/x/sync from 0.5.0 to 0.6.0 (#12478)
* [563011ace](https://github.com/argoproj/argo-workflows/commit/563011ace6d54a45f6a418d5b7d1e71aa8443f5b) chore(deps): bump golang.org/x/term from 0.15.0 to 0.16.0 (#12477)
* [11ee342fe](https://github.com/argoproj/argo-workflows/commit/11ee342fe6f22ec90fdd909f0f98e65fd2de6274) fix: Resolve vulnerabilities in axios (#12470)
* [bcf5672ec](https://github.com/argoproj/argo-workflows/commit/bcf5672ecfa9492e254d6d201699fb7caa06fa7e) fix(docs): handle `fields` examples with `md_in_html` (#12465)
* [96f25af05](https://github.com/argoproj/argo-workflows/commit/96f25af0538bd3ad265d47fcbbb03a7f4953d17a) fix(docs): exclude `docs/requirements.txt` from docs build (#12466)
* [1968436a6](https://github.com/argoproj/argo-workflows/commit/1968436a6e021adc4785ec287635a6136574c5ac) fix(docs): render Mermaid diagrams in docs (#12464)
* [c71e1dc05](https://github.com/argoproj/argo-workflows/commit/c71e1dc05a052778bed597f7c0f50b2ede4a7f32) feat: Support argo plugin stop. Fixes #12333 (#12441)
* [901fdab72](https://github.com/argoproj/argo-workflows/commit/901fdab7241e135a57ac312476e7d0c91bca53ab) fix: CI Artifact Download Timeout. Fixes #12452 (#12454)
* [c63c2bc18](https://github.com/argoproj/argo-workflows/commit/c63c2bc1838bc6069703c3cd31873b51329ce15d) fix: fix missing artifacts for stopped workflows. Fixes #12401 (#12402)
* [198818b55](https://github.com/argoproj/argo-workflows/commit/198818b55518150a530ca8f894f5c25700826fda) fix: Apply workflow level PodSpecPatch in agent pod. Fixes #12387 (#12440)
* [d1cae63b9](https://github.com/argoproj/argo-workflows/commit/d1cae63b90c374e37577147c4865b81f022b631c) fix: documentation links (#12446)
* [af66abc74](https://github.com/argoproj/argo-workflows/commit/af66abc74fa9d669248e7b6b6df9104ff7be48a6) fix: ensure workflow wait for onExit hook for DAG template (#11880) (#12436)
* [ee890e9ad](https://github.com/argoproj/argo-workflows/commit/ee890e9adffaf151c42e95d55fd83bd43be79883) chore(deps): bump tj-actions/changed-files from 40 to 41 (#12433)
* [7b3d040c3](https://github.com/argoproj/argo-workflows/commit/7b3d040c35ec4d7240ff76e952b0ebb2e78448ae) chore(deps): bump github.com/aliyun/aliyun-oss-go-sdk from 3.0.1+incompatible to 3.0.2+incompatible (#12428)
* [04ebbe9bd](https://github.com/argoproj/argo-workflows/commit/04ebbe9bd271439f003e01cbac5039f7a08c5fd3) chore(deps): bump moment from 2.29.4 to 2.30.1 in /ui (#12431)
* [f583400dd](https://github.com/argoproj/argo-workflows/commit/f583400dd363c41a2e35467831b6d1740a23bf67) chore(deps): bump classnames from 2.3.3 to 2.5.1 in /ui (#12430)
* [476891963](https://github.com/argoproj/argo-workflows/commit/47689196355cbafdb2cdf9493f301abe671cf6d4) fix: custom columns not supporting annotations (#12421)
* [0a8ca818c](https://github.com/argoproj/argo-workflows/commit/0a8ca818c90068e2304a273a7b90e0e5898ea232) feat: Allow markdown in workflow title and description. Fixes #10126 (#10553)
* [87c9be744](https://github.com/argoproj/argo-workflows/commit/87c9be7447b98795e06e677485904278a6a1df60) chore(deps): bump github.com/spf13/viper from 1.18.1 to 1.18.2 (#12403)
* [54e02126e](https://github.com/argoproj/argo-workflows/commit/54e02126e03410191872ea3c4744f98101653a23) chore(deps): bump golang.org/x/crypto from 0.16.0 to 0.17.0 (#12405)
* [777efeb27](https://github.com/argoproj/argo-workflows/commit/777efeb271fe19fecee799212287e554181b8ab0) chore(deps): bump react-datepicker from 4.24.0 to 4.25.0 in /ui (#12410)
* [2f89beb6c](https://github.com/argoproj/argo-workflows/commit/2f89beb6c51842ff1f435f04168973e835fa37a8) chore(deps): bump github.com/go-openapi/jsonreference from 0.20.2 to 0.20.4 (#12404)
* [7bcd6166f](https://github.com/argoproj/argo-workflows/commit/7bcd6166ff399f4263c1403d27d0250ae059f558) feat(server): Support supplying a list of headers when keying IPs for rate limiting (#12199)
* [f5b6b17c4](https://github.com/argoproj/argo-workflows/commit/f5b6b17c44a8152aaa0a0fdb92adfe1aee5c7991) feat: support to show inputs dir artifacts in UI (#12350)
* [4f27d4df2](https://github.com/argoproj/argo-workflows/commit/4f27d4df21b1a953786bd08bef881288c6199c77) fix: move log with potential sensitive data to debug loglevel. Fixes: #12366 (#12368)
* [7bb40ef2a](https://github.com/argoproj/argo-workflows/commit/7bb40ef2ad7b8619355484300aeb2008b5a08bf1) chore(deps): bump upload and download artifact to v4 (#12384)
* [5b4b8f600](https://github.com/argoproj/argo-workflows/commit/5b4b8f6007f4b70dd2aeafabc8be5a3f79674ec7) fix: resolve output artifact of steps from expression when it refers … (#12320)
* [d900f074b](https://github.com/argoproj/argo-workflows/commit/d900f074b6f7f3bc06c1d7ca620c9ba42f45d14d) chore(deps): bump github.com/coreos/go-oidc/v3 from 3.8.0 to 3.9.0 (#12380)
* [6471bc002](https://github.com/argoproj/argo-workflows/commit/6471bc0028ac45140482bd1b0e67657ab27d8c06) chore(deps): bump github.com/minio/minio-go/v7 from 7.0.65 to 7.0.66 (#12379)
* [255f50367](https://github.com/argoproj/argo-workflows/commit/255f50367ebf51fada5f92dfdc690acb9a678284) chore(deps): bump cloud.google.com/go/storage from 1.35.1 to 1.36.0 (#12378)
* [99f49d364](https://github.com/argoproj/argo-workflows/commit/99f49d3644979a0b7e57e6366d46a099dc3f8a16) chore(deps): bump google.golang.org/api from 0.153.0 to 0.154.0 (#12377)
* [ceaffd419](https://github.com/argoproj/argo-workflows/commit/ceaffd419a7185f5b02546e1bba94d15ae5c13ce) chore(deps): bump github.com/Azure/azure-sdk-for-go/sdk/azcore from 1.9.0 to 1.9.1 (#12376)
* [460ec5da1](https://github.com/argoproj/argo-workflows/commit/460ec5da1b87049ad76d83a4db80ce2c53453471) chore(deps): bump monaco-editor from 0.44.0 to 0.45.0 in /ui (#12373)
* [5bcf1af15](https://github.com/argoproj/argo-workflows/commit/5bcf1af1565b2b450ad808c939db26fda4290318) fix: delete pending pod when workflow terminated  (#12196)
* [dee7ec5e1](https://github.com/argoproj/argo-workflows/commit/dee7ec5e190a0ee17185f461db12506a467ad89f) feat: add retry count value of custom metric (#11927)
* [51ed59117](https://github.com/argoproj/argo-workflows/commit/51ed59117bfd2e4d4d561be26fc0924bdca43392) fix: liveness check (healthz) type asserts to wrong type (#12353)
* [84454467b](https://github.com/argoproj/argo-workflows/commit/84454467b271e2d80e1f7aca7c5634bfe5d3b990) feat: add default metric of argo_pod_missing (#11857)
* [80c178cb0](https://github.com/argoproj/argo-workflows/commit/80c178cb04b4c93099165593f93f0c79143d534b) fix: create dir when input path is not exist in oss (#12323)
* [b8e98e06f](https://github.com/argoproj/argo-workflows/commit/b8e98e06fe2d86600fcac917c1c0f123ab165cb6) fix: ensure wftmplLifecycleHook wait for each dag task (#12192)
* [a9dc033d5](https://github.com/argoproj/argo-workflows/commit/a9dc033d5e340ad3bc92ad9df0e083333c8bfa4a) chore(deps): bump google.golang.org/api from 0.152.0 to 0.153.0 (#12348)
* [cd5cf92ed](https://github.com/argoproj/argo-workflows/commit/cd5cf92ed902a62af1bcde34a1a7bc2062c8074a) chore(deps): bump github.com/spf13/viper from 1.17.0 to 1.18.1 (#12347)
* [b90772bfa](https://github.com/argoproj/argo-workflows/commit/b90772bfa1217b5b446f44ca53f4558fdbb766dc) chore(deps): bump github.com/itchyny/gojq from 0.12.13 to 0.12.14 (#12346)
* [975bbc58c](https://github.com/argoproj/argo-workflows/commit/975bbc58ca3d389f2b1ef96af88b0398c8276ca2) chore(deps): bump github.com/coreos/go-oidc/v3 from 3.7.0 to 3.8.0 (#12345)
* [b0d170a9e](https://github.com/argoproj/argo-workflows/commit/b0d170a9e2809e610db648ed3db67823bf850cbb) chore(deps): bump golang.org/x/oauth2 from 0.14.0 to 0.15.0 (#12343)
* [2257c0e7b](https://github.com/argoproj/argo-workflows/commit/2257c0e7bb060fa0fbbbac63f17c74c112379a34) chore(deps): bump github.com/minio/minio-go/v7 from 7.0.64 to 7.0.65 (#12344)
* [70c906497](https://github.com/argoproj/argo-workflows/commit/70c9064975fa6f72ffb5e84fde0e1333208b04f2) chore(deps): bump react-datepicker from 4.23.0 to 4.24.0 in /ui (#12340)
* [d310920a6](https://github.com/argoproj/argo-workflows/commit/d310920a63c859c2bce34531ad23f483ad4e8a9f) chore(deps): bump cronstrue from 2.44.0 to 2.47.0 in /ui (#12338)
* [45f595d11](https://github.com/argoproj/argo-workflows/commit/45f595d1187328deae7e3c5f635bbe5e13e142e0) chore(deps): update nixpkgs to nixos-23.11 (#12335)
* [64ee6ae9b](https://github.com/argoproj/argo-workflows/commit/64ee6ae9b674795c114bd66d5b6bdd45be093e9a) refactor: invert conditionals for less nesting in `includeScriptOutput` (#12146)
* [7eee84d34](https://github.com/argoproj/argo-workflows/commit/7eee84d34c2a9939e4c3006a6a1aa62e49c4cfa7) chore(deps): bump golang.org/x/crypto from 0.15.0 to 0.16.0 (#12292)
* [7bcf9087c](https://github.com/argoproj/argo-workflows/commit/7bcf9087c067fe3e6bcfe22f1cc45e654a29f5cf) fix: return failed instead of success when no container status (#12197)
* [c7413929c](https://github.com/argoproj/argo-workflows/commit/c7413929c6493ccf697d2b23760193877ad6fd84) fix: Changes to workflow semaphore does work #12194 (#12284)
* [d2c415ab5](https://github.com/argoproj/argo-workflows/commit/d2c415ab55e06bfd93737fd63ad19b6b8175062c) fix: allow withItems when hooks are involved (#12281)
* [d682c0fd7](https://github.com/argoproj/argo-workflows/commit/d682c0fd71fe452d24513b03cdde41b90fe071f7) fix: Fix variables not substitue bug when creation failed for the first time.  Fixes  (#11487)
* [544ed9069](https://github.com/argoproj/argo-workflows/commit/544ed9069af6852f496fdd543c6b6233acd67a9b) chore(deps): bump github.com/Azure/azure-sdk-for-go/sdk/azcore from 1.8.0 to 1.9.0 (#12298)
* [1f2de884c](https://github.com/argoproj/argo-workflows/commit/1f2de884c76d4576e9d7d0952fdaf46b56cde912) chore(deps): bump google.golang.org/api from 0.151.0 to 0.152.0 (#12299)
* [aeb1bdda1](https://github.com/argoproj/argo-workflows/commit/aeb1bdda1f97faeef2b3333507bb5389bf485fe7) chore(deps): bump github.com/creack/pty from 1.1.20 to 1.1.21 (#12312)
* [03a753ed6](https://github.com/argoproj/argo-workflows/commit/03a753ed61acff9758645d5b8060b930857c06c9) chore(deps): bump golang.org/x/term from 0.14.0 to 0.15.0 (#12311)
* [6c0697928](https://github.com/argoproj/argo-workflows/commit/6c0697928f92160f84e2fc6581551d26e13c0dc5) chore(deps): bump github.com/gorilla/handlers from 1.5.1 to 1.5.2 (#12294)
* [5da4b5ced](https://github.com/argoproj/argo-workflows/commit/5da4b5ceddd9b2fc2b1cc0cc9dcece712f55ea63) chore(deps): bump github.com/google/go-containerregistry from 0.16.1 to 0.17.0 (#12296)
* [cf870b81c](https://github.com/argoproj/argo-workflows/commit/cf870b81c5552777e17599bf396cfed048258357) chore(deps): upgrade `prettier` from v1.x to v3+  (#12290)
* [a2e7c5a1b](https://github.com/argoproj/argo-workflows/commit/a2e7c5a1bef21552e85ca426ac64a439d8ba7a6c) fix: properly resolve exit handler inputs (fixes #12283) (#12288)
* [498734fbc](https://github.com/argoproj/argo-workflows/commit/498734fbca934d2446ae25472c21941ebd819051) fix: missing Object Value when Unmarshaling Plugin struct. Fixes #12202 (#12285)
* [62732b30a](https://github.com/argoproj/argo-workflows/commit/62732b30a3724dad702352160cdf2a9e55a922c6) fix: completed workflow tracking (#12198)
* [bb29d6ab9](https://github.com/argoproj/argo-workflows/commit/bb29d6ab91a2f1acadb0cfd20e9c49ff899709e8) fix: Add identifiable user agent in API client. Fixes #11996 (#12276)
* [cc99dc1bc](https://github.com/argoproj/argo-workflows/commit/cc99dc1bc108efaf0825985f7cbb95e089c91f99) fix: remove deprecated function rand.Seed (#12271)
* [d7b49c865](https://github.com/argoproj/argo-workflows/commit/d7b49c865d60c288ef3fbab3f82fb715d7d2d8a2) refactor(deps): migrate from deprecated `tslint` to `eslint` (#12163)
* [9feda45fc](https://github.com/argoproj/argo-workflows/commit/9feda45fcc83e598c2a63b0d23e39fc5eb99a79a) chore(deps): bump tj-actions/changed-files from 39 to 40 (#12090)
* [f028ac2b7](https://github.com/argoproj/argo-workflows/commit/f028ac2b709dcb7937254685dbcc947e5a448690) chore(deps): bump golang.org/x/crypto from 0.14.0 to 0.15.0 (#12265)
* [c71005ba2](https://github.com/argoproj/argo-workflows/commit/c71005ba2c5418f65760837d280644fb7dab235e) chore(deps): bump google.golang.org/api from 0.149.0 to 0.151.0 (#12262)
* [5cdc8cace](https://github.com/argoproj/argo-workflows/commit/5cdc8cacee72b1979060db2e84715f43e37bf43b) chore(deps): bump github.com/minio/minio-go/v7 from 7.0.63 to 7.0.64 (#12267)
* [a9ecb1199](https://github.com/argoproj/argo-workflows/commit/a9ecb119955a3dac1538f2bacf1e60c5d0a832c0) chore(deps): bump github.com/antonmedv/expr from 1.15.3 to 1.15.5 (#12263)
* [489263f3b](https://github.com/argoproj/argo-workflows/commit/489263f3b622bf87ae4e483a2311e9ecd9a0cac8) chore(deps): bump github.com/upper/db/v4 from 4.6.0 to 4.7.0 (#12260)
* [a3bcd068a](https://github.com/argoproj/argo-workflows/commit/a3bcd068a0b3f2d0d66f3244e13b91e69724fb97) chore(deps): bump cloud.google.com/go/storage from 1.34.1 to 1.35.1 (#12266)
* [2f640f6e5](https://github.com/argoproj/argo-workflows/commit/2f640f6e559e7551a6402e4598928465d22dc22a) chore(deps): bump react-datepicker from 4.21.0 to 4.23.0 in /ui (#12259)
* [b69fc556b](https://github.com/argoproj/argo-workflows/commit/b69fc556b12fc1f9d1ee12a162df82c05187ce63) chore(deps): bump github.com/TwiN/go-color from 1.4.0 to 1.4.1 (#11567)
* [c93913e8a](https://github.com/argoproj/argo-workflows/commit/c93913e8a0dcdfc4f970b438f7c96f5f14595ca7) chore(deps): bump sigs.k8s.io/yaml from 1.3.0 to 1.4.0 (#12092)
* [d7d04616b](https://github.com/argoproj/argo-workflows/commit/d7d04616bf683dcb5af74ca76ecdacbbbd51b346) refactor(ui): consistent imports: `React.use` -> `use` (#12098)
* [453f85ff9](https://github.com/argoproj/argo-workflows/commit/453f85ff926c151b873bf1b88c173cc6bb567d2f) refactor(ui): code-split gigantic Monaco Editor dep (#12150)
* [9eadf2def](https://github.com/argoproj/argo-workflows/commit/9eadf2deff7c291cb458cc4800ada422b671349d) fix: leak stream (#12193)
* [75d7eb190](https://github.com/argoproj/argo-workflows/commit/75d7eb1906179bad8c4e59a8b358374deef5b21f) chore(deps): bump github.com/aliyun/credentials-go from 1.3.1 to 1.3.2 (#12227)
* [63f53eae9](https://github.com/argoproj/argo-workflows/commit/63f53eae9e697af92b89364c0443dd9a9bf9dfbc) chore(deps): bump github.com/gorilla/websocket from 1.5.0 to 1.5.1 (#12226)
* [b6590027a](https://github.com/argoproj/argo-workflows/commit/b6590027ae30bac6c0cb0826eb19311f4b396423) chore(deps): bump golang.org/x/term from 0.13.0 to 0.14.0 (#12225)
* [bce96ab6d](https://github.com/argoproj/argo-workflows/commit/bce96ab6d1c2315c229201bdbaa1592bbfefc4b3) chore(deps): bump cronstrue from 2.41.0 to 2.44.0 in /ui (#12224)
* [206c901c6](https://github.com/argoproj/argo-workflows/commit/206c901c6ff122ad06a32820763e504cc5caf190) fix: Fix for missing steps in the UI (#12203)
* [939ce4029](https://github.com/argoproj/argo-workflows/commit/939ce4029f642c03534ac9710813ecef2153fdf6) fix(server): allow passing loglevels as env vars to Server (#12145)
* [ad5ac52a0](https://github.com/argoproj/argo-workflows/commit/ad5ac52a07138a944a65a58a479f1bb9244861bf) feat: implement IsDirectory for OSS (#12188)
* [222d53cdf](https://github.com/argoproj/argo-workflows/commit/222d53cdfbffb962763689f3d0ba6ac2814e32d0) fix: Resource version incorrectly overridden for wfInformer list requests. Fixes #11948 (#12133)
* [f95a5fc6b](https://github.com/argoproj/argo-workflows/commit/f95a5fc6b9d7c2501691f3c785f43861bd0af1b6) fix: retry S3 on RequestError. Fixes #9914 (#12191)
* [6805c9132](https://github.com/argoproj/argo-workflows/commit/6805c9132c88674bc6913c58e33af3695d75d946) fix: ArtifactGC Fails for Stopped Workflows. Fixes #11879 (#11947)
* [c95d9a1ba](https://github.com/argoproj/argo-workflows/commit/c95d9a1ba55090563f43ec97615a3d06ee117e82) chore(deps): bump golang.org/x/sync from 0.4.0 to 0.5.0 (#12185)
* [877a2e737](https://github.com/argoproj/argo-workflows/commit/877a2e7371720e3e66ae00c5e9f57a23094fc732) chore(deps): bump github.com/go-jose/go-jose/v3 from 3.0.0 to 3.0.1 (#12184)
* [460b30a5c](https://github.com/argoproj/argo-workflows/commit/460b30a5c821515650b478a0abc058abbf0d04e6) chore(deps): bump golang.org/x/time from 0.3.0 to 0.4.0 (#12186)
* [873bacbaf](https://github.com/argoproj/argo-workflows/commit/873bacbafc4da79df15e0da728501ad925eef4fc) fix(ui): Cost Opt should only apply to live Workflows (#12170)
* [249768c9d](https://github.com/argoproj/argo-workflows/commit/249768c9d6e3d0f30d53e92b248b0939822ccfa6) refactor(ui): replace `moment-timezone` with native `Intl` (#12097)
* [a47e46208](https://github.com/argoproj/argo-workflows/commit/a47e46208cd0029ef2e35019207d0b1ee4f85ea4) chore(deps): bump github.com/aliyun/aliyun-oss-go-sdk from 2.2.9+incompatible to 3.0.1+incompatible (#12140)
* [69b47b957](https://github.com/argoproj/argo-workflows/commit/69b47b9571eb713be3cdaf3ae878bea3d86e605c) chore(deps): bump monaco-editor from 0.43.0 to 0.44.0 in /ui (#12142)
* [664569bb1](https://github.com/argoproj/argo-workflows/commit/664569bb1526cb2c23debf638944dd39d9c7521c) chore(deps): bump cronstrue from 2.32.0 to 2.41.0 in /ui (#12144)
* [77173a57f](https://github.com/argoproj/argo-workflows/commit/77173a57fbadfc7d859bcd1696ddaa7a501528c6) chore(deps): bump github.com/creack/pty from 1.1.18 to 1.1.20 (#12139)
* [789c60888](https://github.com/argoproj/argo-workflows/commit/789c60888cb394d8fceb16343e2891c30636d0e8) chore(deps): bump cloud.google.com/go/storage from 1.33.0 to 1.34.1 (#12138)
* [c751e6605](https://github.com/argoproj/argo-workflows/commit/c751e66050a7c16d84ecdfdfe0f7ef59d6ec84c0) fix: regression in memoization without outputs (#12130)
* [4d062d1f2](https://github.com/argoproj/argo-workflows/commit/4d062d1f24bf5168ce19687d0bfdd5d60761c193) refactor(ui): WorkflowsToolbar component from class to functional (#12046)
* [f200826d1](https://github.com/argoproj/argo-workflows/commit/f200826d1ef832cb641e40ef6735eae69f6bbff1) fix: Upgrade axios to v1.6.0. Fixes #12085 (#12111)
* [2f0094c56](https://github.com/argoproj/argo-workflows/commit/2f0094c56ebf22ec80f61843f3ea54d9fc410100) chore(deps): bump react-datepicker and @types/react-datepicker in /ui (#12096)
* [fa15743ab](https://github.com/argoproj/argo-workflows/commit/fa15743aba499a4b0be40e94130d4d1eb59f3add) fix: oss list bucket return all records (#12084)
* [2abc16f6c](https://github.com/argoproj/argo-workflows/commit/2abc16f6c08432d19fce8988a4f41a728fb2333d) fix: conflicting type of "workflow" logging attribute (#12083)
* [08096fc05](https://github.com/argoproj/argo-workflows/commit/08096fc0512ed57a89e4a95ced56512631d8c94b) fix: Revert #11761 to avoid argo-server performance issue (#12068)
* [5896f7561](https://github.com/argoproj/argo-workflows/commit/5896f7561b0ea3c9e4e556df68d9a702ed247881) refactor(ui): use named functions for better tracing (#12062)
* [8f0910842](https://github.com/argoproj/argo-workflows/commit/8f0910842856eee3ad7950d0a9c141218d21ea54) chore(deps): upgrade `swagger-ui-react` to latest 4.x.x (#12058)
* [7567775f3](https://github.com/argoproj/argo-workflows/commit/7567775f3f462099ff0936ac073f39ba1c577585) fix(ui): remove accidentally rendered semi-colon (#12060)
* [03a6168da](https://github.com/argoproj/argo-workflows/commit/03a6168da30717eba3a7e585832841350724e90d) chore(deps): bump github.com/evanphx/json-patch from 5.6.0+incompatible to 5.7.0+incompatible (#11868)
* [5cc8484a5](https://github.com/argoproj/argo-workflows/commit/5cc8484a5b6d85f9a687c08780250f05f6b844e0) chore(deps): bump google.golang.org/api from 0.147.0 to 0.148.0 (#12051)
* [7a25c05dc](https://github.com/argoproj/argo-workflows/commit/7a25c05dc454c9b2a3c528acd6e85fb6ae3680de) chore(deps): bump github.com/coreos/go-oidc/v3 from 3.5.0 to 3.7.0 (#12050)
* [35f7208ec](https://github.com/argoproj/argo-workflows/commit/35f7208ec248c0ba7bf065e0b1a8983a23a7bf7b) chore(deps): automatically `audit fix` UI deps (#12036)
* [e5d8c5357](https://github.com/argoproj/argo-workflows/commit/e5d8c53575923b4fb0f0d3a64614388248645e4d) fix: suppress error about unable to obtain node (#12020)
* [af41c1bac](https://github.com/argoproj/argo-workflows/commit/af41c1bac768cdaef18f843b907b4dbca735f624) chore(deps): use official versions of `bufpipe` and `expr` (#12033)
* [bbb4807e4](https://github.com/argoproj/argo-workflows/commit/bbb4807e4a713e3276550300ae171503d5b87e09) fix: Fixed workflow onexit condition skipped when retry. Fixes #11884 (#12019)
* [5d5c08916](https://github.com/argoproj/argo-workflows/commit/5d5c089167b9b1fd045619b8b06de4dbe798a554) feat: Fall back to retrieve logs from live pods (#12024)
* [d1928e8c5](https://github.com/argoproj/argo-workflows/commit/d1928e8c514cf1a03c364daba78ee74dc34babd6) fix: remove WorkflowSpec VolumeClaimTemplates patch key (#11662)
* [05fa1cba7](https://github.com/argoproj/argo-workflows/commit/05fa1cba7f02be309a06fc105e7a7b93315dea4c) fix: Fix the Maximum Recursion Depth prompt link in the CLI. (#12015)
* [a04d05508](https://github.com/argoproj/argo-workflows/commit/a04d0550876c8940da98a7f902092958b6e6cae1) fix: retry only proper node (#11589) (#11839)
* [32d31ab16](https://github.com/argoproj/argo-workflows/commit/32d31ab16f8164f487f24fdd08f52f7ca346642d) chore(deps): bump google.golang.org/api from 0.143.0 to 0.147.0 (#12001)
* [b1cecff7c](https://github.com/argoproj/argo-workflows/commit/b1cecff7c75c2adf8a41bef54402bd7dccfcd79a) chore(deps): bump react-datepicker and @types/react-datepicker in /ui (#12004)
* [5f7a2cbee](https://github.com/argoproj/argo-workflows/commit/5f7a2cbee42b88eceec387bec84d597786f2beb4) chore(deps): bump github.com/Azure/azure-sdk-for-go/sdk/azidentity from 1.3.1 to 1.4.0 (#12003)
* [2227c9f7e](https://github.com/argoproj/argo-workflows/commit/2227c9f7e33d8f760b86131217f4e81f8860924d) chore(deps): bump golang.org/x/oauth2 from 0.12.0 to 0.13.0 (#12000)
* [67c5fc915](https://github.com/argoproj/argo-workflows/commit/67c5fc915183beb272650b8d140f4810b571706d) fix(ui): don't show pagination warning on first page if all are displayed (#11979)
* [ac105483d](https://github.com/argoproj/argo-workflows/commit/ac105483d55fe3461149e7b885a8a97fb3db60d5) refactor(ui): converting a drop-down component to a functional component (#11901)

<details><summary><h3>Contributors</h3></summary>

* Adrien Delannoy
* Alan Clucas
* AlbeeSo
* Alex
* AloysAqemia
* Anastasiia Kozlova
* Andrei Shevchenko
* Andrew Fenner
* Anton Gilgur
* Baris Erdem
* Blake Pettersson
* Bryce-Huang
* Carlos Santana
* Chris Dolan
* Daan Seynaeve
* Darko Janjic
* David Gamba
* David Pollack
* Dennis Lawler
* Denys Melnyk
* Dillen Padhiar
* Doug Goldstein
* Eduardo Rodrigues
* Garett MacGowan
* GhangZh
* Gongpu Zhu
* Greg Sheremeta
* Harrison Kim
* Helge Willum Thingvad
* Ian Ensor
* Injun Baeg
* Isitha Subasinghe
* James Kang
* Janghun Lee(James)
* Jason Meridth
* Jellyfrog
* Jiacheng Xu
* Joe Bowbeer
* João Pedro
* Julie Vogelman
* Justice
* Kavish Dahekar
* Krunal2017
* Mason Malone
* Matt Fisher
* Matt Menzenski
* Meng Zhuo
* Michael Weibel
* Miltiadis Alexis
* Nagy Attila Gábor
* Oliver Dain
* Omer Levi Hevroni
* Paolo Quadri
* Paul Greenberg
* Phil Brown
* Raffael
* Raymond
* Roel Arents
* Ruin09
* Ryan Currah
* Rémi Cresson
* Sahil Sharma
* Serg Shalavin
* Shabeeb Khalid
* Shiwei Tang
* Shubham
* Shunsuke Suzuki
* Sion Kang
* Sn0rt
* Son Bui
* Takumi Sue
* Tal Yitzhak
* Thor K. Høgås
* Tianchu Zhao
* Tim Collins
* Travis Stevens
* Vasily Chekalkin
* Weidong Cai
* William Van Hevelingen
* Xiaofan Hu
* Yang Lu
* Yuan (Terry) Tang
* Yuan Tang
* Yulin Li
* YunCow
* Yuping Fan
* Zubair Haque
* albertoclarit
* chengjoey
* chenrui
* crazeteam
* dependabot[bot]
* github-actions[bot]
* guangwu
* gussan
* happyso
* heidongxianhua
* instauro
* itayvolo
* ivancili
* jiangjiang
* jingkai
* jswxstw
* leesungbin
* linzhengen
* lukashankeln
* mahdi alizadeh
* moonyoung
* neosu
* origxm
* panicboat
* polarbear567
* redismongo
* renovate[bot]
* rnathuji
* sakai-ast
* sh.yoon
* shangchengbabaiban
* shuangkun tian
* spaced
* static-moonlight
* sycured
* vatine
* williamburgson
* ​Andrzej Ressel
* 刘达
* 名白

</details>

## v3.5.13 (2024-12-02)

Full Changelog: [v3.5.12...v3.5.13](https://github.com/argoproj/argo-workflows/compare/v3.5.12...v3.5.13)

### Selected Changes

* [06c761b8c](https://github.com/argoproj/argo-workflows/commit/06c761b8cc993aa6ab60f8c35c3c95bb334f3da0) Merge commit from fork

<details><summary><h3>Contributors</h3></summary>

* Alan Clucas
* Anton Gilgur

</details>

## v3.5.12 (2024-10-30)

Full Changelog: [v3.5.11...v3.5.12](https://github.com/argoproj/argo-workflows/compare/v3.5.11...v3.5.12)

### Selected Changes

* [8fe8de2e1](https://github.com/argoproj/argo-workflows/commit/8fe8de2e16ec39a5477df17586a3d212ec63a4bd) fix: mark taskresult complete when failed or error. Fixes #12993, Fixes #13533 (#13798)
* [70c58a6bc](https://github.com/argoproj/argo-workflows/commit/70c58a6bc9fa019adf8f54f4458d829dd147f7f6) fix: don't mount SA token when `automountServiceAccountToken: false`. Fixes #12848 (#13820)
* [c68f5cea7](https://github.com/argoproj/argo-workflows/commit/c68f5cea7821728759c0921e449b11e8f0b05abb) fix: better error message for multiple workflow controllers running (#13760)
* [5e4da8178](https://github.com/argoproj/argo-workflows/commit/5e4da8178da407e40762b7802f941cc4f01a31f8) fix(ui): clarify log deletion in log-viewer. Fixes #10993 (#13788)
* [5524cc2ae](https://github.com/argoproj/argo-workflows/commit/5524cc2aee2f68f63b0a92df8b581073515fa51c) fix: only set `ARGO_PROGRESS_FILE` when needed. Partial fix for #13089 (#13743)
* [1daeebb7a](https://github.com/argoproj/argo-workflows/commit/1daeebb7a4e0c63db3feb8b35584fda4b45a9d94) fix(controller): retry transient error on agent pod creation (#13655)
* [816fe8448](https://github.com/argoproj/argo-workflows/commit/816fe8448713edc75dfae86a62f2c374db5fcedd) fix(ui): allow `links` to metadata with dots. Fixes #11741 (#13752)
* [63991d371](https://github.com/argoproj/argo-workflows/commit/63991d3719ff4aa11f59142fc38576c04ca68132) fix(test): fix http-template test (#13737)
* [64d6832e1](https://github.com/argoproj/argo-workflows/commit/64d6832e1c75e4123440f9ea5928fd2e97709239) fix(cli): handle multi-resource yaml in offline lint. Fixes #12137 (#13531)
* [70cdcada9](https://github.com/argoproj/argo-workflows/commit/70cdcada91c5f35f49dc9491f306e57c24e5f38f) fix(emissary): signal SIGINT/SIGTERM in windows correctly (#13693)
* [7826fa147](https://github.com/argoproj/argo-workflows/commit/7826fa147e427b756a0debdc34e80bfa9bc00e56) fix(test): windows tests fixes. Fixes #11994 (#12071)
* [16d5d9d41](https://github.com/argoproj/argo-workflows/commit/16d5d9d416faf2b2a3d1c0ca4ff30246b301c11f) fix: skip clear message when node transition from pending to fail. Fixes #13200 (#13201)
* [c8693f093](https://github.com/argoproj/argo-workflows/commit/c8693f093d5fa08d6a6aad1463692cd33d3c3c2b) fix: all written artifacts should be saved and garbage collected (#13678)
* [5940b8bc1](https://github.com/argoproj/argo-workflows/commit/5940b8bc149522bfe4e8be3af717c0fa66122b90) fix: add `cronWorkflowWorkers` log. Fixes: #13681 (#13688)
* [0f05e6438](https://github.com/argoproj/argo-workflows/commit/0f05e64383fc6afb26a510c5307a9afe7b828146) fix: add retry for invalid connection. Fixes #13578 (#13580)
* [4e23c7d29](https://github.com/argoproj/argo-workflows/commit/4e23c7d296402cd6b805281f1243206cb2f30e43) fix(docs): remove accidental copy+paste from previous commit

<details><summary><h3>Contributors</h3></summary>

* Alan Clucas
* Anton Gilgur
* Greg Sheremeta
* Isitha Subasinghe
* Julie Vogelman
* Michael Weibel
* MinyiZ
* Tianchu Zhao
* Yuping Fan
* github-actions[bot]
* shuangkun tian
* tooptoop4
* wayne

</details>

## v3.5.11 (2024-09-20)

Full Changelog: [v3.5.10...v3.5.11](https://github.com/argoproj/argo-workflows/compare/v3.5.10...v3.5.11)

### Selected Changes

* [a84edd93c](https://github.com/argoproj/argo-workflows/commit/a84edd93c7aafb6a5a9c91a6ee01eb7f11541633) fix(semaphore): ensure `holderKey`s carry all information needed. Fixes #8684 (#13553)
* [dc004a754](https://github.com/argoproj/argo-workflows/commit/dc004a754b1cfa572e74847ff5d755a8657a98bc) fix(controller): handle `nil` `processedTmpl` in DAGs (#13548)
* [fe06e94f5](https://github.com/argoproj/argo-workflows/commit/fe06e94f5d6048a2673b147ac9b3e3f8b484af48) fix(test): it is possible to get 1Tb of RAM on a node (#13606)
* [e5618575e](https://github.com/argoproj/argo-workflows/commit/e5618575ed135b678f3995fe020d291e97237cd1) fix: remove non-transient logs on missing `artifact-repositories` configmap (#13516)
* [f4f7dbded](https://github.com/argoproj/argo-workflows/commit/f4f7dbdede8840f6f7af7f262708d69476654c7c) fix(api): optimise archived list query. Fixes #13295 (#13566)
* [be77948c0](https://github.com/argoproj/argo-workflows/commit/be77948c063019460b5763c9f619f064fcc25e5e) fix(api): `deleteDelayDuration` should be a string (#13543)
* [7f191a496](https://github.com/argoproj/argo-workflows/commit/7f191a496ca1d96161146ff9e0488a19d8c5d3bb) refactor(ci): move api knowledge to the matrix (#13569)
* [1da8a57de](https://github.com/argoproj/argo-workflows/commit/1da8a57de6fdf29efe0ba420e52f14198185c874) fix: ignore error when input artifacts optional. Fixes:#13564 (#13567)
* [9ff8815e6](https://github.com/argoproj/argo-workflows/commit/9ff8815e6d16914b0844c3dd71eed632ae4221b8) fix: aggregate JSON output parameters correctly (#13513)
* [18e3caa49](https://github.com/argoproj/argo-workflows/commit/18e3caa494e057b9697382754990ea60c1679543) fix(executor): add executable permission to staged `script` (#12787)
* [b0ee16d4a](https://github.com/argoproj/argo-workflows/commit/b0ee16d4a5ad43d33595085d7ca032c2e25b671a) fix: Mark task result as completed if pod has been deleted for a while. Fixes #13533 (#13537)
* [472843a3c](https://github.com/argoproj/argo-workflows/commit/472843a3c2d4d05a98105ae3dd6f808cd39a8247) fix: don't clean up old offloaded records during save. Fixes: #13220 (#13286)
* [67253e8ab](https://github.com/argoproj/argo-workflows/commit/67253e8ab502a850415345a7945b2ba00f00cb82) fix: Mark taskResult completed if wait container terminated not gracefully. Fixes #13373 (#13491)
* [14200cacc](https://github.com/argoproj/argo-workflows/commit/14200caccaae6dd1059260c47efcf807f5fde5a1) fix(docs): remove unused spaces (#13487)
* [8ebb8e59e](https://github.com/argoproj/argo-workflows/commit/8ebb8e59e54ba7bd9cd8ace38c5f52eca36425ad) fix(docs): use `sh` instead of `bash` with `busybox` (#13508)
* [269b54cbd](https://github.com/argoproj/argo-workflows/commit/269b54cbd1a54341497c0cb301fbdd03502c62ca) fix: Do not reset the root node by default. Fixes #13196 (#13198)
* [77dc368fb](https://github.com/argoproj/argo-workflows/commit/77dc368fb446c4030fae5a0910d18d5fcfc36252) fix(controller): remove ArtifactGC `finalizer` when no artifacts. Fixes #13499 (#13500)
* [877ff5fd3](https://github.com/argoproj/argo-workflows/commit/877ff5fd3fec3a8d5082c2890bad3dea52391c81) fix: Task result comparison is incorrect, leading to inconsistent comparison results. (#13211)
* [3e3da0776](https://github.com/argoproj/argo-workflows/commit/3e3da0776104b834ffd51bcd1b361ccaeec10626)  ci: Remove Synk ignore for vulnerability for jackc/pgx/v4 (#13481)
* [9c2b12d17](https://github.com/argoproj/argo-workflows/commit/9c2b12d170663672aa7ca4dcc6732159d1a02ad9) fix: avoid exit handler nil pointer when missing outputs. Fixes #13445 (#13448)
* [f4c92bcfc](https://github.com/argoproj/argo-workflows/commit/f4c92bcfca9497c635953f61aad7a165ad3672b0) fix: mark node failed if pod absent. Fixes #12993 (#13454)
* [beec612ed](https://github.com/argoproj/argo-workflows/commit/beec612ed339cbc5f9feb338e9423fefa78e9f9e) fix: Set initial progress from pod metadata if exists. Fixes #13057 (#13260)
* [b881cf075](https://github.com/argoproj/argo-workflows/commit/b881cf075ad9931cfe4d6aeddaddb383347da8c6) fix(docs): Provide versioned links to documentation (#13455)
* [8f65f0200](https://github.com/argoproj/argo-workflows/commit/8f65f0200f67215a1242d99ab5c5b553b6445fcb) fix: Only apply execution control to nodes that are not part of exit handler. (#13016)
* [b18f944bd](https://github.com/argoproj/argo-workflows/commit/b18f944bdce10b1b741db58ba61a8a845faa1525) fix(docs): replace outdated `whalesay` image with `busybox` (#13429)
* [50432f23b](https://github.com/argoproj/argo-workflows/commit/50432f23bf09bcb0571f6e3fce23d43e6a36dc30) fix(test): load to stream tmp file counting (#13366)
* [1892cea4f](https://github.com/argoproj/argo-workflows/commit/1892cea4f9f0aa450cf7495d1644a4cc4a5a99bf) fix(build): slightly optimize UI tasks in `kit` `tasks.yaml` (#13350)
* [f197662f2](https://github.com/argoproj/argo-workflows/commit/f197662f286cd8c185152ecf37643832a09586ac) fix: Mark non-fulfilled taskSetNodes error when agent pod failed. Fixes #12703 (#12723)
* [74708c75f](https://github.com/argoproj/argo-workflows/commit/74708c75f4aad3d304a48cc39ee6ff74a0bdd0c0) fix(devcontainer): `chown` regression for `make codegen` (#13375)
* [b6e591eb8](https://github.com/argoproj/argo-workflows/commit/b6e591eb80ddfe81099ae209aebc1c3f86b5c10e) fix: constraint containerType outboundnode boundary. Fixes #12997 (#13048)
* [abcd1c708](https://github.com/argoproj/argo-workflows/commit/abcd1c7080ab47a8eea4a327ed7c63a4d39d45d6) fix(devcontainer): remove `-R` in `chown` (#13348)
* [8393080f4](https://github.com/argoproj/argo-workflows/commit/8393080f444bb3985f7a5384f2ab36d8fcece10f) fix(cli): `argo lint` with strict should report case-sensitive errors. Fixes #13006 (#13250)
* [834ee93d3](https://github.com/argoproj/argo-workflows/commit/834ee93d3eae8e91b1ccd8f285ad5546be6bc2da) fix(devcontainer): expose ports for all services (#13349)
* [58a77ee38](https://github.com/argoproj/argo-workflows/commit/58a77ee380ae015f08ea4b9d40c26fe7181f27d9) fix: provide fallback for 3.4 to 3.5 transition with absent `NodeFlag`. Fixes #12162 (#13504)

<details><summary><h3>Contributors</h3></summary>

* Alan Clucas
* Anton Gilgur
* Isitha Subasinghe
* Mason Malone
* Miltiadis Alexis
* Thor K. Høgås
* Tianchu Zhao
* William Van Hevelingen
* Xiaofan Hu
* Yuan Tang
* chengjoey
* jingkai
* jswxstw
* origxm
* shangchengbabaiban
* shuangkun tian
* vatine
* ​Andrzej Ressel
* 名白

</details>

## v3.5.10 (2024-08-01)

Full Changelog: [v3.5.9...v3.5.10](https://github.com/argoproj/argo-workflows/compare/v3.5.9...v3.5.10)

### Selected Changes

* [258299274](https://github.com/argoproj/argo-workflows/commit/25829927431d9a0f46d17b72ae74aedb8d700884) fix(release): set `$DOCKER_CONFIG` if unset (#13155)
* [c5922a4f8](https://github.com/argoproj/argo-workflows/commit/c5922a4f863edf7cd888a83d6e2bb9c6af435f57) chore(deps): bump github.com/docker/docker from 24.0.9+incompatible to 26.1.4+incompatible in the go_modules group (#13416)
* [72d0d22e6](https://github.com/argoproj/argo-workflows/commit/72d0d22e6254c2871f7f4f3798a362094409064f) fix(ui): import `getTemplateNameFromNode`
* [3ceecb64c](https://github.com/argoproj/argo-workflows/commit/3ceecb64cca2fda4a5f58ae95b2f6bc463f3730f) chore(deps): use `docker/login-action` consistently instead of `Azure/docker-login` (#12791)
* [d49bcebcb](https://github.com/argoproj/argo-workflows/commit/d49bcebcb99f49e71542586a3e20a7f11bf15a2a) chore(deps): upgrade `actions/cache` and `create-pull-request` to Node v20 (#12775)

<details><summary><h3>Contributors</h3></summary>

* Anton Gilgur
* dependabot[bot]
* github-actions[bot]

</details>

## v3.5.9 (2024-07-30)

Full Changelog: [v3.5.8...v3.5.9](https://github.com/argoproj/argo-workflows/compare/v3.5.8...v3.5.9)

### Selected Changes

* [630f8157a](https://github.com/argoproj/argo-workflows/commit/630f8157a7c207a08f7ab4d156c9136b35226a33) fix(ui): hide `Workflow gone` message when workflow is archived (#13308)
* [12871f752](https://github.com/argoproj/argo-workflows/commit/12871f7524e2e973b3d1f214efcdd4b203bf2120) fix: correct pod names for inline templates. Fixes #12895 (#13261)
* [38bb8d3e2](https://github.com/argoproj/argo-workflows/commit/38bb8d3e247c125507f0e03be60595a2b395db3e) fix(ui): Use proper podname for containersets. Fixes #13038 (#13039)
* [b1c51df63](https://github.com/argoproj/argo-workflows/commit/b1c51df6365fa7900ceb13da87ed54bffaf1704d) refactor: simplify `getPodName` and make consistent with back-end (#12964)
* [fbc56d423](https://github.com/argoproj/argo-workflows/commit/fbc56d423d106610f899cd487c3bb4ae10a5e3d8) fix(cli): `argo lint` with strict should report case-sensitive errors. Fixes #13006 (#13250)
* [e64ee2283](https://github.com/argoproj/argo-workflows/commit/e64ee2283fe0835aa9e7d4c16232a91cff22985f) fix: improve get archived workflow query performance during controller estimation. Fixes #13382 (#13394)
* [861ec70cf](https://github.com/argoproj/argo-workflows/commit/861ec70cf404a51164be342cd6d6b1517324585b) fix(server): don't return `undefined` SA NS (#13347)
* [a828b9da0](https://github.com/argoproj/argo-workflows/commit/a828b9da09b1b3543067ef4513b850cd85958e57) fix(resource): don't use `-f` when patch file is provided (#13317)
* [91ef8452d](https://github.com/argoproj/argo-workflows/commit/91ef8452d4f252efaaa54cc9672b149ae2b4b20c) fix(ui): display Bitbucket Server event source icon in event flow. Fixes #13386 (#13387)
* [9bd2c3130](https://github.com/argoproj/argo-workflows/commit/9bd2c3130ff8b5800744915acb54c1279bf29ffa) fix: constraint containerType outboundnode boundary. Fixes #12997 (#13048)
* [84f3ed169](https://github.com/argoproj/argo-workflows/commit/84f3ed169343261aa68c00e33b2b93a10297193c) fix(docs): correct headings in 3.4 upgrade notes (#13351)
* [f19d6d604](https://github.com/argoproj/argo-workflows/commit/f19d6d60462fb23c95324ba924c0972d92465a67) fix: Only cleanup agent pod if exists. Fixes #12659 (#13294)
* [16bfe2c24](https://github.com/argoproj/argo-workflows/commit/16bfe2c24f9885006213010e0fce6d8ba91c5bd0) fix: allow nodes without `taskResultCompletionStatus` (#13332)
* [b79881cfa](https://github.com/argoproj/argo-workflows/commit/b79881cfa38b618b4e54622c5ec4934e598d5982) fix(cli): Ensure `--dry-run` and `--server-dry-run` flags do not create workflows. fixes #12944 (#13183)
* [123a31612](https://github.com/argoproj/argo-workflows/commit/123a31612ae94136f897088253b84d74ba76d5ff) fix: Update modification timestamps on untar. Fixes #12885 (#13172)
* [37f159576](https://github.com/argoproj/argo-workflows/commit/37f159576ea35e2b4a0a7697161cd533ee166cdb) fix(resource): catch fatal `kubectl` errors (#13321)
* [ebaebcd28](https://github.com/argoproj/argo-workflows/commit/ebaebcd282518bedd79cbd93070b6ea33c6113b2) fix(build): bump golang to 1.21.12 in builder image to fix CVEs (#13311)
* [84a5af1b7](https://github.com/argoproj/argo-workflows/commit/84a5af1b7b754d74e4250329dec341ab14161807) fix: allow artifact gc to delete directory. Fixes #12857 (#13091)
* [ecb2b3917](https://github.com/argoproj/argo-workflows/commit/ecb2b3917531691ce5fdd81a130d4d220fe39e5d) fix(docs): clarify CronWorkflow `startingDeadlineSeconds`. Fixes #12971 (#13280)
* [4e711d6ad](https://github.com/argoproj/argo-workflows/commit/4e711d6ad21bff56d7c0bb06825ec9800a49b688) fix: oss internal error should retry. Fixes #13262 (#13263)
* [deca80891](https://github.com/argoproj/argo-workflows/commit/deca80891e19e39272c843b1ad1d3466eb5d5597) fix(ui): parameter descriptions shouldn't disappear on input (#13244)
* [718f8aff9](https://github.com/argoproj/argo-workflows/commit/718f8aff942d9312c89a83853a27a18b91cbc859) fix(server): switch to `JSON_EXTRACT` and `JSON_UNQUOTE` for MySQL/MariaDB. Fixes #13202 (#13203)
* [d79b9ea9a](https://github.com/argoproj/argo-workflows/commit/d79b9ea9a7797d7911fcf658031e38908a5c8c2f) fix: Mark `Pending` pod nodes as `Failed` when shutting down. Fixes #13210 (#13214)
* [5c85fd366](https://github.com/argoproj/argo-workflows/commit/5c85fd36625fb7bbf7d85513a663c181bf8dc5c5) fix: process metrics later in `executeTemplate`. Fixes #13162 (#13163)

<details><summary><h3>Contributors</h3></summary>

* Alan Clucas
* Alex
* Andrew Fenner
* Anton Gilgur
* Dillen Padhiar
* Gongpu Zhu
* Miltiadis Alexis
* Tianchu Zhao
* Yuan Tang
* github-actions[bot]
* instauro
* jswxstw
* linzhengen
* sh.yoon
* shuangkun tian
* spaced
* 名白

</details>

## v3.5.8 (2024-06-17)

Full Changelog: [v3.5.7...v3.5.8](https://github.com/argoproj/argo-workflows/compare/v3.5.7...v3.5.8)

### Selected Changes

* [d13891154](https://github.com/argoproj/argo-workflows/commit/d1389115484f52d22d1cdcae29139518cbf2fc03) fix(deps): bump `github.com/Azure/azure-sdk-for-go/sdk/azidentity` from 1.5.1 to 1.6.0 to fix CVE (#13197)
* [10488d655](https://github.com/argoproj/argo-workflows/commit/10488d655a78c28bb6e3e6bca490a5496addd605) fix: don't necessarily include all artifacts from templates in node outputs (#13066)
* [c2204ae03](https://github.com/argoproj/argo-workflows/commit/c2204ae03de97acf1c589c898180bdb9942f1524) fix(server): don't use cluster scope list + watch in namespaced mode. Fixes #13177 (#13189)
* [9481bb04c](https://github.com/argoproj/argo-workflows/commit/9481bb04c3e48a85da5ba05ef47c2f0a2ba500f4) fix(server): mutex calls to sqlitex (#13166)
* [ee150afdf](https://github.com/argoproj/argo-workflows/commit/ee150afdf3561f8250c5212e1b6a38628a847b39) fix: only evaluate retry expression for fail/error node. Fixes #13058 (#13165)
* [028f9ec41](https://github.com/argoproj/argo-workflows/commit/028f9ec41cf07056bfcf823a109964b00621797c) fix: Merge templateDefaults into dag task tmpl. Fixes #12821 (#12833)
* [e8f0cae39](https://github.com/argoproj/argo-workflows/commit/e8f0cae398e8f135a6957cd74919368e0b692b6b) fix: Apply podSpecPatch  in `woc.execWf.Spec` and template to pod sequentially (#12476)
* [c1a5f3073](https://github.com/argoproj/argo-workflows/commit/c1a5f3073c58033dcfba5d14fe3dff9092ab258d) fix: don't fail workflow if PDB creation fails (#13102)
* [5c56a161c](https://github.com/argoproj/argo-workflows/commit/5c56a161cb66b1c83fc31e5238bb812bc35f9754) fix: Allow termination of workflow to update on exit handler nodes. fixes #13052 (#13120)
* [e5dfe5d73](https://github.com/argoproj/argo-workflows/commit/e5dfe5d7393c04efc0e4067a02a37aae79231a64) fix: load missing fields for archived workflows (#13136)
* [7dc7fc246](https://github.com/argoproj/argo-workflows/commit/7dc7fc246393295a53308df1b77c585d5b24fe07) fix(ui): `package.json#license` should be Apache (#13040)
* [3622a896d](https://github.com/argoproj/argo-workflows/commit/3622a896d08599e0d325e739c9f389c399419f7d) fix(docs): Fix `gcloud` typo (#13101)

<details><summary><h3>Contributors</h3></summary>

* Alan Clucas
* Anton Gilgur
* Janghun Lee(James)
* Jason Meridth
* Jiacheng Xu
* Julie Vogelman
* Miltiadis Alexis
* Oliver Dain
* Tianchu Zhao
* Travis Stevens
* Yulin Li
* github-actions[bot]
* jswxstw
* leesungbin

</details>

## v3.5.7 (2024-05-27)

Full Changelog: [v3.5.6...v3.5.7](https://github.com/argoproj/argo-workflows/compare/v3.5.6...v3.5.7)

### Selected Changes

* [b2b1ecd7d](https://github.com/argoproj/argo-workflows/commit/b2b1ecd7de378cec31ab0ebb1e8b9665c4b05867) chore(deps): bump tj-actions/changed-files from 40 to 41 (#12433)
* [27a283ac5](https://github.com/argoproj/argo-workflows/commit/27a283ac5db1651ecb6f59c9f693bd9be1ab6fac) fix(ui): show container logs when using `templateRef` (#12973)
* [d2ff152eb](https://github.com/argoproj/argo-workflows/commit/d2ff152ebcb2692ff198031a19d34d78db5fb0e4) fix: Enable realtime metric gc after its workflow is completed. Fixes #12790 (#12830)
* [433bbace9](https://github.com/argoproj/argo-workflows/commit/433bbace9364f6da3478961ba13da0d94b41b2f3) fix: delete skipped node when resubmit with memoized.Fixes: #12936 (#12940)
* [ca947f392](https://github.com/argoproj/argo-workflows/commit/ca947f3920042e04f2c979733258f196e7a3dc53) fix: nodeAntiAffinity is not working as expected when boundaryID is empty. Fixes: #9193 (#12701)
* [210f1f9b2](https://github.com/argoproj/argo-workflows/commit/210f1f9b2d97b6fd240bcd77a84d193b8489ef88) fix: ignore retry node when check succeeded descendant nodes. Fixes: #13003 (#13004)
* [e0925c961](https://github.com/argoproj/argo-workflows/commit/e0925c96100ea7fc85510e8a0f3dce4d5c5b9f7d) fix: setBucketLifecycleRule error in OSS Artifact Driver.  Fixes #12925 (#12926)
* [c26f2da8e](https://github.com/argoproj/argo-workflows/commit/c26f2da8e765f3a0a06e7a2890c327c1ba9497bb) fix(docs): Clarify quick start installation. Fixes #13032  (#13047)
* [a6fec41f7](https://github.com/argoproj/argo-workflows/commit/a6fec41f7f57a2dc6e2904e71c38591a9c371352) feat: add sqlite-based memory store for live workflows. Fixes #12025 (#13021)
* [e103f6bcc](https://github.com/argoproj/argo-workflows/commit/e103f6bcc0c27f4d841261fe781e63946445ef14) fix: don't rebuild `ui/dist/app/index.html` in `argocli-build` stage (#13023)
* [c18b1d00c](https://github.com/argoproj/argo-workflows/commit/c18b1d00c0657ecbeaf56527cb944b30bcdd6f18) fix: use argocli image from pull request in CI (#13018)
* [50dc580ba](https://github.com/argoproj/argo-workflows/commit/50dc580ba0172427d15517dd7aa8454e35a25857) feat: add sqlite-based memory store for live workflows. Fixes #12025 (#12736)
* [db3b1a2ae](https://github.com/argoproj/argo-workflows/commit/db3b1a2aeb2b62547e11f7b695ef8cf908b7f9f6) fix: retry large archived wf. Fixes #12740 (#12741)
* [32c3e030f](https://github.com/argoproj/argo-workflows/commit/32c3e030f9840c21ee17ed26f6f171f945876f3a) fix: `insecureSkipVerify` for `GetUserInfoGroups` (#12982)
* [27a3159e8](https://github.com/argoproj/argo-workflows/commit/27a3159e808ab2cf2ba2a0cce1b8a2a67ca07def) fix: use GetTemplateFromNode to determine template name (#12970)
* [3e8de5d4e](https://github.com/argoproj/argo-workflows/commit/3e8de5d4e9635c881167b496f215a2777cbaaf5d) fix(ui): try retrieving live workflow first then archived (#12972)
* [47f920bfc](https://github.com/argoproj/argo-workflows/commit/47f920bfc0fa5bda684ba02b02d4e516e418933a) fix(ui): remove unnecessary hard reload after delete (#12930)
* [8d2bebc68](https://github.com/argoproj/argo-workflows/commit/8d2bebc682d31726cad2fa0642c0cd4ec0034f5f) fix(ui): use router navigation instead of page load after submit (#12950)
* [437044c24](https://github.com/argoproj/argo-workflows/commit/437044c24f11e6640562f36b432a42c0c9ea179a) fix(ui): handle non-existent labels in `isWorkflowInCluster` (#12898)
* [3f3ac8ea6](https://github.com/argoproj/argo-workflows/commit/3f3ac8ea696b3a41c88d5fd5cc1bf6eeb47abbcc) fix(ui): properly get archive logs when no node was chosen (#12932)

<details><summary><h3>Contributors</h3></summary>

* Alan Clucas
* AlbeeSo
* Anastasiia Kozlova
* Anton Gilgur
* Chris Dolan
* David Gamba
* Doug Goldstein
* Greg Sheremeta
* Jellyfrog
* Jiacheng Xu
* Krunal2017
* Matt Fisher
* Matt Menzenski
* Phil Brown
* Ryan Currah
* Serg Shalavin
* Shabeeb Khalid
* Shubham
* Tim Collins
* Yang Lu
* Yuan (Terry) Tang
* Yuan Tang
* dependabot[bot]
* github-actions[bot]
* heidongxianhua
* itayvolo
* jswxstw
* rnathuji
* shuangkun tian
* sycured

</details>

## v3.5.6 (2024-04-19)

Full Changelog: [v3.5.5...v3.5.6](https://github.com/argoproj/argo-workflows/compare/v3.5.5...v3.5.6)

### Selected Changes

* [200f4d1e5](https://github.com/argoproj/argo-workflows/commit/200f4d1e5ffee0a57a9e7a9995b95da15230eb97) fix: don't load entire archived workflow into memory in list APIs (#12912)
* [fe5c6128c](https://github.com/argoproj/argo-workflows/commit/fe5c6128c6535a636995958c2b44c699c2540be5) fix(ui): default to `main` container name in event source logs API call (#12939)
* [06e6a0df7](https://github.com/argoproj/argo-workflows/commit/06e6a0df7b56b442e5b21071b2584cd593cea9d3) fix(build): close `pkg/apiclient/_.secondary.swagger.json` (#12942)
* [909fdaa98](https://github.com/argoproj/argo-workflows/commit/909fdaa987014e527fbb4f487bce283d682b9854) fix: correct order in artifactGC error log message (#12935)
* [ab7bee7b0](https://github.com/argoproj/argo-workflows/commit/ab7bee7b05fb61b293b89ad4f9f2b1a137b93e84) fix: workflows that are retrying should not be deleted (Fixes #12636) (#12905)
* [9c2581ad0](https://github.com/argoproj/argo-workflows/commit/9c2581ad0f0f83a6fd1754a9fdad9e846a9bc39f) fix: change fatal to panic.  (#12931)
* [01f843828](https://github.com/argoproj/argo-workflows/commit/01f843828b92911581e90dcd3a7d0299a79add9c) fix: Correct log level for agent containers (#12929)
* [30f2e0d93](https://github.com/argoproj/argo-workflows/commit/30f2e0d93cbaaf06a64e70d9cde6648b2ce41f6b) fix: DAG with continueOn in error after retry. Fixes: #11395 (#12817)
* [1c1f43313](https://github.com/argoproj/argo-workflows/commit/1c1f43313578ece6648c1dd7c93d94596b7a4302) fix: use multipart upload method to put files larger than 5Gi to OSS. Fixes #12877 (#12897)
* [8c9a85761](https://github.com/argoproj/argo-workflows/commit/8c9a85761db22284b103f1d500cc9336e95b9766) fix: remove completed taskset status before update workflow. Fixes: #12832 (#12835)
* [ce7cad34b](https://github.com/argoproj/argo-workflows/commit/ce7cad34bca3540a196b56d9b4492bab6cd70d3a) fix: make sure Finalizers has chance to be removed. Fixes: #12836 (#12831)
* [5d03f838c](https://github.com/argoproj/argo-workflows/commit/5d03f838c418272be33eb0abc52d5fbbb271a6ff) fix(test): wait enough time to Trigger Running Hook. Fixes: #12844 (#12855)
* [3d0648893](https://github.com/argoproj/argo-workflows/commit/3d064889300bb323af1c81cc5bcf61c2a65ebcfa) fix: filter hook node to find the correct lastNode. Fixes: #12109 (#12815)
* [c9dd50d35](https://github.com/argoproj/argo-workflows/commit/c9dd50d35b87086421e0e24ccbb481591f6f9425) fix: terminate workflow should not get throttled Fixes #12778 (#12792)
* [faaddf3ac](https://github.com/argoproj/argo-workflows/commit/faaddf3acc2bc82b02600701af5076adebbdf0d2) fix(containerSet): mark container deleted when pod deleted. Fixes: #12210 (#12756)
* [4e7d471c0](https://github.com/argoproj/argo-workflows/commit/4e7d471c0d3ae856ff22056739147b52ea3ba5fc) fix: return itself when getOutboundNodes from memoization Hit steps/DAG. Fixes: #7873 (#12780)
* [519faf03c](https://github.com/argoproj/argo-workflows/commit/519faf03c6df81fa2c34269cb2a3a0fc119a433f) fix: pass dnsconfig to agent pod. Fixes: #12824 (#12825)
* [56d7b2b9c](https://github.com/argoproj/argo-workflows/commit/56d7b2b9c6844d7cb1e69d8711c9322221e2f911) fix: inline template loops should receive more than the first item. Fixes: #12594 (#12628)
* [19a7edebb](https://github.com/argoproj/argo-workflows/commit/19a7edebbb4524e409e0e9f4225f1bf6b0073312) fix: workflow stuck in running state when using activeDeadlineSeconds on template level. Fixes: #12329 (#12761)
* [68c089d49](https://github.com/argoproj/argo-workflows/commit/68c089d49346d72e16017353bcf54d32d1d8b165) fix: ensure workflowtaskresults complete before mark workflow completed status. Fixes: #12615 (#12574)
* [b189afa48](https://github.com/argoproj/argo-workflows/commit/b189afa48d2824cd419fe5db23c55e6204020e49) fix: patch report outputs completed if task result not exists. (#12748)
* [eec6ae0e4](https://github.com/argoproj/argo-workflows/commit/eec6ae0e4dcfd721f2f706e796279b378653438f) fix(log): change task set to task result. (#12749)
* [a20f69571](https://github.com/argoproj/argo-workflows/commit/a20f69571f4cef97b353f8b3a80cd1161b80274d) chore(deps): upgrade `mkdocs-material` from 8.2.6 to 9.x (#12894)
* [c956d70ee](https://github.com/argoproj/argo-workflows/commit/c956d70eead3cedf2f8c1422c028e26fe4b45683) fix(hack): various fixes & improvements to cherry-pick script (#12714)
* [1c09db42e](https://github.com/argoproj/argo-workflows/commit/1c09db42ec69540ec64e5dd60a6daef3473c6783) fix(deps): upgrade x/net to v0.23.0. Fixes CVE-2023-45288 (#12921)
* [1c3401dc6](https://github.com/argoproj/argo-workflows/commit/1c3401dc68236979fc26b35c787256fcb96a7d1f) fix(deps): upgrade `http2` to v0.24. Fixes CVE-2023-45288 (#12901)
* [ddf815fb2](https://github.com/argoproj/argo-workflows/commit/ddf815fb2885b7c207177e211349a6e1a169aec3) chore(deps): bump cloud.google.com/go/storage from 1.35.1 to 1.36.0 (#12378)
* [bc42b0881](https://github.com/argoproj/argo-workflows/commit/bc42b08812d193242522a14964829c7a1bf362a6) chore(deps): bump github.com/Azure/azure-sdk-for-go/sdk/azcore from 1.9.0 to 1.9.1 (#12376)
* [ec84a61c6](https://github.com/argoproj/argo-workflows/commit/ec84a61c6e337b012dcce1a21b7298d07ec3526e) chore(deps): bump github.com/Azure/azure-sdk-for-go/sdk/azcore from 1.8.0 to 1.9.0 (#12298)
* [a1643357c](https://github.com/argoproj/argo-workflows/commit/a1643357c235a84d6838331dc8df7c1d83d58abe) refactor(build): simplify `mkdocs build` scripts (#12463)
* [c8082b6fc](https://github.com/argoproj/argo-workflows/commit/c8082b6fc386408e73063d1ad0402510445fa94c) fix(deps): upgrade `crypto` from v0.20 to v0.22. Fixes CVE-2023-42818 (#12900)
* [4fb03eef9](https://github.com/argoproj/argo-workflows/commit/4fb03eef988d6d7824d6620fca5a75524039e2de) chore(deps): bump `undici` from 5.28.3 to 5.28.4 in /ui (#12891)
* [4ce9e02d3](https://github.com/argoproj/argo-workflows/commit/4ce9e02d382992855269b8381d6bcaec44bdd1cd) chore(deps): bump `follow-redirects` from 1.15.4 to 1.15.6 due to CVE
* [20c81f8a5](https://github.com/argoproj/argo-workflows/commit/20c81f8a522ac8c238b5ec5c35d5596688771643) build(deps): bump github.com/go-jose/go-jose/v3 from 3.0.1 to 3.0.3 (#12879)
* [ceef27bf2](https://github.com/argoproj/argo-workflows/commit/ceef27bf2bb7594ccdaca64c693cf3149baf2be3) build(deps): bump github.com/docker/docker from 24.0.0+incompatible to 24.0.9+incompatible (#12878)
* [8fcadffc1](https://github.com/argoproj/argo-workflows/commit/8fcadffc1cc25461c8ff6cf68f5430c8b494d726) fix(deps): upgrade `pgx` from 4.18.1 to 4.18.2 due to CVE (#12753)
* [43630bd8e](https://github.com/argoproj/argo-workflows/commit/43630bd8ec1207ee882295f47ba682aed8dde534) chore(deps): upgrade Cosign to v2.2.3 (#12850)
* [6d41e8cfa](https://github.com/argoproj/argo-workflows/commit/6d41e8cfa90940d570fe428e3e3fc039d77cd012) fix(deps): upgrade `undici` to 5.28.3 due to CVE (#12763)
* [1f39d328d](https://github.com/argoproj/argo-workflows/commit/1f39d328df494296ef929c6cdac7d5a344fbafe3) chore(deps): bump google.golang.org/protobuf to 1.33.0 to fix CVE-2024-24786 (#12846)
* [c353b0921](https://github.com/argoproj/argo-workflows/commit/c353b092198007f495ce14405fed25914a88a5b8) chore(deps): bump github.com/creack/pty from 1.1.20 to 1.1.21 (#12312)
* [d95791fdf](https://github.com/argoproj/argo-workflows/commit/d95791fdf94f728690e89284df4da7373af6012b) fix: mark task result completed use nodeId instead of podname. Fixes: #12733 (#12755)
* [03f9f7583](https://github.com/argoproj/argo-workflows/commit/03f9f75832dd3dc4aca14b7d40b7e8c22f4e26fd) fix(ui): show correct podGC message for deleteDelayDuration. Fixes: #12395 (#12784)

<details><summary><h3>Contributors</h3></summary>

* AlbeeSo
* Andrei Shevchenko
* Anton Gilgur
* Jiacheng Xu
* Roel Arents
* Shiwei Tang
* Shunsuke Suzuki
* Tianchu Zhao
* Yuan (Terry) Tang
* Yuan Tang
* Yulin Li
* dependabot[bot]
* github-actions[bot]
* guangwu
* shuangkun tian
* static-moonlight

</details>

## v3.5.5 (2024-02-29)

Full Changelog: [v3.5.4...v3.5.5](https://github.com/argoproj/argo-workflows/compare/v3.5.4...v3.5.5)

### Selected Changes

* [6af917eb3](https://github.com/argoproj/argo-workflows/commit/6af917eb322bb84a2733723433a9eb87b7f1e85d) chore(deps): bump github.com/cloudflare/circl to 1.3.7 to fix GHSA-9763-4f94-gfch (#12556)
* [6ee52fc96](https://github.com/argoproj/argo-workflows/commit/6ee52fc96e700190de96a15993b933a26f0389c9) fix: make WF global parameters available in retries (#12698)
* [c2905bda5](https://github.com/argoproj/argo-workflows/commit/c2905bda5c9962fa64474a39a6e0c9b0a842e8c2) chore(deps): fixed medium CVE in github.com/docker/docker v24.0.0+incompatible (#12635)
* [dd8b4705b](https://github.com/argoproj/argo-workflows/commit/dd8b4705bdc3e3207e70eba70af7f72fb812cd3d) fix: documentation links (#12446)
* [72deab92a](https://github.com/argoproj/argo-workflows/commit/72deab92a5dec7b8df87109fb54398509ce24639) fix(docs): render Mermaid diagrams in docs (#12464)
* [9a4c787e7](https://github.com/argoproj/argo-workflows/commit/9a4c787e71e57edfe8a554a2f8f922cbe530430c) fix(docs): exclude `docs/requirements.txt` from docs build (#12466)
* [ae915fe9f](https://github.com/argoproj/argo-workflows/commit/ae915fe9ffae19fc721a790b7611a2428a23c845) fix(docs): handle `fields` examples with `md_in_html` (#12465)
* [a4674b9a1](https://github.com/argoproj/argo-workflows/commit/a4674b9a193451ad8379bd0c55604232c181abea) fix: merge env bug in workflow-controller-configmap and container. Fixes #12424 (#12426)
* [eb71bad60](https://github.com/argoproj/argo-workflows/commit/eb71bad60321fcdb5638471cf21ac67fb8a98a2a) fix: Add missing 'archived' prop for ArtifactPanel component. Fixes #12331 (#12397)
* [288eddcfe](https://github.com/argoproj/argo-workflows/commit/288eddcfeb34d53b14c72f698007c48e9afe7906) fix: wrong values are assigned to input parameters of workflowtemplat… (#12412)
* [c425aa0ee](https://github.com/argoproj/argo-workflows/commit/c425aa0ee572a39ead178add6357595cd4c20a07) fix(docs): remove `workflow-controller-configmap.yaml` self reference (#12654)
* [88332d4c3](https://github.com/argoproj/argo-workflows/commit/88332d4c37f34a71b5adbd4e9d720ff4645864dd) fix: upgrade expr-lang. Fixes #12037 (#12573)
* [a98027078](https://github.com/argoproj/argo-workflows/commit/a98027078fdd98113644b9d3e6833e79ecc57d2f) fix: make sure taskresult completed when mark node succeed when it has outputs (#12537)
* [901cfb636](https://github.com/argoproj/argo-workflows/commit/901cfb63632903b59b0f6858e813b85a104cb486) fix: controller option to not watch configmap (#12622)
* [a5bf99690](https://github.com/argoproj/argo-workflows/commit/a5bf99690c8b8189c439f2775685108e84a9cd02) fix: make etcd errors transient (#12567)
* [02a3e2e39](https://github.com/argoproj/argo-workflows/commit/02a3e2e399d90f59b4cb813aa41ad92aca045f03) fix(build): check for env vars in all dirs (#12652)
* [d4d28b5c7](https://github.com/argoproj/argo-workflows/commit/d4d28b5c7cfc7baf8c2180019bdaa3e9b04decc9) fix: SSO with Jumpcloud "email_verified" field #12257 (#12318)
* [16c4970e7](https://github.com/argoproj/argo-workflows/commit/16c4970e78c5f15ced290b7ae7d330e6c6252467) fix: Fixed mutex with withSequence in http template broken. Fixes #12018 (#12176)
* [23b1a4b24](https://github.com/argoproj/argo-workflows/commit/23b1a4b244e3e2ae1169854bf7f90ad60de2b62f) fix: prevent update race in workflow cache (Fixes #9574) (#12233)
* [8e33da1a1](https://github.com/argoproj/argo-workflows/commit/8e33da1a13ac6f8b09e45cac5ff39eab0927f498) fix: add resource quota evaluation timed out to transient (#12536)
* [8c75a72a5](https://github.com/argoproj/argo-workflows/commit/8c75a72a5b15ac39b5cddfed0886d3f76dcf9e3d) fix: cache configmap don't create with workflow has retrystrategy. Fixes: #12490 #10426 (#12491)
* [33521350e](https://github.com/argoproj/argo-workflows/commit/33521350ebd287ca16c7c76df94bb9a492a4dff9) fix: update minio chart repo (#12552)
* [0319b79d5](https://github.com/argoproj/argo-workflows/commit/0319b79d5e13217e86784f92be67524fed3b8af4) fix: Global Artifact Passing. Fixes #12554 (#12559)
* [56a591185](https://github.com/argoproj/argo-workflows/commit/56a59118541d79be7c4b3ba3feb2a67b4f9c900e) fix(ui): clone the `ListWatch` callback array in `WorkflowsList` (#12562)
* [2a21d1445](https://github.com/argoproj/argo-workflows/commit/2a21d1445df644894f96d0af62d4d7688b93489b) fix: Mark resource && data template report-outputs-completed true (#12544)
* [fcfbfbd0b](https://github.com/argoproj/argo-workflows/commit/fcfbfbd0b5a1251e6cd0cb728131604c613dedc3) fix(resources): improve ressource accounting. Fixes #12468 (#12492)
* [0bffab1dd](https://github.com/argoproj/argo-workflows/commit/0bffab1dd3971ae1c9adbc4a7c2ceb6969098678) fix: Allow valueFrom in dag arguments parameters. Fixes #11900 (#11902)
* [636f79a8b](https://github.com/argoproj/argo-workflows/commit/636f79a8bddea8d021737104bc6d2e4be516e7f4) fix: artifact subdir error when using volumeMount (#12638)
* [93f0b6ebd](https://github.com/argoproj/argo-workflows/commit/93f0b6ebd6757c2f4957cbe151061c7848e68d57) fix: pass through burst and qps for auth.kubeclient (#12575)
* [9b69363ba](https://github.com/argoproj/argo-workflows/commit/9b69363ba62fa76ac994c1d8542904b4fd331d53) fix: retry node with expression status Running -> Pending (#12637)
* [c95c6abc5](https://github.com/argoproj/argo-workflows/commit/c95c6abc510a42dbae2bb8e929589cfb99c811f4) fix(controller): add missing namespace index from workflow informer (#12666)
* [c62e6ad34](https://github.com/argoproj/argo-workflows/commit/c62e6ad34ec5659a391eeb0cf755a3792a21347d) fix(controller): re-allow changing executor `args` (#12609)
* [715791b17](https://github.com/argoproj/argo-workflows/commit/715791b17bc92e3880f14fffea020ecb5af44d85) fix(ui): `ListWatch` should not _both_ set and depend on `nextOffset` (#12672)
* [8207a0890](https://github.com/argoproj/argo-workflows/commit/8207a08900b9e7433d5ae939c44a08c065db5f7b) fix(typo): fix some typo (#12673)
* [ea753f097](https://github.com/argoproj/argo-workflows/commit/ea753f097db03eb057bb54e78d9a8f45b1d924d8) fix: Patch taskset with subresources to delete completed node status.… (#12620)
* [3d4a2cbd6](https://github.com/argoproj/argo-workflows/commit/3d4a2cbd6d7d4a0829d7f6ef8e46788c6e244489) fix: Add limit to number of Workflows in CronWorkflow history (#12681)
* [32918ba55](https://github.com/argoproj/argo-workflows/commit/32918ba5532c8044d3a12c5baf3fb6f696b71bb6) fix: find correct retry node when using `templateRef`. Fixes: #12633 (#12683)

<details><summary><h3>Contributors</h3></summary>

* Alan Clucas
* AloysAqemia
* Anton Gilgur
* Dennis Lawler
* Eduardo Rodrigues
* Garett MacGowan
* Isitha Subasinghe
* Jason Meridth
* João Pedro
* Paolo Quadri
* Raffael
* Ruin09
* Ryan Currah
* Son Bui
* Tal Yitzhak
* Tianchu Zhao
* Yulin Li
* jiangjiang
* jswxstw
* panicboat
* shuangkun tian

</details>

## v3.5.4 (2024-01-13)

Full Changelog: [v3.5.3...v3.5.4](https://github.com/argoproj/argo-workflows/compare/v3.5.3...v3.5.4)

### Selected Changes

* [960af331a](https://github.com/argoproj/argo-workflows/commit/960af331a8c0a3f2e263c8b90f1daf4303816ba8) fix: autolink dep in yarn.lock is incorrect
* [ec7d1f698](https://github.com/argoproj/argo-workflows/commit/ec7d1f698360242dd28f6be5b227c415da63d473) fix: Resolve vulnerabilities in axios (#12470)
* [f5fee5661](https://github.com/argoproj/argo-workflows/commit/f5fee5661b29441e5dae78e44d8b6fc05ffd6565) fix: Switch to upstream go-git. Fixes CVE-2023-49569 (#12515)

<details><summary><h3>Contributors</h3></summary>

* Anton Gilgur
* Yuan Tang

</details>

## v3.5.3 (2024-01-10)

Full Changelog: [v3.5.2...v3.5.3](https://github.com/argoproj/argo-workflows/compare/v3.5.2...v3.5.3)

### Selected Changes

* [46efafea3](https://github.com/argoproj/argo-workflows/commit/46efafea3fbd1ed26ceb92948caf7f9fde1cfa41) chore(deps): bump tj-actions/changed-files from 39 to 40 (#12090)
* [5dcb08928](https://github.com/argoproj/argo-workflows/commit/5dcb08928d491839c37186f1f665d35be2d7b752) chore(deps): bump google.golang.org/api from 0.149.0 to 0.151.0 (#12262)
* [5e8d30181](https://github.com/argoproj/argo-workflows/commit/5e8d3018175acef2f8774554e8d7fbabac1e0fbd) chore(deps): bump github.com/antonmedv/expr from 1.15.3 to 1.15.5 (#12263)
* [5ac12e8e2](https://github.com/argoproj/argo-workflows/commit/5ac12e8e29ed08594b926d131b19432f542caf0c) chore(deps): bump github.com/upper/db/v4 from 4.6.0 to 4.7.0 (#12260)
* [f92b39c69](https://github.com/argoproj/argo-workflows/commit/f92b39c69da4676b1e3a878fd6b64a19feeb43c8) chore(deps): bump cloud.google.com/go/storage from 1.34.1 to 1.35.1 (#12266)
* [2019c8d43](https://github.com/argoproj/argo-workflows/commit/2019c8d434e741dc362cc6e26427727cd356809d) chore(deps): bump react-datepicker from 4.21.0 to 4.23.0 in /ui (#12259)
* [b606eda2f](https://github.com/argoproj/argo-workflows/commit/b606eda2f4f787d3519181b6d94ad7f9bd609d6b) chore(deps): bump sigs.k8s.io/yaml from 1.3.0 to 1.4.0 (#12092)
* [d172b3b9b](https://github.com/argoproj/argo-workflows/commit/d172b3b9b1ec500edd5f86ca4a910cb31daf97cd) chore(deps): bump github.com/aliyun/credentials-go from 1.3.1 to 1.3.2 (#12227)
* [0547738a4](https://github.com/argoproj/argo-workflows/commit/0547738a41420f10792ebc7163d0186311ab9841) chore(deps): bump cronstrue from 2.41.0 to 2.44.0 in /ui (#12224)
* [fcf2f6f5b](https://github.com/argoproj/argo-workflows/commit/fcf2f6f5bf22c41ddf48bb8b1108922c26bb214a) chore(deps): bump golang.org/x/sync from 0.4.0 to 0.5.0 (#12185)
* [6ec24a1bd](https://github.com/argoproj/argo-workflows/commit/6ec24a1bdbee4afa2f38d4bb83752bb9a21a7dc2) chore(deps): bump golang.org/x/time from 0.3.0 to 0.4.0 (#12186)
* [29325d143](https://github.com/argoproj/argo-workflows/commit/29325d143b695a61d67c09b3178c02ab362dd29e) chore(deps): bump monaco-editor from 0.43.0 to 0.44.0 in /ui (#12142)
* [360e37785](https://github.com/argoproj/argo-workflows/commit/360e37785a62fe7b4626c89c71a7dca9078d0b44) chore(deps): bump cloud.google.com/go/storage from 1.33.0 to 1.34.1 (#12138)
* [5e9325dc6](https://github.com/argoproj/argo-workflows/commit/5e9325dc65a4f42486a8adf3352e5e64158239cb) chore(deps): bump react-datepicker and @types/react-datepicker in /ui (#12096)
* [9b3951c38](https://github.com/argoproj/argo-workflows/commit/9b3951c3870d04ddd4a3c5af81cac9188ab0e512) chore(deps): upgrade `swagger-ui-react` to latest 4.x.x (#12058)
* [3cf8ae22f](https://github.com/argoproj/argo-workflows/commit/3cf8ae22ff1858eab2044e7df73adfef4ed595cb) chore(deps): bump google.golang.org/api from 0.147.0 to 0.148.0 (#12051)
* [2b561638c](https://github.com/argoproj/argo-workflows/commit/2b561638c8e137fbbb15dcc046c4b1f74d19b16b) chore(deps): bump github.com/coreos/go-oidc/v3 from 3.5.0 to 3.7.0 (#12050)
* [e1cbeedd5](https://github.com/argoproj/argo-workflows/commit/e1cbeedd52e7ac2afa99e58abc188d8553f1e710) chore(deps): automatically `audit fix` UI deps (#12036)
* [70dc1b4ac](https://github.com/argoproj/argo-workflows/commit/70dc1b4ac46d5f0958893b6ecc8cf19c238fda04) chore(deps): bump google.golang.org/api from 0.143.0 to 0.147.0 (#12001)
* [0b48ece51](https://github.com/argoproj/argo-workflows/commit/0b48ece51a46cd1cf30eafc6a9a9e94845671799) fix: Resolve lint issues in UI code
* [6330c0a02](https://github.com/argoproj/argo-workflows/commit/6330c0a02aa46d74daba9e950386449d0390c0db) chore(deps): bump golang.org/x/crypto from 0.14.0 to 0.15.0 (#12265)
* [9ae27831e](https://github.com/argoproj/argo-workflows/commit/9ae27831e7914726cf774ce28da97371ee468269) chore(deps): bump github.com/gorilla/handlers from 1.5.1 to 1.5.2 (#12294)
* [13b69e719](https://github.com/argoproj/argo-workflows/commit/13b69e719998c0f64d69807eb85b90e1690175a5) chore(deps): update nixpkgs to nixos-23.11 (#12335)
* [3631e9cdf](https://github.com/argoproj/argo-workflows/commit/3631e9cdfa5095fbf6723da0adbd564ebcbaafc5) chore(deps): bump github.com/minio/minio-go/v7 from 7.0.64 to 7.0.65 (#12344)
* [d325af186](https://github.com/argoproj/argo-workflows/commit/d325af1867fff47d0fa62cda7e6a3b904956cc04) chore(deps): bump github.com/itchyny/gojq from 0.12.13 to 0.12.14 (#12346)
* [4fbe64c6d](https://github.com/argoproj/argo-workflows/commit/4fbe64c6d858db250bca74ecbaa0ceda113b2fd6) chore(deps): bump monaco-editor from 0.44.0 to 0.45.0 in /ui (#12373)
* [1bdfff03b](https://github.com/argoproj/argo-workflows/commit/1bdfff03bb6586b787b57e9b2cda1b910a71db9b) chore(deps): bump upload and download artifact to v4 (#12384)
* [cc881166e](https://github.com/argoproj/argo-workflows/commit/cc881166e60137891eaa39905b958d3344659e1c) fix: resolve output artifact of steps from expression when it refers … (#12320)
* [5568a2536](https://github.com/argoproj/argo-workflows/commit/5568a2536dae57406005af08837df8a83cee5d5d) fix: fix missing artifacts for stopped workflows. Fixes #12401 (#12402)
* [852f8a35a](https://github.com/argoproj/argo-workflows/commit/852f8a35a22eec40f039e14173b42fd8e75f115d) fix: remove deprecated function rand.Seed (#12271)
* [35b8b4094](https://github.com/argoproj/argo-workflows/commit/35b8b40942cbc4c7277024a7477ab153eaea1525) fix: Add identifiable user agent in API client. Fixes #11996 (#12276)
* [3ecfe56f5](https://github.com/argoproj/argo-workflows/commit/3ecfe56f50180935f7e621c6e37f1596298a6996) fix: completed workflow tracking (#12198)
* [c4251fa5b](https://github.com/argoproj/argo-workflows/commit/c4251fa5b54be3ce77c7551fc8f78c024f895347) fix: missing Object Value when Unmarshaling Plugin struct. Fixes #12202 (#12285)
* [c1c4936ec](https://github.com/argoproj/argo-workflows/commit/c1c4936ecd7d1fbd722d28e4a59e8b5eff784566) fix: properly resolve exit handler inputs (fixes #12283) (#12288)
* [b998c50d9](https://github.com/argoproj/argo-workflows/commit/b998c50d94ca377e0760264c1c66bd1435fd8bc8) fix: Fix variables not substitue bug when creation failed for the first time.  Fixes  (#11487)
* [29e613e84](https://github.com/argoproj/argo-workflows/commit/29e613e84997f2b742f0c86a826d733226183e20) fix: allow withItems when hooks are involved (#12281)
* [c6702d595](https://github.com/argoproj/argo-workflows/commit/c6702d595a6f052a46e20f8e7ae07ec27dee7559) fix: Changes to workflow semaphore does work #12194 (#12284)
* [8bcf64669](https://github.com/argoproj/argo-workflows/commit/8bcf6466999330546abbafb8e114f8a6c7ee7f06) fix: return failed instead of success when no container status (#12197)
* [1b17b7ad1](https://github.com/argoproj/argo-workflows/commit/1b17b7ad184af4e11ecc1af48290bede2fb90324) fix: ensure wftmplLifecycleHook wait for each dag task (#12192)
* [35ba1c1eb](https://github.com/argoproj/argo-workflows/commit/35ba1c1eb9781716b6b7db2426893e8e37e210be) fix: create dir when input path is not exist in oss (#12323)
* [00719cfeb](https://github.com/argoproj/argo-workflows/commit/00719cfebc30d54fbeb339c0692cf468d9804db4) fix: liveness check (healthz) type asserts to wrong type (#12353)
* [bfb15dae3](https://github.com/argoproj/argo-workflows/commit/bfb15dae310ecf869ce8e43718977391a02a40c9) fix: delete pending pod when workflow terminated  (#12196)
* [b89b16115](https://github.com/argoproj/argo-workflows/commit/b89b16115009da847704032b1ef25eec43dfd68b) fix: move log with potential sensitive data to debug loglevel. Fixes: #12366 (#12368)
* [4cce92063](https://github.com/argoproj/argo-workflows/commit/4cce9206356935234c3cc3f10a41c7ccf9f66356) fix: custom columns not supporting annotations (#12421)
* [aaf919269](https://github.com/argoproj/argo-workflows/commit/aaf919269db09b92733658c4c679bd18c3a5cea1) fix: ensure workflow wait for onExit hook for DAG template (#11880) (#12436)
* [e5d86ed8e](https://github.com/argoproj/argo-workflows/commit/e5d86ed8e045c64f9337575a32ff9d2d367927c6) fix: Apply workflow level PodSpecPatch in agent pod. Fixes #12387 (#12440)
* [299bc169a](https://github.com/argoproj/argo-workflows/commit/299bc169a9af56342a56899cbcfcfe03252ffb8b) fix: CI Artifact Download Timeout. Fixes #12452 (#12454)
* [e8cc7152b](https://github.com/argoproj/argo-workflows/commit/e8cc7152ba15fe2f308ebae586debee0cd8c5cec) fix: http template host header rewrite(#12385) (#12386)
* [5c0ecde28](https://github.com/argoproj/argo-workflows/commit/5c0ecde2875220b07918bd658a84731f89ab8cc5) fix(docs): release-3.5 readthedocs backport (#12475)

<details><summary><h3>Contributors</h3></summary>

* Alan Clucas
* Anton Gilgur
* Bryce-Huang
* Denys Melnyk
* Garett MacGowan
* Jason Meridth
* Julie Vogelman
* Saravanan Balasubramanian
* Son Bui
* Yang Lu
* Yuan (Terry) Tang
* Yuan Tang
* Zubair Haque
* dependabot[bot]
* gussan
* ivancili
* jswxstw
* neosu
* renovate[bot]
* shuangkun tian
* 刘达

</details>

## v3.5.2 (2023-11-27)

Full Changelog: [v3.5.1...v3.5.2](https://github.com/argoproj/argo-workflows/compare/v3.5.1...v3.5.2)

### Selected Changes

* [237addc9d](https://github.com/argoproj/argo-workflows/commit/237addc9dab0f31435e8eb7f98bf254c2d19c480) fix: Update yarn.lock file
* [afd5399cb](https://github.com/argoproj/argo-workflows/commit/afd5399cbd129b267a2d31d278402aa1c06d07c5) fix(ui): Cost Opt should only apply to live Workflows (#12170)
* [c296cf233](https://github.com/argoproj/argo-workflows/commit/c296cf233235e46bd581a0333e0c4e675a5f3e80) fix: ArtifactGC Fails for Stopped Workflows. Fixes #11879 (#11947)
* [82560421a](https://github.com/argoproj/argo-workflows/commit/82560421aaa4845d3e33dc5f98e69a2dc2495b1d) fix: retry S3 on RequestError. Fixes #9914 (#12191)
* [a69ca2342](https://github.com/argoproj/argo-workflows/commit/a69ca234237145ae3ec15dffe7f510e7dfc70b2b) fix: Resource version incorrectly overridden for wfInformer list requests. Fixes #11948 (#12133)
* [1faa1e62e](https://github.com/argoproj/argo-workflows/commit/1faa1e62eb67512cab96a0b435eef640c10947fe) fix(server): allow passing loglevels as env vars to Server (#12145)
* [9c378d162](https://github.com/argoproj/argo-workflows/commit/9c378d162f9d244b775d25ede751c7841d64127d) fix: Fix for missing steps in the UI (#12203)
* [59f5409c9](https://github.com/argoproj/argo-workflows/commit/59f5409c95da83d9045fa936b0ec2dbb09e7724b) fix: leak stream (#12193)
* [4b162df16](https://github.com/argoproj/argo-workflows/commit/4b162df16260053d1493e66bcae64689053f03e2) refactor(ui): code-split gigantic Monaco Editor dep (#12150)
* [8615f5364](https://github.com/argoproj/argo-workflows/commit/8615f5364c0f4c3fc7ca35d86d9739e3bd9210b1) refactor(ui): replace `moment-timezone` with native `Intl` (#12097)
* [93b54c5d0](https://github.com/argoproj/argo-workflows/commit/93b54c5d054fe422b758c902999ddc0a6d97066f) chore(deps): bump github.com/creack/pty from 1.1.18 to 1.1.20 (#12139)
* [4558bfc69](https://github.com/argoproj/argo-workflows/commit/4558bfc69deeb94484dd6e5d6c6a2ab4ca5948d5) chore(deps): bump github.com/aliyun/aliyun-oss-go-sdk from 2.2.9+incompatible to 3.0.1+incompatible (#12140)
* [913c71881](https://github.com/argoproj/argo-workflows/commit/913c718812e91d540f0075457bbc895e9edda598) chore(deps): bump github.com/go-jose/go-jose/v3 from 3.0.0 to 3.0.1 (#12184)
* [92923f960](https://github.com/argoproj/argo-workflows/commit/92923f9605318e10b1b2d241365b0c98adc735d9) chore(deps): bump golang.org/x/term from 0.13.0 to 0.14.0 (#12225)
* [67dff4f22](https://github.com/argoproj/argo-workflows/commit/67dff4f22178028b81253f1b239cda2b06ebe9e1) chore(deps): bump github.com/gorilla/websocket from 1.5.0 to 1.5.1 (#12226)
* [a16ba1df8](https://github.com/argoproj/argo-workflows/commit/a16ba1df88303b40e48e480c91854269d4a45d76) chore(deps): bump github.com/TwiN/go-color from 1.4.0 to 1.4.1 (#11567)
* [30b6a91a5](https://github.com/argoproj/argo-workflows/commit/30b6a91a5a04aef3370f36d1ccc39a76834c79a5) chore(deps): bump github.com/minio/minio-go/v7 from 7.0.63 to 7.0.64 (#12267)

<details><summary><h3>Contributors</h3></summary>

* Alan Clucas
* Anton Gilgur
* Garett MacGowan
* Helge Willum Thingvad
* Weidong Cai
* Yuan (Terry) Tang
* Yuan Tang
* dependabot[bot]

</details>

## v3.5.1 (2023-11-03)

Full Changelog: [v3.5.0...v3.5.1](https://github.com/argoproj/argo-workflows/compare/v3.5.0...v3.5.1)

### Selected Changes

* [877c55230](https://github.com/argoproj/argo-workflows/commit/877c5523066e17687856fe3484c9b2d398e986f5) chore(deps): bump golang.org/x/oauth2 from 0.12.0 to 0.13.0 (#12000)
* [2b44c4ad6](https://github.com/argoproj/argo-workflows/commit/2b44c4ad65e5699adf3a2549bf7cb6ae0a0e09ff) chore(deps): bump github.com/Azure/azure-sdk-for-go/sdk/azidentity from 1.3.1 to 1.4.0 (#12003)
* [1a7d9c940](https://github.com/argoproj/argo-workflows/commit/1a7d9c94043b9b1a4a99a317fcdb4e185a8413a3) chore(deps): bump react-datepicker and @types/react-datepicker in /ui (#12004)
* [16dbb6e49](https://github.com/argoproj/argo-workflows/commit/16dbb6e4907f5d675485e651f01acb4d21d679be) chore(deps): use official versions of `bufpipe` and `expr` (#12033)
* [39b8583bd](https://github.com/argoproj/argo-workflows/commit/39b8583bd47c064639c81ada9c6b04b7e3e6ba21) chore(deps): bump github.com/evanphx/json-patch from 5.6.0+incompatible to 5.7.0+incompatible (#11868)
* [9e04496c3](https://github.com/argoproj/argo-workflows/commit/9e04496c3a4f24c1883a2e1fe57a82e2089c8d4f) fix: Upgrade axios to v1.6.0. Fixes #12085 (#12111)
* [0e04980c6](https://github.com/argoproj/argo-workflows/commit/0e04980c670fa7730af1972db21f07ff1ca8ccd4) fix(ui): don't show pagination warning on first page if all are displayed (#11979)
* [98aba1599](https://github.com/argoproj/argo-workflows/commit/98aba159942c8bdf033cfbfc41da6630a5be8358) fix: retry only proper node (#11589) (#11839)
* [d51a87ace](https://github.com/argoproj/argo-workflows/commit/d51a87acef4d0cad0c50adec72eedf2e1c21b3b8) fix: Fix the Maximum Recursion Depth prompt link in the CLI. (#12015)
* [4997ddd7d](https://github.com/argoproj/argo-workflows/commit/4997ddd7d52d95702a07dfa595b38aa7131dca90) fix: remove WorkflowSpec VolumeClaimTemplates patch key (#11662)
* [49fe42088](https://github.com/argoproj/argo-workflows/commit/49fe4208858099aee1295eb6ff8ba7868fbd822f) fix: Fixed workflow onexit condition skipped when retry. Fixes #11884 (#12019)
* [84d15792a](https://github.com/argoproj/argo-workflows/commit/84d15792a631626dcb1cabebcf56215d0c72b844) fix: suppress error about unable to obtain node (#12020)
* [430faf09d](https://github.com/argoproj/argo-workflows/commit/430faf09d3b134746e84bb6705e1a818ecf48405) fix(ui): remove accidentally rendered semi-colon (#12060)
* [2a34dc1a7](https://github.com/argoproj/argo-workflows/commit/2a34dc1a7de2a7e4b8bed61163c7b39241a1f493) fix: Revert #11761 to avoid argo-server performance issue (#12068)
* [7645b98ac](https://github.com/argoproj/argo-workflows/commit/7645b98ac4d259225e55fa6b9ac194efbd78d1f9) fix: conflicting type of "workflow" logging attribute (#12083)
* [90a92215f](https://github.com/argoproj/argo-workflows/commit/90a92215fc43b0cebcd046cd783c3eb237800126) fix: oss list bucket return all records (#12084)
* [8f55b8da7](https://github.com/argoproj/argo-workflows/commit/8f55b8da721e0694aac22dc9a4d12af07b11dcc1) fix: regression in memoization without outputs (#12130)

<details><summary><h3>Contributors</h3></summary>

* Alan Clucas
* Anton Gilgur
* Ruin09
* Takumi Sue
* Vasily Chekalkin
* Yang Lu
* Yuan (Terry) Tang
* Yuan Tang
* dependabot[bot]
* gussan
* happyso
* shuangkun tian

</details>

## v3.5.0 (2023-10-13)

Full Changelog: [v3.5.0-rc2...v3.5.0](https://github.com/argoproj/argo-workflows/compare/v3.5.0-rc2...v3.5.0)

### Selected Changes

* [bf735a2e8](https://github.com/argoproj/argo-workflows/commit/bf735a2e861d6b1c686dd4a076afc3468aa89c4a) fix(windows): prevent infinite run. Fixes #11810 (#11993)
* [375a860b5](https://github.com/argoproj/argo-workflows/commit/375a860b51e22378ca529da77fe3ed1ecb8e6de6) fix: Fix gRPC and HTTP2 high vulnerabilities (#11986)
* [f01dbb1df](https://github.com/argoproj/argo-workflows/commit/f01dbb1df1584c6e5daa288fd6fe7e8416697bd8) fix: Permit enums w/o values. Fixes #11471. (#11736)
* [96d964375](https://github.com/argoproj/argo-workflows/commit/96d964375f19bf376d51aa1907f5a1b4bcea9964) fix(ui): remove "last month" default date filter mention from New Version Modal (#11982)
* [6b0f04794](https://github.com/argoproj/argo-workflows/commit/6b0f0479495182dfb9e6a26689f5a2f3877a5414) fix(ui): faulty `setInterval` -> `setTimeout` in clipboard (#11945)
* [7576abcee](https://github.com/argoproj/argo-workflows/commit/7576abcee2cd7253c2022fc6c4744e325668993b) fix: show pagination warning on all pages (fixes #11968) (#11973)
* [a45afc0c8](https://github.com/argoproj/argo-workflows/commit/a45afc0c87b0ffa52a110c753b97d48f06cdf166) fix: Replace antonmedv/expr with expr-lang/expr (#11971)
* [8fa8f7970](https://github.com/argoproj/argo-workflows/commit/8fa8f7970bfd3ccc5cff1246ea08a7771a03b8ad) chore(deps): bump github.com/Azure/azure-sdk-for-go/sdk/azcore from 1.7.1 to 1.8.0 (#11958)
* [05c6db12a](https://github.com/argoproj/argo-workflows/commit/05c6db12adfd581331f5ae5b0234b72c407e7760) fix(ui): `ClipboardText` tooltip properly positioned (#11946)
* [743d29750](https://github.com/argoproj/argo-workflows/commit/743d29750784810e26ea46f6e87e91f021c583c0) fix(ui): ensure `WorkflowsRow` message is not too long (#11908)
* [26481a214](https://github.com/argoproj/argo-workflows/commit/26481a2146107ad0937ef7698c27f3686f93c81e) refactor(ui): convert `WorkflowsList` + `WorkflowsFilter` to functional components (#11891)
* [bdc536252](https://github.com/argoproj/argo-workflows/commit/bdc536252b1048b9c110b05af31934b9972499bd) chore(deps): bump google.golang.org/api from 0.138.0 to 0.143.0 (#11915)
* [9469a1bf0](https://github.com/argoproj/argo-workflows/commit/9469a1bf049de784d8416c1f37600413d6762972) fix(ui): use `popup.confirm` instead of browser `confirm` (#11907)
* [a363e6a58](https://github.com/argoproj/argo-workflows/commit/a363e6a5875d0b9b9b2ad9c3fc2a0586f2b70f2c) refactor(ui): optimize Link functionality (#11743)
* [14df2e400](https://github.com/argoproj/argo-workflows/commit/14df2e400d529ffa5b43bf55cb70a3cd135ae8e3) refactor(ui): convert ParametersInput to functional components (#11894)
* [68ad03938](https://github.com/argoproj/argo-workflows/commit/68ad03938be929befba48f70d7c8fdae6839f433) refactor(ui): InputFilter and WorkflowTimeline components from class to functional (#11899)
* [e91c2737f](https://github.com/argoproj/argo-workflows/commit/e91c2737f3dff1fee41ce97991e294a57c53fc93) fix: Correctly retry an archived wf even when it exists in the cluster. Fixes #11903 (#11906)
* [c86a5cdb1](https://github.com/argoproj/argo-workflows/commit/c86a5cdb1ec1155e6ed17e67b46d5df59a566b08) fix: Automate nix updates with renovate (#11887)
* [2e4f28142](https://github.com/argoproj/argo-workflows/commit/2e4f281427e5eb8542ff847cb23d7f37808cbb03) refactor(ui): use async/await in several components (#11882)
* [b5f69a882](https://github.com/argoproj/argo-workflows/commit/b5f69a8826609eabc6e11fb477eea3472ba4f91f) fix: Fixed running multiple workflows with mutex and memo concurrently is broken (#11883)
* [b2c6b55fa](https://github.com/argoproj/argo-workflows/commit/b2c6b55fac3de4a8a8d9d12d75332008ab750932) chore(deps): bump golang.org/x/crypto from 0.12.0 to 0.13.0 (#11873)
* [baa65c5c3](https://github.com/argoproj/argo-workflows/commit/baa65c5c34545d5c9144bfd9dbd2d4a355791baf) chore(deps): bump cloud.google.com/go/storage from 1.32.0 to 1.33.0 (#11870)
* [361af5aaf](https://github.com/argoproj/argo-workflows/commit/361af5aaf54c0858ff886346e91b572afcfb7caa) chore(deps): bump github.com/antonmedv/expr from 1.14.0 to 1.15.3 (#11871)
* [24c1c1083](https://github.com/argoproj/argo-workflows/commit/24c1c10838a59f72716fbbe5f476dae390e5288a) chore(deps): bump github.com/aliyun/aliyun-oss-go-sdk from 2.2.8+incompatible to 2.2.9+incompatible (#11866)
* [a83df9721](https://github.com/argoproj/argo-workflows/commit/a83df9721e57f8c15d26a20187e39b6e23645c78) chore(deps): bump golang.org/x/term from 0.11.0 to 0.12.0 (#11869)
* [eae277cbe](https://github.com/argoproj/argo-workflows/commit/eae277cbe8c4ea27a61d316b709176db420baa4b) chore(deps): bump github.com/tidwall/gjson from 1.15.0 to 1.17.0 (#11867)
* [5def5289a](https://github.com/argoproj/argo-workflows/commit/5def5289a6c010265bb9e8a6bfcd6f1bba80624b) feat: show history about completed runs in each cron workflow (#11811)
* [6fbfedf81](https://github.com/argoproj/argo-workflows/commit/6fbfedf8103d78f85010feec0eb9db03136b86d4) refactor(ui): migrate `UserInfo` to functional component (#11793)
* [0fde6800c](https://github.com/argoproj/argo-workflows/commit/0fde6800cbc5d6e2ee6aeb9840079c75fed1d3c3) fix: when key not present assume NodeRunning. Fixes 11843 (#11847)
* [c6fdb0311](https://github.com/argoproj/argo-workflows/commit/c6fdb0311eecf99ed23e21a6062f093441115500) refactor(ui): migrate `Reports` to functional component and split files (#11794)
* [27132d956](https://github.com/argoproj/argo-workflows/commit/27132d9563d1ba80afd3f294eca596f0f942c5d8) refactor(ui): convert a few components to use hooks (#11800)
* [fbe9375d5](https://github.com/argoproj/argo-workflows/commit/fbe9375d5307bb7f3f30770dc36fc48ef34c290e) fix: shouldn't fail to run cronworkflow because previous got shutdown on its own (race condition) (#11845)

<details><summary><h3>Contributors</h3></summary>

* Anton Gilgur
* Isitha Subasinghe
* Julie Vogelman
* Justice
* Matt Farmer
* Michael Weibel
* PranitRout07
* Ruin09
* Sebast1aan
* Takumi Sue
* Tim Collins
* Yuan (Terry) Tang
* Yusuke Shinoda
* dependabot[bot]
* github-actions[bot]
* gussan
* heidongxianhua
* redenferno

</details>

## v3.5.0-rc2 (2023-09-20)

Full Changelog: [v3.5.0-rc1...v3.5.0-rc2](https://github.com/argoproj/argo-workflows/compare/v3.5.0-rc1...v3.5.0-rc2)

### Selected Changes

* [fa116b63e](https://github.com/argoproj/argo-workflows/commit/fa116b63e8aa9ddb6bd985d479b7e65c9b18785f) fix: use same date filter in the UI and CLI (#11840)
* [a6c83de34](https://github.com/argoproj/argo-workflows/commit/a6c83de3462b882496d58416da93989a8814bc33) feat: Support artifact streaming for HTTP/Artifactory artifact driver (#11823)
* [caedd0ff7](https://github.com/argoproj/argo-workflows/commit/caedd0ff7ade8211039f3dc858f74cd4eb2b1818) chore(deps): bump docker/login-action from 2 to 3 (#11827)
* [246d4f440](https://github.com/argoproj/argo-workflows/commit/246d4f44013b545e963106a9c43e9cee397c55f7) feat: Search by name for WorkflowTemplates in UI (#11684)
* [56d1333c9](https://github.com/argoproj/argo-workflows/commit/56d1333c9460072d806397539877768e622ff424) refactor(ui): migrate several components from class to functional (#11791)
* [d33f26741](https://github.com/argoproj/argo-workflows/commit/d33f267413bb4bd712cc8c19087ee1e94db4b8cb) chore(deps): bump docker/build-push-action from 4 to 5 (#11830)
* [ad7515e86](https://github.com/argoproj/argo-workflows/commit/ad7515e86c4c11006c48f14d0f4344b186ba0a9d) chore(deps): bump docker/setup-qemu-action from 2 to 3 (#11829)
* [0246d993e](https://github.com/argoproj/argo-workflows/commit/0246d993e0ffabe762c5a735faf0050a6efcc550) chore(deps): bump docker/setup-buildx-action from 2 to 3 (#11828)
* [803c5cadb](https://github.com/argoproj/argo-workflows/commit/803c5cadb17f9ab9539085aca9035120d3a1072d) fix: add prometheus label validation for realtime gauge metric (#11825)
* [07c256085](https://github.com/argoproj/argo-workflows/commit/07c25608540171f190d211be1a03c05ed139bab0) fix: Fixed workflow template skip whitespaced parameters. Fixes #11767 (#11781)
* [92a30f2b6](https://github.com/argoproj/argo-workflows/commit/92a30f2b60d7fc1ef84cba3eb57630266ad3910c) refactor(ui): workflow panel components from class to functional (#11803)
* [24ab95c31](https://github.com/argoproj/argo-workflows/commit/24ab95c31f3845623f4140bc298a36f6f856c4e8) fix(ui): merge WF List FTU Panel with New Version Modal (#11742)
* [7aedf9733](https://github.com/argoproj/argo-workflows/commit/7aedf973356c8b57510a554b6b759f2684f88839) fix: close response body when request event-stream failed (#11818)
* [55bb51885](https://github.com/argoproj/argo-workflows/commit/55bb51885d2a6690727f97dce25fffef1afb34f2) fix: fix mergeWithArchivedWorkflows test data to match expected (#11816)
* [4591af60e](https://github.com/argoproj/argo-workflows/commit/4591af60eee1d9d8bb36420e74194179ee735e5e) fix: Only append slash when missing for Artifactory repoURL (#11812)
* [2f5af0ab2](https://github.com/argoproj/argo-workflows/commit/2f5af0ab21463aeb250aa6f1a31cca522aec7408) feat: Support keyFormat for Artifactory (#11798)
* [a480f6d9c](https://github.com/argoproj/argo-workflows/commit/a480f6d9c44122c1f9b794e8fc993d8eced22d82) fix: Correct buckets for operation_duration_seconds metric (#11780)
* [6399ac706](https://github.com/argoproj/argo-workflows/commit/6399ac70619ff037793b773d44131c7a1f9e7fb6) feat: Add user info to suspended node when resuming (#11763)
* [e073dccff](https://github.com/argoproj/argo-workflows/commit/e073dccff3be2e5a9eed1b3e7da6e8b5fe09854f) fix: apply custom cursor pagination where workflows and archived workflows are merged (#11761)
* [582eecdf9](https://github.com/argoproj/argo-workflows/commit/582eecdf9a75995dcd28af2ecac9d404315c74ce) chore(deps): bump monaco-editor from 0.41.0 to 0.43.0 in /ui (#11801)
* [0d8c19e19](https://github.com/argoproj/argo-workflows/commit/0d8c19e19caa026dca960c5abac6292920a17b95) chore(deps): bump cronstrue from 2.31.0 to 2.32.0 in /ui (#11785)
* [f9bb71da8](https://github.com/argoproj/argo-workflows/commit/f9bb71da8504cbcda8c8f90463975e0b6a9f0302) feat: document usage of `filterGroupsRegex` (#11778)
* [7e62657be](https://github.com/argoproj/argo-workflows/commit/7e62657beb6873938dd9fd472ea7c425439730f8) fix(ui): handle `undefined` dates in Workflows List filter (#11792)
* [477b3caf4](https://github.com/argoproj/argo-workflows/commit/477b3caf415d1f65f71dd366d9ebc5c04c64c099) feat: filter sso groups based on regex (#11774)
* [1cf39d21e](https://github.com/argoproj/argo-workflows/commit/1cf39d21e42667cec4b3f3941c78cb66b1599ffa) fix: Correct limit in WorkflowTaskResultInformer List API calls. Fixes #11607 (#11722)
* [75bd0b83a](https://github.com/argoproj/argo-workflows/commit/75bd0b83a479997da1940e048d5161b11cecb303) fix: Workflow controller crash on nil pointer  (#11770)
* [53b470192](https://github.com/argoproj/argo-workflows/commit/53b470192c240c4ae90b32defa44ad8b64a13acd) fix(ui): don't use `Buffer` for FNV hash (#11766)
* [297bea618](https://github.com/argoproj/argo-workflows/commit/297bea61888f70d742fd68237a8a2df1b71c7ac1) fix: Argo DB init conflict when deploy workflow-controller with multiple replicas #11177 (#11569)
* [633c5e92a](https://github.com/argoproj/argo-workflows/commit/633c5e92a72e1adc4fc23bc911950ab9fc6d5964) feat: Set a max recursion depth limit (#11646)
* [48697a12b](https://github.com/argoproj/argo-workflows/commit/48697a12ba30ea0214a3d9ce25b665a292828c80) fix(ui): don't use anti-pattern in CheckboxFilter (#11739)
* [9e7dc2592](https://github.com/argoproj/argo-workflows/commit/9e7dc2592f662c6af5488587943dd94b379ce750) fix(ui): don't reload the page until _after_ deletion (#11711)
* [f5e31f8f3](https://github.com/argoproj/argo-workflows/commit/f5e31f8f36b32883087f783cb1227490bbe36bbd) fix: offset reset when pagination limit onchange (#11703)
* [d3cb45130](https://github.com/argoproj/argo-workflows/commit/d3cb451302d59187098295bc76e719232381bb88) fix(workflow): match discovery burst and qps for `kubectl` with upstream kubectl binary (#11603)
* [e90d6bf6b](https://github.com/argoproj/argo-workflows/commit/e90d6bf6b63bd07c7a3a8aa34dd2d356dbaa53ae) fix: Health check from lister not apiserver (#11375)
* [7b72c0d13](https://github.com/argoproj/argo-workflows/commit/7b72c0d13e18705ca9b43385f187d2f494ae5104) chore(deps): update `monaco-editor` to latest 0.41.0 (#11710)
* [18820c333](https://github.com/argoproj/argo-workflows/commit/18820c333fb28595b6a233ed71205037cfedfdf2) fix: Make defaultWorkflow hooks work more than once (#11693)
* [27f1227bf](https://github.com/argoproj/argo-workflows/commit/27f1227bfb62ffa3d99c14e71aa54de3edbfedc3) fix: Add missing new version modal for v3.5 (#11692)
* [74551e3dc](https://github.com/argoproj/argo-workflows/commit/74551e3dcbd0c82eec790249bc445c3ef6c4d89d) ci(deps): dedupe `yarn.lock`, add check for dupes (#11637)
* [d99efa7bc](https://github.com/argoproj/argo-workflows/commit/d99efa7bc2070c9d1f4072881cc95e5158242645) fix: ensure labels is defined before key access. Fixes #11602 (#11638)
* [9cb378342](https://github.com/argoproj/argo-workflows/commit/9cb378342283c9ef9f2f3b999bec7cf10c8aab91) fix: cron workflow initial filter value. Fixes #11685 (#11686)
* [ac9e2de17](https://github.com/argoproj/argo-workflows/commit/ac9e2de1782c8889b6e97890be3aafc8e0c01905) fix: Surface underlying error when getting a workflow (#11674)
* [ba523bf07](https://github.com/argoproj/argo-workflows/commit/ba523bf073df41c1a272176ed3c17ef7f8c08f16) fix: Change node in paramScope to taskNode at executeDAG (#11422) (#11682)
* [bc9b64473](https://github.com/argoproj/argo-workflows/commit/bc9b64473fdaa9b042b01be101332877576c5523) fix: argo logs completion (#11645)
* [cb8dbbcd6](https://github.com/argoproj/argo-workflows/commit/cb8dbbcd621247e0f88e00e8c60992da2744c4b5) fix: Print valid JSON/YAML when workflow list empty #10873 (#11681)
* [11a931388](https://github.com/argoproj/argo-workflows/commit/11a931388617e93242848a95666e63ce6835e5f3) feat: add artgc podspecpatch fixes #11485 (#11586)
* [05e508ecd](https://github.com/argoproj/argo-workflows/commit/05e508ecdc8589ad3c6445edfa8ec4f5f6b7128e) feat: update nix version and build with ldflags (#11505)
* [f18b339b9](https://github.com/argoproj/argo-workflows/commit/f18b339b94916a1dde2eeb01400da425265da94f) fix(ui): Only redirect/reload to wf list page when wf deletion succeeds (#11676)
* [39ff2842f](https://github.com/argoproj/argo-workflows/commit/39ff2842fc20869ae8c0c8a0ea727c1c8954a4be) chore(deps): remove unneeded Yarn resolutions (#11641)
* [12a3313d9](https://github.com/argoproj/argo-workflows/commit/12a3313d90ae8c6bf020d32655fc8dbfa9233a83) chore(deps): remove unused JS deps (#11630)
* [82ac98026](https://github.com/argoproj/argo-workflows/commit/82ac98026994b8b7b1a0486c6f536103d818fa99) fix: Only confirm DB deletion when there are archived workflows. Fixes #11658 (#11659)
* [efb118156](https://github.com/argoproj/argo-workflows/commit/efb11815656532668ba881ad81184e3b1b3a38d6) chore(deps): upgrade `monaco-editor` to 0.30 (#11593)
* [9693c02f8](https://github.com/argoproj/argo-workflows/commit/9693c02f876ee3fcf0359141a8289986c275ec5e) fix: Fixed parent level memoization broken. Fixes #11612 (#11623)
* [9317360f2](https://github.com/argoproj/argo-workflows/commit/9317360f2ef398de232c217dfdf71219b7a2fa41) fix: do not process withParams when task/step Skipped. Fixes #10173 (#11570)
* [363ee6901](https://github.com/argoproj/argo-workflows/commit/363ee690126b6eeb5956ee9804d48758e9b0a0b3) fix: upgrade module for pull image in google cloud issue #9630 (#11614)
* [8a52da5e8](https://github.com/argoproj/argo-workflows/commit/8a52da5e8ee6eeabffb6c7e5858702129b37b525) fix: TERM signal was catched but not handled properly, which causing … (#11582)
* [41809b58a](https://github.com/argoproj/argo-workflows/commit/41809b58a5feb019b28e4ea229cc67acd62b109b) feat(ui): retry workflows with parameter (#10824) (#11632)
* [027b9c990](https://github.com/argoproj/argo-workflows/commit/027b9c990d4f99253cc776b6fd2b86135f56cc6f) fix: override storedWorkflowSpec when override parameter (#11631) (#11634)
* [8d8aa6e17](https://github.com/argoproj/argo-workflows/commit/8d8aa6e1757010190939750fbf7868119bc72454) chore(deps): bump cloud.google.com/go/storage from 1.31.0 to 1.32.0 (#11619)
* [28821902f](https://github.com/argoproj/argo-workflows/commit/28821902fcaa6598941492042143c0a725ee5129) fix: Upgrade Go to v1.21 Fixes #11556 (#11601)
* [c9c6e5ce3](https://github.com/argoproj/argo-workflows/commit/c9c6e5ce3b17e78db04f81c8cdf4525f696d1b11) fix: deprecated Link(Help-Contact) Issue (#11627)
* [524b4cb58](https://github.com/argoproj/argo-workflows/commit/524b4cb58672d07ce2ed9cff3dd0c58bbcf9d293) chore(deps): bump github.com/Azure/azure-sdk-for-go/sdk/azidentity from 1.3.0 to 1.3.1 (#11622)
* [a1a1fdedc](https://github.com/argoproj/argo-workflows/commit/a1a1fdedce9a2da984e28e3d98671e9f5e415f54) chore(deps): bump github.com/google/go-containerregistry from 0.11.0 to 0.16.1 (#11527)
* [463b8fdde](https://github.com/argoproj/argo-workflows/commit/463b8fddeb5bc39e14d49ff9dc3b09c93977476d) chore(deps): bump github.com/minio/minio-go/v7 from 7.0.61 to 7.0.62 (#11618)
* [67af8c4e0](https://github.com/argoproj/argo-workflows/commit/67af8c4e077edaf2cce95b75d4c6d1101c95690f) chore(deps): bump google.golang.org/api from 0.136.0 to 0.138.0 (#11620)
* [23d6da6ca](https://github.com/argoproj/argo-workflows/commit/23d6da6cad03124ebd23eebf0d04be06c1b80c6f) fix: upgrade base image for security and build support arm64 #10435 (#11613)
* [cd2e4e564](https://github.com/argoproj/argo-workflows/commit/cd2e4e564960d6c5f1a772d9d27672a08b3a6bcf) feat: upgrade expr to v1.14 for richer language definition (#11605)
* [27ffa8301](https://github.com/argoproj/argo-workflows/commit/27ffa8301e090983b8287f8ebdcef0df01b6c8a0) feat(cli): Add a flag status to delete cmd like list cmd of argo cli (#11577)
* [bda561ef0](https://github.com/argoproj/argo-workflows/commit/bda561ef0c343a677c696c3cde2ab15bc2d6fc81) refactor: remove nesting during dry-run of `argo delete` (#11596)
* [0423eb6e2](https://github.com/argoproj/argo-workflows/commit/0423eb6e26b5fe0548c1f5d7bcc089e4e996f2f1) fix(ui): ensure `package.json#name` is not the same as `argo-ui` (#11595)
* [e5d237a24](https://github.com/argoproj/argo-workflows/commit/e5d237a2429a4e2cb810a3c0a2ec1d95cc00a714) refactor(ui): simplify Webpack config a bit (#11594)
* [427656e28](https://github.com/argoproj/argo-workflows/commit/427656e28b168fdc8706ca50d025524e57193a9e) chore(deps): bump cron-parser from 4.8.1 to 4.9.0 in /ui (#11592)
* [5eb50f428](https://github.com/argoproj/argo-workflows/commit/5eb50f42897e969995ad86eef764230e3a023641) chore(deps): bump cronstrue from 2.29.0 to 2.31.0 in /ui (#11591)
* [7cef09c3c](https://github.com/argoproj/argo-workflows/commit/7cef09c3c0c3d09fa8f113f14952a796ece3a4bd) chore(deps): bump superagent from 8.0.9 to 8.1.2 in /ui (#11590)
* [6bccc9904](https://github.com/argoproj/argo-workflows/commit/6bccc9904dfdc4eb87d8c600b730d3bf29664339) fix: upgrade `argo-ui` components to latest (#11585)
* [7b80ce19e](https://github.com/argoproj/argo-workflows/commit/7b80ce19e8afe6690aed5c2f3d6c123c812e468b) feat: support custom CA with s3 repository. Fixes #10560 (#11161)

<details><summary><h3>Contributors</h3></summary>

* Alan Clucas
* Alec Rabold
* Ansuman Swain
* Anton Gilgur
* Antonio Gurgel
* Basanth Jenu H B
* Cheng Wang
* Isitha Subasinghe
* Jesse Suen
* Jiwan Ahn
* Julie Vogelman
* Justice
* KBS
* LEE EUI JOO
* Rick
* Roel Arents
* Ruin09
* Sion Kang
* Son Bui
* Spencer Cai
* Subin Kim
* Suraj Banakar(बानकर) | スラジ
* Thearas
* Weidong Cai
* Yang Lu
* Yuan (Terry) Tang
* Yusuke Shinoda
* b-erdem
* dependabot[bot]
* github-actions[bot]
* guangwu
* gussan
* happyso
* junkmm
* moonyoung
* nsimons
* younggil
* yyzxw
* 一条肥鱼
* 张志强

</details>

## v3.5.0-rc1 (2023-08-15)

Full Changelog: [v3.4.18...v3.5.0-rc1](https://github.com/argoproj/argo-workflows/compare/v3.4.18...v3.5.0-rc1)

### Selected Changes

* [1fd6e40e8](https://github.com/argoproj/argo-workflows/commit/1fd6e40e82a3fbba0d44d99cbb7ae4e02ed22588) fix: fail test on pr #11368 (#11576)
* [031a272c4](https://github.com/argoproj/argo-workflows/commit/031a272c4161c71a6b846869b94b410f1b6ebae2) chore(deps): bump google.golang.org/api from 0.133.0 to 0.136.0 (#11565)
* [8fb05215d](https://github.com/argoproj/argo-workflows/commit/8fb05215dcb75d033f17ae25aebe115b0a972474) chore(deps): bump github.com/antonmedv/expr from 1.12.7 to 1.13.0 (#11566)
* [50d9a4368](https://github.com/argoproj/argo-workflows/commit/50d9a4368c3118b0406b5418d0e8e29ae8dc7ad7) chore(deps): bump cronstrue from 2.28.0 to 2.29.0 in /ui (#11561)
* [311214c70](https://github.com/argoproj/argo-workflows/commit/311214c704ab8f443548c211d848b719a813b62c) fix(server): don't grab SAs if SSO RBAC is not enabled (#11426)
* [105031b88](https://github.com/argoproj/argo-workflows/commit/105031b88d45330a74777c6cd7410742827c3fe7) fix: always fail dag when shutdown is enabled. Fixes #11452 (#11493)
* [587acfcd0](https://github.com/argoproj/argo-workflows/commit/587acfcd098aa68e2acc1aea72d4a34c4bd89cbd) feat: add support for codegen/pre-commit via Nix. Fixes #11443 (#11503)
* [19674de8f](https://github.com/argoproj/argo-workflows/commit/19674de8fa6be8cd5e8213062c8531bfd94e5a75) fix: Update config for metrics, throttler, and entrypoint. Fixes #11542, #11541 (#11553)
* [43f15c6e3](https://github.com/argoproj/argo-workflows/commit/43f15c6e3a0a500dd769371dd49050ad090e7e7f) fix: Upgraded docker distribution go package to v2.8.2 for fixing a high vulnerability (#11554)
* [66e78a520](https://github.com/argoproj/argo-workflows/commit/66e78a520e607981a2421ed55950abb826e67f1d) fix: prevent stdout from disappearing in script templates. Fixes #11330 (#11368)
* [68b7ea6f7](https://github.com/argoproj/argo-workflows/commit/68b7ea6f774704f1c5aa7c1e780722c87aebb3b3) fix: Upgrade hdfs and rpc module #10030 (#11543)
* [1709f9630](https://github.com/argoproj/argo-workflows/commit/1709f96306a2f2f9dbc70cd91e005c667a140e00) fix: workflow-controller-configmap/parallelism setting not working in… (#11546)
* [6e50cb06c](https://github.com/argoproj/argo-workflows/commit/6e50cb06ce62dd19e969570540b5111dfbdde068) fix: Switch to use kong/httpbin to support arm64. Fixes #10427 (#11533)
* [b2e2106d3](https://github.com/argoproj/argo-workflows/commit/b2e2106d3a8ac3e7b77924673b935f2703902508) fix: Added vulnerability fixes for gorestlful gopkg & OS vulnerabilities in golang:1.20-alpine3.16 (#11538)
* [4a3cb0e98](https://github.com/argoproj/argo-workflows/commit/4a3cb0e98d5a72149041043ce13865e4adcade69) fix: Flaky test about lifecycle hooks (#11534)
* [143d0f504](https://github.com/argoproj/argo-workflows/commit/143d0f504c9382976b5a25a36b108b7f5e24ab37) fix: Fixed memoization is unchecked after mutex synchronization. Fixes #11219 (#11456)
* [545bf3803](https://github.com/argoproj/argo-workflows/commit/545bf3803d6f0c59a4c0a93db23d18001462bf3c) fix: Ensure target Workflow hooks not nil (#11521) (#11535)
* [9a9586cf2](https://github.com/argoproj/argo-workflows/commit/9a9586cf20b4377241886daf72dfa5b9a6fe89f5) chore(deps): bump github.com/aliyun/aliyun-oss-go-sdk from 2.2.7+incompatible to 2.2.8+incompatible (#11524)
* [5d8edd72a](https://github.com/argoproj/argo-workflows/commit/5d8edd72acaf335ff9b2c57d8d77f6216bffcfd6) chore(deps): bump golang.org/x/oauth2 from 0.10.0 to 0.11.0 (#11526)
* [9c7724770](https://github.com/argoproj/argo-workflows/commit/9c772477002dc316fa60df1818e89a3804f2f7af) fix: azure hasLocation incorporates endpoint. Fixes #11512 (#11513)
* [b26f5b80e](https://github.com/argoproj/argo-workflows/commit/b26f5b80ef4a3774ea85dcf6dfae95bac2253b47) fix: Support `OOMKilled` with container-set. Fixes #10063 (#11484)
* [cb1713d01](https://github.com/argoproj/argo-workflows/commit/cb1713d01542a7233d9bcb6646cc3c3409c5d870) fix: valueFrom in template parameter should be overridable. Fixes 10182 (#10281)
* [61a4ac45c](https://github.com/argoproj/argo-workflows/commit/61a4ac45cde5fca2788c83cba0383ea3c1cb868d) fix: Ignore failed read of exit code. Fixes #11490 (#11496)
* [f6c6dd7c4](https://github.com/argoproj/argo-workflows/commit/f6c6dd7c4ad6bc41d511adb1bad2e191ed3675d3) fix: Fixed UI workflowDrawer information link broken. Fixes #11494 (#11495)
* [1f6b19f3a](https://github.com/argoproj/argo-workflows/commit/1f6b19f3ab9f8758684bb6c93289d57c5dd1d963) fix: add guard against NodeStatus. Fixes #11102 (#11451)
* [ce9e50cd8](https://github.com/argoproj/argo-workflows/commit/ce9e50cd8f6063bdcd15dad4dfdb32e19b639faa) fix: Datepicker Style Malfunction Issue. Fixes #11476 (#11480)
* [20a741226](https://github.com/argoproj/argo-workflows/commit/20a741226ec44835c28b82273575aa6720ca6b4d) chore(deps): bump github.com/tidwall/gjson from 1.14.4 to 1.15.0 (#11468)
* [6b3620091](https://github.com/argoproj/argo-workflows/commit/6b362009138ac2ee16cb07f9206b56794b6de0c4) feat: Use WorkflowTemplate/ClusterWorkflowTemplate Informers when validating CronWorkflows (#11470)
* [e53a26579](https://github.com/argoproj/argo-workflows/commit/e53a265799bd4ae10681a4c5d4dba8ae03c0a62f) feat: improve alibaba cloud credential providers in OSS artifacts (#11453)
* [be0bdf9b0](https://github.com/argoproj/argo-workflows/commit/be0bdf9b0eab7d9d23fbb8df0426b4075af6830d) feat: Expose the Cron workflow workers as argument (#11457)
* [90930ab88](https://github.com/argoproj/argo-workflows/commit/90930ab88b18b7fba3074cdc06059eb6460b50d9) feat: cli allow retry successful workflow if nodeFieldSelector is set. Fixes #11020 (#11409)
* [f8a34a3b5](https://github.com/argoproj/argo-workflows/commit/f8a34a3b5929fb63a60b50dea50e4b5a6c226d6b) fix: Devcontainer resets /etc/hosts (#11439) (#11440)
* [82876af44](https://github.com/argoproj/argo-workflows/commit/82876af4438ff6ad52b6fd6a7c50e47519e5b030) chore(deps): bump github.com/antonmedv/expr from 1.12.6 to 1.12.7 (#11399)
* [7310e9c41](https://github.com/argoproj/argo-workflows/commit/7310e9c41a03cb128bc644cb3b734d89a8b0436e) fix: UI toolbar sticky (#11444)
* [336d8a41a](https://github.com/argoproj/argo-workflows/commit/336d8a41a455896d97ea751e46c8e2bcb712fa84) feat: logging for client-side throttling (#11437)
* [a76674c82](https://github.com/argoproj/argo-workflows/commit/a76674c829461bfb252e904c3fef9c231cadbb56) feat: Allow memoization without outputs (#11379)
* [593e10130](https://github.com/argoproj/argo-workflows/commit/593e101308d0e919c5c26acb9c666ff5c95b906c) chore(deps): bump google.golang.org/api from 0.132.0 to 0.133.0 (#11434)
* [64de64263](https://github.com/argoproj/argo-workflows/commit/64de64263a11c5a6700c237e8dbae4f161d98907) chore(deps): bump github.com/Azure/azure-sdk-for-go/sdk/azcore from 1.6.0 to 1.7.0 (#11396)
* [2071d147f](https://github.com/argoproj/argo-workflows/commit/2071d147fa76d6434c2d3b463bbcde2c93ca7e73) chore(deps): bump github.com/minio/minio-go/v7 from 7.0.60 to 7.0.61 (#11398)
* [f40a564ee](https://github.com/argoproj/argo-workflows/commit/f40a564eed987a0bb16b3a85c3870372741f7026) chore(deps): bump google.golang.org/api from 0.130.0 to 0.132.0 (#11397)
* [c6a2a4f15](https://github.com/argoproj/argo-workflows/commit/c6a2a4f152f08bf2c4f9dda8cd2c8eb00d9eb712) fix: Apply the creator labels about the user who resubmitted a Workflow (#11415)
* [f5d41f8c9](https://github.com/argoproj/argo-workflows/commit/f5d41f8c9e332305f31f9ad7ef3943995b802683) fix: make archived logs more human friendly in UI (#11420)
* [5cb75d91a](https://github.com/argoproj/argo-workflows/commit/5cb75d91a0e3d2fa329be9efbf096e7b02f9e123) fix: add query string to workflow detail page(#11371) (#11373)
* [5b31ca18b](https://github.com/argoproj/argo-workflows/commit/5b31ca18b306c4bb1c7c218a59cbc75dceb77fd9) fix: persist archived workflows with `Persisted` label (#11367) (#11413)
* [0d7820865](https://github.com/argoproj/argo-workflows/commit/0d782086526b319710f159a950080d92e17556ca) feat: Propagate creator labels of a CronWorkflow to the Workflow to be scheduled (#11407)
* [082f06326](https://github.com/argoproj/argo-workflows/commit/082f063266a512380300290ef8d87ae154d4a077) fix: download subdirs in azure artifact. Fixes #11385 (#11394)
* [869e42d5e](https://github.com/argoproj/argo-workflows/commit/869e42d5e4aa7b758d6c1716b961cc82d29276ca) feat: UI Resubmit workflows with parameter (#4662) (#11083)
* [22d4e179c](https://github.com/argoproj/argo-workflows/commit/22d4e179c3818918c6c4a1fd5ea8d28c816462cc) feat: Improve logging in the oauth2 callback handler (#11370)
* [97b6fa844](https://github.com/argoproj/argo-workflows/commit/97b6fa84441c423c68ecc8a8f1af5e26402d118e) fix: Modify broken ui by archived col (#11366)
* [37f483d1c](https://github.com/argoproj/argo-workflows/commit/37f483d1c76fb8afa187378e8750e9702734945f) chore(deps): bump github.com/minio/minio-go/v7 from 7.0.59 to 7.0.60 (#11363)
* [779922a90](https://github.com/argoproj/argo-workflows/commit/779922a90c9559299560e1c5261a2e8085a9b8ad) chore(deps): bump github.com/antonmedv/expr from 1.12.5 to 1.12.6 (#11365)
* [e08db70fd](https://github.com/argoproj/argo-workflows/commit/e08db70fd5e08126cef0965c706a0dce6178ca93) chore(deps): bump react-datepicker from 4.15.0 to 4.16.0 in /ui (#11362)
* [bda532211](https://github.com/argoproj/argo-workflows/commit/bda532211fd0038ee567922db917bc04e29f9130) fix: Enable the workflow created by a wftmpl to retry after manually stopped (#11355)
* [d992ec58c](https://github.com/argoproj/argo-workflows/commit/d992ec58ce3dcbe5e799570db8f53cae746b8f14) feat: Enable local docker ip in for communication with outside k3d (#11350)
* [43d667ed2](https://github.com/argoproj/argo-workflows/commit/43d667ed2603a004c34be2890ad45ed4f63ce1bc) fix: Correct limit in controller List API calls. Fixes #11134 (#11343)
* [383bb6b2a](https://github.com/argoproj/argo-workflows/commit/383bb6b2ab537b6ec7a4999d106a96df7cf31b31) feat(podGC): add Workflow-level `DeleteDelayDuration` (#11325)
* [6120a2db1](https://github.com/argoproj/argo-workflows/commit/6120a2db18d31f977be6a5b76a4572c1f75da007) feat: Support batch deletion of archived workflows. Fixes #11324 (#11338)
* [fdb3ec03f](https://github.com/argoproj/argo-workflows/commit/fdb3ec03f204ed0960f662d1f7bcb7501b4a80bd) fix: Live workflow takes precedence during merge to correctly display in the UI (#11336)
* [15a83651a](https://github.com/argoproj/argo-workflows/commit/15a83651a47b1a9c3612642ba9c28da24a14a760) chore(deps): bump cronstrue from 2.27.0 to 2.28.0 in /ui (#11329)
* [82310dd45](https://github.com/argoproj/argo-workflows/commit/82310dd459aa080754169d6a1667d30d9b7c75bf) feat: Unified workflows list UI and API (#11121)
* [526458449](https://github.com/argoproj/argo-workflows/commit/5264584496ebb62c7098daa986692284b9e6478a) chore(deps): bump golang.org/x/oauth2 from 0.9.0 to 0.10.0 (#11317)
* [d0b9b03a7](https://github.com/argoproj/argo-workflows/commit/d0b9b03a7350292a6faeeb4b758de2fa70bb4fd4) chore(deps): bump google.golang.org/api from 0.129.0 to 0.130.0 (#11318)
* [f4e9ae7fd](https://github.com/argoproj/argo-workflows/commit/f4e9ae7fd3f18098a15351130bb2d7bf04fc8b99) chore(deps): bump github.com/stretchr/testify from 1.8.2 to 1.8.4 (#11319)
* [a10139ad3](https://github.com/argoproj/argo-workflows/commit/a10139ad364f7d50b5f86894cc6e1ad8147a99c7) fix: Add ^ to semver version (#11310)
* [4ca470b10](https://github.com/argoproj/argo-workflows/commit/4ca470b1053e7e6f660f36dd07c3821b67842d3f) fix: Pin semver to 7.5.2. Fixes SNYK-JS-SEMVER-3247795 (#11306)
* [137d5f8cc](https://github.com/argoproj/argo-workflows/commit/137d5f8cce3ced586b1343541712cb0c1ae4ef53) fix(controller): Enable dummy metrics server on non-leader workflow controller (#11295)
* [6f1cb4843](https://github.com/argoproj/argo-workflows/commit/6f1cb484370e79b2431d2ce507a264cf5769616a) fix(windows): Propagate correct numerical exitCode under Windows (Fixes #11271) (#11276)
* [e5dd8648f](https://github.com/argoproj/argo-workflows/commit/e5dd8648f1b7347c7cba8cc04a66eaa71d2ccb0e) fix: use unformatted templateName as args to PodName. Fixes #11250 (#11251)
* [609539df4](https://github.com/argoproj/argo-workflows/commit/609539df43d0e12adcf0cb85f8c331d1017c17cf) fix: Azure input artifact support optional. Fixes #11179 (#11235)
* [7f155e47c](https://github.com/argoproj/argo-workflows/commit/7f155e47cfffc00c281d45dfa29ea6fd93315321) fix: Argo DB init conflict when deploy workflow-controller with multiple replicas #11177 (#11178)
* [90fe330de](https://github.com/argoproj/argo-workflows/commit/90fe330de06e774fb77791c156f9f7cabcf5d9df) chore(deps): bump google.golang.org/api from 0.128.0 to 0.129.0 (#11286)
* [d815c5582](https://github.com/argoproj/argo-workflows/commit/d815c5582dadea793de8858826aa7a6a9a7ab17a) chore(deps): bump react-datepicker from 4.14.1 to 4.15.0 in /ui (#11289)
* [6dfe5d49e](https://github.com/argoproj/argo-workflows/commit/6dfe5d49ea1ce92aaa5831450cde2ce73968d5ca) chore(deps): bump github.com/minio/minio-go/v7 from 7.0.58 to 7.0.59 (#11285)
* [476eca40f](https://github.com/argoproj/argo-workflows/commit/476eca40f66aaba67088285fdbd59da6670311d4) chore(deps): bump cloud.google.com/go/storage from 1.30.1 to 1.31.0 (#11284)
* [451d27509](https://github.com/argoproj/argo-workflows/commit/451d2750997e9c12e3a8904aaca0a2865ef628c0) fix: fix bugs in throttler and syncManager initialization in WorkflowController (#11210)
* [29d63c564](https://github.com/argoproj/argo-workflows/commit/29d63c5648d6361a9ad59f1ab94e1fe9a4c744ad) feat: Added label selectors to argo cron list. Fixes #11158 (#11202)
* [aa2b66a5b](https://github.com/argoproj/argo-workflows/commit/aa2b66a5b7d8c3ab2af900a5fbda948b13d14085) fix: do not delete pvc when max parallelism has been reached. Fixes #11119 (#11138)
* [f180335b3](https://github.com/argoproj/argo-workflows/commit/f180335b370643b731edcd133b7ef35de36a83e6) chore(deps): bump react-datepicker from 4.14.0 to 4.14.1 in /ui (#11263)
* [40f4d1d2e](https://github.com/argoproj/argo-workflows/commit/40f4d1d2e5b126a8291942e6ba50e208c4a50a15) chore(deps): bump golang.org/x/sync from 0.2.0 to 0.3.0 (#11262)
* [8089f41bd](https://github.com/argoproj/argo-workflows/commit/8089f41bd33cd05eba67a2fdb2f22257131eaf25) chore(deps): bump github.com/minio/minio-go/v7 from 7.0.56 to 7.0.58 (#11261)
* [b89c98bff](https://github.com/argoproj/argo-workflows/commit/b89c98bff9342b472359d2c465a5c5c150c11522) fix: Upgrade windows container to ltsc2022 (#11246)
* [0fd42276b](https://github.com/argoproj/argo-workflows/commit/0fd42276b827e63e6f36e0f2dcb88b6b7a959765) fix: Update Bitbucket SSH host key (#11245)
* [53111f885](https://github.com/argoproj/argo-workflows/commit/53111f88516c398a237da17e598e74f915507c95) fix: Support inputs for inline steps templates (#11074)
* [0ad4169c2](https://github.com/argoproj/argo-workflows/commit/0ad4169c2c781f3be6e7057e2783282055bd6c94) fix: Allow hooks to be specified in workflowDefaults (#11214)
* [245af38b1](https://github.com/argoproj/argo-workflows/commit/245af38b1c7eddfcc95d7e4214b9c3136077d29c) fix: untar empty directories (#11240)
* [abc4e8fa8](https://github.com/argoproj/argo-workflows/commit/abc4e8fa83a0b33690aaedbb86a7413e993c3839) fix: Treat "connection refused" error as a transient network error. (#11237)
* [ac9161ce1](https://github.com/argoproj/argo-workflows/commit/ac9161ce150a961536aef9d76dd65457dea5d378) fix: Workflow list page crashes for workflow rows without labels (#11195)
* [fa95f8d02](https://github.com/argoproj/argo-workflows/commit/fa95f8d02f418bf7fb66ee60908b465ff7af8c9d) fix: prevent memoization accessing wrong config-maps (#11225)
* [d2091a710](https://github.com/argoproj/argo-workflows/commit/d2091a7106bafda89cac33954f2c712cba25a622) chore(deps): bump react-datepicker from 4.12.0 to 4.14.0 in /ui (#11231)
* [da89c2f96](https://github.com/argoproj/argo-workflows/commit/da89c2f965b0298ca9b4ccce6ee872df189388e9) chore(deps): bump golang.org/x/oauth2 from 0.8.0 to 0.9.0 (#11228)
* [572641f9c](https://github.com/argoproj/argo-workflows/commit/572641f9ca173b7f44d1e603025ba7b0449a6f3c) chore(deps): bump github.com/prometheus/client_golang from 1.15.1 to 1.16.0 (#11227)
* [424e2238d](https://github.com/argoproj/argo-workflows/commit/424e2238d2033a25feaed52a80da2cd87544561b) chore(deps): bump google.golang.org/api from 0.124.0 to 0.128.0 (#11229)
* [d91b72172](https://github.com/argoproj/argo-workflows/commit/d91b72172e78d43671109f1a422a45c9306adb12) fix: Remove 401 Unauthorized when customClaimGroup retrieval fails, Fixes #11032 (#11033)
* [d3a6e66a9](https://github.com/argoproj/argo-workflows/commit/d3a6e66a9fb3d7f322fc16de630832ccd0311b20) chore(deps): bump github.com/sirupsen/logrus from 1.9.2 to 1.9.3 (#11200)
* [15d84639b](https://github.com/argoproj/argo-workflows/commit/15d84639b3b715b57b5d30634832558ee8a56b99) feat(ui): Ignore missing vars in custom links (#11164)
* [0c5a6dd4b](https://github.com/argoproj/argo-workflows/commit/0c5a6dd4b09f5b7d75fc0e74cf75e9f8f86879e4) fix: check hooked nodes in executeWfLifeCycleHook and executeTmplLifeCycleHook (#11113, #11117) (#11176)
* [f3c948a04](https://github.com/argoproj/argo-workflows/commit/f3c948a047e01d99a150e82b267b000db850bcbf) chore(deps): bump github.com/itchyny/gojq from 0.12.12 to 0.12.13 (#11170)
* [760299ff9](https://github.com/argoproj/argo-workflows/commit/760299ff9bb44feddae89d43dc600b1ba27b994d) chore(deps): bump github.com/minio/minio-go/v7 from 7.0.55 to 7.0.56 (#11169)
* [2016078c0](https://github.com/argoproj/argo-workflows/commit/2016078c0ac5af19eaab09049ac54d76b864fb6b) fix: add space to fix release action issue (#11160)
* [b9f95446f](https://github.com/argoproj/argo-workflows/commit/b9f95446f02e0ebe8e5f1f74da6a00bf4ef28361) feat: mainEntrypoint variable (#11151)
* [bccba4081](https://github.com/argoproj/argo-workflows/commit/bccba40815a537cedfa48f96ec4c98f56fdcce96) feat: Add functionality for nix. Fixes #10998 (#10999)
* [3273709e0](https://github.com/argoproj/argo-workflows/commit/3273709e060e4a37fcc17017f5ae67fd80bc53e0) feat: Add `Check All` checkbox to workflow dag filter options. Fixes #11129 (#11132)
* [c9ebf424d](https://github.com/argoproj/argo-workflows/commit/c9ebf424db31f08e1172deccacd09c96f5820d32) feat: allow cross-namespace locking for semaphore and mutex (#11096)
* [58793a8ca](https://github.com/argoproj/argo-workflows/commit/58793a8ca54486c0a929ba7197d30b9f3cb3ce17) fix: Make devcontainer able to pre-commit (#11153)
* [b239c615e](https://github.com/argoproj/argo-workflows/commit/b239c615e1d4600632c4256deee29d51ada13269) chore(deps): bump github.com/go-sql-driver/mysql from 1.7.0 to 1.7.1 (#11007)
* [1a51e4fd1](https://github.com/argoproj/argo-workflows/commit/1a51e4fd1161500c56addd342afc26f78ea7a8ea) chore(deps): bump google.golang.org/api from 0.122.0 to 0.124.0 (#11142)
* [afde7ef41](https://github.com/argoproj/argo-workflows/commit/afde7ef41ac6e2127d50e89640e9a203b0253b82) chore(deps): bump react-datepicker from 4.11.0 to 4.12.0 in /ui (#11147)
* [6923cc837](https://github.com/argoproj/argo-workflows/commit/6923cc8375ff43b5a86b2929a7c02e57ac82ea4d) chore(deps): bump github.com/minio/minio-go/v7 from 7.0.52 to 7.0.55 (#11145)
* [04d527c7d](https://github.com/argoproj/argo-workflows/commit/04d527c7dd12aadb00527481b1a83f9db967e4b3) fix: Setup /etc/hosts for running inside devcontainer (#11104)
* [310bb5a7d](https://github.com/argoproj/argo-workflows/commit/310bb5a7dd3b8cc09c3969917618379ca8e7bd95) feat: Make retryPolicy saner in the presence of an expression (#11005)
* [1f6d1baf9](https://github.com/argoproj/argo-workflows/commit/1f6d1baf955086c379af47e7bad78b905933ec47) feat: Support GetWorkflow regardless of its archival status (#11055)
* [1e4a376ab](https://github.com/argoproj/argo-workflows/commit/1e4a376abb38451ce3784cb6430b796616e8191f) fix: Fixed path separator in .tgz output artifacts created on windows. Fixes #10562 (#11097)
* [9a2bb5e80](https://github.com/argoproj/argo-workflows/commit/9a2bb5e80d9f4e2d8792f31b57efbadc7ad41ef1) fix: Disable unreliable test (#11105)
* [7b2a4b1a3](https://github.com/argoproj/argo-workflows/commit/7b2a4b1a3618e1df4e14da473bfe081ac5d238af) chore(deps): bump github.com/sirupsen/logrus from 1.9.0 to 1.9.2 (#11107)
* [901549f1f](https://github.com/argoproj/argo-workflows/commit/901549f1fb023d25e1881139df0301b8a5975020) fix: allow azure blobs to not exist when deleting (#11070)
* [019705eeb](https://github.com/argoproj/argo-workflows/commit/019705eeb92725dff3501249708462ef97952f2b) fix: Update Bitbucket SSH host key (#11091)
* [c1c3a5a1f](https://github.com/argoproj/argo-workflows/commit/c1c3a5a1fe47508da7bc2942dbee4763dccadcd6) fix: Parameter overwritten does not work when resubmitting archived workflow (#11086) (#11087)
* [110ed1f3e](https://github.com/argoproj/argo-workflows/commit/110ed1f3ecf28129ee5030c1ac438094e832b071) chore(deps): bump google.golang.org/api from 0.120.0 to 0.122.0 (#11089)
* [6b604450d](https://github.com/argoproj/argo-workflows/commit/6b604450d43d413e68f7b40b1e6348628d1109f4) chore(deps): bump google.golang.org/api from 0.118.0 to 0.120.0 (#11008)
* [7b3d53dbc](https://github.com/argoproj/argo-workflows/commit/7b3d53dbce40580fe60f027d0cbd6f1308197cdf) chore(deps): bump cronstrue from 2.26.0 to 2.27.0 in /ui (#11078)
* [e2cc77743](https://github.com/argoproj/argo-workflows/commit/e2cc777431d686d5b291008092b8a61176341533) fix: UI crashes when retrying a containerSet workflow. Fixes #11061 (#11073)
* [4225cb8bf](https://github.com/argoproj/argo-workflows/commit/4225cb8bf77abb82dc7c8f5abb78439ef19cca10) fix: ui getPodName should use v2 format by default (fixes #11015) (#11016)
* [5a81dd225](https://github.com/argoproj/argo-workflows/commit/5a81dd22599129d477c5eb139a9f3976db5f3829) chore(deps): bump golang.org/x/crypto from 0.8.0 to 0.9.0 (#11068)
* [8c6982264](https://github.com/argoproj/argo-workflows/commit/8c6982264029dbd179817da06299a5be8bec9da9) chore(deps): bump golang.org/x/oauth2 from 0.7.0 to 0.8.0 (#11058)
* [612adcdab](https://github.com/argoproj/argo-workflows/commit/612adcdabbdd98e2d78196b463b26fdb6a1f2f98) feat: Hide empty fields in user info page. Fixes #11065 (#11066)
* [bd89a776b](https://github.com/argoproj/argo-workflows/commit/bd89a776b8b278b45da96cc57a5069068f2a36e7) chore(deps): bump golang.org/x/sync from 0.1.0 to 0.2.0 (#11041)
* [b0e343b2d](https://github.com/argoproj/argo-workflows/commit/b0e343b2da571d4fa2e0a6191fbe0868177619bc) chore(deps): bump github.com/prometheus/client_golang from 1.15.0 to 1.15.1 (#11029)
* [d4549b3d5](https://github.com/argoproj/argo-workflows/commit/d4549b3d5dc0046584c3855aea10c15d3048d0e1) fix: handle panic from type assertion (#11040)
* [5294f354e](https://github.com/argoproj/argo-workflows/commit/5294f354e38243acac26cd73ce5bcea3d0711fad) fix: change pod OwnerReference to clean workflowtaskresults in large-scale scenarios (#11048)
* [1e22f06ca](https://github.com/argoproj/argo-workflows/commit/1e22f06ca54e8423516780250bd13c4721f46506) chore(deps): bump golang.org/x/term from 0.7.0 to 0.8.0 (#11044)
* [9aa8903de](https://github.com/argoproj/argo-workflows/commit/9aa8903deedf9820b639c53405df399125cb9b7e) chore(deps): bump github.com/klauspost/pgzip from 1.2.5 to 1.2.6 (#11045)
* [a5581f83a](https://github.com/argoproj/argo-workflows/commit/a5581f83abd4b6d45b1bad6c9a5d471077e8427f) fix: Upgrade Go to v1.20. Fixes #11023 (#11027)
* [1af85fd4c](https://github.com/argoproj/argo-workflows/commit/1af85fd4c98ffadb0c130ddb7ba5bb891201c08d) fix: UI crashes after submitting workflows (#11018)
* [f2573ed17](https://github.com/argoproj/argo-workflows/commit/f2573ed179cb5afeead51545a5e318fdd1012da8) fix: Generate useful error message when no expression on hook (#10919)
* [91f2a4548](https://github.com/argoproj/argo-workflows/commit/91f2a4548832d1a669ed2cc32623ead83013fc97) fix: Validate label values from workflowMetadata.labels to avoid controller crash (#10995)
* [c49d33b94](https://github.com/argoproj/argo-workflows/commit/c49d33b94d64683b4f57c5ce7d27696929cf840e) feat: Add lastRetry.message (#10987)
* [48097ea0b](https://github.com/argoproj/argo-workflows/commit/48097ea0baa3683a62ddb465ba8a066fbabf8cdb) fix(controller): Drop Checking daemoned children without nodeID (Fixes #10960) (#10974)
* [8dbdc0250](https://github.com/argoproj/argo-workflows/commit/8dbdc02504f51b6386ef4ddc390146169d16444c) fix: Replace expressions with placeholders in resource manifest template. Fixes #10924 (#10926)
* [2401be8ef](https://github.com/argoproj/argo-workflows/commit/2401be8efd2f846f84d9a49eddd4243fc457ed7b) feat(operator): Add hostNodeName as a template variable (#10950)
* [8786b46ae](https://github.com/argoproj/argo-workflows/commit/8786b46ae9c77aa7bfa23027859884d3e88426fe) fix: unable to connect cluster when AutomountServiceAccountToken is disabled. Fixes #10937 (#10945)
* [1617db0f3](https://github.com/argoproj/argo-workflows/commit/1617db0f32366bc58cd7f00a044b7e1a58fb830e) fix: Check AlreadyExists error when creating PDB. Fixes #10942 (#10944)
* [fd292cab2](https://github.com/argoproj/argo-workflows/commit/fd292cab257842d89a5920671d2e814f540b5ddc) feat: Add operation configuration to gauge metric. Fixes #10662 (#10774)
* [b846eeb90](https://github.com/argoproj/argo-workflows/commit/b846eeb90769bd01fce4a6865260ee0352dc0dae) fix: Check file size before saving to artifact storage. Fixes #10902 (#10903)
* [9d28a02ac](https://github.com/argoproj/argo-workflows/commit/9d28a02acb03a0710889e14fa74fe90705049f0e) fix: Incorrect pod name for inline templates. Fixes #10912 (#10921)
* [d41add41e](https://github.com/argoproj/argo-workflows/commit/d41add41ea4eac9b43c8d581cf0f0dcdbff0f5e1) feat(server): support name claim for RBAC SSO (#10927)
* [09d48ef20](https://github.com/argoproj/argo-workflows/commit/09d48ef205390dc5bf64236d5a97b1fe1b959d85) chore(deps): bump google.golang.org/api from 0.117.0 to 0.118.0 (#10933)
* [c0565d62e](https://github.com/argoproj/argo-workflows/commit/c0565d62e7325f26c83198ad88af774486b4212d) chore(deps): bump github.com/aliyun/aliyun-oss-go-sdk from 2.2.6+incompatible to 2.2.7+incompatible (#10753)
* [819cbc9b4](https://github.com/argoproj/argo-workflows/commit/819cbc9b4d2454497c1eb98071d7b0d140a36ebb) chore(deps): bump google.golang.org/api from 0.114.0 to 0.117.0 (#10878)
* [8766e7a45](https://github.com/argoproj/argo-workflows/commit/8766e7a45cfa8aafb8ee23ceab505f6f1f8b9097) fix: Workflow operation error. Fixes #10285 (#10886)
* [c8e7fa8a7](https://github.com/argoproj/argo-workflows/commit/c8e7fa8a7362664cddbc481b75b37a6cd89be963) fix: Validate label values from workflowMetadata to avoid controller crash. Fixes #10872 (#10892)
* [694cec0a4](https://github.com/argoproj/argo-workflows/commit/694cec0a4fe61efa2aeeb37a9b8c1867a8e1129d) chore(deps): bump github.com/minio/minio-go/v7 from 7.0.51 to 7.0.52 (#10917)
* [afca5e3e5](https://github.com/argoproj/argo-workflows/commit/afca5e3e532ccfafd19250859c81f76d70371fd8) chore(deps): bump github.com/prometheus/client_golang from 1.14.0 to 1.15.0 (#10916)
* [b90485123](https://github.com/argoproj/argo-workflows/commit/b9048512327e2d66b8d6ceb18f7d4eddf2b4dc9c) fix: tableName is empty if wfc.session != nil (#10887)
* [12f465912](https://github.com/argoproj/argo-workflows/commit/12f465912297c79a2ffcb350a21d7aeae77821cc) fix: Flaky test about lifecycle hooks. Fixes #10897 (#10898)
* [b87bdcfcf](https://github.com/argoproj/argo-workflows/commit/b87bdcfcfc042ff226779a27c9b58f463ec9e490) fix: Allow script and container image to be set in templateDefault. Fixes #9633 (#10784)
* [2edf2cf17](https://github.com/argoproj/argo-workflows/commit/2edf2cf17f0ddfafff12da869e9524d34403714e) chore(deps): bump golang.org/x/oauth2 from 0.6.0 to 0.7.0 (#10860)
* [d2bb05261](https://github.com/argoproj/argo-workflows/commit/d2bb0526107d812b178929786a060be7aae29c91) chore(deps): bump github.com/minio/minio-go/v7 from 7.0.50 to 7.0.51 (#10877)
* [610e9a729](https://github.com/argoproj/argo-workflows/commit/610e9a729878338e84d14d0b61920780cf7f44b4) chore(deps): bump golang.org/x/crypto from 0.7.0 to 0.8.0 (#10856)
* [dcf66171c](https://github.com/argoproj/argo-workflows/commit/dcf66171cac0165ee52a7bb885d62efca00bfeda) chore(deps): bump cronstrue from 2.24.0 to 2.26.0 in /ui (#10855)
* [e8433f40d](https://github.com/argoproj/argo-workflows/commit/e8433f40df090dffaa2f38160d0b807c87fd1408) chore(deps): bump peter-evans/create-pull-request from 4 to 5 (#10854)
* [d1812efae](https://github.com/argoproj/argo-workflows/commit/d1812efae6aa5b050cd3765633e50364489b23b0) fix: Upgrade docker to v20.10.24 for CVE-2023-28840, CVE-2023-28841, CVE-2023-28842 (#10868)
* [506158896](https://github.com/argoproj/argo-workflows/commit/506158896b885b71c55cba3d34ea2600ef158f72) fix: Fix not working Running state lifecycle hooks in dag task. Fixes #9897 (#10307)
* [839808d23](https://github.com/argoproj/argo-workflows/commit/839808d23763126f6131cd980621bbb6d2bbcc00) fix: make workflow status change after workflow level lifecycle hooks complete. Fixes #10743, #9591 (#10758)
* [f418eacdd](https://github.com/argoproj/argo-workflows/commit/f418eacddc7e6a95b5793180a2c43f7408d3c0f9) fix: Workflow stuck at running for failed init containers with other names. Fixes #10717 (#10849)
* [e734ae524](https://github.com/argoproj/argo-workflows/commit/e734ae5241f31dcdb64756049dcabff4286daa27) feat: Enable configuration for finalizer removal if artifact GC fails (#10810)
* [49890ec2a](https://github.com/argoproj/argo-workflows/commit/49890ec2a39ce8ca9b8c68f9940bf95b98353482) feat: Expose customized UI links and columns (#10808)
* [3ed6887f3](https://github.com/argoproj/argo-workflows/commit/3ed6887f3fe3ffb247fab25ed8c1d0d8f9ca66c8) feat: add chunk size argument to delete cli command (#10813)
* [bc966848a](https://github.com/argoproj/argo-workflows/commit/bc966848aa07f63a60489c6379122dc71eb9e476) fix: download specific version of kit. Fixes #10768 (#10841)
* [5b7872548](https://github.com/argoproj/argo-workflows/commit/5b787254827777323e8f701b86d6fa235e96573d) fix: Resolve high severity vulnerabilities in UI deps (#10842)
* [c18daff97](https://github.com/argoproj/argo-workflows/commit/c18daff97233c8d1e0b2153127da0fd61b1794e5) fix: Security upgrade ubuntu from 14.04 to trusty-20190515 (#10832)

<details><summary><h3>Contributors</h3></summary>

* Abraham Bah
* Adam Eri
* Akashinfy
* Alan Clucas
* Aleksandr Lossenko
* Alex Collins
* Alexander Crow
* Amit Oren
* Anton Gilgur
* Aqeel Ahmad
* Brian Loss
* Byeonggon Lee
* Carlos M
* Cayde6
* Cheng Wang
* Christoph Buchli
* DahuK
* Dahye
* Dylan Bragdon
* Eduardo Rodrigues
* Elifarley C
* Federico Paparoni
* GeunSam2 (Gray)
* GoshaDozoretz
* Huan-Cheng Chang
* Iain Lane
* Isitha Subasinghe
* James Slater
* Jason Meridth
* Jinsu Park
* Josh Soref
* Julie Vogelman
* Lan
* LilTwo
* Lucas Heinlen
* Lukas Grotz
* Lukas Wöhrl
* Max Xu
* Mickaël Canévet
* Northes
* Oliver Skånberg-Tippen
* Or Shachar
* PanagiotisS
* PeterKoegel
* Rachel Bushrian
* Rafael
* Remington Breeze
* Rick
* Roel Arents
* Rohan Kumar
* RoryDoherty
* Rui Chen
* Ruin09
* Saravanan Balasubramanian
* Son Bui
* Takumi Sue
* Tianchu Zhao
* Tim Collins
* Tom Kahn
* Tore
* Vedant Shrotria
* Waleed Malik
* Yuan (Terry) Tang
* Yuan Tang
* YunCow
* Zubair Haque
* boiledfroginthewell
* cui fliter
* dependabot[bot]
* devops-42
* ehellmann-nydig
* github-actions[bot]
* gussan
* jxlwqq
* jyje
* luyang93
* mouuii
* munenori-harada
* peterandluc
* sakai-ast
* shuangkun tian
* smile-luobin
* tooptoop4
* toyamagu
* vanny96
* vitalyrychkov
* wangxiang
* weafscast
* yeicandoit
* younggil
* 李杰穎 (Jay Lee)

</details>

## v3.4.18 (2024-10-07)

Full Changelog: [v3.4.17...v3.4.18](https://github.com/argoproj/argo-workflows/compare/v3.4.17...v3.4.18)

### Selected Changes

* [c2b9dc6db](https://github.com/argoproj/argo-workflows/commit/c2b9dc6dbc7b209eadf122238a9a9b2082ac8266) fix(ui): remove leading slash in `uiUrl` for ArchivedWorkflowList redirect. Fixes #13056 (#13713)
* [464965c20](https://github.com/argoproj/argo-workflows/commit/464965c20c9cd310ea1f958db2a4ef620023bce6) fix(cli): Ensure `--dry-run` and `--server-dry-run` flags do not create workflows. fixes #12944 (#13183)
* [b373dd299](https://github.com/argoproj/argo-workflows/commit/b373dd2993444382da7dcf71516e54aa10e877ac) fix: Update modification timestamps on untar. Fixes #12885 (#13172)
* [bb7cc532f](https://github.com/argoproj/argo-workflows/commit/bb7cc532f2235a337a22078def2ad32a99a1b69d) revert: "fix: Live workflow takes precedence during merge to correctly display in the UI (#11336)"
* [9bc603944](https://github.com/argoproj/argo-workflows/commit/9bc603944726ba34efbd80720caff58f0ccfa983) revert: "fix: Fix the Maximum Recursion Depth prompt link in the CLI. (#12015)"
* [1f600ff4c](https://github.com/argoproj/argo-workflows/commit/1f600ff4c858a80d19dd5f7e05b2803240619903) revert: "fix(ui): show correct podGC message for deleteDelayDuration. Fixes: #12395 (#12784)"
* [241b7bb96](https://github.com/argoproj/argo-workflows/commit/241b7bb964153a75410c30a9336bdf0cd5f70996) fix(ui): `package.json#license` should be Apache (#13040)
* [981acfe12](https://github.com/argoproj/argo-workflows/commit/981acfe12553a806b16359f107adbc51a6e484fe) fix: Mark `Pending` pod nodes as `Failed` when shutting down. Fixes #13210 (#13214)
* [d5ab9478a](https://github.com/argoproj/argo-workflows/commit/d5ab9478a42f1a8273f6543b52182751fd185f3b) fix: oss internal error should retry. Fixes #13262 (#13263)
* [827388c66](https://github.com/argoproj/argo-workflows/commit/827388c660223bef2b6025e1a4d34be35a9d16d9) fix(docs): clarify CronWorkflow `startingDeadlineSeconds`. Fixes #12971 (#13280)
* [b9e826676](https://github.com/argoproj/argo-workflows/commit/b9e8266765a9d5a859f3b4c9a933a1d3f8ab827a) fix(build): bump golang to 1.21.12 in builder image to fix CVEs (#13311)

<details><summary><h3>Contributors</h3></summary>

* Andrew Fenner
* Anton Gilgur
* Gongpu Zhu
* Huan-Cheng Chang
* Miltiadis Alexis
* Northes
* Oliver Dain
* Travis Stevens
* github-actions[bot]
* jswxstw
* jyje
* shuangkun tian
* tooptoop4
* 名白

</details>

## v3.4.17 (2024-05-12)

Full Changelog: [v3.4.16...v3.4.17](https://github.com/argoproj/argo-workflows/compare/v3.4.16...v3.4.17)

### Selected Changes

* [72efa2f15](https://github.com/argoproj/argo-workflows/commit/72efa2f1509d55c8863cf806c2ad83adf0aea65a) chore(deps): bump github.com/cloudflare/circl to 1.3.7 to fix GHSA-9763-4f94-gfch (#12556)
* [0f71a40db](https://github.com/argoproj/argo-workflows/commit/0f71a40dbd35da3090c8fcbaa88299bcc6c6e037) chore(deps): fixed medium CVE in github.com/docker/docker v24.0.0+incompatible (#12635)
* [6030af483](https://github.com/argoproj/argo-workflows/commit/6030af483b34357b74b46f9760b24379cc2ea2bb) chore(deps): upgrade Cosign to v2.2.3 (#12850)
* [cc258b874](https://github.com/argoproj/argo-workflows/commit/cc258b874cf1fd6af30e8246497a2688be5cf0c5) build(deps): bump github.com/docker/docker from 24.0.0+incompatible to 24.0.9+incompatible (#12878)
* [7e7d99b67](https://github.com/argoproj/argo-workflows/commit/7e7d99b67bb0d237940a30583404eb4b039daea3) build(deps): bump github.com/go-jose/go-jose/v3 from 3.0.1 to 3.0.3 (#12879)
* [6bb096efb](https://github.com/argoproj/argo-workflows/commit/6bb096efb6d2d0b0f692e9ac22c0c795c9b3b67c) chore(deps): bump `express`, `follow-redirects`, and `webpack-dev-middleware` (#12880)
* [a38cab742](https://github.com/argoproj/argo-workflows/commit/a38cab742cb29e4ab97ad1c57325b0564b32f45e) chore(deps): bump `undici` from 5.28.3 to 5.28.4 in /ui (#12891)
* [ae8e2e526](https://github.com/argoproj/argo-workflows/commit/ae8e2e526d1e9fa0f47693eaa805938b2db57704) fix: run linter on docs
* [d08a1c2f2](https://github.com/argoproj/argo-workflows/commit/d08a1c2f2a1b536f17a606e2bfea1a92fc060636) fix: linted typescript files
* [bf0174dba](https://github.com/argoproj/argo-workflows/commit/bf0174dba83300dddcf8340492914c750c26efb2) fix: `insecureSkipVerify` for `GetUserInfoGroups` (#12982)
* [2df039b0b](https://github.com/argoproj/argo-workflows/commit/2df039b0b66abbe3b59f89d0879da2d4135bcaa8) fix(ui): default to `main` container name in event source logs API call (#12939)
* [0f3a00d7f](https://github.com/argoproj/argo-workflows/commit/0f3a00d7fa7fa37a3a56d1576ce441a3049303cf) fix(build): close `pkg/apiclient/_.secondary.swagger.json` (#12942)
* [f1af3263c](https://github.com/argoproj/argo-workflows/commit/f1af3263c97065b7fff32669a98e0a5ccb4b5726) fix: correct order in artifactGC error log message (#12935)
* [627069692](https://github.com/argoproj/argo-workflows/commit/6270696921d66831d639a8c911d56fcf2066eb2a) fix: workflows that are retrying should not be deleted (Fixes #12636) (#12905)
* [caa339be2](https://github.com/argoproj/argo-workflows/commit/caa339be2dd23654bf9a347810fac243185e7679) fix: change fatal to panic.  (#12931)
* [fb08ad044](https://github.com/argoproj/argo-workflows/commit/fb08ad044ed9ed30b18de5de27a4ea12f49e7511) fix: Correct log level for agent containers (#12929)
* [30a756e9e](https://github.com/argoproj/argo-workflows/commit/30a756e9e3655bb7025cc1692136d5f93ed95033) fix(deps): upgrade x/net to v0.23.0. Fixes CVE-2023-45288 (#12921)
* [b0120579d](https://github.com/argoproj/argo-workflows/commit/b0120579dd06c4a351a32cedfe3ecdff16aae73e) fix(deps): upgrade `http2` to v0.24. Fixes CVE-2023-45288 (#12901)
* [de840948c](https://github.com/argoproj/argo-workflows/commit/de840948ce90687cf2b9a7820c2a6e3f5bee2823) fix(deps): upgrade `crypto` from v0.20 to v0.22. Fixes CVE-2023-42818 (#12900)
* [aa2bd8f3e](https://github.com/argoproj/argo-workflows/commit/aa2bd8f3ee2a5eee0c531a213b9975ca35f0f0dd) fix: use multipart upload method to put files larger than 5Gi to OSS. Fixes #12877 (#12897)
* [c5b4935fa](https://github.com/argoproj/argo-workflows/commit/c5b4935fab36ae12c3fcb66daf3a9b1f8c610723) fix: make sure Finalizers has chance to be removed. Fixes: #12836 (#12831)
* [774388a7b](https://github.com/argoproj/argo-workflows/commit/774388a7b0410ca5a94b799a5f7bfabc04333e3b) fix(test): wait enough time to Trigger Running Hook. Fixes: #12844 (#12855)
* [7821fdd0a](https://github.com/argoproj/argo-workflows/commit/7821fdd0a5dd36dfeadeeab9ebb7ba67c7d4d137) fix: terminate workflow should not get throttled Fixes #12778 (#12792)
* [e0c16ff0f](https://github.com/argoproj/argo-workflows/commit/e0c16ff0f52fb29138afb539d1a6b2f296d4ef32) fix: pass dnsconfig to agent pod. Fixes: #12824 (#12825)
* [82d14db2e](https://github.com/argoproj/argo-workflows/commit/82d14db2e50f7996f760772a7f538f1da2b93291) fix(deps): upgrade `undici` from 5.28.2 to 5.28.3 due to CVE (#12763)
* [9eb269d73](https://github.com/argoproj/argo-workflows/commit/9eb269d735fff855a6c20b46b396a8b4475a553a) fix(deps): upgrade `pgx` from 4.18.1 to 4.18.2 due to CVE (#12753)
* [6bd6a6373](https://github.com/argoproj/argo-workflows/commit/6bd6a63736a89edc36e4c0e07588e663fad08c4a) fix: inline template loops should receive more than the first item. Fixes: #12594 (#12628)
* [1f5bb49ce](https://github.com/argoproj/argo-workflows/commit/1f5bb49ce7f8209fbd108598edc9d58eae4a23e5) fix: workflow stuck in running state when using activeDeadlineSeconds on template level. Fixes: #12329 (#12761)
* [1a259cb11](https://github.com/argoproj/argo-workflows/commit/1a259cb11e059ff1ce1f0c1e29215ee8b913dc9e) fix(ui): show correct podGC message for deleteDelayDuration. Fixes: #12395 (#12784)
* [982038a88](https://github.com/argoproj/argo-workflows/commit/982038a88b764b497b4cf8a5e5934b6f4adaa517) fix(hack): various fixes & improvements to cherry-pick script (#12714)
* [c5ebbcf3a](https://github.com/argoproj/argo-workflows/commit/c5ebbcf3a11e44ddcdc4454dcfbeb74c17a9aee6) fix: make WF global parameters available in retries (#12698)
* [56ff88e02](https://github.com/argoproj/argo-workflows/commit/56ff88e02fd1e51a832c8ba95438d9b7284c98b7) fix: find correct retry node when using `templateRef`. Fixes: #12633 (#12683)
* [389492b4c](https://github.com/argoproj/argo-workflows/commit/389492b4cd95ca37edfc8a4b210b769e2c057a39) fix: Patch taskset with subresources to delete completed node status.… (#12620)
* [6194b8ada](https://github.com/argoproj/argo-workflows/commit/6194b8ada7ccf981084058c10dac411b44a695f9) fix(typo): fix some typo (#12673)
* [6cda00d2e](https://github.com/argoproj/argo-workflows/commit/6cda00d2e733ee40b2ae6d2c4f55ca50be72a8fd) fix(controller): re-allow changing executor `args` (#12609)
* [c590b2ef5](https://github.com/argoproj/argo-workflows/commit/c590b2ef564d25a7fef94803a0d03610a060dfec) fix(controller): add missing namespace index from workflow informer (#12666)
* [42ce47626](https://github.com/argoproj/argo-workflows/commit/42ce47626e669ace4011feb59f786c9d07561a39) fix: pass through burst and qps for auth.kubeclient (#12575)
* [4f8dd2ee7](https://github.com/argoproj/argo-workflows/commit/4f8dd2ee7d716ba2fc9e08edd013acb66bc9494c) fix: artifact subdir error when using volumeMount (#12638)
* [3cd016b00](https://github.com/argoproj/argo-workflows/commit/3cd016b004fbc57360b8b23989fc492ae7dd4313) fix: Allow valueFrom in dag arguments parameters. Fixes #11900 (#11902)
* [c15a75b00](https://github.com/argoproj/argo-workflows/commit/c15a75b0076a6a69be0d0f0efb4c6129d3732ec5) fix(resources): improve ressource accounting. Fixes #12468 (#12492)
* [83a49b4b9](https://github.com/argoproj/argo-workflows/commit/83a49b4b9638b160c9320cd0e808179c31482ee5) fix: upgrade expr-lang. Fixes #12037 (#12573)
* [bc7889be3](https://github.com/argoproj/argo-workflows/commit/bc7889be398378bd1875d8ae0532c437695652e2) fix: make etcd errors transient (#12567)
* [b9a22f876](https://github.com/argoproj/argo-workflows/commit/b9a22f8764e69c4feb6a18aab5ea55782180c282) fix: update minio chart repo (#12552)
* [574fd3ad2](https://github.com/argoproj/argo-workflows/commit/574fd3ad23d253d43757c47a6786350826c354e1) fix: add resource quota evaluation timed out to transient (#12536)
* [93e981d78](https://github.com/argoproj/argo-workflows/commit/93e981d78bc32a2ac599c63927ed3116b9cb51f8) fix: prevent update race in workflow cache (Fixes #9574) (#12233)
* [5f4845dbc](https://github.com/argoproj/argo-workflows/commit/5f4845dbc1415e1d0875f0361d8b7225086666d0) fix: Fixed mutex with withSequence in http template broken. Fixes #12018 (#12176)
* [790c0a4d1](https://github.com/argoproj/argo-workflows/commit/790c0a4d14b821af9942a590239ece9f7c30f18d) fix: SSO with Jumpcloud "email_verified" field #12257 (#12318)
* [e1bb99c3c](https://github.com/argoproj/argo-workflows/commit/e1bb99c3c33263d183423ce230e23d803c5fef5f) fix: wrong values are assigned to input parameters of workflowtemplat… (#12412)
* [c9ad89985](https://github.com/argoproj/argo-workflows/commit/c9ad899856529946087ab58fee949af144221657) fix: http template host header rewrite(#12385) (#12386)
* [e6ea4b147](https://github.com/argoproj/argo-workflows/commit/e6ea4b147d761c6118febaabd0f9e05e427185d3) fix: ensure workflow wait for onExit hook for DAG template (#11880) (#12436)
* [7db24e009](https://github.com/argoproj/argo-workflows/commit/7db24e009c0621c95a8e59cf54263df694252255) fix: move log with potential sensitive data to debug loglevel. Fixes: #12366 (#12368)
* [9540f8e0f](https://github.com/argoproj/argo-workflows/commit/9540f8e0f982052584c0080d04ba967703ec3485) fix: resolve output artifact of steps from expression when it refers … (#12320)
* [adf368514](https://github.com/argoproj/argo-workflows/commit/adf368514563d446c5ce8a729caec77320cf2862) fix: delete pending pod when workflow terminated  (#12196)
* [fedfb3790](https://github.com/argoproj/argo-workflows/commit/fedfb3790ad052587b39fa03fee6daf2f15876ea) fix: create dir when input path is not exist in oss (#12323)
* [a68e1f053](https://github.com/argoproj/argo-workflows/commit/a68e1f0530ff1b0fd688a1d05c1d8d126ba3bd79) fix: return failed instead of success when no container status (#12197)
* [eb9bbc8aa](https://github.com/argoproj/argo-workflows/commit/eb9bbc8aac953978371feca37605803bf654f49a) fix: Changes to workflow semaphore does work #12194 (#12284)
* [731366411](https://github.com/argoproj/argo-workflows/commit/731366411a630a7565c3703956b18395a4fc78fd) fix: properly resolve exit handler inputs (fixes #12283) (#12288)
* [58418906f](https://github.com/argoproj/argo-workflows/commit/58418906f2e8406d2e49f59545b49cb10c9d32b4) fix: Add identifiable user agent in API client. Fixes #11996 (#12276)
* [d6c5ed078](https://github.com/argoproj/argo-workflows/commit/d6c5ed078fbd9b9c21cebb97e27391529c7629fa) fix: remove deprecated function rand.Seed (#12271)
* [732b94a73](https://github.com/argoproj/argo-workflows/commit/732b94a73bf7bdb23ba27af5feb568383d0079a1) fix: leak stream (#12193)
* [6daa22b08](https://github.com/argoproj/argo-workflows/commit/6daa22b085625c23f47c34125257578c1ed74051) fix(server): allow passing loglevels as env vars to Server (#12145)
* [e8e9c2a48](https://github.com/argoproj/argo-workflows/commit/e8e9c2a48197c45dc6481f2637694ab524e458c4) fix: retry S3 on RequestError. Fixes #9914 (#12191)
* [18685ad8d](https://github.com/argoproj/argo-workflows/commit/18685ad8da825b9ccd660386fbba078edb9eb211) fix: Fix the Maximum Recursion Depth prompt link in the CLI. (#12015)
* [88d4e0f14](https://github.com/argoproj/argo-workflows/commit/88d4e0f14e85c7fbf4095536361e609ea08b4e77) fix: Only append slash when missing for Artifactory repoURL (#11812)
* [4627aa047](https://github.com/argoproj/argo-workflows/commit/4627aa047f9631babcabf093c8fc9de6a09dab21) fix: upgrade module for pull image in google cloud issue #9630 (#11614)
* [2368b37e6](https://github.com/argoproj/argo-workflows/commit/2368b37e6b773dacd52e8c8a3393af4747ac62d2) fix: Upgrade Go to v1.21 Fixes #11556 (#11601)
* [63af1c414](https://github.com/argoproj/argo-workflows/commit/63af1c414630ca263e55f221555e308921406cd7) fix(ui): ensure `package.json#name` is not the same as `argo-ui` (#11595)
* [c9f96f446](https://github.com/argoproj/argo-workflows/commit/c9f96f44693392ee82134da51324525e37802d52) fix: Devcontainer resets /etc/hosts (#11439) (#11440)
* [b23713e4b](https://github.com/argoproj/argo-workflows/commit/b23713e4b3db4ff847efd20a0765c88c1c22eb23) fix: make archived logs more human friendly in UI (#11420)
* [660bbb68f](https://github.com/argoproj/argo-workflows/commit/660bbb68f2e878700cb256898c68c75f00ee99d1) fix: Live workflow takes precedence during merge to correctly display in the UI (#11336)
* [a4ca4d27e](https://github.com/argoproj/argo-workflows/commit/a4ca4d27e92b83b52b3f79b850524f65b9b4a795) fix: add space to fix release action issue (#11160)
* [5fe8b37a6](https://github.com/argoproj/argo-workflows/commit/5fe8b37a63bcf03051c6c3fbe01580c344eda07d) fix: upgrade `argo-ui` components to latest (3.4 backport) (#12998)

<details><summary><h3>Contributors</h3></summary>

* Alan Clucas
* AlbeeSo
* AloysAqemia
* Andrei Shevchenko
* Anton Gilgur
* Bryce-Huang
* Byeonggon Lee
* Dennis Lawler
* Denys Melnyk
* Eduardo Rodrigues
* Helge Willum Thingvad
* Isitha Subasinghe
* João Pedro
* Raffael
* Ruin09
* Ryan Currah
* Shiwei Tang
* Shunsuke Suzuki
* Son Bui
* Tal Yitzhak
* Tianchu Zhao
* Weidong Cai
* Yang Lu
* Yuan (Terry) Tang
* Yuan Tang
* Yulin Li
* dependabot[bot]
* guangwu
* gussan
* ivancili
* jiangjiang
* jswxstw
* junkmm
* neosu
* shuangkun tian
* static-moonlight
* sycured

</details>

## v3.4.16 (2024-01-14)

Full Changelog: [v3.4.15...v3.4.16](https://github.com/argoproj/argo-workflows/compare/v3.4.15...v3.4.16)

### Selected Changes

* [910a9aabc](https://github.com/argoproj/argo-workflows/commit/910a9aabce5de6568b54350c181a431f8263605a) fix: Fix lint build
* [65befdeec](https://github.com/argoproj/argo-workflows/commit/65befdeecd871f965fc5b5213f269b6eb1fbce09) fix: Switch to upstream go-git. Fixes CVE-2023-49569 (#12515)

<details><summary><h3>Contributors</h3></summary>

* Yuan Tang

</details>

## v3.4.15 (2024-01-13)

Full Changelog: [v3.4.14...v3.4.15](https://github.com/argoproj/argo-workflows/compare/v3.4.14...v3.4.15)

### Selected Changes

* [fbf933563](https://github.com/argoproj/argo-workflows/commit/fbf9335635225eaa54420f3a520ef5c0043e5c34) fix: Resolve vulnerabilities in axios (#12470)
* [feb992505](https://github.com/argoproj/argo-workflows/commit/feb992505ecbd57ddd16b8d27b88e26045bf3588) fix: liveness check (healthz) type asserts to wrong type (#12353)
* [edb0a98c3](https://github.com/argoproj/argo-workflows/commit/edb0a98c31d7a459233131ae11e783ee45301ce6) fix(docs): release-3.4 readthedocs backport (#12474)

<details><summary><h3>Contributors</h3></summary>

* Jason Meridth
* Julie Vogelman
* Saravanan Balasubramanian
* Yuan Tang

</details>

## v3.4.14 (2023-11-27)

Full Changelog: [v3.4.13...v3.4.14](https://github.com/argoproj/argo-workflows/compare/v3.4.13...v3.4.14)

### Selected Changes

* [a34723324](https://github.com/argoproj/argo-workflows/commit/a3472332401f0cff56fd39293eebe3aeca7220ad) fix: Upgrade go-jose to v3.0.1
* [3201f61fb](https://github.com/argoproj/argo-workflows/commit/3201f61fba1a11147a55e57e57972c3df5758cc7) feat: Use WorkflowTemplate/ClusterWorkflowTemplate Informers when validating CronWorkflows (#11470)
* [d9a0797e7](https://github.com/argoproj/argo-workflows/commit/d9a0797e7778b4a109518fe9c4d9f9367c3beac8) fix: Resource version incorrectly overridden for wfInformer list requests. Fixes #11948 (#12133)
* [b3033ea11](https://github.com/argoproj/argo-workflows/commit/b3033ea1133b350e4cc702e1023dd8dc907526d6) Revert "fix: Add missing new version modal for v3.5 (#11692)"
* [f829cb52e](https://github.com/argoproj/argo-workflows/commit/f829cb52e2398f256829e4b4f49af671ee36c2a1) fix(ui): missing `uiUrl` in `ArchivedWorkflowsList` (#12172)
* [0c50de391](https://github.com/argoproj/argo-workflows/commit/0c50de3912e6fa4e725f67e1255280ad4a5475ac) fix: Revert "fix: regression in memoization without outputs (#12130)" (#12201)

<details><summary><h3>Contributors</h3></summary>

* Anton Gilgur
* Julie Vogelman
* Yuan (Terry) Tang
* Yuan Tang

</details>

## v3.4.13 (2023-11-03)

Full Changelog: [v3.4.12...v3.4.13](https://github.com/argoproj/argo-workflows/compare/v3.4.12...v3.4.13)

### Selected Changes

* [bdc1b2590](https://github.com/argoproj/argo-workflows/commit/bdc1b25900f44c194ab36d202821cec01ba96a73) fix: regression in memoization without outputs (#12130)
* [1cf98efef](https://github.com/argoproj/argo-workflows/commit/1cf98efef6e9afbbb99f6c481440d0199904b8b8) chore(deps): bump golang.org/x/oauth2 from 0.12.0 to 0.13.0 (#12000)
* [2a044bf8f](https://github.com/argoproj/argo-workflows/commit/2a044bf8f8af2614cce0d25d019ef669b855a230) fix: Upgrade axios to v1.6.0. Fixes #12085 (#12111)
* [37b5750dc](https://github.com/argoproj/argo-workflows/commit/37b5750dcb23916ddd6f18284b5b70fcfae872da) fix: Workflow controller crash on nil pointer  (#11770)
* [2c6c4d618](https://github.com/argoproj/argo-workflows/commit/2c6c4d61822493a627b13874987e20ec43d8ee26) fix: conflicting type of "workflow" logging attribute (#12083)
* [ade6fb4d7](https://github.com/argoproj/argo-workflows/commit/ade6fb4d72c98f73486d19a147df5c4919f43c99) fix: oss list bucket return all records (#12084)

<details><summary><h3>Contributors</h3></summary>

* Alan Clucas
* Cheng Wang
* Vasily Chekalkin
* Yuan (Terry) Tang
* Yuan Tang
* dependabot[bot]
* shuangkun tian

</details>

## v3.4.12 (2023-10-19)

Full Changelog: [v3.4.11...v3.4.12](https://github.com/argoproj/argo-workflows/compare/v3.4.11...v3.4.12)

### Selected Changes

* [11e61a8fe](https://github.com/argoproj/argo-workflows/commit/11e61a8fe81dd3d110a6bce2f5887f5f9cd3cf3c) fix(ui): remove "last month" default date filter mention from New Version Modal (#11982)
* [f87aba36a](https://github.com/argoproj/argo-workflows/commit/f87aba36a6a858fc5c0b1e43f9ea78e4372c0ccd) feat: filter sso groups based on regex (#11774)
* [b23647a10](https://github.com/argoproj/argo-workflows/commit/b23647a10eb8eea495c28e71d2822ea289a4370b) fix: Fix gRPC and HTTP2 high vulnerabilities (#11986)
* [aa8e6937e](https://github.com/argoproj/argo-workflows/commit/aa8e6937e3e0c66ee10c11a29828f65358ac3622) chore(deps): bump react-datepicker from 4.11.0 to 4.12.0 in /ui (#11147)
* [7979bb0db](https://github.com/argoproj/argo-workflows/commit/7979bb0db73669a34b90d617589a2d1cf690d7c2) chore(deps): bump github.com/go-sql-driver/mysql from 1.7.0 to 1.7.1 (#11007)
* [df23b7e5e](https://github.com/argoproj/argo-workflows/commit/df23b7e5ef16487703e25c8c04bee860fa30d07c) chore(deps): bump cronstrue from 2.27.0 to 2.28.0 in /ui (#11329)
* [a11f8f9be](https://github.com/argoproj/argo-workflows/commit/a11f8f9be9593270337c7870ba4a94c52ef451d1) chore(deps): bump github.com/antonmedv/expr from 1.12.5 to 1.12.6 (#11365)
* [4eb3c116d](https://github.com/argoproj/argo-workflows/commit/4eb3c116d22aa45446e506581fa2313f4cf8f081) chore(deps): bump superagent from 8.0.9 to 8.1.2 in /ui (#11590)
* [d25897ece](https://github.com/argoproj/argo-workflows/commit/d25897eced31dd2327c4ee89cf7aef46f39ee928) chore(deps): upgrade `monaco-editor` to 0.30 (#11593)
* [cac4f0cbd](https://github.com/argoproj/argo-workflows/commit/cac4f0cbd6ea6972a3a5f8bb163a97d5838db9c3) chore(deps): bump docker/build-push-action from 4 to 5 (#11830)
* [bd057ed18](https://github.com/argoproj/argo-workflows/commit/bd057ed1889a45df3ec5cd916e692efb81d81293) chore(deps): bump docker/login-action from 2 to 3 (#11827)
* [370949dc5](https://github.com/argoproj/argo-workflows/commit/370949dc560fc2c28cdcd709a34b2c00b0f034dc) fix: fail test on pr #11368 (#11576)
* [46297cad7](https://github.com/argoproj/argo-workflows/commit/46297cad798cb48627baf23d548a2c7e595ed316) fix: Add missing new version modal for v3.5 (#11692)
* [5b30e3034](https://github.com/argoproj/argo-workflows/commit/5b30e3034cf29652376c6053ac5c4bbe40b8b95c) fix: Health check from lister not apiserver (#11375)
* [f553d7f06](https://github.com/argoproj/argo-workflows/commit/f553d7f06a8ccbde4a722f32288eda5bb07650de) fix(ui): don't use `Buffer` for FNV hash (#11766)
* [9ebb70a26](https://github.com/argoproj/argo-workflows/commit/9ebb70a26ea5f7726e51739bfa10d82aa0703f9b) fix: Correct limit in WorkflowTaskResultInformer List API calls. Fixes #11607 (#11722)
* [b8b70980c](https://github.com/argoproj/argo-workflows/commit/b8b70980c9218260c0e4aa2be4f4a243fd25b902) fix(ui): handle `undefined` dates in Workflows List filter (#11792)
* [d258bcabf](https://github.com/argoproj/argo-workflows/commit/d258bcabf3de43ca505c1d8b5236cfec78c51949) fix: close response body when request event-stream failed (#11818)
* [f6bd94af4](https://github.com/argoproj/argo-workflows/commit/f6bd94af4c4cb5e1d217dc72e81cec6940da5daa) fix(ui): merge WF List FTU Panel with New Version Modal (#11742)
* [71ad0a23c](https://github.com/argoproj/argo-workflows/commit/71ad0a23c35fa47315977a446763560bbc6dbeea) fix: Fixed workflow template skip whitespaced parameters. Fixes #11767 (#11781)
* [a08d73a8f](https://github.com/argoproj/argo-workflows/commit/a08d73a8f2e4b8c350822ab113905a2a3e58416f) fix: add prometheus label validation for realtime gauge metric (#11825)
* [3080ab837](https://github.com/argoproj/argo-workflows/commit/3080ab83773f855ef0bf62d11eb26022c6a85233) fix: shouldn't fail to run cronworkflow because previous got shutdown on its own (race condition) (#11845)
* [20f1c6b3b](https://github.com/argoproj/argo-workflows/commit/20f1c6b3b1dfb50540572c07b1a50e1ce0870d7a) fix: when key not present assume NodeRunning. Fixes 11843 (#11847)
* [4b6cdaeec](https://github.com/argoproj/argo-workflows/commit/4b6cdaeec934086796283ddfb09597bfd4d08774) fix: Fixed running multiple workflows with mutex and memo concurrently is broken (#11883)
* [396be7252](https://github.com/argoproj/argo-workflows/commit/396be72529bb4a05d05f9ab9f471421971d81c88) fix: Automate nix updates with renovate (#11887)
* [96e44c01d](https://github.com/argoproj/argo-workflows/commit/96e44c01d67cf4006486bc2ab2a6f2f6e6247600) fix(ui): `ClipboardText` tooltip properly positioned (#11946)
* [1447472ff](https://github.com/argoproj/argo-workflows/commit/1447472ff1ddfb0853154ff6132cf421cf54831e) fix(ui): faulty `setInterval` -> `setTimeout` in clipboard (#11945)
* [c543932b9](https://github.com/argoproj/argo-workflows/commit/c543932b9870b40cd9b2ad61c285afe90c8ffc29) fix: Permit enums w/o values. Fixes #11471. (#11736)
* [142f5bd65](https://github.com/argoproj/argo-workflows/commit/142f5bd653251e9504fd9f71fbd7626d196d8a2b) fix(windows): prevent infinite run. Fixes #11810 (#11993)
* [4d09777d3](https://github.com/argoproj/argo-workflows/commit/4d09777d3d7b524e043b5e48fb3761e527eb2ea8) fix: remove WorkflowSpec VolumeClaimTemplates patch key (#11662)
* [fe880539a](https://github.com/argoproj/argo-workflows/commit/fe880539ab4d0e3fecdf35b9d9a7c11f3adca117) fix: Fixed workflow onexit condition skipped when retry. Fixes #11884 (#12019)
* [61f00ba56](https://github.com/argoproj/argo-workflows/commit/61f00ba568e7ecbe2c164fb5d114493029c2e47f) fix: suppress error about unable to obtain node (#12020)

<details><summary><h3>Contributors</h3></summary>

* Alec Rabold
* Anton Gilgur
* Basanth Jenu H B
* Isitha Subasinghe
* Julie Vogelman
* Matt Farmer
* Michael Weibel
* Ruin09
* Son Bui
* Takumi Sue
* Thearas
* Tim Collins
* Weidong Cai
* Yuan (Terry) Tang
* dependabot[bot]
* happyso

</details>

## v3.4.11 (2023-09-06)

Full Changelog: [v3.4.10...v3.4.11](https://github.com/argoproj/argo-workflows/compare/v3.4.10...v3.4.11)

### Selected Changes

* [ee939bbd2](https://github.com/argoproj/argo-workflows/commit/ee939bbd2d8950a2fa1badd7cfad3b88c039da26) fix: Support OOMKilled with container-set. Fixes #10063.  FOR 3.4.11 only (#11757)
* [e731cc077](https://github.com/argoproj/argo-workflows/commit/e731cc07797beb6cdaaf6a1d495cb77aab24bfe6) fix: Argo DB init conflict when deploy workflow-controller with multiple replicas #11177 (#11569)
* [aab216029](https://github.com/argoproj/argo-workflows/commit/aab216029c585bccc1e76ec40c413d80dd84ffa9) fix: override storedWorkflowSpec when override parameter (#11631) (#11634)
* [1662e7eae](https://github.com/argoproj/argo-workflows/commit/1662e7eaee2c41c60be8b8dd3dd77d1e33c97b4a) fix: Fix merge conflicts and unit tests
* [edfde1653](https://github.com/argoproj/argo-workflows/commit/edfde165393fdf8f782a3ab8b9551f4de1009b4d) fix: Apply the creator labels about the user who resubmitted a Workflow (#11415)
* [b0909c69e](https://github.com/argoproj/argo-workflows/commit/b0909c69ee79a29917aa6c21b3b724cd51ff737d) fix: upgrade base image for security and build support arm64 #10435 (#11613)
* [80a0cd5e0](https://github.com/argoproj/argo-workflows/commit/80a0cd5e033b0aa2111e6bb7aa13706b1f7ff332) fix: deprecated Link(Help-Contact) Issue (#11627)
* [51107db24](https://github.com/argoproj/argo-workflows/commit/51107db247ad40bdc63ee662cf3fd2bfe5a5c458) fix: do not process withParams when task/step Skipped. Fixes #10173 (#11570)
* [453f84682](https://github.com/argoproj/argo-workflows/commit/453f84682f2469fff3bfdeaa593f068721d04b36) fix: Print valid JSON/YAML when workflow list empty #10873 (#11681)
* [a2a045c37](https://github.com/argoproj/argo-workflows/commit/a2a045c3768308fd1c51391f3afce2c167ef07c5) fix: argo logs completion (#11645)
* [579a8e2d8](https://github.com/argoproj/argo-workflows/commit/579a8e2d8b1dbfbb4a61fb140041e9bca5b34ec1) fix: Change node in paramScope to taskNode at executeDAG (#11422) (#11682)
* [a85c4b860](https://github.com/argoproj/argo-workflows/commit/a85c4b8605486e1098a31aaabc733e7860360d9f) fix(ui): don't use anti-pattern in CheckboxFilter (#11739)
* [ea8bf4dd1](https://github.com/argoproj/argo-workflows/commit/ea8bf4dd1f6936e7412ea01fc34a6efc7acb0bcb) fix: cron workflow initial filter value. Fixes #11685 (#11686)
* [f3f06f70a](https://github.com/argoproj/argo-workflows/commit/f3f06f70ac99bfe8e12218f0b44c80bcc1446de8) fix: Make defaultWorkflow hooks work more than once (#11693)
* [27cd582c8](https://github.com/argoproj/argo-workflows/commit/27cd582c879036e22a692a12136ca1d635b89c9b) fix: TERM signal was catched but not handled properly, which causing … (#11582)
* [33b3a1bc6](https://github.com/argoproj/argo-workflows/commit/33b3a1bc6b0edb791086f72c6ca6dc984363a48e) fix(workflow): match discovery burst and qps for `kubectl` with upstream kubectl binary (#11603)
* [d3e66c749](https://github.com/argoproj/argo-workflows/commit/d3e66c749e9ff43e0fe3b8a931907d47d839b1e6) fix: offset reset when pagination limit onchange (#11703)
* [02d1e1f8f](https://github.com/argoproj/argo-workflows/commit/02d1e1f8f380046580b4108b4e3faaa00b1006f0) fix: always fail dag when shutdown is enabled. Fixes #11452 (#11667)
* [d20363c1e](https://github.com/argoproj/argo-workflows/commit/d20363c1e5850e78ffabc9afc6221e96ed1497ad) fix: add guard against NodeStatus. Fixes #11102  (#11665)
* [3b9b9ad43](https://github.com/argoproj/argo-workflows/commit/3b9b9ad430d723be162629f5ccda338fb759da39) fix: Fixed parent level memoization broken. Fixes #11612 (#11623) (#11660)

<details><summary><h3>Contributors</h3></summary>

* Alan Clucas
* Anton Gilgur
* Antonio Gurgel
* Cheng Wang
* Isitha Subasinghe
* Jinsu Park
* LEE EUI JOO
* Ruin09
* Son Bui
* Suraj Banakar(बानकर) | スラジ
* Yuan (Terry) Tang
* Yuan Tang
* gussan
* happyso
* younggil
* 一条肥鱼
* 张志强

</details>

## v3.4.10 (2023-08-15)

Full Changelog: [v3.4.9...v3.4.10](https://github.com/argoproj/argo-workflows/compare/v3.4.9...v3.4.10)

### Selected Changes

* [bd6cd2555](https://github.com/argoproj/argo-workflows/commit/bd6cd2555d1bb0e57a34ce74b0add36cb7fb6c76) fix: Fixed memoization is unchecked after mutex synchronization. Fixes #11219 (#11578)
* [ad92818d7](https://github.com/argoproj/argo-workflows/commit/ad92818d782c94ce126d08d8dfd1907ed8ead030) fix(server): don't grab SAs if SSO RBAC is not enabled (#11426)
* [bfbee8d17](https://github.com/argoproj/argo-workflows/commit/bfbee8d17cf4ff120dce522790fcd8d7cbd3aa23) fix: Upgrade hdfs and rpc module #10030 (#11543)
* [83756dc0f](https://github.com/argoproj/argo-workflows/commit/83756dc0fa9f597c1740ca0ce0123652da31cf91) fix: Flaky test about lifecycle hooks (#11534)
* [fed2d1e02](https://github.com/argoproj/argo-workflows/commit/fed2d1e028982431ca1a9b3a4dc76bec5db84abf) chore(deps): bump github.com/sirupsen/logrus from 1.9.2 to 1.9.3 (#11200)
* [c5dbb3b35](https://github.com/argoproj/argo-workflows/commit/c5dbb3b35bfd3265a4dd921b17676b2b8b784c00) fix: Upgraded docker distribution go package to v2.8.2 for fixing a high vulnerability (#11554)
* [1513e22ed](https://github.com/argoproj/argo-workflows/commit/1513e22ed4600e2107e8ffc6b3b43e29af88d453) fix: prevent stdout from disappearing in script templates. Fixes #11330 (#11368)
* [1984c1ae4](https://github.com/argoproj/argo-workflows/commit/1984c1ae47a126440076653c660e521a9c548074) fix: Update config for metrics, throttler, and entrypoint. Fixes #11542, #11541 (#11553)
* [8c7489f6c](https://github.com/argoproj/argo-workflows/commit/8c7489f6c192d09564eb994d94c57c00d41450ad) fix: workflow-controller-configmap/parallelism setting not working in… (#11546)
* [dcabc5059](https://github.com/argoproj/argo-workflows/commit/dcabc5059eef6c51a54d6cac3796c6a0b25d3e68) fix: Switch to use kong/httpbin to support arm64. Fixes #10427 (#11533)
* [bbc2f9757](https://github.com/argoproj/argo-workflows/commit/bbc2f975724ff92a6861850df502a6c14d7dd04f) fix: Added vulnerability fixes for gorestlful gopkg & OS vulnerabilities in golang:1.20-alpine3.16 (#11538)
* [f4ede0a47](https://github.com/argoproj/argo-workflows/commit/f4ede0a470b94149852c5334cf130649f331112e) fix: Ensure target Workflow hooks not nil (#11521) (#11535)
* [aff72d098](https://github.com/argoproj/argo-workflows/commit/aff72d0984098d16e1458c8ced9c6d775e72930d) fix: azure hasLocation incorporates endpoint. Fixes #11512 (#11513)
* [579766898](https://github.com/argoproj/argo-workflows/commit/5797668981a08ff441a1b5a7a449cdba2de7fa33) fix: valueFrom in template parameter should be overridable. Fixes 10182 (#10281)
* [9e1d1e531](https://github.com/argoproj/argo-workflows/commit/9e1d1e531ce36ea58b812c0d8d114d227facf1fe) fix: Fixed UI workflowDrawer information link broken. Fixes #11494 (#11495)
* [ecf67d936](https://github.com/argoproj/argo-workflows/commit/ecf67d93624364a1460f34b735c528181e7ff17d) fix: Datepicker Style Malfunction Issue. Fixes #11476 (#11480)
* [d30c5875c](https://github.com/argoproj/argo-workflows/commit/d30c5875c8383643c3951cdde706b418ae86a678) fix: UI toolbar sticky (#11444)
* [214def687](https://github.com/argoproj/argo-workflows/commit/214def68766eee20196d773f0ae6cf707054023f) fix(controller): Enable dummy metrics server on non-leader workflow controller (#11295)
* [1bcdba429](https://github.com/argoproj/argo-workflows/commit/1bcdba4295125812cc27c0fed5ad831472988597) fix(windows): Propagate correct numerical exitCode under Windows (Fixes #11271) (#11276)
* [b694dcc4a](https://github.com/argoproj/argo-workflows/commit/b694dcc4a38f7a24eced052d16fdb3c14228f1f5) fix(controller): Drop Checking daemoned children without nodeID (Fixes #10960) (#10974)

<details><summary><h3>Contributors</h3></summary>

* Anton Gilgur
* Christoph Buchli
* Josh Soref
* LilTwo
* Roel Arents
* Rui Chen
* Ruin09
* Son Bui
* Vedant Shrotria
* Yuan (Terry) Tang
* YunCow
* boiledfroginthewell
* dependabot[bot]
* gussan
* sakai-ast
* younggil

</details>

## v3.4.9 (2023-07-20)

Full Changelog: [v3.4.8...v3.4.9](https://github.com/argoproj/argo-workflows/compare/v3.4.8...v3.4.9)

### Selected Changes

* [b76329f3a](https://github.com/argoproj/argo-workflows/commit/b76329f3a2dedf4c76a9cac5ed9603ada289c8d0) fix: Fix Azure test
* [163d3d4f1](https://github.com/argoproj/argo-workflows/commit/163d3d4f1530e3e18cfcce1311d5d6d732364326) fix: download subdirs in azure artifact. Fixes #11385 (#11394)
* [5836caef1](https://github.com/argoproj/argo-workflows/commit/5836caef1c62a1b9b4949334425ec3d71f55498a) chore(deps): bump golang.org/x/sync from 0.2.0 to 0.3.0 (#11262)
* [3a6975549](https://github.com/argoproj/argo-workflows/commit/3a69755494d3cdff8be8d35c4b25ed35178b30cf) chore(deps): bump golang.org/x/oauth2 from 0.8.0 to 0.9.0 (#11228)
* [95bf965ca](https://github.com/argoproj/argo-workflows/commit/95bf965ca3b8721005be5b27ff88ea7ad60e6b85) chore(deps): bump google.golang.org/api from 0.124.0 to 0.128.0 (#11229)
* [894fcba12](https://github.com/argoproj/argo-workflows/commit/894fcba12c6fffbbbc42fda39534488ce6c3bc08) chore(deps): bump google.golang.org/api from 0.122.0 to 0.124.0 (#11142)
* [912c41f96](https://github.com/argoproj/argo-workflows/commit/912c41f96c105fbaa8e69c76b7589b0398198b35) fix: Setup /etc/hosts for running inside devcontainer (#11104)
* [dcc4f5851](https://github.com/argoproj/argo-workflows/commit/dcc4f585150e1c4cecdc72e53711f7d7eaaec089) fix: Make devcontainer able to pre-commit (#11153)
* [5ef42ee72](https://github.com/argoproj/argo-workflows/commit/5ef42ee729d18a36d9a7c9785112de8c8ad5c3ee) fix: check hooked nodes in executeWfLifeCycleHook and executeTmplLifeCycleHook (#11113, #11117) (#11176)
* [6f57159a1](https://github.com/argoproj/argo-workflows/commit/6f57159a1788dc1e68418749917f3d2151a64a62) fix: Remove 401 Unauthorized when customClaimGroup retrieval fails, Fixes #11032 (#11033)
* [e6d19c980](https://github.com/argoproj/argo-workflows/commit/e6d19c9803db5529e7cb8877bd68e2b1e48282d7) fix: prevent memoization accessing wrong config-maps (#11225)
* [12a8b6f40](https://github.com/argoproj/argo-workflows/commit/12a8b6f4004843e2b79bf336cc5b2e57e55a24bd) fix: Treat "connection refused" error as a transient network error. (#11237)
* [57dbc6edf](https://github.com/argoproj/argo-workflows/commit/57dbc6edffbaf79101c57ad657bb0dad57560c22) fix: untar empty directories (#11240)
* [1a3f17f74](https://github.com/argoproj/argo-workflows/commit/1a3f17f7432c97dd25baeef906b0d38e12028b99) fix: Allow hooks to be specified in workflowDefaults (#11214)
* [1109ab498](https://github.com/argoproj/argo-workflows/commit/1109ab498a50454a2a15afbfd9d178b1a4e6c807) fix: Support inputs for inline steps templates (#11074)
* [def9d653e](https://github.com/argoproj/argo-workflows/commit/def9d653e893a1328fa60b25c82015c0701dc285) fix: Update Bitbucket SSH host key (#11245)
* [c214aaaf7](https://github.com/argoproj/argo-workflows/commit/c214aaaf73760388ae0c6504c13bd6d06f7e7a24) fix: Upgrade windows container to ltsc2022 (#11246)
* [a7db62352](https://github.com/argoproj/argo-workflows/commit/a7db62352743ad0e49b4f9488d9c16159fe08ddf) fix: do not delete pvc when max parallelism has been reached. Fixes #11119 (#11138)
* [78acc81a7](https://github.com/argoproj/argo-workflows/commit/78acc81a75c2db74ec0736ef41561b10cf7a6002) fix: fix bugs in throttler and syncManager initialization in WorkflowController (#11210)
* [f7b307222](https://github.com/argoproj/argo-workflows/commit/f7b307222cd4c5efeb9ee10ece1a4cc04f35085a) fix: Argo DB init conflict when deploy workflow-controller with multiple replicas #11177 (#11178)
* [1222da43e](https://github.com/argoproj/argo-workflows/commit/1222da43e2ef755b828b4cfa29660957c5f4beb3) fix: Azure input artifact support optional. Fixes #11179 (#11235)
* [d5e7a554c](https://github.com/argoproj/argo-workflows/commit/d5e7a554c064fbe7aef8c71e98823575b1323f96) fix: use unformatted templateName as args to PodName. Fixes #11250 (#11251)
* [709170efe](https://github.com/argoproj/argo-workflows/commit/709170efe4feb859a1e8f024d2395fcda46b15d0) fix: Add ^ to semver version (#11310)
* [67064561d](https://github.com/argoproj/argo-workflows/commit/67064561d169b7bb7a62278976b8c786179a48c0) fix: Pin semver to 7.5.2. Fixes SNYK-JS-SEMVER-3247795 (#11306)
* [f7bf6ee4c](https://github.com/argoproj/argo-workflows/commit/f7bf6ee4c968f7d6cdf0e3e71a37b13eb5328da4) fix: Correct limit in controller List API calls. Fixes #11134 (#11343)
* [3e17d5693](https://github.com/argoproj/argo-workflows/commit/3e17d56930a10c5ac1e00f00e41c9d1c011d645a) fix: Enable the workflow created by a wftmpl to retry after manually stopped (#11355)

<details><summary><h3>Contributors</h3></summary>

* Abraham Bah
* Alan Clucas
* Anton Gilgur
* Cheng Wang
* Iain Lane
* James Slater
* Lan
* Lucas Heinlen
* Rachel Bushrian
* Roel Arents
* Tim Collins
* Tom Kahn
* Tore
* Yuan Tang
* dependabot[bot]
* smile-luobin
* toyamagu
* vanny96

</details>

## v3.4.8 (2023-05-25)

Full Changelog: [v3.4.7...v3.4.8](https://github.com/argoproj/argo-workflows/compare/v3.4.7...v3.4.8)

### Selected Changes

* [03c8829cb](https://github.com/argoproj/argo-workflows/commit/03c8829cbe61dc44db2e700421c874cf18752577) chore(deps): bump github.com/sirupsen/logrus from 1.9.0 to 1.9.2 (#11107)
* [179d2a95e](https://github.com/argoproj/argo-workflows/commit/179d2a95e941a0c3e0d812863cb5ee76dedba738) chore(deps): bump google.golang.org/api from 0.120.0 to 0.122.0 (#11089)
* [c65583a45](https://github.com/argoproj/argo-workflows/commit/c65583a4508a743abcc29fe3dabfcf756206113f) chore(deps): bump google.golang.org/api from 0.118.0 to 0.120.0 (#11008)
* [207533458](https://github.com/argoproj/argo-workflows/commit/207533458e8994a977c74273572c72953b853cd8) chore(deps): bump cronstrue from 2.26.0 to 2.27.0 in /ui (#11078)
* [f9c2d33dd](https://github.com/argoproj/argo-workflows/commit/f9c2d33ddf397d260706d327d426c7c22b729d3e) chore(deps): bump golang.org/x/crypto from 0.8.0 to 0.9.0 (#11068)
* [7e10cbc6f](https://github.com/argoproj/argo-workflows/commit/7e10cbc6ff1278baec21c60c83bfc30e5fe73d42) chore(deps): bump golang.org/x/oauth2 from 0.7.0 to 0.8.0 (#11058)
* [8135a4b1c](https://github.com/argoproj/argo-workflows/commit/8135a4b1c32ce4d24ad8ae617b71c1e4ec536b7a) chore(deps): bump golang.org/x/sync from 0.1.0 to 0.2.0 (#11041)
* [dd7432c21](https://github.com/argoproj/argo-workflows/commit/dd7432c21efa3391fb9027ae6ce3cc049b425f26) chore(deps): bump github.com/prometheus/client_golang from 1.15.0 to 1.15.1 (#11029)
* [c152e0169](https://github.com/argoproj/argo-workflows/commit/c152e0169937df97fca7cc3d446c6b02643efa98) chore(deps): bump golang.org/x/term from 0.7.0 to 0.8.0 (#11044)
* [5d888613c](https://github.com/argoproj/argo-workflows/commit/5d888613c19b985c1059d7f1b39769fbc79045ec) chore(deps): bump github.com/klauspost/pgzip from 1.2.5 to 1.2.6 (#11045)
* [0be306dde](https://github.com/argoproj/argo-workflows/commit/0be306ddec53ea940128f8b91207147d1e21a0f1) chore(deps): bump google.golang.org/api from 0.117.0 to 0.118.0 (#10933)
* [06a9df280](https://github.com/argoproj/argo-workflows/commit/06a9df28047a665f2261696e2531d66bfb841f9c) chore(deps): bump github.com/aliyun/aliyun-oss-go-sdk from 2.2.6+incompatible to 2.2.7+incompatible (#10753)
* [b175db558](https://github.com/argoproj/argo-workflows/commit/b175db5587933d8d9f7b6483b87fd8e1863b1f25) chore(deps): bump google.golang.org/api from 0.114.0 to 0.117.0 (#10878)
* [d7cdad322](https://github.com/argoproj/argo-workflows/commit/d7cdad3221170f1d566762471abe743067629a64) chore(deps): bump github.com/minio/minio-go/v7 from 7.0.51 to 7.0.52 (#10917)
* [2d5015752](https://github.com/argoproj/argo-workflows/commit/2d501575207323cc127dc51484873e5a7102ed49) chore(deps): bump github.com/prometheus/client_golang from 1.14.0 to 1.15.0 (#10916)
* [537527b8b](https://github.com/argoproj/argo-workflows/commit/537527b8be2f6a2c85f9836c8a583657d0a89444) chore(deps): bump golang.org/x/oauth2 from 0.6.0 to 0.7.0 (#10860)
* [366ff0f68](https://github.com/argoproj/argo-workflows/commit/366ff0f68e43b9aa63b62712898fc76fa1201aba) chore(deps): bump github.com/minio/minio-go/v7 from 7.0.50 to 7.0.51 (#10877)
* [c9eb1dc15](https://github.com/argoproj/argo-workflows/commit/c9eb1dc15fae3a805641a270ef81702867fab30d) chore(deps): bump golang.org/x/crypto from 0.7.0 to 0.8.0 (#10856)
* [73e90b62b](https://github.com/argoproj/argo-workflows/commit/73e90b62be0f4803b02d5bfcdfdd1bd891cfe3d7) chore(deps): bump cronstrue from 2.24.0 to 2.26.0 in /ui (#10855)
* [53adf3f81](https://github.com/argoproj/argo-workflows/commit/53adf3f81c6a42e243ca3e2ada968c52f0fc9006) chore(deps): bump peter-evans/create-pull-request from 4 to 5 (#10854)
* [146ac86b3](https://github.com/argoproj/argo-workflows/commit/146ac86b362bf9aa81a7021f5b46777c52146713) fix: Fixed path separator in .tgz output artifacts created on windows. Fixes #10562 (#11097)
* [f5f22cffb](https://github.com/argoproj/argo-workflows/commit/f5f22cffbe0e82bab318404be1804ec798318c52) fix: Disable unreliable test (#11105)
* [32c76c847](https://github.com/argoproj/argo-workflows/commit/32c76c8470c223249e6e154c71944875847d6af3) fix: allow azure blobs to not exist when deleting (#11070)
* [55dc31d7f](https://github.com/argoproj/argo-workflows/commit/55dc31d7f679b05f2bde5e85693196666420f7dd) fix: Update Bitbucket SSH host key (#11091)
* [cdaa65536](https://github.com/argoproj/argo-workflows/commit/cdaa65536d26b8b00dc345305dea5b9fd382f6d8) fix: Parameter overwritten does not work when resubmitting archived workflow (#11086) (#11087)
* [8c54d741d](https://github.com/argoproj/argo-workflows/commit/8c54d741da4a7ff52d449f8f39c2df6a7fbae9bb) fix: UI crashes when retrying a containerSet workflow. Fixes #11061 (#11073)
* [9ad94b88d](https://github.com/argoproj/argo-workflows/commit/9ad94b88df42d1c19bcc0cb46539c3e65a39ee0c) fix: ui getPodName should use v2 format by default (fixes #11015) (#11016)
* [92c7a4e9c](https://github.com/argoproj/argo-workflows/commit/92c7a4e9c0831331cd0ab7accd12a37164f62759) fix: handle panic from type assertion (#11040)
* [ccfddb70f](https://github.com/argoproj/argo-workflows/commit/ccfddb70ff5907539bc928d95d0cc808ba172838) fix: change pod OwnerReference to clean workflowtaskresults in large-scale scenarios (#11048)
* [eb3c7b828](https://github.com/argoproj/argo-workflows/commit/eb3c7b828e4fcfd79da322529aaefa988173b3e2) fix: Upgrade Go to v1.20. Fixes #11023 (#11027)
* [9f7e9b516](https://github.com/argoproj/argo-workflows/commit/9f7e9b51664340c907cb965e26214bd3e14377bf) fix: UI crashes after submitting workflows (#11018)
* [470daea44](https://github.com/argoproj/argo-workflows/commit/470daea449c7782c99c82add8572873d5c321a4a) fix: Generate useful error message when no expression on hook (#10919)
* [b3ea4e3bb](https://github.com/argoproj/argo-workflows/commit/b3ea4e3bb0fbdcb8c25236bd37f32c0dcfcc75a4) fix: Validate label values from workflowMetadata.labels to avoid controller crash (#10995)
* [ed0c3490c](https://github.com/argoproj/argo-workflows/commit/ed0c3490cc9f62cef228d31d09b01f282809b34e) fix: Replace expressions with placeholders in resource manifest template. Fixes #10924 (#10926)
* [3dbb9dc57](https://github.com/argoproj/argo-workflows/commit/3dbb9dc573f2b7096d67d45072e6775d1dedd437) fix: unable to connect cluster when AutomountServiceAccountToken is disabled. Fixes #10937 (#10945)
* [131a8541f](https://github.com/argoproj/argo-workflows/commit/131a8541f0bee8646c7c7701f7c205d06c86597d) fix: Check AlreadyExists error when creating PDB. Fixes #10942 (#10944)
* [4c425b4be](https://github.com/argoproj/argo-workflows/commit/4c425b4bed4a4835cb8a368fe2dc3435f983f795) fix: Check file size before saving to artifact storage. Fixes #10902 (#10903)
* [dbdbc746b](https://github.com/argoproj/argo-workflows/commit/dbdbc746b3c937058265e9884c9a97e98f8f8f63) fix: Incorrect pod name for inline templates. Fixes #10912 (#10921)
* [e803de523](https://github.com/argoproj/argo-workflows/commit/e803de52349f15c011788c641c3f8baa8ca068e1) fix: Workflow operation error. Fixes #10285 (#10886)
* [aeb080815](https://github.com/argoproj/argo-workflows/commit/aeb080815e375711cba60689e12b7f0a392ad6dd) fix: Validate label values from workflowMetadata to avoid controller crash. Fixes #10872 (#10892)
* [f98227cea](https://github.com/argoproj/argo-workflows/commit/f98227ceab1e8544abd736ab54a93914a192968e) fix: tableName is empty if wfc.session != nil (#10887)
* [7fc7b589a](https://github.com/argoproj/argo-workflows/commit/7fc7b589a6fd1b8dbe04b3aaaa04b1f3d7703372) fix: Flaky test about lifecycle hooks. Fixes #10897 (#10898)
* [8929ed63f](https://github.com/argoproj/argo-workflows/commit/8929ed63f619b7112c50efe7991818354a231d6e) fix: Allow script and container image to be set in templateDefault. Fixes #9633 (#10784)

<details><summary><h3>Contributors</h3></summary>

* Alan Clucas
* Alexander Crow
* GeunSam2 (Gray)
* Julie Vogelman
* Max Xu
* Or Shachar
* PeterKoegel
* Roel Arents
* RoryDoherty
* Takumi Sue
* Tore
* Yuan Tang
* dependabot[bot]
* shuangkun tian
* tooptoop4
* toyamagu
* yeicandoit

</details>

## v3.4.7 (2023-04-11)

Full Changelog: [v3.4.6...v3.4.7](https://github.com/argoproj/argo-workflows/compare/v3.4.6...v3.4.7)

### Selected Changes

* [f2292647c](https://github.com/argoproj/argo-workflows/commit/f2292647c5a6be2f888447a1fef71445cc05b8fd) fix: Upgrade docker to v20.10.24 for CVE-2023-28840, CVE-2023-28841, CVE-2023-28842 (#10868)
* [a3bfce20a](https://github.com/argoproj/argo-workflows/commit/a3bfce20a3200752aa5fb0ee378992755107f9c6) fix: Fix not working Running state lifecycle hooks in dag task. Fixes #9897 (#10307)
* [87b39105c](https://github.com/argoproj/argo-workflows/commit/87b39105cdb450127ef1a097a10ae3a6a833b5de) fix: make workflow status change after workflow level lifecycle hooks complete. Fixes #10743, #9591 (#10758)
* [672dcd9c2](https://github.com/argoproj/argo-workflows/commit/672dcd9c29596348452cc72c3dd2b33842755465) fix: Workflow stuck at running for failed init containers with other names. Fixes #10717 (#10849)
* [5988c1713](https://github.com/argoproj/argo-workflows/commit/5988c1713994ee2d69ccff4c7a945d32c5fe4d1f) fix: download specific version of kit. Fixes #10768 (#10841)
* [243ec1139](https://github.com/argoproj/argo-workflows/commit/243ec11398102c72aa87f8d2538402402da85d2d) fix: Resolve high severity vulnerabilities in UI deps (#10842)
* [09f5a149a](https://github.com/argoproj/argo-workflows/commit/09f5a149a980e0db2a2fa3f40afa932a9b0289fd) fix: Security upgrade ubuntu from 14.04 to trusty-20190515 (#10832)
* [f4b689cab](https://github.com/argoproj/argo-workflows/commit/f4b689cab0dddd1cbbee675526e650bf72c3e3b2) Revert "feat: Adds TimeZone column in cron list in UI - Fixes #10389 (#10390)"
* [2abca7fa5](https://github.com/argoproj/argo-workflows/commit/2abca7fa55c38e0e7a4363ca19d5708a0581791d) Revert "feat: Parse JSON structured logs in Argo UI. Fixes #6856 (#10145)"
* [7e0418980](https://github.com/argoproj/argo-workflows/commit/7e0418980634db75c79acc03d6a11fad365e75a6) Revert "feat: Surface container waiting reason and message (#10831)"
* [bcc1f332c](https://github.com/argoproj/argo-workflows/commit/bcc1f332cff6b1abaacd14e5209f8d159ea4925a) feat: Surface container waiting reason and message (#10831)
* [303572724](https://github.com/argoproj/argo-workflows/commit/3035727244747ace853112732fc426d891d7ad01) fix: Fix inlined templates in templates (#10786)
* [10111724b](https://github.com/argoproj/argo-workflows/commit/10111724be068feddc4e201680b0cd4bcd5ff3bf) fix(agent): no more requeue when the node succeeded (#10681)
* [40c4575a5](https://github.com/argoproj/argo-workflows/commit/40c4575a5eeec0cc9636fbd8e79d4a6dc5cd6b4f) fix: updates the curl example to use the BASE_HREF. Fixes #7416 (#10759)
* [3114a7de6](https://github.com/argoproj/argo-workflows/commit/3114a7de6a716e3d8ebace2900f44ee6a7b5227d) chore(deps): bump moment-timezone from 0.5.42 to 0.5.43 in /ui (#10802)
* [817a3df4c](https://github.com/argoproj/argo-workflows/commit/817a3df4cf91256892c1c95ed6a984a292e23f03) chore(deps): bump react-datepicker from 4.10.0 to 4.11.0 in /ui (#10800)
* [9ecfca8dc](https://github.com/argoproj/argo-workflows/commit/9ecfca8dc5553d1e2ccef2ac60e8dc7e69de68a6) chore(deps): bump github.com/antonmedv/expr from 1.12.3 to 1.12.5 (#10754)
* [d4a30a556](https://github.com/argoproj/argo-workflows/commit/d4a30a556a7093068624dbe16f05b381705dc6e0) fix: Update GitHub RSA SSH host key (#10779)
* [cbd40e7ac](https://github.com/argoproj/argo-workflows/commit/cbd40e7ac81160718db6ffa247f88edf77335d1e) fix: metrics don't get emitted properly during retry. Fixes #8207 #10463 (#10489)
* [dd2f8cbae](https://github.com/argoproj/argo-workflows/commit/dd2f8cbaea2f96d42accd4df8a22c05de48c9e6e) fix: Immediately release locks by pending workflows that are shutting down. Fixes #10733 (#10735)
* [385de1ebe](https://github.com/argoproj/argo-workflows/commit/385de1ebe6f753eb15428e46e6e0b36c90e889ad) chore(deps): bump cronstrue from 2.23.0 to 2.24.0 in /ui (#10757)
* [13586fe97](https://github.com/argoproj/argo-workflows/commit/13586fe974a987c18ed4fd9668931f2664888bf7) chore(deps): bump moment-timezone from 0.5.41 to 0.5.42 in /ui (#10752)
* [f3f0019de](https://github.com/argoproj/argo-workflows/commit/f3f0019ded27d2612811c9d7882adc875e443812) chore(deps): bump cloud.google.com/go/storage from 1.30.0 to 1.30.1 (#10750)
* [8c2606f53](https://github.com/argoproj/argo-workflows/commit/8c2606f53ff5593205ed902e613f1c011faf1667) chore(deps): bump github.com/minio/minio-go/v7 from 7.0.49 to 7.0.50 (#10751)
* [39ff41a32](https://github.com/argoproj/argo-workflows/commit/39ff41a32fe960f68691b6667d89d8f68079f427) fix: DB sessions are recreated whenever controller configmap updates. Fixes #10498 (#10734)
* [03f129ca2](https://github.com/argoproj/argo-workflows/commit/03f129ca229cacd7c06451a0d0c00176fae7232f) fix: Workflow stuck at running when init container failed but wait container did not. Fixes #10717 (#10740)
* [be5b157f3](https://github.com/argoproj/argo-workflows/commit/be5b157f3aa996c634697d2d721995714b294419) fix: Improve templating diagnostics. Fixes #8311 (#10741)
* [53ea5da29](https://github.com/argoproj/argo-workflows/commit/53ea5da29f7620b5fb142e492db86372b97bebd9) Revert "Fixes #10234 - Postgres SSL Certificate fix" (#10736)
* [7da30bd51](https://github.com/argoproj/argo-workflows/commit/7da30bd510fe40dc070be78056f40bc035933112) feat: Parse JSON structured logs in Argo UI. Fixes #6856 (#10145)
* [12003cad9](https://github.com/argoproj/argo-workflows/commit/12003cad92ab85247cbd7448b4e1639385aa2157) fix: ensure children containers are killed for container sets. Fixes #10491 (#10639)
* [2a9bd6c83](https://github.com/argoproj/argo-workflows/commit/2a9bd6c83601990259fd5162edeb425741757484) fix: Support v1 PDB in k8s v1.25+. Fixes #10649 (#10712)
* [ca97bd2c5](https://github.com/argoproj/argo-workflows/commit/ca97bd2c579709f0ac2ebee225e235fe9ae31078) chore(deps): bump google.golang.org/api from 0.112.0 to 0.114.0 (#10703)
* [f62472a69](https://github.com/argoproj/argo-workflows/commit/f62472a69a18f37f668cfb3e29a17b8be75e6550) fix(ui): reword Workflow `DELETED` error (#10689)
* [801911c95](https://github.com/argoproj/argo-workflows/commit/801911c95eb9614d422507ef03e0c0d48401534f) chore(deps): bump cloud.google.com/go/storage from 1.29.0 to 1.30.0 (#10702)
* [ec856835a](https://github.com/argoproj/argo-workflows/commit/ec856835a3a4ec78164aa737f98d4b1653809781) fix: PVC in wf.status should be reset when retrying workflow (#10685)
* [c1484f9c5](https://github.com/argoproj/argo-workflows/commit/c1484f9c54bf5a6e9b1e34f33d741ae69f3d2b4f) feat: add custom columns support for workflow list views (#10693)
* [f7922fb80](https://github.com/argoproj/argo-workflows/commit/f7922fb80e054da20a6f8aa782b3fbe8aac146a3) fix: ensure error returns before attrs is accessed. Fixes #10691 (#10692)
* [94f66a20e](https://github.com/argoproj/argo-workflows/commit/94f66a20eb5fb3aca63556ecf67a77a9900b9a99) feat: extend links feature for custom workflow views (#10677)
* [77f459438](https://github.com/argoproj/argo-workflows/commit/77f45943888bcba60416773a4bfe8b12fef8fdf5) chore(deps): bump github.com/Azure/azure-sdk-for-go/sdk/azidentity from 1.2.1 to 1.2.2 (#10668)
* [26bad2f6e](https://github.com/argoproj/argo-workflows/commit/26bad2f6e63d95d9349b33a2f0e19515cd494b0a) fix: get configmap data when updating controller config Fixes #10659 (#10660)
* [2bf90c6cb](https://github.com/argoproj/argo-workflows/commit/2bf90c6cb950f7d8a691273bb87acc37a10ee07a) chore(deps): bump google.golang.org/api from 0.111.0 to 0.112.0 (#10665)
* [99e685e73](https://github.com/argoproj/argo-workflows/commit/99e685e73f3156b8f6dcca9ea4332b726adbba3a) chore(deps): bump github.com/antonmedv/expr from 1.12.1 to 1.12.3 (#10669)
* [d6afd2087](https://github.com/argoproj/argo-workflows/commit/d6afd2087951469affd82aebd3e83ab3d50ea1bc) chore(deps): bump github.com/golang/protobuf from 1.5.2 to 1.5.3 (#10666)
* [ad245edff](https://github.com/argoproj/argo-workflows/commit/ad245edff60cabcb29cccf5716200332d95b75e7) chore(deps): bump cron-parser from 4.7.1 to 4.8.1 in /ui (#10670)
* [1acc9668a](https://github.com/argoproj/argo-workflows/commit/1acc9668a3cd33f1043f1a8476b5f82074cf7c9f) fix: executor dir perm changed to 755. Fixes #9651 (#10664)
* [1001424c7](https://github.com/argoproj/argo-workflows/commit/1001424c710b39f8b371edb070f2734afc4cfa96) chore(deps): bump github.com/prometheus/common from 0.41.0 to 0.42.0 (#10667)
* [08bb5d58c](https://github.com/argoproj/argo-workflows/commit/08bb5d58cdcb86806001b6d11ae276d7f59fc927) chore(deps): bump golang.org/x/oauth2 from 0.5.0 to 0.6.0 (#10644)
* [bb296decf](https://github.com/argoproj/argo-workflows/commit/bb296decfa5b7d49328d3ccb612a8f25876d4df4) chore(deps): bump golang.org/x/crypto from 0.6.0 to 0.7.0 (#10643)
* [f421de7c2](https://github.com/argoproj/argo-workflows/commit/f421de7c26cd13d88dfe1be35489454564a0be45) chore(deps): bump github.com/itchyny/gojq from 0.12.11 to 0.12.12 (#10635)
* [1b2c1c674](https://github.com/argoproj/argo-workflows/commit/1b2c1c6742587aa65958349f695bf9a48d7cd732) chore(deps): bump github.com/prometheus/common from 0.40.0 to 0.41.0 (#10636)
* [d536eec36](https://github.com/argoproj/argo-workflows/commit/d536eec36729ad69102cd41dd04ca7a1be878878) fix: Fix broken archive UI Fixes #10606 (#10622)
* [781675ddc](https://github.com/argoproj/argo-workflows/commit/781675ddcf6f1138d697cb9c71dae484daa0548b) fix: added logs related to executing commands in the container (#10530)
* [21c97c5ca](https://github.com/argoproj/argo-workflows/commit/21c97c5ca45288283100e48f24f9290afbc15a39) chore(deps): bump google.golang.org/api from 0.110.0 to 0.111.0 (#10634)
* [837385ffc](https://github.com/argoproj/argo-workflows/commit/837385ffc6024d5e00666b386d96bea64e960810) Add Hera to Ecosystem list, Fixes #10604 (#10603)
* [61ab1bad3](https://github.com/argoproj/argo-workflows/commit/61ab1bad3f3d8b1cc707b788836d006ff5955a96) Revert "chore(deps): bump react-router-dom and @types/react-router-do… (#10590)
* [786639d4e](https://github.com/argoproj/argo-workflows/commit/786639d4e1bb279894e4f36388f83b721990b261) chore(deps): bump github.com/stretchr/testify from 1.8.1 to 1.8.2 (#10589)
* [a36e55bfb](https://github.com/argoproj/argo-workflows/commit/a36e55bfb39f85119df1d4278120750cf389fc58) chore(deps): bump github.com/minio/minio-go/v7 from 7.0.48 to 7.0.49 (#10584)
* [0e809fc59](https://github.com/argoproj/argo-workflows/commit/0e809fc594f4c741e664a066c9db4e3b7e1517f6) chore(deps): bump github.com/antonmedv/expr from 1.12.0 to 1.12.1 (#10582)
* [242e8fe16](https://github.com/argoproj/argo-workflows/commit/242e8fe161d3e9f8f5edf29691570fcde258d66f) chore(deps): bump github.com/prometheus/common from 0.39.0 to 0.40.0 (#10585)
* [51ed115a8](https://github.com/argoproj/argo-workflows/commit/51ed115a8abc3385e97aef135a395a8402096748) fix: panic in offline linter + handling stdin (#10576)
* [2622afa7e](https://github.com/argoproj/argo-workflows/commit/2622afa7e554071004c7dd08d0890ed5a6f558b8) chore(deps): bump react-router-dom and @types/react-router-dom in /ui (#10587)
* [68b22b800](https://github.com/argoproj/argo-workflows/commit/68b22b800c2dde174c8fbac6f3fd829a39738a79) chore(deps): bump moment-timezone from 0.5.40 to 0.5.41 in /ui (#10586)
* [c0db6fd1b](https://github.com/argoproj/argo-workflows/commit/c0db6fd1b25fac6302b6f95c4e5f6b807291737d) Revert "chore(deps): bump react-router-dom and @types/react-router-dom in /ui" (#10575)
* [df5941ea8](https://github.com/argoproj/argo-workflows/commit/df5941ea858c20b0bfc99b8d4177fbb279ef99d0) fix: Priority don't work in workflow spec. Fixes #10374 (#10483)
* [77da05038](https://github.com/argoproj/argo-workflows/commit/77da05038154a97c52db7aa64acbf14bba9794f4) fix: change log severity when artifact is not found (#10561)
* [f918e3a4b](https://github.com/argoproj/argo-workflows/commit/f918e3a4b3293f41d34a41b0a34799d7aad1449b) fix: Resolve issues with offline linter + add tests (#10559)
* [47dd82e80](https://github.com/argoproj/argo-workflows/commit/47dd82e80db71954816515721764873fceb9de05) feat: Enable Codespaces with `kit` (#10532)
* [d75e37e8b](https://github.com/argoproj/argo-workflows/commit/d75e37e8b1c885ac3ebb11205ec452365ee2af67) fix: Correct SIGTERM handling. Fixes #10518 #10337 #10033 #10490 (#10523)
* [a862ea1b8](https://github.com/argoproj/argo-workflows/commit/a862ea1b8aa283eefe4f879d43e358d2d15678b0) fix: remove kubectl binary from argoexec (#10550)
* [5c3c3b3a8](https://github.com/argoproj/argo-workflows/commit/5c3c3b3a8ef23812806a10f7c4a5dc45ec43d782) fix: exit handler variables don't get resolved correctly. Fixes #10393 (#10449)
* [16dfc0020](https://github.com/argoproj/argo-workflows/commit/16dfc0020e18c21d36fe2af30b0229cf5e75eff8) chore(deps): bump react-router-dom and @types/react-router-dom in /ui (#10547)
* [7fea83b32](https://github.com/argoproj/argo-workflows/commit/7fea83b321c005bcc2688af44d3932b6f13cdf7b) chore(deps): bump github.com/minio/minio-go/v7 from 7.0.47 to 7.0.48 (#10545)
* [7dedb5ac6](https://github.com/argoproj/argo-workflows/commit/7dedb5ac6ac9830bcefcd84fe51d194af100df06) chore(deps): bump google.golang.org/api from 0.109.0 to 0.110.0 (#10546)
* [1322f2627](https://github.com/argoproj/argo-workflows/commit/1322f26272b403bb300f276b808a43ba1db136dc) chore(deps): bump github.com/antonmedv/expr from 1.10.5 to 1.12.0 (#10466)
* [35dbc6901](https://github.com/argoproj/argo-workflows/commit/35dbc6901b346fca4fd483b746eb8055086b0707) chore(deps): bump cronstrue from 2.22.0 to 2.23.0 in /ui (#10512)
* [5eda209a5](https://github.com/argoproj/argo-workflows/commit/5eda209a58213103ae517436076fad8acc2654d0) chore(deps): bump cron-parser from 4.7.0 to 4.7.1 in /ui (#10354)
* [04a84ee32](https://github.com/argoproj/argo-workflows/commit/04a84ee322738193039c84278b23473ac2ba7eae) fix: evaluated debug env vars value (#10493)
* [08c85000f](https://github.com/argoproj/argo-workflows/commit/08c85000f44e5cd5cc639be579107a58d0ea8c5e) fix: use env when pod version annotation is missing. Fixes #10237 (#10457)
* [3dc00829c](https://github.com/argoproj/argo-workflows/commit/3dc00829c0ab5118117ca95d96d95f0d6118cd03) feat(ui): View custom container log. Fixes #9913 (#10397)
* [26ac857e9](https://github.com/argoproj/argo-workflows/commit/26ac857e905a75d1822887fef2426f062bf1178c) feat: Adds TimeZone column in cron list in UI - Fixes #10389 (#10390)
* [de8790cf7](https://github.com/argoproj/argo-workflows/commit/de8790cf76702428b404d8f09f6627ceac01f3d1) fix: stop writing RawClaim into authorization cookie to reduce cookie size. Fixes #9530, #10153 (#10170)
* [43766ca5d](https://github.com/argoproj/argo-workflows/commit/43766ca5d6ceabf790d17e336411001ac27b8583) feat: enable full offline lint of all resource types (#10059)
* [9cb3fc64c](https://github.com/argoproj/argo-workflows/commit/9cb3fc64cd51b5a7f5613e4602ecfd4fa53011e2) feat: replace jq with gojq (#10469)
* [b444440c7](https://github.com/argoproj/argo-workflows/commit/b444440c719555015986ab4f671720ccd246fff7) chore(deps): bump golang.org/x/crypto from 0.5.0 to 0.6.0 (#10505)
* [0ad8da783](https://github.com/argoproj/argo-workflows/commit/0ad8da7833e278d5a24debc12f27c94476c0aca3) chore(deps): bump golang.org/x/oauth2 from 0.4.0 to 0.5.0 (#10508)
* [ab178bb0b](https://github.com/argoproj/argo-workflows/commit/ab178bb0b36a5ce34b4c1302cf4855879a0e8cf5) fix: delete PVCs upon onExit error when OnWorkflowCompletion is enabled. Fixes #10408 (#10424)
* [5d0db0038](https://github.com/argoproj/argo-workflows/commit/5d0db00382960317db0da287178c883ab5218985) Fixes #10234 - Postgres SSL Certificate fix (#10300)

<details><summary><h3>Contributors</h3></summary>

* Alan Clucas
* Alex Collins
* Ben Brandt
* Ciprian Anton
* GeunSam2 (Gray)
* GoshaDo
* Hyacin
* Isitha Subasinghe
* J.P. Zivalich
* Jiacheng Xu
* John Daniel Maguire
* Josh Soref
* Julien Duchesne
* Kratik Jain
* Liang Xu
* Lifei Chen
* Mahmoud Abduljawad
* Michał Jastrzębski
* Mike Ringrose
* Mitsuo Heijo
* Petri Kivikangas
* Rajshekar Reddy
* Sandeep Vagulapuram
* Shraddha
* Tim Collins
* Vaibhav Kaushik
* Vasile Razdalovschi
* Yao Lin
* Yuan Tang
* dependabot[bot]
* github-actions[bot]
* hodong
* jannfis
* jxlwqq
* kolorful
* wangxiang
* weafscast

</details>

## v3.4.6 (2023-03-30)

Full Changelog: [v3.4.5...v3.4.6](https://github.com/argoproj/argo-workflows/compare/v3.4.5...v3.4.6)

### Selected Changes

* [988706dd1](https://github.com/argoproj/argo-workflows/commit/988706dd131cf98808f09fb7cc03780e2af94c73) fix: Support v1 PDB in k8s v1.25+. Fixes #10649 (#10712)
* [72a0e5b74](https://github.com/argoproj/argo-workflows/commit/72a0e5b74fe10c1b9c030e9b447f2d72d9713f4c) fix: Update GitHub RSA SSH host key (#10779)
* [8eedf94c6](https://github.com/argoproj/argo-workflows/commit/8eedf94c64da5955c110c8d20529927434c4ae4e) fix: metrics don't get emitted properly during retry. Fixes #8207 #10463 (#10489)
* [edc00836c](https://github.com/argoproj/argo-workflows/commit/edc00836cbd5fe031e4509e997f50ab93501f5f5) fix: Immediately release locks by pending workflows that are shutting down. Fixes #10733 (#10735)
* [1819e3067](https://github.com/argoproj/argo-workflows/commit/1819e3067a015550e6ea1a4c220c4b77c54d7555) fix: DB sessions are recreated whenever controller configmap updates. Fixes #10498 (#10734)
* [e71548868](https://github.com/argoproj/argo-workflows/commit/e715488680ad7bfd5bb3298418d8e38d352c3e38) fix: Workflow stuck at running when init container failed but wait container did not. Fixes #10717 (#10740)
* [a3d64b2d4](https://github.com/argoproj/argo-workflows/commit/a3d64b2d483d256b945a595c70097ef61039517c) fix: Improve templating diagnostics. Fixes #8311 (#10741)
* [99105c142](https://github.com/argoproj/argo-workflows/commit/99105c1424286f9c52be8d5dfc63296d93766740) fix: ensure children containers are killed for container sets. Fixes #10491 (#10639)
* [86b82f316](https://github.com/argoproj/argo-workflows/commit/86b82f316477b2d53351366f99cc33e003ace080) fix: PVC in wf.status should be reset when retrying workflow (#10685)
* [c56f65528](https://github.com/argoproj/argo-workflows/commit/c56f655289c4238de91d9169bed1eb9543831f34) fix: ensure error returns before attrs is accessed. Fixes #10691 (#10692)
* [6b7b4b3bc](https://github.com/argoproj/argo-workflows/commit/6b7b4b3bca44b82634e61e159581bc006f63179e) fix: get configmap data when updating controller config Fixes #10659 (#10660)
* [ac8e7e32b](https://github.com/argoproj/argo-workflows/commit/ac8e7e32ba8b75f1664f4817f6dabd0bc25743c9) fix: executor dir perm changed to 755. Fixes #9651 (#10664)
* [ac84d00a4](https://github.com/argoproj/argo-workflows/commit/ac84d00a4183aa763c94c93bf1beb58269c6e9d3) fix: Fix broken archive UI Fixes #10606 (#10622)
* [584998a7a](https://github.com/argoproj/argo-workflows/commit/584998a7aa777c484ca64f485e4b1acc83bdd343) fix: added logs related to executing commands in the container (#10530)
* [ae06f8519](https://github.com/argoproj/argo-workflows/commit/ae06f85192b708c73f2405b331849365045231d5) fix: Priority don't work in workflow spec. Fixes #10374 (#10483)
* [8470ed295](https://github.com/argoproj/argo-workflows/commit/8470ed295ed57f5b3e6dd744b62004f2a7973fa4) fix: change log severity when artifact is not found (#10561)
* [fe522b69c](https://github.com/argoproj/argo-workflows/commit/fe522b69cb6db0255934a6051fc5652212c01807) fix: Correct SIGTERM handling. Fixes #10518 #10337 #10033 #10490 (#10523)
* [4978d3b25](https://github.com/argoproj/argo-workflows/commit/4978d3b25be3935124c44d6f5ca7667c07ef3984) fix: exit handler variables don't get resolved correctly. Fixes #10393 (#10449)
* [e50c915ce](https://github.com/argoproj/argo-workflows/commit/e50c915ce1376492e20b02da89f186a75e2f3599) fix: evaluated debug env vars value (#10493)
* [ecd0d93d5](https://github.com/argoproj/argo-workflows/commit/ecd0d93d5139e0d633b100b991cadde306f3ed8c) fix: use env when pod version annotation is missing. Fixes #10237 (#10457)
* [5c5c6504a](https://github.com/argoproj/argo-workflows/commit/5c5c6504abdf40ff95c1f04ecbca93b59eb08f66) fix: stop writing RawClaim into authorization cookie to reduce cookie size. Fixes #9530, #10153 (#10170)
* [17ea4bc82](https://github.com/argoproj/argo-workflows/commit/17ea4bc821a9bc1537736759726501aa37b88fac) fix: delete PVCs upon onExit error when OnWorkflowCompletion is enabled. Fixes #10408 (#10424)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins
* Ciprian Anton
* GoshaDo
* Isitha Subasinghe
* Jiacheng Xu
* John Daniel Maguire
* Petri Kivikangas
* Sandeep Vagulapuram
* Shraddha
* Yuan Tang
* kolorful
* wangxiang
* weafscast

</details>

## v3.4.5 (2023-02-06)

Full Changelog: [v3.4.4...v3.4.5](https://github.com/argoproj/argo-workflows/compare/v3.4.4...v3.4.5)

### Selected Changes

* [dc30da81f](https://github.com/argoproj/argo-workflows/commit/dc30da81f8b75804c2cbd4df667be1288d294c8d) fix: return if nil pointer in dag.go. Fixes #10401 (#10402)
* [7d2a8c3d2](https://github.com/argoproj/argo-workflows/commit/7d2a8c3d20107786b9177a8e7b78a889c1e13c45) fix(docs): container-set-template workflow example (#10452)
* [3f329080e](https://github.com/argoproj/argo-workflows/commit/3f329080e49792690a650e14986342e39ed94956) chore(deps): bump google.golang.org/api from 0.108.0 to 0.109.0 (#10467)
* [79c012e19](https://github.com/argoproj/argo-workflows/commit/79c012e195f3b5f94ffce4fbed9e24ebd77a2528) chore(deps): bump docker/build-push-action from 3 to 4 (#10464)
* [898f0649f](https://github.com/argoproj/argo-workflows/commit/898f0649fa15d8899a7561f3c1cf953a21dcf34f) fix: Return correct http error codes. Fixes #9237 (#9916)
* [cfdf80ea1](https://github.com/argoproj/argo-workflows/commit/cfdf80ea10be5cf47508532ebf6193d013b6617f) chore(deps): bump react-datepicker and @types/react-datepicker in /ui (#10437)
* [fd6fd79e7](https://github.com/argoproj/argo-workflows/commit/fd6fd79e7a5b0f331bbe41ae2c73127153a94017) chore(deps): bump react-moment from 1.1.2 to 1.1.3 in /ui (#10355)
* [2a0e91b44](https://github.com/argoproj/argo-workflows/commit/2a0e91b447aa3e1bb644995375d85da50c59c80b) fix(controller): Add locking for read operationin controller. Fixes #… (#9985)
* [f48717ccf](https://github.com/argoproj/argo-workflows/commit/f48717ccf03bbcd7b68639a6fdc7515a6d468e3a) Fixes #10003: retries force exit handlers (#10159)
* [6bb290638](https://github.com/argoproj/argo-workflows/commit/6bb290638e789e61e2ff5576df82c451486eeaa3) feat: support set generateName in the eventbinding (#10371)
* [e7b5b25ef](https://github.com/argoproj/argo-workflows/commit/e7b5b25efa28c78719a3685f8addc02478a186ed) fix(executor):  make the comment of reportOutputs clearer (#10443)
* [d46d5e9fb](https://github.com/argoproj/argo-workflows/commit/d46d5e9fb45b5ff7d9dd974c6d961ef65397ec39) HasArtifactGC() can't access Status through execWf (#10423)
* [1d87b45cc](https://github.com/argoproj/argo-workflows/commit/1d87b45ccca7707bb568906b7ef22bbc5123da25) fix: add message when parse of private key fails due to existing sso secret. Fixes #10420 (#10421)
* [f9e392f2f](https://github.com/argoproj/argo-workflows/commit/f9e392f2fa12f0f4405fbe95eb854e04805b7b17) chore(deps): bump moment-timezone from 0.5.39 to 0.5.40 in /ui (#10438)
* [5ad423eed](https://github.com/argoproj/argo-workflows/commit/5ad423eed7370c41715f5852f0a1c4bb05c7f7bb) chore(deps): bump cronstrue from 2.21.0 to 2.22.0 in /ui (#10436)
* [133b4a384](https://github.com/argoproj/argo-workflows/commit/133b4a384364079e5c82580c4969ded636cbadf5) Fix: Enable users to use Archived Workflows functionality when RBAC is Namespace delegated (#10399)
* [8e7c73447](https://github.com/argoproj/argo-workflows/commit/8e7c7344720994ea1139914953844ee67c67e068) feat: allow switching timezone for date rendering. Fixes #3474 (#10120)
* [22fa3403a](https://github.com/argoproj/argo-workflows/commit/22fa3403ae720342a90fd7eb1b317653ba73c40d) fix: in gcs driver ensure prefix omitted if folder. Fixes #9732 (#10214)
* [605d590ec](https://github.com/argoproj/argo-workflows/commit/605d590ec25f05c3155aaa971d8b2f6421eb0056) chore(deps): bump github.com/go-openapi/jsonreference from 0.20.0 to 0.20.2 (#10382)
* [689df36af](https://github.com/argoproj/argo-workflows/commit/689df36af126bdf2af35b6f7f31a27aeb527d20a) chore(deps): bump superagent from 8.0.8 to 8.0.9 in /ui (#10416)
* [c3c71b955](https://github.com/argoproj/argo-workflows/commit/c3c71b955de9b0f7bab2c54ac2258b4e1fff766c) chore(deps): bump golang.org/x/time from 0.1.0 to 0.3.0 (#10412)
* [4d1e1c07b](https://github.com/argoproj/argo-workflows/commit/4d1e1c07b31cc1bb86cae79cf491658113008be6) chore(deps): bump github.com/Azure/azure-sdk-for-go/sdk/azidentity from 1.1.0 to 1.2.1 (#10411)
* [bca7e7ba1](https://github.com/argoproj/argo-workflows/commit/bca7e7ba1901e1e99aa275230ff2244868b4cb67) chore(deps): bump github.com/gavv/httpexpect/v2 from 2.9.0 to 2.10.0 (#10414)
* [e1be54ed3](https://github.com/argoproj/argo-workflows/commit/e1be54ed32288c5d6a6eb73102c88de2820f49fd) chore(deps): bump github.com/antonmedv/expr from 1.10.1 to 1.10.5 (#10413)
* [5a5de6728](https://github.com/argoproj/argo-workflows/commit/5a5de6728ad717754c14f574d398228edb2cf999) chore(deps): bump google.golang.org/api from 0.107.0 to 0.108.0 (#10385)
* [9e35c9cc0](https://github.com/argoproj/argo-workflows/commit/9e35c9cc0db1630b5d546a661f67ec10bea64463) chore(deps): bump github.com/antonmedv/expr from 1.9.0 to 1.10.1 (#10384)
* [b37cf46b8](https://github.com/argoproj/argo-workflows/commit/b37cf46b87a3ed37e5f55588d75d2ddca6d75530) chore(deps): bump github.com/spf13/viper from 1.14.0 to 1.15.0 (#10380)
* [7fc6ecc84](https://github.com/argoproj/argo-workflows/commit/7fc6ecc84db2832a25ae203b58e67769657b9991) chore(deps): bump superagent from 8.0.6 to 8.0.8 in /ui (#10386)
* [adc7a7060](https://github.com/argoproj/argo-workflows/commit/adc7a7060786531acfcf6cbc8a71092fe65b6fd7) chore(deps): bump github.com/gavv/httpexpect/v2 from 2.8.0 to 2.9.0 (#10383)
* [782717980](https://github.com/argoproj/argo-workflows/commit/7827179808491c8c5a9411eee4d30fdbeeeba3c3) chore(deps): bump cloud.google.com/go/storage from 1.28.0 to 1.29.0 (#10381)
* [651ec79ae](https://github.com/argoproj/argo-workflows/commit/651ec79ae278d45b3fd240d95a40b4108bbae43a) chore(deps): bump google.golang.org/api from 0.106.0 to 0.107.0 (#10353)
* [548f53261](https://github.com/argoproj/argo-workflows/commit/548f53261f8e04a563239eae61354d3899495f15) chore(deps): bump github.com/minio/minio-go/v7 from 7.0.45 to 7.0.47 (#10352)
* [a1db45a60](https://github.com/argoproj/argo-workflows/commit/a1db45a60a7878db3b19d50eeada6416b4e8dd5f) fix: fix not working dex deployment in quickstart manifests (#10346)
* [f1959c8de](https://github.com/argoproj/argo-workflows/commit/f1959c8def101b146876fd128b4383663a719b95) chore(deps): bump google.golang.org/api from 0.105.0 to 0.106.0 (#10325)
* [08ef2928e](https://github.com/argoproj/argo-workflows/commit/08ef2928e4c15de7ef7c5973559543fa7ce2ee33) fix: print template and pod name in workflow controller logs for node failure scenario (#10332)
* [b386d03e0](https://github.com/argoproj/argo-workflows/commit/b386d03e0a2ca16c911665613427325ab32eb252) chore(deps): bump golang.org/x/oauth2 from 0.3.0 to 0.4.0 (#10323)
* [13adf5e4a](https://github.com/argoproj/argo-workflows/commit/13adf5e4a615c18baf237db16253c5324c5e0091) chore(deps): bump github.com/coreos/go-oidc/v3 from 3.4.0 to 3.5.0 (#10324)
* [2c580b31c](https://github.com/argoproj/argo-workflows/commit/2c580b31c61e168e5f58b8357a72a97923ece952) chore(deps): bump golang.org/x/crypto from 0.4.0 to 0.5.0 (#10322)
* [75ce0af25](https://github.com/argoproj/argo-workflows/commit/75ce0af253d6d2441f518cf69fe1f8398f2fcad0) fix: fix minio image at older working version (#10314)
* [12e1b985c](https://github.com/argoproj/argo-workflows/commit/12e1b985c176dee1ea8b70a81e3fe5e3a91bf241) feat: cleanup code (#10316)
* [0f58387c7](https://github.com/argoproj/argo-workflows/commit/0f58387c79728b84037aa96221d1c97a974402a4) feat: HTTP Template respect podMetadata. Fixes #10062 (#10274)
* [a06e83182](https://github.com/argoproj/argo-workflows/commit/a06e83182e06261145d5127cb20a433ca2d82ac4) fix: improve rate at which we catch transient gcp errors in artifact driver Fixes #10282 #10174 (#10292)
* [5b450f6d7](https://github.com/argoproj/argo-workflows/commit/5b450f6d77d47116ed744f9831c77c8ebb3a9ed5) chore(deps): bump nick-fields/retry from 2.8.2 to 2.8.3 (#10293)
* [002641262](https://github.com/argoproj/argo-workflows/commit/002641262319081033ec81e4b1c18c2e4003cbf4) chore(deps): bump superagent from 8.0.5 to 8.0.6 in /ui (#10208)
* [eeaf2c415](https://github.com/argoproj/argo-workflows/commit/eeaf2c415f3968406200ca7289290c51e59c9c0a) Update USERS.md (#10241)
* [28a9ee593](https://github.com/argoproj/argo-workflows/commit/28a9ee593c7e73a10b6d42c44e1cfbe9427a3c97) fix: remove url encoding/decoding on user-supplied URL. Fixes #9935 (#9944)
* [4e25739cb](https://github.com/argoproj/argo-workflows/commit/4e25739cbb966340cee1a9ba251dc8614ef7ebb4) remove debug println (#10252)
* [2607867ab](https://github.com/argoproj/argo-workflows/commit/2607867ab4f424086998ea15fa50126360c3bba8) fix: use podname in failure podName instead of ID. Fixes #10124 (#10268)
* [13620fad8](https://github.com/argoproj/argo-workflows/commit/13620fad8e3d34477911270c3f3bf75c5aa7f27e) Skip Artifact GC test if Workflow fails (#10298)
* [f4a65b11a](https://github.com/argoproj/argo-workflows/commit/f4a65b11a184f7429d0615a6fa65bc2cea4cc425) feat: support finalizers in workflowMetadata (#10243)
* [6c5b50678](https://github.com/argoproj/argo-workflows/commit/6c5b506786957bc7f948fd4cd63c0e58ba7a7584) chore(deps): bump github.com/prometheus/common from 0.38.0 to 0.39.0 (#10247)
* [ae93d0316](https://github.com/argoproj/argo-workflows/commit/ae93d03166b5ca2b0ad8e90db7784e51ee9da8ad) chore(deps): bump google.golang.org/api from 0.104.0 to 0.105.0 (#10245)
* [cd9a9f2bf](https://github.com/argoproj/argo-workflows/commit/cd9a9f2bfb6da1e4d5f259c12110d76e55f9f012) chore(deps): bump github.com/gavv/httpexpect/v2 from 2.7.0 to 2.8.0 (#10246)
* [4ed945425](https://github.com/argoproj/argo-workflows/commit/4ed94542521ea3889e179a5c4721726d7f6bb430) fix: ensure metadata is not undefined when accessing label. Fixes #10227 (#10228)
* [896830dd3](https://github.com/argoproj/argo-workflows/commit/896830dd366cf1a308a341003bacc650119e1f30) fix: ensure HTTP reconciliation occurs for onExit step nodes (#10195)
* [b0f0c589e](https://github.com/argoproj/argo-workflows/commit/b0f0c589e626650ca01a635f773af983e213fbec) fix: Auto update workflow controller configmap (#10218)
* [7b34cb1cf](https://github.com/argoproj/argo-workflows/commit/7b34cb1cf7f5a490992e307493446950b954c9b7) chore(deps): bump golang.org/x/net from 0.2.0 to 0.4.0 (#10204)
* [9283b40b6](https://github.com/argoproj/argo-workflows/commit/9283b40b6a5d520ce04e07b586f04822440df869) chore(deps): bump google.golang.org/api from 0.103.0 to 0.104.0 (#10206)
* [a37c3f0a5](https://github.com/argoproj/argo-workflows/commit/a37c3f0a502f711a4d760bd8d0e728c8f1373dd5) chore(deps): bump github.com/prometheus/common from 0.37.0 to 0.38.0 (#10205)
* [5775c12c5](https://github.com/argoproj/argo-workflows/commit/5775c12c5c736a596830c35582949abba88a5903) chore(deps): bump cronstrue from 2.20.0 to 2.21.0 in /ui (#10210)
* [54e4e4899](https://github.com/argoproj/argo-workflows/commit/54e4e4899d0eb35f7213041547a609423f2633a9) chore(deps): bump github.com/gavv/httpexpect/v2 from 2.6.1 to 2.7.0 (#10202)
* [4e2471aa2](https://github.com/argoproj/argo-workflows/commit/4e2471aa288dd145968e612c75315b5be1fb3f5c) chore(deps): bump golang.org/x/crypto from 0.3.0 to 0.4.0 (#10201)
* [f390f6128](https://github.com/argoproj/argo-workflows/commit/f390f61280cf435c209deddaae55e5710e7f7135) fix: Artifact GC should not reference execWf.Status (#10160)
* [898f738c0](https://github.com/argoproj/argo-workflows/commit/898f738c09334059614be61bb7d75f6895c5861b) fix: add omitted children to the dag Fixes #9852 (#9918)
* [73771ab7d](https://github.com/argoproj/argo-workflows/commit/73771ab7dd7304e068b27fd669935ba8c2574686) chore(deps): bump github.com/Masterminds/sprig/v3 from 3.2.2 to 3.2.3 (#10164)
* [5caf65efa](https://github.com/argoproj/argo-workflows/commit/5caf65efa41df2a924853e34a6d5f018ddd2951d) chore(deps): bump superagent from 8.0.4 to 8.0.5 in /ui (#10166)
* [b9a96c0d5](https://github.com/argoproj/argo-workflows/commit/b9a96c0d56915167b2cf7780eac53e45bd867815) chore(deps): bump github.com/go-sql-driver/mysql from 1.6.0 to 1.7.0 (#10163)
* [f9931743d](https://github.com/argoproj/argo-workflows/commit/f9931743d9b35bd18c75568f2e8c7bf048cf3970) chore(deps): bump github.com/minio/minio-go/v7 from 7.0.44 to 7.0.45 (#10162)
* [18aa5615b](https://github.com/argoproj/argo-workflows/commit/18aa5615b17eb0b3c46582c634c2b2e52334774e) chore(deps): bump cronstrue from 2.19.0 to 2.20.0 in /ui (#10161)
* [55e96972e](https://github.com/argoproj/argo-workflows/commit/55e96972ec67d0fb5b871d1b1acc649837904fd3) fix: Make `jq` work. Fixes #9860 (#10150)
* [a6aca18e1](https://github.com/argoproj/argo-workflows/commit/a6aca18e1d6f73022263628c6e64e317a4d1b326) fix (argo wait): use functions to constrain ctx instead of blocks (#10140)
* [d91544212](https://github.com/argoproj/argo-workflows/commit/d9154421280a82745fcd08c5c5f0e9d075e69b4a) fix: go-git error empty git-upload-pack given. Fixes #9613 (#9982)
* [22c4fca36](https://github.com/argoproj/argo-workflows/commit/22c4fca36f011a4b5ee501727ad2042d20afaea8) feat(server): add kube-api throttle options (#10110)
* [6166464aa](https://github.com/argoproj/argo-workflows/commit/6166464aa6961bcb375705753ed0b58707d68222) feat: Ignore SIGURG in argoexec emissary. Fixes #10129 (#10141)
* [652970c39](https://github.com/argoproj/argo-workflows/commit/652970c39041a552add999825a3419224fbe4d82) feat: implement backoff when deleting artifacts. Fixes #9294. (#10088)
* [f9f231e9f](https://github.com/argoproj/argo-workflows/commit/f9f231e9f56d114ed7467d8d3e30bbd102dda6c6) fix: emissary detects tty and wraps command in pseudo terminal. Fixes #9179 (#10039)
* [c7310079e](https://github.com/argoproj/argo-workflows/commit/c7310079e67f2dca12867309cf811532e0a56b4c) fix: Ensure the SSO http client takes into consideration http proxies, Fixes #9259 (#10046)
* [51625c2c5](https://github.com/argoproj/argo-workflows/commit/51625c2c5f751534635b01758747daedb5efea06) fix: Add --tls-certificate-secret-name parameter to server command. Fixes #5582  (#9789)
* [1225d8b54](https://github.com/argoproj/argo-workflows/commit/1225d8b546c1e5093047d8e6e8a46d053f051d97) fix: error component showing inaccurate errors. Fixes #9274 (#10128)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins
* Amritpal Nagra
* Balaji Siva
* Caelan U
* Dana Pieluszczak
* Dillen Padhiar
* Isitha Subasinghe
* J.P. Zivalich
* Jiacheng Xu
* John Lin
* Jordan (Tao Zhang)
* Julie Vogelman
* Junaid Rahim
* Kacper Kondratek
* Kazuki Suda
* Kevin Holmes
* Mayursinh Sarvaiya
* Nandita
* Paolo Quadri
* Rick
* Rohan Kumar
* Ruben Jenster
* Sarah Henkens
* Saravanan Balasubramanian
* Shota Sugiura
* Sreejith Kesavan
* Sushant20
* Takahiro Yoshikawa
* Tianchu Zhao
* Vladimir Ivanov
* Yuan Tang
* Yuuki Takahashi
* dependabot[bot]
* github-actions[bot]
* huiwq1990
* jessonzj
* shiraOvadia
* wangxiang

</details>

## v3.4.4 (2022-11-28)

Full Changelog: [v3.4.3...v3.4.4](https://github.com/argoproj/argo-workflows/compare/v3.4.3...v3.4.4)

### Selected Changes

* [311f151ac](https://github.com/argoproj/argo-workflows/commit/311f151ac2bd30755c3eaa1adf40fe29da6125a1) fix: Support other output artifact types in argo get (#10125)
* [7c805fefe](https://github.com/argoproj/argo-workflows/commit/7c805fefeebf35b84f2f4927fd9f8ba4e885350f) feat: Workflow title/description in workflow list view. Fixes #6529 (#9805)
* [eb2c54b9f](https://github.com/argoproj/argo-workflows/commit/eb2c54b9fb07a149254b1e5f43e7a204c7e16f04) fix: SSO insecureSkipVerify not work. Fixes #10089 (#10090)
* [225cd97f5](https://github.com/argoproj/argo-workflows/commit/225cd97f5329966d88ce7f8560418639deab2ea5) fix(9656): stores all states except workflows, fixes #9656 (#9846)
* [b5dbd00a4](https://github.com/argoproj/argo-workflows/commit/b5dbd00a4a5871669961c71f726499e7aefc0b4c) chore(deps): bump cronstrue from 2.15.0 to 2.19.0 in /ui (#10116)
* [2f8a57450](https://github.com/argoproj/argo-workflows/commit/2f8a5745011c59536b88a78d70a21de4bc519737) chore(deps): bump github.com/TwiN/go-color from 1.2.0 to 1.4.0 (#10115)
* [26bbb973c](https://github.com/argoproj/argo-workflows/commit/26bbb973c6c4662045a3ace109128a9305b9cfa7) chore(deps): bump superagent from 8.0.0 to 8.0.4 in /ui (#10114)
* [6c653adb0](https://github.com/argoproj/argo-workflows/commit/6c653adb0e87524160e0fbb96d53e86cc679ce83) chore(deps): bump github.com/tidwall/gjson from 1.14.3 to 1.14.4 (#10113)
* [f89034557](https://github.com/argoproj/argo-workflows/commit/f890345570e37d8aeea47d8f7548314bd8bf7387) chore(deps): bump github.com/gavv/httpexpect/v2 from 2.4.1 to 2.6.1 (#10112)
* [bd1474eac](https://github.com/argoproj/argo-workflows/commit/bd1474eac46c42a7ac97f591b70a52c7f2f134b2) chore(deps): bump github.com/minio/minio-go/v7 from 7.0.43 to 7.0.44 (#10111)
* [be759b4c4](https://github.com/argoproj/argo-workflows/commit/be759b4c46a67513da7b961624e47741e6e832b0) fix: Correct behaviour of `CreateBucketIfNotPresent`. Fixes #10083 (#10084)
* [e5ea21ee0](https://github.com/argoproj/argo-workflows/commit/e5ea21ee09e792ca2028896f07b3e66c42f81e75) fix: reconcile wf when taskresult is added/updated. Fixes #10096 (#10097)
* [d03f5e5e0](https://github.com/argoproj/argo-workflows/commit/d03f5e5e084aa4433714f745a617d6ed4cc2725a) fix: use the execWf spec to determine artifactgc strategy (#10066)
* [766f9bdef](https://github.com/argoproj/argo-workflows/commit/766f9bdeff9da0bac8cc639f59b0df60e219d230) fix: Disallow stopping completed workflows (#10087)
* [ab0944899](https://github.com/argoproj/argo-workflows/commit/ab09448992b08d3b933180f9eaa88a0245c49eda) fix: Documentation to clarify need for RoleBinding for ArtifactGC (#10086)
* [f7918baec](https://github.com/argoproj/argo-workflows/commit/f7918baec73c94db0c85ce5d90ba2d4ca8e0472e) chore(deps): bump google.golang.org/api from 0.101.0 to 0.103.0 (#10026)
* [da5f258e8](https://github.com/argoproj/argo-workflows/commit/da5f258e8f0d6c93341bbfcaab7ee31331b0bdcc) chore(deps): bump cron-parser from 4.6.0 to 4.7.0 in /ui (#10071)
* [2eb871bf2](https://github.com/argoproj/argo-workflows/commit/2eb871bf2f1bd9008ab09995b560600e5c594153) fix(operator): Workflow stuck at running when init container failed. Fixes #10045 (#10047)
* [ea8a2b879](https://github.com/argoproj/argo-workflows/commit/ea8a2b879fb35dd45ec52079081900b98bd5de0d) chore(deps): bump github.com/gavv/httpexpect/v2 from 2.3.1 to 2.4.1 (#10067)
* [261e7d40a](https://github.com/argoproj/argo-workflows/commit/261e7d40a9ce7840f6066d999b3bc95ac6428510) chore(deps): bump github.com/spf13/viper from 1.13.0 to 1.14.0 (#10023)
* [193d4dac0](https://github.com/argoproj/argo-workflows/commit/193d4dac0a849102712a4f6e7591b242067e22c0) chore(deps): bump github.com/aliyun/aliyun-oss-go-sdk from 2.2.5+incompatible to 2.2.6+incompatible (#10068)
* [51692bfa9](https://github.com/argoproj/argo-workflows/commit/51692bfa97af5479036d71151ab49b6daa600fb3) chore(deps): bump cronstrue from 2.14.0 to 2.15.0 in /ui (#10075)
* [74766d566](https://github.com/argoproj/argo-workflows/commit/74766d566c41752dcd64eb690cd06abecdf8e79c) fix(ui): use podname for EventPanel name param (#10051) (#10052)
* [4eb6cb781](https://github.com/argoproj/argo-workflows/commit/4eb6cb7817d3b0f2dc9eeecb5856ec8bd10e9f98) fix: Upgrade kubectl to v1.24.8 to fix vulnerabilities (#10008)
* [55ad68022](https://github.com/argoproj/argo-workflows/commit/55ad68022b17763e9265da10fa74d1c26031660d) fix: if artifact GC Pod fails make sure error is propagated as a Condition (#10019)
* [acab9b58e](https://github.com/argoproj/argo-workflows/commit/acab9b58e4af21018911753668bdf18ef8625c91) feat: Support disable retrieval of label values for certain keys (#9999)
* [a758fcd16](https://github.com/argoproj/argo-workflows/commit/a758fcd164f6e1655bd14e1f0ad4ee39041e6286) chore(deps): bump github.com/prometheus/client_golang from 1.13.0 to 1.14.0 (#10025)
* [8b0e125c4](https://github.com/argoproj/argo-workflows/commit/8b0e125c4f95469819a26198a3c7f86655c5658a) chore(deps): bump cloud.google.com/go/storage from 1.27.0 to 1.28.0 (#10024)
* [e2e1f16cd](https://github.com/argoproj/argo-workflows/commit/e2e1f16cda3c7b46cac18e4f1a429a51a90b3a2d) fix(ui): search artifact by uid in archived wf. Fixes #9968 (#10014)
* [67bcdb5e6](https://github.com/argoproj/argo-workflows/commit/67bcdb5e6da76a5f3dfb0fe71a16cf086e7ea26a) fix: use correct node name as args to PodName. Fixes #9906 (#9995)
* [1487bbc19](https://github.com/argoproj/argo-workflows/commit/1487bbc197a54a2e8caae4205aa98283583956f1) fix: default initialisation markNodePhase (#9902)
* [6bc25a8fe](https://github.com/argoproj/argo-workflows/commit/6bc25a8fe51c45378d269b0336ed47c33352c355) Fixes #10003: retry handler even if it succeded (#10004)
* [01c51b458](https://github.com/argoproj/argo-workflows/commit/01c51b458882f859c315f76374ab0549f0ea897a) chore(deps): bump github.com/minio/minio-go/v7 from 7.0.42 to 7.0.43 (#9973)
* [d4e60fa14](https://github.com/argoproj/argo-workflows/commit/d4e60fa149525fcc2d0e73ddaeded3225255ab0f) fix: assume plugins may produce outputs.result and outputs.exitCode (Fixes #9966) (#9967)
* [6ba1fa531](https://github.com/argoproj/argo-workflows/commit/6ba1fa53109014ef5eefb28ac2f248ab703b61ca) fix: cleaned key paths in gcs driver. Fixes #9958 (#9959)
* [b91606a64](https://github.com/argoproj/argo-workflows/commit/b91606a644983d00c4a8aa3439f1d4581c01a478) fix: mount secret with SSE-C key if needed, fix secret key read. Fixes #9867 (#9870)
* [4f1451e9c](https://github.com/argoproj/argo-workflows/commit/4f1451e9c605b807a9a82c298b5d0b74c6ff9b4c) fix: Preserve symlinks in untar. Fixes #9948 (#9949)
* [a5b31b3f0](https://github.com/argoproj/argo-workflows/commit/a5b31b3f07eb545abd7219fdaddc88c55952cad1) fix(test): skip artifact private repo test. Fixes: #8953 (#9838)
* [4c6b6bf4d](https://github.com/argoproj/argo-workflows/commit/4c6b6bf4db06cbf850b50b683e793609864c92a9) fix: show pending workflows in workflow list Fixes #9812 (#9909)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins
* Arjun Gopisetty
* Athitya Kumar
* Chris Jones
* Isitha Subasinghe
* Jason Meridth
* Julie Vogelman
* Justin Marquis
* Lifei Chen
* Michael Crenshaw
* Michael Weibel
* Michal Raška
* Paolo Quadri
* Pedro López Mareque
* Rick
* Steven White
* Tianchu Zhao
* Yuan Tang
* botbotbot
* dependabot[bot]
* fsiegmund
* github-actions[bot]
* neo502721

</details>

## v3.4.3 (2022-10-30)

Full Changelog: [v3.4.2...v3.4.3](https://github.com/argoproj/argo-workflows/compare/v3.4.2...v3.4.3)

### Selected Changes

* [23e3d4d6f](https://github.com/argoproj/argo-workflows/commit/23e3d4d6f646c413d66145ee3e2210ff71eef21d) fix(ui): Apply url encode and decode to a `ProcessURL`. Fixes #9791 (#9912)
* [d612d5d9b](https://github.com/argoproj/argo-workflows/commit/d612d5d9b983a3cc7436d1c9a94dedb4382f6a9a) feat(ui): view artifact in archiveworkflow. Fixes #9627 #9772 #9858 (#9836)
* [a31576576](https://github.com/argoproj/argo-workflows/commit/a315765769867a1e7528f253f7e94bbb5291df7b) refactor: ui, convert cluster workflow template to functional component (#9809)
* [30a6d5eb7](https://github.com/argoproj/argo-workflows/commit/30a6d5eb73f1197380df4b904eed2646dfb3b4aa) feat: Include node.name as a field for interpolation (#9641)
* [1c41dc715](https://github.com/argoproj/argo-workflows/commit/1c41dc7154e947caae22615444cb363ae893ace9) chore(deps): bump google.golang.org/api from 0.99.0 to 0.101.0 (#9927)
* [b1c78de08](https://github.com/argoproj/argo-workflows/commit/b1c78de0868f5588b01122de08fd5d3bb24faa22) Remove wrong braces in documentation (#9903)
* [ff3133fb7](https://github.com/argoproj/argo-workflows/commit/ff3133fb7d049c3d239522ac37f153b69d76b028) Moved elevated permissions to job level (#9917)
* [6b086368f](https://github.com/argoproj/argo-workflows/commit/6b086368f6480a2de5e2d43eec73514de0ad01ac) fix: Mutex is not initialized when controller restart (#9873)

<details><summary><h3>Contributors</h3></summary>

* Amit Auddy
* Andrii Chubatiuk
* Eddie Knight
* Max Görner
* Ryan Copley
* Saravanan Balasubramanian
* Tianchu Zhao
* Tim Collins
* dependabot[bot]
* github-actions[bot]
* maozhi

</details>

## v3.4.2 (2022-10-22)

Full Changelog: [v3.4.1...v3.4.2](https://github.com/argoproj/argo-workflows/compare/v3.4.1...v3.4.2)

### Selected Changes

* [b00550f7b](https://github.com/argoproj/argo-workflows/commit/b00550f7bae3938d324ce2857019529d61382d84) chore(deps): bump github.com/prometheus/client_model from 0.2.0 to 0.3.0 (#9885)
* [a6e5b6ce7](https://github.com/argoproj/argo-workflows/commit/a6e5b6ce78acd210f6d8f42439948ac771084db8) chore(deps): bump github.com/minio/minio-go/v7 from 7.0.41 to 7.0.42 (#9886)
* [c81b07145](https://github.com/argoproj/argo-workflows/commit/c81b071455c7850ae33ff842bf35275ef44a4065) chore(deps): bump github.com/valyala/fasttemplate from 1.2.1 to 1.2.2 (#9887)
* [ec5162983](https://github.com/argoproj/argo-workflows/commit/ec5162983fd5e3032e5d3162245eab28e41b694b) fix: P/R/C reporting in argo list -o wide. Fixes #9281 (#9874)
* [6c432d2c9](https://github.com/argoproj/argo-workflows/commit/6c432d2c980bd37be28ebb22d1e83b176993ce38) fix: upgrade python openapiclient version, fixes #9770 (#9840)
* [36646ef81](https://github.com/argoproj/argo-workflows/commit/36646ef81cb4775c1ef31861f01331bc75166e7b) fix: Support Kubernetes v1.24. Fixes #8320 (#9620)
* [05e1425f8](https://github.com/argoproj/argo-workflows/commit/05e1425f857264076e0de29124d4fbf74b4107b4) fix(server&ui): can't fetch inline artifact. Fixes #9817 (#9853)
* [ce3172804](https://github.com/argoproj/argo-workflows/commit/ce31728046cbfe0a58bfd31e20e63c7edec25437) feat(ui): Display detailed Start/End times in workflow-node-info. Fixes #7920 (#9834)
* [b323bb1e5](https://github.com/argoproj/argo-workflows/commit/b323bb1e570a6cbd347942bbce82e25a05c4ca92) fix(ui): view manifest error on inline node. Fixes #9841 (#9842)
* [9237a72f7](https://github.com/argoproj/argo-workflows/commit/9237a72f7999f375279d054232028e4931d737f3) fix(ui): containerset archive log query params. Fixes #9669 (#9833)
* [a752a583a](https://github.com/argoproj/argo-workflows/commit/a752a583a5b9295fddae5c2978ea5f4cee2687d2) fix: exit code always be '0' in windows container. Fixes #9797 (#9807)
* [af8347c36](https://github.com/argoproj/argo-workflows/commit/af8347c36d305a56c7c1355078b410f97e2ed3d5) chore(deps): Bump github.com/TwiN/go-color from v1.1.0 to v1.2.0 (#9794)
* [102c3ec22](https://github.com/argoproj/argo-workflows/commit/102c3ec22118a49ccfa75b9c3878d62057afb441) fix: migrated from distribution@v2.8.0 to distribution@v2.8.1. Fixes #9850 (#9851)
* [d4a907411](https://github.com/argoproj/argo-workflows/commit/d4a907411a7105ffda52a284e1059c6de9829bcf) fix: trigger startup.sh at devcontainer startup instead of create (#9831)
* [ca750d056](https://github.com/argoproj/argo-workflows/commit/ca750d056db8d2d4005cf2f1dadb32e79be9b76a) chore(deps): bump github.com/TwiN/go-color from 1.1.0 to 1.2.0 (#9822)
* [593eab25c](https://github.com/argoproj/argo-workflows/commit/593eab25cade9f2a5b71fdef028d3886ff5e0e3c) chore(deps): bump google.golang.org/api from 0.98.0 to 0.99.0 (#9823)
* [1670dca60](https://github.com/argoproj/argo-workflows/commit/1670dca6092b51781ed5e1f2d2522b0c0bca0ced) chore(deps): bump github.com/minio/minio-go/v7 from 7.0.40 to 7.0.41 (#9825)
* [e838214ed](https://github.com/argoproj/argo-workflows/commit/e838214ed452d4bff528da4a7a2f101ebf324277) chore(deps): bump cronstrue from 2.12.0 to 2.14.0 in /ui (#9826)
* [7d2081830](https://github.com/argoproj/argo-workflows/commit/7d2081830b8b77de37429958b7968d7073ef5f0c) chore(deps): bump nick-fields/retry from 2.8.1 to 2.8.2 (#9820)
* [f6a8b0130](https://github.com/argoproj/argo-workflows/commit/f6a8b0130dccb5a773fc52fd46354f8537d022cb) fix: SDK CI workflow (#9609)
* [faa0294f5](https://github.com/argoproj/argo-workflows/commit/faa0294f5c29fa10800f94677c21b7180d9b3da4) fix: fixed url encoded link template (#9792)
* [ebae212d7](https://github.com/argoproj/argo-workflows/commit/ebae212d709f039823737b495437c14898690376) fix(ui): missing url href formatting in template link. Fixes #9764 (#9790)
* [d4817efff](https://github.com/argoproj/argo-workflows/commit/d4817efffad2a1d96374f69f3b547bf3f9d758a9) fix: fix iam permissions to retrieve logs from aws s3 (#9798)
* [aa59b4374](https://github.com/argoproj/argo-workflows/commit/aa59b43748f78e599709add871af7ec14e1fd3c1) fix: enable when expressions to use expr; add new json variables to avoid expr conflicts (#9761)
* [0fc883a41](https://github.com/argoproj/argo-workflows/commit/0fc883a41c81c533c57ec64ca8c19279b38e60ec) fix: avoid nil pointer dereference. Fixes #9269 (#9787)
* [cd43bba6c](https://github.com/argoproj/argo-workflows/commit/cd43bba6c87d185bd1530c03c99b874eeceba966) fix: Send workflow UID to plugins. Fixes #8573 (#9784)
* [514aa050c](https://github.com/argoproj/argo-workflows/commit/514aa050cab63bba8e6af20700ad4aa7ed53bfd4) feat(server): server logs to be structured and add more log error #2308 (#9779)
* [f27fe08b1](https://github.com/argoproj/argo-workflows/commit/f27fe08b1b06ee86040371b5fa992b82b27d7980) fix: default not respected in setting global configmap params. Fixes #9745 (#9758)
* [dc48c8cf1](https://github.com/argoproj/argo-workflows/commit/dc48c8cf12eccb1cc447a4f9a32e1c7dfc4f93da) fix: Set scheduling constraints to the agent pod by the workflow. Fixes #9704 (#9771)
* [f767f39d8](https://github.com/argoproj/argo-workflows/commit/f767f39d86acb549ef29d8196f067280683afd4d) fix: artifactory not working. Fixes #9681 (#9782)
* [1fc6460fa](https://github.com/argoproj/argo-workflows/commit/1fc6460fa16b157b0d333b96d6d93b7d273ed91a) fix: Log early abort. Fixes #9573 (#9575)
* [f1bab8947](https://github.com/argoproj/argo-workflows/commit/f1bab8947c44f9fc0483dc6489b098e04e0510f7) fix: a WorkflowTemplate doesn't need to define workflow-level input p… (#9762)
* [b12b5f987](https://github.com/argoproj/argo-workflows/commit/b12b5f9875b2a070bbcb0a3a16154495c196e6b2) fix: SSO integration not considering HTTP_PROXY when making requests. Fixes #9259 (#9760)
* [529dc0fec](https://github.com/argoproj/argo-workflows/commit/529dc0fec443cd33171d32e7f798ceeaddef1587) feat(ui): add v3.4 feature info (#9777)
* [a8e37e9be](https://github.com/argoproj/argo-workflows/commit/a8e37e9bea5d586f8b1811fcbb8df668d00bdb31) fix: Concurrent map read and map write in agent. Fixes #9685 (#9689)
* [1bbdf0d2a](https://github.com/argoproj/argo-workflows/commit/1bbdf0d2ad5a74832ecff5a6e13a758bdf54e909) feat: Added workflow summary to workflow-list page. (#9693)
* [82201d521](https://github.com/argoproj/argo-workflows/commit/82201d521d91cfa2926584864edbdc8a15e9a5ad) chore(deps): bump cronstrue from 2.11.0 to 2.12.0 in /ui (#9774)
* [d7febc928](https://github.com/argoproj/argo-workflows/commit/d7febc92818fa2cbee5eb32cbf6169beb739673d) chore(deps): bump github.com/minio/minio-go/v7 from 7.0.39 to 7.0.40 (#9773)
* [d64b8d397](https://github.com/argoproj/argo-workflows/commit/d64b8d3976c4cd7592b9433be20547a80f28e289) fix: quick-start-\* manifests pointing to invalid httpbin image tag. Fixes #9659 (#9759)
* [de4ea2d51](https://github.com/argoproj/argo-workflows/commit/de4ea2d51262d86f8806fbb710c6b3ae14b24c7f) fix: `value` is required when parameter is of type `enum` (#9753)
* [2312cc9ca](https://github.com/argoproj/argo-workflows/commit/2312cc9ca4f26f06ccc107a10013ea903c10ec15) Revert "Add --tls-certificate-secret-name parameter to server command. Fixes #5582" (#9756)
* [d9d1968de](https://github.com/argoproj/argo-workflows/commit/d9d1968de80fa0ee19a5e46ceea5d2b4cf4b5475) fix: node links on UI use podName instead of workflow name (#9740)
* [9ac6df02e](https://github.com/argoproj/argo-workflows/commit/9ac6df02e7253df5e0764d6f29bda1ac1bdbb071) chore(deps): bump github.com/minio/minio-go/v7 from 7.0.37 to 7.0.39 (#9721)
* [0b957c128](https://github.com/argoproj/argo-workflows/commit/0b957c1289fd6c04b8c0f63ab18463de9074ac91) chore(deps): bump github.com/argoproj/argo-events from 1.7.2 to 1.7.3 (#9722)
* [4ba1a0f9b](https://github.com/argoproj/argo-workflows/commit/4ba1a0f9bcfc2a5cd6dd246b4b4635e2d8cecf6d) chore(deps): bump google.golang.org/api from 0.97.0 to 0.98.0 (#9719)

<details><summary><h3>Contributors</h3></summary>

* Aditya Shrivastava
* Alex Collins
* Andrii Chubatiuk
* Anil Kumar
* Dillen Padhiar
* Eddie Knight
* Felix
* Isitha Subasinghe
* Julie Vogelman
* Lukas Heppe
* Patrice Chalin
* Ricardo Rosales
* Rohan Kumar
* Saravanan Balasubramanian
* Shadow W
* Takumi Sue
* Tianchu Zhao
* TwiN
* Vũ Hải Lâm
* Yuan Tang
* Yuya Kakui
* alexdittmann
* botbotbot
* chen yangxue
* dependabot[bot]
* github-actions[bot]
* jibuji

</details>

## v3.4.1 (2022-09-30)

Full Changelog: [v3.4.0...v3.4.1](https://github.com/argoproj/argo-workflows/compare/v3.4.0...v3.4.1)

### Selected Changes

* [365b6df16](https://github.com/argoproj/argo-workflows/commit/365b6df1641217d1b21b77bb1c2fcb41115dd439) fix: Label on Artifact GC Task no longer exceeds max characters (#9686)
* [0851c36d8](https://github.com/argoproj/argo-workflows/commit/0851c36d8638833b9ecfe0125564e5635641846f) fix: Workflow-controller panic when stop a wf using plugin. Fixes #9587 (#9690)
* [2f5e7534c](https://github.com/argoproj/argo-workflows/commit/2f5e7534c44499a9efce51d12ff87f8c3f725a21) fix: ordering of functionality for setting and evaluating label expressions (#9661)
* [4e34979e1](https://github.com/argoproj/argo-workflows/commit/4e34979e1b132439fe1101a23b46e24a62c0368d) chore(deps): bump argo-events to 1.7.2 (#9624)
* [f0016e054](https://github.com/argoproj/argo-workflows/commit/f0016e054ec32505dcd7f7d610443ad380fc6651) fix: Remove LIST_LIMIT in workflow informer (#9700)
* [e08524d2a](https://github.com/argoproj/argo-workflows/commit/e08524d2acbd474f232f958e711d04d8919681e8) fix: Avoid controller crashes when running large number of workflows (#9691)
* [4158cf11a](https://github.com/argoproj/argo-workflows/commit/4158cf11ad2e5837a76d1194a99b38e6d66f7dd0) Adding Splunk as Argo Workflows User (#9697)
* [ff6aab34e](https://github.com/argoproj/argo-workflows/commit/ff6aab34ecbb5c0de26e36108cd1201c1e1ae2f5) Add --tls-certificate-secret-name parameter to server command. Fixes #5582 (#9423)
* [84c19ea90](https://github.com/argoproj/argo-workflows/commit/84c19ea909cbc5249f684133dcb5a8481a533dab) fix: render template vars in DAGTask before releasing lock.. Fixes #9395 (#9405)
* [b214161b3](https://github.com/argoproj/argo-workflows/commit/b214161b38642da75a38a100548d3809731746ff) fix: add authorization from cookie to metadata (#9663)
* [b219d85ab](https://github.com/argoproj/argo-workflows/commit/b219d85ab57092b37b0b26f9f7c4cfbf5a9bea9a) fix: retry ExecutorPlugin invocation on transient network errors Fixes: #9664 (#9665)
* [b96d446d6](https://github.com/argoproj/argo-workflows/commit/b96d446d666f704ba102077404bf0b7c472c1494) fix: Improve semaphore concurrency performance (#9666)
* [38b55e39c](https://github.com/argoproj/argo-workflows/commit/38b55e39cca03e54da1f38849b066b36e03ba240) fix: sh not available in scratch container but used in argoexec. Fixes #9654 (#9679)
* [67fc0acab](https://github.com/argoproj/argo-workflows/commit/67fc0acabc4a03f374195246b362b177893866b1) chore(deps): bump golangci-lint to v1.49.0 (#9639)
* [56454d0c8](https://github.com/argoproj/argo-workflows/commit/56454d0c8d8d4909e23f0938e561ad2bdb02cef2) chore(deps): bump github.com/minio/minio-go/v7 from 7.0.36 to 7.0.37 (#9673)
* [49c47cbad](https://github.com/argoproj/argo-workflows/commit/49c47cbad0408adaf1371da36c3ece340fdecd65) chore(deps): bump cloud.google.com/go/storage from 1.26.0 to 1.27.0 (#9672)
* [e6eb02fb5](https://github.com/argoproj/argo-workflows/commit/e6eb02fb529b7952227dcef091853edcf20f8248) fix: broken archived workflows ui. Fixes #9614, #9433 (#9634)
* [e556fe3eb](https://github.com/argoproj/argo-workflows/commit/e556fe3eb355bf9ef31a1ef8b057c680a5c24f06) fix: Fixed artifact retrieval when templateRef in use. Fixes #9631, #9644. (#9648)
* [72d3599b9](https://github.com/argoproj/argo-workflows/commit/72d3599b9f75861414475a39950879bddbc4e154) fix: avoid panic when not passing AuthSupplier (#9586)
* [4ab943528](https://github.com/argoproj/argo-workflows/commit/4ab943528c8e1b510549e9c860c03adb8893e96b) chore(deps): bump google.golang.org/api from 0.95.0 to 0.96.0 (#9600)

<details><summary><h3>Contributors</h3></summary>

* Adam
* Alex Collins
* Brian Loss
* Christopher Cutajar
* Dakota Lillie
* Jesse Suen
* Julie Vogelman
* Rohan Kumar
* Seokju Hong
* Takumi Sue
* Vladimir Ivanov
* William Van Hevelingen
* Yuan Tang
* chen yangxue
* dependabot[bot]
* emagana
* github-actions[bot]
* jsvk

</details>

## v3.4.0 (2022-09-18)

Full Changelog: [v3.4.0-rc4...v3.4.0](https://github.com/argoproj/argo-workflows/compare/v3.4.0-rc4...v3.4.0)

### Selected Changes

* [047952afd](https://github.com/argoproj/argo-workflows/commit/047952afd539d06cae2fd6ba0b608b19c1194bba) fix: SDK workflow file
* [97328f1ed](https://github.com/argoproj/argo-workflows/commit/97328f1ed3885663b780f43e6b553208ecba4d3c) chore(deps): bump classnames and @types/classnames in /ui (#9603)
* [47544cc02](https://github.com/argoproj/argo-workflows/commit/47544cc02a8663b5b69e4c213a382ff156deb63e) feat: Support retrying complex workflows with nested group nodes (#9499)
* [30bd96b4c](https://github.com/argoproj/argo-workflows/commit/30bd96b4c030fb728a3da78e0045982bf778d554) fix: Error message if cronworkflow failed to update (#9583)

<details><summary><h3>Contributors</h3></summary>

* 66li
* Ashish Kurmi
* Brian Loss
* JM
* Julie Vogelman
* Saravanan Balasubramanian
* Yuan Tang
* dependabot[bot]
* github-actions[bot]
* zychina

</details>

## v3.4.0-rc4 (2022-09-10)

Full Changelog: [v3.4.0-rc3...v3.4.0-rc4](https://github.com/argoproj/argo-workflows/compare/v3.4.0-rc3...v3.4.0-rc4)

### Selected Changes

* [3950f8c1c](https://github.com/argoproj/argo-workflows/commit/3950f8c1c12ff7451b3e1be96b2ba108025a9677) chore(deps): bump google.golang.org/api from 0.94.0 to 0.95.0 (#9561)
* [8310bdbc9](https://github.com/argoproj/argo-workflows/commit/8310bdbc9d07f87640d944b949e465a044148368) chore(deps): bump github.com/coreos/go-oidc/v3 from 3.3.0 to 3.4.0 (#9560)
* [baaa8d0a9](https://github.com/argoproj/argo-workflows/commit/baaa8d0a9e90f5234ce7d02cbc33f8756a3ad4da) chore(deps): bump github.com/minio/minio-go/v7 from 7.0.35 to 7.0.36 (#9558)
* [aab923452](https://github.com/argoproj/argo-workflows/commit/aab92345267e9e0562ee8495f49ac6d80e06ae28) chore(deps): bump github.com/spf13/viper from 1.12.0 to 1.13.0 (#9559)
* [ec7c210c9](https://github.com/argoproj/argo-workflows/commit/ec7c210c9743d8f85d528d5593bc7390d73ff534) fix: use urlencode instead of htmlencode to sanitize url (#9538)
* [3a3f15997](https://github.com/argoproj/argo-workflows/commit/3a3f1599718453ca79800cfc28f6631ee780911b) fix: enable workflow-aggregate-roles to treat workflowtaskresults. Fixes #9545 (#9546)
* [9d66b69f0](https://github.com/argoproj/argo-workflows/commit/9d66b69f0bca92d7ef0c9aa67e87a2e334797530) fix: for pod that's been GC'ed we need to get the log from the artifact (#9540)
* [34a4e48c3](https://github.com/argoproj/argo-workflows/commit/34a4e48c3f412ba89cd0491469d13a14fdaf51b3) chore(deps): bump github.com/minio/minio-go/v7 from 7.0.34 to 7.0.35 (#9502)
* [ef6bd5710](https://github.com/argoproj/argo-workflows/commit/ef6bd5710e5780afe40321f4d384471d9e02197c) fix: Capture exit code of signaled containers. Fixes #9415 (#9523)
* [6e2f15f9e](https://github.com/argoproj/argo-workflows/commit/6e2f15f9eea82f1344f139800869f9e7fd255b04) feat: added support for DAG task name as variables in workflow templates (#9387)
* [f27475feb](https://github.com/argoproj/argo-workflows/commit/f27475feb850dc43e07c3c5215cc9638947f0859) fix: default to 'main' container in Sensor logs. Fixes #9459 (#9438)
* [c00fbf88f](https://github.com/argoproj/argo-workflows/commit/c00fbf88f15104673b05ba5e109a72fed84dd38e) feat: Add node ID to node info panel (#9500)
* [2a80a2c1a](https://github.com/argoproj/argo-workflows/commit/2a80a2c1a9b0a2370f547492ef9168ee583077f5) fix: revert accidental commit in UI logs viewer (#9515)
* [b9d02cfd5](https://github.com/argoproj/argo-workflows/commit/b9d02cfd59c72b2bc8e437e6591ca4a145a3eb9b) chore(deps): bump cloud.google.com/go/storage from 1.25.0 to 1.26.0 (#9506)
* [9004f5e26](https://github.com/argoproj/argo-workflows/commit/9004f5e263a4ead8a5be4a4a09db03064eb1d453) chore(deps): bump google.golang.org/api from 0.93.0 to 0.94.0 (#9505)
* [a2c20d70e](https://github.com/argoproj/argo-workflows/commit/a2c20d70e8885937532055b8c2791799020057ec) chore(deps): bump react-monaco-editor from 0.49.0 to 0.50.1 in /ui (#9509)
* [1b09c8641](https://github.com/argoproj/argo-workflows/commit/1b09c8641ad11680b90dba582b3eae98dcee01c3) chore(deps): bump github.com/coreos/go-oidc/v3 from 3.2.0 to 3.3.0 (#9504)
* [4053ddf08](https://github.com/argoproj/argo-workflows/commit/4053ddf081755df8819a4a33ce558c92235ea81d) chore(deps): bump github.com/aliyun/aliyun-oss-go-sdk from 2.2.4+incompatible to 2.2.5+incompatible (#9503)
* [06d295752](https://github.com/argoproj/argo-workflows/commit/06d29575210d7b61ca7c7f2fb8e28fdd6c3d5637) feat: log format option for main containers (#9468)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins
* Julie Vogelman
* Rohan Kumar
* Takao Shibata
* Thomas Bonfort
* Tianchu Zhao
* Tim Collins
* Yuan Tang
* dependabot[bot]
* github-actions[bot]
* jsvk

</details>

## v3.4.0-rc3 (2022-08-31)

Full Changelog: [v3.4.0-rc2...v3.4.0-rc3](https://github.com/argoproj/argo-workflows/compare/v3.4.0-rc2...v3.4.0-rc3)

### Selected Changes

* [b941fbcab](https://github.com/argoproj/argo-workflows/commit/b941fbcaba087d5c5569573d1ef1a027313174ce) feat: improve e2e test for ArtifactGC (#9448)
* [94608d1dd](https://github.com/argoproj/argo-workflows/commit/94608d1ddc8781a55563f52ea65476dc99a54f94) feat: added support for artifact GC on GCS (#9420)
* [26ab0aed8](https://github.com/argoproj/argo-workflows/commit/26ab0aed8ba19571ffe3a2b048fcb43cbd1986e3) fix: link to "get artifacts from logs" was assuming Node ID was equal to Pod Name (#9464)
* [9cce91ea0](https://github.com/argoproj/argo-workflows/commit/9cce91ea0ca748cb35bd653c6f401d1aed97e6e8) Update USERS.md (#9471)
* [7118e1224](https://github.com/argoproj/argo-workflows/commit/7118e1224283ecb894794fdd72526089409e1476) feat: support slash in synchronization lock names. Fixes #9394 (#9404)
* [ff4109928](https://github.com/argoproj/argo-workflows/commit/ff4109928bd09a1b1d716cbdf82bd3ca132276d1) fix: Descendants of suspended nodes need to be removed when retrying workflow (#9440)
* [a09172afa](https://github.com/argoproj/argo-workflows/commit/a09172afafdb98ab362058618b5dc61980f0254e) fix: Incorrect alignment for archived workflow. Fixes #9433 (#9439)
* [04d19435c](https://github.com/argoproj/argo-workflows/commit/04d19435cb07e8815f1f95cca6751f8ce6b4bec1) fix: Properly reset suspended and skipped nodes when retrying (#9422)
* [de6b5ae6f](https://github.com/argoproj/argo-workflows/commit/de6b5ae6fa39693b7cd7777b9fcff9ff291476dd) fix(executor): Resource template gets incorrect plural for certain types (#9396)
* [3ddbb5e00](https://github.com/argoproj/argo-workflows/commit/3ddbb5e009f39fdb31cdaa7d77fca71dc3ae3f0e) fix: Only validate manifests for certain resource actions. Fixes #9418 (#9419)
* [a91e0041c](https://github.com/argoproj/argo-workflows/commit/a91e0041c9583deb48751c666dbbef111f3a56f9) fix: Workflow level http template hook status update. Fixes #8529 (#8586)
* [343c29819](https://github.com/argoproj/argo-workflows/commit/343c29819ac92d35f5db8a0de432f63df148ea31) fix: Argo waiter: invalid memory address or nil pointer dereference  (#9408)
* [6f19e50a4](https://github.com/argoproj/argo-workflows/commit/6f19e50a41a17dbf06e6281f005ade6a2f19dba4) fix: Invalid memory address or nil pointer dereference (#9409)
* [7d9319b60](https://github.com/argoproj/argo-workflows/commit/7d9319b60d0bc417b25d35968c1619e51c13b7ec) Fix: UI to reflect Template.ArchiveLocation when showing Artifact's bucket in URN (#9351)
* [fa66ed8e8](https://github.com/argoproj/argo-workflows/commit/fa66ed8e8bc20c4d759eb923b99dd6641ceafa86) chore(deps): bump github.com/tidwall/gjson from 1.14.2 to 1.14.3 (#9401)

<details><summary><h3>Contributors</h3></summary>

* Abirdcfly
* Brian Tate
* Julie Vogelman
* Mriyam Tamuli
* Rohan Kumar
* Saravanan Balasubramanian
* Tim Collins
* William Reed
* Xianglin Gao
* Yuan Tang
* dependabot[bot]
* jsvk
* kasteph
* lkad

</details>

## v3.4.0-rc2 (2022-08-18)

Full Changelog: [v3.4.0-rc1...v3.4.0-rc2](https://github.com/argoproj/argo-workflows/compare/v3.4.0-rc1...v3.4.0-rc2)

### Selected Changes

* [6e8d1629d](https://github.com/argoproj/argo-workflows/commit/6e8d1629d9eebf78dce07f180ee99a233e422a80) fix: Artifact panel crashes when viewing artifacts. Fixes #9391 (#9392)
* [aa23a9ec8](https://github.com/argoproj/argo-workflows/commit/aa23a9ec8b9fc95593fdc41e1632412542a9c050) fix: Exit handle and Lifecycle hook to access {steps/tasks status} (#9229)
* [74cdf5d87](https://github.com/argoproj/argo-workflows/commit/74cdf5d870cc4d0b5576e6d78da7a6fde6a1be99) fix: improper selfLinks for cluster-scoped resources. Fixes #9320 (#9375)
* [f53d4834a](https://github.com/argoproj/argo-workflows/commit/f53d4834a208f39797637d7fad744caf0540cff8) fix: Panic on nill pointer when running a workflow with restricted parallelism (#9385)
* [c756291f7](https://github.com/argoproj/argo-workflows/commit/c756291f701296b36411ccdd639a965a302a5af8) fix: removed error check which prevented deleting successful artGC wfs.  (#9383)
* [81e3d23e7](https://github.com/argoproj/argo-workflows/commit/81e3d23e730d80f24c90feb283fa3ff3b358e215) chore(deps): bump google.golang.org/api from 0.91.0 to 0.93.0 (#9381)
* [62b0db982](https://github.com/argoproj/argo-workflows/commit/62b0db9822ef93732544667739b33c1d9792ccf9) fix(ui): Correctly show icons in DAG. Fixes #9372 & #9373 (#9378)
* [47f59c050](https://github.com/argoproj/argo-workflows/commit/47f59c050ed579cdf9e01eddf0f388ac52fe5713) chore(deps): bump cloud.google.com/go/storage from 1.24.0 to 1.25.0 (#9357)
* [65670a402](https://github.com/argoproj/argo-workflows/commit/65670a402b1e9a96d246fd2ee363dd27a7f3149b) fix: Fix blank workflow details page after workflow submission (#9377)
* [6d08098a8](https://github.com/argoproj/argo-workflows/commit/6d08098a887c701cfffb2ea57f0391d6f7f5d489) feat: add argo delete --force. Fixes #9315. (#9321)
* [12466b7c9](https://github.com/argoproj/argo-workflows/commit/12466b7c9138052150afa6e0e81964d91a0538f5) fix: Retry for http timeout error. Fixes #9271 (#9335)
* [fd08b0339](https://github.com/argoproj/argo-workflows/commit/fd08b0339506f8f11288393061cf8c2eb155403a) fix: ArtifactGC e2e test was looking for the wrong artifact names (#9353)
* [b430180d2](https://github.com/argoproj/argo-workflows/commit/b430180d275adac05d64b82613134b926d4405f1) fix: Deleted pods are not tracked correctly when retrying workflow (#9340)
* [e12c697b7](https://github.com/argoproj/argo-workflows/commit/e12c697b7be2547cdffd18c73bf39e10dfa458f0) feat: fix bugs in retryWorkflow if failed pod node has children nodes. Fix #9244 (#9285)
* [61f252f1d](https://github.com/argoproj/argo-workflows/commit/61f252f1d2083e5e9f262d0acd72058571e27708) fix: TestWorkflowStepRetry's comment accurately reflects what it does. (#9234)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins
* Dillen Padhiar
* Julie Vogelman
* Kyle Wong
* Niels ten Boom
* Robert Kotcher
* Saravanan Balasubramanian
* Savin
* Yash Hegde
* Yuan Tang
* dependabot[bot]
* github-actions[bot]
* jingkai
* smile-luobin

</details>

## v3.4.0-rc1 (2022-08-09)

Full Changelog: [v3.3.10...v3.4.0-rc1](https://github.com/argoproj/argo-workflows/compare/v3.3.10...v3.4.0-rc1)

### Selected Changes

* [f481e3b74](https://github.com/argoproj/argo-workflows/commit/f481e3b7444eb9cbb5c4402a27ef209818b1d817) feat: fix workflow hangs during executeDAGTask. Fixes #6557 (#8992)
* [ec213c070](https://github.com/argoproj/argo-workflows/commit/ec213c070d92f4ac937f55315feab0fcc108fed5) Fixes #8622: fix http1 keep alive connection leak (#9298)
* [0d77f5554](https://github.com/argoproj/argo-workflows/commit/0d77f5554f251771a175a95fc80eeb12489e42b4) fix: Look in correct bucket when downloading artifacts (Template.ArchiveLocation configured) (#9301)
* [b356cb503](https://github.com/argoproj/argo-workflows/commit/b356cb503863da43c0cc5e1fe667ebf602cb5354) feat: Artifact GC (#9255)
* [e246abec1](https://github.com/argoproj/argo-workflows/commit/e246abec1cbe6be8cb8955f798602faf619a943f) feat: modify "argoexec artifact delete" to handle multiple artifacts. Fixes #9143 (#9291)
* [ffefe9402](https://github.com/argoproj/argo-workflows/commit/ffefe9402885a275e7a26c12b5a5e52e7522c4d7) chore(deps): bump github.com/minio/minio-go/v7 from 7.0.32 to 7.0.34 (#9304)
* [c0d26d61c](https://github.com/argoproj/argo-workflows/commit/c0d26d61c02f7fb4140a089139f8984df91eaaf9) chore(deps): bump cron-parser from 4.5.0 to 4.6.0 in /ui (#9307)
* [8d06a83bc](https://github.com/argoproj/argo-workflows/commit/8d06a83bccba87886163143e959369f0d0240943) chore(deps): bump github.com/prometheus/client_golang from 1.12.2 to 1.13.0 (#9306)
* [f83346959](https://github.com/argoproj/argo-workflows/commit/f83346959cf5204fe80b6b70e4d823bf481579fe) chore(deps): bump google.golang.org/api from 0.90.0 to 0.91.0 (#9305)
* [63876713e](https://github.com/argoproj/argo-workflows/commit/63876713e809ceca8e1e540a38b5ad0e650cbb2a) chore(deps): bump github.com/tidwall/gjson from 1.14.1 to 1.14.2 (#9303)
* [06b0a8cce](https://github.com/argoproj/argo-workflows/commit/06b0a8cce637db1adae0bae91670e002cfd0ae4d) fix(gcs): Wrap errors using `%w` to make retrying work (#9280)
* [083f3a21a](https://github.com/argoproj/argo-workflows/commit/083f3a21a601e086ca48d2532463a858cc8b316b) fix: pass correct error obj for azure blob failures (#9276)
* [55d15aeb0](https://github.com/argoproj/argo-workflows/commit/55d15aeb03847771e2b48f11fa84f88ad1df3e7c) feat: support zip for output artifacts archive. Fixes #8861 (#8973)
* [a51e833d9](https://github.com/argoproj/argo-workflows/commit/a51e833d9eea18ce5ef7606e55ddd025efa85de1) chore(deps): bump google.golang.org/api from 0.89.0 to 0.90.0 (#9260)
* [2d1758fe9](https://github.com/argoproj/argo-workflows/commit/2d1758fe90fd60b37d0dfccb55c3f79d8a897289) fix: retryStrategy.Limit is now read properly for backoff strategy. Fixes #9170. (#9213)
* [b565bf358](https://github.com/argoproj/argo-workflows/commit/b565bf35897f529bbb446058c24b72d506024e29) Fix: user namespace override (Fixes #9266) (#9267)
* [0c24ca1ba](https://github.com/argoproj/argo-workflows/commit/0c24ca1ba8a5c38c846d595770e16398f6bd84a5) fix: TestParallel 503 with external url (#9265)
* [fd6c7a7ec](https://github.com/argoproj/argo-workflows/commit/fd6c7a7ec1f2053f9fdd03451d7d29b1339c0408) feat: Add custom event aggregator function with annotations (#9247)
* [be6ba4f77](https://github.com/argoproj/argo-workflows/commit/be6ba4f772f65588af7c79cc9351ff6dea63ed16) fix: add ServiceUnavailable to s3 transient errors list Fixes #9248 (#9249)
* [51538235c](https://github.com/argoproj/argo-workflows/commit/51538235c7a70b89855dd3b96d97387472bdbade) chore(deps): bump github.com/minio/minio-go/v7 from 7.0.31 to 7.0.32 (#9253)
* [5cf5150ef](https://github.com/argoproj/argo-workflows/commit/5cf5150efe1694bb165e98c1d7509f9987d4f524) chore(deps): bump cloud.google.com/go/storage from 1.22.1 to 1.24.0 (#9252)
* [454f19ac8](https://github.com/argoproj/argo-workflows/commit/454f19ac8959f3e0db87bb34ec8f7099558aa737) chore(deps): bump google.golang.org/api from 0.87.0 to 0.89.0 (#9251)
* [6f8592228](https://github.com/argoproj/argo-workflows/commit/6f8592228668457a8b1db072cc53db2c5b01de55) chore(deps): bump github.com/sirupsen/logrus from 1.8.1 to 1.9.0 (#9214)
* [769896eb5](https://github.com/argoproj/argo-workflows/commit/769896eb5bf0a7d8db1a94b423e5bc16cf09d5aa) feat: APIratelimit headers and doc (#9206)
* [bcb596270](https://github.com/argoproj/argo-workflows/commit/bcb59627072c3b4f0cd1cef12f499ec3d8e87815) ui: remove workflowlist searchbox (#9208)
* [15fdf4903](https://github.com/argoproj/argo-workflows/commit/15fdf4903a05c7854656f59f61a676362fe551c6) fix: line return in devcontainer host file  (#9204)
* [44731d671](https://github.com/argoproj/argo-workflows/commit/44731d671d425b0709bab5c5e27ed7c42a0ee92d) feat: adding new CRD type "ArtifactGCTask"  (#9184)
* [d5d4628a3](https://github.com/argoproj/argo-workflows/commit/d5d4628a3573a0e1a75c367243e259844320e021) fix: Set namespace to user namespace obtained from /userinfo service (#9191)
* [e4489f5d1](https://github.com/argoproj/argo-workflows/commit/e4489f5d12c4f62421c87c69d8b997aad71fdea6) feat: log format option for wait and init containers. Fixes #8986 (#9169)
* [573fe98ff](https://github.com/argoproj/argo-workflows/commit/573fe98ffaa119b607bb5d4aafc1fb3c70a4c564) fix: remove unused argument which is triggering in lint (needed for PRs to pass CI) (#9186)
* [1af892133](https://github.com/argoproj/argo-workflows/commit/1af892133cd5b9e6ac22fc61bd4eabd84c568e89) feat: api ratelimiter for argoserver (#8993)
* [0f1d1d9b7](https://github.com/argoproj/argo-workflows/commit/0f1d1d9b7ef9b602b82123a9d92c212b50ac01e1) fix: support RemainingItemCount in archivedWrokflow (#9118)
* [aea581e02](https://github.com/argoproj/argo-workflows/commit/aea581e027fcd0675e785f413e964c588af304ad) fix: Incorrect link to workflows list with the same author (#9173)
* [fd6f3c263](https://github.com/argoproj/argo-workflows/commit/fd6f3c263412a1174de723470a14721b220c4651) feat: Add support for Azure Blob Storage artifacts Fixes #1540 (#9026)
* [26ff2e8a1](https://github.com/argoproj/argo-workflows/commit/26ff2e8a17ff68628090e18a3f246ab87fe950a3) chore(deps): bump google.golang.org/api from 0.86.0 to 0.87.0 (#9157)
* [877f36f37](https://github.com/argoproj/argo-workflows/commit/877f36f370d7ef00a1b8f136bb157e64c1e2769a) fix: Workflow details accessing undefined templateRef. Fixes #9167 (#9168)
* [6c20202ca](https://github.com/argoproj/argo-workflows/commit/6c20202cae8e62bb6c04a067a269e964d181e864) feat: make node info side panel resizable. Fixes #8917 (#8963)
* [96b98dafb](https://github.com/argoproj/argo-workflows/commit/96b98dafbdde5770d4d92c469e13ca81734a753f) chore(deps): bump github.com/prometheus/common from 0.35.0 to 0.37.0 (#9158)
* [cbe17105d](https://github.com/argoproj/argo-workflows/commit/cbe17105d91517f37cafafb49ad5f422b895c239) chore(deps): bump github.com/minio/minio-go/v7 from 7.0.30 to 7.0.31 (#9130)
* [9bbf7e0f0](https://github.com/argoproj/argo-workflows/commit/9bbf7e0f092f0d76c7419d291d3f9dba016b2f3c) feat: Support overriding parameters when retry/resubmit workflows (#9141)
* [42729ff75](https://github.com/argoproj/argo-workflows/commit/42729ff7542760bd27b08a7347a603d8f232466e) fix: Workflow retry should also reset the selected nodes (#9156)
* [559b59c0a](https://github.com/argoproj/argo-workflows/commit/559b59c0a2b9b3254740edf634de8a1c63c84ab0) feat: report Artifact GC failures in user interface. Fixes #8518 (#9115)
* [56d0c664a](https://github.com/argoproj/argo-workflows/commit/56d0c664ad96c95ca6c2311b2d1559dd423a5e4d) fix: Do not error when getting log artifacts from GCS. Fixes #8746 (#9155)
* [2b92b1aef](https://github.com/argoproj/argo-workflows/commit/2b92b1aefbf1e6a12476b946f05559c9b05fffef) fix: Fixed swagger error. Fixes #8922 (#9078)
* [57bac335a](https://github.com/argoproj/argo-workflows/commit/57bac335afac2c28a4eb5ccf1fa97bb5bba63e97) feat: refactoring e2e test timeouts to support multiple environments. (#8925)
* [921ae1ebf](https://github.com/argoproj/argo-workflows/commit/921ae1ebf5f849d4f684c79dee375205f05cfca9) chore(deps): bump moment from 2.29.3 to 2.29.4 in /ui (#9131)
* [c149dc53c](https://github.com/argoproj/argo-workflows/commit/c149dc53c78571778b0589d977dd0445e75d9eec) chore(deps): bump github.com/stretchr/testify from 1.7.5 to 1.8.0 (#9097)
* [a0c9e66c1](https://github.com/argoproj/argo-workflows/commit/a0c9e66c1d1cb3d83c5150814c4b8ccd9acdcfb1) chore(deps): bump react-monaco-editor from 0.48.0 to 0.49.0 in /ui (#9104)
* [0f0e25e03](https://github.com/argoproj/argo-workflows/commit/0f0e25e03ffe00f79e74087044ecd080f2d6242a) [Snyk] Upgrade swagger-ui-react from 4.10.3 to 4.12.0 (#9072)
* [8fc78ca9d](https://github.com/argoproj/argo-workflows/commit/8fc78ca9dce321f2173fba7735e4b4bd48df1b6c) chore(deps): bump cronstrue from 1.125.0 to 2.11.0 in /ui (#9102)
* [50a4d0044](https://github.com/argoproj/argo-workflows/commit/50a4d00443cfc53976db6227394784bbf34fe239) feat: Support retry on nested DAG and node groups (#9028)
* [20f8582a9](https://github.com/argoproj/argo-workflows/commit/20f8582a9e71effee220b160b229b5fd68bf7c95) feat(ui): Add workflow author information to workflow summary and drawer (#9119)
* [154d849b3](https://github.com/argoproj/argo-workflows/commit/154d849b32082a4211487b6dbebbae215b97b9ee) chore(deps): bump cron-parser from 4.4.0 to 4.5.0 in /ui (#9101)
* [ba225d3aa](https://github.com/argoproj/argo-workflows/commit/ba225d3aa586dd9e6770ec1b2f482f1c15fe2add) chore(deps): bump google.golang.org/api from 0.85.0 to 0.86.0 (#9096)
* [ace228486](https://github.com/argoproj/argo-workflows/commit/ace2284869a9574602b602a5bdf4592cd6ae8376) chore(deps): bump github.com/minio/minio-go/v7 from 7.0.29 to 7.0.30 (#9098)
* [61211f9db](https://github.com/argoproj/argo-workflows/commit/61211f9db1568190dd46b7469fa79eb6530bba73) fix: Add workflow failures before hooks run. Fixes #8882 (#9009)
* [c1154ff97](https://github.com/argoproj/argo-workflows/commit/c1154ff975bcb580554f78f393fd908b1f64ea6a) feat: redirect to archive on workflow absence. Fixes #7745 (#7854)
* [f5f1a3438](https://github.com/argoproj/argo-workflows/commit/f5f1a34384ab4bbbebd9863711a3047a08ced7fb) fix: sync lock should be released only if we're retrying (#9063)
* [146e38a3f](https://github.com/argoproj/argo-workflows/commit/146e38a3f91ac8a7b9b749d96c54bd3eab2ce1ab) chore!: Remove dataflow pipelines from codebase (#9071)
* [92eaadffc](https://github.com/argoproj/argo-workflows/commit/92eaadffcd0c244f05b23d4f177fd53f000b1a99) feat: inform users on UI if an artifact will be deleted. Fixes #8667 (#9056)
* [d0cfc6d10](https://github.com/argoproj/argo-workflows/commit/d0cfc6d10b11d9977007bb14373e699e604c1b74) feat: UI default to the namespace associated with ServiceAccount. Fixes #8533 (#9008)
* [1ccc120cd](https://github.com/argoproj/argo-workflows/commit/1ccc120cd5392f877ecbb328cbf5304e6eb89783) feat: added support for binary HTTP template bodies. Fixes #6888 (#8087)
* [443155dea](https://github.com/argoproj/argo-workflows/commit/443155deaa1aa9e19688de0580840bd0f8598dd5) feat: If artifact has been deleted, show a message to that effect in the iFrame in the UI (#8966)
* [11801d044](https://github.com/argoproj/argo-workflows/commit/11801d044cfddfc8100d973e91ddfe9a1252a028) chore(deps): bump superagent from 7.1.6 to 8.0.0 in /ui (#9052)
* [c30493d72](https://github.com/argoproj/argo-workflows/commit/c30493d722c2fd9aa5ccc528327759d96f99fb23) chore(deps): bump github.com/prometheus/common from 0.34.0 to 0.35.0 (#9049)
* [74c1e86b8](https://github.com/argoproj/argo-workflows/commit/74c1e86b8bc302780f36a364d7adb98184bf6e45) chore(deps): bump google.golang.org/api from 0.83.0 to 0.85.0 (#9044)
* [77be291da](https://github.com/argoproj/argo-workflows/commit/77be291da21c5057d0c966adce449a7f9177e0db) chore(deps): bump github.com/stretchr/testify from 1.7.2 to 1.7.5 (#9045)
* [278f61c46](https://github.com/argoproj/argo-workflows/commit/278f61c46309b9df07ad23497a4fd97817af93cc) chore(deps): bump github.com/spf13/cobra from 1.4.0 to 1.5.0 (#9047)
* [d90f11c3e](https://github.com/argoproj/argo-workflows/commit/d90f11c3e4c1f7d88be3220f57c3184d7beaddaf) [Snyk] Upgrade superagent from 7.1.3 to 7.1.4 (#9020)
* [6e962fdca](https://github.com/argoproj/argo-workflows/commit/6e962fdcab5effbb4ac12180249019d7d6241b8c) feat: sanitize config links (#8779)
* [89f3433bf](https://github.com/argoproj/argo-workflows/commit/89f3433bf7cbca7092952aa8ffc5e5c254f28999) fix: workflow.status is now set properly in metrics. Fixes #8895 (#8939)
* [2aa32aea5](https://github.com/argoproj/argo-workflows/commit/2aa32aea5eaf325bc6a3eff852f2ff0052366bf6) fix: check for nil, and add logging to expose root cause of panic in Issue 8968 (#9010)
* [62287487a](https://github.com/argoproj/argo-workflows/commit/62287487a0895a457804f0ac97fdf9c9413dd2ab) fix: Treat 'connection reset by peer' as a transient network error. Fixes #9013 (#9017)
* [2e3177617](https://github.com/argoproj/argo-workflows/commit/2e31776175b2cbb123278920e30807244e2f7a3b) fix: add nil check for retryStrategy.Limit in deadline check. Fixes #8990 (#8991)
* [73487fbee](https://github.com/argoproj/argo-workflows/commit/73487fbeeb645ac8f6229f98aed2ec6eec756571) chore(deps): bump github.com/minio/minio-go/v7 from 7.0.27 to 7.0.29 (#9004)
* [e34e378af](https://github.com/argoproj/argo-workflows/commit/e34e378af05b0ffde14b89e8d9eec9964a903002) chore(deps): bump github.com/argoproj/pkg from 0.13.2 to 0.13.3 (#9002)
* [89f82cea4](https://github.com/argoproj/argo-workflows/commit/89f82cea4b3f3f40d1666d2469ab3a97e3665fdd) feat: log workflow size before hydrating/dehydrating. Fixes #8976 (#8988)
* [a1535fa44](https://github.com/argoproj/argo-workflows/commit/a1535fa446d15bae56656d20577fdbb000353cc2) fix: Workflow Duration metric shouldn't increase after workflow complete (#8989)
* [6106ac722](https://github.com/argoproj/argo-workflows/commit/6106ac7229eeaac9132f8df595b569de2bc68ccf) feat: Support loading manifest from artifacts for resource templates. Fixes #5878 (#8657)
* [e0a1afa91](https://github.com/argoproj/argo-workflows/commit/e0a1afa91d8e51ba2c6aed6c604f2a69bdb1b387) fix: sync cluster Workflow Template Informer before it's used (#8961)
* [6c244f3cb](https://github.com/argoproj/argo-workflows/commit/6c244f3cb400f69b641d7e59c5215806a2870604) fix: long code blocks overflow in ui. Fixes #8916 (#8947)
* [e31ffcd33](https://github.com/argoproj/argo-workflows/commit/e31ffcd339370d6000f86d552845d7d378620d29) fix: Correct kill command. Fixes #8687 (#8908)
* [263977967](https://github.com/argoproj/argo-workflows/commit/263977967a47f24711b9f6110fe950c47d8c5f08) chore(deps): bump google.golang.org/api from 0.82.0 to 0.83.0 (#8951)
* [e96b1b3fd](https://github.com/argoproj/argo-workflows/commit/e96b1b3fd4e27608de8a94763782bd2d41cd5761) chore(deps): bump github.com/stretchr/testify from 1.7.1 to 1.7.2 (#8950)
* [107ed932d](https://github.com/argoproj/argo-workflows/commit/107ed932de466a89feb71dc04950c86d98747cc5) feat: add indexes for improve archived workflow performance. Fixes #8836 (#8860)
* [1d4edb433](https://github.com/argoproj/argo-workflows/commit/1d4edb4333ce4e5efeb44a199b390c3d9d02fc25) feat: Date range filter for workflow list. Fixes #8329 (#8596)
* [a6eef41bf](https://github.com/argoproj/argo-workflows/commit/a6eef41bf961cda347b9a9bd8476fc33e3a467a9) feat: add artifact delete to argoexec CLI. Fixes #8669 (#8913)
* [416fce705](https://github.com/argoproj/argo-workflows/commit/416fce70543059cc81753ba5131b1661a13a0fed) fix: Fork sub-process. Fixes #8454 (#8906)
* [750c4e1f6](https://github.com/argoproj/argo-workflows/commit/750c4e1f699b770a309843f2189b4e703305e44f) fix: Only signal running containers, ignore failures. (#8909)
* [ede1a39e7](https://github.com/argoproj/argo-workflows/commit/ede1a39e7cb48890aa5d4c8221e2c9d94e7ef007) fix: workflowMetadata needs to be loaded into globalParams in both ArgoServer and Controller (#8907)
* [df3764925](https://github.com/argoproj/argo-workflows/commit/df37649251f5791c40802defd923dd735924eb3a) Add left-margin to the question circle next to parameter name in Submit Workflow Panel (#8927)
* [1e17f7ff5](https://github.com/argoproj/argo-workflows/commit/1e17f7ff5232067c9c1c05bfa55322e41e0915d7) chore(deps): bump google.golang.org/api from 0.81.0 to 0.82.0 (#8914)
* [7dacb5bca](https://github.com/argoproj/argo-workflows/commit/7dacb5bcaeae8e3be64bb1fbf54024401d42d867) fix: Fixed Swagger error. Fixes #8830 (#8886)
* [8592e9ce6](https://github.com/argoproj/argo-workflows/commit/8592e9ce6e4de64e55c23bfda460b0cad67e74f7) feat: enable gcflags (compiler flags) to be passed into 'go build' (#8896)
* [7a626aa6a](https://github.com/argoproj/argo-workflows/commit/7a626aa6a1368da59c322f1d768e691b0ee4d7e4) feat: add Artifact.Deleted (#8893)
* [f2c748ac4](https://github.com/argoproj/argo-workflows/commit/f2c748ac44ed41b1d672e6c45a34090992b979d7) feat: Artifact GC Finalizer needs to be added if any Output Artifacts have a strategy (#8856)
* [093a6fe7e](https://github.com/argoproj/argo-workflows/commit/093a6fe7e1b1926f5feaff07a66edb9ff036f866) Add Orchest to ecosystem (#8884)
* [2b5ae622b](https://github.com/argoproj/argo-workflows/commit/2b5ae622bc257a4dafb4fab961e8142accaa484d) Removed Security Nudge and all its invocations (#8838)
* [f0447918d](https://github.com/argoproj/argo-workflows/commit/f0447918d6826b21a8e0cf0d0d218113e69059a8) chore(deps): bump github.com/spf13/viper from 1.11.0 to 1.12.0 (#8874)
* [8b7bdb713](https://github.com/argoproj/argo-workflows/commit/8b7bdb7139e8aa152e95ad3fe6815e7a801afcbb) chore(deps): bump github.com/minio/minio-go/v7 from 7.0.26 to 7.0.27 (#8875)
* [282a72295](https://github.com/argoproj/argo-workflows/commit/282a722950b113008b4efb258309cc4066f925a0) add pismo.io to argo users (#8871)
* [1a517e6f5](https://github.com/argoproj/argo-workflows/commit/1a517e6f5b801feae9416acf824c83ff65dea65c) chore(deps): bump superagent from 3.8.3 to 7.1.3 in /ui (#8851)
* [67dab5d85](https://github.com/argoproj/argo-workflows/commit/67dab5d854a4b1be693571765eae3857559851c6) chore(deps): bump cron-parser from 2.18.0 to 4.4.0 in /ui (#8844)
* [f676ac59a](https://github.com/argoproj/argo-workflows/commit/f676ac59a0794791dc5bdfd74acd9764110f2d2a) chore(deps): bump google.golang.org/api from 0.80.0 to 0.81.0 (#8841)
* [d324faaf8](https://github.com/argoproj/argo-workflows/commit/d324faaf885d32e8666a70e1f20bae7e71db386e) chore(deps): bump github.com/aliyun/aliyun-oss-go-sdk from 2.2.2+incompatible to 2.2.4+incompatible (#8842)
* [cc9d14cf0](https://github.com/argoproj/argo-workflows/commit/cc9d14cf0d60812e177ebb447181df933199b722) feat: Use Pod Names v2 by default (#8748)
* [bc4a80a8d](https://github.com/argoproj/argo-workflows/commit/bc4a80a8d63f869a7a607861374e0c206873f250) feat: remove size limit of 128kb for workflow templates. Fixes #8789 (#8796)
* [d61bea949](https://github.com/argoproj/argo-workflows/commit/d61bea94947526e7ca886891152c565cc15abded) chore(deps): bump js-yaml and @types/js-yaml in /ui (#8823)
* [14ac0392c](https://github.com/argoproj/argo-workflows/commit/14ac0392ce79bddbb9fc44c86fcf315ea1746235) chore(deps): bump cloud.google.com/go/storage from 1.22.0 to 1.22.1 (#8816)
* [ac92a49d0](https://github.com/argoproj/argo-workflows/commit/ac92a49d0f253111bd14bd72699ca3ad8cbeee1d) chore(deps): bump google.golang.org/api from 0.79.0 to 0.80.0 (#8815)
* [bc0100346](https://github.com/argoproj/argo-workflows/commit/bc01003468186ddcb93d1d32e9a49a75046827e7) fix: Change to distroless. Fixes #8805 (#8806)
* [fbb8246cd](https://github.com/argoproj/argo-workflows/commit/fbb8246cdc44d218f70f0de677be0f4dfd0780cf) fix: set NODE_OPTIONS to no-experimental-fetch to prevent yarn start error (#8802)
* [39fbdb2a5](https://github.com/argoproj/argo-workflows/commit/39fbdb2a551482c5ae2860fd266695c0113cb7b7) fix: fix a command in the quick-start page (#8782)
* [961f731b7](https://github.com/argoproj/argo-workflows/commit/961f731b7e9cb60490dd763a394893154c0b3c60) fix: Omitted task result should also be valid (#8776)
* [b07a57694](https://github.com/argoproj/argo-workflows/commit/b07a576945e87915e529d718101319d2f83cd98a) chore(deps): bump react-monaco-editor from 0.47.0 to 0.48.0 in /ui (#8770)
* [6b11707f5](https://github.com/argoproj/argo-workflows/commit/6b11707f50301a125eb8349193dd0be8659a4cdf) chore(deps): bump github.com/coreos/go-oidc/v3 from 3.1.0 to 3.2.0 (#8765)
* [d23693166](https://github.com/argoproj/argo-workflows/commit/d236931667a60266f87fbc446064ceebaf582996) chore(deps): bump github.com/prometheus/client_golang from 1.12.1 to 1.12.2 (#8763)
* [f6d84640f](https://github.com/argoproj/argo-workflows/commit/f6d84640fda435e08cc6a961763669b7572d0e69) fix: Skip TestExitHookWithExpression() completely (#8761)
* [178bbbc31](https://github.com/argoproj/argo-workflows/commit/178bbbc31c594f9ded4b8a66b0beecbb16cfa949) fix: Temporarily fix CI build. Fixes #8757. (#8758)
* [6b9dc2674](https://github.com/argoproj/argo-workflows/commit/6b9dc2674f2092b2198efb0979e5d7e42efffc30) feat: Add WebHDFS support for HTTP artifacts. Fixes #7540 (#8468)
* [354dee866](https://github.com/argoproj/argo-workflows/commit/354dee86616014bcb77afd170685242a18efd07c) fix: Exit lifecycle hook should respect expression. Fixes #8742 (#8744)
* [aa366db34](https://github.com/argoproj/argo-workflows/commit/aa366db345d794f0d330336d51eb2a88f14ebbe6) fix: remove list and watch on secrets. Fixes #8534 (#8555)
* [342abcd6d](https://github.com/argoproj/argo-workflows/commit/342abcd6d72b4cda64b01f30fa406b2f7b86ac6d) fix: mkdocs uses 4space indent for nested list (#8740)
* [1f2417e30](https://github.com/argoproj/argo-workflows/commit/1f2417e30937399e96fd4dfcd3fcc2ed7333291a) feat: running locally through dev container (#8677)
* [515e0763a](https://github.com/argoproj/argo-workflows/commit/515e0763ad4b1bd9d2941fc5c141c52691fc3b12) fix: Simplify return logic in executeTmplLifeCycleHook (#8736)
* [b8f511309](https://github.com/argoproj/argo-workflows/commit/b8f511309adf6443445e6dbf55889538fd39eacc) fix: Template in Lifecycle hook should be optional (#8735)
* [c0cd1f855](https://github.com/argoproj/argo-workflows/commit/c0cd1f855a5ef89d0f7a0d49f8e11781735cfa86) feat: ui, Dependabot auto dependency update (#8706)
* [b3bf327a0](https://github.com/argoproj/argo-workflows/commit/b3bf327a021e4ab5cc329f83bdec8f533c87a4d6) fix: Fix the resursive example to call the coinflip template (#8696)
* [427c16072](https://github.com/argoproj/argo-workflows/commit/427c16072b6c9d677265c95f5fd84e6a37fcc848) feat: Increased default significant figures in formatDuration. Fixes #8650 (#8686)
* [7e2df8129](https://github.com/argoproj/argo-workflows/commit/7e2df81299f660089cf676f7622638156affedf5) chore(deps): bump google.golang.org/api from 0.78.0 to 0.79.0 (#8710)
* [9ddae875f](https://github.com/argoproj/argo-workflows/commit/9ddae875fdb49d3e852f935e3d8b52fae585bc5e) fix: Fixed podName in killing daemon pods. Fixes #8692 (#8708)
* [72d3f32e5](https://github.com/argoproj/argo-workflows/commit/72d3f32e5676207d1511c609b00d26df20a2607e) fix: update go-color path/version (#8707)
* [92b3ef27a](https://github.com/argoproj/argo-workflows/commit/92b3ef27af7a7e6b930045e95072a47c8745b1d3) fix: upgrade moment from 2.29.2 to 2.29.3 (#8679)
* [8d4ac38a1](https://github.com/argoproj/argo-workflows/commit/8d4ac38a158dc2b4708478f7e7db1f2dd488ffed) feat: ui, add node version constraint (#8678)
* [2cabddc9a](https://github.com/argoproj/argo-workflows/commit/2cabddc9a9241061d8b89cf671f1c548405f4cb0) chore(deps): bump github.com/minio/minio-go/v7 from 7.0.24 to 7.0.26 (#8673)
* [859ebe99f](https://github.com/argoproj/argo-workflows/commit/859ebe99f760c6fb30870993359274a92cec2fb9) fix: Terminate, rather than delete, deadlined pods. Fixes #8545 (#8620)
* [dd565208e](https://github.com/argoproj/argo-workflows/commit/dd565208e236bc56230e75bedcc5082d171e6155) fix(git): add auth to fetch (#8664)
* [70f70209d](https://github.com/argoproj/argo-workflows/commit/70f70209d693d3933177a7de2cb6e421b763656f) fix: Handle omitted nodes in DAG enhanced depends logic. Fixes #8654 (#8672)
* [3fdf30d9f](https://github.com/argoproj/argo-workflows/commit/3fdf30d9f9181d74d81ca3184b53bbe661ecb845) fix: Enhance artifact visualization. Fixes #8619 (#8655)
* [16fef4e54](https://github.com/argoproj/argo-workflows/commit/16fef4e5498fac88dc80d33d653c99fec641150d) fix: enable `ARGO_REMOVE_PVC_PROTECTION_FINALIZER` by default. Fixes #8592 (#8661)
* [e4d57c6d5](https://github.com/argoproj/argo-workflows/commit/e4d57c6d560e025a336415aa840d2457eeca79f4) feat: `argo cp` to download artifacts. Fixes #695 (#8582)
* [e6e0c9bb3](https://github.com/argoproj/argo-workflows/commit/e6e0c9bb3b923a6d977875cbbd2744b8bacfce15) chore(deps): bump docker/login-action from 1 to 2 (#8642)
* [05781101d](https://github.com/argoproj/argo-workflows/commit/05781101dc94701aabd1bdbc2d3be4aa383b49f2) chore(deps): bump docker/setup-buildx-action from 1 to 2 (#8641)
* [6a4957135](https://github.com/argoproj/argo-workflows/commit/6a495713593f11514500998f6f69ce8f2e463975) chore(deps): bump docker/setup-qemu-action from 1 to 2 (#8640)
* [02370b51d](https://github.com/argoproj/argo-workflows/commit/02370b51d59bdd60b07c6c938737ed997807e4f2) feat: Track UI event #8402 (#8460)
* [64a2b28a5](https://github.com/argoproj/argo-workflows/commit/64a2b28a5fb51b50fe0e0a30185a8c3400d10548) fix: close http body. Fixes #8622 (#8624)
* [68a2cee6a](https://github.com/argoproj/argo-workflows/commit/68a2cee6a3373214803db009c7a6290954107c37) chore(deps): bump google.golang.org/api from 0.77.0 to 0.78.0 (#8602)
* [ed351ff08](https://github.com/argoproj/argo-workflows/commit/ed351ff084c4524ff4b2a45b53e539f91f5d423a) fix: ArtifactGC moved from Template to Artifact. Fixes #8556. (#8581)
* [87470e1c2](https://github.com/argoproj/argo-workflows/commit/87470e1c2bf703a9110e97bb755614ce8757fdcc) fix: Added artifact Content-Security-Policy (#8585)
* [61b80c90f](https://github.com/argoproj/argo-workflows/commit/61b80c90fd93aebff26df73fcddffa75732d10ec) Fix panic on executor plugin eventhandler (#8588)
* [974031570](https://github.com/argoproj/argo-workflows/commit/97403157054cb779b2005991fbb65c583aa3644c) fix: Polish artifact visualisation. Fixes #7743 (#8552)
* [98dd898be](https://github.com/argoproj/argo-workflows/commit/98dd898bef67e8523a0bf2ed942241dcb69eabe7) fix: Correct CSP. Fixes #8560 (#8579)
* [3d892d9b4](https://github.com/argoproj/argo-workflows/commit/3d892d9b481c5eefeb309b462b3f166a31335bc4) feat: New endpoint capable of serving directory listing or raw file, from non-archived or archived workflow (#8548)
* [71e2073b6](https://github.com/argoproj/argo-workflows/commit/71e2073b66b3b30b1eda658e88b7f6fd89469a92) chore(deps): bump lodash-es from 4.17.20 to 4.17.21 in /ui (#8577)
* [abf3c7411](https://github.com/argoproj/argo-workflows/commit/abf3c7411921dd422804c72b4f68dc2ab2731047) chore(deps): bump github.com/argoproj/pkg from 0.13.1 to 0.13.2 (#8571)
* [ffd5544c3](https://github.com/argoproj/argo-workflows/commit/ffd5544c31da026999b78197f55e6f4d2c8d7628) chore(deps): bump google.golang.org/api from 0.76.0 to 0.77.0 (#8572)
* [dc8fef3e5](https://github.com/argoproj/argo-workflows/commit/dc8fef3e5b1c0b833cc8568dbea23dbd1b310bdc) fix: Support memoization on plugin node. Fixes #8553 (#8554)
* [5b8638fcb](https://github.com/argoproj/argo-workflows/commit/5b8638fcb0f6ab0816f58f35a71f4f178ba9b7d9) fix: modified `SearchArtifact` to return `ArtifactSearchResults`. Fixes #8543 (#8557)
* [9398b0717](https://github.com/argoproj/argo-workflows/commit/9398b0717c14e15c78f6fe314ca9168d0104418d) feat: add more options to ArtifactSearchQuery. Fixes #8542. (#8549)
* [c781a5828](https://github.com/argoproj/argo-workflows/commit/c781a582821c4e08416eba9a3889eb2588596aa6) feat: Make artifacts discoverable in the DAG. Fixes #8494 (#8496)
* [d25b3fec4](https://github.com/argoproj/argo-workflows/commit/d25b3fec49377ea4be6a63d815a2b609636ef607) feat: Improve artifact server response codes. Fixes #8516 (#8524)
* [65b7437f7](https://github.com/argoproj/argo-workflows/commit/65b7437f7b26e19581650c0c2078f9dd8c89a73f) chore(deps): bump github.com/argoproj/pkg from 0.13.0 to 0.13.1 (#8537)
* [ecd91b1c4](https://github.com/argoproj/argo-workflows/commit/ecd91b1c4215a2ab8742f7c43eaade98a1d47eba) fix: added json tag to ArtifactGCStrategies (#8523)
* [f223bb8a3](https://github.com/argoproj/argo-workflows/commit/f223bb8a3c277e96a19e08f30f27ad70c0c425d3) fix: ArtifactGCOnWorkflowDeletion typo quick fix (#8519)
* [b4202b338](https://github.com/argoproj/argo-workflows/commit/b4202b338b5f97552fb730e4d07743c365d6f5ec) feat: Do not return cause of internal server error. Fixes #8514 (#8522)
* [d7bcaa756](https://github.com/argoproj/argo-workflows/commit/d7bcaa7569ac15d85eb293a72a1a98779275bd6e) feat: add finalizer for artifact GC (#8513)
* [c3ae56565](https://github.com/argoproj/argo-workflows/commit/c3ae56565bbe05c9809c5ad1192fcfc3ae717114) fix: Do not log container not found (#8509)
* [9a1345323](https://github.com/argoproj/argo-workflows/commit/9a1345323bb4727ba4fa769363b671213c02ded7) feat: Implement Workflow.SearchArtifacts(). Fixes #8473 (#8517)
* [30d9f8d77](https://github.com/argoproj/argo-workflows/commit/30d9f8d77caa69467f2b388b045fe9c3f8d05cb8) feat: Add correct CSP/XFO to served artifacts. Fixing #8492 (#8511)
* [d3f8db341](https://github.com/argoproj/argo-workflows/commit/d3f8db3417586b307401ecd5d172f9a1f97241db) feat: Save `containerSet` logs in artifact repository. Fixes #7897 (#8491)
* [6769ba720](https://github.com/argoproj/argo-workflows/commit/6769ba7209c1c8ffa6ecd5414d9694e743afe557) feat: add ArtifactGC to template spec (#8493)
* [19e763a3b](https://github.com/argoproj/argo-workflows/commit/19e763a3ba7ceaa890dc34310abeb4e7e4555641) chore(deps): bump google.golang.org/api from 0.75.0 to 0.76.0 (#8495)
* [6e9d42aed](https://github.com/argoproj/argo-workflows/commit/6e9d42aed1623e215a04c98cf1632f08f79a45cb) feat: add capability to choose params in suspend node.Fixes #8425 (#8472)
* [8685433e1](https://github.com/argoproj/argo-workflows/commit/8685433e1c183f1eb56add14c3e19c7b676314bb) feat: Added a delete function to the artifacts storage. Fixes #8470 (#8490)
* [9f5759b5b](https://github.com/argoproj/argo-workflows/commit/9f5759b5bd2a01d0f2930faa20ad5a769395eb99) feat: Enable git artifact clone of single branch (#8465)
* [7376e7cda](https://github.com/argoproj/argo-workflows/commit/7376e7cda4f72f0736fc128d15495acff71b987d) feat: Artifact streaming: enable artifacts to be streamed to users rather than loading the full file to disk first. Fixes #8396 (#8486)
* [06e9445ba](https://github.com/argoproj/argo-workflows/commit/06e9445ba71faba6f1132703762ec592a168ca9b) feat: add empty dir into wait container (#8390)
* [c61770622](https://github.com/argoproj/argo-workflows/commit/c6177062276cc39c3b21644ab1d6989cbcaf075c) fix: Pod `OOMKilled` should fail workflow. Fixes #8456 (#8478)
* [37a8a81df](https://github.com/argoproj/argo-workflows/commit/37a8a81df1d7ef3067596199f96974d31b200b88) feat: add ArtifactGC to workflow and template spec. Fixes #8471 (#8482)
* [ae803bba4](https://github.com/argoproj/argo-workflows/commit/ae803bba4f9b0c85f0d0471c22e44eb1c0f8f5f9) fix: Revert controller readiness changes. Fixes #8441 (#8454)
* [147ca4637](https://github.com/argoproj/argo-workflows/commit/147ca46376a4d86a09bde689d848396af6750b1e) fix: PodGC works with WorkflowTemplate. Fixes #8448 (#8452)
* [b7aeb6298](https://github.com/argoproj/argo-workflows/commit/b7aeb62982d91036edf5ba942eebeb4b22e30a3d) feat: Add darwin-arm64 binary build. Fixes #8450 (#8451)
* [8c0a957c3](https://github.com/argoproj/argo-workflows/commit/8c0a957c3ef0149f3f616a8baef2eb9a164436c1) fix: Fix bug in entrypoint lookup (#8453)
* [79508cc78](https://github.com/argoproj/argo-workflows/commit/79508cc78bd5b79762719c3b2fbe970981277e1f) chore(deps): bump google.golang.org/api from 0.74.0 to 0.75.0 (#8447)
* [24f9db628](https://github.com/argoproj/argo-workflows/commit/24f9db628090e9dfdfc7d657af80d96c176a47fd) chore(deps): bump github.com/argoproj/pkg from 0.11.0 to 0.12.0 (#8439)
* [72bb11305](https://github.com/argoproj/argo-workflows/commit/72bb1130543a3cc81347fe4fcf3257d8b35cd478) chore(deps): bump github.com/argoproj-labs/argo-dataflow (#8440)
* [230c82652](https://github.com/argoproj/argo-workflows/commit/230c8265246d50a095cc3a697fcd437174731aa8) feat: added support for http as option for artifact upload. Fixes #785 (#8414)
* [4f067ab4b](https://github.com/argoproj/argo-workflows/commit/4f067ab4bcb9ae570b9af11b2abd64d592e1fbbc) chore(deps): bump github.com/prometheus/common from 0.33.0 to 0.34.0 (#8427)
* [a2fd0031e](https://github.com/argoproj/argo-workflows/commit/a2fd0031ef13b63fd65520c615043e2aff89dde8) chore(deps): bump github.com/tidwall/gjson from 1.14.0 to 1.14.1 (#8426)
* [3d1ea426a](https://github.com/argoproj/argo-workflows/commit/3d1ea426a28c65c206752e957bb68a57ee8ed32e) fix: Remove binaries from Windows image. Fixes #8417 (#8420)
* [e71fdee07](https://github.com/argoproj/argo-workflows/commit/e71fdee07b8ccd7905752808bffb2283e170077a) Revert "feat: added support for http as an option for artifact upload. Fixes #785 (#8405)"
* [5845efbb9](https://github.com/argoproj/argo-workflows/commit/5845efbb94da8acfb218787846ea10c37fb2eebb) feat: Log result of HTTP requests & artifacts load/saves. Closes #8257 (#8394)
* [d22be825c](https://github.com/argoproj/argo-workflows/commit/d22be825cfb901f1ce59ba3744488cb8e144233b) feat: added support for http as an option for artifact upload. Fixes #785 (#8405)
* [4471b59a5](https://github.com/argoproj/argo-workflows/commit/4471b59a52873ca66d6834a06519407c858f5906) fix: open minio dashboard on different port in quick-start (#8407)
* [f467cc555](https://github.com/argoproj/argo-workflows/commit/f467cc5558bd22330eebfbc352ad4a7607f9fa4c) fix: Daemon step updated 'pod delete' while pod is running (#8399)
* [a648ccdcf](https://github.com/argoproj/argo-workflows/commit/a648ccdcfa3bb4cd5f5684faf921ab9fdab761de) fix: prevent backoff when retryStrategy.limit has been reached. Fixes #7588 (#8090)
* [136ebbc45](https://github.com/argoproj/argo-workflows/commit/136ebbc45b7cba346d7ba72f278624647a6b5a1c) chore(deps): bump github.com/minio/minio-go/v7 from 7.0.23 to 7.0.24 (#8397)
* [73ea7c72c](https://github.com/argoproj/argo-workflows/commit/73ea7c72c99a073dbe3ec0a420e112945916fb94) feat!: Add entrypoint lookup. Fixes #8344 (#8345)
* [283f6b58f](https://github.com/argoproj/argo-workflows/commit/283f6b58f979db1747ca23753d0562a440f95908) fix: Add readiness check to controller. Fixes #8283 (#8285)
* [75b533b61](https://github.com/argoproj/argo-workflows/commit/75b533b61eebd00044f2682540f5de15d6be8fbb) chore(deps): bump github.com/spf13/viper from 1.10.1 to 1.11.0 (#8392)
* [b09b9bdfb](https://github.com/argoproj/argo-workflows/commit/b09b9bdfb132c3967b81718bbc3c6e37fb2a3a42) fix: Absolute submodules in git artifacts. Fixes #8377 (#8381)
* [d47081fb4](https://github.com/argoproj/argo-workflows/commit/d47081fb4664d3a26e802a5c3c36798108388f2f) fix: upgrade react-moment from 1.0.0 to 1.1.1 (#8389)
* [010e359e4](https://github.com/argoproj/argo-workflows/commit/010e359e4c29b1af5653c46112ad53ac9b2679be) fix: upgrade react-datepicker from 2.14.1 to 2.16.0 (#8388)
* [0c9d88b44](https://github.com/argoproj/argo-workflows/commit/0c9d88b4429ff59c656e7b78b2160a55b49976ce) fix: upgrade prop-types from 15.7.2 to 15.8.1 (#8387)
* [54fa39c89](https://github.com/argoproj/argo-workflows/commit/54fa39c897d9883cec841450808102d71bd46fa8) fix: Back-off UI retries. Fixes #5697 (#8333)
* [637d14c88](https://github.com/argoproj/argo-workflows/commit/637d14c88f7d12c1c0355d62c2d1d4b03c4934e1) fix: replace `podName` with `nodeId` in `_.primary.swagger.json` (#8385)
* [95323f87d](https://github.com/argoproj/argo-workflows/commit/95323f87d42c9cf878563bfcb11460171906684b) fix: removed error from artifact server 401 response. Fixes #8382 (#8383)
* [2d91646aa](https://github.com/argoproj/argo-workflows/commit/2d91646aafede0e5671b07b2ac6eb27a057455b1) fix: upgrade js-yaml from 3.13.1 to 3.14.1 (#8374)
* [54eaed060](https://github.com/argoproj/argo-workflows/commit/54eaed0604393106b4dde3e7d7e6ccb41a42de6b) fix: upgrade cron-parser from 2.16.3 to 2.18.0 (#8373)
* [e97b0e66b](https://github.com/argoproj/argo-workflows/commit/e97b0e66b89f131fe6a12f24c26efbb73e16ef2e) fix: Updating complated node status
* [627597b56](https://github.com/argoproj/argo-workflows/commit/627597b5616f4d22e88b89a6d7017a67b6a4143d) fix: Add auth for SDKs. Fixes #8230 (#8367)
* [55ecfeb7b](https://github.com/argoproj/argo-workflows/commit/55ecfeb7b0e300a5d5cc6027c9212365cdaf4a2b) chore(deps): bump github.com/go-openapi/jsonreference (#8363)
* [e9de085d6](https://github.com/argoproj/argo-workflows/commit/e9de085d65a94d4189a54566d99c7177c1a7d735) fix: Erratum in docs. Fixes #8342 (#8359)
* [a3d1d07e1](https://github.com/argoproj/argo-workflows/commit/a3d1d07e1cbd19039771c11aa202bd8fd68198e7) fix: upgrade react-chartjs-2 from 2.10.0 to 2.11.2 (#8357)
* [b199cb947](https://github.com/argoproj/argo-workflows/commit/b199cb9474f7b1a3303a12858a2545aa85484d28) fix: upgrade history from 4.7.2 to 4.10.1 (#8356)
* [e40521556](https://github.com/argoproj/argo-workflows/commit/e4052155679a43cf083daf0c1b3fd5d45a5fbe24) fix: upgrade multiple dependencies with Snyk (#8355)
* [8c893bd13](https://github.com/argoproj/argo-workflows/commit/8c893bd13998b7dee09d0dd0c7a292b22509ca20) fix: upgrade com.google.code.gson:gson from 2.8.9 to 2.9.0 (#8354)
* [ee3765643](https://github.com/argoproj/argo-workflows/commit/ee3765643632fa6d8dbfb528a395cbb28608e2e8) feat: add message column to `kubectl get wf` and `argo list`. Fixes #8307 (#8353)
* [ae3881525](https://github.com/argoproj/argo-workflows/commit/ae3881525ce19a029a4798ff294e1b0c982e3268) fix: examples/README.md: overriten => overridden (#8351)
* [242d53596](https://github.com/argoproj/argo-workflows/commit/242d53596a5cf23b4470c2294204030ce11b01c4) fix: Fix response type for artifact service OpenAPI and SDKs. Fixes #7781 (#8332)
* [ab21eed52](https://github.com/argoproj/argo-workflows/commit/ab21eed527d15fa2c10272f740bff7c7963891c7) fix: upgrade io.swagger:swagger-annotations from 1.6.2 to 1.6.5 (#8335)
* [f708528fb](https://github.com/argoproj/argo-workflows/commit/f708528fbdfb9adecd8a66df866820eaab9a69ea) fix: upgrade react-monaco-editor from 0.36.0 to 0.47.0 (#8339)
* [3c35bd2f5](https://github.com/argoproj/argo-workflows/commit/3c35bd2f55dfdf641882cb5f9085b0b14f6d4d93) fix: upgrade cronstrue from 1.109.0 to 1.125.0 (#8338)
* [7ee17ddb7](https://github.com/argoproj/argo-workflows/commit/7ee17ddb7804e3f2beae87a8f532b1c0e6d1e520) fix: upgrade com.squareup.okhttp3:logging-interceptor from 4.9.1 to 4.9.3 (#8336)
* [68229e37e](https://github.com/argoproj/argo-workflows/commit/68229e37e295e3861cb7f6621ee3b9c7aabf8d67) added new-line to USERS.md (#8340)
* [94472c0ba](https://github.com/argoproj/argo-workflows/commit/94472c0bad4ed92ac06efb8c28563eba7b5bd1ab) chore(deps): bump cloud.google.com/go/storage from 1.20.0 to 1.22.0 (#8341)
* [aa9ff17d5](https://github.com/argoproj/argo-workflows/commit/aa9ff17d5feaa79aa26d9dc9cf9f67533f886b1c) fix: Remove path traversal CWE-23 (#8331)
* [14a9a1dc5](https://github.com/argoproj/argo-workflows/commit/14a9a1dc57f0d83231a19e76095ebdd4711f2594) fix: ui/package.json & ui/yarn.lock to reduce vulnerabilities (#8328)
* [58052c2b7](https://github.com/argoproj/argo-workflows/commit/58052c2b7b72daa928f8d427055be01cf896ff3e) fix: sdks/java/pom.xml to reduce vulnerabilities (#8327)
* [153540fdd](https://github.com/argoproj/argo-workflows/commit/153540fdd0e3b6f00050550abed67cae16299cbe) feat: Remove binaries from argoexec image. Fixes #7486 (#8292)
* [af8077423](https://github.com/argoproj/argo-workflows/commit/af807742343cb1a76926f6a1251466b9af988a47) feat: Always Show Workflow Parameters (#7809)
* [62e0a8ce4](https://github.com/argoproj/argo-workflows/commit/62e0a8ce4e74d2e19f3a9c0fb5e52bd58a6b944b) feat: Remove the PNS executor. Fixes #7804 (#8296)
* [0cdd2b40a](https://github.com/argoproj/argo-workflows/commit/0cdd2b40a8ee2d31476f8078eaedaa16c6827a76) fix: update docker version to address CVE-2022-24921 (#8312)
* [9c901456a](https://github.com/argoproj/argo-workflows/commit/9c901456a44501f11afc2bb1e856f0d0828fd13f) fix: Default value is ignored when loading params from configmap. Fixes #8262 (#8271)
* [9ab0e959a](https://github.com/argoproj/argo-workflows/commit/9ab0e959ac497433bcee2bb9c8d5710f87f1e3ea) fix: reduce number of workflows displayed in UI by default. Fixes #8297 (#8303)
* [13bc01362](https://github.com/argoproj/argo-workflows/commit/13bc013622c3b681bbd3c334dce0eea6870fcfde) fix: fix: git artifact will be checked out even if local file matches name of tracking branch (#8287)
* [65dc0882c](https://github.com/argoproj/argo-workflows/commit/65dc0882c9bb4496f1c4b2e0deb730e775724c82) feat: Fail on invalid config. (#8295)
* [5ac0e314d](https://github.com/argoproj/argo-workflows/commit/5ac0e314da80667e8b3b355c55cf9e1ab9b57b34) fix: `taskresults` owned by pod rather than workflow. (#8284)
* [996655f4f](https://github.com/argoproj/argo-workflows/commit/996655f4f3f03a30bcb82a1bb03f222fd100b8e0) fix: Snyk security recommendations (Golang). Fixes #8288
* [221d99827](https://github.com/argoproj/argo-workflows/commit/221d9982713ca30c060955bb35b48af3143c3754) fix: Snyk security recommendations (Node). Fixes #8288
* [b55dead05](https://github.com/argoproj/argo-workflows/commit/b55dead055139d1de33c464beed2b5ef596f5c8e) Revert "build: Enable governance bot. Fixes #8256 (#8259)" (#8294)
* [e50ec699c](https://github.com/argoproj/argo-workflows/commit/e50ec699cb33a7b84b0cb3c5b99396fe5365facd) chore(deps): bump google.golang.org/api from 0.73.0 to 0.74.0 (#8281)
* [954a3ee7e](https://github.com/argoproj/argo-workflows/commit/954a3ee7e7cc4f02074c07f7add971ca2be3291e) fix: install.yaml missing crb subject ns (#8280)
* [a3c326fdf](https://github.com/argoproj/argo-workflows/commit/a3c326fdf0d2133d5e78ef71854499f576e7e530) Remove hardcoded namespace in kustomize file #8250 (#8266)
* [b198b334d](https://github.com/argoproj/argo-workflows/commit/b198b334dfdb8e77d2ee51cd05b0716a29ab9169) fix: improve error message when the controller is set `templateReferencing: Secure` (#8277)
* [5598b8c7f](https://github.com/argoproj/argo-workflows/commit/5598b8c7fb5d17015e5c941e09953a74d8931436) feat: add resubmit and retry buttons for archived workflows. Fixes #7908 and #7911 (#8272)
* [6975607fa](https://github.com/argoproj/argo-workflows/commit/6975607fa33bf39e752b9cefcb8cb707a46bc6d4) chore(deps): bump github.com/prometheus/common from 0.32.1 to 0.33.0 (#8274)
* [78f01f2b9](https://github.com/argoproj/argo-workflows/commit/78f01f2b9f24a89db15a119885dfe8eb6420c70d) fix: patch workflow status to workflow (#8265)
* [f48998c07](https://github.com/argoproj/argo-workflows/commit/f48998c070c248688d996e5c8a4fec7601f5ab53) feat: Add a link in the UI for WorkflowTemplate. Fixes #4760 (#8208)
* [f02d4b72a](https://github.com/argoproj/argo-workflows/commit/f02d4b72adea9fbd23880c70871f92d66dc183c7) chore(deps): bump github.com/argoproj-labs/argo-dataflow (#8264)
* [f00ec49d6](https://github.com/argoproj/argo-workflows/commit/f00ec49d695bdad108000abcdfd0f82f6af9ca6c) feat!: Refactor/simplify configuration code (#8235)
* [c1f72b662](https://github.com/argoproj/argo-workflows/commit/c1f72b66282012e712e28a715c08dddb1a556c16) feat: add archive retry command to argo CLI. Fixes #7907 (#8229)
* [7a07805b1](https://github.com/argoproj/argo-workflows/commit/7a07805b183d598847bb9323f1009d7e8bbc1ac6) fix: Update argo-server manifests to have read-only root file-system (#8210)
* [0d4b4dc34](https://github.com/argoproj/argo-workflows/commit/0d4b4dc34127a27f7ca6e5c41197f3aaacc79cb8) fix: Panic in Workflow Retry (#8243)
* [61f0decd8](https://github.com/argoproj/argo-workflows/commit/61f0decd873a6a422c3a7159d6023170637338ff) fix: Hook with wftemplateRef (#8242)
* [e232340cc](https://github.com/argoproj/argo-workflows/commit/e232340cc5191c5904afe87f03c80545bb10e430) fix: grep pattern (#8238)
* [1d373c41a](https://github.com/argoproj/argo-workflows/commit/1d373c41afbebcf8de55114582693bcbdc59b342) fix: submodule cloning via git. Fixes #7469 (#8225)
* [6ee1b03f9](https://github.com/argoproj/argo-workflows/commit/6ee1b03f9e83c1e129b45a6bc9292a99add6b36e) fix: do not panic when termination-log is not writeable (#8221)
* [cae38894f](https://github.com/argoproj/argo-workflows/commit/cae38894f96b0d33cde54ef9cdee3cda53692a8d) chore(deps): bump github.com/aliyun/aliyun-oss-go-sdk (#8232)
* [e0e45503e](https://github.com/argoproj/argo-workflows/commit/e0e45503e6704b27e3e9ef0ff4a98169f3b072fa) chore(deps): bump peter-evans/create-pull-request from 3 to 4 (#8216)
* [8c77e89fc](https://github.com/argoproj/argo-workflows/commit/8c77e89fc185ff640e1073692dfc7c043037440a) feat: add archive resubmit command to argo CLI. Fixes #7910 (#8166)
* [d8aa46731](https://github.com/argoproj/argo-workflows/commit/d8aa46731c74730ccca1a40187109a63a675618b) fix: Support `--parameters-file` where ARGO_SERVER specified. Fixes #8160 (#8213)
* [d33d391a4](https://github.com/argoproj/argo-workflows/commit/d33d391a4c06c136b6a0964a51c75850323684e6) feat: Add support to auto-mount service account tokens for plugins. (#8176)
* [8a1fbb86e](https://github.com/argoproj/argo-workflows/commit/8a1fbb86e7c83bf14990805166d04d5cb4479ea3) fix: removed deprecated k8sapi executor. Fixes #7802 (#8205)
* [12cd8bcaa](https://github.com/argoproj/argo-workflows/commit/12cd8bcaa75381b5a9fa65aff03ac13aec706375) fix:  requeue not delete the considererd Task flag (#8194)
* [e2b288318](https://github.com/argoproj/argo-workflows/commit/e2b288318b15fa3e3cdc38c3dc7e66774920be8d) fix: Use `latest` image tag when version is `untagged`. Fixes #8188 (#8191)
* [6d6d23d81](https://github.com/argoproj/argo-workflows/commit/6d6d23d8110165331d924e97b01d5e26214c72db) fix: task worker requeue wrong task. Fixes #8139 (#8186)
* [41fd07aa4](https://github.com/argoproj/argo-workflows/commit/41fd07aa4f8462d70ad3c2c0481d5e09ae97b612) fix: Update `workflowtaskresult` code have own reconciliation loop. (#8135)
* [051c7b8d2](https://github.com/argoproj/argo-workflows/commit/051c7b8d2baf50b55e8076a1e09e7340551c04c1) fix: pkg/errors is no longer maintained (#7440)
* [fbb43b242](https://github.com/argoproj/argo-workflows/commit/fbb43b2429e45346221a119583aac11df4b5f880) fix: workflow.duration' is not available as a real time metric (#8181)
* [0e707cdf6](https://github.com/argoproj/argo-workflows/commit/0e707cdf69f891c7c7483e2244f5ea930d31b1c5) fix: Authentication for plugins. Fixes #8144 (#8147)
* [d4b1afe6f](https://github.com/argoproj/argo-workflows/commit/d4b1afe6f68afc3061a924186fa09556290ec3e1) feat: add retry API for archived workflows. Fixes #7906 (#7988)
* [e7008eada](https://github.com/argoproj/argo-workflows/commit/e7008eada7a885d80952b5184562a29508323c2a) fix: Correctly order emissary combined output. Fixes #8159 (#8175)
* [9101c4939](https://github.com/argoproj/argo-workflows/commit/9101c49396fe95d62ef3040cd4d330fde9f35554) fix: Add instance ID to `workflowtaskresults` (#8150)
* [2b5e4a1d2](https://github.com/argoproj/argo-workflows/commit/2b5e4a1d2df7877d9b7b7fbedd7136a125a39c8d) feat: Use pinned executor version. (#8165)
* [715f6ced6](https://github.com/argoproj/argo-workflows/commit/715f6ced6f42c0b7b5994bf8d16c561f48025fe8) fix: add /etc/mime.types mapping table (#8171)
* [6d6e08aa8](https://github.com/argoproj/argo-workflows/commit/6d6e08aa826c406a912387ac438ec20428c7623d) fix: Limit workflows to 128KB and return a friendly error message (#8169)
* [057c3346f](https://github.com/argoproj/argo-workflows/commit/057c3346f9f792cf10888320c4297b09f3c11e2e) feat: add TLS config option to HTTP template. Fixes #7390 (#7929)
* [013fa2578](https://github.com/argoproj/argo-workflows/commit/013fa2578bc5cace4de754daef04448b30faae32) chore(deps): bump github.com/stretchr/testify from 1.7.0 to 1.7.1 (#8163)
* [ad341c4af](https://github.com/argoproj/argo-workflows/commit/ad341c4af1645c191a5736d91d78a19acc7b2fa7) chore(deps): bump google.golang.org/api from 0.72.0 to 0.73.0 (#8162)
* [5efc9fc99](https://github.com/argoproj/argo-workflows/commit/5efc9fc995ac898672a575b514f8bfc83b220c4c) feat: add mysql options (#8157)
* [cda5737c3](https://github.com/argoproj/argo-workflows/commit/cda5737c37e3ab7c381869d7d820de71285f55a5) chore(deps): bump google.golang.org/api from 0.71.0 to 0.72.0 (#8156)
* [be2dd19a0](https://github.com/argoproj/argo-workflows/commit/be2dd19a0718577348823f1f68b82dbef8d95959) Update USERS.md (#8132)
* [af26ff7ed](https://github.com/argoproj/argo-workflows/commit/af26ff7ed54d4fe508edac34f82fe155f2d54a9d) fix: Remove need for `get pods` from Emissary (#8133)
* [537dd3be6](https://github.com/argoproj/argo-workflows/commit/537dd3be6bf93be37e06d768d9a610038eafb361) feat: Change pod clean-up to use informer. (#8136)
* [1d71fb3c4](https://github.com/argoproj/argo-workflows/commit/1d71fb3c4ebdb2891435ed12257743331ff34436) chore(deps): bump github.com/spf13/cobra from 1.3.0 to 1.4.0 (#8131)
* [972a4e989](https://github.com/argoproj/argo-workflows/commit/972a4e98987296a844a28dce31162d59732e6532) fix(plugins): UX improvements (#8122)
* [437b37647](https://github.com/argoproj/argo-workflows/commit/437b3764783b48a304034cc4291472c6e490689b) feat: add resubmit API for archived workflows. Fixes #7909 (#8079)
* [707cf8321](https://github.com/argoproj/argo-workflows/commit/707cf8321ccaf98b4596695fdbfdb04faf9a9487) update kustomize/kubectl installation (#8095)
* [48348247f](https://github.com/argoproj/argo-workflows/commit/48348247f0a0fd949871a9f982d7ee70c39509a1) chore(deps): bump google.golang.org/api from 0.70.0 to 0.71.0 (#8108)
* [765333dc9](https://github.com/argoproj/argo-workflows/commit/765333dc95575608fdf87328c7548c5e349b557d) fix(executor): Retry kubectl on internal transient error (#8092)
* [4d4890454](https://github.com/argoproj/argo-workflows/commit/4d4890454e454acbc86cef039bb6905c63f79e73) fix: Fix the TestStopBehavior flackiness (#8096)
* [6855f4c51](https://github.com/argoproj/argo-workflows/commit/6855f4c51b5bd667599f072ae5ddde48967006f1) fix: pod deleted due to delayed cleanup. Fixes #8022 (#8061)

<details><summary><h3>Contributors</h3></summary>

* Aatman
* Adam Eri
* Alex Collins
* Aman Verma
* Amil Khan
* BOOK
* Basanth Jenu H B
* Baschtie
* Brian Loss
* Caelan U
* Cash Williams
* Clemens Lange
* Dakota Lillie
* Dana Pieluszczak
* Daniel Helfand
* Deepyaman Datta
* Derek Wang
* Dillen Padhiar
* Doğukan
* Ezequiel Muns
* Felix Seidel
* Fernando Luís da Silva
* Gaurav Gupta
* Grzegorz Bielski
* Hao Xin
* Hidehito Yabuuchi
* Iain Lane
* Ian McGraw
* Isitha Subasinghe
* Iván Sánchez
* Jake Ralston
* JasonZhu
* Jesse Antoszyk
* Jessie Teng
* Jobim Robinsantos
* John Lin
* Jose
* Juan Luis Cano Rodríguez
* Julie Vogelman
* Kesavan
* Kevin George
* Logan Kilpatrick
* LoricAndre
* Manik Sidana
* Marc Abramowitz
* Mark Shields
* Markus Lippert
* Michael Goodness
* Michael Weibel
* Mike Tougeron
* Ming Yu Shi
* Miroslav Boussarov
* Mitsuo Heijo
* Noam Gal
* Peixuan Ding
* Philippe Richard
* Pieter De Clercq
* Qalifah
* Rajshekar Reddy
* Rick
* Rohan Kumar
* Sanjay Tiwari
* Saravanan Balasubramanian
* Shay Nehmad
* Shubham Nazare
* Snyk bot
* Song Juchao
* Soumya Ghosh Dastidar
* Stephanie Palis
* Suraj Narwade
* Surya Oruganti
* Swarnim Pratap Singh
* Takumi Sue
* Tianchu Zhao
* Timo Pagel
* Tristan Colgate-McFarlane
* Tuan
* Vedant Thapa
* Vignesh
* William Van Hevelingen
* Wu Jayway
* Yuan Tang
* alexdittmann
* dependabot[bot]
* github-actions[bot]
* hadesy
* ibuder
* kasteph
* kennytrytek
* lijie
* mihirpandya-greenops
* momom-i
* nikstur
* shirou
* smile-luobin
* tatsuya-ogawa
* tculp
* tim-sendible
* ybyang
* İnanç Dokurel

</details>

## v3.3.10 (2022-11-29)

Full Changelog: [v3.3.9...v3.3.10](https://github.com/argoproj/argo-workflows/compare/v3.3.9...v3.3.10)

### Selected Changes

* [b19870d73](https://github.com/argoproj/argo-workflows/commit/b19870d737a14b21d86f6267642a63dd14e5acd5) fix(operator): Workflow stuck at running when init container failed. Fixes #10045 (#10047)
* [fd31eb811](https://github.com/argoproj/argo-workflows/commit/fd31eb811160c62f16b5aef002bf232235e0d2c6) fix: Upgrade kubectl to v1.24.8 to fix vulnerabilities (#10008)
* [859bcb124](https://github.com/argoproj/argo-workflows/commit/859bcb1243728482d796a983776d84bd53b170ca) fix: assume plugins may produce outputs.result and outputs.exitCode (Fixes #9966) (#9967)
* [33bba51a6](https://github.com/argoproj/argo-workflows/commit/33bba51a61fc2dfcf81efb09629dcbeb8dddb3a1) fix: cleaned key paths in gcs driver. Fixes #9958 (#9959)

<details><summary><h3>Contributors</h3></summary>

* Isitha Subasinghe
* Michael Crenshaw
* Yuan Tang

</details>

## v3.3.9 (2022-08-09)

Full Changelog: [v3.3.8...v3.3.9](https://github.com/argoproj/argo-workflows/compare/v3.3.8...v3.3.9)

### Selected Changes

* [5db53aa0c](https://github.com/argoproj/argo-workflows/commit/5db53aa0ca54e51ca69053e1d3272e37064559d7) Revert "fix: Correct kill command. Fixes #8687 (#8908)"
* [b7b37d5aa](https://github.com/argoproj/argo-workflows/commit/b7b37d5aa2229c09365735fab165b4876c30aa4a) fix: Skip TestRunAsNonRootWithOutputParams
* [e4dca01f1](https://github.com/argoproj/argo-workflows/commit/e4dca01f1a76cefb7cae944ba0c4e54bc0aec427) fix: SignalsSuite test
* [151432f9b](https://github.com/argoproj/argo-workflows/commit/151432f9b754981959e149202d5f4b0617064595) fix: add containerRuntimeExecutor: emissary in ci
* [a3d6a58a7](https://github.com/argoproj/argo-workflows/commit/a3d6a58a71e1603077a4b39c4368d11847d500fb) feat: refactoring e2e test timeouts to support multiple environments. (#8925)
* [f9e2dd21c](https://github.com/argoproj/argo-workflows/commit/f9e2dd21cb09ac90b639be0f97f07da373240202) fix: lint
* [ef3fb421f](https://github.com/argoproj/argo-workflows/commit/ef3fb421f02f96195046ba327beca7b08753530b) fix: Correct kill command. Fixes #8687 (#8908)
* [e85c815a1](https://github.com/argoproj/argo-workflows/commit/e85c815a10fb59cb95cfdf6d2a171cea7c6aec47) fix: set NODE_OPTIONS to no-experimental-fetch to prevent yarn start error (#8802)
* [a19c94bb6](https://github.com/argoproj/argo-workflows/commit/a19c94bb6639540f309883ff0f41b14dd557324b) fix: Omitted task result should also be valid (#8776)
* [15f9d5227](https://github.com/argoproj/argo-workflows/commit/15f9d52270af4bca44553755d095d2dd8badfa14) fix: Fixed podName in killing daemon pods. Fixes #8692 (#8708)
* [6ec0ca088](https://github.com/argoproj/argo-workflows/commit/6ec0ca0883cf4e2222176ab413b3318017a30796) fix: open minio dashboard on different port in quick-start (#8407)
* [d874c1a87](https://github.com/argoproj/argo-workflows/commit/d874c1a87b65b300b2a4c93032bd2970d6f91d8f) fix: ui/package.json & ui/yarn.lock to reduce vulnerabilities (#8328)
* [481137c25](https://github.com/argoproj/argo-workflows/commit/481137c259b05c6a5b3c0e3adab1649c2b512364) fix: sdks/java/pom.xml to reduce vulnerabilities (#8327)
* [f54fb5c24](https://github.com/argoproj/argo-workflows/commit/f54fb5c24dd52a64da6d5aad5972a6554e386769) fix: grep pattern (#8238)
* [73334cae9](https://github.com/argoproj/argo-workflows/commit/73334cae9fbaef96b63889e16a3a2f78c725995e) fix: removed deprecated k8sapi executor. Fixes #7802 (#8205)
* [9c9efa67f](https://github.com/argoproj/argo-workflows/commit/9c9efa67f38620eeb08d1a9d2bb612bf14bf33de) fix: retryStrategy.Limit is now read properly for backoff strategy. Fixes #9170. (#9213)
* [69b5f1d79](https://github.com/argoproj/argo-workflows/commit/69b5f1d7945247a9e219b53f12fb8b3eec6e5e52) fix: Add missing Go module entries

<details><summary><h3>Contributors</h3></summary>

* Alex Collins
* Dillen Padhiar
* Grzegorz Bielski
* Ian McGraw
* Julie Vogelman
* Kesavan
* Rohan Kumar
* Saravanan Balasubramanian
* Snyk bot
* Takumi Sue
* Yuan Tang

</details>

## v3.3.8 (2022-06-23)

Full Changelog: [v3.3.7...v3.3.8](https://github.com/argoproj/argo-workflows/compare/v3.3.7...v3.3.8)

### Selected Changes

* [621b0d1a8](https://github.com/argoproj/argo-workflows/commit/621b0d1a8e09634666ebe403ee7b8fc29db1dc4e) fix: check for nil, and add logging to expose root cause of panic in Issue 8968 (#9010)
* [b7c218c0f](https://github.com/argoproj/argo-workflows/commit/b7c218c0f7b3ea0035dc44ccc9e8416f30429d16) feat: log workflow size before hydrating/dehydrating. Fixes #8976 (#8988)

<details><summary><h3>Contributors</h3></summary>

* Dillen Padhiar
* Julie Vogelman

</details>

## v3.3.7 (2022-06-20)

Full Changelog: [v3.3.6...v3.3.7](https://github.com/argoproj/argo-workflows/compare/v3.3.6...v3.3.7)

### Selected Changes

* [479763c04](https://github.com/argoproj/argo-workflows/commit/479763c04036db98cd1e9a7a4fc0cc932affb8bf) fix: Skip TestExitHookWithExpression() completely (#8761)
* [a1ba42140](https://github.com/argoproj/argo-workflows/commit/a1ba42140154e757b024fe29c61fc7043c741cee) fix: Template in Lifecycle hook should be optional (#8735)
* [f10d6238d](https://github.com/argoproj/argo-workflows/commit/f10d6238d83b410a461d1860d0bb3c7ae4d74383) fix: Simplify return logic in executeTmplLifeCycleHook (#8736)
* [f2ace043b](https://github.com/argoproj/argo-workflows/commit/f2ace043bb7d050e8d539a781486c9f932bca931) fix: Exit lifecycle hook should respect expression. Fixes #8742 (#8744)
* [8c0b43569](https://github.com/argoproj/argo-workflows/commit/8c0b43569bb3e9c9ace21afcdd89d2cec862939c) fix: long code blocks overflow in ui. Fixes #8916 (#8947)
* [1d26628b8](https://github.com/argoproj/argo-workflows/commit/1d26628b8bc5f5a4d90d7a31b6f8185f280a4538) fix: sync cluster Workflow Template Informer before it's used (#8961)
* [4d9f8f7c8](https://github.com/argoproj/argo-workflows/commit/4d9f8f7c832ff888c11a41dad7a755ef594552c7) fix: Workflow Duration metric shouldn't increase after workflow complete (#8989)
* [72e0c6f00](https://github.com/argoproj/argo-workflows/commit/72e0c6f006120f901f02ea3a6bf8b3e7f639eb48) fix: add nil check for retryStrategy.Limit in deadline check. Fixes #8990 (#8991)

<details><summary><h3>Contributors</h3></summary>

* Dakota Lillie
* Dillen Padhiar
* Julie Vogelman
* Saravanan Balasubramanian
* Yuan Tang

</details>

## v3.3.6 (2022-05-25)

Full Changelog: [v3.3.5...v3.3.6](https://github.com/argoproj/argo-workflows/compare/v3.3.5...v3.3.6)

### Selected Changes

* [2b428be80](https://github.com/argoproj/argo-workflows/commit/2b428be8001a9d5d232dbd52d7e902812107eb28) fix: Handle omitted nodes in DAG enhanced depends logic. Fixes #8654 (#8672)
* [7889af614](https://github.com/argoproj/argo-workflows/commit/7889af614c354f4716752942891cbca0a0889df0) fix: close http body. Fixes #8622 (#8624)
* [622c3d594](https://github.com/argoproj/argo-workflows/commit/622c3d59467a2d0449717ab866bd29bbd0469795) fix: Do not log container not found (#8509)
* [7091d8003](https://github.com/argoproj/argo-workflows/commit/7091d800360ad940ec605378324909823911d853) fix: pkg/errors is no longer maintained (#7440)
* [3f4c79fa5](https://github.com/argoproj/argo-workflows/commit/3f4c79fa5f54edcb50b6003178af85c70b5a8a1f) feat: remove size limit of 128kb for workflow templates. Fixes #8789 (#8796)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins
* Dillen Padhiar
* Stephanie Palis
* Yuan Tang
* lijie

</details>

## v3.3.5 (2022-05-03)

Full Changelog: [v3.3.4...v3.3.5](https://github.com/argoproj/argo-workflows/compare/v3.3.4...v3.3.5)

### Selected Changes

* [93cb050e3](https://github.com/argoproj/argo-workflows/commit/93cb050e3933638f0dbe2cdd69630e133b3ad52a) Revert "fix: Pod `OOMKilled` should fail workflow. Fixes #8456 (#8478)"
* [29f3ad844](https://github.com/argoproj/argo-workflows/commit/29f3ad8446ac5f07abda0f6844f3a31a7d50eb23) fix: Added artifact Content-Security-Policy (#8585)
* [a40d27cd7](https://github.com/argoproj/argo-workflows/commit/a40d27cd7535f6d36d5fb8d10cea0226b784fa65) fix: Support memoization on plugin node. Fixes #8553 (#8554)
* [f2b075c29](https://github.com/argoproj/argo-workflows/commit/f2b075c29ee97c95cfebb453b18c0ce5f16a5f04) fix: Pod `OOMKilled` should fail workflow. Fixes #8456 (#8478)
* [ba8c60022](https://github.com/argoproj/argo-workflows/commit/ba8c600224b7147d1832de1bea694fd376570ae9) fix: prevent backoff when retryStrategy.limit has been reached. Fixes #7588 (#8090)
* [c17f8c71d](https://github.com/argoproj/argo-workflows/commit/c17f8c71d40d4e34ef0a87dbc95eda005a57dc39) fix: update docker version to address CVE-2022-24921 (#8312)
* [9d0b7aa56](https://github.com/argoproj/argo-workflows/commit/9d0b7aa56cf065bf70c2cfb43f71ea9f92b5f964) fix: Default value is ignored when loading params from configmap. Fixes #8262 (#8271)
* [beab5b6ef](https://github.com/argoproj/argo-workflows/commit/beab5b6ef40a187e90ff23294bb1d9e2db9cb90a) fix: install.yaml missing crb subject ns (#8280)
* [b0d8be2ef](https://github.com/argoproj/argo-workflows/commit/b0d8be2ef3d3c1c96b15aeda572fcd1596fca9f1) fix:  requeue not delete the considererd Task flag (#8194)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins
* Cash Williams
* Rohan Kumar
* Soumya Ghosh Dastidar
* Wu Jayway
* Yuan Tang
* ybyang

</details>

## v3.3.4 (2022-04-29)

Full Changelog: [v3.3.3...v3.3.4](https://github.com/argoproj/argo-workflows/compare/v3.3.3...v3.3.4)

### Selected Changes

* [02fb874f5](https://github.com/argoproj/argo-workflows/commit/02fb874f5deb3fc3e16f033c6f60b10e03504d00) feat: add capability to choose params in suspend node.Fixes #8425 (#8472)
* [32b1b3a3d](https://github.com/argoproj/argo-workflows/commit/32b1b3a3d505dea1d42fdeb0104444ca4f5e5795) feat: Add support to auto-mount service account tokens for plugins. (#8176)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins
* Basanth Jenu H B

</details>

## v3.3.3 (2022-04-25)

Full Changelog: [v3.3.2...v3.3.3](https://github.com/argoproj/argo-workflows/compare/v3.3.2...v3.3.3)

### Selected Changes

* [9c08aedc8](https://github.com/argoproj/argo-workflows/commit/9c08aedc880026161d394207acbac0f64db29a53) fix: Revert controller readiness changes. Fixes #8441 (#8454)
* [9854dd3fc](https://github.com/argoproj/argo-workflows/commit/9854dd3fccccd34bf3e4f110412dbd063f3316c2) fix: PodGC works with WorkflowTemplate. Fixes #8448 (#8452)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins

</details>

## v3.3.2 (2022-04-20)

Full Changelog: [v3.3.1...v3.3.2](https://github.com/argoproj/argo-workflows/compare/v3.3.1...v3.3.2)

### Selected Changes

* [35492a170](https://github.com/argoproj/argo-workflows/commit/35492a1700a0f279694cac874b6d9c07a08265d1) fix: Remove binaries from Windows image. Fixes #8417 (#8420)
* [bfc3b6cad](https://github.com/argoproj/argo-workflows/commit/bfc3b6cad02c0a38141201d7f77e14e3f0e637a4) fix: Skip TestRunAsNonRootWithOutputParams
* [1c34f9801](https://github.com/argoproj/argo-workflows/commit/1c34f9801b502d1566064726145ce5d68124b213) fix: go.sum
* [be35b54b0](https://github.com/argoproj/argo-workflows/commit/be35b54b00e44339f8dcb63d0411bc80f8983764) fix: create cache lint
* [017a31518](https://github.com/argoproj/argo-workflows/commit/017a3151837ac05cca1b2425a8395d547d86ed09) fix: create cache lint
* [20d601b3d](https://github.com/argoproj/argo-workflows/commit/20d601b3dd2ebef102a1a610e4dbef6924f842ff) fix: create cache lint
* [d8f28586f](https://github.com/argoproj/argo-workflows/commit/d8f28586f82b1bdb9e43446bd1792b3b01b2928a) fix: empty push
* [f41d94e91](https://github.com/argoproj/argo-workflows/commit/f41d94e91648961dfdc6e8536768012569dcd28f) fix: codegen
* [ce195dd52](https://github.com/argoproj/argo-workflows/commit/ce195dd521e195df4edd96bcd27fd950f23ff611) fix: Add auth for SDKs. Fixes #8230 (#8367)
* [00c960619](https://github.com/argoproj/argo-workflows/commit/00c9606197c30c138714b27ca5624dd0272c662d) fix: unittest
* [a0148c1b3](https://github.com/argoproj/argo-workflows/commit/a0148c1b32fef820a0cde5a5fed1975abedb7f82) chore(deps): bump github.com/minio/minio-go/v7 from 7.0.23 to 7.0.24 (#8397)
* [5207d287b](https://github.com/argoproj/argo-workflows/commit/5207d287b5657d9049edd1b67c2b681a13c40420) fix: codegen
* [e68e06c34](https://github.com/argoproj/argo-workflows/commit/e68e06c3453453d70a76c08b1a6cb00635b2d941) fix: Daemon step updated 'pod delete' while pod is running (#8399)
* [b9f8b3587](https://github.com/argoproj/argo-workflows/commit/b9f8b3587345eda47edfaebb7bc18ea1193d430b) fix: Add readiness check to controller. Fixes #8283 (#8285)
* [ed26dc0a0](https://github.com/argoproj/argo-workflows/commit/ed26dc0a09bc38ac2366124621ea98918b95b34a) fix: Absolute submodules in git artifacts. Fixes #8377 (#8381)
* [6f77c0af0](https://github.com/argoproj/argo-workflows/commit/6f77c0af03545611dfef0222bcf5f5f76f30f4d4) fix: Back-off UI retries. Fixes #5697 (#8333)
* [8d5c2f2a3](https://github.com/argoproj/argo-workflows/commit/8d5c2f2a39033972e1f389029f5c08290aa19ccd) fix: replace `podName` with `nodeId` in `_.primary.swagger.json` (#8385)
* [a327edd5a](https://github.com/argoproj/argo-workflows/commit/a327edd5a5c5e7aff4c64131f1a9c3d9e5d9d3eb) fix: removed error from artifact server 401 response. Fixes #8382 (#8383)
* [502cf6d88](https://github.com/argoproj/argo-workflows/commit/502cf6d882ac51fd80950c2f25f90e491b3f13f6) fix: Updating complated node status
* [0a0956864](https://github.com/argoproj/argo-workflows/commit/0a09568648199fcc5a8997e4f5eed55c40bfa974) fix: Fix response type for artifact service OpenAPI and SDKs. Fixes #7781 (#8332)
* [a3bce2aaf](https://github.com/argoproj/argo-workflows/commit/a3bce2aaf94b07a73c3a7a4c9205872be7dc360c) fix: patch workflow status to workflow (#8265)
* [c5174fbee](https://github.com/argoproj/argo-workflows/commit/c5174fbeec69aa0ea4dbad8b239b7e46c76e5873) fix: Update argo-server manifests to have read-only root file-system (#8210)
* [ba795e656](https://github.com/argoproj/argo-workflows/commit/ba795e6562902d66adadd15554f791bc85b779a8) fix: Panic in Workflow Retry (#8243)
* [c95de6bb2](https://github.com/argoproj/argo-workflows/commit/c95de6bb25b8d7294f8f48490fccb2ba95d96f9b) fix: Hook with wftemplateRef (#8242)
* [187c21fa7](https://github.com/argoproj/argo-workflows/commit/187c21fa7b45d87c55dd71f247e439f6c9b776b3) fix: submodule cloning via git. Fixes #7469 (#8225)
* [289d44b9b](https://github.com/argoproj/argo-workflows/commit/289d44b9b0234baf24f1384a0b6743ca10bfb060) fix: do not panic when termination-log is not writeable (#8221)
* [c10ba38a8](https://github.com/argoproj/argo-workflows/commit/c10ba38a86eb2ba4e70812b172a02bea901073f1) fix: Support `--parameters-file` where ARGO_SERVER specified. Fixes #8160 (#8213)
* [239781109](https://github.com/argoproj/argo-workflows/commit/239781109e62e405a6596e88c706df21cf152a6e) fix: Use `latest` image tag when version is `untagged`. Fixes #8188 (#8191)
* [7d00fa9d9](https://github.com/argoproj/argo-workflows/commit/7d00fa9d94427e5b30bea3c3bd7fecd673b95870) fix: task worker requeue wrong task. Fixes #8139 (#8186)
* [ed6907f1c](https://github.com/argoproj/argo-workflows/commit/ed6907f1cafb1cd53a877c1bdebbf0497ab53278) fix: Authentication for plugins. Fixes #8144 (#8147)
* [5ff9bc9aa](https://github.com/argoproj/argo-workflows/commit/5ff9bc9aaba80db7833d513321bb6ae2d305f1f9) fix: Correctly order emissary combined output. Fixes #8159 (#8175)
* [918c27311](https://github.com/argoproj/argo-workflows/commit/918c273113ed14349c8df87d727a5b8070d301a1) fix: Add instance ID to `workflowtaskresults` (#8150)
* [af0cfab8f](https://github.com/argoproj/argo-workflows/commit/af0cfab8f3bd5b62ebe967381fed0bccbd7c7ada) fix: Update `workflowtaskresult` code have own reconciliation loop. (#8135)
* [3a425ec5a](https://github.com/argoproj/argo-workflows/commit/3a425ec5a1010e9b9ac2aac054095e5e9d240693) fix: Authentication for plugins. Fixes #8144 (#8147)
* [cdd1633e4](https://github.com/argoproj/argo-workflows/commit/cdd1633e428d8596467e7673d0d6d5c50ade41af) fix: Correctly order emissary combined output. Fixes #8159 (#8175)
* [22c203fc4](https://github.com/argoproj/argo-workflows/commit/22c203fc44a005e4207fff5b8ce7f4854ed0bf78) fix: Add instance ID to `workflowtaskresults` (#8150)
* [79a9a5b6f](https://github.com/argoproj/argo-workflows/commit/79a9a5b6fcca7953e740a5e171d3bc7f08953854) fix: improve error message when the controller is set `templateReferencing: Secure` (#8277)
* [7e880216a](https://github.com/argoproj/argo-workflows/commit/7e880216a1bf384d15d836877d170bbeea19814d) fix: `taskresults` owned by pod rather than workflow. (#8284)
* [347583132](https://github.com/argoproj/argo-workflows/commit/347583132916fd2f87b3885381fe86281ea3ec33) fix: fix: git artifact will be checked out even if local file matches name of tracking branch (#8287)
* [aa460b9ad](https://github.com/argoproj/argo-workflows/commit/aa460b9adc40ed4854dc373d0d755e6d36b633f8) fix: reduce number of workflows displayed in UI by default. Fixes #8297 (#8303)

<details><summary><h3>Contributors</h3></summary>

* Aatman
* Alex Collins
* Dillen Padhiar
* Markus Lippert
* Michael Weibel
* Rohan Kumar
* Saravanan Balasubramanian
* Takumi Sue
* Tristan Colgate-McFarlane
* Wu Jayway
* dependabot[bot]

</details>

## v3.3.1 (2022-03-18)

Full Changelog: [v3.3.0...v3.3.1](https://github.com/argoproj/argo-workflows/compare/v3.3.0...v3.3.1)

### Selected Changes

* [76ff748d4](https://github.com/argoproj/argo-workflows/commit/76ff748d41c67e1a38ace1352ca3bab8d7ec8a39) feat: add TLS config option to HTTP template. Fixes #7390 (#7929)
* [4c61c8df2](https://github.com/argoproj/argo-workflows/commit/4c61c8df2a3fcbe7abbc04dba34f59d270fe15f3) fix(executor): Retry kubectl on internal transient error (#8092)
* [47b78d4c4](https://github.com/argoproj/argo-workflows/commit/47b78d4c473c5e6e6301181bff298f32456288bd) fix(plugins): UX improvements (#8122)
* [ad7d9058e](https://github.com/argoproj/argo-workflows/commit/ad7d9058ed025481051c8545f26954f87463526f) fix: Authentication for plugins. Fixes #8144 (#8147)
* [5b14e15c2](https://github.com/argoproj/argo-workflows/commit/5b14e15c216995ca72fa5c7fc174913506fbdcd9) feat: add TLS config option to HTTP template. Fixes #7390 (#7929)
* [4e543f268](https://github.com/argoproj/argo-workflows/commit/4e543f268246afd2dcfc309f3d29d3c052ebeef4) fix(plugins): UX improvements (#8122)
* [845a244c7](https://github.com/argoproj/argo-workflows/commit/845a244c71129aa843d06a26d89aeec6da6c57d7) fix(executor): Retry kubectl on internal transient error (#8092)
* [ea36c337d](https://github.com/argoproj/argo-workflows/commit/ea36c337d8805534c3f358d1b44b2f1e50c8141a) fix: workflow.duration' is not available as a real time metric (#8181)
* [d10a7310c](https://github.com/argoproj/argo-workflows/commit/d10a7310c08273209b01c55d325e77407ee5f75c) fix: Correctly order emissary combined output. Fixes #8159 (#8175)
* [442096bf2](https://github.com/argoproj/argo-workflows/commit/442096bf2e893e5034fd0120889244ad6a50387c) fix: Add instance ID to `workflowtaskresults` (#8150)
* [2b87f860d](https://github.com/argoproj/argo-workflows/commit/2b87f860d1dc4007c799337f02101ead89297a11) fix: add /etc/mime.types mapping table (#8171)
* [26471c8ee](https://github.com/argoproj/argo-workflows/commit/26471c8ee2895a275ff3a180e6b92545e7c2dfee) fix: Limit workflows to 128KB and return a friendly error message (#8169)
* [dfca6f1e5](https://github.com/argoproj/argo-workflows/commit/dfca6f1e57eea85e1994a8e39ac56421a1cb466d) fix: Remove need for `get pods` from Emissary (#8133)
* [049d3d11f](https://github.com/argoproj/argo-workflows/commit/049d3d11f3d1e10a4b1b1edddea60030abb80e0b) fix: Fix the TestStopBehavior flackiness (#8096)
* [0cec27390](https://github.com/argoproj/argo-workflows/commit/0cec27390b55bace1c66da8cf7a24194b4ee0c09) fix: pod deleted due to delayed cleanup. Fixes #8022 (#8061)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins
* Felix Seidel
* Ming Yu Shi
* Rohan Kumar
* Saravanan Balasubramanian
* Vignesh
* William Van Hevelingen
* Wu Jayway

</details>

## v3.3.0 (2022-03-14)

Full Changelog: [v3.3.0-rc10...v3.3.0](https://github.com/argoproj/argo-workflows/compare/v3.3.0-rc10...v3.3.0)

### Selected Changes

<details><summary><h3>Contributors</h3></summary>

* Saravanan Balasubramanian

</details>

## v3.3.0-rc10 (2022-03-07)

Full Changelog: [v3.3.0-rc9...v3.3.0-rc10](https://github.com/argoproj/argo-workflows/compare/v3.3.0-rc9...v3.3.0-rc10)

### Selected Changes

* [e6b3ab548](https://github.com/argoproj/argo-workflows/commit/e6b3ab548d1518630954205c6e2ef0f18e74dcf9) fix: Use EvalBool instead of explicit casting (#8094)
* [6640689e3](https://github.com/argoproj/argo-workflows/commit/6640689e36918d3c24b2af8317d0fdadba834770) fix: e2e TestStopBehavior (#8082)

<details><summary><h3>Contributors</h3></summary>

* Caelan U
* Saravanan Balasubramanian
* Simon Behar
* Yuan Tang
* github-actions[bot]

</details>

## v3.3.0-rc9 (2022-03-04)

Full Changelog: [v3.3.0-rc8...v3.3.0-rc9](https://github.com/argoproj/argo-workflows/compare/v3.3.0-rc8...v3.3.0-rc9)

### Selected Changes

* [4decbea99](https://github.com/argoproj/argo-workflows/commit/4decbea991e49313624a3dc71eb9aadb906e82c8) fix: test
* [e2c53e6b9](https://github.com/argoproj/argo-workflows/commit/e2c53e6b9a3194353874b9c22e61696ca228cd24) fix: lint
* [5d8651d5c](https://github.com/argoproj/argo-workflows/commit/5d8651d5cc65cede4f186dd9d99c5f1b644d5f56) fix: e2e
* [4a2b2bd02](https://github.com/argoproj/argo-workflows/commit/4a2b2bd02b3a62daf61987502077877bbdb4bcca) fix: Make workflow.status available to template level (#8066)
* [baa51ae5d](https://github.com/argoproj/argo-workflows/commit/baa51ae5d74b53b8e54ef8d895eae36b9b50375b) feat: Expand `mainContainer` config to support all fields. Fixes #7962 (#8062)
* [cedfb1d9a](https://github.com/argoproj/argo-workflows/commit/cedfb1d9ab7a7cc58c9032dd40509dc34666b3e9) fix: Stop the workflow if activeDeadlineSeconds has beed patched (#8065)
* [662a7295b](https://github.com/argoproj/argo-workflows/commit/662a7295b2e263f001b94820ebde483fcf7f038d) feat: Replace `patch pod` with `create workflowtaskresult`. Fixes #3961 (#8000)
* [9aa04a149](https://github.com/argoproj/argo-workflows/commit/9aa04a1493c01782ed51b01c733ca6993608ea5b) feat: Remove plugin Kube API access by default. (#8028)
* [f9c7ab58e](https://github.com/argoproj/argo-workflows/commit/f9c7ab58e20fda8922fa00e9d468bda89031887a) fix: directory traversal vulnerability (#7187)
* [931cbbded](https://github.com/argoproj/argo-workflows/commit/931cbbded2d770e451895cc906ebe8e489ff92a6) fix(executor): handle podlog in deadlineExceed termination. Fixes #7092 #7081 (#7093)
* [8eb862ee5](https://github.com/argoproj/argo-workflows/commit/8eb862ee57815817e437368d0680b824ded2cda4) feat: fix naming (#8045)
* [b7a525be4](https://github.com/argoproj/argo-workflows/commit/b7a525be4014e3bdd28124c8736c25a007049ae7) feat!: Remove deprecated config flags. Fixes #7971 (#8009)
* [46f901311](https://github.com/argoproj/argo-workflows/commit/46f901311a1fbbdc041a3a15e78ed70c2b889849) feat: Add company AKRA GmbH (#8036)
* [7bf377df7](https://github.com/argoproj/argo-workflows/commit/7bf377df7fe998491ada5023be49521d3a44aba6) Yubo added to users (#8040)
* [fe8ac30b0](https://github.com/argoproj/argo-workflows/commit/fe8ac30b0760f61b679a605569c197670461ad65) fix: Support for custom HTTP headers. Fixes #7985 (#8004)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins
* Anurag Pathak
* Caelan U
* Laurent Rochette
* Niklas Hansson
* Saravanan Balasubramanian
* Tianchu Zhao
* Todor Todorov
* Wojciech Pietrzak
* Yuan Tang
* cui fliter
* dependabot[bot]
* descrepes
* github-actions[bot]
* kennytrytek

</details>

## v3.3.0-rc8 (2022-02-28)

Full Changelog: [v3.3.0-rc7...v3.3.0-rc8](https://github.com/argoproj/argo-workflows/compare/v3.3.0-rc7...v3.3.0-rc8)

### Selected Changes

* [9655a8348](https://github.com/argoproj/argo-workflows/commit/9655a834800c0936dbdc1045b49f587a92d454f6) fix: panic on synchronization if workflow has mutex and semaphore (#8025)
* [957330301](https://github.com/argoproj/argo-workflows/commit/957330301e0b29309ae9b08a376b012a639e1dd5) fix: Fix/client go/releaseoncancel. Fixes  #7613 (#8020)
* [c5c3b3134](https://github.com/argoproj/argo-workflows/commit/c5c3b31344650be516a6c00da88511b06f38f1b8) fix!: Document `workflowtaskset` breaking change. Fixes #8013 (#8015)
* [56dc11cef](https://github.com/argoproj/argo-workflows/commit/56dc11cef56a0b690222116d52976de9a8418e55) feat: fix path for plugin example (#8014)
* [06d4bf76f](https://github.com/argoproj/argo-workflows/commit/06d4bf76fc2f8ececf2b25a0ba5a81f844445b0f) fix: Reduce agent permissions. Fixes #7986 (#7987)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins
* Niklas Hansson
* Saravanan Balasubramanian
* Shyukri Shyukriev
* github-actions[bot]

</details>

## v3.3.0-rc7 (2022-02-25)

Full Changelog: [v3.3.0-rc6...v3.3.0-rc7](https://github.com/argoproj/argo-workflows/compare/v3.3.0-rc6...v3.3.0-rc7)

### Selected Changes

* [20f7516f9](https://github.com/argoproj/argo-workflows/commit/20f7516f916fb2c656ed3bf9d1d7bee18d136d53) fix: Re-factor `assessNodeStatus`. Fixes #7996 (#7998)
* [f0fb0d56d](https://github.com/argoproj/argo-workflows/commit/f0fb0d56d3f896ef74e39c2e391de2c4a30a1a52) chore(deps): bump github.com/minio/minio-go/v7 from 7.0.15 to 7.0.23 (#8003)
* [7e34ac513](https://github.com/argoproj/argo-workflows/commit/7e34ac5138551f0ebe0ca13ebfb4ad1fc8553ef1) feat: Support `workflow.parameters` in workflow meta-data. Fixes #3434 (#7711)
* [aea6c3912](https://github.com/argoproj/argo-workflows/commit/aea6c391256ece81b1d81a1d3cfe59088fa91f8d) chore(deps): bump github.com/gorilla/websocket from 1.4.2 to 1.5.0 (#7991)
* [89d7cc39d](https://github.com/argoproj/argo-workflows/commit/89d7cc39df386507b59c4858968ee06b33168faa) chore(deps): bump github.com/tidwall/gjson from 1.13.0 to 1.14.0 (#7992)
* [7c0e28901](https://github.com/argoproj/argo-workflows/commit/7c0e2890154ee187a8682c8fa6532952d73ef02c) fix: Generate SDKS (#7989)
* [980f2feb7](https://github.com/argoproj/argo-workflows/commit/980f2feb7b887b23513f1fc0717321bfdf134506) chore(deps): bump github.com/gavv/httpexpect/v2 from 2.2.0 to 2.3.1 (#7979)
* [5e45cd95a](https://github.com/argoproj/argo-workflows/commit/5e45cd95a084ec444dfc4c30b27f83ba8503b8e7) chore(deps): bump github.com/antonmedv/expr from 1.8.9 to 1.9.0 (#7967)
* [857768949](https://github.com/argoproj/argo-workflows/commit/8577689491b4d7375dde01faeab4c12eef2ba076) feat: Reduce agent pod permissions. Fixes #7914 (#7915)
* [d57fd0ff4](https://github.com/argoproj/argo-workflows/commit/d57fd0ff409d9f5fa238e0b726c83e0c366012ab) fix: Report container, plugin and HTTP  progress. Fixes #7918 (#7960)
* [412ff1c41](https://github.com/argoproj/argo-workflows/commit/412ff1c41196cb602aa7bb98a39e8ec90c08ada5) feat(controller): skip resolve artifact when when evaluates to fals one on withsequence (#7950)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins
* Niklas Hansson
* Saravanan Balasubramanian
* Sudhanshu Kumar Rai
* Tianchu Zhao
* Tomas Valasek
* William Van Hevelingen
* Yuan Tang
* dependabot[bot]
* github-actions[bot]

</details>

## v3.3.0-rc6 (2022-02-21)

Full Changelog: [v3.3.0-rc5...v3.3.0-rc6](https://github.com/argoproj/argo-workflows/compare/v3.3.0-rc5...v3.3.0-rc6)

### Selected Changes

<details><summary><h3>Contributors</h3></summary>

</details>

## v3.3.0-rc5 (2022-02-21)

Full Changelog: [v3.3.0-rc4...v3.3.0-rc5](https://github.com/argoproj/argo-workflows/compare/v3.3.0-rc4...v3.3.0-rc5)

### Selected Changes

* [79fc4a9be](https://github.com/argoproj/argo-workflows/commit/79fc4a9bea8d76905d314ac41df7018b556a91d6) chore(deps): bump upper.io/db.v3 (#7939)
* [ad312674a](https://github.com/argoproj/argo-workflows/commit/ad312674a0bbe617d199f4497e79b3e0fb6d64a8) fix: Fix broken Windows build (#7933)
* [5b6bfb6d3](https://github.com/argoproj/argo-workflows/commit/5b6bfb6d334914d8a8722f4d78b4794a92520757) fix: Fix `rowserrcheck` lint errors (#7924)
* [848effce0](https://github.com/argoproj/argo-workflows/commit/848effce0c61978de9da4da93d25a9f78ef1a0a8) chore(deps): bump github.com/aliyun/aliyun-oss-go-sdk (#7919)
* [044389b55](https://github.com/argoproj/argo-workflows/commit/044389b55990cb4d13fda279fed48f9bfd3d1112) chore(deps): bump github.com/aliyun/aliyun-oss-go-sdk (#7901)
* [ce00cd8ed](https://github.com/argoproj/argo-workflows/commit/ce00cd8edae68ad8aa5ed6003b574be903a5c346) feat: Support insecureSkipVerify for HTTP templates. Fixes #7790 (#7885)
* [11890b4cc](https://github.com/argoproj/argo-workflows/commit/11890b4cc14405902ee336e9197dd153df27c36b) feat: Update new version modal for v3.3. Fixes #7639 (#7707)
* [3524615b8](https://github.com/argoproj/argo-workflows/commit/3524615b89bd6da041413b88025cddeed8a704ad) fix: Add license to python sdk. Fixes #7881 (#7883)
* [80e7a27bf](https://github.com/argoproj/argo-workflows/commit/80e7a27bf08431204994bf848afdf2d5af8a94c1) fix: Increase padding between elements in workflow template creator. Fixes #7309 (#7420)
* [7776a1113](https://github.com/argoproj/argo-workflows/commit/7776a11131195a580618962f8ec4c0d23fe59cee) Add nil-check in LintWorkflow (#7769)
* [c0c24d24e](https://github.com/argoproj/argo-workflows/commit/c0c24d24e8ac5a2fd69def064dd9f0ed2bcf0326) fix: trim spaces while parse realtime metrics value. Fixes #7819 (#7856)
* [dc82f3f42](https://github.com/argoproj/argo-workflows/commit/dc82f3f428e3b8f17a7ea9121919b6270d1967f7) chore(deps): bump github.com/prometheus/client_golang (#7880)
* [bb8d2858d](https://github.com/argoproj/argo-workflows/commit/bb8d2858da78bf3eb0022688e34020668bbc08a9) fix: workflow-node-info long attribute message cannot be wrapped in the ui (#7876)
* [808c561f1](https://github.com/argoproj/argo-workflows/commit/808c561f1c4a56668c32caa69be5b0441d372610) feat: add container-set retry strategy.  Fixes #7290 (#7377)
* [31cc8bf98](https://github.com/argoproj/argo-workflows/commit/31cc8bf98864c15192845ee6f2349bd0099a71ae) fix(cli): fix typo in argo cron error messages (#7875)
* [87cb15591](https://github.com/argoproj/argo-workflows/commit/87cb1559107ec88dd418229b38113d70ba2a8580) fix: added priorityclass to workflow-controller. Fixes #7733 (#7859)
* [69c5bc79f](https://github.com/argoproj/argo-workflows/commit/69c5bc79f38e4aa7f4526111900904ac56e13d54) fix: Fix go-jose dep. Fixes #7814 (#7874)
* [28412ef7c](https://github.com/argoproj/argo-workflows/commit/28412ef7c37b1e1b2be0d60c46c5327f682a6a00) fix: Add env to argo-server deployment manifest. Fixes #7285 (#7852)
* [fce82d572](https://github.com/argoproj/argo-workflows/commit/fce82d5727b89cfe49e8e3568fff40725bd43734) feat: remove pod workers. Fixes #4090 (#7837)
* [938fde967](https://github.com/argoproj/argo-workflows/commit/938fde9673cf7aabe04587e63a28a3aa34ea049e) fix(ui): unauthorized login screen redirection to token creation docs (#7846)
* [1d7a17714](https://github.com/argoproj/argo-workflows/commit/1d7a17714fda0d8331ce11c765f0c95797c75afe) chore(deps): bump github.com/soheilhy/cmux from 0.1.4 to 0.1.5 (#7848)
* [1113f70fa](https://github.com/argoproj/argo-workflows/commit/1113f70fa0152fef5955a295bd5df50242fe9a67) fix: submitting Workflow from WorkflowTemplate will set correct serviceAccount and securityContext. Fixes #7726 (#7805)

<details><summary><h3>Contributors</h3></summary>

* AdamKorcz
* Alex Collins
* Baz Chalk
* Daniel Helfand
* Dillen Padhiar
* Doğukan Tuna
* Eng Zer Jun
* Felix Seidel
* Isitha Subasinghe
* Jin Dong
* Ken Kaizu
* Lukasz Stolcman
* Markus Lippert
* Niklas Hansson
* Oleg
* Rohan Kumar
* Scott Ernst
* Tianchu Zhao
* Vrukshali Torawane
* Yuan Tang
* Zhong Dai
* dependabot[bot]
* github-actions[bot]

</details>

## v3.3.0-rc4 (2022-02-08)

Full Changelog: [v3.3.0-rc3...v3.3.0-rc4](https://github.com/argoproj/argo-workflows/compare/v3.3.0-rc3...v3.3.0-rc4)

### Selected Changes

* [27977070c](https://github.com/argoproj/argo-workflows/commit/27977070c75e9369e16dd15025893047a95f85a5) chore(deps): bump github.com/go-openapi/spec from 0.20.2 to 0.20.4 (#7817)
* [1a1cc9a9b](https://github.com/argoproj/argo-workflows/commit/1a1cc9a9bc3dfca245c34ab9ecdeed7c52578ed5) feat: Surface container and template name in emissary error message. Fixes #7780 (#7807)
* [fb73d0194](https://github.com/argoproj/argo-workflows/commit/fb73d01940b6d1673c3fbc9238fbd26c88aba3b7) feat: make submit workflow parameter form as textarea to input multi line string easily (#7768)
* [932466540](https://github.com/argoproj/argo-workflows/commit/932466540a109550b98714f41a5c6e1d3fc13158) fix: Use v1 pod name if no template name or ref. Fixes #7595 and #7749 (#7605)
* [e9b873ae3](https://github.com/argoproj/argo-workflows/commit/e9b873ae3067431ef7cbcfa6744c57a19adaa9f5) fix: Missed workflow should not trigger if Forbidden Concurreny with no StartingDeadlineSeconds (#7746)
* [e12827b8b](https://github.com/argoproj/argo-workflows/commit/e12827b8b0ecb11425399608b1feee2ad739575d) feat: add claims.Email into gatekeeper audit log entry (#7748)
* [74d1bbef7](https://github.com/argoproj/argo-workflows/commit/74d1bbef7ba33466366623c82343289ace41f01a) chore(deps): bump cloud.google.com/go/storage from 1.19.0 to 1.20.0 (#7747)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins
* J.P. Zivalich
* Ken Kaizu
* Saravanan Balasubramanian
* William Van Hevelingen
* Youngcheol Jang
* Yuan Tang
* dependabot[bot]
* github-actions[bot]

</details>

## v3.3.0-rc3 (2022-02-03)

Full Changelog: [v3.3.0-rc2...v3.3.0-rc3](https://github.com/argoproj/argo-workflows/compare/v3.3.0-rc2...v3.3.0-rc3)

### Selected Changes

* [70715ecc8](https://github.com/argoproj/argo-workflows/commit/70715ecc8a8d29c5800cc7176923344939038cc6) fix: artifacts.(\*ArtifactServer).GetInputArtifactByUID ensure valid request path (#7730)
* [1277f0579](https://github.com/argoproj/argo-workflows/commit/1277f05796cdf8c50e933ccdf8d665b6bf8d184c) chore(deps): bump gopkg.in/square/go-jose.v2 from 2.5.1 to 2.6.0 (#7740)
* [7e6f2c0d7](https://github.com/argoproj/argo-workflows/commit/7e6f2c0d7bf493ee302737fd2a4e650b9bc136fc) chore(deps): bump github.com/valyala/fasttemplate from 1.1.0 to 1.2.1 (#7727)
* [877d65697](https://github.com/argoproj/argo-workflows/commit/877d6569754be94f032e1c48d1f7226a83adfbec) chore(deps): bump cloud.google.com/go/storage from 1.10.0 to 1.19.0 (#7714)
* [05fc4a795](https://github.com/argoproj/argo-workflows/commit/05fc4a7957f16a37ef018bd715b904ab33ce716b) chore(deps): bump peaceiris/actions-gh-pages from 2.5.0 to 2.9.0 (#7713)
* [bf3b58b98](https://github.com/argoproj/argo-workflows/commit/bf3b58b98ac62870b779ac4aad734130ee5473b2) fix: ContainerSet termination during pending Pod #7635 (#7681)
* [f6c9a6aa7](https://github.com/argoproj/argo-workflows/commit/f6c9a6aa7734263f478b9cef2bcb570d882f135c) fix: Pod "START TIME"/ "END TIME" tooltip shows time in UTC and local timezone Fixes #7488 (#7694)
* [e2e046f6f](https://github.com/argoproj/argo-workflows/commit/e2e046f6fded6581f153598100d3ccf9bb661912) fix: Fix argo lint panic when missing param value in DAG task. Fixes #7701 (#7706)
* [72817f2b8](https://github.com/argoproj/argo-workflows/commit/72817f2b89c60f30d5dc73fc256ae0399e57737e) feat: Add variable substitution on ConfigMapKeySelector. Fixes #7061 (#7542)
* [0f4c48473](https://github.com/argoproj/argo-workflows/commit/0f4c48473c7281671e84d96392f89ec35f38fb42) chore(deps): bump gopkg.in/go-playground/webhooks.v5 (#7704)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins
* Denis Melnik
* Henrik Blixt
* Paco Guzmán
* Tino Schröter
* Yago Riveiro
* Yuan Tang
* dependabot[bot]
* github-actions[bot]

</details>

## v3.3.0-rc2 (2022-01-29)

Full Changelog: [v3.3.0-rc1...v3.3.0-rc2](https://github.com/argoproj/argo-workflows/compare/v3.3.0-rc1...v3.3.0-rc2)

### Selected Changes

* [753509394](https://github.com/argoproj/argo-workflows/commit/75350939442d26f35afc57ebe183280dc3d158ac) fix: Handle release candidate versions in Python SDK version. Fixes #7692 (#7693)

<details><summary><h3>Contributors</h3></summary>

* Yuan Tang

</details>

## v3.3.0-rc1 (2022-01-28)

Full Changelog: [v3.2.11...v3.3.0-rc1](https://github.com/argoproj/argo-workflows/compare/v3.2.11...v3.3.0-rc1)

### Selected Changes

* [45730a9cd](https://github.com/argoproj/argo-workflows/commit/45730a9cdeb588d0e52b1ac87b6e0ca391a95a81) feat: lifecycle hook (#7582)
* [4664aeac4](https://github.com/argoproj/argo-workflows/commit/4664aeac4ffa208114b8483e6300c39b537b402d) chore(deps): bump google.golang.org/grpc from v1.38.0 to v1.41.1 (#7658)
* [ecf2ceced](https://github.com/argoproj/argo-workflows/commit/ecf2cecedcf8fd3f70a846372e85c471b6512aca) chore(deps): bump github.com/grpc-ecosystem/go-grpc-middleware (#7679)
* [67c278cd1](https://github.com/argoproj/argo-workflows/commit/67c278cd1312d695d9925f64f24957c1449219cc) fix: Support terminating with `templateRef`. Fixes #7214 (#7657)
* [1159afc3c](https://github.com/argoproj/argo-workflows/commit/1159afc3c082c62f6142fad35ba461250717a8bb) fix: Match cli display pod names with k8s. Fixes #7646 (#7653)
* [6a97a6161](https://github.com/argoproj/argo-workflows/commit/6a97a616177e96fb80e43bd1f98eac595f0f0a7d) fix: Retry with DAG. Fixes #7617 (#7652)
* [559153417](https://github.com/argoproj/argo-workflows/commit/559153417db5a1291bb1077dc61ee8e6eb787c41) chore(deps): bump github.com/prometheus/common from 0.26.0 to 0.32.1 (#7660)
* [a20150c45](https://github.com/argoproj/argo-workflows/commit/a20150c458c45456e40ef73d91f0fa1561b85a1e) fix: insecureSkipVerify needed. Fixes #7632 (#7651)
* [0137e1980](https://github.com/argoproj/argo-workflows/commit/0137e1980f2952e40c1d11d5bf53e18fe0f3914c) fix: error when path length != 6 (#7648)
* [b7cd2f5a9](https://github.com/argoproj/argo-workflows/commit/b7cd2f5a93effaa6473001da87dc30eaf9814822) feat: add overridable default input artifacts #2026  (#7647)
* [17342bacc](https://github.com/argoproj/argo-workflows/commit/17342bacc991c1eb9cce5639c857936d3ab8c5c9) chore(deps): bump peaceiris/actions-gh-pages from 2.5.0 to 3.8.0 (#7642)
* [6f60703db](https://github.com/argoproj/argo-workflows/commit/6f60703dbfb586607a491c8bebc8425029853c84) fix: Fix non-standard git username support. Fixes #7593 (#7634)
* [0ce9e70ef](https://github.com/argoproj/argo-workflows/commit/0ce9e70ef72274d69c4bfb5a6c83d1fdefa9038a) fix: SSO to handle multiple authorization cookies such as from wildca… (#7607)
* [3614db690](https://github.com/argoproj/argo-workflows/commit/3614db690aea3e0c4e5221fa1b2c851ca70e6b18) feat: adding support for getting tls certificates from kubernetes secret (e.g. (#7621)
* [596f94c90](https://github.com/argoproj/argo-workflows/commit/596f94c900ebbe41930472364e2b2298220e9ca7) feat: customize nav bar background color (#7387)
* [774bf47ee](https://github.com/argoproj/argo-workflows/commit/774bf47ee678ef31d27669f7d309dee1dd84340c) feat: Template executor plugin. (#7256)
* [d2e98d6b4](https://github.com/argoproj/argo-workflows/commit/d2e98d6b45e01ec7d7b614f22291e008faedcf01) fix: Support artifact ref from tmpl in UI. Fixes #7587 (#7591)
* [c6be0fe77](https://github.com/argoproj/argo-workflows/commit/c6be0fe774e736059dd53e5cf80f2a99c4a3c569) feat(ui): Show first-time UX. Fixes #7160 (#7170)
* [2e343eb7f](https://github.com/argoproj/argo-workflows/commit/2e343eb7f1328c8ec242116d38bb7e651703ea26) fix: Upgrade prismjs to v1.26 to fix security scan. Fixes #7599 (#7601)
* [f9fa0e303](https://github.com/argoproj/argo-workflows/commit/f9fa0e303da39accd3e1268361df4f70dc6e391e) fix: Support inputs for inline DAG template. Fixes #7432 (#7439)
* [bc27ada85](https://github.com/argoproj/argo-workflows/commit/bc27ada852c57ebf7a3f87e2eaf161cc72ad7198) fix: Fix inconsistent ordering of workflows with the list command. Fixes #7581 (#7594)
* [af257c178](https://github.com/argoproj/argo-workflows/commit/af257c178b78f0a7cae6af38e15b20bfcf3dba6a) feat: Support templateRef in LifecycleHook. Fixes #7558 (#7570)
* [f1fe3bee4](https://github.com/argoproj/argo-workflows/commit/f1fe3bee498ac7eb895af6f89a0eba5095410467) fix: hanging wait container on save artifact to GCS bucket artifactRepository (#7536)
* [a94b846e6](https://github.com/argoproj/argo-workflows/commit/a94b846e67382252831d44624c2f4b1708f7a30c) fix: fix nil point about Outputs.ExitCode. Fixes #7458 (#7459)
* [e395a5b03](https://github.com/argoproj/argo-workflows/commit/e395a5b0381560d59aba928ea31f5cd4e7c04665) Update workflow-restrictions.md (#7508)
* [b056de384](https://github.com/argoproj/argo-workflows/commit/b056de3847db2e654f761ce15309ac7629ea1dc9) Add new line to render bullets properly. (#7579)
* [4b83de9b5](https://github.com/argoproj/argo-workflows/commit/4b83de9b527e59bc29746a824efbe97daa47e504) fix: More informative error message when artefact repository is not configured. Fixes #7372 (#7498)
* [2ab7dfebe](https://github.com/argoproj/argo-workflows/commit/2ab7dfebe13c20a158d5def3f1932fdbc54041d4) fix: update old buildkit version in buildkit-template.yaml (#7512)
* [c172d1dce](https://github.com/argoproj/argo-workflows/commit/c172d1dcef3e787d49a6fe637922de733a054a84) fix: show invalid cron schedule error on cron status ui (#7441)
* [fbf4751f4](https://github.com/argoproj/argo-workflows/commit/fbf4751f45052750024901f6a2ba56b65587d701) fix: resolve resourcesDuration (#7299)
* [033ed978e](https://github.com/argoproj/argo-workflows/commit/033ed978e2d5ec05c862259a92d3ec35e0bfd1d9) fix(controller): fix pod stuck in running when using podSpecPatch and emissary (#7407)
* [ebdde3392](https://github.com/argoproj/argo-workflows/commit/ebdde3392b0c50b248dfbb8b175ef8acff265ed1) fix: Fix supplied global workflow parameters (#7573)
* [eb1c3e0b4](https://github.com/argoproj/argo-workflows/commit/eb1c3e0b40f74ca1a52ef0f7fd7a7cb79ae2987f) feat: Adds timezone to argo cron list output (#7557) (#7559)
* [dbb1bcfbd](https://github.com/argoproj/argo-workflows/commit/dbb1bcfbd4de3295163900509fc624fb7d363b10) fix: add priority field to submitopts (#7572)
* [bc1f304a9](https://github.com/argoproj/argo-workflows/commit/bc1f304a93149131452687162801e865c7decc14) fix: Fix type assertion bug (#7556)
* [970a503c5](https://github.com/argoproj/argo-workflows/commit/970a503c561a0cdb30a7b1ce2ed8d34b1728e61f) fix: nil-pointer in util.ApplySubmitOpts (#7529)
* [18821c57f](https://github.com/argoproj/argo-workflows/commit/18821c57fbea7c86abc3a347155e1ce0cde92ea0) fix: handle source file is empty for script template (#7467)
* [b476c4af5](https://github.com/argoproj/argo-workflows/commit/b476c4af505b6f24161a3818c358f6f6b012f87e) fix: Make dev version of the Python SDK PEP440 compatible (#7525)
* [26c1224b0](https://github.com/argoproj/argo-workflows/commit/26c1224b0d8b0786ef1a75a58e49914810d3e115) fix: transient errors for s3 artifacts: Fixes #7349 (#7352)
* [3371e7268](https://github.com/argoproj/argo-workflows/commit/3371e7268c1ed5207d840285133a0d2f0417bbb9) fix: http template doesn't update progress. Fixes #7239 (#7450)
* [4b006d5f8](https://github.com/argoproj/argo-workflows/commit/4b006d5f8eb338f91f1b77a813dc8a09d972c131) fix: Global param value incorrectly overridden when loading from configmaps (#7515)
* [0f206d670](https://github.com/argoproj/argo-workflows/commit/0f206d670eb38c6b02c9015b30b04ff0396289c8) fix: only aggregates output from successful nodes (#7517)
* [318927ed6](https://github.com/argoproj/argo-workflows/commit/318927ed6356d10c73fe775790b7765ea17480d4) fix: out of range in MustUnmarshal (#7485)
* [d3ecdf11c](https://github.com/argoproj/argo-workflows/commit/d3ecdf11c145be97c1c1e4ac4d20d5d543ae53ca) feat: add workflow.labels and workflow.annotations as JSON string. Fixes: #7289 (#7396)
* [4f9e299b7](https://github.com/argoproj/argo-workflows/commit/4f9e299b7f7d8d7084ac0def2a6902b26d2b9b5e) fix: shutdown workqueues to avoid goroutine leaks (#7493)
* [dd77dc993](https://github.com/argoproj/argo-workflows/commit/dd77dc9937bdd9ab97c837e7f3f88ef5ecc2cae3) fix: submitting cluster workflow template on namespaced install returns error (#7437)
* [e4b0f6576](https://github.com/argoproj/argo-workflows/commit/e4b0f65762225962d40e0d8cade8467435876470) feat: Add Python SDK versioning script (#7429)
* [d99796b2f](https://github.com/argoproj/argo-workflows/commit/d99796b2f7e8c9fb895205461cc2a461f0cd643d) fix: Disable SDK release from master branch (#7419)
* [dbda60fc5](https://github.com/argoproj/argo-workflows/commit/dbda60fc5c72c02729d98b4e5ff08f89a6bf428c) feat: Python SDK publish (#7363)
* [79d50fc27](https://github.com/argoproj/argo-workflows/commit/79d50fc278d1d5e1dc8fbc27285c28b360426ce4) fix: Correct default emissary bug. Fixes #7224 (#7412)
* [014bac90f](https://github.com/argoproj/argo-workflows/commit/014bac90ff0c62212ebae23d6dd9a1ed8c7d3a8c) fix: added check for initContainer name in workflow template (#7411)
* [81afc8a7b](https://github.com/argoproj/argo-workflows/commit/81afc8a7b482aa9b95e010e02f9ef48dea7d7161) feat: List UID with 'argo archive list' (#7384)
* [8d552fbf6](https://github.com/argoproj/argo-workflows/commit/8d552fbf6b3752025955b233a9462b34098cedf1) feat: added retention controller. Fixes #5369 (#6854)
* [932040848](https://github.com/argoproj/argo-workflows/commit/932040848404d42a007b19bfaea685d4b505c2ef) fix: Skip missed executions if CronWorkflow schedule is changed. Fixes #7182 (#7353)
* [79a95f223](https://github.com/argoproj/argo-workflows/commit/79a95f223396ecab408d831781ab2d38d1fa6de0) feat: Add SuccessCondition to HTTP template  (#7303)
* [aba6599f5](https://github.com/argoproj/argo-workflows/commit/aba6599f5759e57882172c8bc74cc63a2a809148) feat: Adjust name of generated Python SDK (#7328)
* [78dd747c6](https://github.com/argoproj/argo-workflows/commit/78dd747c600541c7ae2e71b473c0652fdd105c66) fix: Propogate errors in task worker and don't return (#7357)
* [8bd7f3971](https://github.com/argoproj/argo-workflows/commit/8bd7f3971e87d86ecd0c1887d49511b325207ab8) fix: argument of PodName function (fixes #7315) (#7316)
* [6423b6995](https://github.com/argoproj/argo-workflows/commit/6423b6995f06188c11eddb3ad23ae6631c2bf025) feat: support workflow template parameter description (#7309) (#7346)
* [1a3b87bdf](https://github.com/argoproj/argo-workflows/commit/1a3b87bdf8edba02ba5e5aed20f3942be1d6f46c) fix: improve error message for ListArchivedWorkflows (#7345)
* [77d87da3b](https://github.com/argoproj/argo-workflows/commit/77d87da3be49ee344090f3ee99498853fdb30ba2) fix: Use and enforce structured logging. Fixes #7243  (#7324)
* [3e727fa38](https://github.com/argoproj/argo-workflows/commit/3e727fa3878adf4133bde56a5fd18e3c50249279) feat: submit workflow make button disable after clicking (#7340)
* [cb8c06369](https://github.com/argoproj/argo-workflows/commit/cb8c06369fec5e499770f5ea1109c862eb213e3b) fix: cannot access HTTP template's outputs (#7200)
* [e0d5abcff](https://github.com/argoproj/argo-workflows/commit/e0d5abcffc9e2d7423454995974a2e91aab6ca24) fix: Use DEFAULT_REQUEUE_TIME for Agent. Fixes #7269 (#7296)
* [242360a4f](https://github.com/argoproj/argo-workflows/commit/242360a4f26a378269aadcbaabca6a8fd6c618bf) fix(ui): Fix events error. Fixes #7320 (#7321)
* [cf78ff6d7](https://github.com/argoproj/argo-workflows/commit/cf78ff6d76b09c4002edbc28048c67335bd1d00f) fix: Validate the type of configmap before loading parameters. Fixes #7312 (#7314)
* [08254f547](https://github.com/argoproj/argo-workflows/commit/08254f547cad5f2e862bca2dd0f8fe52661f1314) fix: Handle the panic in operate function (#7262)
* [d4aa9d1a6](https://github.com/argoproj/argo-workflows/commit/d4aa9d1a6f308a59ec95bd0f0d6221fe899a6e06) feat(controller): Support GC for memoization caches (#6850)
* [77f520900](https://github.com/argoproj/argo-workflows/commit/77f520900bd79c7403aa81cd9e88dea0ba84c675) feat: Add `PodPriorityClassName` to `SubmitOpts`. Fixes #7059 (#7274)
* [88cbea332](https://github.com/argoproj/argo-workflows/commit/88cbea3325d7414a1ea60d2bcde3e71e9f5dfd7b) fix: pod name shown in log when pod deletion (#7301)
* [6c47c91e2](https://github.com/argoproj/argo-workflows/commit/6c47c91e29396df111d5b14867ab8de4befa1153) fix: Use default value for empty env vars (#7297)
* [c2b3e8e93](https://github.com/argoproj/argo-workflows/commit/c2b3e8e93a307842db623c99a7643d3974cee6af) feat: Allow remove of PVC protection finalizer. Fixes #6629 (#7260)
* [160bdc61e](https://github.com/argoproj/argo-workflows/commit/160bdc61e9eaa6e488c9871093504587cb585ab5) feat: Allow parallel HTTP requests (#7113)
* [e0455772a](https://github.com/argoproj/argo-workflows/commit/e0455772a2164093c16f95480a2d21d4ae34a069) fix: Fix `argo auth token`. Fixes #7175 (#7186)
* [0ea855479](https://github.com/argoproj/argo-workflows/commit/0ea85547984583d4919b8139ffd0dc3d2bdaf05e) fix: improve feedback when submitting a workflow from the CLI w/o a serviceaccount specified (#7246)
* [3d47a5d29](https://github.com/argoproj/argo-workflows/commit/3d47a5d29dee66775e6fa871dee1b6ca1ae6acda) feat(emissary executor): Add step to allow users to pause template before and after execution. Fixes #6841 (#6868)
* [1d715a05c](https://github.com/argoproj/argo-workflows/commit/1d715a05c09f1696f693fe8cd3d2e16a05c6368c) fix: refactor/fix pod GC. Fixes #7159 (#7176)
* [389f7f486](https://github.com/argoproj/argo-workflows/commit/389f7f4861653609dd6337b370350bedbe00e5c8) feat(ui): add pagination to workflow-templates (#7163)
* [09987a6dd](https://github.com/argoproj/argo-workflows/commit/09987a6dd03c1119fa286ed55cc97a2f4e588e09) feat: add CreatorUsername label when user is signed in via SSO. Fixes… (#7109)
* [f34715475](https://github.com/argoproj/argo-workflows/commit/f34715475b2c71aeba15e7311f3ef723f394fbbf) fix: add gh ecdsa and ed25519 to known hosts (#7226)
* [eb9a42897](https://github.com/argoproj/argo-workflows/commit/eb9a4289729c0d91bfa45cb5895e5bef61ce483e) fix: Fix ANSI color sequences escaping (#7211)
* [e8a2f3778](https://github.com/argoproj/argo-workflows/commit/e8a2f37784f57c289024f0c5061fde8ec248314e) feat(ui): Support log viewing for user supplied init containers (#7212)
* [1453edca7](https://github.com/argoproj/argo-workflows/commit/1453edca7c510df5b3cfacb8cf1f99a2b9635b1a) fix: Do not patch empty progress. fixes #7184 (#7204)
* [34e5b5477](https://github.com/argoproj/argo-workflows/commit/34e5b54779b25416d7dbd41d78e0effa523c1a21) fix: ci sleep command syntax for macOS 12 (#7203)
* [17fb9d813](https://github.com/argoproj/argo-workflows/commit/17fb9d813d4d0fb15b0e8652caa52e1078f9bfeb) fix: allow wf templates without parameter values (Fixes #6044) (#7124)
* [225a5a33a](https://github.com/argoproj/argo-workflows/commit/225a5a33afb0010346d10b65f459626eed8cd124) fix(test): Make TestMonitorProgress Faster (#7185)
* [52321e2ce](https://github.com/argoproj/argo-workflows/commit/52321e2ce4cb7077f38fca489059c06ec36732c4) feat(controller): Add default container annotation to workflow pod. FIxes: #5643 (#7127)
* [0482964d9](https://github.com/argoproj/argo-workflows/commit/0482964d9bc09585fd908ed5f912fd8c72f399ff) fix(ui): Correctly show zero-state when CRDs not installed. Fixes #7001 (#7169)
* [a6ce659f8](https://github.com/argoproj/argo-workflows/commit/a6ce659f80b3753fb05bbc3057e3b9795e17d211) feat!: Remove the hidden flag `verify` from `argo submit` (#7158)
* [f9e554d26](https://github.com/argoproj/argo-workflows/commit/f9e554d268fd9dbaf0e07f8a10a8ac03097250ce) fix: Relative submodules in git artifacts. Fixes #7141 (#7162)
* [22af73650](https://github.com/argoproj/argo-workflows/commit/22af7365049a34603cd109e2bcfa51eeee5e1393) fix: Reorder CI checks so required checks run first (#7142)
* [bd3be1152](https://github.com/argoproj/argo-workflows/commit/bd3be115299708dc4f97f3559e6f57f38c0c0d48) fix: Return error when YAML submission is invalid (#7135)
* [7886a2b09](https://github.com/argoproj/argo-workflows/commit/7886a2b090d4a31e1cacbc6cff4a8cb18914763c) feat: self reporting workflow progress (#6714)
* [877752428](https://github.com/argoproj/argo-workflows/commit/8777524281bb70e177c3e7f9d530d3cce6505864) feat: Add FAQ link to unknown pod watch error. Fixes #6886 (#6953)
* [209ff9d9b](https://github.com/argoproj/argo-workflows/commit/209ff9d9bd094e1c230be509d2444ae36b4ff04e) fix: Respect template.HTTP.timeoutSeconds (#7136)
* [02165aaeb](https://github.com/argoproj/argo-workflows/commit/02165aaeb83754ee15c635b3707b119a88ec43bd) fix(controller): default volume/mount to emissary (#7125)
* [475d8d54f](https://github.com/argoproj/argo-workflows/commit/475d8d54f0756e147775c28874de0859804e875c) feat: Adds SSO control via individual namespaces. Fixes #6916 (#6990)
* [af32f5799](https://github.com/argoproj/argo-workflows/commit/af32f57995dac8dbfd5ffe1a6477beb3004e254b) Revert "chore: only run API if needed"
* [3d597269e](https://github.com/argoproj/argo-workflows/commit/3d597269e48215080e3318019f1d95ee01d7dacd) fix: typo in node-field-selector.md (#7116)
* [e716aad73](https://github.com/argoproj/argo-workflows/commit/e716aad73072fbea8ed25306634002301909fa93) refactor: Fixing typo WriteTeriminateMessage #6999 (#7043)
* [ca87f2906](https://github.com/argoproj/argo-workflows/commit/ca87f2906995b8fecb796d94299f54f6dfbd6a41) fix: Daemon step in running state, but dependents don't start (#7107)
* [5eab921eb](https://github.com/argoproj/argo-workflows/commit/5eab921eb0f537f1102bbdd6c38b4e52740a88a9) feat: Add workflow logs  selector support. Fixes #6910 (#7067)
* [1e8715954](https://github.com/argoproj/argo-workflows/commit/1e871595414d05e2b250bfa3577cf23b9ab7fa38) fix: Add pod name format annotation. Fixes #6962 and #6989 (#6982)
* [93c11a24f](https://github.com/argoproj/argo-workflows/commit/93c11a24ff06049c2197149acd787f702e5c1f9b) feat: Add TLS to Metrics and Telemetry servers (#7041)
* [c5de76b6a](https://github.com/argoproj/argo-workflows/commit/c5de76b6a2d7b13c6ac7bc798e5c7615bf015de1) fix: Format issue on WorkflowEventBinding parameters. Fixes #7042 (#7087)
* [64fce4a82](https://github.com/argoproj/argo-workflows/commit/64fce4a827692cb67284d800ad92f1af37f654fc) fix: Ensure HTTP reconciliation occurs for onExit nodes (#7084)
* [d6a62c3e2](https://github.com/argoproj/argo-workflows/commit/d6a62c3e26d49ab752851be288bcd503386e8ff6) fix: Ensure HTTP templates have children assigned (#7082)
* [2bbba15cf](https://github.com/argoproj/argo-workflows/commit/2bbba15cf53395e0f4f729fd86f74355827b6d76) feat: Bring Python client to core (#7025)
* [46767b86b](https://github.com/argoproj/argo-workflows/commit/46767b86bc29cd8cb1df08fdcc0b5bb351c243f3) fix(ui): Correct HTTP connection in pipeline view (#7077)
* [201ba5525](https://github.com/argoproj/argo-workflows/commit/201ba552557b9edc5908c5224471fec4823b3302) fix: add outputs.parameters scope to script/containerSet templates. Fixes #6439 (#7045)
* [60f2ae95e](https://github.com/argoproj/argo-workflows/commit/60f2ae95e954e4af35cd93b12f554fbaf6ca1e41) feat: Add user's email in the server gatekeeper logs (#7062)
* [31bf57b64](https://github.com/argoproj/argo-workflows/commit/31bf57b643be995860ec77b942c2b587faa0b4ff) fix: Unit test TestNewOperation order of pipeline execution maybe different to order of submit (#7069)
* [4734cbc44](https://github.com/argoproj/argo-workflows/commit/4734cbc44dedeb2c7e5984aab5dc9b0c846ff491) fix: Precedence of ContainerRuntimeExecutor and ContainerRuntimeExecutors (#7056)
* [56ee94147](https://github.com/argoproj/argo-workflows/commit/56ee94147c1d65b03097b453e090e4930d8da591)  feat: Bring Java client into core.  (#7026)
* [65ff89ac8](https://github.com/argoproj/argo-workflows/commit/65ff89ac81a8350fb5c34043146fcb1ec4ffbf23) fix: Memozie for Step and DAG level (#7028)
* [8d7ca73b0](https://github.com/argoproj/argo-workflows/commit/8d7ca73b04438a17105312a07263fb6e5417f76e) feat: Upgrade to Golang 1.17 (#7029)
* [0baa4a203](https://github.com/argoproj/argo-workflows/commit/0baa4a2039b981e1ca118a04ceb6ac6439a82d0d) fix: Support RFC3339 in creationTimeStamp. Fixes #6906 (#7044)
* [25e1939e2](https://github.com/argoproj/argo-workflows/commit/25e1939e25551cd15d89bd47e4232c8073b40a9c) feat(ui): add label/state filter to cronworkflow. Fixes #7034 (#7035)
* [0758eab11](https://github.com/argoproj/argo-workflows/commit/0758eab11decb8a1e741abef3e0ec08c48a69ab8) feat(server): Sync dispatch of webhook events by default. Fixes #6981 and #6732 (#6995)
* [ba472e131](https://github.com/argoproj/argo-workflows/commit/ba472e1319d1a393107947aa6d5980906d1cb711) fix: Minor corrections to Swagger/JSON schema (#7027)
* [182b696df](https://github.com/argoproj/argo-workflows/commit/182b696df6652981e490af47deb321cb1bd741ff) feat: add unknown pod watch error explanation to FAQ.md (#6988)
* [3f0a531aa](https://github.com/argoproj/argo-workflows/commit/3f0a531aa14142a5f4f749093b23f690c98eb41e) fix(controller): use correct pod.name in retry/podspecpatch scenario. Fixes #7007 (#7008)
* [6a674e7cb](https://github.com/argoproj/argo-workflows/commit/6a674e7cb2e70259efe377db4235b3bc2dbdb9b0) feat(ui): wider stroke width for selected node (#7000)
* [7f5262338](https://github.com/argoproj/argo-workflows/commit/7f526233824c5065c7a9ee63dac59f168f04f95d) fix(ui): labels in report/archive should be sorted (#7009)
* [50813daaf](https://github.com/argoproj/argo-workflows/commit/50813daaf5b718d143af84f0f5847273114734da) fix(controller): fix bugs in processing retry node output parameters. Fixes #6948 (#6956)
* [86ddda592](https://github.com/argoproj/argo-workflows/commit/86ddda592c4f432f629775908bc9b737ab920cde) fix: Restore default pod name version to v1 (#6998)
* [0446f521d](https://github.com/argoproj/argo-workflows/commit/0446f521d045b542734ee11fafea99daa2ee3105) fix(artifact)!: default https to any URL missing a scheme. Fixes #6973 (#6985)
* [cfdebf64e](https://github.com/argoproj/argo-workflows/commit/cfdebf64eed8b87bf0f84f4284323e72f6d14cbb) fix(typo): correct typo in event-dispatch error log (#6688)
* [2a15853ec](https://github.com/argoproj/argo-workflows/commit/2a15853ec32701dd2dbccea2cc735d8334da1680) fix: OAuth2 callback with self-signed Root CA. Fixes #6793 (#6978)
* [6384e5f21](https://github.com/argoproj/argo-workflows/commit/6384e5f2104c3df69070c33da636599d413f7d6c) feat: fix workflow configmap argument cannot be referenced as local variable. Fixes #6869 (#6898)
* [72356abad](https://github.com/argoproj/argo-workflows/commit/72356abad157b26905be9251c654413b5eb9e6c7) fix: Allow self-signed Root CA for SSO. Fixes #6793 (#6961)
* [e1fe5b58a](https://github.com/argoproj/argo-workflows/commit/e1fe5b58a22e3bbac01e1328998591b37c29b1ad) feat(ui): add label filter to template workflow (#6955)
* [c705294c9](https://github.com/argoproj/argo-workflows/commit/c705294c9813b496b2de5c2ecd6f578d86a329b6) fix: response on canceled workflow action (#6859) (#6967)
* [cf9a6cdd0](https://github.com/argoproj/argo-workflows/commit/cf9a6cdd098901873ac584db649b694041530eb2) fix: Unreachable code in util/tls/tls.go. Fixes  #6950 (#6960)
* [6e1f2505a](https://github.com/argoproj/argo-workflows/commit/6e1f2505a18e427d3a39fadafad2fd83f6eff521) fix: multi-steps workflow (#6957)
* [452433989](https://github.com/argoproj/argo-workflows/commit/4524339892ae3e98bf6a5c9f11c5e2f41622f06c) fix(docs): fix data transformation example (#6901)
* [73d60108b](https://github.com/argoproj/argo-workflows/commit/73d60108b74341baf162580c11323624ba3936b5) fix(executor): add test for non-root user creating a script (#6905)
* [79d03a920](https://github.com/argoproj/argo-workflows/commit/79d03a9203d85d270017b5f0104fbf88879c6cdc) fix: Skip empty withParam tasks. Fixes #6834 (#6912)
* [b0d1f6583](https://github.com/argoproj/argo-workflows/commit/b0d1f658388ebd4ab2c1f26a87d66282304fa391) feat(executor): default executor to emissary. Fixes #6785 (#6882)
* [67fe87ba9](https://github.com/argoproj/argo-workflows/commit/67fe87ba9f3b8dbcb0f330a7ef593403d8909061) fix(ui): Change pod names to new format. Fixes #6865 (#6925)
* [1bcfa1aa5](https://github.com/argoproj/argo-workflows/commit/1bcfa1aa5dcb90559772be2a32512ba17d72c4ed) fix: BASE_HREF ignore (#6926)
* [41515d65c](https://github.com/argoproj/argo-workflows/commit/41515d65c2cc3ac1f492942e21fd33c4e31acdb1) fix(controller): Fix getPodByNode, TestGetPodByNode. Fixes #6458 (#6897)
* [5a7708c2c](https://github.com/argoproj/argo-workflows/commit/5a7708c2c449544905bbed474f9edc21e9fcf3e7) fix: do not delete expr tag tmpl values. Fixes #6909 (#6921)
* [2fd4b8aad](https://github.com/argoproj/argo-workflows/commit/2fd4b8aad161f0510fa5318de8f56724ec915e2a) feat(ui): label autocomplete for report tab (#6881)
* [c5b1533d3](https://github.com/argoproj/argo-workflows/commit/c5b1533d34c37d94defe98742a357c8e6b992db8) feat(ui): resume on selected node. Fixes #5763 (#6885)
* [ef6aad617](https://github.com/argoproj/argo-workflows/commit/ef6aad6171c4ed165078e9569364d7d7c54b434f) fix: Parameter with Value and Default (#6887)
* [4d38404df](https://github.com/argoproj/argo-workflows/commit/4d38404dfe2d6b941fece60c56db21a3b6f70c4b) fix: Resource requests on init/wait containers. Fixes #6809 (#6879)
* [cca4792c5](https://github.com/argoproj/argo-workflows/commit/cca4792c5adfd44340238122f7fe4e6010a96676) fix(ui): fixed width button (#6883)
* [b54809771](https://github.com/argoproj/argo-workflows/commit/b54809771b871b9425c476999100b0c72a4900aa) feat(server): archivedWf add namePrefix search. Fixes #6743 (#6801)
* [689ad6818](https://github.com/argoproj/argo-workflows/commit/689ad68182d9f2dc1479dc5f1398ff646cef4357) feat: add autocomplete for labels for archived workflow (#6776)
* [c962bb189](https://github.com/argoproj/argo-workflows/commit/c962bb189b491bcd8d2c4bedb75f778ca1301305) fix: upgrade sprig to v3.2.2 (#6876)

<details><summary><h3>Contributors</h3></summary>

* AdamKorcz
* Alex Collins
* Andy
* Arthur Sudre
* BOOK
* Basanth Jenu H B
* Benny Cornelissen
* Bob Haddleton
* Denis Melnik
* Dillen Padhiar
* Dimas Yudha P
* Dominik Deren
* Fabrice Jammes
* FengyunPan2
* Flaviu Vadan
* Francisco Barón
* Gammal-Skalbagge
* Guillaume Fillon
* Hong Wang
* Isitha Subasinghe
* Iven
* J.P. Zivalich
* JM" (Jason Meridth)
* Jannik Bertram
* Jesse Suen
* Jonathan
* Josh Preuss
* Joshua Carp
* Joyce Piscos
* Julien Duchesne
* Kamil Rokosz
* Ken Kaizu
* Kyle Hanks
* Markus Lippert
* Mathew Wicks
* Micah Beeman
* Michael Crenshaw
* Michael Weibel
* Miroslav Tomasik
* NextNiclas
* Nico Mandery
* Nicoló Lino
* Niklas Hansson
* Nityananda Gohain
* Peixuan Ding
* Peter Evers
* Rob Herley
* Roel van den Berg
* SalvadorC
* Saravanan Balasubramanian
* Sebastiaan Tammer
* Serhat
* Siebjee
* Simon Behar
* Song Juchao
* Takumi Sue
* Tianchu Zhao
* Ting Yuan
* Tino Schröter
* Tom Meadows
* Valér Orlovský
* William Reed
* William Van Hevelingen
* Yuan (Bob) Gong
* Yuan Tang
* Zadkiel
* Ziv Levi
* cod-r
* dependabot[bot]
* github-actions[bot]
* icecoffee531
* jacopo gobbi
* jhoenger
* jwjs36987
* kennytrytek
* khyer
* kostas-theo
* leonharetd
* momom-i
* roofurmston
* smile-luobin
* toohsk
* ybyang
* zorulo
* 大雄
* 阿拉斯加大闸蟹

</details>

## v3.2.11 (2022-05-03)

Full Changelog: [v3.2.10...v3.2.11](https://github.com/argoproj/argo-workflows/compare/v3.2.10...v3.2.11)

### Selected Changes

* [8faf269a7](https://github.com/argoproj/argo-workflows/commit/8faf269a795c0c9cc251152f9e4db4cd49234e52) fix: Remove binaries from Windows image. Fixes #8417 (#8420)

<details><summary><h3>Contributors</h3></summary>

* Markus Lippert

</details>

## v3.2.10 (2022-05-03)

Full Changelog: [v3.2.9...v3.2.10](https://github.com/argoproj/argo-workflows/compare/v3.2.9...v3.2.10)

### Selected Changes

* [877216e21](https://github.com/argoproj/argo-workflows/commit/877216e2159f07bfb27aa1991aa249bc2e9a250c) fix: Added artifact Content-Security-Policy (#8585)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins

</details>

## v3.2.9 (2022-03-02)

Full Changelog: [v3.2.8...v3.2.9](https://github.com/argoproj/argo-workflows/compare/v3.2.8...v3.2.9)

### Selected Changes

* [ce91d7b1d](https://github.com/argoproj/argo-workflows/commit/ce91d7b1d0115d5c73f6472dca03ddf5cc2c98f4) fix(controller): fix pod stuck in running when using podSpecPatch and emissary (#7407)
* [f9268c9a7](https://github.com/argoproj/argo-workflows/commit/f9268c9a7fca807d7759348ea623e85c67b552b0) fix: e2e
* [f581d1920](https://github.com/argoproj/argo-workflows/commit/f581d1920fe9e29dc0318fe628eb5a6982d66d93) fix: panic on synchronization if workflow has mutex and semaphore (#8025)
* [192c6b6a4](https://github.com/argoproj/argo-workflows/commit/192c6b6a4a785fa310b782a4e62e59427ece3bd1) fix: Fix broken Windows build (#7933)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins
* Markus Lippert
* Saravanan Balasubramanian
* Yuan (Bob) Gong

</details>

## v3.2.8 (2022-02-04)

Full Changelog: [v3.2.7...v3.2.8](https://github.com/argoproj/argo-workflows/compare/v3.2.7...v3.2.8)

### Selected Changes

* [8de5416ac](https://github.com/argoproj/argo-workflows/commit/8de5416ac6b8f5640a8603e374d99a18a04b5c8d) fix: Missed workflow should not trigger if Forbidden Concurreny with no StartingDeadlineSeconds (#7746)

<details><summary><h3>Contributors</h3></summary>

* Saravanan Balasubramanian

</details>

## v3.2.7 (2022-01-27)

Full Changelog: [v3.2.6...v3.2.7](https://github.com/argoproj/argo-workflows/compare/v3.2.6...v3.2.7)

### Selected Changes

* [342e44a28](https://github.com/argoproj/argo-workflows/commit/342e44a28e09a5b062745aa8cbea72339b1217b9) fix: Match cli display pod names with k8s. Fixes #7646 (#7653)
* [3429b1617](https://github.com/argoproj/argo-workflows/commit/3429b161783ae6d68ebd580c8c02590c6795abac) fix: Retry with DAG. Fixes #7617 (#7652)
* [7a3b766d4](https://github.com/argoproj/argo-workflows/commit/7a3b766d4a8df693c7fcff867423d56f5658801e) fix: Support artifact ref from tmpl in UI. Fixes #7587 (#7591)
* [e7a628cca](https://github.com/argoproj/argo-workflows/commit/e7a628ccadf50f8a907c4f22a7c8de8cede838a6) fix: Support inputs for inline DAG template. Fixes #7432 (#7439)
* [3f889c484](https://github.com/argoproj/argo-workflows/commit/3f889c484fd50c4e1385c1b81c49d3d7904dc37c) fix: Fix inconsistent ordering of workflows with the list command. Fixes #7581 (#7594)
* [77499bd38](https://github.com/argoproj/argo-workflows/commit/77499bd38308545a21d1e8f9a671b2d19001684d) fix: fix nil point about Outputs.ExitCode. Fixes #7458 (#7459)
* [74ed83a28](https://github.com/argoproj/argo-workflows/commit/74ed83a287b72e45cd9c560d3278cec0c621ee27) fix: Global param value incorrectly overridden when loading from configmaps (#7515)
* [db58583d2](https://github.com/argoproj/argo-workflows/commit/db58583d297d23bc40364150576ef17a86b2c914) fix: only aggregates output from successful nodes (#7517)
* [38fdf4c44](https://github.com/argoproj/argo-workflows/commit/38fdf4c44d78f9c388ee5e0f71e7edf97f81f364) fix: out of range in MustUnmarshal (#7485)
* [e69f2d790](https://github.com/argoproj/argo-workflows/commit/e69f2d7902d3e28e863d72cb81b0e65e55f8fb6e) fix: Support terminating with `templateRef`. Fixes #7214 (#7657)

<details><summary><h3>Contributors</h3></summary>

* AdamKorcz
* Alex Collins
* Dillen Padhiar
* FengyunPan2
* J.P. Zivalich
* Peixuan Ding
* Yuan Tang

</details>

## v3.2.6 (2021-12-17)

Full Changelog: [v3.2.5...v3.2.6](https://github.com/argoproj/argo-workflows/compare/v3.2.5...v3.2.6)

### Selected Changes

* [2a9fb7067](https://github.com/argoproj/argo-workflows/commit/2a9fb706714744eff7f70dbf56703bcc67ea67e0) Revert "fix(controller): default volume/mount to emissary (#7125)"

<details><summary><h3>Contributors</h3></summary>

* Alex Collins

</details>

## v3.2.5 (2021-12-15)

Full Changelog: [v3.2.4...v3.2.5](https://github.com/argoproj/argo-workflows/compare/v3.2.4...v3.2.5)

### Selected Changes

* [fc4c3d51e](https://github.com/argoproj/argo-workflows/commit/fc4c3d51e93858c2119124bbb3cb2ba1c35debcb) fix: lint
* [09ac50b7d](https://github.com/argoproj/argo-workflows/commit/09ac50b7dc09a8f8497897254252760739363d0d) fix: lint
* [c48269fe6](https://github.com/argoproj/argo-workflows/commit/c48269fe678ae74092afda498da2f897ba22d177) fix: codegen
* [e653e4f2f](https://github.com/argoproj/argo-workflows/commit/e653e4f2f3652a95e8584488e657838f04d01f7e) fix: e2e test and codegen
* [970bcc041](https://github.com/argoproj/argo-workflows/commit/970bcc04179a98cfcce31977aeb34fbf1a68ebaf) fix: e2e testcase
* [fbb2edb03](https://github.com/argoproj/argo-workflows/commit/fbb2edb03494160c28a83d2a04546323e119caff) fix: unit test
* [7933f9579](https://github.com/argoproj/argo-workflows/commit/7933f9579680de570f481004d734bd36ea0ca69e) fix: makefile and common variable
* [0eec0f0d5](https://github.com/argoproj/argo-workflows/commit/0eec0f0d5495a0d5174e74e6cac87cc068eb5295) fix: added check for initContainer name in workflow template (#7411)
* [7c2427005](https://github.com/argoproj/argo-workflows/commit/7c2427005cb69f351b081a6c546bda7978ae665f) fix: Skip missed executions if CronWorkflow schedule is changed. Fixes #7182 (#7353)
* [48e7906d5](https://github.com/argoproj/argo-workflows/commit/48e7906d503831385261dcccd4e1c8695c895895) fix: argument of PodName function (fixes #7315) (#7316)
* [3911d0915](https://github.com/argoproj/argo-workflows/commit/3911d091530fc743585c72c7366db3a9c7932bfd) fix: improve error message for ListArchivedWorkflows (#7345)
* [5a472dd39](https://github.com/argoproj/argo-workflows/commit/5a472dd39faaf57a8b4f1e2d748d5167b66d07a0) fix: cannot access HTTP template's outputs (#7200)
* [a85458e86](https://github.com/argoproj/argo-workflows/commit/a85458e86fa80f931f1a0a42230f843d26d84fad) fix(ui): Fix events error. Fixes #7320 (#7321)
* [6bcedb18b](https://github.com/argoproj/argo-workflows/commit/6bcedb18be40005f8f81eedf923e890a33e9d11e) fix: Validate the type of configmap before loading parameters. Fixes #7312 (#7314)
* [a142ac234](https://github.com/argoproj/argo-workflows/commit/a142ac234ee7a4e789ac626636837c00b296be23) fix: Handle the panic in operate function (#7262)
* [34f3d13e7](https://github.com/argoproj/argo-workflows/commit/34f3d13e7e603198548937beb8df7e84f022b918) fix: pod name shown in log when pod deletion (#7301)
* [06e5950b8](https://github.com/argoproj/argo-workflows/commit/06e5950b8f3fbafdfeb7d45a603caf03096f958e) fix: Use default value for empty env vars (#7297)
* [2f96c464a](https://github.com/argoproj/argo-workflows/commit/2f96c464a3098b34dfd94c44cc629c881ea3d33f) fix: Fix `argo auth token`. Fixes #7175 (#7186)
* [f8f93a6b1](https://github.com/argoproj/argo-workflows/commit/f8f93a6b16e4a1ec17060ef916ea6bd2e8cf80a4) fix: refactor/fix pod GC. Fixes #7159 (#7176)
* [728a1ff67](https://github.com/argoproj/argo-workflows/commit/728a1ff67364986cdfe2146dc3179d9705ee26ad) fix: Relative submodules in git artifacts. Fixes #7141 (#7162)
* [274c5f990](https://github.com/argoproj/argo-workflows/commit/274c5f990dd16b8f2523706549b07c40d60a3fab) fix: Reorder CI checks so required checks run first (#7142)
* [49b3f0cb2](https://github.com/argoproj/argo-workflows/commit/49b3f0cb2733dec438d8340f439467b7661b8bc2) fix(controller): default volume/mount to emissary (#7125)
* [f5f6899f5](https://github.com/argoproj/argo-workflows/commit/f5f6899f531126a18f5f42201156c995630fdf1b) fix: Add pod name format annotation. Fixes #6962 and #6989 (#6982)
* [30e34ada8](https://github.com/argoproj/argo-workflows/commit/30e34ada8cab77c56e3917144a29b96fb070a06d) fix: prevent bad commit messages, fix broken builds (#7086)
* [926108028](https://github.com/argoproj/argo-workflows/commit/926108028cea2e0ef305c24c86b9e685a0ac9c5e) fix: Format issue on WorkflowEventBinding parameters. Fixes #7042 (#7087)
* [a0ac28893](https://github.com/argoproj/argo-workflows/commit/a0ac28893b63a73f6d875b4087fc04f420595815) fix: add outputs.parameters scope to script/containerSet templates. Fixes #6439 (#7045)
* [cae69e62b](https://github.com/argoproj/argo-workflows/commit/cae69e62b37a6f8256a9cab53d793fc5102ebfe4) fix: Unit test TestNewOperation order of pipeline execution maybe different to order of submit (#7069)
* [94fe92f12](https://github.com/argoproj/argo-workflows/commit/94fe92f12a21af225c0d44fa7b20a6b335edaadf) fix: OAuth2 callback with self-signed Root CA. Fixes #6793 (#6978)
* [fbb51ac20](https://github.com/argoproj/argo-workflows/commit/fbb51ac2002b896ea3320802b814adb4c3d0d5e4) fix: multi-steps workflow (#6957)
* [6b7e074f1](https://github.com/argoproj/argo-workflows/commit/6b7e074f149085f9fc2da48656777301e87e8aae) fix(docs): fix data transformation example (#6901)
* [24ffd36bf](https://github.com/argoproj/argo-workflows/commit/24ffd36bfc417fe382a1e015b0ec4d89b2a12280) fix: Allow self-signed Root CA for SSO. Fixes #6793 (#6961)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins
* Arthur Sudre
* BOOK
* Dillen Padhiar
* Dominik Deren
* J.P. Zivalich
* Jonathan
* NextNiclas
* Peter Evers
* Saravanan Balasubramanian
* Sebastiaan Tammer
* Simon Behar
* Takumi Sue
* Tianchu Zhao
* Valér Orlovský
* William Van Hevelingen
* Yuan Tang
* Ziv Levi

</details>

## v3.2.4 (2021-11-17)

Full Changelog: [v3.2.3...v3.2.4](https://github.com/argoproj/argo-workflows/compare/v3.2.3...v3.2.4)

### Selected Changes

* [bf72557b5](https://github.com/argoproj/argo-workflows/commit/bf72557b53792cf23ce3ee4cbec779bb7e420ba8) fix: add gh ecdsa and ed25519 to known hosts (#7226)
* [ee6939048](https://github.com/argoproj/argo-workflows/commit/ee6939048ab2b15103ece77b0d74afd6f0d3e691) fix: Fix ANSI color sequences escaping (#7211)
* [02b4c31c4](https://github.com/argoproj/argo-workflows/commit/02b4c31c41e3b509188057d31735b1f3684488f5) fix: ci sleep command syntax for macOS 12 (#7203)
* [e65d9d4a9](https://github.com/argoproj/argo-workflows/commit/e65d9d4a983670c70707d283573d06a68971f6b4) fix: allow wf templates without parameter values (Fixes #6044) (#7124)
* [7ea35fa1f](https://github.com/argoproj/argo-workflows/commit/7ea35fa1fd0fa739f16b5978a52a521fafb90d4e) fix(ui): Correctly show zero-state when CRDs not installed. Fixes #7001 (#7169)
* [bdcca4e17](https://github.com/argoproj/argo-workflows/commit/bdcca4e175ee71e402e567d857209f7ddce79d9a) fix: Return error when YAML submission is invalid (#7135)
* [a4390dd9a](https://github.com/argoproj/argo-workflows/commit/a4390dd9a9bbd1280774fe10cf455d655a4ea873) fix: Respect template.HTTP.timeoutSeconds (#7136)
* [c1553dfd7](https://github.com/argoproj/argo-workflows/commit/c1553dfd73e3734b6dbdb4fdb5828df1e6fff792) fix: typo in node-field-selector.md (#7116)
* [508027b35](https://github.com/argoproj/argo-workflows/commit/508027b3521ef2b51293aa1dc58a911c753d148c) fix: Daemon step in running state, but dependents don't start (#7107)
* [ccc8d839c](https://github.com/argoproj/argo-workflows/commit/ccc8d839c2da3c561bb7f5c078cd26c17ce9a9c5) fix: Ensure HTTP reconciliation occurs for onExit nodes (#7084)
* [00f953286](https://github.com/argoproj/argo-workflows/commit/00f953286f4e3a120b5dff4dc1dbd32adf1c7237) fix: Ensure HTTP templates have children assigned (#7082)
* [9b4dd1e83](https://github.com/argoproj/argo-workflows/commit/9b4dd1e83a3362b8f561e380566a7af3ab68ba8d) fix(ui): Correct HTTP connection in pipeline view (#7077)
* [f43d8b01a](https://github.com/argoproj/argo-workflows/commit/f43d8b01a752829e5c6208215b767e3ab68c9dc2) fix: Memozie for Step and DAG level (#7028)
* [7256dace6](https://github.com/argoproj/argo-workflows/commit/7256dace6c1bb6544f7a0e79220b993c32bc3daf) fix: Support RFC3339 in creationTimeStamp. Fixes #6906 (#7044)
* [0837d0c6a](https://github.com/argoproj/argo-workflows/commit/0837d0c6afc06798820a8b41f0acad35aac11143) fix(controller): use correct pod.name in retry/podspecpatch scenario. Fixes #7007 (#7008)
* [09d07111e](https://github.com/argoproj/argo-workflows/commit/09d07111e21ce9d01469315cc3a67ff10ed05617) fix(typo): correct typo in event-dispatch error log (#6688)
* [26afd8ec9](https://github.com/argoproj/argo-workflows/commit/26afd8ec9db0cfc98a4cee9b7bcd3a211c2119c4) fix: OAuth2 callback with self-signed Root CA. Fixes #6793 (#6978)
* [d9eafeee1](https://github.com/argoproj/argo-workflows/commit/d9eafeee1ce309726b32b3736086da1529487fa8) fix: Allow self-signed Root CA for SSO. Fixes #6793 (#6961)
* [46f88f423](https://github.com/argoproj/argo-workflows/commit/46f88f4230b546863f83ccf56b94697e39ab0e11) fix: response on canceled workflow action (#6859) (#6967)
* [32ecc4654](https://github.com/argoproj/argo-workflows/commit/32ecc4654cda8e84d6bb7a696675e14da8665747) fix: Unreachable code in util/tls/tls.go. Fixes  #6950 (#6960)
* [2fbeb80f0](https://github.com/argoproj/argo-workflows/commit/2fbeb80f0c320805de72c42ea5b106ab31f560a8) fix(executor): add test for non-root user creating a script (#6905)
* [15e9ba84d](https://github.com/argoproj/argo-workflows/commit/15e9ba84d1b783fe26ed0e507b1d5a868b43ee0e) fix: Skip empty withParam tasks. Fixes #6834 (#6912)
* [d31860cd1](https://github.com/argoproj/argo-workflows/commit/d31860cd1d20c07ce28b0e7035fbf210019fa38a) fix: Parameter with Value and Default (#6887)
* [ba4ffdf8d](https://github.com/argoproj/argo-workflows/commit/ba4ffdf8d1948302942c9860a1d2fea8f8d6db8e) fix(ui): fixed width button (#6883)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins
* Bob Haddleton
* Guillaume Fillon
* Iven
* Kyle Hanks
* Mathew Wicks
* Miroslav Tomasik
* NextNiclas
* Rob Herley
* SalvadorC
* Saravanan Balasubramanian
* Simon Behar
* Tianchu Zhao
* Zadkiel
* Ziv Levi
* kennytrytek

</details>

## v3.2.3 (2021-10-26)

Full Changelog: [v3.2.2...v3.2.3](https://github.com/argoproj/argo-workflows/compare/v3.2.2...v3.2.3)

### Selected Changes

* [e5dc961b7](https://github.com/argoproj/argo-workflows/commit/e5dc961b7846efe0fe36ab3a0964180eaedd2672) fix: Precedence of ContainerRuntimeExecutor and ContainerRuntimeExecutors (#7056)
* [3f14c68e1](https://github.com/argoproj/argo-workflows/commit/3f14c68e166a6fbb9bc0044ead5ad4e5b424aab9)  feat: Bring Java client into core.  (#7026)
* [48e1aa974](https://github.com/argoproj/argo-workflows/commit/48e1aa9743b523abe6d60902e3aa8546edcd221b) fix: Minor corrections to Swagger/JSON schema (#7027)
* [10f5db67e](https://github.com/argoproj/argo-workflows/commit/10f5db67ec29c948dfa82d1f521352e0e7eb4bda) fix(controller): fix bugs in processing retry node output parameters. Fixes #6948 (#6956)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins
* Saravanan Balasubramanian
* smile-luobin

</details>

## v3.2.2 (2021-10-21)

Full Changelog: [v3.2.1...v3.2.2](https://github.com/argoproj/argo-workflows/compare/v3.2.1...v3.2.2)

### Selected Changes

* [8897fff15](https://github.com/argoproj/argo-workflows/commit/8897fff15776f31fbd7f65bbee4f93b2101110f7) fix: Restore default pod name version to v1 (#6998)

<details><summary><h3>Contributors</h3></summary>

* J.P. Zivalich

</details>

## v3.2.1 (2021-10-19)

Full Changelog: [v3.2.0...v3.2.1](https://github.com/argoproj/argo-workflows/compare/v3.2.0...v3.2.1)

### Selected Changes

* [74182fb90](https://github.com/argoproj/argo-workflows/commit/74182fb9017e0f05c0fa6afd32196a1988423deb) lint
* [7cdbee05c](https://github.com/argoproj/argo-workflows/commit/7cdbee05c42e5d73e375bcd5d3db264fa6bc0d4b) fix(ui): Change pod names to new format. Fixes #6865 (#6925)
* [5df91b289](https://github.com/argoproj/argo-workflows/commit/5df91b289758e2f4953919621a207129a9418226) fix: BASE_HREF ignore (#6926)
* [d04aabf2c](https://github.com/argoproj/argo-workflows/commit/d04aabf2c3094db557c7edb1b342dcce54ada2c7) fix(controller): Fix getPodByNode, TestGetPodByNode. Fixes #6458 (#6897)
* [72446bf3b](https://github.com/argoproj/argo-workflows/commit/72446bf3bad0858a60e8269f5f476192071229e5) fix: do not delete expr tag tmpl values. Fixes #6909 (#6921)
* [2922a2a9d](https://github.com/argoproj/argo-workflows/commit/2922a2a9d8506ef2e84e2b1d3172168ae7ac6aeb) fix: Resource requests on init/wait containers. Fixes #6809 (#6879)
* [84623a4d6](https://github.com/argoproj/argo-workflows/commit/84623a4d687b962898bcc718bdd98682367586c1) fix: upgrade sprig to v3.2.2 (#6876)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins
* Hong Wang
* J.P. Zivalich
* Micah Beeman
* Saravanan Balasubramanian
* zorulo

</details>

## v3.2.0 (2021-10-05)

Full Changelog: [v3.2.0-rc6...v3.2.0](https://github.com/argoproj/argo-workflows/compare/v3.2.0-rc6...v3.2.0)

### Selected Changes

<details><summary><h3>Contributors</h3></summary>

</details>

## v3.2.0-rc6 (2021-10-05)

Full Changelog: [v3.2.0-rc5...v3.2.0-rc6](https://github.com/argoproj/argo-workflows/compare/v3.2.0-rc5...v3.2.0-rc6)

### Selected Changes

* [994ff7454](https://github.com/argoproj/argo-workflows/commit/994ff7454b32730a50b13bcbf14196b1f6f404a6) fix(UI): use default params on template submit form (#6858)
* [47d713bbb](https://github.com/argoproj/argo-workflows/commit/47d713bbba9ac3a210c0b3c812f7e05522d8e7b4) fix: Add OIDC issuer alias. Fixes #6759 (#6831)
* [11a8c38bb](https://github.com/argoproj/argo-workflows/commit/11a8c38bbe77dcc5f85a60b4f7c298770a03aafc) fix(exec): Failed to load http artifact. Fixes #6825 (#6826)
* [147730d49](https://github.com/argoproj/argo-workflows/commit/147730d49090348e09027182dcd3339654993f41) fix(docs): cron backfill example (#6833)
* [4f4157bb9](https://github.com/argoproj/argo-workflows/commit/4f4157bb932fd277291851fb86ffcb9217c8522e) fix: add HTTP genre and sort (#6798)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins
* Niels ten Boom
* Raymond Wong
* Saravanan Balasubramanian
* Sean Trantalis
* Shea Sullivan
* Tianchu Zhao
* asimhon
* github-actions[bot]
* kennytrytek
* smile-luobin

</details>

## v3.2.0-rc5 (2021-09-29)

Full Changelog: [v3.2.0-rc4...v3.2.0-rc5](https://github.com/argoproj/argo-workflows/compare/v3.2.0-rc4...v3.2.0-rc5)

### Selected Changes

* [87a57328e](https://github.com/argoproj/argo-workflows/commit/87a57328e72d794b29481b7377c49fd58b2b9480) feat: implement WatchEvents for argoKubeWorkflowServiceClient. Fixes #6173 (#6816)
* [543366fab](https://github.com/argoproj/argo-workflows/commit/543366fab79ed79c36f172aba8a288ce73d6f675) fix(apiclient): remove default client in facade. Fixes #6733 (#6800)
* [2c3ac705a](https://github.com/argoproj/argo-workflows/commit/2c3ac705a20ae1cf38d0eb30b15826f2946857ac) fix: Missing duration metrics if controller restart (#6815)
* [a87e94b62](https://github.com/argoproj/argo-workflows/commit/a87e94b620784c93f13543de83cd784e20fad595) fix: Fix expression template random errors. Fixes #6673 (#6786)
* [254c73084](https://github.com/argoproj/argo-workflows/commit/254c73084da5f02a5edfea42d4671ae97703592f) fix: Fix bugs, unable to resolve tasks aggregated outputs in dag outputs. Fixes #6684 (#6692)
* [965309925](https://github.com/argoproj/argo-workflows/commit/96530992502bfd126fd7dcb0a704d3c36c166bd1) fix: remove windows UNC paths from wait/init containers. Fixes #6583 (#6704)
* [ffb0db711](https://github.com/argoproj/argo-workflows/commit/ffb0db711b611633e30a6586b716af02c37a9de6) fix: Missing duration metrics if controller restart (#6815)
* [81bfa21eb](https://github.com/argoproj/argo-workflows/commit/81bfa21eb56cdba45b871f9af577a9dc72aa69f2) feat(controller): add workflow level archivedLogs. Fixes #6663 (#6802)
* [6995d682d](https://github.com/argoproj/argo-workflows/commit/6995d682dabbaac7e44e97f9a18480723932a882) fix: update outdated links for cli (#6791)
* [b35aabe86](https://github.com/argoproj/argo-workflows/commit/b35aabe86be9fa5db80299cebcfb29c32be21047) fix(lint): checking error for viper command flag binding (#6788)
* [96c562649](https://github.com/argoproj/argo-workflows/commit/96c5626497df9eedad062df9b8aaaaeea3561407) feat: Add env vars config for argo-server and workflow-controller (#6767)
* [7a7171f46](https://github.com/argoproj/argo-workflows/commit/7a7171f464e5f2f71526c3cdb63e854e28fd3c01) fix: Fix expression template random errors. Fixes #6673 (#6786)
* [067576ed7](https://github.com/argoproj/argo-workflows/commit/067576ed72750158efd034078ab8102b72438798) fix(controller): fix template archivelocation-archivelog behaviour (#6754)
* [d747fc5ea](https://github.com/argoproj/argo-workflows/commit/d747fc5ea985ad39324282e8410ca6397e05832f) fix(ui): workflow event binding typo (#6782)
* [9dc33f617](https://github.com/argoproj/argo-workflows/commit/9dc33f6172a3bc1e0fc0e64d9ed56ed92981c349) fix: Fix bugs, unable to resolve tasks aggregated outputs in dag outputs. Fixes #6684 (#6692)
* [954292d50](https://github.com/argoproj/argo-workflows/commit/954292d500b1a63c1c467f0d404b38e8b372f22e) fix(controller): TestPodExists unit test, add delay to wait for informer getting pod info (#6783)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins
* Andrey Velichkevich
* Anish Dangi
* Anthony Scott
* Julian Fleischer
* Julien Duchesne
* Niklas Hansson
* Philippe Richard
* Saravanan Balasubramanian
* Tianchu Zhao
* William Van Hevelingen
* Yuan Tang
* github-actions[bot]
* smile-luobin
* tooptoop4
* ygelfand

</details>

## v3.2.0-rc4 (2021-09-21)

Full Changelog: [v3.2.0-rc3...v3.2.0-rc4](https://github.com/argoproj/argo-workflows/compare/v3.2.0-rc3...v3.2.0-rc4)

### Selected Changes

* [710e82366](https://github.com/argoproj/argo-workflows/commit/710e82366dc3b0b17f5bf52004d2f72622de7781) fix: fix a typo in example file dag-conditional-artifacts.yaml (#6775)
* [b82884600](https://github.com/argoproj/argo-workflows/commit/b8288460052125641ff1b4e1bcc4ee03ecfe319b) feat: upgrade Argo Dataflow to v0.0.104 (#6749)
* [1a76e6581](https://github.com/argoproj/argo-workflows/commit/1a76e6581dd079bdcfc76be545b3f7dd1ba48105) fix(controller): TestPodExists unit test (#6763)
* [6875479db](https://github.com/argoproj/argo-workflows/commit/6875479db8c466c443acbc15a3fe04d8d6a8b1d2) fix: Daemond status stuck with Running (#6742)
* [e5b131a33](https://github.com/argoproj/argo-workflows/commit/e5b131a333afac0ed3444b70e2fe846b86dc63e1) feat: Add template node to pod name. Fixes #1319 (#6712)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins
* Brewster Malevich
* J.P. Zivalich
* Saravanan Balasubramanian
* Stephen Raghunath
* TCgogogo
* Tianchu Zhao
* github-actions[bot]
* yyyyyy888

</details>

## v3.2.0-rc3 (2021-09-14)

Full Changelog: [v3.2.0-rc2...v3.2.0-rc3](https://github.com/argoproj/argo-workflows/compare/v3.2.0-rc2...v3.2.0-rc3)

### Selected Changes

* [69e438426](https://github.com/argoproj/argo-workflows/commit/69e438426e4d116e2c9a1716651af7ef14864f04) fix: correct minor typos in docs (#6722)
* [ae5398698](https://github.com/argoproj/argo-workflows/commit/ae5398698afd3676ba180874987bfc6c3563b9a6) fix(executor): allow emptyRepo artifact input. Fixes #5857 (#6711)
* [e57249e64](https://github.com/argoproj/argo-workflows/commit/e57249e647ec15c859e1035d451c65ae76cc27b6) fix: remove windows UNC paths from wait/init containers. Fixes #6583 (#6704)
* [0b3f62cbe](https://github.com/argoproj/argo-workflows/commit/0b3f62cbe747aa82cff1419cf26db6007d1d1079) fix: kill sidecar timeout issue (#6700)
* [cf14cad41](https://github.com/argoproj/argo-workflows/commit/cf14cad41e1a8428cae8382398ee778892e63198) feat(ui): logsViewer use archived log if node finish and archived (#6708)
* [3ba7d5fd6](https://github.com/argoproj/argo-workflows/commit/3ba7d5fd64f5bab7c96b6b4ff65e488f8faa570e) fix(ui): undefined cron timestamp (#6713)
* [4c9c92292](https://github.com/argoproj/argo-workflows/commit/4c9c922924be2a299995fc06efbaef15c7fb0f84) fix: panic in prepareMetricScope (#6720)
* [d1299ec80](https://github.com/argoproj/argo-workflows/commit/d1299ec8073789af8c9b6281770f9236013d5acf) fix(executor): handle hdfs optional artifact at retriving hdfs file stat (#6703)
* [11657fe16](https://github.com/argoproj/argo-workflows/commit/11657fe169e31319da431d77ed3355ab2848401d) feat: Provide context to NewAPIClient (#6667)
* [a1cc0f557](https://github.com/argoproj/argo-workflows/commit/a1cc0f557c08c1206df89e39d2c286f02a6675de) feat: archivewf add name filter. Fixes #5824 (#6706)
* [1e31eb856](https://github.com/argoproj/argo-workflows/commit/1e31eb85655d2118f2e3c3edaa8886f923de4f5b) fix(ui): report phase button alignment (#6707)
* [d45395b6f](https://github.com/argoproj/argo-workflows/commit/d45395b6f3b0cc40444e98af921b9e80284b74e8) fix: run Snyk on UI. Fixes #6604 (#6651)
* [2e174bd4c](https://github.com/argoproj/argo-workflows/commit/2e174bd4c585ccf72e34c8f72703a0950a67460c) fix(ui): button margin (#6699)
* [4b5d7ecfd](https://github.com/argoproj/argo-workflows/commit/4b5d7ecfd1087f22002bc63658dc5ad3fe30927f) fix(emissary): strip trailing slash from artifact src before creating… (#6696)
* [28c8dc7a9](https://github.com/argoproj/argo-workflows/commit/28c8dc7a9054fdf90fd7f98e03f86923dc6e6d2a) feat: Support loading parameter values from configmaps (#6662)
* [9c76cc34c](https://github.com/argoproj/argo-workflows/commit/9c76cc34c7591f0113dea4e35b58b902d8386544) fix(executor): Retry `kubectl` on transient error (#6472)
* [929351267](https://github.com/argoproj/argo-workflows/commit/9293512674c21a2494c704978990cf89eb5ad8c0) fix(cli): Added validatation for StartAt, FinishedAt and ID (#6661)
* [a147f178d](https://github.com/argoproj/argo-workflows/commit/a147f178d9ddbe139551bf5636f73fb1af2e61f8) fix(controller): Set finishedAt for workflow with Daemon steps (#6650)
* [5522d4b4c](https://github.com/argoproj/argo-workflows/commit/5522d4b4c6f3b2de68956998c877b2c596e158af) fix: Do not index complete workflow semaphores (#6639)
* [2ac3c48d3](https://github.com/argoproj/argo-workflows/commit/2ac3c48d33415b804067b07a13185b06d3b416bc) fix: `argo node set` panic: index out of range and correct docs (#6653)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins
* Anish Dangi
* Damian Czaja
* Elliot Maincourt
* Jesse Suen
* Joshua Carp
* Saravanan Balasubramanian
* Tianchu Zhao
* Tim Gallant
* William Van Hevelingen
* Yuan Tang
* github-actions[bot]
* 大雄

</details>

## v3.2.0-rc2 (2021-09-01)

Full Changelog: [v3.2.0-rc1...v3.2.0-rc2](https://github.com/argoproj/argo-workflows/compare/v3.2.0-rc1...v3.2.0-rc2)

### Selected Changes

* [6d46fd9f8](https://github.com/argoproj/argo-workflows/commit/6d46fd9f881a337b5b3d34d62e71d9b56ba05b1a) feat(controller): Add a shared index informer for ConfigMaps (#6644)
* [91abb47db](https://github.com/argoproj/argo-workflows/commit/91abb47db3c8ad20fac80914f1961842bc64a0b9) feat: Upgrade dataflow to v0.0.98 (#6637)
* [d8b90f2b8](https://github.com/argoproj/argo-workflows/commit/d8b90f2b89472f8dce9c134aeccd7cb70ee3b87b) fix: Fixed typo in clusterrole (#6626)
* [51307e11e](https://github.com/argoproj/argo-workflows/commit/51307e11ede253be6231dd007565fcc98ccc564b) fix: Upgrade Dataflow to v0.0.96 (#6622)
* [f1c188f3e](https://github.com/argoproj/argo-workflows/commit/f1c188f3eba61421a37dfcaea68e7e9f61f5842a) fix: Argo Workflow specs link to not go to raw content (#6624)
* [07e29263a](https://github.com/argoproj/argo-workflows/commit/07e29263a6254b9caf7a47e2761cba3e1d39c7b4)  docs: Add slack exit handler example. Resolves #4152  (#6612)
* [29cf73548](https://github.com/argoproj/argo-workflows/commit/29cf73548d7246433cb1d835f25f34ab73389fe4) fix(controller): Initialize throttler during starting workflow-controller. Fixes: #6599 (#6608)
* [a394a91f5](https://github.com/argoproj/argo-workflows/commit/a394a91f59bc3086e0538265c0d9d399a43110c6) fix: manifests/quick-start/sso for running locally PROFILE=sso (#6503)
* [8678f007e](https://github.com/argoproj/argo-workflows/commit/8678f007e86ffa615e6ca90c52c7ca4d1e458b08) fix: Fix `gosec` warnings, disable pprof by default. Fixes #6594 (#6596)
* [3aac377e2](https://github.com/argoproj/argo-workflows/commit/3aac377e223f1a6bad05ec28404c89e435e47687) fix!: Enable authentication by default on Argo Server `/metrics` endpoint. Fixes #6592 (#6595)
* [656639666](https://github.com/argoproj/argo-workflows/commit/6566396666163198f2520c9a0790b01ada3863fd) fix(executor): Disambiguate PNS executor initialization log (#6582)
* [d6f5acb40](https://github.com/argoproj/argo-workflows/commit/d6f5acb407ddf2d6f7afbe3e380eda5a2908dcbd) fix: Fix unit test with missing createRunningPods() (#6585)
* [b0e050e54](https://github.com/argoproj/argo-workflows/commit/b0e050e54a96a1c46b279a37b7daf43b2942f791) feat: upgrade argo-dataflow to v0.0.90 (#6563)
* [30340c427](https://github.com/argoproj/argo-workflows/commit/30340c42785fcff1e864b2078c37139dc13bbfd7) fix(gcs): backoff bool should be false if error is transient (#6577)
* [1e34cec88](https://github.com/argoproj/argo-workflows/commit/1e34cec88e4fd1f65da923139efbf8fb38c97772) feat(artifact): Allow to set bucket logging for OSS artifact driver (#6554)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins
* Andrey Melnikov
* Antoine Dao
* Curtis Vogt
* J.P. Zivalich
* Josh Soref
* Luciano Sá
* Michael Pöllath
* Saravanan Balasubramanian
* Siebjee
* Simon Behar
* Tetsuya Shiota
* Yuan Tang
* github-actions[bot]
* smile-luobin

</details>

## v3.2.0-rc1 (2021-08-19)

Full Changelog: [v3.1.15...v3.2.0-rc1](https://github.com/argoproj/argo-workflows/compare/v3.1.15...v3.2.0-rc1)

### Selected Changes

* [3595ac59c](https://github.com/argoproj/argo-workflows/commit/3595ac59cefe63256bbac38bca27fb5cacee93f9) feat: Adding SSO support for Okta. Fixes #6165 (#6572)
* [f1cf7ee03](https://github.com/argoproj/argo-workflows/commit/f1cf7ee03c741ecdc9698123a3fae4e5ccafbd16) fix: Panic in getTemplateOutputsFromScope (#6560)
* [64fbf6955](https://github.com/argoproj/argo-workflows/commit/64fbf6955840b1bde28d36db106866da04047d4f) fix(executor/pns): panic of pidFileHandles concurrent writes (#6569)
* [ae7eeeb50](https://github.com/argoproj/argo-workflows/commit/ae7eeeb50dd0b7640913e7b30d1fe612c7e0ee4c) fix: Fix `x509: certificate signed by unknown authority` error (#6566)
* [205d233cd](https://github.com/argoproj/argo-workflows/commit/205d233cd8e85af24e451d6268af32e928aeb47c) fix(executor/docker): fix failed to wait for main container to complete: timed out waiting for the condition: container does not exist (#6561)
* [d41c03702](https://github.com/argoproj/argo-workflows/commit/d41c037027e062a149ce821dd377fb6b52269335) feat: S3 Encryption At Rest (#6549)
* [478d79469](https://github.com/argoproj/argo-workflows/commit/478d794693b3a965e3ba587da2c67e5e1effa418) fix: Generate TLS Certificates on startup and only keep in memory (#6540)
* [f711ce4d5](https://github.com/argoproj/argo-workflows/commit/f711ce4d5352b025f366f8e81ebbe9e457cc9054) fix: use golangci-lint v1.37.0 to support apple M1 (#6548)
* [37395d681](https://github.com/argoproj/argo-workflows/commit/37395d6818ba151213a1bb8338356cf553c2404a) fix: replace docker.io with quay.io to avoid the image pull limit (#6539)
* [a1a8d4421](https://github.com/argoproj/argo-workflows/commit/a1a8d4421e3b7e8c6bcd2677e7862ec6f3aed1cc) fix: argo-sever mistype (#6543)
* [a57b3ad9e](https://github.com/argoproj/argo-workflows/commit/a57b3ad9ed2afbcd3f22e912b252dd451d9c7ebc) feat: Show Argo Dataflow pipelines in the UI (#5742)
* [dc4f0a172](https://github.com/argoproj/argo-workflows/commit/dc4f0a172d6992cd34749d858bb0c402172c8eef) fix: use execWf when setting PodMetadata (#6512)
* [903ce68ff](https://github.com/argoproj/argo-workflows/commit/903ce68ffa01400a7b57b2604091482a27ca64d4) fix: Fix the Status update for node with synchronization lock (#6525)
* [a38460342](https://github.com/argoproj/argo-workflows/commit/a38460342472b0515017d5a2ab2cbc6536b5592e) fix: Upgrade pkg to v0.10.1. Fixes #6521 (#6523)
* [0670f652c](https://github.com/argoproj/argo-workflows/commit/0670f652cd7ca5500aa77c682bb8b380bb4c79d3) fix(controller): fix tasket warning in Non-HTTP Template scanerio (#6518)
* [32970f4cd](https://github.com/argoproj/argo-workflows/commit/32970f4cd15923b62d750863c28270bc283071b6) fix: PROFILE=SSO to PROFILE=sso for case-sensitive filesystem (#6502)
* [3d5ac9b2b](https://github.com/argoproj/argo-workflows/commit/3d5ac9b2be71937e86eee1d71a4eefa294b27293) fix(controller): Fix panic in addParamToGlobalScope (#6485)
* [d1d96b0a6](https://github.com/argoproj/argo-workflows/commit/d1d96b0a6e8f045715b83a55f1aad056eb76bd96) feat(ui): use dl tag instead of p tag in user-info ui (#6505)
* [5b8f7977a](https://github.com/argoproj/argo-workflows/commit/5b8f7977a86a43061dca9ea916d32c02e23bd7f5) Add link to latest release in installation.md (#6509)
* [24bb1b77a](https://github.com/argoproj/argo-workflows/commit/24bb1b77a1b5cd2f78251aca26d007c7d75b8993) fix(executor/docker): re-revert -- fix random errors with message "No such container:path". Fixes #6352 (#6508)
* [e2e822dd5](https://github.com/argoproj/argo-workflows/commit/e2e822dd59e3ad62d978cdce0efa5ce7a4a273e2) fix: Remove client private key from client auth REST config (#6506)
* [a3fd704a1](https://github.com/argoproj/argo-workflows/commit/a3fd704a1715900f2144c0362e562f75f1524126) Revert "fix(executor/docker): fix random errors with message "No such container:path". Fixes #6352 (#6483)"
* [a105b137c](https://github.com/argoproj/argo-workflows/commit/a105b137c97e5aea852c6db6e77997ca3713cb08) fix(controller): Delete the PVCs in workflowtemplateRef (#6493)
* [3373dc512](https://github.com/argoproj/argo-workflows/commit/3373dc512804ae51d09ade02be53c597aead3c3f) feat: Annotate pod events with workflow name and UID (#6455)
* [e4a53d4bf](https://github.com/argoproj/argo-workflows/commit/e4a53d4bf021fd4dce1374bb7fd4320d733e57ba) fix(executor/docker): fix random errors with message "No such container:path". Fixes #6352 (#6483)
* [2a2ecc916](https://github.com/argoproj/argo-workflows/commit/2a2ecc916925642fd8cb1efd026588e6828f82e1) fix(controller): JSON-unmarshal marshaled expression template before evaluating (#6285)
* [ec9641531](https://github.com/argoproj/argo-workflows/commit/ec9641531c8283a4e6fcd684c8aecce92c6e14b7) feat(controller): Inline templates. Closes #5105 (#5749)
* [7ef0f4750](https://github.com/argoproj/argo-workflows/commit/7ef0f4750d7da4bb326fb0dab25f176db412993b) fix: Consider onExit children of Retry nodes (#6451)
* [7f2c58972](https://github.com/argoproj/argo-workflows/commit/7f2c58972177c5b7cfdfb6bc8d9ba4189a9f45d0) feat!: Upgrade to Golang 1.16. Fixes #5563 (#6471)
* [5fde8fa72](https://github.com/argoproj/argo-workflows/commit/5fde8fa72f2e5b0bcd7cfb048fd1eb9e24b6a950) fix: Exit template shouldn't fail with max parallelism reached (#6456)
* [c5d2461cf](https://github.com/argoproj/argo-workflows/commit/c5d2461cf5f9cd7569bc07c8a7cfde7e4c86e5a4) fix(controller): fix retry on different hosts (#6429)
* [0f6f36270](https://github.com/argoproj/argo-workflows/commit/0f6f362704e0c124a127438ced5df26e6c91a76b) fix(server): Fix nil pointer error when getting artifacts from a step without artifacts (#6465)
* [903415249](https://github.com/argoproj/argo-workflows/commit/90341524935287c7db30f34132c2a1aa4f1ea170) feat(server): Support OIDC custom claims for AuthN. Closes #5953 (#6444)
* [3e9d8373d](https://github.com/argoproj/argo-workflows/commit/3e9d8373d9165931aca1c1a3b65d81bba5a33720) fix(pods): set resources from script templates (#6450)
* [3abeb0120](https://github.com/argoproj/argo-workflows/commit/3abeb0120c80fcdf9b8b161178c296c6efccb63d) fix: Do not display clipboard if there is no text (#6452)
* [b16a0a098](https://github.com/argoproj/argo-workflows/commit/b16a0a09879413428fb93f196d4d4e63fe51e657) feat(controller): HTTP Template and Agent support feature  (#5750)
* [dc043ce87](https://github.com/argoproj/argo-workflows/commit/dc043ce87b1c946d2ae4fe677862f31e18c758ff) feat(server): support changing MaxGRPCMessageSize using env variable (#6420)
* [51c15764d](https://github.com/argoproj/argo-workflows/commit/51c15764d52f87d8fc5a63e19cb1ad4d0b41a23e) fix(controller): Reinstate support for outputs.results for containers. Fixes #6428 (#6434)
* [40b08240d](https://github.com/argoproj/argo-workflows/commit/40b08240d7eed5ec19bef923201470b69096736f) fix: support archive.none for OSS directory artifacts (#6312)
* [7ec5b3ea9](https://github.com/argoproj/argo-workflows/commit/7ec5b3ea9e55618f1522dd7e50bbf54baad1ca39) fix(controller): Same workflow nodes are not executing parallel even semaphore locks available (#6418)
* [c29b275d5](https://github.com/argoproj/argo-workflows/commit/c29b275d56ef7f2dbf5822ee981f492c2ff61388) fix(controller): Randomly expr expression fail to resolve (#6410)
* [dd3c11252](https://github.com/argoproj/argo-workflows/commit/dd3c112523ea52a832c8df937dae37c43e2c86cd) fix(controller/cli): Resolve global artifacts created in nested workflows (#6422)
* [b17d1bf7b](https://github.com/argoproj/argo-workflows/commit/b17d1bf7b8db75fde30e0f808c2b57fddecf5b32) fix(emissary): throw argo error on file not exist (#6392)
* [946e4a4a6](https://github.com/argoproj/argo-workflows/commit/946e4a4a6254ff935df99095926905440263223a) fix(executor): Remove 15s guard for Docker executor. Fixes #6415 (#6427)
* [29ebc2a6a](https://github.com/argoproj/argo-workflows/commit/29ebc2a6ab40609784419191aef457ba83e8b062) fix(executor): remove unused import preventing compilation
* [cc701a1af](https://github.com/argoproj/argo-workflows/commit/cc701a1affdb4d29b4f48fdfb5dad719192597ec) feat(controller): opt-in to sending pod node events as pod (#6377)
* [959ce6b7f](https://github.com/argoproj/argo-workflows/commit/959ce6b7fe379e4bd79c565862b8bc03112dc154) feat(artifact): enable gcs ListObjects (#6409)
* [30e2518c2](https://github.com/argoproj/argo-workflows/commit/30e2518c2757d726a8164c6347235a88fd54c834) fix(executor/emissary): fix nonroot sidecars + input/output params & artifacts (#6403)
* [4da8fd940](https://github.com/argoproj/argo-workflows/commit/4da8fd94004d535bc79b2cbfa77f6c8683d0c547) fix(controller): Global parameter is not getting updated (#6401)
* [f2d24b1d9](https://github.com/argoproj/argo-workflows/commit/f2d24b1d9b7301fd9d1ffe2c9275caad25772bc1) fix(controller): Force main container name to be "main" as per v3.0. Fixes #6405 (#6408)
* [2df5f66a3](https://github.com/argoproj/argo-workflows/commit/2df5f66a33e197389ae906e6f7b8fb271f49c54c) fix(executor): fix GCS artifact retry (#6302)
* [092b4271b](https://github.com/argoproj/argo-workflows/commit/092b4271b9b57ce9dbff0d988b04ddbf9742425c) fix(controller): Mark workflows wait for semaphore as pending. Fixes #6351 (#6356)
* [453539690](https://github.com/argoproj/argo-workflows/commit/453539690e01827e97fd4921aaa425b2c864a3b1) fix(controller): allow initial duration to be 0 instead of current_time-0 (#6389)
* [b15a79cc3](https://github.com/argoproj/argo-workflows/commit/b15a79cc30509620fea703811f9a9c708f1b64d2)  docs: Add 4intelligence (#6400)
* [f4b89dc8e](https://github.com/argoproj/argo-workflows/commit/f4b89dc8eebc280c5732ae06c2864bdaa1a30e87) fix: Server crash when opening timeline tab for big workflows (#6369)
* [99359a095](https://github.com/argoproj/argo-workflows/commit/99359a0950549515eed306c6839a181a2c356612) Revert "fix: examples/ci.yaml indent (#6328)"
* [66c441006](https://github.com/argoproj/argo-workflows/commit/66c441006e4d1b237de94c91d2f8eb7733ba88d0) fix(gcs): throw argo not found error if key not exist (#6393)
* [3f72fe506](https://github.com/argoproj/argo-workflows/commit/3f72fe506f6c10054692ce07f9b2eaf0f62830a7) fix: examples/ci.yaml indent (#6328)
* [9233a8de7](https://github.com/argoproj/argo-workflows/commit/9233a8de77911d1c22f3a10977a33b48eccb9e63) fix(controller): fix retry on transient errors when validating workflow spec (#6370)
* [488aec3ca](https://github.com/argoproj/argo-workflows/commit/488aec3cad640cd99e21a0c95898463a860a8c0e) fix(controller): allow workflow.duration to pass validator (#6376)
* [d6ec03238](https://github.com/argoproj/argo-workflows/commit/d6ec032388ab8d363faf4e6984b54950dd9abcad) feat(controller): Allow configurable host name label key when retrying different hosts (#6341)
* [bd5a8a99b](https://github.com/argoproj/argo-workflows/commit/bd5a8a99bc470c13a93894be9c0f7f23142a4a31) fix(fields): handle nexted fields when excluding (#6359)
* [cfab7db53](https://github.com/argoproj/argo-workflows/commit/cfab7db53c760ab4354562593b3a5e01e47c733d) feat(controller): sortDAGTasks supports sort by field Depends (#6307)
* [6e58b35c3](https://github.com/argoproj/argo-workflows/commit/6e58b35c34c70df11d7727519249fff46a23ab2b) fix(cli): Overridding name/generateName when creating CronWorkflows if specified (#6308)
* [b388c63d0](https://github.com/argoproj/argo-workflows/commit/b388c63d089cc8c302fdcdf81be3dcd9c12ab6f2) fix(crd): temp fix 34s timeout bug for k8s 1.20+ (#6350)
* [3db467e6b](https://github.com/argoproj/argo-workflows/commit/3db467e6b9bed209404c1a8a0152468ea832f06d) fix(cli): v3.1 Argo Auth Token (#6344)
* [d7c09778a](https://github.com/argoproj/argo-workflows/commit/d7c09778ab9e2c3ce88a2fc6de530832f3770698) fix(controller): Not updating StoredWorkflowSpec when WFT changed during workflow running (#6342)
* [7c38fb01b](https://github.com/argoproj/argo-workflows/commit/7c38fb01bb8862b6933603d73a5f300945f9b031) feat(controller): Differentiate CronWorkflow submission vs invalid spec error metrics (#6309)
* [85c9279a9](https://github.com/argoproj/argo-workflows/commit/85c9279a9019b400ee55d0471778eb3cc4fa20db) feat(controller): Store artifact repository in workflow status. Fixes #6255 (#6299)
* [d07d933be](https://github.com/argoproj/argo-workflows/commit/d07d933bec71675138a73ba53771c45c4f545801) require sso redirect url to be an argo url (#6211)
* [c2360c4c4](https://github.com/argoproj/argo-workflows/commit/c2360c4c47e073fde5df04d32fdb910dd8f7dd77) fix(cli): Only list needed fields. Fixes #6000 (#6298)
* [c11584940](https://github.com/argoproj/argo-workflows/commit/c1158494033321ecff6e12ac1ac8a847a7d278bf) fix(executor): emissary - make /var/run/argo files readable from non-root users. Fixes #6238 (#6304)
* [c9246d3d4](https://github.com/argoproj/argo-workflows/commit/c9246d3d4c162e0f7fe76f2ee37c55bdbfa4b0c6) fix(executor): Tolerate docker re-creating containers. Fixes #6244 (#6252)
* [f78b759cf](https://github.com/argoproj/argo-workflows/commit/f78b759cfca07c47ae41990e1bbe031e862993f6) feat: Introduce when condition to retryStrategy (#6114)
* [05c901fd4](https://github.com/argoproj/argo-workflows/commit/05c901fd4f622aa9aa87b3eabfc87f0bec6dea30) fix(executor): emissary - make argoexec executable from non-root containers. Fixes #6238 (#6247)
* [73a36d8bf](https://github.com/argoproj/argo-workflows/commit/73a36d8bf4b45fd28f1cc80b39bf1bfe265cf6b7) feat: Add support for deletion delay when using PodGC (#6168)
* [19da54109](https://github.com/argoproj/argo-workflows/commit/19da5410943fe0b5f8d7f8b79c5db5d648b65d59) fix(conttroller): Always set finishedAt dote. Fixes #6135 (#6139)
* [92eb8b766](https://github.com/argoproj/argo-workflows/commit/92eb8b766b8501b697043fd1677150e1e565da49) fix: Reduce argoexec image size (#6197)
* [631b0bca5](https://github.com/argoproj/argo-workflows/commit/631b0bca5ed3e9e2436b541b2a270f12796961d1) feat(ui): Add copy to clipboard shortcut (#6217)
* [8d3627d3f](https://github.com/argoproj/argo-workflows/commit/8d3627d3fba46257d32d05be9fd0037ac11b0ab4) fix: Fix certain sibling tasks not connected to parent (#6193)
* [4fd38facb](https://github.com/argoproj/argo-workflows/commit/4fd38facbfb66b06ab0205b04f6e1f1e9943eb6a) fix: Fix security issues related to file closing and paths (G307 & G304) (#6200)
* [cecc379ce](https://github.com/argoproj/argo-workflows/commit/cecc379ce23e708479e4253bbbf14f7907272c9c) refactor: Remove the need for pod annotations to be mounted as a volume (#6022)
* [0e94283ae](https://github.com/argoproj/argo-workflows/commit/0e94283aea641c6c927c9165900165a72022124f) fix(server): Fix issue with auto oauth redirect URL in callback and handle proxies (#6175)
* [0cc5a24c5](https://github.com/argoproj/argo-workflows/commit/0cc5a24c59309438e611223475cdb69c5e3aa01e) fix(controller): Wrong validate order when validate DAG task's argument (#6190)
* [9fe8c1085](https://github.com/argoproj/argo-workflows/commit/9fe8c10858a5a1f024abc812f2e3250f35d7f45e) fix(controller): dehydrate workflow before deleting offloaded node status (#6112)
* [510b4a816](https://github.com/argoproj/argo-workflows/commit/510b4a816dbb2d33f37510db1fd92b841c4d14d3) fix(controller): Allow retry on transient errors when validating workflow spec. Fixes #6163 (#6178)
* [4f847e099](https://github.com/argoproj/argo-workflows/commit/4f847e099ec2a2fef12e98af36b2e4995f8ba3e4) feat(server): Allow redirect_uri to be automatically resolved when using sso (#6167)
* [95ad561ae](https://github.com/argoproj/argo-workflows/commit/95ad561aec5ec360448267b09d8d2238c98012e0) feat(ui): Add checkbox to check all workflows in list. Fixes #6069 (#6158)
* [43f68f4aa](https://github.com/argoproj/argo-workflows/commit/43f68f4aa16ab696d26be6a33b8893418844d838) fix(ui): Fix event-flow scrolling. Fixes #6133 (#6147)
* [9f0cdbdd7](https://github.com/argoproj/argo-workflows/commit/9f0cdbdd78e8eb5b9001243c00cdff5915635401) fix(executor): Capture emissary main-logs. Fixes #6145 (#6146)
* [963bed34b](https://github.com/argoproj/argo-workflows/commit/963bed34bf2ac828384bbbda737e0d8a540bddbb) fix(ui): Fix-up local storage namespaces. Fixes #6109 (#6144)
* [80599325f](https://github.com/argoproj/argo-workflows/commit/80599325feab42bf473925aa9a28a805fc9e1e6e) fix(controller): Performance improvement for Sprig. Fixes #6135 (#6140)
* [868868ee2](https://github.com/argoproj/argo-workflows/commit/868868ee2eb836e9134bdb1f92e7dc2c458722ca) fix: Allow setting workflow input parameters in UI. Fixes #4234 (#5319)
* [357429635](https://github.com/argoproj/argo-workflows/commit/3574296358191edb583bf43d6459259c4156a1e6) build image output to docker (#6128)
* [b38fd1404](https://github.com/argoproj/argo-workflows/commit/b38fd14041e5e61618ea63975997d15704dac8f3) fix(executor): Check whether any errors within checkResourceState() are transient. Fixes #6118. (#6134)
* [db95dbfa1](https://github.com/argoproj/argo-workflows/commit/db95dbfa1edd4a31b1fbd6adbb8e47ca8f2ac428) add troubleshooting notes section for running-locally docs (#6132)
* [b5bd0242d](https://github.com/argoproj/argo-workflows/commit/b5bd0242dd30273161d0ae45bb9e82e85534a53b) Update events.md (#6119)
* [a497e82e0](https://github.com/argoproj/argo-workflows/commit/a497e82e0e6e7e17de20830cc8ea9d306d26d5ca) fix(executor): Fix docker not terminating. Fixes #6064 (#6083)
* [1d76c4815](https://github.com/argoproj/argo-workflows/commit/1d76c4815704e509d7aedc1a79224fbee65ae8ff) feat(manifests): add 'app' label to workflow-controller-metrics service (#6079)
* [1533dd467](https://github.com/argoproj/argo-workflows/commit/1533dd467fa8e0c08a2a5b5fe9d0a1b4dea15b89) fix(executor): Fix emissary kill. Fixes #6030 (#6084)
* [00b56e543](https://github.com/argoproj/argo-workflows/commit/00b56e543092f2af24263ef83595b53c0bae9619) fix(executor): Fix `kubectl` permission error (#6091)
* [7dc6515ce](https://github.com/argoproj/argo-workflows/commit/7dc6515ce1ef76475ac7bd2a7a3c3cdbe795a13c) Point to latest stable release (#6092)
* [be63efe89](https://github.com/argoproj/argo-workflows/commit/be63efe8950e9ba3f15f1ad637e2b3863b85e093) feat(executor)!: Change `argoexec` base image to alpine. Closes #5720 (#6006)
* [937bbb9d9](https://github.com/argoproj/argo-workflows/commit/937bbb9d9a0afe3040afc3c6ac728f9c72759c6a) feat(executor): Configurable interval for wait container to check container statuses. Fixes #5943 (#6088)
* [c111b4294](https://github.com/argoproj/argo-workflows/commit/c111b42942e1edc4e32eb79e78ad86719f2d3f19) fix(executor): Improve artifact error messages. Fixes #6070 (#6086)
* [53bd960b6](https://github.com/argoproj/argo-workflows/commit/53bd960b6e87a3e77cb320e4b53f9f9d95934149) Update upgrading.md
* [493595a78](https://github.com/argoproj/argo-workflows/commit/493595a78258c13b9b0bfc86fd52bf729e8a9a8e) feat: Add TaskSet CRD and HTTP Template (#5628)

<details><summary><h3>Contributors</h3></summary>

* Aaron Mell
* Alex Collins
* Alexander Matyushentsev
* Antoine Dao
* Antony Chazapis
* BOOK
* Brandon High
* Byungjin Park (Claud)
* Caden
* Carlos Montemuino
* Christophe Blin
* Daan Seynaeve
* Daisuke Taniwaki
* David Collom
* Denis Bellotti
* Dominik Deren
* Ed Marks
* Gage Orsburn
* Geoffrey Huntley
* Henrik Blixt
* Huan-Cheng Chang
* Ivan Karol
* Joe McGovern
* KUNG HO BACK
* Kaito Ii
* Kyle Prager
* Luces Huayhuaca
* Marcin Gucki
* Michael Crenshaw
* Miles Croxford
* Mohammad Ali
* Niklas Hansson
* Peixuan Ding
* Reijer Copier
* Saravanan Balasubramanian
* Sebastian Nyberg
* Simon Behar
* Stefan Sedich
* Tetsuya Shiota
* Thiago Bittencourt Gil
* Tianchu Zhao
* Tom Meadows
* Valér Orlovský
* William Van Hevelingen
* Windfarer
* Yuan (Bob) Gong
* Yuan Tang
* Zadkiel
* brgoode
* dpeer6
* github-actions[bot]
* jibuji
* kennytrytek
* meijin
* steve-marmalade
* wanghong230

</details>

## v3.1.15 (2021-11-17)

Full Changelog: [v3.1.14...v3.1.15](https://github.com/argoproj/argo-workflows/compare/v3.1.14...v3.1.15)

### Selected Changes

* [a0d675692](https://github.com/argoproj/argo-workflows/commit/a0d6756922f7ba89f20b034dd265d0b1e393e70f) fix: add gh ecdsa and ed25519 to known hosts (#7226)

<details><summary><h3>Contributors</h3></summary>

* Rob Herley

</details>

## v3.1.14 (2021-10-19)

Full Changelog: [v3.1.13...v3.1.14](https://github.com/argoproj/argo-workflows/compare/v3.1.13...v3.1.14)

### Selected Changes

* [f647435b6](https://github.com/argoproj/argo-workflows/commit/f647435b65d5c27e84ba2d2383f0158ec84e6369) fix: do not delete expr tag tmpl values. Fixes #6909 (#6921)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins

</details>

## v3.1.13 (2021-09-28)

Full Changelog: [v3.1.12...v3.1.13](https://github.com/argoproj/argo-workflows/compare/v3.1.12...v3.1.13)

### Selected Changes

* [78cd6918a](https://github.com/argoproj/argo-workflows/commit/78cd6918a8753a8448ed147b875588d56bd26252) fix: Missing duration metrics if controller restart (#6815)
* [1fe754ef1](https://github.com/argoproj/argo-workflows/commit/1fe754ef10bd95e3fe3485f67fa7e9c5523b1dea) fix: Fix expression template random errors. Fixes #6673 (#6786)
* [3a98174da](https://github.com/argoproj/argo-workflows/commit/3a98174dace34ffac7dd7626a253bbb1101df515) fix: Fix bugs, unable to resolve tasks aggregated outputs in dag outputs. Fixes #6684 (#6692)
* [6e93af099](https://github.com/argoproj/argo-workflows/commit/6e93af099d1c93d1d27fc86aba6d074d6d79cffc) fix: remove windows UNC paths from wait/init containers. Fixes #6583 (#6704)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins
* Anish Dangi
* Saravanan Balasubramanian
* smile-luobin

</details>

## v3.1.12 (2021-09-16)

Full Changelog: [v3.1.11...v3.1.12](https://github.com/argoproj/argo-workflows/compare/v3.1.11...v3.1.12)

### Selected Changes

* [e62b9a8dc](https://github.com/argoproj/argo-workflows/commit/e62b9a8dc8924e545d57d1f90f901fbb0b694e09) feat(ui): logsViewer use archived log if node finish and archived (#6708)
* [da5ce18cf](https://github.com/argoproj/argo-workflows/commit/da5ce18cf24103ca9418137229fc355a9dc725c9) fix: Daemond status stuck with Running (#6742)

<details><summary><h3>Contributors</h3></summary>

* Saravanan Balasubramanian
* Tianchu Zhao

</details>

## v3.1.11 (2021-09-13)

Full Changelog: [v3.1.10...v3.1.11](https://github.com/argoproj/argo-workflows/compare/v3.1.10...v3.1.11)

### Selected Changes

* [665c08d29](https://github.com/argoproj/argo-workflows/commit/665c08d2906f1bb15fdd8c2f21e6877923e0394b) skippied flakytest
* [459a61170](https://github.com/argoproj/argo-workflows/commit/459a61170663729c912a9b387fd7fa5c8a147839) fix(executor): handle hdfs optional artifact at retriving hdfs file stat (#6703)
* [82e408297](https://github.com/argoproj/argo-workflows/commit/82e408297c65a2d64408d9f6fb01766192fcec42) fix: panic in prepareMetricScope (#6720)
* [808d897a8](https://github.com/argoproj/argo-workflows/commit/808d897a844b46487de65ce27ddeb2dad614f417) fix(ui): undefined cron timestamp (#6713)

<details><summary><h3>Contributors</h3></summary>

* Saravanan Balasubramanian
* Tianchu Zhao

</details>

## v3.1.10 (2021-09-10)

Full Changelog: [v3.1.9...v3.1.10](https://github.com/argoproj/argo-workflows/compare/v3.1.9...v3.1.10)

### Selected Changes

* [2730a51a2](https://github.com/argoproj/argo-workflows/commit/2730a51a203d6b587db5fe43a0e3de018a35dbd8) fix: Fix `x509: certificate signed by unknown authority` error (#6566)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins
* makocchi

</details>

## v3.1.9 (2021-09-03)

Full Changelog: [v3.1.8...v3.1.9](https://github.com/argoproj/argo-workflows/compare/v3.1.8...v3.1.9)

### Selected Changes

* [e4f6bcb02](https://github.com/argoproj/argo-workflows/commit/e4f6bcb02f10bea5c76f2f91ff223b8a380b4557) fix the codegen
* [92153dcca](https://github.com/argoproj/argo-workflows/commit/92153dcca774bb3097f00b86b35edf966ead7de4) fixed test
* [117e85f47](https://github.com/argoproj/argo-workflows/commit/117e85f473fd6b4d9e7cebd4406503896f4d0639) fix(cli): Added validatation for StartAt, FinishedAt and ID (#6661)
* [01083d1d1](https://github.com/argoproj/argo-workflows/commit/01083d1d1f485b1ae1fb1e697090db0069e25e96) fix(controller): Set finishedAt for workflow with Daemon steps
* [926e43950](https://github.com/argoproj/argo-workflows/commit/926e439503f61766eea61c2eec079571d778a31e) fix: Do not index complete workflow semaphores (#6639)
* [a039a29ab](https://github.com/argoproj/argo-workflows/commit/a039a29ab27e6ce50ecaf345c3d826d90597523d) fix: `argo node set` panic: index out of range and correct docs (#6653)
* [8f8fc2bd9](https://github.com/argoproj/argo-workflows/commit/8f8fc2bd9e2904729bc75e71611673b70d55c2f6) fix(controller): Initialize throttler during starting workflow-controller. Fixes: #6599 (#6608)
* [940e993ff](https://github.com/argoproj/argo-workflows/commit/940e993ffccb737a45774f9fc623d5a548d57978) fix(gcs): backoff bool should be false if error is transient (#6577)
* [2af306a52](https://github.com/argoproj/argo-workflows/commit/2af306a52de80efd3b50bcbd6db144ddede851d1) fix(executor/pns): panic of pidFileHandles concurrent writes (#6569)
* [1019a13a6](https://github.com/argoproj/argo-workflows/commit/1019a13a6139d5867bb657ca8593fdb671bb3598) fix(executor/docker): fix failed to wait for main container to complete: timed out waiting for the condition: container does not exist (#6561)
* [563bb04c4](https://github.com/argoproj/argo-workflows/commit/563bb04c4f8d5d8e5bf83ecdf080926beb9e4bae) fix: Generate TLS Certificates on startup and only keep in memory (#6540)
* [36d2389f2](https://github.com/argoproj/argo-workflows/commit/36d2389f23dc832fe962025ad7b2a6cf6ed9bce3) fix: use execWf when setting PodMetadata (#6512)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins
* Antoine Dao
* David Collom
* Ed Marks
* Jesse Suen
* Saravanan Balasubramanian
* Windfarer
* Yuan (Bob) Gong
* smile-luobin

</details>

## v3.1.8 (2021-08-18)

Full Changelog: [v3.1.7...v3.1.8](https://github.com/argoproj/argo-workflows/compare/v3.1.7...v3.1.8)

### Selected Changes

* [0df0f3a98](https://github.com/argoproj/argo-workflows/commit/0df0f3a98fac4e2aa5bc02213fb0a2ccce9a682a) fix: Fix `x509: certificate signed by unknown authority` error (#6566)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins

</details>

## v3.1.7 (2021-08-18)

Full Changelog: [v3.1.6...v3.1.7](https://github.com/argoproj/argo-workflows/compare/v3.1.6...v3.1.7)

### Selected Changes

* [5463b5d4f](https://github.com/argoproj/argo-workflows/commit/5463b5d4feb626ac80def3c521bd20e6a96708c4) fix: Generate TLS Certificates on startup and only keep in memory (#6540)

<details><summary><h3>Contributors</h3></summary>

* David Collom

</details>

## v3.1.6 (2021-08-12)

Full Changelog: [v3.1.5...v3.1.6](https://github.com/argoproj/argo-workflows/compare/v3.1.5...v3.1.6)

### Selected Changes

* [14e127857](https://github.com/argoproj/argo-workflows/commit/14e1278572b28d8b1854858ce7de355ce60199c9) ci-build.yaml-with-master-change
* [c0ac267ab](https://github.com/argoproj/argo-workflows/commit/c0ac267ab50ba8face0cc14eef0563dddd3f16f6) ci-build.yaml
* [c87ce923b](https://github.com/argoproj/argo-workflows/commit/c87ce923bfd6723f91213696c4ee3af5f210cdb8) Update ci-build.yaml
* [896bcbd7d](https://github.com/argoproj/argo-workflows/commit/896bcbd7d33348054833af20792f923eac728091) Update ci-build.yaml
* [cefddb273](https://github.com/argoproj/argo-workflows/commit/cefddb273d0edcd622a3df368a542cdf33df7f47) Update workflowpod_test.go
* [47720040a](https://github.com/argoproj/argo-workflows/commit/47720040afd142d5726f28757912e0589f4ea901) fixed codegen
* [501c1720a](https://github.com/argoproj/argo-workflows/commit/501c1720a2cf09907bf05a2641ad802e9d084c86) fix: use execWf when setting PodMetadata (#6512)
* [4458394a8](https://github.com/argoproj/argo-workflows/commit/4458394a8c1af8e7328d06cc417850e410f7dd72) fix: Fix the Status update for node with synchronization lock (#6525)
* [907effbfc](https://github.com/argoproj/argo-workflows/commit/907effbfcd4f3bf058fb0e5bbd6faea512401ea9) fix: Upgrade pkg to v0.10.1. Fixes #6521 (#6523)
* [46e2803f7](https://github.com/argoproj/argo-workflows/commit/46e2803f7e0a6d7fd3213d5f02d58fae9ee78880) fix(controller): Fix panic in addParamToGlobalScope (#6485)
* [e1149b61a](https://github.com/argoproj/argo-workflows/commit/e1149b61aca5fde7b63be2e8f5d9b0be148b5eee) fix(controller): JSON-unmarshal marshaled expression template before evaluating (#6285)
* [e6a3b0c76](https://github.com/argoproj/argo-workflows/commit/e6a3b0c764ae54985a7315e7dbf656e766ae33e8) fix(executor/docker): re-revert -- fix random errors with message "No such container:path". Fixes #6352 (#6508)
* [b37e81a98](https://github.com/argoproj/argo-workflows/commit/b37e81a98b7f7c8c11317edfc06950778cd482ad) fix: Remove client private key from client auth REST config (#6506)
* [cc51e71ce](https://github.com/argoproj/argo-workflows/commit/cc51e71ced57448839e98d44fe34780671f03066) fix(controller): JSON-unmarshal marshaled expression template before evaluating (#6285)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins
* Ed Marks
* Michael Crenshaw
* Saravanan Balasubramanian
* William Van Hevelingen
* Yuan (Bob) Gong

</details>

## v3.1.5 (2021-08-03)

Full Changelog: [v3.1.4...v3.1.5](https://github.com/argoproj/argo-workflows/compare/v3.1.4...v3.1.5)

### Selected Changes

* [3dbee0ec3](https://github.com/argoproj/argo-workflows/commit/3dbee0ec368f3ea8c31f49c8b1a4617cc32bcce9) fix(executor): emissary - make argoexec executable from non-root containers. Fixes #6238 (#6247)

<details><summary><h3>Contributors</h3></summary>

* Yuan (Bob) Gong

</details>

## v3.1.4 (2021-08-03)

Full Changelog: [v3.1.3...v3.1.4](https://github.com/argoproj/argo-workflows/compare/v3.1.3...v3.1.4)

### Selected Changes

* [247776d66](https://github.com/argoproj/argo-workflows/commit/247776d66fa6bf988f861ba82f181e386a972626) removed unused import
* [89d662c39](https://github.com/argoproj/argo-workflows/commit/89d662c39e326977384683a255b7472839d957ee) fix: Exit template shouldn't fail with max parallelism reached (#6456)
* [4556ba27b](https://github.com/argoproj/argo-workflows/commit/4556ba27b81c2291353d93fd59a581e3a2a2bb21) fix(controller): fix retry on different hosts (#6429)
* [fc8260b6e](https://github.com/argoproj/argo-workflows/commit/fc8260b6e1f55d939f16bee682f73ba59774cbb9) fix(controller): fix retry on different hosts (#6429)
* [b489d03b4](https://github.com/argoproj/argo-workflows/commit/b489d03b417ecd89654bd6b524c6daf38675ec63) fix(server): Fix nil pointer error when getting artifacts from a step without artifacts (#6465)
* [4d99aac6e](https://github.com/argoproj/argo-workflows/commit/4d99aac6eb3b065eec2be215439dd5a77f337907) fix(pods): set resources from script templates (#6450)
* [3f594ca8d](https://github.com/argoproj/argo-workflows/commit/3f594ca8dd891149f1a07d123fd53097dc3b4438) fix(emissary): throw argo error on file not exist (#6392)
* [f4e20761f](https://github.com/argoproj/argo-workflows/commit/f4e20761f484ce3bf0b3610457193c0324cffa12) Update umask_windows.go
* [cc84fe94c](https://github.com/argoproj/argo-workflows/commit/cc84fe94cfb2df447bf8d1dbe28cc416b866b159) fix(executor): fix GCS artifact retry (#6302)
* [0b0f52788](https://github.com/argoproj/argo-workflows/commit/0b0f527881f5b0a48d8cf77c9e6a29fbeb27b4dc) fix(gcs): throw argo not found error if key not exist (#6393)

<details><summary><h3>Contributors</h3></summary>

* Antoine Dao
* Marcin Gucki
* Peixuan Ding
* Saravanan Balasubramanian
* Tianchu Zhao
* Yuan (Bob) Gong

</details>

## v3.1.3 (2021-07-27)

Full Changelog: [v3.1.2...v3.1.3](https://github.com/argoproj/argo-workflows/compare/v3.1.2...v3.1.3)

### Selected Changes

* [9337abb00](https://github.com/argoproj/argo-workflows/commit/9337abb002d3c505ca45c5fd2e25447acd80a108) fix(controller): Reinstate support for outputs.results for containers. Fixes #6428 (#6434)
* [d2fc4dd62](https://github.com/argoproj/argo-workflows/commit/d2fc4dd62389b3b6726f12e68a86f3179cf957b2) fix(controller): Same workflow nodes are not executing parallel even semaphore locks available (#6418)
* [13c51d4b2](https://github.com/argoproj/argo-workflows/commit/13c51d4b2c1f2ed2e8b416953de2516b92a59da4) fix(controller): Randomly expr expression fail to resolve (#6410)
* [0e5dfe50b](https://github.com/argoproj/argo-workflows/commit/0e5dfe50b2737e1aa564a8684c1ddd08b95755bf) fix(executor): Remove 15s guard for Docker executor. Fixes #6415 (#6427)
* [4347acffc](https://github.com/argoproj/argo-workflows/commit/4347acffc94b50e6e665045f47b07ea0eedd1611) fix(executor): remove unused import preventing compilation
* [1eaa38199](https://github.com/argoproj/argo-workflows/commit/1eaa3819902aef028151e07deccdad2c7cf4fc0d) fix(executor/emissary): fix nonroot sidecars + input/output params & artifacts (#6403)
* [060b727ee](https://github.com/argoproj/argo-workflows/commit/060b727eeedd32102d918caad50557f9e0aa8cca) fix(controller): Global parameter is not getting updated (#6401)
* [adc17ff26](https://github.com/argoproj/argo-workflows/commit/adc17ff267f3b0951c0bedf0db3c9eab20af7f7c) fix(controller): Force main container name to be "main" as per v3.0. Fixes #6405 (#6408)
* [069816a0a](https://github.com/argoproj/argo-workflows/commit/069816a0aaf89590b98257e1e7360c925ee16ad1) fix(controller): Mark workflows wait for semaphore as pending. Fixes #6351 (#6356)
* [791c26b3c](https://github.com/argoproj/argo-workflows/commit/791c26b3cd6f56af90bfd3b69187921753d61d82) fix(controller): allow initial duration to be 0 instead of current_time-0 (#6389)
* [bd757e86c](https://github.com/argoproj/argo-workflows/commit/bd757e86c21ad9b52473ea8f1c6e3e6730694260) fix: Server crash when opening timeline tab for big workflows (#6369)
* [8b49e8c3a](https://github.com/argoproj/argo-workflows/commit/8b49e8c3a58a487eb9767569ad02ce2ac8a967eb) fix(controller): allow workflow.duration to pass validator (#6376)
* [24ff9450a](https://github.com/argoproj/argo-workflows/commit/24ff9450ad436eff34e383ce9dd625f4b29e3737) fix(fields): handle nexted fields when excluding (#6359)
* [a83ec79dd](https://github.com/argoproj/argo-workflows/commit/a83ec79dddec3c030526e58e9e06b3dc0604e21f) feat(controller): sortDAGTasks supports sort by field Depends (#6307)
* [8472227f5](https://github.com/argoproj/argo-workflows/commit/8472227f5a23435253ad6bfaf732318afdde1bf8) fix(crd): temp fix 34s timeout bug for k8s 1.20+ (#6350)
* [0522a68fc](https://github.com/argoproj/argo-workflows/commit/0522a68fc595a4d199e2bf57a0574ef9f12f875f) Revert "feat: added support for GRPC_MESSAGE_SIZE env var  (#6258)"
* [49db7cd60](https://github.com/argoproj/argo-workflows/commit/49db7cd6038172c0d6c784882a253386c457695f) feat: added support for GRPC_MESSAGE_SIZE env var  (#6258)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins
* Alexander Matyushentsev
* Antoine Dao
* BOOK
* Saravanan Balasubramanian
* Tianchu Zhao
* Yuan (Bob) Gong
* dpeer6

</details>

## v3.1.2 (2021-07-15)

Full Changelog: [v3.1.1...v3.1.2](https://github.com/argoproj/argo-workflows/compare/v3.1.1...v3.1.2)

### Selected Changes

* [98721a96e](https://github.com/argoproj/argo-workflows/commit/98721a96eef8e4fe9a237b2105ba299a65eaea9a) fixed test
* [6041ffe22](https://github.com/argoproj/argo-workflows/commit/6041ffe228c8f79e6578e097a357dfebf768c78f) fix(controller): Not updating StoredWorkflowSpec when WFT changed during workflow running (#6342)
* [d14760182](https://github.com/argoproj/argo-workflows/commit/d14760182851c280b11d688b70a81f3fe014c52f) fix(cli): v3.1 Argo Auth Token (#6344)
* [ce5679c4b](https://github.com/argoproj/argo-workflows/commit/ce5679c4bd1040fa5d68eea24a4a82ef3844d43c) feat(controller): Store artifact repository in workflow status. Fixes
* [74581157f](https://github.com/argoproj/argo-workflows/commit/74581157f9fd8190027021dd5af409cd3e3e781f) fix(executor): Tolerate docker re-creating containers. Fixes #6244 (#6252)
* [cd208e27f](https://github.com/argoproj/argo-workflows/commit/cd208e27ff0e45f82262b18ebb65081ae5978761) fix(executor): emissary - make /var/run/argo files readable from non-root users. Fixes #6238 (#6304)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins
* Michael Crenshaw
* Saravanan Balasubramanian
* Yuan (Bob) Gong

</details>

## v3.1.1 (2021-06-28)

Full Changelog: [v3.1.0...v3.1.1](https://github.com/argoproj/argo-workflows/compare/v3.1.0...v3.1.1)

### Selected Changes

* [4d12bbfee](https://github.com/argoproj/argo-workflows/commit/4d12bbfee13faea6d2715c809fab40ae33a66074) fix(conttroller): Always set finishedAt dote. Fixes #6135 (#6139)
* [401a66188](https://github.com/argoproj/argo-workflows/commit/401a66188d25bef16078bba370fc26d1fbd56288) fix: Fix certain sibling tasks not connected to parent (#6193)
* [99b42eb1c](https://github.com/argoproj/argo-workflows/commit/99b42eb1c0902c7df6a3e2904dafd93b294c9e96) fix(controller): Wrong validate order when validate DAG task's argument (#6190)
* [18b2371e3](https://github.com/argoproj/argo-workflows/commit/18b2371e36f106062d1a2cc2e81ca37052b8296b) fix(controller): dehydrate workflow before deleting offloaded node status (#6112)
* [a58cbdc39](https://github.com/argoproj/argo-workflows/commit/a58cbdc3966188a1ea5d9207f99e289ee758804f) fix(controller): Allow retry on transient errors when validating workflow spec. Fixes #6163 (#6178)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins
* BOOK
* Reijer Copier
* Simon Behar
* Yuan Tang

</details>

## v3.1.0 (2021-06-21)

Full Changelog: [v3.1.0-rc14...v3.1.0](https://github.com/argoproj/argo-workflows/compare/v3.1.0-rc14...v3.1.0)

### Selected Changes

* [fad026e36](https://github.com/argoproj/argo-workflows/commit/fad026e367dd08b0217155c433f2f87c310506c5) fix(ui): Fix event-flow scrolling. Fixes #6133 (#6147)
* [422f5f231](https://github.com/argoproj/argo-workflows/commit/422f5f23176d5ef75e58c5c33b744cf2d9ac38ca) fix(executor): Capture emissary main-logs. Fixes #6145 (#6146)
* [e818b15cc](https://github.com/argoproj/argo-workflows/commit/e818b15ccfdd51b231cb0f9e8872cc673f196e61) fix(ui): Fix-up local storage namespaces. Fixes #6109 (#6144)
* [681e1e42a](https://github.com/argoproj/argo-workflows/commit/681e1e42aa1126d38bbc0cfe4bbd7b1664137c16) fix(controller): Performance improvement for Sprig. Fixes #6135 (#6140)
* [99139fea8](https://github.com/argoproj/argo-workflows/commit/99139fea8ff6325d02bb97a5966388aa37e3bd30) fix(executor): Check whether any errors within checkResourceState() are transient. Fixes #6118. (#6134)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins
* Yuan Tang

</details>

## v3.1.0-rc14 (2021-06-10)

Full Changelog: [v3.1.0-rc13...v3.1.0-rc14](https://github.com/argoproj/argo-workflows/compare/v3.1.0-rc13...v3.1.0-rc14)

### Selected Changes

* [d385e6107](https://github.com/argoproj/argo-workflows/commit/d385e6107ab8d4ea4826bd6972608f8fbc86fbe5) fix(executor): Fix docker not terminating. Fixes #6064 (#6083)
* [83da6deae](https://github.com/argoproj/argo-workflows/commit/83da6deae5eaaeca16e49edb584a0a46980239bb) feat(manifests): add 'app' label to workflow-controller-metrics service (#6079)
* [1c27b5f90](https://github.com/argoproj/argo-workflows/commit/1c27b5f90dea80b5dc7f088bef0dc908e8c19661) fix(executor): Fix emissary kill. Fixes #6030 (#6084)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins
* Daan Seynaeve

</details>

## v3.1.0-rc13 (2021-06-08)

Full Changelog: [v3.1.0-rc12...v3.1.0-rc13](https://github.com/argoproj/argo-workflows/compare/v3.1.0-rc12...v3.1.0-rc13)

### Selected Changes

* [0e37f6632](https://github.com/argoproj/argo-workflows/commit/0e37f6632576ffd5365c7f48d455bd9a9a0deefc) fix(executor): Improve artifact error messages. Fixes #6070 (#6086)
* [4bb4d528e](https://github.com/argoproj/argo-workflows/commit/4bb4d528ee4decba0ac4d736ff1ba6302163fccf) fix(ui): Tweak workflow log viewer (#6074)
* [f8f63e628](https://github.com/argoproj/argo-workflows/commit/f8f63e628674fcb6755e9ef50bea1d148ba49ac2) fix(controller): Handling panic in leaderelection (#6072)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins
* Saravanan Balasubramanian
* Yuan Tang
* github-actions[bot]

</details>

## v3.1.0-rc12 (2021-06-02)

Full Changelog: [v3.1.0-rc11...v3.1.0-rc12](https://github.com/argoproj/argo-workflows/compare/v3.1.0-rc11...v3.1.0-rc12)

### Selected Changes

* [803855bc9](https://github.com/argoproj/argo-workflows/commit/803855bc9754b301603903ec7cb4cd9a2979a12b) fix(executor): Fix compatibility issue when selfLink is no longer populated for k8s>=1.21. Fixes #6045 (#6014)
* [1f3493aba](https://github.com/argoproj/argo-workflows/commit/1f3493abaf18d27e701b9f14083dae35447d289e) feat(ui): Add text filter to logs. Fixes #6059 (#6061)
* [eaeaec71f](https://github.com/argoproj/argo-workflows/commit/eaeaec71fd1fb2b0f2f217aada7f47036ace71dd) fix(controller): Only clean-up pod when both main and wait containers have terminated. Fixes #5981 (#6033)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins
* Yuan Tang
* github-actions[bot]

</details>

## v3.1.0-rc11 (2021-06-01)

Full Changelog: [v3.1.0-rc10...v3.1.0-rc11](https://github.com/argoproj/argo-workflows/compare/v3.1.0-rc10...v3.1.0-rc11)

### Selected Changes

* [ee283ee6d](https://github.com/argoproj/argo-workflows/commit/ee283ee6d360650622fc778f38d94994b20796ab) fix(ui): Add editor nav and make taller (#6047)
* [529c30dd5](https://github.com/argoproj/argo-workflows/commit/529c30dd53ba617a4fbea649fa3f901dd8066af6) fix(ui): Changed placing of chat/get help button. Fixes #5817 (#6016)
* [e262b3afd](https://github.com/argoproj/argo-workflows/commit/e262b3afd7c8ab77ef14fb858a5795b73630485c) feat(controller): Add per-namespace parallelism limits. Closes #6037 (#6039)

<details><summary><h3>Contributors</h3></summary>

* Aayush Rangwala
* Alex Collins
* Kasper Aaquist Johansen
* Simon Behar
* github-actions[bot]

</details>

## v3.1.0-rc10 (2021-05-27)

Full Changelog: [v3.1.0-rc9...v3.1.0-rc10](https://github.com/argoproj/argo-workflows/compare/v3.1.0-rc9...v3.1.0-rc10)

### Selected Changes

* [73539fadb](https://github.com/argoproj/argo-workflows/commit/73539fadbe81b644b912ef0ddddebb178c97cc94) feat(controller): Support rate-limitng pod creation. (#4892)
* [e566c106b](https://github.com/argoproj/argo-workflows/commit/e566c106bbe9baf8ab3628a80235467bb867b57e) fix(server): Only hydrate nodes if they are needed. Fixes #6000 (#6004)
* [d218ea717](https://github.com/argoproj/argo-workflows/commit/d218ea71776fa7d072bbeafa614b36eb34940023) fix(ui): typo (#6027)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins
* Stephan van Maris
* Yuan Tang
* github-actions[bot]

</details>

## v3.1.0-rc9 (2021-05-26)

Full Changelog: [v3.1.0-rc8...v3.1.0-rc9](https://github.com/argoproj/argo-workflows/compare/v3.1.0-rc8...v3.1.0-rc9)

### Selected Changes

* [bad615550](https://github.com/argoproj/argo-workflows/commit/bad61555093f59a647b20df75f83e1cf9687f7b5) fix(ui): Fix link for archived logs (#6019)
* [3cfc96b7c](https://github.com/argoproj/argo-workflows/commit/3cfc96b7c3c90edec77be0841152dad4d9f18f52) revert: "fix(executor): Fix compatibility issue with k8s>=1.21 when s… (#6012)
* [7e27044b7](https://github.com/argoproj/argo-workflows/commit/7e27044b71620dc7c7dd338eac873e0cff244e2d) fix(controller): Increase readiness timeout from 1s to 30s (#6007)
* [79f5fa5f3](https://github.com/argoproj/argo-workflows/commit/79f5fa5f3e348fca5255d9c98b3fb186bc23cb3e) feat: Pass include script output as an environment variable (#5994)
* [d7517cfca](https://github.com/argoproj/argo-workflows/commit/d7517cfcaf141fc06e19720996d7b43ddb3fa7b6) Mention that 'archive' do not support logs of pods (#6005)
* [d7c5cf6c9](https://github.com/argoproj/argo-workflows/commit/d7c5cf6c95056a82ea94e37da925ed566991e548) fix(executor): Fix compatibility issue with k8s>=1.21 when selfLink is no longer populated (#5992)
* [a2c6241ae](https://github.com/argoproj/argo-workflows/commit/a2c6241ae21e749a3c5865153755136ddd878d5c) fix(validate): Fix DAG validation on task names when depends/dependencies is not used. Fixes #5993 (#5998)
* [a99d5b821](https://github.com/argoproj/argo-workflows/commit/a99d5b821bee5edb296f8af1c3badb503025f026) fix(controller): Fix sync manager panic. Fixes #5939 (#5991)
* [80f8473a1](https://github.com/argoproj/argo-workflows/commit/80f8473a13482387b9f54f9288f4a982a210cdea) fix(executor): resource patch for non-json patches regression (#5951)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins
* Antony Chazapis
* Christophe Blin
* Peixuan Ding
* William Reed
* Yuan Tang
* amit
* github-actions[bot]

</details>

## v3.1.0-rc8 (2021-05-24)

Full Changelog: [v3.1.0-rc7...v3.1.0-rc8](https://github.com/argoproj/argo-workflows/compare/v3.1.0-rc7...v3.1.0-rc8)

### Selected Changes

* [f3d95821f](https://github.com/argoproj/argo-workflows/commit/f3d95821faf8b87d416a2d6ee1334b9e45869c84) fix(controller): Listen on :6060 (#5988)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins
* Simon Behar
* github-actions[bot]

</details>

## v3.1.0-rc7 (2021-05-24)

Full Changelog: [v3.1.0-rc6...v3.1.0-rc7](https://github.com/argoproj/argo-workflows/compare/v3.1.0-rc6...v3.1.0-rc7)

### Selected Changes

* [d55a8dbb8](https://github.com/argoproj/argo-workflows/commit/d55a8dbb841a55db70b96568fdd9ef402548d567) feat(controller): Add liveness probe (#5875)
* [46dcaea53](https://github.com/argoproj/argo-workflows/commit/46dcaea53d91b522dfd87b442ce949e3a4de7e76) fix(controller): Lock nodes in pod reconciliation. Fixes #5979 (#5982)
* [60b6b5cf6](https://github.com/argoproj/argo-workflows/commit/60b6b5cf64adec380bc195aa87e4f0b12182fe16) fix(controller): Empty global output param crashes (#5931)
* [453086f94](https://github.com/argoproj/argo-workflows/commit/453086f94c9b540205784bd2944541b1b43555bd) fix(ui): ensure that the artifacts property exists before inspecting it (#5977)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins
* Maximilian Roos
* Saravanan Balasubramanian
* dherman
* github-actions[bot]

</details>

## v3.1.0-rc6 (2021-05-21)

Full Changelog: [v3.1.0-rc5...v3.1.0-rc6](https://github.com/argoproj/argo-workflows/compare/v3.1.0-rc5...v3.1.0-rc6)

### Selected Changes

* [67a38e33e](https://github.com/argoproj/argo-workflows/commit/67a38e33ed1a4d33085c9f566bf64b8b15c8199e) feat: add disableSubmodules for git artifacts (#5910)
* [7b54b182c](https://github.com/argoproj/argo-workflows/commit/7b54b182cfec367d876aead36ae03a1a16632527) small fixes of spelling mistakes (#5886)
* [56b71d07d](https://github.com/argoproj/argo-workflows/commit/56b71d07d91a5aae05b087577f1b47c2acf745df) fix(controller): Revert cb9676e88857193b762b417f2c45b38e2e0967f9. Fixes #5852 (#5933)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins
* Caelan U
* Johannes Olsson
* Lars Kerick
* Michael Crenshaw
* Zach Aller
* github-actions[bot]

</details>

## v3.1.0-rc5 (2021-05-17)

Full Changelog: [v3.1.0-rc4...v3.1.0-rc5](https://github.com/argoproj/argo-workflows/compare/v3.1.0-rc4...v3.1.0-rc5)

### Selected Changes

* [e05f7cbe6](https://github.com/argoproj/argo-workflows/commit/e05f7cbe624ffada191344848d3b0b7fb9ba79ae) fix(controller): Suspend and Resume is not working in WorkflowTemplateRef scenario (#5802)
* [8fde4e4f4](https://github.com/argoproj/argo-workflows/commit/8fde4e4f46f59a6af50e5cc432f632f6f5e774d9) fix(installation): Enable capacity to override namespace with Kustomize (#5907)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins
* Daverkex
* Saravanan Balasubramanian
* github-actions[bot]

</details>

## v3.1.0-rc4 (2021-05-14)

Full Changelog: [v3.1.0-rc3...v3.1.0-rc4](https://github.com/argoproj/argo-workflows/compare/v3.1.0-rc3...v3.1.0-rc4)

### Selected Changes

* [128861c50](https://github.com/argoproj/argo-workflows/commit/128861c50f2b60daded5abb7d47524e124451371) feat: DAG/TASK Custom Metrics Example (#5894)
* [0acaf3b40](https://github.com/argoproj/argo-workflows/commit/0acaf3b40b7704017842c81c0a9108fe4eee906e) Update configure-artifact-repository.md (#5909)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins
* Everton
* Jerguš Lejko
* Yuan Tang
* github-actions[bot]

</details>

## v3.1.0-rc3 (2021-05-13)

Full Changelog: [v3.1.0-rc2...v3.1.0-rc3](https://github.com/argoproj/argo-workflows/compare/v3.1.0-rc2...v3.1.0-rc3)

### Selected Changes

* [e71d33c54](https://github.com/argoproj/argo-workflows/commit/e71d33c54bd3657a4d63ae8bfa3d899b3339d0fb) fix(controller): Fix pod spec jumbling. Fixes #5897 (#5899)
* [9a10bd475](https://github.com/argoproj/argo-workflows/commit/9a10bd475b273a1bc66025b89c8237a2263c840d) fix: workflow-controller: use parentId (#5831)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins
* Jan Heylen

</details>

## v3.1.0-rc2 (2021-05-12)

Full Changelog: [v3.1.0-rc1...v3.1.0-rc2](https://github.com/argoproj/argo-workflows/compare/v3.1.0-rc1...v3.1.0-rc2)

### Selected Changes

<details><summary><h3>Contributors</h3></summary>

* Alex Collins

</details>

## v3.1.0-rc1 (2021-05-12)

Full Changelog: [v3.0.10...v3.1.0-rc1](https://github.com/argoproj/argo-workflows/compare/v3.0.10...v3.1.0-rc1)

### Selected Changes

* [3fff791e4](https://github.com/argoproj/argo-workflows/commit/3fff791e4ef5b7e1de82ccb36cae327e8eb726f6) build!: Automatically add manifests to `v\*` tags (#5880)
* [2687e240c](https://github.com/argoproj/argo-workflows/commit/2687e240c536900a7119a9b988103f5a68234cc5) fix(controller): Fix active pods count in node pending status with pod deleted. (#5836)
* [3428b832d](https://github.com/argoproj/argo-workflows/commit/3428b832d68e1cfb42f4210c3ab5ff4a99620d70) fix(controller): Error template ref exit handlers. Fixes #5835 (#5837)
* [1a5393593](https://github.com/argoproj/argo-workflows/commit/1a5393593c9cc4b61734af63568a21e50b6c4f8c) fix(controller): Remove un-safe Sprig funcs. Fixes #5286 (#5850)
* [c6825acca](https://github.com/argoproj/argo-workflows/commit/c6825acca43ffeb537f8e0d3b62c2addd0d49389) fix(executor): Enable PNS executor to better kill sidecars. Fixes #5779 (#5794)
* [2b3396fad](https://github.com/argoproj/argo-workflows/commit/2b3396fad602013801f5c517567319f60bedb0bb) feat: configurable windows os version (#5816)
* [d66954f5b](https://github.com/argoproj/argo-workflows/commit/d66954f5b9b09e030408483502b03aa29727039a) feat(controller): Add config for potential CPU hogs (#5853)
* [7ec262a56](https://github.com/argoproj/argo-workflows/commit/7ec262a56b7e043aec5913fc9a9be8c6b0a9067d) feat(cli): Support input from device files for lint command (#5851)
* [ab786ecba](https://github.com/argoproj/argo-workflows/commit/ab786ecba6eb3e9d3fa7a717ded42727b8b64df8) fix: Reset started time for each node to current when retrying workflow (#5801)
* [e332be5ec](https://github.com/argoproj/argo-workflows/commit/e332be5ec2048c7a6491407b059339d4b2439a2e) fix(ui): dont show cluster workflows in namespaced mode. Closes #5841 (#5846)
* [c59f59ad0](https://github.com/argoproj/argo-workflows/commit/c59f59ad0e7609cf8b87d6733f73efa9ccf44484) feat: Support Arguments in Exit Handler (#5455)
* [5ff48bbc5](https://github.com/argoproj/argo-workflows/commit/5ff48bbc5c1b1a4589bdad9abacb7b64a37abfe1) feat: Allow to toggle GZip implementations in docker executor (#5698)
* [86545f63e](https://github.com/argoproj/argo-workflows/commit/86545f63e48007684e229c6f35a7dac436d0c1a8) 5739 (#5797)
* [461b0b3cd](https://github.com/argoproj/argo-workflows/commit/461b0b3cda111da1461c217d4a375c9e8a6fba50) fix(executor): Fix artifactory saving files. Fixes #5733 (#5775)
* [507b92cf9](https://github.com/argoproj/argo-workflows/commit/507b92cf93337e18e3f64716081a797e0f60973e) feat(cli): resubmit workflows by label and field selector (#5807)
* [bdd44c723](https://github.com/argoproj/argo-workflows/commit/bdd44c723a324d1c20bcc97f53022b586bfb8348) fix: Add note about hyphenated variables (#5805)
* [b9a79e065](https://github.com/argoproj/argo-workflows/commit/b9a79e065bffb5f442e185767074d1b616ae2aa7) feat(cli): Retry workflows by label selector and field selector (#5795)
* [8f2acee32](https://github.com/argoproj/argo-workflows/commit/8f2acee32e9921241a4e91eee2da4a9e8b5f3f44) fix: Node set updating global output parameter updates global. #5699 (#5700)
* [076ff18a8](https://github.com/argoproj/argo-workflows/commit/076ff18a804bbd3b4aba67024ac73dae82c2f049) feat(controller): Add validation for ContainerSet (#5758)
* [4b3a30f4e](https://github.com/argoproj/argo-workflows/commit/4b3a30f4e7e320538d256adb542715813a5a716d) fix: Reset workflow started time to current when retrying workflow. Fixes #5796 (#5798)
* [4af011318](https://github.com/argoproj/argo-workflows/commit/4af01131889a48989db0c251b8d9711e19ca3325) fix: change log level to warn level (#5790)
* [7e974dcda](https://github.com/argoproj/argo-workflows/commit/7e974dcda79049cbc931169e7134e113bcea5be8) fix(docs): Fix yaml snippet (#5788)
* [4a55e6f0b](https://github.com/argoproj/argo-workflows/commit/4a55e6f0bce53e47066cef75f7aca6c10fd490d6) feat: Support bucket lifecycle for OSS artifact driver (#5731)
* [3cdb22a1e](https://github.com/argoproj/argo-workflows/commit/3cdb22a1e18d02a91391c5282bba857ba3342ba6) feat: Emit WorkflowNodeRunning Event (#5531)
* [66c770993](https://github.com/argoproj/argo-workflows/commit/66c7709937f84cd6c21d92b8e95871b83d808e72) upgrade github.com/gogo/protobuf (#5780)
* [cb55cba07](https://github.com/argoproj/argo-workflows/commit/cb55cba07394cfaf44ae7180d950770c6880d0cb) fix(ui): Fix an UI dropdown flickering issue (#5772)
* [60a64c825](https://github.com/argoproj/argo-workflows/commit/60a64c8254d406ff85e8f936d6c76da8d7a028e8) feat(cli): Stop workflows by label selector and field selector (#5752)
* [05af5edfc](https://github.com/argoproj/argo-workflows/commit/05af5edfc6931e0ea53b0544de579b7ffd56ee86) fix(ui): Fix the UI crashing issue (#5751)
* [407740046](https://github.com/argoproj/argo-workflows/commit/407740046f853e0cac485e410d276ce60a41f649) fix(ui): Remove the ability to change namespaces via the UI in Managed Namespace Mode. Closes #5577 (#5729)
* [2a050348b](https://github.com/argoproj/argo-workflows/commit/2a050348b17274b3bf64ca0e4ca78f2142d6d62f) fix(ui): Fix workflow summary page unscrollable issue (#5743)
* [500d93387](https://github.com/argoproj/argo-workflows/commit/500d93387c1593f3f2315ec633b9d363c7c21e44) fix(ui): Fix greediness in regex for auth token replacement (#5746)
* [284adfe16](https://github.com/argoproj/argo-workflows/commit/284adfe16aeb11536a1c98f1956fdeb76dac4f1c) fix(server): Fix the issue where GetArtifact didn't look for input artifacts (#5705)
* [511bbed2b](https://github.com/argoproj/argo-workflows/commit/511bbed2b35abad5144a99234f48f4dc03b3a97e) fix(ui): Fix workflow list table column width mismatch (#5736)
* [2b8740943](https://github.com/argoproj/argo-workflows/commit/2b87409431bb778a4264296bea2fd4173d00651d) fix(executor): Remove unnecessary check on resource group (#5723)
* [dba2c044e](https://github.com/argoproj/argo-workflows/commit/dba2c044e6d471f65dec868ff2453b90c088bd3e) fix: Only save memoization cache when node succeeded (#5711)
* [8e9e6d676](https://github.com/argoproj/argo-workflows/commit/8e9e6d6760bc0dff260aef4296eac61e6d0bc72f) fix(controller): Fix cron timezone support. Fixes #5653 (#5712)
* [0a6f2fc3a](https://github.com/argoproj/argo-workflows/commit/0a6f2fc3a8271e1a1d168100f0e12a6414114f5b) fix(ui): Fix `showWorkflows` button. Fixes #5645 (#5693)
* [f96355631](https://github.com/argoproj/argo-workflows/commit/f963556312548edc38000b5c6ba36c8ed1c92d63) fix(ui): Fix YAML/JSON toggle. Fixes #5690 (#5694)
* [b267e3cf8](https://github.com/argoproj/argo-workflows/commit/b267e3cf88d084d3dda10307af673753ac73b3af) fix(cli): Validate cron on update. Fixes #5691 (#5692)
* [9a872de13](https://github.com/argoproj/argo-workflows/commit/9a872de13929af14cb2488b98e211ca857d4ee67) fix(executor): Ignore not existing metadata. Fixes #5656 (#5695)
* [91c08cdd8](https://github.com/argoproj/argo-workflows/commit/91c08cdd83386bfcf48fcb237dd05216bc61b7a0) fix(executor): More logs for PNS sidecar termination. #5627 (#5683)
* [f6be5691e](https://github.com/argoproj/argo-workflows/commit/f6be5691e5a25d3f82c708d0bb5bb2f099ab8966) fix(controller): Correct bug for repository ref without default key. Fixes #5646 (#5660)
* [e3d1d1e82](https://github.com/argoproj/argo-workflows/commit/e3d1d1e822c01e2765bab2d57d9537849cd0f720) feat(controller): Allow to disable leader election (#5638) (#5648)
* [860739147](https://github.com/argoproj/argo-workflows/commit/8607391477e816e6e685fa5719c0d3c55ff1bc00) feat(cli): Add offline linting (#5569)
* [a01852364](https://github.com/argoproj/argo-workflows/commit/a01852364ba6c4208146ef676c5918dc3faa1b18) feat(ui): Support expression evaluation in links (#5666)
* [24ac7252d](https://github.com/argoproj/argo-workflows/commit/24ac7252d27454b8f6d0cca02201fe23a35dd915) fix(executor): Correctly surface error when resource is deleted during status checking (#5675)
* [1d367ddfd](https://github.com/argoproj/argo-workflows/commit/1d367ddfd48d8d17b48cca83da9454cee5c6463f) fix(ui): strip inner quotes from argoToken (#5677)
* [bf5d7bfab](https://github.com/argoproj/argo-workflows/commit/bf5d7bfab2d6dde057f3e79e5d0a2fb490a621ee) fix: Increase Name width to 3 and decrease NameSpace width to 1 (#5678)
* [71dfc7974](https://github.com/argoproj/argo-workflows/commit/71dfc797425976e8b013d2b3e1daf46aa6ce04cf) feat(ui): support any yaml reference in link (#5667)
* [ec3b82d92](https://github.com/argoproj/argo-workflows/commit/ec3b82d92ce0f9aba6cfb524b48a6400585441f8) fix: git clone on non-default branch fails (Fixes #5629) (#5630)
* [d5e492c2a](https://github.com/argoproj/argo-workflows/commit/d5e492c2a2f2b5fd65d11c625f628ed75aa8a8ff) fix(executor):Failure node failed to get archived log (#5671)
* [b7d69053d](https://github.com/argoproj/argo-workflows/commit/b7d69053dba478327b926041094349b7295dc499) fix(artifacts): only retry on transient S3 errors (#5579)
* [defbd600e](https://github.com/argoproj/argo-workflows/commit/defbd600e37258c8cdf30f64d4da9f4563eb7901) fix: Default ARGO_SECURE=true. Fixes #5607 (#5626)
* [46ec3028c](https://github.com/argoproj/argo-workflows/commit/46ec3028ca4299deff4966e647857003a89a3d66) fix: Make task/step name extractor robust (#5672)
* [88917cbd8](https://github.com/argoproj/argo-workflows/commit/88917cbd81b5da45c840645ae156baa7afcb7bb6) fix: Surface error during wait timeout for OSS artifact driver API calls (#5601)
* [b76fac754](https://github.com/argoproj/argo-workflows/commit/b76fac754298d0602a2da9902bafa2764e7f6bae) fix(ui): Fix editor. Fixes #5613 Fixes #5617 (#5620)
* [9d175cf9b](https://github.com/argoproj/argo-workflows/commit/9d175cf9b9e0bd57e11ec4e4cce60a6d354ace05) fix(ui): various ui fixes (#5606)
* [b4ce78bbe](https://github.com/argoproj/argo-workflows/commit/b4ce78bbef054e2f4f659e48459eec08a4addf97) feat: Identifiable user agents in various Argo commands (#5624)
* [22a8e93c8](https://github.com/argoproj/argo-workflows/commit/22a8e93c8b52889e9119e6d15d1a9bcc6ae8134a) feat(executor): Support accessing output parameters by PNS executor running as non-root (#5564)
* [2baae1dc2](https://github.com/argoproj/argo-workflows/commit/2baae1dc2fdf990530e62be760fc2ba4104fc286) add -o short option for argo cli get command (#5533)
* [0edd32b5e](https://github.com/argoproj/argo-workflows/commit/0edd32b5e8ae3cbeaf6cb406d7344ff4801d36ba) fix(controller): Workflow hangs indefinitely during ContainerCreating if the Pod or Node unexpectedly dies (#5585)
* [d0a0289ee](https://github.com/argoproj/argo-workflows/commit/d0a0289eea79527d825a10c35f8a9fcbaee29877) feat(ui): let workflow dag and node info scroll independently (#5603)
* [2651bd619](https://github.com/argoproj/argo-workflows/commit/2651bd6193acc491f4a20b6e68c082227f9e60f6) fix: Improve error message when missing required fields in resource manifest (#5578)
* [4f3bbdcbc](https://github.com/argoproj/argo-workflows/commit/4f3bbdcbc9c57dae6c2ce2b93f0395230501f749) feat: Support security token for OSS artifact driver (#5491)
* [9b6c8b453](https://github.com/argoproj/argo-workflows/commit/9b6c8b45321c958b2055236b18449ba6db802878) fix: parse username from git url when using SSH key auth (#5156)
* [7276bc399](https://github.com/argoproj/argo-workflows/commit/7276bc399eae7e318d1937b7b02f86fbe812f9e3) fix(controller): Consider nested expanded task in reference (#5594)
* [4e450e250](https://github.com/argoproj/argo-workflows/commit/4e450e250168e6b4d51a126b784e90b11a0162bc) fix: Switch InsecureSkipVerify to true (#5575)
* [ed54f158d](https://github.com/argoproj/argo-workflows/commit/ed54f158dd8b0b3cee5ba24d703e7de3552ea52d) fix(controller): clean up before insert into argo_archived_workflows_labels (#5568)
* [2b3655ecb](https://github.com/argoproj/argo-workflows/commit/2b3655ecb117beb14bf6dca62b2610fb3ee33283) fix: Remove invalid label value for last hit timestamp from caches (#5528)
* [2ba0a4369](https://github.com/argoproj/argo-workflows/commit/2ba0a4369af0860975250b5fd3d81c563822a6a1) fix(executor): GODEBUG=x509ignoreCN=0 (#5562)
* [3c3754f98](https://github.com/argoproj/argo-workflows/commit/3c3754f983373189ad6d2252b251152e7cba1cf0) fix: Build static files in engineering builds (#5559)
* [23ccd9cf3](https://github.com/argoproj/argo-workflows/commit/23ccd9cf3730e20cd49d37ec5540fea533713898) fix(cli): exit when calling subcommand node without args (#5556)
* [aa0494859](https://github.com/argoproj/argo-workflows/commit/aa0494859341b02189f61561ab4f20ee91718d34) fix: Reference new argo-workflows url in in-app links (#5553)
* [20f00470e](https://github.com/argoproj/argo-workflows/commit/20f00470e8177a89afd0676cedcfb8dac39b34de) fix(server): Disable CN check (Go 15 does not support). Fixes #5539 (#5550)
* [872897ff9](https://github.com/argoproj/argo-workflows/commit/872897ff964df88995410cf2e7f9249439cf7461) fix: allow mountPaths with traling slash (#5521)
* [4c3b0ac53](https://github.com/argoproj/argo-workflows/commit/4c3b0ac530acaac22abb453df3de09e8c74068fb) fix(controller): Enable metrics server on stand-by  controller (#5540)
* [76b6a0eff](https://github.com/argoproj/argo-workflows/commit/76b6a0eff9345ff18f34ba3b2c44847c317293fb) feat(controller): Add last hit timestamp to memoization caches (#5487)
* [a61d84cc0](https://github.com/argoproj/argo-workflows/commit/a61d84cc05b86719d1b2704ea1524afef5bbb9b5) fix: Default to insecure mode when no certs are present (#5511)
* [4a1caca1e](https://github.com/argoproj/argo-workflows/commit/4a1caca1e52b0be87f5a1e05efc240722f2a4a49) fix: add softonic as user (#5522)
* [bbdf651b7](https://github.com/argoproj/argo-workflows/commit/bbdf651b790a0b432d800362210c0f4f072922f6) fix: Spelling Mistake (#5507)
* [b8af3411b](https://github.com/argoproj/argo-workflows/commit/b8af3411b17b5ab4b359852a66ecfc6999fc0da8) fix: avoid short names in deployment manifests (#5475)
* [d964fe448](https://github.com/argoproj/argo-workflows/commit/d964fe4484c6ad4a313deb9994288d402a543018) fix(controller): Use node.Name instead of node.DisplayName for onExit nodes (#5486)
* [80cea6a36](https://github.com/argoproj/argo-workflows/commit/80cea6a3679fa87983643defb6681881228043ae) fix(ui): Correct Argo Events swagger (#5490)
* [865b1fe8b](https://github.com/argoproj/argo-workflows/commit/865b1fe8b501526555e3518410836e277d04184c) fix(executor): Always poll for Docker injected sidecars. Resolves #5448 (#5479)
* [c13755b16](https://github.com/argoproj/argo-workflows/commit/c13755b1692c376468554c20a8fa3f5efd18d896) fix: avoid short names in Dockerfiles (#5474)
* [beb0f26be](https://github.com/argoproj/argo-workflows/commit/beb0f26bed9d33d42d9153fdd4ffd24e7fe62ffd) fix: Add logging to aid troubleshooting (#5501)
* [306594164](https://github.com/argoproj/argo-workflows/commit/306594164ab46d31ee1e7b0d7d773a857b52bdde) fix: Run controller as un-privileged (#5460)
* [2a099f8ab](https://github.com/argoproj/argo-workflows/commit/2a099f8abf97f5be27738e93f76a3cb473622763) fix: certs in non-root (#5476)
* [4eb351cba](https://github.com/argoproj/argo-workflows/commit/4eb351cbaf82bbee5903b91b4ef094190e1e0134) fix(ui): Multiple UI fixes (#5498)
* [dfe6ceb43](https://github.com/argoproj/argo-workflows/commit/dfe6ceb430d2bd7c13987624105450a0994e08fc) fix(controller): Fix workflows with retryStrategy left Running after completion (#5497)
* [ea26a964b](https://github.com/argoproj/argo-workflows/commit/ea26a964b7dffed2fe147db69ccce5c5f542c308) fix(cli): Linting improvements (#5224)
* [513756ebf](https://github.com/argoproj/argo-workflows/commit/513756ebff2d12c1938559a3109d3d13211cd14a) fix(controller): Only set global parameters after workflow validation succeeded to avoid panics (#5477)
* [9a1c046ee](https://github.com/argoproj/argo-workflows/commit/9a1c046ee4e2a2cabc3e358cf8093e71dd8d4090) fix(controller): Enhance output capture (#5450)
* [46aaa700e](https://github.com/argoproj/argo-workflows/commit/46aaa700ebab322e112fa0b54cde96fb2b865ea9) feat(server): Disable Basic auth if server is not running in client mode (#5401)
* [e638981bf](https://github.com/argoproj/argo-workflows/commit/e638981bf0542acc9ee57820849ee569d0dcc91f) fix(controller): Add permissions to create/update configmaps for memoization in quick-start manifests (#5447)
* [b01ca3a1d](https://github.com/argoproj/argo-workflows/commit/b01ca3a1d5f764c8366afb6e31a7de9009880f6b) fix(controller): Fix the issue of {{retries}} in PodSpecPatch not being updated (#5389)
* [72ee1cce9](https://github.com/argoproj/argo-workflows/commit/72ee1cce9e5ba874f3cb84fe1483cb28dacdee45) fix: Set daemoned nodes to Succeeded when boudary ends (#5421)
* [d9f201001](https://github.com/argoproj/argo-workflows/commit/d9f201001bb16b0610e2534515b4aadf38e6f2b2) fix(executor): Ignore non-running Docker kill errors (#5451)
* [7e4e1b78c](https://github.com/argoproj/argo-workflows/commit/7e4e1b78c9be52066573c915aba45d30edff1765) feat: Template defaults (#5410)
* [440a68976](https://github.com/argoproj/argo-workflows/commit/440a689760b56e35beaf3eeb22f276ef71a68743) fix: Fix getStepOrDAGTaskName (#5454)
* [8d2006181](https://github.com/argoproj/argo-workflows/commit/8d20061815b1021558c2f8cca6b3b04903781b5a) fix: Various UI fixes (#5449)
* [2371a6d3f](https://github.com/argoproj/argo-workflows/commit/2371a6d3f49f0c088074a8829e37463d99fc7acc) fix(executor): PNS support artifacts for short-running containers (#5427)
* [07ef0e6b8](https://github.com/argoproj/argo-workflows/commit/07ef0e6b876fddef6e48e889fdfd471af50864a5) fix(test): TestWorkflowTemplateRefWithShutdownAndSuspend flakiness (#5441)
* [c16a471cb](https://github.com/argoproj/argo-workflows/commit/c16a471cb9927248ba84400ec45763f014ec6a3b) fix(cli): Only append parse result when not nil to avoid panic (#5424)
* [8f03970be](https://github.com/argoproj/argo-workflows/commit/8f03970bea3749c0b338dbf533e81ef02c597100) fix(ui): Fix link button. Fixes #5429 (#5430)
* [f4432043c](https://github.com/argoproj/argo-workflows/commit/f4432043c5c1c26612e235bb7069e5c86ec2d050) fix(executor): Surface error when wait container fails to establish pod watch (#5372)
* [d71786571](https://github.com/argoproj/argo-workflows/commit/d717865716ea399284c6193ceff9970e66bc5f45) feat(executor): Move exit code capture to controller. See #5251 (#5328)
* [04f3a957b](https://github.com/argoproj/argo-workflows/commit/04f3a957be7ad9a1f99183c18a900264cc524ed8) fix(test): Fix TestWorkflowTemplateRefWithShutdownAndSuspend flakyness (#5418)
* [ed957dd9c](https://github.com/argoproj/argo-workflows/commit/ed957dd9cf257b1db9a71dcdca49fc38678a4dcb) feat(executor): Switch to use SDK and poll-based resource status checking (#5364)
* [d3eeddb1f](https://github.com/argoproj/argo-workflows/commit/d3eeddb1f5672686d349da7f99517927cad04953) feat(executor) Add injected sidecar support to Emissary (#5383)
* [189b6a8e3](https://github.com/argoproj/argo-workflows/commit/189b6a8e3e0b0d4601d00417b9d205f3c1f77250) fix: Do not allow cron workflow names with more than 52 chars (#5407)
* [8e137582c](https://github.com/argoproj/argo-workflows/commit/8e137582cc41465f07226f8ab0191bebf3c11106) feat(executor): Reduce poll time 3s to 1s for PNS and Emissary executors (#5386)
* [b24aaeaff](https://github.com/argoproj/argo-workflows/commit/b24aaeaffd2199794dc0079a494aac212b6e83a5) feat: Allow transient errors in StopWorkflow() (#5396)
* [1ec7ac0fa](https://github.com/argoproj/argo-workflows/commit/1ec7ac0fa0155f936a407887117c8496bba42241) fix(controller): Fixed wrong error message (#5390)
* [4b7e3513e](https://github.com/argoproj/argo-workflows/commit/4b7e3513e72d88c0f20cbb0bfc659bd16ef2a629) fix(ui): typo (#5391)
* [982e5e9df](https://github.com/argoproj/argo-workflows/commit/982e5e9df483e0ce9aa43080683fabadf54e83f2) fix(test): TestWorkflowTemplateRefWithShutdownAndSuspend flaky (#5381)
* [57c05dfab](https://github.com/argoproj/argo-workflows/commit/57c05dfabb6d5792c29b4d19a7b4733dc4354388) feat(controller): Add failFast flag to DAG and Step templates (#5315)
* [fcb098995](https://github.com/argoproj/argo-workflows/commit/fcb098995e4703028e09e580cb3909986a65a595) fix(executor): Kill injected sidecars. Fixes #5337 (#5345)
* [1f7cf1e3b](https://github.com/argoproj/argo-workflows/commit/1f7cf1e3b31d06d0a4bf32ed0ac1fd0e3ae77262) feat: Add helper functions to expr when parsing metadata. Fixes #5351 (#5374)
* [d828717c5](https://github.com/argoproj/argo-workflows/commit/d828717c51f9ba4275c47d5878b700d7477dcb7b) fix(controller): Fix `podSpecPatch` (#5360)
* [2d331f3a4](https://github.com/argoproj/argo-workflows/commit/2d331f3a47a8bc520873f4a4fc95d42efe995d35) fix: Fix S3 file loading (#5353)
* [9faae18a1](https://github.com/argoproj/argo-workflows/commit/9faae18a1d2d7c890510e01abc18402ac9dccc1b) fix(executor): Make docker executor more robust. (#5363)
* [e0f71f3af](https://github.com/argoproj/argo-workflows/commit/e0f71f3af750064d86c1a5de658db75572f12a01) fix(executor): Fix resource patch when not providing flags. Fixes #5310 (#5311)
* [94e155b08](https://github.com/argoproj/argo-workflows/commit/94e155b0839edf2789175624dac46d38bdd424ee) fix(controller): Correctly log pods/exec call (#5359)
* [80b5ab9b8](https://github.com/argoproj/argo-workflows/commit/80b5ab9b8e35b4dba71396062abe32918cd76ddd) fix(ui): Fix container-set log viewing in UI (#5348)
* [bde9f217e](https://github.com/argoproj/argo-workflows/commit/bde9f217ee19f69230a7ad2d256b86b4b6c28f58) fix: More Makefile fixes (#5347)
* [849a5f9aa](https://github.com/argoproj/argo-workflows/commit/849a5f9aaa75f6ee363708113dae32ce6bc077c9) fix: Ensure release images are 'clean' (#5344)
* [23b8c0319](https://github.com/argoproj/argo-workflows/commit/23b8c031965d5f4bae4bb8f3134a43eec975d6ab) fix: Ensure DEV_BRANCH is correct (#5343)
* [ba949c3a6](https://github.com/argoproj/argo-workflows/commit/ba949c3a64e203197dee4f1d9837c47a993132b6) fix(executor): Fix container set bugs (#5317)
* [9d2e9615e](https://github.com/argoproj/argo-workflows/commit/9d2e9615e4cf7739aabb1df4601265b078d98738) feat: Support structured JSON logging for controller, executor, and server (#5158)
* [7fc1f2f24](https://github.com/argoproj/argo-workflows/commit/7fc1f2f24ebeaba2779140fbc17a4d9745860d62) fix(test): Flaky TestWorkflowShutdownStrategy  (#5331)
* [3dce211c5](https://github.com/argoproj/argo-workflows/commit/3dce211c54e6c54cf55819486133f1d2617bd13b) fix: Only retry on transient errors for OSS artifact driver (#5322)
* [8309fd831](https://github.com/argoproj/argo-workflows/commit/8309fd83169e3540123e44c9f2d427ff34cea393) fix: Minor UI fixes (#5325)
* [67f8ca27b](https://github.com/argoproj/argo-workflows/commit/67f8ca27b323aa9fe3eac7e5ece9fc5b2969f4fd) fix: Disallow object names with more than 63 chars (#5324)
* [b048875dc](https://github.com/argoproj/argo-workflows/commit/b048875dc55aba9bb07d7ee6ea2f6290b82798e6) fix(executor): Delegate PNS wait to K8SAPI executor. (#5307)
* [a5d1accff](https://github.com/argoproj/argo-workflows/commit/a5d1accffcd48c1a666f0c733787087f26d58b87) fix(controller): shutdownstrategy on running workflow (#5289)
* [112378fc7](https://github.com/argoproj/argo-workflows/commit/112378fc70818d45ef41a6acc909be1934dc99fb) fix: Backward compatible workflowTemplateRef from 2.11.x to  2.12.x (#5314)
* [103bf2bca](https://github.com/argoproj/argo-workflows/commit/103bf2bcaa72f42286ebece1f726d599cbeda088) feat(executor): Configurable retry backoff settings for workflow executor (#5309)
* [2e857f095](https://github.com/argoproj/argo-workflows/commit/2e857f095621c385b2541b2bff89cac7f9debaf8) fix: Makefile target (#5313)
* [1c6775a04](https://github.com/argoproj/argo-workflows/commit/1c6775a04fdf702a666b57dd6e3ddfcd0e4cb238) feat: Track nodeView tab in URL (#5300)
* [dc5bb12e5](https://github.com/argoproj/argo-workflows/commit/dc5bb12e53c22388ae618b8897d1613cacc9f61d) fix: Use ScopedLocalStorage instead of direct localStorage (#5301)
* [a31fd4456](https://github.com/argoproj/argo-workflows/commit/a31fd44560587e9a24f81d7964a855eabd6c1b31) feat: Improve OSS artifact driver usability when load/save directories (#5293)
* [757e0be18](https://github.com/argoproj/argo-workflows/commit/757e0be18e34c5d1c34bba40aa925e0c5264d727) fix(executor): Enhance PNS executor. Resolves #5251 (#5296)
* [78ec644cd](https://github.com/argoproj/argo-workflows/commit/78ec644cd9a30539397dda3359bcf9be91d37767) feat: Conditional Artifacts and Parameters (#4987)
* [1a8ce1f13](https://github.com/argoproj/argo-workflows/commit/1a8ce1f1334e34b09cb4e154e2993ec4fc610b4b) fix(executor): Fix emissary resource template bug (#5295)
* [8729587ee](https://github.com/argoproj/argo-workflows/commit/8729587eec647e3f75181888fa3a23d7f9c1d102) feat(controller): Container set template. Closes #2551 (#5099)
* [e56da57a3](https://github.com/argoproj/argo-workflows/commit/e56da57a3bc5cc926079f656a397b4140a6833f8) fix: Use bucket.ListObjects() for OSS ListObjects() implementation (#5283)
* [b6961ce6f](https://github.com/argoproj/argo-workflows/commit/b6961ce6f9f6cb3bb6c033142fc9c7f304e752bc) fix: Fixes around archiving workflows (#5278)
* [ab68ea4c3](https://github.com/argoproj/argo-workflows/commit/ab68ea4c345c698f61cd36c074cde1dd796c1a11) fix: Correctly log sub-resource Kubernetes API requests (#5276)
* [66fa8da0f](https://github.com/argoproj/argo-workflows/commit/66fa8da0f6cef88e49b6c8112c0ac4b0004e1187) feat: Support ListObjects() for Alibaba OSS artifact driver (#5261)
* [b062cbf04](https://github.com/argoproj/argo-workflows/commit/b062cbf0498592ed27732049dfb2fe2b5c569f14) fix: Fix swapped artifact repository key and ref in error message (#5272)
* [69c40c09a](https://github.com/argoproj/argo-workflows/commit/69c40c09a491fda1a0bc8603aa397f908cc5d968) fix(executor): Fix concurrency error in PNS executor. Fixes #5250 (#5258)
* [9b538d922](https://github.com/argoproj/argo-workflows/commit/9b538d9221d7dd6e4c4640c9c6d8d861e85a038a) fix(executor): Fix docker "created" issue. Fixes #5252 (#5249)
* [07283cda6](https://github.com/argoproj/argo-workflows/commit/07283cda6f2de21865bbad53f731c0530e5d307a) fix(controller): Take labels change into account in SignificantPodChange() (#5253)
* [c4bcabd7c](https://github.com/argoproj/argo-workflows/commit/c4bcabd7c4ae253f8fefcf9a4f143614d1c38e19) fix(controller): Work-around Golang bug. Fixes #5192 (#5230)
* [e6fa41a1b](https://github.com/argoproj/argo-workflows/commit/e6fa41a1b91be2e56884ca16427aaaae4558fa00) feat(controller): Expression template tags. Resolves #4548 & #1293 (#5115)
* [bd4b46cd1](https://github.com/argoproj/argo-workflows/commit/bd4b46cd13d955826c013ec3e58ce8184765c9ea) feat(controller): Allow to modify time related configurations in leader election (#5234)
* [cb9676e88](https://github.com/argoproj/argo-workflows/commit/cb9676e88857193b762b417f2c45b38e2e0967f9) feat(controller): Reused existing workflow informer. Resolves #5202 (#5204)
* [d7dc48c11](https://github.com/argoproj/argo-workflows/commit/d7dc48c111948611b57254cc4d039adfd71cd205) fix(controller): Leader lease shared name improvments (#5218)
* [2d2fba30c](https://github.com/argoproj/argo-workflows/commit/2d2fba30c4aeaf7d57d3b0f4bef62fb89d139805) fix(server): Enable HTTPS probe for TLS by default. See #5205 (#5228)
* [fb19af1cf](https://github.com/argoproj/argo-workflows/commit/fb19af1cf9bb065ecb1b57533c8d9f68c6528461) fix: Flakey TestDataTransformationArtifactRepositoryRef (#5226)
* [6412bc687](https://github.com/argoproj/argo-workflows/commit/6412bc687e7a030422163eeb85a6cf3fd74820b8) fix: Do not display pagination warning when there is no pagination (#5221)
* [0c226ca49](https://github.com/argoproj/argo-workflows/commit/0c226ca49e6b709cc2e3a63305ce8676be9117f3) feat: Support for data sourcing and transformation with `data` template (#4958)
* [01d310235](https://github.com/argoproj/argo-workflows/commit/01d310235a9349e6d552c758964cc2250a9e9616) chore(server)!: Required authentication by default. Resolves #5206 (#5211)
* [694690b0e](https://github.com/argoproj/argo-workflows/commit/694690b0e6211d97f8047597fa5045e84e004ae2) fix: Checkbox is not clickable (#5213)
* [f0e8df07b](https://github.com/argoproj/argo-workflows/commit/f0e8df07b855219866f35f86903e557a10ef260a) fix(controller): Leader Lease Shared Name (#5214)
* [47ac32376](https://github.com/argoproj/argo-workflows/commit/47ac32376d4d75c43106ee16106d819d314c0a2d) fix(controller): Support emissary on Windows (#5203)
* [8acdb1baf](https://github.com/argoproj/argo-workflows/commit/8acdb1baf020adf386528bb33b63715aaf20e724) fix(controller): More emissary minor bugs (#5193)
* [48811117c](https://github.com/argoproj/argo-workflows/commit/48811117c83e041c1bef8db657e0b566a1744b0a) feat(cli): Add cost optimization nudges for Argo CLI (#5168)
* [26ce0c090](https://github.com/argoproj/argo-workflows/commit/26ce0c0909eea5aa437343885569aa9f6fc82f12) fix: Ensure whitespaces is allowed between name and bracket (#5176)
* [2abf08eb4](https://github.com/argoproj/argo-workflows/commit/2abf08eb4de46fbffc44e26a16c9f1ff9d5bd4c5) fix: Consder templateRef when filtering by tag (#5190)
* [23415b2c1](https://github.com/argoproj/argo-workflows/commit/23415b2c1a90d1468912c29051fc8287eb30f84b) fix(executor): Fix emissary bugs (#5187)
* [f5dcd1bd4](https://github.com/argoproj/argo-workflows/commit/f5dcd1bd40668b42fdd6aa1ce92e91a4d684608d) fix: Propagate URL changes to react state (#5188)
* [e5a5f0394](https://github.com/argoproj/argo-workflows/commit/e5a5f0394b535784daa21ad213f454e09f408914) fix(controller): Fix timezone support. Fixes #5181  (#5182)
* [199016a6b](https://github.com/argoproj/argo-workflows/commit/199016a6bed5284df3ec5caebbef9f2d018a2d43) feat(server): Enforce TLS >= v1.2 (#5172)
* [ab361667a](https://github.com/argoproj/argo-workflows/commit/ab361667a8b8c5ccf126eb1c34962c86c1b738d4) feat(controller) Emissary executor.  (#4925)

<details><summary><h3>Contributors</h3></summary>

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

</details>

## v3.0.10 (2021-08-18)

Full Changelog: [v3.0.9...v3.0.10](https://github.com/argoproj/argo-workflows/compare/v3.0.9...v3.0.10)

### Selected Changes

* [0177e73b9](https://github.com/argoproj/argo-workflows/commit/0177e73b962136200517b7f301cd98cfbed02a31) Update manifests to v3.0.10
* [587b17539](https://github.com/argoproj/argo-workflows/commit/587b1753968dd5ab4d8bc7e5e60ee6e9ca8e1b7b) fix: Fix `x509: certificate signed by unknown authority` error (#6566)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins

</details>

## v3.0.9 (2021-08-18)

Full Changelog: [v3.0.8...v3.0.9](https://github.com/argoproj/argo-workflows/compare/v3.0.8...v3.0.9)

### Selected Changes

* [d5fd9f14f](https://github.com/argoproj/argo-workflows/commit/d5fd9f14fc6f55c5d6c1f382081b68e86574d74d) Update manifests to v3.0.9
* [4eb16eaa5](https://github.com/argoproj/argo-workflows/commit/4eb16eaa58ea2de4c4b071c6b3a565dc62e4a07a) fix: Generate TLS Certificates on startup and only keep in memory (#6540)
* [419b7af08](https://github.com/argoproj/argo-workflows/commit/419b7af08582252d6f0722930d026ba728fc19d6) fix: Remove client private key from client auth REST config (#6506)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins
* David Collom

</details>

## v3.0.8 (2021-06-21)

Full Changelog: [v3.0.7...v3.0.8](https://github.com/argoproj/argo-workflows/compare/v3.0.7...v3.0.8)

### Selected Changes

* [6d7887cce](https://github.com/argoproj/argo-workflows/commit/6d7887cce650f999bb6f788a43fcefe3ca398185) Update manifests to v3.0.8
* [449237971](https://github.com/argoproj/argo-workflows/commit/449237971ba81e8397667be77a01957ec15d576e) fix(ui): Fix-up local storage namespaces. Fixes #6109 (#6144)
* [87852e94a](https://github.com/argoproj/argo-workflows/commit/87852e94aa2530ebcbd3aeaca647ae8ff42774ac) fix(controller): dehydrate workflow before deleting offloaded node status (#6112)
* [d8686ee1a](https://github.com/argoproj/argo-workflows/commit/d8686ee1ade34d7d5ef687bcb638415756b2f364) fix(executor): Fix docker not terminating. Fixes #6064 (#6083)
* [c2abdb8e6](https://github.com/argoproj/argo-workflows/commit/c2abdb8e6f16486a0785dc852d293c19bd721399) fix(controller): Handling panic in leaderelection (#6072)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins
* Reijer Copier
* Saravanan Balasubramanian

</details>

## v3.0.7 (2021-05-25)

Full Changelog: [v3.0.6...v3.0.7](https://github.com/argoproj/argo-workflows/compare/v3.0.6...v3.0.7)

### Selected Changes

* [e79e7ccda](https://github.com/argoproj/argo-workflows/commit/e79e7ccda747fa4487bf889142c744457c26e9f7) Update manifests to v3.0.7
* [b6e986c85](https://github.com/argoproj/argo-workflows/commit/b6e986c85f36e6a182bf1e58a992d2e26bce1feb) fix(controller): Increase readiness timeout from 1s to 30s (#6007)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins

</details>

## v3.0.6 (2021-05-24)

Full Changelog: [v3.0.5...v3.0.6](https://github.com/argoproj/argo-workflows/compare/v3.0.5...v3.0.6)

### Selected Changes

* [4a7004d04](https://github.com/argoproj/argo-workflows/commit/4a7004d045e2d8f5f90f9e8caaa5e44c013be9d6) Update manifests to v3.0.6
* [10ecb7e5b](https://github.com/argoproj/argo-workflows/commit/10ecb7e5b1264c283d5b88a214431743c8da3468) fix(controller): Listen on :6060 (#5988)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins

</details>

## v3.0.5 (2021-05-24)

Full Changelog: [v3.0.4...v3.0.5](https://github.com/argoproj/argo-workflows/compare/v3.0.4...v3.0.5)

### Selected Changes

* [98b930cb1](https://github.com/argoproj/argo-workflows/commit/98b930cb1a9f4304f879e33177d1c6e5b45119b7) Update manifests to v3.0.5
* [f893ea682](https://github.com/argoproj/argo-workflows/commit/f893ea682f1c30619195f32b58ebc4499f318d21) feat(controller): Add liveness probe (#5875)
* [e64607efa](https://github.com/argoproj/argo-workflows/commit/e64607efac779113dd57a9925cd06f9017186f63) fix(controller): Empty global output param crashes (#5931)
* [eeb5acba4](https://github.com/argoproj/argo-workflows/commit/eeb5acba4565a178cde119ab92a36b291d0b3bb8) fix(ui): ensure that the artifacts property exists before inspecting it (#5977)
* [49979c2fa](https://github.com/argoproj/argo-workflows/commit/49979c2fa5c08602b56cb21ef5e31594a1a9ddd4) fix(controller): Revert cb9676e88857193b762b417f2c45b38e2e0967f9. Fixes #5852 (#5933)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins
* Saravanan Balasubramanian
* dherman

</details>

## v3.0.4 (2021-05-13)

Full Changelog: [v3.0.3...v3.0.4](https://github.com/argoproj/argo-workflows/compare/v3.0.3...v3.0.4)

### Selected Changes

* [d7ebc548e](https://github.com/argoproj/argo-workflows/commit/d7ebc548e30cccc6b6bfc755f69145147dbe73f2) Update manifests to v3.0.4
* [06744da67](https://github.com/argoproj/argo-workflows/commit/06744da6741dd9d8c6bfec3753bb1532f77e8a7b) fix(ui): Fix workflow summary page unscrollable issue (#5743)
* [d3ed51e7a](https://github.com/argoproj/argo-workflows/commit/d3ed51e7a8528fc8051fe64d1a1fda18d64faa85) fix(controller): Fix pod spec jumbling. Fixes #5897 (#5899)
* [d9e583a12](https://github.com/argoproj/argo-workflows/commit/d9e583a12b9ab684c8f44d5258b65b4d9ff24604) fix: Fix active pods count in node pending status with pod deleted. (#5898)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins
* Radolumbo
* Saravanan Balasubramanian
* dinever

</details>

## v3.0.3 (2021-05-11)

Full Changelog: [v3.0.2...v3.0.3](https://github.com/argoproj/argo-workflows/compare/v3.0.2...v3.0.3)

### Selected Changes

* [e450ea7fa](https://github.com/argoproj/argo-workflows/commit/e450ea7facd6811ecc6b4acc8269e1bbb4db7ab5) Update manifests to v3.0.3
* [80142b120](https://github.com/argoproj/argo-workflows/commit/80142b120dae997ecad52b686fb8944f4fc40239) fix(controller): Error template ref exit handlers. Fixes #5835 (#5837)
* [8a4a3729d](https://github.com/argoproj/argo-workflows/commit/8a4a3729dbe4517bde945709f1dfa3dd5b0333f7) fix(executor): Enable PNS executor to better kill sidecars. Fixes #5779 (#5794)
* [cb8a54793](https://github.com/argoproj/argo-workflows/commit/cb8a547936af509ea07e13673e616c9434dad739) feat(controller): Add config for potential CPU hogs (#5853)
* [702bfb245](https://github.com/argoproj/argo-workflows/commit/702bfb245af90d13e6c0ed0616ab9b0d6cb762ab) 5739 (#5797)
* [a4c246b2b](https://github.com/argoproj/argo-workflows/commit/a4c246b2b5d97f5ab856aafb4c5e00d3b73d6f7e) fix(ui): dont show cluster workflows in namespaced mode. Closes #5841 (#5846)
* [910f552de](https://github.com/argoproj/argo-workflows/commit/910f552defa04396cce9f7e2794f35a2845455e5) fix: certs in non-root (#5476)
* [f6493ac36](https://github.com/argoproj/argo-workflows/commit/f6493ac36223f2771a8da4599bfceafc8465ee60) fix(executor): Fix artifactory saving files. Fixes #5733 (#5775)
* [6c16cec61](https://github.com/argoproj/argo-workflows/commit/6c16cec619cc30187de7385bc7820055e1c5f511) fix(controller): Enable metrics server on stand-by  controller (#5540)
* [b6d703475](https://github.com/argoproj/argo-workflows/commit/b6d7034753fa21ba20637dddd806d17905f1bc56) feat(controller): Allow to disable leader election (#5638) (#5648)
* [0ae8061c0](https://github.com/argoproj/argo-workflows/commit/0ae8061c08809c7d96adcd614812a9000692a11e) fix: Node set updating global output parameter updates global. #5699 (#5700)
* [0d3ad801c](https://github.com/argoproj/argo-workflows/commit/0d3ad801c105e442f61ba3f81fd61d2c6689897d) fix: Reset workflow started time to current when retrying workflow. Fixes #5796 (#5798)
* [e67cb424d](https://github.com/argoproj/argo-workflows/commit/e67cb424dae7cdfc623c67573b959d1c59e2444f) fix: change log level to warn level (#5790)
* [cfd0fad05](https://github.com/argoproj/argo-workflows/commit/cfd0fad05a16d1281056a27e750efb2178b2d068) fix(ui): Remove the ability to change namespaces via the UI in Managed Namespace Mode. Closes #5577
* [d2f53eae3](https://github.com/argoproj/argo-workflows/commit/d2f53eae3bab4b9fc1e5110d044fe4681291a19a) fix(ui): Fix greediness in regex for auth token replacement (#5746)

<details><summary><h3>Contributors</h3></summary>

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

</details>

## v3.0.2 (2021-04-20)

Full Changelog: [v3.0.1...v3.0.2](https://github.com/argoproj/argo-workflows/compare/v3.0.1...v3.0.2)

### Selected Changes

* [38fff9c0e](https://github.com/argoproj/argo-workflows/commit/38fff9c0e0f04663b0ee1e44ae0a3183bed6561d) Update manifests to v3.0.2
* [a43caa577](https://github.com/argoproj/argo-workflows/commit/a43caa5770303abb6d489b4105c2a5b8e7524f4d) fix binary build
* [ca8489988](https://github.com/argoproj/argo-workflows/commit/ca84899881844893de4e8fba729b3d44605804d0) fix: Build argosay binary if it doesn't exist
* [9492e12b0](https://github.com/argoproj/argo-workflows/commit/9492e12b05897e7dacf479b31606ecc9a13a5212) fix(executor): More logs for PNS sidecar termination. #5627 (#5683)
* [c8c7ce3bb](https://github.com/argoproj/argo-workflows/commit/c8c7ce3bb2ff5fdb735bd169926f2efdc2b26ba1) fix: Only save memoization cache when node succeeded (#5711)
* [1ba1d61f1](https://github.com/argoproj/argo-workflows/commit/1ba1d61f123ec2e53f160b4666ee3e6637e0bfe9) fix(controller): Fix cron timezone support. Fixes #5653 (#5712)
* [408d31a5f](https://github.com/argoproj/argo-workflows/commit/408d31a5fa289505beb2db857fc65e0fbb704b91) fix(ui): Fix `showWorkflows` button. Fixes #5645 (#5693)
* [b7b4b3f71](https://github.com/argoproj/argo-workflows/commit/b7b4b3f71383ee339003e3d51749e41307903448) fix(ui): Fix YAML/JSON toggle. Fixes #5690 (#5694)
* [279b78b43](https://github.com/argoproj/argo-workflows/commit/279b78b43da692d98bd86dc532f4bc7ad0a308e2) fix(cli): Validate cron on update. Fixes #5691 (#5692)
* [f7200402f](https://github.com/argoproj/argo-workflows/commit/f7200402fa5cdd4ad88bfcfe04efd763192877de) fix(executor): Ignore not existing metadata. Fixes #5656 (#5695)
* [193f87512](https://github.com/argoproj/argo-workflows/commit/193f8751296db9ae5f1f937cb30757cdf6639152) fix(controller): Correct bug for repository ref without default key. Fixes #5646 (#5660)
* [e20813308](https://github.com/argoproj/argo-workflows/commit/e20813308adec6ea05ee8d01b51b489207fe3b96) fix(ui): strip inner quotes from argoToken (#5677)
* [493e5d656](https://github.com/argoproj/argo-workflows/commit/493e5d656fd27f48c14f1a232770532d629edbd9) fix: git clone on non-default branch fails (Fixes #5629) (#5630)
* [f8ab29b4b](https://github.com/argoproj/argo-workflows/commit/f8ab29b4bd8af591154b01da6dc269f8159c282f) fix: Default ARGO_SECURE=true. Fixes #5607 (#5626)
* [49a4926d1](https://github.com/argoproj/argo-workflows/commit/49a4926d15d7fc76b7a79b99beded78cbb1d20ab) fix: Make task/step name extractor robust (#5672)
* [0cea6125e](https://github.com/argoproj/argo-workflows/commit/0cea6125ec6b03e609741dac861b6aabf4844849) fix: Surface error during wait timeout for OSS artifact driver API calls (#5601)
* [026c12796](https://github.com/argoproj/argo-workflows/commit/026c12796b5ea1abfde9c8f59c2cc0836b8044fe) fix(ui): Fix editor. Fixes #5613 Fixes #5617 (#5620)
* [dafa98329](https://github.com/argoproj/argo-workflows/commit/dafa9832920fc5d6b711d88f182d277b76a5c930) fix(ui): various ui fixes (#5606)
* [c17e72e8b](https://github.com/argoproj/argo-workflows/commit/c17e72e8b00126abc972a6fd16b5cadbbbe87523) fix(controller): Workflow hangs indefinitely during ContainerCreating if the Pod or Node unexpectedly dies (#5585)
* [3472b4f5f](https://github.com/argoproj/argo-workflows/commit/3472b4f5ffd345bed318433318a3c721ea0fd62b) feat(ui): let workflow dag and node info scroll independently (#5603)
* [f6c47e4b7](https://github.com/argoproj/argo-workflows/commit/f6c47e4b7a2d33ba5d994d4756270b678ea018fb) fix: parse username from git url when using SSH key auth (#5156)
* [5bc28dee2](https://github.com/argoproj/argo-workflows/commit/5bc28dee20d0439fb50fdd585af268501f649390) fix(controller): Consider nested expanded task in reference (#5594)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins
* Iven
* Michael Ruoss
* Saravanan Balasubramanian
* Simon Behar
* Vladimir Ivanov
* Yuan Tang
* kennytrytek
* tczhao

</details>

## v3.0.1 (2021-04-01)

Full Changelog: [v3.0.0...v3.0.1](https://github.com/argoproj/argo-workflows/compare/v3.0.0...v3.0.1)

### Selected Changes

* [a8c7d54c4](https://github.com/argoproj/argo-workflows/commit/a8c7d54c47b8dc08fd94d8347802d8d0604b09c3) Update manifests to v3.0.1
* [65250dd68](https://github.com/argoproj/argo-workflows/commit/65250dd68c6d9f3b2262197dd6a9d1402057da24) fix: Switch InsecureSkipVerify to true (#5575)
* [0de125ac3](https://github.com/argoproj/argo-workflows/commit/0de125ac3d3d36f7b9f8a18a86b62706c9a442d2) fix(controller): clean up before insert into argo_archived_workflows_labels (#5568)
* [f05789459](https://github.com/argoproj/argo-workflows/commit/f057894594b7f55fb19feaf7bfc386e6c7912f05) fix(executor): GODEBUG=x509ignoreCN=0 (#5562)
* [bda3af2e5](https://github.com/argoproj/argo-workflows/commit/bda3af2e5a7b1dda403c14987eba4e7e867ea8f5) fix: Reference new argo-workflows url in in-app links (#5553)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins
* BOOK
* Simon Behar
* Tim Collins

</details>

## v3.0.0 (2021-03-29)

Full Changelog: [v3.0.0-rc9...v3.0.0](https://github.com/argoproj/argo-workflows/compare/v3.0.0-rc9...v3.0.0)

### Selected Changes

* [46628c88c](https://github.com/argoproj/argo-workflows/commit/46628c88cf7de2f1e0bcd5939a91e4ce1592e236) Update manifests to v3.0.0
* [3089d8a2a](https://github.com/argoproj/argo-workflows/commit/3089d8a2ada5844850e806c89d0574c0635ea43a) fix: Add 'ToBeFailed'
* [5771c60e6](https://github.com/argoproj/argo-workflows/commit/5771c60e67da3082eb856a4c1a1c5bdf586b4c97) fix: Default to insecure mode when no certs are present (#5511)
* [c77f1eceb](https://github.com/argoproj/argo-workflows/commit/c77f1eceba89b5eb27c843d712d9d0022b05cd63) fix(controller): Use node.Name instead of node.DisplayName for onExit nodes (#5486)
* [0e91e5f13](https://github.com/argoproj/argo-workflows/commit/0e91e5f13d1886f0c99062351681017d20067ec9) fix(ui): Correct Argo Events swagger (#5490)
* [aa07d93a2](https://github.com/argoproj/argo-workflows/commit/aa07d93a2e9ddd139705829c85d19662ac07b43a) fix(executor): Always poll for Docker injected sidecars. Resolves #5448 (#5479)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins
* Saravanan Balasubramanian
* Simon Behar

</details>

## v3.0.0-rc9 (2021-03-23)

Full Changelog: [v3.0.0-rc8...v3.0.0-rc9](https://github.com/argoproj/argo-workflows/compare/v3.0.0-rc8...v3.0.0-rc9)

### Selected Changes

* [02b87aa7d](https://github.com/argoproj/argo-workflows/commit/02b87aa7dea873404dc88a91507d8f662465c55f) Update manifests to v3.0.0-rc9
* [0f5a9ad1e](https://github.com/argoproj/argo-workflows/commit/0f5a9ad1e9d630d2d2b5c71b8a66e30041f24fc3) fix(ui): Multiple UI fixes (#5498)
* [ac5f17144](https://github.com/argoproj/argo-workflows/commit/ac5f171440fd0cbec6416319b974af74abf6d41d) fix(controller): Fix workflows with retryStrategy left Running after completion (#5497)
* [3e81ed4c8](https://github.com/argoproj/argo-workflows/commit/3e81ed4c851cdb609d483965f7f0d92678f27be6) fix(controller): Only set global parameters after workflow validation succeeded to avoid panics (#5477)
* [6d70f9cc7](https://github.com/argoproj/argo-workflows/commit/6d70f9cc7801d76c7fa8e80bb04c201be7ed501e) fix: Set daemoned nodes to Succeeded when boudary ends (#5421)
* [de31db412](https://github.com/argoproj/argo-workflows/commit/de31db412713991eb3a97990718ff5aa848f7d02) fix(executor): Ignore non-running Docker kill errors (#5451)
* [f6ada612a](https://github.com/argoproj/argo-workflows/commit/f6ada612aed817ad6f21d02421475358d0efc791) fix: Fix getStepOrDAGTaskName (#5454)
* [586a04c15](https://github.com/argoproj/argo-workflows/commit/586a04c15806422f5abc95980fc61ff1e72d38c0) fix: Various UI fixes (#5449)
* [78939009e](https://github.com/argoproj/argo-workflows/commit/78939009ecc63231dc0ae344db477f1441a9dbd2) fix(executor): PNS support artifacts for short-running containers (#5427)
* [8f0235a01](https://github.com/argoproj/argo-workflows/commit/8f0235a014588f06562fab7cb86501a64067da01) fix(test): TestWorkflowTemplateRefWithShutdownAndSuspend flakiness (#5441)
* [6f1027a1d](https://github.com/argoproj/argo-workflows/commit/6f1027a1d139a7650c5051dfe499012c28bf37b7) fix(cli): Only append parse result when not nil to avoid panic (#5424)
* [5b871adbe](https://github.com/argoproj/argo-workflows/commit/5b871adbe4a7de3183d7a88cb9fcab2189a76f22) fix(ui): Fix link button. Fixes #5429 (#5430)
* [41eaa357d](https://github.com/argoproj/argo-workflows/commit/41eaa357d7ff3c2985eb38725862d037cb2009d3) fix(executor): Surface error when wait container fails to establish pod watch (#5372)
* [f55d41ac8](https://github.com/argoproj/argo-workflows/commit/f55d41ac8495d1fb531c07106faf0c7cf39668a9) fix(test): Fix TestWorkflowTemplateRefWithShutdownAndSuspend flakyness (#5418)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins
* Saravanan Balasubramanian
* Simon Behar
* Yuan Tang

</details>

## v3.0.0-rc8 (2021-03-17)

Full Changelog: [v3.0.0-rc7...v3.0.0-rc8](https://github.com/argoproj/argo-workflows/compare/v3.0.0-rc7...v3.0.0-rc8)

### Selected Changes

* [ff5040016](https://github.com/argoproj/argo-workflows/commit/ff504001640d6e47345ff00b7f3ef14ccec314e9) Update manifests to v3.0.0-rc8
* [50fe7970c](https://github.com/argoproj/argo-workflows/commit/50fe7970c19dc686e752a7b4b8b5db50e16f24c8) fix(server): Enable HTTPS probe for TLS by default. See #5205 (#5228)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins
* Simon Behar

</details>

## v3.0.0-rc7 (2021-03-16)

Full Changelog: [v3.0.0-rc6...v3.0.0-rc7](https://github.com/argoproj/argo-workflows/compare/v3.0.0-rc6...v3.0.0-rc7)

### Selected Changes

* [8049ed820](https://github.com/argoproj/argo-workflows/commit/8049ed820fc45a21acf7c39a35566b1ae53a963b) Update manifests to v3.0.0-rc7
* [c2c441027](https://github.com/argoproj/argo-workflows/commit/c2c4410276c1ef47f1ec4f76a4d1909ea484f3a8) fix(executor): Kill injected sidecars. Fixes #5337 (#5345)
* [701623f75](https://github.com/argoproj/argo-workflows/commit/701623f756bea95fcfcbcae345ea77979925e738) fix(executor): Fix resource patch when not providing flags. Fixes #5310 (#5311)
* [ae34e4d74](https://github.com/argoproj/argo-workflows/commit/ae34e4d74dabe00423d848bc950abdad98263897) fix: Do not allow cron workflow names with more than 52 chars (#5407)
* [4468c26fa](https://github.com/argoproj/argo-workflows/commit/4468c26fa2b0dc6fea2a228265418b12f722352f) fix(test): TestWorkflowTemplateRefWithShutdownAndSuspend flaky (#5381)
* [1ce011e45](https://github.com/argoproj/argo-workflows/commit/1ce011e452c60c643e16e4e3e36033baf90de0f5) fix(controller): Fix `podSpecPatch` (#5360)
* [a4dacde81](https://github.com/argoproj/argo-workflows/commit/a4dacde815116351eddb31c90de2ea5697d2c941) fix: Fix S3 file loading (#5353)
* [452b37081](https://github.com/argoproj/argo-workflows/commit/452b37081fa9687bc37c8fa4f5fb181f469c79ad) fix(executor): Make docker executor more robust. (#5363)
* [83fc1c38b](https://github.com/argoproj/argo-workflows/commit/83fc1c38b215948934b3eb69de56a1f4bee420a3) fix(test): Flaky TestWorkflowShutdownStrategy  (#5331)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins
* Saravanan Balasubramanian
* Simon Behar
* Yuan Tang

</details>

## v3.0.0-rc6 (2021-03-09)

Full Changelog: [v3.0.0-rc5...v3.0.0-rc6](https://github.com/argoproj/argo-workflows/compare/v3.0.0-rc5...v3.0.0-rc6)

### Selected Changes

* [ab611694f](https://github.com/argoproj/argo-workflows/commit/ab611694fd91236ccbfd978834cc3bc1d364e0ac) Update manifests to v3.0.0-rc6
* [309fd1114](https://github.com/argoproj/argo-workflows/commit/309fd1114755401c082a0d8c80a06f6509d25251) fix: More Makefile fixes (#5347)
* [f77340500](https://github.com/argoproj/argo-workflows/commit/f7734050074bb0ddfcb2b2d914ca4014fe84c512) fix: Ensure release images are 'clean' (#5344)
* [ce915f572](https://github.com/argoproj/argo-workflows/commit/ce915f572ef52b50acc0fb758e1e9ca86e2c7308) fix: Ensure DEV_BRANCH is correct (#5343)

<details><summary><h3>Contributors</h3></summary>

* Simon Behar

</details>

## v3.0.0-rc5 (2021-03-08)

Full Changelog: [v3.0.0-rc4...v3.0.0-rc5](https://github.com/argoproj/argo-workflows/compare/v3.0.0-rc4...v3.0.0-rc5)

### Selected Changes

* [3b422776f](https://github.com/argoproj/argo-workflows/commit/3b422776fde866792d16dff25bbe7430d2e08fab) Update manifests to v3.0.0-rc5
* [145847d77](https://github.com/argoproj/argo-workflows/commit/145847d775cd040433a6cfebed5eecbe5b378443) cherry-picked fix(controller): shutdownstrategy on running workflow (#5289)
* [29723f49e](https://github.com/argoproj/argo-workflows/commit/29723f49e221bd0b4897858e6a2e403fb89a1e2c) codegen
* [ec1304654](https://github.com/argoproj/argo-workflows/commit/ec1304654fd199a07dbd08c8690a0f12638b699c) fix: Makefile target (#5313)
* [8c69f4faa](https://github.com/argoproj/argo-workflows/commit/8c69f4faaa456bc55b234b1e92037e01e0359a1d) add json/fix.go
* [4233d0b78](https://github.com/argoproj/argo-workflows/commit/4233d0b7855b8b62c5a64f488f0803735dff1acf) fix: Minor UI fixes (#5325)
* [87b62c085](https://github.com/argoproj/argo-workflows/commit/87b62c0852b179c865066a3325870ebbdf29c99b) fix: Disallow object names with more than 63 chars (#5324)
* [e16bd95b4](https://github.com/argoproj/argo-workflows/commit/e16bd95b438f53c4fb3146cba4595370f579b618) fix(executor): Delegate PNS wait to K8SAPI executor. (#5307)
* [62956be0e](https://github.com/argoproj/argo-workflows/commit/62956be0e1eb9c7c5ec8a33cdda956b9acb37025) fix: Backward compatible workflowTemplateRef from 2.11.x to  2.12.x (#5314)
* [95dd7f4b1](https://github.com/argoproj/argo-workflows/commit/95dd7f4b140e4fdd5c939eaecd00341be4adabdd) feat: Track nodeView tab in URL (#5300)
* [a3c12df51](https://github.com/argoproj/argo-workflows/commit/a3c12df5154dbc8236bf3833157d7d5165ead440) fix: Use ScopedLocalStorage instead of direct localStorage (#5301)
* [f368c32f2](https://github.com/argoproj/argo-workflows/commit/f368c32f299f3361b07c989e6615f592654903d6) fix(executor): Enhance PNS executor. Resolves #5251 (#5296)
* [4b2fd9f7d](https://github.com/argoproj/argo-workflows/commit/4b2fd9f7d3a251840ec283fa320da1b6a43f0aba) fix: Fixes around archiving workflows (#5278)
* [afe2cdb6e](https://github.com/argoproj/argo-workflows/commit/afe2cdb6e6a611707f20736500c359408d6cadef) fix: Correctly log sub-resource Kubernetes API requests (#5276)
* [27956b71c](https://github.com/argoproj/argo-workflows/commit/27956b71c39a7c6042c9df662a438ea8205e76a4) fix(executor): Fix concurrency error in PNS executor. Fixes #5250 (#5258)
* [0a8b8f719](https://github.com/argoproj/argo-workflows/commit/0a8b8f71948d4992cc3e3ebb3fa11e5d37838a59) fix(executor): Fix docker "created" issue. Fixes #5252 (#5249)
* [71d1130d2](https://github.com/argoproj/argo-workflows/commit/71d1130d2b24e1054d8e41b3dfa74762d35ffdf9) fix(controller): Take labels change into account in SignificantPodChange() (#5253)
* [39adcd5f3](https://github.com/argoproj/argo-workflows/commit/39adcd5f3bc36a7a38b4fd15b0eb6c359212da45) fix(controller): Work-around Golang bug. Fixes #5192 (#5230)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins
* Saravanan Balasubramanian
* Simon Behar
* Yuan Tang

</details>

## v3.0.0-rc4 (2021-03-02)

Full Changelog: [v3.0.0-rc3...v3.0.0-rc4](https://github.com/argoproj/argo-workflows/compare/v3.0.0-rc3...v3.0.0-rc4)

### Selected Changes

* [ae5587e97](https://github.com/argoproj/argo-workflows/commit/ae5587e97dad0e4806f7a230672b998fe140a767) Update manifests to v3.0.0-rc4
* [a7ecfc085](https://github.com/argoproj/argo-workflows/commit/a7ecfc085cebd67366aeda62952015789d83198b) feat(controller): Allow to modify time related configurations in leader election (#5234)
* [9b9043a64](https://github.com/argoproj/argo-workflows/commit/9b9043a6483637c01bed703fdf897abd2e4757ab) feat(controller): Reused existing workflow informer. Resolves #5202 (#5204)
* [4e9f6350f](https://github.com/argoproj/argo-workflows/commit/4e9f6350f892266ebf3ac9c65288fd43c0f958d3) fix(controller): Leader lease shared name improvments (#5218)
* [942113933](https://github.com/argoproj/argo-workflows/commit/9421139334d87cd4391e0ee30903e9e1d7f915ba) fix: Do not display pagination warning when there is no pagination (#5221)
* [0891dc2f6](https://github.com/argoproj/argo-workflows/commit/0891dc2f654350c8748a03bd10cca26d3c545ca5) fix: Checkbox is not clickable (#5213)
* [9a1971efd](https://github.com/argoproj/argo-workflows/commit/9a1971efd85c9e4038d6ddf3a364fa12752d315c) fix(controller): Leader Lease Shared Name (#5214)
* [339bf4e89](https://github.com/argoproj/argo-workflows/commit/339bf4e8915933bc42353525e05019fa343b75c2) fix: Ensure whitespaces is allowed between name and bracket (#5176)
* [df032f629](https://github.com/argoproj/argo-workflows/commit/df032f629d17f20ae60840bde393975cf16027d7) fix: Consder templateRef when filtering by tag (#5190)
* [d9d831cad](https://github.com/argoproj/argo-workflows/commit/d9d831cadec897a6f4506aff007e7c6d5de85407) fix: Propagate URL changes to react state (#5188)
* [db6577584](https://github.com/argoproj/argo-workflows/commit/db6577584621ebe0f369f69b4910d180f9964907) fix(controller): Fix timezone support. Fixes #5181  (#5182)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins
* Simon Behar
* Yuan Tang
* Zach Aller

</details>

## v3.0.0-rc3 (2021-02-23)

Full Changelog: [v3.0.0-rc2...v3.0.0-rc3](https://github.com/argoproj/argo-workflows/compare/v3.0.0-rc2...v3.0.0-rc3)

### Selected Changes

* [c0c364c22](https://github.com/argoproj/argo-workflows/commit/c0c364c229e3b72306bd0b73161df090d24e0c31) Update manifests to v3.0.0-rc3
* [a9c420060](https://github.com/argoproj/argo-workflows/commit/a9c42006079398228d6fb666ee9fe5f3e9149499) fix: Specify where in YAML validation error occurred (#5174)
* [4b78a7ee4](https://github.com/argoproj/argo-workflows/commit/4b78a7ee4c2db9e56bf1ff0387c2de3cbe38ebf1) fix: Fix node filters in UI (#5162)
* [d9fb0c30f](https://github.com/argoproj/argo-workflows/commit/d9fb0c30f9aedd4909b7e9c38fc69fb679ddd2f6) feat(controller): Support pod GC strategy based on label selector on pods (#5090)
* [91528cc85](https://github.com/argoproj/argo-workflows/commit/91528cc8526ff1b519ad19d8a3cb92009d4aca90) fix(executor): Do not make unneeded `get pod` when no sidecars (#5161)
* [bec80c868](https://github.com/argoproj/argo-workflows/commit/bec80c86857357a9ba00cc904a90531e477597c1) fix: Better message formating for nodes (#5160)
* [d33b5cc06](https://github.com/argoproj/argo-workflows/commit/d33b5cc0673fe4f66fb63a3ca85d34dfc03c91dc) fix: send periodic keepalive packets on eventstream connections (#5101)
* [0f9b22b6e](https://github.com/argoproj/argo-workflows/commit/0f9b22b6eb20431f4db73c96139808fc4468fc43) fix: Append the error message prior to offloading node status (#5043)
* [4611a1673](https://github.com/argoproj/argo-workflows/commit/4611a167341e922bb1978ed65e5941031769c52d) feat: Support automatically create OSS bucket if not exists (#5133)
* [687479fa4](https://github.com/argoproj/argo-workflows/commit/687479fa4dcf160e293efd3e6199f5e37b523696) feat(controller): Use different container runtime executors for each workflow. Close #4254 (#4998)
* [590df1dca](https://github.com/argoproj/argo-workflows/commit/590df1dcacf557880133e4e8dd5087830d97f815) feat: Add `argo submit --verify` hidden flag. Closes #5136 (#5141)
* [377c5f84c](https://github.com/argoproj/argo-workflows/commit/377c5f84c1c69a2aa7d450fc17a79984dba5ee81) feat: added lint from stdin (#5095)
* [633da2584](https://github.com/argoproj/argo-workflows/commit/633da25843d68ea377ddf35010d9849203d04fb3) feat(server): Write an audit log entry for SSO users (#5145)
* [2ab02d95e](https://github.com/argoproj/argo-workflows/commit/2ab02d95e65ede297040d7e683c7761428d8af72) fix: Revert the unwanted change in example  (#5139)
* [1c7921299](https://github.com/argoproj/argo-workflows/commit/1c79212996312c4b2328b807c74da690862c8e38) fix: Multiple UI fixes (#5140)
* [46538d958](https://github.com/argoproj/argo-workflows/commit/46538d958fae0e689fe24de7261956f8d3bc7bec) feat(ui): Surface result and exit-code outputs (#5137)
* [5e64ec402](https://github.com/argoproj/argo-workflows/commit/5e64ec402805b8de114e9b5cd7fb197eecaaa88e) feat: Build dev-\* branches as engineering builds (#5129)
* [4aa9847e2](https://github.com/argoproj/argo-workflows/commit/4aa9847e25efe424864875ac1b4a7367c916091c) fix(ui): add a tooltip for commonly truncated fields in the events pane (#5062)
* [b1535e533](https://github.com/argoproj/argo-workflows/commit/b1535e533ca513b17589f53d503a1121e0ffc261) feat: Support pgzip as an alternative (de)compression implementation (#5108)

<details><summary><h3>Contributors</h3></summary>

* Alex Collins
* Florian
* Ken Kaizu
* Roi Kramer
* Saravanan Balasubramanian
* Simon Behar
* Yuan Tang
* dherman

</details>

## v3.0.0-rc2 (2021-02-16)

Full Changelog: [v3.0.0-rc1...v3.0.0-rc2](https://github.com/argoproj/argo-workflows/compare/v3.0.0-rc1...v3.0.0-rc2)

### Selected Changes

* [ea3439c91](https://github.com/argoproj/argo-workflows/commit/ea3439c91c9fd0c2a57db0d8a5ccf2b9fb2454a3) Update manifests to v3.0.0-rc2
* [b0685bdd0](https://github.com/argoproj/argo-workflows/commit/b0685bdd08616a0bb909d12f2821fd6e576468eb) fix(executor): Fix S3 policy based auth. Fixes #5110 (#5111)
* [fcf4e9929](https://github.com/argoproj/argo-workflows/commit/fcf4e9929a411a7c6083e67c1c37e9c798e4c7d9) fix: Invalid OpenAPI Spec (Issue 4817) (#4831)
* [19b22f25a](https://github.com/argoproj/argo-workflows/commit/19b22f25a4bfd900752947f695f7a3a1567149ef) feat: Add checker to ensure that env variable doc is up to date (#5091)
* [210080a0c](https://github.com/argoproj/argo-workflows/commit/210080a0c0cb5fc40ec82859cc496a948e30687a) feat(controller): Logs Kubernetes API requests (#5084)
* [2ff4db115](https://github.com/argoproj/argo-workflows/commit/2ff4db115daa4e801da10938ecdb9e27d5810b35) feat(executor): Minimize the number of Kubernetes API requests made by executors (#4954)
* [68979f6e3](https://github.com/argoproj/argo-workflows/commit/68979f6e3dab8225765e166d346502e7e66b0c77) fix: Do not create pods under shutdown strategy (#5055)
* [75d09b0f2](https://github.com/argoproj/argo-workflows/commit/75d09b0f2b48dd87d6562436e220c58dca9e06fa) fix: Synchronization lock handling in Step/DAG Template level (#5081)
* [3b7e373ee](https://github.com/argoproj/argo-workflows/commit/3b7e373eeeb486efa2bef8f722394ef279ba1606) feat(ui): Display pretty cron schedule (#5088)
* [1a0889cf3](https://github.com/argoproj/argo-workflows/commit/1a0889cf3bd2fb3482dd740a929e828744d363b2) fix: Revert "fix(controller): keep special characters in json string when … … 19da392 …use withItems (#4814)" (#5076)
* [893e9c9fe](https://github.com/argoproj/argo-workflows/commit/893e9c9fe1bfc6cb2b3a97debb531614b2b2432a) fix: Prefer to break labels by '-' in UI (#5083)
* [77b23098c](https://github.com/argoproj/argo-workflows/commit/77b23098cf2d361647dd978cbaeaa3628c169a16) fix(controller): Fix creator dashes (#5082)
* [f461b040a](https://github.com/argoproj/argo-workflows/commit/f461b040a537342b996e43989f94d6ac7a3e5205) feat(controller): Add podMetadata field to workflow spec. Resolves #4985 (#5031)
* [3b63e7d85](https://github.com/argoproj/argo-workflows/commit/3b63e7d85257126b7a2098aa72d90fdc47d212b0) feat(controller): Add retry policy to support retry only on transient errors (#4999)
* [21e137bab](https://github.com/argoproj/argo-workflows/commit/21e137bab849a9affb1e0bb0acb4b36ae7663b52) fix(executor): Correct usage of time.Duration. Fixes #5046 (#5049)
* [19a34b1fa](https://github.com/argoproj/argo-workflows/commit/19a34b1fa5c99d9bdfc51b73630c0605a198b8c1) feat(executor): Add user agent to workflow executor (#5014)
* [f31e0c6f9](https://github.com/argoproj/argo-workflows/commit/f31e0c6f92ec5e383d2f32f57a822a518cbbef86) chore!: Remove deprecated fields (#5035)
* [f59d46229](https://github.com/argoproj/argo-workflows/commit/f59d4622990b9d81ce80829431725c43f0a78e16) fix: Invalid URL for API Docs (#5063)
* [daf1a71b4](https://github.com/argoproj/argo-workflows/commit/daf1a71b4602e179796624aadfcdb2acea4af4b8) feat: Allow to specify grace period for pod GC (#5033)
* [26f48a9d9](https://github.com/argoproj/argo-workflows/commit/26f48a9d99932ad608e2614b61b203007433ae90) fix: Use React state to avoid new page load in Workflow view (#5058)
* [a0669b5d0](https://github.com/argoproj/argo-workflows/commit/a0669b5d02e489f234eb396136f3885cec8fa175) fix: Don't allow graph container to have its own scroll (#5056)

<details><summary><h3>Contributors</h3></summary>

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

</details>

## v3.0.0-rc1 (2021-02-08)

Full Changelog: [v2.12.13...v3.0.0-rc1](https://github.com/argoproj/argo-workflows/compare/v2.12.13...v3.0.0-rc1)

### Selected Changes

* [9d0be9081](https://github.com/argoproj/argo-workflows/commit/9d0be9081396d369901f3bdb247a61a8d7af8b32) Update manifests to v3.0.0-rc1
* [425173a28](https://github.com/argoproj/argo-workflows/commit/425173a28057492631590f2fb3b586490d62efb9) fix(cli): Add insecure-skip-verify for HTTP1. Fixes #5008 (#5015)
* [48b669cca](https://github.com/argoproj/argo-workflows/commit/48b669ccab13965900806bd2b1eebcca9b64f975) M is demonstrably not less than 1 in the examples (#5021)
* [5915a2164](https://github.com/argoproj/argo-workflows/commit/5915a216427d2d79d5d54746eede61d4e54f31fe) feat(controller): configurable terminationGracePeriodSeconds (#4940)
* [5824fc6bb](https://github.com/argoproj/argo-workflows/commit/5824fc6bb4fbee74d9016e4da97bc177b4d1f081) Fix golang build (#5039)
* [ef76f729a](https://github.com/argoproj/argo-workflows/commit/ef76f729a853bc8512caa504258462c1ba51630f) feat: DAG render options panel float through scrolling (#5036)
* [b4ea47e05](https://github.com/argoproj/argo-workflows/commit/b4ea47e05dcfe3113f906b252736a18f0c90273c) fix: Skip the Workflow not found error in Concurrency policy (#5030)
* [edbe5bc9e](https://github.com/argoproj/argo-workflows/commit/edbe5bc9eb6256329d6b492921e1ff5fa426dae2) fix(ui): Display all node inputs/output in one tab. Resolves #5027 (#5029)
* [c4e8d1cf2](https://github.com/argoproj/argo-workflows/commit/c4e8d1cf2f42f405c4f4efd80c83b29dde1f1a23) feat(executor): Log `verb kind statusCode` for executor Kubernetes API requests (#4989)
* [d1abcb055](https://github.com/argoproj/argo-workflows/commit/d1abcb05507007676ff12ef97652ca4c8a325ccd) fix: Unmark daemoned nodes after stopping them (#5005)
* [38e98f7ee](https://github.com/argoproj/argo-workflows/commit/38e98f7eecc593b63192c4fcb53d80b06c3cc618) Video (#5019)
* [342caeff5](https://github.com/argoproj/argo-workflows/commit/342caeff5b6126d2bedaf5c6836cd0fe0fc1fca1) fix(ui): Fix event-flow hidden nodes (#5013)
* [d5ccc8e01](https://github.com/argoproj/argo-workflows/commit/d5ccc8e0119c3263e6806b4a13e2fa9ec3fff88c) feat(executor): Upgrade kubectl to v1.19 (#5011)
* [8f5e17ac4](https://github.com/argoproj/argo-workflows/commit/8f5e17ac430a48195cc7695313af6d304a0b6cac) feat: Set CORS headers (#4990)
* [99c049bd2](https://github.com/argoproj/argo-workflows/commit/99c049bd27eb93b3a9719fde9ed8e8c60ca75511) feat(ui): Node search tool in UI Workflow viewer (#5000)
* [5047f0738](https://github.com/argoproj/argo-workflows/commit/5047f07381eb59373db60021ffd13f7a8ca9292e) fix: Fail DAG templates with variables with invalid dependencies (#4992)
* [ccd669e44](https://github.com/argoproj/argo-workflows/commit/ccd669e448bf5d9b39f55421e80dd0db6dbc3a39) fix: Coalesce UI filtering menus (#4972)
* [ce508c896](https://github.com/argoproj/argo-workflows/commit/ce508c8967bbc353d645d1326c9cd77f1335f2b7) feat: Configurable retry backoff settings when retrying API calls (#4979)
* [44a4f7e10](https://github.com/argoproj/argo-workflows/commit/44a4f7e10ce1d88e82d5df86c000b93a422484e2) fix(controller): Revert prepending ExecutorScriptSourcePath which brought a breaking change in args handling (#4884)
* [b68d63eb2](https://github.com/argoproj/argo-workflows/commit/b68d63eb2064be0d0544a6d5997940ba4805f4fa) fix(controller): Adds PNS_PRIVILEGED, fixed termination bug (#4983)
* [d324b43c7](https://github.com/argoproj/argo-workflows/commit/d324b43c7777c500521193ebbdf1223966dfe916) fix: Use button in side panel links (#4977)
* [655c7e253](https://github.com/argoproj/argo-workflows/commit/655c7e253635ecf8b9bb650cbbe36607cb0ad22b) fix: Surface the underlying error on wait timeout. (#4966)
* [a00aa3257](https://github.com/argoproj/argo-workflows/commit/a00aa3257a6f9037c010f2bf6f0ee2c4309eaf5f) fix: Correct usage of wait.ExponentialBackoff (#4962)
* [e00623d61](https://github.com/argoproj/argo-workflows/commit/e00623d614f83afe2aead4bfdf27dc572940bea2) fix(server): Fix missing logs bug (#4960)
* [eabe96376](https://github.com/argoproj/argo-workflows/commit/eabe963761019f2981bfc4967c03a3c6733ce0ee) feat(server): add ServiceAccount info to api/v1/userinfo and ui user tab (#4944)
* [15156d193](https://github.com/argoproj/argo-workflows/commit/15156d1934a3a84f22c97dcd7c4f9fdd16664e4c) Added Astraea (#4855)
* [7404b1f8a](https://github.com/argoproj/argo-workflows/commit/7404b1f8a417a95a57b33d5ad077e0121db447f7) fix(controller): report OOM when wait container OOM (#4930)
* [6166e80c5](https://github.com/argoproj/argo-workflows/commit/6166e80c571783f8acf8e6d7448dac2c11f607b3) feat: Support retry on transient errors during executor status checking (#4946)
* [6e116e46e](https://github.com/argoproj/argo-workflows/commit/6e116e46e3ebc19b757bb7fb65a2d2799fb2cde6) feat(crds): Update CRDs to apiextensions.k8s.io/v1 (#4918)
* [261625324](https://github.com/argoproj/argo-workflows/commit/261625324c531c27353df6377541429a811446ef) feat(server): Add Prometheus metrics. Closes #4751 (#4952)
* [7c69898ed](https://github.com/argoproj/argo-workflows/commit/7c69898ed0df5c12ab48c718c3a4cc33613f7766) fix(cli): Allow full node name in node-field-selector (#4913)
* [c7293062a](https://github.com/argoproj/argo-workflows/commit/c7293062ac0267baa216e32230f8d61823ba7b37) fix(cli): Update the map-reduce example, fix bug. (#4948)
* [e7e51d08a](https://github.com/argoproj/argo-workflows/commit/e7e51d08a9857c5c4e16965cbe20ba4bcb5b6038) feat: Check the workflow is not being deleted for Synchronization workflow (#4935)
* [9d4edaef4](https://github.com/argoproj/argo-workflows/commit/9d4edaef47c2674861d5352e2ae6ecb10bcbb8f1) fix(ui): v3 UI tweaks (#4933)
* [2d73d58a5](https://github.com/argoproj/argo-workflows/commit/2d73d58a5428fa940bf4ef55e161f007b9824475) fix(ui): fix object-editor text render issue (#4921)
* [6e961ec92](https://github.com/argoproj/argo-workflows/commit/6e961ec928ee35e3ae022826f020c9722ad614d6) feat: support K8S json patch (#4908)
* [f872366f3](https://github.com/argoproj/argo-workflows/commit/f872366f3b40fc346266e3ae328bdc25eb2082ec) fix(controller): Report reconciliation errors better (#4877)
* [c8215f972](https://github.com/argoproj/argo-workflows/commit/c8215f972502435e6bc5b232823ecb6df919f952) feat(controller)!: Key-only artifacts. Fixes #3184 (#4618)
* [cd7c16b23](https://github.com/argoproj/argo-workflows/commit/cd7c16b235be369b0e44ade97c71cbe5b6d15f68) fix(ui): objecteditor only runs onChange when values are modified (#4911)
* [ee1f82764](https://github.com/argoproj/argo-workflows/commit/ee1f8276460b287da4df617b5c76a1e05764da3f) fix(ui): Fix workflow refresh bug (#4906)
* [929cd50e4](https://github.com/argoproj/argo-workflows/commit/929cd50e427db88fefff4810d83a4f85fc563de2) fix: Mutex not being released on step completion (#4847)
* [c1f9280a2](https://github.com/argoproj/argo-workflows/commit/c1f9280a204a3e305e378e34acda46d11708140f) fix(ui): UI bug fixes (#4895)
* [25abd1a03](https://github.com/argoproj/argo-workflows/commit/25abd1a03b3f490169220200b9add4da4846ac0b) feat: Support specifying the pattern for transient and retryable errors (#4889)
* [16f25ba09](https://github.com/argoproj/argo-workflows/commit/16f25ba09a87d9c29bee1c7b7aef80ec8424ba1d) Revert "feat(cli): add selector and field-selector option to the stop command. (#4853)"
* [53f7998eb](https://github.com/argoproj/argo-workflows/commit/53f7998ebc88be2db3beedbfbe2ea2f8ae230630) feat(cli): add selector and field-selector option to the stop command. (#4853)
* [1f13241fe](https://github.com/argoproj/argo-workflows/commit/1f13241fe9a7367fa3ebba4006f89b662b912d10) fix(workflow-event-bindings): removing unneeded ':' in protocol (#4893)
* [ecbca6ce7](https://github.com/argoproj/argo-workflows/commit/ecbca6ce7dd454f9df97bc7a6c6ec0b06a09bb0f) fix(ui): Show non-pod nodes (#4890)
* [4a5db1b79](https://github.com/argoproj/argo-workflows/commit/4a5db1b79e98d6ddd9f5cae15d0422624061c0bf) fix(controller): Consider processed retry node in metrics. Fixes #4846 (#4872)
* [dd8c1ba02](https://github.com/argoproj/argo-workflows/commit/dd8c1ba023831e8d127ffc9369b73299fad241b4) feat(controller): optional database migration (#4869)
* [a8e934826](https://github.com/argoproj/argo-workflows/commit/a8e9348261c77cb3b13bef864520128279f2e6b8) feat(ui): Argo Events API and UI. Fixes #888 (#4470)
* [17e79e8a2](https://github.com/argoproj/argo-workflows/commit/17e79e8a2af973711d428d7bb20be16a6aeccceb) fix(controller): make creator label DNS compliant. Fixes #4880 (#4881)
* [2ff11cc98](https://github.com/argoproj/argo-workflows/commit/2ff11cc987f852cd642d45ae058517a817b2b94e) fix(controller): Fix node status when daemon pod deleted but its children nodes are still running (#4683)
* [955a4bb12](https://github.com/argoproj/argo-workflows/commit/955a4bb12a2692b3b447b00558d8d84c7c44f2a9) fix: Do not error on duplicate workflow creation by cron (#4871)
* [622624e81](https://github.com/argoproj/argo-workflows/commit/622624e817705b06d5cb135388063762dd3d8b4f) fix(controller): Add matrix tests for node offload disabled. Resolves #2333 (#4864)
* [f38c9a2d7](https://github.com/argoproj/argo-workflows/commit/f38c9a2d78db061b398583dfc9a86c0da349a290) feat: Expose exitCode to step level metrics (#4861)
* [45c792a59](https://github.com/argoproj/argo-workflows/commit/45c792a59052db20da74713b29bdcd1145fc6748) feat(controller): `k8s_request_total` and `workflow_condition` metrics (#4811)
* [e3320d360](https://github.com/argoproj/argo-workflows/commit/e3320d360a7ba006796ebdb638349153d438dcff) feat: Publish images on Quay.io (#4860)
* [b674aa30b](https://github.com/argoproj/argo-workflows/commit/b674aa30bc1c204a63fd2e34d451f84390cbe7b8) feat: Publish images to Quay.io (#4854)
* [a6301d7c6](https://github.com/argoproj/argo-workflows/commit/a6301d7c64fb27e4ab68209da7ee9718bf257252) refactor: upgrade kube client version to v0.19.6. Fixes #4425, #4791 (#4810)
* [6b3ce5045](https://github.com/argoproj/argo-workflows/commit/6b3ce504508707472d4d31c6c522d1af02104b05) feat: Worker busy and active pod metrics (#4823)
* [53110b61d](https://github.com/argoproj/argo-workflows/commit/53110b61d14a5bdaa5c3b4c12527150dfc40b56a) fix: Preserve the original slice when removing string (#4835)
* [adfa988f9](https://github.com/argoproj/argo-workflows/commit/adfa988f9df64b629e08687737a80f2f6e0a6289) fix(controller): keep special characters in json string when use withItems (#4814)
* [6e158780e](https://github.com/argoproj/argo-workflows/commit/6e158780ef202c9d5fb1cb8161fc57bae80bb763) feat(controller): Retry pod creation on API timeout (#4820)
* [01e6c9d5c](https://github.com/argoproj/argo-workflows/commit/01e6c9d5c87d57611c2f3193d56e8af5e5fc91e7) feat(controller): Add retry on different host (#4679)
* [2243d3497](https://github.com/argoproj/argo-workflows/commit/2243d349781973ee0603c215c284da669a2811d5) fix: Metrics documentation (#4829)
* [f0a315cf4](https://github.com/argoproj/argo-workflows/commit/f0a315cf4353589507a37d5787d2424d65a249f3) fix(crds): Inline WorkflowSteps schema to generate valid OpenAPI spec (#4828)
* [f037fd2b4](https://github.com/argoproj/argo-workflows/commit/f037fd2b4e7bb23dfe1ca0ae793e14b1fab42c36) feat(controller): Adding Eventrecorder on LeaderElection
* [a0024d0d4](https://github.com/argoproj/argo-workflows/commit/a0024d0d4625c8660badff5a7d8eca883e7e2a3e) fix(controller): Various v2.12 fixes. Fixes #4798, #4801, #4806 (#4808)
* [ee59d49d9](https://github.com/argoproj/argo-workflows/commit/ee59d49d91d5cdaaa28a34a73339ecc072f8264e) fix: Memoize Example (Issue 4626) (#4818)
* [b73bd2b61](https://github.com/argoproj/argo-workflows/commit/b73bd2b6179840906ef5d2e0c9cccce987cb069a) feat: Customize workfow metadata from event data (#4783)
* [7e6c799af](https://github.com/argoproj/argo-workflows/commit/7e6c799afc025ecc4a9a861b6e2d36908d9eea41) fix: load all supported authentication plugins for k8s client-go (#4802)
* [78b0bffd3](https://github.com/argoproj/argo-workflows/commit/78b0bffd39ec556182e81374b2328450b8dd2e9b) fix(executor): Do not delete local artifacts after upload. Fixes #4676 (#4697)
* [af03a74fb](https://github.com/argoproj/argo-workflows/commit/af03a74fb334c88493e38ed4cb94f771a97bffc5) refactor(ui): replace node-sass with sass (#4780)
* [4ac436d5c](https://github.com/argoproj/argo-workflows/commit/4ac436d5c7eef4a5fdf93fcb8c6e8a224e236bdd) fix(server): Do not silently ignore sso secret creation error (#4775)
* [442d367b1](https://github.com/argoproj/argo-workflows/commit/442d367b1296722b613dd86658ca0e3764b192ac) feat(controller): unix timestamp support on creationTimestamp var (#4763)
* [9f67b28c7](https://github.com/argoproj/argo-workflows/commit/9f67b28c7f7cc767ff1bfb72eb6c16e46071871a) feat(controller): Rate-limit workflows. Closes #4718 (#4726)
* [aed25fefe](https://github.com/argoproj/argo-workflows/commit/aed25fefe00734de0dfa734860fc7af03dbf62cf) Change argo-server crt/key owner (#4750)
* [fbb4e8d44](https://github.com/argoproj/argo-workflows/commit/fbb4e8d447fec32daf63795a9c7b1d7af3499d46) fix(controller): Support default database port. Fixes #4756 (#4757)
* [69ce2acfb](https://github.com/argoproj/argo-workflows/commit/69ce2acfbef761cd14aefb905aa1e396be9eb21e) refactor(controller): Enhanced pod clean-up scalability (#4728)
* [9c4d735a9](https://github.com/argoproj/argo-workflows/commit/9c4d735a9c01987f093e027332be2da71be85124) feat: Add a minimal prometheus server manifest (#4687)
* [625e3ce26](https://github.com/argoproj/argo-workflows/commit/625e3ce265e17df9315231e82e9a346aba400b14) fix(ui): Remove unused Heebo files. Fixes #4730 (#4739)
* [2e278b011](https://github.com/argoproj/argo-workflows/commit/2e278b011083195c8237522311f1ca94dcba4b59) fix(controller): Fixes resource version misuse. Fixes #4714 (#4741)
* [300db5e62](https://github.com/argoproj/argo-workflows/commit/300db5e628bee4311c1d50c5027abb4af2266564) fix(controller): Requeue when the pod was deleted. Fixes #4719 (#4742)
* [a1f7aedbf](https://github.com/argoproj/argo-workflows/commit/a1f7aedbf21c5930cb507ed495901ae430b10b43) fix(controller): Fixed workflow stuck with mutex lock (#4744)
* [1a7ed7342](https://github.com/argoproj/argo-workflows/commit/1a7ed7342312b658c501ee63ece8cb79d6792f88) feat(controller): Enhanced TTL controller scalability (#4736)
* [7437f4296](https://github.com/argoproj/argo-workflows/commit/7437f42963419e8d84b6da32f780b8be7a120ee0) fix(executor): Always check if resource has been deleted in checkResourceState() (#4738)
* [122c5fd2e](https://github.com/argoproj/argo-workflows/commit/122c5fd2ecd10dfeb3c0695dba7fc680bd5d46f9) fix(executor): Copy main/executor container resources from controller by value instead of reference (#4737)
* [440d732d1](https://github.com/argoproj/argo-workflows/commit/440d732d18c2364fe5d6c74b6e4f14dc437d78fc) fix(ui): Fix YAML for workflows with storedWorkflowTemplateSpec. Fixes #4691 (#4695)
* [ed853eb0e](https://github.com/argoproj/argo-workflows/commit/ed853eb0e366e92889a54a63714f9b9a74e5091f) fix: Allow Bearer token in server mode (#4735)
* [1f421df6b](https://github.com/argoproj/argo-workflows/commit/1f421df6b8eae90882eca974694ecbbf5bf660a6) fix(executor): Deal with the pod watch API call timing out (#4734)
* [724fd80c4](https://github.com/argoproj/argo-workflows/commit/724fd80c4cad6fb30ad665b36652b93e068c9509) feat(controller): Pod deletion grace period. Fixes #4719 (#4725)
* [380268943](https://github.com/argoproj/argo-workflows/commit/380268943efcf509eb28d43f9cbd4ceac195ba61) feat(controller): Add Prometheus metric: `workflow_ttl_queue` (#4722)
* [55019c6ea](https://github.com/argoproj/argo-workflows/commit/55019c6ead5dea100a49cc0c15d99130dff925e3) fix(controller): Fix incorrect main container customization precedence and isResourcesSpecified check (#4681)
* [625189d86](https://github.com/argoproj/argo-workflows/commit/625189d86bc38761b469a18677d83539a487f255) fix(ui): Fix "Using Your Login". Fixes #4707 (#4708)
* [433dc5b99](https://github.com/argoproj/argo-workflows/commit/433dc5b99ab2bbaee8e140a88c4f5860bd8d515a) feat(server): Support email for SSO+RBAC. Closes #4612 (#4644)
* [ae0c0bb84](https://github.com/argoproj/argo-workflows/commit/ae0c0bb84ebcd51b02e3137ea30f9dc215bdf80a) fix(controller): Fixed RBAC on leases (#4715)
* [cd4adda1d](https://github.com/argoproj/argo-workflows/commit/cd4adda1d9737985481dbf73f9ac0bae8a963b2c) fix(controller): Fixed Leader election name (#4709)
* [aec22189f](https://github.com/argoproj/argo-workflows/commit/aec22189f651980e878453009c239348f625412a) fix(test): Fixed Flaky e2e tests TestSynchronizationWfLevelMutex and TestResourceTemplateStopAndTerminate/ResourceTemplateStop (#4688)
* [ab837753b](https://github.com/argoproj/argo-workflows/commit/ab837753bec1f78ad66c0d41b5fbb1739428da88) fix(controller): Fix the RBAC for leader-election (#4706)
* [9669aa522](https://github.com/argoproj/argo-workflows/commit/9669aa522bd18e869c9a5133d8b8acedfc3d22c8) fix(controller): Increate default EventSpamBurst in Eventrecorder (#4698)
* [96a55ce5e](https://github.com/argoproj/argo-workflows/commit/96a55ce5ec91e195c019d648e7f30eafe2a0cf95) feat(controller): HA Leader election support on Workflow-controller (#4622)
* [ad1b6de4d](https://github.com/argoproj/argo-workflows/commit/ad1b6de4d09b6b9284eeed15c5b61217b4da921f) fix: Consider optional artifact arguments (#4672)
* [d9d5f5fb7](https://github.com/argoproj/argo-workflows/commit/d9d5f5fb707d95c1c4d6fe761115ceface26a5cf) feat(controller): Use deterministic name for cron workflow children (#4638)
* [f47fc2227](https://github.com/argoproj/argo-workflows/commit/f47fc2227c5a84a2eace7b977a7761674b81e6f3) fix(controller): Only patch status.active in cron workflows when syncing (#4659)
* [9becf3036](https://github.com/argoproj/argo-workflows/commit/9becf3036f5bfbde8c54a1eebf50c4ce48ca6352) fix(ui): Fixed reconnection hot-loop. Fixes #4580 (#4663)
* [e8cc2fbb4](https://github.com/argoproj/argo-workflows/commit/e8cc2fbb44313b6c9a988072d8947aef2270c038) feat: Support per-output parameter aggregation (#4374)
* [b1e2c2077](https://github.com/argoproj/argo-workflows/commit/b1e2c207722be8ec9f26011957ccdeaa95da2ded) feat(controller): Allow to configure main container resources (#4656)
* [4f9fab981](https://github.com/argoproj/argo-workflows/commit/4f9fab9812ab1bbf5858c51492983774f1f22e93) fix(controller): Cleanup the synchronize  pending queue once Workflow deleted (#4664)
* [705542053](https://github.com/argoproj/argo-workflows/commit/7055420536270fa1cd5560e4bf964bcd65813be9) feat(ui): Make it easy to use SSO login with CLI. Resolves #4630 (#4645)
* [76bcaecde](https://github.com/argoproj/argo-workflows/commit/76bcaecde01dbc539fcd10564925eeff14e30093) feat(ui): add countdown to cronWorkflowList Closes #4636 (#4641)
* [5614700b7](https://github.com/argoproj/argo-workflows/commit/5614700b7bd466aeae8a175ca586a1ff47981430) feat(ui): Add parameter value enum support to the UI. Fixes #4192 (#4365)
* [95ad3349c](https://github.com/argoproj/argo-workflows/commit/95ad3349cf464a421a8beb329d41bf494343cf89) feat: Add shorthanded option -A for --all-namespaces (#4658)
* [3b66f74c9](https://github.com/argoproj/argo-workflows/commit/3b66f74c9b5761f548aa494facecbd06df8fe296) fix(ui): DataLoaderDropdown fix input type from promise to function that (#4655)
* [c4d986ab6](https://github.com/argoproj/argo-workflows/commit/c4d986ab60b8b0a00d9507da34b832845e4630a7) feat(ui): Replace 3 buttons with drop-down (#4648)
* [fafde1d67](https://github.com/argoproj/argo-workflows/commit/fafde1d677361521b4b55a23dd0dbca7f75e3219) fix(controller): Deal with hyphen in creator. Fixes #4058 (#4643)
* [30e172d5e](https://github.com/argoproj/argo-workflows/commit/30e172d5e968e644c80e0739624ec7c8245b4be4) fix(manifests): Drop capabilities, add CNCF badge. Fixes #2614 (#4633)
* [f726b9f87](https://github.com/argoproj/argo-workflows/commit/f726b9f872612e3501a7bcf2a359790c32e4cca0) feat(ui): Add links to init and wait logs (#4642)
* [94be7da35](https://github.com/argoproj/argo-workflows/commit/94be7da35a63aae4b2563f1f3f90647b661f53c7) feat(executor): Auto create s3 bucket if not present. Closes #3586  (#4574)
* [1212df4d1](https://github.com/argoproj/argo-workflows/commit/1212df4d19dd18045fd0aded7fd1dc5726f7d5c5) feat(controller): Support .AnySucceeded / .AllFailed for TaskGroup in depends logic. Closes #3405 (#3964)
* [6175458a6](https://github.com/argoproj/argo-workflows/commit/6175458a6407aae3788b2ffb96b1bd9b14661069) fix: Count Workflows with no phase as Pending for metrics (#4628)
* [a2566b953](https://github.com/argoproj/argo-workflows/commit/a2566b9534c0012038400a5c6ed8884b855d4c64) feat(executor): More informative log when executors do not support output param from base image layer (#4620)
* [e1919c86b](https://github.com/argoproj/argo-workflows/commit/e1919c86b3ecbd1760a404de6d8637ac0ae6ce0b) fix(ui): Fix Snyk issues (#4631)
* [454f3ae35](https://github.com/argoproj/argo-workflows/commit/454f3ae35418c05e114fd6f181a85cf25900a037) fix(ui): Reference secrets in EnvVars. Fixes #3973  (#4419)
* [1f0392075](https://github.com/argoproj/argo-workflows/commit/1f0392075031c83640a7490ab198bc3af9d1b4ba) fix: derive jsonschema and fix up issues, validate examples dir… (#4611)
* [92a283275](https://github.com/argoproj/argo-workflows/commit/92a283275a1bf1ccde7e6a9ae90385459bd1f6fc) fix(argo-server): fix global variable validation error with reversed dag.tasks (#4369)
* [79ca27f35](https://github.com/argoproj/argo-workflows/commit/79ca27f35e8b07c9c6361be342aa3f097d554b53) fix: Fix TestCleanFieldsExclude (#4625)
* [b3336e732](https://github.com/argoproj/argo-workflows/commit/b3336e7321df6dbf7e14bd49ed77fea8cc8f0666) feat(ui): Add columns--narrower-height to AttributeRow (#4371)
* [91bce2574](https://github.com/argoproj/argo-workflows/commit/91bce2574fab15f4fab4bc4df9e50563aa748838) fix(server): Correct webhook event payload marshalling. Fixes #4572 (#4594)
* [39c805fa0](https://github.com/argoproj/argo-workflows/commit/39c805fa0ed167a3cc111556cf1eb864b87627e8) fix: Perform fields filtering server side (#4595)
* [3af8195b2](https://github.com/argoproj/argo-workflows/commit/3af8195b27dfc3e2e426bb649eed923beeaf7e19) fix: Null check pagination variable (#4617)
* [c84d56b64](https://github.com/argoproj/argo-workflows/commit/c84d56b6439cf48814f9ab86e5b899929ab426a8) feat(controller): Enhanced artifact repository ref. See #3184 (#4458)
* [5c538d7a9](https://github.com/argoproj/argo-workflows/commit/5c538d7a918e41029d3911a92c6ac615f04d3b80) fix(executor): Fixed waitMainContainerStart returning prematurely. Closes #4599 (#4601)
* [b92d889a5](https://github.com/argoproj/argo-workflows/commit/b92d889a5a44b01d5d62135848db36be20c20e9d) fix(docs): Bring minio chart instructions up to date (#4586)
* [6c46aab7d](https://github.com/argoproj/argo-workflows/commit/6c46aab7d54678c21df17d6c885473c17f8c66a6) fix(controller): Prevent tasks with names starting with digit to use either 'depends' or 'dependencies' (#4598)
* [4531d7936](https://github.com/argoproj/argo-workflows/commit/4531d7936c25174b3251e926288866c69fc2dba3) refactor: Use polling model for workflow phase metric (#4557)
* [ef779bbf8](https://github.com/argoproj/argo-workflows/commit/ef779bbf8ffc548c4ecc34650f737936ffa5352a) fix(executor): Handle sidecar killing in a process-namespace-shared pod (#4575)
* [9ee4d446c](https://github.com/argoproj/argo-workflows/commit/9ee4d446c1908f59240ca4b814ba565bb1acbc1f) fix(server): serve artifacts directly from disk to support large artifacts (#4589)
* [e3aaf2fb4](https://github.com/argoproj/argo-workflows/commit/e3aaf2fb4f34eeca12778b4caa70c1aa8d80ca14) fix(server): use the correct name when downloading artifacts (#4579)
* [1c62586eb](https://github.com/argoproj/argo-workflows/commit/1c62586eb015e64596bc898166700769364a9d10) feat(controller): Retry transient offload errors. Resolves #4464 (#4482)
* [15fd57942](https://github.com/argoproj/argo-workflows/commit/15fd5794250a2e54e388b394fd288420482df924) feat(controller): Make MAX_OPERATION_TIME configurable. Close #4239 (#4562)

<details><summary><h3>Contributors</h3></summary>

* Alastair Maw
* Alex Capras
* Alex Collins
* Alexey Volkov
* Amim Knabben
* Arthur Outhenin-Chalandre
* BOOK
* Basanth Jenu H B
* Daisuke Taniwaki
* Huan-Cheng Chang
* Isaac Gaskin
* J.P. Zivalich
* Jesse Suen
* Kristoffer Johansson
* Marcin Gucki
* Maximilian Roos
* Michael Albers
* Noah Hanjun Lee
* Paavo Pokkinen
* Paul Brabban
* RossyWhite
* Saravanan Balasubramanian
* Simeon H.K. Fitch
* Simon Behar
* Simon Frey
* Song Juchao
* Stefan Gloutnikov
* Stéphane Este-Gracias
* Takayoshi Nishida
* Tomáš Coufal
* Trevor Wood
* Viktor Farcic
* Wylie Hobbs
* Yuan Tang
* aletepe
* bei-re
* bellevuerails
* cocotyty
* dherman
* ermeaney
* fsiegmund
* hermanhobnob
* joyciep
* kennytrytek
* lonsdale8734
* makocchi
* markterm
* nishant-d
* saranyaeu2987
* tczhao
* zhengchenyu

</details>

## v2.12.13 (2021-08-18)

For v2.12.13 and earlier, see [CHANGELOG-2-x-x.md](CHANGELOG-2-x-x.md)
