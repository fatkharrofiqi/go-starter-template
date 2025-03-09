package converter

import (
	"go-starter-template/internal/dto"
	"go-starter-template/internal/model"
)

func UserToResponse(user *model.User) *dto.UserResponse {
	return &dto.UserResponse{
		UUID:      user.UUID,
		Name:      user.Name,
		Email:     user.Email,
		CreatedAt: user.CreatedAt.Unix(),
		UpdatedAt: user.UpdatedAt.Unix(),
	}
}
