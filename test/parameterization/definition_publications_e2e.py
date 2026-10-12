"""Local browser publication/history/restore -> explicit binding -> native execution.
No production endpoints or delivery resources are used.
"""
import copy,json,os,time,uuid,subprocess
from e2e import api,wait_workflow,NS,PREFIX,BASE
from template_refs_e2e import native,apply,kubectl
from playwright.sync_api import sync_playwright,expect

NAME='000-pub-'+uuid.uuid4().hex[:8]
ANN='parameters.workflows.example.io/definition-'

def get(resource,name):return api('GET',PREFIX+'/catalog/'+resource+'?name='+name)
def create(resource,name,spec):return api('PUT',PREFIX+'/definitions/'+resource+'?name='+name,{'spec':spec,'uid':'','resourceVersion':''})
def publish(resource,source,name):
 return api('POST',PREFIX+'/definition-publications/'+resource,{'source':source['metadata']['name'],'uid':source['metadata']['uid'],'sourceSpec':source['spec'],'name':name,'message':'local version acceptance'})
def observed(resource,name):
 end=time.monotonic()+20
 while time.monotonic()<end:
  doc=get(resource,name)
  if doc.get('status',{}).get('observedGeneration')==doc['metadata'].get('generation'):return doc
  time.sleep(.1)
 raise AssertionError('definition status did not converge')
def denied_patch(resource,name,patch):
 try:kubectl('-n',NS,'patch',resource+'.parameters.workflows.example.io',name,'--type=merge','-p',json.dumps(patch))
 except subprocess.CalledProcessError as e:
  message=(e.stderr or b'').decode();assert 'immutable' in message or 'cannot be removed' in message,(e.stdout,e.stderr)
 else:raise AssertionError('published definition mutation was accepted')

