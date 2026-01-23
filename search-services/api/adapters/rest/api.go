package rest

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"yadro.com/course/api/core"
)

type PingResponse struct {
	Answer map[string]string `json:"replies"`
}

func NewPingHandler(log *slog.Logger, pingers map[string]core.Pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response := PingResponse{}
		response.Answer = make(map[string]string)

		for name, pinger := range pingers {
			if err := pinger.Ping(r.Context()); err != nil {
				response.Answer[name] = "unavailable"
				log.Error("service is not available", "service", name)
				continue
			}
			response.Answer[name] = "ok"
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Error("Response cannot be encoded", "error", err)
		}

	}
}

type WordsResponse struct {
	Words []string `json:"words"`
	Total int      `json:"total"`
}

func NewWordsHandler(log *slog.Logger, norm core.Normalizer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		phrase := r.URL.Query().Get("phrase")
		if phrase == "" {
			log.Error("phrase is empty")
			http.Error(w, "phrase is empty", http.StatusBadRequest)
			return
		}
		words, err := norm.Norm(r.Context(), phrase)
		if err != nil {
			log.Error("phrase '"+phrase+"' cannot be normalize", "error", err)
			if errors.Is(err, core.ErrBadArguments) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		result := WordsResponse{}
		result.Words = words
		result.Total = len(words)
		if err := json.NewEncoder(w).Encode(result); err != nil {
			log.Error("server cannot make reply words", "error", err)
		}
	}
}

func NewUpdateHandler(log *slog.Logger, updater core.Updater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := updater.Update(r.Context())
		if err != nil {
			if err.Error() == core.ErrAlreadyExists.Error() {
				w.WriteHeader(http.StatusAccepted)
				_, err := w.Write([]byte("Request accepted for processing"))
				if err != nil {
					log.Error("Strange error about response", "error", err)
				}
				return
			}
			log.Error("Failed to answer update rest request", "error", err)
			http.Error(w, "Error in server"+err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte("Request accepted for processing"))
		if err != nil {
			log.Error("Strange error about response", "error", err)
		}
	}
}

type StatsReply struct {
	WordsTotal    int `json:"words_total"`
	WordsUnique   int `json:"words_unique"`
	ComicsFetched int `json:"comics_fetched"`
	ComicsTotal   int `json:"comics_total"`
}

func NewUpdateStatsHandler(log *slog.Logger, updater core.Updater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reply, err := updater.Stats(r.Context())
		if err != nil {
			log.Error("Failed to answer stats rest request", "error", err)
			http.Error(w, "Error in server", http.StatusInternalServerError)
			return
		}
		result := StatsReply{
			WordsTotal:    reply.WordsTotal,
			WordsUnique:   reply.WordsUnique,
			ComicsFetched: reply.ComicsFetched,
			ComicsTotal:   reply.ComicsTotal,
		}
		if err := json.NewEncoder(w).Encode(result); err != nil {
			log.Error("server cannot make reply status", "error", err)
		}
	}
}

type StatusReply struct {
	Status string `json:"status"`
}

func NewUpdateStatusHandler(log *slog.Logger, updater core.Updater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		statusReply, err := updater.Status(r.Context())
		if err != nil {
			log.Error("Status cannot be gotten", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		result := StatusReply{}
		result.Status = string(statusReply)
		if err := json.NewEncoder(w).Encode(result); err != nil {
			log.Error("server cannot make reply status", "error", err)
		}
	}
}

func NewDropHandler(log *slog.Logger, updater core.Updater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := updater.Drop(r.Context())
		if err != nil {
			log.Error("Stats cannot be gotten", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte("Command 'drop' has been successfully procceed"))
		if err != nil {
			log.Error("Strange error about response", "error", err)
		}
	}
}

type SearchResponse struct {
	Comics []core.Comics `json:"comics"`
	Total  int           `json:"total"`
}

func NewSearchHandler(log *slog.Logger, searcher core.Searcher) http.HandlerFunc {
	return searchHandlerCommon(log, searcher, false)
}

func NewSearchIndexHandler(log *slog.Logger, searcher core.Searcher) http.HandlerFunc {
	return searchHandlerCommon(log, searcher, true)
}

func searchHandlerCommon(log *slog.Logger, searcher core.Searcher, withIndex bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limitRaw := r.URL.Query().Get("limit")
		var limit int
		if limitRaw == "" {
			limit = 10
		} else {
			var err error
			limit, err = strconv.Atoi(limitRaw)
			if err != nil || limit <= 0 {
				log.Error("Wrong limit param from rest", "error", err)
				http.Error(w, "limit should be not negative integer", http.StatusBadRequest)
				return
			}
		}
		phrase := r.URL.Query().Get("phrase")
		if phrase == "" {
			log.Error("Wrong prase param from rest", "error", errors.New("phrase should be not empty"))
			http.Error(w, "phrase should be not empty", http.StatusBadRequest)
			return
		}

		var answer []core.Comics
		var err error
		if withIndex {
			answer, err = searcher.SearchIndex(r.Context(), phrase, limit)
		} else {
			answer, err = searcher.Search(r.Context(), phrase, limit)
		}

		if err != nil {
			log.Error("Cannot answer search request in rest", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		response := SearchResponse{Comics: answer, Total: len(answer)}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Error("server cannot make reply search request", "error", err)
		}
	}
}

func NewLoginHandler(log *slog.Logger, loginer core.Loginer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		type Credentials struct {
			Name     string `json:"name"`
			Password string `json:"password"`
		}

		var creds Credentials

		err := json.NewDecoder(r.Body).Decode(&creds)
		if err != nil {
			http.Error(w, "Invalid JSON: ", http.StatusUnauthorized)
			return
		}
		defer func() {
			if err := r.Body.Close(); err != nil {
				log.Error("problem with closing wordsClient", "error", err)
			}
		}()

		if creds.Name == "" || creds.Password == "" {
			http.Error(w, "Name and password are required", http.StatusUnauthorized)
			return
		}

		token, err := loginer.Login(creds.Name, creds.Password)
		if err != nil {
			if err.Error() == core.ErrUnauthorized.Error() {
				http.Error(w, "wrong login or password", http.StatusUnauthorized)
				return
			}
			log.Error("Error with login", "error", err)
			http.Error(w, "Unknown error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(token)); err != nil {
			log.Error("error with response", "error", err)
		}
	}
}
