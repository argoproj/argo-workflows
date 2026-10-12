"""Bounded local-k3s acceptance through Argo Server and its native controller.
Run only after manifests/parameterization/local-install.yaml and demos are installed.
"""
import copy,json,os,re,subprocess,time,urllib.request,urllib.error,uuid
from pathlib import Path
BASE=os.environ.get('ARGO_PARAMETERIZATION_URL','http://127.0.0.1:2746')
NS='argo-parameterization-lab';PREFIX='/api/v1/parameterization/'+NS
assert BASE.startswith(('http://127.0.0.1:','http://localhost:')), 'local endpoint only'
RUN='e2e-'+uuid.uuid4().hex[:8]
def api(method,path,body=None,expected=200):
 req=urllib.request.Request(BASE+path,data=json.dumps(body).encode() if body is not None else None,method=method,headers={'Content-Type':'application/json'})
 try:
  with urllib.request.urlopen(req,timeout=25) as r: code,data=r.status,r.read()
 except urllib.error.HTTPError as e: code,data=e.code,e.read()
 result=json.loads(data)
 assert code==expected,(method,path,code,result)
 return result

def wait_workflow(name,predicate,timeout=240):
 start=time.monotonic()
 while time.monotonic()-start<timeout:
  wf=api('GET','/api/v1/workflows/'+NS+'/'+name)
  if predicate(wf):return wf
  if wf.get('status',{}).get('phase') in ('Failed','Error'):raise AssertionError(wf['status'])
  time.sleep(2)
 raise AssertionError('workflow did not reach expected state within bounded timeout: '+name)

