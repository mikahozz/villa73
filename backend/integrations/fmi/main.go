package fmi

func GetWeatherData(id StationId, requestType RequestType) (WeatherDataModel, error) {
	fmi := &FMI_ObservationsModel{}
	err := fmi.LoadObservations(id, requestType)
	if err != nil {
		return WeatherDataModel{}, err
	}
	w, err := fmi.ConvertToWeatherData()
	if err != nil {
		return WeatherDataModel{}, err
	}
	return w, nil
}
