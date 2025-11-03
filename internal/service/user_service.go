package service

import (
	"context"
	"fmt"
	"go-starter-template/internal/dto"
	"go-starter-template/internal/dto/converter"
	"go-starter-template/internal/model"
	"go-starter-template/internal/repository"
	"go-starter-template/internal/utils/errcode"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
    userRepository *repository.UserRepository
    redisService   *RedisService
    log            *logrus.Logger
    tracer         trace.Tracer
    hashPassword   func(password []byte, cost int) ([]byte, error)
}

func NewUserService(userRepository *repository.UserRepository, redisService *RedisService, logrus *logrus.Logger) *UserService {
    return &UserService{userRepository: userRepository, redisService: redisService, log: logrus, tracer: otel.Tracer("UserService"), hashPassword: bcrypt.GenerateFromPassword}
}

// GetUser retrieves a user by UUID.
func (s *UserService) GetUser(ctx context.Context, uuid string) (result string, err error) {
	spanCtx, span := s.tracer.Start(ctx, "UserService.GetUser")
	defer span.End()

	logger := s.log.WithContext(spanCtx)
	cacheKey := fmt.Sprintf("user:me:%s", uuid)

	cachedResponse, found := s.redisService.Get(spanCtx, cacheKey)
	if found {
		logger.Info("user profile retrieved from Redis cache")
		return cachedResponse, nil
	}

	user := new(model.User)
	if err = s.userRepository.FindByUUID(spanCtx, user, uuid); err != nil {
		logger.WithError(err).Warn("failed to find user by UUID")
		return "", errcode.ErrUserNotFound
	}

	result, err = s.redisService.Set(spanCtx, cacheKey, dto.WebResponse[*dto.UserResponse]{
		Data: converter.UserToResponse(user),
	}, 5*time.Minute)
	if err != nil {
		logger.WithError(err).Warn("failed to save user response to Redis")
	}

	return
}

// Search retrieves users based on search criteria.
func (s *UserService) Search(ctx context.Context, request *dto.SearchUserRequest) ([]*dto.UserResponse, int64, error) {
	spanCtx, span := s.tracer.Start(ctx, "UserService.Search")
	defer span.End()

	users, total, err := s.userRepository.Search(spanCtx, request)

	if err != nil {
		s.log.WithContext(spanCtx).WithError(err).Error("Error retrieving users")
		return nil, 0, errcode.ErrUserSearchFailed
	}

	_, convertSpan := s.tracer.Start(spanCtx, "ConvertUsersToDTO")
	responses := make([]*dto.UserResponse, len(users))
	for i, user := range users {
		responses[i] = converter.UserToResponse(user)
	}
	convertSpan.End()

	return responses, total, nil
}

// CreateUser creates a new user.
func (s *UserService) CreateUser(ctx context.Context, request *dto.CreateUserRequest) (*dto.UserResponse, error) {
	spanCtx, span := s.tracer.Start(ctx, "UserService.CreateUser")
	defer span.End()

	logger := s.log.WithContext(spanCtx)
	// Check if email already exists
	count, err := s.userRepository.CountByEmail(spanCtx, request.Email)
	if err != nil {
		logger.WithError(err).Error("Failed to check email existence")
		return nil, errcode.ErrInternalServerError
	}

	if count > 0 {
		logger.Warn("Attempt to add an already existing email")
		return nil, errcode.ErrUserAlreadyExists
	}

	_, hashSpan := s.tracer.Start(spanCtx, "HashPassword")
    hashedPassword, err := s.hashPassword([]byte(request.Password), bcrypt.DefaultCost)
	hashSpan.End()
	if err != nil {
		logger.WithError(err).Error("Failed to hash password")
		return nil, errcode.ErrPasswordEncryption
	}

	// Create user entity
	user := &model.User{
		UUID:     uuid.New().String(),
		Name:     request.Name,
		Email:    request.Email,
		Password: string(hashedPassword),
	}

	// Create user
	if err := s.userRepository.Create(spanCtx, user); err != nil {
		logger.WithError(err).Error("Failed to create user")
		return nil, errcode.ErrInternalServerError
	}

	// Convert to response
	response := converter.UserToResponse(user)
	return response, nil
}

// UpdateUser updates an existing user.
func (s *UserService) UpdateUser(ctx context.Context, uuid string, request *dto.UpdateUserRequest) (*dto.UserResponse, error) {
	spanCtx, span := s.tracer.Start(ctx, "UserService.UpdateUser")
	defer span.End()

	logger := s.log.WithContext(spanCtx)
	// Find user
	user := new(model.User)
	if err := s.userRepository.FindByUUID(spanCtx, user, uuid); err != nil {
		logger.WithError(err).Warn("Failed to find user by UUID")
		return nil, errcode.ErrUserNotFound
	}

	// Check if email already exists (if email is changed)
	if user.Email != request.Email {
		count, err := s.userRepository.CountByEmail(spanCtx, request.Email)
		if err != nil {
			logger.WithError(err).Error("Failed to check email existence")
			return nil, errcode.ErrInternalServerError
		}

		if count > 0 {
			return nil, errcode.ErrUserAlreadyExists
		}
	}

	// Update user fields
	user.Name = request.Name
	user.Email = request.Email

	// Update user
	if err := s.userRepository.Update(spanCtx, user); err != nil {
		logger.WithError(err).Error("Failed to update user")
		return nil, errcode.ErrInternalServerError
	}

	// Convert to response
	response := converter.UserToResponse(user)
	return response, nil
}

// DeleteUser deletes a user by UUID.
func (s *UserService) DeleteUser(ctx context.Context, uuid string) error {
	spanCtx, span := s.tracer.Start(ctx, "UserService.DeleteUser")
	defer span.End()

	logger := s.log.WithContext(spanCtx)
	// Find user
	user := new(model.User)
	if err := s.userRepository.FindByUUID(spanCtx, user, uuid); err != nil {
		logger.WithError(err).Warn("Failed to find user by UUID")
		return errcode.ErrUserNotFound
	}

	// Delete user
	if err := s.userRepository.Delete(spanCtx, user); err != nil {
		logger.WithError(err).Error("Failed to delete user")
		return errcode.ErrInternalServerError
	}

	return nil
}
