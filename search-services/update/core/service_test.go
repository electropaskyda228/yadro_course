package core

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockDB struct {
	mock.Mock
}

func (m *MockDB) Add(ctx context.Context, comics Comics) error {
	args := m.Called(ctx, comics)
	return args.Error(0)
}

func (m *MockDB) IDs(ctx context.Context) ([]int, error) {
	args := m.Called(ctx)
	return args.Get(0).([]int), args.Error(1)
}

func (m *MockDB) Stats(ctx context.Context) (DBStats, error) {
	args := m.Called(ctx)
	return args.Get(0).(DBStats), args.Error(1)
}

func (m *MockDB) Drop(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

type MockXKCD struct {
	mock.Mock
}

func (m *MockXKCD) LastID(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

func (m *MockXKCD) Get(ctx context.Context, id int) (XKCDInfo, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(XKCDInfo), args.Error(1)
}

type MockWords struct {
	mock.Mock
}

func (m *MockWords) Norm(ctx context.Context, phrase string) ([]string, error) {
	args := m.Called(ctx, phrase)
	return args.Get(0).([]string), args.Error(1)
}

type MockPublisher struct {
	mock.Mock
}

func (m *MockPublisher) SendDBChangedEvent(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func TestNewService(t *testing.T) {
	tests := []struct {
		name        string
		concurrency int
		wantErr     bool
	}{
		{
			name:        "valid concurrency",
			concurrency: 5,
			wantErr:     false,
		},
		{
			name:        "zero concurrency",
			concurrency: 0,
			wantErr:     true,
		},
		{
			name:        "negative concurrency",
			concurrency: -1,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := slog.Default()
			db := &MockDB{}
			xkcd := &MockXKCD{}
			words := &MockWords{}
			publisher := &MockPublisher{}

			service, err := NewService(log, db, xkcd, words, publisher, tt.concurrency)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, service)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, service)
				assert.Equal(t, tt.concurrency, service.concurrency)
			}
		})
	}
}

func TestService_Update_Success(t *testing.T) {
	ctx := context.Background()
	log := slog.Default()

	db := &MockDB{}
	xkcd := &MockXKCD{}
	words := &MockWords{}
	publisher := &MockPublisher{}

	xkcd.On("LastID", ctx).Return(3, nil)
	db.On("IDs", ctx).Return([]int{1, 2}, nil)

	comicsInfo := XKCDInfo{
		ID:          3,
		Title:       "Test Title",
		Description: "Test Description",
		SafeTitle:   "Test Safe Title",
		Transcript:  "Test Transcript",
		URL:         "https://xkcd.com/3",
	}

	xkcd.On("Get", ctx, 3).Return(comicsInfo, nil)

	words.On("Norm", ctx, mock.AnythingOfType("string")).Return([]string{"test", "title", "description"}, nil)

	db.On("Add", ctx, mock.MatchedBy(func(c Comics) bool {
		return c.ID == 3 && len(c.Words) > 0
	})).Return(nil)

	publisher.On("SendDBChangedEvent", ctx).Return(nil)

	service, err := NewService(log, db, xkcd, words, publisher, 2)
	require.NoError(t, err)

	err = service.Update(ctx)
	assert.NoError(t, err)

	xkcd.AssertExpectations(t)
	db.AssertExpectations(t)
	words.AssertExpectations(t)
	publisher.AssertExpectations(t)
}

func TestService_Update_AlreadyRunning(t *testing.T) {
	ctx := context.Background()
	log := slog.Default()

	db := &MockDB{}
	xkcd := &MockXKCD{}
	words := &MockWords{}
	publisher := &MockPublisher{}

	service, err := NewService(log, db, xkcd, words, publisher, 2)
	require.NoError(t, err)

	service.mu.Lock()

	var updateErr error
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		updateErr = service.Update(ctx)
	}()

	wg.Wait()
	service.mu.Unlock()

	assert.Error(t, updateErr)
	assert.Equal(t, ErrAlreadyExists, updateErr)
}

