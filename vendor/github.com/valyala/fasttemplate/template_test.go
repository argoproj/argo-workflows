package fasttemplate

import (
	"bytes"
	"io"
	"testing"
)

func TestEmptyTemplate(t *testing.T) {
	tpl := New("", "[", "]")

	s := tpl.ExecuteString(map[string]interface{}{"foo": "bar", "aaa": "bbb"})
	if s != "" {
		t.Fatalf("unexpected string returned %q. Expected empty string", s)
	}
}

func TestEmptyTagStart(t *testing.T) {
	expectPanic(t, func() { NewTemplate("foobar", "", "]") })
}

func TestEmptyTagEnd(t *testing.T) {
	expectPanic(t, func() { NewTemplate("foobar", "[", "") })
}

func TestNoTags(t *testing.T) {
	template := "foobar"
	tpl := New(template, "[", "]")

	s := tpl.ExecuteString(map[string]interface{}{"foo": "bar", "aaa": "bbb"})
	if s != template {
		t.Fatalf("unexpected template value %q. Expected %q", s, template)
	}
}

func TestEmptyTagName(t *testing.T) {
	template := "foo[]bar"
	tpl := New(template, "[", "]")

	s := tpl.ExecuteString(map[string]interface{}{"": "111", "aaa": "bbb"})
	result := "foo111bar"
	if s != result {
		t.Fatalf("unexpected template value %q. Expected %q", s, result)
	}
}

func TestOnlyTag(t *testing.T) {
	template := "[foo]"
	tpl := New(template, "[", "]")

	s := tpl.ExecuteString(map[string]interface{}{"foo": "111", "aaa": "bbb"})
	result := "111"
	if s != result {
		t.Fatalf("unexpected template value %q. Expected %q", s, result)
	}
}

func TestStartWithTag(t *testing.T) {
	template := "[foo]barbaz"
	tpl := New(template, "[", "]")

	s := tpl.ExecuteString(map[string]interface{}{"foo": "111", "aaa": "bbb"})
	result := "111barbaz"
	if s != result {
		t.Fatalf("unexpected template value %q. Expected %q", s, result)
	}
}

func TestEndWithTag(t *testing.T) {
	template := "foobar[foo]"
	tpl := New(template, "[", "]")

	s := tpl.ExecuteString(map[string]interface{}{"foo": "111", "aaa": "bbb"})
	result := "foobar111"
	if s != result {
		t.Fatalf("unexpected template value %q. Expected %q", s, result)
	}
}

func TestTemplateReset(t *testing.T) {
	template := "foo{bar}baz"
	tpl := New(template, "{", "}")
	s := tpl.ExecuteString(map[string]interface{}{"bar": "111"})
	result := "foo111baz"
	if s != result {
		t.Fatalf("unexpected template value %q. Expected %q", s, result)
	}

	template = "[xxxyyyzz"
	if err := tpl.Reset(template, "[", "]"); err == nil {
		t.Fatalf("expecting error for unclosed tag on %q", template)
	}

	template = "[xxx]yyy[zz]"
	if err := tpl.Reset(template, "[", "]"); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	s = tpl.ExecuteString(map[string]interface{}{"xxx": "11", "zz": "2222"})
	result = "11yyy2222"
	if s != result {
		t.Fatalf("unexpected template value %q. Expected %q", s, result)
	}
}

func TestDuplicateTags(t *testing.T) {
	template := "[foo]bar[foo][foo]baz"
	tpl := New(template, "[", "]")

	s := tpl.ExecuteString(map[string]interface{}{"foo": "111", "aaa": "bbb"})
	result := "111bar111111baz"
	if s != result {
		t.Fatalf("unexpected template value %q. Expected %q", s, result)
	}
}

func TestMultipleTags(t *testing.T) {
	template := "foo[foo]aa[aaa]ccc"
	tpl := New(template, "[", "]")

	s := tpl.ExecuteString(map[string]interface{}{"foo": "111", "aaa": "bbb"})
	result := "foo111aabbbccc"
	if s != result {
		t.Fatalf("unexpected template value %q. Expected %q", s, result)
	}
}

func TestLongDelimiter(t *testing.T) {
	template := "foo{{{foo}}}bar"
	tpl := New(template, "{{{", "}}}")

	s := tpl.ExecuteString(map[string]interface{}{"foo": "111", "aaa": "bbb"})
	result := "foo111bar"
	if s != result {
		t.Fatalf("unexpected template value %q. Expected %q", s, result)
	}
}

func TestIdenticalDelimiter(t *testing.T) {
	template := "foo@foo@foo@aaa@"
	tpl := New(template, "@", "@")

	s := tpl.ExecuteString(map[string]interface{}{"foo": "111", "aaa": "bbb"})
	result := "foo111foobbb"
	if s != result {
		t.Fatalf("unexpected template value %q. Expected %q", s, result)
	}
}

func TestDlimitersWithDistinctSize(t *testing.T) {
	template := "foo<?phpaaa?>bar<?phpzzz?>"
	tpl := New(template, "<?php", "?>")

	s := tpl.ExecuteString(map[string]interface{}{"zzz": "111", "aaa": "bbb"})
	result := "foobbbbar111"
	if s != result {
		t.Fatalf("unexpected template value %q. Expected %q", s, result)
	}
}

func TestEmptyValue(t *testing.T) {
	template := "foobar[foo]"
	tpl := New(template, "[", "]")

	s := tpl.ExecuteString(map[string]interface{}{"foo": "", "aaa": "bbb"})
	result := "foobar"
	if s != result {
		t.Fatalf("unexpected template value %q. Expected %q", s, result)
	}
}

