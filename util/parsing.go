package util

import (
	"bytes"
	"errors"
	"golang.org/x/net/html"
	"strings"
)

type extractNodeCondition func(string) bool

// ExtractNode will select the first node to match extractNodeCondition, for example
// res, err := ExtractNode(string(content), func(str string) bool { return str == "title" })
func ExtractNode(content string, fn extractNodeCondition) (*html.Node, error) {
	doc, _ := html.Parse(strings.NewReader(string(content)))
	var n *html.Node
	var crawler func(*html.Node)

	crawler = func(node *html.Node) {
		if node.Type == html.ElementNode && fn(node.Data) {
			n = node
			return
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			crawler(child)
		}
	}
	crawler(doc)
	if n != nil {
		return n, nil
	}
	return nil, errors.New("missing matching tag in the node tree")
}

func ExtractNodeText(n *html.Node, buf *bytes.Buffer) {
	if n.Type == html.TextNode {
		buf.WriteString(n.Data)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		ExtractNodeText(c, buf)
	}
}
