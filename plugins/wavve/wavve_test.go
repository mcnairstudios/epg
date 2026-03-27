package wavve_test

import (
	"testing"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/wavve"
)

func TestParseContent(t *testing.T) {
	content := []byte(`{"pagecount":"37","count":"37","list":[{"cpid":"C3","channelid":"K01","channelname":"KBS 1TV","channelimage":"img.pooq.co.kr/BMS/Channelimage30/image/KBS-1TV-1.jpg","scheduleid":"K01_20220416223000","programid":"","title":"특파원 보고  세계는 지금","image":"wchimg.wavve.com/live/thumbnail/K01.jpg","starttime":"2022-04-16 22:30","endtime":"2022-04-16 23:15","timemachine":"Y","license":"y","livemarks":[],"targetage":"0","tvimage":"img.pooq.co.kr/BMS/Channelimage30/image/KBS 1TV-2.png","ispreorder":"n","preorderlink":"n","alarm":"n"}]}`)

	channel := domain.Channel{
		ID:      "wavve-k01",
		XmltvID: "KBS1TV.kr",
		SiteID:  "K01",
		Lang:    "ko",
	}

	programs, err := wavve.Parse(content, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 1 {
		t.Fatalf("expected 1 program, got %d", len(programs))
	}

	p := programs[0]
	if p.Title != "특파원 보고  세계는 지금" {
		t.Fatalf("unexpected title: %q", p.Title)
	}
	if p.Start.Format("2006-01-02T15:04:05.000Z") != "2022-04-16T13:30:00.000Z" {
		t.Fatalf("unexpected start: %v", p.Start)
	}
	if p.Stop.Format("2006-01-02T15:04:05.000Z") != "2022-04-16T14:15:00.000Z" {
		t.Fatalf("unexpected stop: %v", p.Stop)
	}
}

func TestParseEmptyContent(t *testing.T) {
	content := []byte(`{"pagecount":"0","count":"0","list":[]}`)
	channel := domain.Channel{SiteID: "K01", Lang: "ko"}
	programs, err := wavve.Parse(content, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	channel := domain.Channel{SiteID: "K01", Lang: "ko"}
	programs, err := wavve.Parse([]byte(`not json`), channel)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
