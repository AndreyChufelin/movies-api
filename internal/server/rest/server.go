package rest

import (
	"errors"
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

func NewServer(host, port string, idleTimeout, readTimeout, writeTimeout time.Duration) *Server {
	return &Server{
		addr:        net.JoinHostPort(host, port),
		idleTimeout: idleTimeout,
		readTimeout: readTimeout,
		writeTimout: writeTimeout,
	}
}

func (s *Server) Start() error {
	en := en.New()
	uni := ut.New(en, en)
	trans, _ := uni.GetTranslator("en")

	validate := validator.New(validator.WithRequiredStructEnabled())
	en_translations.RegisterDefaultTranslations(validate, trans)

	e := echo.New()
	e.Validator = &CustomValidator{validator: validate, trans: trans}

	e.Use(middleware.BodyLimit("1M"))
	e.POST("/v1/movies", s.createMovieHandler)
	e.GET("/v1/movies/:id", s.getMovieHandler)
	e.GET("/v1/healthcheck", s.healthcheckHandler)

	fmt.Println(s.addr)
	err := e.Start(s.addr)
	if err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

func (s *Server) createMovieHandler(c echo.Context) error {
	var input struct {
		Title   string          `json:"title"`
		Year    int32           `json:"year"`
		Runtime storage.Runtime `json:"runtime"`
		Genres  []string        `json:"genres"`
	}
	err := c.Bind(&input)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "bad request")
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

	movie := storage.Movie{
		ID:        id,
		CreatedAt: time.Now(),
		Title:     "Casablanca",
		Runtime:   102,
		Genres:    []string{"drama", "romance", "war"},
		Version:   1,
	}

	return c.JSON(http.StatusOK, envelope{
		"movie": movie,
	})
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
