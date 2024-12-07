package static

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReplaceBaseHRef(t *testing.T) {
	testCases := []struct {
		name        string
		data        string
		expected    string
		replaceWith string
	}{
		{
			name: "non-root basepath",
			data: `<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8">
    <title>Argo</title>
    <base href="/">
    <meta name="viewport" content="width=device-width,initial-scale=1">
    <meta name="robots" content="noindex">
    <link rel="icon" type="image/png" href="assets/favicon/favicon-32x32.png" sizes="32x32">
    <link rel="icon" type="image/png" href="assets/favicon/favicon-16x16.png" sizes="16x16">
    <script defer="defer" src="main.js"></script>
  </head>
  <body>
    <div id="app"></div>
  </body>
</html>`,
			expected: `<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8">
    <title>Argo</title>
    <base href="/path1/path2/path3/">
    <meta name="viewport" content="width=device-width,initial-scale=1">
    <meta name="robots" content="noindex">
    <link rel="icon" type="image/png" href="assets/favicon/favicon-32x32.png" sizes="32x32">
    <link rel="icon" type="image/png" href="assets/favicon/favicon-16x16.png" sizes="16x16">
    <script defer="defer" src="main.js"></script>
  </head>
  <body>
    <div id="app"></div>
  </body>
</html>`,
			replaceWith: `<base href="/path1/path2/path3/">`,
		},
		{
			name: "root basepath",
			data: `<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8">
    <title>Argo</title>
    <base href="/any/path/test/">
    <meta name="viewport" content="width=device-width,initial-scale=1">
    <meta name="robots" content="noindex">
    <link rel="icon" type="image/png" href="assets/favicon/favicon-32x32.png" sizes="32x32">
    <link rel="icon" type="image/png" href="assets/favicon/favicon-16x16.png" sizes="16x16">
    <script defer="defer" src="main.js"></script>
  </head>
  <body>
    <div id="app"></div>
  </body>
</html>`,
			expected: `<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8">
    <title>Argo</title>
    <base href="/">
    <meta name="viewport" content="width=device-width,initial-scale=1">
    <meta name="robots" content="noindex">
    <link rel="icon" type="image/png" href="assets/favicon/favicon-32x32.png" sizes="32x32">
    <link rel="icon" type="image/png" href="assets/favicon/favicon-16x16.png" sizes="16x16">
    <script defer="defer" src="main.js"></script>
  </head>
  <body>
    <div id="app"></div>
  </body>
</html>`,
			replaceWith: `<base href="/">`,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := replaceBaseHRef(testCase.data, testCase.replaceWith)
			assert.Equal(t, testCase.expected, result)
		})
	}
}
