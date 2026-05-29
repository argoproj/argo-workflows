# -*- mode: Python -*-
# Tiltfile for argo-workflows local dev and CI. Replaces kit/tasks.yaml.
# Builds the controller, server and executor images and runs them in-cluster.
# Dev (`tilt up`): binaries are compiled on the host and COPYed into small
# alpine images; on change Tilt rebuilds and recreates the pod. CI
# (`tilt ci -- --mode=ci`): builds the real distroless production targets.
#
# Usage:
#   make k3d-up
#   tilt up -- --profile=minimal
#   tilt ci -- --profile=minimal --mode=ci
#
# Settings (normally driven by the Makefile / CI, see `make start`):
#   --auth-mode=<mode>                  argo-server auth mode (hybrid; sso for PROFILE=sso)
#   --secure=true                       argo-server serves TLS
#   --api=false                         don't build or run the argo-server
#   --pod-status-capture-finalizer=...  controller pod-status-capture finalizer toggle

# Allow more resources to build concurrently than the default of 3, so the
# controller/CLI/executor compiles run in parallel on a cold start.
update_settings(max_parallel_updates=4)

config.define_string('profile')
config.define_string('mode')
config.define_string('auth-mode')
config.define_string('secure')
config.define_string('api')
config.define_string('pod-status-capture-finalizer')
cfg = config.parse()
profile = cfg.get('profile', 'minimal')
mode = cfg.get('mode', 'up')
is_ci = mode == 'ci'
auth_mode = cfg.get('auth-mode', 'hybrid')
secure = cfg.get('secure', 'false')
api = cfg.get('api', 'true') != 'false'
finalizer = cfg.get('pod-status-capture-finalizer', 'true')

ctx = k8s_context()
if not ctx.startswith('k3d-'):
    fail("Tiltfile expects a k3d cluster (context k3d-*). Run 'make k3d-up'. Got: %s" % ctx)
allow_k8s_contexts(ctx)

IMAGE_NS = 'quay.io/argoproj'
# Pass the same version metadata the host Makefile computes, so in-cluster
# binaries report the same version as host-built ones (`./dist/argo`): the CLI
# prints a version-mismatch warning that corrupts e2e tests' parsed output.
GIT_COMMIT = str(local('git rev-parse HEAD', quiet=True)).strip()
GIT_TAG = str(local('git describe --exact-match --tags --abbrev=0 2>/dev/null || echo untagged', quiet=True)).strip()
GIT_TREE_STATE = str(local('[ -z "$(git status --porcelain)" ] && echo clean || echo dirty', quiet=True)).strip()
BUILD_ARGS = '--build-arg GIT_COMMIT=%s --build-arg GIT_TAG=%s --build-arg GIT_TREE_STATE=%s' % (GIT_COMMIT, GIT_TAG, GIT_TREE_STATE)

# The argo CRDs are too large for client-side apply (the last-applied
# annotation exceeds 256KB), so apply them server-side out of band, like
# `make install` does. The namespace must exist before anything else; the
# e2e manifests don't create it (kit created it imperatively).
full_yaml = kustomize('test/e2e/manifests/' + profile)
crds, rest = filter_yaml(full_yaml, kind='CustomResourceDefinition')
local('mkdir -p .tilt && cat > .tilt/crds.yaml', stdin=crds, quiet=True, echo_off=True)

if not api:
    # Suites that don't exercise the API skip the server, like the old
    # `make start API=false`. Anchor the kinds: filter_yaml matches them as
    # unanchored regexes, so a bare 'Service' would also remove the
    # 'argo-server' ServiceAccount — and with it the service-account-token
    # Secret the e2e fixtures authenticate with (the token controller deletes
    # token secrets whose ServiceAccount is missing).
    _dep, rest = filter_yaml(rest, kind='^Deployment$', name='^argo-server$')
    _svc, rest = filter_yaml(rest, kind='^Service$', name='^argo-server$')

