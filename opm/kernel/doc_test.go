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
// all accept a doc comment rewrapped as prose. This guard parses the package
// doc into the go/doc/comment model that go doc renders from and checks the
// block types of the parts that must stay a list and code. It pins no
// wording.

// surfaceMinItems is the number of verb groups the Surface list names today.
const surfaceMinItems = 7

// codeExamples lists, per code example, text its first line and its last
// line contain, with any anchors between. Every anchor of an example must sit
// in one and the same *comment.Code block: a single de-indented line splits
// the example into code, paragraph, code, which puts the first and last
// anchors in different blocks.
var codeExamples = []codeExample{
	{"renderAll", []string{
		"func renderAll(",
		"for _, dir := range instanceDirs",
		"close(errs)",
		"for err := range errs",
		"return nil\n}",
	}},
	{"diagnostics", []string{
		"range result.Diagnostics.UnhandledTraits",
		"range result.Diagnostics.ResolvedVersions",
		"range result.Diagnostics.Skipped",
		"!s.ComponentOmitted)\n}",
	}},
	{"replacements", []string{
		"range result.Diagnostics.Replacements",
		"r.By)\n}",
	}},
}

type codeExample struct {
	name    string
	anchors []string
}

// codeMarkers are fragments only Go code carries. Prose in the package doc
// never contains them, so a non-code block that does holds a flattened
// example, including one added after codeExamples was last updated.
var codeMarkers = []string{
	" := ",
	"result.Diagnostics.",
	"log.Printf(",
	"if err != nil",
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

// TestPackageDoc_RefusesOneDeIndentedLine proves the guard catches a partly
// flattened example: de-indenting one statement leaves every anchor in some
// code block but splits the example into code, paragraph, code.
func TestPackageDoc_RefusesOneDeIndentedLine(t *testing.T) {
	src, err := os.ReadFile("doc.go")
	if err != nil {
		t.Fatalf("read doc.go: %v", err)
	}
	for _, line := range []string{
		"wg.Add(1)",
		"defer wg.Done()",
		"wg.Wait()",
		"return nil",
		"s.Component, s.Kind, s.FQN, !s.ComponentOmitted)",
	} {
		t.Run(line, func(t *testing.T) {
			mutated := deIndent(t, src, line)
			if len(docLayoutProblems(parseKernelDoc(t, mutated))) == 0 {
				t.Errorf("guard passed with %q de-indented out of its example", line)
			}
		})
	}
}

// deIndent turns the one doc.go comment line whose code is exactly stmt into
// a plain comment line, which go/doc/comment reads as a paragraph.
func deIndent(t *testing.T, src []byte, stmt string) []byte {
	t.Helper()
	lines := strings.Split(string(src), "\n")
	hits := 0
	for i, l := range lines {
		if strings.HasPrefix(l, "//\t") && strings.TrimSpace(strings.TrimPrefix(l, "//")) == stmt {
			lines[i] = "// " + stmt
			hits++
		}
	}
	if hits != 1 {
		t.Fatalf("doc.go has %d comment code lines %q, want exactly 1", hits, stmt)
	}
	return []byte(strings.Join(lines, "\n"))
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
	for _, ex := range codeExamples {
		if p := codeBlockProblem(d.Content, ex); p != "" {
			problems = append(problems, p)
		}
	}
	problems = append(problems, codeInProseProblems(d.Content)...)
	return problems
}

// codeInProseProblems reports every block other than a code block whose text
// carries a codeMarker.
func codeInProseProblems(blocks []comment.Block) []string {
	var problems []string
	for _, b := range blocks {
		if _, ok := b.(*comment.Code); ok {
			continue
		}
		text := blockText(b)
		for _, m := range codeMarkers {
			if strings.Contains(text, m) {
				problems = append(problems, fmt.Sprintf("package doc: %s contains the code fragment %q outside a *comment.Code block (an example was flattened into prose?)", describe(b), m))
				break
			}
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

// codeBlockProblem reports an example whose anchors are not all inside the
// one *comment.Code block that holds its first anchor.
func codeBlockProblem(blocks []comment.Block, ex codeExample) string {
	first := ex.anchors[0]
	var code *comment.Code
	for _, b := range blocks {
		if c, ok := b.(*comment.Code); ok && strings.Contains(c.Text, first) {
			code = c
			break
		}
	}
	if code == nil {
		for _, b := range blocks {
			if strings.Contains(blockText(b), first) {
				return fmt.Sprintf("package doc: expected the %s example, starting %q, in a *comment.Code block, found it in %s (the example was flattened into prose?)", ex.name, first, describe(b))
			}
		}
		return fmt.Sprintf("package doc: expected the %s example, starting %q, in a *comment.Code block, found it nowhere", ex.name, first)
	}
	for _, a := range ex.anchors[1:] {
		if !strings.Contains(code.Text, a) {
			return fmt.Sprintf("package doc: the %s example is not one *comment.Code block: the block starting %q lacks %q (a line was de-indented out of the example?)", ex.name, first, a)
		}
	}
	return ""
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
