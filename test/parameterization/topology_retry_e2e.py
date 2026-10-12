"""Changed fan-out shape in one Workflow; untouched approvals and Pod UIDs retained."""
import copy,json,uuid
from e2e import api,wait_workflow,PREFIX,NS
from template_refs_e2e import resource,native,apply,kubectl
from nested_retry_e2e import pod_uids
RUN='topology-retry-'+uuid.uuid4().hex[:8]
G='parameters.workflows.example.io'
def fixture(name=RUN,steps=False):
 model={'dialect':'http://json-schema.org/draft-07/schema#','schema':{'type':'object','properties':{'items':{'type':'array','items':{'type':'string'},'maxItems':8}},'required':['items'],'additionalProperties':False}}
 args=lambda value:{'parameters':[{'name':'value','value':value}]}
 calls=[{'name':'approve','template':'approve'},{'name':'stable','template':'echo','arguments':args('stable')},{'name':'expand','template':'echo','withParam':'{{workflow.parameters.items}}','arguments':args('{{item}}')},{'name':'report','template':'report','arguments':args('{{'+('steps' if steps else 'tasks')+'.expand.outputs.parameters.value}}')}]
 root={'name':'main','metadata':{'annotations':{G+'/retry-loop-calls':'["expand"]'}}}
 if steps:root['steps']=[[c] for c in calls]
 else:
  calls[1]['dependencies']=['approve'];calls[2]['dependencies']=['stable'];calls[3]['dependencies']=['expand'];root['dag']={'tasks':calls}
 leaf={'name':'echo','metadata':{'annotations':{G+'/retry-safe':'idempotent',G+'/result-reuse':'same-inputs'}},'inputs':{'parameters':[{'name':'value'}]},'container':{'image':'alpine:3.22','command':['sh','-ec'],'args':['printf "%s" "$VALUE" >/tmp/value'],'env':[{'name':'VALUE','value':'{{inputs.parameters.value}}'}]},'outputs':{'parameters':[{'name':'value','valueFrom':{'path':'/tmp/value'}}]}}
 # Native empty fan-outs are Skipped and have no aggregate output. The
 # consumer explicitly declares the empty list it expects in that case.
 report=copy.deepcopy(leaf);report['name']='report';report['inputs']['parameters'][0]['default']='[]'
 return [resource('ParameterTemplate',name,model),native('WorkflowTemplate',name,{'entrypoint':'main','serviceAccountName':'workflow-executor','arguments':{'parameters':[{'name':'items'}]},'templates':[root,{'name':'approve','suspend':{}},leaf,report]}),resource('WorkflowParameterBinding',name,{'workflowTemplate':name,'parameterTemplate':name,'mappings':[{'path':'/items','parameter':'items','encoding':'json'}]})]
def main():
 evidence=[]
 for steps in [False,True]:
  prefix=RUN+('-steps' if steps else '-dag')
  for obj in fixture(prefix,steps):apply(obj)
  saved=api('PUT',PREFIX+'/sets/'+prefix,{'template':prefix,'values':{'items':['a','b']},'resourceVersion':''})
  preview=api('POST',PREFIX+'/preview',{'binding':prefix,'revision':saved['spec']['revision']});assert not preview['errors'],preview
  wf=api('POST',PREFIX+'/submit',{'binding':prefix,'revision':saved['spec']['revision'],'executionDigest':preview['executionDigest'],'idempotencyKey':prefix});name=wf['metadata']['name'];uid=wf['metadata']['uid']
  wait_workflow(name,lambda w:any(n.get('type')=='Suspend' and n.get('phase')=='Running' for n in w.get('status',{}).get('nodes',{}).values()));api('PUT','/api/v1/workflows/'+NS+'/'+name+'/resume',{})
  done=wait_workflow(name,lambda w:w.get('status',{}).get('phase')=='Succeeded')
  retained={i:n for i,n in done['status']['nodes'].items() if n.get('displayName') in ['approve','stable']};pods=pod_uids(name)
  for attempt,items in enumerate([['b','c','d'],[],['restored']]):
   saved=api('PUT',PREFIX+'/sets/'+prefix,{'template':prefix,'values':{'items':items},'resourceVersion':saved['metadata']['resourceVersion']})
   req={'workflow':name,'uid':uid,'revision':saved['spec']['revision']};plan=api('POST',PREFIX+'/retry-preview',req)
   assert plan['affectedTasks']==['expand','report'],plan
   retry={**req,'resourceVersion':plan['resourceVersion'],'planDigest':plan['digest'],'idempotencyKey':prefix+'-'+str(attempt)}
   api('POST',PREFIX+'/retry',retry)
   done=wait_workflow(name,lambda w:w.get('status',{}).get('phase')=='Succeeded')
   report_node=next(n for n in done['status']['nodes'].values() if n.get('displayName')=='report')
   assert json.loads(report_node['outputs']['parameters'][0]['value'])==items
   assert done['metadata']['uid']==uid
   for i,old in retained.items():
    assert all(done['status']['nodes'][i].get(k)==old.get(k) for k in ['phase','startedAt','finishedAt','inputs','outputs'])
    if i in pods:assert pod_uids(name)[i]==pods[i]
   iterations=[n for n in done['status']['nodes'].values() if n['type']=='Pod' and n.get('displayName','').startswith('expand(')]
   assert len(iterations)==len(items),(items,iterations)
   assert sorted(n['outputs']['parameters'][0]['value'] for n in iterations)==sorted(items)
   assert api('POST',PREFIX+'/retry',retry)['metadata']['uid']==uid
  evidence.append({'workflow':name,'uid':uid,'steps':steps,'topologies':[2,3,0,1],'preservedApprovalAndPod':True})
 print('PASS',json.dumps(evidence),flush=True)
if __name__=='__main__':main()
