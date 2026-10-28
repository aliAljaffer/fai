package jira

import (
	"encoding/json"
	"strings"
)

type adfMark struct {
	Type  string         `json:"type"`
	Attrs map[string]any `json:"attrs"`
}

type adfNode struct {
	Type    string         `json:"type"`
	Text    string         `json:"text"`
	Attrs   map[string]any `json:"attrs"`
	Content []adfNode      `json:"content"`
	Marks   []adfMark      `json:"marks"`
}

// rawToMarkdown accepts either an ADF document (Jira Cloud v3) or a plain
// string (Jira Server and older payloads).
func rawToMarkdown(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var text string
	if json.Unmarshal(raw, &text) == nil {
		return strings.TrimSpace(text)
	}
	var doc adfNode
	if json.Unmarshal(raw, &doc) != nil {
		return ""
	}
	var b strings.Builder
	renderADFBlock(&b, doc, 0)
	return strings.TrimSpace(b.String())
}

func adfAttr(node adfNode, key string) string {
	if v, ok := node.Attrs[key].(string); ok {
		return v
	}
	return ""
}

func renderADFBlock(b *strings.Builder, node adfNode, indent int) {
	pad := strings.Repeat("  ", indent)
	switch node.Type {
	case "doc":
		for _, child := range node.Content {
			renderADFBlock(b, child, indent)
		}
	case "paragraph":
		if text := adfInline(node.Content); text != "" {
			b.WriteString(pad + text + "\n\n")
		}
	case "heading":
		level := 1
		if v, ok := node.Attrs["level"].(float64); ok && v >= 1 && v <= 6 {
			level = int(v)
		}
		b.WriteString(pad + strings.Repeat("#", level) + " " + adfInline(node.Content) + "\n\n")
	case "bulletList", "orderedList":
		marker := "- "
		if node.Type == "orderedList" {
			marker = "1. "
		}
		for _, item := range node.Content {
			var parts []string
			var nested []adfNode
			for _, child := range item.Content {
				switch child.Type {
				case "paragraph":
					parts = append(parts, adfInline(child.Content))
				case "bulletList", "orderedList":
					nested = append(nested, child)
				default:
					parts = append(parts, adfInline([]adfNode{child}))
				}
			}
			b.WriteString(pad + marker + strings.TrimSpace(strings.Join(parts, " ")) + "\n")
			for _, child := range nested {
				renderADFBlock(b, child, indent+1)
			}
		}
		b.WriteString("\n")
	case "codeBlock":
		lang := adfAttr(node, "language")
		b.WriteString(pad + "```" + lang + "\n" + adfPlain(node.Content) + "\n" + pad + "```\n\n")
	case "blockquote":
		var inner strings.Builder
		for _, child := range node.Content {
			renderADFBlock(&inner, child, 0)
		}
		for _, line := range strings.Split(strings.TrimSpace(inner.String()), "\n") {
			b.WriteString(pad + "> " + line + "\n")
		}
		b.WriteString("\n")
	case "rule":
		b.WriteString(pad + "---\n\n")
	case "hardBreak":
		b.WriteString("\n")
	case "text":
		b.WriteString(adfInline([]adfNode{node}))
	default:
		for _, child := range node.Content {
			renderADFBlock(b, child, indent)
		}
	}
}

func adfInline(nodes []adfNode) string {
	var b strings.Builder
	for _, node := range nodes {
		switch node.Type {
		case "text":
			text := node.Text
			for _, mark := range node.Marks {
				switch mark.Type {
				case "strong":
					text = "**" + text + "**"
				case "em":
					text = "*" + text + "*"
				case "strike":
					text = "~~" + text + "~~"
				case "code":
					text = "`" + text + "`"
				case "link":
					if href, ok := mark.Attrs["href"].(string); ok {
						text = "[" + text + "](" + href + ")"
					}
				}
			}
			b.WriteString(text)
		case "hardBreak":
			b.WriteString("\n")
		case "mention":
			if text := adfAttr(node, "text"); text != "" {
				b.WriteString(text)
			} else {
				b.WriteString("@" + adfAttr(node, "id"))
			}
		case "emoji":
			b.WriteString(adfAttr(node, "text"))
		case "inlineCard":
			b.WriteString(adfAttr(node, "url"))
		default:
			if node.Text != "" {
				b.WriteString(node.Text)
			} else {
				b.WriteString(adfInline(node.Content))
			}
		}
	}
	return b.String()
}

func adfPlain(nodes []adfNode) string {
	var b strings.Builder
	for _, node := range nodes {
		if node.Text != "" {
			b.WriteString(node.Text)
		}
		b.WriteString(adfPlain(node.Content))
	}
	return b.String()
}
