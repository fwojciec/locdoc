package main

// CLI defines the command-line interface structure for Kong.
type CLI struct {
	Add    AddCmd    `cmd:"" help:"Add and crawl a documentation project"`
	List   ListCmd   `cmd:"" help:"List all registered projects"`
	Delete DeleteCmd `cmd:"" help:"Delete a project and its documents"`
	Docs   DocsCmd   `cmd:"" help:"List documents for a project"`
	Ask    AskCmd    `cmd:"" help:"Ask a question about project documentation"`
}

// AddCmd is the "add" subcommand.
type AddCmd struct {
	Name        string   `arg:"" help:"Project name"`
	URL         string   `arg:"" help:"Documentation URL"`
	Preview     bool     `short:"p" help:"Show URLs without creating project"`
	Force       bool     `short:"f" help:"Delete existing project first"`
	Filter      []string `short:"F" name:"filter" help:"Filter URLs by regex (repeatable)"`
	Concurrency int      `short:"c" default:"10" help:"Concurrent fetch limit"`
}

// Run executes the add command (stub for now).
func (c *AddCmd) Run() error {
	return nil
}

// ListCmd is the "list" subcommand.
type ListCmd struct{}

// Run executes the list command (stub for now).
func (c *ListCmd) Run() error {
	return nil
}

// DeleteCmd is the "delete" subcommand.
type DeleteCmd struct {
	Name  string `arg:"" help:"Project name"`
	Force bool   `help:"Confirm deletion"`
}

// Run executes the delete command (stub for now).
func (c *DeleteCmd) Run() error {
	return nil
}

// DocsCmd is the "docs" subcommand.
type DocsCmd struct {
	Name string `arg:"" help:"Project name"`
	Full bool   `help:"Show full document content"`
}

// Run executes the docs command (stub for now).
func (c *DocsCmd) Run() error {
	return nil
}

// AskCmd is the "ask" subcommand.
type AskCmd struct {
	Name     string `arg:"" help:"Project name"`
	Question string `arg:"" help:"Question to ask about the documentation"`
}

// Run executes the ask command (stub for now).
func (c *AskCmd) Run() error {
	return nil
}
