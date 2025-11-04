package repository

import (
	"context"
	"database/sql"
	"fmt"
	"go-starter-template/internal/dto"
	"go-starter-template/internal/model"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type UserRepository struct {
	*Repository
	tracer trace.Tracer
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{Repository: &Repository{db}, tracer: otel.Tracer("UserRepository")}
}

func (r *UserRepository) CountByEmail(ctx context.Context, email string) (int64, error) {
	spanCtx, span := r.tracer.Start(ctx, "UserRepository.CountByEmail")
	defer span.End()
	var total int64
	err := r.getExecutor(spanCtx).QueryRowContext(spanCtx, `SELECT COUNT(*) FROM users WHERE email = $1`, email).Scan(&total)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "count by email failed")
		return total, err
	}
	return total, nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, user *model.User, email string) error {
	spanCtx, span := r.tracer.Start(ctx, "UserRepository.FindByEmail")
	defer span.End()
	row := r.getExecutor(spanCtx).QueryRowContext(spanCtx, `SELECT uuid, name, email, password, created_at, updated_at FROM users WHERE email = $1 LIMIT 1`, email)
	if err := row.Scan(&user.UUID, &user.Name, &user.Email, &user.Password, &user.CreatedAt, &user.UpdatedAt); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "find by email failed")
		return err
	}
	return nil
}

func (r *UserRepository) FindByUUID(ctx context.Context, user *model.User, uuid string) error {
	spanCtx, span := r.tracer.Start(ctx, "UserRepository.FindByUUID")
	defer span.End()
	row := r.getExecutor(spanCtx).QueryRowContext(spanCtx, `SELECT uuid, name, email, password, created_at, updated_at FROM users WHERE uuid = $1 LIMIT 1`, uuid)
	if err := row.Scan(&user.UUID, &user.Name, &user.Email, &user.Password, &user.CreatedAt, &user.UpdatedAt); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "find by uuid failed")
		return err
	}

	// Load roles
	rolesRows, err := r.getExecutor(spanCtx).QueryContext(spanCtx, `	
        SELECT r.uuid, r.name
        FROM roles r
        INNER JOIN user_roles ur ON ur.role_uuid = r.uuid
        WHERE ur.user_uuid = $1
    `, uuid)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "query roles failed")
		return err
	}
	defer rolesRows.Close()
	var roles []model.Role
	for rolesRows.Next() {
		var role model.Role
		if err = rolesRows.Scan(&role.UUID, &role.Name); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "scan role failed")
			return err
		}
		roles = append(roles, role)
	}
	user.Roles = roles

	// Load direct permissions
	permRows, err := r.getExecutor(spanCtx).QueryContext(spanCtx, `
        SELECT p.uuid, p.name
        FROM permissions p
        INNER JOIN user_permissions up ON up.permission_uuid = p.uuid
        WHERE up.user_uuid = $1
    `, uuid)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "query direct permissions failed")
		return err
	}
	defer permRows.Close()
    var permissions []model.Permission
    for permRows.Next() {
        var perm model.Permission
        if err = permRows.Scan(&perm.UUID, &perm.Name); err != nil {
            span.RecordError(err)
            span.SetStatus(codes.Error, "scan direct permission failed")
            return err
        }
        permissions = append(permissions, perm)
    }
    // Assign only direct user permissions to user.Permissions
    user.Permissions = permissions

    // Load role-based permissions and attach to each role
    rolePermRows, err := r.getExecutor(spanCtx).QueryContext(spanCtx, `
        SELECT rp.role_uuid, p.uuid, p.name
        FROM permissions p
        INNER JOIN role_permissions rp ON rp.permission_uuid = p.uuid
        INNER JOIN user_roles ur ON ur.role_uuid = rp.role_uuid
        WHERE ur.user_uuid = $1
    `, uuid)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, "query role permissions failed")
        return err
    }
    defer rolePermRows.Close()
    // Map role UUID to index for quick attachment
    roleIndex := make(map[string]int, len(user.Roles))
    for i := range user.Roles {
        roleIndex[user.Roles[i].UUID] = i
    }
    for rolePermRows.Next() {
        var roleUUID string
        var perm model.Permission
        if err := rolePermRows.Scan(&roleUUID, &perm.UUID, &perm.Name); err != nil {
            span.RecordError(err)
            span.SetStatus(codes.Error, "scan role permission failed")
            return err
        }
        if idx, ok := roleIndex[roleUUID]; ok {
            user.Roles[idx].Permissions = append(user.Roles[idx].Permissions, perm)
        }
    }

	return nil
}

func (r *UserRepository) Search(ctx context.Context, request *dto.SearchUserRequest) ([]*model.User, int64, error) {
	spanCtx, span := r.tracer.Start(ctx, "UserRepository.Search")
	defer span.End()
	request.SetDefault()

	// Build filters
	where := ""
	args := []interface{}{}
	if request.Name != "" {
		where += " AND name ILIKE $" + fmt.Sprintf("%d", len(args)+1)
		args = append(args, "%"+request.Name+"%")
	}
	if request.Email != "" {
		where += " AND email ILIKE $" + fmt.Sprintf("%d", len(args)+1)
		args = append(args, "%"+request.Email+"%")
	}
	if where != "" {
		where = "WHERE" + where[4:]
	}

	// Count
	countQuery := "SELECT COUNT(*) FROM users " + where
	var total int64
	if err := r.getExecutor(spanCtx).QueryRowContext(spanCtx, countQuery, args...).Scan(&total); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "count search failed")
		return nil, 0, err
	}

	// Data
	offset := (request.Page - 1) * request.Size
	dataQuery := "SELECT uuid, name, email, created_at, updated_at FROM users " + where + " ORDER BY created_at DESC OFFSET $" + fmt.Sprintf("%d", len(args)+1) + " LIMIT $" + fmt.Sprintf("%d", len(args)+2)
	args = append(args, offset, request.Size)

	rows, err := r.getExecutor(spanCtx).QueryContext(spanCtx, dataQuery, args...)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "query search failed")
		return nil, 0, err
	}
	defer rows.Close()

	var users []*model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.UUID, &u.Name, &u.Email, &u.CreatedAt, &u.UpdatedAt); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "scan user failed")
			return nil, 0, err
		}
		users = append(users, &u)
	}

	return users, total, nil
}

func (r *UserRepository) Create(ctx context.Context, user *model.User) error {
	spanCtx, span := r.tracer.Start(ctx, "UserRepository.Create")
	defer span.End()
	_, err := r.getExecutor(spanCtx).ExecContext(spanCtx, `
        INSERT INTO users (uuid, name, email, password, created_at, updated_at)
        VALUES ($1, $2, $3, $4, NOW(), NOW())
    `, user.UUID, user.Name, user.Email, user.Password)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "create user failed")
	}
	return err
}

func (r *UserRepository) Update(ctx context.Context, user *model.User) error {
	spanCtx, span := r.tracer.Start(ctx, "UserRepository.Update")
	defer span.End()
	_, err := r.getExecutor(spanCtx).ExecContext(spanCtx, `	
        UPDATE users SET name = $1, email = $2, updated_at = NOW() WHERE uuid = $3
    `, user.Name, user.Email, user.UUID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "update user failed")
	}
	return err
}

func (r *UserRepository) Delete(ctx context.Context, user *model.User) error {
	spanCtx, span := r.tracer.Start(ctx, "UserRepository.Delete")
	defer span.End()
	_, err := r.getExecutor(spanCtx).ExecContext(spanCtx, `DELETE FROM users WHERE uuid = $1`, user.UUID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "delete user failed")
	}
	return err
}
