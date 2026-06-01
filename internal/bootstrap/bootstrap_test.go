package bootstrap

import (
	"os"
	"testing"
)

func TestGetEnvStr(t *testing.T) {
	os.Setenv("TEST_STR", "hello")
	defer os.Unsetenv("TEST_STR")

	if got := GetEnvStr("TEST_STR", "default"); got != "hello" {
		t.Errorf("GetEnvStr = %q, want %q", got, "hello")
	}
	if got := GetEnvStr("TEST_STR_MISSING", "default"); got != "default" {
		t.Errorf("GetEnvStr(missing) = %q, want %q", got, "default")
	}
}

func TestGetEnvInt(t *testing.T) {
	os.Setenv("TEST_INT", "42")
	defer os.Unsetenv("TEST_INT")

	if got := GetEnvInt("TEST_INT", 0); got != 42 {
		t.Errorf("GetEnvInt = %d, want 42", got)
	}
	if got := GetEnvInt("TEST_INT_MISSING", 99); got != 99 {
		t.Errorf("GetEnvInt(missing) = %d, want 99", got)
	}

	os.Setenv("TEST_INT_BAD", "not-a-number")
	defer os.Unsetenv("TEST_INT_BAD")
	if got := GetEnvInt("TEST_INT_BAD", 7); got != 7 {
		t.Errorf("GetEnvInt(bad) = %d, want 7", got)
	}
}

func TestGetEnvBool(t *testing.T) {
	tests := []struct {
		env  string
		val  string
		def  bool
		want bool
	}{
		{"TEST_BOOL_TRUE", "true", false, true},
		{"TEST_BOOL_1", "1", false, true},
		{"TEST_BOOL_FALSE", "false", true, false},
		{"TEST_BOOL_0", "0", true, false},
	}

	for _, tt := range tests {
		os.Setenv(tt.env, tt.val)
		defer os.Unsetenv(tt.env)
		if got := GetEnvBool(tt.env, tt.def); got != tt.want {
			t.Errorf("GetEnvBool(%q=%q) = %v, want %v", tt.env, tt.val, got, tt.want)
		}
	}

	// Missing defaults to def.
	if got := GetEnvBool("TEST_BOOL_MISSING", true); got != true {
		t.Errorf("GetEnvBool(missing) = %v, want true", got)
	}
}

func TestPoolConfigFromEnv(t *testing.T) {
	os.Setenv("DB_MAX_CONNS", "50")
	os.Setenv("DB_MIN_CONNS", "10")
	defer os.Unsetenv("DB_MAX_CONNS")
	defer os.Unsetenv("DB_MIN_CONNS")

	cfg := PoolConfigFromEnv()
	if cfg.MaxConns != 50 {
		t.Errorf("MaxConns = %d, want 50", cfg.MaxConns)
	}
	if cfg.MinConns != 10 {
		t.Errorf("MinConns = %d, want 10", cfg.MinConns)
	}
}
