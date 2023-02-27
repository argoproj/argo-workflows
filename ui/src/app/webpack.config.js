"use strict;";

const MonacoWebpackPlugin = require('monaco-editor-webpack-plugin');
const CopyWebpackPlugin = require("copy-webpack-plugin");
const HtmlWebpackPlugin = require("html-webpack-plugin");
const webpack = require("webpack");
const path = require("path");

const isProd = process.env.NODE_ENV === "production";

console.log("isProd=", isProd)

const config = {
  mode: isProd ? "production" : "development",
  entry: {
    main: "./src/app/index.tsx"
  },
  output: {
    filename: "[name].[hash].js",
    path: __dirname + "/../../dist/app"
  },

  devtool: "source-map",

  resolve: {
    extensions: [".ts", ".tsx", ".js", ".json", ".ttf"]
  },

  module: {
    rules: [
      {
        test: /\.tsx?$/,
        loaders: [...(isProd ? [] : ["react-hot-loader/webpack"]), `ts-loader?transpileOnly=${!isProd}&allowTsInNodeModules=true&configFile=${path.resolve("./src/app/tsconfig.json")}`]
      }, {
        enforce: 'pre',
        exclude: [
          /node_modules\/react-paginate/,
          /node_modules\/monaco-editor/,
        ],
        test: /\.js$/,
        loaders: [...(isProd ? ['babel-loader'] : ['source-map-loader'])],
      }, {
        test: /\.scss$/,
        loader: "style-loader!raw-loader!sass-loader"
      }, {
        test: /\.css$/,
        loader: "style-loader!raw-loader"
      }, {
        test: /\.ttf$/,
        use: ['file-loader']
      }
    ]
  },
  node: {
    fs: "empty"
  },
  plugins: [
    new webpack.DefinePlugin({
      "process.env.NODE_ENV": JSON.stringify(process.env.NODE_ENV || "development"),
      SYSTEM_INFO: JSON.stringify({
        version: process.env.VERSION || "latest"
      }), 
      "process.env.DEFAULT_TZ": JSON.stringify("UTC"),
    }),
    new HtmlWebpackPlugin({ template: "src/app/index.html" }),
    new CopyWebpackPlugin([{
      from: "node_modules/argo-ui/src/assets", to: "assets"
    }, {
      from: "node_modules/@fortawesome/fontawesome-free/webfonts", to: "assets/fonts"
    }, {
      from: "../api/openapi-spec/swagger.json", to: "assets/openapi-spec/swagger.json"
    }, {
      from: "../api/jsonschema/schema.json", to: "assets/jsonschema/schema.json"
    }, {
      from: 'node_modules/monaco-editor/min/vs/base/browser/ui/codiconLabel/codicon/codicon.ttf', to: "."
    }]),
    new MonacoWebpackPlugin({"languages":["json","yaml"]})
  ],
  devServer: {
    // this needs to be disable to allow EventSource to work
    compress: false,
    historyApiFallback: {
      disableDotRule: true
    },
    headers: {
      'X-Frame-Options': 'SAMEORIGIN'
    },
    proxy: {
      "/api/v1": {
        "target": isProd ? "" : "http://localhost:2746",
        "secure": false
      },
      "/artifact-files": {
        "target": isProd ? "" : "http://localhost:2746",
        "secure": false
      },
      "/artifacts": {
        "target": isProd ? "" : "http://localhost:2746",
        "secure": false
      },
      "/input-artifacts": {
        "target": isProd ? "" : "http://localhost:2746",
        "secure": false
      },
      "/artifacts-by-uid": {
        "target": isProd ? "" : "http://localhost:2746",
        "secure": false
      },
      "/input-artifacts-by-uid": {
        "target": isProd ? "" : "http://localhost:2746",
        "secure": false
      },
      '/oauth2': {
        'target': isProd ? '' : 'http://localhost:2746',
        'secure': false,
      },
    }
  }
};

module.exports = config;
