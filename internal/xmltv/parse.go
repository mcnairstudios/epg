package xmltv

import (
	"encoding/xml"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
)

var onscreenRegexp = regexp.MustCompile(`S(\d+)E(\d+)`)

func Parse(data []byte, source string) ([]domain.Channel, []domain.Program, error) {
	var tv TV
	if err := xml.Unmarshal(data, &tv); err != nil {
		return nil, nil, fmt.Errorf("failed to parse XMLTV: %w", err)
	}

	channels := make([]domain.Channel, 0, len(tv.Channels))
	for _, ch := range tv.Channels {
		name := ""
		lang := ""
		if len(ch.DisplayName) > 0 {
			name = ch.DisplayName[0].Value
			lang = ch.DisplayName[0].Lang
		}
		logo := ""
		if ch.Icon != nil {
			logo = ch.Icon.Src
		}
		channels = append(channels, domain.Channel{
			ID:      fmt.Sprintf("%s-%s", source, ch.ID),
			Name:    name,
			Source:  source,
			SiteID:  ch.ID,
			Lang:    lang,
			Logo:    logo,
			XmltvID: ch.ID,
		})
	}

	programs := make([]domain.Program, 0, len(tv.Programmes))
	for _, prog := range tv.Programmes {
		p := domain.Program{
			Channel: prog.Channel,
			Source:  source,
		}

		if len(prog.Title) > 0 {
			p.Title = prog.Title[0].Value
			p.Lang = prog.Title[0].Lang
		}
		if len(prog.SubTitle) > 0 {
			p.SubTitle = prog.SubTitle[0].Value
		}
		if len(prog.Desc) > 0 {
			p.Description = prog.Desc[0].Value
		}
		if len(prog.Category) > 0 {
			p.Category = prog.Category[0].Value
		}
		if prog.Icon != nil {
			p.Image = prog.Icon.Src
		}

		if prog.Episode != nil {
			p.Season, p.Episode = parseEpisodeNum(prog.Episode.System, prog.Episode.Value)
		}

		if t, err := parseXMLTVTime(prog.Start); err == nil {
			p.Start = t
		}
		if t, err := parseXMLTVTime(prog.Stop); err == nil {
			p.Stop = t
		}

		programs = append(programs, p)
	}

	return channels, programs, nil
}

func parseXMLTVTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	formats := []string{
		"20060102150405 -0700",
		"20060102150405 +0000",
		"20060102150405",
	}
	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse time: %s", s)
}

func parseEpisodeNum(system, value string) (int, int) {
	switch system {
	case "onscreen":
		matches := onscreenRegexp.FindStringSubmatch(value)
		if len(matches) == 3 {
			s, _ := strconv.Atoi(matches[1])
			e, _ := strconv.Atoi(matches[2])
			return s, e
		}
	case "xmltv_ns":
		parts := strings.Split(value, ".")
		if len(parts) >= 2 {
			s, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
			e, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
			return s + 1, e + 1
		}
	}
	return 0, 0
}
