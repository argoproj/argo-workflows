"""Native retryStrategy attempts and changed-parameter retry retain unrelated history."""
import json
import uuid

from e2e import NS, PREFIX, api, wait_workflow
from nested_retry_e2e import pod_uids
from template_refs_e2e import apply, native, resource

GROUP = 'parameters.workflows.example.io'


def fixture(name, steps):
    model = {'dialect': 'http://json-schema.org/draft-07/schema#', 'schema': {
        'type': 'object', 'properties': {'value': {'type': 'string'}},
        'required': ['value'], 'additionalProperties': False}}

    def call(name, value):
        return {'name': name, 'template': 'operation',
                'arguments': {'parameters': [{'name': 'value', 'value': value}]}}

    approve = {'name': 'approve', 'template': 'approve'}
    stable = call('stable', 'stable')
    change = call('change', '{{workflow.parameters.value}}')
    report = call('report', '{{steps.change.outputs.parameters.result}}' if steps else '{{tasks.change.outputs.parameters.result}}')
    if steps:
        root = {'name': 'main', 'steps': [[approve], [stable, change], [report]]}
    else:
        stable['dependencies'] = ['approve']
        change['dependencies'] = ['approve']
        report['dependencies'] = ['change']
        root = {'name': 'main', 'dag': {'failFast': False, 'tasks': [approve, stable, change, report]}}
    op = {'name': 'operation', 'inputs': {'parameters': [{'name': 'value'}]},
          'retryStrategy': {'limit': '1', 'retryPolicy': 'Always'},
          'metadata': {'annotations': {GROUP + '/retry-safe': 'idempotent', GROUP + '/result-reuse': 'same-inputs'}},
          'container': {'image': 'alpine:3.22', 'command': ['sh', '-ec'],
                        'args': ['test "$ATTEMPT" != 0; test "$VALUE" != bad; printf "%s" "$VALUE" > /tmp/result'],
                        'env': [{'name': 'ATTEMPT', 'value': '{{retries}}'}, {'name': 'VALUE', 'value': '{{inputs.parameters.value}}'}]},
          'outputs': {'parameters': [{'name': 'result', 'valueFrom': {'path': '/tmp/result'}}]}}
    spec = {'entrypoint': 'main', 'serviceAccountName': 'workflow-executor',
            'arguments': {'parameters': [{'name': 'value'}]},
            'templates': [root, {'name': 'approve', 'suspend': {}}, op]}
    return [resource('ParameterTemplate', name, model), native('WorkflowTemplate', name, spec),
            resource('WorkflowParameterBinding', name, {'workflowTemplate': name, 'parameterTemplate': name,
                                                       'mappings': [{'path': '/value', 'parameter': 'value', 'encoding': 'string'}]})]


def run(steps):
    name = 'retry-groups-' + ('steps-' if steps else 'dag-') + uuid.uuid4().hex[:8]
    for obj in fixture(name, steps):
        apply(obj)
    saved = api('PUT', PREFIX + '/sets/' + name, {'template': name, 'resourceVersion': '', 'values': {'value': 'bad'}})
    preview = api('POST', PREFIX + '/preview', {'binding': name, 'revision': saved['spec']['revision']})
    workflow = api('POST', PREFIX + '/submit', {'binding': name, 'revision': saved['spec']['revision'],
                                             'executionDigest': preview['executionDigest'], 'idempotencyKey': name})
    wf_name, uid = workflow['metadata']['name'], workflow['metadata']['uid']
    wait_workflow(wf_name, lambda w: any(n.get('type') == 'Suspend' and n.get('phase') == 'Running'
                                       for n in w.get('status', {}).get('nodes', {}).values()))
    api('PUT', '/api/v1/workflows/' + NS + '/' + wf_name + '/resume', {})
    failed = wait_workflow(wf_name, lambda w: w.get('status', {}).get('phase') in ('Failed', 'Error'))
    stable_id, stable = next((i, n) for i, n in failed['status']['nodes'].items()
                             if n['type'] == 'Retry' and n.get('displayName') == 'stable')
    assert stable['phase'] == 'Succeeded'
    assert len(stable['children']) == 2
    assert failed['status']['nodes'][stable['children'][0]]['phase'] == 'Failed'
    retained_ids = [stable_id, *stable['children'], *[i for i, n in failed['status']['nodes'].items() if n['type'] == 'Suspend']]
    keys = ['id', 'name', 'phase', 'startedAt', 'finishedAt', 'inputs', 'outputs', 'message']
    history = {i: {k: failed['status']['nodes'][i].get(k) for k in keys} for i in retained_ids}
    original_pods = pod_uids(wf_name)
    saved = api('PUT', PREFIX + '/sets/' + name, {'template': name, 'resourceVersion': saved['metadata']['resourceVersion'],
                                               'values': {'value': 'valid'}})
    request = {'workflow': wf_name, 'uid': uid, 'revision': saved['spec']['revision']}
    plan = api('POST', PREFIX + '/retry-preview', request)
    assert plan['affectedTasks'] == ['change', 'report'], plan
    assert set(retained_ids).issubset(plan['retainedNodes'])
    retry = {**request, 'resourceVersion': plan['resourceVersion'], 'planDigest': plan['digest'], 'idempotencyKey': name + '-new'}
    api('POST', PREFIX + '/retry', retry)
    done = wait_workflow(wf_name, lambda w: w.get('status', {}).get('phase') in ('Succeeded', 'Failed', 'Error'))
    assert done['status']['phase'] == 'Succeeded', done['status']
    assert done['metadata']['uid'] == uid
    after_pods = pod_uids(wf_name)
    for i in retained_ids:
        assert {k: done['status']['nodes'][i].get(k) for k in keys} == history[i]
        if i in original_pods:
            assert after_pods[i] == original_pods[i]
    changed = next(n for n in done['status']['nodes'].values() if n['type'] == 'Retry' and n.get('displayName') == 'change')
    assert changed['outputs']['parameters'][0]['value'] == 'valid'
    assert len(changed['children']) == 2
    assert all(after_pods[i] != original_pods[i] for i in changed['children'])
    assert api('POST', PREFIX + '/retry', retry)['metadata']['uid'] == uid
    receipt = api('GET', PREFIX + '/catalog/parameterretries?name=' + done['metadata']['annotations'][GROUP + '/retry'])
    assert receipt['spec']['previousWorkflow']['status']['nodes'][changed['id']]['phase'] == 'Failed'
    print('PASS', json.dumps({'steps': steps, 'workflow': wf_name, 'uid': uid, 'affected': plan['affectedTasks'],
                              'retainedFailedAttemptHistory': True, 'retainedPodUIDs': 2,
                              'approvalPreserved': True, 'changedRetryAttemptsRecreated': True}), flush=True)


if __name__ == '__main__':
    run(False)
    run(True)
