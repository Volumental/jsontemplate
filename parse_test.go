package jsontemplate

import (
	"fmt"
	"reflect"
	"testing"

	"k8s.io/client-go/util/jsonpath"
)

func mustParseJSONPath(s string) *jsonpath.JSONPath {
	var jp = jsonpath.New("template-query")
	if err := jp.Parse(fmt.Sprintf("{%s}", s)); err != nil {
		panic("tests broken")
	}
	return jp
}

func TestParseString(t *testing.T) {
	tests := []struct {
		name       string
		definition string
		wantOut    *Template
		wantErr    bool
	}{
		{
			name:       "empty",
			definition: "",
			wantErr:    true,
		},
		{
			name:       "number",
			definition: "1",
			wantOut: &Template{
				definition: numberConstant(1),
			},
		},
		{
			name:       "string",
			definition: `"foo"`,
			wantOut: &Template{
				definition: stringConstant("foo"),
			},
		},
		{
			name:       "array",
			definition: `[1, 2, true, "foo"]`,
			wantOut: &Template{
				definition: array{
					numberConstant(1),
					numberConstant(2),
					boolConstant(true),
					stringConstant("foo"),
				},
			},
		},
		{
			name: "object",
			definition: `
				{
					"foo": true,
					"bar": 123, # Traliing comma is allowed, unlike JSON.
				}
			`,
			wantOut: &Template{
				definition: object{
					"foo": field{value: boolConstant(true)},
					"bar": field{value: numberConstant(123)},
				},
			},
		},
		{
			name:       "annotation",
			definition: `{@deprecated "foo": true}`,
			wantOut: &Template{
				definition: object{
					"foo": field{
						value:      boolConstant(true),
						annotation: "deprecated",
					},
				},
			},
		},
		// Note: Functions are not comparable in Go, so it we can't test using
		//       one here in any way that isn't already covered elsewhere. But
		//       we can test the error handling of missing ones.
		{
			name:       "missing function",
			definition: `missing("foo", "bar")`,
			wantErr:    true,
		},
		{
			name:       "jsonpath",
			definition: `{"foo": $.foo.bar[234]..baz[*]}`,
			wantOut: &Template{
				definition: object{
					"foo": field{
						value: query{
							expression: mustParseJSONPath("$.foo.bar[234]..baz[*]"),
						},
					},
				},
			},
		},
		{
			name: "complex",
			definition: `
				{
					"foo": [
						123,
						{
							@deprecated "baz": range $..stuff [
								{
									"x": null,
									"y": $.hello[1:5]
								}
							],
							"something": "with trailing comma",
						}
					]
				}
			`,
			wantOut: &Template{
				definition: object{
					"foo": field{value: array{
						numberConstant(123),
						object{
							"baz": field{
								value: generator{
									over: query{expression: mustParseJSONPath("$..stuff")},
									template: object{
										"x": field{value: nullConstant{}},
										"y": field{value: query{expression: mustParseJSONPath("$.hello[1:5]")}},
									},
								},
								annotation: "deprecated",
							},
							"something": field{value: stringConstant("with trailing comma")},
						},
					}},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotT, err := ParseString(tt.definition, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotT, tt.wantOut) {
				t.Errorf("ParseString() = %v, want %v", gotT, tt.wantOut)
			}
		})
	}
}
