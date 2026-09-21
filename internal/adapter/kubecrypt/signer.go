// Package kubecrypt implements IAM's ES256 JWT signer/verifier, backed by a
// key pair kept in a Kubernetes Secret (ADR 0005, ADR 0012) — one of
// possibly several driven adapters under internal/adapter, alongside the
// kubestore data store.
package kubecrypt

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/eu-sovereign-cloud/iam/internal/model"
	"github.com/eu-sovereign-cloud/iam/internal/pkg/kube"
)

const (
	signingKeySecretName = "iam-signing-key"
	signingMethod        = "ES256"
)

// Signer signs JWTs with an ES256 key pair kept in a Kubernetes Secret
// (ADR 0005): asymmetric so a future JWKS endpoint (issue #2) can publish
// the public key straight from the same Secret, generated on first
// startup if the Secret doesn't already exist.
type Signer struct {
	kid        string
	privateKey *ecdsa.PrivateKey
}

// LoadOrCreate reads the signing key Secret, generating and persisting a
// fresh ES256 key pair if it doesn't exist yet.
func LoadOrCreate(ctx context.Context, client kubernetes.Interface, namespace string) (*Signer, error) {
	secrets := client.CoreV1().Secrets(namespace)

	sec, err := secrets.Get(ctx, signingKeySecretName, kube.MetaGetOpts())
	if err == nil {
		return signerFromSecret(sec)
	}
	if !kube.IsNotFound(err) {
		return nil, fmt.Errorf("getting signing key secret: %w", err)
	}

	signer, sec, err := newSigner(namespace)
	if err != nil {
		return nil, err
	}
	if _, err := secrets.Create(ctx, sec, kube.MetaCreateOpts()); err != nil {
		if kube.IsAlreadyExists(err) {
			// Lost a startup race with another instance; read back what it wrote.
			sec, getErr := secrets.Get(ctx, signingKeySecretName, kube.MetaGetOpts())
			if getErr != nil {
				return nil, fmt.Errorf("getting signing key secret after create race: %w", getErr)
			}
			return signerFromSecret(sec)
		}
		return nil, fmt.Errorf("creating signing key secret: %w", err)
	}
	return signer, nil
}

func newSigner(namespace string) (*Signer, *corev1.Secret, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("generating ES256 key: %w", err)
	}
	kid := newKeyID()

	privBytes, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, nil, fmt.Errorf("marshaling private key: %w", err)
	}
	pubBytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return nil, nil, fmt.Errorf("marshaling public key: %w", err)
	}

	sec := &corev1.Secret{}
	sec.Name = signingKeySecretName
	sec.Namespace = namespace
	sec.Type = corev1.SecretTypeOpaque
	sec.Data = map[string][]byte{
		"kid":         []byte(kid),
		"method":      []byte(signingMethod),
		"private.pem": pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privBytes}),
		"public.pem":  pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes}),
	}
	return &Signer{kid: kid, privateKey: key}, sec, nil
}

func signerFromSecret(sec *corev1.Secret) (*Signer, error) {
	block, _ := pem.Decode(sec.Data["private.pem"])
	if block == nil {
		return nil, fmt.Errorf("signing key secret %q: private.pem is not PEM-encoded", sec.Name)
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parsing private key: %w", err)
	}
	ecKey, ok := key.(*ecdsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("signing key secret %q: private key is not ECDSA", sec.Name)
	}
	return &Signer{kid: string(sec.Data["kid"]), privateKey: ecKey}, nil
}

func (s *Signer) Sign(claims model.Claims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["kid"] = s.kid
	signed, err := token.SignedString(s.privateKey)
	if err != nil {
		return "", fmt.Errorf("signing JWT: %w", err)
	}
	return signed, nil
}

// Verify parses and validates a JWT signed by this Signer (or an earlier
// key it loaded), checking the signature and the mandatory expiry —
// mirroring ecp's own gateway verification style
// (ecp/gateway/internal/authn/jwtstd.go). Since a PAT *is* the JWT (ADR
// 0012), this is what IAM uses to authenticate its own incoming bearer
// tokens; it does not check issuer/audience, since only this Signer's own
// key could have produced a validly-signed token in the first place.
func (s *Signer) Verify(tokenString string) (model.Claims, error) {
	claims := &model.Claims{}
	_, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, fmt.Errorf("unexpected signing method %q", token.Method.Alg())
		}
		return &s.privateKey.PublicKey, nil
	}, jwt.WithValidMethods([]string{signingMethod}), jwt.WithExpirationRequired())
	if err != nil {
		return model.Claims{}, fmt.Errorf("invalid token: %w", err)
	}
	return *claims, nil
}

func newKeyID() string {
	buf := make([]byte, 8)
	_, _ = rand.Read(buf)
	return fmt.Sprintf("%x", buf)
}
