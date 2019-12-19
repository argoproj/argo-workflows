import * as aws from 'aws-sdk';
import * as bodyParser from 'body-parser';
import * as express from 'express';
import * as expressWinston from 'express-winston';
import * as fs from 'fs';
import * as http from 'http';
import * as JSONStream from 'json-stream';
import * as Api from 'kubernetes-client';
import * as path from 'path';
import {Observable, Observer} from 'rxjs';
import * as nodeStream from 'stream';
import * as promisify from 'util.promisify';
import * as winston from 'winston';

import * as zlib from 'zlib';
import * as models from '../models/workflows';
import * as consoleProxy from './console-proxy';

import {decodeBase64, reactifyStringStream, streamServerEvents} from './utils';

const winstonTransport = new winston.transports.Console({
    format: winston.format.combine(winston.format.timestamp(), winston.format.simple())
});

const logger = winston.createLogger({
    transports: [winstonTransport]
});

function serve<T>(res: express.Response, action: () => Promise<T>) {
    action()
        .then(val => res.send(val))
        .catch(err => {
            if (err instanceof Error) {
                err = {...err, message: err.message};
            }
            res.status(500).send(err);
            logger.error(err);
        });
}

function fileToString(filePath: string): Promise<string> {
    return new Promise<string>((resolve, reject) => {
        fs.readFile(filePath, 'utf-8', (err, content) => {
            if (err) {
                reject(err);
            } else {
                resolve(content);
            }
        });
    });
}

