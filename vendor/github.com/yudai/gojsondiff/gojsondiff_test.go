package gojsondiff_test

import (
	. "github.com/yudai/gojsondiff"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/yudai/gojsondiff/tests"

	"io/ioutil"
)

var _ = Describe("Gojsondiff", func() {
	Describe("Differ", func() {
		Describe("CompareObjects", func() {

			var (
				a, b   map[string]interface{}
				differ *Differ
			)

			BeforeEach(func() {
				differ = New()
			})

			Context("There are no difference between the two JSON strings", func() {
				It("Detects nothing", func() {
					a = LoadFixture("FIXTURES/base.json")
					b = LoadFixture("FIXTURES/base.json")

					diff := differ.CompareObjects(a, b)
					Expect(diff.Modified()).To(BeFalse())
				})
			})

			Context("There are some values modified", func() {
				It("Detects changes", func() {
					a = LoadFixture("FIXTURES/base.json")
					b = LoadFixture("FIXTURES/base_changed.json")

					diff := differ.CompareObjects(a, b)
					Expect(diff.Modified()).To(BeTrue())
					differ.ApplyPatch(a, diff)
					Expect(a).To(Equal(LoadFixture("FIXTURES/base_changed.json")))
				})
			})

			Context("There are values only types are changed", func() {
				It("Detects changed types", func() {
					a := LoadFixture("FIXTURES/changed_types_from.json")
					b := LoadFixture("FIXTURES/changed_types_to.json")

					diff := differ.CompareObjects(a, b)
					differ.ApplyPatch(a, diff)
					Expect(a).To(Equal(LoadFixture("FIXTURES/changed_types_to.json")))
				})
			})

			Context("There is a moved item in an array", func() {
				It("Detects changed types", func() {
					a := LoadFixture("FIXTURES/move_from.json")
					b := LoadFixture("FIXTURES/move_to.json")

					diff := differ.CompareObjects(a, b)
					Expect(diff.Modified()).To(BeTrue())
					differ.ApplyPatch(a, diff)
					Expect(a).To(Equal(LoadFixture("FIXTURES/move_to.json")))
				})
			})

			Context("There are long text diff", func() {
				It("Detects changes", func() {
					a = LoadFixture("FIXTURES/long_text_from.json")
					b = LoadFixture("FIXTURES/long_text_to.json")

					diff := differ.CompareObjects(a, b)
					Expect(diff.Modified()).To(BeTrue())
					differ.ApplyPatch(a, diff)
					Expect(a).To(Equal(LoadFixture("FIXTURES/long_text_to.json")))
				})
			})
		})
		Describe("CompareArrays", func() {

			var (
				a, b   []interface{}
				differ *Differ
			)

			BeforeEach(func() {
				differ = New()
			})

			Context("There are no difference between the two JSON strings", func() {
				It("Detects nothing", func() {
					a = LoadFixtureAsArray("FIXTURES/array.json")
					b = LoadFixtureAsArray("FIXTURES/array.json")

					diff := differ.CompareArrays(a, b)
					Expect(diff.Modified()).To(BeFalse())
				})
			})

			Context("There are some values modified", func() {
				It("Detects changes", func() {
					a = LoadFixtureAsArray("FIXTURES/array.json")
					b = LoadFixtureAsArray("FIXTURES/array_changed.json")

					diff := differ.CompareArrays(a, b)
					Expect(diff.Modified()).To(BeTrue())
					Expect(len(diff.Deltas())).To(Equal(1))
				})
			})
		})
		Describe("Compare", func() {
			Context("There are some values modified", func() {
				It("Detects changes", func() {
					aFile := "FIXTURES/base.json"
					bFile := "FIXTURES/base_changed.json"
					aObj := LoadFixture(aFile)
					bObj := LoadFixture(bFile)

					differ := New()

					diffObj := differ.CompareObjects(aObj, bObj)
					Expect(diffObj.Modified()).To(BeTrue())

					aStr, err := ioutil.ReadFile(aFile)
					Expect(err).To(BeNil())
					bStr, err := ioutil.ReadFile(bFile)
					Expect(err).To(BeNil())

					diffStr, err := differ.Compare(aStr, bStr)
					Expect(err).To(BeNil())
					Expect(diffStr).To(Equal(diffObj))
				})
			})
		})
	})
})
