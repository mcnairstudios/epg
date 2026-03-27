package telebilbao_test

import (
	"os"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/telebilbao"
)

func TestDebugTelebilbao(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		XmltvID: "TeleBilbao.es",
		SiteID:  "#",
		Lang:    "es",
	}

	date := time.Date(2025, 1, 16, 0, 0, 0, 0, time.UTC)
	programs, err := telebilbao.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	t.Logf("total programs: %d", len(programs))
	for i, p := range programs {
		t.Logf("program[%d]: start=%v title=%q", i, p.Start, p.Title)
	}
}
