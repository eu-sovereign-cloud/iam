//go:build e2e

// Package e2e runs iamd as a real subprocess against a real Kubernetes API
// server (a kind cluster in CI, or any cluster reachable via the current
// kubeconfig locally) and drives it over HTTP exactly the way an operator
// or the ecp gateway would. Build-tagged so `go test ./...` never picks it
// up by accident; run explicitly with `go test -tags e2e ./test/e2e/...`
// (see `make e2e` and `.github/workflows/e2e.yml`).
//
// See doc/adr/0011-e2e-test-against-kind.md for why this exists alongside
// the fake-clientset integration test in internal/service.
package e2e

import (
	"bufio"
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

const (
	testNamespace = "iam-e2e-test"
	listenAddr    = "127.0.0.1:18080"
	baseURL       = "http://" + listenAddr
	roleCRD       = "testdata/crds/authorization.v1.secapi.cloud_roles.yaml"
	roleAssignCRD = "testdata/crds/authorization.v1.secapi.cloud_role-assignments.yaml"
)

func TestEndToEnd(t *testing.T) {
	requireKubectl(t)
	// Registered first so it runs last (t.Cleanup is LIFO): the namespace
	// is deleted only after iamd has stopped and the CRDs/demo object are
	// already gone.
	t.Cleanup(func() {
		_ = exec.Command("kubectl", "delete", "namespace", testNamespace, "--ignore-not-found").Run()
	})

	applyCRDs(t)

	binPath := buildIamd(t)
	proc, stdout := startIamd(t, binPath)
	adminPAT := waitForBootstrapPAT(t, stdout)
	waitForListening(t)

	// --- Core flow: tenant -> user -> grant -> self-service PAT ---
	// Per ADR 0012 the PAT itself is the signed JWT; there is no exchange
	// step. createPAT's returned secret is used directly as the bearer
	// credential everywhere below.

	createTenant(t, adminPAT, "tenant-1", "Tenant One")
	createUser(t, adminPAT, "alice@example.com", "Alice", false)
	createGrant(t, adminPAT, "alice@example.com", "tenant-1", "member")

	aliceID, alicePAT := createPAT(t, adminPAT, "alice@example.com", "laptop")

	claims := decodeJWTClaims(t, alicePAT)
	require.Equal(t, "alice@example.com", claims["sub"])
	require.Equal(t, []any{"tenant-1"}, claims["tenants"])
	require.Contains(t, claims, "iss")
	require.Contains(t, claims, "exp")

	// --- OIDC discovery, JWKS, and /userinfo (issue #2) ---

	// jwks_uri/userinfo_endpoint are derived from IAM_JWT_ISSUER (set to
	// "https://iam.e2e-test.local" below, distinct from the actual
	// listenAddr this test talks to - the issuer is a logical identity,
	// not a dialable address), not from baseURL.
	doc := fetchDiscoveryDocument(t)
	require.Equal(t, "https://iam.e2e-test.local", doc["issuer"])
	require.Equal(t, "https://iam.e2e-test.local/.well-known/jwks.json", doc["jwks_uri"])
	require.Equal(t, "https://iam.e2e-test.local/userinfo", doc["userinfo_endpoint"])

	// The important check: the published key material actually verifies
	// alicePAT's real signature, not just that the endpoint returns
	// plausible-looking JSON.
	verifyOffline(t, fetchJWKS(t), alicePAT)

	userinfoResp := doJSON(t, http.MethodGet, "/userinfo", alicePAT, nil)
	assertStatus(t, http.StatusOK, userinfoResp)
	var info struct {
		Subject string `json:"sub"`
	}
	decodeBody(t, userinfoResp, &info)
	require.Equal(t, "alice@example.com", info.Subject)

	// --- Authorization boundaries ---

	assertStatus(t, http.StatusForbidden, doJSON(t, http.MethodPost, "/api/v1/tenants", alicePAT, map[string]string{"tenantId": "tenant-2"}))
	assertStatus(t, http.StatusForbidden, doJSON(t, http.MethodGet, "/api/v1/users/admin/pats", alicePAT, nil))

	// --- Revocation ---
	// The signed JWT is still validly signed and unexpired, but IAM's own
	// API must now reject it because its jti is no longer known (ADR
	// 0012's revocation gap only applies to *other* verifiers, like ecp).
	// /userinfo must reflect the same revocation immediately (issue #2) -
	// that's the whole point of it existing alongside offline JWKS
	// verification.

	assertStatus(t, http.StatusNoContent, doJSON(t, http.MethodDelete, "/api/v1/users/alice@example.com/pats/"+aliceID, alicePAT, nil))
	resp := doJSON(t, http.MethodGet, "/api/v1/users/alice@example.com/pats", alicePAT, nil)
	assertStatus(t, http.StatusUnauthorized, resp)
	assertStatus(t, http.StatusUnauthorized, doJSON(t, http.MethodGet, "/userinfo", alicePAT, nil))

	// --- Restart: ADR 0009's load-at-startup cache must reflect prior state ---

	stopIamd(t, proc)
	proc2, stdout2 := startIamd(t, binPath)
	requireNoNewBootstrapPAT(t, stdout2)
	waitForListening(t)
	proc = proc2

	tenants := listTenants(t, adminPAT)
	require.Len(t, tenants, 1)
	require.Equal(t, "tenant-1", tenants[0]["tenantId"])

	// --- Vendored ecp CRD compatibility check (not IAM's own behavior; see
	// test/e2e/testdata/crds/README.md and ADR 0011) ---

	applyDemoRoleAssignment(t, "alice@example.com", "tenant-1")

	stopIamd(t, proc)
}

func requireKubectl(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("kubectl"); err != nil {
		t.Fatalf("kubectl not found in PATH: %v", err)
	}
}

func kubectlRun(t *testing.T, args ...string) {
	t.Helper()
	cmd := exec.Command("kubectl", args...) //nolint:gosec // test-only, fixed set of literal args built by this file
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "kubectl %s failed: %s", strings.Join(args, " "), out)
}

