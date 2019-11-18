package parse

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func numberValue(f float64) Value          { return Value{Number: &f} }
func stringValue(s string) Value           { return Value{String: &s} }
func boolValue(b bool) Value               { return Value{Bool: &b} }
func extractorValue(jsonPath string) Value { return Value{Extractor: &jsonPath} }

func TestTextRenderer_Render(t *testing.T) {
	tests := []struct {
		name       string
		definition string
		wantOut    Template
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
			wantOut: Template{
				Root: numberValue(1),
			},
		},
		{
			name:       "string",
			definition: `"foo"`,
			wantOut: Template{
				Root: stringValue("foo"),
			},
		},
		{
			name:       "array",
			definition: `[1, 2, true, "foo"]`,
			wantOut: Template{
				Root: Value{Array: []Value{
					numberValue(1),
					numberValue(2),
					boolValue(true),
					stringValue("foo"),
				}},
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
			wantOut: Template{
				Root: Value{Object: &Object{
					Fields: []AnnotatedField{
						{
							Key:   "foo",
							Value: boolValue(true),
						},
						{
							Key:   "bar",
							Value: numberValue(123),
						},
					},
				}},
			},
		},
		{
			name:       "function",
			definition: `compare("foo", "bar")`,
			wantOut: Template{
				Root: Value{Function: &Function{
					Name: "compare",
					Args: []Value{
						stringValue("foo"),
						stringValue("bar"),
					},
				}},
			},
		},
		{
			name:       "annotation",
			definition: `{@foobar "foo": true}`,
			wantOut: Template{
				Root: Value{Object: &Object{
					Fields: []AnnotatedField{
						{
							Annotation: "foobar",
							Key:        "foo",
							Value:      boolValue(true),
						},
					},
				}},
			},
		},
		{
			name:       "jsonpath",
			definition: `{"foo": $.foo.bar[234]..baz[*]}`,
			wantOut: Template{
				Root: Value{Object: &Object{
					Fields: []AnnotatedField{
						{
							Key:   "foo",
							Value: extractorValue("$.foo.bar[234]..baz[*]"),
						},
					},
				}},
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
			wantOut: Template{
				Root: Value{Object: &Object{Fields: []AnnotatedField{
					{
						Key: "foo",
						Value: Value{
							Array: []Value{
								numberValue(123),
								Value{
									Object: &Object{Fields: []AnnotatedField{
										{
											Annotation: "deprecated",
											Key:        "baz",
											Value: Value{
												Generator: &Generator{
													Range: "$..stuff",
													SubTemplate: Value{
														Object: &Object{Fields: []AnnotatedField{
															{
																Key:   "x",
																Value: Value{Null: true},
															},
															{
																Key:   "y",
																Value: extractorValue("$.hello[1:5]"),
															},
														}},
													},
												},
											},
										},
										{
											Key:   "something",
											Value: stringValue("with trailing comma"),
										},
									}},
								},
							},
						},
					},
				},
				}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out Template
			if err := Parser.ParseString(tt.definition, &out); (err != nil) != tt.wantErr {
				// Print the lexer output in case of a parser error (for debug help).
				var l, _ = lex.Lex(strings.NewReader(tt.definition))
				var symbolNames = map[rune]string{}
				for k, v := range lex.Symbols() {
					symbolNames[v] = k
				}
				for i := 0; i < 10000; i++ {
					if tok, err := l.Next(); err != nil || tok.EOF() {
						break
					} else {
						if strings.TrimSpace(tok.String()) != "" {
							fmt.Println(tok, symbolNames[tok.Type])
						}
					}
				}
				t.Errorf("Parser.ParseString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(out, tt.wantOut) {
				t.Errorf("Parser.ParseString() = %v, want %v", out, tt.wantOut)
			}
		})
	}
}
