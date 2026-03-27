package domain

import "time"

type Program struct {
	Channel     string    `json:"channel"`
	Source      string    `json:"source"`
	Title       string    `json:"title"`
	SubTitle    string    `json:"sub_title,omitempty"`
	Description string    `json:"description,omitempty"`
	Category    string    `json:"category,omitempty"`
	Season      int       `json:"season,omitempty"`
	Episode     int       `json:"episode,omitempty"`
	Image       string    `json:"image,omitempty"`
	Lang        string    `json:"lang,omitempty"`
	Start       time.Time `json:"start"`
	Stop        time.Time `json:"stop"`
}
