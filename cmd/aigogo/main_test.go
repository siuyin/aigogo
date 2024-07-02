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
