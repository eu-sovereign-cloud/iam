package controller_test

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/eu-sovereign-cloud/iam/internal/adapter/memorycrypto"
	"github.com/eu-sovereign-cloud/iam/internal/controller"
)

func TestGetJWKS(t *testing.T) {
	signer := memorycrypto.Signer{}
	c := &controller.GetJWKS{Signer: signer}

	set, err := c.Do(context.Background())
	require.NoError(t, err)
	require.Len(t, set.Keys, 1)

	key := set.Keys[0]
	require.Equal(t, "EC", key.Kty)
	require.Equal(t, "P-256", key.Crv)
	require.Equal(t, "sig", key.Use)
	require.Equal(t, "ES256", key.Alg)
	require.Equal(t, signer.KeyID(), key.Kid)

	x, err := base64.RawURLEncoding.DecodeString(key.X)
	require.NoError(t, err)
	y, err := base64.RawURLEncoding.DecodeString(key.Y)
	require.NoError(t, err)

	// Reassemble the uncompressed SEC1 point (0x04 || X || Y) and confirm
	// it matches the signer's actual public key, so x/y round-trip.
	wantRaw, err := signer.PublicKey().Bytes()
	require.NoError(t, err)
	gotRaw := append([]byte{0x04}, append(append([]byte{}, x...), y...)...)
	require.Equal(t, wantRaw, gotRaw)
}
