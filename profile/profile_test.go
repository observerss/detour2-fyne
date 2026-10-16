package profile

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/observerss/detour2-fyne/utils"
)

func withBuiltins(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	oldName, oldProfs := DefaultProfileName, builtinProfiles
	DefaultProfileName = "hk"
	builtinProfiles = []Profile{
		{Name: "hk", RemoteAddress: "tcp://a:1", Password: "p"},
		{Name: "jp", RemoteAddress: "tcp://b:2", Password: "p"},
	}
	t.Cleanup(func() { DefaultProfileName, builtinProfiles = oldName, oldProfs })
}

func TestFirstRunSeedsBuiltinsOnce(t *testing.T) {
	withBuiltins(t)
	profs, err := LoadProfiles()
	if err != nil || len(profs) != 2 || profs["hk"].RemoteAddress != "tcp://a:1" {
		t.Fatalf("first run = %v, %v", profs, err)
	}
	profs["hk"].RemoteAddress = "changed" // must not leak into the built-ins
	if DefaultProfiles()["hk"].RemoteAddress != "tcp://a:1" {
		t.Fatal("built-in line was mutated")
	}

	delete(profs, "hk")
	delete(profs, "jp")
	if err := SaveProfiles(profs); err != nil {
		t.Fatal(err)
	}
	profs, err = LoadProfiles()
	if err != nil || len(profs) != 0 {
		t.Fatalf("deleted built-ins came back: %v, %v", profs, err)
	}
}

func TestLegacyDefaultProfileKept(t *testing.T) {
	withBuiltins(t)
	path, _ := GetProfilePath()
	os.WriteFile(path, []byte(`{"default":{"name":"default","remoteAddress":"tcp://x:1","password":"p"}}`), 0o600)
	profs, err := LoadProfiles()
	if err != nil || profs["default"] == nil || profs["hk"] != nil {
		t.Fatalf("legacy profile = %v, %v", profs, err)
	}
}

func TestLoadCorruptedReturnsErrorAndFreshMap(t *testing.T) {
	withBuiltins(t)
	path, _ := GetProfilePath()
	os.WriteFile(path, []byte("{broken"), 0o600)
	first, err := LoadProfiles()
	if err == nil {
		t.Fatal("expected error")
	}
	first["x"] = &Profile{Name: "x"}
	if second, _ := LoadProfiles(); len(second) != 0 {
		t.Fatalf("error result shares state: %v", second)
	}
}

func TestSavePermissions(t *testing.T) {
	withBuiltins(t)
	if err := SaveProfiles(DefaultProfiles()); err != nil {
		t.Fatal(err)
	}
	path, _ := GetProfilePath()
	if st, err := os.Stat(path); err != nil || st.Mode().Perm() != 0o600 {
		t.Fatalf("stat = %v, %v", st, err)
	}
}

func TestSetCurrentProfileNameKeepsOtherFields(t *testing.T) {
	withBuiltins(t)
	path, _ := utils.AppConfigFile("run.json")
	os.WriteFile(path, []byte(`{"profileName":"hk","poolSize":8,"socks5Port":"1080"}`), 0o600)
	if err := SetCurrentProfileName("jp"); err != nil {
		t.Fatal(err)
	}
	content, _ := os.ReadFile(path)
	state := map[string]any{}
	json.Unmarshal(content, &state)
	if state["profileName"] != "jp" || state["poolSize"] != float64(8) || state["socks5Port"] != "1080" {
		t.Fatalf("run.json = %s", content)
	}
}