# Apply the Makefile-controlled settings. The old host-process flow injected
# these as env vars on the local processes; patch them into the in-cluster
# manifests instead.
objs = decode_yaml_stream(rest)
for o in objs:
    if not o or o.get('kind') != 'Deployment':
        continue
    container = o['spec']['template']['spec']['containers'][0]
    if o['metadata']['name'] == 'workflow-controller':
        env = container.get('env', [])
        env.append({'name': 'ARGO_POD_STATUS_CAPTURE_FINALIZER', 'value': finalizer})
        container['env'] = env
    if o['metadata']['name'] == 'argo-server':
        # the e2e base sets both args; replace them, or append if a future
        # manifest reshuffle drops them (a silent no-op here would deploy a
        # server with the wrong auth/TLS mode)
        args = container.get('args', [])
        replaced = {'--auth-mode=': False, '--secure=': False}
        values = {'--auth-mode=': auth_mode, '--secure=': secure}
        for i in range(len(args)):
            for prefix in replaced.keys():
                if args[i].startswith(prefix):
                    args[i] = prefix + values[prefix]
                    replaced[prefix] = True
        for prefix in replaced.keys():
            if not replaced[prefix]:
                args.append(prefix + values[prefix])
        container['args'] = args
        if secure == 'true':
            # the e2e base probes over HTTP; a TLS server needs an HTTPS probe
            probe = container.get('readinessProbe', {})
            if not probe.get('httpGet'):
                fail('argo-server has no httpGet readinessProbe; cannot patch its scheme for --secure=true')
            probe['httpGet']['scheme'] = 'HTTPS'
rest = encode_yaml_stream(objs)

k8s_yaml('test/e2e/manifests/argo-ns.yaml')
local_resource('argo-crds',
    cmd='kubectl apply --server-side --force-conflicts -f .tilt/crds.yaml',
    deps=['.tilt/crds.yaml'],
    labels=['setup'])
k8s_yaml(rest)

cluster = ctx[len('k3d-'):]

# Images are delivered to k3d via `k3d image import` (no registry required, so
# this works on any k3d cluster including the devcontainer's). The controller
# and server are in workload pod specs, so Tilt tracks them via custom_build and
# rewrites their refs; the executor is injected into step pods at runtime (not a
# workload), so a standalone resource builds and imports it under the canonical
# tag, the same way the old kit flow did.
# `--mode direct` loads straight into the node containerd without the shared
# k3d-tools container, so concurrent imports don't collide. The import streams
# can still fail transiently under CI load ("read/write on closed pipe"), and
# Tilt doesn't retry failed builds, so retry once here.
def k3d_import(images):
    imp = 'k3d image import --mode direct -c %s %s' % (cluster, images)
    return '(%s || (sleep 5 && %s))' % (imp, imp)

def k3d_build(ref, target, deps, prebuild=None, canonical=None):
    steps = ([prebuild] if prebuild else []) + [
        'docker build -f Dockerfile %s --target %s -t $EXPECTED_REF .' % (BUILD_ARGS, target),
    ]
    images = '$EXPECTED_REF'
    if canonical:
        # also deliver the image under its canonical tag, for test workflows
        # that spawn pods referencing it directly
        steps.append('docker tag $EXPECTED_REF %s' % canonical)
        images += ' ' + canonical
    steps.append(k3d_import(images))
    # skips_local_docker=True: our command fully delivers the image to the
    # cluster (build + k3d import), so Tilt deploys $EXPECTED_REF as-is instead
    # of re-tagging and trying to push/load it itself.
    custom_build(ref, ' && '.join(steps), deps=deps, skips_local_docker=True)

ctrl_src = ['cmd/workflow-controller', 'workflow', 'config', 'errors', 'persist', 'pkg', 'util']
cli_src = ['cmd/argo', 'server', 'workflow', 'config', 'errors', 'persist', 'pkg', 'util']
exec_src = ['cmd/argoexec', 'workflow', 'config', 'errors', 'pkg', 'util', 'Dockerfile']

CLI_CANONICAL = IMAGE_NS + '/argocli:latest'

if is_ci:
    # CI builds the real production images from source for parity.
    k3d_build(IMAGE_NS + '/workflow-controller', 'workflow-controller', ctrl_src + ['ui', 'api', 'Dockerfile'])
    cli_target = 'argocli'
    cli_deps = cli_src + ['ui', 'api', 'Dockerfile']
    cli_prebuild = None
    exec_prebuild = None
    exec_target = 'argoexec'
