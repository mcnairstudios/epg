package plugin

import (
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
)

type Provider interface {
	Name() string
	Description() string
	Channels() ([]domain.Channel, error)
	Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error)
}

// TimezoneProvider is an optional interface plugins may implement to declare
// the timezone of their source. The rolling grabber uses this to determine
// when to refresh (midnight in the source's local time). Plugins that do not
// implement this are treated as UTC.
type TimezoneProvider interface {
	Timezone() string
}

// Timezone returns the IANA timezone for p, or "UTC" if p does not implement
// TimezoneProvider.
func Timezone(p Provider) string {
	if tp, ok := p.(TimezoneProvider); ok {
		return tp.Timezone()
	}
	return "UTC"
}
