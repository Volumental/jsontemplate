package jsontemplate_test

import (
	"encoding/json"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/Volumental/jsontemplate"
)

// Store example from JSONPath.
const benchmarkInput = `
{
	"foo": {
		"bar": [
			{
				"text": "this is a benchmark test",
				"number": 12345.6789,
				"bool": true,
				"array": [1, 2, 3, "hello", true]
			},
			{
				"text": "short"
			},
			{
				"text": "loooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooonger",
				"number": 0,
				"bool": false,
				"array": [1, 2, 3, "hello", true]
			},
			{
				"text": "this is a second benchmark test",
				"number": 4711,
				"bool": null
			},
			{
				"text": "this is the final benchmark test",
				"array": ["nice", 1, 2, 3, 1, 2, 3, 1, 2, 3, 1, 2, 3, 1, 2, 3, 1, 2, 3, 1, 2, 3, 1, 2, 3]
			}
		]
	}
}
`

const benchmarkTemplate = `
{
	"arrays": range $..bar [$.array[0]],
	"mapping": range $.foo.bar[:4] [
		{
			"string": ToUpper($.text),
			"num": $.number,
		}
	],
	"a_text": $.foo.bar[3].text,
}
`

var result interface{}

func Benchmark_core(b *testing.B) {
	var funcs = jsontemplate.FunctionMap{"ToUpper": strings.ToUpper}
	var template, err = jsontemplate.ParseString(benchmarkTemplate, funcs)
	if err != nil {
		panic(err)
	}

	b.SetBytes(int64(len(benchmarkInput)))

	var input interface{}
	if err := json.Unmarshal([]byte(benchmarkInput), &input); err != nil {
		panic(err)
	}

	var output interface{}
	for n := 0; n < b.N; n++ {
		var err error
		output, err = template.Render(input)
		if err != nil {
			panic(err)
		}
	}
	result = output
}

func Benchmark_full(b *testing.B) {
	var funcs = jsontemplate.FunctionMap{"ToUpper": strings.ToUpper}
	var template, err = jsontemplate.ParseString(benchmarkTemplate, funcs)
	if err != nil {
		panic(err)
	}

	b.SetBytes(int64(len(benchmarkInput)))

	var input interface{}
	if err := json.Unmarshal([]byte(benchmarkInput), &input); err != nil {
		panic(err)
	}

	var output interface{}
	for n := 0; n < b.N; n++ {
		if err = template.RenderJSON(ioutil.Discard, strings.NewReader(benchmarkInput)); err != nil {
			panic(err)
		}
	}
	result = output
}
