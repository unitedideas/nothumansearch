// Package testpostgres starts an explicit, disposable PostgreSQL instance for
// release-regression tests when an operator has opted in locally.
package testpostgres

import (
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
)

// EmbeddedOptInEnvironment enables a local real-PostgreSQL fixture only when
// its value is exactly "1". An explicit database DSN always takes precedence.
const EmbeddedOptInEnvironment = "NHS_EMBEDDED_POSTGRES"

// DSN returns an explicitly configured disposable PostgreSQL DSN, or starts
// a temporary PostgreSQL 17 instance when NHS_EMBEDDED_POSTGRES=1. It returns
// an empty string when neither path was selected so existing release tests can
// remain opt-in and never download a binary during ordinary go test runs.
func DSN(t testing.TB, environment string) string {
	t.Helper()

	if dsn := strings.TrimSpace(os.Getenv(environment)); dsn != "" {
		return dsn
	}
	if os.Getenv(EmbeddedOptInEnvironment) != "1" {
		return ""
	}

	port, err := unusedLoopbackPort()
	if err != nil {
		t.Fatalf("allocate embedded PostgreSQL port: %v", err)
	}
	root := t.TempDir()
	config := embeddedpostgres.DefaultConfig().
		Version(embeddedpostgres.V17).
		Port(port).
		Database("nhs_release_test").
		RuntimePath(filepath.Join(root, "runtime")).
		BinariesPath(filepath.Join(root, "binaries")).
		DataPath(filepath.Join(root, "data")).
		CachePath(filepath.Join(root, "cache")).
		Logger(io.Discard).
		StartTimeout(90 * time.Second)
	instance := embeddedpostgres.NewDatabase(config)
	if err := instance.Start(); err != nil {
		t.Fatalf("start embedded PostgreSQL 17: %v", err)
	}
	t.Cleanup(func() {
		if err := instance.Stop(); err != nil {
			t.Errorf("stop embedded PostgreSQL 17: %v", err)
		}
	})
	dsn, err := withSSLDisabled(config.GetConnectionURL())
	if err != nil {
		t.Fatalf("construct embedded PostgreSQL DSN: %v", err)
	}
	return dsn
}

func withSSLDisabled(dsn string) (string, error) {
	parsed, err := url.Parse(dsn)
	if err != nil {
		return "", err
	}
	query := parsed.Query()
	query.Set("sslmode", "disable")
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func unusedLoopbackPort() (uint32, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	port := listener.Addr().(*net.TCPAddr).Port
	if port <= 0 {
		return 0, fmt.Errorf("kernel returned invalid port %d", port)
	}
	return uint32(port), nil
}
