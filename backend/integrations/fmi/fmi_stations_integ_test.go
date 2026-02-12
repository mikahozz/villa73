//go:build integration

package fmi

import "testing"

func TestGetStations(t *testing.T) {
	fmis := FMI_StationsModel{}
	err := fmis.LoadWeatherStations()
	if err != nil {
		t.Fatalf("LoadWeatherStations failed: %v", err)
	}
	_, err = fmis.ConvertToWeatherStations()
	if err != nil {
		t.Errorf("ConvertToWeatherStations failed: %v", err)
	}

}
