package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

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

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := s.db.QueryRow(ctx, query, args).
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

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := s.db.Query(ctx, query, id)
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

func (s Storage) GetAllMovies(title string, genres []string, filters storage.Filters) ([]*storage.Movie, storage.Metadata, error) {
	query := fmt.Sprintf(`
		SELECT  count(*) OVER(), id, created_at, title, year, runtime, genres, version
		FROM movies
		WHERE (to_tsvector('simple', title) @@ plainto_tsquery('simple', @title) OR @title = '')
		AND (genres @> @genres OR @genres = '{""}')
		ORDER BY %s %s, id ASC
		LIMIT @limit OFFSET @offset`, sortColumn(filters), sortDirection(filters))

	args := pgx.NamedArgs{
		"title":  title,
		"genres": genres,
		"limit":  filters.PageSize,
		"offset": filters.Offset(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := s.db.Query(ctx, query, args)
	if err != nil {
		return nil, storage.Metadata{}, fmt.Errorf("failed to query get all movies: %w", err)
	}
	defer rows.Close()

	movies := []*storage.Movie{}
	totalRecords := 0

	for rows.Next() {
		var movie storage.Movie
		err := rows.Scan(
			&totalRecords,
			&movie.ID,
			&movie.CreatedAt,
			&movie.Title,
			&movie.Year,
			&movie.Runtime,
			&movie.Genres,
			&movie.Version,
		)
		if err != nil {
			return nil, storage.Metadata{}, fmt.Errorf("failed to scan all movies: %w", err)
		}
		movies = append(movies, &movie)
	}
	if err = rows.Err(); err != nil {
		return nil, storage.Metadata{}, fmt.Errorf("failed to get all movies: %w", err)
	}

	metadata := storage.NewMetadata(totalRecords, filters.Page, filters.PageSize)
	return movies, metadata, nil
}

func (s Storage) UpdateMovie(movie *storage.Movie) error {
	query := `
		UPDATE movies
		SET title = @title, year = @year, runtime = @runtime, genres = @genres, version = version + 1
		WHERE id = @id AND version = @version
		RETURNING version`

	args := pgx.NamedArgs{
		"id":      movie.ID,
		"title":   movie.Title,
		"year":    movie.Year,
		"runtime": movie.Runtime,
		"genres":  movie.Genres,
		"version": movie.Version,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := s.db.QueryRow(ctx, query, args).
		Scan(&movie.Version)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.ErrEditConflict
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

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := s.db.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrRecordNotFound
	}

	return nil
}

func sortColumn(filters storage.Filters) string {
	for _, safeValue := range filters.SortSafelist {
		if filters.Sort == safeValue {
			return strings.TrimPrefix(filters.Sort, "-")
		}
	}
	panic("unsafe sort parameter: " + filters.Sort)
}

func sortDirection(filters storage.Filters) string {
	if strings.HasPrefix(filters.Sort, "-") {
		return "DESC"
	}
	return "ASC"
}
