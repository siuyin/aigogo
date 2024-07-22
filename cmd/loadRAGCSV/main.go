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

const batchSize = 90

func main() {
	dat := loadRAGCSV()
	res := embed(batchSize, dat[1:])
	outputEmbeddingsGOB(dat[1:], res)
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

func embed(batchSize int, dat [][]string) []*genai.BatchEmbedContentsResponse {
	res := []*genai.BatchEmbedContentsResponse{}
	for i := 0; i < len(dat); i += batchSize {
		end := i + batchSize
		if end > len(dat) {
			end = len(dat)
		}
		bat := dat[i:end]
		fmt.Println(len(bat),i,end)
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
	for _, v := range dat {
		b.AddContentWithTitle(v[1], genai.Text(v[2]))
	}

	res, err := em.BatchEmbedContents(ctx, b)
	if err != nil {
		log.Fatal(err)
	}
	return res

}

func outputEmbeddingsGOB(dat [][]string, res []*genai.BatchEmbedContentsResponse) {
	o, err := os.Create("../../data/embeddings.gob")
	if err != nil {
		log.Fatal(err)
	}
	defer o.Close()

	en := gob.NewEncoder(o)
	i := 0
	for _, v := range res {
		fmt.Println(len(v.Embeddings))
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