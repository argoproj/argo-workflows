"""Actual browser -> form CR -> revision -> native Workflow -> ConfigMap, local k3s only.
Requires playwright (Python) and a local Chromium installation.
"""
import copy,json,os,subprocess,time,uuid
from pathlib import Path
from e2e import api,wait_workflow,NS,PREFIX,BASE
from template_refs_e2e import apply,resource,native,kubectl
from playwright.sync_api import sync_playwright,expect

RUN='form-ui-'+uuid.uuid4().hex[:8]
G='parameters.workflows.example.io'

def fixture():
 schema={'type':'object','additionalProperties':False,'required':['enabled','environment','region','settings','zones','extra'],'properties':{
  'enabled':{'type':'boolean'},'environment':{'type':'string','enum':['test','production']},'region':{'type':'string'},
  'settings':{'type':'object','additionalProperties':False,'required':['count','note'],'properties':{'count':{'type':'integer'},'note':{'type':'string'}}},
  'zones':{'type':'array','minItems':1,'items':{'type':'object','additionalProperties':False,'required':['name','active','count'],'properties':{'name':{'type':'string'},'active':{'type':'boolean'},'count':{'type':'integer'}}}},
  'extra':{'type':'object','additionalProperties':False,'required':['count'],'properties':{'count':{'type':'integer'}}}}}
 values={'enabled':False,'environment':'test','region':'east','settings':{'count':0,'note':'keep hidden value'},'zones':[{'name':'a','active':False,'count':0},{'name':'b','active':True,'count':2}],'extra':{'count':1}}
 form={'parameterTemplate':RUN,'layout':[{'id':'basic','kind':'tab','label':'Basics'},{'id':'network','kind':'tab','label':'Network'},{'id':'zones','parent':'network','kind':'tree','label':'Zone configuration'}],
  'options':{'region-catalog':{'kind':'configmap','name':RUN,'key':'regions','dependsOn':['/environment']}},
  'fields':[{'path':'/enabled','widget':'checkbox','label':'Enable changes','group':'basic'},
   {'path':'/environment','widget':'select','group':'basic'},
   {'path':'/region','widget':'select','group':'basic','optionSource':'region-catalog'},
   {'path':'/settings','group':'basic','enabledWhen':{'path':'/enabled','operator':'Equals','value':True}},
   {'path':'/settings/note','visibleWhen':{'path':'/environment','operator':'Equals','value':'production'}},
   {'path':'/zones','widget':'table','group':'zones'},
   {'path':'/zones/*/active','widget':'checkbox'},
   {'path':'/zones/*/count','enabledWhen':{'path':'/zones/*/active','operator':'Equals','value':True}},
   {'path':'/extra','widget':'yaml','group':'network'}]}
 apply({'apiVersion':'v1','kind':'ConfigMap','metadata':{'name':RUN,'namespace':NS},'data':{'regions':json.dumps([{'value':'east','label':'East','when':{'/environment':'test'}},{'value':'west','label':'West','when':{'/environment':'production'}}])}})
 apply(resource('ParameterTemplate',RUN,{'dialect':'http://json-schema.org/draft-07/schema#','schema':schema}))
 apply(resource('WorkflowForm',RUN,form))
 native_spec={'serviceAccountName':'workflow-executor','entrypoint':'write','arguments':{'parameters':[{'name':'values'}]},'templates':[{'name':'write','resource':{'action':'create','manifest':'apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: "{{workflow.name}}-result"\ndata:\n  values: {{=toJson(workflow.parameters.values)}}\n'}}]}
 apply(native('WorkflowTemplate',RUN,native_spec))
 apply(resource('WorkflowParameterBinding',RUN,{'workflowTemplate':RUN,'parameterTemplate':RUN,'form':RUN,'mappings':[{'path':'','parameter':'values','encoding':'json'}]}))
 saved=api('PUT',PREFIX+'/sets/'+RUN,{'template':RUN,'values':values,'resourceVersion':''})
 return form,values,saved

