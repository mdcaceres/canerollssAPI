package output

import (
	"canerollss/core/domain"
)

type UserRepository interface {
	GetByUsername(username string) (*domain.User, error)
	Save(user *domain.User) error
	Exists(username string) (bool, error)
	Update(user *domain.User) error
}
