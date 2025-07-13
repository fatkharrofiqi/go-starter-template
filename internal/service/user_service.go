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
	if err := s.userRepository.FindByUUID(spanCtx, user, uuid); err != nil {
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
