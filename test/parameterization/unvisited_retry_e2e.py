"""A new binding replaces an unvisited implementation on the same Workflow."""
import copy, json, uuid
from e2e import NS, PREFIX, api, wait_workflow
from template_refs_e2e import apply, native, resource
from nested_retry_e2e import pod_uids

G = 'parameters.workflows.example.io'

def main():
    name = 'unvisited-retry-' + uuid.uuid4().hex[:8]
    ready = name + '-ready'
    apply({'apiVersion':'v1','kind':'ConfigMap','metadata':{'name':ready,'namespace':NS},'data':{'value':'ready'}})
    model = {'dialect':'http://json-schema.org/draft-07/schema#','schema':{'type':'object',
        'properties':{'target':{'type':'string'}},'required':['target'],'additionalProperties':False}}
    annotations = {G+'/retry-safe':'idempotent',G+'/result-reuse':'same-inputs'}
    spec = {'entrypoint':'main','serviceAccountName':'workflow-executor',
        'arguments':{'parameters':[{'name':'target'}]},'templates':[
            {'name':'main','dag':{'tasks':[{'name':'approve','template':'approve'},
                {'name':'recover','template':'recover','dependencies':['approve']},
                {'name':'future','template':'future','dependencies':['recover']}]}},
            {'name':'approve','suspend':{},'metadata':{'annotations':{G+'/retry-dependencies':'inputs'}}},
            {'name':'recover','metadata':{'annotations':annotations},'resource':{'action':'get',
                'manifest':'apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: {{workflow.parameters.target}}\n'}},
            {'name':'future','dag':{'tasks':[{'name':'old','template':'recover'}]}}]}
    def definitions(suffix, body):
        apply(native('WorkflowTemplate',name+suffix,body))
        apply(resource('WorkflowParameterBinding',name+suffix,{'workflowTemplate':name+suffix,
            'parameterTemplate':name,'mappings':[{'path':'/target','parameter':'target','encoding':'string'}]}))
    apply(resource('ParameterTemplate',name,model))
    definitions('',spec)
    saved=api('PUT',PREFIX+'/sets/'+name,{'template':name,'resourceVersion':'','values':{'target':name+'-missing'}})
    preview=api('POST',PREFIX+'/preview',{'binding':name,'revision':saved['spec']['revision']})
    wf=api('POST',PREFIX+'/submit',{'binding':name,'revision':saved['spec']['revision'],
        'executionDigest':preview['executionDigest'],'idempotencyKey':name})
    wn,uid=wf['metadata']['name'],wf['metadata']['uid']
    wait_workflow(wn,lambda w:any(n.get('type')=='Suspend' and n.get('phase')=='Running' for n in w.get('status',{}).get('nodes',{}).values()))
    api('PUT','/api/v1/workflows/'+NS+'/'+wn+'/resume',{})
    failed=wait_workflow(wn,lambda w:w.get('status',{}).get('phase')=='Failed')
    approval_id,approval=next((i,n) for i,n in failed['status']['nodes'].items() if n.get('type')=='Suspend')
    changed=copy.deepcopy(spec)
    changed['templates'][-1]['dag']['tasks']=[{'name':'replacement','template':'recover'}]
    definitions('-next',changed)
    saved=api('PUT',PREFIX+'/sets/'+name,{'template':name,'resourceVersion':saved['metadata']['resourceVersion'],'values':{'target':ready}})
    request={'workflow':wn,'uid':uid,'binding':name+'-next','revision':saved['spec']['revision'],'callPaths':['recover']}
    plan=api('POST',PREFIX+'/retry-preview',request)
    assert approval_id in plan['retainedNodes'] and 'future/replacement' in plan['affectedTasks'],plan
    command={**request,'resourceVersion':plan['resourceVersion'],'planDigest':plan['digest'],'idempotencyKey':name+'-next'}
    api('POST',PREFIX+'/retry',command)
    done=wait_workflow(wn,lambda w:w.get('status',{}).get('phase')=='Succeeded')
    assert done['metadata']['uid']==uid
    assert done['status']['nodes'][approval_id]==approval
    assert any(n.get('displayName')=='replacement' and n.get('phase')=='Succeeded' for n in done['status']['nodes'].values())
    assert not any(n.get('displayName')=='old' for n in done['status']['nodes'].values())
    assert api('GET',PREFIX+'/edit-context/'+wn+'?uid='+uid)['binding']==name+'-next'
    # A following original-input retry must keep the adopted definition.
    request={'workflow':wn,'uid':uid,'callPaths':['future/replacement']}
    plan=api('POST',PREFIX+'/retry-preview',request)
    api('POST',PREFIX+'/retry',{**request,'resourceVersion':plan['resourceVersion'],'planDigest':plan['digest'],'idempotencyKey':name+'-original'})
    done=wait_workflow(wn,lambda w:w.get('status',{}).get('phase')=='Succeeded')
    assert done['metadata']['uid']==uid and done['status']['nodes'][approval_id]==approval
    assert api('GET',PREFIX+'/edit-context/'+wn+'?uid='+uid)['binding']==name+'-next'
    # An executed shared implementation cannot be replaced by another binding.
    bad=copy.deepcopy(changed);bad['templates'][2]['resource']['action']='create'
    definitions('-bad',bad)
    saved=api('PUT',PREFIX+'/sets/'+name,{'template':name,'resourceVersion':saved['metadata']['resourceVersion'],'values':{'target':name+'-other'}})
    api('POST',PREFIX+'/retry-preview',{'workflow':wn,'uid':uid,'binding':name+'-bad','revision':saved['spec']['revision'],'callPaths':['recover']},400)
    print('PASS',json.dumps({'workflow':wn,'uid':uid,'unvisitedBindingAdopted':True,'approvalRetained':True,'originalRetryRetainsBinding':True,'visitedImplementationRejected':True}),flush=True)

if __name__=='__main__':main()
