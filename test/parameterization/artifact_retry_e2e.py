"""Changed-parameter retry reads and verifies actual retained artifact storage."""
import hashlib
import json
import uuid

from e2e import NS, PREFIX, api, wait_workflow
from template_refs_e2e import apply, kubectl, native, resource
from nested_retry_e2e import pod_uids
from artifact_calls_e2e import SERVER

G = 'parameters.workflows.example.io'
RUN = 'artifact-retry-' + uuid.uuid4().hex[:8]
CONTENT = 'original-artifact-content'
SERVER = SERVER.replace(' def do_GET(self):', ' def do_DELETE(self):\n  files.pop(self.path,None);self.send_response(204);self.end_headers()\n def do_GET(self):')


def fixture(name, steps):
    approve = {'name': 'approve', 'template': 'approve'}
    producer = {'name': 'producer', 'template': 'producer'}
    consume = {'name': 'consume', 'template': 'consume', 'arguments': {
        'parameters': [{'name': 'mode', 'value': '{{workflow.parameters.mode}}'}],
        'artifacts': [{'name': 'file', 'from': '{{' + ('steps' if steps else 'tasks') + '.producer.outputs.artifacts.file}}'}]}}
    if steps:
        main = {'name': 'main', 'steps': [[approve], [producer], [consume]]}
    else:
        producer['dependencies'] = ['approve']
        consume['dependencies'] = ['producer']
        main = {'name': 'main', 'dag': {'tasks': [approve, producer, consume]}}
    source = {'name': 'producer', 'metadata': {'annotations': {
        G + '/result-reuse': 'same-inputs', G + '/artifact-checksums': '{"file":"checksum"}'}},
        'container': {'image': 'alpine:3.22', 'command': ['sh', '-ec'],
                      'args': ['printf "%s" "$CONTENT" > /tmp/file; sha256sum /tmp/file | cut -d" " -f1 > /tmp/checksum'],
                      'env': [{'name': 'CONTENT', 'value': CONTENT}]},
        'outputs': {'parameters': [{'name': 'checksum', 'valueFrom': {'path': '/tmp/checksum'}}],
                    'artifacts': [{'name': 'file', 'path': '/tmp/file', 'archive': {'none': {}},
                                   'http': {'url': 'http://' + RUN + ':8080/{{workflow.name}}/file'}}]}}
    consumer = {'name': 'consume', 'metadata': {'annotations': {G + '/retry-safe': 'idempotent'}},
                'inputs': {'parameters': [{'name': 'mode'}], 'artifacts': [{'name': 'file', 'path': '/tmp/file'}]},
                'container': {'image': 'alpine:3.22', 'command': ['sh', '-ec'],
                              'args': ['test "$MODE" = good; test "$(cat /tmp/file)" = "$CONTENT"; sha256sum /tmp/file | cut -d" " -f1 > /tmp/digest'],
                              'env': [{'name': 'MODE', 'value': '{{inputs.parameters.mode}}'}, {'name': 'CONTENT', 'value': CONTENT}]},
                'outputs': {'parameters': [{'name': 'digest', 'valueFrom': {'path': '/tmp/digest'}}]}}
    return [resource('ParameterTemplate', name, {'dialect': 'http://json-schema.org/draft-07/schema#', 'schema': {
        'type': 'object', 'properties': {'mode': {'type': 'string'}}, 'required': ['mode'], 'additionalProperties': False}}),
            native('WorkflowTemplate', name, {'entrypoint': 'main', 'serviceAccountName': 'workflow-executor',
                                             'arguments': {'parameters': [{'name': 'mode'}]},
                                             'templates': [main, {'name': 'approve', 'suspend': {}}, source, consumer]}),
            resource('WorkflowParameterBinding', name, {'workflowTemplate': name, 'parameterTemplate': name,
                                                       'mappings': [{'path': '/mode', 'parameter': 'mode', 'encoding': 'string'}]})]


def change_file(workflow, data):
    source = 'import urllib.request;urllib.request.urlopen(urllib.request.Request(' + repr('http://127.0.0.1:8080/' + workflow + '/file') + ',data=' + repr(data.encode() if data is not None else None) + ',method=' + repr('PUT' if data is not None else 'DELETE') + ')).close()'
    kubectl('-n', NS, 'exec', 'deployment/' + RUN, '--', 'python', '-c', source)


