package telegram

import "testing"

func TestFormatDigestHTML(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "plain prose is untouched",
			in:   "Adversaries may steal or forge certificates.",
			want: "Adversaries may steal or forge certificates.",
		},
		{
			name: "bare ampersand is escaped",
			in:   "Groups & Companies",
			want: "Groups &amp; Companies",
		},
		{
			name: "stray angle brackets are escaped",
			in:   "a < b > c",
			want: "a &lt; b &gt; c",
		},
		{
			name: "entities are escaped once, not twice",
			in:   "AT&amp;T",
			want: "AT&amp;amp;T",
		},
		{
			name: "inline link with label",
			in:   "[Unsecured Credentials](https://attack.mitre.org/techniques/T1552)",
			want: `<a href="https://attack.mitre.org/techniques/T1552">Unsecured Credentials</a>`,
		},
		{
			name: "link surrounded by prose",
			in:   "may enable [Lateral Movement](https://attack.mitre.org/tactics/TA0008). Yes.",
			want: `may enable <a href="https://attack.mitre.org/tactics/TA0008">Lateral Movement</a>. Yes.`,
		},
		{
			name: "several links in one paragraph",
			in:   "[Persistence](https://a/1) and [Valid Accounts](https://a/2) too",
			want: `<a href="https://a/1">Persistence</a> and <a href="https://a/2">Valid Accounts</a> too`,
		},
		{
			name: "html code span is kept",
			in:   "the <code>net user /add /domain</code> command",
			want: "the <code>net user /add /domain</code> command",
		},
		{
			name: "code span and link together",
			in:   "run <code>cmd.exe</code> then see [X](https://a/1)",
			want: "run <code>cmd.exe</code> then see <a href=\"https://a/1\">X</a>",
		},
		{
			// found in the real ATT&CK dump: a markdown link inside a code span
			name: "link nested in a code span",
			in:   "access to <code>[at](https://attack.mitre.org/software/S0110)</code> can be managed",
			want: "access to <code><a href=\"https://attack.mitre.org/software/S0110\">at</a></code> can be managed",
		},
		{
			name: "ampersand inside a code span is escaped",
			in:   "<code>a && b</code>",
			want: "<code>a &amp;&amp; b</code>",
		},
		{
			name: "unclosed code span degrades to text",
			in:   "broken <code>never closed",
			want: "broken &lt;code&gt;never closed",
		},
		{
			name: "unknown tag is escaped, not passed through",
			in:   "<script>alert(1)</script>",
			want: "&lt;script&gt;alert(1)&lt;/script&gt;",
		},
		{
			name: "stray bracket stays literal",
			in:   "an array[0] here",
			want: "an array[0] here",
		},
		{
			name: "non http scheme is not linkified",
			in:   "[click](javascript:alert(1))",
			want: "[click](javascript:alert(1))",
		},
		{
			name: "quote in an href is escaped so it cannot break out",
			in:   `[x](https://a/1"onload="y)`,
			want: `<a href="https://a/1&quot;onload=&quot;y">x</a>`,
		},
		{
			name: "empty input",
			in:   "",
			want: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := FormatDigestHTML(tc.in); got != tc.want {
				t.Fatalf("FormatDigestHTML(%q)\n got: %q\nwant: %q", tc.in, got, tc.want)
			}
		})
	}
}

// href ampersands must survive without being double escaped
func TestFormatDigestHTMLEscapesHrefAmpersand(t *testing.T) {
	got := FormatDigestHTML("[x](https://a/1?b=2&c=3)")
	want := `<a href="https://a/1?b=2&amp;c=3">x</a>`
	if got != want {
		t.Fatalf("\n got: %q\nwant: %q", got, want)
	}
}

func TestEscapeHTML(t *testing.T) {
	if got := EscapeHTML("a & b < c > d"); got != "a &amp; b &lt; c &gt; d" {
		t.Fatalf("got %q", got)
	}
	// no reserved chars must return the input unchanged, not a rebuilt copy
	if got := EscapeHTML("plain text"); got != "plain text" {
		t.Fatalf("got %q", got)
	}
}
