package domain

type Source struct {
	Name           string `json:"name"`
	Description    string `json:"description,omitempty"`
	URL            string `json:"url,omitempty"`
	Type           string `json:"type"`
	TotalChannels  int    `json:"total_channels"`
}
