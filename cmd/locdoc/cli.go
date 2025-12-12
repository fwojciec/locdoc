package main

import (
	"context"
	"io"

	"github.com/fwojciec/locdoc"
	"github.com/fwojciec/locdoc/crawl"
	"github.com/fwojciec/locdoc/sqlite"
)

// Dependencies holds all services and configuration for command execution.
type Dependencies struct {
	Ctx       context.Context
	Stdout    io.Writer
	Stderr    io.Writer
	DB        *sqlite.DB
	Projects  locdoc.ProjectService
	Documents locdoc.DocumentService
	Sitemaps  locdoc.SitemapService
	Crawler   *crawl.Crawler
	Asker     locdoc.Asker
}

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

// ListCmd is the "list" subcommand.
type ListCmd struct{}

// DeleteCmd is the "delete" subcommand.
type DeleteCmd struct {
	Name  string `arg:"" help:"Project name"`
	Force bool   `help:"Confirm deletion"`
}

// DocsCmd is the "docs" subcommand.
type DocsCmd struct {
	Name string `arg:"" help:"Project name"`
	Full bool   `help:"Show full document content"`
}

// AskCmd is the "ask" subcommand.
type AskCmd struct {
	Name     string `arg:"" help:"Project name"`
	Question string `arg:"" help:"Question to ask about the documentation"`
}
