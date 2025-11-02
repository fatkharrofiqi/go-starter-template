package env

import (
    "os"
    "path/filepath"
    "testing"
    "time"

    "github.com/stretchr/testify/require"
)

// TestConfig_Getters verifies that the simple getters for secrets and expirations
// return the expected values based on the Config struct.
func TestConfig_Getters(t *testing.T) {
    cfg := &Config{}

    // Set secrets
    cfg.JWT.Secret = "access-secret"
    cfg.JWT.RefreshSecret = "refresh-secret"
    cfg.JWT.CsrfSecret = "csrf-secret"

    // Set expirations in seconds (as durations), getters multiply by time.Second
    cfg.JWT.AccessTokenExpiration = time.Duration(15)
    cfg.JWT.RefreshTokenExpiration = time.Duration(30)
    cfg.JWT.CsrfTokenExpiration = time.Duration(10)

    // Validate secrets
    require.Equal(t, "access-secret", cfg.GetAccessSecret())
    require.Equal(t, "refresh-secret", cfg.GetRefreshSecret())
    require.Equal(t, "csrf-secret", cfg.GetCsrfSecret())

    // Validate expirations
    require.Equal(t, 15*time.Second, cfg.GetAccessTokenExpiration())
    require.Equal(t, 30*time.Second, cfg.GetRefreshTokenExpiration())
    require.Equal(t, 10*time.Second, cfg.GetCsrfTokenExpiration())
}

// TestNewConfig_Success ensures NewConfig reads a YAML file and unmarshals correctly.
func TestNewConfig_Success(t *testing.T) {
    // Prepare a temporary working directory with a valid config.yml
    tmp := t.TempDir()
    yml := []byte(`
app:
  name: TestApp
web:
  port: 8088
  prefork: false
  cors:
    allow_origins: "*"
jwt:
  secret: "access"
  csrf_secret: "csrf"
  refresh_secret: "refresh"
  csrf_token_expiration: 10
  access_token_expiration: 20
  refresh_token_expiration: 30
redis:
  address: "localhost:6379"
  password: ""
  db: 0
  pool:
    size: 10
    min_idle: 1
    max_idle: 5
    lifetime: 60
    idle_timeout: 30
log:
  level: 4
database:
  dsn: "postgres://user:pass@localhost/db"
  pool:
    idle: 1
    max: 5
    lifetime: 60
  log:
    level: 2
monitoring:
  otel:
    host: "http://localhost:4317"
`)
    require.NoError(t, os.WriteFile(filepath.Join(tmp, "config.yml"), yml, 0644))

    // Switch to the temp dir where ./config.yml exists
    cwd, _ := os.Getwd()
    require.NoError(t, os.Chdir(tmp))
    defer os.Chdir(cwd)

    cfg := NewConfig()
    require.NotNil(t, cfg)
    require.Equal(t, "TestApp", cfg.App.Name)
    require.Equal(t, 8088, cfg.Web.Port)
    require.Equal(t, "access", cfg.GetAccessSecret())
    require.Equal(t, 20*time.Second, cfg.GetAccessTokenExpiration())
    require.Equal(t, "http://localhost:4317", cfg.Monitoring.Otel.Host)
}

// TestNewConfig_PanicWhenMissingFile ensures NewConfig panics when no config file is found.
func TestNewConfig_PanicWhenMissingFile(t *testing.T) {
    tmp := t.TempDir()
    cwd, _ := os.Getwd()
    require.NoError(t, os.Chdir(tmp))
    defer os.Chdir(cwd)

    require.Panics(t, func() { _ = NewConfig() })
}

// TestNewConfig_PanicOnUnmarshal ensures NewConfig panics when config has invalid types.
func TestNewConfig_PanicOnUnmarshal(t *testing.T) {
    tmp := t.TempDir()
    // Invalid type for web.port (string instead of int) to force unmarshal error
    bad := []byte(`
app:
  name: Broken
web:
  port: "oops"
`)
    require.NoError(t, os.WriteFile(filepath.Join(tmp, "config.yml"), bad, 0644))

    cwd, _ := os.Getwd()
    require.NoError(t, os.Chdir(tmp))
    defer os.Chdir(cwd)

    require.Panics(t, func() { _ = NewConfig() })
}