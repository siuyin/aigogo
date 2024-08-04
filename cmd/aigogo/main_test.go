package main

import (
	"fmt"
	"os"
	"testing"

	"googlemaps.github.io/maps"
)

func TestTimezone(t *testing.T) {
	if os.Getenv("MAPS_API_KEY") != "" {
		id, name := localTimezoneName(&maps.LatLng{Lat: 1.3545457, Lng: 103.7636865})
		fmt.Println(id, name)
	}
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
	s := getLogEntries(logEntr,"123456")
	if s == "" {
		t.Errorf("%s: should not be empty", s)
	}
}
