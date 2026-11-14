"""Saved revision -> two calls of one nested DAG -> same Workflow local retry."""
import json,uuid
from e2e import api,wait_workflow,PREFIX,NS
from template_refs_e2e import resource,native,apply,kubectl
RUN='nested-retry-'+uuid.uuid4().hex[:8]
G='parameters.workflows.example.io'

def fixture(name=RUN):
 def args(value):return {'parameters':[{'name':'payload','value':value},{'name':'items','value':'{{workflow.parameters.items}}'}]}
 model={'dialect':'http://json-schema.org/draft-07/schema#','schema':{'type':'object','required':['value','items'],'additionalProperties':False,'properties':{'value':{'type':'string'},'items':{'type':'array','minItems':1,'items':{'type':'string'}}}}}
 spec={'entrypoint':'main','serviceAccountName':'workflow-executor','arguments':{'parameters':[{'name':'value'},{'name':'items'}]},'templates':[
  {'name':'main','inputs':{'parameters':[{'name':'value'}]},'dag':{'failFast':False,'tasks':[{'name':'stable','template':'group','arguments':args('valid')},{'name':'vary','template':'group','arguments':args('{{inputs.parameters.value}}')},{'name':'report','template':'change','dependencies':['vary'],'arguments':{'parameters':[{'name':'payload','value':'{{tasks.vary.outputs.parameters.result}}'}]}}]}},
  {'name':'group','inputs':{'parameters':[{'name':'payload'},{'name':'items'}]},'outputs':{'parameters':[{'name':'result','valueFrom':{'parameter':'{{tasks.change.outputs.parameters.result}}'}}]},'dag':{'tasks':[
   {'name':'approve','template':'approve'},
   {'name':'keep','template':'keep','dependencies':['approve'],'withParam':'{{inputs.parameters.items}}','arguments':{'parameters':[{'name':'item','value':'{{item}}'}]}},
   {'name':'change','template':'change','dependencies':['keep'],'arguments':{'parameters':[{'name':'payload','value':"{{=inputs['parameters']['payload']}}"}]}}
  ]}},
  {'name':'approve','suspend':{}},
  {'name':'keep','inputs':{'parameters':[{'name':'item'}]},'metadata':{'annotations':{G+'/result-reuse':'same-inputs'}},'container':{'image':'alpine:3.22','command':['sh','-ec'],'args':['printf "%s" "$ITEM" > /tmp/result'],'env':[{'name':'ITEM','value':'{{inputs.parameters.item}}'}]},'outputs':{'parameters':[{'name':'result','valueFrom':{'path':'/tmp/result'}}]}},
  {'name':'change','inputs':{'parameters':[{'name':'payload'}]},'metadata':{'annotations':{G+'/result-reuse':'same-inputs',G+'/retry-safe':'idempotent'}},'container':{'image':'alpine:3.22','command':['sh','-ec'],'args':['test "$PAYLOAD" = valid; printf "%s" "$PAYLOAD" > /tmp/result'],'env':[{'name':'PAYLOAD','value':'{{inputs.parameters.payload}}'}]},'outputs':{'parameters':[{'name':'result','valueFrom':{'path':'/tmp/result'}}]}}
 ]}
 return [resource('ParameterTemplate',name,model),native('WorkflowTemplate',name,spec),resource('WorkflowParameterBinding',name,{'workflowTemplate':name,'parameterTemplate':name,'mappings':[{'path':'/value','parameter':'value','encoding':'string'},{'path':'/items','parameter':'items','encoding':'json'}]})]

def pod_uids(name):
 pods=json.loads(kubectl('-n',NS,'get','pods','-l','workflows.argoproj.io/workflow='+name,'-o','json'))['items']
 return {p['metadata']['annotations']['workflows.argoproj.io/node-id']:p['metadata']['uid'] for p in pods}

