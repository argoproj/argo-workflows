"""Actual Vault KV-v2 -> immutable native Secret -> parameter binding -> retry/revocation."""
import json,os,time,uuid
from e2e import api,wait_workflow,PREFIX,NS
from template_refs_e2e import apply,kubectl
from secret_versions_e2e import fixture,G
RUN='vault-'+uuid.uuid4().hex[:8]
TOKEN='local-synthetic-'+uuid.uuid4().hex

def wait_for(fn,timeout=80):
 end=time.monotonic()+timeout
 while time.monotonic()<end:
  value=fn()
  if value:return value
  time.sleep(2)
 raise AssertionError('bounded external secret transition did not complete')
def native(kind,name,spec=None,**extra):
 return {'apiVersion':'apps/v1' if kind=='Deployment' else ('rbac.authorization.k8s.io/v1' if kind in ['Role','RoleBinding'] else 'v1'),'kind':kind,'metadata':{'name':name,'namespace':NS},**({'spec':spec} if spec else {}),**extra}
def index_name():
 raw=kubectl('-n',NS,'get','configmap',RUN,'--ignore-not-found','-o','json')
 if not raw:return None
 value=json.loads(raw).get('data',{}).get('current');return json.loads(value)['name'] if value else None
def vault(*args):
 return kubectl('-n',NS,'exec','deployment/'+RUN,'--','vault',*args)
