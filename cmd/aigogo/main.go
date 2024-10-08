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
	"github.com/siuyin/randw"
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

func init() {
	client.ModelName = "gemini-1.5-flash-latest"
	cl = client.New()
	// safety settings doc: https://cloud.google.com/vertex-ai/generative-ai/docs/multimodal/configure-safety-attributes#gemini-TASK-samples-go
	// HarmBlockNone is available only on "invoiced accounts".
	cl.Model.SafetySettings = []*genai.SafetySetting{
		{Category: genai.HarmCategoryDangerousContent, Threshold: genai.HarmBlockOnlyHigh}, // need to set to High to avoid Daisy Bell trigger
		{Category: genai.HarmCategorySexuallyExplicit, Threshold: genai.HarmBlockOnlyHigh}, // default is low as Daisy Bell triggers this
		{Category: genai.HarmCategoryHarassment, Threshold: genai.HarmBlockOnlyHigh},
		{Category: genai.HarmCategoryHateSpeech, Threshold: genai.HarmBlockOnlyHigh},
		// {Category: genai.HarmCategoryDangerous, Threshold: genai.HarmBlockOnlyHigh},  // 400 response
		// {Category: genai.HarmCategorySexual, Threshold: genai.HarmBlockOnlyHigh},  // 400 response
		// {Category: genai.HarmCategoryViolence, Threshold: genai.HarmBlockOnlyHigh},  // 400 response
		// {Category: genai.HarmCategoryToxicity, Threshold: genai.HarmBlockOnlyHigh},  // 400 response
		// {Category: genai.HarmCategoryDerogatory, Threshold: genai.HarmBlockOnlyHigh}, // 400 response
		// {Category: genai.HarmCategoryUnspecified, Threshold: genai.HarmBlockOnlyHigh}, // 400 response
		// {Category: genai.HarmCategoryMedical, Threshold: genai.HarmBlockOnlyHigh}, // 400 response
	}
	temp := float32(0.0)
	cl.Model.GenerationConfig.Temperature = &temp
	em = initEmbeddingClient()
	collection = initDB()
	mapsClient = initMapsClient()
	initAigogoDataPath()

	log.Println("application initialised")
}

