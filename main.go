package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/altereitay/FinalProjectBackend/db"
	"github.com/altereitay/FinalProjectBackend/helpers"
)

func handleFile(w http.ResponseWriter, r *http.Request) {
	log.Println("Handling a new article")
	helpers.HandleFile(w, r)
}

func handleFrontend() http.Handler {
	distDir := "../FinalProjectUI/dist"
	fs := http.FileServer(http.Dir(distDir))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to serve the file
		f, err := os.Open(filepath.Join(distDir, r.URL.Path))
		if err == nil {
			f.Close()
			fs.ServeHTTP(w, r)
			return
		}

		// If file doesn't exist, serve index.html
		http.ServeFile(w, r, filepath.Join(distDir, "index.html"))
	})
}

func handleArticles(w http.ResponseWriter, r *http.Request) {
	log.Println("Retriving all articles")
	helpers.HandleArticles(w, r)
}

func initMQTT() {
	err := helpers.InitMQTT()
	if err != nil {
		log.Fatal(err)
	}

	if err := helpers.Subscribe(helpers.SIMPLIFY_TOPIC, helpers.HandleSimplifiedArticles); err != nil {
		log.Fatal(err)
	}

	if err := helpers.Subscribe(helpers.TERMS_TOPIC, helpers.HandleTerms); err != nil {
		log.Fatal(err)
	}
}

func enableCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		// Handle preflight
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		h.ServeHTTP(w, r)
	})
}

func main() {
	mux := http.NewServeMux()
	port := 8082

	initMQTT()

	mux.HandleFunc("POST /article/new", handleFile)

	mux.Handle("/", handleFrontend())

	mux.HandleFunc("GET /articles", handleArticles)

	log.Println("Server running on 0.0.0.0:", port)

	err := db.InitMongo()
	if err != nil {
		log.Fatal(err)
	}

	wrapped := enableCORS(mux)

	log.Fatal(http.ListenAndServe(fmt.Sprintf("0.0.0.0:%v", port), wrapped))
}
