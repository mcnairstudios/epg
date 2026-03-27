package watchyourtv_test

import (
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/watchyourtv"
)

func TestParseContent(t *testing.T) {
	content := `[{"name":"TVS Classic Sports","icon":"https://www.watchyour.tv/epg/channellogos/tvs-classic-sports.png","language":"English","id":"735","shows":[{"name":"1979 WVU vs Penn State","category":"Sports","start_day":"2022-10-03","start":"04:00:00","end_day":"2022-10-03","end":"06:00:45","duration":"121","url":"http://rpn1.bozztv.com/36bay2/gusa-tvs/index-1664769600-7245.m3u8?token=f7410a9414f61579dced17ac1bbdb971","icon":"https://example.com/icon.png","timezone":"+0000","tms":"1664769600"},{"name":"1958 NCAA University of Kentucky vs Seattle U","category":"Sports","start_day":"2022-10-04","start":"00:58:50","end_day":"2022-10-04","end":"01:44:11","duration":"46","url":"http://rpn1.bozztv.com/36bay2/gusa-tvs/index.m3u8?token=93e7b201f544c87296076b73f9d880ae","icon":"","timezone":"+0000","tms":"1664845130"}]}]`

	date := time.Date(2022, 10, 3, 0, 0, 0, 0, time.UTC)
	channel := domain.Channel{
		SiteID:  "735",
		XmltvID: "TVSClassicSports.us",
		Lang:    "en",
	}

	programs, err := watchyourtv.Parse([]byte(content), channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 1 {
		t.Fatalf("expected 1 program, got %d", len(programs))
	}

	p := programs[0]

	if p.Title != "1979 WVU vs Penn State" {
		t.Fatalf("expected title '1979 WVU vs Penn State', got %q", p.Title)
	}

	expectedStart := time.Date(2022, 10, 3, 4, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2022, 10, 3, 6, 1, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Image != "https://example.com/icon.png" {
		t.Fatalf("expected image 'https://example.com/icon.png', got %q", p.Image)
	}

	if p.Category != "Sports" {
		t.Fatalf("expected category 'Sports', got %q", p.Category)
	}

	if p.Channel != "TVSClassicSports.us" {
		t.Fatalf("expected channel 'TVSClassicSports.us', got %q", p.Channel)
	}

	if p.Source != "watchyour.tv" {
		t.Fatalf("expected source 'watchyour.tv', got %q", p.Source)
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := watchyourtv.Parse([]byte(""), domain.Channel{}, time.Now())
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseNoMatchingChannel(t *testing.T) {
	content := `[{"name":"Other","id":"999","shows":[]}]`
	channel := domain.Channel{SiteID: "735"}
	programs, err := watchyourtv.Parse([]byte(content), channel, time.Now())
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
