package handler

import (
	"net/http"
	"reflect"
	"testing"

	"golog/entity"
	"golog/system"

	"github.com/gin-gonic/gin"
)

func TestWebAuthnRelyingPartyUsesConfiguredOrigin(t *testing.T) {
	oldConfig := system.Config
	system.Config = &entity.Config{
		WebAuthnOrigins: []string{"https://example.com/admin"},
	}
	defer func() {
		system.Config = oldConfig
	}()

	rpid, origins, err := webAuthnRelyingParty(&gin.Context{
		Request: &http.Request{
			Host: "attacker.example",
		},
	})
	if err != nil {
		t.Fatalf("webAuthnRelyingParty returned error: %v", err)
	}
	if rpid != "example.com" {
		t.Fatalf("rpid = %q, want %q", rpid, "example.com")
	}
	if want := []string{"https://example.com"}; !reflect.DeepEqual(origins, want) {
		t.Fatalf("origins = %#v, want %#v", origins, want)
	}
}

func TestWebAuthnRelyingPartyFallsBackToRequestOrigin(t *testing.T) {
	oldConfig := system.Config
	system.Config = &entity.Config{}
	defer func() {
		system.Config = oldConfig
	}()

	req := &http.Request{
		Host:   "localhost:8080",
		Header: http.Header{"X-Forwarded-Proto": []string{"https"}},
	}
	rpid, origins, err := webAuthnRelyingParty(&gin.Context{Request: req})
	if err != nil {
		t.Fatalf("webAuthnRelyingParty returned error: %v", err)
	}
	if rpid != "localhost" {
		t.Fatalf("rpid = %q, want %q", rpid, "localhost")
	}
	if want := []string{"https://localhost:8080"}; !reflect.DeepEqual(origins, want) {
		t.Fatalf("origins = %#v, want %#v", origins, want)
	}
}
