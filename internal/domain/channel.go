package domain

import "strings"

type Channel struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Source  string `json:"source"`
	SiteID  string `json:"site_id"`
	Lang    string `json:"lang"`
	Logo    string `json:"logo,omitempty"`
	Genre   string `json:"genre,omitempty"`
	Quality string `json:"quality,omitempty"`
	Region  string `json:"region,omitempty"`
	XmltvID string `json:"xmltv_id"`
}

var radioKeywords = []string{
	"radio", " fm", " am", "bbc r1", "bbc r2", "bbc r3", "bbc r4", "bbc r5",
	"bbc 6 music", "bbc asian net", "bbc world sv", "bbc local radio",
}

func (c Channel) IsRadio() bool {
	lower := strings.ToLower(c.Name)
	for _, kw := range radioKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	lower = strings.ToLower(c.XmltvID)
	return strings.Contains(lower, "radio") || strings.Contains(lower, ".fm")
}
