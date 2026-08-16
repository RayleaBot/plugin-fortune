package plugin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"text/template"
)

var stylesheetURLPattern = regexp.MustCompile(`url\(\s*["']?([^"')]+)["']?\s*\)`)

func TestRenderTemplatePackageIsComplete(t *testing.T) {
	pluginRoot := filepath.Clean(filepath.Join("..", ".."))
	manifestBytes, err := os.ReadFile(filepath.Join(pluginRoot, "info.json"))
	if err != nil {
		t.Fatalf("read plugin manifest: %v", err)
	}
	var manifest struct {
		RenderTemplates []struct {
			Path string `json:"path"`
		} `json:"render_templates"`
	}
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatalf("decode plugin manifest: %v", err)
	}

	wantIDs := map[string]string{
		"templates/card":  "card",
		"templates/stats": "stats",
	}
	if len(manifest.RenderTemplates) != len(wantIDs) {
		t.Fatalf("render template declarations = %d, want %d", len(manifest.RenderTemplates), len(wantIDs))
	}
	for _, declaration := range manifest.RenderTemplates {
		wantID, ok := wantIDs[declaration.Path]
		if !ok {
			t.Fatalf("unexpected render template path %q", declaration.Path)
		}
		validateRenderTemplatePackage(t, pluginRoot, declaration.Path, wantID)
	}
}

func validateRenderTemplatePackage(t *testing.T, pluginRoot, relativeDir, wantID string) {
	t.Helper()
	templateDir := filepath.Join(pluginRoot, filepath.FromSlash(relativeDir))
	manifestPath := filepath.Join(templateDir, "template.json")
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read %s: %v", manifestPath, err)
	}
	var manifest struct {
		ID          string `json:"id"`
		EntryHTML   string `json:"entry_html"`
		Stylesheet  string `json:"stylesheet"`
		InputSchema string `json:"input_schema"`
	}
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatalf("decode %s: %v", manifestPath, err)
	}
	if manifest.ID != wantID {
		t.Fatalf("template id = %q, want %q", manifest.ID, wantID)
	}

	for _, name := range []string{manifest.EntryHTML, manifest.Stylesheet, manifest.InputSchema} {
		if strings.TrimSpace(name) == "" {
			t.Fatalf("template %s has an empty required file path", wantID)
		}
		if _, err := os.Stat(filepath.Join(templateDir, filepath.FromSlash(name))); err != nil {
			t.Fatalf("template %s file %q: %v", wantID, name, err)
		}
	}

	htmlBytes, err := os.ReadFile(filepath.Join(templateDir, filepath.FromSlash(manifest.EntryHTML)))
	if err != nil {
		t.Fatalf("read template %s HTML: %v", wantID, err)
	}
	if _, err := template.New(wantID).Parse(string(htmlBytes)); err != nil {
		t.Fatalf("parse template %s HTML: %v", wantID, err)
	}

	stylesheetBytes, err := os.ReadFile(filepath.Join(templateDir, filepath.FromSlash(manifest.Stylesheet)))
	if err != nil {
		t.Fatalf("read template %s stylesheet: %v", wantID, err)
	}
	for _, match := range stylesheetURLPattern.FindAllStringSubmatch(string(stylesheetBytes), -1) {
		reference := strings.TrimSpace(match[1])
		if reference == "" || strings.HasPrefix(reference, "data:") || strings.Contains(reference, "://") {
			continue
		}
		assetPath := filepath.Clean(filepath.Join(templateDir, filepath.FromSlash(reference)))
		relative, err := filepath.Rel(pluginRoot, assetPath)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			t.Fatalf("template %s asset %q escapes the plugin package", wantID, reference)
		}
		if _, err := os.Stat(assetPath); err != nil {
			t.Fatalf("template %s asset %q: %v", wantID, reference, err)
		}
	}
}
