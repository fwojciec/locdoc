package main

import (
	"fmt"

	"github.com/fwojciec/locdoc"
)

// Run executes the delete command.
func (c *DeleteCmd) Run(deps *Dependencies) error {
	if !c.Force {
		fmt.Fprintf(deps.Stderr, "error: use --force to confirm deletion\n")
		return locdoc.Errorf(locdoc.EINVALID, "use --force to confirm deletion")
	}

	projects, err := deps.Projects.FindProjects(deps.Ctx, locdoc.ProjectFilter{Name: &c.Name})
	if err != nil {
		fmt.Fprintf(deps.Stderr, "error: %s\n", locdoc.ErrorMessage(err))
		return err
	}

	if len(projects) == 0 {
		fmt.Fprintf(deps.Stderr, "error: project %q not found. Use 'locdoc list' to see available projects.\n", c.Name)
		return locdoc.Errorf(locdoc.ENOTFOUND, "project %q not found", c.Name)
	}

	project := projects[0]
	if err := deps.Projects.DeleteProject(deps.Ctx, project.ID); err != nil {
		fmt.Fprintf(deps.Stderr, "error: %s\n", locdoc.ErrorMessage(err))
		return err
	}

	fmt.Fprintf(deps.Stdout, "Deleted project %q\n", project.Name)
	return nil
}
