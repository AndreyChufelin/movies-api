package rest

import (
	"fmt"
	"net"
	"net/http"
	"reflect"
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
	CreateMovie(movie *storage.Movie) error
	GetMovie(id int64) (*storage.Movie, error)
	UpdateMovie(movie *storage.Movie) error
	DeleteMovie(id int64) error
	GetAllMovies(title string, genres []string, filters storage.Filters) ([]*storage.Movie, storage.Metadata, error)
}

type envelope map[string]interface{}

type CustomValidator struct {
	validator *validator.Validate
	trans     ut.Translator
}

type ValidationError struct {
	Field   string
	Message string
}

func (cv *CustomValidator) Validate(i interface{}) error {
	if err := cv.validator.Struct(i); err != nil {
		fmt.Println("errs", err)
		var errs []ValidationError
		for _, err := range err.(validator.ValidationErrors) {
			errs = append(errs, ValidationError{
				Field:   err.Field(),
				Message: err.Translate(cv.trans),
			})
		}

		return echo.NewHTTPError(http.StatusUnprocessableEntity, errs)
	}
	return nil
}

func sortInSafelist(fl validator.FieldLevel) bool {
	sortField := fl.Field().String()
	safelistField := fl.Parent().FieldByName("SortSafelist")

	if safelistField.Kind() == reflect.Slice {
		for i := 0; i < safelistField.Len(); i++ {
			if sortField == safelistField.Index(i).String() {
				return true
			}
		}
	}
	return false
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
	validate.RegisterTranslation("safesort", trans, func(ut ut.Translator) error {
		return ut.Add("safesort", "{0} must have value from list", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("safesort", fe.Field())

		return t
	})
	err = validate.RegisterValidation("safesort", sortInSafelist)
	if err != nil {
		return fmt.Errorf("failed to register sort validation: %w", err)
	}

	e := echo.New()
	e.Validator = &CustomValidator{validator: validate, trans: trans}

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
