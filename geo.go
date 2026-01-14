package main

import (
	"encoding/json"
	"net/http"
	"regexp"
)

// geoLookup performs a geo lookup by grid and returns the JCC and address associated with the grid.
// The lookup is performed by querying the 430SSB website with the grid as a parameter.
// The response is expected to be JSON containing an array of strings, where the first string is the address associated with the grid,
// and the second string is the JCC associated with the grid.
// If the response is not JSON, or the array is empty, an empty string is returned for both JCC and address.
// If the response is JSON and the array is not empty, the first string in the array is returned as the address and the second string is used to extract the JCC.
func geoLookup(grid string) (jcc, addr string) {
	resp, err := http.Get("https://www.430ssb.net/search/geo?query=" + grid)
	if err != nil {
		return "", ""
	}
	defer resp.Body.Close()

	var arr []string
	if err := json.NewDecoder(resp.Body).Decode(&arr); err != nil || len(arr) == 0 {
		return "", ""
	}

	reJCC := regexp.MustCompile(`JCC:\s*(\d+)`)
	if m := reJCC.FindStringSubmatch(arr[0]); len(m) > 1 {
		jcc = m[1]
	}

	addr = arr[0]
	return
}
