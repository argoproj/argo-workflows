process.env.NODE_TLS_REJECT_UNAUTHORIZED = '0';

import * as path from 'path';
import * as yargs from 'yargs';
import * as app from './app';

const argv = yargs.argv;

const ip = argv.ip || '0.0.0.0';
const port = argv.port || '8001';
// tslint:disable-next-line
console.log(`start argo-ui on ${argv.ip}:${argv.port}`);

app.create(
    argv.uiDist || path.join(__dirname, '..', '..', 'dist', 'app'),
    argv.uiBaseHref || '/',
    argv.inCluster === 'true',
    argv.namespace || 'default',
    argv.forceNamespaceIsolation === 'true',
    argv.instanceId || undefined,
    argv.crdVersion || 'v1alpha1'
).listen(port, ip);