def main():
 leaf={'name':'provision','inputs':{'parameters':[{'name':'count'}]},'outputs':{'parameters':[{'name':'resourceName','valueFrom':{'jsonPath':'{.metadata.name}'}}]},'resource':{'action':'create','manifest':'apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: "{{workflow.name}}-result"\ndata:\n  count: "{{inputs.parameters.count}}"\n'}}
 apply(native('WorkflowTemplate',NAME,{'serviceAccountName':'workflow-executor','entrypoint':'main','arguments':{'parameters':[{'name':'count'}]},'templates':[{'name':'main','dag':{'tasks':[{'name':'make','template':'provision','arguments':{'parameters':[{'name':'count','value':'{{workflow.parameters.count}}'}]}}]}},leaf]}))
 model={'dialect':'http://json-schema.org/draft-07/schema#','schema':{'type':'object','additionalProperties':False,'required':['count'],'properties':{'count':{'type':'integer','minimum':1,'default':2}}}}
 source=create('parametertemplates',NAME,model)
 workflow_ids=lambda:{x['metadata']['uid'] for x in json.loads(kubectl('-n',NS,'get','workflows','-o','json'))['items']}
 before=workflow_ids()
 with sync_playwright() as pw:
  browser=pw.chromium.launch(headless=True,executable_path=os.environ['ARGO_PARAMETERIZATION_CHROMIUM'])
  page=browser.new_page(viewport={'width':1440,'height':1080});errors=[];page.on('pageerror',lambda e:errors.append(str(e)))
  version=api('GET','/api/v1/version')['version'].rsplit('.',1)[0]
  page.add_init_script('localStorage.setItem("modal/ftu",JSON.stringify("dismissed"));localStorage.setItem("modal/version",JSON.stringify('+json.dumps(version)+'));')
  try:
   page.goto(BASE+'/parameters/'+NS)
   page.get_by_role('button',name='Resource definitions and records',exact=True).click()
   expect(page.get_by_role('button',name='Refresh resources',exact=True)).to_be_enabled()
   for _ in range(10):
    if page.get_by_role('button',name=NAME,exact=True).count():break
    page.get_by_role('button',name='Load more resources',exact=True).click()
   page.get_by_role('button',name=NAME,exact=True).click()
   page.get_by_role('button',name='Publish named version',exact=True).click()
   page.get_by_label('Publication note',exact=True).fill('local version acceptance')
   with page.expect_response(lambda r:r.request.method=='POST' and '/definition-publications/' in r.url) as response:
    page.get_by_role('button',name='Confirm publish definition',exact=True).click()
   v1=response.value.json();assert response.value.ok,v1
   expect(page.get_by_role('button',name='Edit definition spec',exact=True)).to_have_count(0)
   denied_patch('parametertemplates',NAME+'-v1',{'spec':{'schema':{'properties':{'count':{'default':9}}}}})
   denied_patch('parametertemplates',NAME+'-v1',{'metadata':{'annotations':{ANN+'digest':None}}})
   # A draft changes independently; another publication does not overwrite v1.
   current=observed('parametertemplates',NAME);changed=copy.deepcopy(model);changed['schema']['properties']['count']['default']=5
   current=api('PUT',PREFIX+'/definitions/parametertemplates?name='+NAME,{'spec':changed,'uid':current['metadata']['uid'],'resourceVersion':current['metadata']['resourceVersion']})
   v2=publish('parametertemplates',current,NAME+'-v2')
   kubectl('-n',NS,'delete','parametertemplates.parameters.workflows.example.io',NAME)
   # Lost-response recovery and history survive deletion of the mutable source.
   assert publish('parametertemplates',source,NAME+'-v1')['metadata']['uid']==v1['metadata']['uid']
   page.get_by_role('button',name='View published versions',exact=True).click()
   expect(page.get_by_role('button',name='Open version '+NAME+'-v2',exact=True)).to_be_visible()
   page.get_by_role('button',name='Compare '+NAME+'-v2 with selected definition',exact=True).click()
   expect(page.get_by_label('Definition version comparison')).to_contain_text('/schema/properties/count/default')
   page.get_by_role('button',name='Clone as new definition',exact=True).click()
   restored=NAME+'-restored'
   page.get_by_label('Definition name',exact=True).fill(restored)
   with page.expect_response(lambda r:r.request.method=='PUT' and '/definitions/parametertemplates' in r.url) as response:
    page.get_by_role('button',name='Save definition',exact=True).click()
   clone=response.value.json();assert response.value.ok,clone
   assert clone['spec']==model and ANN+'digest' not in clone['metadata'].get('annotations',{})
   expect(page.get_by_role('button',name='Edit definition spec',exact=True)).to_be_visible()
   # All six kinds remain ordinary native references; no extra Workflow is created by publication.
   restored_version=publish('parametertemplates',clone,restored+'-v1')
   form={'parameterTemplate':restored+'-v1','fields':[{'path':'/count','widget':'number','label':'Copies'}]}
   action={'inputSchema':{'type':'object','properties':{'count':{'type':'integer'}},'required':['count']},'outputSchema':{'type':'object','properties':{'resourceName':{'type':'string'}}},'template':leaf}
   result_spec={'title':'Created resource','source':{'parameter':'resourceName','format':'text'},'schema':{'type':'string'},'views':[{'title':'Resource','type':'text'}]}
   definitions=[('workflowforms',form),('actiondefinitions',action),('actionresulttemplates',result_spec),('workflowactionbindings',{'workflowTemplate':NAME,'calls':[{'template':'provision','actionDefinition':NAME+'-v1','resultBindings':[{'name':'resource','resultTemplate':NAME+'-v1'}]}]}),('workflowparameterbindings',{'workflowTemplate':NAME,'parameterTemplate':restored+'-v1','form':NAME+'-v1','actionBinding':NAME+'-v1','mappings':[{'path':'/count','parameter':'count','encoding':'decimal'}]})]
   for resource,spec in definitions:
    draft=create(resource,NAME,spec);published=publish(resource,draft,NAME+'-v1')
    assert published['spec']==spec
    modified=copy.deepcopy(spec)
    if resource=='workflowforms':modified['labels']={'/count':'changed'}
    elif resource=='actiondefinitions':modified['inputSchema']['title']='changed'
    elif resource=='actionresulttemplates':modified['title']='changed'
    else:modified['workflowTemplate']='changed'
    denied_patch(resource,NAME+'-v1',{'spec':modified})
   assert workflow_ids()==before,'publication/restore must not execute'
   page.goto(BASE+'/parameters/'+NS+'?binding='+NAME+'-v1')
   page.get_by_label('Parameter set',exact=True).fill(NAME)
   page.get_by_label('/count',exact=True).fill('3')
   with page.expect_response(lambda r:r.request.method=='PUT' and '/sets/'+NAME in r.url) as response:
    page.get_by_role('button',name='Save revision',exact=True).click()
   assert response.value.ok,response.value.json()
   expect(page.get_by_role('button',name='Validate and preview',exact=True)).to_be_enabled()
   page.get_by_role('button',name='Validate and preview',exact=True).click()
   expect(page.get_by_role('button',name='Submit this revision',exact=True)).to_be_visible()
   with page.expect_response(lambda r:r.request.method=='POST' and r.url.endswith('/submit')) as response:
    page.get_by_role('button',name='Submit this revision',exact=True).click()
   wf=response.value.json();assert response.value.ok,wf
   done=wait_workflow(wf['metadata']['name'],lambda w:w.get('status',{}).get('phase')=='Succeeded')
   result=json.loads(kubectl('-n',NS,'get','configmap',done['metadata']['name']+'-result','-o','json'))
   assert result['data']['count']=='3'
   assert workflow_ids()-before=={done['metadata']['uid']}
   assert get('parametertemplates',NAME+'-v1')['spec']==model
   assert get('parametertemplates',NAME+'-v2')['spec']==changed
   assert not errors,errors
   page.screenshot(path='/tmp/parameterization-publications-success.png',full_page=True)
   print('PASS',json.dumps({'name':NAME,'publishedKinds':6,'immutableAPIServerProtection':True,'sourceDeletionHistory':True,'lostResponseSameUID':True,'restoredModel':restored,'workflow':done['metadata']['name'],'phase':done['status']['phase']}),flush=True)
  except Exception:
   page.screenshot(path='/tmp/parameterization-publications-failure.png',full_page=True);raise
  finally:browser.close()
if __name__=='__main__':main()
