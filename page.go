package main

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type Page struct {
	Title      string
	Url		string
	Links      []string
	StatusCode int
	Redirects  []string
}

// get makes an HTTP GET request to the specified URL and returns a Page struct and a boolean indicating whether the page was skipped.
func get(url string) (Page, bool, error) {
	urlToGet, shouldSkip, err := normalizeLink(url)
	if err != nil {
		return Page{}, false, err
	}
	if shouldSkip {
		return Page{}, true, nil
	}

	var redirects []string

	client := &http.Client{
		Timeout: time.Second * 10,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Push the upcoming redirect onto the redirects slice
			redirects = append(redirects, req.URL.String())
			return nil
		},
	}

	req, err := http.NewRequest("GET", urlToGet, nil)
	if err != nil {
		return Page{}, false, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/75.0.3770.100 Safari/537.36")
	resp, err := client.Do(req)
	if err != nil {
		return Page{}, false, err
	}
	defer resp.Body.Close()
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return Page{}, false, err
	}

	fmt.Println("Successfully crawled: ", urlToGet, " Status code: ", resp.StatusCode)

	links := getLinks(doc)
	return Page{
		Title:      doc.Find("title").Text(),
		Url:        urlToGet,
		Links:      links,
		StatusCode: resp.StatusCode,
		Redirects:  redirects,
	}, false, nil
}

// getLinks gets all links from the page and return a slice of strings
func getLinks(doc *goquery.Document) []string {
	var links []string
	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		link, _ := s.Attr("href")
		links = append(links, link)
	})
	return links
}

// normalizeLink takes a link and returns a normalized link and a boolean indicating whether the link should be skipped. It also takes care of adding to the urlsHandled map.
func normalizeLink(link string) (string, bool, error) {
	// Return early if we've already handled the non-normalized link
	if urlsHandled[link] {
		return "", true, nil
	}

	u, err := url.Parse(link)
	if err != nil {
		return "", false, err
	}

	// Remove any trailing slashes from the path to stay consistent
	u.Path = strings.TrimSuffix(u.Path, "/")
	// Remove any fragments from the path
    u.Fragment = ""
	// Remove any query params from the path
	u.RawQuery = ""

	// Now that we've normalized the link without fragments, query params, and trailing slashes check if we've already handled it
	if urlsHandled[u.String()] {
		return "", true, nil
	}

	// If the URL is not absolute, then we need to add the initialHost to it
	if !u.IsAbs() {
		u.Host = initialHost
	}
	// If the host is not initialHost, then it is external to the site and we should ignore it
	if u.Host != initialHost {
		urlsHandled[u.String()] = true
		return "", true, nil
	}

	u.Scheme = initialScheme

	// Fully normalized link with scheme and host. We check once more if we've already handled it.
	if urlsHandled[u.String()] {
		return "", true, nil
	}
	// Set the normalized link as handled
	urlsHandled[u.String()] = true
	// Set the initial, unnormalized link as handled
	urlsHandled[link] = true

	return u.String(), false, nil
}

// func normalizeAndValidateLink(link string) (string, error) {
// 	// if the link is already visited or ignored, return an error
// 	if urlsHandled[link] {
// 		return "", errors.New("link already visited or ignored: " + link)
// 	}
// 	// if the link has a prefix of / then it is a relative link and we need to add the baseUrl to it
// 	if strings.HasPrefix(link, "/") {
// 		link = baseUrl + link
// 	// if the link does not have a prefix of the baseUrl, then it is not a link to a page on the site
// 	} else if !strings.HasPrefix(link, baseUrl) {
// 		return "", errors.New("error: external link? link must start with '/'. Link is: " + link)
// 	} else {
// 		// if the link is none of the above if statements, then it is a link to a page on the site and we dont have to do anything
// 		// we dont need this block but leaving it here for clarity right now
// 	}

// 	// after normalizing, if the link is already visited or ignored, return an error
// 	if urlsHandled[link] {
// 			return "", errors.New("link already visited or ignored: " + link)
// 	}
// 	// we've handled this link/url, so add it to the urlsHandled map so we do not visit it again
// 	urlsHandled[link] = true

// 	return link, nil
// }
