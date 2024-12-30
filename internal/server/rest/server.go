package rest

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/AndreyChufelin/movies-api/internal/storage"
	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
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
	CreateMovie(*storage.Movie) error
	GetMovie(int64) (*storage.Movie, error)
	UpdateMovie(*storage.Movie) error
	DeleteMovie(id int64) error
}

type envelope map[string]interface{}

type CustomValidator struct {
	validator *validator.Validate
	trans     ut.Translator
}

func (cv *CustomValidator) Validate(i interface{}) error {
	if err := cv.validator.Struct(i); err != nil {
		var errs []map[string]string
		for _, err := range err.(validator.ValidationErrors) {
			errs = append(errs, map[string]string{
				"field":   err.Field(),
				"message": err.Translate(cv.trans),
			})
		}

		return echo.NewHTTPError(http.StatusUnprocessableEntity, errs)
	}
	return nil
}

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
	en := en.New()
	uni := ut.New(en, en)
	trans, _ := uni.GetTranslator("en")

	validate := validator.New(validator.WithRequiredStructEnabled())
	err := en_translations.RegisterDefaultTranslations(validate, trans)
	if err != nil {
		return fmt.Errorf("failed to register default translations: %w", err)
	}

	e := echo.New()
	e.Validator = &CustomValidator{validator: validate, trans: trans}

	e.Use(middleware.BodyLimit("1M"))
	e.POST("/v1/movies", s.createMovieHandler)
	e.GET("/v1/movies/:id", s.getMovieHandler)
	e.PATCH("/v1/movies/:id", s.updateMovieHandler)
	e.DELETE("/v1/movies/:id", s.deleteMovieHandler)
	e.GET("/v1/healthcheck", s.healthcheckHandler)

	fmt.Println(s.addr)
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
