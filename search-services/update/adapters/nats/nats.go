package nats

import (
	"context"
	"log/slog"

	"github.com/nats-io/nats.go"
)

type NatsPublisher struct {
	log *slog.Logger
	nc  *nats.Conn
}

func NewNatsPublisher(log *slog.Logger, brokerAddress string) (*NatsPublisher, error) {
	nc, err := nats.Connect(brokerAddress)
	if err != nil {
		return &NatsPublisher{}, err
	}
	log.Info("connected to broker")

	return &NatsPublisher{
		log: log,
		nc:  nc,
	}, nil
}

func (np *NatsPublisher) Close() {
	np.nc.Close()
}

func (np *NatsPublisher) SendDBChangedEvent(ctx context.Context) error {
	err := np.nc.Publish("xkcd.db.updated", []byte("XKCD DB has been updated"))
	if err != nil {
		np.log.Error("could not publish message", "error", err)
		return err
	}
	np.log.Info("DB changed event has been sent")
	if err := np.nc.Flush(); err != nil {
		np.log.Error("Error flushing nuts", "error", err)
	}
	return nil
}
