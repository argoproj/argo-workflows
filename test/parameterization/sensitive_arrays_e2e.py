"""Local browser -> encrypted array inputs -> revisions/Diff -> native loop -> retry.
Uses synthetic credentials, namespace-scoped objects and no external resources.
"""
import copy,json,os,uuid
from e2e import api,wait_workflow,NS,PREFIX,BASE
from template_refs_e2e import native,apply,kubectl,resource
from playwright.sync_api import sync_playwright,expect

RUN='secret-array-'+uuid.uuid4().hex[:8]
G='parameters.workflows.example.io'

def main():
 ref={'type':'object','properties':{'secretRef':{'type':'object','properties':{'name':{'type':'string'},'key':{'type':'string'}},'required':['name','key'],'additionalProperties':False}},'required':['secretRef'],'additionalProperties':False}
 model={'dialect':'http://json-schema.org/draft-07/schema#','schema':{'type':'object','properties':{'accounts':{'type':'array','minItems':1,'maxItems':8,'items':{'type':'object','properties':{'id':{'type':'string','minLength':1},'credential':ref},'required':['id'],'additionalProperties':False}}},'required':['accounts'],'additionalProperties':False},'sensitivePaths':['/accounts/*/credential'],'secretSchemas':{'/accounts/*/credential':{'type':'string','minLength':8}},'identityRules':[{'path':'/accounts','keyPath':'/id'}]}
 apply(resource('ParameterTemplate',RUN,model))
 apply(resource('WorkflowForm',RUN,{'parameterTemplate':RUN,'fields':[{'path':'/accounts','widget':'table'},{'path':'/accounts/*/credential','widget':'secret'}]}))
 consume={'name':'consume','inputs':{'parameters':[{'name':'id'},{'name':'credential-name'},{'name':'credential-key'}]},'container':{'image':'python:3.12-alpine','command':['python','-c'],'args':['import os; assert os.environ["TOKEN"].startswith("synthetic-"+os.environ["ROW"]+"-"); print("consumed:"+os.environ["ROW"])'],'env':[{'name':'ROW','value':'{{inputs.parameters.id}}'},{'name':'TOKEN','valueFrom':{'secretKeyRef':{'name':'{{inputs.parameters.credential-name}}','key':'{{inputs.parameters.credential-key}}'}}}]}}
 spec={'serviceAccountName':'workflow-executor','entrypoint':'main','arguments':{'parameters':[{'name':'accounts'}]},'templates':[{'name':'main','dag':{'tasks':[{'name':'consume','template':'consume','withParam':'{{workflow.parameters.accounts}}','arguments':{'parameters':[{'name':'id','value':'{{item.id}}'},{'name':'credential-name','value':"{{=jsonpath(item.credential, '$.secretRef.name')}}"},{'name':'credential-key','value':"{{=jsonpath(item.credential, '$.secretRef.key')}}"}]}},{'name':'recover','depends':'consume.Succeeded','template':'recover'}]}},consume,{'name':'recover','resource':{'action':'get','manifest':'apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: {{workflow.name}}-ready\n'}}]}
 apply(native('WorkflowTemplate',RUN,spec))
 binding={'workflowTemplate':RUN,'parameterTemplate':RUN,'form':RUN,'mappings':[{'path':'/accounts','parameter':'accounts','encoding':'secret-reference-json'}]}
 apply(resource('WorkflowParameterBinding',RUN,binding))
 initial=api('PUT',PREFIX+'/sets/'+RUN,{'template':RUN,'values':{'accounts':[{'id':'alpha'},{'id':'beta'}]},'resourceVersion':''})
 plaintexts=['synthetic-'+row+'-'+uuid.uuid4().hex for row in ['alpha','beta','gamma']]
 with sync_playwright() as pw:
  browser=pw.chromium.launch(headless=True,executable_path=os.environ['ARGO_PARAMETERIZATION_CHROMIUM'])
  page=browser.new_page(viewport={'width':1440,'height':1080})
  version=api('GET','/api/v1/version')['version'];major_minor=version.rsplit('.',1)[0]
  page.add_init_script('localStorage.setItem("modal/ftu",JSON.stringify("dismissed"));localStorage.setItem("modal/version",JSON.stringify('+json.dumps(major_minor)+'));')
  try:
   page.goto(BASE+'/parameters/'+NS+'?binding='+RUN)
   page.get_by_label('Parameter set',exact=True).fill(RUN)
   page.get_by_role('button',name='Load saved set',exact=True).click()
   refs=[]
   for i in range(2):
    page.get_by_label('New secret /accounts/'+str(i)+'/credential',exact=True).fill(plaintexts[i])
    with page.expect_response(lambda r:r.request.method=='POST' and r.url.endswith('/secrets')) as response:
     page.get_by_role('button',name='Store new encrypted secret version',exact=True).nth(i).click()
    refs.append(response.value.json());assert response.value.ok
    expect(page.get_by_role('button',name='Save revision',exact=True)).to_be_enabled()
   with page.expect_response(lambda r:r.request.method=='PUT' and '/sets/'+RUN in r.url) as response:
    page.get_by_role('button',name='Save revision',exact=True).click()
   first=response.value.json();assert response.value.ok,first
   page.get_by_role('button',name='Move up',exact=True).nth(1).click()
   expect(page.get_by_label('/accounts/0/id',exact=True)).to_have_value('beta')
   expect(page.get_by_label('New secret /accounts/0/credential',exact=True)).to_have_value('')
   page.get_by_role('button',name='Add accounts item',exact=True).click()
   page.get_by_label('/accounts/2/id',exact=True).fill('gamma')
   page.get_by_label('New secret /accounts/2/credential',exact=True).fill(plaintexts[2])
   with page.expect_response(lambda r:r.request.method=='POST' and r.url.endswith('/secrets')) as response:
    page.get_by_role('button',name='Store new encrypted secret version',exact=True).nth(2).click()
   refs.append(response.value.json());assert response.value.ok
   expect(page.get_by_role('button',name='Save revision',exact=True)).to_be_enabled()
   page.get_by_label('Remove /accounts/1',exact=True).click()
   with page.expect_response(lambda r:r.request.method=='PUT' and '/sets/'+RUN in r.url) as response:
    page.get_by_role('button',name='Save revision',exact=True).click()
   second=response.value.json();assert response.value.ok,second
   assert second['spec']['values']=={'accounts':[{'id':'beta','credential':refs[1]},{'id':'gamma','credential':refs[2]}]}
   diff=api('POST',PREFIX+'/diff',{'left':first['spec']['revision'],'right':second['spec']['revision']})
   assert any(c['op']=='move' and c.get('itemIdentity')=='beta' for c in diff['changes'])
   bad=copy.deepcopy(second['spec']['values']);bad['accounts'][1]['credential']=plaintexts[2]
   api('PUT',PREFIX+'/sets/'+RUN+'-bad',{'template':RUN,'values':bad,'resourceVersion':''},400)
   page.get_by_role('button',name='Validate and preview',exact=True).click()
   with page.expect_response(lambda r:r.request.method=='POST' and r.url.endswith('/submit')) as response:
    page.get_by_role('button',name='Submit this revision',exact=True).click()
   wf=response.value.json();assert response.value.ok,wf
   failed=wait_workflow(wf['metadata']['name'],lambda w:w.get('status',{}).get('phase') in ['Failed','Error'])
   successes={k:(n['phase'],n.get('finishedAt')) for k,n in failed['status']['nodes'].items() if n['type']=='Pod' and n['phase']=='Succeeded'}
   assert len(successes)==2,failed['status']
   apply({'apiVersion':'v1','kind':'ConfigMap','metadata':{'name':wf['metadata']['name']+'-ready','namespace':NS},'data':{'ready':'true'}})
   retry={'workflow':wf['metadata']['name'],'uid':wf['metadata']['uid'],'resourceVersion':failed['metadata']['resourceVersion'],'idempotencyKey':RUN+'-retry'}
   assert api('POST',PREFIX+'/retry',retry)['metadata']['uid']==wf['metadata']['uid']
   done=wait_workflow(wf['metadata']['name'],lambda w:w.get('status',{}).get('phase')=='Succeeded')
   for k,state in successes.items():assert (done['status']['nodes'][k]['phase'],done['status']['nodes'][k].get('finishedAt'))==state
   logs=kubectl('-n',NS,'logs','-l','workflows.argoproj.io/workflow='+wf['metadata']['name'],'-c','main').decode()
   assert 'consumed:beta' in logs and 'consumed:gamma' in logs
   history=api('GET',PREFIX+'/history/'+RUN)
   receipt=api('GET',PREFIX+'/catalog/workflowsubmissions?name='+done['metadata']['annotations'][G+'/submission'])
   exposed=json.dumps([first,second,diff,done,history,receipt])+logs
   for secret in plaintexts:assert secret not in exposed
   kubectl('-n',NS,'delete','secret',refs[2]['secretRef']['name'])
   api('POST',PREFIX+'/preview',{'binding':RUN,'revision':second['spec']['revision']},400)
   page.screenshot(path='/tmp/parameterization-sensitive-arrays-success.png',full_page=True)
   print('PASS',json.dumps({'run':RUN,'workflow':wf['metadata']['name'],'sameWorkflowUID':done['metadata']['uid'],'retainedSecretConsumers':len(successes),'browserEncryptedInputs':3,'remainingRows':2,'diffMoves':True,'revocationRejected':True,'plaintextLeaks':False}),flush=True)
  except Exception:
   page.screenshot(path='/tmp/parameterization-sensitive-arrays-failure.png',full_page=True)
   raise
  finally:browser.close()
if __name__=='__main__':main()
