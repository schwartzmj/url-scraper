package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/fatih/color"

	"github.com/PuerkitoBio/goquery"
)

// isHttpBasedHref returns true if the href is a http(s) based href, i.e. it returns false for things like ftp:, mailto:, or tel:
func isHttpBasedHref(url *url.URL) bool {
	// TODO: Cloudflare turns mailto: into https://support.cloudflare.com/hc/en-us/categories/200275218-Getting-Started or https://support.cloudflare.com/hc/en-us/articles/200170016-What-is-Email-Address-Obfuscation-
	// So then we visit them, and they error. How do we handle this? We don't want to ignore all external links?
	// Maybe we can have the user enter the URL and the IP and visit that way, bypassing Cloudflare?

	// If scheme is empty, then it is either a relative or absolute path, so it is http(s) based.
	if url.Scheme == "" {
		return true
	}
	// If scheme exists, and it is http(s), then it is http(s) based
	return url.Scheme == "http" || url.Scheme == "https"
}

// isInternalHref expects an url.URL object that has already been confirmed to be http(s) based. It returns true if the url is internal, false otherwise (external).
func isInternalHref(u *url.URL) bool {
	// If host is empty, then it is either a relative or absolute path, so it is internal.
	if u.Host == "" {
		return true
	}
	// If Host exists, and it is the same as the initial host, then it is internal.
	if strings.EqualFold(u.Host, initialHost) {
		return true
	}
	// Otherwise it is an external href
	return false
}

func getHref(link string) (*http.Response, error) {
	client := &http.Client{
		Timeout: time.Second * 10,
		//CheckRedirect: func(req *http.Request, via []*http.Request) error {
		//	return http.ErrUseLastResponse
		//},
	}
	req, err := http.NewRequest("GET", link, nil)
	if err != nil {
		return &http.Response{}, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/75.0.3770.100 Safari/537.36")
	resp, err := client.Do(req)
	if err != nil {
		return &http.Response{}, err
	}
	return resp, nil
}

func getAndCrawlHref(href string) {
	resp, err := getHref(href)
	//defer resp.Body.Close()

	if err != nil {
		fmt.Println(err)
		return
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		fmt.Println(err)
		return
	}

	internalPagesVisitedMutex.mu.Lock()
	internalPagesVisitedMutex.VisitedPages = append(internalPagesVisitedMutex.VisitedPages, VisitedPage{
		GivenHref:  href,
		Url:        resp.Request.URL.String(),
		StatusCode: resp.StatusCode,
	})
	internalPagesVisitedMutex.mu.Unlock()

	color.Set(color.FgCyan)
	fmt.Println(resp.StatusCode, resp.Request.URL.Path)
	color.Unset()

	anchorTags := getAnchorTagsAndHrefAttribute(doc, resp.Request.URL.String())

	handleHrefs(anchorTags)
	// TODO: why can I not use defer at the top? Seems like resp.Body does not exist anymore if I use defer at the top.
	io.Copy(ioutil.Discard, resp.Body)
	resp.Body.Close()
}

// getLinks gets all anchor tag hrefs from the page and returns an array of each href value
func getAnchorTagsAndHrefAttribute(doc *goquery.Document, currentUrl string) []AnchorTag {
	var anchorTags []AnchorTag
	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")

		innerText := ""
		if !exists {
			// get the inner text for the anchor tag if it doesn't have an href attribute, so user can find it easier
			innerText = strings.TrimSpace(s.Text())
			if len(innerText) > 50 {
				innerText = innerText[:50] + "..."
			}
		}

		anchorTags = append(anchorTags, AnchorTag{HrefValue: href, HrefExists: exists, InnerTextForNonExistentHref: innerText, FoundOn: currentUrl})
	})
	return anchorTags
}
