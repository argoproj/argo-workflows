package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

var tracingHeaderTmpl = template.Must(template.New("header").Parse(`{{.Banner}}
    //
    //go:generate go run ./builder --tracingGo {{.Filename}}
    package telemetry

    import (
    	"context"

    	"go.opentelemetry.io/otel/trace"
        sdktrace "go.opentelemetry.io/otel/sdk/trace"
	    "go.opentelemetry.io/otel/attribute"

	    "github.com/argoproj/argo-workflows/v3/util/logging"
    )

func AllNoParentSpans() Spans {
    return append(Root, AnyParent...)
}

`))

var spanTmpl = template.Must(template.New("span").Parse(`var {{.TypeName}} = Span{
    name: "{{.RuntimeName}}",
`))

func createTracingGo(filename string, spans *spansList, attributes *attributesList) error {
	err := writeTracingGo(filename, spans, attributes)
	if err != nil {
		return err
	}
	return goFmtFile(filename)
}

func writeTracingGo(filename string, spans *spansList, attributes *attributesList) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	err = tracingHeaderTmpl.Execute(f, map[string]string{"Banner": generatedBanner, "Filename": filepath.Base(filename)})
	if err != nil {
		return err
	}

	generateRoot(f, spans)
	generateAnyParent(f, spans)
	// Generate option types and functions for metrics with optional attributes
	for _, span := range *spans {
		if hasOptionalAttributes(&span.Attributes) {
			optionTypeName := fmt.Sprintf("%s%sOption", span.Name, spanType)
			err = generateSpanOption(f, &span.common, optionTypeName, attributes)
			if err != nil {
				return err
			}
		}
	}

	// Build a map of parent -> children for generating children fields
	childrenMap := buildChildrenMap(spans)

	for _, span := range *spans {
		if span.DocsOnly {
			continue
		}
		err = spanTmpl.Execute(f, map[string]string{
			"TypeName":    span.typeName(),
			"RuntimeName": toLowerCamelCase(span.Name),
		})
		if err != nil {
			return err
		}
		// Generate children field if this span has children
		if children, ok := childrenMap[span.Name]; ok && len(children) > 0 {
			fmt.Fprintf(f, "children: []*Span{")
			for i, child := range children {
				if i > 0 {
					fmt.Fprintf(f, ", ")
				}
				fmt.Fprintf(f, "&Span%s", child)
			}
			fmt.Fprintf(f, "},\n")
		}
		if len(span.Attributes) > 0 {
			fmt.Fprintf(f, "\tattributes: []BuiltinAttribute{\n")
			for _, attrib := range span.Attributes {
				fmt.Fprintf(f, "\t\t{\n\t\t\tname: %s,\n", attrib.AttribName())
				if attrib.Optional {
					fmt.Fprintf(f, "\t\t\toptional: true,\n")
				}
				fmt.Fprintf(f, "\t\t},\n")
			}
			fmt.Fprintf(f, "\t},\n")
		}

		fmt.Fprintf(f, "}\n\n")
		err = generateStart(f, &span, attributes)
		if err != nil {
			return err
		}

	}

	return nil
}

// buildChildrenMap builds a map of parent name -> list of child names
func buildChildrenMap(spans *spansList) map[string][]string {
	childrenMap := make(map[string][]string)
	for _, span := range *spans {
		if span.DocsOnly {
			continue
		}
		for _, parent := range span.Parents {
			childrenMap[parent] = append(childrenMap[parent], span.Name)
		}
	}
	return childrenMap
}

var spanStartTmpl = template.Must(template.New("start").Funcs(funcMap).Parse(`// {{.MethodName}} starts a {{.DisplayName}} span
func (t *Tracing) {{.MethodName}}({{.Params}}) (context.Context, trace.Span) {
    parent := trace.SpanFromContext(ctx)
    if roParent, ok := parent.(sdktrace.ReadOnlySpan); ok {
        parentName := roParent.Name()
{{if .Parents -}}
        if {{range $i, $p := .Parents}}{{if $i}} && {{end}}parentName != "{{$p}}"{{end}} {
            logging.RequireLoggerFromContext(ctx).WithFields(logging.Fields{"startMethod": "{{.MethodName}}", "expectedParents": "{{.ParentsStr}}", "actualParent": parentName}).Error(ctx, "incorrect trace parentage")
        }
{{else -}}
        logging.RequireLoggerFromContext(ctx).WithFields(logging.Fields{"startMethod": "{{.MethodName}}", "actualParent": parentName}).Info(ctx, "trace parent") // TODO remove
{{end -}}
    }
    {{.Attribs}}
    return {{.Call}}
}
`))

func spanKind(kind string) string {
	// kind is already validated by here
	if kind == "" {
		kind = "Internal"
	}
	return fmt.Sprintf("trace.WithSpanKind(trace.SpanKind%s)", kind)
}

func (s *span) typeName() string {
	return fmt.Sprintf("%s%s", spanType, s.Name)
}

