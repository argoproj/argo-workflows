'use strict;';

const CopyWebpackPlugin = require('copy-webpack-plugin');
const HtmlWebpackPlugin = require('html-webpack-plugin');
const webpack = require('webpack');
const path = require('path');

const isProd = process.env.NODE_ENV === 'production';

const config = {
    entry: './src/app/index.tsx',
    output: {
        filename: '[name].[chunkhash].js',
        path: __dirname + '/../../dist/app'
    },

    devtool: 'source-map',

    resolve: {
        extensions: ['.ts', '.tsx', '.js', '.json']
    },

    module: {
        rules: [
            {
                test: /\.tsx?$/,
                loaders: [ ...( isProd ? [] : ['react-hot-loader/webpack']), `ts-loader?allowTsInNodeModules=true&configFile=${path.resolve('./src/app/tsconfig.json')}`]
            }, {
                enforce: 'pre',
                test: /\.js$/,
                loader: 'source-map-loader'
            }, {
                test: /\.scss$/,
                loader: 'style-loader!raw-loader!sass-loader'
            }, {
                test: /\.css$/,
                loader: 'style-loader!raw-loader'
            },
        ]
    },
    node: {
        fs: 'empty',
    },
    plugins: [
        new webpack.DefinePlugin({
            'process.env.NODE_ENV': JSON.stringify(process.env.NODE_ENV || 'development'),
            SYSTEM_INFO: JSON.stringify({
                version: process.env.ARGO_VERSION || 'latest',
            }),
        }),
        new HtmlWebpackPlugin({ template: 'src/app/index.html' }),
        new CopyWebpackPlugin([{
            from: 'node_modules/argo-ui/src/assets', to: 'assets'
        }, {
            from: 'node_modules/@fortawesome/fontawesome-free/webfonts', to: 'assets/fonts'
        }]),
    ],
    devServer: {
        historyApiFallback: true,
        proxy: {
            '/api': {
                'target': process.env.ARGO_API_URL || 'http://localhost:8081',
                'secure': false,
            }
        }
    }
};

module.exports = config;
