package telegram

import "strings"

const (
	codeOpen  = "<code>"
	codeClose = "</code>"
)

// bot.EscapeMarkdown only covers MarkdownV2, which needs 18 escapes instead of 3
func EscapeHTML(s string) string {
	if !strings.ContainsAny(s, "&<>") {
		return s
	}
	var b strings.Builder
	b.Grow(len(s) + 16)
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		default:
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

func escapeHTMLAttr(s string) string {
	escaped := EscapeHTML(s)
	if !strings.Contains(escaped, `"`) {
		return escaped
	}
	return strings.ReplaceAll(escaped, `"`, "&quot;")
}

// the datasets mix markdown links with html code spans, so both need translating
func FormatDigestHTML(src string) string {
	if src == "" {
		return ""
	}

	var b strings.Builder
	b.Grow(len(src) + len(src)/8 + 16)
	writeHTML(&b, src, false)
	return b.String()
}

// inCode suppresses nested code spans, cuz MITRE also puts links inside one
func writeHTML(b *strings.Builder, src string, inCode bool) {
	i := 0
	for i < len(src) {
		rest := src[i:]

		if !inCode && strings.HasPrefix(rest, codeOpen) {
			body, width, ok := cutCodeSpan(rest)
			if !ok {
				// unbalanced tag, so treat the remainder as prose
				b.WriteString(EscapeHTML(rest))
				return
			}
			b.WriteString(codeOpen)
			writeHTML(b, body, true)
			b.WriteString(codeClose)
			i += width
			continue
		}

		if rest[0] == '[' {
			if text, href, width, ok := cutMarkdownLink(rest); ok {
				b.WriteString(`<a href="`)
				b.WriteString(escapeHTMLAttr(href))
				b.WriteString(`">`)
				b.WriteString(EscapeHTML(text))
				b.WriteString(`</a>`)
				i += width
				continue
			}
		}

		idx := strings.IndexAny(rest, "&<[")
		switch {
		case idx < 0:
			b.WriteString(EscapeHTML(rest))
			return
		case idx > 0:
			b.WriteString(EscapeHTML(rest[:idx]))
			i += idx
		default:
			b.WriteString(EscapeHTML(rest[:1]))
			i++
		}
	}
}

func cutCodeSpan(s string) (body string, width int, ok bool) {
	end := strings.Index(s, codeClose)
	if end < 0 {
		return "", 0, false
	}
	return s[len(codeOpen):end], end + len(codeClose), true
}

// rejects anything that is not really a link, so a stray bracket stays literal
func cutMarkdownLink(s string) (text, href string, width int, ok bool) {
	rel := strings.Index(s, "](")
	if rel < 0 {
		return "", "", 0, false
	}
	end := strings.Index(s[rel+2:], ")")
	if end < 0 {
		return "", "", 0, false
	}

	text = s[1:rel]
	href = s[rel+2 : rel+2+end]
	if text == "" || href == "" {
		return "", "", 0, false
	}
	// nested brackets mean this is not a simple link
	if strings.ContainsAny(text, "[]") {
		return "", "", 0, false
	}
	// spaces and non-http schemes are not linkable
	if strings.ContainsAny(href, " \t\n") {
		return "", "", 0, false
	}
	if !strings.HasPrefix(href, "http://") && !strings.HasPrefix(href, "https://") {
		return "", "", 0, false
	}

	return text, href, rel + 2 + end + 1, true
}