def main():
 apply(native('Secret',RUN,immutable=True,stringData={'token':TOKEN}))
 apply(native('ServiceAccount',RUN))
 apply(native('Role',RUN,rules=[{'apiGroups':[''],'resources':['secrets','configmaps'],'verbs':['get','list','create','update','delete']}]))
 apply(native('RoleBinding',RUN,roleRef={'apiGroup':'rbac.authorization.k8s.io','kind':'Role','name':RUN},subjects=[{'kind':'ServiceAccount','name':RUN,'namespace':NS}]))
 config={'address':'http://'+RUN+':8200','mount':'secret','entry':'demo','field':'token','tokenFile':'/credential/token'}
 apply(native('ConfigMap',RUN+'-config',data={'config.json':json.dumps(config)}))
 apply(native('Deployment',RUN,{'selector':{'matchLabels':{'app':RUN}},'template':{'metadata':{'labels':{'app':RUN}},'spec':{'automountServiceAccountToken':False,'containers':[{'name':'vault','image':'hashicorp/vault:1.19.0','args':['server','-dev','-dev-listen-address=0.0.0.0:8200'],'env':[{'name':'VAULT_DEV_ROOT_TOKEN_ID','valueFrom':{'secretKeyRef':{'name':RUN,'key':'token'}}},{'name':'VAULT_TOKEN','valueFrom':{'secretKeyRef':{'name':RUN,'key':'token'}}},{'name':'VAULT_ADDR','value':'http://127.0.0.1:8200'}],'readinessProbe':{'httpGet':{'path':'/v1/sys/health','port':8200},'periodSeconds':2}}]}}}))
 apply(native('Service',RUN,{'selector':{'app':RUN},'ports':[{'port':8200,'targetPort':8200}]}))
 try:
  kubectl('-n',NS,'rollout','status','deployment/'+RUN,'--timeout=90s')
  first='synthetic-short';second='synthetic-'+uuid.uuid4().hex
  vault('kv','put','secret/demo','token='+first)
  apply(native('Deployment',RUN+'-bridge',{'selector':{'matchLabels':{'app':RUN+'-bridge'}},'template':{'metadata':{'labels':{'app':RUN+'-bridge'}},'spec':{'serviceAccountName':RUN,'containers':[{'name':'bridge','image':os.environ.get('ARGO_EXTERNAL_SECRET_IMAGE','argo-parameterization-secret:local-v47-v4.0.1'),'imagePullPolicy':'Never','args':['--namespace='+NS,'--index='+RUN,'--config=/config/config.json','--allow-development-http'],'volumeMounts':[{'name':'config','mountPath':'/config','readOnly':True},{'name':'credential','mountPath':'/credential','readOnly':True}]}],'volumes':[{'name':'config','configMap':{'name':RUN+'-config'}},{'name':'credential','secret':{'secretName':RUN}}]}}}))
  v1=wait_for(index_name);assert v1.endswith('-v1')
  for obj in fixture(RUN):apply(obj)
  saved=api('PUT',PREFIX+'/sets/'+RUN,{'template':RUN,'values':{'credential':{'latestSecretRef':{'configMap':RUN,'key':'current'}}},'resourceVersion':''})
  rev=saved['spec']['revision'];preview=api('POST',PREFIX+'/preview',{'binding':RUN,'revision':rev});assert not preview['errors'],preview
  wf=api('POST',PREFIX+'/submit',{'binding':RUN,'revision':rev,'executionDigest':preview['executionDigest'],'idempotencyKey':RUN});name=wf['metadata']['name'];uid=wf['metadata']['uid']
  wait_workflow(name,lambda w:any(n.get('type')=='Suspend' and n.get('phase')=='Running' for n in w.get('status',{}).get('nodes',{}).values()));api('PUT','/api/v1/workflows/'+NS+'/'+name+'/resume',{})
  failed=wait_workflow(name,lambda w:w.get('status',{}).get('phase')=='Failed')
  approval=next((i,n) for i,n in failed['status']['nodes'].items() if n['type']=='Suspend')
  vault('kv','put','secret/demo','token='+second)
  v2=wait_for(lambda:(v if (v:=index_name()) and v.endswith('-v2') else None))
  api('POST',PREFIX+'/retry',{'workflow':name,'uid':uid,'resourceVersion':failed['metadata']['resourceVersion'],'idempotencyKey':RUN+'-original'})
  failed=wait_workflow(name,lambda w:w.get('status',{}).get('phase')=='Failed')
  assert v1 in failed['spec']['arguments']['parameters'][0]['value']
  vault('kv','destroy','-versions=1','secret/demo')
  wait_for(lambda:not kubectl('-n',NS,'get','secret',v1,'--ignore-not-found','-o','name'))
  error=api('POST',PREFIX+'/retry',{'workflow':name,'uid':uid,'resourceVersion':failed['metadata']['resourceVersion'],'idempotencyKey':RUN+'-revoked'},400);assert 'revoked' in json.dumps(error)
  req={'workflow':name,'uid':uid,'revision':rev};plan=api('POST',PREFIX+'/retry-preview',req)
  api('POST',PREFIX+'/retry',{**req,'resourceVersion':plan['resourceVersion'],'planDigest':plan['digest'],'idempotencyKey':RUN+'-rotate'})
  done=wait_workflow(name,lambda w:w.get('status',{}).get('phase')=='Succeeded');assert done['metadata']['uid']==uid
  assert all(done['status']['nodes'][approval[0]].get(k)==approval[1].get(k) for k in ['phase','finishedAt','outputs'])
  raw=json.dumps([saved,preview,done,api('GET',PREFIX+'/history/'+RUN)])
  logs=kubectl('-n',NS,'logs','-l','workflows.argoproj.io/workflow='+name,'-c','main','--tail=100').decode()
  assert first not in raw+logs and second not in raw+logs and TOKEN not in raw+logs
  kubectl('-n',NS,'scale','deployment/'+RUN,'--replicas=0')
  wait_for(lambda:not kubectl('-n',NS,'get','secret',v2,'--ignore-not-found','-o','name'),90)
  api('POST',PREFIX+'/preview',{'binding':RUN,'revision':rev},400)
  print('PASS',json.dumps({'workflow':name,'uid':uid,'vaultVersions':[1,2],'originalRetryPinned':True,'revokedVersionBlocked':True,'sameWorkflowRotation':True,'backendOutageExpiresInjection':True,'plaintextAbsentFromHistoryAndLogs':True}),flush=True)
 finally:
  kubectl('-n',NS,'delete','deployment',RUN,RUN+'-bridge','--ignore-not-found=true')
  kubectl('-n',NS,'delete','service,secret,serviceaccount,role,rolebinding',RUN,'--ignore-not-found=true')
if __name__=='__main__':main()
