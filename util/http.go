package util

import (
	"github.com/5HT2/taro-bot/bot"
	"github.com/5HT2C/http-bash-requests/httpBashRequests"
	"io"
	"net/http"
	"time"
)

// RequestUrl will return the bytes of the body of url
func RequestUrl(url string, method string) ([]byte, *http.Response, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, nil, err
	}

	return RequestUrlReq(req)
}

// RequestUrlFn will execute fn on req, then return the bytes of the body of url
func RequestUrlFn(url string, method string, fn func(req *http.Request)) ([]byte, *http.Response, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, nil, err
	}
	fn(req)

	return RequestUrlReq(req)
}

// RequestUrlReq will return the bytes of the body of request
func RequestUrlReq(req *http.Request) ([]byte, *http.Response, error) {
	res, err := bot.HttpClient.Do(req)
	if err != nil {
		return nil, nil, err
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, err := io.ReadAll(res.Body)
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

func RegisterHttpBashRequests() {
	client := httpBashRequests.Client{Addr: "http://localhost:6016", HttpClient: &http.Client{Timeout: 5 * time.Minute}}
	httpBashRequests.Setup(&client)
}
