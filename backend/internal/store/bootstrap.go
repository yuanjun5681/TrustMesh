package store

import (
	"time"

	"go.uber.org/zap"
	"trustmesh/backend/internal/config"
)

func NewWithConfig(cfg config.Config, log *zap.Logger) (*Store, error) {
	s := New()
	s.log = log
	s.heartbeatTTL = cfg.HeartbeatTTL
	if s.heartbeatTTL <= 0 {
		s.heartbeatTTL = 30 * time.Second
	}
	s.heartbeatSweepInterval = cfg.HeartbeatSweepInterval
	if s.heartbeatSweepInterval <= 0 {
		s.heartbeatSweepInterval = 5 * time.Second
	}
	s.heartbeatStopCh = make(chan struct{})
	s.heartbeatDoneCh = make(chan struct{})
	if cfg.MongoEnabled {
		if err := s.enableMongo(cfg, log); err != nil {
			return nil, err
		}
	}
	go s.heartbeatLoop()
	return s, nil
}
