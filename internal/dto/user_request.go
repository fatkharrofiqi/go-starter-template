package dto

type SearchUserRequest struct {
	Name  string `json:"name" validate:"max=100"`
	Email string `json:"email" validate:"max=200"`
	Page  int    `json:"page" validate:"min=1"`
	Size  int    `json:"size" validate:"min=1,max=100"`
}
