package channels

import (
	"encoding/json"
	"fmt"
	"sort"

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

// GroupBy reorders channels, stably, so those sharing a key are adjacent.
// Plugins whose channels share an upstream document use it so a grab needs
// each document only for one stretch of the run.
func GroupBy(chs []domain.Channel, key func(domain.Channel) string) []domain.Channel {
	sort.SliceStable(chs, func(i, j int) bool { return key(chs[i]) < key(chs[j]) })
	return chs
}
