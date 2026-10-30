"""Steps and fixed nested loops: saved parameters -> native partial retry -> retained evidence."""
import copy,json,uuid
from e2e import api,wait_workflow,PREFIX,NS
from template_refs_e2e import apply,kubectl
from nested_retry_e2e import fixture as nested_fixture,pod_uids
RUN='steps-retry-'+uuid.uuid4().hex[:8]

def fixture(name=RUN,mode='steps'):
 objects=nested_fixture(name);spec=objects[1]['spec']
 if mode=='steps':
  for template in spec['templates']:
   if 'dag' not in template:continue
   template['steps']=[]
   for task in template.pop('dag')['tasks']:
    task.pop('dependencies',None)
    template['steps'].append([task])
   if 'outputs' in template:template['outputs']=json.loads(json.dumps(template['outputs']).replace('tasks.','steps.'))
  spec=json.loads(json.dumps(spec).replace('tasks.vary.','steps.vary.'))
  # Run stable and varying invocations in one parallel group; their own Steps
  # sequences stay native. Keep the successful stable branch throughout retry.
  spec['templates'][0]['steps']=[spec['templates'][0]['steps'][0]+spec['templates'][0]['steps'][1],spec['templates'][0]['steps'][2]]
  objects[1]['spec']=spec
 else:
  spec['templates'][0]['dag']['tasks'][1]['withItems']=['alpha','beta']
  # Each invocation receives the same updated payload; its iteration identity
  # stays unchanged. Downstream consumes the native aggregate as JSON.
  spec['templates'][0]['dag']['tasks'][2]['arguments']['parameters'][0]['value']='{{=jsonpath(tasks.vary.outputs.parameters.result, "$[0]")}}'
 return objects

def main():
 evidence=[]
 for mode in ['steps','nested-loop']:
  prefix=RUN+'-'+mode
  for obj in fixture(prefix,mode):apply(obj)
  values={'value':'invalid','items':['one','two']}
  saved=api('PUT',PREFIX+'/sets/'+prefix,{'template':prefix,'values':values,'resourceVersion':''})
  preview=api('POST',PREFIX+'/preview',{'binding':prefix,'revision':saved['spec']['revision']})
  w=api('POST',PREFIX+'/submit',{'binding':prefix,'revision':saved['spec']['revision'],'executionDigest':preview['executionDigest'],'idempotencyKey':prefix})
  name=w['metadata']['name'];uid=w['metadata']['uid'];expected_approvals=2 if mode=='steps' else 3
  wait_workflow(name,lambda w:len([n for n in w.get('status',{}).get('nodes',{}).values() if n.get('type')=='Suspend' and n.get('phase')=='Running'])==expected_approvals)
  api('PUT','/api/v1/workflows/'+NS+'/'+name+'/resume',{})
  before=wait_workflow(name,lambda w:w.get('status',{}).get('phase') in ['Failed','Error'])
  assert before['status']['phase']=='Failed',before['status']
  retained={i:{k:n.get(k) for k in ['phase','finishedAt','inputs','outputs']} for i,n in before['status']['nodes'].items() if n['phase']=='Succeeded'}
  approvals={i for i,n in before['status']['nodes'].items() if n['type']=='Suspend'}
  assert len(approvals)==expected_approvals and approvals.issubset(retained)
  oldpods=pod_uids(name);preserved={i:v for i,v in oldpods.items() if i in retained}
  saved=api('PUT',PREFIX+'/sets/'+prefix,{'template':prefix,'values':{**values,'value':'valid'},'resourceVersion':saved['metadata']['resourceVersion']})
  request={'workflow':name,'uid':uid,'revision':saved['spec']['revision']}
  plan=api('POST',PREFIX+'/retry-preview',request)
  assert plan['affectedTasks']==['report','vary/change'],plan
  assert set(retained).issubset(plan['retainedNodes']),set(retained)-set(plan['retainedNodes'])
  retry={**request,'resourceVersion':plan['resourceVersion'],'planDigest':plan['digest'],'idempotencyKey':prefix+'-changed'}
  api('POST',PREFIX+'/retry',retry)
  done=wait_workflow(name,lambda w:w.get('status',{}).get('phase') in ['Succeeded','Failed','Error'])
  assert done['status']['phase']=='Succeeded',done['status']
  assert done['metadata']['uid']==uid
  for i,data in retained.items():assert {k:done['status']['nodes'][i].get(k) for k in data}==data,(i,done['status']['nodes'][i])
  newpods=pod_uids(name);assert all(newpods[i]==v for i,v in preserved.items())
  assert api('POST',PREFIX+'/retry',retry)['metadata']['uid']==uid
  assert len(json.loads(kubectl('-n',NS,'get','workflows','-l','workflows.argoproj.io/workflow-template='+prefix,'-o','json'))['items'])==1
  evidence.append({'workflow':name,'uid':uid,'mode':mode,'retainedRecords':len(retained),'retainedApprovals':len(approvals),'retainedPodUIDs':len(preserved),'affected':plan['affectedTasks']})
 print('PASS',json.dumps(evidence),flush=True)
if __name__=='__main__':main()
