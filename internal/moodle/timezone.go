package moodle

import (
	"time"
	_ "time/tzdata"
)

var swissLocation = loadSwissLocation()

func loadSwissLocation() *time.Location {
	location, err := time.LoadLocation("Europe/Zurich")
	if err != nil {
		return time.Local
	}
	return location
}

func parseSwissTimestamp(value string) (time.Time, error) {
	return time.ParseInLocation("2.1.2006 15:04", value, swissLocation)
}
