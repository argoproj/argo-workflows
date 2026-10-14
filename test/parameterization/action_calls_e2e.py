"""Named invocation -> native adapter -> actual outputs -> same Workflow partial retry."""
import copy,json,uuid
from e2e import api,wait_workflow,PREFIX,NS
from template_refs_e2e import resource,native,apply,kubectl
from nested_retry_e2e import pod_uids
RUN='action-calls-'+uuid.uuid4().hex[:8]
G='parameters.workflows.example.io'

def fixture(name=RUN,steps=False):
 def args(value):return {'parameters':[{'name':'message','value':value}]}
 def leaf(name='base',action=False):
  source='value' if action else 'message';target='answer' if action else 'reply'
  return {'name':name,'inputs':{'parameters':[{'name':source}]},'metadata':{'annotations':{G+'/result-reuse':'same-inputs',G+'/retry-safe':'idempotent'}},'container':{'image':'alpine:3.22','command':['sh','-ec'],'args':['test "$VALUE" != invalid; printf "'+('action' if action else 'base')+':%s" "$VALUE" >/tmp/result'],'env':[{'name':'VALUE','value':'{{inputs.parameters.'+source+'}}'}]},'outputs':{'parameters':[{'name':target,'valueFrom':{'path':'/tmp/result'}}]}}
 task={'name':'emit','template':'base','arguments':args('{{inputs.parameters.message}}')}
 group={'name':'group','inputs':{'parameters':[{'name':'message'}]},'metadata':{'annotations':{G+'/bindable-calls':'["emit"]'}},'outputs':{'parameters':[{'name':'reply','valueFrom':{'parameter':'{{'+('steps' if steps else 'tasks')+'.emit.outputs.parameters.reply}}'}}]}}
 if steps:group['steps']=[[task]]
 else:group['dag']={'tasks':[task]}
 spec={'entrypoint':'main','serviceAccountName':'workflow-executor','arguments':{'parameters':[{'name':'message'}]},'templates':[
 {'name':'main','dag':{'failFast':False,'tasks':[{'name':'approve','template':'approve'},{'name':'east','dependencies':['approve'],'template':'group','arguments':args('{{workflow.parameters.message}}')},{'name':'west','dependencies':['approve'],'template':'group','arguments':args('unchanged')},{'name':'report','dependencies':['east'],'template':'base','arguments':args('{{tasks.east.outputs.parameters.reply}}')}]}},
 {'name':'approve','suspend':{}},group,leaf()]}
 schema=lambda prop:{'type':'object','properties':{prop:{'type':'string'}},'required':[prop],'additionalProperties':False}
 return [resource('ParameterTemplate',name,{'dialect':'http://json-schema.org/draft-07/schema#','schema':schema('message')}),native('WorkflowTemplate',name+'-impl',{'templates':[leaf('implementation',True)]}),native('WorkflowTemplate',name,spec),resource('ActionDefinition',name,{'inputSchema':schema('value'),'outputSchema':schema('answer'),'implementationRef':{'name':name+'-impl','template':'implementation'}}),resource('WorkflowActionBinding',name,{'workflowTemplate':name,'calls':[{'callPath':['main','east','emit'],'actionDefinition':name,'inputMappings':[{'from':'message','to':'value'}],'outputMappings':[{'from':'answer','to':'reply'}]}]}),resource('WorkflowParameterBinding',name,{'workflowTemplate':name,'parameterTemplate':name,'actionBinding':name,'mappings':[{'path':'/message','parameter':'message','encoding':'string'}]})]

def main():
 evidence=[]
 for steps in [False,True]:
  prefix=RUN+('-steps' if steps else '-dag')
  objects=fixture(prefix,steps)
  for obj in objects:apply(obj)
  kubectl('-n',NS,'wait','--for=condition=DefinitionValid','workflowparameterbinding/'+prefix,'--timeout=45s')
  saved=api('PUT',PREFIX+'/sets/'+prefix,{'template':prefix,'values':{'message':'valid' if steps else 'invalid'},'resourceVersion':''})
  preview=api('POST',PREFIX+'/preview',{'binding':prefix,'revision':saved['spec']['revision']})
  assert not preview['errors'],preview
  original=json.loads(kubectl('-n',NS,'get','workflowtemplate',prefix,'-o','json'))
  assert original['spec']==objects[2]['spec'],'shared source must not be changed'
  w=api('POST',PREFIX+'/submit',{'binding':prefix,'revision':saved['spec']['revision'],'executionDigest':preview['executionDigest'],'idempotencyKey':prefix})
  name=w['metadata']['name'];uid=w['metadata']['uid']
  wait_workflow(name,lambda w:any(n.get('type')=='Suspend' and n.get('phase')=='Running' for n in w.get('status',{}).get('nodes',{}).values()))
  api('PUT','/api/v1/workflows/'+NS+'/'+name+'/resume',{})
  before=wait_workflow(name,lambda w:w.get('status',{}).get('phase') in ['Succeeded','Failed','Error'])
  if steps:
   assert before['status']['phase']=='Succeeded',before['status'];done=before;preserved={}
  else:
   assert before['status']['phase']=='Failed',before['status']
   retained={i:{k:n.get(k) for k in ['phase','finishedAt','inputs','outputs']} for i,n in before['status']['nodes'].items() if n['phase']=='Succeeded'}
   oldpods=pod_uids(name);preserved={i:v for i,v in oldpods.items() if i in retained}
   assert len(preserved)==1 and any(n['type']=='Suspend' and n['phase']=='Succeeded' for n in before['status']['nodes'].values())
   saved=api('PUT',PREFIX+'/sets/'+prefix,{'template':prefix,'values':{'message':'fixed'},'resourceVersion':saved['metadata']['resourceVersion']})
   request={'workflow':name,'uid':uid,'revision':saved['spec']['revision']}
   plan=api('POST',PREFIX+'/retry-preview',request)
   assert all('west' not in p and 'approve' not in p for p in plan['affectedTasks']),plan
   retry={**request,'resourceVersion':plan['resourceVersion'],'planDigest':plan['digest'],'idempotencyKey':prefix+'-retry'}
   api('POST',PREFIX+'/retry',retry)
   done=wait_workflow(name,lambda w:w.get('status',{}).get('phase') in ['Succeeded','Failed','Error'])
   assert done['status']['phase']=='Succeeded',done['status']
   for i,data in retained.items():assert {k:done['status']['nodes'][i].get(k) for k in data}==data
   assert all(pod_uids(name)[i]==v for i,v in preserved.items())
   assert api('POST',PREFIX+'/retry',retry)['metadata']['uid']==uid
  assert done['metadata']['uid']==uid
  for suffix,expected in [('east','action:'+('valid' if steps else 'fixed')),('west','base:unchanged'),('report','base:action:'+('valid' if steps else 'fixed'))]:
   node=next(n for n in done['status']['nodes'].values() if n['name']==name+'.'+suffix)
   assert node['outputs']['parameters'][0]['value']==expected,node
  assert len(json.loads(kubectl('-n',NS,'get','workflows','-l','workflows.argoproj.io/workflow-template='+prefix,'-o','json'))['items'])==1
  evidence.append({'workflow':name,'uid':uid,'steps':steps,'unrelatedPodUIDsRetained':len(preserved),'outputsVerified':True})
 print('PASS',json.dumps(evidence),flush=True)
if __name__=='__main__':main()
