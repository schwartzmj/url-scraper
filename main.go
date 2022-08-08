package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"path"
	"path/filepath"
	"runtime"
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

var wg = sync.WaitGroup{}

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
	wg.Add(1)
	crawl(u.String())

	wg.Wait()

	file, _ := json.MarshalIndent(pagesMutex.pages, "", " ")

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("No caller information")
	}
	pathToSave := filepath.Join(path.Dir(filename), "pages.json")
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
	// Note we wg.Add(1) before the very first call of this function (done in main())
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

	// if getPageResult.Page.StatusCode != http.StatusOK ||  {
	// 	fmt.Println("Error. Status code:", getPageResult.Page.StatusCode)
	// 	return
	// }

	// For each getPageResult.Page.Links, call crawl on each link concurrently
	for _, link := range getPageResult.Page.Links {
		wg.Add(1)
		go crawl(link)
	}
}
