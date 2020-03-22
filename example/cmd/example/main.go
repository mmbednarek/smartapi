package main

import (
	"log"

	"github.com/mmbednarek/smartapi"
	"github.com/mmbednarek/smartapi/example"
	"github.com/mmbednarek/smartapi/example/storage"
)

func main() {
	api := example.NewAPI(storage.NewStorage())
	if err := smartapi.StartAPI(api, ":4000"); err != nil {
		log.Fatal(err)
	}
}
