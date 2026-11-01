"""Real files: raw artifact -> mapped action -> native upload -> downstream download."""
import copy,json,uuid
from e2e import api,wait_workflow,PREFIX,NS
from template_refs_e2e import resource,native,apply,kubectl
RUN='artifact-calls-'+uuid.uuid4().hex[:8]
G='parameters.workflows.example.io'
SERVER='''from http.server import BaseHTTPRequestHandler,HTTPServer
files={}
class Handler(BaseHTTPRequestHandler):
 def do_PUT(self):
  if self.headers.get('Transfer-Encoding')=='chunked':
   chunks=[]
   while True:
    size=int(self.rfile.readline().strip().split(b';')[0],16)
    if size==0:
     self.rfile.readline();break
    chunks.append(self.rfile.read(size));self.rfile.read(2)
   data=b''.join(chunks)
  else:data=self.rfile.read(int(self.headers['Content-Length']))
  files[self.path]=data;self.send_response(201);self.end_headers()
 def do_GET(self):
  value=files.get(self.path);self.send_response(200 if value is not None else 404);self.end_headers();self.wfile.write(value or b'')
 def log_message(self,*args):pass
HTTPServer(('0.0.0.0',8080),Handler).serve_forever()
'''
def fixture(name=RUN):
 schema={'type':'object','properties':{'message':{'type':'string'}},'required':['message'],'additionalProperties':False}
 empty={'type':'object','properties':{},'additionalProperties':False}
 def leaf(name,source,target):
  return {'name':name,'inputs':{'artifacts':[{'name':source,'path':'/tmp/input'}]},'container':{'image':'alpine:3.22','command':['sh','-ec'],'args':['cat /tmp/input >/tmp/output; printf ":mapped" >>/tmp/output']},'outputs':{'artifacts':[{'name':target,'path':'/tmp/output','archive':{'none':{}},'http':{'url':'http://'+RUN+':8080/{{workflow.name}}/{{pod.name}}'}}]}}
 base=leaf('base','source','report');impl=leaf('implementation','data','result')
 spec={'entrypoint':'main','serviceAccountName':'workflow-executor','arguments':{'parameters':[{'name':'message'}]},'templates':[{'name':'main','metadata':{'annotations':{G+'/bindable-calls':'["transform"]'}},'dag':{'tasks':[{'name':'approve','template':'approve'},{'name':'transform','dependencies':['approve'],'template':'base','arguments':{'artifacts':[{'name':'source','raw':{'data':'{{workflow.parameters.message}}'}}]}},{'name':'gate','dependencies':['transform'],'template':'gate'},{'name':'consume','dependencies':['gate'],'template':'consume','arguments':{'artifacts':[{'name':'data','from':'{{tasks.transform.outputs.artifacts.report}}'}]}}]}},{'name':'approve','suspend':{}},{'name':'gate','resource':{'action':'get','manifest':'apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: {{workflow.name}}-ready\n'}},base,{'name':'consume','inputs':{'artifacts':[{'name':'data','path':'/tmp/input'}]},'container':{'image':'alpine:3.22','command':['sh','-ec'],'args':['test "$(cat /tmp/input)" = "artifact-content:mapped"; sha256sum /tmp/input | cut -d" " -f1 >/tmp/hash']},'outputs':{'parameters':[{'name':'hash','valueFrom':{'path':'/tmp/hash'}}]}}]}
 return [resource('ParameterTemplate',name,{'dialect':'http://json-schema.org/draft-07/schema#','schema':schema}),native('WorkflowTemplate',name,spec),resource('ActionDefinition',name,{'inputSchema':empty,'outputSchema':empty,'template':impl}),resource('WorkflowActionBinding',name,{'workflowTemplate':name,'calls':[{'callPath':['main','transform'],'actionDefinition':name,'inputArtifactMappings':[{'from':'source','to':'data'}],'outputArtifactMappings':[{'from':'result','to':'report'}]}]}),resource('WorkflowParameterBinding',name,{'workflowTemplate':name,'parameterTemplate':name,'actionBinding':name,'mappings':[{'path':'/message','parameter':'message','encoding':'string'}]})]
