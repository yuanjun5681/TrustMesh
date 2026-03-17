package clawsynapse

import (
	"context"
	"time"

	"go.uber.org/zap"
	"trustmesh/backend/internal/store"
)

type PeerSyncer struct {
	client   *Client
	store    *store.Store
	interval time.Duration
	log      *zap.Logger
	stopCh   chan struct{}
	doneCh   chan struct{}
}

func NewPeerSyncer(client *Client, st *store.Store, interval time.Duration, log *zap.Logger) *PeerSyncer {
	if client == nil || st == nil || interval <= 0 {
		return nil
	}
	return &PeerSyncer{
		client:   client,
		store:    st,
		interval: interval,
		log:      log,
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
	}
}

func (s *PeerSyncer) Start() {
	if s == nil {
		return
	}

	go func() {
		defer close(s.doneCh)
		s.sync()

		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.sync()
			case <-s.stopCh:
				return
			}
		}
	}()
}

func (s *PeerSyncer) Close() {
	if s == nil {
		return
	}
	close(s.stopCh)
	<-s.doneCh
}

func (s *PeerSyncer) sync() {
	ctx, cancel := context.WithTimeout(context.Background(), s.interval)
	defer cancel()

	peers, err := s.client.GetPeers(ctx)
	if err != nil {
		if s.log != nil {
			s.log.Warn("clawsynapse peer sync failed", zap.Error(err))
		}
		return
	}

	items := make([]store.AgentPresence, 0, len(peers))
	for _, peer := range peers {
		lastSeen := time.UnixMilli(peer.LastSeenMs).UTC()
		if peer.LastSeenMs <= 0 {
			lastSeen = time.Now().UTC()
		}
		items = append(items, store.AgentPresence{
			NodeID:     peer.NodeID,
			LastSeenAt: lastSeen,
		})
	}

	updated := s.store.SyncAgentPresence(items, time.Now().UTC())
	if s.log != nil && updated > 0 {
		s.log.Debug("clawsynapse peer sync updated agents", zap.Int("count", updated))
	}
}
