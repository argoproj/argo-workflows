import json
from http.server import BaseHTTPRequestHandler, HTTPServer


class Plugin(BaseHTTPRequestHandler):

    def do_POST(self):
        if self.path == "/WorkflowLifecycleHook.WorkflowPreOperate":
            self.send_response(200)
            self.end_headers()
            self.wfile.write(json.dumps({}).encode("UTF-8"))
        else:
            self.send_response(404)
            self.end_headers()


if __name__ == '__main__':
    httpd = HTTPServer(('', 7584), Plugin)
    httpd.serve_forever()
