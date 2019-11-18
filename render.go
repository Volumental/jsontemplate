package jsontemplate

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"runtime"
	"runtime/debug"

	"k8s.io/client-go/util/jsonpath"
)

// MissingKeyPolicy discates how the rendering should handle references to keys
// that are missing from the input data.
type MissingKeyPolicy int

const (
	// NullOnMissing makes query expressions referencing missing values evaluate
	// to null.
	NullOnMissing MissingKeyPolicy = iota

	// ErrorOnMissing causes the renderer to return an error if a query
	// expression references a missing value.
	ErrorOnMissing
)

type options struct {
	MissingKeys MissingKeyPolicy
}

type template interface {
	interpolate(data interface{}, opt options) interface{}
}

type stringConstant string
type boolConstant bool
type numberConstant float64
type nullConstant struct{}

type object map[string]field

type field struct {
	value      template
	annotation string
}

type array []template

type query struct {
	expression *jsonpath.JSONPath
}

type generator struct {
	over     query
	template template
}

type function struct {
	name     string      // For giving informative error messages.
	function interface{} // Must be a function with a single return value.
	args     []template
}

func (s stringConstant) interpolate(data interface{}, opt options) interface{} { return string(s) }
func (b boolConstant) interpolate(data interface{}, opt options) interface{}   { return bool(b) }
func (n numberConstant) interpolate(data interface{}, opt options) interface{} { return float64(n) }
func (n nullConstant) interpolate(data interface{}, opt options) interface{}   { return nil }

func (o object) interpolate(data interface{}, opt options) interface{} {
	var res = make(map[string]interface{})
	for name, field := range o {
		res[name] = field.value.interpolate(data, opt)
	}
	return res
}

func (a array) interpolate(data interface{}, opt options) interface{} {
	var res = make([]interface{}, len(a))
	for i, templ := range a {
		res[i] = templ.interpolate(data, opt)
	}
	return res
}

func (q query) interpolate(data interface{}, opt options) interface{} {
	if data == nil {
		switch opt.MissingKeys {
		case NullOnMissing:
			return nil
		case ErrorOnMissing:
			panic(fmt.Errorf("jsontemplate: cannot execute query, input is null"))
		}
	}
	q.expression.AllowMissingKeys(opt.MissingKeys == NullOnMissing)
	var hits, err = q.expression.FindResults(data)
	if err != nil {
		panic(fmt.Errorf("jsontemplate: error executing query: %v", err))
	}
	switch len(hits[0]) {
	case 0:
		return nil
	case 1:
		return hits[0][0].Interface()
	default: // Many, make an array.
		var res = make([]interface{}, len(hits[0]))
		for i, v := range hits[0] {
			res[i] = v.Interface()
		}
		return res
	}
}

func (g generator) interpolate(data interface{}, opt options) interface{} {
	if data == nil {
		switch opt.MissingKeys {
		case NullOnMissing:
			return nil
		case ErrorOnMissing:
			panic(fmt.Errorf("jsontemplate: cannot generate array, input is null"))
		}
	}
	g.over.expression.AllowMissingKeys(opt.MissingKeys == NullOnMissing)
	var hits, err = g.over.expression.FindResults(data)
	if err != nil {
		panic(err)
	}
	var res = make([]interface{}, len(hits[0]))
	for i, v := range hits[0] {
		var inner interface{}
		if v.IsValid() {
			inner = v.Interface()
		}
		res[i] = g.template.interpolate(inner, opt)
	}
	return res
}

func (f function) interpolate(data interface{}, opt options) interface{} {
	var args = make([]reflect.Value, len(f.args))
	var ftype = reflect.TypeOf(f.function)
	for i, templ := range f.args {
		// The Call function of the reflect library doesn't handle nil
		// interfaces the way we want (it will create an invalid Value) so we
		// need some special handling of nil arguments here.
		var val = templ.interpolate(data, opt)
		var expected reflect.Type
		if ftype.IsVariadic() && i >= ftype.NumIn()-1 {
			// Variadic arguments are represented as a final array argument.
			expected = ftype.In(ftype.NumIn() - 1).Elem()
		} else {
			expected = ftype.In(i)
		}
		if val == nil {
			switch expected.Kind() {
			case reflect.Chan, reflect.Func, reflect.Interface,
				reflect.Map, reflect.Ptr, reflect.Slice:
				// These types can be assigned 'nil'.
				args[i] = reflect.Zero(expected)
			default:
				panic(fmt.Errorf("jsontemplate: cannot pass nil as argument %d of %s, expecting %v", i+1, f.name, expected))
			}
			continue
		}
		// If the value is not nil, we check that it matches the argument of
		// the function, to give a more informative error message than Call
		// would give us.
		var rval = reflect.ValueOf(val)
		var actual = rval.Type()
		if expected.Kind() == reflect.Ptr && actual.Kind() != reflect.Ptr {
			// If the function wants a pointer and we have a value, we make a
			// pointer to a copy and pass that. This allows declaring
			// functions taking nullable arguments by means of pointers.
			var pointer = reflect.New(actual)
			pointer.Elem().Set(rval)
			actual = pointer.Type()
			rval = pointer
		}
		if !actual.AssignableTo(expected) {
			panic(fmt.Errorf("jsontemplate: cannot pass %v (%v) as argument %d of %s, expecting %v", val, reflect.TypeOf(val), i+1, f.name, expected))
		}
		args[i] = rval
	}
	// The parser should already have asserted that this is valid, yielding a
	// nicer and earlier panic than we could do here, so no extra checks here.
	return reflect.ValueOf(f.function).Call(args)[0].Interface()
}

// Template represents a transformation from one JSON-like structure to another.
type Template struct {
	definition template

	// MissingKeys defines the policy for how to handle keys referenced in
	// queries that are absent in the input data. The default is to substitute
	// them with null.
	MissingKeys MissingKeyPolicy
}

// Render generates a JSON-like structure based on the template definition,
// using the passed `data` as source data for query expressions.
func (t *Template) Render(data interface{}) (res interface{}, err error) {
	// We handle errors in the recurstion using panics that stop here.
	// This is similar to how the json library does it.
	defer func() {
		var r = recover()
		switch r := r.(type) {
		case nil: // Nothing.
		case runtime.Error:
			err = fmt.Errorf("jsontemplate: while rendering: %v\n%s", r, debug.Stack())
		case error:
			fmt.Println("was here!")
			err = r
		default:
			err = fmt.Errorf("jsontemplate: panic during interpolation: %v", r)
		}
	}()
	res = t.definition.interpolate(data, options{MissingKeys: t.MissingKeys})
	return
}

// RenderJSON generates JSON output based on the template definition, using JSON
// input as source data for query expressions.
//
// Note that RenderJSON will only attempt to read a single JSON value from the
// input stream. If the stream contains multiple white-space delimited JSON
// values that you wish to transform, RenderJSON can be called repeatedly with
// the same arguments.
//
// If EOF is encountered on the input stream before the start of a JSON value,
// RenderJSON will return io.EOF.
func (t *Template) RenderJSON(out io.Writer, in io.Reader) error {
	var dec = json.NewDecoder(in)
	var input interface{}
	if err := dec.Decode(&input); err != nil {
		if err == io.EOF {
			return err
		}
		return fmt.Errorf("jsontemplate: invalid input: %v", err)
	}
	var output, err = t.Render(input)
	if err != nil {
		return err
	}
	var enc = json.NewEncoder(out)
	if err := enc.Encode(&output); err != nil {
		return fmt.Errorf("jsontemplate: error writing output: %v", err)
	}
	return nil
}
