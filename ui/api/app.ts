import * as express from 'express';
import * as Api from 'kubernetes-client';
import * as bodyParser from 'body-parser';
import * as fallback from 'express-history-api-fallback';
import * as models from '../src/app/models';
import { Observable } from 'rxjs/Observable';
import { Observer } from 'rxjs/Observer';
import { async } from '@angular/core/testing';

function reactifyStream(stream, converter = item => item) {
  return new Observable((observer: Observer<any>) => {
      stream.on('data', (d) => observer.next(converter(d)));
      stream.on('end', () => observer.complete());
      stream.on('error', e => observer.error(e));
  });
}

function reactifyStringStream(stream) {
  return reactifyStream(stream, item => item.toString());
}

function streamServerEvents<T>(req: express.Request, res: express.Response, source: Observable<T>, formatter: (input: T) => string) {
  res.setHeader('Content-Type', 'text/event-stream');
  res.setHeader('Transfer-Encoding', 'chunked');
  res.setHeader('X-Content-Type-Options', 'nosniff');

  const subscription = source.subscribe(
      (info) => res.write(`data:${formatter(info)}\n\n`),
      (err) => res.end(),
      () => res.end());
  req.on('close', () => subscription.unsubscribe());
}

function serve<T>(res: express.Response, action: () => Promise<T>) {
  action().then(val => res.send(val)).catch(err => res.status(500).send(err));
}

export function create(
    uiDist: string,
    inCluster: boolean,
    namespace: string,
    version = 'v1',
    group = 'argoproj.io') {
  const config = Object.assign(
    {}, inCluster ? Api.config.getInCluster() : Api.config.fromKubeconfig(), {namespace, version, group, promises: true });
  const core = new Api.Core(config);
  const crd = new Api.CustomResourceDefinitions(config);
  crd.addResource('workflows');
  const app = express();
  app.use(bodyParser.json({type: () => true}));

  app.get('/api/workflows', (req, res) => serve(res, async () => {
    const workflowList = <models.WorkflowList> await crd.ns['workflows'].get();
    workflowList.items.sort((first, second) => first.metadata.creationTimestamp - second.metadata.creationTimestamp);
    return workflowList;
  }));
  app.get('/api/workflows/:name', async (req, res) => serve(res, () => crd.ns['workflows'].get(req.params.name)));
  app.get('/api/steps/:name/logs', (req, res) => {
    const logsSource = reactifyStringStream(core.ns.po(req.params.name).log.getStream({ qs: { container: 'main', follow: true } }));
    streamServerEvents(req, res, logsSource, item => item.toString());
  });
  app.use(express.static(uiDist));
  app.use(fallback('index.html', { root: uiDist }));
  return app;
}
