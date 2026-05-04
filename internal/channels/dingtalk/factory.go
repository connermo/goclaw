// Package dingtalk implements a DingTalk Stream Mode channel.
//
// Status: SKELETON — the package compiles, registers with the channel manager,
// and validates configuration, but Stream connect/handle/reply paths are
// intentionally stubbed. To finish, integrate
// github.com/open-dingtalk/dingtalk-stream-sdk-go and replace the TODO blocks
// in client.go and handler.go.
//
// Why DingTalk Stream Mode (vs webhook): the user-facing requirement is a
// long-lived bidirectional channel that does not require GoClaw to expose a
// public webhook URL. Stream mode opens an outbound WSS to DingTalk's edge,
// authenticated by client_id+client_secret. As a long-poll-equivalent
// long-running consumer it must run on a singleton (worker pod, replicas=1)
// — multiple streams using the same credentials would race events.
package dingtalk

import (
	"encoding/json"
	"fmt"

	"github.com/nextlevelbuilder/goclaw/internal/bus"
	"github.com/nextlevelbuilder/goclaw/internal/channels"
	"github.com/nextlevelbuilder/goclaw/internal/store"
)

// creds maps the encrypted credentials column on channel_instances.
type creds struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

// instanceConfig maps the plaintext config JSONB on channel_instances.
type instanceConfig struct {
	RobotCode   string   `json:"robot_code,omitempty"`
	GroupPolicy string   `json:"group_policy,omitempty"` // mention_only | open
	DMPolicy    string   `json:"dm_policy,omitempty"`
	AllowFrom   []string `json:"allow_from,omitempty"`
	BlockReply  *bool    `json:"block_reply,omitempty"`
}

// Factory creates a DingTalk channel from DB instance data.
func Factory(name string, credsJSON json.RawMessage, cfgJSON json.RawMessage,
	msgBus *bus.MessageBus, pairingSvc store.PairingStore) (channels.Channel, error) {

	var c creds
	if len(credsJSON) > 0 {
		if err := json.Unmarshal(credsJSON, &c); err != nil {
			return nil, fmt.Errorf("decode dingtalk credentials: %w", err)
		}
	}
	if c.ClientID == "" {
		return nil, fmt.Errorf("dingtalk client_id is required")
	}
	if c.ClientSecret == "" {
		return nil, fmt.Errorf("dingtalk client_secret is required")
	}

	var ic instanceConfig
	if len(cfgJSON) > 0 {
		if err := json.Unmarshal(cfgJSON, &ic); err != nil {
			return nil, fmt.Errorf("decode dingtalk config: %w", err)
		}
	}

	ch, err := New(Config{
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
		RobotCode:    ic.RobotCode,
		GroupPolicy:  ic.GroupPolicy,
		DMPolicy:     ic.DMPolicy,
		AllowFrom:    ic.AllowFrom,
		BlockReply:   ic.BlockReply,
	}, msgBus, pairingSvc)
	if err != nil {
		return nil, err
	}
	ch.SetName(name)
	ch.SetType(channels.TypeDingTalk)
	return ch, nil
}
