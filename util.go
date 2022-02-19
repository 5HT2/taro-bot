package main

import (
	"bytes"
	"errors"
	"golang.org/x/net/html"
	"image/color"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
)

var (
	imageExtensions = []string{".jpg", ".jpeg", ".png", ".gif", ".gifv"}
)

type retryFunction func() ([]byte, error)

func LogPanic() {
	if x := recover(); x != nil {
		// recovering from a panic; x contains whatever was passed to panic()
		log.Printf("runtime panic: %v", x)
		panic(x)
	}
}

// RetryFunc will re-try fn by n number of times, in addition to one regular try
func RetryFunc(fn retryFunction, n int) ([]byte, error) {
	if n < 0 {
		n = 0
	}

	for n > 0 {
		b, err := fn()
		if err == nil {
			return b, err
		}
		n--
	}

	return fn()
}

func FileExtMatches(s []string, file string) bool {
	found := false
	file = strings.ToLower(file)

	for _, e := range s {
		if filepath.Ext(file) == e {
			found = true
			break
		}
	}

	return found
}

// GetUserMention will return a formatted user mention from an id
func GetUserMention(id int64) string {
	return "<@!" + strconv.FormatInt(id, 10) + ">"
}

// JoinInt64Slice will join i with sep
func JoinInt64Slice(i []int64, sep string, prefix string, suffix string) string {
	elems := make([]string, 0)
	for _, e := range i {
		elems = append(elems, prefix+strconv.FormatInt(e, 10)+suffix)
	}
	return strings.Join(elems, sep)
}

// StringSliceContains will return if slice s contains e
func StringSliceContains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// Int64SliceContains will return true if slice s contains e
func Int64SliceContains(s []int64, e int64) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// Int64SliceRemove will remove i from s
func Int64SliceRemove(s []int64, i int64) []int64 {
	ns := make([]int64, 0)
	for _, in := range s {
		if in != i {
			ns = append(ns, in)
		}
	}
	return ns
}

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

func RenderNode(n *html.Node) string {
	var buf bytes.Buffer
	w := io.Writer(&buf)
	html.Render(w, n)
	return buf.String()
}

func ExtractNodeText(n *html.Node, buf *bytes.Buffer) {
	if n.Type == html.TextNode {
		buf.WriteString(n.Data)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		ExtractNodeText(c, buf)
	}
}

// RequestUrl will return the bytes of the body of url
func RequestUrl(url string, method string) ([]byte, *http.Response, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, nil, err
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, nil, err
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, nil, err
	}

	return body, res, nil
}

// RequestUrlRetry will return the bytes of the body of the first successful url
func RequestUrlRetry(urls []string, method string, code int) (bytes []byte) {
	for _, url := range urls {
		content, res, err := RequestUrl(url, method)
		if err == nil && res.StatusCode == code {
			return content
		}
	}

	return nil
}

// ConvertColorToInt32 will convert 3 uint8s into one int32
func ConvertColorToInt32(c color.RGBA) int32 {
	return int32((uint32(c.R) << 16) | (uint32(c.G) << 8) | (uint32(c.B) << 0))
}

// ParseHexColorFast will take a hex string, and convert it to a color.RGBA
func ParseHexColorFast(s string) (c color.RGBA, err error) {
	c.A = 0xff

	if s[0] != '#' {
		return c, GenericError("ParseHexColorFast", "parsing \""+s+"\"", "missing #")
	}

	hexToByte := func(b byte) byte {
		switch {
		case b >= '0' && b <= '9':
			return b - '0'
		case b >= 'a' && b <= 'f':
			return b - 'a' + 10
		case b >= 'A' && b <= 'F':
			return b - 'A' + 10
		}
		err = SyntaxError("ParseHexColorFast", s)
		return 0
	}

	switch len(s) {
	case 7:
		c.R = hexToByte(s[1])<<4 + hexToByte(s[2])
		c.G = hexToByte(s[3])<<4 + hexToByte(s[4])
		c.B = hexToByte(s[5])<<4 + hexToByte(s[6])
	case 4:
		c.R = hexToByte(s[1]) * 17
		c.G = hexToByte(s[2]) * 17
		c.B = hexToByte(s[3]) * 17
	default:
		err = SyntaxError("ParseHexColorFast", s)
	}
	return
}
