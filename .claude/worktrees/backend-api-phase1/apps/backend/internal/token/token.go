package token

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

func SignAccessToken(userID, role, secret string) (string, error) {
	claims := Claims{
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

func SignRefreshToken(userID, role, secret string) (string, error) {
	claims := Claims{
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

// Verify checks the token's signature and expiry against secret and returns
// its claims. Used for both access and refresh tokens — callers pass
// whichever secret matches the token they are verifying.
func Verify(tokenString, secret string) (*Claims, error) {
	claims := &Claims{}
	parsed, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	if !parsed.Valid {
		return nil, jwt.ErrTokenSignatureInvalid
	}
	return claims, nil
}
