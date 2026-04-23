package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"back/internal/ent"
	"back/internal/ent/userqrsession"
)

var ErrQRSessionExpired = errors.New("qr session expired")
var ErrQRSessionNotFound = errors.New("qr session not found")
var ErrQRSessionRevoked = errors.New("qr session revoked")

type QRSessionService struct {
	Client *ent.Client
}

type QRResponse struct {
	Token     string `json:"token"`
	ExpiresIn int    `json:"expires_in"`
}

type QRSessionInfo struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	IssuedAt  time.Time `json:"issued_at"`
	ExpiresAt time.Time `json:"expires_at"`
	IsRevoked bool      `json:"is_revoked"`
}

func NewQRSessionService(client *ent.Client) *QRSessionService {
	return &QRSessionService{Client: client}
}

// GenerateQRSession crea una nueva sesión QR válida por 15 horas
func (s *QRSessionService) GenerateQRSession(ctx context.Context, userID int) (*QRResponse, error) {
	// Generar token aleatorio (64 caracteres hex = 32 bytes)
	token, hash, err := NewQRToken()
	if err != nil {
		return nil, err
	}

	// Definir expiración: ahora + 15 horas
	expiresAt := time.Now().Add(15 * time.Hour)

	// Revocar cualquier QR anterior del mismo usuario que siga activo
	// (opcional, para evitar múltiples QRs simultáneos)
	_, _ = s.Client.UserQRSession.
		Update().
		Where(userqrsession.UserIDEQ(userID)).
		Where(userqrsession.ExpiresAtGT(time.Now())).
		Where(userqrsession.IsRevokedEQ(false)).
		SetIsRevoked(true).
		Save(ctx)

	// Crear nueva sesión
	_, err = s.Client.UserQRSession.
		Create().
		SetUserID(userID).
		SetTokenHash(hash).
		SetIssuedAt(time.Now()).
		SetExpiresAt(expiresAt).
		SetIsRevoked(false).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	// Retornar token plano + segundos hasta expiración
	expiresIn := int(expiresAt.Sub(time.Now()).Seconds())

	return &QRResponse{
		Token:     token,
		ExpiresIn: expiresIn,
	}, nil
}

// ValidateAndGetQRSession valida que el token exista, no esté revocado y no haya expirado
func (s *QRSessionService) ValidateAndGetQRSession(ctx context.Context, tokenPlain string) (*QRSessionInfo, *ent.User, error) {
	// Hashear el token pasado
	hash := HashQRToken(tokenPlain)

	// Buscar la sesión activa
	qr, err := s.Client.UserQRSession.
		Query().
		Where(userqrsession.TokenHashEQ(hash)).
		WithUser().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil, ErrQRSessionNotFound
		}
		return nil, nil, err
	}

	// Validar que no esté revocado
	if qr.IsRevoked {
		return nil, nil, ErrQRSessionRevoked
	}

	// Validar que no haya expirado
	if time.Now().After(qr.ExpiresAt) {
		return nil, nil, ErrQRSessionExpired
	}

	return &QRSessionInfo{
		ID:        qr.ID,
		UserID:    qr.UserID,
		IssuedAt:  qr.IssuedAt,
		ExpiresAt: qr.ExpiresAt,
		IsRevoked: qr.IsRevoked,
	}, qr.Edges.User, nil
}

// RevokeQRSession revoca una sesión QR
func (s *QRSessionService) RevokeQRSession(ctx context.Context, tokenPlain string) error {
	hash := HashQRToken(tokenPlain)

	err := s.Client.UserQRSession.
		Update().
		Where(userqrsession.TokenHashEQ(hash)).
		SetIsRevoked(true).
		Exec(ctx)
	if err != nil {
		return err
	}
	return nil
}

// NewQRToken genera un token QR aleatorio de 64 caracteres hex (32 bytes)
func NewQRToken() (plain string, hash string, err error) {
	b := make([]byte, 32)
	_, err = rand.Read(b)
	if err != nil {
		return "", "", err
	}
	plain = hex.EncodeToString(b)
	hash = HashQRToken(plain)
	return plain, hash, nil
}

// HashQRToken hashea un token QR con SHA256
func HashQRToken(plain string) string {
	h := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(h[:])
}
