// Package locdoc provides a local, CLI-based documentation search tool.
// It crawls documentation sites, extracts content as markdown, indexes it
// for semantic search, and provides a CLI query interface for natural
// language questions.
//
// This package contains domain types and interfaces following Ben Johnson's
// Standard Package Layout. Implementations live in subdirectories named
// after their primary dependency (e.g., sqlite/, katana/, ollama/).
package locdoc
