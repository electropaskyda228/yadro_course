package main

import (
	"context"
	"flag"
	"log"
	"net"

	"github.com/ilyakaznacheev/cleanenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	wordspb "yadro.com/course/proto/words"
	"yadro.com/course/words/words"
)

type Config struct {
	Address string `yaml:"words_address" env:"WORDS_ADDRESS" env-default:"80"`
}

func LoadConfig() *Config {
	var configPath string
	flag.StringVar(&configPath, "config", "config.yaml", "server configuration file")
	flag.Parse()

	var cfg Config

	if configPath != "" {
		err := cleanenv.ReadConfig(configPath, &cfg)
		if err != nil {
			log.Fatalf("Error reading config file: %v", err)
		}
	} else {
		err := cleanenv.ReadEnv(&cfg)
		if err != nil {
			log.Fatalf("Error reading environment variables: %v", err)
		}
	}

	return &cfg
}

type server struct {
	wordspb.UnimplementedWordsServer
}

func (s *server) Ping(_ context.Context, in *emptypb.Empty) (*emptypb.Empty, error) {
	return nil, nil
}

func (s *server) Norm(_ context.Context, in *wordspb.WordsRequest) (*wordspb.WordsReply, error) {
	if len(in.Phrase) > 4096 {
		return nil, status.Errorf(
			codes.ResourceExhausted,
			"message size exceeds 4 KiB limit: got %d bytes",
			len(in.Phrase),
		)
	}

	return &wordspb.WordsReply{
		Words: words.Norm(in.Phrase),
	}, nil
}

func main() {
	cfg := LoadConfig()

	listener, err := net.Listen("tcp", cfg.Address)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	wordspb.RegisterWordsServer(s, &server{})
	reflection.Register(s)

	if err := s.Serve(listener); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
