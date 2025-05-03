package main

import (
	"github.com/altereitay/FinalProjectBackend/db"
	"github.com/altereitay/FinalProjectBackend/helpers"
	"log"
	"net/http"
)

func handleFile(w http.ResponseWriter, r *http.Request) {
	helpers.HandleFile(w, r)
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /article/new", handleFile)

	log.Println("Server running on localhost:8081")

	err := db.InitMongo()
	if err != nil {
		log.Fatal(err)
	}

	log.Fatal(http.ListenAndServe("0.0.0.0:8081", mux))
}
