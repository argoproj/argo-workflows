"""Save -> ref binding -> stale protection -> frozen native Pods -> original retry."""
import copy,json,uuid
from e2e import api,wait_workflow,PREFIX,NS
from template_refs_e2e import resource,native,apply,kubectl
RUN='action-ref-'+uuid.uuid4().hex[:8]

def main():
 schema={'type':'object','properties':{'value':{'type':'string'}},'required':['value'],'additionalProperties':False}
 apply(resource('ParameterTemplate',RUN,{'dialect':'http://json-schema.org/draft-07/schema#','schema':schema}))
 leaf={'name':'emit','inputs':{'parameters':[{'name':'value'}]},'container':{'image':'alpine:3.22','command':['sh','-ec'],'args':['printf "%s" "$VALUE" >/tmp/result'],'env':[{'name':'VALUE','value':'{{inputs.parameters.value}}'}]},'outputs':{'parameters':[{'name':'value','valueFrom':{'path':'/tmp/result'}}]}}
 libraries=[]
 for suffix,cluster in [('ns',False),('cluster',True)]:
  name=RUN+'-'+suffix
  library=native('ClusterWorkflowTemplate' if cluster else 'WorkflowTemplate',name,{'podMetadata':{'labels':{'action-source':suffix}},'templates':[copy.deepcopy(leaf)]})
  libraries.append(library);apply(library)
  action=resource('ActionDefinition',name,{'inputSchema':schema,'outputSchema':schema,'implementationRef':{'name':name,'template':'emit','clusterScope':cluster}})
  apply(action)
  # API server itself enforces exactly one implementation before persistence.
  invalid=copy.deepcopy(action);invalid['metadata']['name']=name+'-invalid';invalid['spec']['template']=leaf
  import subprocess
  rejected=subprocess.run(['kubectl','--context','k3d-argo-parameterization','apply','--dry-run=server','-f','-'],input=json.dumps(invalid).encode(),capture_output=True)
  assert rejected.returncode and b'exactly one' in rejected.stderr,rejected.stderr
 root=native('WorkflowTemplate',RUN,{'entrypoint':'main','serviceAccountName':'workflow-executor','arguments':{'parameters':[{'name':'value'}]},'templates':[
  {'name':'main','dag':{'tasks':[
   {'name':'approve','template':'approve'},
   {'name':'first','dependencies':['approve'],'template':'first','arguments':{'parameters':[{'name':'value','value':'{{workflow.parameters.value}}'}]}},
   {'name':'second','dependencies':['first'],'template':'second','arguments':{'parameters':[{'name':'value','value':'{{tasks.first.outputs.parameters.value}}'}]}},
   {'name':'recover','dependencies':['second'],'template':'recover'}]}},
  {'name':'approve','suspend':{}},{'name':'first','suspend':{}},{'name':'second','suspend':{}},
  {'name':'recover','resource':{'action':'get','manifest':'apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: {{workflow.name}}-recovery\n'}}]})
 apply(root)
 apply(resource('WorkflowActionBinding',RUN,{'workflowTemplate':RUN,'calls':[{'template':'first','actionDefinition':RUN+'-ns'},{'template':'second','actionDefinition':RUN+'-cluster'}]}))
 apply(resource('WorkflowParameterBinding',RUN,{'workflowTemplate':RUN,'parameterTemplate':RUN,'actionBinding':RUN,'mappings':[{'path':'/value','parameter':'value','encoding':'string'}]}))
 kubectl('-n',NS,'wait','--for=condition=DefinitionValid','workflowparameterbinding/'+RUN,'--timeout=45s')
 saved=api('PUT',PREFIX+'/sets/'+RUN,{'template':RUN,'values':{'value':'original'},'resourceVersion':''})
 preview=api('POST',PREFIX+'/preview',{'binding':RUN,'revision':saved['spec']['revision']})
 assert len([key for key in preview['definitionVersions'] if key.startswith('templateRef/')])==2
 assert preview['definitions']['actions'][RUN+'-ns']['implementationRef']['name']==RUN+'-ns'
 for name in ['first','second']:
  frozen=next(t for t in preview['workflow']['spec']['templates'] if t['name']==name)
  assert frozen['container'] and frozen['metadata']['labels']['action-source']
 libraries[0]['spec']['podMetadata']['labels']['version']='two';apply(libraries[0])
 submit={'binding':RUN,'revision':saved['spec']['revision'],'executionDigest':preview['executionDigest'],'idempotencyKey':RUN}
 api('POST',PREFIX+'/submit',submit,409)
 preview=api('POST',PREFIX+'/preview',{'binding':RUN,'revision':saved['spec']['revision']})
 submit['executionDigest']=preview['executionDigest'];wf=api('POST',PREFIX+'/submit',submit);name=wf['metadata']['name'];uid=wf['metadata']['uid']
 wait_workflow(name,lambda w:any(n.get('type')=='Suspend' and n.get('phase')=='Running' for n in w.get('status',{}).get('nodes',{}).values()))
 # Deleting only these synthetic source libraries cannot change a submitted execution.
 kubectl('-n',NS,'delete','workflowtemplate',RUN+'-ns')
 kubectl('delete','clusterworkflowtemplate',RUN+'-cluster')
 kubectl('-n',NS,'wait','--for=condition=DefinitionValid=false','workflowparameterbinding/'+RUN,'--timeout=45s')
 api('PUT','/api/v1/workflows/'+NS+'/'+name+'/resume',{})
 failed=wait_workflow(name,lambda w:w.get('status',{}).get('phase') in ('Failed','Error'))
 outputs=[p['value'] for n in failed['status']['nodes'].values() for p in n.get('outputs',{}).get('parameters',[]) if p['name']=='value']
 assert outputs==['original','original'],outputs
 retained={k:(n['phase'],n.get('finishedAt'),n.get('outputs')) for k,n in failed['status']['nodes'].items() if n['type'] in ('Pod','Suspend') and n['phase']=='Succeeded'}
 pods=json.loads(kubectl('-n',NS,'get','pods','-l','workflows.argoproj.io/workflow='+name,'-o','json'))['items']
 pod_ids={p['metadata']['name']:p['metadata']['uid'] for p in pods if p['status']['phase']=='Succeeded'}
 assert len(pod_ids)==2
 assert {p['metadata']['labels'].get('action-source') for p in pods if p['status']['phase']=='Succeeded'}=={'ns','cluster'}
 apply({'apiVersion':'v1','kind':'ConfigMap','metadata':{'name':name+'-recovery','namespace':NS},'data':{'ok':'true'}})
 failed=api('GET','/api/v1/workflows/'+NS+'/'+name)
 retry={'workflow':name,'uid':uid,'resourceVersion':failed['metadata']['resourceVersion'],'idempotencyKey':RUN+'-retry'}
 api('POST',PREFIX+'/retry',retry)
 done=wait_workflow(name,lambda w:w.get('status',{}).get('phase')=='Succeeded')
 assert done['metadata']['uid']==uid
 for key,value in retained.items():
  node=done['status']['nodes'][key];assert (node['phase'],node.get('finishedAt'),node.get('outputs'))==value
 pods=json.loads(kubectl('-n',NS,'get','pods','-l','workflows.argoproj.io/workflow='+name,'-o','json'))['items']
 for pod,identity in pod_ids.items():assert next(p['metadata']['uid'] for p in pods if p['metadata']['name']==pod)==identity
 assert api('POST',PREFIX+'/retry',retry)['metadata']['uid']==uid
 print('PASS',json.dumps({'workflow':name,'uid':uid,'scopes':['namespaced','cluster'],'frozenSourcesDeleted':True,'retainedRecords':len(retained),'retainedPods':len(pod_ids),'outputs':outputs}),flush=True)
if __name__=='__main__':main()
