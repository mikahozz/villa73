package fmi

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type FMI_ObservationsModel struct {
	Observations ObservationCollection `xml:"FeatureCollection" validate:"required"`
}

type ObservationCollection struct {
	Resolution    Resolution `validate:"required"`
	BeginPosition string     `xml:"member>GridSeriesObservation>phenomenonTime>TimePeriod>beginPosition" validate:"required,ISO8601date"`
	EndPosition   string     `xml:"member>GridSeriesObservation>phenomenonTime>TimePeriod>endPosition" validate:"required,ISO8601date"`
	Measures      string     `xml:"member>GridSeriesObservation>result>MultiPointCoverage>rangeSet>DataBlock>doubleOrNilReasonTupleList" validate:"required"`
	Fields        []Field    `xml:"member>GridSeriesObservation>result>MultiPointCoverage>rangeType>DataRecord>field" validate:"gt=3,dive"`
}
type Field struct {
	Name string `xml:"name,attr" validate:"required"`
}
type Resolution int64

const (
	Hours Resolution = iota + 1
	Minutes
)

type RequestType int64

const (
	Observations RequestType = iota + 1
	Forecast
)

func (obs *FMI_ObservationsModel) LoadObservations(location StationId, requestType RequestType) error {
	q := ""
	switch requestType {
	case Observations:
		obs.Observations.Resolution = Minutes
		q = fmt.Sprintf("http://opendata.fmi.fi/wfs?service=WFS&version=2.0.0&request=getFeature&storedquery_id=fmi::observations::weather::multipointcoverage&fmisid=%s",
			location)
	case Forecast:
		obs.Observations.Resolution = Hours
		q = fmt.Sprintf("http://opendata.fmi.fi/wfs?service=WFS&version=2.0.0&request=getFeature&storedquery_id=fmi::forecast::harmonie::surface::point::multipointcoverage&parameters=Temperature,Humidity,WindSpeedMS,WindGust,WindDirection,precipitation1h,Pressure,DewPoint,Visibility,TotalCloudCover,SmartSymbol&place=%s",
			location)
	default:
		return errors.Errorf("Invalid requestType: %v", requestType)
	}

	resp, err := http.Get(q)
	if err != nil {
		return errors.Wrap(err, "Error fetching data from FMI")
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrapf(err, "Error reading body from FMI request: StatusCode: %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("Error fetching data from FMI: StatusCode: %d, Body: %s", resp.StatusCode, body)
	}

	err = xml.Unmarshal(body, &obs.Observations)
	if err != nil {
		return errors.Wrapf(err, "Error parsing body to FMI_ObservationsModel. Body: %v", body)
	}
	return obs.Validate()
}

func ISO8601Date(fl validator.FieldLevel) bool {
	ISO8601DateRegexString := "^(?:[1-9]\\d{3}-(?:(?:0[1-9]|1[0-2])-(?:0[1-9]|1\\d|2[0-8])|(?:0[13-9]|1[0-2])-(?:29|30)|(?:0[13578]|1[02])-31)|(?:[1-9]\\d(?:0[48]|[2468][048]|[13579][26])|(?:[2468][048]|[13579][26])00)-02-29)T(?:[01]\\d|2[0-3]):[0-5]\\d:[0-5]\\d(?:\\.\\d{1,9})?(?:Z|[+-][01]\\d:[0-5]\\d)$"
	ISO8601DateRegex := regexp.MustCompile(ISO8601DateRegexString)
	return ISO8601DateRegex.MatchString(fl.Field().String())
}

func (f FMI_ObservationsModel) Validate() error {
	validate := validator.New()
	validate.RegisterValidation("ISO8601date", ISO8601Date)
	err := validate.Struct(f)
	if err != nil {
		log.Debug().Msgf("Validation error: Content: %+v", f)
		return errors.Wrap(err, "Validation error")
	}
	return nil
}

func (fm FMI_ObservationsModel) ConvertToWeatherData() (WeatherDataModel, error) {
	wData := WeatherDataModel{}
	obs := fm.Observations
	if obs.Resolution == 0 {
		return wData, errors.New("Resolution is not set, cannot convert to WeatherData")
	}
	lines := strings.Split(
		strings.TrimSpace(
			strings.ReplaceAll(obs.Measures, "\r\n", "\n"),
		),
		"\n")
	beginDate, err := time.Parse(time.RFC3339, obs.BeginPosition)
	if err != nil {
		return wData, errors.Wrapf(err, "Failed to parse date: %s", obs.BeginPosition)
	}
	dt := beginDate
	var timeAdd time.Duration
	if obs.Resolution == Hours {
		timeAdd = time.Hour
	}
	if obs.Resolution == Minutes {
		timeAdd = time.Minute * 10
	}
	for i, line := range lines {
		w := WeatherData{}
		w.Time = dt.UTC().Format(time.RFC3339)
		values := strings.Split(strings.TrimSpace(line), " ")
		fields := obs.Fields
		if len(values) != len(fields) {
			return wData, errors.Errorf("The amount of measures doesn't match the fields: Measures len: %d, fields len: %d", len(values), len(fields))
		}
		for j, field := range fields {
			value, err := strconv.ParseFloat(values[j], 64)
			if err != nil {
				return wData, errors.Wrapf(err, "Failed to parse string measure %s from position %d from line %d: %v", values[j], j, i, err)
			}
			switch field.Name {
			case "TA_PT1H_AVG", "t2m", "Temperature":
				w.Temp = valueOrZero(value)
			case "TA_PT1H_MAX":
				w.TempMax = valueOrZero(value)
			case "TA_PT1H_MIN":
				w.TempMin = valueOrZero(value)
			case "RH_PT1H_AVG", "rh", "Humidity":
				w.Humidity = valueOrZero(value)
			case "WS_PT1H_AVG", "ws_10min", "WindSpeedMS":
				w.WindSpeed = valueOrZero(value)
			case "WS_PT1H_MAX", "wg_10min", "WindGust":
				w.MaxWindSpeed = valueOrZero(value)
			case "WS_PT1H_MIN":
				w.MinWindSpeed = valueOrZero(value)
			case "WD_PT1H_AVG", "wd_10min", "WindDirection":
				w.WindDirection = valueOrZero(value)
			case "PRA_PT1H_ACC", "r_1h", "precipitation1h":
				w.Rain = valueOrZero(value)
			case "PRI_PT1H_MAX", "ri_10min":
				w.MaxRainIntensity = valueOrZero(value)
			case "PA_PT1H_AVG", "p_sea", "Pressure":
				w.Pressure = valueOrZero(value)
			case "WAWA_PT1H_RANK", "wawa":
				w.Weather = int(valueOrZero(value))
			case "td", "DewPoint":
				w.DewPoint = valueOrZero(value)
			case "snow_aws":
				w.SnowDepth = valueOrZero(value)
			case "vis", "Visibility":
				w.Visibility = valueOrZero(value)
			case "n_man", "TotalCloudCover":
				w.CloudCover = valueOrZero(value)
			case "SmartSymbol":
				w.Weather = int(valueOrZero(value))
			}
		}
		wData.WeatherData = append(wData.WeatherData, w)
		dt = dt.Add(timeAdd)
	}
	return wData, nil
}

func valueOrZero(v float64) float64 {
	if math.IsNaN(v) {
		return 0.0
	}
	return v
}
