package core

import (
	"context"
	"log/slog"
	"sort"
)

type Service struct {
	log   *slog.Logger
	db    DB
	words Words
	index map[string]map[int]bool // index: word -> id
}

func (s *Service) UpdateIndex(ctx context.Context) error {
	comics, err := s.db.FindAll(ctx)
	if err != nil {
		s.log.Error("Failed to update index", "error", err)
		return err
	}

	for _, comics := range comics.Comics {
		if _, ok := s.index[comics.Word]; !ok {
			s.index[comics.Word] = make(map[int]bool)
		}
		tmp := s.index[comics.Word]

		for _, index := range comics.Comics_ids {
			tmp[index] = true
		}

		s.index[comics.Word] = tmp
	}
	return nil
}

func NewService(
	log *slog.Logger, db DB, words Words,
) (*Service, error) {
	return &Service{
		log:   log,
		db:    db,
		words: words,
		index: make(map[string]map[int]bool),
	}, nil
}

func (s *Service) Search(ctx context.Context, request SearchRequest) (*SearchReply, error) {
	words, err := s.words.Norm(ctx, request.Phrase)
	if err != nil {
		return &SearchReply{}, err
	}

	reply, err := s.db.Find(ctx, words, request.Limit)
	if err != nil {
		return &SearchReply{}, err
	}
	return reply, nil
}

func (s *Service) SearchIndex(ctx context.Context, request SearchRequest) (*SearchReply, error) {
	// Normilize
	words, err := s.words.Norm(ctx, request.Phrase)
	if err != nil {
		return &SearchReply{}, err
	}

	// Find relevant comics id
	comicsMatches := make(map[int]int)
	for _, word := range words {
		if comics, ok := s.index[word]; ok {
			for comicId := range comics {
				comicsMatches[comicId]++
			}
		}
	}

	if len(comicsMatches) == 0 {
		return &SearchReply{}, nil
	}

	type ComicScore struct {
		ID    int
		Score int
	}

	var scoredComics []ComicScore
	for comicID, score := range comicsMatches {
		scoredComics = append(scoredComics, ComicScore{ID: comicID, Score: score})
	}

	sort.Slice(scoredComics, func(i, j int) bool {
		if scoredComics[i].Score == scoredComics[j].Score {
			return scoredComics[i].ID < scoredComics[j].ID
		}
		return scoredComics[i].Score > scoredComics[j].Score
	})

	resultCount := min(len(scoredComics), request.Limit)
	result := make([]int, resultCount)
	for i := 0; i < resultCount; i++ {
		result[i] = scoredComics[i].ID
	}

	// Find needed ids in db
	reply := make([]Comics, 0)
	for _, index := range result {
		comicsRaw, err := s.db.GetById(ctx, index)
		if err == nil {
			reply = append(reply, *comicsRaw)
		}
	}

	return &SearchReply{Comics: reply}, nil
}
