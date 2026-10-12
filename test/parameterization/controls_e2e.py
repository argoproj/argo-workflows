"""Native lock/terminate/resume/exit behavior with a frozen parameterized spec."""
import json, uuid
from e2e import api, PREFIX, NS, wait_workflow
from template_refs_e2e import resource, native, apply, kubectl
RUN = 'controls-' + uuid.uuid4().hex[:8]

def main():
    apply(resource('ParameterTemplate', RUN, {'dialect':'http://json-schema.org/draft-07/schema#','schema':{'type':'object','properties':{},'additionalProperties':False}}))
    spec = {'entrypoint':'hold','onExit':'exit','serviceAccountName':'workflow-executor',
            'synchronization':{'mutexes':[{'name':RUN}]},
            'volumes':[{'name':'data','configMap':{'name':'{{workflow.name}}-input','optional':True}}],
            'templates':[{'name':'hold','suspend':{}},{'name':'exit','resource':{'action':'create','manifest':'apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: {{workflow.name}}-exit\ndata:\n  done: "true"\n'}}]}
    apply(native('WorkflowTemplate', RUN, spec))
    apply(resource('WorkflowParameterBinding', RUN, {'workflowTemplate':RUN,'parameterTemplate':RUN,'mappings':[]}))
    saved = api('PUT', PREFIX+'/sets/'+RUN, {'template':RUN,'values':{},'resourceVersion':''})
    preview = api('POST', PREFIX+'/preview', {'binding':RUN,'revision':saved['spec']['revision']})
    def submit(label):
        return api('POST', PREFIX+'/submit', {'binding':RUN,'revision':saved['spec']['revision'],'executionDigest':preview['executionDigest'],'idempotencyKey':RUN+'-'+label})
    first, second, third = submit('first'), None, None
    first_name = first['metadata']['name']
    wait_workflow(first_name, lambda w:any(n.get('type')=='Suspend' and n.get('phase')=='Running' for n in w.get('status',{}).get('nodes',{}).values()))
    second = submit('second'); second_name = second['metadata']['name']
    wait_workflow(second_name, lambda w:w.get('status',{}).get('phase')=='Pending')
    api('PUT','/api/v1/workflows/'+NS+'/'+second_name+'/terminate',{})
    wait_workflow(second_name, lambda w:w.get('status',{}).get('phase')=='Succeeded')
    third = submit('third'); third_name = third['metadata']['name']
    wait_workflow(third_name, lambda w:w.get('status',{}).get('phase')=='Pending')
    assert api('GET','/api/v1/workflows/'+NS+'/'+first_name)['status']['phase']=='Running'
    api('PUT','/api/v1/workflows/'+NS+'/'+first_name+'/resume',{})
    first_done = wait_workflow(first_name, lambda w:w.get('status',{}).get('phase')=='Succeeded')
    wait_workflow(third_name, lambda w:any(n.get('type')=='Suspend' and n.get('phase')=='Running' for n in w.get('status',{}).get('nodes',{}).values()))
    api('PUT','/api/v1/workflows/'+NS+'/'+third_name+'/resume',{})
    third_done = wait_workflow(third_name, lambda w:w.get('status',{}).get('phase')=='Succeeded')
    for w in [first_done, third_done]:
        assert w['spec']['volumes'][0]['configMap']['name']=='{{workflow.name}}-input'
        cm = json.loads(kubectl('-n',NS,'get','configmap',w['metadata']['name']+'-exit','-o','json'))
        assert cm['data']['done']=='true'
        assert not w['status'].get('synchronization',{}).get('mutex',{}).get('holding')
    print('PASS', json.dumps({'run':RUN,'cases':['queued-termination','unrelated-lock-retained','native-resume','exit-resource-execution','raw-input-expression-preserved'],'workflowUIDs':[w['metadata']['uid'] for w in [first, second, third]]}), flush=True)

if __name__ == '__main__': main()
