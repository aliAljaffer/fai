package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/alialjaffer/fai/backend/internal/store"
	"github.com/alialjaffer/fai/backend/internal/store/lite"
)

func TestAppearancePersistsAndValidates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fai.db")
	st, err := lite.New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	server := httptest.NewServer(NewRouter(&Server{Store: st}))
	defer server.Close()

	res := sendJSON(t, http.MethodGet, server.URL+"/api/appearance", "")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("get status = %d", res.StatusCode)
	}
	var appearance store.AppearanceSettings
	if err := json.NewDecoder(res.Body).Decode(&appearance); err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if appearance.GlobalPaletteID != "light-steel" {
		t.Fatalf("default global = %q", appearance.GlobalPaletteID)
	}
	if len(appearance.WorkspacePalettes) != 0 || len(appearance.CustomPalettes) != 0 {
		t.Fatalf("default appearance = %#v", appearance)
	}

	body := `{
		"global_palette_id":"olive-garden-feast",
		"workspace_palettes":{"1":"custom-studio-ab12"},
		"custom_palettes":[{
			"id":"custom-studio-ab12",
			"name":"Studio",
			"colors":["#F4EBD0","#D9C5A0","#9C6B4A","#543A2A","#1E1712"]
		}]
	}`
	res = sendJSON(t, http.MethodPut, server.URL+"/api/appearance", body)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("put status = %d", res.StatusCode)
	}
	if err := json.NewDecoder(res.Body).Decode(&appearance); err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if appearance.GlobalPaletteID != "olive-garden-feast" {
		t.Fatalf("saved global = %q", appearance.GlobalPaletteID)
	}
	if appearance.WorkspacePalettes["1"] != "custom-studio-ab12" {
		t.Fatalf("saved workspace = %#v", appearance.WorkspacePalettes)
	}
	if len(appearance.CustomPalettes) != 1 || appearance.CustomPalettes[0].Name != "Studio" {
		t.Fatalf("saved custom = %#v", appearance.CustomPalettes)
	}

	res = sendJSON(t, http.MethodGet, server.URL+"/api/appearance", "")
	var reloaded store.AppearanceSettings
	if err := json.NewDecoder(res.Body).Decode(&reloaded); err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if reloaded.GlobalPaletteID != "olive-garden-feast" ||
		reloaded.WorkspacePalettes["1"] != "custom-studio-ab12" {
		t.Fatalf("reloaded appearance = %#v", reloaded)
	}

	res = sendJSON(t, http.MethodPut, server.URL+"/api/appearance",
		`{"global_palette_id":"light-steel","workspace_palettes":{},"custom_palettes":[]}`)
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("reset status = %d", res.StatusCode)
	}

	res = sendJSON(t, http.MethodPut, server.URL+"/api/appearance",
		`{"global_palette_id":"missing-theme","workspace_palettes":{},"custom_palettes":[]}`)
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("missing built-in should be accepted status = %d", res.StatusCode)
	}

	res = sendJSON(t, http.MethodPut, server.URL+"/api/appearance",
		`{"global_palette_id":"custom-missing","workspace_palettes":{},"custom_palettes":[]}`)
	res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("unknown custom palette status = %d, want 400", res.StatusCode)
	}

	res = sendJSON(t, http.MethodPut, server.URL+"/api/appearance",
		`{"global_palette_id":"light-steel","workspace_palettes":{},"custom_palettes":[{"id":"custom-bad","name":"Bad","colors":["#fff"]}]}`)
	res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("invalid color status = %d, want 400", res.StatusCode)
	}
}