def main():
 if os.environ.get('ARGO_CHECK_LOCAL_ASSETS')=='true':
  expected=Path('ui/dist/app/index.html').read_text()
  actual=urllib.request.urlopen(BASE+'/',timeout=10).read().decode()
  local_main=re.findall(r'main\.[a-f0-9]+\.js',expected)
  assert local_main and local_main==re.findall(r'main\.[a-f0-9]+\.js',actual),'deployed UI does not match the completed local build'
 for template in ['dynamic-delivery-demo','training-demo','secret-demo','intermediate-demo']:
  native=api('GET','/api/v1/workflow-templates/'+NS+'/'+template)
  assert native['metadata']['labels']['workflows.argoproj.io/controller-instanceid']=='parameterization-lab'
 import yaml
 catalog=api('GET',PREFIX+'/catalog')
 assert {'ParameterTemplate','WorkflowForm','WorkflowParameterBinding','ActionDefinition','WorkflowActionBinding'} <= {x['kind'] for x in catalog}
 for resource in catalog:
  page=api('GET',PREFIX+'/catalog/'+resource['resource']);assert isinstance(page['items'],list)
 binding=api('GET',PREFIX+'/catalog/workflowparameterbindings?name=dynamic-delivery')
 model=api('GET',PREFIX+'/catalog/parametertemplates?name='+binding['spec']['parameterTemplate'])
 assert model['spec']['schema']['type']=='object'
 assert api('GET',PREFIX+'/catalog/workflowforms?name='+binding['spec']['form'])['kind']=='WorkflowForm'
 api('GET',PREFIX+'/catalog/secrets',expected=400)
 values=yaml.safe_load(Path('examples/parameterization/topology-values.yaml').read_text())
 request={'template':'topology-v1','values':values,'resourceVersion':'','message':'E2E initial'}
 s=api('PUT',PREFIX+'/sets/'+RUN,request);r1=s['spec']['revision']
 request['resourceVersion']=s['metadata']['resourceVersion']
 same=api('PUT',PREFIX+'/sets/'+RUN,request);assert same['spec']['revision']==r1
 changed=copy.deepcopy(values);changed['zones'].reverse();changed['environment']='production'
 nextreq={**request,'values':changed,'message':'change order and environment'}
 s2=api('PUT',PREFIX+'/sets/'+RUN,nextreq);r2=s2['spec']['revision']
 api('PUT',PREFIX+'/sets/'+RUN,{**nextreq,'values':values},409)
 history=api('GET',PREFIX+'/history/'+RUN);assert len(history['items'])==2
 diff=api('POST',PREFIX+'/diff',{'left':r1,'right':r2});assert any(c['path']=='/environment' for c in diff['changes'])
 invalid={**changed,'zones':[]}
 s3=api('PUT',PREFIX+'/sets/'+RUN,{**nextreq,'resourceVersion':s2['metadata']['resourceVersion'],'values':invalid});r3=s3['spec']['revision']
 invalidpreview=api('POST',PREFIX+'/preview',{'binding':'dynamic-delivery','revision':r3});assert invalidpreview['errors']
 restored=api('PUT',PREFIX+'/sets/'+RUN,{**request,'resourceVersion':s3['metadata']['resourceVersion'],'restoredFrom':r1})
 preview=api('POST',PREFIX+'/preview',{'binding':'dynamic-delivery','revision':restored['spec']['revision']});assert not preview['errors']
 submit={'binding':'dynamic-delivery','revision':restored['spec']['revision'],'executionDigest':preview['executionDigest'],'idempotencyKey':RUN}
 wf=api('POST',PREFIX+'/submit',submit);name=wf['metadata']['name'];uid=wf['metadata']['uid']
 repeat=api('POST',PREFIX+'/submit',submit);assert repeat['metadata']['uid']==uid
 api('POST',PREFIX+'/submit',{**submit,'revision':r2},409)
 print('submitted',name,'revision/diff/CAS/idempotency passed',flush=True)
 wait_workflow(name,lambda x:any(n.get('type')=='Suspend' and n.get('phase')=='Running' for n in x.get('status',{}).get('nodes',{}).values()))
 api('PUT','/api/v1/workflows/'+NS+'/'+name+'/resume',{})
 complete=wait_workflow(name,lambda x:x.get('status',{}).get('phase')=='Succeeded')
 raw=subprocess.check_output(['kubectl','--context',os.environ.get('ARGO_PARAMETERIZATION_CONTEXT','k3d-argo-parameterization'),'-n',NS,'get','configmap','-l','demo.workflow='+name,'-o','json'])
 cms=json.loads(raw)['items'];assert len(cms)==6,len(cms)
 assert {(c['data']['zone'],c['data']['role']) for c in cms}=={('zone-a','admin'),('zone-a','dns'),('zone-b','admin'),('zone-b','dns')}
 assert all(c['data']['environment']=='test' for c in cms)
 print('dynamic topology: 2 zones, 3 subnets, 2 roles -> 6 native Resource executions succeeded',flush=True)
 t=api('PUT',PREFIX+'/sets/'+RUN+'-training',{'template':'training-v1','values':{'dataset':'example','epochs':2,'shuffle':False},'resourceVersion':''})
 p=api('POST',PREFIX+'/preview',{'binding':'training','revision':t['spec']['revision']})
 tw=api('POST',PREFIX+'/submit',{'binding':'training','revision':t['spec']['revision'],'executionDigest':p['executionDigest'],'idempotencyKey':RUN+'-training'})
 trained=wait_workflow(tw['metadata']['name'],lambda x:x.get('status',{}).get('phase')=='Succeeded')
 source={'workflow':trained['metadata']['name'],'uid':trained['metadata']['uid']}
 edit=api('GET',PREFIX+'/edit-context/'+source['workflow']+'?uid='+source['uid'])
 assert edit['binding']=='training' and edit['revision']['metadata']['name']==t['spec']['revision']
 updated=copy.deepcopy(edit['revision']['spec']['values']);updated['epochs']=3
 t2=api('PUT',PREFIX+'/sets/'+RUN+'-training',{'template':'training-v1','values':updated,'resourceVersion':edit['set']['metadata']['resourceVersion'],'restoredFrom':edit['revision']['metadata']['name']})
 p2=api('POST',PREFIX+'/preview',{'binding':'training','revision':t2['spec']['revision']})
 linked={'binding':'training','revision':t2['spec']['revision'],'executionDigest':p2['executionDigest'],'idempotencyKey':RUN+'-training-edited','sourceExecution':source}
 api('POST',PREFIX+'/submit',{**linked,'sourceExecution':{**source,'uid':'replaced'}},400)
 tw2=api('POST',PREFIX+'/submit',linked)
 assert tw2['metadata']['uid']!=trained['metadata']['uid']
 assert tw2['metadata']['annotations']['parameters.workflows.example.io/source-workflow-uid']==trained['metadata']['uid']
 assert api('POST',PREFIX+'/submit',linked)['metadata']['uid']==tw2['metadata']['uid']
 wait_workflow(tw2['metadata']['name'],lambda x:x.get('status',{}).get('phase')=='Succeeded')
 still=api('GET','/api/v1/workflows/'+NS+'/'+trained['metadata']['name'])
 assert still['spec']==trained['spec'] and still['status']['nodes']==trained['status']['nodes']
 reopened=api('GET',PREFIX+'/edit-context/'+source['workflow']+'?uid='+source['uid'])
 assert reopened['revision']['metadata']['name']==t['spec']['revision'] and reopened['set']['spec']['revision']==t2['spec']['revision']
 print('execution→original binding/revision→CAS edit→linked new Workflow succeeded; original Workflow unchanged',flush=True)
 # A distinct changed revision is explicitly submitted as a new execution.
 failing=copy.deepcopy(values);failing['fail']=True
 failedset=api('PUT',PREFIX+'/sets/'+RUN,{**request,'values':failing,'resourceVersion':restored['metadata']['resourceVersion']})
 fp=api('POST',PREFIX+'/preview',{'binding':'dynamic-delivery','revision':failedset['spec']['revision']})
 fw=api('POST',PREFIX+'/submit',{'binding':'dynamic-delivery','revision':failedset['spec']['revision'],'executionDigest':fp['executionDigest'],'idempotencyKey':RUN+'-retry'})
 fn=fw['metadata']['name']
 wait_workflow(fn,lambda x:any(n.get('type')=='Suspend' and n.get('phase')=='Running' for n in x.get('status',{}).get('nodes',{}).values()))
 api('PUT','/api/v1/workflows/'+NS+'/'+fn+'/resume',{})
 failed=wait_workflow(fn,lambda x:x.get('status',{}).get('phase')=='Failed')
 # Native Argo recalculates enclosing DAG/TaskGroup summaries. Concrete
 # successful actions and human suspend decisions must not execute again.
 preserved={key:(n['phase'],n.get('finishedAt')) for key,n in failed['status']['nodes'].items() if n['phase']=='Succeeded' and n['type'] in ('Pod','Suspend','HTTP','Plugin','Container')}
 fixture={'apiVersion':'v1','kind':'ConfigMap','metadata':{'name':fn+'-recovery','namespace':NS},'data':{'allow':'true'}}
 subprocess.run(['kubectl','--context',os.environ.get('ARGO_PARAMETERIZATION_CONTEXT','k3d-argo-parameterization'),'apply','-f','-'],input=json.dumps(fixture).encode(),check=True,stdout=subprocess.DEVNULL)
 # Re-read after the native controller has persisted completion.
 failed=api('GET','/api/v1/workflows/'+NS+'/'+fn)
 retry={'workflow':fn,'uid':failed['metadata']['uid'],'resourceVersion':failed['metadata']['resourceVersion'],'idempotencyKey':RUN+'-attempt'}
 rr=api('POST',PREFIX+'/retry',retry);assert rr['metadata']['uid']==failed['metadata']['uid']
 recovered=wait_workflow(fn,lambda x:x.get('status',{}).get('phase')=='Succeeded')
 for key,value in preserved.items():assert (recovered['status']['nodes'][key]['phase'],recovered['status']['nodes'][key].get('finishedAt'))==value,(key,value,recovered['status']['nodes'][key])
 duplicate=api('POST',PREFIX+'/retry',retry);assert duplicate['status']['phase']=='Succeeded'
 assert duplicate['metadata']['uid']==failed['metadata']['uid']
 print('PASS',json.dumps({'run':RUN,'workflow':name,'uid':uid,'training':tw['metadata']['name'],'retried':fn,'preservedSuccessfulNodes':len(preserved)}),flush=True)
if __name__=='__main__':main()
