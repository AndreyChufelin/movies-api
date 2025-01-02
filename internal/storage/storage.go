package storage

import (
	"errors"
	"fmt"
	"math"
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

type Filters struct {
	Page         int    `validate:"gt=0,max=10000000"`
	PageSize     int    `validate:"gt=0,max=100"`
	Sort         string `validate:"safesort"`
	SortSafelist []string
}

func (f Filters) Offset() int {
	return (f.Page - 1) * f.PageSize
}

type Metadata struct {
	CurrentPage  int `json:"current_page,omitempty"`
	PageSize     int `json:"page_size,omitempty"`
	FirstPage    int `json:"first_page,omitempty"`
	LastPage     int `json:"last_page,omitempty"`
	TotalRecords int `json:"total_records,omitempty"`
}

func NewMetadata(totalRecords, page, pageSize int) Metadata {
	if totalRecords == 0 {
		return Metadata{}
	}
	return Metadata{
		CurrentPage:  page,
		PageSize:     pageSize,
		FirstPage:    1,
		LastPage:     int(math.Ceil(float64(totalRecords) / float64(pageSize))),
		TotalRecords: totalRecords,
	}
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
