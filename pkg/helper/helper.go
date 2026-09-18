package helper

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

func GetJSON(uri string) ([]byte, error) {
	return GetJSONWithHeaders(uri, nil)
}

func GetJSONWithHeaders(uri string, headers map[string]string) ([]byte, error) {
	if strings.HasPrefix(uri, "http") {
		if headers == nil {
			headers = map[string]string{}
		}
		if _, exists := headers["User-Agent"]; !exists {
			headers["User-Agent"] = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/127.0.0.0 Safari/537.36"
		}

		request, err := http.NewRequest(http.MethodGet, uri, nil)
		if err != nil {
			return nil, fmt.Errorf("cannot create request for %q: %v", uri, err)
		}
		for key, value := range headers {
			request.Header.Set(key, value)
		}

		resp, err := http.DefaultClient.Do(request)
		if err != nil {
			return nil, fmt.Errorf("cannot fetch URL %q: %v", uri, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			if _, exists := headers["User-Agent"]; exists && headers["User-Agent"] != "" {
				request.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)")
				resp.Body.Close()
				resp, err = http.DefaultClient.Do(request)
				if err != nil {
					return nil, fmt.Errorf("cannot fetch URL %q: %v", uri, err)
				}
				defer resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					return nil, fmt.Errorf("unexpected http GET status: %s", resp.Status)
				}
			} else {
				return nil, fmt.Errorf("unexpected http GET status: %s", resp.Status)
			}
		}

		bytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("unable read response body %s", err.Error())
		}
		return bytes, nil
	}
	bytes, err := os.ReadFile(uri)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}
