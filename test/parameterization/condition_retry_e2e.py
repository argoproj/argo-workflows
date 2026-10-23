"""Retry rebinds producer outputs and consumer conditions in nested DAG/Steps."""
import json
import uuid

from e2e import api, wait_workflow, PREFIX, NS
from template_refs_e2e import resource, native, apply, kubectl
from nested_retry_e2e import pod_uids

G = 'parameters.workflows.example.io'
RUN = 'condition-retry-' + uuid.uuid4().hex[:8]


def fixture(name, steps):
    scope = 'steps' if steps else 'tasks'
    contract = {G + '/result-reuse': 'same-inputs', G + '/retry-safe': 'idempotent',
                G + '/retry-dependencies': 'inputs'}
    args = lambda **values: {'parameters': [{'name': k, 'value': v} for k, v in values.items()]}
    producer = {
        'name': 'producer', 'metadata': {'annotations': contract},
        'inputs': {'parameters': [{'name': 'seed'}]},
        'container': {'image': 'alpine:3.22', 'command': ['sh', '-ec'],
                      'args': ['printf "%s-%s" "$SEED" "$(cat /source/value)" >/tmp/value'],
                      'env': [{'name': 'SEED', 'value': '{{inputs.parameters.seed}}'}],
                      'volumeMounts': [{'name': 'source', 'mountPath': '/source', 'readOnly': True}]},
        'outputs': {'parameters': [{'name': 'value', 'valueFrom': {'path': '/tmp/value'}}]}}
    consumer = {
        'name': 'consumer', 'metadata': {'annotations': contract},
        'inputs': {'parameters': [{'name': 'value'}, {'name': 'expected'}]},
        'container': {'image': 'alpine:3.22', 'command': ['sh', '-ec'],
                      'args': ['test "$VALUE" = "$EXPECTED"; printf "%s" "$VALUE" >/tmp/value'],
                      'env': [{'name': 'VALUE', 'value': '{{inputs.parameters.value}}'},
                              {'name': 'EXPECTED', 'value': '{{inputs.parameters.expected}}'}]},
        'outputs': {'parameters': [{'name': 'value', 'valueFrom': {'path': '/tmp/value'}}]}}
    calls = [
        {'name': 'producer', 'template': 'producer', 'arguments': args(seed='{{inputs.parameters.seed}}')},
        {'name': 'consumer', 'template': 'consumer',
         'when': "{{=" + scope + ".producer.outputs.parameters.value == inputs.parameters.expected}}",
         'arguments': args(value='{{' + scope + '.producer.outputs.parameters.value}}',
                           expected='{{inputs.parameters.expected}}')}]
    group = {'name': 'group', 'inputs': {'parameters': [{'name': 'seed'}, {'name': 'expected'}]}}
    outer = [
        {'name': 'approve', 'template': 'approve'},
        {'name': 'stable', 'template': 'producer', 'arguments': args(seed='stable')},
        {'name': 'vary', 'template': 'group', 'when': '{{workflow.parameters.enabled}} == true',
         'arguments': args(seed='{{workflow.parameters.seed}}', expected='{{workflow.parameters.expected}}')}]
    root = {'name': 'main'}
    if steps:
        group['steps'] = [[c] for c in calls]
        root['steps'] = [[c] for c in outer]
    else:
        calls[1]['dependencies'] = ['producer']
        group['dag'] = {'tasks': calls}
        outer[1]['dependencies'] = ['approve']
        outer[2]['dependencies'] = ['stable']
        root['dag'] = {'tasks': outer}
    schema = {'dialect': 'http://json-schema.org/draft-07/schema#', 'schema': {
        'type': 'object', 'additionalProperties': False, 'required': ['seed', 'expected', 'enabled'],
        'properties': {'seed': {'type': 'string'}, 'expected': {'type': 'string'}, 'enabled': {'type': 'boolean'}}}}
    spec = {'entrypoint': 'main', 'serviceAccountName': 'workflow-executor',
            'arguments': args(seed='', expected='', enabled='true'),
            'volumes': [{'name': 'source', 'configMap': {'name': name}}],
            'templates': [root, {'name': 'approve', 'suspend': {}}, group, producer, consumer]}
    return [resource('ParameterTemplate', name, schema), native('WorkflowTemplate', name, spec),
            resource('WorkflowParameterBinding', name, {'workflowTemplate': name, 'parameterTemplate': name,
                'mappings': [{'path': '/' + key, 'parameter': key, 'encoding': encoding}
                             for key, encoding in [('seed', 'string'), ('expected', 'string'), ('enabled', 'json')]]})]


def source(name, value):
    apply({'apiVersion': 'v1', 'kind': 'ConfigMap', 'metadata': {'name': name, 'namespace': NS},
           'data': {'value': value}})


