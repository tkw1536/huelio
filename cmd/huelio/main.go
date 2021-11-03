package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/amimof/huego"
	"github.com/tkw1536/huelio"
)

func main() {
	args := flag.Args()
	if len(args) <= 0 {
		log.Fatal("Required at least one argument")
	}

	// create a huelio client
	client := huelio.NewEngine(&huego.Bridge{Host: apiHost, User: apiUsername})

	// run the query!
	results, err := client.Query(strings.Join(args, " "))
	if err != nil {
		log.Fatal(err)
	}

	// print the results!
	for _, arg := range results {
		fmt.Println(arg)
	}
}

var apiHost string
var apiUsername string

func init() {
	defer flag.Parse()

	flag.StringVar(&apiHost, "host", os.Getenv("HUE_HOST"), "hue hostname")
	flag.StringVar(&apiUsername, "user", os.Getenv("HUE_USER"), "hue username")
}
