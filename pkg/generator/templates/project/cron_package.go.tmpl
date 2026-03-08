package cron

import (
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog"
	"github.com/samber/do/v2"
)

type Scheduler struct {
	c      *cron.Cron
	logger *zerolog.Logger
}

func NewScheduler(i do.Injector) (*Scheduler, error) {
	logger := do.MustInvoke[*zerolog.Logger](i)
	c := cron.New(cron.WithSeconds())
	s := &Scheduler{c: c, logger: logger}
	s.registerJobs()
	return s, nil
}

func (s *Scheduler) registerJobs() {
	// JOBS_PLACEHOLDER
}

func (s *Scheduler) Start() {
	s.c.Start()
	s.logger.Info().Msg("Cron scheduler started")
}

func (s *Scheduler) Stop() {
	s.c.Stop()
	s.logger.Info().Msg("Cron scheduler stopped")
}

func (s *Scheduler) Shutdown() error {
	s.Stop()
	return nil
}

var Package = do.Package(
	do.Lazy(NewScheduler),
)