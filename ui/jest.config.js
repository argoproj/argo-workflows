module.exports = {
  preset: "ts-jest",
  testEnvironment: "jsdom",
  setupFilesAfterEnv: ["<rootDir>/setupTests.ts"],
  transformIgnorePatterns: [
    "node_modules/(?!(argo-ui)/)"
  ],
  moduleNameMapper: {
    "\\.(css|scss)$": "<rootDir>/__mocks__/styleMock.js",
    '^formidable$': '<rootDir>/__mocks__/formidable.js',
    '^node:fs$': 'fs',
  },
};
