package services

import (
	"context"
	"errors"
	"time"

	"back/internal/auth"
	"back/internal/config"
	"back/internal/ent"
	"back/internal/ent/refreshtoken"
	"back/internal/ent/user"
)

var ErrInvalidRefreshToken = errors.New("invalid refresh token")

type TokenService struct {
	Cfg    *config.Config
	Client *ent.Client
}

type TokenPair struct {
	AccessToken  string
	AccessExp    time.Time
	RefreshToken string
	RefreshExp   time.Time
}

type SessionInfo struct {
	ID        int       `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

func NewTokenService(cfg *config.Config, client *ent.Client) *TokenService {
	return &TokenService{Cfg: cfg, Client: client}
}

func (s *TokenService) IssueForUser(ctx context.Context, u *ent.User) (*TokenPair, error) {
	access, accessExp, err := auth.GenerateAccessToken(s.Cfg, u.ID, u.Username, u.Role)
	if err != nil {
		return nil, err
	}

	plain, hash, err := auth.NewRefreshToken()
	if err != nil {
		return nil, err
	}

	refreshExp := time.Now().Add(time.Duration(s.Cfg.JWT.RefreshTTLDays) * 24 * time.Hour)

	tx, err := s.Client.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.RefreshToken.
		Create().
		SetTokenHash(hash).
		SetExpiresAt(refreshExp).
		SetUserID(u.ID).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	// Limitar a máximo 3 sesiones activas (dentro de la TX)
	if err := enforceMaxSessionsTx(ctx, tx, u.ID, 3); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  access,
		AccessExp:    accessExp,
		RefreshToken: plain,
		RefreshExp:   refreshExp,
	}, nil
}

// Rotación: revoca el refresh actual y entrega uno nuevo
func (s *TokenService) Rotate(ctx context.Context, refreshPlain string) (*TokenPair, *ent.User, error) {
	hash := auth.HashRefreshToken(refreshPlain)

	rt, err := s.Client.RefreshToken.
		Query().
		Where(refreshtoken.TokenHashEQ(hash)).
		Where(refreshtoken.RevokedAtIsNil()).
		Only(ctx)
	if err != nil {
		return nil, nil, ErrInvalidRefreshToken
	}

	if time.Now().After(rt.ExpiresAt) {
		return nil, nil, ErrInvalidRefreshToken
	}

	// cargar user
	u, err := rt.QueryUser().Only(ctx)
	if err != nil {
		return nil, nil, ErrInvalidRefreshToken
	}
	if !u.IsActive {
		return nil, nil, ErrInvalidRefreshToken
	}

	now := time.Now()
	if _, err := s.Client.RefreshToken.
		UpdateOneID(rt.ID).
		SetRevokedAt(now).
		Save(ctx); err != nil {
		return nil, nil, err
	}

	pair, err := s.IssueForUser(ctx, u)
	if err != nil {
		return nil, nil, err
	}
	return pair, u, nil
}

func (s *TokenService) Revoke(ctx context.Context, refreshPlain string) error {
	hash := auth.HashRefreshToken(refreshPlain)

	rt, err := s.Client.RefreshToken.
		Query().
		Where(refreshtoken.TokenHashEQ(hash)).
		Where(refreshtoken.RevokedAtIsNil()).
		Only(ctx)
	if err != nil {
		// por seguridad: no revelamos si existe o no
		return ErrInvalidRefreshToken
	}

	// si ya expiró, lo tratamos igual como inválido
	if time.Now().After(rt.ExpiresAt) {
		return ErrInvalidRefreshToken
	}

	now := time.Now()
	_, err = s.Client.RefreshToken.
		UpdateOneID(rt.ID).
		SetRevokedAt(now).
		Save(ctx)
	return err
}

func (s *TokenService) RevokeAllForUser(ctx context.Context, userID int) (int, error) {
	now := time.Now()
	n, err := s.Client.RefreshToken.
		Update().
		Where(refreshtoken.HasUserWith(user.IDEQ(userID))).
		Where(refreshtoken.RevokedAtIsNil()).
		SetRevokedAt(now).
		Save(ctx)
	return n, err
}

func (s *TokenService) enforceMaxSessions(ctx context.Context, userID int, max int) error {
	if max <= 0 {
		return nil
	}

	// Traemos todos los refresh activos del usuario, ordenados por created_at DESC (más nuevos primero)
	active, err := s.Client.RefreshToken.
		Query().
		Where(refreshtoken.HasUserWith(user.IDEQ(userID))).
		Where(refreshtoken.RevokedAtIsNil()).
		Order(ent.Desc(refreshtoken.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return err
	}

	if len(active) <= max {
		return nil
	}

	// Los que sobran (desde el índice max hacia adelante) se revocan
	var revokeIDs []int
	for _, rt := range active[max:] {
		revokeIDs = append(revokeIDs, rt.ID)
	}

	now := time.Now()
	_, err = s.Client.RefreshToken.
		Update().
		Where(refreshtoken.IDIn(revokeIDs...)).
		SetRevokedAt(now).
		Save(ctx)

	return err
}

func enforceMaxSessionsTx(ctx context.Context, tx *ent.Tx, userID int, max int) error {
	if max <= 0 {
		return nil
	}

	active, err := tx.RefreshToken.
		Query().
		Where(refreshtoken.HasUserWith(user.IDEQ(userID))).
		Where(refreshtoken.RevokedAtIsNil()).
		Order(ent.Desc(refreshtoken.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return err
	}

	if len(active) <= max {
		return nil
	}

	var revokeIDs []int
	for _, rt := range active[max:] {
		revokeIDs = append(revokeIDs, rt.ID)
	}

	now := time.Now()
	_, err = tx.RefreshToken.
		Update().
		Where(refreshtoken.IDIn(revokeIDs...)).
		SetRevokedAt(now).
		Save(ctx)

	return err
}

func (s *TokenService) ListActiveSessions(ctx context.Context, userID int) ([]SessionInfo, error) {
	rows, err := s.Client.RefreshToken.
		Query().
		Where(refreshtoken.HasUserWith(user.IDEQ(userID))).
		Where(refreshtoken.RevokedAtIsNil()).
		Order(ent.Desc(refreshtoken.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]SessionInfo, 0, len(rows))
	for _, rt := range rows {
		out = append(out, SessionInfo{
			ID:        rt.ID,
			CreatedAt: rt.CreatedAt,
			ExpiresAt: rt.ExpiresAt,
		})
	}
	return out, nil
}
