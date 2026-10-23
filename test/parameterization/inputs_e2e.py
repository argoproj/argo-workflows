"""Typed intermediate values go through native suspend/Set and native outputs."""
import json,uuid,subprocess,os
from e2e import api,wait_workflow,NS,PREFIX
run='input-e2e-'+uuid.uuid4().hex[:8]
saved=api('PUT',PREFIX+'/sets/'+run,{'template':'empty-v1','values':{},'resourceVersion':''})
p=api('POST',PREFIX+'/preview',{'binding':'intermediate-demo','revision':saved['spec']['revision']})
wf=api('POST',PREFIX+'/submit',{'binding':'intermediate-demo','revision':saved['spec']['revision'],'executionDigest':p['executionDigest'],'idempotencyKey':run})
name=wf['metadata']['name']
wait_workflow(name,lambda w:any(n['type']=='Suspend' and n['phase']=='Running' for n in w['status'].get('nodes',{}).values()))
pending=api('GET',PREFIX+'/inputs/'+name)
assert len(pending['items'])==1
req={'uid':pending['uid'],'resourceVersion':pending['resourceVersion'],'nodeId':pending['items'][0]['nodeId'],'requestId':run,'values':{'replicas':0,'release':'canary'}}
api('POST',PREFIX+'/inputs/'+name,req,400)
req['values']['replicas']=3
result=api('POST',PREFIX+'/inputs/'+name,req)
assert result['metadata']['uid']==wf['metadata']['uid']
complete=wait_workflow(name,lambda w:w['status']['phase']=='Succeeded')
again=api('POST',PREFIX+'/inputs/'+name,req)
assert again['metadata']['uid']==wf['metadata']['uid'] and again['status']['phase']=='Succeeded'
readback=api('GET',PREFIX+'/inputs/'+name)
assert not readback['items'] and len(readback['submitted'])==1
assert readback['submitted'][0]['values']==req['values']
api('POST',PREFIX+'/inputs/'+name,{**req,'values':{'replicas':4,'release':'stable'}},409)
logs=subprocess.check_output(['kubectl','--context',os.environ.get('ARGO_PARAMETERIZATION_CONTEXT','k3d-argo-parameterization'),'-n',NS,'logs','-l','workflows.argoproj.io/workflow='+name,'-c','main']).decode()
assert 'replicas=3 release=canary' in logs
print('PASS schema rejection -> native suspend form -> supplied outputs -> native downstream execution -> response-loss recovery; UID='+wf['metadata']['uid'])
