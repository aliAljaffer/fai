package jira

import "testing"

func TestADFToMarkdown(t *testing.T) {
	raw := []byte(`{"type":"doc","version":1,"content":[
	  {"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Title"}]},
	  {"type":"paragraph","content":[
	    {"type":"text","text":"Bold","marks":[{"type":"strong"}]},
	    {"type":"text","text":" and "},
	    {"type":"text","text":"link","marks":[{"type":"link","attrs":{"href":"https://example.com"}}]}
	  ]},
	  {"type":"bulletList","content":[
	    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"One"}]}]},
	    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Two"}]}]}
	  ]},
	  {"type":"codeBlock","attrs":{"language":"go"},"content":[{"type":"text","text":"fmt.Println()"}]}
	]}`)

	got := rawToMarkdown(raw)
	want := "## Title\n\n**Bold** and [link](https://example.com)\n\n- One\n- Two\n\n```go\nfmt.Println()\n```"
	if got != want {
		t.Fatalf("markdown =\n%q\nwant\n%q", got, want)
	}
}
