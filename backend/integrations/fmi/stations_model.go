package fmi

import (
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
)

type WeatherStationModel struct {
	WeatherStations []WeatherStation `validate:"required,dive"`
}
type WeatherStation struct {
	Id     string `validate:"required"`
	Region string
	Name   string `validate:"required"`
}

func (ws WeatherStationModel) Validate() error {
	validate := validator.New()
	err := validate.Struct(ws)
	if err != nil {
		return errors.Wrap(err, "Validation error")
	}
	return nil
}