func TestService_Update_XKCDError(t *testing.T) {
	ctx := context.Background()
	log := slog.Default()

	db := &MockDB{}
	xkcd := &MockXKCD{}
	words := &MockWords{}
	publisher := &MockPublisher{}

	expectedErr := errors.New("xkcd error")
	xkcd.On("LastID", ctx).Return(0, expectedErr)

	service, err := NewService(log, db, xkcd, words, publisher, 2)
	require.NoError(t, err)

	err = service.Update(ctx)
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)

	xkcd.AssertExpectations(t)
	db.AssertNotCalled(t, "IDs", ctx)
}

func TestService_Update_DBError(t *testing.T) {
	ctx := context.Background()
	log := slog.Default()

	db := &MockDB{}
	xkcd := &MockXKCD{}
	words := &MockWords{}
	publisher := &MockPublisher{}

	xkcd.On("LastID", ctx).Return(3, nil)
	expectedErr := errors.New("db error")
	db.On("IDs", ctx).Return([]int{}, expectedErr)

	service, err := NewService(log, db, xkcd, words, publisher, 2)
	require.NoError(t, err)

	err = service.Update(ctx)
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)

	xkcd.AssertExpectations(t)
	db.AssertExpectations(t)
	words.AssertNotCalled(t, "Norm", ctx, mock.Anything)
}

func TestService_Update_Comics404(t *testing.T) {
	ctx := context.Background()
	log := slog.Default()

	db := &MockDB{}
	xkcd := &MockXKCD{}
	words := &MockWords{}
	publisher := &MockPublisher{}

	xkcd.On("LastID", ctx).Return(5, nil)
	db.On("IDs", ctx).Return([]int{1, 2, 3, 4}, nil)

	comicsInfo5 := XKCDInfo{
		ID:          5,
		Title:       "Test Title 5",
		Description: "Test Description 5",
		SafeTitle:   "Test Safe Title 5",
		Transcript:  "Test Transcript 5",
		URL:         "https://xkcd.com/5",
	}

	xkcd.On("Get", ctx, 5).Return(comicsInfo5, nil)

	words.On("Norm", ctx, mock.AnythingOfType("string")).Return([]string{"test", "title"}, nil).Maybe()

	words.On("Norm", ctx, mock.MatchedBy(func(s string) bool {
		return strings.Contains(s, "404") || strings.Contains(s, "Not found")
	})).Return([]string{"404", "not", "found"}, nil).Maybe()

	db.On("Add", ctx, mock.MatchedBy(func(c Comics) bool {
		return c.ID == 5 || c.ID == 404
	})).Return(nil)

	publisher.On("SendDBChangedEvent", ctx).Return(nil)

	service, err := NewService(log, db, xkcd, words, publisher, 2)
	require.NoError(t, err)

	err = service.Update(ctx)
	assert.NoError(t, err)

	xkcd.AssertExpectations(t)
	db.AssertExpectations(t)
	words.AssertExpectations(t)
	publisher.AssertExpectations(t)
	xkcd.AssertNotCalled(t, "Get", ctx, 404)
}

func TestService_Stats(t *testing.T) {
	ctx := context.Background()
	log := slog.Default()

	db := &MockDB{}
	xkcd := &MockXKCD{}
	words := &MockWords{}
	publisher := &MockPublisher{}

	dbStats := DBStats{
		WordsTotal:    100,
		WordsUnique:   80,
		ComicsFetched: 10,
	}

	db.On("Stats", ctx).Return(dbStats, nil)
	xkcd.On("LastID", ctx).Return(100, nil)

	service, err := NewService(log, db, xkcd, words, publisher, 2)
	require.NoError(t, err)

	stats, err := service.Stats(ctx)
	assert.NoError(t, err)
	assert.Equal(t, dbStats, stats.DBStats)
	assert.Equal(t, 100, stats.ComicsTotal)

	db.AssertExpectations(t)
	xkcd.AssertExpectations(t)
}

func TestService_Stats_DBError(t *testing.T) {
	ctx := context.Background()
	log := slog.Default()

	db := &MockDB{}
	xkcd := &MockXKCD{}
	words := &MockWords{}
	publisher := &MockPublisher{}

	expectedErr := errors.New("db stats error")
	db.On("Stats", ctx).Return(DBStats{}, expectedErr)

	service, err := NewService(log, db, xkcd, words, publisher, 2)
	require.NoError(t, err)

	stats, err := service.Stats(ctx)
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.Equal(t, ServiceStats{}, stats)

	db.AssertExpectations(t)
	xkcd.AssertNotCalled(t, "LastID", ctx)
}

