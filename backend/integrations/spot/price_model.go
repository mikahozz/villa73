package spot

import (
	"time"
)

type SpotPrice struct {
	DateTime  time.Time `json:"DateTime"`
	PriceCkwh float64   `json:"Price"`
}

type SpotPriceList struct {
	Prices []SpotPrice
}
