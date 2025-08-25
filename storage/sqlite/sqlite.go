package sqlite

import (
	"context"
	"database/sql"
	"death-clock/storage"
	"errors"
	"fmt"
)

type Storage struct {
	db *sql.DB
}

var ErrNoSavedUsers = errors.New("no saved users")

func New(path string) (*Storage, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("can't open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("can't connect to database: %w", err)
	}

	return &Storage{db: db}, nil
}

func (s *Storage) InitUser(ctx context.Context, p *storage.User) error {
	q := `
	CREATE TABLE IF NOT EXISTS users (
		user_name TEXT PRIMARY KEY,
		is_death_age_asked BOOLEAN DEFAULT 0,
		is_birthday_asked BOOLEAN DEFAULT 0,
		death_age INTEGER DEFAULT NULL,
		birthday INTEGER DEFAULT NULL
	)`
	_, err := s.db.ExecContext(ctx, q)
	if err != nil {
		return fmt.Errorf("can't create table: %w", err)
	}

	// если нужно сразу создать пользователя
	_, err = s.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO users (user_name) VALUES (?)`,
		p.UserName,
	)
	if err != nil {
		return fmt.Errorf("can't init user: %w", err)
	}

	return nil
}

func (s *Storage) SaveUser(ctx context.Context, u *storage.User) error {
	q := `
	INSERT INTO users (user_name, is_death_age_asked, is_birthday_asked, death_age, birthday)
	VALUES (?, ?, ?, ?, ?)
	ON CONFLICT(user_name) DO UPDATE SET
		is_death_age_asked = excluded.is_death_age_asked,
		is_birthday_asked = excluded.is_birthday_asked,
		death_age = excluded.death_age,
		birthday = excluded.birthday
	`
	_, err := s.db.ExecContext(ctx, q,
		u.UserName,
		u.IsDeathAgeAsked,
		u.IsBirthdayAsked,
		u.DeathAge,
		u.BirthsDay,
	)
	if err != nil {
		return fmt.Errorf("can't save user: %w", err)
	}
	return nil
}

func (s *Storage) IsUserExists(ctx context.Context, userName string) (bool, error) {
	q := `SELECT COUNT(*) FROM users WHERE user_name = ?`
	var count int
	err := s.db.QueryRowContext(ctx, q, userName).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("can't check user existence: %w", err)
	}
	return count > 0, nil
}

func (s *Storage) GetUserData(ctx context.Context, userName string) (*storage.User, error) {
	q := `SELECT user_name, is_death_age_asked, is_birthday_asked, death_age, birthday 
	      FROM users WHERE user_name = ? LIMIT 1`

	u := &storage.User{}
	err := s.db.QueryRowContext(ctx, q, userName).Scan(
		&u.UserName,
		&u.IsDeathAgeAsked,
		&u.IsBirthdayAsked,
		&u.DeathAge,
		&u.BirthsDay,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNoSavedUsers
	}
	if err != nil {
		return nil, fmt.Errorf("can't get user data: %w", err)
	}
	return u, nil
}

func (s *Storage) InitSchema(ctx context.Context) error {
	q := `
	CREATE TABLE IF NOT EXISTS users (
		user_name TEXT PRIMARY KEY,
		is_death_age_asked BOOLEAN DEFAULT 0,
		is_birthday_asked BOOLEAN DEFAULT 0,
		death_age INTEGER DEFAULT NULL,
		birthday INTEGER DEFAULT NULL
	)`
	_, err := s.db.ExecContext(ctx, q)
	if err != nil {
		return fmt.Errorf("can't create table: %w", err)
	}
	return nil
}
