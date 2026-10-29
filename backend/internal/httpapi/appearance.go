package httpapi

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/alialjaffer/fai/backend/internal/store"
)

const defaultPaletteID = "light-steel"

var paletteIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,79}$`)
var paletteColorPattern = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

type appearanceBody struct {
	GlobalPaletteID   string                `json:"global_palette_id"`
	WorkspacePalettes map[string]string     `json:"workspace_palettes"`
	CustomPalettes    []store.CustomPalette `json:"custom_palettes"`
}

func (s *Server) getAppearance(w http.ResponseWriter, r *http.Request) {
	settings, err := s.Store.GetAppearance(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

func (s *Server) updateAppearance(w http.ResponseWriter, r *http.Request) {
	var body appearanceBody
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	settings, err := normalizeAppearance(body)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	saved, err := s.Store.SaveAppearance(r.Context(), settings)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

func normalizeAppearance(body appearanceBody) (store.AppearanceSettings, error) {
	global := strings.TrimSpace(body.GlobalPaletteID)
	if global == "" {
		global = defaultPaletteID
	}
	if !paletteIDPattern.MatchString(global) {
		return store.AppearanceSettings{}, errInvalidPaletteID
	}

	custom := make([]store.CustomPalette, 0, len(body.CustomPalettes))
	seen := map[string]bool{}
	for _, palette := range body.CustomPalettes {
		id := strings.TrimSpace(palette.ID)
		name := strings.TrimSpace(palette.Name)
		if !strings.HasPrefix(id, "custom-") || !paletteIDPattern.MatchString(id) {
			return store.AppearanceSettings{}, errInvalidPaletteID
		}
		if name == "" || len(name) > 80 {
			return store.AppearanceSettings{}, errInvalidPaletteName
		}
		if seen[id] {
			return store.AppearanceSettings{}, errDuplicatePaletteID
		}
		if len(palette.Colors) < 3 || len(palette.Colors) > 12 {
			return store.AppearanceSettings{}, errInvalidPaletteColors
		}
		colors := make([]string, 0, len(palette.Colors))
		for _, color := range palette.Colors {
			value := strings.TrimSpace(color)
			if !paletteColorPattern.MatchString(value) {
				return store.AppearanceSettings{}, errInvalidPaletteColors
			}
			colors = append(colors, "#"+strings.ToUpper(value[1:]))
		}
		seen[id] = true
		custom = append(custom, store.CustomPalette{ID: id, Name: name, Colors: colors})
	}
	if len(custom) > 50 {
		return store.AppearanceSettings{}, errTooManyPalettes
	}

	workspaces := map[string]string{}
	for rawID, paletteID := range body.WorkspacePalettes {
		workspaceID, err := strconv.ParseInt(strings.TrimSpace(rawID), 10, 64)
		if err != nil || workspaceID <= 0 {
			return store.AppearanceSettings{}, errInvalidWorkspace
		}
		value := strings.TrimSpace(paletteID)
		if !paletteIDPattern.MatchString(value) {
			return store.AppearanceSettings{}, errInvalidPaletteID
		}
		workspaces[strconv.FormatInt(workspaceID, 10)] = value
	}

	refs := append([]string{global}, mapValues(workspaces)...)
	for _, id := range refs {
		if strings.HasPrefix(id, "custom-") && !seen[id] {
			return store.AppearanceSettings{}, errUnknownCustomPalette
		}
	}

	return store.AppearanceSettings{
		GlobalPaletteID:   global,
		WorkspacePalettes: workspaces,
		CustomPalettes:    custom,
	}, nil
}

func mapValues(values map[string]string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, value)
	}
	return out
}

var (
	errInvalidPaletteID     = appearanceError("palette id must be a lowercase slug")
	errInvalidPaletteName   = appearanceError("custom palette name is required")
	errInvalidPaletteColors = appearanceError("custom palettes need 3 to 12 hex colors")
	errDuplicatePaletteID   = appearanceError("custom palette ids must be unique")
	errTooManyPalettes      = appearanceError("too many custom palettes")
	errInvalidWorkspace     = appearanceError("workspace id must be a positive integer")
	errUnknownCustomPalette = appearanceError("custom palette is not defined")
)

type appearanceError string

func (e appearanceError) Error() string { return string(e) }
