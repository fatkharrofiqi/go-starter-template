package service

import (
	"errors"
	"go-starter-template/internal/config/env"
	"go-starter-template/internal/utils/apperrors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
)

type CsrfClaims struct {
	Path string `json:"path"`
	jwt.RegisteredClaims
}

type CsrfService struct {
	Config *env.Config
	Log    *logrus.Logger
}

func NewCsrfService(config *env.Config, log *logrus.Logger) *CsrfService {
	return &CsrfService{
		Config: config,
		Log:    log,
	}
}

func (csrfService *CsrfService) GenerateCsrfToken(path string) (string, error) {
	claims := CsrfClaims{
		Path: path,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(csrfService.Config.JWT.CsrfTokenExpiration * time.Second)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        path + time.Now().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(csrfService.Config.JWT.CsrfSecret))
}

func (csrfService *CsrfService) ValidateCsrfToken(csrfToken string) (*CsrfClaims, error) {
	claims := &CsrfClaims{}
	token, err := jwt.ParseWithClaims(csrfToken, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(csrfService.Config.JWT.CsrfSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, apperrors.ErrInvalidToken
	}

	return claims, nil
}
