package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/enobufs/go-nats/nats"
)

func check(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(1)
	}
}

func main() {
	server := flag.String("s", "stun.ekiga.net:3478", "STUN server address. (defaults to \"stun.ekiga.net:3478\"")
	verbose := flag.Bool("v", false, "Verbose")

	flag.Parse()

	res, err := nats.Discover(*server, *verbose)
	check(err)

	bytes, err := json.MarshalIndent(res, "", "  ")
	check(err)

	fmt.Println(string(bytes))
}
