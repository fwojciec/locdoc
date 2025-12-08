package locdoc

import (
	"context"
	"time"
)

// Project represents a documentation source to be crawled and indexed.
type Project struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	SourceURL string    `json:"sourceUrl"`
	LocalPath string    `json:"localPath"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Validate returns an error if the project contains invalid fields.
func (p *Project) Validate() error {
	if p.Name == "" {
		return Errorf(EINVALID, "project name required")
	}
	if p.SourceURL == "" {
		return Errorf(EINVALID, "project source URL required")
	}
	return nil
}

// ProjectService represents a service for managing projects.
type ProjectService interface {
	// CreateProject creates a new project.
	CreateProject(ctx context.Context, project *Project) error

	// FindProjectByID retrieves a project by ID.
	// Returns ENOTFOUND if project does not exist.
	FindProjectByID(ctx context.Context, id string) (*Project, error)

	// FindProjects retrieves projects matching the filter.
	FindProjects(ctx context.Context, filter ProjectFilter) ([]*Project, error)

	// UpdateProject updates an existing project.
	// Returns ENOTFOUND if project does not exist.
	UpdateProject(ctx context.Context, id string, upd ProjectUpdate) (*Project, error)

	// DeleteProject permanently removes a project and all associated documents.
	// Returns ENOTFOUND if project does not exist.
	DeleteProject(ctx context.Context, id string) error
}

// ProjectFilter represents a filter for FindProjects.
type ProjectFilter struct {
	ID   *string `json:"id"`
	Name *string `json:"name"`

	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}

// ProjectUpdate represents fields that can be updated on a project.
type ProjectUpdate struct {
	Name      *string `json:"name"`
	SourceURL *string `json:"sourceUrl"`
	LocalPath *string `json:"localPath"`
}
