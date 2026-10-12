"""Real Secret consumption; no production credentials or external services."""
import json,os,subprocess,uuid
from e2e import api,wait_workflow,NS,PREFIX

run='secret-e2e-'+uuid.uuid4().hex[:8]
plaintext='synthetic-'+uuid.uuid4().hex
request={'template':'secret-v1','path':'/credential','requestId':str(uuid.uuid4()),'value':plaintext}
ref=api('POST',PREFIX+'/secrets',request)
assert ref==api('POST',PREFIX+'/secrets',request)
api('POST',PREFIX+'/secrets',{**request,'value':plaintext+'changed'},409)
api('PUT',PREFIX+'/sets/'+run,{'template':'secret-v1','values':{'credential':plaintext},'resourceVersion':''},400)
saved=api('PUT',PREFIX+'/sets/'+run,{'template':'secret-v1','values':{'credential':ref},'resourceVersion':''})
revision=saved['spec']['revision']
preview=api('POST',PREFIX+'/preview',{'binding':'secret-demo','revision':revision})
assert not preview['errors']
wf=api('POST',PREFIX+'/submit',{'binding':'secret-demo','revision':revision,'executionDigest':preview['executionDigest'],'idempotencyKey':run})
completed=wait_workflow(wf['metadata']['name'],lambda x:x.get('status',{}).get('phase')=='Succeeded')
history=api('GET',PREFIX+'/history/'+run)
serialized=json.dumps([saved,preview,completed,history])
assert plaintext not in serialized
args=['kubectl','--context',os.environ.get('ARGO_PARAMETERIZATION_CONTEXT','k3d-argo-parameterization'),'-n',NS]
logs=subprocess.check_output(args+['logs','-l','workflows.argoproj.io/workflow='+wf['metadata']['name'],'-c','main']).decode()
assert 'credential-consumed' in logs and plaintext not in logs
if os.environ.get('ARGO_TEST_SECRET_ROTATION')=='true':
 assert os.environ.get('ARGO_PARAMETERIZATION_CONTEXT','k3d-argo-parameterization')=='k3d-argo-parameterization'
 before=json.loads(subprocess.check_output(args+['get','secret',ref['secretRef']['name'],'-o','json']))
 subprocess.run(['docker','exec','k3d-argo-parameterization-server-0','k3s','secrets-encrypt','rotate-keys'],check=True,timeout=90,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
 after=json.loads(subprocess.check_output(args+['get','secret',ref['secretRef']['name'],'-o','json']))
 assert before['metadata']['uid']==after['metadata']['uid'] and before['data']==after['data']
 unchanged=api('PUT',PREFIX+'/sets/'+run,{'template':'secret-v1','values':{'credential':ref},'resourceVersion':saved['metadata']['resourceVersion']})
 assert unchanged['spec']['revision']==revision
 print('PASS backend key rotation retains the pinned Secret and parameter revision',flush=True)
# Replacement gets a different immutable version and a reference-only Diff.
replacement=api('POST',PREFIX+'/secrets',{**request,'requestId':str(uuid.uuid4()),'value':plaintext+'replaced'})
s2=api('PUT',PREFIX+'/sets/'+run,{'template':'secret-v1','values':{'credential':replacement},'resourceVersion':saved['metadata']['resourceVersion']})
diff=api('POST',PREFIX+'/diff',{'left':revision,'right':s2['spec']['revision']})
assert diff['changes'] and plaintext not in json.dumps(diff)
# Revocation is explicit, affects only this synthetic test Secret, and must be
# detected on preview instead of pretending the earlier reference is usable.
subprocess.run(args+['delete','secret',replacement['secretRef']['name']],check=True,stdout=subprocess.DEVNULL)
error=api('POST',PREFIX+'/preview',{'binding':'secret-demo','revision':s2['spec']['revision']},400)
assert plaintext not in json.dumps(error)
print('PASS secret input -> encrypted Kubernetes backend -> immutable reference -> revision -> native SecretKeyRef consumption -> version diff -> revoked reference rejection',flush=True)
