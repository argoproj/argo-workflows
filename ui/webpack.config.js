'use strict;' /* eslint-env node */ /* eslint-disable @typescript-eslint/no-var-requires */;

const MonacoWebpackPlugin = require('monaco-editor-webpack-plugin');
const CopyWebpackPlugin = require('copy-webpack-plugin');
const HtmlWebpackPlugin = require('html-webpack-plugin');
// const BundleAnalyzerPlugin = require('webpack-bundle-analyzer').BundleAnalyzerPlugin;
const webpack = require('webpack');

const isProd = process.env.NODE_ENV === 'production';
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
        path: __dirname + '/../../dist/app'
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
        new HtmlWebpackPlugin({template: 'src/app/index.html'}),
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
        historyApiFallback: {
            disableDotRule: true
        },
        headers: {
            'X-Frame-Options': 'SAMEORIGIN'
        },
        proxy: [
            {
                context: ['/api/v1', '/artifact-files', '/artifacts', '/input-artifacts', '/artifacts-by-uid', '/input-artifacts-by-uid', '/oauth2'],
                target: proxyTarget,
                secure: false,
                xfwd: true // add x-forwarded-* headers to simulate real-world reverse proxy servers
            }
        ]
    }
};

module.exports = config;
