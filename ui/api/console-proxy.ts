import * as http from 'http';
import * as WebSocket from 'ws';
import * as url from 'url';

import * as utils from './utils';

function safeCallback(callback) {
  const self = this;
  return function() {
    try {
      return callback.apply(self, arguments);
    } catch (e) {
      console.error(e);
    }
  };
}

export function create(server: http.Server, core) {
  const wss = new WebSocket.Server({ server });

    wss.on('connection', safeCallback((ws, req) => {
      const location = url.parse(req.url, true);
      const match = location.path.match(/\/api\/steps\/([^/]*)\/([^/]*)\/exec/);
      if (match) {
        const cmd = [location.query.cmd];
        const [_, ns, pod] = match;
        const apiUri = url.parse(core.url).host;
        let uri = `wss://${apiUri}/api/v1/namespaces/${ns}/pods/${pod}/exec?stdout=1&stdin=1&stderr=1&tty=1&container=main`;
        cmd.forEach(subCmd => uri += `&command=${encodeURIComponent(subCmd)}`);

        const kubeClient = new WebSocket(uri, 'base64.channel.k8s.io', {
          headers : {
            Authorization: `Bearer ${core.requestOptions.auth.bearer}`
          }
        });

        kubeClient.on('message', safeCallback((data) => {
          if (data[0].match(/^[0-3]$/)) {
            ws.send(utils.decodeBase64(data.slice(1)));
          }
        }));
        kubeClient.on('close', safeCallback(() => {
          ws.terminate();
        }));
        kubeClient.on('error', safeCallback(err => {
          ws.send(err.message);
          ws.terminate();
        }));

        ws.on('message', safeCallback(message => {
          kubeClient.send('0' + utils.encodeBase64(message));
        }));
      } else {
        ws.close(1002, 'Invalid URL');
      }
    }));

}
