import json
import os
from http.server import BaseHTTPRequestHandler, HTTPServer
from urllib.request import urlopen, Request


class Plugin(BaseHTTPRequestHandler):

    def args(self):
        return json.loads(self.rfile.read(int(self.headers.get('Content-Length'))))

    def reply(self, reply):
        self.send_response(200)
        self.end_headers()
        self.wfile.write(json.dumps(reply).encode("UTF-8"))

    def unsupported(self):
        self.send_response(404)
        self.end_headers()

    def do_POST(self):
        if self.path == '/api/v1/workflow.postOperate':
            args = self.args()
            before = args['old']['metadata'].get('labels', {}).get('workflows.argoproj.io/phase', '')
            after = args['new']['metadata'].get('labels', {}).get('workflows.argoproj.io/phase', '')
            if before == 'Running' and after == 'Failed':
                workflow_name = args['new']['metadata']['name']
                print("creating alert for " + workflow_name)
                x = urlopen(
                    Request(os.getenv('URL', 'https://events.pagerduty.com/generic/2010-04-15/create_event.json'),
                            data=json.dumps({
                                "service_key": os.getenv("SERVICE_KEY"),
                                "event_type": "trigger",
                                "incident_key": workflow_name,
                                "description": f"Argo Workflow ${workflow_name} Failed",
                                "client": "Argo Workflows",
                            }).encode()))
                if x.status != 200:
                    raise Exception("not 2002")
            self.reply({})
        else:
            self.unsupported()


if __name__ == '__main__':
    httpd = HTTPServer(('', 7243), Plugin)
    httpd.serve_forever()
