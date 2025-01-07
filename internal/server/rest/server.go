package rest

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/AndreyChufelin/movies-api/internal/logger"
	"github.com/AndreyChufelin/movies-api/internal/storage"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type Server struct {
	addr        string
	log         *logger.Logger
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

func NewServer(
	logger *logger.Logger,
	host,
	port string,
	idleTimeout,
	readTimeout,
	writeTimeout time.Duration,
	storage Storage,
) *Server {
	return &Server{
		log:         logger,
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

	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus:   true,
		LogURI:      true,
		LogError:    true,
		HandleError: true,
		LogValuesFunc: func(_ echo.Context, v middleware.RequestLoggerValues) error {
			if v.Error == nil {
				s.log.LogAttrs(context.Background(), slog.LevelInfo, "REQUEST",
					slog.String("uri", v.URI),
					slog.Int("status", v.Status),
				)
			} else {
				s.log.LogAttrs(context.Background(), slog.LevelError, "REQUEST_ERROR",
					slog.String("uri", v.URI),
					slog.Int("status", v.Status),
					slog.String("err", v.Error.Error()),
				)
			}
			return nil
		},
	}))
	e.Use(middleware.BodyLimit("1M"))
	e.POST("/v1/movies", s.createMovieHandler)
	e.GET("/v1/movies/:id", s.getMovieHandler)
	e.GET("/v1/movies", s.listMoviesHandler)
	e.PATCH("/v1/movies/:id", s.updateMovieHandler)
	e.DELETE("/v1/movies/:id", s.deleteMovieHandler)
	e.GET("/v1/healthcheck", s.healthcheckHandler)

	s.log.Info("starting REST server")
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

	var he *echo.HTTPError
	if ok := errors.As(err, &he); ok {
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
