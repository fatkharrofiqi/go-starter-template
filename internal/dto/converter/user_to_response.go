package converter

import (
	"go-starter-template/internal/dto"
	"go-starter-template/internal/model"
)

func UserToResponse(user *model.User) *dto.UserResponse {
    // Collect direct user permissions only (do not combine with role permissions)
    directPermissions := make([]string, len(user.Permissions))
    for i, perm := range user.Permissions {
        directPermissions[i] = perm.Name
    }

	// Convert roles to RoleResponse
	roles := make([]dto.RoleResponse, len(user.Roles))
	for i, role := range user.Roles {
		permissions := make([]string, len(role.Permissions))
		for j, perm := range role.Permissions {
			permissions[j] = perm.Name
		}
		roles[i] = dto.RoleResponse{
			Name:        role.Name,
			Permissions: permissions,
		}
	}

    return &dto.UserResponse{
        UUID:        user.UUID,
        Name:        user.Name,
        Email:       user.Email,
        CreatedAt:   user.CreatedAt.Unix(),
        UpdatedAt:   user.UpdatedAt.Unix(),
        Roles:       roles,
        Permissions: directPermissions,
    }
}
