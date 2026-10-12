"""Optional dynamic fields: initial UI -> preview -> frozen input UI -> native Pod."""
import json,os,uuid
from e2e import api,wait_workflow,PREFIX,NS,BASE
from template_refs_e2e import resource,native,apply
from playwright.sync_api import sync_playwright,expect
RUN='optional-options-'+uuid.uuid4().hex[:8]

def fixture(prefix=RUN):
 objects=[]
 optional={'environment':{'type':'string'},'region':{'type':'string'}}
 initial={'dialect':'http://json-schema.org/draft-07/schema#','schema':{'type':'object','required':['count'],'additionalProperties':False,'properties':{'count':{'type':'integer','minimum':1},**optional}}}
 runtime={'dialect':'http://json-schema.org/draft-07/schema#','schema':{'type':'object','required':['approved'],'additionalProperties':False,'properties':{'approved':{'type':'boolean','const':True},**optional}}}
 def form(name):return {'parameterTemplate':name,'options':{'/region':{'kind':'configmap','name':prefix+'-absent','key':'choices','dependsOn':['/environment']}},'fields':[{'path':'/region','widget':'select'}]}
 for name,model in [(prefix,initial),(prefix+'-input',runtime)]:
  objects.append(resource('ParameterTemplate',name,model));f=form(name)
  if name.endswith('-input'):f['fields'].append({'path':'/approved','widget':'checkbox'})
  objects.append(resource('WorkflowForm',name,f))
 spec={'serviceAccountName':'workflow-executor','entrypoint':'main','arguments':{'parameters':[{'name':'count'}]},'templates':[
  {'name':'main','dag':{'tasks':[{'name':'choose','template':'choose'},{'name':'consume','dependencies':['choose'],'template':'consume','arguments':{'parameters':[{'name':'approved','value':'{{tasks.choose.outputs.parameters.approved}}'}]}}]}},
  {'name':'choose','suspend':{},'outputs':{'parameters':[{'name':'approved','valueFrom':{'supplied':{}}}]}},
  {'name':'consume','inputs':{'parameters':[{'name':'approved'}]},'container':{'image':'alpine:3.22','command':['sh','-ec'],'args':['test "$COUNT:$APPROVED" = 2:true; printf "%s" "$COUNT:$APPROVED" >/tmp/result'],'env':[{'name':'COUNT','value':'{{workflow.parameters.count}}'},{'name':'APPROVED','value':'{{inputs.parameters.approved}}'}]},'outputs':{'parameters':[{'name':'observed','valueFrom':{'path':'/tmp/result'}}]}}]}
 objects.append(native('WorkflowTemplate',prefix,spec))
 objects.append(resource('WorkflowParameterBinding',prefix,{'workflowTemplate':prefix,'parameterTemplate':prefix,'form':prefix,'mappings':[{'path':'/count','parameter':'count','encoding':'decimal'}],'inputs':[{'template':'choose','parameterTemplate':prefix+'-input','form':prefix+'-input','mappings':[{'path':'/approved','parameter':'approved','encoding':'boolean'}]}]}))
 return objects

def main():
 for obj in fixture():apply(obj)
 saved=api('PUT',PREFIX+'/sets/'+RUN,{'template':RUN,'values':{'count':2},'resourceVersion':''})
 with sync_playwright() as pw:
  browser=pw.chromium.launch(headless=True,executable_path=os.environ['ARGO_PARAMETERIZATION_CHROMIUM']);page=browser.new_page(viewport={'width':1440,'height':1000})
  version=api('GET','/api/v1/version')['version'];page.add_init_script('localStorage.setItem("modal/ftu",JSON.stringify("dismissed"));localStorage.setItem("modal/version",JSON.stringify('+json.dumps(version.rsplit('.',1)[0])+'));')
  option_requests=[];page.on('request',lambda r:option_requests.append(r.url) if r.url.endswith('/options') else None)
  try:
   page.goto(BASE+'/parameters/'+NS+'?binding='+RUN)
   page.get_by_label('Parameter set',exact=True).fill(RUN);page.get_by_role('button',name='Load saved set',exact=True).click()
   expect(page.get_by_label('/count',exact=True)).to_have_value('2')
   expect(page.get_by_text('Choose dependent values first: /environment',exact=True)).to_be_visible()
   expect(page.get_by_label('/region',exact=True)).to_be_disabled()
   with page.expect_response(lambda r:r.request.method=='POST' and r.url.endswith('/preview')) as response:
    page.get_by_role('button',name='Validate and preview',exact=True).click()
   preview=response.value.json();assert response.value.ok and not preview['errors'],preview
   assert not [k for k in preview['definitionVersions'] if k.startswith('options/')]
   with page.expect_response(lambda r:r.request.method=='POST' and r.url.endswith('/submit')) as response:
    page.get_by_role('button',name='Submit this revision',exact=True).click()
   wf=response.value.json();assert response.value.ok,wf
   name=wf['metadata']['name'];uid=wf['metadata']['uid']
   wait_workflow(name,lambda w:any(n.get('type')=='Suspend' and n.get('phase')=='Running' for n in w.get('status',{}).get('nodes',{}).values()))
   pending=api('GET',PREFIX+'/inputs/'+name);node=pending['items'][0]
   request={'uid':uid,'resourceVersion':pending['resourceVersion'],'nodeId':node['nodeId'],'requestId':RUN+'-invalid','values':{}}
   api('POST',PREFIX+'/inputs/'+name,request,400)
   api('POST',PREFIX+'/inputs/'+name,{**request,'values':{'approved':True,'region':'east'}},400)
   assert not api('GET',PREFIX+'/inputs/'+name)['submitted']
   page.goto(BASE+'/workflows/'+NS+'/'+name)
   expect(page.get_by_text('Choose dependent values first: /environment',exact=True)).to_be_visible()
   expect(page.get_by_label('/region',exact=True)).to_be_disabled()
   page.get_by_label('/approved',exact=True).check()
   with page.expect_response(lambda r:r.request.method=='POST' and r.url.endswith('/inputs/'+name)) as response:
    page.get_by_role('button',name='Validate input and continue this node',exact=True).click()
   assert response.value.ok,response.value.text()
   done=wait_workflow(name,lambda w:w.get('status',{}).get('phase')=='Succeeded');assert done['metadata']['uid']==uid
   submitted=api('GET',PREFIX+'/inputs/'+name)['submitted'];assert len(submitted)==1 and not submitted[0].get('optionVersions')
   assert any(p['value']=='2:true' for n in done['status']['nodes'].values() for p in n.get('outputs',{}).get('parameters',[]) if p['name']=='observed')
   assert not option_requests,option_requests
   page.screenshot(path='/tmp/parameterization-optional-options-success.png',full_page=True)
   print('PASS',json.dumps({'workflow':name,'uid':uid,'initialAndRuntimeBrowser':True,'missingOptionalProviderNotConsumed':True,'requiredAndSuppliedInvalidFieldsRejected':True,'nativeOutput':'2:true'}),flush=True)
  except Exception:
   page.screenshot(path='/tmp/parameterization-optional-options-failure.png',full_page=True);raise
  finally:browser.close()
if __name__=='__main__':main()
