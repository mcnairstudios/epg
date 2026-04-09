package xmltv

import (
	"encoding/xml"
	"strconv"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
)

type TV struct {
	XMLName    xml.Name    `xml:"tv"`
	Date       string      `xml:"date,attr,omitempty"`
	Generator  string      `xml:"generator-info-name,attr,omitempty"`
	Channels   []TVChannel `xml:"channel"`
	Programmes []Programme `xml:"programme"`
}

type TVChannel struct {
	ID          string  `xml:"id,attr"`
	DisplayName []Name  `xml:"display-name"`
	Icon        *Icon   `xml:"icon,omitempty"`
}

type Programme struct {
	Start    string `xml:"start,attr"`
	Stop     string `xml:"stop,attr"`
	Channel  string `xml:"channel,attr"`
	Title    []Name `xml:"title"`
	SubTitle []Name `xml:"sub-title,omitempty"`
	Desc     []Name `xml:"desc,omitempty"`
	Category []Name `xml:"category,omitempty"`
	Icon     *Icon  `xml:"icon,omitempty"`
	Episode  *EpisodeNum `xml:"episode-num,omitempty"`
}

type Name struct {
	Lang  string `xml:"lang,attr,omitempty"`
	Value string `xml:",chardata"`
}

type Icon struct {
	Src string `xml:"src,attr"`
}

type EpisodeNum struct {
	System string `xml:"system,attr"`
	Value  string `xml:",chardata"`
}

const timeFormat = "20060102150405 -0700"

func Generate(channels []domain.Channel, programs []domain.Program) ([]byte, error) {
	tv := TV{
		Date:      time.Now().UTC().Format("20060102"),
		Generator: "epg",
	}

	for _, ch := range channels {
		tvCh := TVChannel{
			ID:          ch.XmltvID,
			DisplayName: []Name{{Lang: ch.Lang, Value: ch.Name}},
		}
		if ch.Logo != "" {
			tvCh.Icon = &Icon{Src: ch.Logo}
		}
		tv.Channels = append(tv.Channels, tvCh)
	}

	for _, p := range programs {
		prog := Programme{
			Start:   p.Start.UTC().Format(timeFormat),
			Stop:    p.Stop.UTC().Format(timeFormat),
			Channel: p.Channel,
			Title:   []Name{{Lang: p.Lang, Value: p.Title}},
		}
		if p.SubTitle != "" {
			prog.SubTitle = []Name{{Lang: p.Lang, Value: p.SubTitle}}
		}
		if p.Description != "" {
			prog.Desc = []Name{{Lang: p.Lang, Value: p.Description}}
		}
		if p.Category != "" {
			prog.Category = []Name{{Lang: p.Lang, Value: p.Category}}
		}
		if p.Image != "" {
			prog.Icon = &Icon{Src: p.Image}
		}
		if p.Season > 0 || p.Episode > 0 {
			prog.Episode = &EpisodeNum{
				System: "xmltv_ns",
				Value:  formatEpisodeNum(p.Season, p.Episode),
			}
		}
		tv.Programmes = append(tv.Programmes, prog)
	}

	output, err := xml.MarshalIndent(tv, "", "  ")
	if err != nil {
		return nil, err
	}

	return append([]byte(xml.Header), output...), nil
}

func formatEpisodeNum(season, episode int) string {
	s := ""
	e := ""
	if season > 0 {
		s = strconv.Itoa(season - 1)
	}
	if episode > 0 {
		e = strconv.Itoa(episode - 1)
	}
	return s + "." + e + "."
}