func applyCRDs(t *testing.T) {
	t.Helper()
	kubectlRun(t, "apply", "-f", roleCRD)
	kubectlRun(t, "apply", "-f", roleAssignCRD)
	t.Cleanup(func() {
		_ = exec.Command("kubectl", "delete", "-f", roleAssignCRD, "--ignore-not-found").Run()
		_ = exec.Command("kubectl", "delete", "-f", roleCRD, "--ignore-not-found").Run()
	})
}

func applyDemoRoleAssignment(t *testing.T, subject, tenantID string) {
	t.Helper()
	name := "iam-e2e-demo"
	manifest := fmt.Sprintf(`apiVersion: authorization.v1.secapi.cloud/v1
kind: RoleAssignment
metadata:
  name: %s
  namespace: default
spec:
  subs: [%q]
  roles: ["member"]
  scopes:
    - tenants: [%q]
`, name, subject, tenantID)

	tmp, err := os.CreateTemp(t.TempDir(), "roleassignment-*.yaml")
	require.NoError(t, err)
	_, err = tmp.WriteString(manifest)
	require.NoError(t, err)
	require.NoError(t, tmp.Close())

	kubectlRun(t, "apply", "-f", tmp.Name())
	t.Cleanup(func() {
		_ = exec.Command("kubectl", "delete", "roleassignment", name, "-n", "default", "--ignore-not-found").Run()
	})
}

func buildIamd(t *testing.T) string {
	t.Helper()
	binPath := t.TempDir() + "/iamd"
	cmd := exec.Command("go", "build", "-o", binPath, "../../cmd/iamd") //nolint:gosec // test-only, fixed literal args
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "building iamd: %s", out)
	return binPath
}

