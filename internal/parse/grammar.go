package parse

import (
	"github.com/alecthomas/participle"
	"github.com/alecthomas/participle/lexer"
	"github.com/alecthomas/participle/lexer/ebnf"
)

type Generator struct {
	Range       string `"range" @JSONPath`
	SubTemplate Value  `"[" @@ "]"`
}

type AnnotatedField struct {
	Annotation string `("@" @Ident)?`
	Key        string `@String ":"`
	Value      Value  `@@`
}

type Object struct {
	Fields []AnnotatedField `"{" (@@ ("," @@)* ","?)? "}"`
}

type Function struct {
	Name string  `@Ident`
	Args []Value `"(" (@@ ("," @@)*)? ")"`
}

type Value struct {
	// These are standard JSON fields.
	String *string  `  @String`
	Number *float64 `| @Number`
	Object *Object  `| @@`
	Array  []Value  `| "[" (@@ ("," @@)* ","?)? "]"`
	Bool   *bool    `| (@"true" | "false")`
	Null   bool     `| @"null"`

	// These are template elements generating JSON fields.
	Generator *Generator `| @@`
	Extractor *string    `| @JSONPath`
	Function  *Function  `| @@`
}

type Template struct {
	Root Value `@@`
}

var lex = lexer.Must(ebnf.New(`
	Comment = "#" { "\u0000"…"\uffff"-"\n" } .
	Ident = (alpha | "_") { "_" | alpha | digit } .
	String = "\"" { "\u0000"…"\uffff"-"\""-"\\" | "\\" any } "\"" .
	Number = Int | Float .
	Int = [ "-" ] digit { digit } .
	Float = [ "-" ] [ digit ] "." digit { digit } .
	JSONPath = "$" { "." { "." } JSONPathExpr } .
	JSONPathExpr = "*" | (Ident { "[" { "\u0000"…"\uffff"-"]" } "]" }) .
	Punct = "!"…"/" | ":"…"@" | "["…` + "\"`\"" + ` | "{"…"~" .
	Whitespace = " " | "\t" | "\n" | "\r" .

	alpha = "a"…"z" | "A"…"Z" .
	digit = "0"…"9" .
	any = "\u0000"…"\uffff" .
`))

var Parser = participle.MustBuild(
	&Template{},
	participle.Lexer(lex),
	participle.Unquote("String"),
	participle.Elide("Whitespace", "Comment"),
)
