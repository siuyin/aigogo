package main

import (
	"context"
	"io"
	"log"
	"net/http"

	"github.com/google/generative-ai-go/genai"
	"github.com/siuyin/aigotut/client"
	"github.com/siuyin/aigotut/gfmt"
)

var cl *client.Info

func main() {
	h1 := func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, "Hello from a HandleFunc #1.\n")
	}
	h2 := func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, "Hello from a HandleFunc #2!\n")
	}

	life := func(w http.ResponseWriter, _ *http.Request) {
		meaningOfLife(w)
	}
	defer cl.Close()

	http.HandleFunc("/", h1)
	http.HandleFunc("/endpoint", h2)
	http.HandleFunc("/life", life)

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func init() {
	cl = client.New()
}

func meaningOfLife(w http.ResponseWriter) {
	resp, err := cl.Model.GenerateContent(context.Background(), genai.Text("What is the meaning of life"))
	if err != nil {
		log.Fatal(err)
	}

	gfmt.FprintResponse(w, resp)
}
