package main

import (
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/google/generative-ai-go/genai"
	"github.com/philippgille/chromem-go"
	"github.com/siuyin/aigogo/cmd/aigogo/internal/public"
	"github.com/siuyin/aigogo/cmd/aigogo/internal/vecdb"
	"github.com/siuyin/aigogo/rag"
	"github.com/siuyin/aigotut/client"
	"github.com/siuyin/dflt"
	"google.golang.org/api/iterator"
	"googlemaps.github.io/maps"
)

var (
	cl *client.Info // LLM client

	emCl       *client.Info // embedding client
	em         *genai.EmbeddingModel
	collection *chromem.Collection
	db         *chromem.DB

	mapsClient *maps.Client
)

type mapResponse struct {
	Results []struct {
		FormattedAddress string `json:"formatted_address"`
	} `json:"results"`
}

func main() {
	depl := dflt.EnvString("DEPLOY", "DEV")
	if depl == "DEV" {
		http.Handle("/", http.FileServer(http.Dir("./internal/public"))) // DEV
	} else {
		http.Handle("/", http.FileServer(http.FS(public.Content))) // PROD
	}

	retrievalFunc := func(w http.ResponseWriter, r *http.Request) {
		qry := r.FormValue("userPrompt")
		doc := retrieveDocsForAugmentation(qry)
		//writeRetrievedDocs(w, doc)
		augmentGenerationWithDoc(w, r, doc)
	}
	http.HandleFunc("/retr", retrievalFunc)

	locationFunc := func(w http.ResponseWriter, r *http.Request) {
		res := getLocationAPIResp(r)

		var mapRes *mapResponse
		mapRes = decodeLocationAPIResp(res, mapRes)
		fmt.Fprintf(w, "%s", mapRes.Results[0].FormattedAddress)
	}
	http.HandleFunc("/loc", locationFunc)

	life := func(w http.ResponseWriter, r *http.Request) {
		latlng := r.FormValue("latlng")
		meaningOfLife(w, r.FormValue("loc"), time.Now().In(tzLoc(latlng)).Format("Monday, 03:04PM, 2 January 2006"))
	}
	defer cl.Close()
	http.HandleFunc("/life", life)

	http.HandleFunc("/hello/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello World! It is %v\n", time.Now().Format("15:04:05.000 MST"))
	})

	h1 := func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, "Hello from a HandleFunc #1.\n")
	}
	http.HandleFunc("/h1", h1)

	h2 := func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, "Hello from a HandleFunc #2!\n")
	}
	http.HandleFunc("/endpoint", h2)

	log.Println("starting web server")
	log.Fatal(http.ListenAndServe(":"+dflt.EnvString("HTTP_PORT", "8080"), nil))
}

func init() {
	if os.Getenv("API_KEY") == "" || os.Getenv("MAPS_API_KEY") == "" { // we are in testing mode
		return
	}
	cl = client.New()
	em = initEmbeddingClient()
	collection = initDB()
	mapsClient = initMapsClient()
}

func initEmbeddingClient() *genai.EmbeddingModel {
	client.ModelName = "text-embedding-004"
	emCl = client.New()
	em := emCl.Client.EmbeddingModel(client.ModelName)
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
func initMapsClient() *maps.Client {
	cl, err := maps.NewClient(maps.WithAPIKey(os.Getenv("MAPS_API_KEY")))
	if err != nil {
		log.Fatal(err)
	}
	return cl
}

// func writeRetrievedDocs(w http.ResponseWriter, doc []string) {
// 	for _, d := range doc {
// 		io.WriteString(w, "<p>"+d+"</p>")
// 	}
// }

func augmentGenerationWithDoc(w http.ResponseWriter, r *http.Request, doc []string) {
	//writeRetrievedDocs(w, doc)
	userPrompt := r.FormValue("userPrompt")
	location := r.FormValue("loc")
	latlng := r.FormValue("latlng")
	currentTime := time.Now().In(tzLoc(latlng)).Format("Monday, 03:04PM, 2 January 2006")

	cl.Model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text(fmt.Sprintf(`You are a considerate and kind
		caregiver for an aged person. If asked your name is AiGoGo.
		You  aim to entertain and engage with the
		person to maintain her mental acuity and to stave off dementia.
		Your responses are kind but authoritative and firm. Below are RESOURCE 1 and
		RESOURCE 2 from experienced caregiver KitSiew. 
		
		RESOURCE 1: %s

		RESOURCE 2: %s

		Prioritize including suggestions from the above resources if and only if they
		are relevent. Where there
		are multiple equally feasible options, pick one at random. If you feel
		one is particularly relevant explain why the reasoning behind your choice.

		If a RESOURCE has safety relevant warnings like "it is not well lit" you must
		adhere to this infomation and do not express your own opinion. Stick to the facts.

		If a RESOURCE is not relevant to the user's question, you may ignore its contents.

		If there is insufficent data, please supplement your response with what you know.
		Some of the user prompts or queries will relate to singing songs. In this case,
		look up what you know about the relevant song title's lyrics and include some
		portion of the lyrics into your response.

		DO NOT quote RESOURCE 1 or RESOURCE 2 directly -- they are for your internal use and
		reference.

		When formulating your response consider the current date and time: %s
		and also the user's location: %s. This is particulary important when your response
		includes an outdoor activity as the elderly may trip and fall in the dark.

		If you are quoting a time or a day part, eg. morning, afternoon, evening,
		be sure to mention the time together with the
		long form timezone (eg. Singapore Time or Mountain Standard Time).

		Try to weave in a relevant Aesop fable and look up Kit Siew's life lessons
		on https://beyondbroadcast.com/ . Do not overuse the Tortoise and the Hare.
		
		Make at least two recommendations, the main recommendation and the alternative.
		Make it clear that the user has a choice.`,
			doc[0], doc[1], currentTime, location))},
	}

	log.Println("calling generate content stream with: ", userPrompt)
	iter := cl.Model.GenerateContentStream(context.Background(),
		genai.Text(userPrompt))
	for {
		resp, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			io.WriteString(w, "<p>hmm.. apparently I have an issue:"+err.Error())
			return
		}

		fPrintResponse(w, resp)
	}
}

