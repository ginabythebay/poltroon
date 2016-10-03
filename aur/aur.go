// Package aur provides an interface to https://aur.archlinux.org/rpc.php
package aur

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
)

// GetVersion queries the AUR for the version for the named package.
func GetVersion(name string) (version string, err error) {
	query := fmt.Sprintf("v=5&type=info&arg[]=%s", url.QueryEscape(name))
	url := fmt.Sprintf("https://aur.archlinux.org/rpc/?%s", query)
	fmt.Println(url)
	resp, err := http.Get(url)
	if err != nil {
		return "", errors.Wrap(err, "GetVersion get")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", errors.Errorf("GetVersion got unexpected status %d/%s", resp.StatusCode, resp.Status)
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "GetVersion readbody")
	}
	return decodeVersion(data)
}

func decodeVersion(data []byte) (string, error) {
	var response infoResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return "", errors.Wrapf(err, "unmarshal %s", string(data))
	}
	if len(response.Results) == 0 {
		// this happens sometimes.  I'm not sure why.
		return "", nil
	}
	if len(response.Results) != 1 {
		return "", errors.Errorf("Error parsing %s.  We expected 1 result and found %d results", string(data), len(response.Results))
	}
	version, err := response.Results[0].version()
	if err != nil {
		return "", errors.Wrap(err, "extract version")
	}
	return version, nil
}

// Used to decode json rpc response for GetVersion call.
type infoResponse struct {
	Version     int
	Type        string
	ResultCount int
	Results     []infoResult
}

// Contains a lot of glop we don't care about here.
type infoResult map[string]interface{}

func (r infoResult) version() (version string, err error) {
	value, ok := r["Version"]
	if !ok {
		return "", errors.Errorf("Missing Version in %v", r)
	}
	s, ok := value.(string)
	if !ok {
		return "", errors.Errorf("Invalid result %v.  Expected string Version but got %q", r, value)
	}
	return s, nil
}