def run(steps):
    name = RUN + ('-steps' if steps else '-dag')
    for obj in fixture(name, steps):
        apply(obj)
    saved = api('PUT', PREFIX + '/sets/' + name, {'template': name, 'values': {'mode': 'bad'}, 'resourceVersion': ''})
    preview = api('POST', PREFIX + '/preview', {'binding': name, 'revision': saved['spec']['revision']})
    assert not preview['errors'], preview
    wf = api('POST', PREFIX + '/submit', {'binding': name, 'revision': saved['spec']['revision'],
                                        'executionDigest': preview['executionDigest'], 'idempotencyKey': name})
    workflow, uid = wf['metadata']['name'], wf['metadata']['uid']
    wait_workflow(workflow, lambda w: any(n.get('type') == 'Suspend' and n.get('phase') == 'Running' for n in w.get('status', {}).get('nodes', {}).values()))
    api('PUT', '/api/v1/workflows/' + NS + '/' + workflow + '/resume', {})
    failed = wait_workflow(workflow, lambda w: w.get('status', {}).get('phase') in ('Failed', 'Error'))
    producer_id, producer = next((i, n) for i, n in failed['status']['nodes'].items() if n.get('displayName') == 'producer')
    assert producer['phase'] == 'Succeeded', producer
    approval_id, approval = next((i, n) for i, n in failed['status']['nodes'].items() if n['type'] == 'Suspend')
    original_pods = pod_uids(workflow)
    saved = api('PUT', PREFIX + '/sets/' + name, {'template': name, 'values': {'mode': 'good'}, 'resourceVersion': saved['metadata']['resourceVersion']})
    request = {'workflow': workflow, 'uid': uid, 'revision': saved['spec']['revision']}
    plan = api('POST', PREFIX + '/retry-preview', request)
    assert plan['affectedTasks'] == ['consume'], plan
    retry = {**request, 'resourceVersion': plan['resourceVersion'], 'planDigest': plan['digest'], 'idempotencyKey': name + '-retry'}
    change_file(workflow, 'tampered file')
    assert 'content changed' in json.dumps(api('POST', PREFIX + '/retry', retry, 400))
    assert 'content changed' in json.dumps(api('POST', PREFIX + '/retry-preview', request, 400))
    unchanged = api('GET', '/api/v1/workflows/' + NS + '/' + workflow)
    assert unchanged['metadata']['resourceVersion'] == plan['resourceVersion']
    assert unchanged['status']['nodes'] == failed['status']['nodes']
    change_file(workflow, None)
    assert 'unavailable' in json.dumps(api('POST', PREFIX + '/retry-preview', request, 400))
    change_file(workflow, CONTENT)
    plan = api('POST', PREFIX + '/retry-preview', request)
    api('POST', PREFIX + '/retry', {**retry, 'resourceVersion': plan['resourceVersion'], 'planDigest': plan['digest']})
    done = wait_workflow(workflow, lambda w: w.get('status', {}).get('phase') in ('Succeeded', 'Failed', 'Error'))
    assert done['status']['phase'] == 'Succeeded', done['status']
    assert done['metadata']['uid'] == uid
    assert pod_uids(workflow)[producer_id] == original_pods[producer_id]
    for node_id, before in [(producer_id, producer), (approval_id, approval)]:
        for key in ['phase', 'startedAt', 'finishedAt', 'inputs', 'outputs']:
            assert done['status']['nodes'][node_id].get(key) == before.get(key)
    consumer = next(n for n in done['status']['nodes'].values() if n.get('displayName') == 'consume')
    assert consumer['outputs']['parameters'][0]['value'] == hashlib.sha256(CONTENT.encode()).hexdigest()
    print('PASS', json.dumps({'steps': steps, 'workflow': workflow, 'uid': uid,
                              'tamperedArtifactRejectedBeforeRetry': True, 'producerPodRetained': True,
                              'approvalRetained': True, 'downstreamDownloadedOriginalFile': True}), flush=True)


def main():
    apply({'apiVersion': 'apps/v1', 'kind': 'Deployment', 'metadata': {'name': RUN, 'namespace': NS},
           'spec': {'selector': {'matchLabels': {'app': RUN}}, 'template': {'metadata': {'labels': {'app': RUN}},
                    'spec': {'automountServiceAccountToken': False, 'containers': [{'name': 'store', 'image': 'python:3.12-alpine', 'command': ['python', '-u', '-c', SERVER]}]}}}})
    apply({'apiVersion': 'v1', 'kind': 'Service', 'metadata': {'name': RUN, 'namespace': NS},
           'spec': {'selector': {'app': RUN}, 'ports': [{'port': 8080, 'targetPort': 8080}]}})
    try:
        kubectl('-n', NS, 'rollout', 'status', 'deployment/' + RUN, '--timeout=60s')
        run(False)
        run(True)
    finally:
        kubectl('-n', NS, 'delete', 'deployment,service', RUN, '--ignore-not-found=true')


if __name__ == '__main__':
    main()
