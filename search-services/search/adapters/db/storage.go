package db

import (
	"context"
	"log/slog"
	"strconv"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"yadro.com/course/search/core"
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

func (db *DB) Find(ctx context.Context, words []string, limit int) (*core.SearchReply, error) {
	db.log.Info("Start searching comics for words: " + strings.Join(words, ", "))
	query := `
    SELECT id, url FROM comics 
	WHERE words && $1::text[] 
	ORDER BY 
	CASE WHEN (SELECT COUNT(*) FROM unnest(words) AS word WHERE word = ANY($1::text[])) = array_length($1::text[], 1) THEN 0 ELSE 1 END,
	(
		SELECT COUNT(*) 
		FROM unnest(words) AS word 
		WHERE word = ANY($1::text[])
	) DESC,
	array_length(words, 1) ASC
	LIMIT $2
	`
	rows, err := db.conn.Query(query, pq.Array(words), limit)
	if err != nil {
		db.log.Error("Failed to find comics by needed words", "error", err)
		return &core.SearchReply{}, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			db.log.Error("Failed to close db information", "error", err)
		}
	}()

	var comics []core.Comics
	for rows.Next() {
		var comic core.Comics
		err := rows.Scan(&comic.ID, &comic.URL)
		if err != nil {
			db.log.Error("Failed to scan comics", "error", err)
			return &core.SearchReply{}, err
		}
		comics = append(comics, comic)
	}
	db.log.Info("All information about needed comics has been recieved. Total amount of comics: " + strconv.Itoa(len(comics)))
	return &core.SearchReply{Comics: comics}, nil
}

func (db *DB) FindAll(ctx context.Context) (*core.IndexInfo, error) {
	db.log.Info("Start load all comics in db")

	type row struct {
		Word       string        `db:"word"`
		Comics_ids pq.Int32Array `db:"comics_ids"`
	}

	var rows []row
	err := db.conn.SelectContext(ctx, &rows, `
        SELECT 
            word,
            array_agg(id) as comics_ids
        FROM (
            SELECT id, unnest(words) as word
            FROM comics
        ) expanded
        GROUP BY word
        ORDER BY word;
    `)
	if err != nil {
		db.log.Error("Failed to load information about db", "error", err)
		return &core.IndexInfo{}, err
	}

	comics := make([]core.IndexInfoOne, len(rows))
	for i, r := range rows {
		// Конвертируем []int32 в []int
		comicIDs := make([]int, len(r.Comics_ids))
		for j, id := range r.Comics_ids {
			comicIDs[j] = int(id)
		}

		comics[i] = core.IndexInfoOne{
			Word:       r.Word,
			Comics_ids: comicIDs,
		}
	}

	db.log.Info("All information about needed comics has been received. Total amount of words: " + strconv.Itoa(len(comics)))
	return &core.IndexInfo{Comics: comics}, nil
}

func (db *DB) GetById(ctx context.Context, id int) (*core.Comics, error) {
	db.log.Info("Start to load comics with id: " + strconv.Itoa(id))
	query := `SELECT url FROM comics WHERE id = $1`
	var url string

	err := db.conn.GetContext(ctx, &url, query, id)
	if err != nil {
		db.log.Error("Failed to find comics with id: "+strconv.Itoa(id), "error", err)
		return &core.Comics{}, err
	}

	return &core.Comics{ID: id, URL: url}, nil
}
