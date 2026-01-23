package core

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Mock implementations
type MockDB struct {
	mock.Mock
}

func (m *MockDB) Find(ctx context.Context, words []string, limit int) (*SearchReply, error) {
	args := m.Called(ctx, words, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*SearchReply), args.Error(1)
}

func (m *MockDB) FindAll(ctx context.Context) (*IndexInfo, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*IndexInfo), args.Error(1)
}

func (m *MockDB) GetById(ctx context.Context, id int) (*Comics, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Comics), args.Error(1)
}

type MockWords struct {
	mock.Mock
}

func (m *MockWords) Norm(ctx context.Context, phrase string) ([]string, error) {
	args := m.Called(ctx, phrase)
	return args.Get(0).([]string), args.Error(1)
}

func TestNewService(t *testing.T) {
	log := slog.Default()
	db := &MockDB{}
	words := &MockWords{}

	service, err := NewService(log, db, words)

	assert.NoError(t, err)
	assert.NotNil(t, service)
	assert.Equal(t, log, service.log)
	assert.Equal(t, db, service.db)
	assert.Equal(t, words, service.words)
	assert.NotNil(t, service.index)
	assert.Empty(t, service.index)
}

func TestService_UpdateIndex_Success(t *testing.T) {
	ctx := context.Background()
	log := slog.Default()
	db := &MockDB{}
	words := &MockWords{}

	indexData := &IndexInfo{
		Comics: []IndexInfoOne{
			{
				Word:       "test",
				Comics_ids: []int{1, 2, 3},
			},
			{
				Word:       "hello",
				Comics_ids: []int{1, 4},
			},
		},
	}

	db.On("FindAll", ctx).Return(indexData, nil)

	service, err := NewService(log, db, words)
	require.NoError(t, err)

	err = service.UpdateIndex(ctx)
	assert.NoError(t, err)

	assert.Len(t, service.index, 2)
	assert.Equal(t, map[int]bool{1: true, 2: true, 3: true}, service.index["test"])
	assert.Equal(t, map[int]bool{1: true, 4: true}, service.index["hello"])

	db.AssertExpectations(t)
}

func TestService_UpdateIndex_DBError(t *testing.T) {
	ctx := context.Background()
	log := slog.Default()
	db := &MockDB{}
	words := &MockWords{}

	expectedErr := errors.New("db error")
	db.On("FindAll", ctx).Return(nil, expectedErr)

	service, err := NewService(log, db, words)
	require.NoError(t, err)

	err = service.UpdateIndex(ctx)
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)

	db.AssertExpectations(t)
}

func TestService_Search_Success(t *testing.T) {
	ctx := context.Background()
	log := slog.Default()
	db := &MockDB{}
	words := &MockWords{}

	request := SearchRequest{
		Phrase: "test search",
		Limit:  10,
	}

	normalizedWords := []string{"test", "search"}
	expectedReply := &SearchReply{
		Comics: []Comics{
			{ID: 1, URL: "https://xkcd.com/1"},
			{ID: 2, URL: "https://xkcd.com/2"},
		},
	}

	words.On("Norm", ctx, request.Phrase).Return(normalizedWords, nil)
	db.On("Find", ctx, normalizedWords, request.Limit).Return(expectedReply, nil)

	service, err := NewService(log, db, words)
	require.NoError(t, err)

	reply, err := service.Search(ctx, request)
	assert.NoError(t, err)
	assert.Equal(t, expectedReply, reply)

	words.AssertExpectations(t)
	db.AssertExpectations(t)
}

func TestService_Search_NormalizationError(t *testing.T) {
	ctx := context.Background()
	log := slog.Default()
	db := &MockDB{}
	words := &MockWords{}

	request := SearchRequest{
		Phrase: "test search",
		Limit:  10,
	}

	expectedErr := errors.New("normalization error")
	words.On("Norm", ctx, request.Phrase).Return([]string{}, expectedErr)

	service, err := NewService(log, db, words)
	require.NoError(t, err)

	reply, err := service.Search(ctx, request)
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.Equal(t, &SearchReply{}, reply)

	words.AssertExpectations(t)
	db.AssertNotCalled(t, "Find", ctx, mock.Anything, mock.Anything)
}

