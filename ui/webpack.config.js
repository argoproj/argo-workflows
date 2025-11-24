'use strict;' /* eslint-env node */

const MonacoWebpackPlugin = require('monaco-editor-webpack-plugin');
const CopyWebpackPlugin = require('copy-webpack-plugin');
const HtmlWebpackPlugin = require('html-webpack-plugin');
// const BundleAnalyzerPlugin = require('webpack-bundle-analyzer').BundleAnalyzerPlugin;
const webpack = require('webpack');

const isProd = process.env.NODE_ENV === 'production';
const base = process.env.ARGO_BASE_HREF || '/';
let proxyTarget = '';
if (!isProd) {
    const isSecure = process.env.ARGO_SECURE === 'true';
    proxyTarget = `${isSecure ? 'https' : 'http'}://localhost:2746`;
}

console.log(`Bundling for ${isProd ? 'production' : 'development'}...`);

const config = {
    mode: isProd ? 'production' : 'development',
    entry: {
        main: './src/index.tsx'
    },
    output: {
        filename: '[name].[contenthash].js',
        path: __dirname + '/dist/app',
    },

    devtool: isProd ? 'source-map' : 'eval',

    resolve: {
        extensions: ['.ts', '.tsx', '.js', '.json', '.ttf'],
        fallback: {fs: false} // ignore `node:fs` on front-end
    },

    module: {
        rules: [
            {
                test: /\.tsx?$/,
                loader: 'esbuild-loader'
            },
            {
                enforce: 'pre',
                exclude: [/node_modules\/monaco-editor/],
                test: /\.js$/,
                use: ['esbuild-loader']
            },
            {
                test: /\.scss$/,
                use: ['style-loader', 'raw-loader', 'sass-loader']
            },
            {
                test: /\.css$/,
                use: ['style-loader', 'raw-loader']
            },
            {
                test: /\.(png|svg|jpg|jpeg|gif)$/i,
                type: 'asset/resource'
            },
            {
                test: /\.(woff|woff2|eot|ttf|otf)$/i,
                type: 'asset/resource'
            }
        ]
    },
    plugins: [
        new webpack.DefinePlugin({
            'process.env.DEFAULT_TZ': JSON.stringify('UTC'),
            'process.env.NODE_ENV': JSON.stringify(process.env.NODE_ENV || 'development'),
            'SYSTEM_INFO': JSON.stringify({
                version: process.env.VERSION || 'latest'
            })
        }),
        new HtmlWebpackPlugin({
            template: 'src/index.html',
            // Inject <base href="..."> tag into <head> to support non-root base using "--base-href" or "ARGO_BASE_HREF"
            base,
        }),
        new CopyWebpackPlugin({
            patterns: [
                {
                    from: 'node_modules/argo-ui/src/assets',
                    to: 'assets'
                },
                {
                    from: 'node_modules/@fortawesome/fontawesome-free/webfonts',
                    to: 'assets/fonts'
                },
                {
                    from: '../api/openapi-spec/swagger.json',
                    to: 'assets/openapi-spec/swagger.json'
                },
                {
                    from: '../api/jsonschema/schema.json',
                    to: 'assets/jsonschema/schema.json'
                },
                {
                    from: 'node_modules/monaco-editor/min/vs/base/browser/ui/codicons/codicon/',
                    to: '.'
                }
            ]
        }),
        new MonacoWebpackPlugin({languages: ['json', 'yaml']})
        // new BundleAnalyzerPlugin()
    ],

    devServer: {
        server: process.env.ARGO_UI_SECURE === 'true' ? 'https' : 'http',
        // this needs to be disabled to allow EventSource to work
        compress: false,
        // Docs: https://github.com/bripkens/connect-history-api-fallback
        historyApiFallback: {
            disableDotRule: true,
            // Needed to fix 404s: https://github.com/webpack/webpack-dev-server/issues/1457#issuecomment-415527819
            index: base
        },
        devMiddleware: {
            publicPath: base,
        },
        headers: {
            'X-Frame-Options': 'SAMEORIGIN'
        },
        proxy: [
            {
                // Proxy paths handled by the API server defined at https://github.com/argoproj/argo-workflows/blob/cb7ebd9393f3322abf455d906e39a3a976421b30/server/apiserver/argoserver.go#L413-L428
                context: ['api/v1', 'artifact-files', 'artifacts', 'input-artifacts', 'artifacts-by-uid', 'input-artifacts-by-uid', 'oauth2']
                    .map(path => `${base}${path}`),
                target: proxyTarget,
                secure: false,
                // Rewrite the base href for non-root paths
                // Docs: https://github.com/chimurai/http-proxy-middleware?tab=readme-ov-file#pathrewrite-objectfunction
                pathRewrite: { [`^${base}`]: '' },
                xfwd: true // add x-forwarded-* headers to simulate real-world reverse proxy servers
            }
        ]
    }
};

module.exports = config;