def intermediate(page):
 name=RUN+'-input'
 input_form={'parameterTemplate':'decision-v1','layout':[{'id':'review','kind':'tab','label':'Decision'},{'id':'scale','kind':'tab','label':'Scaling'}],
  'fields':[{'path':'/release','widget':'select','group':'review','label':'Release channel'},
   {'path':'/replicas','widget':'number','group':'scale','enabledWhen':{'path':'/release','operator':'Exists'}}]}
 apply(resource('WorkflowForm',name,input_form))
 template=api('GET','/api/v1/workflow-templates/'+NS+'/intermediate-demo')
 apply(native('WorkflowTemplate',name,template['spec']))
 binding={'workflowTemplate':name,'parameterTemplate':'empty-v1','mappings':[],'inputs':[{'template':'decision','parameterTemplate':'decision-v1','form':name,
  'mappings':[{'path':'/replicas','parameter':'count','encoding':'decimal'},{'path':'/release','parameter':'release','encoding':'string'}]}]}
 apply(resource('WorkflowParameterBinding',name,binding))
 saved=api('PUT',PREFIX+'/sets/'+name,{'template':'empty-v1','values':{},'resourceVersion':''})
 preview=api('POST',PREFIX+'/preview',{'binding':name,'revision':saved['spec']['revision']});assert not preview['errors'],preview
 wf=api('POST',PREFIX+'/submit',{'binding':name,'revision':saved['spec']['revision'],'executionDigest':preview['executionDigest'],'idempotencyKey':name})
 wait_workflow(wf['metadata']['name'],lambda w:any(n.get('type')=='Suspend' and n.get('phase')=='Running' for n in w.get('status',{}).get('nodes',{}).values()))
 # This execution must use its frozen form after the source definition changes.
 changed=copy.deepcopy(input_form);changed['layout'][0]['label']='Changed after submission'
 apply(resource('WorkflowForm',name,changed))
 page.goto(BASE+'/workflows/'+NS+'/'+wf['metadata']['name'])
 page.get_by_role('tab',name='Scaling',exact=True).click()
 expect(page.get_by_label('/replicas',exact=True)).to_be_disabled()
 page.get_by_role('tab',name='Decision',exact=True).click()
 page.get_by_label('/release',exact=True).select_option('"canary"')
 page.get_by_role('tab',name='Scaling',exact=True).click()
 page.get_by_label('/replicas',exact=True).fill('3')
 page.get_by_role('button',name='Validate input and continue this node',exact=True).click()
 done=wait_workflow(wf['metadata']['name'],lambda w:w.get('status',{}).get('phase')=='Succeeded')
 submitted=api('GET',PREFIX+'/inputs/'+done['metadata']['name'])
 assert len(submitted['submitted'])==1 and submitted['submitted'][0]['values']=={'release':'canary','replicas':3}
 logs=kubectl('-n',NS,'logs','-l','workflows.argoproj.io/workflow='+done['metadata']['name'],'-c','main').decode()
 assert 'replicas=3 release=canary' in logs
 return done['metadata']['name']

