package storage

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var (
	ErrRecordNotFound       = errors.New("record not found")
	ErrInvalidRuntimeFormat = errors.New("invalid runtime format")
	ErrEditConflict         = errors.New("edit conflict")
)

type Movie struct {
	ID        int64     `db:"id" json:"id"`
	CreatedAt time.Time `db:"created_at" json:"-"`
	Title     string    `db:"title" json:"title" validate:"required,lt=500"`
	Year      int32     `db:"year" json:"year" validate:"required,min=1888,max=2100"`
	Runtime   Runtime   `db:"runtime" json:"runtime" validate:"required,gt=0"`
	Genres    []string  `db:"genres" json:"genres" validate:"required,min=1,max=5"`
	Version   int32     `db:"version" json:"version"`
}

type Runtime int32

func (r Runtime) MarshalJSON() ([]byte, error) {
	jsonValue := fmt.Sprintf("%d mins", r)
	quotedJSONValue := strconv.Quote(jsonValue)
	return []byte(quotedJSONValue), nil
}

func (r *Runtime) UnmarshalJSON(jsonValue []byte) error {
	unquotedJSONValue, err := strconv.Unquote(string(jsonValue))
	if err != nil {
		return ErrInvalidRuntimeFormat
	}
	parts := strings.Split(unquotedJSONValue, " ")
	if len(parts) != 2 || parts[1] != "mins" {
		return ErrInvalidRuntimeFormat
	}
	i, err := strconv.ParseInt(parts[0], 10, 32)
	if err != nil {
		return ErrInvalidRuntimeFormat
	}

	*r = Runtime(i)
	return nil
}
