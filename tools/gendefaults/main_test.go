package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseLines(t *testing.T) {
	lines, err := parseLines(" hk=tcp://a:1 ; jp=tcp://b:2,tcp://c:3;")
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 2 || lines[0] != (line{"hk", "tcp://a:1"}) || lines[1] != (line{"jp", "tcp://b:2,tcp://c:3"}) {
		t.Fatalf("lines = %+v", lines)
	}
	for _, bad := range []string{"hk", "=tcp://a:1", "hk=", "hk=a;hk=b"} {
		if _, err := parseLines(bad); err == nil {
			t.Errorf("parseLines(%q) succeeded", bad)
		}
	}
}

func TestRun(t *testing.T) {
	dir := t.TempDir()
	config := filepath.Join(dir, "build.env")
	out := filepath.Join(dir, "defaults_gen.go")
	write := func(s string) {
		if err := os.WriteFile(config, []byte(s), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	write("DETOUR2_DEFAULT_PROFILES=hk=tcp://a:1;jp=tcp://b:2\nDETOUR2_DEFAULT_PROFILE=jp\nDETOUR2_DEFAULT_PASSWORD=\"p\\\"q\"\n")
	if err := run(config, out, "profile"); err != nil {
		t.Fatal(err)
	}
	src, _ := os.ReadFile(out)
	for _, want := range []string{`DefaultProfileName = "jp"`, `Name: "hk", RemoteAddress: "tcp://a:1", Password: "p\"q"`} {
		if !strings.Contains(string(src), want) {
			t.Errorf("missing %s in:\n%s", want, src)
		}
	}

	t.Setenv(keyDefault, "xx")
	if err := run(config, out, "profile"); err == nil {
		t.Error("unknown default line accepted")
	}
	t.Setenv(keyDefault, "")
	write("DETOUR2_DEFAULT_PROFILES=hk=tcp://a:1\n")
	if err := run(config, out, "profile"); err == nil {
		t.Error("lines without password accepted")
	}

	os.Remove(config) // no config: builds without built-in lines
	t.Setenv(keyProfiles, "")
	if err := run(config, out, "profile"); err != nil {
		t.Fatal(err)
	}
	src, _ = os.ReadFile(out)
	if !strings.Contains(string(src), `DefaultProfileName = ""`) || strings.Contains(string(src), "Name:") {
		t.Errorf("expected empty defaults:\n%s", src)
	}
}
