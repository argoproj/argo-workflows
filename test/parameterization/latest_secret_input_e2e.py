"""Browser version policy -> frozen runtime input -> native Secret consumer."""
import json,os,uuid
from e2e import api,wait_workflow,PREFIX,NS,BASE
from template_refs_e2e import apply,kubectl
from sensitive_inputs_e2e import fixture as input_fixture
from secret_versions_e2e import fixture as secret_fixture
from playwright.sync_api import sync_playwright,expect
RUN='latest-input-'+uuid.uuid4().hex[:8]

def main():
 objects=input_fixture(RUN,False)
 objects[0]['spec']['schema']['properties']['credential']=secret_fixture()[0]['spec']['schema']['properties']['credential']
 for obj in objects:apply(obj)
 token='local-private-'+uuid.uuid4().hex
 apply({'apiVersion':'v1','kind':'Secret','metadata':{'name':RUN+'-v1','namespace':NS},'immutable':True,'stringData':{'token':token}})
 apply({'apiVersion':'v1','kind':'ConfigMap','metadata':{'name':RUN,'namespace':NS},'data':{'current':json.dumps({'name':RUN+'-v1','key':'token'})}})
 saved=api('PUT',PREFIX+'/sets/'+RUN,{'template':'empty-v1','values':{},'resourceVersion':''})
 preview=api('POST',PREFIX+'/preview',{'binding':RUN,'revision':saved['spec']['revision']})
 wf=api('POST',PREFIX+'/submit',{'binding':RUN,'revision':saved['spec']['revision'],'executionDigest':preview['executionDigest'],'idempotencyKey':RUN})
 name=wf['metadata']['name'];uid=wf['metadata']['uid']
 wait_workflow(name,lambda w:any(n.get('type')=='Suspend' and n.get('phase')=='Running' for n in w.get('status',{}).get('nodes',{}).values()))
 with sync_playwright() as pw:
  browser=pw.chromium.launch(headless=True,executable_path=os.environ['ARGO_PARAMETERIZATION_CHROMIUM']);page=browser.new_page(viewport={'width':1440,'height':1000})
  version=api('GET','/api/v1/version')['version'];page.add_init_script('localStorage.setItem("modal/ftu",JSON.stringify("dismissed"));localStorage.setItem("modal/version",JSON.stringify('+json.dumps(version.rsplit('.',1)[0])+'));')
  try:
   page.goto(BASE+'/workflows/'+NS+'/'+name)
   page.get_by_label('Secret version policy /credential',exact=True).select_option('latest')
   page.get_by_label('Secret version index /credential',exact=True).fill(RUN)
   page.get_by_label('/note',exact=True).fill('reference-only')
   expect(page.get_by_text('Latest at submission: '+RUN+' / current',exact=True)).to_be_visible()
   with page.expect_response(lambda r:r.request.method=='POST' and r.url.endswith('/inputs/'+name)) as response:
    page.get_by_role('button',name='Validate input and continue this node',exact=True).click()
   assert response.value.ok,response.value.text();request=response.value.request.post_data_json
   done=wait_workflow(name,lambda w:w.get('status',{}).get('phase')=='Succeeded')
   receipts=api('GET',PREFIX+'/inputs/'+name)['submitted']
   assert receipts[0]['values']['credential']=={'secretRef':{'name':RUN+'-v1','key':'token'}},receipts
   assert any(k.startswith('latestSecret/') for k in receipts[0]['secretVersions'])
   assert token not in json.dumps([done,request,receipts])
   kubectl('-n',NS,'delete','configmap',RUN)
   assert api('POST',PREFIX+'/inputs/'+name,request)['metadata']['uid']==uid
   page.screenshot(path='/tmp/parameterization-latest-secret-input-success.png',full_page=True)
   print('PASS',json.dumps({'workflow':name,'uid':uid,'browserPolicy':True,'runtimeVersionPinned':True,'inputRecoveryAfterIndexDeletion':True}),flush=True)
  except Exception:
   page.screenshot(path='/tmp/parameterization-latest-secret-input-failure.png',full_page=True);raise
  finally:browser.close()
if __name__=='__main__':main()