else:
    # Dev: compile each binary once on the host; the dev images just COPY it in.
    local_resource('controller-compile', cmd='make dist/workflow-controller',
        deps=ctrl_src, labels=['compile'])
    k3d_build(IMAGE_NS + '/workflow-controller', 'workflow-controller-dev',
        ['dist/workflow-controller', 'hack/ssh_known_hosts', 'hack/nsswitch.conf', 'Dockerfile'])
    cli_target = 'argocli-dev'
    cli_deps = ['dist/argo', 'hack/ssh_known_hosts', 'hack/nsswitch.conf', 'Dockerfile']
    cli_prebuild = 'make dist/argo STATIC_FILES=false'
    exec_prebuild = 'make dist/argoexec'
    exec_target = 'argoexec-dev'

# The argocli image is needed even without the server: e2e test workflows
# spawn pods running the canonical quay.io/argoproj/argocli:latest (e.g.
# `argo stop` steps in the artifact and retry suites).
if api:
    if not is_ci:
        local_resource('cli-compile', cmd='make dist/argo STATIC_FILES=false',
            deps=cli_src, labels=['compile'])
    k3d_build(IMAGE_NS + '/argocli', cli_target, cli_deps, canonical=CLI_CANONICAL)
else:
    cli_cmd = ' && '.join(([cli_prebuild] if cli_prebuild else []) + [
        'docker build -f Dockerfile %s --target %s -t %s .' % (BUILD_ARGS, cli_target, CLI_CANONICAL),
        k3d_import(CLI_CANONICAL),
    ])
    local_resource('argocli-image', cmd=cli_cmd, deps=cli_deps, labels=['images'])

exec_cmd = ' && '.join(([exec_prebuild] if exec_prebuild else []) + [
    'docker build -f Dockerfile %s --target %s -t %s/argoexec:latest .' % (BUILD_ARGS, exec_target, IMAGE_NS),
    k3d_import('%s/argoexec:latest' % IMAGE_NS),
])
local_resource('argoexec-image', cmd=exec_cmd, deps=exec_src, labels=['images'])

# Bind the forwards to 0.0.0.0 (not Tilt's default 127.0.0.1) so they are
# reachable via the container's IP — e.g. when running in a devcontainer via the
# devcontainer CLI, which (unlike the VS Code extension) doesn't forward ports.
# This matches kit, whose server process bound 0.0.0.0.
k8s_resource('workflow-controller', port_forwards=[port_forward(9090, host='0.0.0.0')],
             resource_deps=(['argo-crds'] if is_ci else ['argo-crds', 'controller-compile']))
if api:
    k8s_resource('argo-server', port_forwards=[port_forward(2746, host='0.0.0.0')],
                 resource_deps=(['argo-crds'] if is_ci else ['argo-crds', 'cli-compile']))

# Port-forward the backing services that the e2e tests (and SSO logins) reach
# over localhost, replacing the old kit/kubeauto forwards. /etc/hosts maps
# minio, postgres, mysql and dex to 127.0.0.1 — see running-locally.md.
backing_forwards = {'minio': [9000]}
if profile == 'mysql':
    backing_forwards['mysql'] = [3306]
if profile == 'postgres':
    backing_forwards['postgres'] = [5432]
if profile == 'sso':
    backing_forwards['dex'] = [5556]
for name in backing_forwards:
    k8s_resource(name,
        port_forwards=[port_forward(p, host='0.0.0.0') for p in backing_forwards[name]],
        labels=['services'])

if not is_ci and api:
    local_resource('ui-deps',
        cmd='yarn --cwd ui install',
        deps=['ui/package.json', 'ui/yarn.lock'],
        labels=['ui'])
    local_resource('ui',
        serve_cmd='yarn --cwd ui start',
        deps=['ui/src'],
        resource_deps=['ui-deps', 'argo-server'],
        links=['http://localhost:8080'],
        labels=['ui'])
