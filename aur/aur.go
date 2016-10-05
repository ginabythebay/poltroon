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

const urlBase = "https://aur.archlinux.org"

// PkgInfo contains information about an AUR package.
type PkgInfo struct {
	Version     string
	SnapshotURL string
}

// GetInfo queries the AUR for information for the named package.
func GetInfo(name string) (*PkgInfo, error) {
	query := fmt.Sprintf("v=5&type=info&arg[]=%s", url.QueryEscape(name))
	url := fmt.Sprintf("%s/rpc/?%s", urlBase, query)
	resp, err := http.Get(url)
	if err != nil {
		return nil, errors.Wrap(err, "GetVersion get")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("GetVersion got unexpected status %d/%s", resp.StatusCode, resp.Status)
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "GetVersion readbody")
	}
	return decodeInfo(data)
}

func decodeInfo(data []byte) (*PkgInfo, error) {
	var response infoResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, errors.Wrapf(err, "unmarshal %s", string(data))
	}
	if len(response.Results) == 0 {
		// this happens sometimes.  I'm not sure why.
		return &PkgInfo{"", ""}, nil
	}
	if len(response.Results) != 1 {
		return nil, errors.Errorf("Error parsing %s.  We expected 1 result and found %d results", string(data), len(response.Results))
	}
	version, err := response.Results[0].field("Version")
	if err != nil {
		return nil, errors.Wrap(err, "extract version")
	}
	partialURL, err := response.Results[0].field("URLPath")
	if err != nil {
		return nil, errors.Wrap(err, "extract URLPath")
	}
	snapshotURL := fmt.Sprintf("%s%s", urlBase, partialURL)
	return &PkgInfo{version, snapshotURL}, nil
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

func (r infoResult) field(name string) (string, error) {
	value, ok := r[name]
	if !ok {
		return "", errors.Errorf("Missing %s in %v", name, r)
	}
	s, ok := value.(string)
	if !ok {
		return "", errors.Errorf("Invalid result %v.  Expected string %s but got %q", r, name, value)
	}
	return s, nil
}