func TestService_Status(t *testing.T) {
	log := slog.Default()
	db := &MockDB{}
	xkcd := &MockXKCD{}
	words := &MockWords{}
	publisher := &MockPublisher{}

	service, err := NewService(log, db, xkcd, words, publisher, 2)
	require.NoError(t, err)

	assert.Equal(t, StatusIdle, service.Status(context.Background()))

	service.updateProcessing.Store(true)
	assert.Equal(t, StatusRunning, service.Status(context.Background()))

	service.updateProcessing.Store(false)
	assert.Equal(t, StatusIdle, service.Status(context.Background()))
}

func TestService_Drop(t *testing.T) {
	ctx := context.Background()
	log := slog.Default()

	db := &MockDB{}
	xkcd := &MockXKCD{}
	words := &MockWords{}
	publisher := &MockPublisher{}

	db.On("Drop", ctx).Return(nil)

	service, err := NewService(log, db, xkcd, words, publisher, 2)
	require.NoError(t, err)

	err = service.Drop(ctx)
	assert.NoError(t, err)

	db.AssertExpectations(t)
}

func TestService_Drop_Error(t *testing.T) {
	ctx := context.Background()
	log := slog.Default()

	db := &MockDB{}
	xkcd := &MockXKCD{}
	words := &MockWords{}
	publisher := &MockPublisher{}

	expectedErr := errors.New("drop error")
	db.On("Drop", ctx).Return(expectedErr)

	service, err := NewService(log, db, xkcd, words, publisher, 2)
	require.NoError(t, err)

	err = service.Drop(ctx)
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)

	db.AssertExpectations(t)
}

func TestGetComicsById_NormalizationError(t *testing.T) {
	ctx := context.Background()
	log := slog.Default()

	db := &MockDB{}
	xkcd := &MockXKCD{}
	words := &MockWords{}
	publisher := &MockPublisher{}

	service, err := NewService(log, db, xkcd, words, publisher, 2)
	require.NoError(t, err)

	comicsInfo := XKCDInfo{
		ID:          1,
		Title:       "Test",
		Description: "Test",
		SafeTitle:   "Test",
		Transcript:  "Test",
		URL:         "https://xkcd.com/1",
	}

	xkcd.On("Get", ctx, 1).Return(comicsInfo, nil)

	expectedErr := errors.New("normalization error")
	words.On("Norm", ctx, mock.AnythingOfType("string")).Return([]string{}, expectedErr)

	comics, err := getComicsById(service, ctx, 1)
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.Equal(t, Comics{}, comics)

	xkcd.AssertExpectations(t)
	words.AssertExpectations(t)
}

func TestSplitWordsIntoChunks(t *testing.T) {
	tests := []struct {
		name     string
		words    []string
		maxSize  int
		expected []string
	}{
		{
			name:     "empty words",
			words:    []string{},
			maxSize:  10,
			expected: []string{},
		},
		{
			name:     "single word fits",
			words:    []string{"hello"},
			maxSize:  10,
			expected: []string{"hello"},
		},
		{
			name:     "multiple words fit in one chunk",
			words:    []string{"hello", "world"},
			maxSize:  20,
			expected: []string{"hello world"},
		},
		{
			name:     "words split into multiple chunks",
			words:    []string{"a", "verylongword", "short", "anotherverylongword"},
			maxSize:  15,
			expected: []string{"a verylongword", "short", "anotherverylongword"},
		},
		{
			name:     "exact fit",
			words:    []string{"1234567", "890"},
			maxSize:  11,
			expected: []string{"1234567 890"},
		},
		{
			name:     "single word exceeds max size",
			words:    []string{"extremelylongwordthatexceedsthemaxsize"},
			maxSize:  10,
			expected: []string{"extremelylongwordthatexceedsthemaxsize"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitWordsIntoChunks(tt.words, tt.maxSize)
			if tt.name == "empty words" {
				assert.Len(t, result, 0)
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
