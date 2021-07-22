package scenario

import (
	"net/http"
	"strings"
)

func parseLinkHeader(hres *http.Response) (prev string, next string) {
	if hres == nil {
		return
	}

	linkHeader := hres.Header.Get("Link")
	if linkHeader == "" {
		return
	}
	links := strings.Split(linkHeader, ",")

	for _, l := range links {
		tags := strings.Split(l, ";")
		if strings.Contains(tags[1], "prev") {
			urlTag := strings.TrimSpace(tags[0])
			prev = urlTag[1:len(urlTag)-1]
		} else if strings.Contains(tags[1], "next") {
			urlTag := strings.TrimSpace(tags[0])
			next = urlTag[1:len(urlTag)-1]
		}
	}
	return
}
