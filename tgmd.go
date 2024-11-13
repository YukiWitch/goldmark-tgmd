package tgmd

import (
	"bytes"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	ext "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/renderer"
	textm "github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// TGMD (telegramMarkdown) endpoint.
func TGMD() goldmark.Markdown {
	return goldmark.New(
		goldmark.WithRenderer(
			renderer.NewRenderer(
				renderer.WithNodeRenderers(util.Prioritized(NewRenderer(), 1000)),
			),
		),
		goldmark.WithExtensions(Strikethroughs),
		goldmark.WithExtensions(Hidden),
		goldmark.WithExtensions(DoubleSpace),
	)
}

// Renderer implement renderer.NodeRenderer object.
type Renderer struct{}

// NewRenderer initialize Renderer as renderer.NodeRenderer.
func NewRenderer() renderer.NodeRenderer {
	return &Renderer{}
}

// RegisterFuncs add AST objects to Renderer.
func (r *Renderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindDocument, r.document)
	reg.Register(ast.KindParagraph, r.paragraph)

	reg.Register(ast.KindText, r.renderText)
	reg.Register(ast.KindString, r.renderString)
	reg.Register(ast.KindEmphasis, r.emphasis)

	reg.Register(ast.KindHeading, r.heading)
	reg.Register(ast.KindList, r.list)
	reg.Register(ast.KindListItem, r.listItem)
	reg.Register(ast.KindLink, r.link)

	reg.Register(ast.KindBlockquote, r.blockquote)
	reg.Register(ast.KindFencedCodeBlock, r.code)
	reg.Register(ast.KindCodeSpan, r.codeSpan)

	reg.Register(ext.KindStrikethrough, r.strikethrough)
	reg.Register(KindHidden, r.hidden)
	reg.Register(KindDoubleSpace, r.doubleSpace)
}

func (r *Renderer) heading(w util.BufWriter, _ []byte, node ast.Node, entering bool) (
	ast.WalkStatus, error,
) {
	n := node.(*ast.Heading)
	if entering {
		writeNewLine(w)

		if n.HasBlankPreviousLines() {
			writeNewLine(w)
		}

		Config.headings[n.Level-1].writeStart(w)
	} else {
		Config.headings[n.Level-1].writeEnd(w)
	}
	return ast.WalkContinue, nil
}

func (r *Renderer) paragraph(w util.BufWriter, source []byte, node ast.Node, entering bool) (
	ast.WalkStatus, error,
) {
	n := node.(*ast.Paragraph)

	if entering {
		if n.Parent().Kind() == ast.KindDocument {
			writeNewLine(w)
			if n.HasBlankPreviousLines() {
				writeNewLine(w)
			}
		}
	}

	return ast.WalkContinue, nil
}

func (r *Renderer) list(w util.BufWriter, source []byte, node ast.Node, entering bool) (
	ast.WalkStatus, error,
) {
	return ast.WalkContinue, nil
}

func (r *Renderer) listItem(w util.BufWriter, source []byte, node ast.Node, entering bool) (
	ast.WalkStatus, error,
) {
	n := node.(*ast.ListItem)
	if entering {
		writeNewLine(w)

		if n.HasBlankPreviousLines() {
			writeNewLine(w)
		}

		if n.Parent().Parent().Kind().String() == ast.KindDocument.String() {
			writeRowBytes(w, SpaceChar.Bytes(2))
			writeRune(w, Config.listBullets[0])
		} else {
			if n.Parent().Parent().Parent().Parent() != nil {
				if n.Parent().Parent().Parent().Parent().Kind().String() == ast.KindListItem.String() {
					writeRowBytes(w, SpaceChar.Bytes(6))
					writeRune(w, Config.listBullets[2])
				} else {
					writeRowBytes(w, SpaceChar.Bytes(4))
					writeRune(w, Config.listBullets[1])
				}
			}
		}
		writeRowBytes(w, []byte{SpaceChar.Byte()})
	}

	return ast.WalkContinue, nil
}

