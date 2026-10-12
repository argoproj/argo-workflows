"""Frozen runtime form -> encrypted write -> native suspend -> Secret consumer -> native retry.
Synthetic credentials and isolated local k3s only; no external service calls.
"""
import copy,json,os,subprocess,uuid
from e2e import api,wait_workflow,NS,PREFIX,BASE
from template_refs_e2e import native,apply,kubectl,resource
from playwright.sync_api import sync_playwright,expect

RUN='secret-input-'+uuid.uuid4().hex[:8]
G='parameters.workflows.example.io'

def fixture(name=RUN, failure=True):
 ref={'type':'object','properties':{'secretRef':{'type':'object','properties':{'name':{'type':'string'},'key':{'type':'string'}},'required':['name','key'],'additionalProperties':False}},'required':['secretRef'],'additionalProperties':False}
 model={'dialect':'http://json-schema.org/draft-07/schema#','schema':{'type':'object','properties':{'credential':ref,'note':{'type':'string'}},'required':['credential'],'additionalProperties':False},'sensitivePaths':['/credential'],'secretSchemas':{'/credential':{'type':'string','minLength':8}}}
 form={'parameterTemplate':name,'layout':[{'id':'authorization','kind':'tab','label':'Operation authorization'}],'fields':[{'path':'/credential','widget':'secret','group':'authorization'},{'path':'/note','group':'authorization'}]}
 tasks=[{'name':'authorize','template':'authorize'},{'name':'consume','dependencies':['authorize'],'template':'consume','arguments':{'parameters':[{'name':'credential','value':'{{tasks.authorize.outputs.parameters.credential}}'}]}}]
 templates=[{'name':'authorize','suspend':{},'outputs':{'parameters':[{'name':'credential','valueFrom':{'supplied':{}}},{'name':'note','valueFrom':{'supplied':{}}}]}},
 {'name':'consume','inputs':{'parameters':[{'name':'credential'}]},'container':{'image':'python:3.12-alpine','command':['python','-c'],'args':['import os; assert len(os.environ["TOKEN"]) >= 8; print("runtime-credential-consumed")'],'env':[{'name':'TOKEN','valueFrom':{'secretKeyRef':{'name':"{{=jsonpath(inputs.parameters.credential, '$.secretRef.name')}}",'key':"{{=jsonpath(inputs.parameters.credential, '$.secretRef.key')}}"}}}]}}]
 if failure:
  tasks.append({'name':'recover','dependencies':['consume'],'template':'recover'})
  templates.append({'name':'recover','resource':{'action':'get','manifest':'apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: {{workflow.name}}-ready\n'}})
 spec={'serviceAccountName':'workflow-executor','entrypoint':'main','templates':[{'name':'main','dag':{'tasks':tasks}},*templates]}
 binding={'workflowTemplate':name,'parameterTemplate':'empty-v1','mappings':[],'inputs':[{'template':'authorize','parameterTemplate':name,'form':name,'mappings':[{'path':'/credential','parameter':'credential','encoding':'secret-reference-json'},{'path':'/note','parameter':'note','encoding':'string'}]}]}
 return [resource('ParameterTemplate',name,model),resource('WorkflowForm',name,form),native('WorkflowTemplate',name,spec),resource('WorkflowParameterBinding',name,binding)]

