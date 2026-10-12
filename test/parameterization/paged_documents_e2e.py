"""Local API -> paged metadata/history -> exact revision restore -> native execution.
Unique synthetic resources only; no delivery system or remote services.
"""
import copy,json,urllib.parse,uuid
from e2e import api,PREFIX,NS,wait_workflow
from template_refs_e2e import apply,resource,native,kubectl
RUN='pages-'+uuid.uuid4().hex[:8]
def main():
 blob='synthetic-large-document-'*3000
 model={'dialect':'http://json-schema.org/draft-07/schema#','schema':{'type':'object','properties':{'count':{'type':'integer'},'document':{'type':'string'}},'required':['count','document']}}
 apply(resource('ParameterTemplate',RUN,model))
 # Force native Kubernetes pagination, independently of existing lab fixtures.
 apply({'apiVersion':'v1','kind':'List','items':[resource('ParameterTemplate',RUN+'-'+str(i),{'dialect':'http://json-schema.org/draft-07/schema#','schema':{'type':'object'}}) for i in range(51)]})
 catalog=[];cursor='';pages=0
 while True:
  page=api('GET',PREFIX+'/catalog/parametertemplates?view=metadata'+('&continue='+urllib.parse.quote(cursor,safe='') if cursor else ''))
  assert page['kind']=='PartialObjectMetadataList';assert len(page['items'])<=50
  for item in page['items']:
   assert 'spec' not in item and 'status' not in item
   assert 'annotations' not in item['metadata'] and 'managedFields' not in item['metadata']
  catalog+=page['items'];pages+=1;cursor=page.get('metadata',{}).get('continue','')
  if not cursor:break
  assert pages<50,'unexpected fixture growth'
 assert len([x for x in catalog if x['metadata']['name'].startswith(RUN)])==52
 assert pages>=2
 detail=api('GET',PREFIX+'/catalog/parametertemplates?name='+RUN)
 assert detail['spec']==model
 saved=None;revisions=[]
 for count in range(27):
  saved=api('PUT',PREFIX+'/sets/'+RUN,{'template':RUN,'values':{'count':count,'document':blob},'resourceVersion':saved['metadata']['resourceVersion'] if saved else '', 'message':'revision '+str(count)})
  revisions.append(saved['spec']['revision'])
 original=api('GET',PREFIX+'/revisions/'+revisions[0])
 orphan=copy.deepcopy(original['spec']);orphan['parent']=revisions[-1];orphan['message']='uncommitted CAS loser';orphan['errors']=orphan.get('errors') or []
 apply(resource('ParameterRevision',RUN+'-orphan',orphan))
 first=api('GET',PREFIX+'/history/'+RUN+'?view=summary')
 assert first['kind']=='ParameterRevisionSummaryList' and len(first['items'])==25
 assert all(not ({'values','model','sources','errors'} & set(x['spec'])) for x in first['items'])
 assert len(json.dumps(first))<40000
 # Advance the editable draft while keeping the cursor valid for older entries.
 saved=api('PUT',PREFIX+'/sets/'+RUN,{'template':RUN,'values':{'count':27,'document':blob},'resourceVersion':saved['metadata']['resourceVersion']})
 second=api('GET',PREFIX+'/history/'+RUN+'?view=summary&continue='+urllib.parse.quote(first['metadata']['continue'],safe=''))
 assert len(second['items'])==2 and not second.get('metadata',{}).get('continue')
 seen=[x['metadata']['name'] for x in first['items']+second['items']]
 assert seen==list(reversed(revisions)) and RUN+'-orphan' not in seen
 historic=api('GET',PREFIX+'/revisions/'+second['items'][-1]['metadata']['name'])
 assert historic['spec']['values']=={'count':0,'document':blob}
 restored=api('PUT',PREFIX+'/sets/'+RUN,{'template':RUN,'values':historic['spec']['values'],'restoredFrom':historic['metadata']['name'],'resourceVersion':saved['metadata']['resourceVersion']})
 restored_revision=api('GET',PREFIX+'/revisions/'+restored['spec']['revision'])
 assert restored_revision['spec']['restoredFrom']==revisions[0]
 assert api('GET',PREFIX+'/revisions/'+revisions[0])==original
 diff=api('POST',PREFIX+'/diff',{'left':revisions[0],'right':restored['spec']['revision']});assert not diff['changes']
 apply(native('WorkflowTemplate',RUN,{'entrypoint':'apply','serviceAccountName':'workflow-executor','arguments':{'parameters':[{'name':'count'}]},'templates':[{'name':'apply','resource':{'action':'create','manifest':'apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: {{workflow.name}}-result\ndata:\n  count: "{{workflow.parameters.count}}"\n'}}]}))
 apply(resource('WorkflowParameterBinding',RUN,{'workflowTemplate':RUN,'parameterTemplate':RUN,'mappings':[{'path':'/count','parameter':'count','encoding':'decimal'}]}))
 preview=api('POST',PREFIX+'/preview',{'binding':RUN,'revision':restored['spec']['revision']})
 assert not preview['errors']
 wf=api('POST',PREFIX+'/submit',{'binding':RUN,'revision':restored['spec']['revision'],'executionDigest':preview['executionDigest'],'idempotencyKey':RUN})
 done=wait_workflow(wf['metadata']['name'],lambda w:w.get('status',{}).get('phase')=='Succeeded')
 cm=json.loads(kubectl('-n',NS,'get','configmap',wf['metadata']['name']+'-result','-o','json'));assert cm['data']=={'count':'0'}
 print('PASS '+json.dumps({'run':RUN,'catalogPages':pages,'fixtureDefinitions':52,'historyPages':2,'historyVersions':27,'concurrentSavePreservedCursor':True,'orphanExcluded':True,'historicalDocumentBytes':len(blob),'historyFirstPageBytes':len(json.dumps(first)),'restoreCreatedRevision':restored['spec']['revision'],'workflow':wf['metadata']['name'],'phase':done['status']['phase']}),flush=True)
if __name__=='__main__':main()
