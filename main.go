package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	neturl "net/url"
	"os"
	"sync"

	"golang.org/x/net/html"
)

type HTTPFetcher interface {
	// Fetch fetches a url and returns all urls in the page or an error
	Fetch(url string) (urls []string, err error)
	Verbose() bool
}

// Store performs state management for the scraper, keeping track of visited links
type Store struct {
	mutex        *sync.RWMutex
	VisitedLinks map[string]struct{}
}

// Crawler encapsulates the crawling logic, while acting as a dependency that can be swapped out
type Crawler struct {
	store   *Store
	host    string
	scheme  string
	verbose bool
}

func (s *Store) write(key string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.VisitedLinks[key] = struct{}{}
}

func (s *Store) read(key string) bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	_, ok := s.VisitedLinks[key]
	return ok
}

func crawl(url neturl.URL, depth int, store *Store, c HTTPFetcher, wg *sync.WaitGroup) {
	defer wg.Done()
	urlString := url.String()
	if depth < 0 {
		return
	}
	if store.read(urlString) {
		return
	}
	urls, err := c.Fetch(urlString)
	if err != nil {
		log.Printf("ERROR: error crawling %s, err: %v", urlString, err)
		return
	}
	fmt.Println("-", urlString)
	for _, u := range urls {
		wg.Add(1)
		parsedU, err := url.Parse(u)
		if err != nil {
			continue
		}
		if c.Verbose() {
			fmt.Println("   -", parsedU)
		}
		go crawl(*parsedU, depth-1, store, c, wg)
	}
}

func (c *Crawler) Fetch(url string) (urls []string, err error) {
	c.store.write(url)
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	urls, err = c.processPage(res.Body)
	return
}

func (c *Crawler) processPage(page io.Reader) (urls []string, err error) {
	doc, err := html.Parse(page)
	if err != nil {
		return nil, err
	}
	urls = c.findLinks(doc, urls)
	return
}

func (c *Crawler) Verbose() bool {
	return c.verbose
}

func (c *Crawler) processLink(link string) (bool, string) {
	r, err := neturl.Parse(link)
	if err != nil {
		return false, ""
	}
	if r.Host == c.host || r.Scheme == "" {
		return true, link
	}
	return false, ""
}

func (c *Crawler) findLinks(node *html.Node, foundLinks []string) []string {
	if node.Type == html.ElementNode && node.Data == "a" {
		for _, a := range node.Attr {
			if a.Key == "href" {
				valid, link := c.processLink(a.Val)
				if valid {
					foundLinks = append(foundLinks, link)
				}
				break
			}
		}
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		foundLinks = c.findLinks(child, foundLinks)
	}
	return foundLinks
}

func main() {
	maxDepth := flag.Int("depth", 1, "max depth to follow through links")
	startUrl := flag.String("url", "", "start url for crawler")
	verbose := flag.Bool("verbose", false, "show visited links as well as links in those pages")
	flag.Parse()

	l, err := neturl.Parse(*startUrl)
	if err != nil || *startUrl == "" || l.Host == "" || l.Scheme == "" {
		fmt.Println("Invalid start url")
		os.Exit(0)
	}

	wg := &sync.WaitGroup{}
	mutex := &sync.RWMutex{}
	store := &Store{mutex: mutex, VisitedLinks: make(map[string]struct{})}
	crawler := &Crawler{host: l.Host, scheme: l.Scheme, store: store, verbose: *verbose}
	wg.Add(1)
	crawl(*l, *maxDepth, store, crawler, wg)
	wg.Wait()
}
