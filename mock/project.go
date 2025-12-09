package mock

import (
	"context"

	"github.com/fwojciec/locdoc"
)

var _ locdoc.ProjectService = (*ProjectService)(nil)

// ProjectService is a mock implementation of locdoc.ProjectService.
type ProjectService struct {
	CreateProjectFn   func(ctx context.Context, project *locdoc.Project) error
	FindProjectByIDFn func(ctx context.Context, id string) (*locdoc.Project, error)
	FindProjectsFn    func(ctx context.Context, filter locdoc.ProjectFilter) ([]*locdoc.Project, error)
	UpdateProjectFn   func(ctx context.Context, id string, upd locdoc.ProjectUpdate) (*locdoc.Project, error)
	DeleteProjectFn   func(ctx context.Context, id string) error
}

func (s *ProjectService) CreateProject(ctx context.Context, project *locdoc.Project) error {
	return s.CreateProjectFn(ctx, project)
}

func (s *ProjectService) FindProjectByID(ctx context.Context, id string) (*locdoc.Project, error) {
	return s.FindProjectByIDFn(ctx, id)
}

func (s *ProjectService) FindProjects(ctx context.Context, filter locdoc.ProjectFilter) ([]*locdoc.Project, error) {
	return s.FindProjectsFn(ctx, filter)
}

func (s *ProjectService) UpdateProject(ctx context.Context, id string, upd locdoc.ProjectUpdate) (*locdoc.Project, error) {
	return s.UpdateProjectFn(ctx, id, upd)
}

func (s *ProjectService) DeleteProject(ctx context.Context, id string) error {
	return s.DeleteProjectFn(ctx, id)
}