def main():
 for obj in fixture():apply(obj)
 values={'value':'invalid','items':['one','two']}
 saved=api('PUT',PREFIX+'/sets/'+RUN,{'template':RUN,'values':values,'resourceVersion':''})
 previous=saved['spec']['revision']
 preview=api('POST',PREFIX+'/preview',{'binding':RUN,'revision':previous});assert not preview['errors'],preview
 w=api('POST',PREFIX+'/submit',{'binding':RUN,'revision':previous,'executionDigest':preview['executionDigest'],'idempotencyKey':RUN});name=w['metadata']['name'];uid=w['metadata']['uid']
 wait_workflow(name,lambda w:len([n for n in w.get('status',{}).get('nodes',{}).values() if n.get('type')=='Suspend' and n.get('phase')=='Running'])==2)
 api('PUT','/api/v1/workflows/'+NS+'/'+name+'/resume',{})
 before=wait_workflow(name,lambda w:w.get('status',{}).get('phase')=='Failed')
 retained={i:{k:n.get(k) for k in ['phase','finishedAt','inputs','outputs']} for i,n in before['status']['nodes'].items() if n['phase']=='Succeeded'}
 approvals={i for i,n in before['status']['nodes'].items() if n['type']=='Suspend'}
 assert len(approvals)==2 and approvals.issubset(retained)
 before_pods=pod_uids(name)
 kept_pods={i:u for i,u in before_pods.items() if i in retained}
 assert len(kept_pods)==5
 assert len([n for n in before['status']['nodes'].values() if n['type']=='Pod' and n['phase']=='Succeeded'])==5
 saved=api('PUT',PREFIX+'/sets/'+RUN,{'template':RUN,'values':{**values,'value':'valid'},'resourceVersion':saved['metadata']['resourceVersion']})
 revision=saved['spec']['revision']
 diff=api('POST',PREFIX+'/diff',{'left':previous,'right':revision});assert [x['path'] for x in diff['changes']]==['/value']
 request={'workflow':name,'uid':uid,'revision':revision}
 plan=api('POST',PREFIX+'/retry-preview',request)
 assert plan['affectedTasks']==['report','vary/change'],plan
 assert set(retained).issubset(plan['retainedNodes']),plan
 retry={**request,'resourceVersion':plan['resourceVersion'],'planDigest':plan['digest'],'idempotencyKey':RUN+'-changed'}
 result=api('POST',PREFIX+'/retry',retry);assert result['metadata']['uid']==uid
 done=wait_workflow(name,lambda w:w.get('status',{}).get('phase') in ['Succeeded','Failed','Error'])
 assert done['status']['phase']=='Succeeded',done['status']
 for i,evidence in retained.items():assert {k:done['status']['nodes'][i].get(k) for k in evidence}==evidence,(i,done['status']['nodes'][i])
 varying=next(n for n in done['status']['nodes'].values() if n['name']==name+'.vary')
 assert {p['name']:p['value'] for p in varying['inputs']['parameters']}['payload']=='valid',varying
 assert varying['outputs']['parameters'][0]['value']=='valid'
 assert done['metadata']['annotations'][G+'/revision']==revision
 after_pods=pod_uids(name)
 assert all(after_pods[i]==u for i,u in kept_pods.items())
 changed_id=next(i for i,n in done['status']['nodes'].items() if n['name']==name+'.vary.change')
 assert after_pods[changed_id]!=before_pods[changed_id]
 receipt=api('GET',PREFIX+'/catalog/parameterretries?name='+done['metadata']['annotations'][G+'/retry'])
 old_vary=next(n for n in receipt['spec']['previousWorkflow']['status']['nodes'].values() if n['name']==name+'.vary')
 assert {p['name']:p['value'] for p in old_vary['inputs']['parameters']}['payload']=='invalid'
 assert api('POST',PREFIX+'/retry',retry)['metadata']['uid']==uid
 assert len(json.loads(kubectl('-n',NS,'get','workflows','-l','workflows.argoproj.io/workflow-template='+RUN,'-o','json'))['items'])==1
 print('PASS',json.dumps({'workflow':name,'uid':uid,'affected':plan['affectedTasks'],'retainedRecords':len(retained),'preservedApprovals':len(approvals),'retainedPodUIDs':len(kept_pods),'changedPodUID':True,'sameTemplateIndependentCalls':True,'boundaryInputsUpdated':True,'workflowCount':1}),flush=True)

if __name__=='__main__':main()
