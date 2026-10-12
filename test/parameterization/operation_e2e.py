"""Parameter API -> native Resource -> optional operator -> external receipt -> output.
Only isolated local k3s; external HTTP service is synthetic, never a cloud account.
"""
import copy,json,time,uuid,os
from pathlib import Path
import yaml
from e2e import api,PREFIX,NS
from template_refs_e2e import resource,native,apply,kubectl
RUN='operation-'+uuid.uuid4().hex[:8];GROUP='parameters.workflows.example.io'
def wait(fn,message,timeout=150):
 end=time.monotonic()+timeout
 while time.monotonic()<end:
  r=fn()
  if r:return r
  time.sleep(1)
 raise AssertionError(message)
def get(kind,name):
 return json.loads(kubectl('-n',NS,'get',kind,name,'-o','json'))
def phase(name,expected):
 w=api('GET','/api/v1/workflows/'+NS+'/'+name);return w if w.get('status',{}).get('phase') in expected else None
def operation(w):return get('resourceoperation','ro-'+w['metadata']['uid'])
def stats():
 return json.loads(kubectl('-n',NS,'exec','deployment/'+RUN+'-server','--','python3','-c',"import urllib.request;print(urllib.request.urlopen('http://127.0.0.1:8080/stats').read().decode())"))
def prepare():
 # Separate SA, namespace-scoped observation and status writes; no Workflow writes.
 apply({'apiVersion':'v1','kind':'ServiceAccount','metadata':{'name':RUN,'namespace':NS}})
 rules=[{'apiGroups':[GROUP],'resources':['resourceoperations'],'verbs':['get','list']},{'apiGroups':[GROUP],'resources':['resourceoperations/status'],'verbs':['update']},{'apiGroups':[GROUP],'resources':['workflowsubmissions'],'verbs':['get']},{'apiGroups':[''],'resources':['pods'],'verbs':['get']},{'apiGroups':['argoproj.io'],'resources':['workflows'],'verbs':['get']}]
 apply({'apiVersion':'rbac.authorization.k8s.io/v1','kind':'Role','metadata':{'name':RUN,'namespace':NS},'rules':rules})
 apply({'apiVersion':'rbac.authorization.k8s.io/v1','kind':'RoleBinding','metadata':{'name':RUN,'namespace':NS},'roleRef':{'apiGroup':'rbac.authorization.k8s.io','kind':'Role','name':RUN},'subjects':[{'kind':'ServiceAccount','name':RUN,'namespace':NS}]})
 apply({'apiVersion':'rbac.authorization.k8s.io/v1','kind':'Role','metadata':{'name':RUN+'-invoke','namespace':NS},'rules':[{'apiGroups':[GROUP],'resources':['resourceoperations'],'verbs':['get','list','watch','create','patch','update']}]})
 apply({'apiVersion':'rbac.authorization.k8s.io/v1','kind':'RoleBinding','metadata':{'name':RUN+'-invoke','namespace':NS},'roleRef':{'apiGroup':'rbac.authorization.k8s.io','kind':'Role','name':RUN+'-invoke'},'subjects':[{'kind':'ServiceAccount','name':'workflow-executor','namespace':NS}]})
 for suffix,container,sa in [('server',{'name':'server','image':'python:3.12-alpine','command':['python3','-c',Path('test/parameterization/operation_server.py').read_text()]},None),('controller',{'name':'controller','image':os.environ.get('ARGO_OPERATION_IMAGE','argo-operation-example:local-v35.1-v4.0.1'),'imagePullPolicy':'Never','args':['--namespace='+NS,'--instance='+RUN,'--endpoint=http://'+RUN+'-server:8080','--allow-insecure-demo-http']},RUN)]:
  ps={'containers':[container]}
  if sa:ps['serviceAccountName']=sa
  else:ps['automountServiceAccountToken']=False
  apply({'apiVersion':'apps/v1','kind':'Deployment','metadata':{'name':RUN+'-'+suffix,'namespace':NS},'spec':{'replicas':1,'selector':{'matchLabels':{'app':RUN+'-'+suffix}},'template':{'metadata':{'labels':{'app':RUN+'-'+suffix}},'spec':ps}}})
 apply({'apiVersion':'v1','kind':'Service','metadata':{'name':RUN+'-server','namespace':NS},'spec':{'selector':{'app':RUN+'-server'},'ports':[{'port':8080,'targetPort':8080}]}})
 for name in ['server','controller']:kubectl('-n',NS,'rollout','status','deployment/'+RUN+'-'+name,'--timeout=90s')
