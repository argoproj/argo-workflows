"""Local native action -> Vault Agent -> real renewable PostgreSQL credential.

No controller-managed credential leases or external business services are used.
Only a reference to a synthetic, least-privilege bootstrap token enters Argo.
"""
import json
import os
import subprocess
import sys
import time
import uuid

from e2e import NS, PREFIX, api, wait_workflow
from secret_versions_e2e import G, fixture
from template_refs_e2e import CONTEXT, apply, kubectl, resource

INSTALL_DEMO = '--install-demo' in sys.argv
RUN = 'dynamic-credential-demo' if INSTALL_DEMO else 'leases-' + uuid.uuid4().hex[:8]
IMAGE = os.environ.get('ARGO_VAULT_TASK_IMAGE', 'argo-parameterization-vault-task:local-v52')
ROOT = 'synthetic-root-' + uuid.uuid4().hex
PASSWORD = 'synthetic-db-' + uuid.uuid4().hex


def wait_for(fn, seconds=90):
    end = time.monotonic() + seconds
    while time.monotonic() < end:
        value = fn()
        if value:
            return value
        time.sleep(2)
    raise AssertionError('bounded lease transition timed out')


def vault(*args, data=None, allow_empty=False):
    command = ['kubectl', '--context', CONTEXT, '-n', NS, 'exec']
    if data is not None:
        command.append('-i')
    command += ['deployment/' + RUN, '-c', 'vault', '--', 'vault', *args]
    result = subprocess.run(command, input=data, capture_output=True, timeout=25)
    # Vault errors may contain credential-related response fields. Do not echo them.
    if allow_empty and result.returncode == 2:
        return b'[]'
    if result.returncode:
        raise AssertionError('local Vault command failed: ' + args[0])
    return result.stdout


def leases():
    raw = vault('list', '-format=json', 'sys/leases/lookup/database/creds/demo', allow_empty=True)
    return set(json.loads(raw))


def pods(name):
    return json.loads(kubectl('-n', NS, 'get', 'pods', '-l',
                              'workflows.argoproj.io/workflow=' + name, '-o', 'json'))['items']


def logs(pod):
    return kubectl('-n', NS, 'logs', pod, '-c', 'main').decode()


def policy(allowed):
    rules = 'path "database/creds/demo" { capabilities = ["' + ('read' if allowed else 'deny') + '"] }'
    vault('policy', 'write', RUN, '-', data=rules.encode())


