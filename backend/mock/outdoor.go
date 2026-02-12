package mock

import (
	"fmt"
	"time"
)

func OutdoorWeathernNow() (string, error) {
	timeSubstitutes := ConvertStrArrayToInterface(
		GenerateFutureDates(time.Minute*time.Duration(10), 6, true, false))

	return fmt.Sprintf(`
	[
		{
		  "datetime": "%s",
		  "temperature": 2.7,
		  "humidity": 0.0
		},
		{
		  "datetime": "%s",
		  "temperature": 2.7,
		  "humidity": 0.0
		},
		{
		  "datetime": "%s",
		  "temperature": 3.0,
		  "humidity": 0.0
		},
		{
		  "datetime": "%s",
		  "temperature": 3.0,
		  "humidity": 0.0
		},
		{
		  "datetime": "%s",
		  "temperature": 3.1,
		  "humidity": 0.0
		},
		{
		  "datetime": "%s",
		  "temperature": 3.4,
		  "humidity": 0.0
		}
	  ]
	  `, timeSubstitutes...), nil
}

func OutdoorWeatherFore() (string, error) {
	timeSubstitutes := ConvertStrArrayToInterface(
		GenerateFutureDates(time.Hour*time.Duration(1), 14, false, true))

	return fmt.Sprintf(`
	[
		{
		  "Datetime": "%s",
		  "Temperature": -5.26,
		  "Pressure": 1040.61,
		  "Humidity": 95.76,
		  "WindDirection": 92.0,
		  "WindSpeedMS": 1.17,
		  "MaximumWind": 1.3,
		  "WindGust": 1.8,
		  "DewPoint": -6.05,
		  "TotalCloudCover": 12.22,
		  "SmartSymbol": 1.0,
		  "LowCloudCover": 0.0,
		  "MediumCloudCover": 0.0,
		  "HighCloudCover": 0.0,
		  "Precipitation1h": 0.0,
		  "PrecipitationAmount": 0.0
		},
		{
		  "Datetime": "%s",
		  "Temperature": -6.1,
		  "Pressure": 1040.63,
		  "Humidity": 95.58,
		  "WindDirection": 103.0,
		  "WindSpeedMS": 1.74,
		  "MaximumWind": 1.87,
		  "WindGust": 2.81,
		  "DewPoint": -6.92,
		  "TotalCloudCover": 12.01,
		  "SmartSymbol": 1.0,
		  "LowCloudCover": 0.0,
		  "MediumCloudCover": 0.0,
		  "HighCloudCover": 0.0,
		  "Precipitation1h": 0.0,
		  "PrecipitationAmount": 0.0
		},
		{
		  "Datetime": "%s",
		  "Temperature": -7.24,
		  "Pressure": 1040.42,
		  "Humidity": 94.55,
		  "WindDirection": 106.0,
		  "WindSpeedMS": 1.61,
		  "MaximumWind": 1.87,
		  "WindGust": 2.57,
		  "DewPoint": -8.25,
		  "TotalCloudCover": 5.15,
		  "SmartSymbol": 1.0,
		  "LowCloudCover": 0.0,
		  "MediumCloudCover": 0.0,
		  "HighCloudCover": 0.0,
		  "Precipitation1h": 0.0,
		  "PrecipitationAmount": 0.0
		},
		{
		  "Datetime": "%s",
		  "Temperature": -7.22,
		  "Pressure": 1040.32,
		  "Humidity": 92.8,
		  "WindDirection": 157.0,
		  "WindSpeedMS": 2.24,
		  "MaximumWind": 2.33,
		  "WindGust": 3.51,
		  "DewPoint": -8.57,
		  "TotalCloudCover": 0.0,
		  "SmartSymbol": 1.0,
		  "LowCloudCover": 0.0,
		  "MediumCloudCover": 0.0,
		  "HighCloudCover": 0.0,
		  "Precipitation1h": 0.0,
		  "PrecipitationAmount": 0.0
		},
		{
		  "Datetime": "%s",
		  "Temperature": -7.66,
		  "Pressure": 1040.44,
		  "Humidity": 91.63,
		  "WindDirection": 176.0,
		  "WindSpeedMS": 2.28,
		  "MaximumWind": 2.42,
		  "WindGust": 3.42,
		  "DewPoint": -9.23,
		  "TotalCloudCover": 0.0,
		  "SmartSymbol": 1.0,
		  "LowCloudCover": 0.0,
		  "MediumCloudCover": 0.0,
		  "HighCloudCover": 0.0,
		  "Precipitation1h": 0.0,
		  "PrecipitationAmount": 0.0
		},
		{
		  "Datetime": "%s",
		  "Temperature": -8.08,
		  "Pressure": 1040.29,
		  "Humidity": 91.37,
		  "WindDirection": 164.0,
		  "WindSpeedMS": 2.43,
		  "MaximumWind": 2.53,
		  "WindGust": 3.73,
		  "DewPoint": -9.7,
		  "TotalCloudCover": 0.0,
		  "SmartSymbol": 1.0,
		  "LowCloudCover": 0.0,
		  "MediumCloudCover": 0.0,
		  "HighCloudCover": 0.0,
		  "Precipitation1h": 0.0,
		  "PrecipitationAmount": 0.0
		},
		{
		  "Datetime": "%s",
		  "Temperature": -8.26,
		  "Pressure": 1039.99,
		  "Humidity": 90.84,
		  "WindDirection": 161.0,
		  "WindSpeedMS": 2.72,
		  "MaximumWind": 2.81,
		  "WindGust": 4.22,
		  "DewPoint": -9.98,
		  "TotalCloudCover": 0.0,
		  "SmartSymbol": 1.0,
		  "LowCloudCover": 0.0,
		  "MediumCloudCover": 0.0,
		  "HighCloudCover": 0.0,
		  "Precipitation1h": 0.0,
		  "PrecipitationAmount": 0.0
		},
		{
		  "Datetime": "%s",
		  "Temperature": -8.2,
		  "Pressure": 1039.59,
		  "Humidity": 89.49,
		  "WindDirection": 160.0,
		  "WindSpeedMS": 3.03,
		  "MaximumWind": 3.14,
		  "WindGust": 4.76,
		  "DewPoint": -10.18,
		  "TotalCloudCover": 0.0,
		  "SmartSymbol": 1.0,
		  "LowCloudCover": 0.0,
		  "MediumCloudCover": 0.0,
		  "HighCloudCover": 0.0,
		  "Precipitation1h": 0.0,
		  "PrecipitationAmount": 0.0
		},
		{
		  "Datetime": "%s",
		  "Temperature": -7.31,
		  "Pressure": 1039.14,
		  "Humidity": 87.56,
		  "WindDirection": 158.0,
		  "WindSpeedMS": 3.94,
		  "MaximumWind": 4.04,
		  "WindGust": 6.33,
		  "DewPoint": -9.7,
		  "TotalCloudCover": 0.0,
		  "SmartSymbol": 1.0,
		  "LowCloudCover": 0.0,
		  "MediumCloudCover": 0.0,
		  "HighCloudCover": 0.0,
		  "Precipitation1h": 0.0,
		  "PrecipitationAmount": 0.0
		},
		{
		  "Datetime": "%s",
		  "Temperature": -6.72,
		  "Pressure": 1038.75,
		  "Humidity": 85.77,
		  "WindDirection": 161.0,
		  "WindSpeedMS": 4.13,
		  "MaximumWind": 4.23,
		  "WindGust": 6.7,
		  "DewPoint": -9.48,
		  "TotalCloudCover": 0.0,
		  "SmartSymbol": 1.0,
		  "LowCloudCover": 0.0,
		  "MediumCloudCover": 0.0,
		  "HighCloudCover": 0.0,
		  "Precipitation1h": 0.0,
		  "PrecipitationAmount": 0.0
		},
		{
		  "Datetime": "%s",
		  "Temperature": -6.4,
		  "Pressure": 1038.33,
		  "Humidity": 85.24,
		  "WindDirection": 166.0,
		  "WindSpeedMS": 4.01,
		  "MaximumWind": 4.32,
		  "WindGust": 6.58,
		  "DewPoint": -9.28,
		  "TotalCloudCover": 0.0,
		  "SmartSymbol": 1.0,
		  "LowCloudCover": 0.0,
		  "MediumCloudCover": 0.0,
		  "HighCloudCover": 0.0,
		  "Precipitation1h": 0.0,
		  "PrecipitationAmount": 0.0
		},
		{
		  "Datetime": "%s",
		  "Temperature": -4.95,
		  "Pressure": 1038.01,
		  "Humidity": 82.7,
		  "WindDirection": 162.0,
		  "WindSpeedMS": 4.39,
		  "MaximumWind": 4.53,
		  "WindGust": 7.36,
		  "DewPoint": -8.41,
		  "TotalCloudCover": 0.0,
		  "SmartSymbol": 1.0,
		  "LowCloudCover": 0.0,
		  "MediumCloudCover": 0.0,
		  "HighCloudCover": 0.0,
		  "Precipitation1h": 0.0,
		  "PrecipitationAmount": 0.0
		},
		{
		  "Datetime": "%s",
		  "Temperature": -2.78,
		  "Pressure": 1038.04,
		  "Humidity": 78.6,
		  "WindDirection": 177.0,
		  "WindSpeedMS": 4.48,
		  "MaximumWind": 4.67,
		  "WindGust": 7.71,
		  "DewPoint": -7.22,
		  "TotalCloudCover": 0.0,
		  "SmartSymbol": 1.0,
		  "LowCloudCover": 0.0,
		  "MediumCloudCover": 0.0,
		  "HighCloudCover": 0.0,
		  "Precipitation1h": 0.0,
		  "PrecipitationAmount": 0.0
		},
		{
		  "Datetime": "%s",
		  "Temperature": -0.79,
		  "Pressure": 1037.97,
		  "Humidity": 73.77,
		  "WindDirection": 190.0,
		  "WindSpeedMS": 4.49,
		  "MaximumWind": 4.72,
		  "WindGust": 7.89,
		  "DewPoint": -6.46,
		  "TotalCloudCover": 0.0,
		  "SmartSymbol": 1.0,
		  "LowCloudCover": 0.0,
		  "MediumCloudCover": 0.0,
		  "HighCloudCover": 0.0,
		  "Precipitation1h": 0.0,
		  "PrecipitationAmount": 0.0
		}
	  ]	
`, timeSubstitutes...), nil
}
