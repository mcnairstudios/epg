package domain

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
