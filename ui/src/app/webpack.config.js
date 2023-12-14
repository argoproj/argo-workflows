'use strict;' /* eslint-env node */ /* eslint-disable @typescript-eslint/no-var-requires */;

const MonacoWebpackPlugin = require('monaco-editor-webpack-plugin');
const CopyWebpackPlugin = require('copy-webpack-plugin');
const HtmlWebpackPlugin = require('html-webpack-plugin');
// const BundleAnalyzerPlugin = require('webpack-bundle-analyzer').BundleAnalyzerPlugin;
const webpack = require('webpack');
const path = require('path');

const isProd = process.env.NODE_ENV === 'production';
const proxyConf = {
    target: isProd ? '' : 'http://localhost:2746',
    secure: false
};

console.log(`Bundling for ${isProd ? 'production' : 'development'}...`);

const config = {
    mode: isProd ? 'production' : 'development',
    entry: {
        main: './src/app/index.tsx'
    },
    output: {
        filename: '[name].[contenthash].js',
        path: __dirname + '/../../dist/app'
    },

    devtool: 'source-map',

    resolve: {
        extensions: ['.ts', '.tsx', '.js', '.json', '.ttf'],
        fallback: {fs: false} // ignore `node:fs` on front-end
    },

    module: {
        rules: [
            {
                test: /\.tsx?$/,
                use: [
                    ...(isProd ? [] : ['react-hot-loader/webpack']),
                    `ts-loader?transpileOnly=${!isProd}&allowTsInNodeModules=true&configFile=${path.resolve('./src/app/tsconfig.json')}`
                ]
            },
            {
                enforce: 'pre',
                exclude: [/node_modules\/react-paginate/, /node_modules\/monaco-editor/],
                test: /\.js$/,
                use: [...(isProd ? ['babel-loader'] : ['source-map-loader'])]
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
        // this needs to be disabled to allow EventSource to work
        compress: false,
        historyApiFallback: {
            disableDotRule: true
        },
        headers: {
            'X-Frame-Options': 'SAMEORIGIN'
        },
        proxy: {
            '/api/v1': proxyConf,
            '/artifact-files': proxyConf,
            '/artifacts': proxyConf,
            '/input-artifacts': proxyConf,
            '/artifacts-by-uid': proxyConf,
            '/input-artifacts-by-uid': proxyConf,
            '/oauth2': proxyConf
        }
    }
};

module.exports = config;
