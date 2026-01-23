package core

import "context"

type Normalizer interface {
	Norm(context.Context, string) ([]string, error)
}

type Pinger interface {
	Ping(context.Context) error
}

type Updater interface {
	Update(context.Context) error
	Stats(context.Context) (UpdateStats, error)
	Status(context.Context) (UpdateStatus, error)
	Drop(context.Context) error
}

type Searcher interface {
	Search(context.Context, string, int) ([]Comics, error)
	SearchIndex(context.Context, string, int) ([]Comics, error)
}

type Loginer interface {
	Login(name, password string) (string, error)
}

type ConcurrencyLimiter interface {
	Start() error
	Stop()
	Wait()
	Submit(f func()) string
}

type RateLimiter interface {
	Wait(ctx context.Context) error
	Submit(f func()) string
	Stop()
	Start() error
}
