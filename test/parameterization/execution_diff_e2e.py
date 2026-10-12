"""Local full path: saved execution -> definitions advance -> layered Diff -> frozen execution.
Only unique synthetic resources are changed. Comparing never submits or retries.
"""
import copy,json,uuid
from e2e import api,PREFIX,NS,wait_workflow
from template_refs_e2e import apply,resource,native,kubectl
RUN='execution-diff-'+uuid.uuid4().hex[:8]
GROUP='parameters.workflows.example.io'
def main():
 apply(resource('ParameterTemplate',RUN,{'dialect':'http://json-schema.org/draft-07/schema#','schema':{'type':'object','properties':{'count':{'type':'integer','minimum':0}},'required':['count'],'additionalProperties':False}}))
 action={'inputSchema':{'type':'object'},'outputSchema':{'type':'object'},'template':{'name':'apply','resource':{'action':'create','manifest':'apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: {{workflow.name}}-result\ndata:\n  implementation: original\n  count: "{{workflow.parameters.count}}"\n'}}}
 apply(resource('ActionDefinition',RUN+'-old',action))
 calls={'workflowTemplate':RUN,'calls':[{'template':'apply','actionDefinition':RUN+'-old'}]}
 apply(resource('WorkflowActionBinding',RUN,calls))
 form={'parameterTemplate':RUN,'labels':{'/count':'Original count'}}
 apply(resource('WorkflowForm',RUN,form))
 wf_spec={'entrypoint':'apply','suspend':True,'serviceAccountName':'workflow-executor','arguments':{'parameters':[{'name':'count'}]},'templates':[{'name':'apply','container':{'image':'alpine:3.22','command':['echo','placeholder']}}]}
 apply(native('WorkflowTemplate',RUN,wf_spec))
 binding={'workflowTemplate':RUN,'parameterTemplate':RUN,'form':RUN,'actionBinding':RUN,'mappings':[{'path':'/count','parameter':'count','encoding':'decimal'}]}
 apply(resource('WorkflowParameterBinding',RUN,binding))
 one=api('PUT',PREFIX+'/sets/'+RUN,{'template':RUN,'values':{'count':2},'resourceVersion':''})
 preview=api('POST',PREFIX+'/preview',{'binding':RUN,'revision':one['spec']['revision']})
 assert preview['definitions']['parameterBinding']==binding
 wf=api('POST',PREFIX+'/submit',{'binding':RUN,'revision':one['spec']['revision'],'executionDigest':preview['executionDigest'],'idempotencyKey':RUN})
 name=wf['metadata']['name'];uid=wf['metadata']['uid'];source={'workflow':name,'uid':uid}
 receipt=api('GET',PREFIX+'/catalog/workflowsubmissions?name='+wf['metadata']['annotations'][GROUP+'/submission'])
 assert RUN+'-old' in receipt['spec']['definitions']['actions']
 # Change every definition layer with the same parameter version; old source
 # definitions are removed, so reading latest cannot accidentally pass this test.
 action['template']['resource']['manifest']=action['template']['resource']['manifest'].replace('original','replacement')
 action['inputSchema']['additionalProperties']=False
 apply(resource('ActionDefinition',RUN+'-new',action))
 calls['calls'][0]['actionDefinition']=RUN+'-new';calls['workflowTemplate']=RUN+'-new'
 apply(resource('WorkflowActionBinding',RUN+'-new',calls))
 form['labels']['/count']='Changed count';apply(resource('WorkflowForm',RUN,form))
 apply(native('WorkflowTemplate',RUN+'-new',wf_spec))
 binding['workflowTemplate']=RUN+'-new';binding['actionBinding']=RUN+'-new';apply(resource('WorkflowParameterBinding',RUN,binding))
 kubectl('-n',NS,'delete','actiondefinition',RUN+'-old')
 kubectl('-n',NS,'delete','workflowtemplate',RUN)
 before={x['metadata']['uid'] for x in json.loads(kubectl('-n',NS,'get','workflows','-o','json'))['items']}
 diff=api('POST',PREFIX+'/execution-diff',{'source':source,'revision':one['spec']['revision']})
 assert not diff['parameters']['changes']
 layers={x['name']:x for x in diff['layers']}
 for key in ['Parameter binding','Action binding','Form','Action definitions','Action contracts','Implementations and workflow structure']:
  assert layers[key]['available'] and layers[key]['changes'],key
 assert not layers['Compiled inputs']['changes']
 assert RUN+'-old' in json.dumps(diff) and 'original' in json.dumps(layers['Implementations and workflow structure'])
 two=api('PUT',PREFIX+'/sets/'+RUN,{'template':RUN,'values':{'count':4},'resourceVersion':one['metadata']['resourceVersion']})
 changed=api('POST',PREFIX+'/execution-diff',{'source':source,'revision':two['spec']['revision']})
 assert changed['parameters']['changes'] and next(x for x in changed['layers'] if x['name']=='Compiled inputs')['changes']
 api('POST',PREFIX+'/execution-diff',{'source':{**source,'uid':'wrong-uid'},'revision':two['spec']['revision']},400)
 foreign=api('PUT',PREFIX+'/sets/'+RUN+'-other',{'template':RUN,'values':{'count':4},'resourceVersion':''})
 api('POST',PREFIX+'/execution-diff',{'source':source,'revision':foreign['spec']['revision']},400)
 assert before=={x['metadata']['uid'] for x in json.loads(kubectl('-n',NS,'get','workflows','-o','json'))['items']}
 unchanged=api('GET','/api/v1/workflows/'+NS+'/'+name)
 assert unchanged['spec']==wf['spec'] and unchanged['metadata']['uid']==uid
 # The original native Workflow still executes the frozen implementation even
 # though its definitions disappeared. Compare must not silently adopt a draft.
 api('PUT','/api/v1/workflows/'+NS+'/'+name+'/resume',{})
 complete=wait_workflow(name,lambda w:w.get('status',{}).get('phase')=='Succeeded')
 cm=json.loads(kubectl('-n',NS,'get','configmap',name+'-result','-o','json'))
 assert cm['data']=={'implementation':'original','count':'2'}
 print(json.dumps({'run':RUN,'workflow':name,'uid':uid,'layers':[x['name'] for x in diff['layers']],'comparisonWorkflowWrites':0,'originalFrozenResult':cm['data'],'phase':complete['status']['phase']}),flush=True)
if __name__=='__main__':main()
