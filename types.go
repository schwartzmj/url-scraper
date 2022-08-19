package main

import "sync"

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

type AnchorTag struct {
	HrefValue                   string
	HrefExists                  bool
	InnerTextForNonExistentHref string
	FoundOn                     string
}
