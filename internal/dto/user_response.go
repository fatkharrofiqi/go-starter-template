package dto

type UserResponse struct {
	UUID        string         `json:"uuid,omitempty"`
	Name        string         `json:"name,omitempty"`
	Email       string         `json:"email,omitempty"`
	CreatedAt   int64          `json:"created_at,omitempty"`
	UpdatedAt   int64          `json:"updated_at,omitempty"`
	Roles       []RoleResponse `json:"roles,omitempty"`
	Permissions []string       `json:"permissions,omitempty"`
}

type RoleResponse struct {
	Name        string   `json:"name,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
}