def main():
 apply({'apiVersion':'apps/v1','kind':'Deployment','metadata':{'name':RUN,'namespace':NS},'spec':{'selector':{'matchLabels':{'app':RUN}},'template':{'metadata':{'labels':{'app':RUN}},'spec':{'automountServiceAccountToken':False,'containers':[{'name':'server','image':'python:3.12-alpine','command':['python','-u','-c',SERVER]}]}}}})
 apply({'apiVersion':'v1','kind':'Service','metadata':{'name':RUN,'namespace':NS},'spec':{'selector':{'app':RUN},'ports':[{'port':8080,'targetPort':8080}]}})
 try:
  kubectl('-n',NS,'rollout','status','deployment/'+RUN,'--timeout=60s')
  objects=fixture()
  for obj in objects:apply(obj)
  saved=api('PUT',PREFIX+'/sets/'+RUN,{'template':RUN,'values':{'message':'artifact-content'},'resourceVersion':''})
  preview=api('POST',PREFIX+'/preview',{'binding':RUN,'revision':saved['spec']['revision']});assert not preview['errors'],preview
  source=json.loads(kubectl('-n',NS,'get','workflowtemplate',RUN,'-o','json'));assert source['spec']==objects[1]['spec']
  wf=api('POST',PREFIX+'/submit',{'binding':RUN,'revision':saved['spec']['revision'],'executionDigest':preview['executionDigest'],'idempotencyKey':RUN});name=wf['metadata']['name']
  wait_workflow(name,lambda w:any(n.get('type')=='Suspend' and n.get('phase')=='Running' for n in w.get('status',{}).get('nodes',{}).values()))
  kubectl('-n',NS,'delete','actiondefinition',RUN)
  api('PUT','/api/v1/workflows/'+NS+'/'+name+'/resume',{})
  failed=wait_workflow(name,lambda w:w.get('status',{}).get('phase') in ['Failed','Error'])
  producer=next((i,n) for i,n in failed['status']['nodes'].items() if n.get('displayName')=='invoke')
  assert producer[1]['phase']=='Succeeded',producer
  apply({'apiVersion':'v1','kind':'ConfigMap','metadata':{'name':name+'-ready','namespace':NS},'data':{'ready':'true'}})
  api('POST',PREFIX+'/retry',{'workflow':name,'uid':wf['metadata']['uid'],'resourceVersion':failed['metadata']['resourceVersion'],'idempotencyKey':RUN+'-recovery'})
  done=wait_workflow(name,lambda w:w.get('status',{}).get('phase')=='Succeeded')
  assert done['metadata']['uid']==wf['metadata']['uid']
  assert all(done['status']['nodes'][producer[0]].get(k)==producer[1].get(k) for k in ['outputs','finishedAt','startedAt'])
  import hashlib
  expected=hashlib.sha256(b'artifact-content:mapped').hexdigest()
  consumer=next(n for n in done['status']['nodes'].values() if n.get('displayName')=='consume')
  assert consumer['outputs']['parameters'][0]['value']==expected
  wrapper=next(n for n in done['status']['nodes'].values() if n.get('displayName')=='transform')
  assert wrapper['outputs']['artifacts'][0]['name']=='report'
  assert 'artifact-content:mapped' not in json.dumps(done['status']),'artifact contents must not become parameters'
  print('PASS',json.dumps({'workflow':name,'uid':done['metadata']['uid'],'fileDigestVerified':True,'sourceDeletedAfterSubmit':True}),flush=True)
 finally:kubectl('-n',NS,'delete','deployment,service',RUN,'--ignore-not-found=true')
if __name__=='__main__':main()
