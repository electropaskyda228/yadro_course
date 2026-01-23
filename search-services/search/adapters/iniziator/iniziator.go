package iniziator

import (
	"context"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
	"yadro.com/course/search/core"
)

type Iniziator struct {
	log      *slog.Logger
	searcher core.Searcher
	stopCh   chan struct{}
	ttl      time.Duration
	nc       *nats.Conn
}

func New(log *slog.Logger, searcher core.Searcher, ttl time.Duration, brokerAddress string) (*Iniziator, error) {
	nc, err := nats.Connect(brokerAddress)
	if err != nil {
		return &Iniziator{}, err
	}
	return &Iniziator{
		log:      log,
		searcher: searcher,
		stopCh:   make(chan struct{}),
		ttl:      ttl,
		nc:       nc,
	}, nil
}

func (i *Iniziator) Start(ctx context.Context) {
	sub, err := i.nc.Subscribe("xkcd.db.updated", func(msg *nats.Msg) {
		i.log.Info("received message", "data", msg.Data)
		if err := i.searcher.UpdateIndex(ctx); err != nil {
			i.log.Error("failed to rebuild index", "error", err)
		}
	})

	if err != nil {
		i.log.Error("failed to subscribe db change event publisher", "error", err)
		panic(err)
	}

	for {
		select {
		case <-i.stopCh:
			i.log.Info("stopping index initiator due to Close call")
			if err := sub.Unsubscribe(); err != nil {
				i.log.Error("failed to unsubscribe db change event publisher")
			}
			return
		case <-ctx.Done():
			i.log.Info("stopping index initiator due to context cancellation")
			if err := sub.Unsubscribe(); err != nil {
				i.log.Error("failed to unsubscribe db change event publisher")
			}
			return
		}
	}
}

func (i *Iniziator) Stop() {
	close(i.stopCh)
}
