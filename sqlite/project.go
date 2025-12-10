package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/fwojciec/locdoc"
	"github.com/google/uuid"
)

// Compile-time interface verification.
var _ locdoc.ProjectService = (*ProjectService)(nil)

// ProjectService implements locdoc.ProjectService using SQLite.
type ProjectService struct {
	db *DB
}

// NewProjectService creates a new ProjectService.
func NewProjectService(db *DB) *ProjectService {
	return &ProjectService{db: db}
}

// CreateProject creates a new project.
func (s *ProjectService) CreateProject(ctx context.Context, project *locdoc.Project) error {
	if err := project.Validate(); err != nil {
		return err
	}

	project.ID = uuid.New().String()
	now := time.Now().UTC()
	project.CreatedAt = now
	project.UpdatedAt = now

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO projects (id, name, source_url, local_path, filter, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, project.ID, project.Name, project.SourceURL, project.LocalPath, project.Filter,
		project.CreatedAt.Format(time.RFC3339), project.UpdatedAt.Format(time.RFC3339))

	return err
}

// FindProjectByID retrieves a project by ID.
func (s *ProjectService) FindProjectByID(ctx context.Context, id string) (*locdoc.Project, error) {
	var project locdoc.Project
	var createdAt, updatedAt string

	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, source_url, local_path, filter, created_at, updated_at
		FROM projects
		WHERE id = ?
	`, id).Scan(&project.ID, &project.Name, &project.SourceURL, &project.LocalPath, &project.Filter,
		&createdAt, &updatedAt)

	if err == sql.ErrNoRows {
		return nil, locdoc.Errorf(locdoc.ENOTFOUND, "project not found")
	}
	if err != nil {
		return nil, err
	}

	var parseErr error
	project.CreatedAt, parseErr = time.Parse(time.RFC3339, createdAt)
	if parseErr != nil {
		return nil, fmt.Errorf("failed to parse created_at: %w", parseErr)
	}
	project.UpdatedAt, parseErr = time.Parse(time.RFC3339, updatedAt)
	if parseErr != nil {
		return nil, fmt.Errorf("failed to parse updated_at: %w", parseErr)
	}

	return &project, nil
}

// FindProjects retrieves projects matching the filter.
func (s *ProjectService) FindProjects(ctx context.Context, filter locdoc.ProjectFilter) ([]*locdoc.Project, error) {
	var query strings.Builder
	var args []any

	query.WriteString("SELECT id, name, source_url, local_path, filter, created_at, updated_at FROM projects WHERE 1=1")

	if filter.ID != nil {
		query.WriteString(" AND id = ?")
		args = append(args, *filter.ID)
	}
	if filter.Name != nil {
		query.WriteString(" AND name = ?")
		args = append(args, *filter.Name)
	}

	query.WriteString(" ORDER BY created_at DESC")

	if filter.Limit > 0 {
		query.WriteString(" LIMIT ?")
		args = append(args, filter.Limit)
	}
	if filter.Offset > 0 {
		query.WriteString(" OFFSET ?")
		args = append(args, filter.Offset)
	}

	rows, err := s.db.QueryContext(ctx, query.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []*locdoc.Project
	for rows.Next() {
		var project locdoc.Project
		var createdAt, updatedAt string

		if err := rows.Scan(&project.ID, &project.Name, &project.SourceURL, &project.LocalPath, &project.Filter,
			&createdAt, &updatedAt); err != nil {
			return nil, err
		}

		var parseErr error
		project.CreatedAt, parseErr = time.Parse(time.RFC3339, createdAt)
		if parseErr != nil {
			return nil, fmt.Errorf("failed to parse created_at: %w", parseErr)
		}
		project.UpdatedAt, parseErr = time.Parse(time.RFC3339, updatedAt)
		if parseErr != nil {
			return nil, fmt.Errorf("failed to parse updated_at: %w", parseErr)
		}

		projects = append(projects, &project)
	}

	return projects, rows.Err()
}

// UpdateProject updates an existing project.
func (s *ProjectService) UpdateProject(ctx context.Context, id string, upd locdoc.ProjectUpdate) (*locdoc.Project, error) {
	// First check if project exists
	project, err := s.FindProjectByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if upd.Name != nil {
		project.Name = *upd.Name
	}
	if upd.SourceURL != nil {
		project.SourceURL = *upd.SourceURL
	}
	if upd.LocalPath != nil {
		project.LocalPath = *upd.LocalPath
	}
	if upd.Filter != nil {
		project.Filter = *upd.Filter
	}

	// Validate before persisting
	if err := project.Validate(); err != nil {
		return nil, err
	}

	project.UpdatedAt = time.Now().UTC()

	_, err = s.db.ExecContext(ctx, `
		UPDATE projects
		SET name = ?, source_url = ?, local_path = ?, filter = ?, updated_at = ?
		WHERE id = ?
	`, project.Name, project.SourceURL, project.LocalPath, project.Filter,
		project.UpdatedAt.Format(time.RFC3339), id)

	if err != nil {
		return nil, err
	}

	return project, nil
}

// DeleteProject permanently removes a project.
func (s *ProjectService) DeleteProject(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, "DELETE FROM projects WHERE id = ?", id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return locdoc.Errorf(locdoc.ENOTFOUND, "project not found")
	}

	return nil
}