func getLocationAPIResp(r *http.Request) *http.Response {
	key := dflt.EnvString("MAPS_API_KEY", "use your real api key")
	latlng := r.FormValue("latlng")
	res, err := http.Get(fmt.Sprintf("https://maps.googleapis.com/maps/api/geocode/json?latlng=%s&result_type=street_address&key=%s", latlng, key))
	if err != nil {
		log.Fatal(err)
	}
	return res
}
func decodeLocationAPIResp(res *http.Response, mapRes *mapResponse) *mapResponse {
	dec := json.NewDecoder(res.Body)
	if err := dec.Decode(&mapRes); err != nil {
		log.Fatal(err)
	}

	res.Body.Close()
	if res.StatusCode > 299 {
		log.Fatalf("Response failed with status code: %d\n", res.StatusCode)
	}

	if len(mapRes.Results) == 0 {
		log.Fatal("no results returned")
	}
	return mapRes
}
func loadDocuments() []chromem.Document {
	var (
		f    io.ReadCloser
		err  error
		depl string
	)
	depl = dflt.EnvString("DEPLOY", "DEV")
	if depl == "DEV" {
		f, err = os.Open("./internal/vecdb/embeddings.gob") // DEV
		if err != nil {
			log.Fatal(err)
		}
	} else {
		f, err = vecdb.Content.Open("embeddings.gob") // PROD
		if err != nil {
			log.Fatal(err)
		}
	}
	defer f.Close()

	var rec rag.Doc
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

func addDoc(docs []chromem.Document, rec *rag.Doc) []chromem.Document {
	d := chromem.Document{
		ID:      rec.ID,
		Content: rec.Title + " | " + rec.Content,
		Metadata: map[string]string{"context":rec.Context},
	}
	if os.Getenv("DEBUG") != "" {
		log.Println("adding: ", rec.Title)
	}
	d.Embedding = append(d.Embedding, rec.Embedding...)
	docs = append(docs, d)
	return docs
}

func meaningOfLife(w http.ResponseWriter, location string, currentTime string) {
	cl.Model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text(fmt.Sprintf(`You are a philosophy professor
		who likes to quote Shakespear and answers questions with questions.
		Your response should be at least 100 words long.
		Weave into your response the user's location: %s
		and the current time %s.
		Format your output as plain text.`, location, currentTime))},
	}
	iter := cl.Model.GenerateContentStream(context.Background(),
		genai.Text("What is the meaning of life?"))
	for {
		resp, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			io.WriteString(w, "<p>hmm.. apparently I have an issue:"+err.Error())
			return
		}

		fPrintResponse(w, resp)
	}
}
func fPrintResponse(w http.ResponseWriter, resp *genai.GenerateContentResponse) {
	f, _ := w.(http.Flusher)
	for _, cand := range resp.Candidates {
		if cand.Content != nil {
			for _, part := range cand.Content.Parts {
				w.Write([]byte(part.(genai.Text)))
				f.Flush()
			}
		}
	}
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

func localTimezoneName(latlng *maps.LatLng) (string, string) {
	r := &maps.TimezoneRequest{Timestamp: time.Now(), Location: latlng}

	resp, err := mapsClient.Timezone(context.Background(), r)
	if err != nil {
		log.Fatalf("Timezone: %v", err)
	}
	return resp.TimeZoneID, resp.TimeZoneName
}

func latLng(latlng string) *maps.LatLng {
	part := strings.Split(latlng, ",")
	lat, err := strconv.ParseFloat(part[0], 64)
	if err != nil {
		log.Fatal(err)
	}

	lng, err := strconv.ParseFloat(part[1], 64)
	if err != nil {
		log.Fatal(err)
	}
	return &maps.LatLng{Lat: lat, Lng: lng}
}

func tzLoc(latlng string) *time.Location {
	zoneName, _ := localTimezoneName(latLng(latlng))
	loc, err := time.LoadLocation(zoneName)
	if err != nil {
		log.Fatal(err)
	}
	return loc
}

