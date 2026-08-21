# Changelog

## v4.1.2 (2026-08-21)

Full Changelog: [v4.1.1...v4.1.2](https://github.com/argoproj/argo-workflows/compare/v4.1.1...v4.1.2)

### Selected Changes

* [16a52d67d](https://github.com/argoproj/argo-workflows/commit/16a52d67daf2f4a8a76fa8bec02a76a46aa46257) fix: fail nodes waiting for a sync lock on workflow shutdown (cherry-pick #16777 for 4.1) (#16787)
* [3f106f1ac](https://github.com/argoproj/argo-workflows/commit/3f106f1acc4bcc085083f37a62b7f0be651b2892) fix: requeue workflow on transient sync lock errors instead of failing (cherry-pick #16745 for 4.1) (#16789)
* [8736391d3](https://github.com/argoproj/argo-workflows/commit/8736391d3aaf36b9dde8aebee89f0cb491ef1721) fix(server): use symmetric encryption for SSO session tokens. Fixes #16744 (cherry-pick #16748 for 4.1) (#16788)
* [eb4b3b4aa](https://github.com/argoproj/argo-workflows/commit/eb4b3b4aaac9bee15eafd0fde4e055a86d6245c4) fix: fail, not succeed, workflows terminated while pending on a sync lock (cherry-pick #16776 for 4.1) (#16786)
* [f888616fe](https://github.com/argoproj/argo-workflows/commit/f888616fe4a3c0d5855adce9cb257c3bef7001f5) fix(controller): refuse reapplyUpdate when workflow UID has changed (cherry-pick #16775 for 4.1) (#16778)
* [b632a7371](https://github.com/argoproj/argo-workflows/commit/b632a7371058acc4406ba997f4c8537f554d6799) fix: omit OpenTelemetry process owner detection (cherry-pick #16736 for 4.1) (#16765)
* [b4b29c77f](https://github.com/argoproj/argo-workflows/commit/b4b29c77fed063252a93cafe26b94677d682bf0a) fix: re-enter workingDir after init-less input artifact staging. Fixes #16728 (cherry-pick #16738 for 4.1) (#16760)
* [6e59f8782](https://github.com/argoproj/argo-workflows/commit/6e59f8782f2e2dc14e66d092da45cd729e1ab998) chore(deps): update module github.com/moby/go-archive to v0.3.0 [security] (release-4.1) (#16757)
* [f97bd2db0](https://github.com/argoproj/argo-workflows/commit/f97bd2db07f61b9e72b933982586d87c4248f02d) chore(deps): update module github.com/google/cel-go to v0.30.0 [security] (release-4.1) (#16756)
* [a60772796](https://github.com/argoproj/argo-workflows/commit/a607727961c94415c61b1b3608b3c5bb1f9b5931) chore(deps): update module github.com/valyala/fasthttp to v1.70.0 [security] (release-4.1) (#16758)
* [9902c65c7](https://github.com/argoproj/argo-workflows/commit/9902c65c736be2968781cf88ebd277b5f06d60af) fix: apply MySQL driver-level options from persistence config. Fixes #16707 (cherry-pick #16725 for 4.1) (#16734)

<details><summary><h3>Contributors</h3></summary>

* Alan Clucas
* Claude Fable 5
* Nitin Moningi

</details>

## v4.1.1 (2026-08-14)

Full Changelog: [v4.1.0...v4.1.1](https://github.com/argoproj/argo-workflows/compare/v4.1.0...v4.1.1)

### Selected Changes

* [eaefb4518](https://github.com/argoproj/argo-workflows/commit/eaefb4518483312d69ace640479b565ccd688fdc) fix(logging): remove process-wide signal handler from the init logger. Fixes #15863 (cherry-pick #16693 for 4.1) (#16714)
* [c8bc68f0e](https://github.com/argoproj/argo-workflows/commit/c8bc68f0e90231103851bd958c86ba24b5736543) fix(ci): create ~/.m2 before bind mounting it for Java SDK publish (cherry-pick #16701 for 4.1) (#16704)
* [72a8e6c14](https://github.com/argoproj/argo-workflows/commit/72a8e6c1475c6028a0401be6dacc5bf92fee8878) chore(deps): update k8s.io/kube-openapi digest to d427ff9 (release-4.1) (#16660)
* [47062f954](https://github.com/argoproj/argo-workflows/commit/47062f95426c5f06999a84cf10c8348b9a1bad40) chore(deps): update google.golang.org/genproto/googleapis/api digest to ec0a776 (release-4.1) (#16659)
* [799f79665](https://github.com/argoproj/argo-workflows/commit/799f79665c74becc9d9d1892334b09699d717079) chore(deps): update golang docker tag to v1.26.5 (main) (cherry-pick #16644 for 4.1) (#16651)
* [ed2e836a3](https://github.com/argoproj/argo-workflows/commit/ed2e836a37fe6ffa2cfa08206ed326a7cc1fe0e2) fix(sync): remove the acquiring key from the pending queue, not the front. Fixes #16567 (cherry-pick #16613 for 4.1) (#16641)
* [13eb8a69a](https://github.com/argoproj/argo-workflows/commit/13eb8a69ac2464379c9e223477ddff7c9bfc7b6a) fix(test): de-flake TestParallelismUpdate on coarse-resolution clocks (cherry-pick #16224 for 4.1) (#16646)
* [3a18911b1](https://github.com/argoproj/argo-workflows/commit/3a18911b1c7c6cc31575b6966bd0c55338fa278c) fix: avoid nil pointer (cherry-pick #16608 for 4.1) (#16629)
* [6dfc3a5ef](https://github.com/argoproj/argo-workflows/commit/6dfc3a5efeaf8e6870185b6511ae27e0251e9581) chore(deps): sync sdk go modules with root dependencies (cherry-pick #16612 for 4.1) (#16627)

<details><summary><h3>Contributors</h3></summary>

* Alan Clucas
* Claude Fable 5
* Claude Opus 4.8 (1M context)
* Claude Opus 5 (1M context)
* ham
* Max Xu
* spaced

</details>

## v4.1.0 (2026-08-11)

Full Changelog: [v4.1.0-rc2...v4.1.0](https://github.com/argoproj/argo-workflows/compare/v4.1.0-rc2...v4.1.0)

### Selected Changes

* [e5ed20d5c](https://github.com/argoproj/argo-workflows/commit/e5ed20d5cb54d4708d5aeb29148b3e49922f795c) feat: add argo-workflows-crdinstaller image. Fixes #16621 (cherry-pick #16622 for 4.1) (#16630)

<details><summary><h3>Contributors</h3></summary>

* Alan Clucas
* Claude Fable 5

</details>

## v4.1.0-rc2 (2026-07-31)

Full Changelog: [v4.1.0-rc1...v4.1.0-rc2](https://github.com/argoproj/argo-workflows/compare/v4.1.0-rc1...v4.1.0-rc2)

### Selected Changes

* [c5cb81b7c](https://github.com/argoproj/argo-workflows/commit/c5cb81b7c3b4e2ffd1b18a9be66e1908d4e623ca) feat: add first-class DRA resource claims. Fixes #16576 (#16587)
* [b365f9089](https://github.com/argoproj/argo-workflows/commit/b365f908959d6791b81203d55e2b3d4cf0f6eb34) refactor(executor): parse env at the argoexec composition root, enforce with forbidigo (#16554)
* [9642f1e15](https://github.com/argoproj/argo-workflows/commit/9642f1e15b822e1e94511d2e4909b839d4b1dcdd) fix(controller): archive each workflow once, and retry failed archives. Fixes #16575 (#16577)
* [f86b3523b](https://github.com/argoproj/argo-workflows/commit/f86b3523be851bce8bddac7aefd60921a2865415) fix(ui): show CronWorkflow errors in list. Fixes #10264 (#16578)
* [5992135e9](https://github.com/argoproj/argo-workflows/commit/5992135e9f69b950835752e7a9f2639d0de4aa65) chore(deps): update module github.com/go-logr/logr to v1.4.4 (#16583)
* [4f741c926](https://github.com/argoproj/argo-workflows/commit/4f741c926afc7aefbba57045a47fe176125f4ea0) chore(deps): update module github.com/prometheus/common to v0.70.1 (#16585)
* [ea918bd5b](https://github.com/argoproj/argo-workflows/commit/ea918bd5b8f20540ec6c59a1a62fadfbe6654283) chore(deps): update module github.com/go-git/go-git/v5 to v5.19.2 (#16582)
* [0ebb540ae](https://github.com/argoproj/argo-workflows/commit/0ebb540ae69c263726f0a6707e28e2dd51e5ca55) chore(deps): update module github.com/klauspost/compress to v1.19.1 (#16584)
* [1df5e9c54](https://github.com/argoproj/argo-workflows/commit/1df5e9c54cd007fe078ed83f1b33944e2a976dac) chore(deps): update actions/stale action to v11 (main) (#16586)
* [a862e823b](https://github.com/argoproj/argo-workflows/commit/a862e823b0ddc8bff4531acdc533a5b63ea38d45) fix(controller): do not postpone already-Running workflows. Fixes #14123 (#16569)
* [8498c4efb](https://github.com/argoproj/argo-workflows/commit/8498c4efbe604e81c68dcd17ecbb18c0915b072d) feat(controller): strip managedFields from informer caches. Fixes #16564 (#16563)
* [bf648dde4](https://github.com/argoproj/argo-workflows/commit/bf648dde474b3247beb7ca0f9cf91d2bd86aeccf) fix(ui): keep markdown tooltips within the viewport (#16549)
* [b39b572bc](https://github.com/argoproj/argo-workflows/commit/b39b572bc71165ccad229fa627146758dfe35cf2) perf(controller): avoid full Workflow DeepCopy for postponed workflows. Fixes #16559 (#16560)
* [402bbe098](https://github.com/argoproj/argo-workflows/commit/402bbe098c3619ec01230c03ac47d824b90b8c2e) chore(deps): update module github.com/google/cel-go to v0.29.0 [security] (main) (#16547)
* [3c6644803](https://github.com/argoproj/argo-workflows/commit/3c6644803c6ec6f6d0b703bf8165fd306017c087) chore(deps): update dependency pymdown-extensions to v11 [security] (main) (#16532)

<details><summary><h3>Contributors</h3></summary>

* Alan Clucas
* Arthur Kepler
* Claude Fable 5
* hyzaw
* Joibel
* Juhef
* panicboat
* Zuhef Ahmed
* 秀吉

</details>

## v4.1.0-rc1 (2026-07-23)

Full Changelog: [v4.0.10...v4.1.0-rc1](https://github.com/argoproj/argo-workflows/compare/v4.0.10...v4.1.0-rc1)

### Selected Changes

* [1d8853c3b](https://github.com/argoproj/argo-workflows/commit/1d8853c3b04833683918aa2e5c9f19eb7ec86ad8) feat(cli): Extend gRPC and HTTP clients to support client certificate. Fixes #13437 (#13447)
* [35fe25f51](https://github.com/argoproj/argo-workflows/commit/35fe25f51bb178d1d649b61713d679b655749927) chore(deps): update module google.golang.org/grpc to v1.82.1 [security] (#16526)
* [88cf2ed55](https://github.com/argoproj/argo-workflows/commit/88cf2ed55cc5258ea6e0a55a15c3958e31833308) chore(deps): update dependency @types/superagent to v8.1.11 (main) (#15350)
* [2be90ed6a](https://github.com/argoproj/argo-workflows/commit/2be90ed6a28a92288625f61bb60d4c68aad48e03) chore(deps): update actions/checkout action to v7.0.1 (#16510)
* [25498c828](https://github.com/argoproj/argo-workflows/commit/25498c82812c3c02689921c281d9daeda8d3c300) chore(deps): update actions/setup-go action to v7 (main) (#16512)
* [cb05e7d94](https://github.com/argoproj/argo-workflows/commit/cb05e7d94ae641c310d46fd867f9f8d43214d3ac) chore(deps): update actions/setup-java action to v5.6.0 (main) (#16511)
* [8296cf2a2](https://github.com/argoproj/argo-workflows/commit/8296cf2a2f9d8ebb3b55df0a9556f957c99d5c68) chore(deps): update actions/setup-node action to v7 (main) (#16513)
* [f9eeb64d0](https://github.com/argoproj/argo-workflows/commit/f9eeb64d0a26af6a47eecce570000408cb5ab631) fix: replace configmap watchers with informers (#16408)
* [bda66ce79](https://github.com/argoproj/argo-workflows/commit/bda66ce79d1a5fa6bdddc4a3ea9f04af051393e6) chore(deps): update actions/setup-python action to v7 (main) (#16514)
* [010f2ebf4](https://github.com/argoproj/argo-workflows/commit/010f2ebf4ded28bf48169683179758aba11a538e) feat: add support for pod level resource requests/limits (#16399)
* [f7f4676b5](https://github.com/argoproj/argo-workflows/commit/f7f4676b52066ee32167f594dcf48f1ed84b2238) fix: normalize GCS artifact keys to forward slashes on Windows. Fixes #16470 (#16476)
* [f4398395d](https://github.com/argoproj/argo-workflows/commit/f4398395dc690492fedf62c56f8873dbbe80024b) fix(controller): mark reapply-failed on persist errors to keep throttler slot (#16482)
* [682d741c4](https://github.com/argoproj/argo-workflows/commit/682d741c41a45b361fd77a4fcf0bef6d289fe535) feat(controller): hot-reload namespaceParallelism from config (#16486)
* [e31b6c1b7](https://github.com/argoproj/argo-workflows/commit/e31b6c1b7dff1e9a92549aabc60c42ec96971412) fix(controller): treat client-go rate limiter wait deadline as transient (#16485)
* [0fac33e12](https://github.com/argoproj/argo-workflows/commit/0fac33e12ee4d8ac01a04031f005c9ff662617c8) fix(controller): allow onExit DAG handler to complete under Stop shutdown (#16488)
* [c92b436ac](https://github.com/argoproj/argo-workflows/commit/c92b436ac17d9a9006c832f7833f986b8a4d70ef) fix(errors): treat client-go response body read failures as transient (#16484)
* [d0ce6aeaf](https://github.com/argoproj/argo-workflows/commit/d0ce6aeaf3b1fbada1d383454e6a912157bd4dda) fix(errors): treat gRPC client request timeout as transient (#16487)
* [4d4229011](https://github.com/argoproj/argo-workflows/commit/4d4229011ec2701b791e85f2e6c103d9164d86db) fix(ui): handle exceptions when retrieving user info (#16491)
* [170fdf7f1](https://github.com/argoproj/argo-workflows/commit/170fdf7f1941fa5dc2a3c18a2b5afce3605ab59d) feat: add metrics for mutexes and semaphores (#14906)
* [e94bcb1b1](https://github.com/argoproj/argo-workflows/commit/e94bcb1b1e82a8b08118ec15ed13f08f8cbc03b2) feat: Upload input artifacts when submitting workflows from the UI (#15237)
* [b4d04a95c](https://github.com/argoproj/argo-workflows/commit/b4d04a95ccc125aee31d4dd68d6e1703e7ad6b78) chore(deps): update module github.com/andybalholm/brotli to v1.2.2 (main) (#16424)
* [4eac377ce](https://github.com/argoproj/argo-workflows/commit/4eac377ce1bd7fa209cc24dc671ec20a0fc204ff) chore(deps): update module google.golang.org/grpc to v1.82.1 (#16472)
* [0ae90ae65](https://github.com/argoproj/argo-workflows/commit/0ae90ae6553d004beff0040b593cfc4d73eb45d5) chore(deps): update actions/setup-go action to v7 (main) (#16473)
* [116ae4b37](https://github.com/argoproj/argo-workflows/commit/116ae4b37d75fd30238a619c1c174567ccfc77b1) fix: don't leak semaphore slots when limit fetch fails during release (#16405)
* [8544de5c1](https://github.com/argoproj/argo-workflows/commit/8544de5c1c782d4dd20b6c918124ff2268c0bdef) refactor(controller): extract pure podBuilder from createWorkflowPod (#16377)
* [3870787b1](https://github.com/argoproj/argo-workflows/commit/3870787b168a79d9294d87b6c8cb6e358aa09056) chore(deps): update module go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc to v0.69.0 (main) (#15701)
* [353a173de](https://github.com/argoproj/argo-workflows/commit/353a173de61a24983769d78dc741e84528905abb) fix: make workflow retry reset deterministic. Fixes #16450 (#16451)
* [47edab814](https://github.com/argoproj/argo-workflows/commit/47edab814b6c3e45dd616db863b61139b023e363) fix: complete orphaned TaskGroup nodes stuck Running. Fixes #16450 (#16454)
* [955fa9168](https://github.com/argoproj/argo-workflows/commit/955fa91682e36cf4f297d65082016a00b343c41a) chore(deps): update dependency qs to v6.15.3 (#16456)
* [5a980f606](https://github.com/argoproj/argo-workflows/commit/5a980f606c353851243ace282d0a2154aed6bac4) chore(deps): update dependency rxjs to v7.8.2 (#16457)
* [6eca095fb](https://github.com/argoproj/argo-workflows/commit/6eca095fb61bf10827f9448243e5a115f46405e3) chore(deps): update dependency linkify-it to v6 (main) (#16444)
* [801c0d351](https://github.com/argoproj/argo-workflows/commit/801c0d3512f9e6670ce85719266115245a2f76ca) fix: log missing optional output parameter at warn level, not error. Fixes #16395 (#16402)
* [2ffd608b4](https://github.com/argoproj/argo-workflows/commit/2ffd608b4948f2dc7dbf5ae93de4c5c8a877bbe1) chore(deps): update module cloud.google.com/go/storage to v1.63.1 (#16442)
* [fcbd3e4eb](https://github.com/argoproj/argo-workflows/commit/fcbd3e4eb306e0b45fadf81756a65ebc0f5da208) chore(deps): update gcr.io/distroless/static-debian13:latest docker digest to 9197324 (#16439)
* [55ee72c2d](https://github.com/argoproj/argo-workflows/commit/55ee72c2d3709570957c93df077d02730fe353d7) chore(deps): update alpine docker tag to v3.24 (main) (#16423)
* [3bbb0fc78](https://github.com/argoproj/argo-workflows/commit/3bbb0fc78f1bba94d5b7a202a195be1e20fe9025) chore(deps): update dependency @types/dagre to v0.7.54 (#16447)
* [289f896b4](https://github.com/argoproj/argo-workflows/commit/289f896b4cbe2c5d519574013545632f2a54b9e1) chore(deps): update module github.com/prometheus/common to v0.70.0 (main) (#16427)
* [4dcc2f102](https://github.com/argoproj/argo-workflows/commit/4dcc2f102b419bc3ef8b03f23ad7c4abbd2d15e2) chore(deps): update google.golang.org/genproto/googleapis/api digest to f5fc221 (main) (#16440)
* [08581d7bb](https://github.com/argoproj/argo-workflows/commit/08581d7bb68f3428b37c90e666bab0d835251a20) chore(deps): update dependency linkify-it to v5.0.2 (#16443)
* [71c767059](https://github.com/argoproj/argo-workflows/commit/71c767059415d35345f087a6e4f3963d0b234237) chore(deps): update actions/setup-java action to v5.5.0 (main) (#16420)
* [3171e882f](https://github.com/argoproj/argo-workflows/commit/3171e882f8aef5193e41ee793b68442842945852) chore(deps): update module github.com/coreos/go-oidc/v3 to v3.20.0 (main) (#16425)
* [5d6130607](https://github.com/argoproj/argo-workflows/commit/5d6130607b16e28066194af8b4642003ef56fe2e) chore(deps): update marocchino/sticky-pull-request-comment action to v3.0.5 (#16441)
* [69bf801d5](https://github.com/argoproj/argo-workflows/commit/69bf801d5783eea2cf65fcc24889e448ca7a5fe8) feat: Add workflow-level executor plugin settings definition (#15440)
* [80394f853](https://github.com/argoproj/argo-workflows/commit/80394f85386f59afeebe840635abe19700f45073) chore(deps): update module github.com/aws/aws-sdk-go-v2/config to v1.32.30 (#16417)
* [af37b91f7](https://github.com/argoproj/argo-workflows/commit/af37b91f757eab46fa0fe775e3e13c589b85d17f) chore(deps): update module github.com/argoproj/argo-events to v1.9.11 (#16415)
* [225ca4f41](https://github.com/argoproj/argo-workflows/commit/225ca4f416858ab900d30aaa5d44fff03fd4a2a9) chore(deps): update module github.com/argoproj/argo-workflows/v4 to v4.0.7 (#16416)
* [31c910ac6](https://github.com/argoproj/argo-workflows/commit/31c910ac62098c26540fbcc59afc1a7e21120a1c) chore(deps): update module github.com/xsam/otelsql to v0.43.0 (main) (#16428)
* [bdd1689df](https://github.com/argoproj/argo-workflows/commit/bdd1689df7afc7d7a22e62e66a643de5ef7e8923) chore(deps): update module golang.org/x/net to v0.57.0 (main) (#16430)
* [b88b28fa2](https://github.com/argoproj/argo-workflows/commit/b88b28fa28f85f0720293ac68d6c7823423b624a) chore(deps): update actions/setup-node action to v7 (main) (#16435)
* [6ec57c8e1](https://github.com/argoproj/argo-workflows/commit/6ec57c8e12e3876e30f87234fcc8f8af3e382d15) chore(deps): update module github.com/go-openapi/jsonreference to v1 (main) (#16436)
* [741cbf12b](https://github.com/argoproj/argo-workflows/commit/741cbf12bc3aae9baec1f1160ca83478e94b9441) chore(deps): update module github.com/klauspost/compress to v1.19.0 (main) (#16426)
* [e7698b2ef](https://github.com/argoproj/argo-workflows/commit/e7698b2ef130250b07041eb1ed2bfbcdd486d6fc) chore(deps): update module golang.org/x/crypto to v0.54.0 (main) (#16429)
* [d8b1d3a3a](https://github.com/argoproj/argo-workflows/commit/d8b1d3a3a69aaa69c827ba527c7e67a15ca0c14f) chore(deps): update actions/stale action to v10.4.0 (main) (#16422)
* [f8fa2cdf5](https://github.com/argoproj/argo-workflows/commit/f8fa2cdf51a707515508dde4bd116d213cbcda70) chore(deps): update softprops/action-gh-release digest to 3d0d988 (main) (#16414)
* [87e418a22](https://github.com/argoproj/argo-workflows/commit/87e418a22350375044ec978163f2dbe8ccec4300) feat(ui): add markdown rendering support to Tooltip (#15904)
* [8414072c4](https://github.com/argoproj/argo-workflows/commit/8414072c4b09ea514c059e4a4b0422116a298e2f) fix(ui): resolve login logo paths with non-root baseHref (#16385)
* [a99e2f4cf](https://github.com/argoproj/argo-workflows/commit/a99e2f4cf70935b98dd5c81a77c16c64252bbad1) feat: add SaveStream to artifact drivers with plugin gRPC streaming (#16407)
* [a8a875ceb](https://github.com/argoproj/argo-workflows/commit/a8a875ceb63bb5d08f4c82a0dde8bc4bf49deb92) feat: add pendingTimeout for non-deadline timeout  Fixes #10341 (#12762)
* [4d81cb3b4](https://github.com/argoproj/argo-workflows/commit/4d81cb3b4379c0d85205b6ba0c73760c2ac23475) feat: configurable compression algorithms for node compression. Fixes #16262 (#16261)
* [e7f5710e1](https://github.com/argoproj/argo-workflows/commit/e7f5710e1d8ba8b0035960bc64776bb7d41d3a7d) fix: prevent NaN display in cron workflow number input fields (#16275)
* [a3b5778ca](https://github.com/argoproj/argo-workflows/commit/a3b5778ca3faa9fb6e5b93eac9de6ef76cd9b61f) chore(deps): update docker/login-action action to v4.4.0 (main) (#16389)
* [3c0099353](https://github.com/argoproj/argo-workflows/commit/3c009935383ba76886e74c8f527f87049d51cd7f) fix: add configurable database connection timeout (#16291)
* [b29f5a2b0](https://github.com/argoproj/argo-workflows/commit/b29f5a2b0de9f1d7ac903288a3e8adc18333301b) fix(ui): Fixed Azure Queue Storage icon in event flow diagram Fixes #16384 (#16390)
* [655c4765a](https://github.com/argoproj/argo-workflows/commit/655c4765a1ea3f6bc66ab1c418deddace576e0f6) chore(deps): update module github.com/aws/aws-sdk-go-v2/service/sts to v1.44.0 (main) (#16369)
* [e00dcad10](https://github.com/argoproj/argo-workflows/commit/e00dcad1019d2362cc33d9d50cd27fbb3b659a5a) chore(deps): update google.golang.org/genproto/googleapis/api digest to f0a9213 (main) (#16387)
* [20f85dfcc](https://github.com/argoproj/argo-workflows/commit/20f85dfcca629314f52191981a9383b320602334) chore(deps): update k8s.io/utils digest to cf1189d (main) (#16388)
* [37d0dd7fa](https://github.com/argoproj/argo-workflows/commit/37d0dd7fae84ae9a895e63af9be55d51062e0e98) fix: log "Max parallelism reached" at info level, not error. Fixes #16378 (#16379)
* [0233ce5cc](https://github.com/argoproj/argo-workflows/commit/0233ce5cc5949ea2905faec7640918a35dfd1bf6) feat: add opt-in init-less pod layout (beta) (#16161)
* [7fefac747](https://github.com/argoproj/argo-workflows/commit/7fefac747c9f87e48757bffe8d6749533a027944) chore(deps): update k8s.io/utils digest to be93311 (main) (#16373)
* [4284f07b2](https://github.com/argoproj/argo-workflows/commit/4284f07b226a5491014c5ef4cbb80000ed3c18c8) fix(ci): unbreak cherry-pick automation after actions/checkout v7 bump (#16374)
* [a65c331c7](https://github.com/argoproj/argo-workflows/commit/a65c331c7a9f0c54957b6d7f741fb61c9c83a87e) feat: configurable allowlist (#16344)
* [a7a7a8dfb](https://github.com/argoproj/argo-workflows/commit/a7a7a8dfb53a35314b81616ec35b5e3f7270b250) fix: reject stale copies of completed workflows using resourceVersion comparison (#16357)
* [0d35b564c](https://github.com/argoproj/argo-workflows/commit/0d35b564ca7a4dc04493a0c69a2382d686979045) chore(deps): update module github.com/aws/aws-sdk-go-v2/feature/rds/auth to v1.6.30 (main) (#16366)
* [6b92ad082](https://github.com/argoproj/argo-workflows/commit/6b92ad082a24f753002093b7156fb7066750381d) chore(deps): update module github.com/google/go-containerregistry to v0.21.7 (main) (#16367)
* [35d2e8079](https://github.com/argoproj/argo-workflows/commit/35d2e8079398f0c938e6912c0252e79063dc2950) chore(deps): update module github.com/aws/aws-sdk-go-v2/config to v1.32.27 (main) (#16364)
* [d6f8fa1c8](https://github.com/argoproj/argo-workflows/commit/d6f8fa1c82e1e164c13c003a5a4086299b39aa8f) chore(deps): update module github.com/minio/minio-go/v7 to v7.2.1 (main) (#16368)
* [4930578f0](https://github.com/argoproj/argo-workflows/commit/4930578f0496a58a0102bf8e9b3901ec62e97aff) chore(deps): update module github.com/alibabacloud-go/tea to v1.5.2 (main) (#16362)
* [43a020ef6](https://github.com/argoproj/argo-workflows/commit/43a020ef62429fd174644847058eceb60accc18c) chore(deps): update module github.com/argoproj/argo-workflows/v4 to v4.0.6 (main) (#16363)
* [978d48660](https://github.com/argoproj/argo-workflows/commit/978d486606ae57dd0df470b1505b2e220d86da3c) chore(deps): update docker/setup-buildx-action action to v4.2.0 (main) (#16360)
* [75070c15b](https://github.com/argoproj/argo-workflows/commit/75070c15b78ac149c209046ea5aa17964db0ef99) chore(deps): update dependency markdown-link-check to v3.14.2 (main) (#16358)
* [9881855ad](https://github.com/argoproj/argo-workflows/commit/9881855ad085544588920c20b26b8f2654db979e) chore(deps): update softprops/action-gh-release digest to 718ea10 (main) (#16355)
* [d2d618cd2](https://github.com/argoproj/argo-workflows/commit/d2d618cd2e0bd1e7e97130f1ce07152ed7f11511) chore(deps): update stefanzweifel/git-auto-commit-action action to v7.2.0 (main) (#16361)
* [bbc1c2a81](https://github.com/argoproj/argo-workflows/commit/bbc1c2a81caf22c0e9df91e786007f4a84791a1b) chore(deps): update dependency markdownlint-cli to v0.49.0 (main) (#16359)
* [1ac77decc](https://github.com/argoproj/argo-workflows/commit/1ac77decc2832a048a72abcaf8cc9f3208c8b02f) chore(deps): update dependency buildx to v0.35.0 (main) (#16356)
* [d1f52eaab](https://github.com/argoproj/argo-workflows/commit/d1f52eaab3a9860f818998d4fcbe3f07548e5add) chore(deps): update module github.com/azure/azure-sdk-for-go/sdk/azidentity to v1.14.0 (main) (#16349)
* [ca8e0edb3](https://github.com/argoproj/argo-workflows/commit/ca8e0edb3aa78dcac512c9ef9a97eed3e9548ac2) chore(deps): update module github.com/testcontainers/testcontainers-go/modules/mysql to v0.43.0 (main) (#16350)
* [dffd802bd](https://github.com/argoproj/argo-workflows/commit/dffd802bd77ff9fc06913d288931bce599714079) chore(deps): update module cloud.google.com/go/storage to v1.63.0 (main) (#16327)
* [4b192115e](https://github.com/argoproj/argo-workflows/commit/4b192115e375b676072ebfc20040e8d0268e4d3b) chore(deps): update module github.com/moby/moby/api to v1.55.0 (main) (#16330)
* [86e85e2ef](https://github.com/argoproj/argo-workflows/commit/86e85e2ef76b8add9b459f9426ac08db1fc06c13) chore(deps): update module github.com/testcontainers/testcontainers-go/modules/postgres to v0.43.0 (main) (#16351)
* [ddbf3e28c](https://github.com/argoproj/argo-workflows/commit/ddbf3e28c27b8481eb20cc38f75d5a1369515277) chore(deps): update docker/build-push-action action to v7.3.0 (main) (#16346)
* [1e3039a88](https://github.com/argoproj/argo-workflows/commit/1e3039a88e3f884e03c4c6aa24c641ad06bafe2c) fix: automatically document the availability & lifecycle of variables (#16043)
* [381eb01eb](https://github.com/argoproj/argo-workflows/commit/381eb01eb1a4f07deb6513a033365010509334b5) chore(deps): update module github.com/testcontainers/testcontainers-go to v0.43.0 (main) (#16335)
* [6c62ab5fb](https://github.com/argoproj/argo-workflows/commit/6c62ab5fbd647a881e28b9b58a10f6f32c5f8e28) chore(deps): update module github.com/coreos/go-oidc/v3 to v3.19.0 (main) (#16334)
* [ef174e3b7](https://github.com/argoproj/argo-workflows/commit/ef174e3b73e5236e2a581e2e2de99cba07c85a5b) chore(deps): update docker/login-action action to v4.3.0 (main) (#16347)
* [383eafb39](https://github.com/argoproj/argo-workflows/commit/383eafb391dd0389c59cc37975086b61d9422d9f) chore(deps): update docker/setup-qemu-action action to v4.2.0 (main) (#16348)
* [2b8f57ade](https://github.com/argoproj/argo-workflows/commit/2b8f57adefb6e72cc777d69462292848e8020cf9) chore(deps): update nixbuild/nix-quick-install-action action to v35 (main) (#16352)
* [e4d03cfe7](https://github.com/argoproj/argo-workflows/commit/e4d03cfe789628710cce0dc2ac6e0bb551653463) chore(deps): update module google.golang.org/grpc to v1.82.0 (main) (#16340)
* [a6a2bb58e](https://github.com/argoproj/argo-workflows/commit/a6a2bb58e25fc043176c021141b08cf4014801b2) chore(deps): update module github.com/azure/azure-sdk-for-go/sdk/azcore to v1.22.0 (main) (#16328)
* [d2594d761](https://github.com/argoproj/argo-workflows/commit/d2594d761e0301196ae3309517fd0b991fd4b881) chore(deps): update module github.com/azure/azure-sdk-for-go/sdk/storage/azblob to v1.8.0 (main) (#16329)
* [438683ef3](https://github.com/argoproj/argo-workflows/commit/438683ef3e2cb5624c1bc08d8d1fe8f1d1cbd2b0) chore(deps): update module github.com/prometheus/common to v0.69.0 (main) (#16331)
* [f3f40205f](https://github.com/argoproj/argo-workflows/commit/f3f40205f8f66d8fe63d144fb1b2fc182f3f6f63) chore(deps): update module golang.org/x/crypto to v0.53.0 (main) (#16332)
* [8202b20bf](https://github.com/argoproj/argo-workflows/commit/8202b20bff6de03b2ccbd8e1aee68cd720ac7ed7) chore(deps): update module golang.org/x/sync to v0.21.0 (main) (#16333)
* [e70945c53](https://github.com/argoproj/argo-workflows/commit/e70945c53da48d297c2976bcb4ff17e771072797) chore(deps): update module google.golang.org/api to v0.287.0 (main) (#16339)
* [d6fed6b40](https://github.com/argoproj/argo-workflows/commit/d6fed6b40097f50ccb57e6b3ae41e2b36d732400) chore(deps): update actions/checkout action to v7 (main) (#16342)
* [b74927601](https://github.com/argoproj/argo-workflows/commit/b7492760116d8138a82c3b25bb41cfac2e39d1b3) chore(deps): update module golang.org/x/sys to v0.46.0 (main) (#16337)
* [ab1629ff5](https://github.com/argoproj/argo-workflows/commit/ab1629ff5dbd0cb74ac97044b39cbde9948a79ed) chore(deps): update zgosalvez/github-actions-ensure-sha-pinned-actions action to v5.0.5 (#16321)
* [dbcd931a6](https://github.com/argoproj/argo-workflows/commit/dbcd931a699b29da73a155d4ab533eb4413ddea3) fix(ci): build gomod2nix from repo flake so it uses the pinned Go
* [a3fddbdc3](https://github.com/argoproj/argo-workflows/commit/a3fddbdc35cbcce9150cbcd37ccbad22d0686016) chore(deps): update codecov/codecov-action action to v7 (main) (#16343)
* [cbab925a4](https://github.com/argoproj/argo-workflows/commit/cbab925a4175a7682c2268abd1fb39117d627a13) chore(deps): update actions/setup-go action to v6.5.0 (main) (#16323)
* [98a314118](https://github.com/argoproj/argo-workflows/commit/98a314118a1734b99e25ca6e444b14b02e3c2d21) chore(deps): update actions/setup-python action to v6.3.0 (main) (#16325)
* [21bcde5b9](https://github.com/argoproj/argo-workflows/commit/21bcde5b9ca311270836cf12747b81fd99542a5d) chore(deps): update docker/dockerfile docker tag to v1.25 (main) (#16326)
* [5ec2378ab](https://github.com/argoproj/argo-workflows/commit/5ec2378abe016536a75772c17740263b1e99ccbc) chore(deps): update actions/cache action to v6 (main) (#16341)
* [bd6e496b5](https://github.com/argoproj/argo-workflows/commit/bd6e496b5dacf065cb3c1455670d46d064d032e4) chore(deps): update actions/setup-java action to v5.4.0 (main) (#16324)
* [b471bb413](https://github.com/argoproj/argo-workflows/commit/b471bb413ed37af5be506db01bdc0378157f4fd1) fix: resolve race condition in custom metric initialization (#16238)
* [913be5f2c](https://github.com/argoproj/argo-workflows/commit/913be5f2c6b2c5f1cf360dc2323a1b4a7da51788) fix(ci): trigger Codegen when its source inputs change (#16308)
* [acfb73a39](https://github.com/argoproj/argo-workflows/commit/acfb73a390337c370f046ba773c0669f28316ef6) fix(auth): mask sensitive token in sso callback logs (#16268)
* [589e31aa6](https://github.com/argoproj/argo-workflows/commit/589e31aa6024d278327db2dba2db54386085d1b8) fix: return 404 instead of panic when archived workflow is not yet persisted (#16302)
* [cec639645](https://github.com/argoproj/argo-workflows/commit/cec639645bedde59909a785fceb6bb2fa2dc4fa3) feat(ci): add PR readiness helper bot (#16231)
* [9577e45f2](https://github.com/argoproj/argo-workflows/commit/9577e45f23f48b5a6d573bd3b55e949803f2560c) feat: add addressingStyle field to S3Bucket for virtual-hosted-style support (Fixes #10851) (#15734)
* [efbf0a07b](https://github.com/argoproj/argo-workflows/commit/efbf0a07b328a733b1f3a49028497f2f81fe980e) chore(deps): update dependency linkify-it to v5.0.1 [security] (#16299)
* [326c5d77b](https://github.com/argoproj/argo-workflows/commit/326c5d77b773d918619c350122df30d7cfca8ac1) fix(sso): fix "asymmetric encryption algorithms not supported for JWT". Fixes #16232 (#16292)
* [2e0513f61](https://github.com/argoproj/argo-workflows/commit/2e0513f61da1a3f449a1d38a5455d07e0c6553bb) fix: honor ?? and ?. guards in strict missing-variable check (#16274)
* [4cb7eb4e8](https://github.com/argoproj/argo-workflows/commit/4cb7eb4e89299595ba8fddd3b7f9d95872c3b39d) fix: mark retry wrapper node as Failed when workflow is terminated during active retry (#15740)
* [5bfb9c5f2](https://github.com/argoproj/argo-workflows/commit/5bfb9c5f2412fe9989d4f5c46deaf4ed53f8fafd) fix!: drop values when skipped arguments are being substituted (#16223)
* [179747d2d](https://github.com/argoproj/argo-workflows/commit/179747d2d475b5ef555038c4a9ec208d1d68553f) chore(deps): update dependency @babel/core to v7.29.6 [security] (main) (#16278)
* [8bf2a9be7](https://github.com/argoproj/argo-workflows/commit/8bf2a9be7911c05c5224d6bc6a47391b6e5e16b6) fix: addressed a typo in devenv config (#16260)
* [a904a124e](https://github.com/argoproj/argo-workflows/commit/a904a124e3900ff0a630b520664368d97ed57d08) Merge commit from fork
* [deef55979](https://github.com/argoproj/argo-workflows/commit/deef55979e913a4c5a6b07060533719bd9967916) fix: automate nix dependecy updates. Fixes #11691 (#15557)
* [a681fe1cb](https://github.com/argoproj/argo-workflows/commit/a681fe1cb0881d12cabf2d6958622f85826f0070) fix: WorkflowTaskSets size bloat for large workflows (#16075)
* [d81ac3f78](https://github.com/argoproj/argo-workflows/commit/d81ac3f781050c8e5fe0ee6f98033d3f789c08c5) fix: do not re-run `onExitNode`. Fixes #14392 (#16088)
* [631fd14d6](https://github.com/argoproj/argo-workflows/commit/631fd14d629ba0e18bda32fbd95e0f1de6bb8811) fix: allow cron aliases in schedule validation (#16100)
* [97382fcc6](https://github.com/argoproj/argo-workflows/commit/97382fcc65c2709e0f0907b763a6f3768aa7193f) fix: address semaphore/mutex unsoundness for Initalize (#16160)
* [fd8faa158](https://github.com/argoproj/argo-workflows/commit/fd8faa158f7a977fca87e99017d3bf3c413e5b0e) fix(ui): fix error on "Graph" tab when viewing CronWorkflow and add tests. Fixes #15275 (#16071)
* [eecd3289b](https://github.com/argoproj/argo-workflows/commit/eecd3289bcb327a45199b5d775fbaddb2f0a3467) fix(ui): fix mixed bold/notbold markdown in title annotations (#16064)
* [a3a6c4638](https://github.com/argoproj/argo-workflows/commit/a3a6c463800f8da01e877062d0f48819a74c5316) fix(ui): show full tag value on hover in TagsInput. Fixes #16096 (#16095)
* [37699caf8](https://github.com/argoproj/argo-workflows/commit/37699caf80abc0da28e20fcb6e02029ea7ffc114) chore(deps): update k8s.io/kube-openapi digest to 865597e (main) (#16194)
* [c786f25b9](https://github.com/argoproj/argo-workflows/commit/c786f25b9cf7bf096ae3f9162df8e32cc63b1652) chore(deps): update module github.com/go-jose/go-jose/v3 to v4 (main) (#16213)
* [85e490eb0](https://github.com/argoproj/argo-workflows/commit/85e490eb00be1f757626498cb4cd237fe2b4bdb3) fix(crds): escape template variables in CRD descriptions to prevent Helm rendering errors (#16036)
* [a42ddaeaf](https://github.com/argoproj/argo-workflows/commit/a42ddaeaf899bb4118931e48290dfe6b6115cfb6) feat: #12442 improve s3 upload speed (#15260)
* [e0fbea440](https://github.com/argoproj/argo-workflows/commit/e0fbea44012ed99a20eaf31e3f2e5c0a05de1315) chore(deps): update module golang.org/x/net to v0.55.0 [security] (main) (#16132)
* [5814af2f9](https://github.com/argoproj/argo-workflows/commit/5814af2f9275403443668059bf34785d6a1ab574) chore(deps): update module cloud.google.com/go/storage to v1.62.3 (#16225)
* [29e1ac9f2](https://github.com/argoproj/argo-workflows/commit/29e1ac9f21ca95cee6514bcf23b2ec8cd633f7a7) chore(deps): update module github.com/azure/azure-sdk-for-go/sdk/storage/azblob to v1.7.0 (main) (#16183)
* [c4137226d](https://github.com/argoproj/argo-workflows/commit/c4137226d62a394315bc9e149ef470687389ee54) chore(deps): upgrade Node.js to v24 in all GitHub Actions workflows (#16218)
* [c1b47e1a7](https://github.com/argoproj/argo-workflows/commit/c1b47e1a774c313027a29376ef5008f01f2675d0) chore(deps): update module go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc to v1.44.0 (main) (#16221)
* [c045a4d39](https://github.com/argoproj/argo-workflows/commit/c045a4d390fb77aa36a048ea881334ccaa1ea5e2) chore(deps): update module go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc to v1.44.0 (main) (#16207)
* [f2cbb972f](https://github.com/argoproj/argo-workflows/commit/f2cbb972ff2612462796e3eedb0a66f8404c5db2) chore(deps): update module go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp to v1.44.0 (main) (#16206)
* [5ddf2e645](https://github.com/argoproj/argo-workflows/commit/5ddf2e64530ee2cd94fc4ad171543288f133711e) fix: retry for database transaction errors. Fixes #16101 (#16102)
* [41a199192](https://github.com/argoproj/argo-workflows/commit/41a1991925f36cc24d6fddea55825da127528fbc) chore(deps): update module github.com/aws/aws-sdk-go-v2/config to v1.32.20 (#16164)
* [96d0572f3](https://github.com/argoproj/argo-workflows/commit/96d0572f36bf47a0fd8e0acfe43fe93a2b799f2b) chore(deps): update module github.com/minio/minio-go/v7 to v7.2.0 (main) (#16186)
* [0b94415f2](https://github.com/argoproj/argo-workflows/commit/0b94415f2a92b85c886e97d69d82bf3a8641a30f) chore(deps): update module go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp to v1.44.0 (main) (#16208)
* [00ff8ef92](https://github.com/argoproj/argo-workflows/commit/00ff8ef924bcf6c136a74ebe1dc2a7329b6d525d) chore(deps): update module golang.org/x/crypto to v0.52.0 [security] (main) (#16128)
* [643da1ec4](https://github.com/argoproj/argo-workflows/commit/643da1ec49992d0adf5465c73fbc171edc157c00) chore(deps): update module github.com/aws/aws-sdk-go-v2/feature/rds/auth to v1.6.27 (#16196)
* [3b0a1484f](https://github.com/argoproj/argo-workflows/commit/3b0a1484f8e9ccf8979e4e5b8214cc96c29bf745) chore(deps): update module github.com/aws/aws-sdk-go-v2/credentials to v1.19.21 (main) (#16195)
* [246b8d746](https://github.com/argoproj/argo-workflows/commit/246b8d746f22cfd55e6e1b4a453b67fe581c4c12) chore(deps): update module github.com/prometheus/common to v0.68.1 (main) (#16200)
* [eef18d1cd](https://github.com/argoproj/argo-workflows/commit/eef18d1cdce83220ed6c226c74d0d6d19d63b9e5) fix: replace deprecated h2c.NewHandler with http.Server.Protocols (#16220)
* [89c5b6ef0](https://github.com/argoproj/argo-workflows/commit/89c5b6ef01e124d472ab686cd1048f9c9db5ee7d) chore(deps): update distroless base image (#16191)
* [b8ce2717c](https://github.com/argoproj/argo-workflows/commit/b8ce2717ce57d36b118be052500645a865947303) Revert "chore(deps): update module go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc to v1.44.0 (main)" (#16219)
* [4615290ab](https://github.com/argoproj/argo-workflows/commit/4615290ab7ba6e490224f93db31848d8f362db47) chore(deps): update docker/build-push-action action to v7.2.0 (main) (#16177)
* [d0bfb3868](https://github.com/argoproj/argo-workflows/commit/d0bfb386857732c5f20d6195aa50d7effa31bff1) chore(deps): update module go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp to v0.69.0 (main) (#16204)
* [269b57bc5](https://github.com/argoproj/argo-workflows/commit/269b57bc5d40f124f223df89f4bf55881c139011) chore(deps): update k8s.io/gengo digest to 25e2208 (main) (#16193)
* [30fdffb4f](https://github.com/argoproj/argo-workflows/commit/30fdffb4fd06c6f9ba208969f7286db7beb8dc1b) chore(deps): update module google.golang.org/api to v0.283.0 (main) (#16211)
* [f06c08dd3](https://github.com/argoproj/argo-workflows/commit/f06c08dd31328a43c60bddffe4b5d04e96a87ef7) chore(deps): update module go.opentelemetry.io/otel/exporters/prometheus to v0.66.0 (main) (#16209)
* [30187c766](https://github.com/argoproj/argo-workflows/commit/30187c7661de95c8e3380af4145727997b517c12) chore(deps): update module go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc to v1.44.0 (main) (#16205)
* [491c8ac7e](https://github.com/argoproj/argo-workflows/commit/491c8ac7eb7ea5b5aedfb0879d092156a5192058) chore(deps): update actions/setup-node action to v6.4.0 (main) (#16175)
* [3f91cb2ea](https://github.com/argoproj/argo-workflows/commit/3f91cb2eabd22665ec6e649886fe6445908920f0) chore(deps): update module golang.org/x/term to v0.43.0 (main) (#16210)
* [6b56479e7](https://github.com/argoproj/argo-workflows/commit/6b56479e7ea25e4d0534eaa03a12bceed9300c05) chore(deps): update dependency qs to v6.15.2 [security] (main) (#16131)
* [614217f42](https://github.com/argoproj/argo-workflows/commit/614217f4255b8ac3c2f1079fffe152d8ae368161) chore(deps): update module github.com/google/go-containerregistry to v0.21.6 (main) (#16198)
* [24130e018](https://github.com/argoproj/argo-workflows/commit/24130e018ab86df1a7ae528f25d86a56879f70ec) chore(deps): update google.golang.org/genproto/googleapis/api digest to 3dc84a4 (main) (#16192)
* [533142eeb](https://github.com/argoproj/argo-workflows/commit/533142eeb7746af73d6ead2372f8fd695e2a1c59) chore(deps): update docker/setup-buildx-action action to v4.1.0 (main) (#16180)
* [bf2d8f09c](https://github.com/argoproj/argo-workflows/commit/bf2d8f09cf62331f879f6e1451de750431f4edfd) chore(deps): update module github.com/tidwall/gjson to v1.19.0 (main) (#16202)
* [45e421448](https://github.com/argoproj/argo-workflows/commit/45e421448befb19dc168f3d202cbcadd3453948b) chore(deps): update module go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws to v0.69.0 (main) (#16203)
* [2df4348b1](https://github.com/argoproj/argo-workflows/commit/2df4348b16501228ccd441fb064f6e2a4811ed76) chore(deps): update ubuntu docker tag to v26 (main) (#16215)
* [f97e535ef](https://github.com/argoproj/argo-workflows/commit/f97e535efd285f0708d256c8f081a13a93f1dd89) chore(deps): update module github.com/go-openapi/jsonreference to v0.21.6 (#16197)
* [a22aafd17](https://github.com/argoproj/argo-workflows/commit/a22aafd17f42515b41984ccfb1921d4c09cb8906) chore(deps): update codecov/codecov-action action to v6.0.1 (#16173)
* [9cbd7bf12](https://github.com/argoproj/argo-workflows/commit/9cbd7bf1293260cf9cb8cd493a18e8e1d67905c6) chore(deps): update actions/checkout action to v6.0.3 (#16172)
* [01e160f19](https://github.com/argoproj/argo-workflows/commit/01e160f19cadfa5d95e8eed416a5476ba9c5814e) chore(deps): update module github.com/alibabacloud-go/tea to v1.5.0 (main) (#16182)
* [3fa31c6ab](https://github.com/argoproj/argo-workflows/commit/3fa31c6ab3166420c2bdfb41135cb32ac311d470) chore(deps): update module go.opentelemetry.io/contrib/instrumentation/runtime to v0.69.0 (main) (#16170)
* [8b3a547bb](https://github.com/argoproj/argo-workflows/commit/8b3a547bbfd67d2ada71dd7a6da429ffe01fc1f1) chore(deps): update module github.com/fsnotify/fsnotify to v1.10.1 (main) (#16184)
* [9215a7858](https://github.com/argoproj/argo-workflows/commit/9215a78581751acdd998f95b7c6f5b115fc07db8) chore(deps): update tj-actions/changed-files action to v47.0.6 (#16166)
* [b77d9a652](https://github.com/argoproj/argo-workflows/commit/b77d9a652fc34b2a0c11c3abd36edb10f1a3b5d2) chore(deps): update ubuntu docker tag to mantic-20240530 (#16167)
* [d4a68305a](https://github.com/argoproj/argo-workflows/commit/d4a68305aa40004b621f668257aac1d3ef0d9018) chore(deps): update module github.com/prometheus/common to v0.68.0 (main) (#16187)
* [6ba8d501b](https://github.com/argoproj/argo-workflows/commit/6ba8d501b2ec2ab7503658523bdeb7a40b5dd860) chore(deps): update softprops/action-gh-release action to v3 (main) (#16188)
* [a9def2785](https://github.com/argoproj/argo-workflows/commit/a9def27858681ba8e601e9f0704ef85013189ffb) chore(deps): update module cloud.google.com/go/storage to v1.62.2 (#16174)
* [6961eb916](https://github.com/argoproj/argo-workflows/commit/6961eb916f0ea98b4f098c9398fd2b089ef1cf13) chore(deps): update docker/setup-qemu-action action to v4.1.0 (main) (#16181)
* [be7703f7e](https://github.com/argoproj/argo-workflows/commit/be7703f7e9d5b41fb9afe8251b5b74a68c9026bb) chore(deps): update docker/dockerfile docker tag to v1.24 (main) (#16178)
* [a6815184c](https://github.com/argoproj/argo-workflows/commit/a6815184c47a5792382e4818a2478da80dd7e2bb) chore(deps): update dependabot/fetch-metadata action to v3.1.0 (main) (#16176)
* [2fb482217](https://github.com/argoproj/argo-workflows/commit/2fb4822174d2abda72810a3bdb5a8c101e06ca63) chore(deps): update docker/login-action action to v4.2.0 (main) (#16179)
* [824ab0f0e](https://github.com/argoproj/argo-workflows/commit/824ab0f0ee11c1c3eb8d5816a61399464e97a96e) chore(deps): update module github.com/go-sql-driver/mysql to v1.10.0 (main) (#16185)
* [4d74a3d21](https://github.com/argoproj/argo-workflows/commit/4d74a3d218816f4e987868f7ce1fdcfdb2d18b9a) chore(deps): update actions/stale action to v10.3.0 (main) (#16169)
* [4eea30e2a](https://github.com/argoproj/argo-workflows/commit/4eea30e2ac795549413963d36709822a0cba8edd) chore(deps): update k8s.io/utils digest to ff6756f (main) (#16163)
* [aeadb9c57](https://github.com/argoproj/argo-workflows/commit/aeadb9c5719426017bee229d0a328b9ce1b14726) chore(deps): update actions/create-github-app-token action to v3.2.0 (main) (#16168)
* [0f1164ee0](https://github.com/argoproj/argo-workflows/commit/0f1164ee088de1d15ba9d9019e16a508a95b0457) chore(deps): update sigstore/cosign-installer action to v4.1.2 (#16165)
* [134224f57](https://github.com/argoproj/argo-workflows/commit/134224f57b33c5abdadd515213ea9302d9ea704f) fix: populate scope with empty values for outputs of skipped/omitted … (#15932)
* [2fc922d95](https://github.com/argoproj/argo-workflows/commit/2fc922d950bc5746001176edce8a095209ee4cee) fix(ci): use client-id for GitHub Action "Create GitHub App Token" (#16156)
* [8e52e7aa9](https://github.com/argoproj/argo-workflows/commit/8e52e7aa93ad68eee6275c8835d2f1383e13c4eb) fix: change log level because behavior is expected (#16124)
* [96d1684b9](https://github.com/argoproj/argo-workflows/commit/96d1684b938fbf0bfda7bd2f80c3e960eb65f29c) fix: MariaDB compatibility for JSON queries and native password auth. Fixes #15413 (#15936)
* [351d8cd6c](https://github.com/argoproj/argo-workflows/commit/351d8cd6c81e4e4bab5b5fdd66aaeb73804ab3b7) feat: added config option to disable agent pod creation. Fixes #7891 (#15844)
* [89047a9a5](https://github.com/argoproj/argo-workflows/commit/89047a9a53ab54cbae39959becd94fe92b10ab5d) fix: metadata merge. Fixes #15870 (#16103)
* [6cc0d115b](https://github.com/argoproj/argo-workflows/commit/6cc0d115b7ab4b4a299ef5b46e699f6d53cfae61) chore(deps): update module golang.org/x/sys to v0.44.0 [security] (main) (#16136)
* [3d5ebbc5c](https://github.com/argoproj/argo-workflows/commit/3d5ebbc5c664f9b45e00e7567ca4807693fff9b6) fix: remove redundant WaitForWorkflow in TestInputOnMount (#15982)
* [202199b82](https://github.com/argoproj/argo-workflows/commit/202199b82033deba1a672062ae9d39cb88102293) fix: ignore resource version (match) when continue token present (#16099)
* [7051e4e1a](https://github.com/argoproj/argo-workflows/commit/7051e4e1ad7c105d9b15675baeae15567bff2412) feat(ui): autocomplete namespace filter from displayed resources (refs #7405) (#16087)
* [7cce696e0](https://github.com/argoproj/argo-workflows/commit/7cce696e0375b431d22e99b1fc9b7ec929aa0f28) chore(deps): bump github.com/go-git/go-git/v5 from 5.19.0 to 5.19.1 in the go_modules group across 1 directory (#16114)
* [90a3252f3](https://github.com/argoproj/argo-workflows/commit/90a3252f386c9bcbf9fee9d53aede8a3cfa8f9fa) fix(validate): validate placeholder step names (#15991)
* [351c30607](https://github.com/argoproj/argo-workflows/commit/351c306070548bc1403ff17d5829cc3bd65e1eef) chore(deps): bump github.com/go-git/go-git/v5 from 5.18.0 to 5.19.0 in the go_modules group across 1 directory (#16081)
* [80dc102f4](https://github.com/argoproj/argo-workflows/commit/80dc102f42b0867019e3464e9a341bc3e6bfa310) fix(ui): toggle filter dropdown closed when clicking anchor (#16014)
* [0eb6267ac](https://github.com/argoproj/argo-workflows/commit/0eb6267ac9f3a6a4ae4c3a9e50b065b038b3a237) feat(ui): Add namespace input to ClusterWorkflowTemplate submit. Fixes #10398 (#13596)
* [7a7e60839](https://github.com/argoproj/argo-workflows/commit/7a7e60839796b0cb3a1343f2ec0968bd2ac2d694) fix: classify bare 5xx S3 responses as transient. Fixes #15565 (#16016)
* [ab34c01ca](https://github.com/argoproj/argo-workflows/commit/ab34c01caca3b271d5d3e7c1194a5ee8d46803d3) chore(deps): update module github.com/argoproj/argo-workflows/v4 to v4.0.5 [security] (main) (#16032)
* [4cdc4b6e7](https://github.com/argoproj/argo-workflows/commit/4cdc4b6e7a049a46416019ecb2b06dac09f8b095) chore(deps): update module github.com/jackc/pgx/v5 to v5.9.2 [security] (main) (#16025)
* [c7cdd0da8](https://github.com/argoproj/argo-workflows/commit/c7cdd0da828be1ce84c2ca02d3abefd00d4b04b7) fix(ui): respect target field in workflow-list scope links (#16021)
* [4d9f021f2](https://github.com/argoproj/argo-workflows/commit/4d9f021f2380951d1dbd394ffa55f9948cc0c4b7) Merge commit from fork
* [7dc19c8a3](https://github.com/argoproj/argo-workflows/commit/7dc19c8a3306a5085ae92ceab56726933fb1aa9e) Merge commit from fork
* [3ebc2aee4](https://github.com/argoproj/argo-workflows/commit/3ebc2aee49b04efbe79c10f59143b3536e1fabe7) Merge commit from fork
* [d2568b549](https://github.com/argoproj/argo-workflows/commit/d2568b549a22bc86da8369365f7b6ef723d7b5d3) Merge commit from fork
* [203fc9517](https://github.com/argoproj/argo-workflows/commit/203fc95178c79b63ec4863fce61b46958c2f3156) Merge commit from fork
* [ec96b3ee5](https://github.com/argoproj/argo-workflows/commit/ec96b3ee520853537904a340ef5ddcb2b8278edb) Merge commit from fork
* [ab931dce7](https://github.com/argoproj/argo-workflows/commit/ab931dce79ff7b53c0d0196c4278333a7917ccb2) fix: prevent `failed to get a template` when using inline template. Fixes #15051 (#15574)
* [1fb836c8a](https://github.com/argoproj/argo-workflows/commit/1fb836c8a49ad6b57fe03626124ecc06a13f29c1) fix(controller): guard realtime workflow.duration against zero StartedAt (#15935)
* [87699371c](https://github.com/argoproj/argo-workflows/commit/87699371c544abbed4d7149f5d5f7ced6640136f) chore(deps): update module go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws to v0.68.0 (main) (#15973)
* [e291b59d7](https://github.com/argoproj/argo-workflows/commit/e291b59d757b903c1043ee2ffafc9c920de0f7aa) fix(ui): sort user info groups alphabetically for readability (#15940)
* [30c3c9d3c](https://github.com/argoproj/argo-workflows/commit/30c3c9d3cb5bd84e68a83869c90e0175d47ead1e) chore(deps): update codecov/codecov-action action to v6 (main) (#15987)
* [4b0fd6322](https://github.com/argoproj/argo-workflows/commit/4b0fd63227281003d103c3c31cbb672d36fc5950) fix: 401s when accessing artifact directories with SSO enabled. Fixes #15800 (#15994)
* [8ca2732c1](https://github.com/argoproj/argo-workflows/commit/8ca2732c1ac174110035455cda91bc751d6099eb) chore(deps): update module k8s.io/kubectl to v0.35.4 (main) (#15970)
* [47846fc0e](https://github.com/argoproj/argo-workflows/commit/47846fc0e454e749d913400ab22da6721d75ac4c) chore(deps): update module github.com/aws/aws-sdk-go-v2/config to v1.32.16 (main) (#15961)
* [23e19896c](https://github.com/argoproj/argo-workflows/commit/23e19896cbdfa2fab4d5344f9f2037be2a8b0b9a) chore(deps): update module github.com/argoproj/argo-workflows/v4 to v4.0.4 (main) (#15984)
* [969c20c36](https://github.com/argoproj/argo-workflows/commit/969c20c3622592b40208c7e0301f6466cf4a9d17) chore(deps): update module google.golang.org/api to v0.276.0 (main) (#15985)
* [5a093f7c3](https://github.com/argoproj/argo-workflows/commit/5a093f7c3b67aab7395bbc08d922d4e569dbb333) chore(deps): update module k8s.io/cli-runtime to v0.35.4 (main) (#15968)
* [6934eb68c](https://github.com/argoproj/argo-workflows/commit/6934eb68c741216265f595259dd4ebeba6e0b787) chore(deps): update module k8s.io/apiextensions-apiserver to v0.35.4 (main) (#15966)
* [e9a32361e](https://github.com/argoproj/argo-workflows/commit/e9a32361ee49c6c7dd7af8f5f77c437b2703cb13) chore(deps): update module k8s.io/apimachinery to v0.35.4 (main) (#15967)
* [89cee605c](https://github.com/argoproj/argo-workflows/commit/89cee605ccfcfab7e6bdacab3a2c6e3a699934c4) chore(deps): update github.com/google/go-containerregistry/pkg/authn/k8schain digest to f80cb9a (main) (#15959)
* [025feea70](https://github.com/argoproj/argo-workflows/commit/025feea70d1196493255b220f911ccf9143646a4) chore(deps): update actions/create-github-app-token action to v3 (main) (#15986)
* [41e42db1a](https://github.com/argoproj/argo-workflows/commit/41e42db1a827c3060f0b793134a58badb09c32ed) chore(deps): update dependabot/fetch-metadata action to v3 (main) (#15988)
* [fca8ce4fb](https://github.com/argoproj/argo-workflows/commit/fca8ce4fb2ae0a2de7d06da14b177870ef56f240) chore(deps): update actions/cache action to v5.0.5 (main) (#15983)
* [ddeb3eb10](https://github.com/argoproj/argo-workflows/commit/ddeb3eb103275c13d29dc6627a3fc45f9e532a53) chore(deps): update google.golang.org/genproto/googleapis/api digest to afd174a (main) (#15960)
* [b6242a91b](https://github.com/argoproj/argo-workflows/commit/b6242a91baa61befe7e6b11b3c976f625b0e8980) chore(deps): update module go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc to v1.43.0 (main) (#15975)
* [2b18e6cd3](https://github.com/argoproj/argo-workflows/commit/2b18e6cd37818cd4d97abd3e76fd90321d525d30) chore(deps): update module go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc to v1.43.0 (main) (#15974)
* [c2d366fe8](https://github.com/argoproj/argo-workflows/commit/c2d366fe8c36ca5e65f032c0e23e967bcce71b49) chore(deps): update module k8s.io/api to v0.35.4 (main) (#15965)
* [bec82ce0e](https://github.com/argoproj/argo-workflows/commit/bec82ce0ef618a69d621d315a4506bc58a8115f7) chore(deps): update module github.com/azure/azure-sdk-for-go/sdk/azcore to v1.21.1 (main) (#15964)
* [d3268bd77](https://github.com/argoproj/argo-workflows/commit/d3268bd77a5d4ead4fab519f9f04e2f243b2e90a) chore(deps): update module github.com/aws/aws-sdk-go-v2/credentials to v1.19.14 (main) (#15962)
* [4e488f86c](https://github.com/argoproj/argo-workflows/commit/4e488f86ca28953e0ee83cda71b53e4b766efe5e) chore(deps): update module github.com/go-git/go-git/v5 to v5.18.0 (main) (#15972)
* [026a94ef2](https://github.com/argoproj/argo-workflows/commit/026a94ef21e50f488ba57db9aeb9e951e9802753) chore(deps): update module github.com/coreos/go-oidc/v3 to v3.18.0 (main) (#15971)
* [75a118d1a](https://github.com/argoproj/argo-workflows/commit/75a118d1ab9d453ae9d9f182ac9a2521dbf40bdb) chore(deps): update module go.opentelemetry.io/otel/exporters/prometheus to v0.65.0 (main) (#15976)
* [b2e623d12](https://github.com/argoproj/argo-workflows/commit/b2e623d121655320fcac1c0d4d4329c3f270d787) chore(deps): update module github.com/jackc/pgx/v5 to v5.9.0 [security] (main) (#15945)
* [a7eb4c531](https://github.com/argoproj/argo-workflows/commit/a7eb4c53134153395102031684dc75974fc6e4d1) chore(deps): bump github.com/jackc/pgx/v5 from 5.7.5 to 5.9.0 in the go_modules group across 1 directory (#15955)
* [1df64a17e](https://github.com/argoproj/argo-workflows/commit/1df64a17ee738be3ab4e151fd24ee2b2d68409e8) chore(deps): bump github.com/moby/spdystream from 0.5.0 to 0.5.1 in the go_modules group across 1 directory (#15951)
* [51ea54b6c](https://github.com/argoproj/argo-workflows/commit/51ea54b6cf6a34859e2c892c1ffee7b7f9a38f2f) fix: delete stale TaskGroup children on retry with parameter override. Fixes #15802 (#15827)
* [50854f788](https://github.com/argoproj/argo-workflows/commit/50854f7882a7a170e360034c6a11c5dfc1383de4) chore(deps): update module github.com/testcontainers/testcontainers-go/modules/mysql to v0.42.0 (main) (#15921)
* [5b543853e](https://github.com/argoproj/argo-workflows/commit/5b543853e0b3a772950d5c23efeff10a7ae95f1d) chore(deps): update module github.com/testcontainers/testcontainers-go/modules/postgres to v0.42.0 (main) (#15922)
* [939ff851e](https://github.com/argoproj/argo-workflows/commit/939ff851eab4bd9d30da10371c5400fb48fdf537) chore(deps): update module github.com/testcontainers/testcontainers-go to v0.42.0 (main) (#15920)
* [972aad148](https://github.com/argoproj/argo-workflows/commit/972aad148681f588cd705685bb6dc9973e0bfb8e) chore(deps): bump follow-redirects from 1.15.6 to 1.16.0 in /ui in the deps group across 1 directory (#15930)
* [b8139b0fb](https://github.com/argoproj/argo-workflows/commit/b8139b0fbd062146f502cfe16dc291f912a2c5b5) chore(deps): update module cloud.google.com/go/storage to v1.62.1 (main) (#15919)
* [8d4e02a65](https://github.com/argoproj/argo-workflows/commit/8d4e02a65d72dde7f92705c1c2ef200e34b3dfd2) chore(deps): update actions/setup-go action to v6.4.0 (main) (#15916)
* [248935d7a](https://github.com/argoproj/argo-workflows/commit/248935d7a145c34dbc588dbc501256d22848463a) chore(deps): update module github.com/aliyun/credentials-go to v1.4.12 (main) (#15910)
* [7671d6462](https://github.com/argoproj/argo-workflows/commit/7671d6462dc1355a78c91073b1850673ebda884e) chore(deps): update softprops/action-gh-release digest to 3bb1273 (main) (#15905)
* [8a19de7eb](https://github.com/argoproj/argo-workflows/commit/8a19de7eb09268750134387b8ac215836e5cc9ea) chore(deps): update docker/build-push-action action to v7.1.0 (main) (#15917)
* [489298677](https://github.com/argoproj/argo-workflows/commit/489298677a6902b0f16d825f1ff655c07093328d) chore(deps): update docker/login-action action to v4.1.0 (main) (#15918)
* [d4b349066](https://github.com/argoproj/argo-workflows/commit/d4b3490661db6f70d1b88e690d2333cb9c1d7063) chore(deps): update module github.com/xsam/otelsql to v0.42.0 (main) (#15923)
* [906b4e6f5](https://github.com/argoproj/argo-workflows/commit/906b4e6f5e22187bb93661953ecfbdead4c0bb2d) chore(deps): update module go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp to v0.68.0 (main) (#15924)
* [f76e718a4](https://github.com/argoproj/argo-workflows/commit/f76e718a4a62024db03bb30c3bc8e548ce687e35) chore(deps): update module github.com/google/go-containerregistry to v0.21.5 (main) (#15912)
* [dcc17b55a](https://github.com/argoproj/argo-workflows/commit/dcc17b55ab13a2f06584d63e70ed7ae0fc9c3ce3) chore(deps): update minio-go to include non-DualStack region fix (#2205) (#15838)
* [0f6ecce69](https://github.com/argoproj/argo-workflows/commit/0f6ecce6923d040856fe5d8dcd48b7b5602219dc) chore(deps): update module go.opentelemetry.io/contrib/instrumentation/runtime to v0.68.0 (main) (#15925)
* [3525b5efd](https://github.com/argoproj/argo-workflows/commit/3525b5efd89367bd77cc201a749a9f5f32fa60fb) chore(deps): update module github.com/itchyny/gojq to v0.12.19 (main) (#15913)
* [65c825df1](https://github.com/argoproj/argo-workflows/commit/65c825df1516470b26ca78dec8225f40b39d6f5d) chore(deps): update module github.com/go-git/go-git/v5 to v5.17.2 (main) (#15911)
* [9d0adf46f](https://github.com/argoproj/argo-workflows/commit/9d0adf46fd1ace6c54702836114474da4b70700f) chore(deps): update module github.com/lib/pq to v1.12.3 (main) (#15914)
* [f40d0912b](https://github.com/argoproj/argo-workflows/commit/f40d0912ba85bd43e447878854e28c01974d407b) chore(deps): update zgosalvez/github-actions-ensure-sha-pinned-actions action to v5.0.4 (main) (#15915)
* [c3e5175a4](https://github.com/argoproj/argo-workflows/commit/c3e5175a47b0c5eb58f70e891abc45375757d989) chore(deps): update codecov/codecov-action action to v5.5.4 (main) (#15907)
* [88a5b0509](https://github.com/argoproj/argo-workflows/commit/88a5b05096aaf4f2bbdffd79a546604f0e74729e) chore(deps): update actions/upload-artifact action to v7.0.1 (main) (#15906)
* [d2df9823d](https://github.com/argoproj/argo-workflows/commit/d2df9823d018e8fbd9d9bdd79729870b8bfc8c92) chore(deps): update module go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp to v1.43.0 [security] (main) (#15886)
* [f32df71d1](https://github.com/argoproj/argo-workflows/commit/f32df71d19bcc250f119f1b2b29af498c5b72ae8) chore(deps): update sigstore/cosign-installer action to v4.1.1 (main) (#15909)
* [e71fbe65f](https://github.com/argoproj/argo-workflows/commit/e71fbe65fde112f4691082c1be07fed3e2dedab9) chore(deps): update peter-evans/create-pull-request action to v8.1.1 (main) (#15908)
* [c365a1097](https://github.com/argoproj/argo-workflows/commit/c365a1097a64d5ffe6768e237d2b7af53a5ec1ff) chore(deps): bump go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp from 1.42.0 to 1.43.0 in the go_modules group across 1 directory (#15901)
* [158aec31b](https://github.com/argoproj/argo-workflows/commit/158aec31b05ffe2a7b8171177b39ec5b1f455f73) chore(deps): update dependency @types/react-datepicker to v4.19.6 (main) (#15333)
* [6b71c2974](https://github.com/argoproj/argo-workflows/commit/6b71c2974e5c26a16dcfa6d3117d3618c43a70d0) chore(deps): update dependency @types/react-form to v2.16.15 (main) (#15349)
* [f7bcf22ee](https://github.com/argoproj/argo-workflows/commit/f7bcf22ee927ba4d3c1e7d99d917a32c2372b131) chore(deps): update module go.opentelemetry.io/otel/sdk to v1.43.0 [security] (main) (#15888)
* [670ab1b64](https://github.com/argoproj/argo-workflows/commit/670ab1b6450462fe90fd5ce6f0e8932b1ba33ea6) fix: changed log level (#15898)
* [065d443f3](https://github.com/argoproj/argo-workflows/commit/065d443f37347c5d70ed49d0f35163a957b353ac) fix(test): for "CI / Windows Unit Tests", skip TestSessionReconnect (#15895)
* [88a2b50d1](https://github.com/argoproj/argo-workflows/commit/88a2b50d1ec6b39dcdc4d310754defaec74508c6) fix: downgrade serialization level (#15881)
* [b5e8a3e6d](https://github.com/argoproj/argo-workflows/commit/b5e8a3e6d2edeab91657d4cd2d2863431b615631) fix(ui): add `target='_blank' rel='noreferrer'` to existing external links (#15860)
* [8d62590a3](https://github.com/argoproj/argo-workflows/commit/8d62590a3319ba5302d31d33abe9d725e3c7f83e) feat: retry connections on network errors. Fixes #15011 (#15006)
* [d9d23b4ed](https://github.com/argoproj/argo-workflows/commit/d9d23b4ed5634ceadffe2d300f53ce995b3fd26e) fix(ui): add `target='_blank' rel='noreferrer'` to logs link (#15871)
* [9c77acc59](https://github.com/argoproj/argo-workflows/commit/9c77acc59949e4595775b379800c3e82fa1e75d1) chore(deps): bump github.com/go-jose/go-jose/v4 from 4.1.3 to 4.1.4 in the go_modules group across 1 directory (#15855)
* [e45bf64fb](https://github.com/argoproj/argo-workflows/commit/e45bf64fb92fa21c721c084f291c110a06cbc2f5) chore(deps): bump lodash from 4.17.23 to 4.18.1 in /ui in the deps group across 1 directory (#15853)
* [6e0d9c1d6](https://github.com/argoproj/argo-workflows/commit/6e0d9c1d62057c355900ddb6756a02f8f7bb285b) fix(test): increase ListAll archive poll timeout (#15816)
* [0e742e6d7](https://github.com/argoproj/argo-workflows/commit/0e742e6d7f4d23c9f242e60c4f39e8751f1bc9e1) fix: tolerate expression template runtime failures when allowUnresolved is true. Fixes #15832, #15824 (#15839)
* [a7e51586b](https://github.com/argoproj/argo-workflows/commit/a7e51586b80c773f3d3d48a820deffaaedaad465) fix(test): poll in TestWorkflowArchiveServiceList (#15818)
* [9a363465b](https://github.com/argoproj/argo-workflows/commit/9a363465b085c77126743cc43ed3b6ee02b02d36) fix: populate scope with empty values for outputs of skipped/omitted DAG ancestors (#15841)
* [725f2dfe7](https://github.com/argoproj/argo-workflows/commit/725f2dfe7317c32b8b586ad6f651041d5edbea94) fix(ui): add tooltips to tab icons (#15840)
* [961553b9f](https://github.com/argoproj/argo-workflows/commit/961553b9f15dc921999b92c10069d318e67ab849) chore(deps): bump lodash-es from 4.17.23 to 4.18.1 in /ui in the deps group across 1 directory (#15845)
* [1ec82d0c4](https://github.com/argoproj/argo-workflows/commit/1ec82d0c48d8e10ed08363d5ac9e82da239a87df) fix(ui): retain query parameters when closing workflow side panel. Fixes #15795 (#15797)
* [c5b41e5a9](https://github.com/argoproj/argo-workflows/commit/c5b41e5a9c9c96309f780d883ff73f5417205e43) fix(log): honor stderrthreshold when logtostderr is enabled (#15814)
* [0283f79db](https://github.com/argoproj/argo-workflows/commit/0283f79db7e8ad1125e623e7bde59a603e106623) chore(deps): update module github.com/google/go-containerregistry to v0.21.3 (main) (#15835)
* [2b4744d34](https://github.com/argoproj/argo-workflows/commit/2b4744d34dad49a39f4072321474149f40fd1f43) chore(deps): update github.com/google/go-containerregistry/pkg/authn/k8schain digest to c612a9b (main) (#15288)
* [0f407abb5](https://github.com/argoproj/argo-workflows/commit/0f407abb5030f3081aaf34877c955c649b4cbdb3) chore(deps): update k8s.io/kube-openapi digest to 16be699 (main) (#15297)
* [dd9b143fb](https://github.com/argoproj/argo-workflows/commit/dd9b143fba4dff124d342e2d5289618443838afa) feat: added support for AWS RDS via IAM. Fixes #15834 (#15833)
* [eba2ac841](https://github.com/argoproj/argo-workflows/commit/eba2ac841ba47058076adf897817f3fd0b0f8b52) chore(deps): migrate to go.yaml.in/yaml/v3 (#15815)
* [6a8e6462d](https://github.com/argoproj/argo-workflows/commit/6a8e6462da781af61b2d1ea4c8c37022c85bc2f6) chore(deps): update module github.com/lib/pq to v1.12.1 (main) (#15836)
* [17e58ee07](https://github.com/argoproj/argo-workflows/commit/17e58ee07a36976a66480d3479f108b0785013af) fix(ui): add margin-top to WidgetGallery (#15826)
* [54c127b65](https://github.com/argoproj/argo-workflows/commit/54c127b6512162931c58bc1a3fb5c1def8dc3a54) fix(test): clean up sync configmap between re-runs (#15817)
* [3338ceff4](https://github.com/argoproj/argo-workflows/commit/3338ceff45f58b3049149de62376a47a2544e265) chore(deps): bump github.com/go-git/go-git/v5 from 5.17.0 to 5.17.1 in the go_modules group across 1 directory (#15828)
* [1e95e6de7](https://github.com/argoproj/argo-workflows/commit/1e95e6de7fafde948f47dfb5c656892239ba3428) perf(sqldb): batch insert archived workflow labels. (#15821)
* [23266a784](https://github.com/argoproj/argo-workflows/commit/23266a7849a49fe92b20e62630a1ed70eee66fd9) fix: remove holder from waiting list when semaphore lock is acquired. (#15239)
* [17280a2cb](https://github.com/argoproj/argo-workflows/commit/17280a2cb471a8e94ff4392f1a4400cc4bc8c018) fix(ui): add margin-top to workflow-creator (#15806)
* [acfaee622](https://github.com/argoproj/argo-workflows/commit/acfaee6221041c901eab31c8abd3bf57e2aac995) fix(ui): update webpack to 5.105.4 and webpack-dev-server to v5 (security) (#15799)
* [6c14b107a](https://github.com/argoproj/argo-workflows/commit/6c14b107a576c77cd096c9669fc0ebb4ea31b1b2) fix(ui): populate URL filter parameters on first load. Fixes #15794 (#15796)
* [84779fb11](https://github.com/argoproj/argo-workflows/commit/84779fb11facd28a014dd8af86cf15689295b759) chore(deps): bump yaml from 2.5.1 to 2.8.3 in /ui in the deps group across 1 directory (#15811)
* [bf4d83a04](https://github.com/argoproj/argo-workflows/commit/bf4d83a04f4a7fa8e19eb364ca3c95d28c22fa4b) chore(deps): bump picomatch from 2.3.1 to 2.3.2 in /ui in the deps group across 1 directory (#15807)
* [93385709f](https://github.com/argoproj/argo-workflows/commit/93385709f8dc309b092bef477d666a09fad41792) fix: webhook dispatch applies template metadata (#15798)
* [673920c37](https://github.com/argoproj/argo-workflows/commit/673920c3723042cdcd31a0f324b36ceceeeef3b7) fix(test): replace flaky sleep with WaitForPod in TestDeletingPendingPod (#15793)
* [dbeccfc06](https://github.com/argoproj/argo-workflows/commit/dbeccfc06a295b2f186510af9fed14e40f19a9d9) chore(deps): update module k8s.io/apimachinery to v0.35.3 (main) (#15783)
* [03ebaaca0](https://github.com/argoproj/argo-workflows/commit/03ebaaca08c692015338fc88e2fbdff75840cc34) chore(deps): update module k8s.io/api to v0.35.3 (main) (#15781)
* [5456d80f2](https://github.com/argoproj/argo-workflows/commit/5456d80f25c2437c4d717e6e5f266e26ab496449) chore(deps): update module k8s.io/apiextensions-apiserver to v0.35.3 (main) (#15782)
* [cdc0347ce](https://github.com/argoproj/argo-workflows/commit/cdc0347ce0dff0e9fb63d5945fbd5670fce69d39) fix: retry workflow archiving on transient DB errors (#15780)
* [3e50fd9d6](https://github.com/argoproj/argo-workflows/commit/3e50fd9d6f700fe60b976c76e40ad85d4e27d882) fix: close stdout fd when combined log file open fails (#15790)
* [f22735c8f](https://github.com/argoproj/argo-workflows/commit/f22735c8f6abc55fc261e084b2fb63fd50333d61) chore(deps): update module k8s.io/kubectl to v0.35.3 (main) (#15786)
* [09ce7b009](https://github.com/argoproj/argo-workflows/commit/09ce7b009396de80b82befb6938f168b487160ea) fix: poll for Persisted status in TestWorkflowServiceListArchived/ListAll (#15777)
* [89b9d6989](https://github.com/argoproj/argo-workflows/commit/89b9d698964d00bac4d1c1f5d3f33504b365954d) chore(deps): bump flatted from 3.2.9 to 3.4.2 in /ui in the deps group across 1 directory (#15789)
* [9876cdfad](https://github.com/argoproj/argo-workflows/commit/9876cdfada816fd35ed1021634a057045ed2a171) chore(deps): update module google.golang.org/api to v0.272.0 (main) (#15721)
* [353c1ce69](https://github.com/argoproj/argo-workflows/commit/353c1ce69fba154538845ebe967f3d5f9b4f27a3) chore(deps): update module github.com/lib/pq to v1.12.0 (main) (#15763)
* [90bf686e9](https://github.com/argoproj/argo-workflows/commit/90bf686e9ed2375d2d8f333c619be9eec75e4dc4) fix(test): prevent cascading subtest failures in ArgoServerSuite e2e tests (#15776)
* [7f8605892](https://github.com/argoproj/argo-workflows/commit/7f860589285b6ab49249d566b54cc2d82a33afc6) chore(deps): update module google.golang.org/grpc to v1.79.3 [security] (main) (#15769)
* [a6c087f5c](https://github.com/argoproj/argo-workflows/commit/a6c087f5cc9a0506c446f96da6698cdf93aecd59) chore(deps): update module k8s.io/klog/v2 to v2.140.0 (main) (#15764)
* [55678c8a6](https://github.com/argoproj/argo-workflows/commit/55678c8a6b84fc70b9f80dfcbab2837a949c1424) chore(deps): update actions/create-github-app-token action to v2.2.2 (main) (#15762)
* [c306137ef](https://github.com/argoproj/argo-workflows/commit/c306137ef31416e48e32a435c52f139ff5df4055) chore(deps): golang to 1.26 (#15631)
* [7824aa613](https://github.com/argoproj/argo-workflows/commit/7824aa613546b8e7cb7d406d79768947805ff802) chore(deps): update actions/cache action to v5.0.4 (main) (#15761)
* [7fe62c830](https://github.com/argoproj/argo-workflows/commit/7fe62c8302e42fd69f54bdfee81a93607d7252b0) chore(deps): update k8s.io libraries to v0.35.1 (#15573)
* [3b00da074](https://github.com/argoproj/argo-workflows/commit/3b00da0741a341d6caf829292886667970bcb0ae) fix: add stepgroup and taskgroup to scope. Fixes #15737 (#15736)
* [4d2a73bf2](https://github.com/argoproj/argo-workflows/commit/4d2a73bf2ca3167502f5dcb08bb32c839c98e396) fix(templates): prevent inline placeholder leak (#15744)
* [c64c7d755](https://github.com/argoproj/argo-workflows/commit/c64c7d75528fde21ef7ef0a0f0660faa0a9600cf) fix(validate): match internal placeholder prefix (#15743)
* [68fe23ee1](https://github.com/argoproj/argo-workflows/commit/68fe23ee12b2b202fb4c73a947d635ed9e324818) chore(deps): pin distroless base to debian13 (#15741)
* [7ca7b1c3c](https://github.com/argoproj/argo-workflows/commit/7ca7b1c3c94f25e0059c23d4b3d085900849dbf5) chore(deps): update module go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp to v1.42.0 (main) (#15708)
* [ae28d19be](https://github.com/argoproj/argo-workflows/commit/ae28d19be786ea952e76cee86222c0376f28e3b0) chore(deps): update module github.com/testcontainers/testcontainers-go/modules/mysql to v0.41.0 (main) (#15698)
* [ba7b77d18](https://github.com/argoproj/argo-workflows/commit/ba7b77d18f430e79a9946510e370165ad27a05be) chore(deps): update google.golang.org/genproto/googleapis/api digest to 84a4fc4 (main) (#15682)
* [cf7003e35](https://github.com/argoproj/argo-workflows/commit/cf7003e3590fb50f7a8b05e45b1ed078fce9f031) chore(deps): update module github.com/testcontainers/testcontainers-go to v0.41.0 (main) (#15697)
* [f2cf209bb](https://github.com/argoproj/argo-workflows/commit/f2cf209bb6b74426b4384301c6094160ee3f9ec3) chore(deps): update module go.opentelemetry.io/contrib/instrumentation/runtime to v0.67.0 (main) (#15703)
* [91daf2dc8](https://github.com/argoproj/argo-workflows/commit/91daf2dc80694255fb89f0aa07005eb5010ae267) chore(deps): update module github.com/testcontainers/testcontainers-go/modules/postgres to v0.41.0 (main) (#15699)
* [349dfad92](https://github.com/argoproj/argo-workflows/commit/349dfad9287f25b8f9299fa5c65cf97520b533da) chore(deps): update module github.com/go-git/go-git/v5 to v5.17.0 (main) (#15695)
* [a1157125d](https://github.com/argoproj/argo-workflows/commit/a1157125d97a21778ec2f5f9cb215950c8f290bc) chore(deps): update module go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp to v1.42.0 (main) (#15706)
* [0d83b7315](https://github.com/argoproj/argo-workflows/commit/0d83b731534116daeeaa0013f0068e23be668465) chore(deps): update docker/dockerfile docker tag to v1.22 (main) (#15693)
* [56d423ef1](https://github.com/argoproj/argo-workflows/commit/56d423ef175ee020b66357545537432aff04a9aa) fix(cron): embed tzdata and validate timezone (#15732)
* [d2747a6f8](https://github.com/argoproj/argo-workflows/commit/d2747a6f8ce6fb6ad2c505e208af6d725613cc02) chore(deps): update module github.com/nao1215/markdown to v0.13.0 (main) (#15696)
* [a7e82ce0f](https://github.com/argoproj/argo-workflows/commit/a7e82ce0f168f9bc7bd02c27191a4a5340c198e3) chore(deps): update module cloud.google.com/go/storage to v1.61.3 (main) (#15694)
* [23a0f4f38](https://github.com/argoproj/argo-workflows/commit/23a0f4f382c80f31b3678d05a9c0863b2e9990d0) chore(deps): update docker/build-push-action action to v7 (main) (#15726)
* [aef6f81d6](https://github.com/argoproj/argo-workflows/commit/aef6f81d600a089c716ec466f00de045773dc867) chore(deps): update actions/download-artifact action to v8 (main) (#15724)
* [c57d12313](https://github.com/argoproj/argo-workflows/commit/c57d12313ce02d9b60c9b188341fc620372c1d95) chore(deps): update actions/setup-node action to v6.3.0 (main) (#15681)
* [57f2d516e](https://github.com/argoproj/argo-workflows/commit/57f2d516ece8e94111faf6145a614ecd49da8c3b) chore(deps): update actions/setup-go action to v6.3.0 (main) (#15680)
* [cacb1580d](https://github.com/argoproj/argo-workflows/commit/cacb1580d2de5151dfa53110942fe44c6cd620cf) chore(deps): update module go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc to v1.42.0 (main) (#15707)
* [513b3f602](https://github.com/argoproj/argo-workflows/commit/513b3f602b5711508a6dd864431041011ce8de39) chore(deps): update sigstore/cosign-installer action to v4.1.0 (main) (#15723)
* [e032df8ea](https://github.com/argoproj/argo-workflows/commit/e032df8ea863c0d56866b6a40b098c1fd0dbd79d) chore(deps): update module go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp to v0.67.0 (main) (#15702)
* [3cd33b8fc](https://github.com/argoproj/argo-workflows/commit/3cd33b8fc882fec606442604d46341b6df8572f5) chore(deps): update module go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc to v1.42.0 (main) (#15705)
* [857cfcdd9](https://github.com/argoproj/argo-workflows/commit/857cfcdd929d888659a12c0dcf97129d529f4a5b) chore(deps): update actions/upload-artifact action to v7 (main) (#15725)
* [89744c8a5](https://github.com/argoproj/argo-workflows/commit/89744c8a5043ef7f5aaa3939ca0599dda0a26dae) chore(deps): update docker/setup-buildx-action action to v4 (main) (#15728)
* [61d03d3f7](https://github.com/argoproj/argo-workflows/commit/61d03d3f74cbc377e1d89ddb262ce053fa6cdca5) chore(deps): update docker/login-action action to v4 (main) (#15727)
* [6184cc9c3](https://github.com/argoproj/argo-workflows/commit/6184cc9c3d24c6bb6d631dc273a16abcc5f28317) chore(deps): update module go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws to v0.67.0 (main) (#15700)
* [a0842b5b3](https://github.com/argoproj/argo-workflows/commit/a0842b5b3a1166fd65d5d25efdc965c5769400d1) chore(deps): update module go.opentelemetry.io/otel/sdk/metric to v1.42.0 (main) (#15712)
* [395f88309](https://github.com/argoproj/argo-workflows/commit/395f8830996ab40b4804a406d9dbcf39dabd58b0) chore(deps): update module go.opentelemetry.io/otel/exporters/prometheus to v0.64.0 (main) (#15709)
* [d0da5e9db](https://github.com/argoproj/argo-workflows/commit/d0da5e9db40fd5bdd0a432e30847b0fd0a107edd) chore(deps): update module golang.org/x/net to v0.52.0 (main) (#15715)
* [d7b591841](https://github.com/argoproj/argo-workflows/commit/d7b59184110caea8da703b9df26c7f2a870972a9) chore(deps): update module golang.org/x/sync to v0.20.0 (main) (#15717)
* [a8938e89f](https://github.com/argoproj/argo-workflows/commit/a8938e89f08f059c918d83d6b688320bfede2287) chore(deps): update module github.com/go-openapi/jsonreference to v0.21.5 (main) (#15688)
* [18a62e8b0](https://github.com/argoproj/argo-workflows/commit/18a62e8b096b8e83dfdda4a69b31efadf4f6ef5e) chore(deps): update tj-actions/changed-files action to v47.0.5 (main) (#15692)
* [2c58750a4](https://github.com/argoproj/argo-workflows/commit/2c58750a4fecacb83097bd931aafc36a888ac5f0) chore(deps): update module github.com/minio/minio-go/v7 to v7.0.99 (main) (#15690)
* [4351a3d42](https://github.com/argoproj/argo-workflows/commit/4351a3d42669ffe4db9abb65ea38575d20d332a2) chore(deps): update module github.com/sirupsen/logrus to v1.9.4 (main) (#15691)
* [74290d21d](https://github.com/argoproj/argo-workflows/commit/74290d21d2dc9fac38c0e44f32b352d2b68c8f96) chore(deps): update module go.opentelemetry.io/otel/trace to v1.42.0 (main) (#15713)
* [6c9104651](https://github.com/argoproj/argo-workflows/commit/6c910465118461c957c0f7734d4420661df0b3ce) chore(deps): update module golang.org/x/oauth2 to v0.36.0 (main) (#15716)
* [874f52507](https://github.com/argoproj/argo-workflows/commit/874f52507dbdc61766ed3abe867246087eea6435) chore(deps): update module golang.org/x/time to v0.15.0 (main) (#15720)
* [228f4ef12](https://github.com/argoproj/argo-workflows/commit/228f4ef12fdb7af747e7abc57c5c8884a794d65e) chore(deps): update module github.com/expr-lang/expr to v1.17.8 (main) (#15687)
* [a62195f5f](https://github.com/argoproj/argo-workflows/commit/a62195f5fea7ede76519f624b9ee1ff0de2169d6) chore(deps): update module github.com/aws/aws-sdk-go-v2/config to v1.32.11 (main) (#15684)
* [c71276607](https://github.com/argoproj/argo-workflows/commit/c71276607bd61413578ac840c5b11f9bd0426fac) chore(deps): update module github.com/aws/aws-sdk-go-v2/service/sts to v1.41.8 (main) (#15686)
* [f405d24e4](https://github.com/argoproj/argo-workflows/commit/f405d24e4c3dd83d904e8558947aad7f974bebe9) chore(deps): update zgosalvez/github-actions-ensure-sha-pinned-actions action to v5.0.1 (main) (#15679)
* [0855e420d](https://github.com/argoproj/argo-workflows/commit/0855e420db93e1f6523f79826899d425d5221033) chore(deps): update module github.com/google/go-containerregistry to v0.21.2 (main) (#15689)
* [089d5dc6e](https://github.com/argoproj/argo-workflows/commit/089d5dc6ed3b535a722e1311d5af319472635036) chore(deps): update module github.com/argoproj/argo-workflows/v4 to v4.0.2 (main) (#15683)
* [ad04978e2](https://github.com/argoproj/argo-workflows/commit/ad04978e22b20ac86d61a548a59c3225d8f546fe) chore(deps): update module golang.org/x/term to v0.41.0 (main) (#15719)
* [11158fd6d](https://github.com/argoproj/argo-workflows/commit/11158fd6db1fcb1bd1b423953c7c6d1734cda0c7) chore(deps): update docker/setup-qemu-action action to v4 (main) (#15729)
* [4964e2286](https://github.com/argoproj/argo-workflows/commit/4964e2286ca58a8254cc2e0a5b2407bb4cb56adc) fix: include spec.arguments in archived and live workflow list responses. Fixes #13946 (#15669)
* [19de7fd76](https://github.com/argoproj/argo-workflows/commit/19de7fd76896cffb255e973c3d0412d4e5fbd375) fix: correct TTL strategy precedence comment in gc-ttl example (#15654)
* [34afaf9c0](https://github.com/argoproj/argo-workflows/commit/34afaf9c0c36f1ba8645d483ea4752cfc4a391e8) Merge commit from fork
* [534f4ff1c](https://github.com/argoproj/argo-workflows/commit/534f4ff1cbd86908e8ff76d97d553ad5a49a950d) Merge commit from fork
* [59f1089b9](https://github.com/argoproj/argo-workflows/commit/59f1089b9875723ddffd524513e6bd5cb37e5e31) chore(deps): bump immutable from 4.3.4 to 4.3.8 in /ui in the deps group across 1 directory (#15662)
* [f12355aa1](https://github.com/argoproj/argo-workflows/commit/f12355aa1a7cce69d99c9b938160cf2323cb53fc) fix(docs): use server-side apply in quick-start guide for v4.0+ compatibility (#15650)
* [ebc0c1774](https://github.com/argoproj/argo-workflows/commit/ebc0c177400330d31a4640f2abed013e525b3a0a) fix: move workflow field extraction inside s.Run closures to prevent nil pointer panic on test re-run (#15647)
* [a77b5c8ec](https://github.com/argoproj/argo-workflows/commit/a77b5c8ec601917114ea4aee10c40f7fc3783015) chore(deps): update module golang.org/x/net to v0.51.0 [security] (main) (#15638)
* [d84169f15](https://github.com/argoproj/argo-workflows/commit/d84169f154879d4c1c2d99e4e0dd65585bee7522) fix: requeue workflow if expected variables are missing. Fixes #15513 (#15442)
* [7e47fc7c4](https://github.com/argoproj/argo-workflows/commit/7e47fc7c486006099b5605b59413904eb6c596d3) fix!: remove 1s sleep and INFORMER_WRITE_BACK from persistUpdates (#15627)
* [b9a545c3e](https://github.com/argoproj/argo-workflows/commit/b9a545c3e9e9b216aad72f2f690d11ef3b79396a) chore(deps): bump github.com/cloudflare/circl from 1.6.1 to 1.6.3 in the go_modules group across 1 directory (#15628)
* [2f60d96fa](https://github.com/argoproj/argo-workflows/commit/2f60d96fab57e524c609d2548554c8e40997c318) fix(lint): enable shadow linter and fix issues (#15615)
* [e5aa55a82](https://github.com/argoproj/argo-workflows/commit/e5aa55a824b49f8d26c1a512f9fc580b573a0985) fix: add metrics shutdown (#15586)
* [578ca54f8](https://github.com/argoproj/argo-workflows/commit/578ca54f8f8c0323f732b11e8a36283dad43ffcc) feat: add tracing span instrumentation to controller and executor (#15585)
* [c811f057f](https://github.com/argoproj/argo-workflows/commit/c811f057f1ef059611294979ba60ba97b1494179) fix: go module versioning (#15622)
* [dd7769e09](https://github.com/argoproj/argo-workflows/commit/dd7769e094216cc2bd06f18d44870963ca7b9d92) fix(sync): variable shadowing causes semaphore holders to be lost on restart (#15609)
* [bf7107b60](https://github.com/argoproj/argo-workflows/commit/bf7107b603b72518fc29c84d6c824796c13f6803) chore(deps): update module github.com/google/go-containerregistry to v0.21.0 (main) (#15610)
* [fcfdd60e2](https://github.com/argoproj/argo-workflows/commit/fcfdd60e2d60b72ecce53ada377e98e058e7b4e8) fix: correct 'taring' typo to 'tarring'. Fixes #15556 (#15608)
* [44b632f1b](https://github.com/argoproj/argo-workflows/commit/44b632f1b73e43644ce7216ed6686a3e5ed8ddd2) chore(deps): update module cloud.google.com/go/storage to v1.60.0 (main) (#15595)
* [3fcf9ab1f](https://github.com/argoproj/argo-workflows/commit/3fcf9ab1f620d5e027448984de74a69e1f6691da) fix(test): skip flaky workflow-template retry-with-steps example (#15607)
* [87d390867](https://github.com/argoproj/argo-workflows/commit/87d39086792a00082e9ddb9891fb2a387d0be431) chore(deps): update tj-actions/changed-files action to v47.0.4 (main) (#15592)
* [978c518c1](https://github.com/argoproj/argo-workflows/commit/978c518c188a6692c6eec7adb30142509c6898ca) chore(deps): update module golang.org/x/net to v0.50.0 (main) (#15598)
* [b07f96b4e](https://github.com/argoproj/argo-workflows/commit/b07f96b4ebce76d52e8b61314a5b538a8ff35d48) fix: add sleep before submit tests to wait for informer cache sync (#15601)
* [f919ec37e](https://github.com/argoproj/argo-workflows/commit/f919ec37e42040f2a793c28818621ebf79be9917) chore(deps): update module golang.org/x/crypto to v0.48.0 (main) (#15597)
* [5db6b9e8c](https://github.com/argoproj/argo-workflows/commit/5db6b9e8c70df0df85bff27fe03678511f38f9b3) chore(deps): update module github.com/lib/pq to v1.11.2 (main) (#15596)
* [bd55c5ae1](https://github.com/argoproj/argo-workflows/commit/bd55c5ae12ad799a72e80398e21cdcbb6ae6c6a6) chore(deps): update module golang.org/x/term to v0.40.0 (main) (#15600)
* [06dd77717](https://github.com/argoproj/argo-workflows/commit/06dd7771722a350c44aa28cef55d3fd722ef6080) fix: use fixed reference time in SQLite store tests to prevent flakiness (#15599)
* [d7fe36fcf](https://github.com/argoproj/argo-workflows/commit/d7fe36fcf7cb2a9c5b6b78018c2b8a84991046f7) chore(deps): update actions/stale action to v10.2.0 (main) (#15593)
* [dcfc231c8](https://github.com/argoproj/argo-workflows/commit/dcfc231c83ae1015aeac8519c6d467ed4aea4826) chore(deps): update docker/build-push-action action to v6.19.2 (main) (#15594)
* [6a0aed9f5](https://github.com/argoproj/argo-workflows/commit/6a0aed9f5323de25ad39eb7949300d8c2fbbf0af) feat!: allow for usage of workflow name in `archive` subcommand. Fixes #15199 (#15198)
* [6de736834](https://github.com/argoproj/argo-workflows/commit/6de736834402a3c8822dd95fe12d434b6eb79a33) chore(deps): update module filippo.io/edwards25519 to v1.1.1 [security] (main) (#15590)
* [7a11f69d1](https://github.com/argoproj/argo-workflows/commit/7a11f69d1d06e78791ecc6be0521cca2408783a4) chore(deps): bump filippo.io/edwards25519 from 1.1.0 to 1.1.1 in /sdks/go/grpc-client in the go_modules group across 1 directory (#15588)
* [8a42b48c9](https://github.com/argoproj/argo-workflows/commit/8a42b48c96f1120dbed9624ab6cf8f554d54484f) refactor: thread DBType through DB session creation instead of detecting from driver (#15576)
* [28d5f1c23](https://github.com/argoproj/argo-workflows/commit/28d5f1c239df1dd84df213563468019c5024a6ce) fix: use workflow key for semaphore nextWorkflow callback (#15558)
* [0f377c5e5](https://github.com/argoproj/argo-workflows/commit/0f377c5e5373e9d866d3194254d56ab9beeab50c) feat: add support for azure postgres/entra. Fixes #15530 (#15511)
* [837a6989f](https://github.com/argoproj/argo-workflows/commit/837a6989fb9f658086dfc33bff062c28e571366e) fix: configure logrus for `argoproj/pkg` internal usage (#15563)
* [82eec0304](https://github.com/argoproj/argo-workflows/commit/82eec0304d60780786280b8ec5a43909d512d0d9) feat: add !=, == support for namespace field selector (#15098)
* [aff93a7bd](https://github.com/argoproj/argo-workflows/commit/aff93a7bd7267db60fe8708faed86950dac32459) feat: add WorkflowTemplate name as label when using `workflowTemplateRef`. Fixes #12670 (#12677)
* [a84a927a4](https://github.com/argoproj/argo-workflows/commit/a84a927a48d9a7cb0780abdccce55a9ecf9c6f9b) fix: make lint to work on macOS machine (#15562)
* [bcfe8d98c](https://github.com/argoproj/argo-workflows/commit/bcfe8d98cbe202a96048ef8f617c8917cd8065ec) chore(deps): update dependency qs to v6.14.2 [security] (main) (#15559)
* [2d95ada90](https://github.com/argoproj/argo-workflows/commit/2d95ada900e410de0c949c7b75bb1e76e44cacae) fix: change RLock to Lock in sync manager methods (#15546)
* [7d04b34e4](https://github.com/argoproj/argo-workflows/commit/7d04b34e473cc6df9b5561a94ba5cd181a70a630) feat: Add proxy support to CLI API client (#12527)
* [2bd409de7](https://github.com/argoproj/argo-workflows/commit/2bd409de7ec7229a93157bb77e176a1327e99f2d) chore(deps): update module github.com/go-git/go-git/v5 to v5.16.5 [security] (main) (#15542)
* [eb7135ab5](https://github.com/argoproj/argo-workflows/commit/eb7135ab5fb1b5fea070f94a50ad48f5c5d18394) chore(deps): update module go.opentelemetry.io/otel/exporters/prometheus to v0.62.0 (main) (#15491)
* [23de65b6f](https://github.com/argoproj/argo-workflows/commit/23de65b6f635fa33095a34883d230d6c29f1b0b2) chore(deps): update module google.golang.org/api to v0.265.0 (main) (#15535)
* [aca208852](https://github.com/argoproj/argo-workflows/commit/aca2088524621bff49217b2f53ccaf58dfe715f3) chore(deps): update zgosalvez/github-actions-ensure-sha-pinned-actions action to v5 (main) (#15538)
* [adb3958ef](https://github.com/argoproj/argo-workflows/commit/adb3958ef6bf1c8bde99ea232ece87c22cd457f9) chore(deps): update module golang.org/x/sys to v0.41.0 (main) (#15534)
* [abaf1fdc3](https://github.com/argoproj/argo-workflows/commit/abaf1fdc3a38018c8d82d51c0b04007d88d1a899) fix: Add optional UID query parameter to GetWorkflow (#15251)
* [a739d3c68](https://github.com/argoproj/argo-workflows/commit/a739d3c68a11d50fbf8a226f01e59a631707675b) fix: support large container args (#15265)
* [a9cedccca](https://github.com/argoproj/argo-workflows/commit/a9cedccca44b0985376e68920ba15dbe21a5aec7) chore(deps): update module go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc to v1.40.0 (main) (#15489)
* [1d582912a](https://github.com/argoproj/argo-workflows/commit/1d582912a701d1bd3ee6199cf077fbfc87cb69f9) chore(deps): update module go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp to v1.40.0 (main) (#15490)
* [4f51e11c4](https://github.com/argoproj/argo-workflows/commit/4f51e11c4fa04702f6a4edbf2f50d674573584b4) chore(deps): update module go.opentelemetry.io/otel/sdk/metric to v1.40.0 (main) (#15494)
* [9f79ae96e](https://github.com/argoproj/argo-workflows/commit/9f79ae96ef244cd4639aad82bd1435293cb35db4) chore(deps): update module go.opentelemetry.io/contrib/instrumentation/runtime to v0.65.0 (main) (#15487)
* [c602ec568](https://github.com/argoproj/argo-workflows/commit/c602ec568d8833eca26148e6b9a0e3f60654993e) fix: preserve existing parameter value for suspend enum inputs (#15510)
* [0b58cb60f](https://github.com/argoproj/argo-workflows/commit/0b58cb60f9e55f00881548161ab0d91714939bcd) chore(deps): update module github.com/argoproj/argo-events to v1.9.10 (main) (#15363)
* [7f8a6558b](https://github.com/argoproj/argo-workflows/commit/7f8a6558bcd096eb69906c09f2cae11164d93270) chore(deps): update google.golang.org/genproto/googleapis/api digest to 546029d (main) (#15485)
* [a563f88fc](https://github.com/argoproj/argo-workflows/commit/a563f88fca03e137f0e81d7e5ecf5aee29df5623) chore(deps): update module go.opentelemetry.io/otel to v1.40.0 (main) (#15488)
* [e27c74108](https://github.com/argoproj/argo-workflows/commit/e27c74108bce3f7b99ba6869373fa7d20e163bb8) chore(deps): update module go.opentelemetry.io/otel/metric to v1.40.0 (main) (#15492)
* [4f954d726](https://github.com/argoproj/argo-workflows/commit/4f954d7262fe104fb8995449495183afdd15b037) chore(deps): update actions/cache action to v5.0.3 (main) (#15486)
* [876f45441](https://github.com/argoproj/argo-workflows/commit/876f45441bf1bf7d7ab017617f772c17f88d0c24) fix: use input.defaults for suspend templates (#15240)
* [1305c8a10](https://github.com/argoproj/argo-workflows/commit/1305c8a10618846efba9a375c52c2de5330a9789) fix: Add instanceID label to WorkflowTaskSet. Fixes #15219 (#15220)
* [1e967a386](https://github.com/argoproj/argo-workflows/commit/1e967a3860f2b47bb8017b12897996e961711283) chore(deps): update module github.com/prometheus/client_golang to v1.23.2 (main) (#15438)
* [7105872ac](https://github.com/argoproj/argo-workflows/commit/7105872ac93b66f916f75b3a351607ba416f4c22) chore(deps): update module github.com/prometheus/common to v0.67.5 (main) (#15439)
* [04d2cf52d](https://github.com/argoproj/argo-workflows/commit/04d2cf52dc8bdae5b617c0483b6ffbe0b44b96b4) chore(deps): update module github.com/testcontainers/testcontainers-go/modules/mysql to v0.40.0 (main) (#15455)
* [b459c632e](https://github.com/argoproj/argo-workflows/commit/b459c632ee8df650e13c65d1c10d73d725c143c3) chore(deps): update module google.golang.org/grpc to v1.78.0 (main) (#15458)
* [333a1fc64](https://github.com/argoproj/argo-workflows/commit/333a1fc64acbee5e21339b7888f3bc7a3be54cc1) chore(deps): update module google.golang.org/api to v0.264.0 (main) (#15457)
* [f58ebb80f](https://github.com/argoproj/argo-workflows/commit/f58ebb80f3aff5be58d7e569e39cf5743eb76fa0) chore(deps): update module github.com/azure/azure-sdk-for-go/sdk/azcore to v1.21.0 (main) (#15454)
* [b519c9054](https://github.com/argoproj/argo-workflows/commit/b519c9054e66b2f0a25eec06709717bd1362f72e) fix: emissary deadline handling (#15352)
* [255f714dc](https://github.com/argoproj/argo-workflows/commit/255f714dc67a1626387c18c776e851b14481bf5c) chore(deps): update module github.com/testcontainers/testcontainers-go/modules/postgres to v0.40.0 (main) (#15456)
* [df349b335](https://github.com/argoproj/argo-workflows/commit/df349b3353eb713862e23a110036c88272a09f90) chore(deps): update module golang.org/x/oauth2 to v0.34.0 (main) (#15449)
* [d232fcdc6](https://github.com/argoproj/argo-workflows/commit/d232fcdc63d291f1939501f5539ec1914fcbe3ff) chore(deps): update module golang.org/x/net to v0.49.0 (main) (#15448)
* [68ca98b13](https://github.com/argoproj/argo-workflows/commit/68ca98b13b8a128a17293c15400d52b14755a1e7) fix: sync example is wrong for mutexes (#15431)
* [e074c692a](https://github.com/argoproj/argo-workflows/commit/e074c692a6e66af250dab113211e0cbd528870cd) chore(deps): update module golang.org/x/crypto to v0.47.0 (main) (#15447)
* [278a8ea29](https://github.com/argoproj/argo-workflows/commit/278a8ea2924c651d6fa1bf4da1ca05c6f076c488) chore(deps): update module cloud.google.com/go/storage to v1.59.2 (main) (#15433)
* [345d4dbe8](https://github.com/argoproj/argo-workflows/commit/345d4dbe8f050e8ae04b712c92f423575fa1a017) chore(deps): update module github.com/gavv/httpexpect/v2 to v2.17.0 (main) (#15437)
* [12bbb26c1](https://github.com/argoproj/argo-workflows/commit/12bbb26c1856ea4949b5c49133fbc025b4f44ff7) chore(deps): update module github.com/alibabacloud-go/tea to v1.4.0 (main) (#15434)
* [7e73f401b](https://github.com/argoproj/argo-workflows/commit/7e73f401b4e79b0f9e715b366fb19919b51591a4) chore(deps): update google.golang.org/genproto/googleapis/api digest to 8636f87 (main) (#15443)
* [c621fe479](https://github.com/argoproj/argo-workflows/commit/c621fe479ff75b6fb39d247357577d97a3773ab3) chore(deps): update docker/login-action action to v3.7.0 (main) (#15432)
* [86c179881](https://github.com/argoproj/argo-workflows/commit/86c1798811d2497c34a5eec5932bdf505eddb0ca) chore(deps): update module github.com/aws/aws-sdk-go-v2/config to v1.32.7 (main) (#15435)
* [1072d331c](https://github.com/argoproj/argo-workflows/commit/1072d331cf6072c0c20fd5f540ea1e55a25cdee7) chore(deps): update module github.com/coreos/go-oidc/v3 to v3.17.0 (main) (#15436)
* [3291e64f7](https://github.com/argoproj/argo-workflows/commit/3291e64f7f8f78e6ed919a82d940d3a9269c90cb) chore(deps): update golang docker tag to v1.25.6 (main) (#15361)
* [06c2b0110](https://github.com/argoproj/argo-workflows/commit/06c2b01103add1070081a0d37d805a650c732729) chore(deps): update actions/download-artifact action to v7 (main) (#15423)
* [3cfd92866](https://github.com/argoproj/argo-workflows/commit/3cfd92866b65eda319d995a9c9f2d3e339b00a53) chore(deps): update tj-actions/changed-files action to v47 (main) (#15418)
* [794b36440](https://github.com/argoproj/argo-workflows/commit/794b36440d8fb5052b43dda2bad6a3c1c7368edb) chore(deps): update actions/checkout action to v6 (main) (#15422)
* [269239469](https://github.com/argoproj/argo-workflows/commit/26923946989611a2e72f6ba58e09a95c1ea41e6e) chore(deps): update actions/upload-artifact action to v6 (main) (#15425)
* [86835e9bc](https://github.com/argoproj/argo-workflows/commit/86835e9bc7b58e5bdf43f871da13a882e432c3bd) chore(deps): update zgosalvez/github-actions-ensure-sha-pinned-actions action to v4 (main) (#15419)
* [a7da1b8c8](https://github.com/argoproj/argo-workflows/commit/a7da1b8c894db1b3093978d109cbe288ce26695a) chore(deps): update peter-evans/create-pull-request action to v8 (main) (#15415)
* [869418715](https://github.com/argoproj/argo-workflows/commit/86941871541481cbe26c811052ac5e6741584ea3) chore(deps): update module go.uber.org/mock to v0.6.0 (main) (#15420)
* [1b74180ba](https://github.com/argoproj/argo-workflows/commit/1b74180ba36f3e4529ca80a04fc95ad904a22314) fix(security): update qs to 6.14.1 (#15427)
* [23779dda4](https://github.com/argoproj/argo-workflows/commit/23779dda4418bf3a24f4f8d25007ae076da5da38) chore(deps): update actions/stale action to v10 (main) (#15424)
* [789c898e3](https://github.com/argoproj/argo-workflows/commit/789c898e38ba78b6484987a69021eea5f5d1170a) chore(deps): update softprops/action-gh-release action to v2 (main) (#15417)
* [5c2411d35](https://github.com/argoproj/argo-workflows/commit/5c2411d35b8902338363471fdf2f327c57fa1ca9) chore(deps): update sigstore/cosign-installer action to v4 (main) (#15416)
* [c5dda69cb](https://github.com/argoproj/argo-workflows/commit/c5dda69cbd2d3ef30dfada668a820ecb988927e6) chore(deps): update amannn/action-semantic-pull-request action to v6 (main) (#15426)
* [0c9fce6af](https://github.com/argoproj/argo-workflows/commit/0c9fce6af8c02e6fd31c59735a646aa9c5dafea9) chore(deps): update actions/cache action to v5 (main) (#15421)
* [ceb9d0aa6](https://github.com/argoproj/argo-workflows/commit/ceb9d0aa666aac3ba8f9434189f392fff387c490) chore(deps): update module golang.org/x/time to v0.14.0 (main) (#15414)
* [6d75e1a02](https://github.com/argoproj/argo-workflows/commit/6d75e1a02b9fdf0b68cd6fb73ab0e94de2bd3db5) chore(deps): update dependency @types/uuid to v9.0.8 (main) (#15353)
* [f2896fb6b](https://github.com/argoproj/argo-workflows/commit/f2896fb6bb20998bc88c6fc7fc1fc65aa51d7d48) chore(deps): update dependency copy-webpack-plugin to v12.0.2 (main) (#15354)
* [e33f40a84](https://github.com/argoproj/argo-workflows/commit/e33f40a8494ea09380f165fa4eb245a34a850edf) chore(deps): update module github.com/aliyun/credentials-go to v1.4.11 (main) (#15362)
* [a0f8e0b5a](https://github.com/argoproj/argo-workflows/commit/a0f8e0b5af13fe4905563b64848b041adada8522) chore(deps): update actions/setup-java action to v5 (main) (#15356)
* [791792df8](https://github.com/argoproj/argo-workflows/commit/791792df834a5c6320f2e1bd7f486600c7a383fa) chore(deps): update actions/setup-node action to v6 (main) (#15357)
* [254b7421d](https://github.com/argoproj/argo-workflows/commit/254b7421da423343f586b2c2603853b144c69ad9) chore(deps): update actions/setup-go action to v6 (main) (#15355)
* [12942bc78](https://github.com/argoproj/argo-workflows/commit/12942bc782a8267ccdae95acab5113efa1fbe75f) chore(deps): update dependabot/fetch-metadata action to v2 (main) (#15360)
* [61c1ce2b6](https://github.com/argoproj/argo-workflows/commit/61c1ce2b67f67c8f9b8fb21f72e2b5ffd218a645) chore(deps): update codecov/codecov-action action to v5 (main) (#15359)
* [c4808ee1e](https://github.com/argoproj/argo-workflows/commit/c4808ee1eaeb181720a997bb3882b073f28b9416) chore(deps): update actions/setup-python action to v6 (main) (#15358)
* [bd39434b6](https://github.com/argoproj/argo-workflows/commit/bd39434b65ae1b3b1dd92c4d39e31d1ae5485b98) chore(deps): update module github.com/azure/azure-sdk-for-go/sdk/storage/azblob to v1.6.4 (main) (#15364)
* [3a994e92f](https://github.com/argoproj/argo-workflows/commit/3a994e92f6e680b1ba5e49a564abe3fd799baeba) chore(deps): update module github.com/spf13/cobra to v1.10.2 (main) (#15365)
* [ab67b034f](https://github.com/argoproj/argo-workflows/commit/ab67b034fcf20d8efe6f98769c4b491a677a354f) chore(deps): update dependency @types/react-dom to v18.3.7 (main) (#15334)
* [de7dd74bb](https://github.com/argoproj/argo-workflows/commit/de7dd74bbe81bafc0d660ceb88c0c741ff175637) chore(deps): update module github.com/nao1215/markdown to v0.10.0 (main) (#15335)
* [0888d1690](https://github.com/argoproj/argo-workflows/commit/0888d1690020102c7908770b70b4a5d216c467ec) chore(deps): update module github.com/testcontainers/testcontainers-go to v0.40.0 (main) (#15337)
* [a8dd65e9e](https://github.com/argoproj/argo-workflows/commit/a8dd65e9e8db3aab0abf4c20ef4c32cc37c94fcd) chore(deps): update google.golang.org/genproto/googleapis/api digest to d11affd (main) (#15347)
* [d42911d29](https://github.com/argoproj/argo-workflows/commit/d42911d29f33aa3436a87b194ccb962b2b6fb73e) chore(deps): update actions/setup-java action to v4.8.0 (main) (#15308)
* [e2b837392](https://github.com/argoproj/argo-workflows/commit/e2b837392229f183a5d361b3b288c121f5c45334) chore(deps): update module github.com/spf13/viper to v1.21.0 (main) (#15344)
* [27a8750f5](https://github.com/argoproj/argo-workflows/commit/27a8750f5b15f9d37791c273defcc924077d1df0) chore(deps): update module go.opentelemetry.io/contrib/instrumentation/runtime to v0.64.0 (main) (#15338)
* [9a3070eb3](https://github.com/argoproj/argo-workflows/commit/9a3070eb3a4db35f7434da002b3902891a0ecabb) chore(deps): update docker/dockerfile docker tag to v1.21 (main) (#15340)
* [914ccd7ad](https://github.com/argoproj/argo-workflows/commit/914ccd7adf92747930e653ccea449ad07c7d4df0) chore(deps): update docker/setup-qemu-action action to v3.7.0 (main) (#15343)
* [d7f32479d](https://github.com/argoproj/argo-workflows/commit/d7f32479df11a3ca5281a06ad54862925db7e9b7) chore(deps): update docker/setup-buildx-action action to v3.12.0 (main) (#15342)
* [421791740](https://github.com/argoproj/argo-workflows/commit/42179174035cc9a836754902b7add5d90d7e259f) chore(deps): update docker/login-action action to v3.6.0 (main) (#15341)
* [b8e5b707c](https://github.com/argoproj/argo-workflows/commit/b8e5b707c8c455c816be04ffcdc6659574b5c656) chore(deps): update module github.com/stretchr/testify to v1.11.1 (main) (#15336)
* [f8fcaee0d](https://github.com/argoproj/argo-workflows/commit/f8fcaee0dde283c52da3057c8b0ef3cce165c7ea) chore(deps): update module github.com/minio/minio-go/v7 to v7.0.98 (main) (#15326)
* [e769ffa19](https://github.com/argoproj/argo-workflows/commit/e769ffa194ddcaad65063bb04d0d5fbbf81bb454) chore(deps): update module github.com/itchyny/gojq to v0.12.18 (main) (#15325)
* [8430e5ea3](https://github.com/argoproj/argo-workflows/commit/8430e5ea32561701797e5b5f6aa2e8d29b9579d9) chore(deps): update actions/checkout action to v4.3.1 (main) (#15311)
* [04fc492ab](https://github.com/argoproj/argo-workflows/commit/04fc492ab51392caef9f70ee88fc9225f63a9df2) chore(deps): update module github.com/google/go-containerregistry to v0.20.7 (main) (#15324)
* [98bf16875](https://github.com/argoproj/argo-workflows/commit/98bf168759a0af90ed92c2b466897ca0cc17af50) chore(deps): update codecov/codecov-action action to v4.6.0 (main) (#15321)
* [20e434035](https://github.com/argoproj/argo-workflows/commit/20e4340350c4f29c2c52e3c39ae5d2614ad8d896) chore(deps): update actions/upload-artifact action to v4.6.2 (main) (#15319)
* [9d1102eb5](https://github.com/argoproj/argo-workflows/commit/9d1102eb5a072238e1d796f6b3519ebc1ff868a3) chore(deps): update actions/setup-go action to v5.6.0 (main) (#15307)
* [620ee89be](https://github.com/argoproj/argo-workflows/commit/620ee89bec1c2ce45143b773b4e793f408ed72f5) chore(deps): update module sigs.k8s.io/yaml to v1.6.0 (main) (#15327)
* [8185e914c](https://github.com/argoproj/argo-workflows/commit/8185e914c5227b5fe669e2b083c622e1ce97f391) chore(deps): update peter-evans/create-pull-request action to v6.1.0 (main) (#15328)
* [97647b8b6](https://github.com/argoproj/argo-workflows/commit/97647b8b65bda5fdb0064b56004d16689c8cb6f3) chore(deps): update module github.com/sethvargo/go-limiter to v1.1.0 (main) (#15330)
* [3f794f967](https://github.com/argoproj/argo-workflows/commit/3f794f967524aa93cdc5f96d7234b9f1625da93c) chore(deps): update sigstore/cosign-installer action to v3.10.1 (main) (#15329)
* [81d142eb7](https://github.com/argoproj/argo-workflows/commit/81d142eb7f9f11fbac09bcebf264e6959c1b929e) chore(deps): update module github.com/go-openapi/jsonreference to v0.21.4 (main) (#15323)
* [78542ccd1](https://github.com/argoproj/argo-workflows/commit/78542ccd1aa3df67262afc505308228e75ed656a) chore(deps): update module github.com/spf13/pflag to v1.0.10 (main) (#15317)
* [90db17988](https://github.com/argoproj/argo-workflows/commit/90db179884a2dbb5c20eedfbd417cf230669e8b3) chore(deps): update module github.com/mattn/go-sqlite3 to v1.14.33 (main) (#15316)
* [1c9b2e6c5](https://github.com/argoproj/argo-workflows/commit/1c9b2e6c5f64fc70e71f14a62a0b441788348cd2) chore(deps): update module github.com/go-git/go-git/v5 to v5.16.4 (main) (#15314)
* [ef27c2546](https://github.com/argoproj/argo-workflows/commit/ef27c254699fa7a470ef5942b5bc74df0782ee14) chore(deps): update module github.com/go-sql-driver/mysql to v1.9.3 (main) (#15315)
* [e3d77a1cb](https://github.com/argoproj/argo-workflows/commit/e3d77a1cb9478f09766aeaeafa63bc14e4d8dae1) chore(deps): update zgosalvez/github-actions-ensure-sha-pinned-actions action to v3.0.25 (main) (#15309)
* [950ef2d20](https://github.com/argoproj/argo-workflows/commit/950ef2d201d65a961efe5636d06af4bc421a0ffe) chore(deps): update github.com/knetic/govaluate digest to 7625b7f (main) (#15289)
* [7d09939c8](https://github.com/argoproj/argo-workflows/commit/7d09939c8970c161890b92ff9702a93df0093a26) chore(deps): update actions/download-artifact action to v4.3.0 (main) (#15313)
* [d3ca11899](https://github.com/argoproj/argo-workflows/commit/d3ca1189947869ad77ea1083fc7e89277fb7cab5) chore(deps): update actions/create-github-app-token action to v2.2.1 (main) (#15312)
* [636e5d952](https://github.com/argoproj/argo-workflows/commit/636e5d95297fadd83076617a56d042f08f8414e9) chore(deps): update actions/cache action to v4.3.0 (main) (#15310)
* [904222734](https://github.com/argoproj/argo-workflows/commit/9042227345fca0b6faea6b8283a44332bf895d01) chore(deps): update actions/stale action to v9.1.0 (main) (#15318)
* [4195c948f](https://github.com/argoproj/argo-workflows/commit/4195c948fe78a2a3b34d129333cd7b54c2cd1060) chore(deps): update actions/setup-python action to v5.6.0 (main) (#15320)
* [75e18d4c1](https://github.com/argoproj/argo-workflows/commit/75e18d4c1d2cbdf1d20c6dfb22907ec57770a8ce) chore(deps): update dependabot/fetch-metadata action to v1.7.0 (main) (#15322)
* [91d8c76e1](https://github.com/argoproj/argo-workflows/commit/91d8c76e1e527ac40a921f1c78625211af36fbe0) chore(deps): update dependency @types/dagre to v0.7.53 (main) (#15305)
* [43d0e21d7](https://github.com/argoproj/argo-workflows/commit/43d0e21d7efb58372e51cfd429a9c11b3b92ae63) chore(deps): update dependency @types/prop-types to v15.7.15 (main) (#15306)
* [3de5f59fb](https://github.com/argoproj/argo-workflows/commit/3de5f59fb10c5b2009f4830106665d1fc045ca47) chore(deps): update docker/build-push-action action to v6 (main) (#15295)
* [a7b99671d](https://github.com/argoproj/argo-workflows/commit/a7b99671d8a84e1fd845797433d95d19ea3b4e52) chore(deps): update k8s.io/utils digest to 914a6e7 (main) (#15298)
* [867ec2520](https://github.com/argoproj/argo-workflows/commit/867ec2520ab3f4d63d8220ee851093dcf4c8611c) chore(deps): update dependency @testing-library/react to v16.3.2 (main) (#15303)
* [12980daad](https://github.com/argoproj/argo-workflows/commit/12980daad1dce7a61600fd8461c6cc6405caa836) chore(deps): update dependency @types/classnames to v2.3.4 (main) (#15304)
* [0aaf5e320](https://github.com/argoproj/argo-workflows/commit/0aaf5e32096fddc442e34af931acfd05763f2c72) chore(deps): update dependency @testing-library/dom to v10.4.1 (main) (#15302)
* [7d2df6de6](https://github.com/argoproj/argo-workflows/commit/7d2df6de67f32597894aa6e586758999285a178f) chore(deps): update actions/checkout action to v5.0.1 (main) (#15301)
* [6b8ea9472](https://github.com/argoproj/argo-workflows/commit/6b8ea9472f70c38c1e59ec3aec85afce50260afa) chore(deps): update google.golang.org/genproto/googleapis/api digest to 8e98ce8 (main) (#15293)
* [978c4e7d6](https://github.com/argoproj/argo-workflows/commit/978c4e7d669c0ef66a974b088eaf8203b07fa088) chore(deps): update sigs.k8s.io/json digest to 2d32026 (main) (#15300)
* [dfd351dd6](https://github.com/argoproj/argo-workflows/commit/dfd351dd67a177d8800545295862a1bd0d3a6e80) fix: optimize index to prevent 'out of sort memory' error Fixes #14240 (#15250)
* [e95af2a16](https://github.com/argoproj/argo-workflows/commit/e95af2a160aa2d9d4246afacebf8272b8d45dd1e) chore(deps): update k8s.io/gengo digest to 5ee0d03 (main) (#15294)
* [d75eb296c](https://github.com/argoproj/argo-workflows/commit/d75eb296c3b6ec68516439f52a8bddb317a036b7) chore(deps): update snyk/actions action to v1 (main) (#15296)
* [b8cf9ed85](https://github.com/argoproj/argo-workflows/commit/b8cf9ed856d9f1a1c6abd4a6efdbdf3a5e179099) fix: prevent int64 overflow in retry backoff calculation (#15277)
* [e16db0d10](https://github.com/argoproj/argo-workflows/commit/e16db0d10991fe049f442aa4a9fe7cac6aad347c) feat: Add rate_limiter latency metrics (#15256)
* [4c94708dd](https://github.com/argoproj/argo-workflows/commit/4c94708dd3783bc86d069530dafc52f37fd63c2f) refactor: convert telemetry generation to use go generate (#15202)
* [2c6ea4872](https://github.com/argoproj/argo-workflows/commit/2c6ea4872ce06fd9b7c8a19c8ad27f7564aa678c) fix: don't lint on protobuf generation (#15072)
* [a6bbd9a0e](https://github.com/argoproj/argo-workflows/commit/a6bbd9a0ee578d0dbce9ac270fc31dae68a618dd) fix(ci): bump `golang.x/tools` to `v0.35.0` to fix codegen (#15273)
* [64be64a56](https://github.com/argoproj/argo-workflows/commit/64be64a56e73ce70675416673f506cfb98089e2b) chore(deps): bump lodash from 4.17.21 to 4.17.23 in /ui in the deps group across 1 directory (#15269)
* [306effa96](https://github.com/argoproj/argo-workflows/commit/306effa96acef0df91bd571579ad87194b05b9f2) chore(deps): bump lodash-es from 4.17.21 to 4.17.23 in /ui in the deps group across 1 directory (#15268)

<details><summary><h3>Contributors</h3></summary>

* Aaron Mark
* Aaron Mell
* Alan Clucas
* Ali Asghar
* amarkdotdev
* AnaySh
* Anay Sharma
* Andre Kurait
* antoinetran
* Anton Gilgur
* Anton Pechenin
* arpechenin
* AsKc2000
* Bartek Kowalczyk
* Claude
* Claude Fable 5
* Claude Opus 4.5
* Claude Opus 4.6
* Claude Opus 4.6 (1M context)
* Claude Opus 4.7 (1M context)
* Claude Opus 4.8
* Claude Opus 4.8 (1M context)
* Claude Sonnet 4.6
* Copilot Autofix powered by AI
* Dennis Lawler
* downfa11
* Eduardo Rodrigues
* Elliot Gunton
* Ferhat Güneri
* fguneri
* Gagan H R
* Gary Hsu
* gaurang_mishra
* Goutham Annem
* GPT 5.5
* heitor
* Heitor Pinto
* himeshp
* Himesh Panchal
* Isitha Subasinghe
* isubasinghe
* Jason Meridth
* John Kelly
* Joibel
* Knut Zuidema
* krisling049
* Liketosweep
* mariadb-MarioEApostolov
* Mason Malone
* Miltiadis Alexis
* Morgan Allen
* nakatani-yo
* Nancy Sangani
* Nebojša Jaćović
* Nikhil J
* Nitish Kumar
* panaxging
* panicboat
* Pierluigi Lenoci
* Pradeep Sagitra
* Rin
* Rohan Sood
* rohansood10
* sakamoto
* SEONGHYUN HONG
* shuangkun tian
* spaced
* Stanley Shen
* Tay
* Tiago Silva
* Tim Collins
* Tommaso TBA. BARBERIS
* Umang Tiwary
* Uziel David Sulkies
* Ville Vesilehto
* workflow-automation
* Yu-Hong Shen
* zvdy

</details>

## v4.0.10 (2026-08-21)

Full Changelog: [v4.0.9...v4.0.10](https://github.com/argoproj/argo-workflows/compare/v4.0.9...v4.0.10)

### Selected Changes

* [2334ea9dd](https://github.com/argoproj/argo-workflows/commit/2334ea9dd8ccb0e5a6bfd1399fb84a114279e7d6) fix: fail, not succeed, workflows terminated while pending on a sync lock (cherry-pick #16776 for 4.0) (#16791)
* [cbf8cfa47](https://github.com/argoproj/argo-workflows/commit/cbf8cfa4736ba2bf6407bc51b29777aad3a847f5) fix: requeue workflow on transient sync lock errors instead of failing (cherry-pick #16745 for 4.0) (#16792)
* [1b1e5d958](https://github.com/argoproj/argo-workflows/commit/1b1e5d95841d6885f276cd69a36dcc5cd93b9817) fix: fail nodes waiting for a sync lock on workflow shutdown (cherry-pick #16777 for 4.0) (#16790)
* [f2ab4b919](https://github.com/argoproj/argo-workflows/commit/f2ab4b9199d251a31e39324ff81d389f9c5a3b6f) fix(controller): refuse reapplyUpdate when workflow UID has changed (cherry-pick #16775 for 4.0) (#16779)
* [c69e5a3bc](https://github.com/argoproj/argo-workflows/commit/c69e5a3bca66edc0052188f6d3fb375065aae123) chore(deps): update module github.com/moby/go-archive to v0.3.0 [security] (release-4.0) (#16754)
* [ec0adcc47](https://github.com/argoproj/argo-workflows/commit/ec0adcc47e5bc8b3ea700d2256e7938b754ce3b9) chore(deps): update module github.com/google/cel-go to v0.30.0 [security] (release-4.0) (#16753)
* [2faeff4a6](https://github.com/argoproj/argo-workflows/commit/2faeff4a605f35218a78ab42d664aa3dacdff440) chore(deps): update module github.com/valyala/fasthttp to v1.70.0 [security] (release-4.0) (#16755)

<details><summary><h3>Contributors</h3></summary>

* Alan Clucas
* Claude Fable 5
* Nitin Moningi

</details>

## v4.0.9 (2026-08-14)

Full Changelog: [v4.0.8...v4.0.9](https://github.com/argoproj/argo-workflows/compare/v4.0.8...v4.0.9)

### Selected Changes

* [bea996636](https://github.com/argoproj/argo-workflows/commit/bea996636e9614155bd7c8b505a0ed992c7c80ff) fix(logging): remove process-wide signal handler from the init logger. Fixes #15863 (cherry-pick #16693 for 4.0) (#16713)
* [e37f9035d](https://github.com/argoproj/argo-workflows/commit/e37f9035d090cc103d23d2626ad1dd8c5aece53a) fix(controller): memoize outputs for steps templates (release-4.0) (#16696)
* [aa35e305c](https://github.com/argoproj/argo-workflows/commit/aa35e305ceb04bfd799ee081a2d43936783a3bbd) chore(deps): update k8s.io/utils digest to cf1189d (release-4.0) (#16657)
* [0e40809a6](https://github.com/argoproj/argo-workflows/commit/0e40809a68e75cb0330e99d1ed9fbb5e67dfe2c1) chore(deps): update google.golang.org/genproto/googleapis/api digest to ec0a776 (release-4.0) (#16656)
* [02be3e32a](https://github.com/argoproj/argo-workflows/commit/02be3e32af67209465b106eefd42e2f2fde6bc8e) chore(deps): update golang to v1.25.12 (release-4.0) (#16650)
* [880f9f86b](https://github.com/argoproj/argo-workflows/commit/880f9f86b60afd725b28ecffd5bbb8ea4cf761b2) fix(sync): remove the acquiring key from the pending queue, not the front. Fixes #16567 (cherry-pick #16613 for 4.0) (#16642)
* [9d385c413](https://github.com/argoproj/argo-workflows/commit/9d385c413e0a672777a78fb0732060116f905d87) fix(test): de-flake TestParallelismUpdate on coarse-resolution clocks (cherry-pick #16224 for 4.0) (#16647)
* [37da91cfc](https://github.com/argoproj/argo-workflows/commit/37da91cfcbd312b58711b45232ffe4512b0baad8) feat: add argo-workflows-crdinstaller image. Fixes #16621 (cherry-pick #16622 for 4.0) (#16634)
* [06bd7b19a](https://github.com/argoproj/argo-workflows/commit/06bd7b19a180d71ade98e262e121a68380719511) fix: avoid nil pointer (cherry-pick #16608 for 4.0) (#16628)
* [a6d15d0dd](https://github.com/argoproj/argo-workflows/commit/a6d15d0ddd48968e1ec83d596d1744ea09b6b7fa) chore(deps): update module github.com/go-git/go-git/v5 to v5.19.2 [security] (release-4.0) (#16617)
* [4d636bab9](https://github.com/argoproj/argo-workflows/commit/4d636bab900dc2e3bdf864c567aad4a94a5e25c6) fix(controller): archive each workflow once, and retry failed archives. Fixes #16575 (cherry-pick #16577 for 4.0) (#16591)
* [57107c6e3](https://github.com/argoproj/argo-workflows/commit/57107c6e3f3edec3ded8c9fa111d8f43c6d9f626) fix(controller): do not postpone already-Running workflows. Fixes #14123 (cherry-pick #16569 for 4.0) (#16572)
* [3d58f8ca1](https://github.com/argoproj/argo-workflows/commit/3d58f8ca1c2235e1f1145346a19a5f2cb741abd2) chore(deps): update module github.com/klauspost/compress to v1.18.7 [security] (release-4.0) (#16581)
* [4eeafbd63](https://github.com/argoproj/argo-workflows/commit/4eeafbd63f947bdc54802477d7e643907e489ec3) chore(deps): update module github.com/google/cel-go to v0.29.0 [security] (release-4.0) (#16542)
* [df6f1e380](https://github.com/argoproj/argo-workflows/commit/df6f1e380fe8537326554be11dfc20fa51b9a2af) chore(deps): update module go.opentelemetry.io/otel to v1.44.0 [security] (release-4.0) (#16535)
* [f334b4ab3](https://github.com/argoproj/argo-workflows/commit/f334b4ab32b58bf498d819e116fc36428f6cf1ee) chore(deps): update dependency pymdown-extensions to v11 [security] (release-4.0) (#16536)

<details><summary><h3>Contributors</h3></summary>

* Alan Clucas
* Arthur Kepler
* Claude Fable 5
* Claude Opus 4.8 (1M context)
* Claude Opus 5 (1M context)
* heewonham
* spaced
* 秀吉

</details>

## v4.0.8 (2026-07-22)

Full Changelog: [v4.0.7...v4.0.8](https://github.com/argoproj/argo-workflows/compare/v4.0.7...v4.0.8)

### Selected Changes

* [570c47058](https://github.com/argoproj/argo-workflows/commit/570c470582fc7fe41b1963d8703111679cc3d25a) chore(deps): update module golang.org/x/text to v0.39.0 [security] (release-4.0) (#16523)
* [48f1537f9](https://github.com/argoproj/argo-workflows/commit/48f1537f9b358b3c8f66ace8d6f7ee36632d0e74) chore(deps): update module google.golang.org/grpc to v1.82.1 [security] (release-4.0) (#16520)
* [cfa106cea](https://github.com/argoproj/argo-workflows/commit/cfa106ceae72b6e3eefdab60c7adbcef72018441) chore(deps): update module golang.org/x/net to v0.56.0 [security] (release-4.0) (#16522)
* [94d3e4768](https://github.com/argoproj/argo-workflows/commit/94d3e4768b767b6042f09f0b33339438bca29ec3) fix: replace configmap watchers with informers (cherry-pick #16408 for 4.0) (#16516)
* [86e0b7bbf](https://github.com/argoproj/argo-workflows/commit/86e0b7bbf7b42295dfbde4ff3f187467a97f66a2) chore(deps): update module golang.org/x/net to v0.56.0 [security] (release-4.0) (#16518)
* [d356c8b8b](https://github.com/argoproj/argo-workflows/commit/d356c8b8bf398d5058f70583a4b9b7aa81d773ca) fix: normalize GCS artifact keys to forward slashes on Windows. Fixes #16470 (cherry-pick #16476 for 4.0) (#16507)
* [a329a178b](https://github.com/argoproj/argo-workflows/commit/a329a178b6aa00409679a36cd66ae6793a2854d1) fix(controller): mark reapply-failed on persist errors to keep throttler slot (cherry-pick #16482 for 4.0) (#16505)
* [c90df9a82](https://github.com/argoproj/argo-workflows/commit/c90df9a82337d75d711b51f48d7d12108a838c76) fix(errors): treat gRPC client request timeout as transient (cherry-pick #16487 for 4.0) (#16504)
* [7d8802649](https://github.com/argoproj/argo-workflows/commit/7d8802649639cc2270b2eda559a84a6ad4081e02) feat(controller): hot-reload namespaceParallelism from config (cherry-pick #16486 for 4.0) (#16501)
* [6cd56ceae](https://github.com/argoproj/argo-workflows/commit/6cd56ceaeac558a26f04cd619223fb5c5198c810) fix(controller): treat client-go rate limiter wait deadline as transient (cherry-pick #16485 for 4.0) (#16500)
* [9c6aa812c](https://github.com/argoproj/argo-workflows/commit/9c6aa812cc2f680e630d9f8b86cbf25779a9c168) fix(controller): allow onExit DAG handler to complete under Stop shutdown (cherry-pick #16488 for 4.0) (#16496)
* [751349b8f](https://github.com/argoproj/argo-workflows/commit/751349b8f4b2837ffd7e614e2241a5e5f79c7b3a) fix(errors): treat client-go response body read failures as transient (cherry-pick #16484 for 4.0) (#16498)
* [f0b8b5649](https://github.com/argoproj/argo-workflows/commit/f0b8b56491b1e6db6e3f598217e555f953e6bc10) fix(ui): handle exceptions when retrieving user info (cherry-pick #16491 for 4.0) (#16493)
* [11895e122](https://github.com/argoproj/argo-workflows/commit/11895e122b3d92aec8fe1ed1e99363bc6424b66e) fix: don't leak semaphore slots when limit fetch fails during release (cherry-pick #16405 for 4.0) (#16471)
* [fef0aee08](https://github.com/argoproj/argo-workflows/commit/fef0aee08a2d35637d7106857ba43eea90f1ac58) fix: make workflow retry reset deterministic. Fixes #16450 (cherry-pick #16451 for 4.0) (#16467)
* [30e3da0ac](https://github.com/argoproj/argo-workflows/commit/30e3da0ac2028a06340541e2cdda8d05bf7ce783) fix: complete orphaned TaskGroup nodes stuck Running. Fixes #16450 (cherry-pick #16454 for 4.0) (#16465)
* [445b84e6f](https://github.com/argoproj/argo-workflows/commit/445b84e6f99c5a3724bc8a0d3e5f7b7a220fffd3) chore(deps): update gcr.io/distroless/static-debian13:latest docker digest to 9197324 (release-4.0) (#16460)
* [61e0bdc09](https://github.com/argoproj/argo-workflows/commit/61e0bdc093aa112132d57a98cefab7dea6419a25) chore(deps): update dependency rxjs to v7.8.2 (release-4.0) (#16462)
* [fe2a3b88d](https://github.com/argoproj/argo-workflows/commit/fe2a3b88df060c50e6fc1ac87df284d237586b5d) chore(deps): update dependency qs to v6.15.3 (release-4.0) (#16461)
* [fefe0c678](https://github.com/argoproj/argo-workflows/commit/fefe0c6784ccd67284a2a939e9f944e510b5efcc) fix: log missing optional output parameter at warn level, not error. Fixes #16395 (cherry-pick #16402 for 4.0) (#16455)
* [6119ae208](https://github.com/argoproj/argo-workflows/commit/6119ae2085ccf0d858d643b7b99d361b27be6e92) chore(deps): bump loadash+loadash-es for snyk (release-4.0) (#16452)
* [afc4b74b6](https://github.com/argoproj/argo-workflows/commit/afc4b74b6ef886f84ad47f8c3ade8b32bac0beed) chore(deps): update dependency @types/dagre to v0.7.54 (release-4.0) (#16449)
* [e19d0138c](https://github.com/argoproj/argo-workflows/commit/e19d0138c1bfa69320152ce7920b61e446605cbf) chore(deps): update dependency linkify-it to v5.0.2 (release-4.0) (#16446)
* [52dba927d](https://github.com/argoproj/argo-workflows/commit/52dba927d957eff27ede9bc58d3936f88379e55b) fix(ui): resolve login logo paths with non-root baseHref (cherry-pick #16385 for 4.0) (#16412)
* [12e83f192](https://github.com/argoproj/argo-workflows/commit/12e83f192ba3260fe44c625b1e999a4b190c84a7) feat: configurable allowlist (cherry-pick #16344 for 4.0) (#16401)

<details><summary><h3>Contributors</h3></summary>

* Aaron Mark
* Alan Clucas
* Ali Asghar
* amarkdotdev
* Claude Fable 5
* Goutham Annem
* Isitha Subasinghe
* krisling049
* Mason Malone
* shuangkun tian

</details>

## v4.0.7 (2026-07-07)

Full Changelog: [v4.0.6...v4.0.7](https://github.com/argoproj/argo-workflows/compare/v4.0.6...v4.0.7)

### Selected Changes

* [9aeb47ce1](https://github.com/argoproj/argo-workflows/commit/9aeb47ce10339f4a14819335c6a00027353ba0df) fix(ui): Fixed Azure Queue Storage icon in event flow diagram Fixes #16384 (cherry-pick #16390 for 4.0) (#16392)
* [bb4e7ff00](https://github.com/argoproj/argo-workflows/commit/bb4e7ff00c92777ab359a567f98200e6e8a74b83) fix: reject stale copies of completed workflows using resourceVersion comparison (cherry-pick #16357 for 4.0) (#16382)
* [0f415c4dc](https://github.com/argoproj/argo-workflows/commit/0f415c4dc2437735ead88f7113b285240e91e94f) fix: log "Max parallelism reached" at info level, not error. Fixes #16378 (cherry-pick #16379 for 4.0) (#16380)
* [6cb6553c9](https://github.com/argoproj/argo-workflows/commit/6cb6553c9be5ec98cd32ea0062364306dcc979b5) fix(validate): validate placeholder step names (cherry-pick #15991 for 4.0) (#16318)
* [5f856f117](https://github.com/argoproj/argo-workflows/commit/5f856f1179cb96e2bcca93c105bfeff307e45eed) fix: honor ?? and ?. guards in strict missing-variable check (cherry-pick #16274 for 4.0) (#16316)
* [638a02fae](https://github.com/argoproj/argo-workflows/commit/638a02faee642dbda3982f85deafc549450b5821) fix!: drop values when skipped arguments are being substituted (cherry-pick #16223 for 4.0) (#16314)
* [1a36c5de9](https://github.com/argoproj/argo-workflows/commit/1a36c5de93f6fc2a79fafef424962a4d92af3f1a) fix: resolve race condition in custom metric initialization (cherry-pick #16238 for 4.0) (#16312)
* [f2b671c85](https://github.com/argoproj/argo-workflows/commit/f2b671c858e1e0d2ebeb83478cbd0c5eaddd930c) fix: return 404 instead of panic when archived workflow is not yet persisted (cherry-pick #16302 for 4.0) (#16307)
* [76fb8628c](https://github.com/argoproj/argo-workflows/commit/76fb8628ce64057b67be95cd1195e41ac392c22c) fix(auth): mask sensitive token in sso callback logs (cherry-pick #16268 for 4.0) (#16309)
* [109a2fe03](https://github.com/argoproj/argo-workflows/commit/109a2fe03b0cc75030b75dea1db30deb8e7651f3) chore(deps): update dependency linkify-it to v5.0.1 [security] (release-4.0) (#16301)
* [33baac709](https://github.com/argoproj/argo-workflows/commit/33baac709eb8388b86a0adcd49bef0be64d2cf73) chore(deps): update dependency @babel/core to v7.29.6 [security] (release-4.0) (#16280)

<details><summary><h3>Contributors</h3></summary>

* Alan Clucas
* Claude Fable 5
* Claude Opus 4.8 (1M context)
* Isitha Subasinghe
* Knut Zuidema
* Liketosweep
* Nebojša Jaćović
* panaxging
* Rin
* Tommaso TBA. BARBERIS
* Ville Vesilehto
* zvdy

</details>

## v4.0.6 (2026-06-10)

Full Changelog: [v4.0.5...v4.0.6](https://github.com/argoproj/argo-workflows/compare/v4.0.5...v4.0.6)

### Selected Changes

* [277e9cef0](https://github.com/argoproj/argo-workflows/commit/277e9cef0ad16d7eaaab253573d0695951a65dbd) Merge commit from fork
* [5ac265d6a](https://github.com/argoproj/argo-workflows/commit/5ac265d6aef7418e57c64d66168ea18b6dd8f609) fix: WorkflowTaskSets size bloat for large workflows (cherry-pick #16075 for 4.0) (#16253)
* [03fbf9f1d](https://github.com/argoproj/argo-workflows/commit/03fbf9f1db191297d60dcb99f121e62ad7013169) fix: address semaphore/mutex unsoundness for Initalize (cherry-pick #16160 for 4.0) (#16252)
* [b604bfc95](https://github.com/argoproj/argo-workflows/commit/b604bfc95bb0500b4bb90af6a5d9b2f6925c97f9) fix: change log level because behavior is expected (cherry-pick #16124 for 4.0) (#16251)
* [7466a0831](https://github.com/argoproj/argo-workflows/commit/7466a083102316db0312d590b64a611433c71cd7) fix: retry for database transaction errors. Fixes #16101 (cherry-pick #16102 for 4.0) (#16249)
* [c7a1816c4](https://github.com/argoproj/argo-workflows/commit/c7a1816c4e23ecba647a72c02afc49fc49504297) fix(crds): escape template variables in CRD descriptions to prevent Helm rendering errors (cherry-pick #16036 for 4.0) (#16243)
* [b5d864662](https://github.com/argoproj/argo-workflows/commit/b5d864662cf4cbfd10d9da5b41eb5a62170f3cfc) fix: allow cron aliases in schedule validation (cherry-pick #16100 for 4.0) (#16245)
* [51980e5e6](https://github.com/argoproj/argo-workflows/commit/51980e5e6bffca4c80e54897e75cce50486f04f0) fix: do not re-run `onExitNode`. Fixes #14392 (cherry-pick #16088 for 4.0) (#16247)
* [c365da494](https://github.com/argoproj/argo-workflows/commit/c365da49401d1276c65cd97c57bc26cc953645ef) fix(ui): fix mixed bold/notbold markdown in title annotations (cherry-pick #16064 for 4.0) (#16230)
* [2f884facc](https://github.com/argoproj/argo-workflows/commit/2f884faccdd87792a7086b426d1c4524abfdf7c9) chore(deps): update module golang.org/x/crypto to v0.52.0 [security] (release-4.0) (#16130)
* [e982bee95](https://github.com/argoproj/argo-workflows/commit/e982bee951aa866de93fa1082ff50497a571ceea) chore(deps): update module golang.org/x/net to v0.55.0 [security] (release-4.0) (#16067)
* [a90250376](https://github.com/argoproj/argo-workflows/commit/a902503763f254e304f590a4d582d2b23b51fcbd) fix(ui): toggle filter dropdown closed when clicking anchor (cherry-pick #16014 for 4.0) (#16148)
* [b29342673](https://github.com/argoproj/argo-workflows/commit/b29342673fd3cadf903a7d2da4974c8bc77b8cc3) fix: classify bare 5xx S3 responses as transient. Fixes #15565 (cherry-pick #16016 for 4.0) (#16033)
* [e52434e6a](https://github.com/argoproj/argo-workflows/commit/e52434e6a39b661233fe6319c2f6cde14ed24cd6) chore(deps): update module github.com/go-git/go-git/v5 to v5.19.1 [security] (release-4.0) (#16118)
* [d3e736b85](https://github.com/argoproj/argo-workflows/commit/d3e736b858d443202eea5d0dbc2866b6ff159981) chore(deps): update distroless base image (release-4.0) (#16217)
* [f4381b6e6](https://github.com/argoproj/argo-workflows/commit/f4381b6e66030d0a66518b912c5722c513dc385b) chore(deps): update dependency qs to v6.15.2 [security] (release-4.0) (#16135)
* [11eb27276](https://github.com/argoproj/argo-workflows/commit/11eb272765c45e06d63e6b46d80667860950bba4) fix: metadata merge. Fixes #15870 (cherry-pick #16103 for 4.0) (#16147)
* [7b27535de](https://github.com/argoproj/argo-workflows/commit/7b27535de253e35c9a70cbff633ce9689ed028d1) chore(deps): update module golang.org/x/sys to v0.44.0 [security] (release-4.0) (#16138)
* [3b72a8c13](https://github.com/argoproj/argo-workflows/commit/3b72a8c1313661ab1f7762be9759619b72e1bd24) fix: ignore resource version (match) when continue token present (cherry-pick #16099 for 4.0) (#16142)
* [1267ad8ad](https://github.com/argoproj/argo-workflows/commit/1267ad8ad633cabf605a15c6ecdc7f7d72ab450e) chore(deps): update module github.com/go-git/go-git/v5 to v5.19.0 [security] (release-4.0) (#16084)
* [6d300d4ff](https://github.com/argoproj/argo-workflows/commit/6d300d4ff64e6f110952cbfb6913537df535ab4f) fix(ui): populate URL filter parameters on first load. Fixes #15794 (cherry-pick #15796 for 4.0) (#16080)

<details><summary><h3>Contributors</h3></summary>

* Alan Clucas
* Ali Asghar
* Anton Pechenin
* Claude Opus 4.8 (1M context)
* Elliot Gunton
* Ferhat Güneri
* Isitha Subasinghe
* isubasinghe
* John Kelly
* nakatani-yo
* Pradeep Sagitra
* spaced
* Umang Tiwary

</details>

## v4.0.5 (2026-04-23)

Full Changelog: [v4.0.4...v4.0.5](https://github.com/argoproj/argo-workflows/compare/v4.0.4...v4.0.5)

### Selected Changes

* [0ab145214](https://github.com/argoproj/argo-workflows/commit/0ab1452144d8f4d57c50b37ce50dad218868e950) chore(deps): update module github.com/jackc/pgx/v5 to v5.9.2 [security] (release-4.0) (#16027)
* [0954bb218](https://github.com/argoproj/argo-workflows/commit/0954bb2182cfcab605b0fd87639217a5f1de8074) fix(ui): respect target field in workflow-list scope links (cherry-pick #16021 for 4.0) (#16022)
* [2727f3f70](https://github.com/argoproj/argo-workflows/commit/2727f3f701677d467dfb5e053c57237cbc752c3c) Merge commit from fork
* [7abb4de6c](https://github.com/argoproj/argo-workflows/commit/7abb4de6c3599e2d5d960ba4d5de4cf1df109965) Merge commit from fork
* [09fff05e0](https://github.com/argoproj/argo-workflows/commit/09fff05e0830c14a5e36cc40597ad84881db1ab6) Merge commit from fork
* [c4cc17d0c](https://github.com/argoproj/argo-workflows/commit/c4cc17d0c034fa9a9cc01ef1af6c8016c93071d4) Merge commit from fork
* [4fe54e529](https://github.com/argoproj/argo-workflows/commit/4fe54e529eff5519233287251e5adf9a61b9fc67) Merge commit from fork
* [bdd409085](https://github.com/argoproj/argo-workflows/commit/bdd40908580f727c590c8743836e338b04fe4a87) Merge commit from fork
* [91697ce35](https://github.com/argoproj/argo-workflows/commit/91697ce3596e143adc706dcffbb72fbabf0e0f5f) fix: delete stale TaskGroup children on retry with parameter override. Fixes #15802 (cherry-pick #15827 for 4.0) (#16010)
* [16f4914ce](https://github.com/argoproj/argo-workflows/commit/16f4914cede0aeddd8b198b3cccb7bbc6dce6e40) fix: prevent `failed to get a template` when using inline template. Fixes #15051 (cherry-pick #15574 for 4.0) (#16007)
* [245cb9b74](https://github.com/argoproj/argo-workflows/commit/245cb9b741d80637b7d550a028128076bf1babb8) fix(controller): guard realtime workflow.duration against zero StartedAt (cherry-pick #15935 for 4.0) (#16005)
* [adb055138](https://github.com/argoproj/argo-workflows/commit/adb055138c734b824bb52b1ce2c5d0cca2aa5f29) fix: 401s when accessing artifact directories with SSO enabled. Fixes #15800 (cherry-pick #15994 for 4.0) (#15998)
* [b65e27e8f](https://github.com/argoproj/argo-workflows/commit/b65e27e8fdeca8f39d9f854598b44244679fe636) chore(deps): update module github.com/go-git/go-git/v5 to v5.18.0 [security] (release-4.0) (#15990)
* [7a5ecf7b5](https://github.com/argoproj/argo-workflows/commit/7a5ecf7b5085e187622acd66b23e227ea93da577) chore(deps): update module github.com/moby/spdystream to v0.5.1 [security] (release-4.0) (#15957)
* [d54c13ff0](https://github.com/argoproj/argo-workflows/commit/d54c13ff07482a7142effb3bb621dc382216b1b3) chore(deps): update k8s.io/gengo digest to 25e2208 (release-4.0) (#15980)
* [bfb330f82](https://github.com/argoproj/argo-workflows/commit/bfb330f82eee29aa0055b829b7c712cfcf783192) chore(deps): update k8s.io/utils digest to 28399d8 (release-4.0) (#15981)
* [7761ec8e2](https://github.com/argoproj/argo-workflows/commit/7761ec8e22ddcf98e9c879e6332e4d26c0b14e7d) chore(deps): update module github.com/jackc/pgx/v5 to v5.9.0 [security] (release-4.0) (#15947)
* [63ae70510](https://github.com/argoproj/argo-workflows/commit/63ae70510501d75470f53e13b77681a157d93a63) chore(deps): update module go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp to v1.43.0 [security] (release-4.0) (#15891)
* [97d616cca](https://github.com/argoproj/argo-workflows/commit/97d616cca5a98a6dfb0809067a1df46ed24dcc66) chore(deps): update minio-go to include non-DualStack region fix (#2205) (cherry-pick #15838 for 4.0) (#15928)
* [66a384d28](https://github.com/argoproj/argo-workflows/commit/66a384d28982ed01bbf4f4c3b47800e4391a1e86) chore(deps): update module github.com/go-jose/go-jose/v3 to v3.0.5 [security] (release-4.0) (#15858)
* [8a8c7325e](https://github.com/argoproj/argo-workflows/commit/8a8c7325e6e01c744769b0a93c04632a18f2257e) chore(deps): update module go.opentelemetry.io/otel/sdk to v1.43.0 [security] (release-4.0) (#15902)
* [288da9183](https://github.com/argoproj/argo-workflows/commit/288da91832f4d36040ffec1f464edc5b6d1e6cde) fix: changed log level (cherry-pick #15898 for 4.0) (#15899)
* [65d2b618a](https://github.com/argoproj/argo-workflows/commit/65d2b618a19fac53ca96c4201181df666f23e464) chore(deps): update module github.com/go-jose/go-jose/v4 to v4.1.4 [security] (release-4.0) (#15883)
* [aa8ee507a](https://github.com/argoproj/argo-workflows/commit/aa8ee507a90e4870dd9ac91c1dddd3176936620c) chore(deps): update module go.opentelemetry.io/otel/sdk to v1.43.0 [security] (release-4.0) (#15892)

<details><summary><h3>Contributors</h3></summary>

* Alan Clucas
* AsKc2000
* Claude Opus 4.6
* Claude Opus 4.7 (1M context)
* Isitha Subasinghe
* Mason Malone
* panicboat
* Ville Vesilehto
* Yu-Hong Shen

</details>

## v4.0.4 (2026-04-02)

Full Changelog: [v4.0.3...v4.0.4](https://github.com/argoproj/argo-workflows/compare/v4.0.3...v4.0.4)

### Selected Changes

* [fe0af1198](https://github.com/argoproj/argo-workflows/commit/fe0af119897a54f4c7db117a5912a5559c46532f) fix: tolerate expression template runtime failures when allowUnresolved is true. Fixes #15832, #15824 (cherry-pick #15839 for 4.0) (#15850)
* [107542b44](https://github.com/argoproj/argo-workflows/commit/107542b4424240db6241555d253a8a6bf2619bda) fix: populate scope with empty values for outputs of skipped/omitted DAG ancestors (cherry-pick #15841 for 4.0) (#15849)
* [0ba0565d7](https://github.com/argoproj/argo-workflows/commit/0ba0565d720785a57d3c92886fe9bc78ea09430b) fix(ui): add tooltips to tab icons (cherry-pick #15840 for 4.0) (#15846)
* [73eb45dba](https://github.com/argoproj/argo-workflows/commit/73eb45dba03cd89bfedfd5f88269062ec2d63417) chore(deps): update module github.com/go-git/go-git/v5 to v5.17.1 [security] (release-4.0) (#15831)
* [92ee3e221](https://github.com/argoproj/argo-workflows/commit/92ee3e2213f1919d5657d11e461cb8081ed9d8de) chore(deps): update module github.com/docker/cli to v29 [security] (release-4.0) (#15813)
* [1e2059a2a](https://github.com/argoproj/argo-workflows/commit/1e2059a2a7aa1d997661992314ebd54bf7a9de94) fix: remove holder from waiting list when semaphore lock is acquired. (cherry-pick #15239 for 4.0) (#15823)
* [f9004cddf](https://github.com/argoproj/argo-workflows/commit/f9004cddf3c4e96212f83788c883196e46f5207e) chore(deps): update dependency yaml to v2.8.3 [security] (release-4.0) (#15810)
* [251beac5e](https://github.com/argoproj/argo-workflows/commit/251beac5e8d9ce195d04bf18696a26a694023510) chore(deps): update module google.golang.org/grpc to v1.79.3 [security] (release-4.0) (#15775)
* [077c3186c](https://github.com/argoproj/argo-workflows/commit/077c3186c5f4fd72a66c7dc11c447d722003eaf7) chore(deps): update module google.golang.org/grpc to v1.79.3 [security] (release-4.0) (#15771)

<details><summary><h3>Contributors</h3></summary>

* Alan Clucas
* Claude Opus 4.6
* nakatani-yo
* shuangkun tian

</details>

## v4.0.3 (2026-03-18)

Full Changelog: [v4.0.2...v4.0.3](https://github.com/argoproj/argo-workflows/compare/v4.0.2...v4.0.3)

### Selected Changes

* [8cae17b6f](https://github.com/argoproj/argo-workflows/commit/8cae17b6fb49f077eeb269fcf21ce021b37a97f1) chore(deps): go 1.24.10->1.25.7 (release-4.0) (#15759)
* [c459b9817](https://github.com/argoproj/argo-workflows/commit/c459b9817285b12e4356f58cf929e7c3f01a545c) fix: add stepgroup and taskgroup to scope. Fixes #15737 (#15736). (cherry-pick #15736 for 4.0) (#15757)
* [341f3ae5d](https://github.com/argoproj/argo-workflows/commit/341f3ae5d1f99bb119c9b33d77425e8ede41702b) fix(cron): embed tzdata and validate timezone (cherry-pick #15732 for 4.0) (#15739)
* [d0919730c](https://github.com/argoproj/argo-workflows/commit/d0919730c286e5c3ce1f897ed60950f8ccaaae7b) chore(deps): pin distroless base to debian13 (cherry-pick #15741 for 4.0) (#15751)
* [658b43198](https://github.com/argoproj/argo-workflows/commit/658b4319856d6204cd2190691ca5b05d6c52be80) chore(deps): update k8s.io/utils digest to b8788ab (release-4.0) (#15731)
* [e8df68c81](https://github.com/argoproj/argo-workflows/commit/e8df68c819ebc8ee8674e28c0eb6b346214f1b87) fix: optimize index to prevent 'out of sort memory' error Fixes #14240 (cherry-pick #15250 for 4.0) (#15675)
* [4f58ea73b](https://github.com/argoproj/argo-workflows/commit/4f58ea73b926129c1a432194cb4dd83f05203722) fix: include spec.arguments in archived and live workflow list responses. Fixes #13946 (cherry-pick #15669 for 4.0) (#15678)
* [5df42c4f0](https://github.com/argoproj/argo-workflows/commit/5df42c4f06cc0e8170c60a5821e8c25c01000bac) fix: correct TTL strategy precedence comment in gc-ttl example (cherry-pick #15654 for 4.0) (#15672)

<details><summary><h3>Contributors</h3></summary>

* Alan Clucas
* Claude Sonnet 4.6
* downfa11
* heitor
* Heitor Pinto
* himeshp
* Himesh Panchal
* Isitha Subasinghe
* Ville Vesilehto

</details>

## v4.0.2 (2026-03-11)

Full Changelog: [v4.0.1...v4.0.2](https://github.com/argoproj/argo-workflows/compare/v4.0.1...v4.0.2)

### Selected Changes

* [32afec3c4](https://github.com/argoproj/argo-workflows/commit/32afec3c401c0920517bf1e890fa7dba6170cdf2) Merge commit from fork
* [4cac12c75](https://github.com/argoproj/argo-workflows/commit/4cac12c75de720889ad2cae8a6cc63c566b1d8d8) Merge commit from fork
* [9f32c3843](https://github.com/argoproj/argo-workflows/commit/9f32c384364c35094c6d8862c1927c17c2304ace) fix: requeue workflow if expected variables are missing. Fixes #15513 (cherry-pick #15442 for 4.0) (#15656)
* [c0b7f2f87](https://github.com/argoproj/argo-workflows/commit/c0b7f2f87b6b32e859482588b3dab5a27fa4ed52) fix(docs): use server-side apply in quick-start guide for v4.0+ compatibility (cherry-pick #15650 for 4.0) (#15651)
* [5adb9bd30](https://github.com/argoproj/argo-workflows/commit/5adb9bd3011051e7f032a0cb9d0f6242f7ff9edd) chore(deps): update module github.com/cloudflare/circl to v1.6.3 [security] (release-4.0) (#15635)
* [069bc11ed](https://github.com/argoproj/argo-workflows/commit/069bc11ed14ae105015e700b567ab4ebdc3eed9e) fix: update go paths to v4 (cherry-pick #15622 for 4.0) (#15626)
* [35f6c11f7](https://github.com/argoproj/argo-workflows/commit/35f6c11f7dd93d4216c0c9e1e61f0a90bbe9e061) fix: use fixed reference time in SQLite store tests to prevent flakiness (cherry-pick #15599 for 4.0) (#15618)
* [35cd9236f](https://github.com/argoproj/argo-workflows/commit/35cd9236f3352c97cbd858dcbbc47d22a9c85d10) fix: use workflow key for semaphore nextWorkflow callback (cherry-pick #15558 for 4.0) (#15620)
* [c85c2efee](https://github.com/argoproj/argo-workflows/commit/c85c2efeefa34e10df272a6476e1ff80de3f5180) fix(sync): variable shadowing causes semaphore holders to be lost on restart (cherry-pick #15609 for 4.0) (#15612)
* [a917fd688](https://github.com/argoproj/argo-workflows/commit/a917fd68876b75a05fe68a77831d27142fa69fbe) chore(deps): update lycheeverse/lychee-action digest to 8646ba3 (release-4.0) (#15611)
* [bf91ab96b](https://github.com/argoproj/argo-workflows/commit/bf91ab96b88388c3a8067f802711e666142bd594) chore(deps): update module filippo.io/edwards25519 to v1.1.1 [security] (release-4.0) (#15605)
* [f22cdadaa](https://github.com/argoproj/argo-workflows/commit/f22cdadaaa0f40cabbcf0f18259c4c562e82c4bd) chore(deps): update module go.opentelemetry.io/otel/sdk to v1.40.0 [security] (release-4.0) (#15606)
* [bb301ad54](https://github.com/argoproj/argo-workflows/commit/bb301ad545d5a4cb75e8b009ad67ca9cc869cc32) chore(deps): update module go.opentelemetry.io/otel/sdk to v1.40.0 [security] (release-4.0) (#15603)
* [2b4dc19b0](https://github.com/argoproj/argo-workflows/commit/2b4dc19b023476aea3c80ddb7ca86b3daef3a273) fix: Add optional UID query parameter to GetWorkflow (cherry-pick #15251 for 4.0) (#15579)

<details><summary><h3>Contributors</h3></summary>

* Alan Clucas
* Claude Opus 4.5
* Eduardo Rodrigues
* Isitha Subasinghe
* Nancy Sangani
* Ville Vesilehto

</details>

## v4.0.1 (2026-02-16)

Full Changelog: [v4.0.0...v4.0.1](https://github.com/argoproj/argo-workflows/compare/v4.0.0...v4.0.1)

### Selected Changes

* [73cf19d22](https://github.com/argoproj/argo-workflows/commit/73cf19d222553a90a5c05ef58b8468c554a4c9df) fix: configure logrus for `argoproj/pkg` internal usage (cherry-pick #15563 for 4.0) (#15569)
* [898ba41aa](https://github.com/argoproj/argo-workflows/commit/898ba41aa972903d85327e0b0dd83727c07f4f4a) chore(deps): update dependency qs to v6.14.2 [security] (release-4.0) (#15561)
* [db17a6c82](https://github.com/argoproj/argo-workflows/commit/db17a6c82aaf0d9db7b397c80978ee33c679e51f) fix: support large container args (cherry-pick #15265 for 4.0) (#15520)
* [0ed630ec2](https://github.com/argoproj/argo-workflows/commit/0ed630ec261303624268efd93b74b068341d9599) fix: emissary deadline handling (cherry-pick #15352 for 4.0) (#15553)
* [eb5508a3a](https://github.com/argoproj/argo-workflows/commit/eb5508a3a2ebc10ebfe828cffcf8ee34b04c1260) fix: change RLock to Lock in sync manager methods (cherry-pick #15546 for 4.0) (#15549)
* [e1c300172](https://github.com/argoproj/argo-workflows/commit/e1c300172f8a8edbaeecc662febdfc4556d002f4) chore(deps): update module github.com/go-git/go-git/v5 to v5.16.5 [security] (release-4.0) (#15544)
* [abebb174a](https://github.com/argoproj/argo-workflows/commit/abebb174a22934144c310690996a1f4c7d7efd8f) chore(deps): update github.com/knetic/govaluate digest to 7625b7f (release-4.0) (#15499)
* [0e9f4881b](https://github.com/argoproj/argo-workflows/commit/0e9f4881beecf376ef2e37029e5a69332266f6bb) chore(deps): update sigs.k8s.io/json digest to 2d32026 (release-4.0) (#15505)
* [b9e88dffb](https://github.com/argoproj/argo-workflows/commit/b9e88dffb7f762f62a95dd057dd2d1b411bed1e8) fix: prevent int64 overflow in retry backoff calculation (cherry-pick #15277 for 4.0) (#15508)
* [32eecf6c4](https://github.com/argoproj/argo-workflows/commit/32eecf6c4f510628975c8dedaab6cdadaa75f3fc) chore(deps): update k8s.io/gengo digest to 5ee0d03 (release-4.0) (#15501)
* [f8c5b1114](https://github.com/argoproj/argo-workflows/commit/f8c5b11143e3aa16c2b35ff3eb56c80bdb8aa8c9) chore(deps): update k8s.io/utils digest to 914a6e7 (release-4.0) (#15503)
* [a9df572a4](https://github.com/argoproj/argo-workflows/commit/a9df572a4870a2e78083bc6737833d3c2c90e5d1) chore(deps): update google.golang.org/genproto/googleapis/api digest to 546029d (release-4.0) (#15500)
* [8b6ffe8e4](https://github.com/argoproj/argo-workflows/commit/8b6ffe8e47752c4f92b68f3a08b222eb04b3839e) fix: Add instanceID label to WorkflowTaskSet. Fixes #15219 (cherry-pick #15220 for 4.0) (#15481)
* [ef75c9d6d](https://github.com/argoproj/argo-workflows/commit/ef75c9d6d5b0d31a580a5caaddeeebabcd52b8b3) fix: use input.defaults for suspend templates (cherry-pick #15240 for 4.0) (#15482)
* [01c6cb72e](https://github.com/argoproj/argo-workflows/commit/01c6cb72e1420ecd6ea629484c41b65bc06c0d50) fix: sync example is wrong for mutexes (cherry-pick #15431 for 4.0) (#15507)
* [2f608d5b3](https://github.com/argoproj/argo-workflows/commit/2f608d5b32bee1c47ee9075fc3aa90237f3c7f87) chore(deps): update lycheeverse/lychee-action digest to 631725a (release-4.0) (#15504)

<details><summary><h3>Contributors</h3></summary>

* Aaron Mell
* Alan Clucas
* AnaySh
* Anay Sharma
* Andre Kurait
* Uziel David Sulkies

</details>

## v4.0.0 (2026-02-04)

Full Changelog: [v4.0.0-rc4...v4.0.0](https://github.com/argoproj/argo-workflows/compare/v4.0.0-rc4...v4.0.0)

### Selected Changes

* [a8bff4a72](https://github.com/argoproj/argo-workflows/commit/a8bff4a72130bc1a13d95c6f73ff3fafec880287) fix(security): update qs to 6.14.1 (#15427)

<details><summary><h3>Contributors</h3></summary>

* Alan Clucas

</details>

## v4.0.0-rc4 (2026-01-21)

Full Changelog: [v4.0.0-rc3...v4.0.0-rc4](https://github.com/argoproj/argo-workflows/compare/v4.0.0-rc3...v4.0.0-rc4)

### Selected Changes

* [a67673e13](https://github.com/argoproj/argo-workflows/commit/a67673e13ecea933b414fc60a8c116437cadb258) chore(deps): bump golang to 1.25 (#15235)
* [159a5c562](https://github.com/argoproj/argo-workflows/commit/159a5c56285ecd4d3bb0a67aeef4507779a44e17) fix(security): stored XSS in artifact directory listings (#15255)
* [2824d2121](https://github.com/argoproj/argo-workflows/commit/2824d21213830d2a17eb05cac070cf36c6678c47) fix: ensure single trailing newline in feature generator output (MD047) (#15242)
* [b7670b678](https://github.com/argoproj/argo-workflows/commit/b7670b6789577454a8478e8eb5e8d1b9529546dc) fix: workflow controller to detect stale workflows (#15090)
* [9872c296d](https://github.com/argoproj/argo-workflows/commit/9872c296d29dcc5e9c78493054961ede9fc30797) fix: Optimize DAG sort for retry (#15241)

<details><summary><h3>Contributors</h3></summary>

* Alan Clucas
* Benjamin Scott Pruett
* Eduardo Rodrigues
* Gianluca
* Joibel
* Mason Malone
* panicboat
* William Van Hevelingen

</details>

## v4.0.0-rc3 (2026-01-08)

Full Changelog: [v4.0.0-rc2...v4.0.0-rc3](https://github.com/argoproj/argo-workflows/compare/v4.0.0-rc2...v4.0.0-rc3)

### Selected Changes

* [9f3f13c58](https://github.com/argoproj/argo-workflows/commit/9f3f13c5803249a028361d419c96f744cbbdc2a2) fix: quick-start should not include artifact plugin (#15223)
* [ec0f3619c](https://github.com/argoproj/argo-workflows/commit/ec0f3619ca46004ed82aaabb089902b48e3d17ed) fix: dedupe realtime metrics and handle delete tombstone (#15216)
* [9febcd280](https://github.com/argoproj/argo-workflows/commit/9febcd2802eb4960c3037298ed643f4eeafe598c) fix: server, label actor on retrywf (#15201)
* [980e99eaa](https://github.com/argoproj/argo-workflows/commit/980e99eaa67af25ecc7c4a5e7ec8e59e697c3b48) fix: Docs reference a UI service that does not exist (#15204)
* [caed4ae01](https://github.com/argoproj/argo-workflows/commit/caed4ae01894d62a2eea243bb04fb58cea93ae69) fix: Avoid resetting resourceVersion for watch. Fixes #15106 (#15107)

<details><summary><h3>Contributors</h3></summary>

* Alan Clucas
* Claude Opus 4.5
* Jack Cui
* Jake Leahy
* Joibel
* jswxstw
* lif
* oninowang
* Tianchu Zhao
* Yuan Tang

</details>

## v4.0.0-rc2 (2025-12-22)

Full Changelog: [v4.0.0-rc1...v4.0.0-rc2](https://github.com/argoproj/argo-workflows/compare/v4.0.0-rc1...v4.0.0-rc2)

### Selected Changes

* [5f7c7ec6f](https://github.com/argoproj/argo-workflows/commit/5f7c7ec6f5f6d6589a2c0cbc20acd79ca078f76f) fix: move logger context population before access (#15193)
* [4bece58be](https://github.com/argoproj/argo-workflows/commit/4bece58bee16720874a9571583df853155fa64ab) fix: test CheckWorkflowExistence check both mutex and sem (#15158)
* [7bfb4f26d](https://github.com/argoproj/argo-workflows/commit/7bfb4f26df7f90381b5d015b2da0fa18b9fe66be) fix: remove pod finalizer in abnormal scenarios (#15164)
* [7fd533ac2](https://github.com/argoproj/argo-workflows/commit/7fd533ac2dd6fb663d82b8f14ee9da9220cdf212) fix: make CRDs smaller (in general) and fix cron schedule regex (#15175)
* [9c59a607a](https://github.com/argoproj/argo-workflows/commit/9c59a607abb92078003807d32a22ba9e4c1aea0e) feat(controller): auto-restart pods that failed before starting (#15086)
* [64caebb59](https://github.com/argoproj/argo-workflows/commit/64caebb59a033fdf9ddb5d76a39e6934ea65381b) fix(ui): fix column alignment in workflow list (#15185)
* [0c774328a](https://github.com/argoproj/argo-workflows/commit/0c774328ae758b0257582a59fbe874067be153f9) fix: add timeout for database query in workflow estimation to prevent blocking. (#15116)
* [4a9227b27](https://github.com/argoproj/argo-workflows/commit/4a9227b27e3624914f6d864e3e3ed613e54feec0) chore(deps): bump the go_modules group across 2 directories with 1 update (#15172)
* [c8b66b244](https://github.com/argoproj/argo-workflows/commit/c8b66b244339803c16dbe78e43822fff085d040e) fix: type-aware application of template defaults (#14899) (#15144)
* [e621d1f34](https://github.com/argoproj/argo-workflows/commit/e621d1f34c5a618fde453610ef0519cd231a4cda) fix: check ClusterWorkflowTemplate RBAC cluster wide instead of namespaced. Fixes #15071 (#15162)
* [4117a1ec1](https://github.com/argoproj/argo-workflows/commit/4117a1ec14d34c8b0587f89b9f8a47e067fb1355) fix: upgrade expr to bring in bugfix. Fixes #15093 (#15168)
* [e991e6dc4](https://github.com/argoproj/argo-workflows/commit/e991e6dc49c45ac00d2388b60abb9f5b21ce4247) fix: use alpine 3.23 version in tests, examples, workflows (#15160)
* [67db97f2e](https://github.com/argoproj/argo-workflows/commit/67db97f2eaca3f45ed20c5c6497df97d3ba37db8) fix: set current platform as default platform when retrieving image entrypoint. Fixes #15058 (#15059)
* [76eda6f83](https://github.com/argoproj/argo-workflows/commit/76eda6f83d6ab0567806ca359151205fd35df124) fix: handle escape chars in withParam/withItems. Fixes #13718 (#14864)
* [46b0dddf9](https://github.com/argoproj/argo-workflows/commit/46b0dddf970953b42d67ce7fad3c5f32199548fb) fix: if pod fails without container termination, don't mark node succeeded always (#15150)

<details><summary><h3>Contributors</h3></summary>

* Alan Clucas
* AlbeeSo
* Claude
* Eduardo Rodrigues
* Giovanni Campagna
* guamian-delicious
* Isitha Subasinghe
* jiazhen.liu
* Joibel
* jswxstw
* Lars F. Karlström
* Mason Malone
* shuangkun tian
* Tzu-Ting
* Wael
* Yuan Tang

</details>

## v4.0.0-rc1 (2025-12-11)

Full Changelog: [v3.7.18...v4.0.0-rc1](https://github.com/argoproj/argo-workflows/compare/v3.7.18...v4.0.0-rc1)

### Selected Changes

* [afb0b7abe](https://github.com/argoproj/argo-workflows/commit/afb0b7abed27ceba08fde624d18d9d8b7e93ab36) perf: set ResourceVersion=0 in deleteTaskResults to reduce etcd pressure. (#15115)
* [0f31d8cd0](https://github.com/argoproj/argo-workflows/commit/0f31d8cd0d9f8ac6a9d1280fb3c87fa871a5c1e0) fix: make executable (#15129)
* [3e13180bf](https://github.com/argoproj/argo-workflows/commit/3e13180bf55b74c3bf4c61ea5b1f3f4c0d0f2b35) fix: more release fixing (#15128)
* [9b4e92f28](https://github.com/argoproj/argo-workflows/commit/9b4e92f28573fdaf017e8ff09b085cddc224a1f2) fix: release process bugs from #15124 (#15127)
* [6b92af23f](https://github.com/argoproj/argo-workflows/commit/6b92af23f35aed4d4de8b04adcaf19d68f006de1) Merge commit from fork
* [a5b57b126](https://github.com/argoproj/argo-workflows/commit/a5b57b126558f15ca9fd09ef557b77016bb7a01c) fix: always `convert` singular to plural (#15092)
* [36dcb410a](https://github.com/argoproj/argo-workflows/commit/36dcb410ad113cee013a0ca378a99fba5b6e67eb) feat(ui): add warning message when deleting CronWorkflow about orphaned Workflows (#14727)
* [092e36b1e](https://github.com/argoproj/argo-workflows/commit/092e36b1e134a7afdf6092111ffbf064e4a84ad5) feat: disable write back informer by default (#15079)
* [6a7caf056](https://github.com/argoproj/argo-workflows/commit/6a7caf0562ee40c415e906090c7aca8c07466c2e) feat: CLI convert command (#14996)
* [4058937ee](https://github.com/argoproj/argo-workflows/commit/4058937ee0313be01e3cc42b38d44e891d37941a) fix: expose not equals to UI (#15089)
* [26ef39445](https://github.com/argoproj/argo-workflows/commit/26ef39445bcf325358e8068e0661ce66c6361900) feat: support kubernetes equality-based ops for field selector `metadata.name` (#13476)
* [999d04c57](https://github.com/argoproj/argo-workflows/commit/999d04c57d801222579bc0d7f827d05e47687355) chore(deps): bump node-forge from 1.3.1 to 1.3.2 in /ui in the deps group across 1 directory (#15085)
* [b9d18b156](https://github.com/argoproj/argo-workflows/commit/b9d18b15697a614d89901f3d47243b5a02698859) fix: rename variables #15061 and add failure error (#15067)
* [256b2c4e7](https://github.com/argoproj/argo-workflows/commit/256b2c4e7c62e63c04fd86364ca311d75a258768) fix: http template read response.Body after cancel(), sometimes it return a context canceled error (#14853)
* [bee7b1430](https://github.com/argoproj/argo-workflows/commit/bee7b14301fe71758ebd87de406d2440ee87f29f) fix(ui): fix BASE_HREF in production index.html. Fixes #15046 (#15066)
* [f763101d1](https://github.com/argoproj/argo-workflows/commit/f763101d1884c2cf209645ad12788ab4cb66b0f6) chore(deps): bump golang.org/x/crypto from 0.43.0 to 0.45.0 in the go_modules group across 1 directory (#15062)
* [59f9b7b06](https://github.com/argoproj/argo-workflows/commit/59f9b7b06e00c24693904a6064d596b4321b2e76) fix: fix CRDs and add test for #14991 (#15055)
* [6199d8f4e](https://github.com/argoproj/argo-workflows/commit/6199d8f4ec4bc7561a920447bf70bd718e6fa311) feat(sso): allow custom ca configuration (#14989)
* [199a1373a](https://github.com/argoproj/argo-workflows/commit/199a1373a525955ed13ff77fdf9337de139fddb5) fix: Fixes parameterized global artifacts resolution in exit handlers. Fixes #11610 (#14991)
* [34b495039](https://github.com/argoproj/argo-workflows/commit/34b49503933b98c4c93959793d08d8179091d31f) feat: add CEL validation rules to the CRDs (#15028)
* [2672ba3be](https://github.com/argoproj/argo-workflows/commit/2672ba3be138ea37268f78c29e42e2f4b9fdbb09) fix: preserve global scope variables in withSequence expressions (#14718)
* [d3ca7c694](https://github.com/argoproj/argo-workflows/commit/d3ca7c69444e51c8c8f0b9082c5531ac6c21716a) fix: add archive location if artifact is needed in data source. Fixes #14990 (#15004)
* [f61191984](https://github.com/argoproj/argo-workflows/commit/f611919848dda359e028f32d1960026a96b2d2e9) chore(deps): bump golang version from 1.24.4 to 1.24.10 (#15037)
* [f4eef2b55](https://github.com/argoproj/argo-workflows/commit/f4eef2b55a477e025adc06bcd8d9a5dcbc8e49d9) fix: add a special case for `item` variable during global expression replacement (#15033)
* [e98b6b339](https://github.com/argoproj/argo-workflows/commit/e98b6b3396f66ed7f93cfa09531ba20f5b3f5d63) fix!: return latest workflow for retried-persisted workflows (#15030)
* [436a8d56f](https://github.com/argoproj/argo-workflows/commit/436a8d56fb1307b7c894a42331993dc970af235a) fix: enforce metrics attributes, also fixing one doc (#14979)
* [e372a0770](https://github.com/argoproj/argo-workflows/commit/e372a0770807457dc0965efc42b2a43a70be02d2) fix!: status->phase in the gauge metric (#14976)
* [c9cba3b19](https://github.com/argoproj/argo-workflows/commit/c9cba3b1927b860b0099f6160060e4d3dfa24a31) fix: prevent nil pointer dereference in GetTemplateFromRef with podMetadata. Fixes #14968 (#14970)
* [801b82a8f](https://github.com/argoproj/argo-workflows/commit/801b82a8fe57ec9825ad1e6c74c7da833d9b7c27) feat(deprecation): remove 3.6 deprecated features for 4.0 (#14189)
* [a2924e6f6](https://github.com/argoproj/argo-workflows/commit/a2924e6f6299f97b8825a3712e8c6f08d99731b2) fix: allow legacy name validation scheme for prometheus metrics (#14879)
* [c13a49220](https://github.com/argoproj/argo-workflows/commit/c13a49220a81a5da4733fbe04be472511d60438e) feat(ui): add label filter sync with URL query params (#14816)
* [de8c12cbf](https://github.com/argoproj/argo-workflows/commit/de8c12cbf2c670c937af81430d1aced2bc3d3647) refactor: modernize go code with stdlib improvements (#14946)
* [70f833695](https://github.com/argoproj/argo-workflows/commit/70f83369519619f221ffca2e123ffe7dec623be0) fix: cherry picker message (#14975)
* [2b100dad0](https://github.com/argoproj/argo-workflows/commit/2b100dad072330557fff4d23610703997f540896) feat: update featuregen to use plural "Authors:" field (#14958)
* [bc8531b23](https://github.com/argoproj/argo-workflows/commit/bc8531b23f2511decf643fc55ccc7a1249572499) feat: artifact plugins (#14915)
* [2bd2fc20d](https://github.com/argoproj/argo-workflows/commit/2bd2fc20d5d97f3005b78ea79b439f08b3c869ab) fix: call WaitGroup.Wait before goroutine launch (#14948)
* [231b40b30](https://github.com/argoproj/argo-workflows/commit/231b40b30b82a3bde0f2116adb2858a99b93d949) fix: missing quote (#14957)
* [0f74e7601](https://github.com/argoproj/argo-workflows/commit/0f74e76011d5758aa15544a07a39cb221fef0cd5) fix: cherry-pick title with quotes (#14947)
* [06edbe038](https://github.com/argoproj/argo-workflows/commit/06edbe038fb2f73ea18e43757f381c79d6703f59) fix: allow `labelsFrom` to be specified in `workflowDefaults`. Fixes #14927 (#14941)
* [a48868cf8](https://github.com/argoproj/argo-workflows/commit/a48868cf8ff74fad0577efd9404c3b11bb2369b3) fix: typo paramaters to parameters in cli help text (#14939)
* [5d75b62e2](https://github.com/argoproj/argo-workflows/commit/5d75b62e2ccd4498020dffc94c18bd15a0062578) fix(ui): support base HREF in dev environment (#14894)
* [d129ccbd7](https://github.com/argoproj/argo-workflows/commit/d129ccbd714d0df6abc3f86eeb4a28365c24cb55) fix(ui): login/logout bug when base HREF set. Fixes #14897 (#14909)
* [e611a1aaa](https://github.com/argoproj/argo-workflows/commit/e611a1aaaaa2a50c38d7c3b9817f6ec0fb14320f) fix(ui): avoid phantom dependencies with pnpm (#14890)
* [cebe4b3e7](https://github.com/argoproj/argo-workflows/commit/cebe4b3e7746765e80ebceec7c3bcbd44437d83b) fix: fetch sa in delegate namespace when no sa matched. Fixes:#14610 (#14614)
* [179571ee1](https://github.com/argoproj/argo-workflows/commit/179571ee154c55c831b2d74e685420072cdea1ea) feat(server): optimize pagination when counting workflows in archive. Fixes:#13948 (#14892)
* [8c7445c74](https://github.com/argoproj/argo-workflows/commit/8c7445c747a4630741a65738b74a84a31470e845) chore(deps): bump x/crypto for CVE-2025-47913 (#14924)
* [4bacc6c6b](https://github.com/argoproj/argo-workflows/commit/4bacc6c6b51760071f0ee083cc9d9fb4aae1412f) Merge commit from fork
* [c61dee7c5](https://github.com/argoproj/argo-workflows/commit/c61dee7c5ee713ededf6e4135437c32ef453bb9f) Merge commit from fork
* [4d830d37f](https://github.com/argoproj/argo-workflows/commit/4d830d37fab4c86fa88fd48438c0d2dae1306200) feat: Add support for creating a database semaphore config using CLI (#14784)
* [90cfa7e4f](https://github.com/argoproj/argo-workflows/commit/90cfa7e4f8b97368ef03a534769daf13e7d51103) fix: cache calls to prevent exponential recursion. Fixes #14904 (#14920)
* [0ebee7878](https://github.com/argoproj/argo-workflows/commit/0ebee7878f05fa4fecaa944fb810bbab263a2be7) fix: 3.7 version modal and docs (#14921)
* [d1b280ece](https://github.com/argoproj/argo-workflows/commit/d1b280ecec529e2aeb37485e2bc50f2820d8aae0) fix: skip validate dynamic templateref (#14850)
* [2c033c168](https://github.com/argoproj/argo-workflows/commit/2c033c168d3898e77371d190119d24a3439a1422) fix: avoid deleting from the throttle queue when reapply fails. (#14855)
* [4eb7b013f](https://github.com/argoproj/argo-workflows/commit/4eb7b013ff7fa0d70b76c9f84d78046244b53c6d) fix: linting pointer error when template name is invalid (#14896)
* [1d9d037ee](https://github.com/argoproj/argo-workflows/commit/1d9d037ee8bc7419a13e8b45b8291ae57e4aa4c4) fix: change retry workflow phase from running to unknown.  (#14854)
* [6eeccb280](https://github.com/argoproj/argo-workflows/commit/6eeccb28034d6e4fdaf3a6c0a9fbcc3405d64e94) fix: retry when occasionally timeout.  (#14868)
* [cb7ebd939](https://github.com/argoproj/argo-workflows/commit/cb7ebd9393f3322abf455d906e39a3a976421b30) fix: CVE-2025-47913. Fixes:#14866 (#14871)
* [ece9e9007](https://github.com/argoproj/argo-workflows/commit/ece9e9007a63971d73ebe7b81bd755fe22913e39) feat: Add support for creating a configmap semaphore config using CLI. Fixes #14671 (#14724)
* [41a9328ef](https://github.com/argoproj/argo-workflows/commit/41a9328efd150a3763590dab2379c403da4841ef) fix: don't need to add workflow to pending queue when running. (#14856)
* [ea3c5882f](https://github.com/argoproj/argo-workflows/commit/ea3c5882f4f8d62e5796ac5a1cb22a5dfe5d07a3) fix: fix WithFields and WithError in init logger (#14862)
* [a49773c7c](https://github.com/argoproj/argo-workflows/commit/a49773c7c422a1271fb8bd9148942cf16dc063ce) fix(manifests): add descriptions to CRDs. Fixes #8190 (#14777)
* [3e2c22149](https://github.com/argoproj/argo-workflows/commit/3e2c22149f103a7e38f09ce93c8e9fe54503cff7) fix: reset the taskgroup. fixes #14769 (#14832)
* [87b2c640d](https://github.com/argoproj/argo-workflows/commit/87b2c640d5aa0c22e204ba97a0043cfce7bb12d0) chore(deps): bump superagent to 10.2.3 (#14840)
* [e1855be1a](https://github.com/argoproj/argo-workflows/commit/e1855be1a073f5ad399368f0bcc5594de73b7071) fix: only reset failed or error retry node. Fixes #14796 (#14803)
* [74353577f](https://github.com/argoproj/argo-workflows/commit/74353577f83416e51d7aaf44ecab86f91a81e973) fix: Add default value for creationtimestamp column addition. Fixes #… (#14797)
* [149db428a](https://github.com/argoproj/argo-workflows/commit/149db428a5962879517cafe2ca5212cd6ea187a7) fix: Fixes git over azure devops Fixes #11705 (#13875)
* [812308e75](https://github.com/argoproj/argo-workflows/commit/812308e7575b894461ee968f07142419018258fb) fix: cluster workflow template store is not initialized in namespace mode. Fixes #14763 (#14766)
* [5c34b581b](https://github.com/argoproj/argo-workflows/commit/5c34b581beb5656b7f2f916b271942e8dbfc83cd) fix: prevent EEXIST error when devcontainer is installed via homebrew. Fixes: #14513 (#14758)
* [759ee47f8](https://github.com/argoproj/argo-workflows/commit/759ee47f8c143a65140a610d9ce6e73b97db4c94) fix: ci step titles MySQL->Database (#14695)
* [448c492be](https://github.com/argoproj/argo-workflows/commit/448c492beea334e5405aba58b91c57e97f85b32e) fix: ensure pod used container templates are copies (#14773)
* [ba9257b7a](https://github.com/argoproj/argo-workflows/commit/ba9257b7ace65c10d15e5aa67806f8d06297116a) fix: Fix ST1005 linting issue. Fixes #14405 (#14699)
* [71e0da411](https://github.com/argoproj/argo-workflows/commit/71e0da411a3c8a4a5632aec96ab8d7f81d245f49) chore(deps): update nao1215/markdown to v0.8.0 (#14787)
* [41e1e8eea](https://github.com/argoproj/argo-workflows/commit/41e1e8eea386c792a4f5d9c51664da6b78f90e60) fix: retry strategy being ignored by daemoned nodes. Fix #14715 (#14782)
* [b47c0b058](https://github.com/argoproj/argo-workflows/commit/b47c0b058fd4250ec3776daa6e0e4aa2b06fa2ba) fix: moved off deprecated bitnami Docker images. Fixes 14785 (#14786)
* [a4b9cb3a3](https://github.com/argoproj/argo-workflows/commit/a4b9cb3a3efbbcb480063dee690226d1a739acee) fix(controller): check optional artifacts in node output validation (#14778)
* [895239040](https://github.com/argoproj/argo-workflows/commit/895239040b5b4b4ded9bd082bed382147b4ae939) chore(deps): bump github.com/go-viper/mapstructure/v2 from 2.3.0 to 2.4.0 in the go_modules group (#14779)
* [da5304e5b](https://github.com/argoproj/argo-workflows/commit/da5304e5b5ef5dc2196b7ee9ff893f74d38bd8cd) fix: only mark the realtime metrics of the workflow itself as completed. Fixes #14694 (#14764)
* [9719c2176](https://github.com/argoproj/argo-workflows/commit/9719c2176bee741f798c1bc33ef0eba9ee9f6b6b) fix: S3 artifacts failing with missing region when using roleARN (#14761)
* [06aa37025](https://github.com/argoproj/argo-workflows/commit/06aa3702506ee68b122e366f943dc3abf3fbc6fa) fix: added createdAfter/finishedBefore query support. Fixes #14722 (#14721)
* [57e62a2fc](https://github.com/argoproj/argo-workflows/commit/57e62a2fcf932f28b419247fa8341868e486467d) fix: allow query parameters for DELETE method and disable body marshaling. Fixes #14753 (#14754)
* [8fb4bbaf7](https://github.com/argoproj/argo-workflows/commit/8fb4bbaf783d7b912db3a9c69629962172893f73) fix: cronworkfow update via UI when parameter is using valueFrom. Fixes #14550 (#14745)
* [5f6b126fc](https://github.com/argoproj/argo-workflows/commit/5f6b126fcca65aeee45c1c7806a022d6111c0eab) fix: submit workflow template with enum. Fix #14704 (#14748)
* [d9ea1e82a](https://github.com/argoproj/argo-workflows/commit/d9ea1e82a82f6f21210cce44c2880e650ef4cee3) fix: watch for dynamic changes to the Controller ConfigMap in its installation namespace. Fixes #14673 (#14675)
* [14a58fb27](https://github.com/argoproj/argo-workflows/commit/14a58fb27e6280be50a2bc126381f42924e3d0fb) fix: use Float64ObservableCounter for counter metrics. Fixes #14425 (#14700)
* [b8234d242](https://github.com/argoproj/argo-workflows/commit/b8234d2423e657d45091200152de686cf63102d7) fix: only init Informers if SA has apropriate RBAC access. Fixes #14688 (#14731)
* [49c82f5a8](https://github.com/argoproj/argo-workflows/commit/49c82f5a850f220e7b7cc8147833ed281c4de512) fix: cron patch use defaultRetry. Fixes #14712 (#14713)
* [eacb7eb4a](https://github.com/argoproj/argo-workflows/commit/eacb7eb4a098abc1637a50b89e7cc926d8e4f946) fix: prevent thundering herd on cache save/load. Fixes #14701 (#14703)
* [e8de7024e](https://github.com/argoproj/argo-workflows/commit/e8de7024ef46a2ceeda4e6d64212991155c87d24) feat: remove logrus and propogate context everywhere (#14680)
* [9ed9ed997](https://github.com/argoproj/argo-workflows/commit/9ed9ed9972616395dd3457a47092de07b75fb5f9) fix: support ALPN and H2C with gRPC. Fixes #14627 (#14567)
* [6e2b95a9f](https://github.com/argoproj/argo-workflows/commit/6e2b95a9f4e1dd5fbf37276df91673a337b36d15) fix(ui): Add multibyte character support tests for pod name generation (#14653)
* [92778487d](https://github.com/argoproj/argo-workflows/commit/92778487d388267568a71b40ae86c2c2763e55cc) refactor(ui): Similar login page to argo cd. Fixes #10816 (#14662)
* [0588e1b0f](https://github.com/argoproj/argo-workflows/commit/0588e1b0fedef1ed9f1efd2391f1694b0a77cb7e) fix: GC realtime metrics after workflow completed. Fixes #14694 (#14696)
* [a5f5e5814](https://github.com/argoproj/argo-workflows/commit/a5f5e58144255f5b455063a9268c4343df77b096) feat: take effect total parallelism without restart controller. Fixes: #14689 (#14690)
* [81309bb6f](https://github.com/argoproj/argo-workflows/commit/81309bb6f201784a949ac3156f6e1d30a785f5cb) fix: only remove when pending namespace queue exists. Fixes:#14669 (#14670)
* [d86867cd9](https://github.com/argoproj/argo-workflows/commit/d86867cd97ba397b26ae790f2fdc89c07986e6cb) fix: Sidecar terminates itself after the main container is finished. Closes #10612 (#14633)
* [85b4c649d](https://github.com/argoproj/argo-workflows/commit/85b4c649d01e64dafc75760cf96c1abfdce463f8) fix: process aggregate outputs for steps node with retries. Fixes #14647 (#14651)
* [7fd6b10f8](https://github.com/argoproj/argo-workflows/commit/7fd6b10f87879ac0dcdb85466c4e2e2f613ace29) feat: Respect NameFilter in Workflow Archive (fixes #14069) (#14473)
* [175322c9c](https://github.com/argoproj/argo-workflows/commit/175322c9c9d8396d2d3e742f091f30f37c34f4c9) fix: Make codegen easier to understand when it fails (#14619)
* [48b247f12](https://github.com/argoproj/argo-workflows/commit/48b247f12a517cad01d390a889d468faa0de11e6) fix: merge collision fix (#14643)
* [f73e8b0e7](https://github.com/argoproj/argo-workflows/commit/f73e8b0e728fe0cbc9d2b8448cbe21d1299d6d0a) fix: Make the phase of the node unchange when the pod is completed and outputs are not set in the status.node (#14625)
* [05ec2d61e](https://github.com/argoproj/argo-workflows/commit/05ec2d61ecbfe165f48b02fbfb3d24e4b547c3ae) feat: support open custom links in new tab.Part of #13114 (#14314)
* [e41963711](https://github.com/argoproj/argo-workflows/commit/e4196371134016313bf4d9308c1e9dccf1155c46) feat: logging refactor to `slog`. Fixes #11120 (#14527)
* [3a95f4ef2](https://github.com/argoproj/argo-workflows/commit/3a95f4ef2a70cf6ede5e8a3e15a43a6c5d319197) fix: fix for feature note changed files (#14640)
* [d5bbf1fb9](https://github.com/argoproj/argo-workflows/commit/d5bbf1fb91b6e8547645103ec45a19c77895d9b1) fix: retry when the server is temporarily unavailable.  (#14637)
* [27ece5842](https://github.com/argoproj/argo-workflows/commit/27ece58423d27c2808c4cfac82b2a49ee785d6af) fix: do PR check with some depth for merge-base (#14636)
* [1b963d336](https://github.com/argoproj/argo-workflows/commit/1b963d336c8e1f2fe3ec66ed467531d3e80da7b1) fix: ensure task results sync when calling fullfilled. Fixes #14568 (#14536)
* [9b276dfc0](https://github.com/argoproj/argo-workflows/commit/9b276dfc0dd4b1aa6436c65a9c1ea61437eac5fb) feat: new-features automated documentation (#14491)
* [a5d43eb20](https://github.com/argoproj/argo-workflows/commit/a5d43eb2025fffe86817d23b8cf488596d7ca0d8) fix: avoid healthz check restart controller. Fixes: #14526 (#14613)
* [85e96a1f7](https://github.com/argoproj/argo-workflows/commit/85e96a1f70226992e8e7a35c6aa631f52b872cc2) fix: correct finding the closest ancestor retry node. Fixes #14517 (#14576)
* [162e6454d](https://github.com/argoproj/argo-workflows/commit/162e6454d881de4fa42cd8c756b7dc9f84ccbcd1) fix: create task results only once. Fixes: #14617 (#14618)
* [5807fadbd](https://github.com/argoproj/argo-workflows/commit/5807fadbd16633ddcc2e59225fe0045b56c4ac3c) fix: add etcd too many requests transient. (#14621)
* [9c4775598](https://github.com/argoproj/argo-workflows/commit/9c477559800dda8981306679fe79a3a99a34eefc) chore(deps): bump golang version (#14596)
* [62302816f](https://github.com/argoproj/argo-workflows/commit/62302816ff8de28bcb0f92fc5efd1915d130ef9d) chore(deps): remove open-golang dependency (#14591)
* [a58f34fb6](https://github.com/argoproj/argo-workflows/commit/a58f34fb66d051bd8d325d2a9809b826f47cf93b) chore(deps): replace golang/mock with maintained port by uber (#14592)
* [48e7984b7](https://github.com/argoproj/argo-workflows/commit/48e7984b78d00f5539585807bcce589258dd10fc) feat: Display when Conditions in CronWorkflow UI : Fixes: #14334 (#14585)
* [e634b1d4f](https://github.com/argoproj/argo-workflows/commit/e634b1d4f31c2a2be22d5f8f47676e1069f4e656) fix: set creator when use X509 client certificates. Fixes: #14578 (#14579)
* [ec99485c0](https://github.com/argoproj/argo-workflows/commit/ec99485c062bdc4955fd849db9c994628e2cab08) chore(deps): bump github.com/go-viper/mapstructure/v2 from 2.2.1 to 2.3.0 in the go_modules group (#14611)
* [883b502da](https://github.com/argoproj/argo-workflows/commit/883b502dac1583b8f5f464ab293668e7565f474d) fix: prevent running workflow throttle by parallelism (#14606)
* [ca4bfb874](https://github.com/argoproj/argo-workflows/commit/ca4bfb874a79b7384a8c0b30b06130e8df8df3bd) fix: executor workflowtaskresult retry should use the default retry and configurable (#14598)

<details><summary><h3>Contributors</h3></summary>

* akash khamkar
* Alan Clucas
* Andrey Kuznetsov
* AnonyScorpio
* antoinetran
* Armin Friedl
* Beomgi Kim | 김범기
* Bjoern Weidlich
* Bradford Wagner
* Cayde6
* chenrui
* chenrui7
* Claude
* Copilot
* cyzlmh
* Darko Janjic
* Dohyeong Lee
* downfa11
* edaniel
* Eduardo Rodrigues
* edward
* Elliot Gunton
* garireo2549
* Gianluca
* Isitha Subasinghe
* isubasinghe
* ItielOlenick
* Jason Meridth
* Jemin Seo
* J. Gavin Ray
* jiwlee
* Joep Keijsers
* Joibel
* Jose M. Abuin
* J.P. Zivalich
* jswxstw
* k3rnL
* Ken Cochrane
* Lee Jiny
* littlejian
* lons
* Marcus Weiner
* Mason Malone
* Matt McLane
* Milas Bowman
* Miltiadis Alexis
* 민선 (minnie)
* Nitin Bhojwani
* okzw999
* oninowang
* Scott Melhop
* Sebastien Dionne
* Sergey K.
* Shota Sugiura
* shuangkun tian
* Tianchu Zhao
* Tim Collins
* tooptoop4
* William Van Hevelingen
* Xavier Hardy
* Yuan Tang

</details>

## v3.7.18 (2026-08-14)

For v3 releases, see [CHANGELOG-3-x-x.md](CHANGELOG-3-x-x.md)

## v2.12.13 (2021-08-18)

For v2 releases, see [CHANGELOG-2-x-x.md](CHANGELOG-2-x-x.md)
