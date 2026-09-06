package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims is the payload of a short-lived access token. Access tokens are
// stateless (not stored server-side); revocation is handled at the refresh-token
// layer, so access-token TTL is kept short.
type Claims struct {
	jwt.RegisteredClaims
	Email string `json:"email"`
}

type tokenIssuer struct {
	secret    []byte
	issuer    string
	accessTTL time.Duration
}

// IssueAccessToken mints a signed HS256 JWT for the given user.
func (t tokenIssuer) IssueAccessToken(userID uuid.UUID, email string) (string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(t.accessTTL)
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			Issuer:    t.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			ID:        uuid.NewString(),
		},
		Email: email,
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(t.secret)
	return signed, expiresAt, err
}

// ParseAccessToken validates the signature and standard claims and returns the
// user id encoded in the subject.
func (t tokenIssuer) ParseAccessToken(raw string) (uuid.UUID, *Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return t.secret, nil
	}, jwt.WithIssuer(t.issuer), jwt.WithValidMethods([]string{"HS256"}))
	if err != nil || !token.Valid {
		return uuid.Nil, nil, ErrInvalidToken
	}
	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, nil, ErrInvalidToken
	}
	return userID, claims, nil
}
