package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/AndreyChufelin/movies-api/internal/storage"
	"github.com/jackc/pgx/v5"
)

func (s Storage) CreateMovie(movie *storage.Movie) error {
	query := `
		INSERT INTO movies (title, year, runtime, genres)
		VALUES (@title, @year, @runtime, @genres)
		RETURNING id, created_at, version`
	args := pgx.NamedArgs{
		"title":   movie.Title,
		"year":    movie.Year,
		"runtime": movie.Runtime,
		"genres":  movie.Genres,
	}

	err := s.db.QueryRow(context.Background(), query, args).
		Scan(&movie.ID, &movie.CreatedAt, &movie.Version)
	if err != nil {
		return fmt.Errorf("failed to query create movie: %w", err)
	}

	return nil
}

func (s Storage) GetMovie(id int64) (*storage.Movie, error) {
	if id < 1 {
		return nil, storage.ErrRecordNotFound
	}
	query := `
		SELECT id, created_at, title, year, runtime, genres, version
		FROM movies
		WHERE id = $1`

	rows, err := s.db.Query(context.Background(), query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to query get movie: %w", err)
	}
	movie, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[storage.Movie])
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrRecordNotFound
		}
		return nil, fmt.Errorf("failed to get movie: %w", err)
	}

	return &movie, nil
}

func (s Storage) UpdateMovie(movie *storage.Movie) error {
	query := `
		UPDATE movies
		SET title = @title, year = @year, runtime = @runtime, genres = @genres, version = version + 1
		WHERE id = @id
		RETURNING version`

	args := pgx.NamedArgs{
		"id":      movie.ID,
		"title":   movie.Title,
		"year":    movie.Year,
		"runtime": movie.Runtime,
		"genres":  movie.Genres,
	}

	err := s.db.QueryRow(context.Background(), query, args).
		Scan(&movie.Version)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.ErrRecordNotFound
		}
		return fmt.Errorf("failed to query update movie: %w", err)
	}

	return nil
}

func (s Storage) DeleteMovie(id int64) error {
	if id < 1 {
		return storage.ErrRecordNotFound
	}

	query := `
		DELETE FROM movies
		WHERE id = $1`
	result, err := s.db.Exec(context.Background(), query, id)
	if err != nil {
		return err
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrRecordNotFound
	}

	return nil
}
