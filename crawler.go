package main

import (
	"fmt"
	"iter"
	"net/http"
	"net/url"
	"slices"
	"time"

	"golang.org/x/net/html"
)

const RateLimit = 5
const UserAgent = "GoLearnerBot/1.0"

func Crawl(repo Repo, startLink string) error {
	c := crawler{
		urlch:   make(chan string),
		results: make(chan Result),
		done:    make(chan struct{}),
		client:  &http.Client{},

		limiter: make(chan struct{}, RateLimit),

		repo: repo,
	}

	isCrawlCompleted, err := repo.IsCrawlCompleted(startLink)

	if err != nil {
		return err
	}

	if isCrawlCompleted {
		fmt.Println("Crawl task is already completed")
		return nil
	}

	robotsTxt, err := newRobots(startLink, UserAgent)

	if err != nil {
		return fmt.Errorf("error while preparing robots.txt checker: %w", err)
	}

	c.robotsTxt = robotsTxt

	for range RateLimit {
		go c.start()
	}
	// important for worker to start with all tokens capacity
	c.fillTokens()
	created, err := repo.StartCrawl(startLink)

	if err != nil {
		return err
	}

	if created {
		repo.Scheduled(startLink)
	}

	go c.scheduler()
	go c.limiterLoop()

	<-c.done

	return nil
}

type Result struct {
	link   string
	childs []string
}

type crawler struct {
	urlch   chan string
	results chan Result

	done chan struct{}

	limiter chan struct{}

	client *http.Client

	robotsTxt robotsTxt

	repo Repo
}

func (c *crawler) start() {
	for link := range c.urlch {
		<-c.limiter
		c.results <- Result{link, c.processLink(link)}
	}
}

func (c *crawler) processLink(link string) []string {
	urlLink, err := url.Parse(link)

	if err != nil {
		return nil
	}

	// for now this just prints the link to stdout
	// but it may be a write to a file or some db
	fmt.Println(link)
	req, err := http.NewRequest("GET", link, nil)
	req.Header.Set("User-Agent", UserAgent)

	if err != nil {
		fmt.Printf("error: %v\n", err)
		return nil
	}

	res, err := c.client.Do(req)

	if err != nil {
		fmt.Printf("error: %v\n", err)
		return nil
	}

	defer res.Body.Close()
	doc, err := html.Parse(res.Body)

	if err != nil {
		fmt.Printf("html parse error: %v\n", err)
		return nil
	}

	return slices.Collect(
		c.linksIter(doc, urlLink.Host, urlLink.Scheme),
	)
}

// it doesn't really make sense using iterator here
// since the whole list is now used instead of sending each link to a channel
// but since it is a pet project I'll leave it as is
// just because I only recently learned Go iterators and
// I kinda like how it looks ^_^
func (c *crawler) linksIter(doc *html.Node, host, scheme string) iter.Seq[string] {
	return func(yield func(string) bool) {
		for n := range doc.Descendants() {
			if n.Type == html.ElementNode && n.Data == "a" {
				href := getHref(n)

				if href == "" {
					// a without href -> skip it
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

func (c *crawler) scheduler() {
	defer close(c.urlch)
	defer close(c.done)

	var inFlight int
	visited := make(map[string]struct{})

	for l := range c.repo.ScheduledSeq() {
		inFlight++
		go func() { c.urlch <- l }()
	}

	for inFlight > 0 {
		result := <-c.results
		c.repo.Processed(result.link)
		inFlight--

		for _, link := range result.childs {
			// link visited should be stored to persistent store
			if _, ok := visited[link]; ok {
				continue
			}
			if !c.robotsTxt.IsPathAllowed(link) {
				continue
			}
			visited[link] = struct{}{}
			c.repo.Scheduled(link)
			inFlight++
			// in production it would be better to use
			// some buffered channel here
			go func(l string) { c.urlch <- l }(link)
		}
	}

}

func (c *crawler) limiterLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.fillTokens()
		case <-c.done:
			return
		}
	}
}

func (c *crawler) fillTokens() {
	for range RateLimit {
		select {
		case c.limiter <- struct{}{}:
		default:
		}
	}
}

func getHref(n *html.Node) string {
	for _, attr := range n.Attr {
		if attr.Key == "href" {
			return attr.Val
		}
	}
	return ""
}

func getCompleteLink(rawUrl, pHost, pScheme string) (string, error) {
	parsedLink, err := url.Parse(rawUrl)

	if err != nil {
		return "", err
	}

	if parsedLink.Host == "" || parsedLink.Host != pHost {
		parsedLink.Host = pHost
		parsedLink.Scheme = pScheme
	}

	return parsedLink.String(), nil
}
