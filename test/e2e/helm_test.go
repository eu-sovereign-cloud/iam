//go:build e2e

package e2e

import (
	"bufio"
	"encoding/json"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestEndToEndHelm is a second, independent e2e test alongside
// TestEndToEnd: instead of running iamd as a local Go subprocess, it
// builds the real Docker image, loads it into a local kind cluster, and
// installs the real Helm chart (doc/adr/0020) - the first thing that
// actually exercises the chart's RBAC (does the ServiceAccount really have
// what internal/adapter/kuberbac needs?), Service, and liveness/readiness
// probes end to end, not just via `helm template`/a dry-run.
//
// It only runs against a local kind cluster: a locally-built image has
// nowhere else to be loaded without a registry push, and this must never
// be pointed at a real shared cluster (see requireKindCluster).
const (
	helmNamespace   = "iam-e2e-helm"
	helmRelease     = "iam-e2e"
	helmChartPath   = "../../deploy/helm/iam"
	helmImage       = "iam-e2e"
	helmImageTag    = "test"
	helmForwardAddr = "127.0.0.1:18082"
	helmDeployment  = "deploy/" + helmRelease + "-iam"
	helmService     = "svc/" + helmRelease + "-iam"
)

func TestEndToEndHelm(t *testing.T) {
	requireKubectl(t)
	kindCluster := requireKindCluster(t)
	requireTool(t, "docker")
	requireTool(t, "helm")
	requireTool(t, "kind")

	t.Cleanup(func() {
		_ = exec.Command("kubectl", "delete", "namespace", helmNamespace, "--ignore-not-found").Run()
	})

	applyCRDs(t)

	buildAndLoadImage(t, kindCluster)
	helmInstall(t)

	adminPAT := fetchBootstrapPATFromLogs(t)
	forward := startPortForward(t)
	withBaseURL(t, "http://"+helmForwardAddr)

	// --- Core flow, same shape as TestEndToEnd, now against a real
	// Helm-deployed pod behind a real Service ---

	createTenant(t, adminPAT, "tenant-1", "Tenant One")
	createUser(t, adminPAT, "alice@example.com", "Alice", false)
	createGrant(t, adminPAT, "alice@example.com", "tenant-1", "member")

	aliceID, alicePAT := createPAT(t, adminPAT, "alice@example.com", "laptop")

	claims := decodeJWTClaims(t, alicePAT)
	require.Equal(t, "alice@example.com", claims["sub"])
	require.Equal(t, []any{"tenant-1"}, claims["tenants"])

	doc := fetchDiscoveryDocument(t)
	require.Equal(t, "https://iam.e2e-test.local", doc["issuer"])
	verifyOffline(t, fetchJWKS(t), alicePAT)

	userinfoResp := doJSON(t, http.MethodGet, "/userinfo", alicePAT, nil)
	assertStatus(t, http.StatusOK, userinfoResp)

	assertStatus(t, http.StatusNoContent, doJSON(t, http.MethodDelete, "/api/v1/users/alice@example.com/pats/"+aliceID, alicePAT, nil))
	assertStatus(t, http.StatusUnauthorized, doJSON(t, http.MethodGet, "/api/v1/users/alice@example.com/pats", alicePAT, nil))
	assertStatus(t, http.StatusUnauthorized, doJSON(t, http.MethodGet, "/userinfo", alicePAT, nil))

	// --- Pod restart: proves ADR 0009's load-at-startup state survives a
	// real Kubernetes pod eviction/reschedule, not just a local SIGTERM
	// (already covered by TestEndToEnd), and that the signing key/state -
	// held only in Secrets/ConfigMaps via the API, never local disk (ADR
	// 0020) - really do come back on a fresh pod. ---

	forward.stop()
	kubectlRun(t, "delete", "pod", "-n", helmNamespace, "-l", "app.kubernetes.io/instance="+helmRelease)
	kubectlRun(t, "rollout", "status", helmDeployment, "-n", helmNamespace, "--timeout=60s")
	startPortForward(t) // cleanup self-registered; forward above already stopped
	withBaseURL(t, "http://"+helmForwardAddr)

	tenants := listTenants(t, adminPAT)
	require.Len(t, tenants, 1)
	require.Equal(t, "tenant-1", tenants[0]["tenantId"])

	// --- Vendored ecp CRD compatibility check, same as TestEndToEnd ---
	applyDemoRoleAssignment(t, "alice@example.com", "tenant-1")
}

func requireTool(t *testing.T, name string) {
	t.Helper()
	if _, err := exec.LookPath(name); err != nil {
		t.Skipf("%s not found in PATH: %v; skipping TestEndToEndHelm", name, err)
	}
}

// requireKindCluster returns the current kind cluster's name, or skips the
// test if the current kubeconfig context isn't a kind cluster. A
// locally-built image has no registry to be pushed to, so this test can
// only ever target kind - it must never run against a real shared cluster.
func requireKindCluster(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("kubectl", "config", "current-context").Output()
	require.NoError(t, err)
	ctx := strings.TrimSpace(string(out))
	name, ok := strings.CutPrefix(ctx, "kind-")
	if !ok {
		t.Skipf("TestEndToEndHelm requires a local kind cluster (current context %q is not kind); skipping", ctx)
	}
	return name
}

func buildAndLoadImage(t *testing.T, kindCluster string) {
	t.Helper()
	image := helmImage + ":" + helmImageTag
	out, err := exec.Command("docker", "build", "-t", image, "../..").CombinedOutput() //nolint:gosec // test-only, fixed literal args
	require.NoErrorf(t, err, "docker build: %s", out)

	out, err = exec.Command("kind", "load", "docker-image", image, "--name", kindCluster).CombinedOutput() //nolint:gosec // test-only, fixed literal args
	require.NoErrorf(t, err, "kind load docker-image: %s", out)
}

func helmInstall(t *testing.T) {
	t.Helper()
	args := []string{
		"install", helmRelease, helmChartPath,
		"--namespace", helmNamespace, "--create-namespace",
		"--wait", "--timeout", "90s",
		"--set", "image.repository=" + helmImage,
		"--set", "image.tag=" + helmImageTag,
		"--set", "image.pullPolicy=Never",
		"--set", "config.jwtIssuer=https://iam.e2e-test.local",
		"--set", "config.jwtAudience[0]=ecp-gateway",
	}
	out, err := exec.Command("helm", args...).CombinedOutput() //nolint:gosec // test-only, fixed literal args
	require.NoErrorf(t, err, "helm install: %s", out)

	t.Cleanup(func() {
		_ = exec.Command("helm", "uninstall", helmRelease, "-n", helmNamespace).Run()
	})
}

// fetchBootstrapPATFromLogs parses iamd's one-time bootstrap admin PAT log
// line (ADR 0006) out of the pod's logs, retrying briefly since helm
// install --wait only guarantees the readiness probe passed, not that
// every earlier startup log line has been flushed/scraped yet.
func fetchBootstrapPATFromLogs(t *testing.T) string {
	t.Helper()
	deadline := time.After(15 * time.Second)
	for {
		out, err := exec.Command("kubectl", "logs", "-n", helmNamespace, helmDeployment).Output() //nolint:gosec // test-only, fixed literal args
		if err == nil {
			scanner := bufio.NewScanner(strings.NewReader(string(out)))
			for scanner.Scan() {
				var entry map[string]any
				if json.Unmarshal(scanner.Bytes(), &entry) != nil {
					continue
				}
				if pat, ok := entry["pat"].(string); ok && pat != "" {
					return pat
				}
			}
		}
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for bootstrap admin PAT in pod logs (last error: %v)", err)
		case <-time.After(500 * time.Millisecond):
		}
	}
}

// portForward wraps the background `kubectl port-forward` process. stop is
// idempotent (sync.Once-guarded) since this test both stops it explicitly
// mid-test (to restart it around the pod-restart check) and relies on
// t.Cleanup for the final one - calling exec.Cmd.Wait twice would panic.
type portForward struct {
	cmd  *exec.Cmd
	once sync.Once
}

func (p *portForward) stop() {
	p.once.Do(func() {
		if p.cmd == nil || p.cmd.Process == nil {
			return
		}
		_ = p.cmd.Process.Kill()
		_ = p.cmd.Wait()
	})
}

// startPortForward starts `kubectl port-forward` to the Helm-deployed
// Service and waits until the local end accepts connections.
func startPortForward(t *testing.T) *portForward {
	t.Helper()
	cmd := exec.Command("kubectl", "port-forward", "-n", helmNamespace, helmService, //nolint:gosec // test-only, fixed literal args
		strings.TrimPrefix(helmForwardAddr, "127.0.0.1:")+":8080")
	require.NoError(t, cmd.Start())
	pf := &portForward{cmd: cmd}
	t.Cleanup(pf.stop)
	waitForListening(t, helmForwardAddr)
	return pf
}
