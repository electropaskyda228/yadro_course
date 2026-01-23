package search

import (
	"context"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"yadro.com/course/api/core"
	searchpb "yadro.com/course/proto/search"
)

type Client struct {
	log        *slog.Logger
	client     searchpb.SearchClient
	connection *grpc.ClientConn
}

func NewClient(address string, log *slog.Logger) (*Client, error) {
	conn, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}
	return &Client{
		client:     searchpb.NewSearchClient(conn),
		log:        log,
		connection: conn,
	}, nil
}

func (c Client) Ping(ctx context.Context) error {
	_, err := c.client.Ping(ctx, nil)
	return err
}

func (c Client) Close() error {
	return c.connection.Close()
}

func (c Client) Search(ctx context.Context, phrase string, limit int) ([]core.Comics, error) {
	return c.searchCommon(ctx, phrase, limit, false)
}

func (c Client) SearchIndex(ctx context.Context, phrase string, limit int) ([]core.Comics, error) {
	return c.searchCommon(ctx, phrase, limit, true)
}

func (c Client) searchCommon(ctx context.Context, phrase string, limit int, withIndex bool) ([]core.Comics, error) {
	c.log.Info("Send request to search server")
	var answer *searchpb.ComicsResponse
	var err error
	if withIndex {
		answer, err = c.client.SearchIndex(ctx, &searchpb.ComicsRequest{Limit: int64(limit), Words: phrase})
	} else {
		answer, err = c.client.Search(ctx, &searchpb.ComicsRequest{Limit: int64(limit), Words: phrase})
	}
	if err != nil {
		c.log.Error("Failed to get response from search server", "error", err)
		return nil, err
	}
	result := make([]core.Comics, len(answer.Comics))
	for index, comic := range answer.Comics {
		result[index] = core.Comics{ID: int(comic.Id), URL: comic.Url}
	}
	c.log.Info("Response from search server has been recieved")
	return result, nil
}