func TestService_Search_DBError(t *testing.T) {
	ctx := context.Background()
	log := slog.Default()
	db := &MockDB{}
	words := &MockWords{}

	request := SearchRequest{
		Phrase: "test search",
		Limit:  10,
	}

	normalizedWords := []string{"test", "search"}
	expectedErr := errors.New("db find error")

	words.On("Norm", ctx, request.Phrase).Return(normalizedWords, nil)
	db.On("Find", ctx, normalizedWords, request.Limit).Return(nil, expectedErr)

	service, err := NewService(log, db, words)
	require.NoError(t, err)

	reply, err := service.Search(ctx, request)
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.Equal(t, &SearchReply{}, reply)

	words.AssertExpectations(t)
	db.AssertExpectations(t)
}

func TestService_SearchIndex_Success(t *testing.T) {
	ctx := context.Background()
	log := slog.Default()
	db := &MockDB{}
	words := &MockWords{}

	request := SearchRequest{
		Phrase: "test hello",
		Limit:  10,
	}

	normalizedWords := []string{"test", "hello"}

	words.On("Norm", ctx, request.Phrase).Return(normalizedWords, nil)

	service, err := NewService(log, db, words)
	require.NoError(t, err)

	service.index["test"] = map[int]bool{1: true, 2: true, 3: true}
	service.index["hello"] = map[int]bool{1: true, 4: true}
	service.index["world"] = map[int]bool{5: true}

	comic1 := &Comics{ID: 1, URL: "https://xkcd.com/1"}
	comic2 := &Comics{ID: 2, URL: "https://xkcd.com/2"}
	comic3 := &Comics{ID: 3, URL: "https://xkcd.com/3"}
	comic4 := &Comics{ID: 4, URL: "https://xkcd.com/4"}

	db.On("GetById", ctx, 1).Return(comic1, nil)
	db.On("GetById", ctx, 2).Return(comic2, nil)
	db.On("GetById", ctx, 3).Return(comic3, nil)
	db.On("GetById", ctx, 4).Return(comic4, nil)

	reply, err := service.SearchIndex(ctx, request)
	assert.NoError(t, err)
	assert.Len(t, reply.Comics, 4)

	assert.Equal(t, 1, reply.Comics[0].ID)
	assert.Equal(t, 2, reply.Comics[1].ID)
	assert.Equal(t, 3, reply.Comics[2].ID)
	assert.Equal(t, 4, reply.Comics[3].ID)

	words.AssertExpectations(t)
	db.AssertExpectations(t)
}

func TestService_SearchIndex_WithLimit(t *testing.T) {
	ctx := context.Background()
	log := slog.Default()
	db := &MockDB{}
	words := &MockWords{}

	request := SearchRequest{
		Phrase: "test hello",
		Limit:  2,
	}

	normalizedWords := []string{"test", "hello"}

	words.On("Norm", ctx, request.Phrase).Return(normalizedWords, nil)

	service, err := NewService(log, db, words)
	require.NoError(t, err)

	service.index["test"] = map[int]bool{1: true, 2: true, 3: true}
	service.index["hello"] = map[int]bool{1: true, 4: true}

	comic1 := &Comics{ID: 1, URL: "https://xkcd.com/1"}
	comic2 := &Comics{ID: 2, URL: "https://xkcd.com/2"}

	db.On("GetById", ctx, 1).Return(comic1, nil)
	db.On("GetById", ctx, 2).Return(comic2, nil)

	reply, err := service.SearchIndex(ctx, request)
	assert.NoError(t, err)
	assert.Len(t, reply.Comics, 2)
	assert.Equal(t, 1, reply.Comics[0].ID)
	assert.Equal(t, 2, reply.Comics[1].ID)

	words.AssertExpectations(t)
	db.AssertExpectations(t)
}

