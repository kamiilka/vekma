package app

import (
	"log/slog"
	"sync"
	"time"
)

const minBackgroundJobSchedulerInterval = time.Second

type BackgroundJobScheduler struct {
	store    *Store
	logger   *slog.Logger
	interval time.Duration

	mu        sync.Mutex
	started   bool
	startOnce sync.Once
	stopOnce  sync.Once
	stopCh    chan struct{}
	doneCh    chan struct{}
}

func NewBackgroundJobScheduler(store *Store, logger *slog.Logger, interval time.Duration) *BackgroundJobScheduler {
	if interval < minBackgroundJobSchedulerInterval {
		interval = minBackgroundJobSchedulerInterval
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &BackgroundJobScheduler{
		store:    store,
		logger:   logger,
		interval: interval,
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
	}
}

func (s *BackgroundJobScheduler) Start() {
	if s == nil || s.store == nil {
		return
	}
	s.startOnce.Do(func() {
		s.mu.Lock()
		s.started = true
		s.mu.Unlock()
		go s.loop()
	})
}

func (s *BackgroundJobScheduler) Stop() {
	if s == nil {
		return
	}
	s.mu.Lock()
	started := s.started
	s.mu.Unlock()
	if !started {
		return
	}
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
	<-s.doneCh
}

func (s *BackgroundJobScheduler) loop() {
	defer close(s.doneCh)

	s.runOnce("startup")

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.runOnce("tick")
		}
	}
}

func (s *BackgroundJobScheduler) runOnce(source string) {
	processed, err := s.store.RunDueBackgroundJobs()
	if err != nil {
		s.logger.Error("background jobs scheduler run failed", "source", source, "error", err)
		return
	}
	if len(processed) == 0 {
		return
	}
	s.logger.Info("background jobs scheduler processed jobs", "source", source, "count", len(processed))
}
