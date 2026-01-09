package main

import (
	"context"
	"io"

	"github.com/fwojciec/locdoc"
	"github.com/fwojciec/locdoc/crawl"
)

// Dependencies holds all services and configuration for command execution.
type Dependencies struct {
	Ctx        context.Context
	Stdout     io.Writer
	Stderr     io.Writer
	Sitemaps   locdoc.SitemapService
	Discoverer *crawl.Discoverer
	Crawler    *crawl.Crawler
}

// FetchCmd handles the main fetch operation.
type FetchCmd struct {
	URL         string
	Name        string
	Path        string
	Preview     bool
	Concurrency int
}
