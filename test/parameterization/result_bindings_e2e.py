"""Local full path: per-call result CRs -> native actions -> read-only views -> retries.

Two call sites share one action and use independent result templates. A third
native Resource action delegates a query to an example controller. No production
provider, business SDK or HTTP query implementation lives in Argo Server.
"""
import copy, json, os, subprocess, time, uuid
from pathlib import Path
from e2e import api, wait_workflow, PREFIX, NS, BASE
from template_refs_e2e import resource, native, apply, kubectl
from action_calls_e2e import fixture
from nested_retry_e2e import pod_uids

RUN = 'results-' + uuid.uuid4().hex[:8]
G = 'parameters.workflows.example.io'

def result_spec(title, view='summary'):
    return {'title': title, 'source': {'parameter': 'payload', 'format': 'json'},
            'schema': {'type': 'object', 'required': ['status', 'target'], 'properties': {'status': {'type': 'string'}, 'target': {'type': 'string'}}},
            'views': [{'title': title, 'type': view, 'columns': [{'title': 'Target', 'path': '/target'}, {'title': 'State', 'path': '/status', 'format': 'status'}],
                       **({'childrenPath': '/children', 'path': '/children'} if view == 'tree' else {})}]}

def result_url(wf, node, history=''):
    return PREFIX + '/node-results/' + wf['metadata']['name'] + '?uid=' + wf['metadata']['uid'] + '&nodeId=' + node['id'] + ('&history=' + history if history else '')

def install_observer():
    subprocess.run(['kubectl', '--context', 'k3d-argo-parameterization', 'apply', '-f', 'examples/parameterization/result-observer/crd.yaml'], check=True, capture_output=True)
    apply({'apiVersion': 'v1', 'kind': 'ConfigMap', 'metadata': {'name': RUN, 'namespace': NS}, 'data': {'adapter.py': Path('examples/parameterization/result-observer/adapter.py').read_text()}})
    apply({'apiVersion': 'v1', 'kind': 'ServiceAccount', 'metadata': {'name': RUN, 'namespace': NS}})
    apply({'apiVersion': 'rbac.authorization.k8s.io/v1', 'kind': 'Role', 'metadata': {'name': RUN, 'namespace': NS}, 'rules': [
        {'apiGroups': ['results.example.io'], 'resources': ['resultqueries'], 'verbs': ['get', 'list', 'watch', 'create']},
        {'apiGroups': ['results.example.io'], 'resources': ['resultqueries/status'], 'verbs': ['patch']}]})
    apply({'apiVersion': 'rbac.authorization.k8s.io/v1', 'kind': 'RoleBinding', 'metadata': {'name': RUN, 'namespace': NS},
           'roleRef': {'apiGroup': 'rbac.authorization.k8s.io', 'kind': 'Role', 'name': RUN},
           'subjects': [{'kind': 'ServiceAccount', 'name': n, 'namespace': NS} for n in [RUN, 'workflow-executor', 'argo-server']]})
    containers = []
    for name in ['provider', 'controller']:
        containers.append({'name': name, 'image': 'python:3.12-alpine', 'command': ['python', '/app/adapter.py'],
                           'env': [{'name': 'MODE', 'value': name}, {'name': 'NAMESPACE', 'value': NS}, {'name': 'QUERY_SELECTOR', 'value': 'result-demo=' + RUN}, {'name': 'PROVIDER_URL', 'value': 'http://127.0.0.1:8080'}],
                           'volumeMounts': [{'name': 'app', 'mountPath': '/app', 'readOnly': True}],
                           'resources': {'requests': {'cpu': '10m', 'memory': '32Mi'}, 'limits': {'cpu': '100m', 'memory': '64Mi'}}})
    apply({'apiVersion': 'apps/v1', 'kind': 'Deployment', 'metadata': {'name': RUN, 'namespace': NS}, 'spec': {'replicas': 1, 'selector': {'matchLabels': {'app': RUN}}, 'template': {'metadata': {'labels': {'app': RUN}}, 'spec': {'serviceAccountName': RUN, 'containers': containers, 'volumes': [{'name': 'app', 'configMap': {'name': RUN}}]}}}})
    kubectl('-n', NS, 'rollout', 'status', 'deployment/' + RUN, '--timeout=60s')

