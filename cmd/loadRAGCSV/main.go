package main

import (
	"context"
	"encoding/csv"
	"encoding/gob"
	"fmt"
	"log"
	"os"

	"github.com/google/generative-ai-go/genai"
	"github.com/siuyin/aigogo/rag"
	"github.com/siuyin/aigotut/client"
	"github.com/siuyin/dflt"
)

const batchSize = 100

func main() {
	dat := loadRAGCSV()
	res := batchEmbedNew(batchSize, dat[1:])
	outputEmbeddingsGOBNew(dat[1:], res)
	// res := batchEmbed(dat)
	// outputEmbeddingsGOB(dat, res)
}

func loadRAGCSV() [][]string {
	fn := dflt.EnvString("RAGCSV", "/home/siuyin/Downloads/aigogo data - General.csv")
	f, err := os.Open(fn)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	c := csv.NewReader(f)
	dat, err := c.ReadAll()
	if err != nil {
		log.Fatal(err)
	}
	return dat
}

func batchEmbedNew(batchSize int, dat [][]string) []*genai.BatchEmbedContentsResponse {
	res := []*genai.BatchEmbedContentsResponse{}
	for i := 0; i < len(dat); i += batchSize {
		end := i + batchSize
		if end > len(dat) {
			end = len(dat)
		}
		bat := dat[i:end]
		r := batchEmbed(bat)
		res = append(res, r)
	}
	return res
}
func batchEmbed(dat [][]string) *genai.BatchEmbedContentsResponse {
	client.ModelName = "text-embedding-004"
	cl := client.New()
	defer cl.Close()

	ctx := context.Background()
	em := cl.Client.EmbeddingModel(client.ModelName)
	b := em.NewBatch()
	for _, v := range dat[1:] {
		b.AddContentWithTitle(v[1], genai.Text(v[2]))
	}

	res, err := em.BatchEmbedContents(ctx, b)
	if err != nil {
		log.Fatal(err)
	}
	return res

}

func outputEmbeddingsGOBNew(dat [][]string, res []*genai.BatchEmbedContentsResponse) {
	o, err := os.Create("../aigogo/internal/vecdb/embeddings.gob")
	if err != nil {
		log.Fatal(err)
	}
	defer o.Close()

	en := gob.NewEncoder(o)
	i := 0
	for _, v := range res {
		for _, w := range v.Embeddings {
			r := rag.Doc{}
			r.ID = dat[i][0]
			r.Title = dat[i][1]
			r.Content = dat[i][2]
			r.Context = dat[i][3]
			r.Embedding = w.Values
			if os.Getenv("DEBUG") != "" {
				fmt.Println(r.ID, r.Title, i)
			}
			if err := en.Encode(r); err != nil {
				log.Fatal(err)
			}
			i += 1
		}
	}
}
func outputEmbeddingsGOB(dat [][]string, res *genai.BatchEmbedContentsResponse) {
	o, err := os.Create("../aigogo/internal/vecdb/embeddings.gob")
	if err != nil {
		log.Fatal(err)
	}
	defer o.Close()

	en := gob.NewEncoder(o)
	for i, v := range dat[1:] {
		r := rag.Doc{}
		r.ID = v[0]
		r.Title = v[1]
		r.Content = v[2]
		r.Context = v[3]
		r.Embedding = res.Embeddings[i].Values
		if os.Getenv("DEBUG") != "" {
			log.Println(r.ID, r.Title, i)
		}
		if err := en.Encode(r); err != nil {
			log.Fatal(err)
		}
	}

}
