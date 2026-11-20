package static

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v4/ui"
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

func TestAddStyle(t *testing.T) {
	testCases := []struct {
		name     string
		data     string
		css      string
		expected string
	}{
		{
			name:     "before body end tag",
			data:     `<html><body><div id="app"></div></body></html>`,
			css:      `.login { background: red; }`,
			expected: `<html><body><div id="app"></div><style>.login { background: red; }</style></body></html>`,
		},
		{
			name:     "without body end tag",
			data:     `<html><body><div id="app"></div>`,
			css:      `.login { background: red; }`,
			expected: `<html><body><div id="app"></div><style>.login { background: red; }</style>`,
		},
		{
			name:     "cannot close the style element",
			data:     `<body></body>`,
			css:      `</style><script>alert(1)</script>`,
			expected: `<body><style><\/style><script>alert(1)<\/script></style></body>`,
		},
		{
			name:     "with byte order mark",
			data:     `<body></body>`,
			css:      "\ufeff.login { background: red; }",
			expected: `<body><style>.login { background: red; }</style></body>`,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expected, string(addStyle([]byte(testCase.data), testCase.css)))
		})
	}
}

func TestGetIndexDataCSS(t *testing.T) {
	data, err := NewFilesServer("/", false, "", "", ui.Embedded, "").getIndexData()
	require.NoError(t, err)
	assert.NotContains(t, string(data), "<style>")

	data, err = NewFilesServer("/", false, "", "", ui.Embedded, ".login { background: red; }").getIndexData()
	require.NoError(t, err)
	assert.Contains(t, string(data), "<style>.login { background: red; }</style>")
}
