// Package aur provides an interface to https://aur.archlinux.org/rpc.php
package aur

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"
)

const (
	urlBase = "https://aur.archlinux.org"

	// Used to put together info requests for batches of names
	namePrefix    = "&arg[]="
	namePrefixLen = len(namePrefix)
)

// PkgInfo contains information about an AUR package.
type PkgInfo struct {
	Name        string
	Version     string
	SnapshotURL string
}

// GetInfos queries the AUR for every name in allNames.  The result
// map will not contain an entry for every input if the AUR didn't
// return anything for the package.  Perhaps this happens if the
// package has been removed?  If an error is returned, we just return
// that one error.
func GetInfos(allNames []string) (map[string]*PkgInfo, error) {
	result := map[string]*PkgInfo{}
	nameBatches := escapeAndBatch(1024, allNames)
	for i, names := range nameBatches {
		infoBatch, err := fetch(names)
		if err != nil {
			return result, errors.Wrapf(err, "Fetching batch %d with %d entries", i, len(names))
		}
		for _, info := range infoBatch {
			result[info.Name] = info
		}
	}
	return result, nil
}

// breaks a single slice of names into a slice of slices, based on
// size of escaped name.  Escapes the names as part of this process.,
// attempting to keep each batch below maxSize (if one name would go
// over maxSize, we create a batch with just that name).
func escapeAndBatch(maxSize int, allNames []string) [][]string {
	result := [][]string{}
	batch := []string{}
	batchSize := 0
	for _, name := range allNames {
		name = url.QueryEscape(name)
		delta := namePrefixLen + len(name)
		if batchSize+delta > maxSize && len(batch) > 0 {
			result = append(result, batch)
			batch = []string{}
			batchSize = 0
		}
		batch = append(batch, name)
		batchSize += delta
	}
	if len(batch) > 0 {
		result = append(result, batch)
	}
	return result
}

func fetch(names []string) ([]*PkgInfo, error) {
	argString := namePrefix + strings.Join(names, namePrefix)
	query := fmt.Sprintf("v=5&type=info%s", argString)
	url := fmt.Sprintf("%s/rpc/?%s", urlBase, query)
	resp, err := http.Get(url)
	if err != nil {
		return nil, errors.Wrapf(err, "fetch get for %v", names)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("fetch got unexpected status %d/%s for %v", resp.StatusCode, resp.Status, names)
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrapf(err, "fetch readbody for %v", names)
	}
	return decodeResults(data)

}

func decodeResults(data []byte) ([]*PkgInfo, error) {
	result := []*PkgInfo{}
	var response infoResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return result, errors.Wrapf(err, "unmarshal %s", string(data))
	}

	for _, r := range response.Results {
		result = append(result, r.makePkgInfo())
	}
	return result, nil
}

// Used to decode json rpc response for GetVersion call.
type infoResponse struct {
	Version     int
	Type        string
	ResultCount int
	Results     []infoResult
}

// Used to decode result portion of json response.
type infoResult struct {
	Name    string
	Version string
	URLPath string
}

func (r infoResult) makePkgInfo() *PkgInfo {
	return &PkgInfo{r.Name, r.Version, urlBase + r.URLPath}
}