func generateStart(f io.Writer, s *span, attributes *attributesList) error {
	methodName := fmt.Sprintf("Start%s", s.Name)

	params := buildParameterList([]string{"ctx context.Context"}, &s.common, spanType, attributes)
	methodCall := fmt.Sprintf("t.tracer.Start(ctx, \"%s\"", toLowerCamelCase(s.Name)) // TODO best practice

	// Convert parents to their runtime (lowerCamelCase) names
	var runtimeParents []string
	for _, p := range s.Parents {
		runtimeParents = append(runtimeParents, toLowerCamelCase(p))
	}

	return spanStartTmpl.Execute(f, map[string]any{
		"MethodName":  methodName,
		"DisplayName": s.displayName(),
		"Params":      strings.Join(params, ", "),
		"Attribs":     buildTracingWithAttributes(&s.Attributes, attributes),
		"Call":        buildTracingCall(methodCall, &s.common, spanKind(s.Kind)),
		"Parents":     runtimeParents,
		"ParentsStr":  strings.Join(runtimeParents, ", "),
	})
}

func buildTracingWithAttributes(allowed *allowedAttributeList, attributes *attributesList) string {
	requiredAttrs := getRequiredAttributes(allowed)
	var b strings.Builder

	if len(requiredAttrs) > 0 || hasOptionalAttributes(allowed) {
		b.WriteString("\tattribs := []attribute.KeyValue")
		if len(requiredAttrs) > 0 {
			b.WriteString(" {\n")
			for _, allowedAttr := range requiredAttrs {
				attr := getAttribByName(allowedAttr.Name, attributes)
				if attr == nil {
					panic("Failed to find attribute from allowed list, should have failed validation")
				}
				paramName := toLowerCamelCase(allowedAttr.Name)
				fmt.Fprintf(&b, "\t\t%s(%s, %s),\n", attr.kvConstructor(), allowedAttr.AttribName(), paramName)
			}
			b.WriteString("\t}\n")
		} else {
			b.WriteString("\n")
		}

		if hasOptionalAttributes(allowed) {
			b.WriteString("\tfor _, opt := range opts {\n")
			b.WriteString("\t\topt(&attribs)\n")
			b.WriteString("\t}\n")
		}
	}

	return b.String()
}

func buildTracingCall(methodCall string, c *common, kind string) string {
	if len(getRequiredAttributes(&c.Attributes)) > 0 || hasOptionalAttributes(&c.Attributes) {
		return fmt.Sprintf("\t%s, trace.WithAttributes(attribs...), %s)", methodCall, kind)
	}
	return fmt.Sprintf("\t%s, %s)", methodCall, kind)
}

var spanOptionTmpl = template.Must(template.New("optionType").Parse(`// {{.OptionTypeName}} is a functional option for configuring optional attributes on {{.SpanName}}
type {{.OptionTypeName}} func(*[]attribute.KeyValue)

`))

var spanOptionFuncTmpl = template.Must(template.New("optionFunc").Parse(`// {{.FuncName}} sets the {{.DisplayName}} attribute
func {{.FuncName}}({{.ParamName}} {{.ParamType}}) {{.OptionTypeName}} {
	return func(a *[]attribute.KeyValue) {
		*a = append(*a, {{.AttrConstructor}}(
			Attrib{{.AttrName}},
			{{.ParamName}},
		))
	}
}

`))

func generateSpanOption(f io.Writer, c *common, optionTypeName string, attributes *attributesList) error {
	// Generate the option type
	err := spanOptionTmpl.Execute(f, map[string]string{
		"OptionTypeName": optionTypeName,
		"SpanName":       c.Name,
	})
	if err != nil {
		return err
	}

	// Generate option functions for each optional attribute
	for _, attr := range c.Attributes {
		if attr.Optional {
			attribDef := getAttribByName(attr.Name, attributes)
			paramName := toLowerCamelCase(attr.Name)
			paramType := attribDef.attrType()
			funcName := fmt.Sprintf("With%s", attr.Name)

			err = spanOptionFuncTmpl.Execute(f, map[string]string{
				"FuncName":        funcName,
				"DisplayName":     attribDef.displayName(),
				"ParamName":       paramName,
				"ParamType":       paramType,
				"OptionTypeName":  optionTypeName,
				"AttrName":        attr.Name,
				"AttrConstructor": getAttribByName(attr.Name, attributes).kvConstructor(),
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// findSpansInTrace returns all spans that are descendants of the given root span
func findSpansInTrace(rootName string, spans *spansList) spansList {
	var result spansList

	// Find the root span first
	for _, s := range *spans {
		if s.Name == rootName {
			result = append(result, s)
			break
		}
	}

	// Iteratively find children until no more are found
	for {
		foundNew := false
		for _, s := range *spans {
			if len(s.Parents) == 0 {
				continue
			}
			// Check if any parent is already in result
			parentInResult := false
			alreadyAdded := false
			for _, r := range result {
				for _, p := range s.Parents {
					if r.Name == p {
						parentInResult = true
					}
				}
				if r.Name == s.Name {
					alreadyAdded = true
				}
			}
			if parentInResult && !alreadyAdded {
				result = append(result, s)
				foundNew = true
			}
		}
		if !foundNew {
			break
		}
	}

	return result
}

func generateAnyParent(f io.Writer, spans *spansList) {
	generateSpansList(f, func(s span) bool { return s.AnyParent == true }, "AnyParent", spans)
}

func generateRoot(f io.Writer, spans *spansList) {
	generateSpansList(f, func(s span) bool { return s.Root == true }, "Root", spans)
}

func generateSpansList(f io.Writer, cond func(span) bool, name string, spans *spansList) {
	fmt.Fprintf(f, "var %s = Spans {\n", name)
	for _, span := range *spans {
		if cond(span) {
			fmt.Fprintf(f, "\t&%s,\n", span.typeName())
		}
	}
	fmt.Fprintf(f, "}\n\n")
}
