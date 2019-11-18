package jsontemplate

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
)

func Test_stringConstant_interpolate(t *testing.T) {
	tests := []struct {
		name string
		s    stringConstant
		want interface{}
	}{
		{
			name: "empty",
			s:    "",
			want: "",
		},
		{
			name: "not empty",
			s:    "foo",
			want: "foo",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.interpolate(nil, options{}); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("stringConstant.interpolate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_boolConstant_interpolate(t *testing.T) {
	tests := []struct {
		name string
		b    boolConstant
		want interface{}
	}{
		{
			name: "true",
			b:    true,
			want: true,
		},
		{
			name: "false",
			b:    false,
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.b.interpolate(nil, options{}); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("boolConstant.interpolate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_numberConstant_interpolate(t *testing.T) {
	tests := []struct {
		name string
		n    numberConstant
		want interface{}
	}{
		{
			name: "arbitrary",
			n:    4711,
			want: float64(4711),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.n.interpolate(nil, options{}); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("numberConstant.interpolate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_nullConstant_interpolate(t *testing.T) {
	var n nullConstant
	if n.interpolate(nil, options{}) != nil {
		t.Errorf("nullConstant.interpolate() != nil")
	}
}

func Test_object_interpolate(t *testing.T) {
	type args struct {
		data interface{}
		opt  options
	}
	tests := []struct {
		name string
		o    object
		args args
		want interface{}
	}{
		{
			name: "empty",
			o:    object{},
			want: map[string]interface{}{},
		},
		{
			name: "simple",
			o: object{
				"x": field{value: numberConstant(123)},
			},
			want: map[string]interface{}{
				"x": float64(123),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.o.interpolate(tt.args.data, tt.args.opt); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("object.interpolate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_array_interpolate(t *testing.T) {
	tests := []struct {
		name string
		a    array
		want interface{}
	}{
		{
			name: "empty",
			a:    array{},
			want: []interface{}{},
		},
		{
			name: "not empty",
			a: array{
				numberConstant(1),
				numberConstant(2),
				boolConstant(true),
				stringConstant("foo"),
			},
			want: []interface{}{float64(1), float64(2), true, "foo"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.a.interpolate(nil, options{}); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("array.interpolate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_query_interpolate(t *testing.T) {
	type args struct {
		data interface{}
		opt  options
	}
	tests := []struct {
		name       string
		expression string
		args       args
		want       interface{}
		wantPanic  bool
	}{
		{
			name:       "trivial",
			expression: "$",
			args:       args{data: 123},
			want:       123,
		},
		{
			name:       "array",
			expression: "$.*",
			args: args{
				data: struct{ X, Y int }{X: 1, Y: 1},
			},
			want: []interface{}{1, 1},
		},
		{
			name:       "nested",
			expression: "$.X.Y",
			args: args{
				data: struct{ X interface{} }{X: struct{ Y int }{Y: 123}},
			},
			want: 123,
		},
		{
			name:       "search",
			expression: "$..Y",
			args: args{
				data: struct{ X interface{} }{X: struct{ Y int }{Y: 123}},
			},
			want: 123,
		},
		{
			name:       "missing chain",
			expression: "$.a.b.c",
			args: args{
				data: map[string]int{"a": 123},
			},
			want: nil,
		},
		{
			name:       "missing chain error",
			expression: "$.a.b.c",
			args: args{
				data: map[string]int{"a": 123},
				opt:  options{MissingKeys: ErrorOnMissing},
			},
			want:      nil,
			wantPanic: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := query{
				expression: mustParseJSONPath(tt.expression),
			}
			if tt.wantPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("query.interpolate() did not panic as expected")
					}
				}()
			}
			if got := q.interpolate(tt.args.data, tt.args.opt); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("query.interpolate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_generator_interpolate(t *testing.T) {
	type args struct {
		data interface{}
		opt  options
	}
	tests := []struct {
		name      string
		g         generator
		args      args
		want      interface{}
		wantPanic bool
	}{
		{
			name: "trivial (no input)",
			g: generator{
				over:     query{expression: mustParseJSONPath("$")},
				template: query{expression: mustParseJSONPath("$")},
			},
			want: nil,
		},
		{
			name: "trivial (no input, error)",
			g: generator{
				over:     query{expression: mustParseJSONPath("$")},
				template: query{expression: mustParseJSONPath("$")},
			},
			args:      args{opt: options{MissingKeys: ErrorOnMissing}},
			want:      nil,
			wantPanic: true,
		},
		{
			name: "trivial (input)",
			g: generator{
				over:     query{expression: mustParseJSONPath("$")},
				template: query{expression: mustParseJSONPath("$")},
			},
			args: args{
				data: "foo",
			},
			want: []interface{}{"foo"},
		},
		{
			name: "list",
			g: generator{
				over:     query{expression: mustParseJSONPath("$.*")},
				template: query{expression: mustParseJSONPath("$")},
			},
			args: args{
				data: []string{"foo", "bar", "baz"},
			},
			want: []interface{}{"foo", "bar", "baz"},
		},
		{
			name: "search",
			g: generator{
				over:     query{expression: mustParseJSONPath("$..Inner")},
				template: query{expression: mustParseJSONPath("$")},
			},
			args: args{
				data: []struct {
					Inner string
				}{
					{Inner: "hello"},
					{Inner: "world"},
				},
			},
			want: []interface{}{"hello", "world"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("query.interpolate() did not panic as expected")
					}
				}()
			}
			if got := tt.g.interpolate(tt.args.data, tt.args.opt); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("generator.interpolate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_function_interpolate(t *testing.T) {
	var hello = func() string { return "hello" }
	var fancy = func(n float64, np *float64, bs []byte, i interface{}, more ...interface{}) string { return "ok" }
	tests := []struct {
		name      string
		f         function
		want      interface{}
		wantPanic bool
	}{
		{
			name: "nullary",
			f: function{
				name:     "hello",
				function: hello,
			},
			want: "hello",
		},
		{
			name: "binary",
			f: function{
				name:     "compare",
				function: strings.Compare,
				args: []template{
					stringConstant("foo"),
					stringConstant("bar"),
				},
			},
			want: 1,
		},
		{
			name: "fancy",
			f: function{
				name:     "fancy",
				function: fancy,
				args: []template{
					numberConstant(1),
					nullConstant{},
					nullConstant{},
					nullConstant{},
				},
			},
			want: "ok",
		},
		{
			name: "null to float",
			f: function{
				name:     "fancy",
				function: fancy,
				args: []template{
					nullConstant{},
					nullConstant{},
					nullConstant{},
					nullConstant{},
				},
			},
			wantPanic: true,
		},
		{
			name: "number to interface",
			f: function{
				name:     "fancy",
				function: fancy,
				args: []template{
					numberConstant(1),
					nullConstant{},
					nullConstant{},
					numberConstant(1),
				},
			},
			want: "ok",
		},
		{
			name: "variadic",
			f: function{
				name:     "fancy",
				function: fancy,
				args: []template{
					numberConstant(1),
					nullConstant{},
					nullConstant{},
					numberConstant(1),
					numberConstant(2),
					nullConstant{},
					numberConstant(3),
				},
			},
			want: "ok",
		},
		{
			name: "number to pointer",
			f: function{
				name:     "fancy",
				function: fancy,
				args: []template{
					numberConstant(1),
					numberConstant(1),
					nullConstant{},
					nullConstant{},
				},
			},
			want: "ok",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("query.interpolate() did not panic as expected")
					}
				}()
			}
			if got := tt.f.interpolate(nil, options{}); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("array.interpolate() = %v, want %v", got, tt.want)
			}
		})
	}
}

var testData = map[string]interface{}{
	"string": "hello world",
	"number": 123,
	"bool":   true,
	"nil":    nil,
	"object": map[string]interface{}{
		"first":  "hello",
		"second": "world",
	},
	"array": []interface{}{
		"text",
		123,
		true,
		nil,
	},
	"nested": map[string]interface{}{
		"first": map[string]interface{}{
			"a": 123,
			"b": true,
		},
		"second": map[string]interface{}{
			"a": 321,
			"b": true,
		},
	},
	"array_of_objects": []interface{}{
		map[string]interface{}{
			"n": 123,
		},
		map[string]interface{}{
			"n": 321,
		},
	},
}

func TestTemplate_Render(t *testing.T) {
	var funcMap = map[string]interface{}{
		"to_upper": strings.ToUpper,
	}
	type args struct {
		data interface{}
		opt  options
	}
	tests := []struct {
		name       string
		definition string
		args       args
		wantRes    interface{}
		wantErr    bool
	}{
		{
			name:       "trivial",
			definition: "1",
			wantRes:    float64(1),
		},
		{
			name:       "static",
			definition: `{"foo": "hello", "bar": [1, 2, 3]}`,
			wantRes: map[string]interface{}{
				"foo": "hello",
				"bar": []interface{}{float64(1), float64(2), float64(3)},
			},
		},
		{
			name:       "query field",
			definition: `$.number`,
			wantRes:    123,
			args:       args{data: testData},
		},
		{
			name:       "query object",
			definition: `$.object`,
			wantRes: map[string]interface{}{
				"first":  "hello",
				"second": "world",
			},
			args: args{data: testData},
		},
		{
			name:       "query recursive",
			definition: `$..b`,
			wantRes:    []interface{}{true, true},
			args:       args{data: testData},
		},
		{
			name: "composed",
			definition: `
				{
					"foo": $.array[2:],
					"bar": range $.array_of_objects[*] [
						{ "x": $.n }
					],
					"greeting": to_upper($.string)
				}
			`,
			wantRes: map[string]interface{}{
				"foo": []interface{}{true, nil},
				"bar": []interface{}{
					map[string]interface{}{"x": 123},
					map[string]interface{}{"x": 321},
				},
				"greeting": "HELLO WORLD",
			},
			args: args{data: testData},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			templ, err := ParseString(tt.definition, funcMap)
			if err != nil {
				panic(fmt.Sprintf("broken test: %v", err))
			}
			templ.MissingKeys = tt.args.opt.MissingKeys
			gotRes, err := templ.Render(tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Template.Render() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotRes, tt.wantRes) {
				t.Errorf("Template.Render() = %v, want %v", gotRes, tt.wantRes)
			}
		})
	}
}

func TestTemplate_RenderJSON(t *testing.T) {
	var funcMap = map[string]interface{}{
		"to_upper": strings.ToUpper,
	}
	tests := []struct {
		name       string
		definition string
		input      string
		wantOut    string
		wantErr    bool
	}{
		{
			name: "composed",
			definition: `
				{
					"foo": $.array[2:],
					"bar": range $.array_of_objects[*] [
						{ "x": $.n }
					],
					"greeting": to_upper($.string)
				}
			`,
			input: `
				{
					"array": [1, 2, "hello", 3],
					"array_of_objects": [
						{ "n": 123, "m": 321 },
						{ "n": true, "m": false },
						{ "n": "A", "m": "B" }
					],
					"string": "hello world"
				}
			`,
			wantOut: `{"bar":[{"x":123},{"x":true},{"x":"A"}],"foo":["hello",3],"greeting":"HELLO WORLD"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			templ, err := ParseString(tt.definition, funcMap)
			if err != nil {
				panic(fmt.Sprintf("broken test: %v", err))
			}
			out := &bytes.Buffer{}
			if err := templ.RenderJSON(out, strings.NewReader(tt.input)); (err != nil) != tt.wantErr {
				t.Errorf("Template.RenderJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotOut := out.String(); strings.TrimSpace(gotOut) != tt.wantOut {
				t.Errorf("Template.RenderJSON() = %v, want %v", gotOut, tt.wantOut)
			}
		})
	}
}

func ExampleTemplate_RenderJSON() {
	const input = `{ "snakeCase": 123 }`
	template, _ := ParseString(`{ "CamelCase": $.snakeCase }`, nil)
	template.RenderJSON(os.Stdout, strings.NewReader(input))
	os.Stdout.Sync()
	// Output: {"CamelCase":123}
}
