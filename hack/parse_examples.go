package main

import (
	"os"
	"regexp"
)

const (
	newHeader = `<summary>Examples with this field (click to open)</summary>
<br>
<ul>`
	newHeaderAlt = `<summary>Examples (click to open)</summary>
<br>
<ul>`
	newLink    = `    <li> <a href="$2">$1</a>`
	newDetails = `</ul>
</details>`
)

var (
	headerRegex    = regexp.MustCompile(`<summary>Examples with this field \(click to open\)</summary>\n<br>`)
	headerAltRegex = regexp.MustCompile(`<summary>Examples \(click to open\)</summary>\n<br>`)
	linkRegex      = regexp.MustCompile(`- \[\x60(.+?)\x60\]\((.+?)\)`)
	detailsRegex   = regexp.MustCompile(`</details>`)
)

func parseExamples() {
	file, err := os.ReadFile("site/fields/index.html")
	if err != nil {
		panic(err)
	}

	file = headerRegex.ReplaceAll(file, []byte(newHeader))
	file = headerAltRegex.ReplaceAll(file, []byte(newHeaderAlt))
	file = linkRegex.ReplaceAll(file, []byte(newLink))
	file = detailsRegex.ReplaceAll(file, []byte(newDetails))

	err = os.WriteFile("site/fields/index.html", file, 0o600)
	if err != nil {
		panic(err)
	}
}
