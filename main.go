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
	server := flag.String("s", "stun.sipgate.net:3478", "STUN server address.")
	verbose := flag.Bool("v", false, "Verbose")

	flag.Parse()

	n, err := nats.NewNATS(&nats.Config{
		Server:  *server,
		Verbose: *verbose,
	})
	check(err)

	res, err := n.Discover()
	check(err)

	bytes, err := json.MarshalIndent(res, "", "  ")
	check(err)

	fmt.Println(string(bytes))
}
