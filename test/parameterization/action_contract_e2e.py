"""Real API -> frozen action binding -> native Pod/HTTP -> contract -> downstream."""
import json,time,uuid
from e2e import api,wait_workflow,PREFIX,NS
from template_refs_e2e import resource,native,apply,kubectl
RUN='action-'+uuid.uuid4().hex[:8]
def obj_schema(props,required=None):return {'type':'object','properties':props,'required':required or list(props),'additionalProperties':False}
def execute(binding,values,key):
 s=api('PUT',PREFIX+'/sets/'+key,{'template':RUN,'values':values,'resourceVersion':''})
 p=api('POST',PREFIX+'/preview',{'binding':binding,'revision':s['spec']['revision']});assert not p['errors']
 w=api('POST',PREFIX+'/submit',{'binding':binding,'revision':s['spec']['revision'],'executionDigest':p['executionDigest'],'idempotencyKey':key})
 return wait_workflow(w['metadata']['name'],lambda v:v.get('status',{}).get('phase') in ['Succeeded','Failed','Error'])
def main():
 assert kubectl('auth','can-i','patch','workflowtasksets/status','-n',NS,'--as=system:serviceaccount:'+NS+':argo').decode().strip()=='yes','controller needs native taskset status reconciliation permission'
 # Namespace-local synthetic HTTP service; no external business dependency.
 code="from http.server import BaseHTTPRequestHandler,HTTPServer\nclass H(BaseHTTPRequestHandler):\n def do_GET(self):\n  b=b'{\"ok\":true}' if self.path=='/valid' else b'{\"ok\":\"invalid\"}';self.send_response(200);self.end_headers();self.wfile.write(b)\nHTTPServer(('0.0.0.0',8080),H).serve_forever()"
 apply({'apiVersion':'apps/v1','kind':'Deployment','metadata':{'name':RUN,'namespace':NS},'spec':{'replicas':1,'selector':{'matchLabels':{'app':RUN}},'template':{'metadata':{'labels':{'app':RUN}},'spec':{'automountServiceAccountToken':False,'containers':[{'name':'http','image':'python:3.12-alpine','command':['python3','-c',code],'readinessProbe':{'httpGet':{'path':'/valid','port':8080},'periodSeconds':2}}]}}}})
 apply({'apiVersion':'v1','kind':'Service','metadata':{'name':RUN,'namespace':NS},'spec':{'selector':{'app':RUN},'ports':[{'port':8080,'targetPort':8080}]}})
 kubectl('-n',NS,'rollout','status','deployment/'+RUN,'--timeout=90s')
 apply(resource('ParameterTemplate',RUN,{'dialect':'http://json-schema.org/draft-07/schema#','schema':obj_schema({'count':{'type':'string'},'out':{'type':'string'},'url':{'type':'string'}})}))
 container={'name':'call','inputs':{'parameters':[{'name':'count'},{'name':'out'}]},'container':{'image':'alpine:3.22','command':['sh','-ec'],'args':['printf "%s" "$OUT" > /tmp/count'],'env':[{'name':'OUT','value':'{{inputs.parameters.out}}'}]},'outputs':{'parameters':[{'name':'count','valueFrom':{'path':'/tmp/count'}}]}}
 actions={'container':{'inputSchema':obj_schema({'count':{'type':'integer','minimum':1},'out':{'type':'string'}}),'outputSchema':obj_schema({'count':{'type':'integer','minimum':1}}),'template':container},'http':{'inputSchema':obj_schema({'url':{'type':'string'}}),'outputSchema':obj_schema({'result':obj_schema({'ok':{'type':'boolean','const':True}})}),'template':{'name':'call','inputs':{'parameters':[{'name':'url'}]},'http':{'url':'{{inputs.parameters.url}}','timeoutSeconds':10}}}}
 actions['script']={'inputSchema':obj_schema({'out':{'type':'string'}}),'outputSchema':obj_schema({'result':{'type':'integer','minimum':1}}),'template':{'name':'call','inputs':{'parameters':[{'name':'out'}]},'script':{'image':'alpine:3.22','command':['sh'],'source':'printf \"%s\" \"$OUT\"','env':[{'name':'OUT','value':'{{inputs.parameters.out}}'}]}}}
 for kind,definition in actions.items():
  name=RUN+'-'+kind
  if kind in ('http','script'):
   implementation=definition.pop('template')
   apply(native('WorkflowTemplate',name+'-library',{'templates':[implementation]}))
   definition['implementationRef']={'name':name+'-library','template':implementation['name']}
  apply(resource('ActionDefinition',name,definition))
  params=['count','out'] if kind=='container' else (['out'] if kind=='script' else ['url'])
  output='{{tasks.call.outputs.parameters.count}}' if kind=='container' else '{{tasks.call.outputs.result}}'
  spec={'entrypoint':'main','serviceAccountName':'workflow-executor','arguments':{'parameters':[{'name':x} for x in ['count','out','url']]},'templates':[{'name':'main','dag':{'tasks':[{'name':'call','template':'call','arguments':{'parameters':[{'name':x,'value':'{{workflow.parameters.'+x+'}}'} for x in params]}},{'name':'after','dependencies':['call'],'template':'after','arguments':{'parameters':[{'name':'result','value':output}]}}]}},{'name':'call','suspend':{}},{'name':'after','inputs':{'parameters':[{'name':'result'}]},'container':{'image':'alpine:3.22','command':['echo','{{inputs.parameters.result}}']}}]}
  apply(native('WorkflowTemplate',name,spec));apply(resource('WorkflowActionBinding',name,{'workflowTemplate':name,'calls':[{'template':'call','actionDefinition':name}]}));apply(resource('WorkflowParameterBinding',name,{'workflowTemplate':name,'parameterTemplate':RUN,'actionBinding':name,'mappings':[{'path':'/'+x,'parameter':x,'encoding':'string'} for x in ['count','out','url']]}))
 base={'count':'1','out':'1','url':'http://'+RUN+':8080/valid'};evidence=[]
 for label,kind,changes,success in [('input','container',{'count':'0'},False),('output','container',{'out':'0'},False),('valid','container',{},True),('http-invalid','http',{'url':'http://'+RUN+':8080/invalid'},False),('http-valid','http',{},True),('script-invalid','script',{'out':'0'},False),('script-valid','script',{},True)]:
  w=execute(RUN+'-'+kind,{**base,**changes},RUN+'-'+label)
  assert (w['status']['phase']=='Succeeded')==success,(label,w['status'])
  nodes=w['status']['nodes'];after=[n for n in nodes.values() if n['name'].endswith('.after') and n['type']=='Pod'];assert bool(after)==success,(label,after)
  if not success:assert any('action contract' in n.get('message','') or 'action JSON' in n.get('message','') for n in nodes.values()),nodes
  if label=='input':assert not json.loads(kubectl('-n',NS,'get','pods','-l','workflows.argoproj.io/workflow='+w['metadata']['name'],'-o','json'))['items']
  evidence.append({'case':label,'workflow':w['metadata']['name'],'phase':w['status']['phase']});print('PASS',label,flush=True)
 print(json.dumps({'run':RUN,'cases':evidence}),flush=True)
 kubectl('-n',NS,'delete','deployment,service',RUN,'--ignore-not-found=true')
if __name__=='__main__':main()