def submit(label,mode='normal',delay='2',policy='Detach',deadline=None):
 name=RUN+'-'+label
 model={'type':'object','properties':{'mode':{'type':'string'},'delay':{'type':'string'}},'required':['mode','delay'],'additionalProperties':False}
 apply(resource('ParameterTemplate',name,{'dialect':'http://json-schema.org/draft-07/schema#','schema':model}))
 output={'type':'object','properties':{'reportURI':{'type':'string'}},'required':['reportURI'],'additionalProperties':False}
 manifest=resource('ResourceOperation','ro-{{workflow.uid}}',{'control':'Execute','operation':{'actionRef':{'kind':'ActionDefinition','name':name,'uid':'{{workflow.parameters.actionUID}}','resourceVersion':'{{workflow.parameters.actionRV}}'},'workflowUID':'{{workflow.uid}}','nodeId':'{{workflow.name}}.call','idempotencyKey':'{{workflow.uid}}:{{workflow.name}}.call','attempt':0,'terminationPolicy':policy,'target':{'connectionRef':RUN,'resourceType':'report'},'inputs':{'mode':'{{inputs.parameters.mode}}','delay':'{{inputs.parameters.delay}}'},'outputSchema':output}})
 manifest['metadata']['labels']={GROUP+'/adapter':RUN}
 manifest['metadata']['annotations']={GROUP+'/invoker-pod':'{{pod.name}}',GROUP+'/workflow':'{{workflow.name}}'}
 implementation={'name':'call','inputs':{'parameters':[{'name':'mode'},{'name':'delay'}]},'resource':{'action':'apply','manifest':yaml.safe_dump(manifest,sort_keys=False),'successCondition':'status.phase == Succeeded','failureCondition':'status.phase in (Failed, Cancelled)'},'outputs':{'parameters':[{'name':'reportURI','valueFrom':{'jsonPath':'{.status.outputs.reportURI}'}}]}}
 if deadline:implementation['activeDeadlineSeconds']=deadline
 apply(resource('ActionDefinition',name,{'inputSchema':model,'outputSchema':output,'template':implementation}));action=get('actiondefinition',name)
 wf={'entrypoint':'main','serviceAccountName':'workflow-executor','arguments':{'parameters':[{'name':'mode'},{'name':'delay'},{'name':'actionUID','value':action['metadata']['uid']},{'name':'actionRV','value':action['metadata']['resourceVersion']}]},'templates':[{'name':'main','dag':{'tasks':[{'name':'call','template':'call','arguments':{'parameters':[{'name':n,'value':'{{workflow.parameters.'+n+'}}'} for n in ['mode','delay']]}},{'name':'after','dependencies':['call'],'template':'after','arguments':{'parameters':[{'name':'result','value':'{{tasks.call.outputs.parameters.reportURI}}'}]}}]}},{'name':'call','suspend':{}},{'name':'after','inputs':{'parameters':[{'name':'result'}]},'container':{'image':'alpine:3.22','command':['echo','{{inputs.parameters.result}}']}}]}
 apply(native('WorkflowTemplate',name,wf));apply(resource('WorkflowActionBinding',name,{'workflowTemplate':name,'calls':[{'template':'call','actionDefinition':name}]}));apply(resource('WorkflowParameterBinding',name,{'workflowTemplate':name,'parameterTemplate':name,'actionBinding':name,'mappings':[{'path':'/'+n,'parameter':n,'encoding':'string'} for n in ['mode','delay']]}))
 before_ops=json.loads(kubectl('-n',NS,'get','resourceoperations','-o','json'))['items']
 saved=api('PUT',PREFIX+'/sets/'+name,{'template':name,'values':{'mode':mode,'delay':delay},'resourceVersion':''});preview=api('POST',PREFIX+'/preview',{'binding':name,'revision':saved['spec']['revision']});assert not preview['errors'],preview
 # Saving/previewing cannot create an operation.
 assert {o['metadata']['uid'] for o in before_ops}=={o['metadata']['uid'] for o in json.loads(kubectl('-n',NS,'get','resourceoperations','-o','json'))['items']}
 return api('POST',PREFIX+'/submit',{'binding':name,'revision':saved['spec']['revision'],'executionDigest':preview['executionDigest'],'idempotencyKey':name})
def observed(w,want):
 try:o=operation(w)
 except Exception:return None
 return o if o.get('status',{}).get('phase') in want else None
