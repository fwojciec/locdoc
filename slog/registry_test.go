package slog_test

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/fwojciec/locdoc"
	"github.com/fwojciec/locdoc/mock"
	locslog "github.com/fwojciec/locdoc/slog"
	"github.com/stretchr/testify/assert"
)

func TestLoggingRegistry_GetForHTML(t *testing.T) {
	t.Parallel()

	t.Run("logs detected framework with duration", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))
		mockSelector := &mock.LinkSelector{}
		inner := &mock.LinkSelectorRegistry{
			GetForHTMLFn: func(html string) locdoc.LinkSelector {
				return mockSelector
			},
		}
		detector := &mock.FrameworkDetector{
			DetectFn: func(html string) locdoc.Framework {
				return locdoc.FrameworkDocusaurus
			},
		}

		registry := locslog.NewLoggingRegistry(inner, detector, logger)
		selector := registry.GetForHTML("<html>docusaurus</html>")

		assert.Equal(t, mockSelector, selector)
		output := buf.String()
		assert.Contains(t, output, "framework detection")
		assert.Contains(t, output, "framework=docusaurus")
		assert.Contains(t, output, "duration=")
	})

	t.Run("logs unknown framework", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))
		mockSelector := &mock.LinkSelector{}
		inner := &mock.LinkSelectorRegistry{
			GetForHTMLFn: func(html string) locdoc.LinkSelector {
				return mockSelector
			},
		}
		detector := &mock.FrameworkDetector{
			DetectFn: func(html string) locdoc.Framework {
				return locdoc.FrameworkUnknown
			},
		}

		registry := locslog.NewLoggingRegistry(inner, detector, logger)
		registry.GetForHTML("<html>unknown</html>")

		output := buf.String()
		assert.Contains(t, output, "framework=(unknown)")
	})
}

func TestLoggingRegistry_Get(t *testing.T) {
	t.Parallel()

	t.Run("delegates to inner registry", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))
		mockSelector := &mock.LinkSelector{}
		inner := &mock.LinkSelectorRegistry{
			GetFn: func(framework locdoc.Framework) locdoc.LinkSelector {
				return mockSelector
			},
		}

		registry := locslog.NewLoggingRegistry(inner, nil, logger)
		selector := registry.Get(locdoc.FrameworkDocusaurus)

		assert.Equal(t, mockSelector, selector)
	})
}

func TestLoggingRegistry_Register(t *testing.T) {
	t.Parallel()

	t.Run("delegates to inner registry", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))
		var registeredFramework locdoc.Framework
		var registeredSelector locdoc.LinkSelector
		mockSelector := &mock.LinkSelector{}
		inner := &mock.LinkSelectorRegistry{
			RegisterFn: func(framework locdoc.Framework, selector locdoc.LinkSelector) {
				registeredFramework = framework
				registeredSelector = selector
			},
		}

		registry := locslog.NewLoggingRegistry(inner, nil, logger)
		registry.Register(locdoc.FrameworkDocusaurus, mockSelector)

		assert.Equal(t, locdoc.FrameworkDocusaurus, registeredFramework)
		assert.Equal(t, mockSelector, registeredSelector)
	})
}

func TestLoggingRegistry_List(t *testing.T) {
	t.Parallel()

	t.Run("delegates to inner registry", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))
		inner := &mock.LinkSelectorRegistry{
			ListFn: func() []locdoc.Framework {
				return []locdoc.Framework{locdoc.FrameworkDocusaurus, locdoc.FrameworkSphinx}
			},
		}

		registry := locslog.NewLoggingRegistry(inner, nil, logger)
		frameworks := registry.List()

		assert.Equal(t, []locdoc.Framework{locdoc.FrameworkDocusaurus, locdoc.FrameworkSphinx}, frameworks)
	})
}
