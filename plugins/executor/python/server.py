import json
from http.server import BaseHTTPRequestHandler, HTTPServer


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

            template = args['template']
            plugin = template.get('plugin', {})

            if 'python' in plugin:
                spec = plugin['python']

                # convert parameters into easy to use dict
                # artifacts are not supported
                parameters = {}
                for parameter in template.get('inputs', {}).get('parameters', []):
                    parameters[parameter['name']] = parameter['value']

                try:
                    code = compile(spec['expression'], "<string>", "eval")

                    # only allow certain names (primitive sand-boxing)
                    allowed_names = {
                        # allow common type conversions
                        'bool': bool,
                        'float': float,
                        'int': int,
                        'str': str,
                        # TODO - do  we want iterable built-ins, e.g. len, min, max
                        # allow input parameters
                        'parameters': parameters
                    }
                    if code.co_names:
                        for name in code.co_names:
                            if name not in allowed_names:
                                raise NameError(f"Use of name '{name}' not allowed")

                    result = eval(code, {"__builtins__": {}}, allowed_names)

                    # convert parameters back from easy to use dict
                    # artifacts are not supported
                    parameters = []
                    for key, value in result.items():
                        parameters.append({'name': key, 'value': value})

                    self.reply({'node': {'phase': 'Succeeded', 'outputs': {'parameters': parameters}}})

                except Exception as ex:
                    self.reply({'node': {'phase': 'Failed', 'message': repr(ex)}})
            else:
                self.reply({})
        else:
            self.unsupported()


if __name__ == '__main__':
    httpd = HTTPServer(('', 7984), Plugin)
    httpd.serve_forever()
