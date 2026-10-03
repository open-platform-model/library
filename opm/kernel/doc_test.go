package kernel_test

import (
	"fmt"
	"go/ast"
	"go/doc"
	"go/doc/comment"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// The package doc is the render contract AGENTS.md points readers to, and
// nothing else notices when its layout is flattened: gofmt, go vet and lint
// all accept a doc comment rewrapped as prose (commit 7729b92 did exactly
// that). This guard parses the package doc into the go/doc/comment model
// that go doc renders from and checks the block types of the parts that
// must stay a list and code. It pins no wording.

// surfaceMinItems is the number of verb groups the Surface list names today.
const surfaceMinItems = 7

// codeAnchors are text each code example contains; each must be found inside
// a code block.
var codeAnchors = []string{
	"func renderAll(",
	"result.Diagnostics.UnhandledTraits",
	"result.Diagnostics.Replacements",
}

func TestPackageDoc_KeepsListAndCodeBlocks(t *testing.T) {
	src, err := os.ReadFile("doc.go")
	if err != nil {
		t.Fatalf("read doc.go: %v", err)
	}
	for _, problem := range docLayoutProblems(parseKernelDoc(t, src)) {
		t.Error(problem)
	}
}

// parseKernelDoc parses every non-test Go file of this package, as go doc
// does, with docGo standing in for doc.go, and returns the parsed package
// doc. The whole package is parsed because [Kernel.X] links resolve only
// when the package's declarations are known.
func parseKernelDoc(t *testing.T, docGo []byte) *comment.Doc {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package dir: %v", err)
	}
	fset := token.NewFileSet()
	var files []*ast.File
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		var src any
		if name == "doc.go" {
			src = docGo
		}
		f, err := parser.ParseFile(fset, name, src, parser.ParseComments)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		files = append(files, f)
	}
	pkg, err := doc.NewFromFiles(fset, files, "github.com/open-platform-model/library/opm/kernel")
	if err != nil {
		t.Fatalf("build package doc: %v", err)
	}
	return pkg.Parser().Parse(pkg.Doc)
}

// docLayoutProblems returns one message per part of the package doc that no
// longer renders as the block it must be: the list after the Surface
// heading, and each code example.
func docLayoutProblems(d *comment.Doc) []string {
	var problems []string
	if p := surfaceListProblem(d.Content); p != "" {
		problems = append(problems, p)
	}
	for _, anchor := range codeAnchors {
		if p := codeBlockProblem(d.Content, anchor); p != "" {
			problems = append(problems, p)
		}
	}
	return problems
}

func surfaceListProblem(blocks []comment.Block) string {
	at := -1
	for i, b := range blocks {
		if h, ok := b.(*comment.Heading); ok && inlineText(h.Text) == "Surface" {
			at = i
			break
		}
	}
	if at < 0 {
		return "package doc: no \"# Surface\" heading found"
	}
	// At most one lead-in paragraph sits between the heading and the list.
	i := at + 1
	if i < len(blocks) {
		if _, ok := blocks[i].(*comment.Paragraph); ok {
			i++
		}
	}
	if i >= len(blocks) {
		return "package doc: expected a list after the Surface heading, found the end of the doc"
	}
	list, ok := blocks[i].(*comment.List)
	if !ok {
		return fmt.Sprintf("package doc: expected a *comment.List of the Kernel verbs after the Surface heading, found %s (the list was flattened into prose?)", describe(blocks[i]))
	}
	if len(list.Items) < surfaceMinItems {
		return fmt.Sprintf("package doc: the Surface list has %d items, want at least %d", len(list.Items), surfaceMinItems)
	}
	for n, item := range list.Items {
		if !startsWithKernelLink(item) {
			return fmt.Sprintf("package doc: Surface list item %d does not start with a [Kernel.X] doc link: %s", n+1, describeItem(item))
		}
	}
	return ""
}

func startsWithKernelLink(item *comment.ListItem) bool {
	if len(item.Content) == 0 {
		return false
	}
	p, ok := item.Content[0].(*comment.Paragraph)
	if !ok || len(p.Text) == 0 {
		return false
	}
	link, ok := p.Text[0].(*comment.DocLink)
	return ok && link.Recv == "Kernel"
}

func codeBlockProblem(blocks []comment.Block, anchor string) string {
	for _, b := range blocks {
		if c, ok := b.(*comment.Code); ok && strings.Contains(c.Text, anchor) {
			return ""
		}
	}
	for _, b := range blocks {
		if strings.Contains(blockText(b), anchor) {
			return fmt.Sprintf("package doc: expected the example containing %q in a *comment.Code block, found it in %s (the example was flattened into prose?)", anchor, describe(b))
		}
	}
	return fmt.Sprintf("package doc: expected the example containing %q in a *comment.Code block, found it nowhere", anchor)
}

// describe names a block's Go type and the start of its text.
func describe(b comment.Block) string {
	text := strings.Join(strings.Fields(blockText(b)), " ")
	if len(text) > 60 {
		text = text[:60] + "..."
	}
	return fmt.Sprintf("%T %q", b, text)
}

func describeItem(item *comment.ListItem) string {
	if len(item.Content) == 0 {
		return "an empty item"
	}
	return describe(item.Content[0])
}

func blockText(b comment.Block) string {
	switch b := b.(type) {
	case *comment.Paragraph:
		return inlineText(b.Text)
	case *comment.Heading:
		return inlineText(b.Text)
	case *comment.Code:
		return b.Text
	case *comment.List:
		var sb strings.Builder
		for _, item := range b.Items {
			for _, c := range item.Content {
				sb.WriteString(blockText(c))
				sb.WriteString(" ")
			}
		}
		return sb.String()
	}
	return ""
}

func inlineText(texts []comment.Text) string {
	var sb strings.Builder
	for _, t := range texts {
		switch t := t.(type) {
		case comment.Plain:
			sb.WriteString(string(t))
		case comment.Italic:
			sb.WriteString(string(t))
		case *comment.Link:
			sb.WriteString(inlineText(t.Text))
		case *comment.DocLink:
			sb.WriteString(inlineText(t.Text))
		}
	}
	return sb.String()
}
