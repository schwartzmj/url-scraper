package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
)

var initialScheme string
var initialHost string
var initialPath string

var urlsHandled = make(map[string]bool)

var pages = []Page{}

func main() {
	baseUrl := "https://www.damselsbd.com/"
	u, err := url.Parse(baseUrl)
	if err != nil {
		log.Fatal(err)
	}
	initialScheme = u.Scheme
	initialHost = u.Host
	initialPath = u.Path

	// normalizedUrl, err := normalizeLink(baseUrl)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	crawl(u.String())

	file, _ := json.MarshalIndent(pages, "", " ")
	_ = ioutil.WriteFile("pages.json", file, 0644)

	fmt.Println("Number of pages visited: ", len(pages))
	fmt.Println("Done!")
}

func crawl(url string) {
	page, skipped, err := get(url)
	if err != nil {
		fmt.Println(err)
		return
	}
	// if page is empty, then we have already visited this page and we should return
	if skipped {
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
