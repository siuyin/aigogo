package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/google/generative-ai-go/genai"
	"github.com/siuyin/aigogo/cmd/aigogo/internal/public"
	"github.com/siuyin/aigotut/client"
	"github.com/siuyin/aigotut/gfmt"
	"github.com/siuyin/dflt"
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

	http.HandleFunc("/h1", h1)
	http.HandleFunc("/endpoint", h2)
	http.HandleFunc("/life", life)

	http.HandleFunc("/hello/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello World! It is %v\n", time.Now().Format("15:04:05.000 MST"))
	})

	//http.Handle("/", http.FileServer(http.Dir("./internal/public"))) // uncomment for development
	http.Handle("/", http.FileServer(http.FS(public.Content))) // uncomment for deployment

	log.Fatal(http.ListenAndServe(":"+dflt.EnvString("HTTP_PORT", "8080"), nil))
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
