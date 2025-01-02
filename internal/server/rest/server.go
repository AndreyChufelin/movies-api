package rest

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/AndreyChufelin/movies-api/internal/storage"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type Server struct {
	addr        string
	idleTimeout time.Duration
	readTimeout time.Duration
	writeTimout time.Duration
	storage     Storage
}

type Storage interface {
	CreateMovie(movie *storage.Movie) error
	GetMovie(id int64) (*storage.Movie, error)
	UpdateMovie(movie *storage.Movie) error
	DeleteMovie(id int64) error
	GetAllMovies(title string, genres []string, filters storage.Filters) ([]*storage.Movie, storage.Metadata, error)
}

type envelope map[string]interface{}

func NewServer(host, port string, idleTimeout, readTimeout, writeTimeout time.Duration, storage Storage) *Server {
	return &Server{
		addr:        net.JoinHostPort(host, port),
		idleTimeout: idleTimeout,
		readTimeout: readTimeout,
		writeTimout: writeTimeout,
		storage:     storage,
	}
}

func (s *Server) Start() error {
	e := echo.New()

	validator, err := NewValidator()
	if err != nil {
		return fmt.Errorf("failed to create validator: %w", err)
	}
	e.Validator = validator
	e.HTTPErrorHandler = customHTTPErrorHandler

	e.Use(middleware.BodyLimit("1M"))
	e.POST("/v1/movies", s.createMovieHandler)
	e.GET("/v1/movies/:id", s.getMovieHandler)
	e.GET("/v1/movies", s.listMoviesHandler)
	e.PATCH("/v1/movies/:id", s.updateMovieHandler)
	e.DELETE("/v1/movies/:id", s.deleteMovieHandler)
	e.GET("/v1/healthcheck", s.healthcheckHandler)

	err = e.Start(s.addr)
	if err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

func (s *Server) healthcheckHandler(c echo.Context) error {
	version := "1.0.0"
	return c.JSON(http.StatusOK, envelope{
		"status": "available",
		"system_info": map[string]string{
			"environment": "development",
			"version":     version,
		},
	})
}

func customHTTPErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}

	if he, ok := err.(*echo.HTTPError); ok {
		err := c.JSON(he.Code, envelope{
			"error": he.Message,
		})
		if err != nil {
			c.Logger().Error(err)
		}
		return
	}

	if err := c.JSON(http.StatusInternalServerError, envelope{
		"error": "internal server error",
	}); err != nil {
		c.Logger().Error(err)
	}
}
