"""Local full path: typed parameters -> nested refs -> frozen execution -> original retry.
All resources are synthetic and names are per-run; no existing templates change.
"""
import copy,json,os,subprocess,uuid
from e2e import api,wait_workflow,PREFIX,NS
CONTEXT=os.environ.get('ARGO_PARAMETERIZATION_CONTEXT','k3d-argo-parameterization')
assert CONTEXT=='k3d-argo-parameterization'
RUN='refs-'+uuid.uuid4().hex[:8]
CLUSTER_ROOT=os.environ.get('ARGO_PARAMETERIZATION_CLUSTER_ROOT')=='true'
GROUP='parameters.workflows.example.io/v1alpha1'
def kubectl(*args,body=None):
 r=subprocess.run(['kubectl','--context',CONTEXT,*args],input=json.dumps(body).encode() if body else None,capture_output=True,check=True)
 return r.stdout

def resource(kind,name,spec):
 return {'apiVersion':GROUP,'kind':kind,'metadata':{'name':name,'namespace':NS},'spec':spec}
def native(kind,name,spec):
 return {'apiVersion':'argoproj.io/v1alpha1','kind':kind,'metadata':{'name':name,**({} if kind=='ClusterWorkflowTemplate' else {'namespace':NS}),'labels':{'workflows.argoproj.io/controller-instanceid':'parameterization-lab'}},'spec':spec}
def apply(obj):kubectl('apply','-f','-',body=obj)

