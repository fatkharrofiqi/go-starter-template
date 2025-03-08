package dto

import "go-starter-template/internal/model"

type AuthResponse struct {
	Token string     `json:"token"`
	User  model.User `json:"user"`
}

type LoginResponse struct {
	TokenResponse
}

type RefreshTokenResponse struct {
	TokenResponse
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}
