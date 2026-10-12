"""Local-only execution -> paged layered comparison -> frozen native execution.

Creates unique synthetic definitions and one Workflow; never retries a business run.
"""
import json,uuid
from e2e import api,PREFIX,NS,wait_workflow
from template_refs_e2e import apply,resource,native,kubectl

def main():
 run='execution-pages-'+uuid.uuid4().hex[:8]
 model={'dialect':'http://json-schema.org/draft-07/schema#','schema':{'type':'object','properties':{'items':{'type':'object','additionalProperties':{'type':'integer'}}},'required':['items']}}
 apply(resource('ParameterTemplate',run,model))
 form={'parameterTemplate':run,'labels':{'/items/item%03d'%i:'Original %d'%i for i in range(130)}}
 apply(resource('WorkflowForm',run,form))
 apply(native('WorkflowTemplate',run,{'entrypoint':'apply','suspend':True,'serviceAccountName':'workflow-executor','arguments':{'parameters':[{'name':'count'}]},'templates':[{'name':'apply','resource':{'action':'create','manifest':'apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: {{workflow.name}}-result\ndata:\n  count: "{{workflow.parameters.count}}"\n'}}]}))
 apply(resource('WorkflowParameterBinding',run,{'workflowTemplate':run,'parameterTemplate':run,'form':run,'mappings':[{'path':'/items/item000','parameter':'count','encoding':'decimal'}]}))
 original={'items':{'item%03d'%i:i for i in range(130)}}
 first=api('PUT',PREFIX+'/sets/'+run,{'template':run,'values':original,'resourceVersion':''})
 preview=api('POST',PREFIX+'/preview',{'binding':run,'revision':first['spec']['revision']})
 wf=api('POST',PREFIX+'/submit',{'binding':run,'revision':first['spec']['revision'],'executionDigest':preview['executionDigest'],'idempotencyKey':run})
 name=wf['metadata']['name'];source={'workflow':name,'uid':wf['metadata']['uid']}
 changed={'items':{k:v+1 for k,v in original['items'].items()}}
 second=api('PUT',PREFIX+'/sets/'+run,{'template':run,'values':changed,'resourceVersion':first['metadata']['resourceVersion']})
 form['labels']={k:v.replace('Original','Changed') for k,v in form['labels'].items()}
 apply(resource('WorkflowForm',run,form))
 req={'source':source,'revision':second['spec']['revision']}
 rows=[];cursor='';pages=0;snapshot='';first_cursor=''
 while True:
  page=api('POST',PREFIX+'/execution-diff',{**req,'continue':cursor,'snapshot':snapshot})
  assert page['pagination']['offset']==len(rows)
  current=[]
  for key in ['changes','modelChanges','sourceChanges']:
   current += [(key,item) for item in page['parameters'][key]]
  for layer in page['layers']:
   current += [(layer['name'],item) for item in layer['changes']]
  assert len(current)<=100
  rows+=current;pages+=1
  assert page['parameters']['totals']['changes']==130
  assert next(x for x in page['layers'] if x['name']=='Form')['total']==130
  cursor=page['pagination']['continue'];snapshot=page['pagination']['snapshot']
  if pages==1:
   first_cursor=cursor
   api('POST',PREFIX+'/execution-diff',{**req,'source':{**source,'uid':'wrong'},'continue':cursor,'snapshot':snapshot},400)
  if not cursor:break
  assert pages<6
 assert pages==3 and len(rows)==page['pagination']['total']
 assert len({(layer,item['path'],item['op']) for layer,item in rows})==len(rows)
 assert len([1 for layer,_ in rows if layer=='changes'])==130
 assert len([1 for layer,_ in rows if layer=='Form'])==130
 assert any(layer=='Compiled inputs' for layer,_ in rows)
 api('POST',PREFIX+'/execution-diff',{**req,'limit':201},400)
 # A live definition changes between requests: neither forward nor backward
 # navigation may silently join a new comparison to old visible pages.
 form['labels']['/items/item000']='Changed again';apply(resource('WorkflowForm',run,form))
 api('POST',PREFIX+'/execution-diff',{**req,'continue':first_cursor,'snapshot':snapshot},400)
 api('POST',PREFIX+'/execution-diff',{**req,'snapshot':snapshot},400)
 refreshed=api('POST',PREFIX+'/execution-diff',req)
 assert refreshed['pagination']['snapshot']!=snapshot
 unchanged=api('GET','/api/v1/workflows/'+NS+'/'+name)
 assert unchanged['spec']==wf['spec'] and unchanged['metadata']['uid']==source['uid']
 matching=json.loads(kubectl('-n',NS,'get','workflows','-l','workflows.argoproj.io/workflow-template='+run,'-o','json'))['items']
 assert len(matching)==1
 api('PUT','/api/v1/workflows/'+NS+'/'+name+'/resume',{})
 done=wait_workflow(name,lambda w:w.get('status',{}).get('phase')=='Succeeded')
 cm=json.loads(kubectl('-n',NS,'get','configmap',name+'-result','-o','json'))
 assert cm['data']=={'count':'0'}
 print('PASS '+json.dumps({'run':run,'pages':pages,'changes':len(rows),'workflow':name,'workflowCount':1,'comparisonWorkflowWrites':0,'frozenResult':cm['data'],'phase':done['status']['phase']}),flush=True)

if __name__=='__main__':main()
