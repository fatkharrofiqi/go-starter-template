package converter

import (
	"go-starter-template/internal/dto"
	"go-starter-template/internal/model"
)

func UserToResponse(user *model.User) *dto.UserResponse {
	// Collect role-based permissions
	rolePermissionsMap := make(map[string]struct{})
	for _, role := range user.Roles {
		for _, perm := range role.Permissions {
			rolePermissionsMap[perm.Name] = struct{}{}
		}
	}

	// Collect direct permissions
	for _, perm := range user.Permissions {
		rolePermissionsMap[perm.Name] = struct{}{}
	}

	// Convert map to slice
	combinedPermissions := make([]string, 0, len(rolePermissionsMap))
	for perm := range rolePermissionsMap {
		combinedPermissions = append(combinedPermissions, perm)
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
		Permissions: combinedPermissions,
	}
}
