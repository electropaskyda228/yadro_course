package grpc

import (
	"context"

	"google.golang.org/protobuf/types/known/emptypb"
	searchpb "yadro.com/course/proto/search"
	"yadro.com/course/search/core"
)

type Server struct {
	searchpb.UnimplementedSearchServer
	service core.Searcher
}

func NewServer(service core.Searcher) *Server {
	return &Server{service: service}
}

func (s *Server) Ping(_ context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	return nil, nil
}

func (s *Server) Search(ctx context.Context, in *searchpb.ComicsRequest) (*searchpb.ComicsResponse, error) {
	reply, err := s.service.Search(ctx, core.SearchRequest{Limit: int(in.Limit), Phrase: in.Words})
	if err != nil {
		return nil, err
	}

	response := make([]*searchpb.Comics, len(reply.Comics))
	for index, comic := range reply.Comics {
		response[index] = &searchpb.Comics{Id: int64(comic.ID), Url: comic.URL}
	}
	return &searchpb.ComicsResponse{Comics: response}, nil
}

func (s *Server) SearchIndex(ctx context.Context, in *searchpb.ComicsRequest) (*searchpb.ComicsResponse, error) {
	reply, err := s.service.SearchIndex(ctx, core.SearchRequest{Limit: int(in.Limit), Phrase: in.Words})
	if err != nil {
		return nil, err
	}

	response := make([]*searchpb.Comics, len(reply.Comics))
	for index, comic := range reply.Comics {
		response[index] = &searchpb.Comics{Id: int64(comic.ID), Url: comic.URL}
	}
	return &searchpb.ComicsResponse{Comics: response}, nil
}
