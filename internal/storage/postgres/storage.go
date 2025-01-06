package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Storage struct {
	db       *pgxpool.Pool
	host     string
	port     string
	user     string
	password string
	name     string
}

func NewStorage(host, port, user, password, name string) Storage {
	return Storage{
		host:     host,
		port:     port,
		user:     user,
		password: password,
		name:     name,
	}
}

func (s *Storage) Connect(ctx context.Context) error {
	var err error
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", s.user, s.password, s.host, s.port, s.name)
	s.db, err = pgxpool.New(ctx, dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to postgres: %w", err)
	}

	return nil
}

func (s *Storage) Close(_ context.Context) error {
	if s.db == nil {
		return fmt.Errorf("no connection to close")
	}
	s.db.Close()

	return nil
}
