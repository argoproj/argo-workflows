"""Real admission enforces immutable documents without depending on policy CEL."""
import json, subprocess, uuid, time
from contextlib import contextmanager
from e2e import NS
from template_refs_e2e import apply, kubectl, resource

@contextmanager
def only_webhook_for_fixture(name):
    # Exclude only this synthetic object from the newer CEL policy, so the
    # same test demonstrates enforcement on clusters that lack that policy.
    kind='validatingadmissionpolicybinding'; policy='parameterization-immutable-snapshots'
    raw=kubectl('get',kind,policy,'--ignore-not-found=true','-o','json')
    if not raw:
        yield
        return
    before=json.loads(raw)
    import copy
    after=copy.deepcopy(before)
    match=after['spec'].setdefault('matchResources',{})
    selector=match.setdefault('objectSelector',{})
    selector.setdefault('matchExpressions',[]).append({'key':'parameterization-webhook-test','operator':'NotIn','values':[name]})
    applied=json.loads(kubectl('replace','-f','-','-o','json',body=after))
    try:
        yield
    finally:
        current=json.loads(kubectl('get',kind,policy,'-o','json'))
        assert current['spec']==applied['spec'],'policy changed concurrently'
        before['metadata']['resourceVersion']=current['metadata']['resourceVersion']
        kubectl('replace','-f','-',body=before)

def main():
    name = 'snapshot-webhook-' + uuid.uuid4().hex[:8]
    obj = resource('ActionDefinition', name, {
        'inputSchema': {'type':'object'}, 'outputSchema': {'type':'object'},
        'template': {'name':'action','container':{'image':'alpine:3.20','command':['true']}}})
    obj['metadata']['labels']={'parameterization-webhook-test':name}
    apply(obj)
    current = json.loads(kubectl('-n',NS,'get','actiondefinition',name,'-o','json'))
    current['spec']['template']['container']['command']=['false']
    try:
        with only_webhook_for_fixture(name):
            for attempt in range(10):
                try:
                    apply(current)
                    break
                except subprocess.CalledProcessError as error:
                    if b"ValidatingAdmissionPolicy" not in (error.stderr or b"") or attempt == 9:
                        raise
                    time.sleep(0.5)
    except subprocess.CalledProcessError as error:
        output = (error.stderr or error.output or b'')
        if isinstance(output,bytes): output=output.decode()
        assert 'snapshot-contract.parameters.workflows.example.io' in output, output
        assert 'snapshot is immutable' in output, output
    else:
        raise AssertionError('snapshot mutation was accepted')
    stored=json.loads(kubectl('-n',NS,'get','actiondefinition',name,'-o','json'))
    assert stored['metadata']['uid']==current['metadata']['uid']
    assert stored['spec']==obj['spec']
    print('PASS immutable snapshot verified through the namespace webhook',flush=True)

if __name__=='__main__':main()
