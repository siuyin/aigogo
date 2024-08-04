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
