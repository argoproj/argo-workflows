"""Latest version index -> pinned native SecretKeyRef -> reviewed in-place rotation."""
import json,uuid
from e2e import api,wait_workflow,PREFIX,NS
from template_refs_e2e import resource,native,apply,kubectl
from nested_retry_e2e import pod_uids
RUN='secret-versions-'+uuid.uuid4().hex[:8]
G='parameters.workflows.example.io'

def fixture(name=RUN):
 ref={'type':'object','properties':{'secretRef':{'type':'object','properties':{'name':{'type':'string'},'key':{'type':'string'}},'required':['name','key'],'additionalProperties':False},'latestSecretRef':{'type':'object','properties':{'configMap':{'type':'string'},'key':{'type':'string'}},'required':['configMap','key'],'additionalProperties':False}},'oneOf':[{'required':['secretRef']},{'required':['latestSecretRef']}],'additionalProperties':False}
 model={'dialect':'http://json-schema.org/draft-07/schema#','schema':{'type':'object','properties':{'credential':ref},'required':['credential'],'additionalProperties':False},'sensitivePaths':['/credential'],'secretSchemas':{'/credential':{'type':'string','minLength':5}}}
 consume={'name':'consume','inputs':{'parameters':[{'name':'credential'}]},'metadata':{'annotations':{G+'/retry-safe':'idempotent',G+'/result-reuse':'same-inputs'}},'container':{'image':'alpine:3.22','command':['sh','-ec'],'args':['test "${#TOKEN}" -gt 25; echo credential-consumed'],'env':[{'name':'TOKEN','valueFrom':{'secretKeyRef':{'name':"{{=jsonpath(inputs.parameters.credential, '$.secretRef.name')}}",'key':"{{=jsonpath(inputs.parameters.credential, '$.secretRef.key')}}"}}}]}}
 spec={'entrypoint':'main','serviceAccountName':'workflow-executor','arguments':{'parameters':[{'name':'credential'}]},'templates':[{'name':'main','dag':{'tasks':[{'name':'approve','template':'approve'},{'name':'consume','dependencies':['approve'],'template':'consume','arguments':{'parameters':[{'name':'credential','value':'{{workflow.parameters.credential}}'}]}}]}},{'name':'approve','suspend':{}},consume]}
 return [resource('ParameterTemplate',name,model),native('WorkflowTemplate',name,spec),resource('WorkflowParameterBinding',name,{'workflowTemplate':name,'parameterTemplate':name,'mappings':[{'path':'/credential','parameter':'credential','encoding':'secret-reference-json'}]})]

def main():
 for obj in fixture():apply(obj)
 credentials={'v1':'synthetic-short','v2':'synthetic-'+uuid.uuid4().hex}
 for suffix in ['v1','v2']:
  apply({'apiVersion':'v1','kind':'Secret','metadata':{'name':RUN+'-'+suffix,'namespace':NS},'immutable':True,'stringData':{'token':credentials[suffix]}})
 def point(version):
  apply({'apiVersion':'v1','kind':'ConfigMap','metadata':{'name':RUN,'namespace':NS},'data':{'current':json.dumps({'name':RUN+'-'+version,'key':'token'})}})
 point('v1')
 values={'credential':{'latestSecretRef':{'configMap':RUN,'key':'current'}}}
 saved=api('PUT',PREFIX+'/sets/'+RUN,{'template':RUN,'values':values,'resourceVersion':''});revision=saved['spec']['revision']
 preview=api('POST',PREFIX+'/preview',{'binding':RUN,'revision':revision})
 point('v2')
 submit={'binding':RUN,'revision':revision,'executionDigest':preview['executionDigest'],'idempotencyKey':RUN}
 api('POST',PREFIX+'/submit',submit,409)
 point('v1');preview=api('POST',PREFIX+'/preview',{'binding':RUN,'revision':revision});submit['executionDigest']=preview['executionDigest']
 w=api('POST',PREFIX+'/submit',submit);name=w['metadata']['name'];uid=w['metadata']['uid']
 wait_workflow(name,lambda w:any(n.get('type')=='Suspend' and n.get('phase')=='Running' for n in w.get('status',{}).get('nodes',{}).values()))
 point('v2')
 api('PUT','/api/v1/workflows/'+NS+'/'+name+'/resume',{})
 failed=wait_workflow(name,lambda w:w.get('status',{}).get('phase')=='Failed')
 approval=next((i,n) for i,n in failed['status']['nodes'].items() if n['type']=='Suspend')
 # Original-parameter retry still consumes v1 even though latest now points at v2.
 request={'workflow':name,'uid':uid,'resourceVersion':failed['metadata']['resourceVersion'],'idempotencyKey':RUN+'-original'}
 api('POST',PREFIX+'/retry',request)
 again=wait_workflow(name,lambda w:w.get('status',{}).get('phase')=='Failed' and w.get('metadata',{}).get('annotations',{}).get(G+'/retry') is not None)
 assert RUN+'-v1' in again['spec']['arguments']['parameters'][0]['value']
 # Same saved revision, explicitly re-resolved for a reviewed parameter retry.
 request={'workflow':name,'uid':uid,'revision':revision}
 plan=api('POST',PREFIX+'/retry-preview',request)
 assert plan['affectedTasks']==['consume'],plan
 retry={**request,'resourceVersion':plan['resourceVersion'],'planDigest':plan['digest'],'idempotencyKey':RUN+'-rotation'}
 api('POST',PREFIX+'/retry',retry)
 done=wait_workflow(name,lambda w:w.get('status',{}).get('phase')=='Succeeded')
 assert done['metadata']['uid']==uid
 assert {k:done['status']['nodes'][approval[0]].get(k) for k in ['id','phase','startedAt','finishedAt','inputs','outputs','message']}=={k:approval[1].get(k) for k in ['id','phase','startedAt','finishedAt','inputs','outputs','message']}
 assert RUN+'-v2' in done['spec']['arguments']['parameters'][0]['value']
 receipt=api('GET',PREFIX+'/catalog/parameterretries?name='+done['metadata']['annotations'][G+'/retry'])
 assert any(k.startswith('latestSecret/') for k in receipt['spec']['definitions']['versions'])
 assert 'secret/'+RUN+'-v2' in receipt['spec']['definitions']['versions']
 history=api('GET',PREFIX+'/history/'+RUN)
 assert credentials['v1'] not in json.dumps([saved,preview,history,receipt,done])
 assert credentials['v2'] not in json.dumps([saved,preview,history,receipt,done])
 # Identical retries recover the frozen receipt even if the alias disappears.
 kubectl('-n',NS,'delete','configmap',RUN)
 assert api('POST',PREFIX+'/retry',retry)['metadata']['uid']==uid
 assert api('POST',PREFIX+'/submit',submit)['metadata']['uid']==uid
 assert len(json.loads(kubectl('-n',NS,'get','workflows','-l','workflows.argoproj.io/workflow-template='+RUN,'-o','json'))['items'])==1
 print('PASS',json.dumps({'workflow':name,'uid':uid,'staleLatestRejected':True,'originalRetryPinned':True,'reviewedRotationSameRevision':True,'approvalRetained':True,'receiptSurvivesIndexDeletion':True}),flush=True)
if __name__=='__main__':main()
