package main

import (
	"fmt"

	"github.com/fwojciec/locdoc"
)

// Run executes the list command.
func (c *ListCmd) Run(deps *Dependencies) error {
	projects, err := deps.Projects.FindProjects(deps.Ctx, locdoc.ProjectFilter{})
	if err != nil {
		fmt.Fprintf(deps.Stderr, "error: %s\n", locdoc.ErrorMessage(err))
		return err
	}

	if len(projects) == 0 {
		fmt.Fprintln(deps.Stdout, "No projects found. Use 'locdoc add' to create one.")
		return nil
	}

	for _, p := range projects {
		fmt.Fprintf(deps.Stdout, "%s  %s  %s\n", p.ID, p.Name, p.SourceURL)
	}

	return nil
}
