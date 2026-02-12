package spot

import "encoding/xml"

type AcknowledgementMarketDocument struct {
	XMLName xml.Name `xml:"Acknowledgement_MarketDocument"`
	Reason  Reason   `xml:"Reason"`
}

type Reason struct {
	Code string `xml:"code"`
	Text string `xml:"text"`
}