def setup():
    apply({'apiVersion': 'v1', 'kind': 'Secret', 'metadata': {'name': RUN + '-admin', 'namespace': NS},
           'immutable': True, 'stringData': {'token': ROOT, 'password': PASSWORD}})
    containers = [
        {'name': 'vault', 'image': 'hashicorp/vault:1.19.0',
         'args': ['server', '-dev', '-dev-listen-address=0.0.0.0:8200'],
         'env': [{'name': name, 'valueFrom': {'secretKeyRef': {'name': RUN + '-admin', 'key': 'token'}}}
                 for name in ['VAULT_DEV_ROOT_TOKEN_ID', 'VAULT_TOKEN']] +
                [{'name': 'VAULT_ADDR', 'value': 'http://127.0.0.1:8200'}],
         'readinessProbe': {'httpGet': {'path': '/v1/sys/health', 'port': 8200}, 'periodSeconds': 2}},
        {'name': 'postgres', 'image': IMAGE, 'imagePullPolicy': 'Never',
         'command': ['docker-entrypoint.sh', 'postgres'],
         'env': [{'name': 'POSTGRES_PASSWORD', 'valueFrom': {'secretKeyRef': {'name': RUN + '-admin', 'key': 'password'}}}],
         'readinessProbe': {'exec': {'command': ['pg_isready', '-U', 'postgres']}, 'periodSeconds': 2},
         'volumeMounts': [{'name': 'db', 'mountPath': '/var/lib/postgresql/data'}]}
    ]
    apply({'apiVersion': 'apps/v1', 'kind': 'Deployment', 'metadata': {'name': RUN, 'namespace': NS},
           'spec': {'selector': {'matchLabels': {'app': RUN}}, 'template': {
               'metadata': {'labels': {'app': RUN}}, 'spec': {'automountServiceAccountToken': False,
                   'containers': containers, 'volumes': [{'name': 'db', 'emptyDir': {}}]}}}})
    apply({'apiVersion': 'v1', 'kind': 'Service', 'metadata': {'name': RUN, 'namespace': NS},
           'spec': {'selector': {'app': RUN}, 'ports': [{'name': 'vault', 'port': 8200}, {'name': 'db', 'port': 5432}]}})
    kubectl('-n', NS, 'rollout', 'status', 'deployment/' + RUN, '--timeout=120s')
    vault('secrets', 'enable', 'database')
    connection = {'plugin_name': 'postgresql-database-plugin', 'allowed_roles': 'demo',
                  'connection_url': 'postgresql://{{username}}:{{password}}@127.0.0.1:5432/postgres?sslmode=disable',
                  'username': 'postgres', 'password': PASSWORD}
    vault('write', 'database/config/demo', '-', data=json.dumps(connection).encode())
    role = {'db_name': 'demo', 'default_ttl': '15s', 'max_ttl': '90s',
            'creation_statements': 'CREATE ROLE "{{name}}" WITH LOGIN PASSWORD \'{{password}}\' VALID UNTIL \'{{expiration}}\'; GRANT CONNECT ON DATABASE postgres TO "{{name}}";'}
    vault('write', 'database/roles/demo', '-', data=json.dumps(role).encode())
    policy(True)
    token = json.loads(vault('token', 'create', '-format=json', '-policy=' + RUN, '-ttl=' + ('24h' if INSTALL_DEMO else '10m')))['auth']['client_token']
    apply({'apiVersion': 'v1', 'kind': 'Secret', 'metadata': {'name': RUN, 'namespace': NS},
           'immutable': True, 'stringData': {'token': token}})
    return token


def definitions():
    objects = fixture(RUN)
    workflow = objects[1]
    config = '''auto_auth {
  method "token_file" { config = { token_file_path = "/scratch/token" } }
}
vault { address = "http://BACKEND:8200" retry { num_retries = 1 } }
template_config { exit_on_retry_failure = true }
env_template "PGUSER" {
  contents = "[[ with secret \\"database/creds/demo\\" ]][[ .Data.username ]][[ end ]]"
  left_delimiter = "[["
  right_delimiter = "]]"
  error_on_missing_key = true
}
env_template "PGPASSWORD" {
  contents = "[[ with secret \\"database/creds/demo\\" ]][[ .Data.password ]][[ end ]]"
  left_delimiter = "[["
  right_delimiter = "]]"
  error_on_missing_key = true
}
env_template "LEASE_ID" {
  contents = "[[ with secret \\"database/creds/demo\\" ]][[ .LeaseID ]][[ end ]]"
  left_delimiter = "[["
  right_delimiter = "]]"
  error_on_missing_key = true
}
exec {
  command = ["/bin/sh", "-ec", "printf %s \\"$LEASE_ID\\" > /scratch/lease-id; unset LEASE_ID; exec /bin/sh /demo/query.sh"]
  restart_on_secret_changes = "never"
}
'''.replace('BACKEND', RUN)
    consume = workflow['spec']['templates'][-1]
    consume['activeDeadlineSeconds'] = 75
    consume['outputs'] = {'parameters': [{'name': 'leaseId', 'valueFrom': {'path': '/scratch/lease-id', 'default': 'not-issued'}}]}
    consume['metadata']['annotations'][G + '/dynamic-credential-policy'] = 'vault-agent/per-attempt; renewable; no-process-restart; ttl-bound-cleanup'
    consume['container'] = {
        'image': IMAGE, 'imagePullPolicy': 'Never', 'command': ['sh', '-ec'],
        'args': ['umask 077\nprintf %s "$BOOTSTRAP" > /scratch/token\nunset BOOTSTRAP\ncat > /scratch/agent.hcl <<\'CONFIG\'\n' +
                 config + 'CONFIG\nexec timeout -s TERM 65 vault agent -config=/scratch/agent.hcl -log-level=error'],
        'env': [{'name': 'BOOTSTRAP', 'valueFrom': {'secretKeyRef': {
            'name': "{{=jsonpath(inputs.parameters.credential, '$.secretRef.name')}}",
            'key': "{{=jsonpath(inputs.parameters.credential, '$.secretRef.key')}}"}}},
            {'name': 'PGHOST', 'value': RUN}, {'name': 'PGDATABASE', 'value': 'postgres'},
            {'name': 'PGCONNECT_TIMEOUT', 'value': '3'}, {'name': 'PGSSLMODE', 'value': 'disable'}],
        'volumeMounts': [{'name': 'scratch', 'mountPath': '/scratch'}]}
    workflow['spec']['volumes'] = [{'name': 'scratch', 'emptyDir': {'medium': 'Memory'}}]
    objects.append(resource('ActionDefinition', RUN, {
        'inputSchema': {'type': 'object', 'properties': {'credential': objects[0]['spec']['schema']['properties']['credential']},
                        'required': ['credential'], 'additionalProperties': False},
        'outputSchema': {'type': 'object', 'properties': {'leaseId': {'type': 'string', 'minLength': 1}},
                         'required': ['leaseId'], 'additionalProperties': False},
        'template': consume}))
    workflow['spec']['templates'][-1] = {'name': 'consume', 'suspend': {}}
    objects.append(resource('WorkflowActionBinding', RUN, {
        'workflowTemplate': RUN, 'calls': [{'template': 'consume', 'actionDefinition': RUN}]}))
    objects[2]['spec']['actionBinding'] = RUN
    return objects


