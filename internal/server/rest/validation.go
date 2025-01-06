package rest

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
	"github.com/labstack/echo/v4"
)

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
		var errs []ValidationError
		var verr validator.ValidationErrors
		if ok := errors.As(err, &verr); !ok {
			panic("error must be of type ValidationErrors")
		}
		for _, err := range verr {
			errs = append(errs, ValidationError{
				Field:   err.Field(),
				Message: err.Translate(cv.trans),
			})
		}

		return echo.NewHTTPError(http.StatusUnprocessableEntity, errs)
	}
	return nil
}

func NewValidator() (*CustomValidator, error) {
	en := en.New()
	uni := ut.New(en, en)
	trans, _ := uni.GetTranslator("en")

	validate := validator.New(validator.WithRequiredStructEnabled())
	err := en_translations.RegisterDefaultTranslations(validate, trans)
	if err != nil {
		return nil, fmt.Errorf("failed to register default translations: %w", err)
	}

	err = validate.RegisterTranslation("safesort", trans, func(ut ut.Translator) error {
		return ut.Add("safesort", "{0} must have value from list", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("safesort", fe.Field())

		return t
	})
	if err != nil {
		return nil, err
	}

	err = validate.RegisterValidation("safesort", sortInSafelist)
	if err != nil {
		return nil, fmt.Errorf("failed to register sort validation: %w", err)
	}

	return &CustomValidator{validator: validate, trans: trans}, nil
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
