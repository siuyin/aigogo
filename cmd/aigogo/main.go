package main

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"math/rand"
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
	"github.com/siuyin/aigotut/gfmt"
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

	tmpl *template.Template
)

const dataPath = "/data/aigogo"

type mapResponse struct {
	Results []struct {
		FormattedAddress string `json:"formatted_address"`
	} `json:"results"`
}

type tmplDat struct {
	Body string
	JS   string
}

func main() {

	depl := dflt.EnvString("DEPLOY", "DEV")
	if depl == "DEV" {
		tmpl = template.Must(template.ParseGlob("./internal/public/*.html"))
		http.Handle("/", http.FileServer(http.Dir("./internal/public"))) // DEV
	} else {
		tmpl = template.Must(template.ParseFS(public.Content, "*.html"))
		http.Handle("/", http.FileServer(http.FS(public.Content))) // PROD
	}

	indexFunc := func(w http.ResponseWriter, r *http.Request) {
		if err := tmpl.ExecuteTemplate(w, "main.html", tmplDat{Body: "main", JS: "/main.js"}); err != nil {
			io.WriteString(w, err.Error())
		}
	}
	http.HandleFunc("/{$}", indexFunc)

	personalLogFunc := func(w http.ResponseWriter, r *http.Request) {
		if err := tmpl.ExecuteTemplate(w, "main.html", tmplDat{Body: "personallog", JS: "/personallog.js"}); err != nil {
			io.WriteString(w, err.Error())
		}
	}
	http.HandleFunc("/personallog", personalLogFunc)

	memoriesFunc := func(w http.ResponseWriter, r *http.Request) {
		if err := tmpl.ExecuteTemplate(w, "main.html", tmplDat{Body: "memories", JS: "/memories.js"}); err != nil {
			io.WriteString(w, err.Error())
		}
	}
	http.HandleFunc("/memories", memoriesFunc)

	memGenFunc := func(w http.ResponseWriter, r *http.Request) {
		logEntr := randSelection(personalLogEntries(r.FormValue("userID")), 5)
		generateMemories(logEntr, w, r)
	}
	http.HandleFunc("/memgen", memGenFunc)

	http.HandleFunc("/ref", func(w http.ResponseWriter, r *http.Request) {
		personalLogDetails(w, r)
	})

	retrievalFunc := func(w http.ResponseWriter, r *http.Request) {
		qry := r.FormValue("userPrompt")
		doc := retrieveDocsForAugmentation(r, qry)
		if len(doc) == 0 {
			io.WriteString(w, "No relevant documents found")
			return
		}
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

	dataWrite := func(w http.ResponseWriter, r *http.Request) {
		ter := r.FormValue("ter")
		if ter != "" {
			processTestRequest(w, r)
		}
		filename := r.FormValue("filename")
		if filename != "" {
			saveAudioLog(w, r)
		}
		editedlog := r.FormValue("editedlog")
		if editedlog != "" {
			saveEditedLogAndSummary(w, r)
		}
	}
	http.HandleFunc("/data", dataWrite)

	loadSelFunc := func(w http.ResponseWriter, r *http.Request) {
		s := loadCustomHighlights()
		b, err := json.Marshal(s)
		if err != nil {
			fmt.Fprintf(w, "could not retrieve highlights.txt file: %v", err)
		}
		w.Write(b)
	}
	http.HandleFunc("/getHighlightSelections", loadSelFunc)

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
	temp := float32(0.0)
	cl.Model.SafetySettings = []*genai.SafetySetting{
		// {Category: genai.HarmCategoryDangerousContent, Threshold: genai.HarmBlockOnlyHigh},
		// {Category: genai.HarmCategoryMedical,Threshold: genai.HarmBlockMediumAndAbove},
	}
	cl.Model.GenerationConfig.Temperature = &temp
	em = initEmbeddingClient()
	collection = initDB()
	mapsClient = initMapsClient()
	initAigogoDataPath()

	log.Println("application initialised")
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

func initAigogoDataPath() {
	if err := os.MkdirAll(dataPath, 0750); err != nil {
		log.Fatal("ERROR: could not make aigogo data folder:", err)
		return
	}
}
func augmentGenerationWithDoc(w http.ResponseWriter, r *http.Request, doc []string) {
	defineSystemInstructionWithDocs(doc, r)
	streamResponseFromUserPrompt(r.FormValue("userPrompt"), w)
}
func streamResponseFromUserPrompt(userPrompt string, w http.ResponseWriter) {
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

func defineSystemInstructionWithDocs(doc []string, r *http.Request) {
	location := r.FormValue("loc")
	latlng := r.FormValue("latlng")
	weatherJSON := r.FormValue("weather")
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
		timezone: %s,
		and also the user's location: %s. This is particulary important when your response
		includes an outdoor activity as the elderly may trip and fall in the dark.
		You must always mention the time and timezone in your response.

		Evaluate the following weather forecast JSON:
		%s

		If the user is contemplating an outdoor activity you must provide 
		a summary of your interpretation of the weather forecast that
		includes actual temperature range, perceived temperature range and chance of
		percipitation as a percentage.

		If you are quoting a time or a day part, eg. morning, afternoon, evening,
		be sure to mention the time together with the
		long form timezone (eg. Singapore Time or Mountain Standard Time).

		Try to weave in a relevant Aesop fable and look up Kit Siew's life lessons
		on https://beyondbroadcast.com/ . Do not overuse the Tortoise and the Hare.
		
		Make at least two recommendations, the main recommendation and the alternative.
		Make it clear that the user has a choice.`,
			doc[0], doc[1], currentTime, tzLoc(latlng).String(), location, weatherJSON))},
	}

}

func getLocationAPIResp(r *http.Request) *http.Response {
	key := dflt.EnvString("MAPS_API_KEY", "use your real api key")
	latlng := r.FormValue("latlng")
	res, err := http.Get(fmt.Sprintf("https://maps.googleapis.com/maps/api/geocode/json?latlng=%s&result_type=street_address&key=%s", latlng, key))
	if err != nil {
		log.Println(err)
	}
	return res
}
func decodeLocationAPIResp(res *http.Response, mapRes *mapResponse) *mapResponse {
	dec := json.NewDecoder(res.Body)
	if err := dec.Decode(&mapRes); err != nil {
		log.Println(err)
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
		ID:       rec.ID,
		Content:  rec.Title + " | " + rec.Content,
		Metadata: map[string]string{"context": rec.Context},
	}
	if os.Getenv("DEBUG") != "" {
		log.Println("adding: ", rec.Title, rec.Context)
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

func retrieveDocsForAugmentation(r *http.Request, qry string) []string {
	ctx := context.Background()
	res, err := em.EmbedContent(ctx, genai.Text(qry))
	if err != nil {
		log.Fatal(err)
	}

	numResults := 2
	usrCtx := r.FormValue("ctx")
	qres, err := collection.QueryEmbedding(ctx, res.Embedding.Values, numResults, map[string]string{"context": usrCtx}, nil)
	if err != nil {
		log.Fatal(err)
	}

	doc := []string{}
	for i := 0; i < len(qres); i++ {
		if os.Getenv("DEBUG") != "" {
			fmt.Println("vector DB:", qres[i].ID, qres[i].Content)
		}
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

type sampleData struct {
	ID      string
	User    string
	TimeStr string
	Time    time.Time
}

func saveAudioLog(w http.ResponseWriter, r *http.Request) {
	aud := saveAudioFile(w, r)
	transcribeAudio(aud, w)
}

func transcribeAudio(dat []byte, w http.ResponseWriter) {
	customNames := loadCustomNames()
	cl.Model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text("")}}
	prompt := fmt.Sprintf(`Please transcribe the following audio.
	If you come across terms that you are unfamiliar with look up the following table to see one of the entries matches:
	%s`, customNames)
	resp, err := cl.Model.GenerateContent(context.Background(), genai.Blob{MIMEType: "audio/ogg", Data: dat}, genai.Text(prompt))
	if err != nil {
		log.Printf("WARNING: transcription failure: %v", err)
		return
	}
	gfmt.FprintResponse(w, resp)
}

func loadCustomNames() string {
	f, err := os.Open(dataPath + "/123456/names.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	b, err := io.ReadAll(f)
	if err != nil {
		log.Fatal(err)
	}
	return string(b)
}

func saveAudioFile(w http.ResponseWriter, r *http.Request) []byte {
	dat, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Fprintf(w, "could not read request body: %v", err)
		return []byte{}
	}
	af := logFile{
		userID:   r.FormValue("userID"),
		basename: r.FormValue("filename"),
		ext:      "ogg",
		body:     dat,
	}
	createFile(af)

	return dat
}

type logFile struct {
	userID   string
	basename string
	ext      string
	body     []byte
}

func createFile(lf logFile) {
	f, err := os.Create(dataPath + "/" + lf.userID + "/" + lf.basename + "." + lf.ext)
	if err != nil {
		log.Fatalf("ERROR: could not create %s.%s: %v", lf.basename, lf.ext, err)
		return
	}
	defer f.Close()

	f.Write(lf.body)
}

func saveEditedLogAndSummary(w http.ResponseWriter, r *http.Request) {
	editedLog := saveEditedLog(w, r)
	summary := summarize(editedLog, w)
	sm := logFile{
		userID:   r.FormValue("userID"),
		basename: r.FormValue("editedlog"),
		ext:      "summary.txt",
		body:     summary,
	}
	createFile(sm)
}

func summarize(dat []byte, w http.ResponseWriter) []byte {
	prompt := fmt.Sprintf(`Please summarize the following text in the first person.
	Keep the metadata (lines following the ---) intact:
	%s`, dat)
	resp, err := cl.Model.GenerateContent(context.Background(), genai.Text(prompt))
	if err != nil {
		log.Printf("WARNING: summarization failure: %v", err)
		return []byte{}
	}
	gfmt.FprintResponse(w, resp)

	var b bytes.Buffer
	gfmt.FprintResponse(&b, resp)

	return b.Bytes()
}
func saveEditedLog(w http.ResponseWriter, r *http.Request) []byte {
	metadata := fmt.Sprintf("\n---\nlatlng:%s, neighborhood:%s, primaryHighlight:%s, secondaryHighlight:%s, people:%s",
		r.FormValue("latlng"), r.FormValue("neighborhood"), r.FormValue("primary"), r.FormValue("secondary"),
		r.FormValue("people"))

	dat, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Fprintf(w, "could not read request body: %v", err)
		return []byte{}
	}
	dat = []byte(string(dat) + metadata)

	editedLog := logFile{
		userID:   r.FormValue("userID"),
		basename: r.FormValue("editedlog"),
		ext:      "txt",
		body:     dat,
	}
	createFile(editedLog)
	return dat
}

func processTestRequest(w http.ResponseWriter, r *http.Request) {
	ter := r.FormValue("ter")
	log.Printf("rececived: ter=%s", ter)
	dat, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Fprintf(w, "could not read request body: %v", err)
		return
	}
	var sd sampleData
	if err := json.Unmarshal(dat, &sd); err != nil {
		log.Printf("ERROR: could not unmarshal data: %v", err)
		return
	}

	t, err := time.Parse(time.RFC3339, sd.TimeStr)
	if err != nil {
		log.Printf("ERROR: could not parse time string: %s: %v", sd.TimeStr, err)
		return
	}
	sd.Time = t

	if err := os.MkdirAll(dataPath+"/"+sd.ID, 0750); err != nil {
		log.Fatalf("ERROR: could not create user folder: %v", err)
		return
	}

	f, err := os.Create(dataPath + "/" + sd.ID + "/test.json")
	if err != nil {
		log.Fatalf("ERROR: could not create test.json: %v", err)
		return
	}
	defer f.Close()

	f.Write(dat)

	fmt.Fprintf(w, "data write request received: %#v", sd)
}

func loadCustomHighlights() []string {
	f, err := os.Open(dataPath + "/123456/highlights.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	b, err := io.ReadAll(f)
	if err != nil {
		log.Fatal(err)
	}
	s := strings.Split(string(b), "\n")
	return s[:len(s)-1]
}

func personalLogEntries(userID string) []string {
	d := os.DirFS(dataPath + "/" + userID)
	m, err := fs.Glob(d, "*.summary.txt")
	if err != nil {
		log.Fatal(err)
	}
	return m
}

func randSelection(list []string, n int) []string {
	if l := len(list); l < n {
		n = l
	}
	perms := rand.Perm(n)
	s := []string{}
	for _, p := range perms {
		s = append(s, list[p])
	}
	return s
}

func generateMemories(logEntr []string, w http.ResponseWriter, r *http.Request) {
	cl.Model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text(`You are a young personal
		assitant to an older person. You have a bubbly and cheerful personality. 
		If asked, your name is AiGoGo.
		You  aim to entertain and engage with the
		person to maintain her mental acuity and to stave off dementia.

		When quoting event, you must state the day, date and/or time in the form
		"(Monday, 5 Aug 2024)" or "(5 Aug 2024, 3:25pm)".
		Extract the day,date and time from the 
		log entry line (eg. "log-2024-08-04T02:25:10.513Z").
		
		If the data provided in the user prompt is not relevant, you may
		extrapolate and generate content. However you must explicitly state
		that you are doing this.

		At the end of your output you must quote the log entries
		just only the lines similar to "log-2024-08-04T02:25:10.513Z",
		wrapped in html links similar to
		<a href="/ref?log=log-2024-08-04T02:25:10.513Z" class="popup">log-2024-08-04T02:25:10.513Z</a>
		comma seperated,
		preceeded by "ref:[" and closed with "]". 
		`)},
	}
	logEntries := getLogEntries(logEntr, r.FormValue("userID"))
	userPrompt := r.FormValue("userPrompt") + "\n" + logEntries
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

func logBasename(fn string) string {
	return strings.Split(fn, ".summary.txt")[0]
}

func getLogEntries(logEntr []string, userID string) string {
	s := ""
	for _, e := range logEntr {
		bn := logBasename(e)
		body := getBody(bn+".txt", userID) // use the transcript and not the summary
		s += bn + ":\n" + body + "\n\n"
	}
	return s
}

func getBody(fn string, userID string) string {
	f, err := os.Open(dataPath + "/" + userID + "/" + fn)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	b, err := io.ReadAll(f)
	if err != nil {
		log.Fatal(err)
	}
	return string(b)
}

func writeLogEntryMarkdown(logEntr []string, w http.ResponseWriter, r *http.Request) {
	s := "\n\n"
	for _, e := range logEntr {
		s += fmt.Sprintf("%s:\n\n", logBasename(e))
		s += fmt.Sprintln(getBody(e, r.FormValue("userID")))
		s += fmt.Sprintln(`\n[transcript](/)  [audio](/)\n\n`)
	}
	io.WriteString(w, s)
}

func personalLogDetails(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "TODO:")
	io.WriteString(w, r.FormValue("log"))
}
