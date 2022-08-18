package main

import (
	"fmt"
	"github.com/fatih/color"
	"log"
	"net/url"
	"strings"
	"sync"
)

var wg = sync.WaitGroup{}

func initiateCrawl(u string) {
	baseUrl, err := url.Parse(u)
	if err != nil {
		log.Fatal(err)
	}
	// Mark absolute base path as handled
	markHrefAsHandled(baseUrl)

	// Mark the root path as handled if the given baseUrl is the home page
	if baseUrl.Path == "/" || baseUrl.Path == "" {
		rootSlashPath, err := url.Parse("/")
		if err != nil {
			log.Fatal(err)
		}
		markHrefAsHandled(rootSlashPath)
	}

	getAndCrawlHref(baseUrl.String())
}

func handleHrefs(hrefs []AnchorTag) {

	// loop through and handle all links, mark them internal/external/whatever else
	for _, href := range hrefs {
		if !href.HrefExists {
			handleHrefDoesNotExist(href)
			continue
		}
		u, err := url.Parse(href.HrefValue)
		if err != nil {
			fmt.Println(err)
			continue
		}
		u.Fragment = "" // Remove fragments, since web servers do not respond to them, it is browser-only.
		if u.String() == "" {
			//fmt.Println("Empty href (either initially or after removing fragments):", href)
			continue
		}
		previouslyHandled := markHrefAsHandled(u)
		if previouslyHandled {
			continue
		}
		if !isHttpBasedHref(u) {
			continue
		}
		//fmt.Println("\033[31m", u.String(), href, "\033[0m")
		// Now we can kick off the fetching of the href/page, how we do so depends on if it is internal or external.

		isInternal := isInternalHref(u)
		if isInternal {
			// Need to do this because u.String() might just be "/something" or "something"
			actualUrlToGet := actualUrlToGet(u)
			wg.Add(1)
			go func() {
				getAndCrawlHref(actualUrlToGet)
				wg.Done()
			}()
		} else {
			wg.Add(1)
			go func() {
				handleExternalHref(u.String())
				wg.Done()
			}()
		}
	}
}

// Need to do this because u.String() might just be "/something" or "something"
func actualUrlToGet(u *url.URL) string {
	if u.Scheme == "http" || u.Scheme == "https" {
		return u.String()
	}
	if strings.HasPrefix(u.String(), "/") {
		return initialScheme + "://" + initialHost + u.String()
	}
	return initialScheme + "://" + initialHost + "/" + u.String()
}

func markHrefAsHandled(u *url.URL) bool {
	// Mark the href as handled
	urlsHandledMutex.mu.Lock()
	defer urlsHandledMutex.mu.Unlock()

	if timesSeen, ok := urlsHandledMutex.urls[u.String()]; ok {
		urlsHandledMutex.urls[u.String()] = timesSeen + 1
		return true
	}
	urlsHandledMutex.urls[u.String()] = 1
	return false
}

func handleHrefDoesNotExist(href AnchorTag) {
	anchorTagsWithoutHrefMutex.mu.Lock()
	anchorTagsWithoutHrefMutex.Tags = append(anchorTagsWithoutHrefMutex.Tags, href)
	anchorTagsWithoutHrefMutex.mu.Unlock()
	color.Set(color.FgRed)
	fmt.Println("No href on <a>, found on: ", href.FoundOn)
	color.Unset()
}

func handleExternalHref(url string) {
	resp, err := getHref(url)
	if err != nil {
		fmt.Println(err)
		return
	}
	externalPagesVisitedMutex.mu.Lock()
	externalPagesVisitedMutex.VisitedPages = append(externalPagesVisitedMutex.VisitedPages, VisitedPage{
		GivenHref:  url,
		Url:        resp.Request.URL.String(),
		StatusCode: resp.StatusCode,
	})
	externalPagesVisitedMutex.mu.Unlock()

	color.Set(color.FgWhite)
	fmt.Println(resp.StatusCode, resp.Request.URL.String())
	color.Unset()
}
