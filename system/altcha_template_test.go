package system

import (
	"bytes"
	"testing"

	"golog/entity"
)

// TestAltchaStandaloneRender verifies the altcha page renders as a standalone
// document, not inheriting template.html.
func TestAltchaStandaloneRender(t *testing.T) {
	for _, theme := range []string{"default", "note"} {
		t.Run(theme, func(t *testing.T) {
			Config = &entity.Config{Theme: theme, Locale: "zh-CN"}
			SetConfigWriter(func(*entity.Config) error { return nil })
			if err := SaveConfig(); err != nil {
				t.Fatalf("SaveConfig: %v", err)
			}
			var buf bytes.Buffer
			if err := PowTmpl.Execute(&buf, map[string]any{
				"Config":      Config,
				"CSRF":        "csrf-token",
				"PowRedirect": "/post/hello",
			}); err != nil {
				t.Fatalf("execute: %v", err)
			}
			out := buf.Bytes()
			if !bytes.Contains(out, []byte("<!DOCTYPE html>")) {
				t.Fatalf("expected standalone doctype, got:\n%s", out)
			}
			if !bytes.Contains(out, []byte("altcha-loading-text")) {
				t.Fatalf("expected loading text, got:\n%s", out)
			}
		})
	}
}
