module.exports = {
   preset: 'ts-jest',
     "transformIgnorePatterns": [
       "node_modules/(?!(argo-ui)/)"
     ],
      "moduleNameMapper": {
         "\\.(css|scss)$": "<rootDir>/__mocks__/styleMock.js"
       },
};