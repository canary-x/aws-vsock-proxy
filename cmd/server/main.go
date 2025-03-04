package main

import (
	"log"

	"github.com/canary-x/aws-vsock-proxy/internal"
)

func main() {
	if err := internal.Run(); err != nil {
		log.Fatalf("Fatal error: %+v\n", err)
	}
}