// startIamd starts iamd and returns the process plus a channel of its
// combined stdout/stderr lines (JSON log lines), which the caller drains
// for the bootstrap PAT and can keep reading from for later assertions.
func startIamd(t *testing.T, binPath string) (*exec.Cmd, <-chan string) {
	t.Helper()
	cmd := exec.Command(binPath)
	cmd.Env = append(os.Environ(),
		"IAM_LISTEN_ADDR="+listenAddr,
		"IAM_NAMESPACE="+testNamespace,
		"IAM_JWT_ISSUER=https://iam.e2e-test.local",
		"IAM_JWT_AUDIENCE=ecp-gateway",
	)
	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)
	cmd.Stderr = cmd.Stdout

	require.NoError(t, cmd.Start())
	t.Cleanup(func() { stopIamd(t, cmd) })

	lines := make(chan string, 64)
	go func() {
		defer close(lines)
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			lines <- scanner.Text()
		}
	}()
	return cmd, lines
}

func stopIamd(t *testing.T, cmd *exec.Cmd) {
	t.Helper()
	if cmd.Process == nil {
		return
	}
	_ = cmd.Process.Signal(syscall.SIGTERM)
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		_ = cmd.Process.Kill()
		<-done
	}
}

func waitForBootstrapPAT(t *testing.T, lines <-chan string) string {
	t.Helper()
	deadline := time.After(15 * time.Second)
	for {
		select {
		case line, ok := <-lines:
			if !ok {
				t.Fatal("iamd exited before logging a bootstrap PAT")
			}
			var entry map[string]any
			if json.Unmarshal([]byte(line), &entry) != nil {
				continue
			}
			if pat, ok := entry["pat"].(string); ok && pat != "" {
				return pat
			}
		case <-deadline:
			t.Fatal("timed out waiting for bootstrap admin PAT in iamd logs")
		}
	}
}

func requireNoNewBootstrapPAT(t *testing.T, lines <-chan string) {
	t.Helper()
	select {
	case line, ok := <-lines:
		if !ok {
			return
		}
		var entry map[string]any
		if json.Unmarshal([]byte(line), &entry) == nil {
			require.NotContains(t, fmt.Sprint(entry["msg"]), "bootstrap admin PAT",
				"a restart with an existing admin must not mint a second bootstrap PAT")
		}
	case <-time.After(500 * time.Millisecond):
	}
}

func waitForListening(t *testing.T) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", listenAddr, time.Second)
		if err == nil {
			_ = conn.Close()
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("iamd never started listening on %s", listenAddr)
}

func doJSON(t *testing.T, method, path, bearer string, body any) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
	}
	req, err := http.NewRequest(method, baseURL+path, &buf)
	require.NoError(t, err)
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })
	return resp
}

func assertStatus(t *testing.T, want int, resp *http.Response) {
	t.Helper()
	require.Equal(t, want, resp.StatusCode)
}

func decodeBody(t *testing.T, resp *http.Response, v any) {
	t.Helper()
	require.NoError(t, json.NewDecoder(resp.Body).Decode(v))
}

func createTenant(t *testing.T, adminPAT, tenantID, displayName string) {
	t.Helper()
	resp := doJSON(t, http.MethodPost, "/api/v1/tenants", adminPAT, map[string]string{"tenantId": tenantID, "displayName": displayName})
	assertStatus(t, http.StatusCreated, resp)
}

func createUser(t *testing.T, adminPAT, subject, displayName string, admin bool) {
	t.Helper()
	resp := doJSON(t, http.MethodPost, "/api/v1/users", adminPAT, map[string]any{"subject": subject, "displayName": displayName, "admin": admin})
	assertStatus(t, http.StatusCreated, resp)
}

func createGrant(t *testing.T, adminPAT, subject, tenantID, role string) {
	t.Helper()
	resp := doJSON(t, http.MethodPost, "/api/v1/users/"+subject+"/grants", adminPAT, map[string]any{"tenantId": tenantID, "roles": []string{role}})
	assertStatus(t, http.StatusCreated, resp)
}

