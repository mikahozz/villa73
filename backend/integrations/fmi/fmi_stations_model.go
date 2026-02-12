package fmi

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
)

type FMI_StationsModel struct {
	StationsCol StationCollection `xml:"FeatureCollection" validate:"required"`
}

type StationCollection struct {
	Stations []Station `xml:"member>EnvironmentalMonitoringFacility" validate:"required,dive"` // Weather stations
}
type Station struct {
	Id    StationId `xml:"identifier" validate:"required"`
	Names []Name    `xml:"name" validate:"gt=1,dive"`
	Point string    `xml:"representativePoint>Point>pos" validate:"required"`
}
type StationId string
type Name struct {
	Key   string `xml:"codeSpace,attr"`
	Value string `xml:",chardata"`
}

func (f FMI_StationsModel) Validate() error {
	validate := validator.New()
	err := validate.Struct(f)
	if err != nil {
		return errors.Wrap(err, "Validation error")
	}
	return nil
}

func (fmis *FMI_StationsModel) LoadWeatherStations() error {
	q := fmt.Sprintf("https://opendata.fmi.fi/wfs/fin?request=getFeature&storedquery_id=fmi::ef::stations")
	resp, err := http.Get(q)
	if err != nil {
		return errors.Wrap(err, "Error fetching stations from FMI")
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrapf(err, "Error reading body from FMI stations request: StatusCode: %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("Error fetching data from FMI stations: StatusCode: %d, Body: %s", resp.StatusCode, body)
	}

	err = xml.Unmarshal(body, &fmis.StationsCol)
	if err != nil {
		return errors.Wrapf(err, "Error parsing body to FMI_StationsModel. Body: %v", body)
	}
	return fmis.Validate()
}

func (s *FMI_StationsModel) ConvertToWeatherStations() (WeatherStationModel, error) {
	wsm := WeatherStationModel{}
	for _, station := range s.StationsCol.Stations {
		weatherStation := WeatherStation{
			Id: string(station.Id),
		}
		for _, name := range station.Names {
			switch name.Key {
			case "http://xml.fmi.fi/namespace/locationcode/name":
				weatherStation.Name = name.Value
			case "http://xml.fmi.fi/namespace/location/region":
				weatherStation.Region = name.Value
			}
		}
		wsm.WeatherStations = append(wsm.WeatherStations, weatherStation)
	}
	return wsm, wsm.Validate()
}
