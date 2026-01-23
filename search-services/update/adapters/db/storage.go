package db

import (
	"context"
	"log/slog"
	"strconv"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"yadro.com/course/update/core"
)

type DB struct {
	log  *slog.Logger
	conn *sqlx.DB
}

func New(log *slog.Logger, address string) (*DB, error) {

	db, err := sqlx.Connect("pgx", address)
	if err != nil {
		log.Error("connection problem", "address", address, "error", err)
		return nil, err
	}

	return &DB{
		log:  log,
		conn: db,
	}, nil
}

func (db *DB) Add(ctx context.Context, comics core.Comics) error {
	query := `
		INSERT INTO comics (id, url, words)
		VALUES ($1, $2, $3)
		ON CONFLICT (id) DO UPDATE SET
			url = EXCLUDED.url,
			words = EXCLUDED.words
	`
	_, err := db.conn.Exec(query, comics.ID, comics.URL, comics.Words)
	if err != nil {
		db.log.Error("Failed to add "+strconv.Itoa(comics.ID)+" comics to db", "error", err)
		return err
	}
	db.log.Info("Comics " + strconv.Itoa(comics.ID) + " has been added to db")
	return nil
}

func (db *DB) Stats(ctx context.Context) (core.DBStats, error) {
	var stats core.DBStats

	db.log.Info("Start getting stats from db")
	err := db.conn.GetContext(ctx, &stats.ComicsFetched, "SELECT COUNT(*) FROM comics")
	if err != nil {
		db.log.Error("Failed to get amount of comics in db", "error", err)
		return core.DBStats{}, err
	}

	err = db.conn.GetContext(ctx, &stats.WordsTotal, `
		SELECT COUNT(*)
		FROM (
			SELECT unnest(words)
			FROM comics
		) AS all_words
	`)
	if err != nil {
		db.log.Error("Failed to get amount of words in db", "error", err)
		return core.DBStats{}, err
	}

	err = db.conn.GetContext(ctx, &stats.WordsUnique, `
		SELECT COUNT(DISTINCT words)
		FROM (
			SELECT unnest(words) as words
			FROM comics
		) AS unique_words
	`)
	if err != nil {
		db.log.Error("Failed to get amount of unique words in db", "error", err)
		return core.DBStats{}, err
	}
	return stats, nil
}

func (db *DB) IDs(ctx context.Context) ([]int, error) {
	query := "SELECT id FROM comics ORDER BY id"
	var ids []int

	err := db.conn.SelectContext(ctx, &ids, query)
	if err != nil {
		db.log.Error("Falied to get comics' ids from db", "error", err)
		return nil, err
	}
	db.log.Info("Comics' ids in db has been gotten")
	return ids, nil
}

func (db *DB) Drop(ctx context.Context) error {
	if _, err := db.conn.Exec("DELETE FROM comics"); err != nil {
		db.log.Error("Failed to delete information from comics in db", "error", err)
		return err
	}
	db.log.Info("All information about comics have been deleted in db")
	return nil
}
