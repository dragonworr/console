package owl

import (
	"fmt"
	"github.com/cloudhut/kowl/backend/pkg/git"
	"github.com/cloudhut/kowl/backend/pkg/kafka"
	"go.uber.org/zap"
)

// Service offers all methods to serve the responses for the REST API. This usually only involves fetching
// several responses from Kafka concurrently and constructing them so, that they are
type Service struct {
	kafkaSvc *kafka.Service
	gitSvc   *git.Service // Git service can be nil if not configured
	logger   *zap.Logger
}

// NewService for the Owl package
func NewService(cfg Config, logger *zap.Logger, kafkaSvc *kafka.Service) (*Service, error) {
	var gitSvc *git.Service
	cfg.TopicDocumentation.Git.AllowedFileExtensions = []string{"md"}
	if cfg.TopicDocumentation.Enabled && cfg.TopicDocumentation.Git.Enabled {
		svc, err := git.NewService(cfg.TopicDocumentation.Git, logger, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create git service: %w", err)
		}
		gitSvc = svc
	}
	return &Service{
		kafkaSvc: kafkaSvc,
		gitSvc:   gitSvc,
		logger:   logger,
	}, nil
}

// Start starts all the (background) tasks which are required for this service to work properly. If any of these
// tasks can not be setup an error will be returned which will cause the application to exit.
func (s *Service) Start() error {
	if s.gitSvc == nil {
		return nil
	}
	return s.gitSvc.Start()
}
