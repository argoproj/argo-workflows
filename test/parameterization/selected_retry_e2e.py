"""Explicit original/new-input retry keeps independent failures and approvals."""
import json
import uuid
from e2e import NS, PREFIX, api, wait_workflow
from template_refs_e2e import apply, native, resource
from nested_retry_e2e import pod_uids

G = 'parameters.workflows.example.io'

def main():
    name = 'selected-retry-' + uuid.uuid4().hex[:8]
    target = name + '-ready'
    apply({'apiVersion':'v1','kind':'ConfigMap','metadata':{'name':target,'namespace':NS},'data':{'public':'ready'}})
    model = {'dialect':'http://json-schema.org/draft-07/schema#','schema':{
        'type':'object','properties':{x:{'type':'string'} for x in ('left','right')},
        'required':['left','right'],'additionalProperties':False}}
    annotations = {G+'/retry-safe':'idempotent',G+'/result-reuse':'same-inputs'}
    leaves = [{'name':x,'metadata':{'annotations':annotations},'resource':{'action':'get',
        'manifest':'apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: {{workflow.parameters.'+x+'}}\n'}} for x in ('left','right')]
    spec = {'entrypoint':'main','serviceAccountName':'workflow-executor',
        'arguments':{'parameters':[{'name':x} for x in ('left','right')]},
        'templates':[{'name':'main','dag':{'failFast':False,'tasks':[
            {'name':'approve','template':'approve'},
            *[{'name':x,'template':x,'dependencies':['approve']} for x in ('left','right')],
            {'name':'later-approval','template':'approve','dependencies':['left']}]}},
            {'name':'approve','suspend':{},'metadata':{'annotations':{G+'/retry-dependencies':'inputs'}}},*leaves]}
    for obj in [resource('ParameterTemplate',name,model),native('WorkflowTemplate',name,spec),
        resource('WorkflowParameterBinding',name,{'workflowTemplate':name,'parameterTemplate':name,
        'mappings':[{'path':'/'+x,'parameter':x,'encoding':'string'} for x in ('left','right')]})]:apply(obj)
    values = {'left':name+'-missing-left','right':name+'-missing-right'}
    saved = api('PUT',PREFIX+'/sets/'+name,{'template':name,'resourceVersion':'','values':values})
    preview = api('POST',PREFIX+'/preview',{'binding':name,'revision':saved['spec']['revision']})
    assert not preview['errors'],preview
    wf = api('POST',PREFIX+'/submit',{'binding':name,'revision':saved['spec']['revision'],
        'executionDigest':preview['executionDigest'],'idempotencyKey':name})
    wf_name,uid = wf['metadata']['name'],wf['metadata']['uid']
    wait_workflow(wf_name,lambda w:any(n.get('type')=='Suspend' and n.get('phase')=='Running' for n in w.get('status',{}).get('nodes',{}).values()))
    api('PUT','/api/v1/workflows/'+NS+'/'+wf_name+'/resume',{})
    done = wait_workflow(wf_name,lambda w:w.get('status',{}).get('phase')=='Failed')
    def by_name(w,x):return next((i,n) for i,n in w['status']['nodes'].items() if n.get('displayName')==x)
    approval_id,approval = by_name(done,'approve')
    later = None
    for index,(selected,changed) in enumerate([('left',False),('left',True),('right',True),('left',False)]):
        other = 'right' if selected=='left' else 'left'
        retained_id,retained = by_name(done,other)
        retained_pod = pod_uids(wf_name).get(retained_id)
        req = {'workflow':wf_name,'uid':uid,'callPaths':[selected]}
        if changed:
            values[selected]=target
            saved=api('PUT',PREFIX+'/sets/'+name,{'template':name,'resourceVersion':saved['metadata']['resourceVersion'],'values':values})
            req['revision']=saved['spec']['revision']
        plan=api('POST',PREFIX+'/retry-preview',req)
        expected = [selected] if index >= 2 else [selected, 'later-approval']
        assert sorted(plan['affectedTasks'])==sorted(expected),plan
        assert retained_id in plan['retainedNodes'] and approval_id in plan['retainedNodes']
        command={**req,'resourceVersion':plan['resourceVersion'],'planDigest':plan['digest'],'idempotencyKey':name+'-'+str(index)}
        api('POST',PREFIX+'/retry',command)
        if index==1:
            wait_workflow(wf_name,lambda w:any(n.get('type')=='Suspend' and n.get('phase')=='Running' for n in w.get('status',{}).get('nodes',{}).values()))
            api('PUT','/api/v1/workflows/'+NS+'/'+wf_name+'/resume',{})
        done=wait_workflow(wf_name,lambda w:w.get('status',{}).get('phase') in ('Failed','Error','Succeeded'))
        assert done['metadata']['uid']==uid
        assert done['status']['phase']==('Succeeded' if index>=2 else 'Failed'),done['status']
        for key in ['phase','startedAt','finishedAt','inputs','outputs','message']:
            assert done['status']['nodes'][retained_id].get(key)==retained.get(key)
            assert done['status']['nodes'][approval_id].get(key)==approval.get(key)
        assert pod_uids(wf_name).get(retained_id)==retained_pod
        if later:
            later_id,later_status=later
            for key in ['phase','startedAt','finishedAt','inputs','outputs','message']:
                assert done['status']['nodes'][later_id].get(key)==later_status.get(key)
        if index==1:later=by_name(done,'later-approval')
        assert api('POST',PREFIX+'/retry',command)['metadata']['uid']==uid
    print('PASS',json.dumps({'workflow':wf_name,'uid':uid,'originalAndChangedSelectedRetries':True,
        'independentFailedAndSucceededPodsPreserved':True,'priorAndLaterApprovalsPreserved':True,'sameWorkflow':True}),flush=True)

if __name__=='__main__':main()
