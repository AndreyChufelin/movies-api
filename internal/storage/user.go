package storage

import "errors"

var AnonymousUser = &User{}

type User struct {
	ID        int64
	Activated bool
}

func (u *User) IsAnonymous() bool {
	return u == AnonymousUser
}

var (
	ErrInvalidToken  = errors.New("invalid token")
	ErrInternalError = errors.New("internal error")
)
