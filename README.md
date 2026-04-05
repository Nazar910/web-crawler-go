# Web Crawler in Go — A Learning Experience

A hands-on exploration of building a concurrent web crawler in Go, working through real-world production problems one by one.

The crawler is tested against site built for web scraping testing:
* http://books.toscrape.com
* http://crawler-test.com
* http://toscrape.com
* http://quotes.toscrape.com

## 10 Most Common Problems When Building a Production-Grade Web Crawler

- [x] **1. Duplicate URL Visitation** — How do you avoid crawling the same URL twice? At scale, the set of seen URLs grows enormous and naive in-memory maps won't cut it. URL normalization (trailing slashes, query param ordering, fragments, encoding) makes this harder than it looks.

- [x] **2. Politeness & Rate Limiting** — Hammering a single host with hundreds of concurrent requests will get you blocked — or worse, take down a small site. How do you enforce per-domain rate limits and respect `robots.txt` directives?

- [x] **3. Concurrency Control** — Go makes concurrency easy with goroutines, but unbounded concurrency will exhaust file descriptors, memory, and remote servers. How do you control the number of in-flight requests globally and per-host without deadlocking your pipeline?

- [ ] **4. Graceful Shutdown & State Persistence** — A long-running crawl may take hours or days. What happens when you need to stop it? How do you drain in-flight work, save the frontier (the queue of URLs yet to crawl), and resume later without losing progress or re-crawling everything?

- [ ] **5. Handling Malformed & Adversarial HTML** — The real web is messy. Broken tags, relative URLs, JavaScript-rendered content, meta redirects, `<base>` tags, and intentional spider traps (infinite URL spaces). How do you extract links reliably without getting stuck?

- [ ] **6. Redirect Loops & Depth Explosion** — HTTP redirects can form cycles. URL paths can grow infinitely deep (e.g., calendar pages like `/2026/03/29/next/next/next/...`). How do you detect and bail out of these traps?

- [ ] **7. Timeouts, Retries & Transient Failures** — Connections hang, servers return 503s, DNS resolution fails intermittently. How do you set appropriate timeouts at each stage (DNS, connect, TLS handshake, response headers, body read) and decide what to retry vs. what to abandon?

- [ ] **8. Memory Management Under Load** — Large HTML pages, response bodies you forgot to close, unbounded queues, and goroutine leaks will all slowly eat your memory. How do you cap resource usage and ensure every `http.Response.Body` is fully drained and closed?

- [ ] **9. Content-Type & Encoding Detection** — Not everything behind a URL is HTML. You'll hit PDFs, images, binary files, and pages served with wrong or missing `Content-Type` headers. Charset can be declared in HTTP headers, HTML `<meta>` tags, or BOM bytes — and they often disagree. How do you handle this correctly?

- [ ] **10. Crawl Prioritization & Frontier Management** — Not all URLs are equally valuable. How do you decide what to crawl next? A naive FIFO queue treats a site's terms-of-service page the same as its homepage. Breadth-first vs. depth-first vs. priority-based ordering has real consequences on what you discover and how fast.
