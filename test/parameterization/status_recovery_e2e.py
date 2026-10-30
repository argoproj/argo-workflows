"""Local real-controller recovery after a rejected completion write.
The temporary webhook matches only this synthetic Workflow's terminal updates.
No Workflow retry, status patch, or controller restart is used to recover it.
"""
import json,time,uuid
from e2e import api,PREFIX,NS,wait_workflow
from template_refs_e2e import apply,resource,native,kubectl
RUN='write-recovery-'+uuid.uuid4().hex[:8]

def main():
 apply(resource('ParameterTemplate',RUN,{'dialect':'http://json-schema.org/draft-07/schema#','schema':{'type':'object','properties':{'count':{'type':'integer'}},'required':['count']}}))
 apply(native('WorkflowTemplate',RUN,{'entrypoint':'apply','suspend':True,'serviceAccountName':'workflow-executor','arguments':{'parameters':[{'name':'count'}]},'templates':[{'name':'apply','resource':{'action':'create','manifest':'apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: {{workflow.name}}-result\ndata:\n  count: "{{workflow.parameters.count}}"\n'}}]}))
 apply(resource('WorkflowParameterBinding',RUN,{'workflowTemplate':RUN,'parameterTemplate':RUN,'mappings':[{'path':'/count','parameter':'count','encoding':'decimal'}]}))
 saved=api('PUT',PREFIX+'/sets/'+RUN,{'template':RUN,'values':{'count':17},'resourceVersion':''})
 preview=api('POST',PREFIX+'/preview',{'binding':RUN,'revision':saved['spec']['revision']})
 wf=api('POST',PREFIX+'/submit',{'binding':RUN,'revision':saved['spec']['revision'],'executionDigest':preview['executionDigest'],'idempotencyKey':RUN})
 name=wf['metadata']['name'];uid=wf['metadata']['uid']
 hook={'apiVersion':'admissionregistration.k8s.io/v1','kind':'ValidatingWebhookConfiguration','metadata':{'name':RUN},'webhooks':[{'name':RUN+'.example.test','admissionReviewVersions':['v1'],'sideEffects':'None','failurePolicy':'Fail','timeoutSeconds':2,'namespaceSelector':{'matchLabels':{'kubernetes.io/metadata.name':NS}},'matchConditions':[{'name':'this-completion-only','expression':"object.metadata.name == '"+name+"' && has(object.status) && has(object.status.phase) && object.status.phase == 'Succeeded'"}],'rules':[{'apiGroups':['argoproj.io'],'apiVersions':['v1alpha1'],'operations':['UPDATE'],'resources':['workflows'],'scope':'Namespaced'}],'clientConfig':{'service':{'namespace':NS,'name':RUN+'-unavailable','path':'/validate','port':443}}}]}
 injected=False
 try:
  apply(hook);injected=True
  api('PUT','/api/v1/workflows/'+NS+'/'+name+'/resume',{})
  pod=None
  for _ in range(90):
   pods=json.loads(kubectl('-n',NS,'get','pods','-l','workflows.argoproj.io/workflow='+name,'-o','json'))['items']
   if len(pods)==1 and pods[0].get('status',{}).get('phase')=='Succeeded':pod=pods[0];break
   time.sleep(1)
  assert pod,'synthetic action did not complete'
  result=json.loads(kubectl('-n',NS,'get','configmap',name+'-result','-o','json'))
  assert result['data']=={'count':'17'}
  # Observe an actual rejected write, rather than assuming the webhook fired.
  rejected=False
  for _ in range(35):
   logs=kubectl('-n',NS,'logs','deployment/workflow-controller','--since=2m').decode()
   if any(name in line and RUN+'.example.test' in line and 'Error updating workflow' in line for line in logs.splitlines()):rejected=True;break
   time.sleep(1)
  assert rejected,'completion write failure was not observed'
  assert api('GET','/api/v1/workflows/'+NS+'/'+name)['status']['phase']=='Running'
 finally:
  if injected:kubectl('delete','validatingwebhookconfiguration',RUN)
 complete=wait_workflow(name,lambda w:w.get('status',{}).get('phase')=='Succeeded',timeout=60)
 pods=json.loads(kubectl('-n',NS,'get','pods','-l','workflows.argoproj.io/workflow='+name,'-o','json'))['items']
 assert len(pods)==1 and pods[0]['metadata']['uid']==pod['metadata']['uid'],'must not execute the action twice'
 assert json.loads(kubectl('-n',NS,'get','configmap',name+'-result','-o','json'))['metadata']['uid']==result['metadata']['uid']
 assert complete['metadata']['uid']==uid
 print('PASS '+json.dumps({'workflow':name,'phase':'Succeeded','rejectedCompletionObserved':True,'workflowRetryRequests':0,'controllerRestarts':0,'sameWorkflowUID':uid,'samePodUID':pod['metadata']['uid'],'sameResultUID':result['metadata']['uid']}),flush=True)
if __name__=='__main__':main()
