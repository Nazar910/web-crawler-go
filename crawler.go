package main

import (
	"errors"
	"fmt"
	"iter"
	"net/http"
	"net/url"
	"slices"

	"golang.org/x/net/html"
)

const startUrl = "http://books.toscrape.com"

// const startUrl = "http://crawler-test.com"
// const startUrl = "http://toscrape.com"
// const startUrl = "http://quotes.toscrape.com"

func Crawl() error {
	c := crawler{
		urlch:   make(chan string),
		results: make(chan []string),
		done:    make(chan struct{}),
		client:  &http.Client{},
	}
	for range 10 {
		go c.start()
	}
	go c.schedule(startUrl)

	<-c.done

	return nil
}

type crawler struct {
	urlch   chan string
	results chan []string

	done chan struct{}

	client *http.Client
}

func (c *crawler) start() {
	for link := range c.urlch {
		c.results <- c.eachl(link)
	}
}

func (c *crawler) eachl(link string) []string {
	urlLink, err := url.Parse(link)

	if err != nil {
		return []string{}
	}

	fmt.Println(link)
	req, err := http.NewRequest("GET", link, nil)

	if err != nil {
		fmt.Printf("error: %v\n", err)
		return []string{}
	}

	res, err := c.client.Do(req)

	if err != nil {
		fmt.Printf("error: %v\n", err)
		return []string{}
	}

	defer res.Body.Close()
	doc, err := html.Parse(res.Body)

	if err != nil {
		fmt.Printf("html parse error: %v\n", err)
		return []string{}
	}

	return slices.AppendSeq(
		make([]string, 0),
		c.linksIter(doc, urlLink.Host, urlLink.Scheme),
	)
}

func (c *crawler) linksIter(doc *html.Node, host, scheme string) iter.Seq[string] {
	return func(yield func(string) bool) {
		for n := range doc.Descendants() {
			if n.Type == html.ElementNode && n.Data == "a" {
				href, err := getHref(n)

				if err != nil {
					fmt.Printf("%v\n", err)
					continue
				}

				link, err := getCompleteLink(href, host, scheme)

				if err != nil {
					fmt.Printf("%v\n", err)
					continue
				}

				if !yield(link) {
					break
				}
			}
		}
	}
}

func (c *crawler) schedule(seed string) {
	defer close(c.done)

	inFlight := 1

	go func() { c.urlch <- seed }()

	for inFlight > 0 {
		links := <-c.results
		inFlight--

		for _, link := range links {
			inFlight++
			go func(l string) { c.urlch <- l }(link)
		}
	}

	close(c.urlch)
}

func getHref(n *html.Node) (string, error) {
	for _, attr := range n.Attr {
		if attr.Key == "href" {
			return attr.Val, nil
		}
	}
	return "", errors.New("no href in a elem")
}

func getCompleteLink(rawUrl, pHost, pScheme string) (string, error) {
	parsedLink, err := url.Parse(rawUrl)

	if err != nil {
		return "", err
	}

	if parsedLink.Host == "" {
		parsedLink.Host = pHost
		parsedLink.Scheme = pScheme
	}

	return parsedLink.String(), nil
}
