package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"
)

//type InternalPage struct {
//	Title      string
//	Url        string
//	Host       string
//	Path       string
//	Scheme     string
//	Hrefs      []string
//	StatusCode int
//	//Redirects  []string // eventually make this an array of Redirect struct itself (from, to, status code)?
//}
//
//type ExternalPage struct {
//	Url        string
//	Host       string
//	Path       string
//	Scheme     string
//	StatusCode int
//}

var initialScheme string
var initialHost string
var initialPath string

// make a urlsHandled syncmap
var urlsHandledMutex = UrlsHandledMutex{urls: make(map[string]int)}

type UrlsHandledMutex struct {
	mu   sync.Mutex
	urls map[string]int
}

type VisitedPage struct {
	GivenHref  string
	Url        string
	StatusCode int
}

type PagesVisitedMutex struct {
	mu           sync.Mutex
	VisitedPages []VisitedPage
}

type AnchorTagsWithoutHrefMutex struct {
	mu   sync.Mutex
	Tags []AnchorTag
}

var internalPagesVisitedMutex = PagesVisitedMutex{}
var externalPagesVisitedMutex = PagesVisitedMutex{}
var anchorTagsWithoutHrefMutex = AnchorTagsWithoutHrefMutex{}

func main() {
	startTime := time.Now()
	defer func() {
		fmt.Println("Time taken total:", time.Since(startTime))
	}()
	// TODO: maybe this should return `initialUrl, err` instead of storing these globally?
	baseUrl, err := handleArgs()
	if err != nil {
		log.Fatal(err)
	}

	initiateCrawl(baseUrl)

	wg.Wait()
	saveAndPrintResults()
}

func handleArgs() (string, error) {
	baseUrlPtr := flag.String("url", "", "Base URL to crawl")
	flag.Parse()

	baseUrl, err := url.Parse(*baseUrlPtr)
	if err != nil {
		log.Fatal(err)
	}
	if !baseUrl.IsAbs() {
		log.Fatal("Base URL must be absolute")
	}

	if len(baseUrl.String()) == 0 {
		log.Fatal("Please provide a base URL to crawl")
	}

	initialScheme = baseUrl.Scheme
	initialHost = baseUrl.Host
	initialPath = baseUrl.Path
	return baseUrl.String(), nil
}

func saveAndPrintResults() {
	toSave := make(map[string]interface{})
	toSave["anchorTagsWithoutHref"] = anchorTagsWithoutHrefMutex.Tags
	toSave["internalPages"] = internalPagesVisitedMutex.VisitedPages
	toSave["externalPages"] = externalPagesVisitedMutex.VisitedPages
	toSave["urlsHandled"] = urlsHandledMutex.urls

	file, _ := json.MarshalIndent(toSave, "", " ")

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
	fmt.Println("Completed! Number of pages visited:")
	fmt.Println("Internal: ", len(internalPagesVisitedMutex.VisitedPages))
	fmt.Println("External: ", len(externalPagesVisitedMutex.VisitedPages))
	fmt.Println("<a> without href: ", len(anchorTagsWithoutHrefMutex.Tags))
	fmt.Println("Saved to:", pathToSave)
	fmt.Println("\033[32m" + "---------------" + "\033[0m")
}
