package main

import (
	"log"
	"net/http"

	llmkit "github.com/iaingblack/llm-kit/llmkit"
)

func main() {
	store := llmkit.NewMemoryStore(llmkit.DefaultConfig())
	settings := llmkit.NewHandler(store)

	mux := http.NewServeMux()
	settings.Register(mux, "/api/llm")

	log.Println("listening on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
