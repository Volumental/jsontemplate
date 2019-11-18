# JSONTemplate

`jsontemplate` is a JSON transformation and templating language and library implemented in Go.

In simple terms, it renders arbitrary JSON structures based on a JSON-like template definition, populating it with data from an some other JSON structure.

## Feature overview

- Low-clutter template syntax, aimed to look similar to the final output.
- [JSONPath](https://goessner.net/articles/JsonPath/) expressions to fetch values from the input.
- Array generator expressions with subtemplates, allowing mapping of arrays of objects.
- Ability to call Go functions from within a template.

## Getting started

The following is a complete but minimal program that loads a template and transforms an input JSON object using it.

```go
package main

import (
	"os"
	"strings"

	"github.com/Volumental/jsontemplate"
)

const input = `{ "snakeCase": 123 }`

func main() {
	template, _ := jsontemplate.ParseString(`{ "CamelCase": $.snakeCase }`, nil)
	template.RenderJSON(os.Stdout, strings.NewReader(input))
	os.Stdout.Sync()
}
```

Running the above program will output:
```
{"CamelCase":123}
```

## Features by example

This example illustrates some of the features in `jsontemplate`. For further details, please see the library documentation.

Consider the following input JSON structure:

```json
{
	"store": {
		"book": [ 
			{
				"category": "reference",
				"author": "Nigel Rees",
				"title": "Sayings of the Century",
				"price": 8.95
			},
			{
				"category": "fiction",
				"author": "Evelyn Waugh",
				"title": "Sword of Honour",
				"price": 12.99
			},
			{
				"category": "fiction",
				"author": "Herman Melville",
				"title": "Moby Dick",
				"isbn": "0-553-21311-3",
				"price": 8.99
			},
			{
				"category": "fiction",
				"author": "J. R. R. Tolkien",
				"title": "The Lord of the Rings",
				"isbn": "0-395-19395-8",
				"price": 22.99
			}
		],
		"bicycle": {
			"color": "red",
			"price": 19.95
		}
	}
}
```

When fed through the following template:

```
{
	# Pick an invidual field.
	"bicycle_color": $.store.bicycle.color,

	"book_info": {
		# Slice an array, taking the first three elements.
		"top_three": $.store.book[:3],

		# Map a list of objects.
		"price_list": range $.store.book[*] [
			{
				"title": $.title,
				"price": $.price,
			}
		],
	},
	
	# Calculate the average of all price fields by calling a Go function.
	"avg_price": Avg($..price),
}
```

...the following output is yielded:

```json
{
	"avg_price": 14.774000000000001,
	"bicycle_color": "red",
	"book_info": {
		"price_list": [
			{
				"price": 8.95,
				"title": "Sayings of the Century"
			},
			{
				"price": 12.99,
				"title": "Sword of Honour"
			},
			{
				"price": 8.99,
				"title": "Moby Dick"
			},
			{
				"price": 22.99,
				"title": "The Lord of the Rings"
			}
		],
		"top_three": [
			{
				"author": "Nigel Rees",
				"category": "reference",
				"price": 8.95,
				"title": "Sayings of the Century"
			},
			{
				"author": "Evelyn Waugh",
				"category": "fiction",
				"price": 12.99,
				"title": "Sword of Honour"
			},
			{
				"author": "Herman Melville",
				"category": "fiction",
				"isbn": "0-553-21311-3",
				"price": 8.99,
				"title": "Moby Dick"
			}
		]
	}
}
```

## Performance

`jsontemplate` has first and foremost been designed with correctness and ease of use in mind. As such, optimum performance has not been the primary objective. Nevertheless, you can expect to see in the order of 10 MB/s on a single CPU core, around half of which is JSON parsing/encoding. We expect this to be more than adequate for most production use-cases.

## Maturity

`jsontemplate` is provided as-is, and you should assume it has bugs. That said, at the time of writing, the library is being used for production workloads at [Volumental](https://www.volumental.com).

Until a 1.0 release is made, incompatible changes may occur, though we will generally strive to maintain full backwards compatibility. Incompatible changes to the template definition format are unlikely to be introduced at this point.
