"""Example-only result provider/controller. Argo never imports this adapter.

The controller owns external reads and status projection. Service URL/method
allowlist come from operator configuration; Workflow inputs only select a target.
"""
import json, os, ssl, threading, time, urllib.request, urllib.parse
from datetime import datetime, timezone
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

state = {'generation': 1, 'reads': 0}
lock = threading.Lock()

class Provider(BaseHTTPRequestHandler):
    def log_message(self, *_):
        pass

    def send(self, status, body):
        data = json.dumps(body).encode()
        self.send_response(status)
        self.send_header('Content-Type', 'application/json')
        self.end_headers()
        self.wfile.write(data)

    def do_GET(self):
        if self.path == '/healthz':
            return self.send(200, {'ok': True})
        if self.path == '/metrics':
            with lock:
                return self.send(200, dict(state))
        if not self.path.startswith('/reports/'):
            return self.send(404, {'error': 'unknown operation'})
        target = urllib.parse.unquote(self.path[len('/reports/'):])
        with lock:
            state['reads'] += 1
            generation = state['generation']
        self.send(200, {'target': target, 'generation': generation, 'status': 'Healthy',
                        'children': [{'target': target + '-child', 'status': 'Complete', 'children': []}]})

    def do_POST(self):
        # Synthetic fixture controls. Never exposes a real external provider.
        if self.path != '/advance':
            return self.send(404, {'error': 'unknown fixture control'})
        with lock:
            state['generation'] += 1
            return self.send(200, dict(state))


def controller():
    namespace = os.environ['NAMESPACE']
    selector = os.environ['QUERY_SELECTOR']
    provider = os.environ['PROVIDER_URL']
    assert provider.startswith('http://')
    base = 'https://kubernetes.default.svc'
    token_path = '/var/run/secrets/kubernetes.io/serviceaccount/token'
    tls = ssl.create_default_context(cafile='/var/run/secrets/kubernetes.io/serviceaccount/ca.crt')
    path = '/apis/results.example.io/v1alpha1/namespaces/' + namespace + '/resultqueries'

    def kube(method, url, body=None):
        with open(token_path) as source:
            token = source.read().strip()
        headers = {'Authorization': 'Bearer ' + token, 'Content-Type': 'application/merge-patch+json'}
        req = urllib.request.Request(base + url, data=json.dumps(body).encode() if body is not None else None,
                                     headers=headers, method=method)
        with urllib.request.urlopen(req, context=tls, timeout=5) as response:
            return json.load(response)

    while True:
        try:
            items = kube('GET', path + '?labelSelector=' + urllib.parse.quote(selector))['items']
            for obj in items:
                spec = obj['spec']
                if obj['metadata'].get('deletionTimestamp') or spec.get('operation') != 'report.read':
                    continue
                target = spec['target']
                url = provider + '/reports/' + urllib.parse.quote(target, safe='')
                with urllib.request.urlopen(url, timeout=5) as response:
                    document = json.load(response)
                # resourceVersion protects against replaced or edited query requests.
                kube('PATCH', path + '/' + obj['metadata']['name'] + '/status', {
                    'metadata': {'resourceVersion': obj['metadata']['resourceVersion']},
                    'status': {'ready': True, 'document': document,
                               'observedAt': datetime.now(timezone.utc).isoformat().replace('+00:00', 'Z')}})
        except Exception as error:
            # Do not log response bodies or authorization credentials.
            print(type(error).__name__, flush=True)
        time.sleep(2)

if __name__ == '__main__':
    if os.environ.get('MODE') == 'controller':
        controller()
    else:
        ThreadingHTTPServer(('0.0.0.0', 8080), Provider).serve_forever()
