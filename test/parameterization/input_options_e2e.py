"""Frozen runtime form -> dependent options -> native outputs, isolated k3s only."""
import copy,json,os,time,uuid
from e2e import api,wait_workflow,NS,PREFIX,BASE
from template_refs_e2e import apply,resource,native,kubectl
from playwright.sync_api import sync_playwright,expect
RUN='input-options-'+uuid.uuid4().hex[:8]
G='parameters.workflows.example.io'

def fixture(name=RUN):
 model={'dialect':'http://json-schema.org/draft-07/schema#','schema':{'type':'object','required':['environment','region'],'additionalProperties':False,'properties':{'environment':{'type':'string','enum':['test','production']},'region':{'type':'string'}}}}
 form={'parameterTemplate':name,'options':{'regions':{'kind':'configmap','name':name,'key':'choices','dependsOn':['/environment']}},'fields':[{'path':'/environment','widget':'select'},{'path':'/region','widget':'select','optionSource':'regions'}]}
 choices=[{'value':'east','label':'Test East','when':{'/environment':'test'}},{'value':'west','label':'Production West','when':{'/environment':'production'}}]
 cm={'apiVersion':'v1','kind':'ConfigMap','metadata':{'name':name,'namespace':NS},'data':{'choices':json.dumps(choices)}}
 spec={'serviceAccountName':'workflow-executor','entrypoint':'main','templates':[{'name':'main','dag':{'tasks':[{'name':'choose','template':'choose'},{'name':'consume','template':'consume','dependencies':['choose'],'arguments':{'parameters':[{'name':'region','value':'{{tasks.choose.outputs.parameters.region}}'}]}}]}},{'name':'choose','suspend':{},'outputs':{'parameters':[{'name':'region','valueFrom':{'supplied':{}}}]}},{'name':'consume','inputs':{'parameters':[{'name':'region'}]},'container':{'image':'alpine:3.22','command':['sh','-ec'],'args':['test "$REGION" = west; printf "%s" "$REGION" > /tmp/region'],'env':[{'name':'REGION','value':'{{inputs.parameters.region}}'}]},'outputs':{'parameters':[{'name':'region','valueFrom':{'path':'/tmp/region'}}]}}]}
 binding={'parameterTemplate':'empty-v1','workflowTemplate':name,'mappings':[],'inputs':[{'template':'choose','parameterTemplate':name,'form':name,'mappings':[{'path':'/region','parameter':'region','encoding':'string'}]}]}
 return [resource('ParameterTemplate',name,model),resource('WorkflowForm',name,form),cm,native('WorkflowTemplate',name,spec),resource('WorkflowParameterBinding',name,binding)]

