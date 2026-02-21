package markdown

import (
	"strings"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

// Parse parses markdown bytes and returns the AST document node.
func Parse(md []byte) ast.Node {
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock
	p := parser.NewWithExtensions(extensions)
	return p.Parse(md)
}

// Render renders a parsed markdown AST to HTML bytes.
func Render(doc ast.Node) []byte {
	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)
	return markdown.Render(doc, renderer)
}

// ExtractRelativeImages walks the AST and returns a set of image destination
// paths that are relative (i.e. not fully-qualified URLs). Fully-qualified
// URLs (http://, https://, //, etc.) are skipped — the browser fetches those
// directly without involving this server.
func ExtractRelativeImages(doc ast.Node) map[string]bool {
	allowed := make(map[string]bool)
	ast.WalkFunc(doc, func(node ast.Node, entering bool) ast.WalkStatus {
		img, ok := node.(*ast.Image)
		if !ok || !entering {
			return ast.GoToNext
		}
		dest := string(img.Destination)
		if isAbsoluteURL(dest) {
			return ast.GoToNext
		}
		allowed[dest] = true
		return ast.GoToNext
	})
	return allowed
}

// isAbsoluteURL reports whether s is a fully-qualified URL that the browser
// will fetch directly (http://, https://, or protocol-relative //).
func isAbsoluteURL(s string) bool {
	return strings.HasPrefix(s, "http://") ||
		strings.HasPrefix(s, "https://") ||
		strings.HasPrefix(s, "//")
}