def main():
 model=resource('ParameterTemplate',RUN,{'dialect':'http://json-schema.org/draft-07/schema#','schema':{'type':'object','properties':{'values':{'type':'array','items':{'type':'string'},'minItems':1}},'required':['values'],'additionalProperties':False}})
 apply(model)
 # Independent cluster-scoped leaf, namespaced nested steps and DAG root.
 leaf=native('ClusterWorkflowTemplate',RUN+'-leaf',{'templates':[{'name':'emit','inputs':{'parameters':[{'name':'value'}]},'container':{'image':'alpine:3.22','command':['sh','-ec'],'args':['test "$VALUE" != ""; printf "%s" "$VALUE" > /tmp/result'],'env':[{'name':'VALUE','value':'{{inputs.parameters.value}}'}]},'outputs':{'parameters':[{'name':'echo','valueFrom':{'path':'/tmp/result'}}]}}]})
 apply(leaf)
 library=native('WorkflowTemplate',RUN+'-lib',{'templates':[{'name':'sequence','inputs':{'parameters':[{'name':'value'}]},'steps':[[{'name':'emit','templateRef':{'name':RUN+'-leaf','template':'emit','clusterScope':True},'arguments':{'parameters':[{'name':'value','value':'{{inputs.parameters.value}}'}]}}]],'outputs':{'parameters':[{'name':'echo','valueFrom':{'parameter':'{{steps.emit.outputs.parameters.echo}}'}}]}}]})
 apply(library)
 root=native('ClusterWorkflowTemplate' if CLUSTER_ROOT else 'WorkflowTemplate',RUN,{'entrypoint':'main','serviceAccountName':'workflow-executor','arguments':{'parameters':[{'name':'values'}]},'templates':[
 {'name':'main','dag':{'tasks':[
  {'name':'approve','template':'approve'},
  {'name':'nested','depends':'approve.Succeeded','templateRef':{'name':RUN+'-lib','template':'sequence'},'withParam':'{{workflow.parameters.values}}','arguments':{'parameters':[{'name':'value','value':'{{item}}'}]}},
  {'name':'recover','depends':'nested.Succeeded','template':'recover'}]}},
 {'name':'approve','suspend':{}},
 {'name':'recover','resource':{'action':'get','manifest':'apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: {{workflow.name}}-recovery\n'}}]})
 apply(root)
 binding=resource('WorkflowParameterBinding',RUN,{'workflowTemplate':RUN,'clusterScope':CLUSTER_ROOT,'parameterTemplate':RUN,'mappings':[{'path':'/values','parameter':'values','encoding':'json'}]})
 apply(binding)
 saved=api('PUT',PREFIX+'/sets/'+RUN,{'template':RUN,'values':{'values':['alpha','beta','gamma']},'resourceVersion':''})
 preview=api('POST',PREFIX+'/preview',{'binding':RUN,'revision':saved['spec']['revision']})
 assert len([k for k in preview['definitionVersions'] if k.startswith('templateRef/')])==2
 assert preview['definitionVersions']['workflowTemplate'].startswith('ClusterWorkflowTemplate/')==CLUSTER_ROOT
 assert '"templateRef"' not in json.dumps(preview['workflow']['spec']['templates'])
 # Changes between preview and commit are rejected.
 leaf['spec']['templates'][0]['metadata']={'labels':{'source-revision':'2'}};apply(leaf)
 submit={'binding':RUN,'revision':saved['spec']['revision'],'executionDigest':preview['executionDigest'],'idempotencyKey':RUN}
 api('POST',PREFIX+'/submit',submit,409)
 preview=api('POST',PREFIX+'/preview',{'binding':RUN,'revision':saved['spec']['revision']})
 submit['executionDigest']=preview['executionDigest']
 wf=api('POST',PREFIX+'/submit',submit);name=wf['metadata']['name'];uid=wf['metadata']['uid']
 source_label='workflows.argoproj.io/'+('cluster-workflow-template' if CLUSTER_ROOT else 'workflow-template')
 assert wf['metadata']['labels'][source_label]==RUN
 listed=json.loads(kubectl('-n',NS,'get','workflow','-l',source_label+'='+RUN,'-o','json'))['items']
 assert [w['metadata']['uid'] for w in listed]==[uid], 'native template history must include the frozen execution'
 wait_workflow(name,lambda w:any(n.get('type')=='Suspend' and n.get('phase')=='Running' for n in w.get('status',{}).get('nodes',{}).values()))
 # Delete only this test's source definitions after the execution was frozen.
 kubectl('delete','clusterworkflowtemplate',RUN+'-leaf')
 kubectl('-n',NS,'delete','workflowtemplate',RUN+'-lib')
 if CLUSTER_ROOT:kubectl('delete','clusterworkflowtemplate',RUN)
 api('PUT','/api/v1/workflows/'+NS+'/'+name+'/resume',{})
 failed=wait_workflow(name,lambda w:w.get('status',{}).get('phase') in ('Failed','Error'))
 assert any(n.get('displayName')=='recover' and n.get('phase') in ('Failed','Error') for n in failed['status']['nodes'].values()),failed['status']
 echoes=[]
 retained={}
 for key,node in failed['status']['nodes'].items():
  if node['type'] in ('Pod','Suspend') and node['phase']=='Succeeded':retained[key]=(node['phase'],node.get('finishedAt'))
  if node['type']=='Pod':
   echoes.extend(p['value'] for p in node.get('outputs',{}).get('parameters',[]) if p['name']=='echo')
 assert sorted(echoes)==['alpha','beta','gamma'],echoes
 apply({'apiVersion':'v1','kind':'ConfigMap','metadata':{'name':name+'-recovery','namespace':NS},'data':{'allow':'true'}})
 failed=api('GET','/api/v1/workflows/'+NS+'/'+name)
 retry={'workflow':name,'uid':uid,'resourceVersion':failed['metadata']['resourceVersion'],'idempotencyKey':RUN+'-retry'}
 api('POST',PREFIX+'/retry',retry)
 done=wait_workflow(name,lambda w:w.get('status',{}).get('phase')=='Succeeded')
 assert done['metadata']['uid']==uid
 for key,status in retained.items():assert (done['status']['nodes'][key]['phase'],done['status']['nodes'][key].get('finishedAt'))==status
 print('PASS',json.dumps({'run':RUN,'workflow':name,'uid':uid,'clusterRoot':CLUSTER_ROOT,'nestedReferences':2,'loopOutputs':echoes,'retainedNodes':len(retained),'sourceDefinitionsDeletedAfterSubmit':True}),flush=True)
if __name__=='__main__':main()