func TestNoValue(t *testing.T) {
	template := "foobar[foo]x[aaa]"
	tpl := New(template, "[", "]")

	s := tpl.ExecuteString(map[string]interface{}{"aaa": "bbb"})
	result := "foobarxbbb"
	if s != result {
		t.Fatalf("unexpected template value %q. Expected %q", s, result)
	}
}

func TestNoEndDelimiter(t *testing.T) {
	template := "foobar[foo"
	_, err := NewTemplate(template, "[", "]")
	if err == nil {
		t.Fatalf("expected non-nil error. got nil")
	}

	expectPanic(t, func() { New(template, "[", "]") })
}

func TestUnsupportedValue(t *testing.T) {
	template := "foobar[foo]"
	tpl := New(template, "[", "]")

	expectPanic(t, func() {
		tpl.ExecuteString(map[string]interface{}{"foo": 123, "aaa": "bbb"})
	})
}

func TestMixedValues(t *testing.T) {
	template := "foo[foo]bar[bar]baz[baz]"
	tpl := New(template, "[", "]")

	s := tpl.ExecuteString(map[string]interface{}{
		"foo": "111",
		"bar": []byte("bbb"),
		"baz": TagFunc(func(w io.Writer, tag string) (int, error) { return w.Write([]byte(tag)) }),
	})
	result := "foo111barbbbbazbaz"
	if s != result {
		t.Fatalf("unexpected template value %q. Expected %q", s, result)
	}
}

func TestExecuteFunc(t *testing.T) {
	testExecuteFunc(t, "", "")
	testExecuteFunc(t, "a", "a")
	testExecuteFunc(t, "abc", "abc")
	testExecuteFunc(t, "{foo}", "xxxx")
	testExecuteFunc(t, "a{foo}", "axxxx")
	testExecuteFunc(t, "{foo}a", "xxxxa")
	testExecuteFunc(t, "a{foo}bc", "axxxxbc")
	testExecuteFunc(t, "{foo}{foo}", "xxxxxxxx")
	testExecuteFunc(t, "{foo}bar{foo}", "xxxxbarxxxx")

	// unclosed tag
	testExecuteFunc(t, "{unclosed", "{unclosed")
	testExecuteFunc(t, "{{unclosed", "{{unclosed")
	testExecuteFunc(t, "{un{closed", "{un{closed")

	// test unknown tag
	testExecuteFunc(t, "{unknown}", "zz")
	testExecuteFunc(t, "{foo}q{unexpected}{missing}bar{foo}", "xxxxqzzzzbarxxxx")
}

func testExecuteFunc(t *testing.T, template, expectedOutput string) {
	var bb bytes.Buffer
	ExecuteFunc(template, "{", "}", &bb, func(w io.Writer, tag string) (int, error) {
		if tag == "foo" {
			return w.Write([]byte("xxxx"))
		}
		return w.Write([]byte("zz"))
	})

	output := string(bb.Bytes())
	if output != expectedOutput {
		t.Fatalf("unexpected output for template=%q: %q. Expected %q", template, output, expectedOutput)
	}
}

func TestExecute(t *testing.T) {
	testExecute(t, "", "")
	testExecute(t, "a", "a")
	testExecute(t, "abc", "abc")
	testExecute(t, "{foo}", "xxxx")
	testExecute(t, "a{foo}", "axxxx")
	testExecute(t, "{foo}a", "xxxxa")
	testExecute(t, "a{foo}bc", "axxxxbc")
	testExecute(t, "{foo}{foo}", "xxxxxxxx")
	testExecute(t, "{foo}bar{foo}", "xxxxbarxxxx")

	// unclosed tag
	testExecute(t, "{unclosed", "{unclosed")
	testExecute(t, "{{unclosed", "{{unclosed")
	testExecute(t, "{un{closed", "{un{closed")

	// test unknown tag
	testExecute(t, "{unknown}", "")
	testExecute(t, "{foo}q{unexpected}{missing}bar{foo}", "xxxxqbarxxxx")
}

func testExecute(t *testing.T, template, expectedOutput string) {
	var bb bytes.Buffer
	Execute(template, "{", "}", &bb, map[string]interface{}{"foo": "xxxx"})
	output := string(bb.Bytes())
	if output != expectedOutput {
		t.Fatalf("unexpected output for template=%q: %q. Expected %q", template, output, expectedOutput)
	}
}

func TestExecuteString(t *testing.T) {
	testExecuteString(t, "", "")
	testExecuteString(t, "a", "a")
	testExecuteString(t, "abc", "abc")
	testExecuteString(t, "{foo}", "xxxx")
	testExecuteString(t, "a{foo}", "axxxx")
	testExecuteString(t, "{foo}a", "xxxxa")
	testExecuteString(t, "a{foo}bc", "axxxxbc")
	testExecuteString(t, "{foo}{foo}", "xxxxxxxx")
	testExecuteString(t, "{foo}bar{foo}", "xxxxbarxxxx")

	// unclosed tag
	testExecuteString(t, "{unclosed", "{unclosed")
	testExecuteString(t, "{{unclosed", "{{unclosed")
	testExecuteString(t, "{un{closed", "{un{closed")

	// test unknown tag
	testExecuteString(t, "{unknown}", "")
	testExecuteString(t, "{foo}q{unexpected}{missing}bar{foo}", "xxxxqbarxxxx")
}

func testExecuteString(t *testing.T, template, expectedOutput string) {
	output := ExecuteString(template, "{", "}", map[string]interface{}{"foo": "xxxx"})
	if output != expectedOutput {
		t.Fatalf("unexpected output for template=%q: %q. Expected %q", template, output, expectedOutput)
	}
}

func expectPanic(t *testing.T, f func()) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("missing panic")
		}
	}()
	f()
}
