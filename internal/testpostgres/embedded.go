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

// Instance owns one local PostgreSQL 17 process and its temporary data paths.
// It is for explicit test and release-verification commands only.
type Instance struct {
	database *embeddedpostgres.EmbeddedPostgres
	dsn      string
}

// Start launches PostgreSQL 17 using only paths nested under root. The caller
// owns root and must close the returned instance before deleting it.
func Start(root string) (*Instance, error) {
	if strings.TrimSpace(root) == "" {
		return nil, fmt.Errorf("embedded PostgreSQL root is required")
	}
	port, err := unusedLoopbackPort()
	if err != nil {
		return nil, fmt.Errorf("allocate embedded PostgreSQL port: %w", err)
	}
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
	database := embeddedpostgres.NewDatabase(config)
	if err := database.Start(); err != nil {
		return nil, fmt.Errorf("start embedded PostgreSQL 17: %w", err)
	}
	dsn, err := withSSLDisabled(config.GetConnectionURL())
	if err != nil {
		_ = database.Stop()
		return nil, fmt.Errorf("construct embedded PostgreSQL DSN: %w", err)
	}
	return &Instance{database: database, dsn: dsn}, nil
}

// DSN is the locally scoped connection string. It is intentionally not logged.
func (instance *Instance) DSN() string {
	if instance == nil {
		return ""
	}
	return instance.dsn
}

// Close stops the owned PostgreSQL process.
func (instance *Instance) Close() error {
	if instance == nil || instance.database == nil {
		return nil
	}
	return instance.database.Stop()
}

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

	instance, err := Start(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := instance.Close(); err != nil {
			t.Errorf("stop embedded PostgreSQL 17: %v", err)
		}
	})
	return instance.DSN()
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
