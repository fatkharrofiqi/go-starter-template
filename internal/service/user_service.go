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

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

type UserService struct {
	db             *gorm.DB
	userRepository *repository.UserRepository
	redisService   *RedisService
	log            *logrus.Logger
	tracer         trace.Tracer
}

func NewUserService(db *gorm.DB, userRepository *repository.UserRepository, redisService *RedisService, logrus *logrus.Logger) *UserService {
	return &UserService{db, userRepository, redisService, logrus, otel.Tracer("UserService")}
}

// GetUser retrieves a user by UUID.
func (s *UserService) GetUser(ctx context.Context, uuid string) (string, error) {
	userContext, span := s.tracer.Start(ctx, "GetUser")
	defer span.End()

	cacheKey := fmt.Sprintf("user:me:%s", uuid)
	if cachedResponse, found := s.redisService.Get(userContext, cacheKey); found {
		s.log.Info("user profile retrieved from Redis cache")
		return cachedResponse, nil
	}

	user := new(model.User)
	if err := s.userRepository.FindByUUID(userContext, user, uuid); err != nil {
		s.log.WithError(err).Warn("Failed to find user by UUID")
		return "", errcode.ErrUserNotFound
	}

	result, err := s.redisService.Set(userContext, cacheKey, dto.WebResponse[*dto.UserResponse]{
		Data: converter.UserToResponse(user),
	}, 5*time.Minute)

	if err != nil {
		s.log.WithError(err).Warn("failed to save user response to redis")
		return "", errcode.ErrRedisSet
	}

	return result, nil
}

// Search retrieves users based on search criteria.
func (s *UserService) Search(ctx context.Context, request *dto.SearchUserRequest) ([]*dto.UserResponse, int64, error) {
	userContext, span := s.tracer.Start(ctx, "Search")
	defer span.End()

	users, total, err := s.userRepository.Search(userContext, request)
	if err != nil {
		s.log.WithError(err).Error("Error retrieving users")
		return nil, 0, errcode.ErrUserSearchFailed
	}

	responses := make([]*dto.UserResponse, len(users))
	for i, user := range users {
		responses[i] = converter.UserToResponse(user)
	}

	return responses, total, nil
}
