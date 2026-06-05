package usecase

import (
	"canerollss/core/domain"
	"canerollss/ports/output"
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthUseCase struct {
	repo   output.UserRepository
	secret []byte
}

func NewAuthUseCase(repo output.UserRepository) *AuthUseCase {
	return &AuthUseCase{
		repo:   repo,
		secret: []byte(os.Getenv("JWT_SECRET")),
	}
}

func (u *AuthUseCase) Register(user *domain.User, rawPassword string) error {
	exists, _ := u.repo.Exists(user.Username)
	if exists {
		return errors.New("el usuario ya existe")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(rawPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.PasswordHash = string(hash)

	return u.repo.Save(user)
}

func (u *AuthUseCase) Login(username, password string) (string, *domain.User, error) {
	user, err := u.repo.GetByUsername(username)
	if err != nil || !user.IsActive {
		return "", nil, errors.New("credenciales inválidas")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return "", nil, errors.New("credenciales inválidas")
	}

	token, err := u.generateToken(user)
	return token, user, err
}

func (u *AuthUseCase) generateToken(user *domain.User) (string, error) {
	claims := jwt.MapClaims{
		"sub":  user.ID,
		"role": user.Role,
		"exp":  time.Now().Add(30 * 24 * time.Hour).Unix(), // Sesión de 30 días
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(u.secret)
}
