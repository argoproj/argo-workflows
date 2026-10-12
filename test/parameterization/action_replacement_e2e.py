"""Author review -> rejected replacement -> native execution -> same Workflow retry.

Only the isolated local fixture namespace is used. No external provider is invoked.
"""
import copy, json, uuid
from e2e import api, wait_workflow, PREFIX, NS
from action_calls_e2e import fixture, G
from template_refs_e2e import apply, kubectl
from nested_retry_e2e import pod_uids

RUN = 'action-contract-' + uuid.uuid4().hex[:8]
IMAGE = 'alpine@sha256:5291449c3df73caf6ed85e649dec1b9e818b39a5d8c871e97afc13e9cd5e8fa8'

def main():
    objects = fixture(RUN)
    contract = {'effect': 'Pure', 'targetScope': 'local-string-calculation-v1',
                'permissionScope': 'no-external-access-v1', 'outputMeaning': 'prefixed-text-v1',
                'capabilities': [{'name': 'Idempotency', 'condition': 'same input text'}]}
    objects[3]['spec']['executionContract'] = contract
    for obj in [objects[1], objects[2]]:
        for template in obj['spec']['templates']:
            if 'container' in template:
                template['container']['image'] = IMAGE
    for obj in objects:
        apply(obj)
    saved = api('PUT', PREFIX+'/sets/'+RUN, {'template': RUN, 'values': {'message': 'invalid'}, 'resourceVersion': ''})
    request = {'binding': RUN, 'revision': saved['spec']['revision']}
    rejected = api('POST', PREFIX+'/preview', request, expected=400)
    assert 'author-approved baseline' in json.dumps(rejected), rejected
    review = api('GET', PREFIX+'/action-reviews/'+RUN)
    assert review['definition']['executionContract'] == contract
    assert review['definition']['template']['container']['image'] == IMAGE
    baseline = next(t for t in objects[2]['spec']['templates'] if t['name'] == 'base')
    policy = {'contract': contract, 'implementationDigests': [review['implementationDigest']],
              'interface': {key: objects[4]['spec']['calls'][0][key] for key in ['inputMappings', 'outputMappings']}}
    baseline['metadata']['annotations'][G+'/action-replacement-policy'] = json.dumps(policy)
    apply(objects[2])
    kubectl('-n', NS, 'wait', '--for=condition=DefinitionValid', 'workflowparameterbinding/'+RUN, '--timeout=45s')
    preview = api('POST', PREFIX+'/preview', request)
    assert not preview['errors'], preview
    assert preview['definitions']['actions'][RUN]['executionContract'] == contract

    # A library can change independently of the immutable ActionDefinition.
    # Neither preview nor submit may use a stale implementation approval.
    changed = copy.deepcopy(objects[1])
    changed['spec']['templates'][0]['container']['args'][0] += '; printf different'
    apply(changed)
    changed_review = api('GET', PREFIX+'/action-reviews/'+RUN)
    assert changed_review['implementationDigest'] != review['implementationDigest']
    rejected = api('POST', PREFIX+'/preview', request, expected=400)
    assert 'not been reviewed' in json.dumps(rejected)
    api('POST', PREFIX+'/submit', {**request, 'executionDigest': preview['executionDigest'], 'idempotencyKey': RUN+'-unreviewed'}, expected=400)
    assert not json.loads(kubectl('-n', NS, 'get', 'workflows', '-l', 'workflows.argoproj.io/workflow-template='+RUN, '-o', 'json'))['items']

    # Changing the source policy must reject the old semantics, even if its
    # allowed digest list still contains the exact implementation.
    incompatible = copy.deepcopy(objects[2])
    policy2 = copy.deepcopy(policy)
    policy2['contract']['targetScope'] = 'different-calculation-v1'
    next(t for t in incompatible['spec']['templates'] if t['name']=='base')['metadata']['annotations'][G+'/action-replacement-policy'] = json.dumps(policy2)
    apply(objects[1]); apply(incompatible)
    rejected = api('POST', PREFIX+'/preview', request, expected=400)
    assert 'changes effect, target' in json.dumps(rejected)
    apply(objects[2])
    preview = api('POST', PREFIX+'/preview', request)
    wf = api('POST', PREFIX+'/submit', {**request, 'executionDigest': preview['executionDigest'], 'idempotencyKey': RUN})
    name, uid = wf['metadata']['name'], wf['metadata']['uid']
    wait_workflow(name, lambda w: any(n.get('type')=='Suspend' and n.get('phase')=='Running' for n in w.get('status', {}).get('nodes', {}).values()))
    api('PUT', '/api/v1/workflows/'+NS+'/'+name+'/resume', {})
    before = wait_workflow(name, lambda w: w.get('status', {}).get('phase') in ['Failed', 'Error'])
    pods = pod_uids(name)
    preserved = {i: p for i, p in pods.items() if before['status']['nodes'][i]['phase']=='Succeeded'}
    assert len(preserved)==1
    approval = next(n for n in before['status']['nodes'].values() if n['type']=='Suspend')
    saved = api('PUT', PREFIX+'/sets/'+RUN, {'template': RUN, 'values': {'message': 'fixed'}, 'resourceVersion': saved['metadata']['resourceVersion']})
    retry = {'workflow': name, 'uid': uid, 'revision': saved['spec']['revision']}
    plan = api('POST', PREFIX+'/retry-preview', retry)
    api('POST', PREFIX+'/retry', {**retry, 'resourceVersion': plan['resourceVersion'], 'planDigest': plan['digest'], 'idempotencyKey': RUN+'-retry'})
    done = wait_workflow(name, lambda w: w.get('status', {}).get('phase') in ['Succeeded', 'Failed', 'Error'])
    assert done['status']['phase']=='Succeeded', done['status']
    assert done['metadata']['uid']==uid
    retained_approval = next(n for n in done['status']['nodes'].values() if n['type']=='Suspend')
    assert {k: retained_approval.get(k) for k in ['id', 'phase', 'startedAt', 'finishedAt', 'outputs']} == {k: approval.get(k) for k in ['id', 'phase', 'startedAt', 'finishedAt', 'outputs']}
    assert all(pod_uids(name)[i]==v for i, v in preserved.items())
    east = next(n for n in done['status']['nodes'].values() if n['name']==name+'.east')
    assert east['outputs']['parameters'][0]['value']=='action:fixed'
    assert len(json.loads(kubectl('-n', NS, 'get', 'workflows', '-l', 'workflows.argoproj.io/workflow-template='+RUN, '-o', 'json'))['items'])==1
    print('PASS', json.dumps({'workflow': name, 'uid': uid, 'reviewedDigest': review['implementationDigest'], 'unreviewedSubmitCreatedNoWorkflow': True, 'approvalAndUnrelatedPodRetained': True}), flush=True)

if __name__ == '__main__':
    main()
