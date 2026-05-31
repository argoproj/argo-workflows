module.exports = {
  preset: "ts-jest",
  testEnvironment: "jsdom",
  // jest-environment-jsdom defaults customExportConditions to ["browser"], which makes
  // packages like `yaml` resolve to their ESM browser build that jest can't transform.
  // Prefer the "node" condition so they resolve to their CommonJS dist entry instead.
  testEnvironmentOptions: {
    customExportConditions: ["node"],
  },
  setupFilesAfterEnv: ["<rootDir>/setupTests.ts"],
  transform: {
    "^.+\\.tsx?$": ["ts-jest", {}],
    // react-markdown@10 and its dependency chain ship as pure ESM .js files; babel-jest
    // (with preset-env) transpiles them to CommonJS so jest can execute them. Only the
    // packages allow-listed in transformIgnorePatterns reach this transform.
    "^.+\\.m?js$": ["babel-jest", {presets: [["@babel/preset-env", {targets: {node: "current"}}]]}],
  },
  transformIgnorePatterns: [
    // react-markdown@10 and its remark/rehype/micromark/unified dependency chain are pure ESM.
    // jest can't execute ESM in node_modules, so we transform argo-ui plus that whole closure.
    // The closure list is derived from the actual react-markdown + remark-gfm dependency tree in yarn.lock.
    "node_modules/(?!(argo-ui|@ungap/structured-clone|bail|ccount|character-entities|character-entities-html4|character-entities-legacy|character-reference-invalid|comma-separated-tokens|decode-named-character-reference|devlop|escape-string-regexp|estree-util-is-identifier-name|hast-util-to-jsx-runtime|hast-util-whitespace|html-url-attributes|is-alphabetical|is-alphanumerical|is-decimal|is-hexadecimal|is-plain-obj|longest-streak|markdown-table|mdast-util-find-and-replace|mdast-util-from-markdown|mdast-util-gfm|mdast-util-gfm-autolink-literal|mdast-util-gfm-footnote|mdast-util-gfm-strikethrough|mdast-util-gfm-table|mdast-util-gfm-task-list-item|mdast-util-mdx-expression|mdast-util-mdx-jsx|mdast-util-mdxjs-esm|mdast-util-phrasing|mdast-util-to-hast|mdast-util-to-markdown|mdast-util-to-string|micromark|micromark-core-commonmark|micromark-extension-gfm|micromark-extension-gfm-autolink-literal|micromark-extension-gfm-footnote|micromark-extension-gfm-strikethrough|micromark-extension-gfm-table|micromark-extension-gfm-tagfilter|micromark-extension-gfm-task-list-item|micromark-factory-destination|micromark-factory-label|micromark-factory-space|micromark-factory-title|micromark-factory-whitespace|micromark-util-character|micromark-util-chunked|micromark-util-classify-character|micromark-util-combine-extensions|micromark-util-decode-numeric-character-reference|micromark-util-decode-string|micromark-util-encode|micromark-util-html-tag-name|micromark-util-normalize-identifier|micromark-util-resolve-all|micromark-util-sanitize-uri|micromark-util-subtokenize|micromark-util-symbol|micromark-util-types|parse-entities|property-information|react-markdown|remark-gfm|remark-parse|remark-rehype|remark-stringify|space-separated-tokens|stringify-entities|trim-lines|trough|unified|unist-util-is|unist-util-position|unist-util-stringify-position|unist-util-visit|unist-util-visit-parents|vfile|vfile-message|zwitch)/)"
  ],
  moduleNameMapper: {
    "\\.(css|scss)$": "<rootDir>/__mocks__/styleMock.js",
    '^formidable$': '<rootDir>/__mocks__/formidable.js',
    '^node:fs$': 'fs',
  },
};
