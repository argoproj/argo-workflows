"""Install the optional admission verifier ONLY in the dedicated local k3s lab.
Creates a private, lab-only serving CA; no system trust store is changed.
"""
import base64,json,os,subprocess,tempfile
from pathlib import Path
from template_refs_e2e import apply,kubectl
from e2e import NS
NAME='parameterization-admission'
G='parameters.workflows.example.io'

def main():
 assert os.environ.get('ARGO_PARAMETERIZATION_CONTEXT','k3d-argo-parameterization')=='k3d-argo-parameterization'
 existing=json.loads(kubectl('-n',NS,'get','secret',NAME+'-tls','--ignore-not-found=true','-o','json') or b'{}')
 if existing:
  ca=existing['data']['tls.crt']
  # Reusing a valid serving certificate makes repeated installs non-disruptive.
  subprocess.run(['openssl','x509','-checkend','3600','-noout'],input=base64.b64decode(ca),stdout=subprocess.DEVNULL,check=True)
 else:
  with tempfile.TemporaryDirectory(prefix='argo-admission-tls-') as tmp:
   cert=Path(tmp)/'tls.crt';key=Path(tmp)/'tls.key'
   subprocess.run(['openssl','req','-x509','-newkey','rsa:2048','-nodes','-days','14','-subj','/CN='+NAME+'.'+NS+'.svc','-addext','subjectAltName=DNS:'+NAME+'.'+NS+'.svc','-keyout',str(key),'-out',str(cert)],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL,check=True)
   ca=base64.b64encode(cert.read_bytes()).decode()
   apply({'apiVersion':'v1','kind':'Secret','metadata':{'name':NAME+'-tls','namespace':NS},'type':'kubernetes.io/tls','data':{'tls.crt':ca,'tls.key':base64.b64encode(key.read_bytes()).decode()}})
 apply({'apiVersion':'v1','kind':'ServiceAccount','metadata':{'name':NAME,'namespace':NS}})
 rules=[{'apiGroups':[G],'resources':['parametertemplates','workflowsubmissions','parameterrevisions','workflowparameterbindings','workflowactionbindings','actiondefinitions','actionresulttemplates','workflowforms','parameterretries','parameternodeinputs'],'verbs':['get']},
        {'apiGroups':['argoproj.io'],'resources':['workflowtemplates'],'verbs':['get']},
        {'apiGroups':[''],'resources':['configmaps','secrets'],'verbs':['get','list']}]
 apply({'apiVersion':'rbac.authorization.k8s.io/v1','kind':'Role','metadata':{'name':NAME,'namespace':NS},'rules':rules})
 subject={'kind':'ServiceAccount','name':NAME,'namespace':NS}
 apply({'apiVersion':'rbac.authorization.k8s.io/v1','kind':'RoleBinding','metadata':{'name':NAME,'namespace':NS},'roleRef':{'apiGroup':'rbac.authorization.k8s.io','kind':'Role','name':NAME},'subjects':[subject]})
 apply({'apiVersion':'rbac.authorization.k8s.io/v1','kind':'ClusterRole','metadata':{'name':NAME+'-lab'},'rules':[{'apiGroups':['authorization.k8s.io'],'resources':['subjectaccessreviews'],'verbs':['create']},{'apiGroups':['argoproj.io'],'resources':['clusterworkflowtemplates'],'verbs':['get']}]})
 apply({'apiVersion':'rbac.authorization.k8s.io/v1','kind':'ClusterRoleBinding','metadata':{'name':NAME+'-lab'},'roleRef':{'apiGroup':'rbac.authorization.k8s.io','kind':'ClusterRole','name':NAME+'-lab'},'subjects':[subject]})
 apply({'apiVersion':'apps/v1','kind':'Deployment','metadata':{'name':NAME,'namespace':NS},'spec':{'replicas':1,'selector':{'matchLabels':{'app':NAME}},'template':{'metadata':{'labels':{'app':NAME}},'spec':{'serviceAccountName':NAME,'containers':[{'name':'webhook','image':os.environ.get('ARGO_ADMISSION_IMAGE','argo-parameterization-admission:local-v37-v4.0.1'),'imagePullPolicy':'Never','args':['--namespace='+NS],'volumeMounts':[{'name':'tls','mountPath':'/tls','readOnly':True}],'readinessProbe':{'httpGet':{'path':'/healthz','scheme':'HTTPS','port':8443}},'resources':{'requests':{'cpu':'50m','memory':'64Mi'},'limits':{'memory':'256Mi'}},'securityContext':{'runAsNonRoot':True,'runAsUser':8737,'allowPrivilegeEscalation':False,'capabilities':{'drop':['ALL']}}}],'volumes':[{'name':'tls','secret':{'secretName':NAME+'-tls'}}]}}}})
 apply({'apiVersion':'v1','kind':'Service','metadata':{'name':NAME,'namespace':NS},'spec':{'selector':{'app':NAME},'ports':[{'port':443,'targetPort':8443}]}})
 kubectl('-n',NS,'rollout','status','deployment/'+NAME,'--timeout=60s')
 keys='["'+G+'/submission", "'+G+'/revision", "'+G+'/execution-digest"]'
 adopted=lambda obj:'('+obj+' != null && has('+obj+'.metadata.annotations) && '+keys+'.exists(k, k in '+obj+'.metadata.annotations))'
 apply({'apiVersion':'admissionregistration.k8s.io/v1','kind':'ValidatingWebhookConfiguration','metadata':{'name':NAME+'-lab'},'webhooks':[{
  'name':'workflow-contract.'+G,'admissionReviewVersions':['v1'],'sideEffects':'None','failurePolicy':'Fail','timeoutSeconds':10,
  'namespaceSelector':{'matchLabels':{'kubernetes.io/metadata.name':NS}},
  'matchConditions':[{'name':'explicit-parameter-contract','expression':adopted('object')+' || '+adopted('oldObject')}],
  'rules':[{'apiGroups':['argoproj.io'],'apiVersions':['v1alpha1'],'resources':['workflows','workflows/status'],'operations':['CREATE','UPDATE'],'scope':'Namespaced'}],
  'clientConfig':{'service':{'namespace':NS,'name':NAME,'path':'/validate','port':443},'caBundle':ca}
 },{
  'name':'snapshot-contract.'+G,'admissionReviewVersions':['v1'],'sideEffects':'None','failurePolicy':'Fail','timeoutSeconds':10,
  'namespaceSelector':{'matchLabels':{'kubernetes.io/metadata.name':NS}},
  'rules':[{'apiGroups':[G],'apiVersions':['v1alpha1'],'resources':['parameterrevisions','workflowsubmissions','actiondefinitions','actionresulttemplates','workflowactionbindings','parameterretries','parameternodeinputs','resourceoperations','parametertemplates','workflowforms','workflowparameterbindings'],'operations':['UPDATE'],'scope':'Namespaced'}],
  'clientConfig':{'service':{'namespace':NS,'name':NAME,'path':'/validate','port':443},'caBundle':ca}
 }]})
 print('Installed read-only admission in '+NS+'; unrelated Workflows are excluded.')

if __name__=='__main__':main()
