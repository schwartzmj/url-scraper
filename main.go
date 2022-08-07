package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
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

const baseUrl = "https://www.wemaketechsimple.com"

var urlsHandled = make(map[string]bool)

var pages = []Page{}

func main() {
	// TODO: as of now, we must make sure the baseUrl does NOT have a trailing /
	rand.Seed(time.Now().UnixNano())
	crawl(baseUrl)

	file, _ := json.MarshalIndent(pages, "", " ")
	_ = ioutil.WriteFile("pages.json", file, 0644)

	fmt.Println("Number of pages visited: ", len(pages))
	fmt.Println("Done!")
}

func crawl(url string) {
	page, err := get(url)
	if err != nil {
		// fmt.Println(err)
		return
	}
	// add the page to the pages slice
	pages = append(pages, page)

	if page.StatusCode != http.StatusOK {
		fmt.Println("Error. Status code:", page.StatusCode)
		return
	}

	for _, link := range page.Links {
		crawl(link)
	}
}

// Make a HTTP GET request to the specified URL and return a Page struct
func get(url string) (Page, error) {
	urlToGet, err := normalizeAndValidateLink(url)

	if err != nil {
		return Page{}, err
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
		return Page{}, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/75.0.3770.100 Safari/537.36")
	resp, err := client.Do(req)
	if err != nil {
		return Page{}, err
	}
	defer resp.Body.Close()
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return Page{}, err
	}

	fmt.Println("Successfully crawled: ", urlToGet, " Status code: ", resp.StatusCode)

	links := getLinks(doc)
	return Page{
		Title:      doc.Find("title").Text(),
		Url:        urlToGet,
		Links:      links,
		StatusCode: resp.StatusCode,
		Redirects:  redirects,
	}, nil
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

func normalizeAndValidateLink(link string) (string, error) {
	// if the link is already visited or ignored, return an error
	if urlsHandled[link] {
		return "", errors.New("link already visited or ignored: " + link)
	}
	// if the link has a prefix of / then it is a relative link and we need to add the baseUrl to it
	if strings.HasPrefix(link, "/") {
		link = baseUrl + link
	// if the link does not have a prefix of the baseUrl, then it is not a link to a page on the site
	} else if !strings.HasPrefix(link, baseUrl) {
		return "", errors.New("error: external link? link must start with '/'. Link is: " + link)
	} else {
		// if the link is none of the above if statements, then it is a link to a page on the site and we dont have to do anything
		// we dont need this block but leaving it here for clarity right now
	}

	// after normalizing, if the link is already visited or ignored, return an error
	if urlsHandled[link] {
			return "", errors.New("link already visited or ignored: " + link)
	}
	// we've handled this link/url, so add it to the urlsHandled map so we do not visit it again
	urlsHandled[link] = true

	return link, nil
}
