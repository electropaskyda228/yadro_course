package update

import (
	"context"
	"errors"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"yadro.com/course/api/core"
	updatepb "yadro.com/course/proto/update"
)

type Client struct {
	log    *slog.Logger
	client updatepb.UpdateClient
}

func NewClient(address string, log *slog.Logger) (*Client, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &Client{
		client: updatepb.NewUpdateClient(conn),
		log:    log,
	}, nil
}

func (c Client) Ping(ctx context.Context) error {
	_, err := c.client.Ping(ctx, nil)
	return err
}

func (c Client) Status(ctx context.Context) (core.UpdateStatus, error) {
	result, err := c.client.Status(ctx, nil)
	if err != nil {
		return core.StatusUpdateUnknown, err
	}
	switch result.Status {
	case updatepb.Status_STATUS_UNSPECIFIED:
		return core.StatusUpdateUnknown, nil
	case updatepb.Status_STATUS_IDLE:
		return core.StatusUpdateIdle, nil
	case updatepb.Status_STATUS_RUNNING:
		return core.StatusUpdateRunning, nil
	}
	return core.StatusUpdateUnknown, nil
}

func (c Client) Stats(ctx context.Context) (core.UpdateStats, error) {
	result, err := c.client.Stats(ctx, nil)
	if err != nil {
		return core.UpdateStats{}, err
	}
	return core.UpdateStats{
		WordsTotal:    int(result.WordsTotal),
		WordsUnique:   int(result.WordsUnique),
		ComicsFetched: int(result.ComicsFetched),
		ComicsTotal:   int(result.ComicsTotal),
	}, nil
}

func (c Client) Update(ctx context.Context) error {
	_, err := c.client.Update(ctx, nil)
	if err != nil {
		if st, ok := status.FromError(err); ok {
			return errors.New(st.Message())
		}
	}
	return err
}

func (c Client) Drop(ctx context.Context) error {
	_, err := c.client.Drop(ctx, nil)
	return err
}
