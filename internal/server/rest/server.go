package rest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/AndreyChufelin/movies-api/internal/auth"
	"github.com/AndreyChufelin/movies-api/internal/logger"
	"github.com/AndreyChufelin/movies-api/internal/storage"
	"github.com/AndreyChufelin/movies-api/pkg/validator"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"
)

type Server struct {
	e              *echo.Echo
	addr           string
	log            *logger.Logger
	idleTimeout    time.Duration
	readTimeout    time.Duration
	writeTimout    time.Duration
	storage        Storage
	limit          int
	limiterEnabled bool
	auth           *auth.Auth
	corsOrigins    []string
}

type Storage interface {
	CreateMovie(movie *storage.Movie) error
	GetMovie(id int64) (*storage.Movie, error)
	UpdateMovie(movie *storage.Movie) error
	DeleteMovie(id int64) error
	GetAllMovies(title string, genres []string, filters storage.Filters) ([]*storage.Movie, storage.Metadata, error)
}

type envelope map[string]interface{}

type AuthContext struct {
	echo.Context
}

func (c *AuthContext) GetUser() *storage.User {
	user, ok := c.Get("user").(*storage.User)
	if !ok {
		return nil
	}
	return user
}

func (c *AuthContext) SetUser(user *storage.User) {
	c.Set("user", user)
}

func NewServer(
	logger *logger.Logger,
	auth *auth.Auth,
	host,
	port string,
	idleTimeout,
	readTimeout,
	writeTimeout time.Duration,
	storage Storage,
	limit int,
	limiterEnabled bool,
	corsOrigins []string,
) *Server {
	return &Server{
		log:            logger,
		auth:           auth,
		addr:           net.JoinHostPort(host, port),
		idleTimeout:    idleTimeout,
		readTimeout:    readTimeout,
		writeTimout:    writeTimeout,
		storage:        storage,
		limit:          limit,
		limiterEnabled: limiterEnabled,
		corsOrigins:    corsOrigins,
	}
}

func (s *Server) Start() error {
	e := echo.New()

	validator, err := NewValidator()
	if err != nil {
		return fmt.Errorf("failed to create validator: %w", err)
	}
	e.Binder = &CustomBinder{}
	e.Validator = validator
	e.HTTPErrorHandler = customHTTPErrorHandler

	if s.limiterEnabled {
		e.Use(
			middleware.RateLimiter(
				middleware.NewRateLimiterMemoryStore(rate.Limit(s.limit)),
			),
		)
	}
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
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: s.corsOrigins,
		AllowHeaders: []string{"Authorization", "Content-Type"},
	}))
	e.Use(middleware.BodyLimit("1M"))
	e.Use(s.authMiddleware)
	m := e.Group("/v1/movies")
	// m.Use(s.requireActivatedUser)
	m.POST("", s.requirePermission("movies:write", s.createMovieHandler))
	m.GET("/:id", s.requirePermission("movies:read", s.getMovieHandler))
	m.GET("", s.requirePermission("movies:read", s.listMoviesHandler))
	m.PATCH("/:id", s.requirePermission("movies:write", s.updateMovieHandler))
	m.DELETE("/:id", s.requirePermission("movies:write", s.deleteMovieHandler))
	e.GET("/v1/healthcheck", s.healthcheckHandler)

	s.e = e
	s.log.Info("starting REST server")
	err = e.Start(s.addr)
	if err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	s.log.Info("shutting down rest server")
	if err := s.e.Shutdown(ctx); err != nil {
		return err
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

func (s *Server) authMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		cc := &AuthContext{c}
		cc.Response().Header().Set("Vary", "Authorization")
		authHeader := cc.Request().Header.Get("Authorization")
		if authHeader == "" {
			s.log.Info("set anonymous user")
			cc.Set("user", storage.AnonymousUser)
			return next(cc)
		}

		headerParts := strings.Split(authHeader, " ")
		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			s.log.Warn("token must be bearer")
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
		}
		token := headerParts[1]

		user, err := s.auth.Verify(context.TODO(), token)
		if err != nil {
			if errors.Is(err, storage.ErrInvalidToken) {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
		}

		s.log.Info("authenticate user", "user_id", user.ID)
		cc.Set("user", user)
		return next(cc)
	}
}

func (s *Server) requireAuthenticatedUser(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		cc := AuthContext{c}
		user := cc.GetUser()

		if user.IsAnonymous() {
			return echo.NewHTTPError(http.StatusUnauthorized, "you must be authenticated to access this resource")
		}

		return next(cc)
	}
}

func (s *Server) requireActivatedUser(next echo.HandlerFunc) echo.HandlerFunc {
	fn := func(c echo.Context) error {
		cc := AuthContext{c}
		user := cc.GetUser()

		if !user.Activated {
			return echo.NewHTTPError(http.StatusForbidden, "your account must be activated")
		}
		return next(cc)
	}
	return s.requireAuthenticatedUser(fn)
}

func (s *Server) requirePermission(code string, next echo.HandlerFunc) echo.HandlerFunc {
	fn := func(c echo.Context) error {
		cc := AuthContext{c}
		user := cc.GetUser()
		if !user.IncludePermission(code) {
			return echo.NewHTTPError(http.StatusForbidden, "not permitted")
		}

		return next(cc)
	}

	return s.requireActivatedUser(fn)
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

type CustomBinder struct{}

func (cb *CustomBinder) Bind(i interface{}, c echo.Context) (err error) {
	db := new(echo.DefaultBinder)
	if err := db.Bind(i, c); err != nil {
		var jerr *json.UnmarshalTypeError
		if ok := errors.As(err, &jerr); ok {
			return echo.NewHTTPError(http.StatusBadRequest, validator.ValidationError{
				Field:   jerr.Field,
				Message: "invalid value",
			})
		}
		if errors.Is(err, storage.ErrInvalidRuntimeFormat) {
			return echo.NewHTTPError(http.StatusBadRequest, validator.ValidationError{
				Field:   "runtime",
				Message: "invalid value",
			},
			)
		}
		return echo.NewHTTPError(http.StatusBadRequest, "bad request")
	}

	return
}
