package main

import "log"

func main() {
	err := Crawl()

	if err != nil {
		log.Fatal("error: %v", err)
	}

	log.Println("Success")
}
