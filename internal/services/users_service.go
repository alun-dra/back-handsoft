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
var ErrUserEmailAlreadyExists = errors.New("email already exists")
var ErrUserAccessCodeAlreadyExists = errors.New("access code already exists")

type UsersService struct {
	Client *ent.Client
}

func NewUsersService(client *ent.Client) *UsersService {
	return &UsersService{Client: client}
}

type CreateUserInput struct {
	Username     string
	Password     string
	Role         string
	FirstName    string
	LastName     string
	MiddleName   *string
	Email        string
	EmployeeCode *string
	AccessCode   string
	IsActive     *bool
}

type PatchUserInput struct {
	Username     *string
	Password     *string
	Role         *string
	FirstName    *string
	LastName     *string
	MiddleName   *string
	Email        *string
	EmployeeCode *string
	AccessCode   *string
	IsActive     *bool
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

func (s *UsersService) Create(ctx context.Context, in CreateUserInput) (*ent.User, error) {
	in.Username = strings.TrimSpace(in.Username)
	in.Password = strings.TrimSpace(in.Password)
	in.Role = strings.TrimSpace(in.Role)
	in.FirstName = strings.TrimSpace(in.FirstName)
	in.LastName = strings.TrimSpace(in.LastName)
	in.Email = strings.TrimSpace(in.Email)
	in.AccessCode = strings.TrimSpace(in.AccessCode)

	if in.MiddleName != nil {
		v := strings.TrimSpace(*in.MiddleName)
		in.MiddleName = &v
	}
	if in.EmployeeCode != nil {
		v := strings.TrimSpace(*in.EmployeeCode)
		in.EmployeeCode = &v
	}

	if in.Username == "" ||
		in.Password == "" ||
		in.FirstName == "" ||
		in.LastName == "" ||
		in.Email == "" ||
		in.AccessCode == "" {
		return nil, ErrInvalidInput
	}

	if in.Role == "" {
		in.Role = "user"
	}

	existsUsername, err := s.Client.User.
		Query().
		Where(user.UsernameEQ(in.Username)).
		Exist(ctx)
	if err != nil {
		return nil, err
	}
	if existsUsername {
		return nil, ErrUserAlreadyExists
	}

	existsEmail, err := s.Client.User.
		Query().
		Where(user.EmailEQ(in.Email)).
		Exist(ctx)
	if err != nil {
		return nil, err
	}
	if existsEmail {
		return nil, ErrUserEmailAlreadyExists
	}

	existsAccessCode, err := s.Client.User.
		Query().
		Where(user.AccessCodeEQ(in.AccessCode)).
		Exist(ctx)
	if err != nil {
		return nil, err
	}
	if existsAccessCode {
		return nil, ErrUserAccessCodeAlreadyExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), 12)
	if err != nil {
		return nil, err
	}

	create := s.Client.User.
		Create().
		SetUsername(in.Username).
		SetPasswordHash(string(hash)).
		SetRole(in.Role).
		SetFirstName(in.FirstName).
		SetLastName(in.LastName).
		SetEmail(in.Email).
		SetAccessCode(in.AccessCode)

	if in.MiddleName != nil && *in.MiddleName != "" {
		create.SetMiddleName(*in.MiddleName)
	}
	if in.EmployeeCode != nil && *in.EmployeeCode != "" {
		create.SetEmployeeCode(*in.EmployeeCode)
	}
	if in.IsActive != nil {
		create.SetIsActive(*in.IsActive)
	}

	return create.Save(ctx)
}

func (s *UsersService) List(ctx context.Context) ([]*ent.User, error) {
	return s.Client.User.
		Query().
		Order(ent.Asc(user.FieldFirstName), ent.Asc(user.FieldLastName), ent.Asc(user.FieldUsername)).
		All(ctx)
}

func (s *UsersService) GetByID(ctx context.Context, userID int) (*ent.User, error) {
	return s.Client.User.Get(ctx, userID)
}

func (s *UsersService) GetByUsername(ctx context.Context, username string) (*ent.User, error) {
	return s.Client.User.
		Query().
		Where(user.UsernameEQ(strings.TrimSpace(username))).
		Only(ctx)
}

func (s *UsersService) Patch(ctx context.Context, userID int, in PatchUserInput) (*ent.User, error) {
	if userID <= 0 {
		return nil, ErrInvalidInput
	}
	if in.Username == nil &&
		in.Password == nil &&
		in.Role == nil &&
		in.FirstName == nil &&
		in.LastName == nil &&
		in.MiddleName == nil &&
		in.Email == nil &&
		in.EmployeeCode == nil &&
		in.AccessCode == nil &&
		in.IsActive == nil {
		return nil, ErrInvalidInput
	}

	u, err := s.Client.User.Get(ctx, userID)
	if err != nil {
		return nil, err
	}

	upd := u.Update()

	if in.Username != nil {
		v := strings.TrimSpace(*in.Username)
		if v == "" {
			return nil, ErrInvalidInput
		}

		exists, err := s.Client.User.
			Query().
			Where(user.UsernameEQ(v)).
			Where(user.IDNEQ(u.ID)).
			Exist(ctx)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, ErrUserAlreadyExists
		}

		upd.SetUsername(v)
	}

	if in.Password != nil {
		v := strings.TrimSpace(*in.Password)
		if v == "" {
			return nil, ErrInvalidInput
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(v), 12)
		if err != nil {
			return nil, err
		}
		upd.SetPasswordHash(string(hash))
	}

	if in.Role != nil {
		v := strings.TrimSpace(*in.Role)
		if v == "" {
			return nil, ErrInvalidInput
		}
		upd.SetRole(v)
	}

	if in.FirstName != nil {
		v := strings.TrimSpace(*in.FirstName)
		if v == "" {
			return nil, ErrInvalidInput
		}
		upd.SetFirstName(v)
	}

	if in.LastName != nil {
		v := strings.TrimSpace(*in.LastName)
		if v == "" {
			return nil, ErrInvalidInput
		}
		upd.SetLastName(v)
	}

	if in.MiddleName != nil {
		v := strings.TrimSpace(*in.MiddleName)
		if v == "" {
			upd.ClearMiddleName()
		} else {
			upd.SetMiddleName(v)
		}
	}

	if in.Email != nil {
		v := strings.TrimSpace(*in.Email)
		if v == "" {
			return nil, ErrInvalidInput
		}

		exists, err := s.Client.User.
			Query().
			Where(user.EmailEQ(v)).
			Where(user.IDNEQ(u.ID)).
			Exist(ctx)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, ErrUserEmailAlreadyExists
		}

		upd.SetEmail(v)
	}

	if in.EmployeeCode != nil {
		v := strings.TrimSpace(*in.EmployeeCode)
		if v == "" {
			upd.ClearEmployeeCode()
		} else {
			upd.SetEmployeeCode(v)
		}
	}

	if in.AccessCode != nil {
		v := strings.TrimSpace(*in.AccessCode)
		if v == "" {
			return nil, ErrInvalidInput
		}

		exists, err := s.Client.User.
			Query().
			Where(user.AccessCodeEQ(v)).
			Where(user.IDNEQ(u.ID)).
			Exist(ctx)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, ErrUserAccessCodeAlreadyExists
		}

		upd.SetAccessCode(v)
	}

	if in.IsActive != nil {
		upd.SetIsActive(*in.IsActive)
	}

	return upd.Save(ctx)
}

func (s *UsersService) Delete(ctx context.Context, userID int) error {
	if userID <= 0 {
		return ErrInvalidInput
	}
	return s.Client.User.DeleteOneID(userID).Exec(ctx)
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
