package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
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

	ex, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	pathToSave := filepath.Join(filepath.Dir(ex), "pages.json")
	err = ioutil.WriteFile(pathToSave, file, 0644)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("\033[32m" + "---------------" + "\033[0m")
	fmt.Println("Completed! Number of pages visited: ", len(pagesMutex.pages))
	fmt.Println("Saved to:", pathToSave)
	fmt.Println("\033[32m" + "---------------" + "\033[0m")
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
