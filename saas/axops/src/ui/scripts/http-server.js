'use strict';

process.env.NODE_TLS_REJECT_UNAUTHORIZED = '0';

let express = require('express');
let fallback = require('express-history-api-fallback');
let httpProxy = require('http-proxy');

let app = express();
let root = __dirname + '/../dist';

app.use(express.static(root));
let apiProxy = httpProxy.createProxyServer();
app.all('/v1/*', function(req, res) {
    apiProxy.web(req, res, {
        target: 'https://' + (process.env.AX_CLUSTER_HOST || 'dev.applatix.net'),
        secure: false
    });
});
app.use(fallback('index.html', { root: root }));

app.listen(process.env.PORT || 3000);