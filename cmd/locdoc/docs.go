package main

import (
	"fmt"

	"github.com/fwojciec/locdoc"
)

// Run executes the docs command.
func (c *DocsCmd) Run(deps *Dependencies) error {
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

	docs, err := deps.Documents.FindDocuments(deps.Ctx, locdoc.DocumentFilter{
		ProjectID: &project.ID,
		SortBy:    locdoc.SortByPosition,
	})
	if err != nil {
		fmt.Fprintf(deps.Stderr, "error: %s\n", locdoc.ErrorMessage(err))
		return err
	}

	if len(docs) == 0 {
		fmt.Fprintf(deps.Stderr, "error: project %q has no documents. To re-add, first run 'locdoc delete %s --force', then run 'locdoc add %s <url>'.\n", c.Name, c.Name, c.Name)
		return locdoc.Errorf(locdoc.ENOTFOUND, "project %q has no documents", c.Name)
	}

	if c.Full {
		// Print full formatted content (same as what ask sends to LLM)
		fmt.Fprintln(deps.Stdout, locdoc.FormatDocuments(docs))
		return nil
	}

	// Print summary listing
	fmt.Fprintf(deps.Stdout, "Documents for %s (%d total):\n\n", c.Name, len(docs))
	for i, doc := range docs {
		title := doc.Title
		if title == "" {
			title = doc.SourceURL
		}
		fmt.Fprintf(deps.Stdout, "  %d. %s\n     %s\n", i+1, title, doc.SourceURL)
	}

	return nil
}
