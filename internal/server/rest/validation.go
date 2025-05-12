package rest

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/AndreyChufelin/movies-api/pkg/validator"
	"github.com/labstack/echo/v4"
)

type CustomValidator struct {
	validator *validator.Validator
}

func (cv *CustomValidator) Validate(i interface{}) error {
	err := cv.validator.Validate(i)
	var validationErrs *validator.ValidationErrors
	if errors.As(err, &validationErrs) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, validationErrs.Errors)
	}
	return err
}

func NewValidator() (*CustomValidator, error) {
	v, err := validator.NewValidator()
	if err != nil {
		return nil, fmt.Errorf("error creating validator %w", err)
	}

	return &CustomValidator{
		validator: v,
	}, nil
}
