package dingtalk

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/nextlevelbuilder/goclaw/internal/bus"
	"github.com/nextlevelbuilder/goclaw/internal/channels"
)

func TestFactory_RequiresClientID(t *testing.T) {
	_, err := Factory("ding-1", json.RawMessage(`{"client_secret":"s"}`), nil, bus.New(), nil)
	if err == nil || !strings.Contains(err.Error(), "client_id") {
		t.Fatalf("expected client_id required error, got %v", err)
	}
}

func TestFactory_RequiresClientSecret(t *testing.T) {
	_, err := Factory("ding-1", json.RawMessage(`{"client_id":"id"}`), nil, bus.New(), nil)
	if err == nil || !strings.Contains(err.Error(), "client_secret") {
		t.Fatalf("expected client_secret required error, got %v", err)
	}
}

func TestFactory_BuildsChannel(t *testing.T) {
	ch, err := Factory("ding-1",
		json.RawMessage(`{"client_id":"id","client_secret":"sec"}`),
		json.RawMessage(`{"robot_code":"rc","group_policy":"mention_only"}`),
		bus.New(), nil)
	if err != nil {
		t.Fatalf("Factory returned error: %v", err)
	}
	if ch.Name() != "ding-1" {
		t.Fatalf("Name=%q want ding-1", ch.Name())
	}
	if ch.Type() != channels.TypeDingTalk {
		t.Fatalf("Type=%q want %q", ch.Type(), channels.TypeDingTalk)
	}
	if ch.IsRunning() {
		t.Fatalf("expected not running before Start")
	}
}

func TestStartStopRoundtrip(t *testing.T) {
	ch, err := Factory("ding-1",
		json.RawMessage(`{"client_id":"id","client_secret":"sec"}`),
		nil, bus.New(), nil)
	if err != nil {
		t.Fatalf("Factory: %v", err)
	}
	ctx := context.Background()
	if err := ch.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if !ch.IsRunning() {
		t.Fatalf("expected running after Start")
	}
	if err := ch.Stop(ctx); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if ch.IsRunning() {
		t.Fatalf("expected not running after Stop")
	}
}