func main() {
	defer cl.Close()

	depl := dflt.EnvString("DEPLOY", "DEV")
	if depl == "DEV" {
		tmpl = template.Must(template.ParseGlob("./internal/public/*.html"))
		http.Handle("/", http.FileServer(http.Dir("./internal/public"))) // DEV
	} else {
		tmpl = template.Must(template.ParseFS(public.Content, "*.html"))
		http.Handle("/", http.FileServer(http.FS(public.Content))) // PROD
	}

	http.HandleFunc("/{$}", indexFunc)

	http.HandleFunc("/personallog", personalLogFunc)

	http.HandleFunc("/memories", memoriesFunc)

	http.HandleFunc("/memgen", memGenFunc)

	http.HandleFunc("/ref", personalLogDetails)

	http.HandleFunc("/retr", retrievalFunc)

	http.HandleFunc("/loc", locationFunc)

	http.HandleFunc("/data", dataWrite)

	http.HandleFunc("/getHighlightSelections", loadSelFunc)

	http.HandleFunc("/userIDExist", userIDExistFunc)

	http.HandleFunc("/life", life)

	log.Println("starting web server")
	log.Fatal(http.ListenAndServe(":"+dflt.EnvString("HTTP_PORT", "8080"), nil))
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

// ------------------------------------------------

func indexFunc(w http.ResponseWriter, _ *http.Request) {
	if err := tmpl.ExecuteTemplate(w, "main.html", tmplDat{Body: "main", JS: "/main.js"}); err != nil {
		io.WriteString(w, err.Error())
	}
}

func personalLogFunc(w http.ResponseWriter, _ *http.Request) {
	if err := tmpl.ExecuteTemplate(w, "main.html", tmplDat{Body: "personallog", JS: "/personallog.js"}); err != nil {
		io.WriteString(w, err.Error())
	}
}

func memoriesFunc(w http.ResponseWriter, _ *http.Request) {
	if err := tmpl.ExecuteTemplate(w, "main.html", tmplDat{Body: "memories", JS: "/memories.js"}); err != nil {
		io.WriteString(w, err.Error())
	}
}

func memGenFunc(w http.ResponseWriter, r *http.Request) {
	logEntr := randSelection(personalLogEntries(r.FormValue("userID")), 5)
	generateMemories(logEntr, w, r)
}

func retrievalFunc(w http.ResponseWriter, r *http.Request) {
	qry := r.FormValue("userPrompt")
	doc := retrieveDocsForAugmentation(r, qry)
	if len(doc) == 0 {
		io.WriteString(w, "No relevant documents found")
		return
	}
	//writeRetrievedDocs(w, doc)
	if os.Getenv("TESTING") != "" {
		fmt.Fprintf(w, "calling augmentGenerationWithDoc: %v", doc)
		return
	}
	augmentGenerationWithDoc(w, r, doc)
}

func locationFunc(w http.ResponseWriter, r *http.Request) {
	if r.FormValue("latlng") == "" {
		io.WriteString(w, "latlng required")
		return
	}
	if os.Getenv("TESTING") != "" {
		io.WriteString(w, "123 A Street, B City")
		return
	}

	res := getLocationAPIResp(r)

	var mapRes *mapResponse
	mapRes = decodeLocationAPIResp(res, mapRes)
	fmt.Fprintf(w, "%s", mapRes.Results[0].FormattedAddress)
}

func augmentGenerationWithDoc(w http.ResponseWriter, r *http.Request, doc []string) {
	defineSystemInstructionWithDocs(doc, r)
	streamResponseFromUserPrompt(r.FormValue("userPrompt"), w)
}

func dataWrite(w http.ResponseWriter, r *http.Request) {
	if r.FormValue("userID") == "" || (r.FormValue("filename") == "" && r.FormValue("editedlog") == "") {
		io.WriteString(w, "userID and (filename or editedlog required)")
		return
	}

	ter := r.FormValue("ter") // test request
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

func life(w http.ResponseWriter, r *http.Request) {
	latlng := r.FormValue("latlng")
	meaningOfLife(w, r.FormValue("loc"), time.Now().In(tzLoc(latlng)).Format("Monday, 15:04PM, 2 January 2006"))
}

func loadSelFunc(w http.ResponseWriter, r *http.Request) {
	if r.FormValue("userID") == "" {
		io.WriteString(w, "userID required")
		return
	}

	s := loadCustomHighlights(r.FormValue("userID"))
	if os.Getenv("TESTING") != "" {
		fmt.Fprintf(w, "custom highlights loaded: %v", s)
		return
	}

	b, err := json.Marshal(s)
	if err != nil {
		fmt.Fprintf(w, "could not retrieve highlights.txt file: %v", err)
	}
	w.Write(b)
}

func userIDExistFunc(w http.ResponseWriter, r *http.Request) {
	if r.FormValue("userID") != "123456" {
		io.WriteString(w, "")
		return
	}
	io.WriteString(w, "Kit Siew")
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
			fmt.Fprintf(w, "<p>I've encounted an issue: %v:", err)
			resp = iter.MergedResponse()
			if resp == nil {
				return
			}
			log.Printf("num cand: %v, finishReason: %v", len(resp.Candidates), resp.Candidates[0].FinishReason)
			for _, c := range resp.Candidates {
				for _, sr := range c.SafetyRatings {
					fmt.Fprintf(w, " category:%v, probability: %v, blocked: %v", sr.Category, sr.Probability, sr.Blocked)
				}
			}
			return
		}
		fPrintResponse(w, resp)
	}
}

