package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"back/internal/config"
)

var ErrInvalidToken = errors.New("invalid token")

type Claims struct {
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

func GenerateAccessToken(cfg *config.Config, userID int, username, role string) (string, time.Time, error) {
	now := time.Now()
	exp := now.Add(time.Duration(cfg.JWT.AccessTTLMinutes) * time.Minute)

	claims := Claims{
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprintf("%d", userID),
			Issuer:    cfg.JWT.Issuer,
			Audience:  jwt.ClaimStrings(cfg.JWT.Audience),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}

	tkn := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tkn.SignedString([]byte(cfg.JWT.Secret))
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, exp, nil
}

func ParseAndValidate(cfg *config.Config, tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (any, error) {

		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, ErrInvalidToken
		}
		return []byte(cfg.JWT.Secret), nil
	})
	if err != nil {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	v := jwt.NewValidator(jwt.WithLeeway(0))
	if err := v.Validate(claims); err != nil {
		return nil, ErrInvalidToken
	}

	if claims.Issuer != cfg.JWT.Issuer {
		return nil, ErrInvalidToken
	}

	if !audienceMatches([]string(claims.Audience), cfg.JWT.Audience) {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

func audienceMatches(tokenAud []string, allowedAud []string) bool {
	if len(tokenAud) == 0 || len(allowedAud) == 0 {
		return false
	}
	for _, ta := range tokenAud {
		for _, aa := range allowedAud {
			if ta == aa {
				return true
			}
		}
	}
	return false
}
