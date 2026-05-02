package chatgptapp

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Store struct {
	db *sql.DB
}

type UserCredentials struct {
	UserID            string
	MobileSessionJSON string
	CalendarURL       string
}

func OpenStore(databaseURL string) (*Store, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(0)
	db.SetConnMaxLifetime(2 * time.Minute)
	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) CredentialsForAPIKey(ctx context.Context, apiKey string) (UserCredentials, error) {
	if s == nil || s.db == nil {
		return UserCredentials{}, fmt.Errorf("credential store is not configured")
	}
	hash := HashAPIKey(apiKey)
	var out UserCredentials
	err := s.db.QueryRowContext(ctx, `
		select user_id, mobile_session_json::text, coalesce(calendar_url, '')
		from moodle_app_users
		where api_key_hash = $1
	`, hash).Scan(&out.UserID, &out.MobileSessionJSON, &out.CalendarURL)
	if err == sql.ErrNoRows {
		return UserCredentials{}, ErrUnauthorized
	}
	if err != nil {
		return UserCredentials{}, err
	}
	return out, nil
}

func (s *Store) Ping(ctx context.Context) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("credential store is not configured")
	}
	return s.db.PingContext(ctx)
}
