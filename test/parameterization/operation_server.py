"""Synthetic external service with independent receipts and request counters."""
import json,time,urllib.parse
from http.server import BaseHTTPRequestHandler,HTTPServer
jobs={};starts=0;cancels=0
class Handler(BaseHTTPRequestHandler):
 def log_message(self,*args):pass
 def key(self):return urllib.parse.unquote(self.path.removeprefix('/operations/'))
 def respond(self,status,value):
  b=json.dumps(value).encode();self.send_response(status);self.send_header('Content-Type','application/json');self.send_header('Content-Length',str(len(b)));self.end_headers();self.wfile.write(b)
 def observe(self,job):
  if job['Phase']=='Running' and time.time()>=job['finish']:
   job['Phase']='Succeeded';job['Outputs']={'reportURI':123 if job['input']['inputs'].get('mode')=='invalid-output' else 'demo://reports/'+job['ExternalID']}
  return {k:v for k,v in job.items() if k in ('Found','Phase','ExternalID','Outputs')}
 def do_GET(self):
  if self.path=='/stats':return self.respond(200,{'starts':starts,'cancels':cancels,'jobs':{k:self.observe(v) for k,v in jobs.items()}})
  j=jobs.get(self.key())
  if j and j['input']['inputs'].get('mode')=='unknown':return self.respond(404,{})
  self.respond(200,self.observe(j)) if j else self.respond(404,{})
 def do_POST(self):
  global starts
  starts+=1;body=json.loads(self.rfile.read(int(self.headers['Content-Length'])));key=self.key()
  if key in jobs:
   if jobs[key]['input']!=body:return self.respond(409,{})
   return self.respond(200,self.observe(jobs[key]))
  jobs[key]={'Found':True,'ExternalID':'job-'+str(len(jobs)+1),'Phase':'Running','input':body,'finish':time.time()+float(body['inputs']['delay'])}
  if body['inputs'].get('mode')in ('lost-response','unknown'):self.close_connection=True;return
  self.respond(200,self.observe(jobs[key]))
 def do_DELETE(self):
  global cancels
  cancels+=1;j=jobs.get(self.key())
  if not j:return self.respond(404,{})
  j['Phase']='Cancelled';self.respond(200,self.observe(j))
HTTPServer(('0.0.0.0',8080),Handler).serve_forever()
