package spot

import (
	"encoding/xml"
)

type PublicationMarketDocument struct {
	XMLName    xml.Name     `xml:"Publication_MarketDocument"`
	TimeSeries []TimeSeries `xml:"TimeSeries"`
}

type TimeSeries struct {
	MRID         string `xml:"mRID"`
	Period       Period `xml:"Period"`
	InDomain     string `xml:"in_Domain>mRID"`
	OutDomain    string `xml:"out_Domain>mRID"`
	CurrencyUnit string `xml:"currency_Unit.name"`
	PriceUnit    string `xml:"price_Measure_Unit.name"`
}

type Period struct {
	TimeInterval Interval `xml:"timeInterval"`
	Resolution   string   `xml:"resolution"`
	Points       []Point  `xml:"Point"`
}

type Interval struct {
	Start string `xml:"start"`
	End   string `xml:"end"`
}

type Point struct {
	Position int     `xml:"position"`
	Price    float64 `xml:"price.amount"`
}
