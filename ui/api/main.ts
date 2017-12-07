process.env.NODE_TLS_REJECT_UNAUTHORIZED = '0';

import * as yargs from 'yargs';
import * as app from './app';
import * as path from 'path';

const argv = yargs.argv;

app.create(
  argv.uiDist || path.join(__dirname, '..', 'dist'),
  argv.inCluster === 'true',
  argv.namespace || 'default',
  argv.crdVersion || 'v1alpha1',
).listen(8001);
