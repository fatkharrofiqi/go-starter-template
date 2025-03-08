package dto

type UserResponse struct {
	UID       string `json:"uid,omitempty"`
	Name      string `json:"name,omitempty"`
	Email     string `json:"email,omitempty"`
	CreatedAt int64  `json:"created_at,omitempty"`
	UpdatedAt int64  `json:"updated_at,omitempty"`
}
