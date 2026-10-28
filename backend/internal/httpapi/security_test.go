package httpapi

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRejectNonPublicAddress(t *testing.T) {
	blocked := []string{
		"127.0.0.1:443",
		"10.0.0.7:443",
		"172.16.5.4:443",
		"192.168.1.10:443",
		"169.254.169.254:80",
		"0.0.0.0:443",
		"[::1]:443",
		"[fd00::1]:443",
	}
	for _, address := range blocked {
		if err := rejectNonPublicAddress("tcp", address, nil); err == nil {
			t.Errorf("expected %s to be blocked", address)
		}
	}

	allowed := []string{"1.1.1.1:443", "203.0.113.9:443", "[2606:4700:4700::1111]:443"}
	for _, address := range allowed {
		if err := rejectNonPublicAddress("tcp", address, nil); err != nil {
			t.Errorf("expected %s to be allowed, got %v", address, err)
		}
	}
}

func TestAttachmentsDirSize(t *testing.T) {
	dir := t.TempDir()
	if _, err := attachmentsDirSize(dir); err != nil {
		t.Fatalf("empty dir: %v", err)
	}
	for name, size := range map[string]int{"a": 10, "b": 25} {
		if err := os.WriteFile(filepath.Join(dir, name), make([]byte, size), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	total, err := attachmentsDirSize(dir)
	if err != nil {
		t.Fatal(err)
	}
	if total != 35 {
		t.Fatalf("total = %d, want 35", total)
	}
}
