package hardware

import "context"

// Service exposes the host hardware inventory. It is read-only and holds no
// persistent state, so construction simply resolves an Adapter.
type Service struct {
	adapter Adapter
}

// NewService builds a Service. A nil adapter resolves to the default (Linux on
// hosts with /proc, dev stub otherwise), matching the nil-fallback convention
// used across the other domains.
func NewService(adapter Adapter) *Service {
	if adapter == nil {
		adapter = NewDefaultAdapter()
	}
	return &Service{adapter: adapter}
}

func (s *Service) Inventory(ctx context.Context) (Inventory, error) {
	return s.adapter.Inventory(ctx)
}
