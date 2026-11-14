"""Generate local-only, cloud-free demos of dynamically bound native Argo actions."""
from pathlib import Path
import yaml,json
ROOT=Path(__file__).resolve().parent
NS='argo-parameterization-lab';API='parameters.workflows.example.io/v1alpha1'
def entity(kind,name,spec):return {'apiVersion':API,'kind':kind,'metadata':{'name':name,'namespace':NS},'spec':spec}
def string(**more):return {'type':'string',**more}
name=string(pattern='^[a-z][a-z0-9-]{0,20}$')
subnet={'type':'object','required':['id','cidr'],'properties':{'id':name,'cidr':string()},'additionalProperties':False}
zone={'type':'object','required':['name','subnets'],'properties':{'name':name,'subnets':{'type':'array','minItems':1,'maxItems':4,'items':subnet}},'additionalProperties':False}
schema={'type':'object','required':['region','zones','roles'],'properties':{'region':name,'environment':string(enum=['test','production'],default='test'),'zones':{'type':'array','minItems':1,'maxItems':6,'items':zone},'roles':{'type':'array','minItems':1,'maxItems':4,'uniqueItems':True,'items':name},'fail':{'type':'boolean','default':False}},'additionalProperties':False}
model={'dialect':'http://json-schema.org/draft-07/schema#','schema':schema,'identityRules':[{'path':'/zones','keyPath':'/name'}]}
items=[entity('ParameterTemplate','topology-v1',model),entity('WorkflowForm','topology-form-v1',{'options':{'/region':{'kind':'configmap','name':'demo-region-catalog','key':'regions'}},'labels':{'/zones':'Availability zones','/roles':'Node roles','/fail':'Simulate final check failure'}})]
params=lambda **values:{'parameters':[{'name':k,'value':v} for k,v in values.items()]}
wf={'apiVersion':'argoproj.io/v1alpha1','kind':'WorkflowTemplate','metadata':{'name':'dynamic-delivery-demo','namespace':NS,'labels':{'workflows.argoproj.io/controller-instanceid':'parameterization-lab'}},'spec':{'entrypoint':'main','serviceAccountName':'workflow-executor','volumes':[{'name':'recovery','configMap':{'name':'{{workflow.name}}-recovery','optional':True}}],'parallelism':4,'arguments':{'parameters':[{'name':n} for n in ['region','environment','zones','roles','fail']]},'templates':[
 {'name':'main','dag':{'tasks':[{'name':'approve','template':'approve'},{'name':'zones','dependencies':['approve'],'template':'zone','withParam':'{{workflow.parameters.zones}}','arguments':params(zone='{{item.name}}',subnets='{{item.subnets}}')},{'name':'verify','dependencies':['zones'],'template':'verify'}]}},
 {'name':'approve','suspend':{}},
 {'name':'zone','inputs':{'parameters':[{'name':'zone'},{'name':'subnets'}]},'dag':{'tasks':[{'name':'subnets','template':'subnet','withParam':'{{inputs.parameters.subnets}}','arguments':params(zone='{{inputs.parameters.zone}}',subnet='{{item.id}}',cidr='{{item.cidr}}') }]}},
 {'name':'subnet','inputs':{'parameters':[{'name':n} for n in ['zone','subnet','cidr']]},'dag':{'tasks':[{'name':'roles','template':'provision','withParam':'{{workflow.parameters.roles}}','arguments':params(name='{{inputs.parameters.zone}}-{{inputs.parameters.subnet}}-{{item}}',zone='{{inputs.parameters.zone}}',cidr='{{inputs.parameters.cidr}}',role='{{item}}')}]}},
 {'name':'provision','suspend':{}},
 {'name':'verify','container':{'image':'alpine:3.22','command':['sh','-ec'],'args':['if [ "$FAIL" = "true" ] && [ "$(cat /demo/allow 2>/dev/null || true)" != true ]; then exit 7; fi; echo topology-ready'],'volumeMounts':[{'name':'recovery','mountPath':'/demo','readOnly':True}],'env':[{'name':'FAIL','value':'{{workflow.parameters.fail}}'}]}}
]}}
items.append(wf)
inputs={n:string() for n in ['name','zone','cidr','role']}
action={'inputSchema':{'type':'object','properties':inputs,'required':list(inputs)},'outputSchema':{'type':'object','properties':{'resourceName':string()}},'template':{'name':'apply-record','inputs':{'parameters':[{'name':n} for n in inputs]},'outputs':{'parameters':[{'name':'resourceName','valueFrom':{'jsonPath':'{.metadata.name}'}}]},'resource':{'action':'apply','manifest':'''apiVersion: v1
kind: ConfigMap
metadata:
  name: '{{workflow.name}}-{{inputs.parameters.name}}'
  labels:
    demo.workflow: '{{workflow.name}}'
data:
  zone: '{{inputs.parameters.zone}}'
  cidr: '{{inputs.parameters.cidr}}'
  role: '{{inputs.parameters.role}}'
  region: '{{workflow.parameters.region}}'
  environment: '{{workflow.parameters.environment}}'
'''}}}
items.extend([entity('ActionDefinition','apply-record-v1',action),entity('WorkflowActionBinding','delivery-actions-v1',{'workflowTemplate':'dynamic-delivery-demo','calls':[{'template':'provision','actionDefinition':'apply-record-v1'}]}),entity('WorkflowParameterBinding','dynamic-delivery',{'workflowTemplate':'dynamic-delivery-demo','parameterTemplate':'topology-v1','form':'topology-form-v1','actionBinding':'delivery-actions-v1','mappings':[{'path':'/'+n,'parameter':n,'encoding':encoding} for n,encoding in [('region','string'),('environment','string'),('zones','json'),('roles','json'),('fail','boolean')]]})])
items.append({'apiVersion':'v1','kind':'ConfigMap','metadata':{'name':'demo-region-catalog','namespace':NS},'data':{'regions':json.dumps([{'value':'demo-north','label':'Demo North'},{'value':'demo-south','label':'Demo South'}])}})
# A second, unrelated domain uses the same revision/form/binding APIs.
items.append(entity('ParameterTemplate','training-v1',{'dialect':model['dialect'],'schema':{'type':'object','required':['dataset','epochs'],'properties':{'dataset':string(),'epochs':{'type':'integer','minimum':1,'maximum':100},'shuffle':{'type':'boolean','default':False}},'additionalProperties':False}}))
items.append({'apiVersion':'argoproj.io/v1alpha1','kind':'WorkflowTemplate','metadata':{'name':'training-demo','namespace':NS},'spec':{'entrypoint':'train','serviceAccountName':'workflow-executor','arguments':{'parameters':[{'name':n} for n in ['dataset','epochs','shuffle']]},'templates':[{'name':'train','container':{'image':'alpine:3.22','command':['sh','-ec'],'args':['printf "Dataset=%s epochs=%s shuffle=%s\\n" "$DATASET" "$EPOCHS" "$SHUFFLE"'],'env':[{'name':n.upper(),'value':'{{workflow.parameters.'+n+'}}'} for n in ['dataset','epochs','shuffle']]}}]}})
items.append(entity('WorkflowParameterBinding','training',{'workflowTemplate':'training-demo','parameterTemplate':'training-v1','mappings':[{'path':'/'+n,'parameter':n,'encoding':e} for n,e in [('dataset','string'),('epochs','decimal'),('shuffle','boolean')]]}))
# Secret demonstration uses synthetic input only. Plaintext is consumed by a
# native SecretKeyRef, never by workflow arguments or resource manifests.
secretref={'type':'object','required':['secretRef'],'properties':{'secretRef':{'type':'object','required':['name','key'],'properties':{'name':string(),'key':string()},'additionalProperties':False}},'additionalProperties':False}
items.append(entity('ParameterTemplate','secret-v1',{'dialect':model['dialect'],'schema':{'type':'object','required':['credential'],'properties':{'credential':secretref},'additionalProperties':False},'sensitivePaths':['/credential'],'secretSchemas':{'/credential':string(minLength=16)}}))
items.append({'apiVersion':'argoproj.io/v1alpha1','kind':'WorkflowTemplate','metadata':{'name':'secret-demo','namespace':NS},'spec':{'entrypoint':'consume','serviceAccountName':'workflow-executor','templates':[{'name':'consume','container':{'image':'alpine:3.22','command':['sh','-ec'],'args':['test "${#TOKEN}" -ge 16; case "$TOKEN" in synthetic-*) echo credential-consumed ;; *) exit 2 ;; esac']}}]}})
items.append(entity('WorkflowParameterBinding','secret-demo',{'workflowTemplate':'secret-demo','parameterTemplate':'secret-v1','mappings':[],'secrets':[{'path':'/credential','template':'consume','environment':'TOKEN'}]}))
# Typed input is bound to a native suspend, and becomes its supplied outputs.
items.append(entity('ParameterTemplate','empty-v1',{'dialect':model['dialect'],'schema':{'type':'object','additionalProperties':False}}))
items.append(entity('ParameterTemplate','decision-v1',{'dialect':model['dialect'],'schema':{'type':'object','required':['replicas','release'],'properties':{'replicas':{'type':'integer','minimum':1,'maximum':8},'release':string(enum=['stable','canary'])},'additionalProperties':False}}))
items.append({'apiVersion':'argoproj.io/v1alpha1','kind':'WorkflowTemplate','metadata':{'name':'intermediate-demo','namespace':NS},'spec':{'entrypoint':'main','serviceAccountName':'workflow-executor','templates':[{'name':'main','steps':[[{'name':'decision','template':'decision'}],[{'name':'consume','template':'consume','arguments':params(count='{{steps.decision.outputs.parameters.count}}',release='{{steps.decision.outputs.parameters.release}}')}]]},{'name':'decision','suspend':{},'outputs':{'parameters':[{'name':n,'valueFrom':{'supplied':{}}} for n in ['count','release']]}},{'name':'consume','inputs':{'parameters':[{'name':'count'},{'name':'release'}]},'container':{'image':'alpine:3.22','command':['sh','-ec'],'args':['printf "replicas=%s release=%s\\n" "$COUNT" "$RELEASE"'],'env':[{'name':'COUNT','value':'{{inputs.parameters.count}}'},{'name':'RELEASE','value':'{{inputs.parameters.release}}'}]}}]}})
items.append(entity('WorkflowParameterBinding','intermediate-demo',{'workflowTemplate':'intermediate-demo','parameterTemplate':'empty-v1','mappings':[],'inputs':[{'template':'decision','parameterTemplate':'decision-v1','mappings':[{'path':'/replicas','parameter':'count','encoding':'decimal'},{'path':'/release','parameter':'release','encoding':'string'}]}]}))
for item in items:
 if item['kind']=='WorkflowTemplate':
  item['metadata'].setdefault('labels',{})['workflows.argoproj.io/controller-instanceid']='parameterization-lab'
(ROOT/'demo.yaml').write_text(yaml.safe_dump_all(items,sort_keys=False))
(ROOT/'topology-values.yaml').write_text(yaml.safe_dump({'region':'demo-north','environment':'test','zones':[{'name':'zone-a','subnets':[{'id':'one','cidr':'10.10.1.0/24'},{'id':'two','cidr':'10.10.2.0/24'}]},{'name':'zone-b','subnets':[{'id':'one','cidr':'10.20.1.0/24'}]}],'roles':['admin','dns'],'fail':False},sort_keys=False))
