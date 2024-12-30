package rest

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/AndreyChufelin/movies-api/internal/storage"
	"github.com/labstack/echo/v4"
)

func (s *Server) createMovieHandler(c echo.Context) error {
	var input struct {
		Title   string          `json:"title"`
		Year    int32           `json:"year"`
		Runtime storage.Runtime `json:"runtime"`
		Genres  []string        `json:"genres"`
	}
	err := c.Bind(&input)
	if err != nil {
		return bindMovieError(err)
	}

	movie := &storage.Movie{
		Title:   input.Title,
		Year:    input.Year,
		Runtime: input.Runtime,
		Genres:  input.Genres,
	}
	if err = c.Validate(movie); err != nil {
		return err
	}

	err = s.storage.CreateMovie(movie)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	c.Response().Header().Set("Location", fmt.Sprintf("/v1/movies/%d", movie.ID))

	return c.JSON(http.StatusOK, envelope{
		"movie": movie,
	})
}

func (s *Server) getMovieHandler(c echo.Context) error {
	var id int64
	err := echo.PathParamsBinder(c).
		Int64("id", &id).
		BindError()
	if err != nil {
		var verr *echo.BindingError
		if ok := errors.As(err, &verr); ok {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("wrong %s", verr.Field))
		}
		panic("failed to bind pathparams in getMovieHandler")
	}

	movie, err := s.storage.GetMovie(id)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrRecordNotFound):
			return echo.NewHTTPError(http.StatusNotFound, "movie not found")
		default:
			return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
		}
	}

	return c.JSON(http.StatusOK, envelope{
		"movie": movie,
	})
}

func (s *Server) updateMovieHandler(c echo.Context) error {
	var input struct {
		ID      int64            `param:"id"`
		Title   *string          `json:"title"`
		Year    *int32           `json:"year"`
		Runtime *storage.Runtime `json:"runtime"`
		Genres  []string         `json:"genres"`
	}

	err := c.Bind(&input)
	if err != nil {
		return bindMovieError(err)
	}

	movie, err := s.storage.GetMovie(input.ID)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrRecordNotFound):
			return echo.NewHTTPError(http.StatusNotFound, "movie not found")
		default:
			return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
		}
	}

	if input.Title != nil {
		movie.Title = *input.Title
	}
	if input.Year != nil {
		movie.Year = *input.Year
	}
	if input.Runtime != nil {
		movie.Runtime = *input.Runtime
	}
	if input.Genres != nil {
		movie.Genres = input.Genres
	}

	if err = c.Validate(movie); err != nil {
		return err
	}

	err = s.storage.UpdateMovie(movie)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrEditConflict):
			return echo.NewHTTPError(
				http.StatusNotFound,
				"unable to update the record due to an edit conflict, please try again",
			)
		default:
			return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
		}
	}

	return c.JSON(http.StatusOK, envelope{
		"movie": movie,
	})
}

func (s *Server) deleteMovieHandler(c echo.Context) error {
	var id int64
	err := echo.PathParamsBinder(c).
		Int64("id", &id).
		BindError()
	if err != nil {
		var verr *echo.BindingError
		if ok := errors.As(err, &verr); ok {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("wrong %s", verr.Field))
		}
		panic("failed to bind pathparams in getMovieHandler")
	}

	err = s.storage.DeleteMovie(id)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrRecordNotFound):
			return echo.NewHTTPError(http.StatusNotFound, "movie not found")
		default:
			return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
		}
	}

	return c.JSON(http.StatusOK, envelope{
		"message": "movie successfully deleted",
	})
}

func bindMovieError(err error) error {
	if err != nil {
		var jerr *json.UnmarshalTypeError
		if ok := errors.As(err, &jerr); ok {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid %s", jerr.Field))
		}
		if errors.Is(err, storage.ErrInvalidRuntimeFormat) {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid runtime value")
		}
		return echo.NewHTTPError(http.StatusBadRequest, "bad request")
	}

	return nil
}
