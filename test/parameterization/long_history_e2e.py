"""Local-only complete path: >1000 saves -> paged committed history -> restore -> execute."""
import copy,json,time,urllib.parse,uuid
from e2e import api,PREFIX,NS,wait_workflow
from template_refs_e2e import apply,resource,native,kubectl

def main():
 run='long-history-'+uuid.uuid4().hex[:8]
 apply(resource('ParameterTemplate',run,{'dialect':'http://json-schema.org/draft-07/schema#','schema':{'type':'object','properties':{'count':{'type':'integer'}},'required':['count']}}))
 saved=None;revisions=[];start=time.monotonic()
 for i in range(1006):
  assert time.monotonic()-start<240,'bounded local fixture creation exceeded four minutes'
  saved=api('PUT',PREFIX+'/sets/'+run,{'template':run,'values':{'count':i},'resourceVersion':saved['metadata']['resourceVersion'] if saved else ''})
  revisions.append(saved['spec']['revision'])
  if (i+1)%200==0:print('saved',i+1,flush=True)
 original=api('GET',PREFIX+'/revisions/'+revisions[0])
 orphan=copy.deepcopy(original['spec']);orphan['parent']=revisions[-1];orphan['message']='uncommitted revision';orphan['errors']=orphan.get('errors') or []
 apply(resource('ParameterRevision',run+'-orphan',orphan))
 seen=[];cursor='';pages=0
 while True:
  page=api('GET',PREFIX+'/history/'+run+'?view=summary'+('&continue='+urllib.parse.quote(cursor,safe='') if cursor else ''))
  assert len(page['items'])<=25
  assert all(not ({'values','model','sources','errors'} & set(x['spec'])) for x in page['items'])
  seen += [x['metadata']['name'] for x in page['items']]
  cursor=page.get('metadata',{}).get('continue','');pages+=1
  if pages==1:
   api('GET',PREFIX+'/history/'+run+'?continue='+urllib.parse.quote(cursor+'tampered',safe=''),expected=410)
   saved=api('PUT',PREFIX+'/sets/'+run,{'template':run,'values':{'count':99999},'resourceVersion':saved['metadata']['resourceVersion']})
  if not cursor:break
  assert pages<42
 assert pages==41 and seen==list(reversed(revisions)) and run+'-orphan' not in seen
 fresh=api('GET',PREFIX+'/history/'+run+'?view=summary')
 assert fresh['items'][0]['metadata']['name']==saved['spec']['revision']
 restored=api('PUT',PREFIX+'/sets/'+run,{'template':run,'values':original['spec']['values'],'restoredFrom':original['metadata']['name'],'resourceVersion':saved['metadata']['resourceVersion']})
 assert api('GET',PREFIX+'/revisions/'+revisions[0])==original
 assert not api('POST',PREFIX+'/diff',{'left':revisions[0],'right':restored['spec']['revision']})['changes']
 apply(native('WorkflowTemplate',run,{'entrypoint':'apply','serviceAccountName':'workflow-executor','arguments':{'parameters':[{'name':'count'}]},'templates':[{'name':'apply','resource':{'action':'create','manifest':'apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: {{workflow.name}}-result\ndata:\n  count: "{{workflow.parameters.count}}"\n'}}]}))
 apply(resource('WorkflowParameterBinding',run,{'workflowTemplate':run,'parameterTemplate':run,'mappings':[{'path':'/count','parameter':'count','encoding':'decimal'}]}))
 preview=api('POST',PREFIX+'/preview',{'binding':run,'revision':restored['spec']['revision']})
 wf=api('POST',PREFIX+'/submit',{'binding':run,'revision':restored['spec']['revision'],'executionDigest':preview['executionDigest'],'idempotencyKey':run})
 done=wait_workflow(wf['metadata']['name'],lambda w:w.get('status',{}).get('phase')=='Succeeded')
 cm=json.loads(kubectl('-n',NS,'get','configmap',wf['metadata']['name']+'-result','-o','json'))
 assert cm['data']=={'count':'0'}
 print('PASS '+json.dumps({'run':run,'versions':1006,'pages':pages,'orphanExcluded':True,'concurrentHeadChange':True,'workflow':wf['metadata']['name'],'result':cm['data'],'phase':done['status']['phase']}),flush=True)

if __name__=='__main__':main()
