"""Local definition lifecycle -> UI status -> saved version -> native execution.
Only unique synthetic definitions are changed/deleted. No delivery resources.
"""
import copy, json, os, time, uuid
from e2e import api, BASE, PREFIX, NS, wait_workflow
from template_refs_e2e import kubectl, apply, native, resource
from playwright.sync_api import sync_playwright

RUN = 'status-' + uuid.uuid4().hex[:8]
GROUP = '.parameters.workflows.example.io'

def get(kind):
    return api('GET', PREFIX + '/catalog/' + kind + '?name=' + RUN)

def wait_status(kind, expected, predicate=lambda obj: True):
    until = time.monotonic() + 90
    while time.monotonic() < until:
        obj = get(kind)
        conditions = obj.get('status', {}).get('conditions', [])
        if any(c['type'] == 'DefinitionValid' and c['status'] == expected for c in conditions) and obj['status'].get('observedGeneration') == obj['metadata']['generation'] and predicate(obj):
            return obj
        time.sleep(.4)
    raise AssertionError(('definition did not reconcile', kind, expected, obj))

def main():
    # The status controller cannot change execution state or definition specs.
    who = 'system:serviceaccount:' + NS + ':parameterization-definitions'
    import subprocess
    for verb, target in [('create', 'workflows.argoproj.io'), ('update', 'workflowparameterbindings' + GROUP), ('get', 'secrets')]:
        result = subprocess.run(['kubectl', '--context', 'k3d-argo-parameterization', '-n', NS, 'auth', 'can-i', verb, target, '--as=' + who], capture_output=True, text=True)
        assert result.returncode == 1 and result.stdout.strip() == 'no', (verb, target, result.stdout)
    model = resource('ParameterTemplate', RUN, {'dialect': 'http://json-schema.org/draft-07/schema#', 'schema': {'type': 'object', 'properties': {'count': {'type': 'integer'}}, 'required': ['count']}})
    form = resource('WorkflowForm', RUN, {'parameterTemplate': RUN, 'fields': [{'path': '/count', 'widget': 'number'}]})
    workflow = native('WorkflowTemplate', RUN, {'entrypoint': 'apply', 'serviceAccountName': 'workflow-executor', 'arguments': {'parameters': [{'name': 'count'}]}, 'templates': [{'name': 'apply', 'resource': {'action': 'create', 'manifest': 'apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: {{workflow.name}}-result\ndata:\n  count: "{{workflow.parameters.count}}"\n'}}]})
    binding = resource('WorkflowParameterBinding', RUN, {'workflowTemplate': RUN, 'parameterTemplate': RUN, 'form': RUN, 'mappings': [{'path': '/count', 'parameter': 'count', 'encoding': 'decimal'}]})
    apply(workflow); apply(form); apply(binding)
    wait_status('workflowparameterbindings', 'False')
    apply(model)
    healthy = wait_status('workflowparameterbindings', 'True')
    assert len(healthy['status']['resolvedReferences']) == 3
    assert healthy['spec'] == binding['spec']
    first_uid = get('parametertemplates')['metadata']['uid']
    saved = api('PUT', PREFIX + '/sets/' + RUN, {'template': RUN, 'values': {'count': 4}, 'resourceVersion': ''})
    revision = saved['spec']['revision']
    preview = api('POST', PREFIX + '/preview', {'binding': RUN, 'revision': revision})
    # Status-only Kubernetes writes must not turn a reviewed preview stale.
    kubectl('-n', NS, 'patch', 'workflowparameterbindings' + GROUP, RUN, '--subresource=status', '--type=merge', '-p', json.dumps({'status': {'digest': 'synthetic-status-only-write'}}))
    assert get('workflowparameterbindings')['metadata']['resourceVersion'] != healthy['metadata']['resourceVersion']
    second = api('POST', PREFIX + '/preview', {'binding': RUN, 'revision': revision})
    assert preview['executionDigest'] == second['executionDigest']
    wf = api('POST', PREFIX + '/submit', {'binding': RUN, 'revision': revision, 'executionDigest': preview['executionDigest'], 'idempotencyKey': RUN})
    done = wait_workflow(wf['metadata']['name'], lambda w: w.get('status', {}).get('phase') == 'Succeeded')
    original_nodes = copy.deepcopy(done['status']['nodes'])
    submission_name = done['metadata']['annotations']['parameters.workflows.example.io/submission']
    receipt = json.loads(kubectl('-n', NS, 'get', 'workflowsubmissions' + GROUP, submission_name, '-o', 'json'))
    original_receipt = copy.deepcopy(receipt['spec'])
    cm = json.loads(kubectl('-n', NS, 'get', 'configmap', wf['metadata']['name'] + '-result', '-o', 'json'))
    assert cm['data']['count'] == '4'
    # Invalid source change propagates through form and binding automatically.
    invalid = copy.deepcopy(model)
    invalid['spec']['schema']['properties']['count']['type'] = 'string'
    apply(invalid)
    wait_status('workflowforms', 'False')
    wait_status('workflowparameterbindings', 'False')
    apply(model)
    wait_status('workflowparameterbindings', 'True')
    # Deletion/recreation is a new UID even when source contents are identical.
    kubectl('-n', NS, 'delete', 'parametertemplates' + GROUP, RUN, '--wait=true')
    wait_status('workflowparameterbindings', 'False')
    apply(model)
    healthy = wait_status('workflowparameterbindings', 'True', lambda o: any(r['kind'] == 'ParameterTemplate' and r['uid'] != first_uid for r in o['status']['resolvedReferences']))
    # Native reference changes also invalidate and recover the binding.
    invalid_workflow = copy.deepcopy(workflow)
    invalid_workflow['spec']['arguments']['parameters'] = []
    apply(invalid_workflow)
    wait_status('workflowparameterbindings', 'False')
    apply(workflow)
    healthy = wait_status('workflowparameterbindings', 'True')
    # No endless resourceVersion feedback between dependent status updates.
    stable = {kind: get(kind)['metadata']['resourceVersion'] for kind in ('parametertemplates', 'workflowforms', 'workflowparameterbindings')}
    time.sleep(2)
    assert stable == {kind: get(kind)['metadata']['resourceVersion'] for kind in stable}
    after = api('GET', '/api/v1/workflows/' + NS + '/' + wf['metadata']['name'])
    assert after['metadata']['uid'] == wf['metadata']['uid'] and after['status']['phase'] == 'Succeeded'
    assert after['status']['nodes'] == original_nodes
    assert json.loads(kubectl('-n', NS, 'get', 'workflowsubmissions' + GROUP, submission_name, '-o', 'json'))['spec'] == original_receipt
    assert api('GET', PREFIX + '/revisions/' + revision)['spec']['values'] == {'count': 4}
    # The normal catalog renders the current-generation status and identities.
    with sync_playwright() as p:
        browser = p.chromium.launch(executable_path=os.environ.get('ARGO_PARAMETERIZATION_CHROMIUM'), headless=True)
        page = browser.new_page(viewport={'width': 1500, 'height': 1000})
        version = api('GET', '/api/v1/version')['version']
        version = version.rsplit('.', 1)[0] if '.' in version else version
        page.add_init_script('localStorage.setItem("modal/ftu",JSON.stringify("dismissed"));localStorage.setItem("modal/version",JSON.stringify(' + json.dumps(version) + '));')
        page.goto(BASE + '/parameters/' + NS)
        page.get_by_role('button', name='Resource definitions and records', exact=True).click()
        catalog = page.get_by_role('region', name='Parameter resource catalog')
        catalog.get_by_label('Parameter resource type').select_option('workflowparameterbindings')
        while catalog.get_by_role('button', name=RUN, exact=True).count() == 0:
            more = catalog.get_by_role('button', name='Load more resources')
            more.wait_for(state='visible'); more.click(); page.wait_for_timeout(200)
        catalog.get_by_role('button', name=RUN, exact=True).click()
        status = catalog.get_by_role('region', name='Definition validation')
        status.get_by_text('Static checks passed', exact=True).wait_for()
        status.get_by_text('Observed definition references (3)', exact=True).click()
        assert status.get_by_text('ParameterTemplate/' + RUN, exact=True).is_visible()
        page.screenshot(path='/tmp/parameterization-definition-status-success.png', full_page=True)
        browser.close()
    print('PASS ' + json.dumps({'name': RUN, 'missingReferenceRecovered': True, 'schemaChangePropagated': True, 'recreatedUIDObserved': True, 'statusOnlyPreviewStable': True, 'noStatusFeedback': True, 'leastPrivilege': True, 'workflow': wf['metadata']['name'], 'phase': 'Succeeded', 'frozenReceiptAndNodesPreserved': True, 'browserStatus': True}), flush=True)

if __name__ == '__main__':
    main()
