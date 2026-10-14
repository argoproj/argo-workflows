"""Save -> native failure -> edit/diff -> impact preview -> SAME Workflow retry.
Synthetic tasks only, preserving approval and successful loop outputs.
"""
import copy,json,uuid
from e2e import api,wait_workflow,PREFIX,NS
from template_refs_e2e import resource,native,apply,kubectl
RUN='partial-'+uuid.uuid4().hex[:8]
G='parameters.workflows.example.io'
def main():
 apply(resource('ParameterTemplate',RUN,{'dialect':'http://json-schema.org/draft-07/schema#','schema':{'type':'object','properties':{'value':{'type':'string'},'valueExtra':{'type':'string'},'items':{'type':'array','items':{'type':'string'},'minItems':1}},'required':['value','valueExtra','items'],'additionalProperties':False}}))
 apply({'apiVersion':'v1','kind':'ConfigMap','metadata':{'name':RUN+'-retry-gate','namespace':NS},'data':{'allow':'false'}})
 root=native('WorkflowTemplate',RUN,{'entrypoint':'main','serviceAccountName':'workflow-executor','arguments':{'parameters':[{'name':'value'},{'name':'valueExtra'},{'name':'items'}]},'volumes':[{'name':'retry-gate','configMap':{'name':RUN+'-retry-gate'}}],'templates':[
 {'name':'main','dag':{'tasks':[
  {'name':'approve','template':'approve'},
  {'name':'keep','template':'keep','dependencies':['approve'],'withParam':'{{workflow.parameters.items}}','arguments':{'parameters':[{'name':'item','value':'{{item}}'},{'name':'stable','value':'{{workflow.parameters.valueExtra}}'}]}},
  {'name':'change','template':'change','dependencies':['keep'],'arguments':{'parameters':[{'name':'value','value':"{{=workflow['parameters']['value']}}"}]}},
  {'name':'report','template':'report','dependencies':['change'],'arguments':{'parameters':[{'name':'value','value':'{{tasks.change.outputs.parameters.echo}}'}]}}
 ]}},
 {'name':'approve','suspend':{}},
 {'name':'keep','inputs':{'parameters':[{'name':'item'},{'name':'stable'}]},'metadata':{'annotations':{G+'/result-reuse':'same-inputs','example/description':'workflow.parameters.value is literal documentation'}},'container':{'image':'alpine:3.22','command':['sh','-ec'],'args':['test "$STABLE" = unchanged; printf "%s" "$ITEM" > /tmp/result'],'env':[{'name':'ITEM','value':'{{inputs.parameters.item}}'},{'name':'STABLE','value':'{{inputs.parameters.stable}}'}]},'outputs':{'parameters':[{'name':'echo','valueFrom':{'path':'/tmp/result'}}]}},
 {'name':'change','inputs':{'parameters':[{'name':'value'}]},'metadata':{'annotations':{G+'/retry-safe':'idempotent'}},'container':{'image':'alpine:3.22','command':['sh','-ec'],'args':['test "$VALUE" = valid; test "$(cat /gate/allow)" = true; printf "%s" "$VALUE" > /tmp/result'],'env':[{'name':'VALUE','value':'{{inputs.parameters.value}}'}],'volumeMounts':[{'name':'retry-gate','mountPath':'/gate','readOnly':True}]},'outputs':{'parameters':[{'name':'echo','valueFrom':{'path':'/tmp/result'}}]}},
 {'name':'report','inputs':{'parameters':[{'name':'value'}]},'metadata':{'annotations':{G+'/retry-safe':'idempotent'}},'container':{'image':'alpine:3.22','command':['sh','-ec'],'args':['test "$VALUE" = valid'],'env':[{'name':'VALUE','value':'{{inputs.parameters.value}}'}]}}
 ]})
 apply(root)
 apply(resource('WorkflowParameterBinding',RUN,{'workflowTemplate':RUN,'parameterTemplate':RUN,'mappings':[{'path':'/value','parameter':'value','encoding':'string'},{'path':'/valueExtra','parameter':'valueExtra','encoding':'string'},{'path':'/items','parameter':'items','encoding':'json'}]}))
 s=api('PUT',PREFIX+'/sets/'+RUN,{'template':RUN,'values':{'value':'invalid','valueExtra':'unchanged','items':['one','two','three']},'resourceVersion':''});original_revision=s['spec']['revision']
 p=api('POST',PREFIX+'/preview',{'binding':RUN,'revision':original_revision})
 w=api('POST',PREFIX+'/submit',{'binding':RUN,'revision':original_revision,'executionDigest':p['executionDigest'],'idempotencyKey':RUN});name=w['metadata']['name'];uid=w['metadata']['uid']
 wait_workflow(name,lambda x:any(n.get('type')=='Suspend' and n.get('phase')=='Running' for n in x.get('status',{}).get('nodes',{}).values()))
 api('PUT','/api/v1/workflows/'+NS+'/'+name+'/resume',{})
 failed=wait_workflow(name,lambda x:x.get('status',{}).get('phase')=='Failed')
 retained={key:(n['phase'],n.get('finishedAt'),n.get('outputs')) for key,n in failed['status']['nodes'].items() if n['type'] in ['Pod','Suspend'] and n['phase']=='Succeeded'}
 assert len(retained)==4
 context=api('GET',PREFIX+'/edit-context/'+name+'?uid='+uid)
 s=api('PUT',PREFIX+'/sets/'+RUN,{'template':RUN,'values':{'value':'valid','valueExtra':'unchanged','items':['one','two','three']},'resourceVersion':context['set']['metadata']['resourceVersion']})
 new_revision=s['spec']['revision']
 diff=api('POST',PREFIX+'/diff',{'left':original_revision,'right':new_revision});assert [c['path'] for c in diff['changes']]==['/value']
 req={'workflow':name,'uid':uid,'revision':new_revision}
 plan=api('POST',PREFIX+'/retry-preview',req);assert plan['affectedTasks']==['change','report'];assert set(retained).issubset(plan['retainedNodes'])
 # Preview is read-only and refuses structural/identity changes.
 assert api('GET','/api/v1/workflows/'+NS+'/'+name)['status']['phase']=='Failed'
 topology=api('PUT',PREFIX+'/sets/'+RUN,{'template':RUN,'values':{'value':'valid','valueExtra':'unchanged','items':['one','new']},'resourceVersion':s['metadata']['resourceVersion']})
 api('POST',PREFIX+'/retry-preview',{**req,'revision':topology['spec']['revision']},400)
 retry={**req,'resourceVersion':plan['resourceVersion'],'planDigest':plan['digest'],'idempotencyKey':RUN+'-retry'}
 api('POST',PREFIX+'/retry',{**retry,'planDigest':'stale'},400)
 result=api('POST',PREFIX+'/retry',retry);assert result['metadata']['uid']==uid
 second_failure=wait_workflow(name,lambda x:x.get('status',{}).get('phase')=='Failed')
 assert second_failure['metadata']['annotations'][G+'/revision']==new_revision
 for key,value in retained.items():
  n=second_failure['status']['nodes'][key];assert (n['phase'],n.get('finishedAt'),n.get('outputs'))==value
 partial_receipt=second_failure['metadata']['annotations'][G+'/retry']
 apply({'apiVersion':'v1','kind':'ConfigMap','metadata':{'name':RUN+'-retry-gate','namespace':NS},'data':{'allow':'true'}})
 original_retry={'workflow':name,'uid':uid,'resourceVersion':second_failure['metadata']['resourceVersion'],'idempotencyKey':RUN+'-original-after-new'}
 api('POST',PREFIX+'/retry',original_retry)
 done=wait_workflow(name,lambda x:x.get('status',{}).get('phase')=='Succeeded')
 assert done['metadata']['uid']==uid and done['metadata']['annotations'][G+'/revision']==new_revision
 for key,value in retained.items():
  n=done['status']['nodes'][key];assert (n['phase'],n.get('finishedAt'),n.get('outputs'))==value
 duplicate=api('POST',PREFIX+'/retry',original_retry);assert duplicate['metadata']['uid']==uid and duplicate['status']['phase']=='Succeeded'
 receipt=api('GET',PREFIX+'/catalog/parameterretries?name='+partial_receipt)
 assert receipt['spec']['previousRevision']==original_revision and receipt['spec']['previousWorkflow']['status']['phase']=='Failed'
 assert receipt['spec']['definitions']['parameterBinding']['workflowTemplate']==RUN
 assert receipt['spec']['plan']['executionDigest']
 original_receipt=api('GET',PREFIX+'/catalog/parameterretries?name='+done['metadata']['annotations'][G+'/retry'])
 assert original_receipt['spec']['definitions']==receipt['spec']['definitions']
 compared=api('POST',PREFIX+'/execution-diff',{'source':{'workflow':name,'uid':uid},'revision':new_revision})
 assert all(layer['available'] and not layer['changes'] for layer in compared['layers'])
 assert not compared['parameters']['changes']
 edit=api('GET',PREFIX+'/edit-context/'+name+'?uid='+uid);assert edit['revision']['metadata']['name']==new_revision
 listed=json.loads(kubectl('-n',NS,'get','workflows','-l','workflows.argoproj.io/workflow-template='+RUN,'-o','json'))['items'];assert len(listed)==1
 print('PASS',json.dumps({'run':RUN,'workflow':name,'uid':uid,'retainedResultsAndApproval':4,'affectedTasks':plan['affectedTasks'],'workflowCount':1,'mixedNewThenOriginalRetry':True,'exactNamesAndBracketReferences':True,'oldRevision':original_revision,'newRevision':new_revision}),flush=True)
if __name__=='__main__':main()
