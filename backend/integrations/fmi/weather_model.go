package fmi

type WeatherDataModel struct {
	WeatherData []WeatherData
}
type WeatherData struct {
	Time             string  `json:"datetime"`
	Temp             float64 `json:"temperature"`
	TempMax          float64 `json:"temp_max"`
	TempMin          float64 `json:"temp_min"`
	Humidity         float64 `json:"humidity"`
	WindSpeed        float64 `json:"wind_speed"`
	MaxWindSpeed     float64 `json:"max_wind"`
	MinWindSpeed     float64 `json:"min_wind"`
	WindDirection    float64 `json:"wind_dir"`
	Rain             float64 `json:"rain,omitempty"`
	MaxRainIntensity float64 `json:"max_rain"`
	Pressure         float64 `json:"pressure"`
	Weather          int     `json:"weather"`
	DewPoint         float64 `json:"dew"`
	SnowDepth        float64 `json:"snow"`
	Visibility       float64 `json:"visibility"`
	CloudCover       float64 `json:"clouds"`
}