def main():
 objects=fixture()
 for obj in objects:apply(obj)
 cm=objects[2]
 for attempt in range(100):
  status=api('GET',PREFIX+'/catalog/workflowparameterbindings?name='+RUN).get('status',{})
  if status.get('conditions') and status['conditions'][0]['status']=='True':break
  time.sleep(.2)
 else:raise AssertionError(status)

 saved=api('PUT',PREFIX+'/sets/'+RUN,{'template':'empty-v1','values':{},'resourceVersion':''})
 preview=api('POST',PREFIX+'/preview',{'binding':RUN,'revision':saved['spec']['revision']});assert not preview['errors'],preview
 wf=api('POST',PREFIX+'/submit',{'binding':RUN,'revision':saved['spec']['revision'],'executionDigest':preview['executionDigest'],'idempotencyKey':RUN})
 name=wf['metadata']['name'];uid=wf['metadata']['uid']
 wait_workflow(name,lambda w:any(n.get('type')=='Suspend' and n.get('phase')=='Running' for n in w.get('status',{}).get('nodes',{}).values()))
 pending=api('GET',PREFIX+'/inputs/'+name);item=pending['items'][0]
 context={'workflow':name,'uid':uid,'nodeId':item['nodeId'],'contractDigest':item['contractDigest']}
 kubectl('-n',NS,'delete','parametertemplate',RUN);kubectl('-n',NS,'delete','workflowform',RUN)
 query={'input':context,'path':'/region','values':{'environment':'test'}}
 for key in ['uid','nodeId','contractDigest']:api('POST',PREFIX+'/options',{**query,'input':{**context,key:'wrong'}},400)
 api('POST',PREFIX+'/options',{**query,'binding':RUN},400)
 assert [o['value'] for o in api('POST',PREFIX+'/options',query)['items']]==['east']
 with sync_playwright() as pw:
  browser=pw.chromium.launch(headless=True,executable_path=os.environ['ARGO_PARAMETERIZATION_CHROMIUM']);page=browser.new_page(viewport={'width':1440,'height':1000})
  version=api('GET','/api/v1/version')['version'];page.add_init_script('localStorage.setItem("modal/ftu",JSON.stringify("dismissed"));localStorage.setItem("modal/version",JSON.stringify('+json.dumps(version.rsplit('.',1)[0])+'));')
  try:
   page.goto(BASE+'/workflows/'+NS+'/'+name)
   page.get_by_label('/environment',exact=True).select_option(json.dumps('test'))
   expect(page.get_by_label('/region',exact=True)).to_be_enabled()
   page.get_by_label('/region',exact=True).select_option(json.dumps('east'))
   page.get_by_label('/environment',exact=True).select_option(json.dumps('production'))
   expect(page.get_by_label('/region',exact=True).get_by_role('option',name='Production West',exact=True)).to_be_attached()
   expect(page.get_by_label('/region',exact=True).get_by_role('option',name='Previous selection is unavailable',exact=True)).to_be_attached()
   page.get_by_label('/region',exact=True).select_option(json.dumps('west'))
   # A server-side source change after UI lookup cannot authorize a stale choice.
   cm['data']['choices']=json.dumps([{'value':'west','label':'Production West','when':{'/environment':'production'},'disabled':True}]);apply(cm)
   request={'uid':uid,'resourceVersion':pending['resourceVersion'],'nodeId':item['nodeId'],'requestId':str(uuid.uuid4()),'values':{'environment':'production','region':'west'}}
   request['resourceVersion']=api('GET',PREFIX+'/inputs/'+name)['resourceVersion']
   api('POST',PREFIX+'/inputs/'+name,request,400)
   assert api('GET',PREFIX+'/inputs/'+name)['items']
   # Missing source access also fails explicitly and leaves the suspend pending.
   kubectl('-n',NS,'delete','configmap',RUN)
   api('POST',PREFIX+'/options',{**query,'values':{'environment':'production'}},404)
   cm['data']['choices']=json.dumps([{'value':'west','label':'Production West','when':{'/environment':'production'}}]);apply(cm)
   with page.expect_response(lambda r:r.request.method=='POST' and r.url.endswith('/inputs/'+name)) as accepted:
    page.get_by_role('button',name='Validate input and continue this node',exact=True).click()
   assert accepted.value.ok,accepted.value.text()
   done=wait_workflow(name,lambda w:w.get('status',{}).get('phase')=='Succeeded');assert done['metadata']['uid']==uid
   submitted=api('GET',PREFIX+'/inputs/'+name)['submitted'];assert len(submitted)==1 and '/region' in submitted[0]['optionVersions']
   assert any(p['value']=='west' for n in done['status']['nodes'].values() if n['type']=='Pod' for p in n.get('outputs',{}).get('parameters',[]) if p['name']=='region')
   kubectl('-n',NS,'delete','configmap',RUN)
   assert api('GET',PREFIX+'/inputs/'+name)['submitted']==submitted
   page.screenshot(path='/tmp/parameterization-input-options-success.png',full_page=True)
   print('PASS',json.dumps({'workflow':name,'uid':uid,'frozenSourcesDeleted':True,'browserDependencies':True,'staleSelectionRejected':True,'missingSourceRejected':True,'sourceVersionRecorded':True,'nativeOutput':'west'}),flush=True)
  except Exception:
   page.screenshot(path='/tmp/parameterization-input-options-failure.png',full_page=True);raise
  finally:browser.close()
if __name__=='__main__':main()
