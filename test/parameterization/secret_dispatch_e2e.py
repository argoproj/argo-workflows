"""Waiting -> expired external verification -> no Pod -> refresh -> same Workflow retry."""
import datetime
import json
import uuid

from e2e import NS, PREFIX, api, wait_workflow
from secret_versions_e2e import fixture
from template_refs_e2e import apply, kubectl

RUN = 'secret-dispatch-' + uuid.uuid4().hex[:8]
GROUP = 'parameters.workflows.example.io'


def main():
    for obj in fixture(RUN):
        apply(obj)
    value = 'synthetic-' + uuid.uuid4().hex

    def renew(hours):
        until = (datetime.datetime.now(datetime.timezone.utc) + datetime.timedelta(hours=hours)).isoformat()
        apply({'apiVersion': 'v1', 'kind': 'Secret',
               'metadata': {'name': RUN, 'namespace': NS,
                            'labels': {GROUP + '/external-source': 'synthetic-expiry-test'},
                            'annotations': {GROUP + '/external-valid-until': until}},
               'immutable': True, 'stringData': {'token': value}})

    renew(1)
    secret_uid = json.loads(kubectl('-n', NS, 'get', 'secret', RUN, '-o', 'json'))['metadata']['uid']
    saved = api('PUT', PREFIX + '/sets/' + RUN, {
        'template': RUN, 'resourceVersion': '',
        'values': {'credential': {'secretRef': {'name': RUN, 'key': 'token'}}}})
    preview = api('POST', PREFIX + '/preview', {'binding': RUN, 'revision': saved['spec']['revision']})
    workflow = api('POST', PREFIX + '/submit', {
        'binding': RUN, 'revision': saved['spec']['revision'],
        'executionDigest': preview['executionDigest'], 'idempotencyKey': RUN})
    name, uid = workflow['metadata']['name'], workflow['metadata']['uid']
    wait_workflow(name, lambda w: any(n.get('type') == 'Suspend' and n.get('phase') == 'Running'
                                    for n in w.get('status', {}).get('nodes', {}).values()))
    # Controlled validity expiry while native approval waits; no real provider disruption.
    renew(-1)
    api('PUT', '/api/v1/workflows/' + NS + '/' + name + '/resume', {})
    failed = wait_workflow(name, lambda w: w.get('status', {}).get('phase') in ('Failed', 'Error', 'Succeeded'))
    assert 'verification expired' in json.dumps(failed['status']), failed['status']
    pods = json.loads(kubectl('-n', NS, 'get', 'pods', '-l', 'workflows.argoproj.io/workflow=' + name, '-o', 'json'))
    assert not pods['items'], 'Expired credential must be rejected before creating an execution Pod'
    approval_id, approval = next((i, n) for i, n in failed['status']['nodes'].items() if n['type'] == 'Suspend')
    assert approval['phase'] == 'Succeeded'
    retry = {'workflow': name, 'uid': uid, 'resourceVersion': failed['metadata']['resourceVersion'], 'idempotencyKey': RUN + '-retry'}
    denied = api('POST', PREFIX + '/retry', retry, 400)
    assert 'expired' in json.dumps(denied)
    renew(1)
    assert json.loads(kubectl('-n', NS, 'get', 'secret', RUN, '-o', 'json'))['metadata']['uid'] == secret_uid
    failed = api('GET', '/api/v1/workflows/' + NS + '/' + name)
    retry['resourceVersion'] = failed['metadata']['resourceVersion']
    api('POST', PREFIX + '/retry', retry)
    done = wait_workflow(name, lambda w: w.get('status', {}).get('phase') == 'Succeeded')
    assert done['metadata']['uid'] == uid
    keys = ['id', 'phase', 'startedAt', 'finishedAt', 'inputs', 'outputs', 'message']
    assert {k: done['status']['nodes'][approval_id].get(k) for k in keys} == {k: approval.get(k) for k in keys}
    pods = json.loads(kubectl('-n', NS, 'get', 'pods', '-l', 'workflows.argoproj.io/workflow=' + name, '-o', 'json'))['items']
    assert len(pods) == 1
    main = next(c for c in pods[0]['spec']['containers'] if c['name'] == 'main')
    assert next(e for e in main['env'] if e['name'] == 'TOKEN')['valueFrom']['secretKeyRef']['name'] == RUN
    logs = kubectl('-n', NS, 'logs', pods[0]['metadata']['name'], '-c', 'main').decode()
    assert 'credential-consumed' in logs
    assert value not in json.dumps([preview, done, saved, denied, logs])
    receipt = api('GET', PREFIX + '/catalog/parameterretries?name=' + done['metadata']['annotations'][GROUP + '/retry'])
    assert value not in json.dumps(receipt)
    # A later expiry does not retroactively invalidate the successful Workflow.
    renew(-1)
    same = api('GET', '/api/v1/workflows/' + NS + '/' + name)
    assert same['status']['phase'] == 'Succeeded'
    assert api('POST', PREFIX + '/retry', retry)['metadata']['uid'] == uid
    print('PASS', json.dumps({'workflow': name, 'uid': uid, 'expiredDispatchCreatedNoPod': True,
                              'retryWithoutRefreshRejected': True, 'sameImmutableSecretRenewed': True,
                              'approvalPreserved': True, 'nativeExecutionSucceeded': True}), flush=True)


if __name__ == '__main__':
    main()
