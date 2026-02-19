package services

import (
	"context"
	"errors"
	"strings"

	"back/internal/ent"
	"back/internal/ent/user"

	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidInput = errors.New("invalid input")
var ErrUserAlreadyExists = errors.New("user already exists")
var ErrInvalidCredentials = errors.New("invalid credentials")
var ErrInactiveUser = errors.New("inactive user")

type UsersService struct {
	Client *ent.Client
}

func NewUsersService(client *ent.Client) *UsersService {
	return &UsersService{Client: client}
}

func (s *UsersService) Register(ctx context.Context, username, password, role string) (int, error) {
	username = strings.TrimSpace(username)
	role = strings.TrimSpace(role)

	if username == "" || password == "" {
		return 0, ErrInvalidInput
	}
	if role == "" {
		role = "user"
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return 0, err
	}

	u, err := s.Client.User.
		Create().
		SetUsername(username).
		SetPasswordHash(string(hash)).
		SetRole(role).
		Save(ctx)

	if err != nil {
		if ent.IsConstraintError(err) {
			return 0, ErrUserAlreadyExists
		}
		return 0, err
	}

	return u.ID, nil
}

func (s *UsersService) GetByUsername(ctx context.Context, username string) (*ent.User, error) {
	return s.Client.User.
		Query().
		Where(user.UsernameEQ(username)).
		Only(ctx)
}

func (s *UsersService) VerifyLogin(ctx context.Context, username, password string) (*ent.User, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return nil, ErrInvalidCredentials
	}

	u, err := s.GetByUsername(ctx, username)
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	if !u.IsActive {
		return nil, ErrInactiveUser
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}
	return u, nil
}
