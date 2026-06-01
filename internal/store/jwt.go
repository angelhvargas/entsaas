package store

import (
	"errors"
	"os"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	jwtSecret   []byte
	jwtSecretMu sync.Once
	ErrJWTSecret = errors.New("JWT_SECRET must be set (≥32 chars)")
)

// lazyLoadSecret loads JWT_SECRET from env on first use.
func lazyLoadSecret() {
	jwtSecretMu.Do(func() {
		s := os.Getenv("JWT_SECRET")
		if len(s) >= 32 {
			jwtSecret = []byte(s)
		}
	})
}

// Claims extends jwt.RegisteredClaims with application-specific fields.
type Claims struct {
	jwt.RegisteredClaims
	UserID string `json:"uid"`
	OrgID  string `json:"oid"`
	Email  string `json:"email"`
	Role   string `json:"role"`
}

// GenerateJWT creates a signed JWT for the given user.
func GenerateJWT(userID, orgID, email, role string, ttl time.Duration) (string, error) {
	lazyLoadSecret()
	if jwtSecret == nil {
		return "", ErrJWTSecret
	}

	now := time.Now()
	issuer := os.Getenv("ENTSAAS_JWT_ISSUER")
	if issuer == "" {
		issuer = "entsaas"
	}
	audience := os.Getenv("ENTSAAS_JWT_AUDIENCE")
	if audience == "" {
		audience = "entsaas-api"
	}

	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			Issuer:    issuer,
			Audience:  jwt.ClaimStrings{audience},
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
		UserID: userID,
		OrgID:  orgID,
		Email:  email,
		Role:   role,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// VerifyJWT parses and validates a JWT string, returning the claims.
func VerifyJWT(tokenStr string) (*Claims, error) {
	lazyLoadSecret()
	if jwtSecret == nil {
		return nil, ErrJWTSecret
	}

	issuer := os.Getenv("ENTSAAS_JWT_ISSUER")
	if issuer == "" {
		issuer = "entsaas"
	}
	audience := os.Getenv("ENTSAAS_JWT_AUDIENCE")
	if audience == "" {
		audience = "entsaas-api"
	}

	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return jwtSecret, nil
	},
		jwt.WithIssuer(issuer),
		jwt.WithAudience(audience),
	)
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}

	return claims, nil
}
