"""Browser definition authoring -> form -> immutable revision -> native action.
Runs only against the isolated local parameterization lab.
"""
import copy,json,os,uuid,time
from e2e import api,wait_workflow,NS,PREFIX,BASE
from template_refs_e2e import native,apply,kubectl
from playwright.sync_api import sync_playwright,expect

NAME='000-def-'+uuid.uuid4().hex[:8]

def write(page,resource,spec):
 page.get_by_label('Parameter resource type',exact=True).select_option(resource)
 expect(page.get_by_role('button',name='Refresh resources',exact=True)).to_be_enabled()
 page.get_by_role('button',name='Create definition',exact=True).click()
 page.get_by_label('Definition name',exact=True).fill(NAME)
 page.get_by_label('Definition spec YAML',exact=True).fill(json.dumps(spec))
 page.get_by_role('button',name='Validate definition',exact=True).click()
 expect(page.get_by_text('Installed CRD schema validation passed.',exact=False)).to_be_visible()
 api('GET',PREFIX+'/catalog/'+resource+'?name='+NAME,expected=404)
 with page.expect_response(lambda r:r.request.method=='PUT' and '/definitions/'+resource in r.url) as response:
  page.get_by_role('button',name='Save definition',exact=True).click()
 saved=response.value.json();assert response.value.ok,saved
 expect(page.get_by_role('button',name='Clone as new definition',exact=True)).to_be_visible()
 return saved

def observed(resource):
 # A second author obtains its own current version after asynchronous status.
 until=time.monotonic()+20
 while time.monotonic()<until:
  doc=api('GET',PREFIX+'/catalog/'+resource+'?name='+NAME)
  if doc.get('status',{}).get('observedGeneration')==doc['metadata'].get('generation'):return doc
  time.sleep(.1)
 raise AssertionError('definition observation timed out')

