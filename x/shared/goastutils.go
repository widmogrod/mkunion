package shared

import (
	"go/ast"
	"strings"
)

// Comment implementation was copied from func (g *CommentGroup) Text() string
// and reduced return all comments lines. In contrast to original implementation
// that skips declaration comments like "//go:generate" which is important
// for inferring union types.
func Comment(g *ast.CommentGroup) string {
	if g == nil {
		return ""
	}
	comments := make([]string, len(g.List))
	for i, c := range g.List {
		comments[i] = c.Text
	}

	lines := make([]string, 0, 10) // most comments are less than 10 lines
	for _, c := range comments {
		// Remove comment markers.
		// The parser has given us exactly the comment text.
		switch c[1] {
		case '/':
			//-style comment (no newline at the end)
			c = c[2:]
			if len(c) == 0 {
				// empty line
				break
			}
			if c[0] == ' ' {
				// strip first space - required for Example tests
				c = c[1:]
				break
			}
		case '*':
			/*-style comment */
			c = c[2 : len(c)-2]
		}

		lines = append(lines, c)
	}

	return strings.Join(lines, "\n")
}
