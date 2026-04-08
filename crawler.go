package main

import (
	"context"
	"fmt"
	"iter"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"slices"
	"sync"
	"syscall"
	"time"

	"golang.org/x/net/html"
)

const RateLimit = 5
const UserAgent = "GoLearnerBot/1.0"

func Crawl(repo Repo, startLink string) error {
	isCrawlCompleted, err := repo.IsCrawlCompleted(startLink)

	if err != nil {
		return err
	}

	logger := NewLogger("crawler")

	if isCrawlCompleted {
		logger.Info("Crawl task is already completed")
		return nil
	}

	robotsTxt, err := newRobots(startLink, UserAgent)

	if err != nil {
		return fmt.Errorf("error while preparing robots.txt checker: %w", err)
	}

	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	c := crawler{
		urlch:   make(chan string, RateLimit),
		results: make(chan Result, RateLimit),
		done:    make(chan struct{}),
		client:  &http.Client{},

		limiter: make(chan struct{}, RateLimit),

		repo:      repo,
		robotsTxt: robotsTxt,

		signalCtx: signalCtx,

		logger: logger,
	}

	c.exitWg.Add(RateLimit + 1)

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

	select {
	case <-signalCtx.Done():
		c.logger.Debug("Signal processing")
		c.exitWg.Wait()
		c.logger.Debug("workers and scheduler exited, proceed with closing repo")
		repo.Close()
	case <-c.done:
		repo.EndCrawl(startLink)
	}

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

	signalCtx context.Context
	exitWg    sync.WaitGroup

	logger *slog.Logger
}

func (c *crawler) start() {
	defer c.exitWg.Done()
	for link := range c.urlch {
		c.results <- Result{link, c.processLink(link)}
	}
	c.logger.Debug("worker end loop")
}

func (c *crawler) processLink(link string) []string {
	urlLink, err := url.Parse(link)

	if err != nil {
		return nil
	}

	// for now this just prints the link to stdout
	// but it may be a write to a file or some db
	c.logger.Info(fmt.Sprintf("worker: %s", link))
	req, err := http.NewRequestWithContext(c.signalCtx, "GET", link, nil)
	req.Header.Set("User-Agent", UserAgent)

	if err != nil {
		c.logger.Error(fmt.Sprintf("error: %v", err))
		return nil
	}

	res, err := c.client.Do(req)

	if err != nil {
		c.logger.Warn(fmt.Sprintf("error: %v", err))
		return nil
	}

	defer res.Body.Close()
	doc, err := html.Parse(res.Body)

	if err != nil {
		c.logger.Warn(fmt.Sprintf("html parser error: %v", err))
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
					c.logger.Warn(fmt.Sprintf("link format error: %v", err))
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

	pendingch := make(chan string)

	defer close(pendingch)

	go func() {
		for l := range pendingch {
			<-c.limiter
			c.urlch <- l
		}
	}()

	pendingQueue := newUniqueQueue()

	for l := range c.repo.ScheduledSeq() {
		pendingQueue.enqueue(l)
	}

	// current implementation chooses at least once
	// scheduling of the link instead of exactly once
	// for the sake of simplicity
	for inFlight > 0 || pendingQueue.len() > 0 {
		c.logger.Debug(fmt.Sprintf("loop: inFlight=%d, len(pendingQueue)=%d\n", inFlight, pendingQueue.len()))
		var activePendingch chan string
		var nextLink string

		// new Go knowledge unlocked here:
		// if your channel is nil then sending to it
		// is a forever blocking operation
		// which is super convenient in this case
		// to not try to push anything into pengingch
		// as this should means that we're waiting for
		// workers to do the job and gather their results
		if pendingQueue.len() > 0 {
			activePendingch = pendingch
			nextLink = pendingQueue.dequeue()
		}

		select {
		case res := <-c.results:
			inFlight--
			for _, l := range res.childs {
				visited, err := c.repo.IsProcessed(l)

				if err != nil {
					panic(err)
				}
				if visited || !c.robotsTxt.IsPathAllowed(l) {
					continue
				}
				if err := c.repo.Scheduled(l); err != nil {
					panic(err)
				}

				pendingQueue.enqueue(l)
			}

			if err := c.repo.Processed(res.link); err != nil {
				panic(err)
			}
		case activePendingch <- nextLink:
			inFlight++

		case <-c.signalCtx.Done():
			for inFlight > 0 {
				<-c.results
				inFlight--
			}
			c.exitWg.Done()
			return
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

type uniqueQueue struct {
	set   map[string]struct{}
	queue []string
}

func newUniqueQueue() uniqueQueue {
	return uniqueQueue{
		set:   make(map[string]struct{}),
		queue: make([]string, 0),
	}
}

func (q *uniqueQueue) enqueue(e string) {
	if _, ok := q.set[e]; ok {
		return
	}

	q.set[e] = struct{}{}
	q.queue = append(q.queue, e)
}

func (q *uniqueQueue) dequeue() string {
	e := q.queue[0]
	q.queue = q.queue[1:]
	delete(q.set, e)
	return e
}

func (q *uniqueQueue) len() int {
	return len(q.queue)
}