def main():
    token = setup()
    if INSTALL_DEMO:
        for obj in definitions():
            apply(obj)
        print('Installed dynamic-credential-demo; choose Secret reference dynamic-credential-demo/token. No Workflow submitted.', flush=True)
        return
    try:
        for obj in definitions():
            try:
                apply(obj)
            except subprocess.CalledProcessError as error:
                raise AssertionError(obj['kind'] + ': ' + error.stderr.decode()) from None
        saved = api('PUT', PREFIX + '/sets/' + RUN, {'template': RUN, 'resourceVersion': '',
                    'values': {'credential': {'secretRef': {'name': RUN, 'key': 'token'}}}})
        preview = api('POST', PREFIX + '/preview', {'binding': RUN, 'revision': saved['spec']['revision']})
        assert not preview['errors'], preview['errors']
        workflow = api('POST', PREFIX + '/submit', {'binding': RUN, 'revision': saved['spec']['revision'],
                       'executionDigest': preview['executionDigest'], 'idempotencyKey': RUN})
        name, uid = workflow['metadata']['name'], workflow['metadata']['uid']
        wait_workflow(name, lambda w: any(n.get('type') == 'Suspend' and n.get('phase') == 'Running'
                                        for n in w.get('status', {}).get('nodes', {}).values()))
        api('PUT', '/api/v1/workflows/' + NS + '/' + name + '/resume', {})
        pod = wait_for(lambda: next((p['metadata']['name'] for p in pods(name)
                                    if p.get('status', {}).get('phase') == 'Running'), None))
        wait_for(lambda: 'credential-query-ok' in logs(pod))
        # Capture synthetic child environment only in test memory; never print it.
        child_env = kubectl('-n', NS, 'exec', pod, '-c', 'main', '--', 'sh', '-ec',
                            'pid=$(pgrep -f "^/bin/sh /demo/query.sh$"); cat /proc/$pid/environ')
        issued = dict(v.split(b'=', 1) for v in child_env.split(b'\x00') if b'=' in v)
        assert b'BOOTSTRAP' not in issued and b'VAULT_TOKEN' not in issued
        active = leases()
        assert len(active) == 1, 'Agent must share one lease between username/password templates'
        lease = 'database/creds/demo/' + next(iter(active))
        initial = json.loads(vault('lease', 'lookup', '-format=json', lease))['data']
        wait_for(lambda: logs(pod).count('credential-query-ok') >= 11, 35)
        renewed = json.loads(vault('lease', 'lookup', '-format=json', lease))['data']
        assert renewed['last_renewal'] != initial['last_renewal'], 'real lease was not renewed'
        vault('lease', 'revoke', lease)
        failed = wait_workflow(name, lambda w: w.get('status', {}).get('phase') in ('Failed', 'Error'))
        assert 'credential-query-rejected' in logs(pod)
        approval_id, approval = next((i, n) for i, n in failed['status']['nodes'].items() if n['type'] == 'Suspend')
        # Denial must not silently switch to another identity or start the SQL child.
        policy(False)
        previous_pods = {p['metadata']['uid'] for p in pods(name)}
        api('POST', PREFIX + '/retry', {'workflow': name, 'uid': uid,
            'resourceVersion': failed['metadata']['resourceVersion'], 'idempotencyKey': RUN + '-denied'})
        blocked = wait_workflow(name, lambda w: w.get('status', {}).get('phase') in ('Failed', 'Error'))
        rejected_pod = next(p for p in pods(name) if p['metadata']['uid'] not in previous_pods)
        denied_logs = logs(rejected_pod['metadata']['name'])
        assert 'credential-query-ok' not in denied_logs and 'credential-task-complete' not in denied_logs
        policy(True)
        previous_leases = leases()
        api('POST', PREFIX + '/retry', {'workflow': name, 'uid': uid,
            'resourceVersion': blocked['metadata']['resourceVersion'], 'idempotencyKey': RUN + '-restored'})
        restored_leases = wait_for(lambda: leases() - previous_leases)
        assert next(iter(active)) not in restored_leases, 'retry reused the revoked credential lease'
        done = wait_workflow(name, lambda w: w.get('status', {}).get('phase') == 'Succeeded')
        assert done['metadata']['uid'] == uid
        assert all(done['status']['nodes'][approval_id].get(k) == approval.get(k)
                   for k in ['id', 'phase', 'startedAt', 'finishedAt', 'inputs', 'outputs'])
        actual_lease = next(p['value'] for n in done['status']['nodes'].values() if n['type'] == 'Pod'
                            for p in n.get('outputs', {}).get('parameters', []) if p['name'] == 'leaseId')
        assert actual_lease.split('/')[-1] in restored_leases
        wait_for(lambda: not leases(), 35)
        payloads = json.dumps([saved, preview, done, api('GET', PREFIX + '/history/' + RUN)])
        combined_logs = '\n'.join(logs(p['metadata']['name']) for p in pods(name))
        for value in [token, ROOT, PASSWORD, issued[b'PGUSER'].decode(), issued[b'PGPASSWORD'].decode()]:
            assert value not in payloads + combined_logs, 'credential leaked'
        assert 'PGPASSWORD=' not in combined_logs
        # Native cancellation stops the Agent; outstanding leases expire under the declared TTL.
        cancelled = api('POST', PREFIX + '/submit', {'binding': RUN, 'revision': saved['spec']['revision'],
            'executionDigest': preview['executionDigest'], 'idempotencyKey': RUN + '-cancel'})
        cancel_name = cancelled['metadata']['name']
        wait_workflow(cancel_name, lambda w: any(n.get('type') == 'Suspend' and n.get('phase') == 'Running'
                      for n in w.get('status', {}).get('nodes', {}).values()))
        api('PUT', '/api/v1/workflows/' + NS + '/' + cancel_name + '/resume', {})
        wait_for(leases)
        api('PUT', '/api/v1/workflows/' + NS + '/' + cancel_name + '/stop', {'message': 'local lease cancellation test'})
        wait_workflow(cancel_name, lambda w: w.get('status', {}).get('phase') in ('Failed', 'Error'))
        wait_for(lambda: not leases(), 35)
        print('PASS', json.dumps({'workflow': name, 'uid': uid, 'leaseRenewed': True,
            'revocationFailedClosed': True, 'denialStartedNoChild': True,
            'retryObtainedNewLease': True, 'approvalRetained': True,
            'publicLeaseIdentityRecorded': True, 'completionAndCancellationExpireLeases': True}), flush=True)
    finally:
        kubectl('-n', NS, 'delete', 'deployment,service', RUN, '--ignore-not-found=true')
        kubectl('-n', NS, 'delete', 'secret', RUN, RUN + '-admin', '--ignore-not-found=true')


if __name__ == '__main__':
    main()