func TestService_SearchIndex_NoResults(t *testing.T) {
	ctx := context.Background()
	log := slog.Default()
	db := &MockDB{}
	words := &MockWords{}

	request := SearchRequest{
		Phrase: "unknown words",
		Limit:  10,
	}

	normalizedWords := []string{"unknown", "words"}

	words.On("Norm", ctx, request.Phrase).Return(normalizedWords, nil)

	service, err := NewService(log, db, words)
	require.NoError(t, err)

	service.index["test"] = map[int]bool{1: true}

	reply, err := service.SearchIndex(ctx, request)
	assert.NoError(t, err)
	assert.NotNil(t, reply)
	assert.Empty(t, reply.Comics)

	words.AssertExpectations(t)
	db.AssertNotCalled(t, "GetById", ctx, mock.Anything)
}

func TestService_SearchIndex_EmptyPhrase(t *testing.T) {
	ctx := context.Background()
	log := slog.Default()
	db := &MockDB{}
	words := &MockWords{}

	request := SearchRequest{
		Phrase: "",
		Limit:  10,
	}

	normalizedWords := []string{}

	words.On("Norm", ctx, request.Phrase).Return(normalizedWords, nil)

	service, err := NewService(log, db, words)
	require.NoError(t, err)

	service.index["test"] = map[int]bool{1: true}

	reply, err := service.SearchIndex(ctx, request)
	assert.NoError(t, err)
	assert.NotNil(t, reply)
	assert.Empty(t, reply.Comics)

	words.AssertExpectations(t)
	db.AssertNotCalled(t, "GetById", ctx, mock.Anything)
}

func TestService_SearchIndex_GetByIdError(t *testing.T) {
	ctx := context.Background()
	log := slog.Default()
	db := &MockDB{}
	words := &MockWords{}

	request := SearchRequest{
		Phrase: "test",
		Limit:  10,
	}

	normalizedWords := []string{"test"}

	words.On("Norm", ctx, request.Phrase).Return(normalizedWords, nil)

	service, err := NewService(log, db, words)
	require.NoError(t, err)

	service.index["test"] = map[int]bool{1: true, 2: true}

	comic1 := &Comics{ID: 1, URL: "https://xkcd.com/1"}
	expectedErr := errors.New("db error")

	db.On("GetById", ctx, 1).Return(comic1, nil)
	db.On("GetById", ctx, 2).Return(nil, expectedErr)

	reply, err := service.SearchIndex(ctx, request)
	assert.NoError(t, err)
	assert.Len(t, reply.Comics, 1)
	assert.Equal(t, 1, reply.Comics[0].ID)

	words.AssertExpectations(t)
	db.AssertExpectations(t)
}

func TestService_SearchIndex_NormalizationError(t *testing.T) {
	ctx := context.Background()
	log := slog.Default()
	db := &MockDB{}
	words := &MockWords{}

	request := SearchRequest{
		Phrase: "test search",
		Limit:  10,
	}

	expectedErr := errors.New("normalization error")
	words.On("Norm", ctx, request.Phrase).Return([]string{}, expectedErr)

	service, err := NewService(log, db, words)
	require.NoError(t, err)

	reply, err := service.SearchIndex(ctx, request)
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.Equal(t, &SearchReply{}, reply)

	words.AssertExpectations(t)
	db.AssertNotCalled(t, "GetById", ctx, mock.Anything)
}

func TestService_SearchIndex_TieBreaking(t *testing.T) {
	ctx := context.Background()
	log := slog.Default()
	db := &MockDB{}
	words := &MockWords{}

	request := SearchRequest{
		Phrase: "test",
		Limit:  5,
	}

	normalizedWords := []string{"test"}

	words.On("Norm", ctx, request.Phrase).Return(normalizedWords, nil)

	service, err := NewService(log, db, words)
	require.NoError(t, err)

	service.index["test"] = map[int]bool{
		5: true,
		1: true,
		3: true,
		2: true,
		4: true,
	}

	for i := 1; i <= 5; i++ {
		comic := &Comics{ID: i, URL: "https://xkcd.com/" + string(rune('0'+i))}
		db.On("GetById", ctx, i).Return(comic, nil)
	}

	reply, err := service.SearchIndex(ctx, request)
	assert.NoError(t, err)

	assert.Len(t, reply.Comics, 5)
	for i, comic := range reply.Comics {
		assert.Equal(t, i+1, comic.ID)
	}

	words.AssertExpectations(t)
	db.AssertExpectations(t)
}
