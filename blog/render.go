package blog

import (
	"bytes"
	"html/template"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

// authorMarkdown renders content written in the admin — posts and pages.
// Raw HTML is kept: the authors are admins and their embeds (YouTube
// iframes, Instagram scripts) depend on it. This is a deliberate decision,
// recorded in docs/superpowers/specs/2026-09-22-server-side-markdown-design.md;
// a post or page can therefore run JavaScript on the site.
//
// Fenced code is coloured here rather than by highlight.js in the browser
// (#625). Almost no fence in existing posts names its language, so an
// unlabelled one is guessed; a guess chroma cannot make renders as plain
// text, and labelling that fence fixes it. The colours are CSS classes,
// styled in www/css/base.css so a theme can restyle them.
var authorMarkdown = goldmark.New(
	goldmark.WithExtensions(extension.GFM, highlighting.NewHighlighting(
		highlighting.WithGuessLanguage(true),
		highlighting.WithFormatOptions(chromahtml.WithClasses(true)),
	)),
	goldmark.WithRendererOptions(html.WithUnsafe()),
)

// commentMarkdown renders what visitors write. Raw HTML is dropped by
// goldmark and whatever markdown produces is then filtered by
// commentPolicy, because commenters are anonymous.
var commentMarkdown = goldmark.New(goldmark.WithExtensions(extension.GFM))

// commentPolicy keeps ordinary prose — emphasis, lists, code, links — and
// drops everything else, including javascript: URLs and event handlers.
var commentPolicy = bluemonday.UGCPolicy()

// renderMarkdown turns an author's markdown into HTML, raw HTML included.
// A render error falls back to the escaped source, so a broken document
// shows its text rather than an empty page.
func renderMarkdown(src string) template.HTML {
	var buf bytes.Buffer
	if err := authorMarkdown.Convert([]byte(src), &buf); err != nil {
		return template.HTML(template.HTMLEscapeString(src))
	}
	return template.HTML(buf.String())
}

// renderComment turns a visitor's markdown into sanitised HTML.
func renderComment(src string) template.HTML {
	var buf bytes.Buffer
	if err := commentMarkdown.Convert([]byte(src), &buf); err != nil {
		return template.HTML(template.HTMLEscapeString(src))
	}
	return template.HTML(commentPolicy.SanitizeBytes(buf.Bytes()))
}
