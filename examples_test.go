package jsontemplate_test

import (
	"os"
	"strings"

	"github.com/Volumental/jsontemplate"
)

// Store example from JSONPath.
const Input = `
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
`

const Template = `
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
	
	# Calculate the average of all price fields.
	"avg_price": Avg($..price),
}
`

// Helper function we'll use in the template.
func Avg(values []interface{}) float64 {
	var sum = 0.0
	var cnt = 0
	for _, val := range values {
		if num, ok := val.(float64); ok {
			sum += num
			cnt += 1
		}
	}
	return sum / float64(cnt)
}

func Example() {
	var funcs = jsontemplate.FunctionMap{"Avg": Avg}
	var template, _ = jsontemplate.ParseString(Template, funcs)

	template.RenderJSON(os.Stdout, strings.NewReader(Input))
	os.Stdout.Sync()
	// Output: {"avg_price":14.774000000000001,"bicycle_color":"red","book_info":{"price_list":[{"price":8.95,"title":"Sayings of the Century"},{"price":12.99,"title":"Sword of Honour"},{"price":8.99,"title":"Moby Dick"},{"price":22.99,"title":"The Lord of the Rings"}],"top_three":[{"author":"Nigel Rees","category":"reference","price":8.95,"title":"Sayings of the Century"},{"author":"Evelyn Waugh","category":"fiction","price":12.99,"title":"Sword of Honour"},{"author":"Herman Melville","category":"fiction","isbn":"0-553-21311-3","price":8.99,"title":"Moby Dick"}]}}
}
