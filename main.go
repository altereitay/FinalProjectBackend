package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/altereitay/FinalProjectBackend/db"
	"github.com/altereitay/FinalProjectBackend/helpers"
)

func handleFile(w http.ResponseWriter, r *http.Request) {
	log.Println("Handling a new article")
	helpers.HandleFile(w, r)
}

func handleFrontend() http.Handler {
	fs := http.FileServer(http.Dir("../FinalProjectUI/dist/"))
	return fs
}

func initMQTT() {
	err := helpers.InitMQTT()
	if err != nil {
		log.Fatal(err)
	}

	if err := helpers.Subscribe("articles/simplified", helpers.HandleSimplifiedArticles); err != nil {
		log.Fatal(err)
	}
}

func main() {
	mux := http.NewServeMux()
	port := 8082

	mux.HandleFunc("POST /article/new", handleFile)

	mux.Handle("/", handleFrontend())

	log.Println("Server running on 0.0.0.0:", port)

	err := db.InitMongo()
	if err != nil {
		log.Fatal(err)
	}

	log.Fatal(http.ListenAndServe(fmt.Sprintf("0.0.0.0:%v", port), mux))
}
