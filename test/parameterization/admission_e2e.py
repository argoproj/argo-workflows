"""API preview/save -> direct Kubernetes admission -> native execution/controls.
Includes invalid and denied imports, dry-run, contract tampering and no-Pod checks.
"""
import copy,json,subprocess,uuid
from e2e import api,PREFIX,NS,wait_workflow
from template_refs_e2e import resource,native,apply,kubectl
RUN='admission-'+uuid.uuid4().hex[:8];G='parameters.workflows.example.io'

def raw_create(obj,*options):
 import os
 return subprocess.run(['kubectl','--context',os.environ.get('ARGO_PARAMETERIZATION_CONTEXT','k3d-argo-parameterization'),*options,'create','-f','-','-o','json'],input=json.dumps(obj),text=True,capture_output=True)

def main():
 apply(resource('ParameterTemplate',RUN,{'dialect':'http://json-schema.org/draft-07/schema#','schema':{'type':'object','properties':{'count':{'type':'integer','minimum':1}},'required':['count'],'additionalProperties':False}}))
 apply(native('WorkflowTemplate',RUN,{'entrypoint':'hold','serviceAccountName':'workflow-executor','arguments':{'parameters':[{'name':'count'}]},'templates':[{'name':'hold','suspend':{}}]}))
 apply(resource('WorkflowParameterBinding',RUN,{'workflowTemplate':RUN,'parameterTemplate':RUN,'mappings':[{'path':'/count','parameter':'count','encoding':'decimal'}]}))
 saved=api('PUT',PREFIX+'/sets/'+RUN,{'template':RUN,'values':{'count':1},'resourceVersion':''})
 preview=api('POST',PREFIX+'/preview',{'binding':RUN,'revision':saved['spec']['revision']})
 wf=preview['workflow'];wf['metadata'].update(name=RUN,annotations={G+'/submission':RUN,G+'/revision':saved['spec']['revision'],G+'/execution-digest':preview['executionDigest']},labels={'workflows.argoproj.io/controller-instanceid':'parameterization-lab'})
 receipt=resource('WorkflowSubmission',RUN,{'requestDigest':'native-demo','revision':saved['spec']['revision'],'binding':RUN,'executionDigest':preview['executionDigest'],'workflow':wf,'definitionVersions':preview['definitionVersions']});apply(receipt)
 # New typed definition snapshots must agree with the resolved contract. The
 # original receipt above intentionally omits snapshots to exercise compatibility.
 forged=copy.deepcopy(receipt);forged['metadata']['name']=RUN+'-forged'
 forged_wf=forged['spec']['workflow'];forged_wf['metadata']['name']=RUN+'-forged';forged_wf['metadata']['annotations'][G+'/submission']=RUN+'-forged'
 forged['spec']['definitions']=copy.deepcopy(preview['definitions']);forged['spec']['definitions']['parameterBinding']['workflowTemplate']='forged-template'
 apply(forged);denied=raw_create(forged_wf);assert denied.returncode and 'definition snapshot' in denied.stderr,denied.stderr
 bad=copy.deepcopy(wf);bad['spec']['arguments']['parameters'][0]['value']='wrong'
 denied=raw_create(bad);assert denied.returncode and 'frozen submission' in denied.stderr,denied.stderr
 api('POST','/api/v1/workflows/'+NS,{'namespace':NS,'workflow':bad},400)
 bad=copy.deepcopy(wf);bad['spec']['serviceAccountName']='argo';denied=raw_create(bad);assert denied.returncode
 # A caller able to create Workflows but unable to read revisions is rejected.
 actor=RUN+'-reader'
 apply({'apiVersion':'v1','kind':'ServiceAccount','metadata':{'name':actor,'namespace':NS}})
 apply({'apiVersion':'rbac.authorization.k8s.io/v1','kind':'Role','metadata':{'name':actor,'namespace':NS},'rules':[{'apiGroups':['argoproj.io'],'resources':['workflows'],'verbs':['create']}]})
 apply({'apiVersion':'rbac.authorization.k8s.io/v1','kind':'RoleBinding','metadata':{'name':actor,'namespace':NS},'subjects':[{'kind':'ServiceAccount','name':actor,'namespace':NS}],'roleRef':{'apiGroup':'rbac.authorization.k8s.io','kind':'Role','name':actor}})
 denied=raw_create(wf,'--as=system:serviceaccount:'+NS+':'+actor);assert denied.returncode and 'frozen WorkflowSubmission' in denied.stderr,denied.stderr
 # Admission dry-run performs all checks without starting execution.
 result=raw_create(wf,'--dry-run=server');assert result.returncode==0,result.stderr
 assert not json.loads(kubectl('-n',NS,'get','pods','-l','workflows.argoproj.io/workflow='+RUN,'-o','json'))['items']
 result=raw_create(wf);assert result.returncode==0,result.stderr
 current=wait_workflow(RUN,lambda w:any(n.get('type')=='Suspend' and n.get('phase')=='Running' for n in w.get('status',{}).get('nodes',{}).values()))
 # Strip annotations or alter an argument through kubectl: both denied.
 for patch in [{'metadata':{'annotations':{G+'/revision':None,G+'/submission':None,G+'/execution-digest':None}}},{'spec':{'arguments':{'parameters':[{'name':'count','value':'0'}]}}}]:
  try:kubectl('-n',NS,'patch','workflow',RUN,'--type=merge','-p',json.dumps(patch))
  except subprocess.CalledProcessError:pass
  else:raise AssertionError('contract tampering was accepted')
 # Removing mutable source definitions does not invalidate a running Workflow.
 kubectl('-n',NS,'delete','workflowtemplate',RUN)
 kubectl('-n',NS,'delete','workflowparameterbinding',RUN)
 api('PUT','/api/v1/workflows/'+NS+'/'+RUN+'/resume',{})
 done=wait_workflow(RUN,lambda w:w.get('status',{}).get('phase')=='Succeeded');assert done['metadata']['uid']==current['metadata']['uid']
 # Plain native workflows remain outside the extension's admission dependency.
 plain=native('Workflow',RUN+'-plain',{'entrypoint':'wait','templates':[{'name':'wait','suspend':{}}]});apply(plain)
 api('PUT','/api/v1/workflows/'+NS+'/'+RUN+'-plain/terminate',{})
 print('PASS',json.dumps({'run':RUN,'workflowUID':done['metadata']['uid'],'cases':['forged-definition-snapshot-rejected','legacy-receipt-compatible','direct-api-type-rejection','kubectl-contract-rejection','execution-identity-rejection','caller-rbac','server-dry-run-no-execution','valid-native-create','identity-removal-rejected','argument-tampering-rejected','source-deletion-resume','unrelated-native-workflow']}),flush=True)

if __name__=='__main__':main()
