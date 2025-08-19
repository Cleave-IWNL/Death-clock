package sqlite

import (
	"context"
	"database/sql"
	"death-clock/storage"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

type Storage struct {
	db *sql.DB
}

func New(path string) (*Storage, error) {
	db, err := sql.Open("sqlite3", path)

	if err != nil {
		return nil, fmt.Errorf("can't open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("can't connect to the database: %w", err)
	}

	return &Storage{
		db: db,
	}, nil
}

func (s *Storage) Save(ctx context.Context, p *storage.Page) error {
	q := `INSERT INTO pages (url, user_name) VALUES (?, ?)`

	if _, err := s.db.ExecContext(ctx, q, p.URL, p.UserName); err != nil {
		return fmt.Errorf("can't save page: %w", err)
	}

	return nil
}

func (s *Storage) PickRandom(ctx context.Context, userName string) (*storage.Page, error) {
	q := `SELECT url FROM pages WHERE user_name = ? ORDER BY RANDOM() LIMIT 1`

	var url string

	err := s.db.QueryRowContext(ctx, q, userName).Scan(&url)

	if err == sql.ErrNoRows {
		return nil, storage.ErrNoSavedPages
	}

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("can't pick random page: %w", err)
	}

	return &storage.Page{
		URL:      url,
		UserName: userName,
	}, nil
}

func (s *Storage) Remove(ctx context.Context, p *storage.Page) error {
	q := `DELETE FROM pages WHERE url = ? AND user_name = ?`

	if _, err := s.db.ExecContext(ctx, q, p.URL, p.UserName); err != nil {
		return fmt.Errorf("can't remove page: %w", err)
	}

	return nil
}

func (s *Storage) IsExists(ctx context.Context, p *storage.Page) (bool, error) {
	q := `SELECT COUNT(*) FROM pages WHERE url = ? AND user_name = ?`

	var count int
	err := s.db.QueryRowContext(ctx, q, p.URL, p.UserName).Scan(&count)

	if err == sql.ErrNoRows {
		return false, fmt.Errorf("can't check if page exists: %w", err)
	}

	return count > 0, nil
}

func (s *Storage) Init(ctx context.Context) error {
	q := `CREATE TABLE IF NOT EXISTS pages (url TEXT, user_name TEXT, is_death_age_asked BOOLEAN DEFAULT 0)`

	_, err := s.db.ExecContext(ctx, q)

	if err != nil {
		return fmt.Errorf("can't create table %w", err)
	}

	return nil
}

func (s *Storage) IsDea(ctx context.Context, p *storage.Page) (bool, error) {
	q := `SELECT COUNT(*) FROM pages WHERE url = ? AND user_name = ?`

	var count int
	err := s.db.QueryRowContext(ctx, q, p.URL, p.UserName).Scan(&count)

	if err == sql.ErrNoRows {
		return false, fmt.Errorf("can't check if page exists: %w", err)
	}

	return count > 0, nil
}

func (s *Storage) GetLatestPage(ctx context.Context, userName string) (*storage.Page, error) {
	q := `SELECT url, is_death_age_asked FROM pages WHERE user_name = ? ORDER BY ROWID DESC LIMIT 1`
	var url string
	var isDeathAgeAsked bool
	err := s.db.QueryRowContext(ctx, q, userName).Scan(&url, &isDeathAgeAsked)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("can't get latest page: %w", err)
	}
	return &storage.Page{
		URL:             url,
		UserName:        userName,
		IsDeathAgeAsked: isDeathAgeAsked,
	}, nil
}
