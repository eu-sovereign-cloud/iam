package model_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/eu-sovereign-cloud/iam/internal/model"
)

func TestValidateDNS1123Label(t *testing.T) {
	for _, valid := range []string{"a", "tenant-1", "acme-prod", "123-abc", strings.Repeat("a", 63)} {
		require.NoErrorf(t, model.ValidateDNS1123Label("field", valid), "expected %q to be valid", valid)
	}

	for _, invalid := range []string{"", "Tenant-One", "tenant_one", "-tenant", "tenant-", "tenant one", strings.Repeat("a", 64)} {
		err := model.ValidateDNS1123Label("field", invalid)
		require.Errorf(t, err, "expected %q to be rejected", invalid)
		require.ErrorIs(t, err, model.ErrInvalid)
	}
}

func TestValidateSubject(t *testing.T) {
	for _, valid := range []string{"admin", "alice@example.com", "bob.smith+test@example.co.uk", "a"} {
		require.NoErrorf(t, model.ValidateSubject(valid), "expected %q to be valid", valid)
	}

	for _, invalid := range []string{"", "@example.com", "alice@", "alice example.com", "alice\texample.com", "alice<script>"} {
		err := model.ValidateSubject(invalid)
		require.Errorf(t, err, "expected %q to be rejected", invalid)
		require.ErrorIs(t, err, model.ErrInvalid)
	}

	require.NoError(t, model.ValidateSubject(strings.Repeat("a", 253)))
	require.Error(t, model.ValidateSubject(strings.Repeat("a", 254)))
}
