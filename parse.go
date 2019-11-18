package jsontemplate

import (
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/Volumental/jsontemplate/internal/parse"
	"k8s.io/client-go/util/jsonpath"
)

type builder struct {
	funcs FunctionMap
}

func (b *builder) buildObject(o *parse.Object) object {
	var res = object{}
	for _, f := range o.Fields {
		res[f.Key] = field{
			value:      b.buildValue(&f.Value),
			annotation: f.Annotation,
		}
	}
	return res
}

func (b *builder) buildQuery(q *string) query {
	var jp = jsonpath.New("template-query")
	if err := jp.Parse(fmt.Sprintf("{%s}", *q)); err != nil {
		panic(fmt.Errorf("jsontemplate: invalid jsonpath: %v", err))
	}
	return query{expression: jp}
}

func (b *builder) buildFunction(node *parse.Function) template {
	var res = function{
		name: node.Name,
		args: make([]template, len(node.Args)),
	}
	var ok bool
	if res.function, ok = b.funcs[node.Name]; !ok {
		panic(fmt.Errorf("jsontemplate: no such function: %s", node.Name))
	} else if fun := reflect.ValueOf(res.function); fun.Kind() != reflect.Func {
		panic(fmt.Sprintf("%s is not a function", node.Name)) // Actual panic.
	}
	// TODO(josef): Verify that the signature has a single return value.
	for i, v := range node.Args {
		res.args[i] = b.buildValue(&v)
	}
	return res
}

func (b *builder) buildValue(v *parse.Value) template {
	switch {
	case v.String != nil:
		return stringConstant(*v.String)
	case v.Number != nil:
		return numberConstant(*v.Number)
	case v.Bool != nil:
		return boolConstant(*v.Bool)
	case v.Null:
		return nullConstant{}
	case v.Object != nil:
		return b.buildObject(v.Object)
	case v.Array != nil:
		var res = make(array, len(v.Array))
		for i, v := range v.Array {
			res[i] = b.buildValue(&v)
		}
		return res
	case v.Generator != nil:
		return generator{
			over:     b.buildQuery(&v.Generator.Range),
			template: b.buildValue(&v.Generator.SubTemplate),
		}
	case v.Extractor != nil:
		return b.buildQuery(v.Extractor)
	case v.Function != nil:
		return b.buildFunction(v.Function)
	default:
		panic("unhandled case")
	}
}

// FunctionMap is a map of named functions that may be called from within a
// template.
type FunctionMap map[string]interface{}

// ParseString works like Parse, but takes a string as input rather than a
// Reader.
func ParseString(s string, funcs FunctionMap) (*Template, error) {
	return Parse(strings.NewReader(s), funcs)
}

// Parse reads a template definition from a textual format.
//
// The template definition format has a grammar similar to a regular JSON value,
// with some additions that allow interpolation and transformation. These are
// outlined below. As a special case, a regular JSON file is interpreted as a
// template taking no inputs.
//
// Queries
//
// A template can pull data from the input data using JSONPath expressions (see
// https://goessner.net/articles/JsonPath/). Each query expression can evaluate
// to either a JSON value, which will then be inserted in its place, or a range
// of values, which will yield an array.
//     {
//         "single_x": $.foo.x,
//         "array_of_all_x_recursively": $..x
//     }
//
// Generators
//
// Generators is a mechanism that allows repeating a sub-template within an
// array. It takes the format of the keyword `range` followed an array
// expression and a sub-template, as such:
//     range $.some_array_of_xy[*] [
//         { "foo": $.x, "bar": $.y }
//     ]
// Inside the sub-template, the `$` refers to the root of each element in the
// input array. Thus, the example above maps the fields `x` and `y` to `foo` and
// `bar`, respectively, in the objects in the output array.
//
// Functions
//
// Regular Go functions can be exposed to and called from within the template.
// This allows more complex transformations. For example, by adding a Coalesce
// function that returns the first non-nil argument, fallback defaults can be
// introduced as follows:
//     { "foo": Coalesce($.some_input, "default value") }
//
// Field annotations
//
// Members in objects can be prefixed with an annotation, starting with an `@`
// character.
//     {
//         @deprecated "field": "value"
//     }
// Annotations are stripped from the final output. However, future versions of
// this library may allow some control over how annotated fields are rendered.
// For example, it could be used to elide deprecated fields.
//
// Other differences
//
// To help users clarify and document intentions, the template format allows
// comments in the template definition. These are preceded by a `#` character.
// Anything from and including this character to the end of the line will be
// ignored when parsing the template.
//
// Finally, unlike JSON, the template format tolerates trailing commas after the
// last element of objects and arrays.
func Parse(r io.Reader, funcs FunctionMap) (t *Template, err error) {
	var ast parse.Template
	if err := parse.Parser.Parse(r, &ast); err != nil {
		return nil, fmt.Errorf("jsontemplate: parse error: %v", err)
	}
	// We handle errors in the recurstion using panics that stop here.
	// This is similar to how the json library does it.
	defer func() {
		var r = recover()
		switch r := r.(type) {
		case nil: // Nothing.
		case error:
			err = r
		default:
			panic(fmt.Sprintf("jsontemplate: panic during parsing: %v", r))
		}
	}()
	var b = builder{funcs: funcs}
	return &Template{definition: b.buildValue(&ast.Root)}, nil
}
