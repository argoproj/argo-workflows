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
        if self.path == '/api/v1/template.execute':
            args = self.args()
            if 'slack' in args['template'].get('plugin', {}):
                x = urlopen(
                    Request(os.getenv('URL'),
                            data=json.dumps({'text': args['template']['plugin']['slack']['text']}).encode()))
                if x.status != 200:
                    raise Exception("not 200")
                self.reply({'node': {'phase': 'Succeeded', 'message': 'Slack message sent'}})
            else:
                self.reply({})
        else:
            self.unsupported()


if __name__ == '__main__':
    httpd = HTTPServer(('', 7522), Plugin)
    httpd.serve_forever()
