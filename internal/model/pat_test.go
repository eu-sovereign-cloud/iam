package model_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/eu-sovereign-cloud/iam/internal/model"
)

func TestPATExpired(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	assert.False(t, model.PAT{}.Expired(now), "no ExpiresAt means it never expires")

	past := now.Add(-time.Minute)
	assert.True(t, model.PAT{ExpiresAt: &past}.Expired(now))

	future := now.Add(time.Minute)
	assert.False(t, model.PAT{ExpiresAt: &future}.Expired(now))
}
