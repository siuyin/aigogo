package main

import (
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"runtime"
	"time"

	"github.com/google/generative-ai-go/genai"
	"github.com/philippgille/chromem-go"
	"github.com/siuyin/aigogo/cmd/aigogo/internal/public"
	"github.com/siuyin/aigogo/cmd/aigogo/internal/vecdb"
	"github.com/siuyin/aigotut/client"
	"github.com/siuyin/aigotut/emb"
	"github.com/siuyin/aigotut/gfmt"
	"github.com/siuyin/dflt"
)

var (
	cl *client.Info

	em         *genai.EmbeddingModel
	collection *chromem.Collection
	db         *chromem.DB
)

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
	retrievalFunc := func(w http.ResponseWriter, r *http.Request) {
		qry := r.FormValue("userPrompt")
		doc := retrieveDocsForAugmentation(qry)
		for _, d := range doc {
			io.WriteString(w, "<p>"+d+"</p>")
		}
	}

	life := func(w http.ResponseWriter, _ *http.Request) {
		meaningOfLife(w)
	}
	defer cl.Close()

	http.HandleFunc("/h1", h1)
	http.HandleFunc("/endpoint", h2)
	http.HandleFunc("/life", life)
	http.HandleFunc("/loc", neighborhood)
	http.HandleFunc("/retr", retrievalFunc)

	http.HandleFunc("/hello/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello World! It is %v\n", time.Now().Format("15:04:05.000 MST"))
	})

	// 	http.Handle("/", http.FileServer(http.Dir("./internal/public"))) // DEV
	http.Handle("/", http.FileServer(http.FS(public.Content))) // PROD

	log.Fatal(http.ListenAndServe(":"+dflt.EnvString("HTTP_PORT", "8080"), nil))
}

func init() {
	cl = client.New()
	em = initEmbeddingClient()
	collection = initDB()
}

func initEmbeddingClient() *genai.EmbeddingModel {
	client.ModelName = "text-embedding-004"
	cl = client.New()
	em := cl.Client.EmbeddingModel(client.ModelName)
	return em
}

func initDB() *chromem.Collection {
	docs := loadDocuments()

	db = chromem.NewDB()
	c, err := db.CreateCollection("aigogo", nil, nil)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	c.AddDocuments(ctx, docs, runtime.NumCPU())
	return c

}

func loadDocuments() []chromem.Document {
	// 	f, err := os.Open("./internal/vecdb/embeddings.gob") // DEV
	f, err := vecdb.Content.Open("embeddings.gob") // PROD
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	var rec emb.Rec
	dec := gob.NewDecoder(f)
	docs := []chromem.Document{}
	for {
		if err := dec.Decode(&rec); err != nil {
			break
		}
		docs = addDoc(docs, &rec)
	}
	return docs
}

func addDoc(docs []chromem.Document, rec *emb.Rec) []chromem.Document {
	d := chromem.Document{
		ID:      rec.ID,
		Content: rec.Title + " | " + rec.Content,
	}
	d.Embedding = append(d.Embedding, rec.Embedding...)
	docs = append(docs, d)
	return docs
}

func meaningOfLife(w http.ResponseWriter) {
	resp, err := cl.Model.GenerateContent(context.Background(), genai.Text("What is the meaning of life"))
	if err != nil {
		log.Fatal(err)
	}

	gfmt.FprintResponse(w, resp)
}

func retrieveDocsForAugmentation(qry string) []string {
	ctx := context.Background()
	res, err := em.EmbedContent(ctx, genai.Text(qry))
	if err != nil {
		log.Fatal(err)
	}

	numResults := 2
	qres, err := collection.QueryEmbedding(ctx, res.Embedding.Values, numResults, nil, nil)
	if err != nil {
		log.Fatal(err)
	}

	doc := []string{}
	for i := 0; i < len(qres); i++ {
		// fmt.Println(qres[i].ID, qres[i].Content)
		doc = append(doc, qres[i].Content)
	}
	return doc
}