func (r *Renderer) code(w util.BufWriter, source []byte, node ast.Node, entering bool) (
	ast.WalkStatus, error,
) {
	n := node.(interface {
		Lines() *textm.Segments
	})
	var content []byte
	l := n.Lines().Len()
	for i := 0; i < l; i++ {
		line := n.Lines().At(i)
		content = append(content, line.Value(source)...)
	}
	content = bytes.ReplaceAll(
		content,
		[]byte{TabChar.Byte()},
		[]byte{SpaceChar.Byte(), SpaceChar.Byte(), SpaceChar.Byte()},
	)
	nn := node.(*ast.FencedCodeBlock)
	if entering {
		writeNewLine(w)
		if node.HasBlankPreviousLines() {
			writeNewLine(w)
		}
		writeWrapperArr(w.Write(CodeTg.Bytes()))
		writeWrapperArr(w.Write(nn.Language(source)))
	} else {
		writeNewLine(w)
		writeWrapperArr(w.Write(content))
		writeWrapperArr(w.Write(CodeTg.Bytes()))
		writeNewLine(w)
	}
	return ast.WalkContinue, nil
}

func (r *Renderer) renderText(w util.BufWriter, source []byte, node ast.Node, entering bool) (
	ast.WalkStatus, error,
) {
	if !entering {
		return ast.WalkContinue, nil
	}
	n := node.(*ast.Text)
	text := n.Segment.Value(source)
	if n.HardLineBreak() {
		text = append(text, "\n"...)
	}
	render(w, text)
	return ast.WalkContinue, nil
}

func (r *Renderer) renderString(w util.BufWriter, source []byte, node ast.Node, entering bool) (
	ast.WalkStatus, error,
) {
	if !entering {
		return ast.WalkContinue, nil
	}
	n := node.(*ast.String)
	_, _ = w.Write(n.Value)
	return ast.WalkContinue, nil
}

func (r *Renderer) emphasis(w util.BufWriter, _ []byte, node ast.Node, _ bool) (
	ast.WalkStatus, error,
) {
	n := node.(*ast.Emphasis)
	if n.Level == 2 {
		writeRowBytes(w, BoldTg.Bytes())
	}
	if n.Level == 1 {
		writeRowBytes(w, ItalicsTg.Bytes())
	}
	return ast.WalkContinue, nil
}

func (r *Renderer) link(w util.BufWriter, _ []byte, node ast.Node, entering bool) (
	ast.WalkStatus, error,
) {
	n := node.(*ast.Link)
	if entering {
		writeRowBytes(w, []byte{OpenBracketChar.Byte()})
	} else {
		writeRowBytes(w, []byte{CloseBracketChar.Byte(), OpenParenChar.Byte()})
		writeRowBytes(w, n.Destination)
		writeRowBytes(w, []byte{CloseParenChar.Byte()})
	}
	return ast.WalkContinue, nil
}

func (r *Renderer) blockquote(w util.BufWriter, _ []byte, _ ast.Node, entering bool) (
	ast.WalkStatus, error,
) {
	if entering {
		writeNewLine(w)
		writeRowBytes(w, []byte{GreaterThanChar.Byte()})
	}
	return ast.WalkContinue, nil
}

func (r *Renderer) codeSpan(w util.BufWriter, _ []byte, _ ast.Node, _ bool) (
	ast.WalkStatus, error,
) {
	writeWrapperArr(w.Write(SpanTg.Bytes()))
	return ast.WalkContinue, nil
}

func (r *Renderer) strikethrough(w util.BufWriter, _ []byte, _ ast.Node, _ bool) (
	ast.WalkStatus, error,
) {
	writeWrapperArr(w.Write(StrikethroughTg.Bytes()))
	return ast.WalkContinue, nil
}

func (r *Renderer) hidden(w util.BufWriter, _ []byte, _ ast.Node, _ bool) (
	ast.WalkStatus, error,
) {
	writeWrapperArr(w.Write(HiddenTg.Bytes()))
	return ast.WalkContinue, nil
}

func (r *Renderer) doubleSpace(_ util.BufWriter, _ []byte, _ ast.Node, _ bool) (
	ast.WalkStatus, error,
) {
	return ast.WalkContinue, nil
}

func (r *Renderer) document(_ util.BufWriter, _ []byte, _ ast.Node, _ bool) (
	ast.WalkStatus, error,
) {
	return ast.WalkContinue, nil
}
