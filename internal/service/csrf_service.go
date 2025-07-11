package service

import (
	"errors"
	"go-starter-template/internal/config/env"
	"go-starter-template/internal/utils/errcode"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
)

type CsrfClaims struct {
	Path string `json:"path"`
	jwt.RegisteredClaims
}

type CsrfService struct {
	config *env.Config
	log    *logrus.Logger
}

func NewCsrfService(config *env.Config, log *logrus.Logger) *CsrfService {
	return &CsrfService{
		config: config,
		log:    log,
	}
}

func (c *CsrfService) GenerateCsrfToken(path string) (string, error) {
	claims := CsrfClaims{
		Path: path,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(c.config.GetCsrfTokenExpiration())),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        path + time.Now().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(c.config.GetCsrfSecret()))
}

func (c *CsrfService) ValidateCsrfToken(csrfToken string) (*CsrfClaims, error) {
	claims := &CsrfClaims{}
	token, err := jwt.ParseWithClaims(csrfToken, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(c.config.GetCsrfSecret()), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errcode.ErrInvalidToken
	}

	return claims, nil
}