def main():
    evidence = []
    for steps in [False, True]:
        key = RUN + ('-steps' if steps else '-dag')
        source(key, '1')
        for obj in fixture(key, steps):
            apply(obj)
        values = {'seed': 'old', 'expected': 'old-1', 'enabled': True}
        saved = api('PUT', PREFIX + '/sets/' + key, {'template': key, 'values': values, 'resourceVersion': ''})
        preview = api('POST', PREFIX + '/preview', {'binding': key, 'revision': saved['spec']['revision']})
        assert not preview['errors'], preview
        wf = api('POST', PREFIX + '/submit', {'binding': key, 'revision': saved['spec']['revision'],
                 'executionDigest': preview['executionDigest'], 'idempotencyKey': key})
        name, uid = wf['metadata']['name'], wf['metadata']['uid']
        wait_workflow(name, lambda w: any(n.get('type') == 'Suspend' and n.get('phase') == 'Running'
                                         for n in w.get('status', {}).get('nodes', {}).values()))
        api('PUT', '/api/v1/workflows/' + NS + '/' + name + '/resume', {})
        done = wait_workflow(name, lambda w: w.get('status', {}).get('phase') == 'Succeeded')
        preserved = {i: n for i, n in done['status']['nodes'].items() if n.get('displayName') in ['approve', 'stable']}
        original_pods = pod_uids(name)
        observed = []
        attempts = [
            ({'expected': 'other'}, None, ['vary/consumer'], 'Skipped'),
            ({'seed': 'new', 'expected': 'new-1'}, None, ['vary/consumer', 'vary/producer'], 'Succeeded'),
            ({}, '2', ['vary/consumer', 'vary/producer'], 'Skipped'),
            ({'expected': 'new-2'}, None, ['vary/consumer'], 'Succeeded'),
            ({'enabled': False}, None, ['vary/consumer', 'vary/producer'], None),
            ({'enabled': True}, None, ['vary/consumer', 'vary/producer'], 'Succeeded'),
        ]
        for attempt, (changes, remote, affected, phase) in enumerate(attempts):
            previous = done
            previous_pods = pod_uids(name)
            req = {'workflow': name, 'uid': uid}
            if remote:
                source(key, remote)
                req['callPaths'] = ['vary/producer']
            else:
                values = {**values, **changes}
                saved = api('PUT', PREFIX + '/sets/' + key,
                            {'template': key, 'values': values, 'resourceVersion': saved['metadata']['resourceVersion']})
                req['revision'] = saved['spec']['revision']
            plan = api('POST', PREFIX + '/retry-preview', req)
            assert plan['affectedTasks'] == affected, plan
            retry = {**req, 'resourceVersion': plan['resourceVersion'], 'planDigest': plan['digest'],
                     'idempotencyKey': key + '-' + str(attempt)}
            api('POST', PREFIX + '/retry', retry)
            done = wait_workflow(name, lambda w: w.get('status', {}).get('phase') == 'Succeeded')
            assert done['metadata']['uid'] == uid
            for i, old in preserved.items():
                assert all(done['status']['nodes'][i].get(f) == old.get(f)
                           for f in ['phase', 'startedAt', 'finishedAt', 'inputs', 'outputs'])
                if i in original_pods:
                    assert pod_uids(name)[i] == original_pods[i]
            if phase:
                consumer = next(n for n in done['status']['nodes'].values() if n.get('displayName') == 'consumer')
                assert consumer['phase'] == phase, consumer
                if phase == 'Succeeded':
                    assert consumer['outputs']['parameters'][0]['value'] == values['expected'], consumer
                if affected == ['vary/consumer']:
                    producer_id = next(i for i, n in previous['status']['nodes'].items()
                                       if n.get('displayName') == 'producer')
                    assert pod_uids(name)[producer_id] == previous_pods[producer_id]
            else:
                varying = next(n for n in done['status']['nodes'].values() if n.get('displayName') == 'vary')
                assert varying['phase'] == 'Skipped', varying
            assert api('POST', PREFIX + '/retry', retry)['metadata']['uid'] == uid
            observed.append({'affected': affected, 'consumer': phase or 'nested invocation skipped'})
        workflows = json.loads(kubectl('-n', NS, 'get', 'workflows', '-l',
                                      'workflows.argoproj.io/workflow-template=' + key, '-o', 'json'))['items']
        assert len(workflows) == 1
        evidence.append({'workflow': name, 'uid': uid, 'steps': steps, 'attempts': observed,
                         'preservedApprovalAndPod': True, 'workflowCount': 1})
    print('PASS', json.dumps(evidence), flush=True)


if __name__ == '__main__':
    main()
