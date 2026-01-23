package core

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

const maxChunkSize = 4 * 1024

type Service struct {
	log         *slog.Logger
	db          DB
	xkcd        XKCD
	words       Words
	publisher   DBPublisher
	concurrency int

	updateProcessing atomic.Bool
	mu               sync.Mutex
}

func NewService(
	log *slog.Logger, db DB, xkcd XKCD, words Words, publisher DBPublisher, concurrency int,
) (*Service, error) {
	if concurrency < 1 {
		return nil, fmt.Errorf("wrong concurrency specified: %d", concurrency)
	}
	return &Service{
		log:         log,
		db:          db,
		xkcd:        xkcd,
		words:       words,
		publisher:   publisher,
		concurrency: concurrency,
	}, nil
}

func (s *Service) Update(ctx context.Context) error {
	if ok := s.mu.TryLock(); !ok {
		s.log.Error("service already runs update")
		return ErrAlreadyExists
	}
	defer s.mu.Unlock()

	s.updateProcessing.Store(true)
	defer s.updateProcessing.Store(false)

	s.log.Info("Start updating db")
	lastId, err := s.xkcd.LastID(ctx)
	if err != nil {
		s.log.Error("Failed update db", "error", err)
		return err
	}
	ids, err := s.db.IDs(ctx)
	if err != nil {
		s.log.Error("Failed update db", "error", err)
		return err
	}

	// Gorrutins
	s.log.Info("Start gorutings")
	jobs := make(chan int, lastId)

	var wg sync.WaitGroup
	for i := 0; i < s.concurrency; i++ {
		wg.Add(1)
		go worker(s, ctx, jobs, &wg)
	}

	existingIDs := make(map[int]bool)
	for _, id := range ids {
		existingIDs[id] = true
	}

	shouldSendEvent := false
	for i := 1; i <= lastId; i++ {
		if !existingIDs[i] {
			jobs <- i
			shouldSendEvent = true
		}
	}

	close(jobs)

	wg.Wait()
	s.log.Info("End gorutings")

	if shouldSendEvent {
		if err := s.publisher.SendDBChangedEvent(ctx); err != nil {
			s.log.Error("Error publishing db changed event")
		}
	}

	return nil
}

func worker(s *Service, ctx context.Context, jobs <-chan int, wg *sync.WaitGroup) {
	defer wg.Done()
	for job := range jobs {
		comics, err := getComicsById(s, ctx, job)
		if err != nil {
			s.log.Error("Failed to add comics "+strconv.Itoa(job)+" to db", "error", err)
			continue
		}
		err = s.db.Add(ctx, comics)
		if err != nil {
			s.log.Error("Failed to add comics "+strconv.Itoa(job)+" to db", "error", err)
			continue
		}
		s.log.Info("Comics " + strconv.Itoa(job) + " has been added to db")
	}
}

func getComicsById(s *Service, ctx context.Context, i int) (Comics, error) {
	s.log.Info("Load info about comics " + strconv.Itoa(i))
	var comicsRaw XKCDInfo
	if i == 404 {
		comicsRaw = XKCDInfo{ID: i, Title: "404", Description: "Not found", SafeTitle: "404", Transcript: "Not found"}
	} else {
		var err error
		comicsRaw, err = s.xkcd.Get(ctx, i)
		if err != nil {
			s.log.Error("Failed load info about comics "+strconv.Itoa(i), "error", nil)
			return Comics{}, err
		}
	}

	s.log.Info("Starting normilize comics " + strconv.Itoa(i))

	words := strings.Fields(comicsRaw.Title + " " + comicsRaw.Description + " " + comicsRaw.SafeTitle + " " + comicsRaw.Transcript)
	chunks := splitWordsIntoChunks(words, maxChunkSize)
	var finalNormalized []string

	for _, chunk := range chunks {
		normalized, err := s.words.Norm(ctx, chunk)
		if err != nil {
			s.log.Error("failed to normalize chunk", "error", err)
			return Comics{}, err
		}
		finalNormalized = append(finalNormalized, normalized...)
	}

	s.log.Info("End normilize comics " + strconv.Itoa(i))

	comics := Comics{
		ID:    comicsRaw.ID,
		URL:   comicsRaw.URL,
		Words: finalNormalized,
	}
	return comics, nil
}

func splitWordsIntoChunks(words []string, maxSize int) []string {
	var chunks []string
	var currentChunk strings.Builder

	for _, word := range words {
		if currentChunk.Len()+len(word)+1 > maxSize && currentChunk.Len() > 0 {
			chunks = append(chunks, currentChunk.String())
			currentChunk.Reset()
		}

		if currentChunk.Len() > 0 {
			currentChunk.WriteString(" ")
		}
		currentChunk.WriteString(word)
	}

	if currentChunk.Len() > 0 {
		chunks = append(chunks, currentChunk.String())
	}

	return chunks
}

func (s *Service) Stats(ctx context.Context) (ServiceStats, error) {
	var serviceStats ServiceStats
	s.log.Info("Start getting service stats")

	stats, err := s.db.Stats(ctx)
	if err != nil {
		s.log.Error("Failed to get stats information in db", "error", err)
		return ServiceStats{}, err
	}
	serviceStats.DBStats = stats

	cntComics, err := s.xkcd.LastID(ctx)
	if err != nil {
		s.log.Error("Failed to get lastId in xkcd", "error", err)
		return ServiceStats{}, err
	}
	serviceStats.ComicsTotal = cntComics

	return serviceStats, nil

}

func (s *Service) Status(ctx context.Context) ServiceStatus {
	if s.updateProcessing.Load() {
		return StatusRunning
	}
	return StatusIdle

}

func (s *Service) Drop(ctx context.Context) error {
	if err := s.db.Drop(ctx); err != nil {
		s.log.Error("Failed to delete information about comics", "error", err)
		return err
	}
	return nil
}
