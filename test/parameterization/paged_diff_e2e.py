"""Local-only save -> paged semantic diff -> restore -> native Resource execution."""
import json,uuid
from e2e import api,PREFIX,NS,wait_workflow
from template_refs_e2e import apply,resource,native,kubectl

def main():
 run='diff-pages-'+uuid.uuid4().hex[:8]
 model={'dialect':'http://json-schema.org/draft-07/schema#','schema':{'type':'object','properties':{'items':{'type':'object','additionalProperties':{'type':'integer'}}},'required':['items']}}
 apply(resource('ParameterTemplate',run,model))
 original={'items':{'item%03d'%i:i for i in range(260)}}
 changed={'items':{k:v+1 for k,v in original['items'].items()}}
 first=api('PUT',PREFIX+'/sets/'+run,{'template':run,'values':original,'resourceVersion':''})
 second=api('PUT',PREFIX+'/sets/'+run,{'template':run,'values':changed,'resourceVersion':first['metadata']['resourceVersion']})
 pair={'left':first['spec']['revision'],'right':second['spec']['revision']}
 seen=[];cursor='';pages=0
 while True:
  page=api('POST',PREFIX+'/diff',{**pair,'continue':cursor})
  assert page['pagination']['offset']==len(seen) and page['pagination']['total']==260
  assert len(page['changes'])<=100 and not page['modelChanges'] and not page['sourceChanges']
  assert page['totals']['changes']==260
  seen+=page['changes'];pages+=1
  if pages==1:
   api('POST',PREFIX+'/diff',{'left':pair['left'],'right':pair['left'],'continue':page['pagination']['continue']},400)
  cursor=page['pagination']['continue']
  if not cursor:break
  assert pages<4
 assert pages==3 and len({x['path'] for x in seen})==260
 for i,change in enumerate(seen):
  assert change['path']=='/items/item%03d'%i and change['before']==i and change['after']==i+1
 api('POST',PREFIX+'/diff',{**pair,'limit':201},400)
 restored=api('PUT',PREFIX+'/sets/'+run,{'template':run,'values':original,'restoredFrom':pair['left'],'resourceVersion':second['metadata']['resourceVersion']})
 assert not api('POST',PREFIX+'/diff',{'left':pair['left'],'right':restored['spec']['revision']})['changes']
 apply(native('WorkflowTemplate',run,{'entrypoint':'apply','serviceAccountName':'workflow-executor','arguments':{'parameters':[{'name':'count'}]},'templates':[{'name':'apply','resource':{'action':'create','manifest':'apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: {{workflow.name}}-result\ndata:\n  count: "{{workflow.parameters.count}}"\n'}}]}))
 apply(resource('WorkflowParameterBinding',run,{'workflowTemplate':run,'parameterTemplate':run,'mappings':[{'path':'/items/item000','parameter':'count','encoding':'decimal'}]}))
 preview=api('POST',PREFIX+'/preview',{'binding':run,'revision':restored['spec']['revision']});assert not preview['errors']
 wf=api('POST',PREFIX+'/submit',{'binding':run,'revision':restored['spec']['revision'],'executionDigest':preview['executionDigest'],'idempotencyKey':run})
 done=wait_workflow(wf['metadata']['name'],lambda w:w.get('status',{}).get('phase')=='Succeeded')
 result=json.loads(kubectl('-n',NS,'get','configmap',wf['metadata']['name']+'-result','-o','json'))
 assert result['data']=={'count':'0'}
 print('PASS '+json.dumps({'run':run,'pages':pages,'changes':len(seen),'restoredVersion':restored['spec']['revision'],'workflow':wf['metadata']['name'],'phase':done['status']['phase']}),flush=True)

if __name__=='__main__':main()
