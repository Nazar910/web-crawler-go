//go:build ignore

package main

import (
	"flag"
	"log"
	"os"
	"os/exec"
)

func main() {
	args := []string{"go"}
	cmd := flag.String("cmd", "build", "available commands: build, test")
	flag.Parse()
	switch *cmd {
	default:
		log.Fatalf("unknown command: %s", *cmd)
	case "build":
		args = append(args, "build", "-trimpath", "-ldflags=-s -w", "-o=bin/crawler", ".")
	case "test":
		args = append(args, "test", ".")
	case "run":
		args = append(args, "run", ".")
	}

	// exec.Command requires at least 1 arg
	// that's why I'm passing first argument separately
	command := exec.Command(args[0], args[1:]...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		os.Exit(1)
	}

}