def main():
 prepare();evidence=[]
 w=submit('lost','lost-response','12');name=w['metadata']['name']
 wait(lambda:observed(w,['Running','Unknown']),'operation dispatched')
 kubectl('-n',NS,'rollout','restart','deployment/'+RUN+'-controller');kubectl('-n',NS,'rollout','status','deployment/'+RUN+'-controller','--timeout=90s')
 done=wait(lambda:phase(name,['Succeeded','Failed','Error']),'lost response recovered');assert done['status']['phase']=='Succeeded',done['status']
 assert stats()['starts']==1;assert operation(w)['status']['outputs']['reportURI'].startswith('demo://')
 evidence.append({'case':'lost-response-controller-restart-output','workflow':name,'phase':'Succeeded','creates':1});print('PASS lost response/controller restart',flush=True)
 w=submit('retry','normal','35',deadline=15);name=w['metadata']['name']
 failed=wait(lambda:phase(name,['Failed','Error']),'deadline must stop native task');assert operation(w)['status'].get('startedAt')
 wait(lambda:observed(w,['Succeeded']),'external operation completes after native timeout')
 before=stats()['starts'];current=api('GET','/api/v1/workflows/'+NS+'/'+name)
 api('POST',PREFIX+'/retry',{'workflow':name,'uid':w['metadata']['uid'],'resourceVersion':current['metadata']['resourceVersion'],'idempotencyKey':RUN+'-retry'})
 done=wait(lambda:phase(name,['Succeeded','Failed','Error']),'original retry recovered');assert done['status']['phase']=='Succeeded',done['status'];assert done['metadata']['uid']==w['metadata']['uid'];assert stats()['starts']==before
 evidence.append({'case':'native-original-retry-same-operation','workflow':name,'phase':'Succeeded','creates':1});print('PASS original retry',flush=True)
 for label,policy in [('cancel','CancelIfSupported'),('detach','Detach')]:
  w=submit(label,'normal','20',policy);name=w['metadata']['name'];wait(lambda:observed(w,['Running']),'operation running')
  before=stats()['cancels'];api('PUT','/api/v1/workflows/'+NS+'/'+name+'/terminate',{})
  o=wait(lambda:observed(w,['Cancelled'] if label=='cancel' else ['Succeeded']),'termination policy observed')
  assert stats()['cancels']==before+(label=='cancel')
  evidence.append({'case':label,'workflow':name,'operationPhase':o['status']['phase']});print('PASS',label,flush=True)
 w=submit('unknown','unknown','300');name=w['metadata']['name'];wait(lambda:observed(w,['Unknown']),'lost unqueryable receipt must stay unknown');before=stats()['starts'];time.sleep(3);assert stats()['starts']==before;assert not operation(w)['status'].get('outputs')
 api('PUT','/api/v1/workflows/'+NS+'/'+name+'/terminate',{})
 evidence.append({'case':'unknown-outcome-no-repeat-mutation','workflow':name,'phase':'Unknown','creates':1});print('PASS unknown outcome',flush=True)
 w=submit('invalid','invalid-output','1');name=w['metadata']['name'];done=wait(lambda:phase(name,['Failed','Error','Succeeded']),'output rejected');assert done['status']['phase'] in ('Failed','Error');assert not operation(w)['status'].get('outputs');assert not any(n.get('displayName')=='after' and n['phase']=='Succeeded' for n in done['status']['nodes'].values())
 evidence.append({'case':'invalid-provider-output','workflow':name,'phase':done['status']['phase']})
 before=stats()['starts'];bad=copy.deepcopy(operation(w));bad.pop('status',None);bad['metadata']={'name':RUN+'-forged','namespace':NS,'labels':{GROUP+'/adapter':RUN},'annotations':{GROUP+'/workflow':name,GROUP+'/invoker-pod':'nonexistent'}};apply(bad)
 time.sleep(3);assert stats()['starts']==before;assert not get('resourceoperation',bad['metadata']['name']).get('status')
 original=operation(w);changed=copy.deepcopy(original);changed['spec']['operation']['inputs']['mode']='other'
 try:apply(changed)
 except Exception:pass
 else:raise AssertionError('immutable operation accepted changed inputs')
 assert operation(w)['spec']==original['spec'];assert stats()['starts']==before
 evidence.append({'case':'forged-invoker-and-immutable-snapshot','newRequests':0})
 print(json.dumps({'run':RUN,'cases':evidence,'externalCounters':stats()}),flush=True)
 # Preserve receipts and native workflows; stop only this synthetic adapter/server.
 if os.environ.get('ARGO_KEEP_OPERATION_DEMO')!='true':kubectl('-n',NS,'scale','deployment/'+RUN+'-controller','deployment/'+RUN+'-server','--replicas=0')
if __name__=='__main__':main()
