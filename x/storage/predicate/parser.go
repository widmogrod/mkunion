package predicate

import (
	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
	"strings"
)

var (
	predicateLexer = lexer.MustSimple([]lexer.SimpleRule{
		{"Whitespace", `\s+`},
		{"Keyword", `AND|OR|NOT`},
		{"Operator", `(<>|<=|>=|=|<|>|!=)`},
		{"Bind", `:[a-zA-Z][a-zA-Z0-9]*`},
		{"Ident", `[a-zA-Z][a-zA-Z0-9\#\.\[\]]*`},
	})
	predicateParser = participle.MustBuild[Expression](
		participle.Lexer(predicateLexer),
		participle.Elide("Whitespace"),
		participle.UseLookahead(2),
	)
)

func Parse(input string) (Predicate, error) {
	// Parse the input and build a Predicate value
	ast, err := predicateParser.ParseString("", strings.TrimSpace(input))
	if err != nil {
		return nil, err
	}

	// Convert the AST to a Predicate value
	return ast.ToPredicate()
}

type Comparable struct {
	Location string `( @Ident`
	Operator string `  @( "<>" | "<=" | ">=" | "=" | "<" | ">" | "!=" )`
	BindName string `  @Bind | @Ident)`
}

func (a Comparable) ToPredicate() (Predicate, error) {
	return &Compare{
		Location:  a.Location,
		Operation: a.Operator,
		BindValue: a.BindName,
	}, nil
}

type Boolean bool

func (b *Boolean) Capture(values []string) error {
	*b = values[0] == "TRUE"
	return nil
}

type Expression struct {
	Or []*OrCondition `@@ ( "OR" @@ )*`
}

func (e Expression) ToPredicate() (Predicate, error) {
	result := Or{L: nil}
	for _, or := range e.Or {
		p, err := or.ToPredicate()
		if err != nil {
			return nil, err
		}
		result.L = append(result.L, p)
	}

	return &result, nil
}

type OrCondition struct {
	And []*Condition `@@ ( "AND" @@ )*`
}

func (c *OrCondition) ToPredicate() (Predicate, error) {
	result := And{L: nil}
	for _, and := range c.And {
		p, err := and.ToPredicate()
		if err != nil {
			return nil, err
		}
		result.L = append(result.L, p)
	}

	return &result, nil
}

type Condition struct {
	Operand *Comparable `  @@`
	Not     *Condition  `| "NOT" @@`
}

func (c *Condition) ToPredicate() (Predicate, error) {
	if c.Not != nil {
		p, err := c.Not.ToPredicate()
		if err != nil {
			return nil, err
		}
		return &Not{p}, nil
	}

	return c.Operand.ToPredicate()
}
