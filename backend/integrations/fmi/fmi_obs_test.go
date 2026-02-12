package fmi

import (
	"encoding/json"
	"encoding/xml"
	"io/ioutil"
	"strings"
	"testing"
)

func LoadXml(t *testing.T, fn string, fmiObs *FMI_ObservationsModel, r Resolution) {
	data, err := ioutil.ReadFile(fn)
	if err != nil {
		t.Fatalf("Could not retrieve %s file: %v", fn, err)
	}
	err = xml.Unmarshal(data, &fmiObs.Observations)
	if err != nil {
		t.Fatalf("Could not retrieve %s file: %v", fn, err)
	}
	fmiObs.Observations.Resolution = r
}

type TestValues struct {
	RequestType          RequestType
	BeginPosition        string
	EndPosition          string
	FieldsLen            int
	TupleListMinLen      int
	ObservationsLen      int
	FirstObservationTime string
	LastObservationTime  string
	Resolution           Resolution
}

func TestWeatherDataMinutes(t *testing.T) {
	v := TestValues{
		RequestType:          Observations,
		Resolution:           Minutes,
		BeginPosition:        "2022-10-10T02:50:00Z",
		EndPosition:          "2022-10-10T14:50:00Z",
		FieldsLen:            13,
		TupleListMinLen:      13 * 73 * 4,
		ObservationsLen:      73,
		FirstObservationTime: "2022-10-10T02:50:00Z",
		LastObservationTime:  "2022-10-10T14:50:00Z",
	}
	weatherDataTests(t, v)
}

func TestWeatherDataHours(t *testing.T) {
	v := TestValues{
		RequestType:          Observations,
		Resolution:           Hours,
		BeginPosition:        "2022-10-01T07:00:00Z",
		EndPosition:          "2022-10-02T07:00:00Z",
		FieldsLen:            12,
		TupleListMinLen:      12 * 25 * 4,
		ObservationsLen:      25,
		FirstObservationTime: "2022-10-01T07:00:00Z",
		LastObservationTime:  "2022-10-02T07:00:00Z",
	}
	weatherDataTests(t, v)
}

func TestForecast(t *testing.T) {
	v := TestValues{
		RequestType:          Forecast,
		BeginPosition:        "2022-11-02T18:00:00Z",
		EndPosition:          "2022-11-04T19:00:00Z",
		FieldsLen:            11,
		TupleListMinLen:      11 * 25 * 4,
		ObservationsLen:      50,
		FirstObservationTime: "2022-11-02T18:00:00Z",
		LastObservationTime:  "2022-11-04T19:00:00Z",
	}
	weatherDataTests(t, v)
}

func TestInvalidXml(t *testing.T) {
	fmiObs := &FMI_ObservationsModel{}
	LoadXml(t, "testdata/exampleEmpty.xml", fmiObs, Minutes)
	//log.Printf("%+v", fc)
	LoadXml(t, "testdata/exampleInvalid.xml", fmiObs, Minutes)
	//log.Printf("%+v", fc)
}

func weatherDataTests(t *testing.T, test TestValues) {
	fmiObs := &FMI_ObservationsModel{}
	// Initialize xml
	if test.RequestType == Observations && test.Resolution == Minutes {
		LoadXml(t, "testdata/exampleMinutes.xml", fmiObs, Minutes)
	} else if test.RequestType == Observations && test.Resolution == Hours {
		LoadXml(t, "testdata/exampleHours.xml", fmiObs, Hours)
	} else if test.RequestType == Forecast {
		LoadXml(t, "testdata/exampleForecast.xml", fmiObs, Hours)
	} else {
		t.Errorf("Invalid test data: RequestType: %v, Resolution: %v", test.RequestType, test.Resolution)
	}
	//log.Printf("%+v", featureCollection)
	obs := fmiObs.Observations
	err := fmiObs.Validate()
	if err != nil {
		t.Errorf("Error validating observations model: %v", err)
	}
	if obs.BeginPosition != test.BeginPosition {
		t.Errorf("BeginPosition, got %s, want %s", obs.BeginPosition, test.BeginPosition)
	}
	if obs.EndPosition != test.EndPosition {
		t.Errorf("EndPosition, got %s, want %s", obs.EndPosition, test.EndPosition)
	}
	if len(obs.Fields) != test.FieldsLen {
		t.Errorf("Observation.Fields len, got %d, want %d", len(obs.Fields), test.FieldsLen)
	}
	if len(strings.TrimSpace(obs.Measures)) < test.TupleListMinLen {
		t.Errorf("Measures min len, got %d, want %d", len(strings.TrimSpace(obs.Measures)), test.TupleListMinLen)
	}

	// Load xml into WeatherData
	weather, err := fmiObs.ConvertToWeatherData()
	if err != nil {
		t.Errorf("ConvertToWeatherData failed: %v", err)
	}
	_, err = json.Marshal(weather)
	if err != nil {
		t.Errorf("Failed to marshal json from: %+v. Err: %v", weather, err)
	}
	// log.Print(string(json))

	if len(weather.WeatherData) != test.ObservationsLen {
		t.Errorf("len(observations), got %d, want %d", len(weather.WeatherData), test.ObservationsLen)
	}
	if weather.WeatherData[0].Time != test.FirstObservationTime {
		t.Errorf("observations[0].Time, got %s, want %s", weather.WeatherData[0].Time, test.FirstObservationTime)
	}
	if weather.WeatherData[len(weather.WeatherData)-1].Time != test.LastObservationTime {
		t.Errorf("last weather time != LastObservationTime, got %s, want %s", weather.WeatherData[len(weather.WeatherData)-1].Time, test.LastObservationTime)
	}
}
