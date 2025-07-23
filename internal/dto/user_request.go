package dto

type SearchUserRequest struct {
	Name  string `json:"name" validate:"max=100"`
	Email string `json:"email" validate:"max=200"`
	Page  int    `json:"page" validate:"min=1"`
	Size  int    `json:"size" validate:"min=1,max=100"`
}

func (r *SearchUserRequest) SetDefault() {
	if r.Page == 0 {
		r.Page = 1
	}
	if r.Size == 0 {
		r.Size = 10
	}
}

type CreateUserRequest struct {
	Name     string   `json:"name" validate:"required,min=3,max=100"`
	Email    string   `json:"email" validate:"required,email,max=200"`
	Password string   `json:"password" validate:"required,min=6,max=100"`
	Roles    []string `json:"roles,omitempty" validate:"omitempty"`
}

type UpdateUserRequest struct {
	Name  string   `json:"name" validate:"required,min=3,max=100"`
	Email string   `json:"email" validate:"required,email,max=200"`
	Roles []string `json:"roles,omitempty" validate:"omitempty"`
}