def main():
 provision={'name':'provision','inputs':{'parameters':[{'name':'count'}]},'outputs':{'parameters':[{'name':'resourceName','valueFrom':{'jsonPath':'{.metadata.name}'}}]},'resource':{'action':'create','manifest':'apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: "{{workflow.name}}-result"\ndata:\n  count: "{{inputs.parameters.count}}"\n'}}
 spec={'serviceAccountName':'workflow-executor','entrypoint':'main','arguments':{'parameters':[{'name':'count'}]},'templates':[{'name':'main','dag':{'tasks':[{'name':'make','template':'provision','arguments':{'parameters':[{'name':'count','value':'{{workflow.parameters.count}}'}]}}]}},provision]}
 apply(native('WorkflowTemplate',NAME,spec))
 model={'dialect':'http://json-schema.org/draft-07/schema#','schema':{'type':'object','additionalProperties':False,'required':['count'],'properties':{'count':{'type':'integer','minimum':1,'default':2}}}}
 form={'parameterTemplate':NAME,'layout':[{'id':'main','kind':'tab','label':'Workload'}],'fields':[{'path':'/count','widget':'number','group':'main','label':'Copies'}]}
 action={'inputSchema':{'type':'object','properties':{'count':{'type':'integer'}},'required':['count']},'outputSchema':{'type':'object','properties':{'resourceName':{'type':'string'}}},'template':provision}
 action_binding={'workflowTemplate':NAME,'calls':[{'template':'provision','actionDefinition':NAME}]}
 binding={'workflowTemplate':NAME,'parameterTemplate':NAME,'form':NAME,'actionBinding':NAME,'mappings':[{'path':'/count','parameter':'count','encoding':'decimal'}]}
 with sync_playwright() as pw:
  browser=pw.chromium.launch(headless=True,executable_path=os.environ['ARGO_PARAMETERIZATION_CHROMIUM'])
  page=browser.new_page(viewport={'width':1440,'height':1080})
  version=api('GET','/api/v1/version')['version'];version=version.rsplit('.',1)[0] if '.' in version else version
  page.add_init_script('localStorage.setItem("modal/ftu", JSON.stringify("dismissed"));localStorage.setItem("modal/version",JSON.stringify('+json.dumps(version)+'));')
  errors=[];page.on('pageerror',lambda e:errors.append(str(e)))
  try:
   page.goto(BASE+'/parameters/'+NS)
   page.get_by_role('button',name='Resource definitions and records',exact=True).click()
   write(page,'parametertemplates',model)
   saved=write(page,'workflowforms',form)
   assert observed('workflowforms')['metadata']['resourceVersion']!=saved['metadata']['resourceVersion']
   # The page still holds the pre-status RV. A normal edit must recover once.
   page.get_by_role('button',name='Edit definition spec',exact=True).click()
   page.get_by_role('button',name='Save definition',exact=True).click()
   expect(page.get_by_role('button',name='Clone as new definition',exact=True)).to_be_visible()
   # A second author edits the same live form while the first editor remains open.
   page.get_by_role('button',name='Edit definition spec',exact=True).click()
   local=copy.deepcopy(form);local['fields'][0]['label']='Unsaved local label'
   page.get_by_label('Definition spec YAML',exact=True).fill(json.dumps(local))
   remote=copy.deepcopy(form);remote['fields'][0]['label']='Concurrent author label'
   saved=observed('workflowforms')
   saved_remote=api('PUT',PREFIX+'/definitions/workflowforms?name='+NAME,{'spec':remote,'uid':saved['metadata']['uid'],'resourceVersion':saved['metadata']['resourceVersion'],'validateOnly':False})
   page.get_by_role('button',name='Save definition',exact=True).click()
   expect(page.get_by_role('region',name='Definition editor',exact=True).get_by_role('alert')).to_contain_text('definition changed')
   expect(page.get_by_label('Definition spec YAML')).to_have_value(json.dumps(local))
   assert api('GET',PREFIX+'/catalog/workflowforms?name='+NAME)['spec']==remote
   saved_remote=observed('workflowforms')
   invalid={**remote,'unexpectedDefinitionField':True}
   api('PUT',PREFIX+'/definitions/workflowforms?name='+NAME,{'spec':invalid,'uid':saved_remote['metadata']['uid'],'resourceVersion':saved_remote['metadata']['resourceVersion'],'validateOnly':False},400)
   original_action=write(page,'actiondefinitions',action)
   page.get_by_role('button',name='Clone as new definition',exact=True).click()
   expect(page.get_by_label('Definition name')).to_have_value(NAME+'-next')
   with page.expect_response(lambda r:r.request.method=='PUT' and '/definitions/actiondefinitions' in r.url) as response:
    page.get_by_role('button',name='Save definition',exact=True).click()
   cloned=response.value.json();assert response.value.ok and cloned['metadata']['uid']!=original_action['metadata']['uid']
   assert api('GET',PREFIX+'/catalog/actiondefinitions?name='+NAME)['spec']==action
   original_action=observed('actiondefinitions')
   api('PUT',PREFIX+'/definitions/actiondefinitions?name='+NAME,{'spec':{**action,'inputSchema':{'type':'object'}},'uid':original_action['metadata']['uid'],'resourceVersion':original_action['metadata']['resourceVersion']},400)
   action_binding['calls'][0]['actionDefinition']=NAME+'-next'
   write(page,'workflowactionbindings',action_binding)
   write(page,'workflowparameterbindings',binding)
   page.goto(BASE+'/parameters/'+NS+'?binding='+NAME)
   page.get_by_label('Parameter set',exact=True).fill(NAME)
   expect(page.get_by_role('tab',name='Workload',exact=True)).to_be_visible()
   page.get_by_label('/count',exact=True).fill('4')
   with page.expect_response(lambda r:r.request.method=='PUT' and '/sets/'+NAME in r.url) as response:
    page.get_by_role('button',name='Save revision',exact=True).click()
   revision=response.value.json()['spec']['revision']
   page.get_by_role('button',name='Validate and preview',exact=True).click()
   expect(page.get_by_role('button',name='Submit this revision',exact=True)).to_be_visible()
   page.screenshot(path='/tmp/parameterization-definition-success.png',full_page=True)
   with page.expect_response(lambda r:r.request.method=='POST' and r.url.endswith('/submit')) as response:
    page.get_by_role('button',name='Submit this revision',exact=True).click()
   wf=response.value.json();assert response.value.ok,wf
   done=wait_workflow(wf['metadata']['name'],lambda w:w.get('status',{}).get('phase')=='Succeeded')
   result=json.loads(kubectl('-n',NS,'get','configmap',done['metadata']['name']+'-result','-o','json'))
   assert result['data']['count']=='4'
   receipt=api('GET',PREFIX+'/catalog/workflowsubmissions?name='+done['metadata']['annotations']['parameters.workflows.example.io/submission'])
   assert receipt['spec']['definitions']['form']==remote
   # Execution receipts never use the mutable definition editor endpoint.
   api('PUT',PREFIX+'/definitions/parameterrevisions?name='+revision,{'spec':{}},400)
   assert not errors,errors
   print('PASS',json.dumps({'name':NAME,'createdKinds':5,'workflow':done['metadata']['name'],'phase':done['status']['phase'],'dryRunNoWrite':True,'concurrentEditRejected':True,'unknownFieldsRejected':True,'frozenDefinitionVerified':True,'statusOnlyEditRecovered':True}),flush=True)
  except Exception:
   page.screenshot(path='/tmp/parameterization-definition-failure.png',full_page=True)
   raise
  finally:browser.close()
if __name__=='__main__':main()
