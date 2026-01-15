package main

import (
	"context"
	"io"

	"github.com/fwojciec/locdoc"
)

// Dependencies holds all services and configuration for command execution.
type Dependencies struct {
	Ctx    context.Context
	Stdout io.Writer
	Stderr io.Writer

	// 3-interface architecture
	Source  locdoc.URLSource
	Fetcher locdoc.PageFetcher
	Store   locdoc.PageStore
}

// FetchCmd handles the main fetch operation.
type FetchCmd struct {
	URL         string
	Name        string
	Path        string
	Preview     bool
	Concurrency int
}
