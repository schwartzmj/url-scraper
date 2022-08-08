package main

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type Page struct {
	Title      string
	Url        string
	Host       string
	Path       string
	Scheme     string
	Links      []string
	StatusCode int
	Redirects  []string // eventually make this an array of Redirect struct itself (from, to, status code)?
}

type NormalizedLinkResult struct {
	Link string
	IsAlias bool
	Skip bool
	NonRelative bool
	Err  error
}

type GetPageResult struct {
	Page    Page
	Skipped bool
	Err     error
}

// get makes an HTTP GET request to the specified URL and returns a Page struct and a boolean indicating whether the page was skipped.
func get(url string) GetPageResult {
	normalizedLinkResult := normalizeLink(url)

	// fmt.Println(urlsHandledMutex.urls, normalizedLinkResult.Link)
	if normalizedLinkResult.Err != nil {
		return GetPageResult{Err: normalizedLinkResult.Err}
	}
	if normalizedLinkResult.Skip {
		return GetPageResult{Skipped: true}
	}

	var redirects []string

	client := &http.Client{
		Timeout: time.Second * 10,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			redirects = append(redirects, req.URL.String())
			return nil
		},
	}

	req, err := http.NewRequest("GET", normalizedLinkResult.Link, nil)
	if err != nil {
		return GetPageResult{Err: err}
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/75.0.3770.100 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return GetPageResult{Err: err}

	}
	defer resp.Body.Close()

	var statusCode int
	// if len(redirects) > 0 {
	// 	statusCode = http.StatusMultipleChoices
	// } else {
	// 	statusCode = resp.StatusCode
	// }

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return GetPageResult{Err: err}
	}

	// Switch resp.StatusCode. fmt.Println in colorGreen if it is 200, in colorYellow if it is 300, and in colorRed if it is anything else.
	var statusCodeText string

	if len(redirects) > 0 {
		statusCode = 300
	} else {
		statusCode = resp.StatusCode
	}

	switch statusCode {
	case http.StatusOK:
 		statusCodeText += "\033[32m" + strconv.Itoa(statusCode) + "\033[0m"
	case http.StatusMultipleChoices:
 		statusCodeText += "\033[33m" + strconv.Itoa(statusCode) + "\033[0m"
	default:
 		statusCodeText += "\033[31m" + strconv.Itoa(statusCode) + "\033[0m"
	}

	var otherNotesText string
	if normalizedLinkResult.IsAlias {
		otherNotesText += " \033[33m" + "(alias)" +"\033[0m"
	}
	if normalizedLinkResult.NonRelative {
		otherNotesText += " \033[31m" + "(non-relative)" +"\033[0m"
	}
	fmt.Println(statusCodeText, resp.Request.URL.Path, otherNotesText)

	links := getLinks(doc)
	return GetPageResult{
		Page: Page{
			Title:      doc.Find("title").Text(),
			Url:        normalizedLinkResult.Link,
			Host:       resp.Request.URL.Host,
			Path:       resp.Request.URL.Path,
			Scheme:     resp.Request.URL.Scheme,
			Links:      links,
			StatusCode: statusCode,
			Redirects:  redirects,
		},
	}
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

func getHostAlias(host string) string {
	// If host starts with www., return the host without the www.
	// If host does not start with www., return the host with www.
	if host[:4] == "www." {
		return host[4:]
	}
	return "www." + host
}

// normalizeLink takes a link and returns a normalized link and a boolean indicating whether the link should be skipped. It also takes care of adding to the urlsHandled map.
func normalizeLink(link string) NormalizedLinkResult {
	normalizedLink := NormalizedLinkResult{}

	urlsHandledMutex.mu.Lock()
	defer urlsHandledMutex.mu.Unlock()
	// Return early if we've already handled the non-normalized link
	if urlsHandledMutex.urls[link] {
		normalizedLink.Skip = true
		return normalizedLink
	}

	u, err := url.Parse(link)
	if err != nil {
		normalizedLink.Err = err
		return normalizedLink
	}

	// Remove any fragments from the path
	u.Fragment = ""
	// Remove any query params from the path
	u.RawQuery = ""

	// Now that we've parsed the url, check if we've visited the path. We have to check both / and non-/ paths.
	// TODO: I think we can add a custom function to the struct to do this logic?
	// We use this temporaryCheck variable at the end of this function too.
	noTrailingSlashPath := u.Path
	noTrailingSlashPath = strings.TrimSuffix(noTrailingSlashPath, "/")
	if urlsHandledMutex.urls[noTrailingSlashPath] || urlsHandledMutex.urls[noTrailingSlashPath+"/"] {
		normalizedLink.Skip = true
		return normalizedLink
	}

	// Now that we've normalized the link without fragments, query params check if we've already handled it
	// if urlsHandledMutex.urls[u.String()] {
	// 	normalizedLink.Skip = true
	// 	return normalizedLink
	// }

	// If the URL is not absolute, then we need to add the initialHost to it
	// If the URL is absolute (it has a scheme), then we mark it non-relative for informational purposes.
	if u.Host=="" {
		u.Host = initialHost
	} else if len(urlsHandledMutex.urls) == 0 {
		// this is initial crawl, so we will not mark it non-relative since it is the full url
	} else {
		// NOTE: marking non-relative isnt all that helpful considering it is only the first time we visit the link. would be more helpful to find all links on the site that arent relative instead of just this one instance
		normalizedLink.NonRelative = true
	}

	// Now that we've added a host to all relative links, we can confirm that the link is an internal link.
	if u.Host != initialHost {
		// Check if the link is an alias (www. or root domain depending on our initialHost). If so, consider it valid, but mark it as an alias.
		if getHostAlias(initialHost) == u.Host {
			normalizedLink.IsAlias = true
		} else {
			// If the host is not initialHost nor an alias, then it is external to the site and we should ignore it.
			urlsHandledMutex.urls[link] = true
			normalizedLink.Skip = true
			return normalizedLink
		}
	}


	// Fully normalized link with scheme and host. We check once more if we've already handled it.
	// if urlsHandledMutex.urls[u.String()] {
	// 	normalizedLink.Skip = true
	// 	return normalizedLink
	// }
	// Set the normalized link as handled
	// urlsHandledMutex.urls[u.String()] = true
	// Set the Path as handled (TODO: should we just be doing this instead of u.String()?)
	// urlsHandledMutex.urls[u.Path] = true

	// Set the initial, unnormalized link as handled so we can break early out of this function in the future.
	urlsHandledMutex.urls[link] = true
	// Add the path to our handled list. Add both trailing slash and non-trailing slash. This is our primary check alongside the un-normalized link.
	urlsHandledMutex.urls[noTrailingSlashPath] = true
	urlsHandledMutex.urls[noTrailingSlashPath+"/"] = true

	u.Scheme = initialScheme // set the scheme to the initial scheme (if it started relative it would just be host+path without https/http)
	normalizedLink.Link = u.String()
	return normalizedLink

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