def main():
    install_observer()
    objects = fixture(RUN)
    root = objects[2]
    # The same generic action is selected independently at east and west.
    implementation = objects[1]['spec']['templates'][0]
    implementation['container'] = {'image': 'python:3.12-alpine', 'command': ['python', '-c'],
        'args': ['import os,json;v=os.environ["VALUE"];assert v!="invalid";json.dump({"target":v,"status":"Complete","children":[]},open("/tmp/result","w"))'],
        'env': [{'name': 'VALUE', 'value': '{{inputs.parameters.value}}'}]}
    calls = objects[4]['spec']['calls']
    calls[0]['resultBindings'] = [{'name': 'report', 'resultTemplate': RUN + '-summary', 'outputMappings': [{'from': 'answer', 'to': 'payload'}]}]
    west = copy.deepcopy(calls[0]); west['callPath'][1] = 'west'; west['resultBindings'][0]['resultTemplate'] = RUN + '-tree'; calls.append(west)
    # No terminal echo of JSON: keep the downstream node independent of its quoting.
    root['spec']['templates'][0]['dag']['tasks'] = [t for t in root['spec']['templates'][0]['dag']['tasks'] if t['name'] != 'report']
    manifest = 'apiVersion: results.example.io/v1alpha1\nkind: ResultQuery\nmetadata:\n  name: {{workflow.name}}-query\n  labels:\n    result-demo: '+RUN+'\nspec:\n  operation: report.read\n  target: "{{inputs.parameters.target}}"\n'
    query = {'name': 'query', 'metadata': {'annotations': {G+'/result-reuse': 'same-inputs'}}, 'inputs': {'parameters': [{'name': 'target'}]},
             'resource': {'action': 'create', 'setOwnerReference': True, 'successCondition': 'status.ready == true', 'manifest': manifest},
             'outputs': {'parameters': [{'name': 'query-name', 'valueFrom': {'jsonPath': '{.metadata.name}'}}, {'name': 'query-uid', 'valueFrom': {'jsonPath': '{.metadata.uid}'}}]}}
    root['spec']['templates'][0]['metadata'] = {'annotations': {G+'/bindable-calls': '["observe"]'}}
    root['spec']['templates'].append(copy.deepcopy(query))
    root['spec']['templates'][0]['dag']['tasks'].append({'name': 'observe', 'dependencies': ['approve'], 'template': 'query', 'arguments': {'parameters': [{'name': 'target', 'value': 'dynamic-target'}]}})
    schema = {'type': 'object', 'properties': {'target': {'type': 'string'}}, 'required': ['target'], 'additionalProperties': False}
    objects += [resource('ActionDefinition', RUN+'-query', {'inputSchema': schema, 'outputSchema': {'type': 'object', 'properties': {n: {'type': 'string'} for n in ['query-name', 'query-uid']}}, 'template': query})]
    live = result_spec('Live controller report', 'tree'); live['source'] = {'format': 'json', 'resource': {'group': 'results.example.io', 'version': 'v1alpha1', 'resource': 'resultqueries', 'nameOutput': 'name', 'uidOutput': 'uid', 'documentPath': '/status/document', 'observedAtPath': '/status/observedAt', 'maxAgeSeconds': 30}}
    calls.append({'callPath': ['main', 'observe'], 'actionDefinition': RUN+'-query', 'resultBindings': [{'name': 'observation', 'resultTemplate': RUN+'-live', 'outputMappings': [{'from': 'query-name', 'to': 'name'}, {'from': 'query-uid', 'to': 'uid'}]}]})
    objects += [resource('ActionResultTemplate', RUN+'-summary', result_spec('Original summary')), resource('ActionResultTemplate', RUN+'-tree', result_spec('Original tree', 'tree')), resource('ActionResultTemplate', RUN+'-live', live)]
    for obj in objects: apply(obj)
    kubectl('-n', NS, 'wait', '--for=condition=DefinitionValid', 'workflowparameterbinding/'+RUN, '--timeout=45s')
    saved = api('PUT', PREFIX+'/sets/'+RUN, {'template': RUN, 'values': {'message': 'invalid'}, 'resourceVersion': ''})
    preview = api('POST', PREFIX+'/preview', {'binding': RUN, 'revision': saved['spec']['revision']})
    assert not preview['errors'], preview
    assert len(preview['definitions']['callResults']) == 3
    assert len([k for k in preview['definitionVersions'] if k.startswith('result/')]) == 3
    wf = api('POST', PREFIX+'/submit', {'binding': RUN, 'revision': saved['spec']['revision'], 'executionDigest': preview['executionDigest'], 'idempotencyKey': RUN})
    name, uid = wf['metadata']['name'], wf['metadata']['uid']
    wait_workflow(name, lambda w: any(n['type']=='Suspend' and n['phase']=='Running' for n in w.get('status', {}).get('nodes', {}).values()))
    api('PUT', '/api/v1/workflows/'+NS+'/'+name+'/resume', {})
    failed = wait_workflow(name, lambda w: w.get('status', {}).get('phase') in ['Failed', 'Error'])
    west = next(n for n in failed['status']['nodes'].values() if n['templateName'].startswith('action-') and '.west.' in n['name'])
    observer = next(n for n in failed['status']['nodes'].values() if n['templateName'].startswith('action-') and '.observe.' in n['name'])
    west_before = api('GET', result_url(failed, west)); assert west_before['items'][0]['value']['target'] == 'unchanged'
    assert west_before['items'][0]['views'][0]['type'] == 'tree'
    live_before = api('GET', result_url(failed, observer)); assert live_before['items'][0]['scope'] == 'LiveObservation' and live_before['items'][0]['available'], live_before
    queries = json.loads(kubectl('-n', NS, 'get', 'resultqueries', '-l', 'result-demo='+RUN, '-o', 'json'))['items']; assert len(queries) == 1
    # The observer executes another GET while Argo only reads the same result CR.
    kubectl('-n', NS, 'exec', 'deployment/'+RUN, '-c', 'provider', '--', 'python', '-c', 'import urllib.request;urllib.request.urlopen(urllib.request.Request("http://127.0.0.1:8080/advance",method="POST",data=b""))')
    for _ in range(15):
        current = api('GET', result_url(failed, observer))
        if current['items'][0].get('value', {}).get('generation', 0) > live_before['items'][0]['value']['generation']: break
        time.sleep(1)
    else: raise AssertionError('controller did not refresh its projection')
    assert current['invocation'] == live_before['invocation']
    retained = {i: copy.deepcopy(n) for i,n in failed['status']['nodes'].items() if n['phase']=='Succeeded'}
    pods_before = pod_uids(name)
    # Mutable draft result definitions never rewrite already executed nodes.
    objects[-2]['spec']['title'] = 'New tree'; apply(objects[-2])
    objects[-3]['spec']['title'] = 'New summary'; apply(objects[-3])
    assert api('GET', result_url(failed, west)) == west_before
    saved = api('PUT', PREFIX+'/sets/'+RUN, {'template': RUN, 'values': {'message': 'fixed'}, 'resourceVersion': saved['metadata']['resourceVersion']})
    req = {'workflow': name, 'uid': uid, 'revision': saved['spec']['revision']}
    plan = api('POST', PREFIX+'/retry-preview', req)
    assert all('west' not in p and 'approve' not in p and 'observe' not in p for p in plan['affectedTasks']), plan
    api('POST', PREFIX+'/retry', {**req, 'resourceVersion': plan['resourceVersion'], 'planDigest': plan['digest'], 'idempotencyKey': RUN+'-retry'})
    done = wait_workflow(name, lambda w:w.get('status', {}).get('phase')=='Succeeded')
    assert done['metadata']['uid'] == uid
    for i,n in retained.items(): assert {k:v for k,v in done['status']['nodes'][i].items() if k not in ['children','outboundNodes']} == {k:v for k,v in n.items() if k not in ['children','outboundNodes']}
    assert all(pod_uids(name)[i] == value for i,value in pods_before.items() if i in retained)
    assert api('GET', result_url(done, west)) == west_before, 'retained nodes must retain original result contracts'
    east = next(n for n in done['status']['nodes'].values() if n['templateName'].startswith('action-') and '.east.' in n['name'])
    results = api('GET', result_url(done, east)); assert results['items'][0]['title']=='New summary' and results['items'][0]['value']['target']=='fixed', results
    history = done['metadata']['annotations'][G+'/retry']
    assert api('GET', result_url(done, west, history))['items']==west_before['items']
    assert len(json.loads(kubectl('-n', NS, 'get', 'workflows', '-l', 'workflows.argoproj.io/workflow-template='+RUN, '-o', 'json'))['items'])==1
    assert len(json.loads(kubectl('-n', NS, 'get', 'resultqueries', '-l', 'result-demo='+RUN, '-o', 'json'))['items'])==1
    # Browser uses the same contract; refreshing must never submit or retry.
    from playwright.sync_api import sync_playwright, expect
    with sync_playwright() as pw:
        browser = pw.chromium.launch(headless=True, executable_path=os.environ['ARGO_PARAMETERIZATION_CHROMIUM'])
        page = browser.new_page(viewport={'width': 1440, 'height': 1080})
        version = api('GET', '/api/v1/version')['version'].rsplit('.', 1)[0]
        page.add_init_script('localStorage.setItem("modal/ftu", JSON.stringify("dismissed"));localStorage.setItem("modal/version",JSON.stringify('+json.dumps(version)+'));')
        errors = []; page.on('pageerror', lambda e: errors.append(str(e)))
        page.goto(BASE+'/workflows/'+NS+'/'+name+'?nodeId='+east['id'])
        panel = page.get_by_role('region', name='Action results', exact=True)
        expect(panel).to_be_visible(timeout=15000)
        expect(panel.get_by_role('heading', name='New summary', exact=True)).to_be_visible()
        expect(panel.get_by_text('fixed', exact=True)).to_be_visible()
        writes = []; page.on('request', lambda r: writes.append(r.url) if r.method not in ['GET', 'HEAD'] else None)
        with page.expect_response(lambda r: '/node-results/' in r.url):
            panel.get_by_role('button', name='Refresh results', exact=True).click()
        assert not writes and not errors, (writes, errors)
        page.goto(BASE+'/workflows/'+NS+'/'+name+'?nodeId='+observer['id'])
        expect(page.get_by_role('region', name='Action results').get_by_text('dynamic-target-child', exact=True)).to_be_visible(timeout=15000)
        page.get_by_role('region', name='Action results').scroll_into_view_if_needed()
        page.screenshot(path='/tmp/parameterization-result-bindings.png', full_page=True)
        browser.close()
    # Leave the successful demo for manual UI testing; quiet its fixture poller.
    kubectl('-n', NS, 'scale', 'deployment/'+RUN, '--replicas=0')
    print('PASS', json.dumps({'workflow':name,'uid':uid,'callSites':3,'sameWorkflowRetry':True,'retainedResults':True,'controllerRefresh':True}), flush=True)

if __name__ == '__main__': main()
