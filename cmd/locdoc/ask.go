package main

import (
	"fmt"

	"github.com/fwojciec/locdoc"
)

// Run executes the ask command.
func (c *AskCmd) Run(deps *Dependencies) error {
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

	answer, err := deps.Asker.Ask(deps.Ctx, project.ID, c.Question)
	if err != nil {
		fmt.Fprintf(deps.Stderr, "error: %s\n", locdoc.ErrorMessage(err))
		return err
	}

	fmt.Fprintln(deps.Stdout, answer)
	return nil
}