export function create(
    uiDist: string,
    uiBaseHref: string,
    inCluster: boolean,
    namespace: string,
    forceNamespaceIsolation: boolean,
    instanceId: string,
    version,
    group = 'argoproj.io'
) {
    const config = Object.assign({}, inCluster ? Api.config.getInCluster() : Api.config.fromKubeconfig(), {namespace, promises: true});
    const core = new Api.Core(config);
    const crd = new Api.CustomResourceDefinitions(Object.assign(config, {version, group}));
    crd.addResource('workflows');
    const app = express();
    app.use(bodyParser.json({type: () => true}));

    app.use(
        expressWinston.logger({
            transports: [winstonTransport],
            meta: false,
            msg: '{{res.statusCode}} {{req.method}} {{res.responseTime}}ms {{req.url}}'
        })
    );

    function getWorkflowLabelSelector(req) {
        const labelSelector: string[] = [];
        if (instanceId) {
            labelSelector.push(`workflows.argoproj.io/controller-instanceid = ${instanceId}`);
        }
        if (req.query.phase) {
            const phases = req.query.phase instanceof Array ? req.query.phase : [req.query.phase];
            if (phases.length > 0) {
                labelSelector.push(`workflows.argoproj.io/phase in (${phases.join(',')})`);
            }
        }
        return labelSelector;
    }

    app.get('/api/workflows', (req, res) =>
        serve(res, async () => {
            const labelSelector = getWorkflowLabelSelector(req);

            const workflowList = (await (forceNamespaceIsolation ? crd.ns(namespace) : crd).workflows.get({
                qs: {labelSelector: labelSelector.join(',')}
            })) as models.WorkflowList;

            workflowList.items.sort(models.compareWorkflows);
            workflowList.items = await Promise.all(workflowList.items.map(deCompressNodes));
            return workflowList;
        })
    );

    app.get('/api/workflows/:namespace/:name', async (req, res) =>
        serve(res, () => (forceNamespaceIsolation ? crd.ns(namespace) : crd.ns(req.params.namespace)).workflows.get(req.params.name).then(deCompressNodes))
    );

    app.get('/api/workflows/live', async (req, res) => {
        const ns = getNamespace(req);
        let updatesSource = new Observable((observer: Observer<any>) => {
            const labelSelector = getWorkflowLabelSelector(req);
            let stream = (ns ? crd.ns(ns) : crd).workflows.getStream({qs: {watch: true, labelSelector: labelSelector.join(',')}});
            stream.on('end', () => observer.complete());
            stream.on('error', e => observer.error(e));
            stream.on('close', () => observer.complete());
            stream = stream.pipe(new JSONStream());
            stream.on('data', data => data && observer.next(data));
        }).flatMap(change => Observable.fromPromise(deCompressNodes(change.object).then(workflow => ({...change, object: workflow}))));
        if (ns) {
            updatesSource = updatesSource.filter(change => {
                return change.object.metadata.namespace === ns;
            });
        }
        if (req.query.name) {
            updatesSource = updatesSource.filter(change => change.object.metadata.name === req.query.name);
        }
        streamServerEvents(req, res, updatesSource, item => JSON.stringify(item));
    });

    function getNamespace(req: express.Request) {
        return forceNamespaceIsolation ? namespace : req.query.namespace || req.params.namespace;
    }

    function getWorkflow(ns: string, name: string): Promise<models.Workflow> {
        return crd
            .ns(ns)
            .workflows.get(name)
            .then(deCompressNodes);
    }

    async function deCompressNodes(workFlow: models.Workflow): Promise<models.Workflow> {
        if (workFlow.status.compressedNodes !== undefined && workFlow.status.compressedNodes !== '') {
            const buffer = Buffer.from(workFlow.status.compressedNodes, 'base64');
            const unCompressedBuffer = await promisify(zlib.unzip)(buffer);
            workFlow.status.nodes = JSON.parse(unCompressedBuffer.toString());
            delete workFlow.status.compressedNodes;
            return workFlow;
        } else {
            return workFlow;
        }
    }

    function loadNodeArtifact(wf: models.Workflow, nodeId: string, artifactName: string): Promise<{data: Buffer; fileName: string}> {
        return new Promise(async (resolve, reject) => {
            const node = wf.status.nodes[nodeId];
            const artifact = node.outputs.artifacts.find(item => item.name === artifactName);
            if (artifact.s3) {
                try {
                    const secretAccessKey = decodeBase64(
                        (await core.ns(wf.metadata.namespace).secrets.get(artifact.s3.secretKeySecret.name)).data[artifact.s3.secretKeySecret.key]
                    ).trim();
                    const accessKeyId = decodeBase64(
                        (await core.ns(wf.metadata.namespace).secrets.get(artifact.s3.accessKeySecret.name)).data[artifact.s3.accessKeySecret.key]
                    ).trim();
                    const s3 = new aws.S3({
                        region: artifact.s3.region,
                        secretAccessKey,
                        accessKeyId,
                        endpoint: `http://${artifact.s3.endpoint}`,
                        s3ForcePathStyle: true,
                        signatureVersion: 'v4'
                    });
                    s3.getObject({Bucket: artifact.s3.bucket, Key: artifact.s3.key}, (err, data) => {
                        if (err) {
                            reject(err);
                        } else {
                            resolve({data: data.Body as Buffer, fileName: path.basename(artifact.s3.key)});
                        }
                    });
                } catch (e) {
                    reject(e);
                }
            } else {
                reject({code: 'INTERNAL_ERROR', message: 'Artifact source is not supported'});
            }
        });
    }

    app.get('/api/workflows/:namespace/:name/artifacts/:nodeId/:artifactName', async (req, res) => {
        try {
            const wf = await getWorkflow(getNamespace(req), req.params.name);
            const artifact = await loadNodeArtifact(wf, req.params.nodeId, req.params.artifactName);
            const readStream = new nodeStream.PassThrough();
            readStream.end(artifact.data);
            res.set('Content-disposition', 'attachment; filename=' + artifact.fileName);
            readStream.pipe(res);
        } catch (err) {
            res.status(500).send(err);
            logger.error(err);
        }
    });

    app.get('/api/logs/:namespace/:name/:nodeId/:container', async (req: express.Request, res: express.Response) => {
        try {
            const wf = await getWorkflow(getNamespace(req), req.params.name);
            try {
                await core.ns(wf.metadata.namespace).pods.get(req.params.nodeId);
                const logsSource = reactifyStringStream(
                    core
                        .ns(wf.metadata.namespace)
                        .po(req.params.nodeId)
                        .log.getStream({qs: {container: req.params.container, follow: true}})
                );
                streamServerEvents(req, res, logsSource, item => item.toString());
            } catch (e) {
                if (e.code === 404) {
                    // Try load logs from S3 if pod already deleted
                    const artifact = await loadNodeArtifact(wf, req.params.nodeId, 'main-logs');
                    streamServerEvents(req, res, Observable.from(artifact.data.toString('utf8').split('\n')), line => line);
                } else {
                    throw e;
                }
            }
        } catch (e) {
            logger.error(e);
            res.send(e);
        }
    });

    const serveIndex = (req: express.Request, res: express.Response) => {
        fileToString(`${uiDist}/index.html`)
            .then(content => {
                return content.replace(`<base href="/">`, `<base href="${uiBaseHref}">`);
            })
            .then(indexContent => res.send(indexContent))
            .catch(err => res.send(err));
    };

    app.get('/index.html', serveIndex);
    app.use(express.static(uiDist, {index: false}));
    app.use(async (req: express.Request, res: express.Response, next: express.NextFunction) => {
        if ((req.method === 'GET' || req.method === 'HEAD') && req.accepts('html')) {
            serveIndex(req, res);
        } else {
            next();
        }
    });

    const server = http.createServer(app);
    consoleProxy.create(server, core);

    return server;
}
