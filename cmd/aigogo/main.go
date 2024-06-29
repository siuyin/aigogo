package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/google/generative-ai-go/genai"
	_ "github.com/siuyin/aigogo/cmd/aigogo/internal/public"
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

	type mapResponse struct {
		Results []struct {
			FormattedAddress string `json:"formatted_address"`
		} `json:"results"`
	}
	neighborhood := func(w http.ResponseWriter, r *http.Request) {
		key := dflt.EnvString("MAPS_API_KEY", "use your real api key")
		latlng := r.FormValue("latlng")
		res, err := http.Get(fmt.Sprintf("https://maps.googleapis.com/maps/api/geocode/json?latlng=%s&result_type=neighborhood&key=%s", latlng, key))
		if err != nil {
			log.Fatal(err)
		}

		var mapRes mapResponse
		dec := json.NewDecoder(res.Body)
		if err = dec.Decode(&mapRes); err != nil {
			log.Fatal(err)
		}

		res.Body.Close()
		if res.StatusCode > 299 {
			log.Fatalf("Response failed with status code: %d\n", res.StatusCode)
		}
		if err != nil {
			log.Fatal(err)
		}
		if len(mapRes.Results) == 0 {
			log.Fatal("no results returned")
		}
		fmt.Fprintf(w, "%s", mapRes.Results[0].FormattedAddress)

	}

	life := func(w http.ResponseWriter, _ *http.Request) {
		meaningOfLife(w)
	}
	defer cl.Close()

	http.HandleFunc("/h1", h1)
	http.HandleFunc("/endpoint", h2)
	http.HandleFunc("/life", life)
	http.HandleFunc("/loc", neighborhood)

	http.HandleFunc("/hello/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello World! It is %v\n", time.Now().Format("15:04:05.000 MST"))
	})

	http.Handle("/", http.FileServer(http.Dir("./internal/public"))) // DEV
	//  	http.Handle("/", http.FileServer(http.FS(public.Content))) // PROD

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
