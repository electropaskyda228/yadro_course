package words

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"yadro.com/course/api/core"
	wordspb "yadro.com/course/proto/words"
)

type Client struct {
	log        *slog.Logger
	client     wordspb.WordsClient
	connection *grpc.ClientConn
}

func NewClient(address string, log *slog.Logger) (*Client, error) {
	conn, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithConnectParams(grpc.ConnectParams{
			Backoff: backoff.Config{
				BaseDelay:  1 * time.Second,
				Multiplier: 1.5,
				MaxDelay:   20 * time.Second,
			},
			MinConnectTimeout: 20 * time.Second,
		}),
	)
	if err != nil {
		return nil, err
	}
	return &Client{
		client:     wordspb.NewWordsClient(conn),
		log:        log,
		connection: conn,
	}, nil
}

func (c Client) Norm(ctx context.Context, phrase string) ([]string, error) {
	answer, err := c.client.Norm(ctx, &wordspb.WordsRequest{Phrase: phrase})
	if err != nil {
		if status.Code(err) == codes.ResourceExhausted {
			return nil, core.ErrBadArguments
		}
		return nil, err
	}
	return answer.GetWords(), nil
}

func (c Client) Ping(ctx context.Context) error {
	_, err := c.client.Ping(ctx, nil)
	return err
}

func (c *Client) Close() error {
	return c.connection.Close()
}
