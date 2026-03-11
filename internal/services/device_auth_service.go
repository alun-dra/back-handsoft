package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"back/internal/ent"
	"back/internal/ent/device"
)

var ErrInvalidDeviceCredentials = errors.New("invalid device credentials")

type DeviceAuthService struct {
	Client *ent.Client
	Tokens *TokenService
}

func NewDeviceAuthService(client *ent.Client, tokens *TokenService) *DeviceAuthService {
	return &DeviceAuthService{
		Client: client,
		Tokens: tokens,
	}
}

type DeviceLoginResult struct {
	AccessToken   string
	ExpiresAt     time.Time
	ExpiresIn     int
	TokenType     string
	Role          string
	Username      string
	DeviceID      int
	AccessPointID int
	Direction     string
	Name          string
}

func (s *DeviceAuthService) Login(ctx context.Context, username, password string) (*DeviceLoginResult, error) {
	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)

	if username == "" || password == "" {
		return nil, ErrInvalidDeviceCredentials
	}

	d, err := s.Client.Device.
		Query().
		Where(device.UsernameEQ(username)).
		Only(ctx)
	if err != nil {
		return nil, ErrInvalidDeviceCredentials
	}

	if !d.IsActive {
		return nil, ErrInvalidDeviceCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(d.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidDeviceCredentials
	}

	now := time.Now()
	if _, err := s.Client.Device.
		UpdateOneID(d.ID).
		SetLastLoginAt(now).
		Save(ctx); err != nil {
		return nil, err
	}

	pair, err := s.Tokens.IssueForDevice(ctx, d)
	if err != nil {
		return nil, err
	}

	expiresIn := int(time.Until(pair.AccessExp).Seconds())
	if expiresIn < 0 {
		expiresIn = 0
	}

	return &DeviceLoginResult{
		AccessToken:   pair.AccessToken,
		ExpiresAt:     pair.AccessExp,
		ExpiresIn:     expiresIn,
		TokenType:     "Bearer",
		Role:          d.Role,
		Username:      d.Username,
		DeviceID:      d.ID,
		AccessPointID: d.AccessPointID,
		Direction:     d.Direction,
		Name:          d.Name,
	}, nil
}
