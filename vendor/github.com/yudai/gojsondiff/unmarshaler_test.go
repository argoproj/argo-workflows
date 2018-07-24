package gojsondiff_test

import (
	. "github.com/yudai/gojsondiff"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/yudai/gojsondiff/tests"

	"encoding/json"
	"fmt"
	"github.com/yudai/pp"
)

var _ = Describe("Gojsondiff", func() {
	Describe("Unmarshaller", func() {
		Describe("CompareObjects", func() {
			It("", func() {
				um := NewUnmarshaller()
				diff, err := um.UnmarshalString(`
{
  "summary": [
    "@@ -638,17 +638,17 @@\n via, Bra\n-z\n+s\n il,  %0ACh\n@@ -916,20 +916,13 @@\n re a\n-lso known as\n+.k.a.\n  Car\n",
    0,
    2
  ],
  "surface": [
    17840000,
    0,
    0
  ],
  "demographics": {
    "population": [
      385742554,
      385744896
    ]
  },
  "languages": {
    "2": [
      "inglés"
    ],
    "_t": "a",
    "_2": [
      "english",
      0,
      0
    ]
  },
  "countries": {
    "0": {
      "capital": [
        "Buenos Aires",
        "Rawson"
      ]
    },
    "9": [
      {
        "name": "Antártida",
        "unasur": false
      }
    ],
    "10": {
      "population": [
        42888594
      ]
    },
    "_t": "a",
    "_4": [
      "",
      10,
      3
    ],
    "_8": [
      "",
      2,
      3
    ],
    "_10": [
      {
        "name": "Uruguay",
        "capital": "Montevideo",
        "independence": "1825-08-25T07:00:00.000Z",
        "unasur": true
      },
      0,
      0
    ],
    "_11": [
      {
        "name": "Venezuela",
        "capital": "Caracas",
        "independence": "1811-07-05T07:00:00.000Z",
        "unasur": true
      },
      0,
      0
    ]
  },
  "spanishName": [
    "Sudamérica"
  ]
}
`)
				Expect(err).To(BeNil())
				pp.Print(diff)

				a := LoadFixture("FIXTURES/jsondiffpatch.json")
				differ := New()
				differ.ApplyPatch(a, diff)
				pp.Println(a)
				result, _ := json.Marshal(a)
				fmt.Println(string(result))
			})
		})
	})
})
