package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var initialScheme string
var initialHost string
var initialPath string

// make a urlsHandled syncmap
var urlsHandledMutex = UrlsHandledMutex{urls: make(map[string]bool)}

type UrlsHandledMutex struct {
	mu   sync.Mutex
	urls map[string]bool
}

type PagesMutex struct {
	mu    sync.Mutex
	pages []Page
}

var pagesMutex = PagesMutex{}

var wg sync.WaitGroup

func main() {
	start := time.Now()
	defer func() {
		fmt.Println("Time taken:", time.Since(start))
	}()

	baseUrl := "https://www.wemaketechsimple.com/"
	u, err := url.Parse(baseUrl)
	if err != nil {
		log.Fatal(err)
	}
	initialScheme = u.Scheme
	initialHost = u.Host
	initialPath = u.Path

	// Initiate recursive crawl
	crawl(u.String())

	wg.Wait()

	ex, err := os.Getwd()
    if err != nil {
        panic(err)
    }
	file, _ := json.MarshalIndent(pagesMutex.pages, "", " ")
	pathToSave := filepath.Join(filepath.Dir(ex), "pages.json")
	fmt.Println("Saving to:", pathToSave)
	err = ioutil.WriteFile(pathToSave, file, 0644)
	if err != nil {
		log.Fatal(err)
	}

	// fmt.Println("Pages Mutex: ", pagesMutex.pages)
	fmt.Println("Number of pages visited: ", len(pagesMutex.pages))
	fmt.Println("Done.")
}

func crawl(url string) {
	wg.Add(1)
	defer wg.Done()

	getPageResult := get(url)

	if getPageResult.Err != nil {
		fmt.Println(getPageResult.Err)
		return
	}
	// if page is empty, then we have already visited this page and we should return
	if getPageResult.Skipped {
		return
	}

	// add the page to the pages slice
	pagesMutex.mu.Lock()
	pagesMutex.pages = append(pagesMutex.pages, getPageResult.Page)
	pagesMutex.mu.Unlock()

	if getPageResult.Page.StatusCode != http.StatusOK {
		fmt.Println("Error. Status code:", getPageResult.Page.StatusCode)
		return
	}
	// For each getPageResult.Page.Links, call crawl on each link concurrently
	for _, link := range getPageResult.Page.Links {
		ch := make(chan bool)
		go func(link string) {
			crawl(link)
			ch <- true
		}(link)
	}
}
