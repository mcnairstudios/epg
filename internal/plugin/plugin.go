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
