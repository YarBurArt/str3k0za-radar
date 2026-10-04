package telegram

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"
)

// a broken parse mode silently kills the digest send, so check every real one
func TestFormatDigestHTMLAgainstRealATTACKDump(t *testing.T) {
	const path = "../../../data/enterprise-attack.json"
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("dataset not available: %v", err)
	}

	var doc struct {
		Objects []struct {
			Type        string                        `json:"type"`
			Description string                        `json:"description"`
			ExternalRef []struct{ ExternalID string } `json:"external_references"`
			Name        string                        `json:"name"`
		} `json:"objects"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("parse dump: %v", err)
	}

	allowedTag := regexp.MustCompile(`</?(a|code|b|i)\b[^>]*>`)
	descriptions := 0
	links, codes := 0, 0

	for _, obj := range doc.Objects {
		if obj.Description == "" {
			continue
		}
		descriptions++

		out := FormatDigestHTML(obj.Description)

		bare := allowedTag.ReplaceAllString(out, "")
		if strings.ContainsAny(bare, "<>") {
			t.Fatalf("unescaped angle bracket in output for %q:\n%s", obj.Name, out)
		}
		if strings.Contains(bare, "&") && !regexp.MustCompile(`&(amp|lt|gt|quot);`).MatchString(bare) {
			t.Fatalf("suspicious ampersand for %q:\n%s", obj.Name, out)
		}
		if i := strings.Index(bare, "&"); i >= 0 && !strings.Contains(bare[i:], "&amp;") &&
			!strings.Contains(bare[i:], "&lt;") && !strings.Contains(bare[i:], "&gt;") &&
			!strings.Contains(bare[i:], "&quot;") {
			t.Fatalf("raw ampersand for %q:\n%s", obj.Name, out)
		}
		if strings.Count(out, `<a href="`) != strings.Count(out, "</a>") {
			t.Fatalf("unbalanced anchor for %q:\n%s", obj.Name, out)
		}
		if strings.Count(out, codeOpen) != strings.Count(out, codeClose) {
			t.Fatalf("unbalanced code span for %q:\n%s", obj.Name, out)
		}
		links += strings.Count(out, "<a href=\"")
		codes += strings.Count(out, codeOpen)

		// leftover markdown link syntax means the parser gave up
		if regexp.MustCompile(`\]\(https?://`).MatchString(out) {
			t.Fatalf("unconverted markdown link in %s %q:\n%s", obj.Type, obj.Name, out)
		}
	}

	if descriptions == 0 {
		t.Fatal("no descriptions parsed, the fixture is wrong")
	}
	t.Logf("checked %d descriptions, produced %d anchors and %d code spans", descriptions, links, codes)
}
