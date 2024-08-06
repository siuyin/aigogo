package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"googlemaps.github.io/maps"
)

func TestTimezone(t *testing.T) {
	if os.Getenv("TESTING") != "" {
		t.Skip()
		return
	}
	id, name := localTimezoneName(&maps.LatLng{Lat: 1.3545457, Lng: 103.7636865})
	fmt.Println(id, name)
}

func TestPersonalLogEntries(t *testing.T) {
	le := personalLogEntries("123456")
	if n := len(le); n == 0 {
		t.Errorf("number of entries: %v should not be zero", n)
	}
}

func TestRandSlection(t *testing.T) {
	list := personalLogEntries("123456")
	sample := randSelection(list, 5)
	if len(sample) == 0 {
		t.Errorf("sample:%#v should not be empty", sample)
	}
}

func TestLogBasename(t *testing.T) {
	fn := "log-2024-08-04T02:25:10.513Z.summary.txt"
	if bn := logBasename(fn); bn != "log-2024-08-04T02:25:10.513Z" {
		t.Errorf("basename should not be %s", bn)
	}
}

func TestGetLogEntries(t *testing.T) {
	logEntr := randSelection(personalLogEntries("123456"), 5)
	s := getLogEntries(logEntr, "123456")
	if s == "" {
		t.Errorf("%s: should not be empty", s)
	}
}

func TestPages(t *testing.T) {
	tmpl = template.Must(template.ParseGlob("./internal/public/*.html"))

	t.Run("PersonalLog", func(t *testing.T) {
		testPage(t, personalLogFunc, "/personallog", "<h1>Personal Log")
	})
	t.Run("Memories", func(t *testing.T) {
		testPage(t, memoriesFunc, "/memories", "<h1>Memories")
	})
	t.Run("Index", func(t *testing.T) {
		testPage(t, indexFunc, "/", "<h1>AiGoGo")
	})
	t.Run("MemoryGen", func(t *testing.T) {
		testPage(t, memGenFunc, "/memories?userID=123456", "calling GenerateContentStream")
	})
	t.Run("LogDetails", func(t *testing.T) {
		testPage(t, personalLogDetails, "/ref?userID=123456&log=log-somedate", "populating log details")
	})
	t.Run("RetrieveAugmentDoc", func(t *testing.T) {
		testPage(t, retrievalFunc, "/retr?userPrompt=someprompt", "calling augmentGenerationWithDoc: [testDoc1 testDoc2]")
	})
	t.Run("Location", func(t *testing.T) {
		testPage(t, locationFunc, "/loc?latlng=1.23,4.56", "123 A Street, B City")
	})
	t.Run("dataSaveAudio", func(t *testing.T) {
		testPage(t, dataWrite, "/data?filename=somefile&userID=123456", "calling saveAudioFile and transcribeAudio")
	})
	t.Run("dataSaveEditedAndSummary", func(t *testing.T) {
		testPage(t, dataWrite, "/data?editedlog=somefile&userID=123456", "calling saveEditedLog and saving summary")
	})
	t.Run("GetHighlighSelections", func(t *testing.T) {
		testPage(t, loadSelFunc, "/getHighlightSelections?userID=123456", "custom highlights loaded:")
	})
}

func testPage(t *testing.T, fn func(w http.ResponseWriter, r *http.Request), path string, fragment string) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fn(w, r) }))
	defer ts.Close()

	res, err := http.Get(ts.URL + path)
	if err != nil {
		t.Error(err)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Error(err)
	}
	if len(body) == 0 {
		t.Errorf("body should not be empty:\n%s", body)
	}
	if !bytes.Contains(body, []byte(fragment)) {
		t.Errorf("expected fragment: %s not found in body: %s", fragment, body)
	}

}
