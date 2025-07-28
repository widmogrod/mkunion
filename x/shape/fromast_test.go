package shape

import (
	"github.com/google/go-cmp/cmp"
	"go/ast"
	"go/token"
	"testing"
)

func buildAST() ast.Expr {
	// Positions are faked here. Normally these come from parser.
	pos := token.Pos(1)

	return &ast.IndexExpr{
		X: &ast.IndexExpr{
			X: &ast.Ident{
				NamePos: pos + 1,
				Name:    "Ok",
			},
			Lbrack: pos + 2,
			Index: &ast.Ident{
				NamePos: pos + 3,
				Name:    "User",
			},
			Rbrack: pos + 4,
		},
		Lbrack: pos + 5,
		Index: &ast.BasicLit{
			ValuePos: pos + 6,
			Kind:     token.STRING,
			Value:    `"APIError"`,
		},
		Rbrack: pos + 7,
	}
}

func TestFromAST(t *testing.T) {
	useCases := []struct {
		name     string
		ast      ast.Expr
		expected Shape
	}{
		{
			name: "type",
			ast:  buildAST(), //  func(ok *Ok[Option[User], APIError]) string {
			expected: &RefName{
				Name: "Ok",
				Indexed: []Shape{
					&RefName{
						Name: "User",
					},
					&RefName{
						Name: "APIError",
					},
				},
			},
		},
	}
	for _, uc := range useCases {
		t.Run(uc.name, func(t *testing.T) {
			result := FromAST(uc.ast)
			if diff := cmp.Diff(uc.expected, result); diff != "" {
				t.Fatalf("FromAST: diff: (-want +got)\n%s", diff)
			}
		})
	}
}
