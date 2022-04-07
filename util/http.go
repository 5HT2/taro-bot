package util

import (
	"github.com/5HT2/taro-bot/bot"
	"io/ioutil"
	"net/http"
)

// RequestUrl will return the bytes of the body of url
func RequestUrl(url string, method string) ([]byte, *http.Response, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, nil, err
	}

	res, err := bot.HttpClient.Do(req)
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
