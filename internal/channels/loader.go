package channels

import (
	"encoding/json"
	"fmt"

	"github.com/gavinmcnair/epg/internal/domain"
)

type Entry struct {
	SiteID  string `json:"site_id"`
	Name    string `json:"name"`
	Lang    string `json:"lang"`
	Region  string `json:"region,omitempty"`
	XmltvID string `json:"xmltv_id,omitempty"`
	Logo    string `json:"logo,omitempty"`
}

func Load(data []byte, source string) ([]domain.Channel, error) {
	var entries []Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("invalid channels data: %w", err)
	}

	channels := make([]domain.Channel, 0, len(entries))
	for _, e := range entries {
		channels = append(channels, domain.Channel{
			ID:      source + "-" + e.SiteID,
			Name:    e.Name,
			Source:  source,
			SiteID:  e.SiteID,
			Lang:    e.Lang,
			Logo:    e.Logo,
			Region:  e.Region,
			XmltvID: e.XmltvID,
		})
	}

	return channels, nil
}