def main():
 for obj in fixture():apply(obj)
 saved=api('PUT',PREFIX+'/sets/'+RUN,{'template':'empty-v1','values':{},'resourceVersion':''})
 preview=api('POST',PREFIX+'/preview',{'binding':RUN,'revision':saved['spec']['revision']});assert not preview['errors'],preview
 wf=api('POST',PREFIX+'/submit',{'binding':RUN,'revision':saved['spec']['revision'],'executionDigest':preview['executionDigest'],'idempotencyKey':RUN})
 name=wf['metadata']['name'];uid=wf['metadata']['uid'];native_url='/api/v1/workflows/'+NS+'/'+name
 wait_workflow(name,lambda w:any(n.get('type')=='Suspend' and n.get('phase')=='Running' for n in w.get('status',{}).get('nodes',{}).values()))
 pending=api('GET',PREFIX+'/inputs/'+name);item=pending['items'][0]
 assert pending['encryptedSecretInput']
 context={'workflow':name,'uid':uid,'nodeId':item['nodeId'],'contractDigest':item['contractDigest']}
 # Delete only the generated sources. Both UI and encrypted input must use the frozen receipt.
 kubectl('-n',NS,'delete','parametertemplate',RUN)
 kubectl('-n',NS,'delete','workflowform',RUN)
 plaintext='synthetic-runtime-'+uuid.uuid4().hex
 request={'input':context,'path':'/credential','requestId':str(uuid.uuid4()),'value':plaintext}
 for key in ['uid','nodeId','contractDigest']:
  api('POST',PREFIX+'/secrets',{**request,'input':{**context,key:'not-the-frozen-identity'}},400)
 api('POST',PREFIX+'/secrets',{**request,'template':'empty-v1'},400)
 api('POST',PREFIX+'/secrets',{**request,'value':'short'},400)
 bad_input={'uid':uid,'resourceVersion':pending['resourceVersion'],'nodeId':item['nodeId'],'requestId':str(uuid.uuid4()),'values':{'credential':plaintext,'note':'test'}}
 api('POST',PREFIX+'/inputs/'+name,bad_input,400)
 revoked=api('POST',PREFIX+'/secrets',request)
 kubectl('-n',NS,'delete','secret',revoked['secretRef']['name'])
 api('POST',PREFIX+'/inputs/'+name,{**bad_input,'values':{'credential':revoked,'note':'test'}},400)
 with sync_playwright() as pw:
  browser=pw.chromium.launch(headless=True,executable_path=os.environ['ARGO_PARAMETERIZATION_CHROMIUM'])
  page=browser.new_page(viewport={'width':1440,'height':1080})
  version=api('GET','/api/v1/version')['version'];page.add_init_script('localStorage.setItem("modal/ftu",JSON.stringify("dismissed"));localStorage.setItem("modal/version",JSON.stringify('+json.dumps(version.rsplit('.',1)[0])+'));')
  lost=[];ref=None
  try:
   page.goto(BASE+'/workflows/'+NS+'/'+name)
   expect(page.get_by_role('tab',name='Operation authorization',exact=True)).to_be_visible()
   page.get_by_label('New secret /credential',exact=True).fill(plaintext)
   with page.expect_response(lambda r:r.request.method=='POST' and r.url.endswith('/secrets')) as result:
    page.get_by_role('button',name='Store new encrypted secret version',exact=True).click()
   assert result.value.ok,result.value.text();ref=result.value.json()
   expect(page.get_by_label('New secret /credential',exact=True)).to_have_value('')
   page.get_by_label('/note',exact=True).fill('frozen-runtime-input')
   # Commit to the real server, then lose exactly its response; recovery uses the identical request.
   def lose_response(route):
    if route.request.method!='POST' or lost:return route.continue_()
    response=route.fetch();assert response.ok,response.text()
    lost.append(route.request.post_data_json)
    route.fulfill(status=503,content_type='application/json',body=json.dumps({'message':'synthetic response loss after commit'}))
   page.route('**/api/v1/parameterization/'+NS+'/inputs/'+name,lose_response)
   with page.expect_response(lambda r:r.request.method=='POST' and r.url.endswith('/inputs/'+name)) as lost_response:
    page.get_by_role('button',name='Validate input and continue this node',exact=True).click()
   assert lost_response.value.status==503
   assert len(lost)==1 and lost[0]['values']=={'credential':ref,'note':'frozen-runtime-input'}
   recovered=api('POST',PREFIX+'/inputs/'+name,lost[0]);assert recovered['metadata']['uid']==uid
   failed=wait_workflow(name,lambda w:w.get('status',{}).get('phase') in ('Failed','Error'))
   retained={k:(n['phase'],n.get('finishedAt'),n.get('outputs')) for k,n in failed['status']['nodes'].items() if n['type'] in ('Suspend','Pod') and n['phase']=='Succeeded'}
   assert len(retained)==2,retained
   submitted=api('GET',PREFIX+'/inputs/'+name)['submitted'];assert len(submitted)==1 and len(submitted[0]['secretVersions'])==1
   # Native Argo retry (not extension retry) consumes the original contract without restarting approvals.
   apply({'apiVersion':'v1','kind':'ConfigMap','metadata':{'name':name+'-ready','namespace':NS},'data':{'allow':'true'}})
   api('PUT',native_url+'/retry',{})
   done=wait_workflow(name,lambda w:w.get('status',{}).get('phase')=='Succeeded');assert done['metadata']['uid']==uid
   for key,status in retained.items():assert (done['status']['nodes'][key]['phase'],done['status']['nodes'][key].get('finishedAt'),done['status']['nodes'][key].get('outputs'))==status
   logs=kubectl('-n',NS,'logs','-l','workflows.argoproj.io/workflow='+name,'-c','main').decode();assert 'runtime-credential-consumed' in logs
   assert plaintext not in json.dumps([saved,preview,pending,submitted,done,lost,logs])
   # Revoke only the synthetic credential; API and admission must reject a reset.
   secret=json.loads(kubectl('-n',NS,'get','secret',ref['secretRef']['name'],'-o','json'))
   kubectl('-n',NS,'delete','secret',ref['secretRef']['name'])
   api('PUT',native_url+'/retry',{'restartSuccessful':True,'nodeFieldSelector':'displayName=consume'},400)
   rejected=subprocess.run(['kubectl','--context','k3d-argo-parameterization','-n',NS,'patch','workflow',name,'--type=merge','-p',json.dumps({'status':{'phase':'Running'}})],capture_output=True,text=True)
   assert rejected.returncode!=0 and 'revoked' in rejected.stderr,rejected.stderr
   apply({'apiVersion':'v1','kind':'Secret','metadata':{'name':ref['secretRef']['name'],'namespace':NS},'immutable':True,'type':'Opaque','data':secret['data']})
   api('PUT',native_url+'/retry',{'restartSuccessful':True,'nodeFieldSelector':'displayName=consume'},400)
   assert api('GET',native_url)['status']['phase']=='Succeeded'
   page.screenshot(path='/tmp/parameterization-sensitive-inputs-success.png',full_page=True)
   print('PASS',json.dumps({'workflow':name,'uid':uid,'frozenSourcesDeleted':True,'encryptedBrowserInput':True,'responseLossRecovered':True,'nativeRetryRetained':len(retained),'apiAndAdmissionRevocationRejected':True,'plaintextLeaks':False}),flush=True)
  except Exception:
   page.screenshot(path='/tmp/parameterization-sensitive-inputs-failure.png',full_page=True);raise
  finally:browser.close()

if __name__=='__main__':main()
