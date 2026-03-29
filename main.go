package main

import "log"

func main() {
	const startUrl = "http://books.toscrape.com"

	// const startUrl = "http://crawler-test.com"
	// const startUrl = "http://toscrape.com"
	// const startUrl = "http://quotes.toscrape.com"

	err := Crawl(startUrl)

	if err != nil {
		log.Fatalf("error: %v", err)
	}

	log.Println("Success")
}
