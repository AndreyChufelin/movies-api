package storage

import "errors"

var AnonymousUser = &User{}

type User struct {
	ID          int64
	Activated   bool
	Permissions []string
}

func (u *User) IsAnonymous() bool {
	return u == AnonymousUser
}

func (u *User) IncludePermission(code string) bool {
	for _, p := range u.Permissions {
		if code == p {
			return true
		}
	}
	return false
}

var (
	ErrInvalidToken  = errors.New("invalid token")
	ErrInternalError = errors.New("internal error")
)
