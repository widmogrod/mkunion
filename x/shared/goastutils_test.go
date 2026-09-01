package shared

import (
	"go/ast"
	"testing"

	"github.com/stretchr/testify/assert"
)

func mkComments(texts ...string) *ast.CommentGroup {
	group := &ast.CommentGroup{}
	for _, text := range texts {
		group.List = append(group.List, &ast.Comment{Text: text})
	}
	return group
}

func TestComment(t *testing.T) {
	useCases := map[string]struct {
		in   *ast.CommentGroup
		want string
	}{
		"nil group":            {nil, ""},
		"line comment":         {mkComments("// hello"), "hello"},
		"no space after slash": {mkComments("//hello"), "hello"},
		"empty line comment":   {mkComments("//"), ""},
		"block comment":        {mkComments("/* block */"), " block "},
		"multiple lines": {
			mkComments("// first", "// second"),
			"first\nsecond",
		},
		// the whole point of this fork of (*ast.CommentGroup).Text():
		// directives must survive, they carry union definitions
		"go directive is preserved": {
			mkComments("//go:generate go tool mkunion", "// doc line"),
			"go:generate go tool mkunion\ndoc line",
		},
		"go tag is preserved": {
			mkComments(`//go:tag mkunion:"State"`),
			`go:tag mkunion:"State"`,
		},
	}
	for name, uc := range useCases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, uc.want, Comment(uc.in))
		})
	}
}
