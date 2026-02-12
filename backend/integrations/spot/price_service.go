package spot

import (
	"encoding/xml"
	"fmt"
	"time"
)

type SpotService struct {
	client      HTTPClient
	apiEndpoint string
}

func NewSpotService(client HTTPClient, apiEndpoint string) *SpotService {
	return &SpotService{
		client:      client,
		apiEndpoint: apiEndpoint,
	}
}

func (s *SpotService) GetSpotPrices(periodStart, periodEnd time.Time) (*SpotPriceList, error) {
	document, err := s.getDocument(periodStart, periodEnd)
	if err != nil {
		return nil, err
	}

	return ConvertToSpotPriceList(document, periodStart, periodEnd, periodStart.Location())
}

func (s *SpotService) getDocument(periodStart, periodEnd time.Time) (*PublicationMarketDocument, error) {
	body, err := s.client.Get(s.apiEndpoint, periodStart, periodEnd)
	if err != nil {
		return nil, err
	}

	var document PublicationMarketDocument
	err = xml.Unmarshal(body, &document)
	if err != nil {
		// If unmarshalling fails, try to parse as AcknowledgementMarketDocument
		var ack AcknowledgementMarketDocument
		ackErr := xml.Unmarshal(body, &ack)
		if ackErr == nil && ack.Reason.Code == "999" {
			return nil, &NoDataError{Message: ack.Reason.Text}
		}
		return nil, fmt.Errorf("error unmarshalling API response: %w", err)
	}

	return &document, nil
}
