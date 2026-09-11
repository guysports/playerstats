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
			return nil, fmt.Errorf("unexpected http GET status: %s", resp.Status)
		}

		// We could check the resulting content type
		// here if desired.
		bytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("unable read response body %s", err.Error())
		}
		return bytes, nil
	}
	// Read data from file
	bytes, err := os.ReadFile(uri)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}
