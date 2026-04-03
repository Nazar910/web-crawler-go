package main

import (
	"flag"
	"log"
)

func main() {
	target := flag.String("target", "http://books.toscrape.com", `
	--target=http://books.toscrape.com
	`)
	flag.Parse()

	err := Crawl(*target)

	if err != nil {
		log.Fatalf("error: %v", err)
	}

	log.Println("Success")
}