func defineSystemInstructionWithDocs(doc []string, r *http.Request) {
	location := r.FormValue("loc")
	latlng := r.FormValue("latlng")
	weatherJSON := r.FormValue("weather")
	currentTime := time.Now().In(tzLoc(latlng)).Format("Monday, 15:04PM, 2 January 2006")

	rwords := fmt.Sprintf("%v", randw.Select(5))
	if !strings.Contains(strings.ToLower(r.FormValue("userPrompt")), "random") {
		rwords = ""
	}
	log.Printf("random words: %v", rwords)

	cl.Model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text(fmt.Sprintf(`You are a considerate and kind
		caregiver for an aged person. If asked your name is AiGoGo.
		You  aim to entertain and engage with the
		person to maintain her mental acuity and to stave off dementia.
		Your responses are kind but authoritative and firm. Below are RESOURCE 1 and
		RESOURCE 2 from experienced caregiver KitSiew. 
		
		RESOURCE 1: %v

		RESOURCE 2: %v

		Prioritize including suggestions from the above resources if and only if they
		are relevent. Where there
		are multiple equally feasible options, pick one at random. If you feel
		one is particularly relevant explain why the reasoning behind your choice.

		If a RESOURCE has safety relevant warnings like "it is not well lit" you must
		adhere to this infomation and do not express your own opinion. Stick to the facts.

		If a RESOURCE is not relevant to the user's question, you may ignore its contents.

		If the user's prompt includes the word "Randomize" or "random" you must use the words
		in the RANDOM WORDS section below in your output. 

		RANDOM WORDS: %v

		If there is insufficent data, please supplement your response with what you know.
		Some of the user prompts or queries will relate to singing songs. In this case,
		look up what you know about the relevant song title's lyrics and include some
		portion of the lyrics into your response.

		DO NOT quote RESOURCE 1, RESOURCE 2 or RANDOM WORDS -- they are for your internal use and
		reference.

		When formulating your response consider the current date and time: %s,
		timezone: %s,and also the user's location: %s.
		
		This is particulary important when your response includes an outdoor activity
		as the elderly may trip and fall in the dark.
		You may mention the time and timezone in your response.

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
		on https://beyondbroadcast.com/ . Choose a fable that is connected to a word
		in RANDOM WORDS.
		
		Make at least two recommendations, the main recommendation and the alternative.
		Make it clear that the user has a choice.`,
			doc[0], doc[1], rwords, currentTime, tzLoc(latlng).String(), location, weatherJSON))},
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
		`, location, currentTime))},
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
	if os.Getenv("TESTING") != "" {
		return []string{"testDoc1", "testDoc2"}
	}
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
	if os.Getenv("TESTING") != "" {
		io.WriteString(w, "calling saveAudioFile and transcribeAudio")
		return
	}
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
	if os.Getenv("TESTING") != "" {
		io.WriteString(w, "calling saveEditedLog and saving summary")
		return
	}
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

func loadCustomHighlights(userID string) []string {
	f, err := os.Open(dataPath + "/" + userID + "/highlights.txt")
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
	l := len(list)
	if l < n {
		n = l
	}
	perms := rand.Perm(l)
	s := []string{}
	for _, p := range perms[:n] {
		s = append(s, list[p])
	}
	return s
}

func generateMemories(logEntr []string, w http.ResponseWriter, r *http.Request) {
	if r.FormValue("userID") == "" {
		io.WriteString(w, "Error: empty userID received")
		return
	}

	cl.Model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text(`You are a young personal
		assitant to an older person. You have a bubbly and cheerful personality. 
		If asked, your name is AiGoGo.
		You  aim to entertain and engage with the
		person to maintain her mental acuity and to stave off dementia.

		When quoting an event, you must state the date and/or time in the form
		"(5 Aug 2024)" or "(5 Aug 2024, 15:25UTC)".
		Extract the day,date and time from the 
		log entry line (eg. "log-2024-08-04T02:25:10.513Z").
		
		If the data provided in the user prompt is not relevant, you may
		extrapolate and generate content. However you must explicitly state
		that you are doing this.

		At the end of your output you must quote all the log entries
		i.e. the lines similar to "log-2024-08-04T02:25:10.513Z",
		wrapped in html links similar to
		<a href="/ref?log=log-2024-08-04T02:25:10.513Z" class="popup">log-2024-08-04T02:25:10.513Z</a>
		comma seperated,
		preceeded by "ref:[" and closed with "]". 

		Limit your output to 65 words.
		`)},
	}

	logEntries := getLogEntries(logEntr, r.FormValue("userID"))
	userPrompt := r.FormValue("userPrompt") + "\n" + logEntries
	if os.Getenv("TESTING") != "" {
		io.WriteString(w, "calling GenerateContentStream")
		return
	}

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

func personalLogDetails(w http.ResponseWriter, r *http.Request) {
	if r.FormValue("log") == "" || r.FormValue("userID") == "" {
		io.WriteString(w, "log and userID required")
		return
	}

	if os.Getenv("TESTING") != "" {
		io.WriteString(w, "populating log details")
		return
	}

	type logDet struct {
		UserID     string
		Basename   string
		Date       string
		Summary    string
		Transcript string
		Audio      []byte
	}
	dt, err := time.Parse("log-2006-01-02T15:04:05.000Z", r.FormValue("log"))
	if err != nil {
		log.Printf("could not parse time from log basename: %v", err)
		return
	}
	det := logDet{
		UserID: r.FormValue("userID"), Basename: r.FormValue("log"),
		Date:       dt.Format("Monday, 2 Jan 2006, 15:04:05 UTC"),
		Summary:    getBody(r.FormValue("log")+".summary.txt", r.FormValue("userID")),
		Transcript: getBody(r.FormValue("log")+".txt", r.FormValue("userID")),
		Audio:      []byte(getBody(r.FormValue("log")+".ogg", r.FormValue("userID"))),
	}
	b, err := json.Marshal(det)
	if err != nil {
		log.Println(err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}
