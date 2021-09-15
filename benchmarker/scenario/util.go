package scenario

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/isucon/isucandar/failure"

	"github.com/isucon/isucon11-final/benchmarker/fails"
)

var linkRegexp = regexp.MustCompile(`(?i)<([^>]+)>;\s+rel="([^"]+)"`)

// parseLinkHeader は Link header をパースする
func parseLinkHeader(hres *http.Response) (prev string, next string, err error) {
	if hres == nil {
		return
	}

	linkHeader := hres.Header.Get("Link")
	if linkHeader == "" {
		return
	}

	// 意図的にフロントの実装 link_helper.ts に合わせている
	links := strings.Split(linkHeader, ",")
	for _, link := range links {
		linkInfo := linkRegexp.FindStringSubmatch(link)
		if linkInfo != nil && (linkInfo[2] == "prev" || linkInfo[2] == "next") {
			u, err := url.Parse(linkInfo[1])
			if err != nil {
				return "", "", failure.NewError(fails.ErrApplication, fmt.Errorf("link header の URL が不正です"))
			}
			s := u.Path + "?" + u.RawQuery
			switch linkInfo[2] {
			case "prev":
				prev = s
			case "next":
				next = s
			}
		}
	}
	return
}

// keysの要素がすべてsの部分文字列であればtrue
func containsAll(s string, keys []string) bool {
	for _, key := range keys {
		if !strings.Contains(s, key) {
			return false
		}
	}
	return true
}
