package core

import (
	"context"
)

type Searcher interface {
	Search(context context.Context, request SearchRequest) (*SearchReply, error)
	SearchIndex(context context.Context, request SearchRequest) (*SearchReply, error)
	UpdateIndex(context context.Context) error
}

type DB interface {
	Find(context context.Context, words []string, limit int) (*SearchReply, error)
	FindAll(context context.Context) (*IndexInfo, error)
	GetById(context context.Context, id int) (*Comics, error)
}

type Words interface {
	Norm(ctx context.Context, phrase string) ([]string, error)
}