def main():
 form,initial,saved=fixture()
 with sync_playwright() as pw:
  executable=os.environ.get('ARGO_PARAMETERIZATION_CHROMIUM')
  browser=pw.chromium.launch(headless=True,**({'executable_path':executable} if executable else {}))
  page=browser.new_page(viewport={'width':1440,'height':1080})
  # Native welcome/news dialogs are outside this feature. Persist their normal
  # dismissed state before rendering, including the asynchronously loaded version.
  version=api('GET','/api/v1/version')['version']
  major_minor=version.rsplit('.',1)[0] if '.' in version else version
  page.add_init_script('localStorage.setItem("modal/ftu", JSON.stringify("dismissed")); localStorage.setItem("modal/version", JSON.stringify('+json.dumps(major_minor)+'));')
  errors=[];page.on('pageerror',lambda e:errors.append(str(e)))
  try:
   page.goto(BASE+'/parameters/'+NS+'?binding='+RUN)
   page.get_by_label('Parameter set',exact=True).fill(RUN)
   page.get_by_role('button',name='Load saved set',exact=True).click()
   expect(page.get_by_label('/settings/count',exact=True)).to_be_disabled()
   expect(page.get_by_label('/settings/note',exact=True)).to_have_count(0)
   page.get_by_label('/enabled',exact=True).check()
   page.get_by_label('/settings/count',exact=True).fill('3')
   page.get_by_label('/environment',exact=True).select_option('"production"')
   expect(page.get_by_label('/settings/note',exact=True)).to_have_value('keep hidden value')
   page.get_by_label('/region',exact=True).select_option('"west"')
   page.get_by_role('tab',name='Network',exact=True).click()
   expect(page.get_by_label('/zones/0/count',exact=True)).to_be_disabled()
   page.get_by_label('/zones/0/active',exact=True).check()
   page.get_by_label('/zones/0/count',exact=True).fill('4')
   page.get_by_label('Remove /zones/1',exact=True).click()
   page.get_by_role('button',name='Add zones item',exact=True).click()
   page.get_by_label('/zones/1/name',exact=True).fill('c')
   page.get_by_label('/zones/1/active',exact=True).check()
   page.get_by_label('/zones/1/count',exact=True).fill('5')
   page.get_by_label('/extra',exact=True).fill('count: [')
   expect(page.get_by_role('button',name='Save revision',exact=True)).to_be_disabled()
   page.get_by_role('tab',name='Basics',exact=True).click()
   expect(page.get_by_role('button',name='Save revision',exact=True)).to_be_disabled()
   page.get_by_role('tab',name='Network',exact=True).click()
   page.get_by_label('/extra',exact=True).fill('count: 0')
   expect(page.get_by_role('button',name='Save revision',exact=True)).to_be_enabled()
   with page.expect_response(lambda r: r.request.method=='PUT' and '/sets/'+RUN in r.url) as response:
    page.get_by_role('button',name='Save revision',exact=True).click()
   new=response.value.json();assert response.value.ok,new
   want={'enabled':True,'environment':'production','region':'west','settings':{'count':3,'note':'keep hidden value'},'zones':[{'name':'a','active':True,'count':4},{'name':'c','active':True,'count':5}],'extra':{'count':0}}
   assert new['spec']['values']==want,new
   revision=new['spec']['revision'];assert revision!=saved['spec']['revision']
   page.get_by_role('button',name='Validate and preview',exact=True).click()
   with page.expect_response(lambda r:r.request.method=='POST' and r.url.endswith('/submit')) as response:
    page.get_by_role('button',name='Submit this revision',exact=True).click()
   wf=response.value.json();assert response.value.ok,wf
   done=wait_workflow(wf['metadata']['name'],lambda w:w.get('status',{}).get('phase')=='Succeeded')
   result=json.loads(kubectl('-n',NS,'get','configmap',wf['metadata']['name']+'-result','-o','json'))
   assert json.loads(result['data']['values'])==want
   # Compare the saved versions and inspect the frozen presentation, not the live form.
   diff=api('POST',PREFIX+'/diff',{'left':saved['spec']['revision'],'right':revision});assert diff['changes']
   receipt=api('GET',PREFIX+'/catalog/workflowsubmissions?name='+done['metadata']['annotations'][G+'/submission'])
   assert receipt['spec']['definitions']['form']==form
   # Removing a now-selected option prevents API preview too, even though it was selected in the UI.
   apply({'apiVersion':'v1','kind':'ConfigMap','metadata':{'name':RUN,'namespace':NS},'data':{'regions':'[]'}})
   rejected=api('POST',PREFIX+'/preview',{'binding':RUN,'revision':revision});assert rejected['errors'][0]['code']=='OptionUnavailable'
   apply(resource('WorkflowForm',RUN,{**form,'layout':[{'id':'basic','parent':'basic','kind':'tab'}]}))
   api('POST',PREFIX+'/preview',{'binding':RUN,'revision':revision},400)
   input_workflow=intermediate(page)
   assert not errors,errors
   print('PASS',json.dumps({'run':RUN,'workflow':done['metadata']['name'],'phase':done['status']['phase'],'actualBrowser':True,'intermediateWorkflow':input_workflow,'revision':revision,'dynamicRows':2,'hiddenValueRetained':True,'formSnapshotFrozen':True,'invalidDefinitionsAndOptionsRejected':True}),flush=True)
  except Exception:
   page.screenshot(path=os.environ.get('ARGO_PARAMETERIZATION_SCREENSHOT','/tmp/parameterization-form-failure.png'),full_page=True)
   raise
  finally:
   # Keep the example usable after negative cases; no real service is contacted.
   apply(resource('WorkflowForm',RUN,form))
   apply({'apiVersion':'v1','kind':'ConfigMap','metadata':{'name':RUN,'namespace':NS},'data':{'regions':json.dumps([{'value':'east','label':'East','when':{'/environment':'test'}},{'value':'west','label':'West','when':{'/environment':'production'}}])}})
   browser.close()
if __name__=='__main__':main()
