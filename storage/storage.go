package storage

import (
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"

	"death-clock/lib/e"
)

type Storage interface {
	SaveUser(ctx context.Context, p *User) error
	IsUserExists(ctx context.Context, p *User) (bool, error)
	GetUserData(ctx context.Context, userName string) (*User, error)
}

var ErrNoSavedPages = errors.New("no saved pages")

type User struct {
	UserName        string
	IsDeathAgeAsked bool
	IsBirthdayAsked bool
	DeathAge        int
	BirthsDay       int
}

func (p User) Hash() (string, error) {
	h := sha1.New()

	if _, err := io.WriteString(h, p.UserName); err != nil {
		return "", e.Wrap("can't calculate hash", err)
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
