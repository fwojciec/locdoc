package locdoc_test

import (
	"testing"

	"github.com/fwojciec/locdoc"
	"github.com/stretchr/testify/assert"
)

func TestErrorf(t *testing.T) {
	t.Parallel()

	err := locdoc.Errorf(locdoc.ENOTFOUND, "project %q not found", "test")

	assert.Equal(t, locdoc.ENOTFOUND, locdoc.ErrorCode(err))
	assert.Equal(t, "project \"test\" not found", locdoc.ErrorMessage(err))
}

func TestErrorCode_NilError(t *testing.T) {
	t.Parallel()

	assert.Empty(t, locdoc.ErrorCode(nil))
}

func TestErrorMessage_NilError(t *testing.T) {
	t.Parallel()

	assert.Empty(t, locdoc.ErrorMessage(nil))
}