func createPAT(t *testing.T, callerPAT, subject, name string) (id, secret string) {
	t.Helper()
	resp := doJSON(t, http.MethodPost, "/api/v1/users/"+subject+"/pats", callerPAT, map[string]string{"name": name})
	assertStatus(t, http.StatusCreated, resp)
	var created struct {
		ID     string `json:"id"`
		Secret string `json:"secret"`
	}
	decodeBody(t, resp, &created)
	require.NotEmpty(t, created.ID)
	require.NotEmpty(t, created.Secret)
	return created.ID, created.Secret
}

func listTenants(t *testing.T, adminPAT string) []map[string]any {
	t.Helper()
	resp := doJSON(t, http.MethodGet, "/api/v1/tenants", adminPAT, nil)
	assertStatus(t, http.StatusOK, resp)
	var out []map[string]any
	decodeBody(t, resp, &out)
	return out
}

func fetchDiscoveryDocument(t *testing.T) map[string]any {
	t.Helper()
	resp := doJSON(t, http.MethodGet, "/.well-known/openid-configuration", "", nil)
	assertStatus(t, http.StatusOK, resp)
	var doc map[string]any
	decodeBody(t, resp, &doc)
	return doc
}

func fetchJWKS(t *testing.T) map[string]any {
	t.Helper()
	resp := doJSON(t, http.MethodGet, "/.well-known/jwks.json", "", nil)
	assertStatus(t, http.StatusOK, resp)
	var set map[string]any
	decodeBody(t, resp, &set)
	return set
}

// verifyOffline confirms token's signature actually verifies against the
// public key published in jwks (issue #2's whole point) - not just that
// the JWKS endpoint returns plausible-looking JSON.
func verifyOffline(t *testing.T, jwks map[string]any, token string) {
	t.Helper()
	keys, ok := jwks["keys"].([]any)
	require.True(t, ok, "jwks response has no keys array: %v", jwks)
	require.NotEmpty(t, keys, "jwks response has no keys: %v", jwks)
	key, ok := keys[0].(map[string]any)
	require.True(t, ok, "jwks key is not an object: %v", keys[0])

	x, err := base64.RawURLEncoding.DecodeString(key["x"].(string))
	require.NoError(t, err)
	y, err := base64.RawURLEncoding.DecodeString(key["y"].(string))
	require.NoError(t, err)
	// Reassemble the uncompressed SEC1 point (0x04 || X || Y) - avoids the
	// deprecated ecdsa.PublicKey.X/Y big.Int accessors.
	point := append([]byte{0x04}, append(x, y...)...)
	pub, err := ecdsa.ParseUncompressedPublicKey(elliptic.P256(), point)
	require.NoError(t, err)

	claims := jwt.RegisteredClaims{}
	tok, err := jwt.ParseWithClaims(token, &claims, func(*jwt.Token) (any, error) {
		return pub, nil
	}, jwt.WithValidMethods([]string{"ES256"}))
	require.NoError(t, err)
	require.True(t, tok.Valid)
}

// decodeJWTClaims decodes the JWT payload without verifying its signature -
// this test already confirms end to end that iamd itself signs and returns
// the token; decoding here is only to inspect the claims shape.
func decodeJWTClaims(t *testing.T, token string) map[string]any {
	t.Helper()
	parts := strings.Split(token, ".")
	require.Lenf(t, parts, 3, "not a compact JWT: %s", token)
	payload := parts[1]
	if m := len(payload) % 4; m != 0 {
		payload += strings.Repeat("=", 4-m)
	}
	raw, err := base64.URLEncoding.DecodeString(payload)
	require.NoError(t, err)
	var claims map[string]any
	require.NoError(t, json.Unmarshal(raw, &claims))
	return claims
}
