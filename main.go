package main

import (
	"flag"
	"log"
)

func main() {
	target := flag.String("target", "http://books.toscrape.com", "domain of the target site to crawl")
	flag.Parse()

	repo, err := NewBboltRepo()

	if err != nil {
		log.Fatalf("error on repo init: %v", err)
	}

	err = Crawl(repo, *target)

	if err != nil {
		log.Fatalf("error: %v", err)
	}

	log.Println("Success")
}
