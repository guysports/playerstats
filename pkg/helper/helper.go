package helper

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

func GetJSON(uri string) ([]byte, error) {
	if strings.HasPrefix(uri, "https") {
		// Read data from URI
		resp, err := http.Get(uri)
		if err != nil {
			return nil, fmt.Errorf("cannot fetch URL %q: %v", uri, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected http GET status: %s", resp.Status)
		}

		// We could check the resulting content type
		// here if desired.
		bytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("unable read response body %s", err.Error())
		}
		return bytes, nil
	}
	// Read data from file
	bytes, err := ioutil.ReadFile(uri)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}
