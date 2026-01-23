package grpc

import (
	"context"

	"google.golang.org/protobuf/types/known/emptypb"
	updatepb "yadro.com/course/proto/update"
	"yadro.com/course/update/core"
)

func NewServer(service core.Updater) *Server {
	return &Server{service: service}
}

type Server struct {
	updatepb.UnimplementedUpdateServer
	service core.Updater
}

func (s *Server) Ping(_ context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	return nil, nil
}

func (s *Server) Status(ctx context.Context, _ *emptypb.Empty) (*updatepb.StatusReply, error) {
	status := s.service.Status(ctx)
	var result updatepb.Status
	switch status {
	case core.StatusIdle:
		result = updatepb.Status_STATUS_IDLE
	case core.StatusRunning:
		result = updatepb.Status_STATUS_RUNNING
	default:
		result = updatepb.Status_STATUS_UNSPECIFIED
	}
	return &updatepb.StatusReply{Status: result}, nil
}

func (s *Server) Update(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	if err := s.service.Update(ctx); err != nil {
		return nil, err
	}
	return nil, nil
}

func (s *Server) Stats(ctx context.Context, _ *emptypb.Empty) (*updatepb.StatsReply, error) {
	serviceStats, err := s.service.Stats(ctx)
	if err != nil {
		return nil, err
	}
	return &updatepb.StatsReply{
		WordsTotal:    int64(serviceStats.WordsTotal),
		WordsUnique:   int64(serviceStats.WordsUnique),
		ComicsTotal:   int64(serviceStats.ComicsTotal),
		ComicsFetched: int64(serviceStats.ComicsFetched),
	}, nil
}

func (s *Server) Drop(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	err := s.service.Drop(ctx)
	if err != nil {
		return nil, err
	}
	return nil, nil
}
