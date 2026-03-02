package cmd

import (
	"testing"
)

func TestParseServer_HostPort(t *testing.T) {
	host, port, err := parseServer("10.0.0.1:3000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if host != "10.0.0.1" {
		t.Errorf("host: got %q, want %q", host, "10.0.0.1")
	}
	if port != 3000 {
		t.Errorf("port: got %d, want 3000", port)
	}
}

func TestParseServer_HostOnly(t *testing.T) {
	host, port, err := parseServer("myserver")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if host != "myserver" {
		t.Errorf("host: got %q, want %q", host, "myserver")
	}
	if port != 3000 {
		t.Errorf("port: got %d, want 3000 (default)", port)
	}
}

func TestParseServer_Empty(t *testing.T) {
	_, _, err := parseServer("")
	if err == nil {
		t.Fatal("expected error for empty server")
	}
}

func TestParseServer_Whitespace(t *testing.T) {
	_, _, err := parseServer("   ")
	if err == nil {
		t.Fatal("expected error for whitespace-only server")
	}
}

func TestParseServer_CustomPort(t *testing.T) {
	host, port, err := parseServer("example.com:8080")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if host != "example.com" {
		t.Errorf("host: got %q, want %q", host, "example.com")
	}
	if port != 8080 {
		t.Errorf("port: got %d, want 8080", port)
	}
}

func TestParseServer_InvalidPort(t *testing.T) {
	// Invalid port falls back to host-only with default port
	host, port, err := parseServer("example.com:abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if host != "example.com:abc" {
		t.Errorf("host: got %q, want %q", host, "example.com:abc")
	}
	if port != 3000 {
		t.Errorf("port: got %d, want 3000 (default)", port)
	}
}

func TestParseServer_IPv6(t *testing.T) {
	// IPv6 with port: [::1]:3000 - the last colon splits correctly
	host, port, err := parseServer("[::1]:3000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if host != "[::1]" {
		t.Errorf("host: got %q, want %q", host, "[::1]")
	}
	if port != 3000 {
		t.Errorf("port: got %d, want 3000", port)
	}
}

func TestParseServer_Trimmed(t *testing.T) {
	host, port, err := parseServer("  10.0.0.1:3000  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if host != "10.0.0.1" {
		t.Errorf("host: got %q, want %q", host, "10.0.0.1")
	}
	if port != 3000 {
		t.Errorf("port: got %d, want 3000", port)
	}
}

func TestLoginCmd_SchemeDefault(t *testing.T) {
	f := loginCmd.Flags().Lookup("scheme")
	if f == nil {
		t.Fatal("scheme flag not registered")
	}
	if f.DefValue != "http" {
		t.Errorf("scheme default: got %q, want %q", f.DefValue, "http")
	}
}

func TestLoginCmd_Exists(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "login" {
			found = true
			break
		}
	}
	if !found {
		t.Error("login command not registered")
	}
}

func TestLoginCmd_Flags(t *testing.T) {
	flags := []string{"server", "scheme", "username", "password", "context"}
	for _, name := range flags {
		f := loginCmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("flag %q not registered on login command", name)
		}
	}
}
