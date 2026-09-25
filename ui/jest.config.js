module.exports = {
  preset: "ts-jest",
  testEnvironment: "jsdom",
  // Playwright specs under e2e/ are run by `yarn e2e`, not Jest.
  testPathIgnorePatterns: ["<rootDir>/node_modules/", "<rootDir>/e2e/"],
  setupFilesAfterEnv: ["<rootDir>/setupTests.ts"],
  transformIgnorePatterns: [
    "node_modules/(?!(argo-ui)/)"
  ],
  moduleNameMapper: {
    "\\.(css|scss)$": "<rootDir>/__mocks__/styleMock.js",
    '^formidable$': '<rootDir>/__mocks__/formidable.js',
    '^react-markdown$': '<rootDir>/__mocks__/react-markdown.js',
    '^remark-breaks$': '<rootDir>/__mocks__/remark-breaks.js',
    '^remark-gfm$': '<rootDir>/__mocks__/remark-gfm.js',
    '^node:fs$': 'fs',
  },
};
