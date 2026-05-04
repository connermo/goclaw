package dingtalk

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/nextlevelbuilder/goclaw/internal/bus"
	"github.com/nextlevelbuilder/goclaw/internal/channels"
	"github.com/nextlevelbuilder/goclaw/internal/store"
)

// Config is the runtime configuration assembled by Factory.
type Config struct {
	ClientID     string
	ClientSecret string
	RobotCode    string

	GroupPolicy string
	DMPolicy    string
	AllowFrom   []string
	BlockReply  *bool
}

// Channel is the DingTalk Stream Mode channel implementation.
type Channel struct {
	*channels.BaseChannel

	cfg    Config
	stream *streamClient

	mu     sync.Mutex
	cancel context.CancelFunc
}

// New constructs a DingTalk channel. It does not establish the WSS connection;
// that happens in Start.
func New(cfg Config, msgBus *bus.MessageBus, pairingSvc store.PairingStore) (*Channel, error) {
	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		return nil, errors.New("dingtalk: client_id and client_secret are required")
	}
	base := channels.NewBaseChannel("dingtalk", msgBus, cfg.AllowFrom)
	base.SetType(channels.TypeDingTalk)
	if pairingSvc != nil {
		base.SetPairingService(pairingSvc)
	}
	base.MarkRegistered("Configured")
	return &Channel{BaseChannel: base, cfg: cfg}, nil
}

// Start opens the Stream Mode WSS connection and begins consuming events.
//
// TODO(dingtalk): integrate github.com/open-dingtalk/dingtalk-stream-sdk-go.
// Outline:
//   1. client := stream.NewStreamClient(stream.WithAppCredential(...))
//   2. client.RegisterRouter(stream.NewSubscription(...), c.handleEvent) for
//      each topic (bot/messages/get, bot/messages/group, card/callback).
//   3. client.Start(ctx) on a goroutine; mark health Healthy on success.
//   4. On stream disconnect/error, mark Degraded and let SDK reconnect.
func (c *Channel) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.IsRunning() {
		return nil
	}

	c.MarkStarting("Connecting to DingTalk Stream")

	streamCtx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	c.stream = newStreamClient(streamClientDeps{
		clientID:     c.cfg.ClientID,
		clientSecret: c.cfg.ClientSecret,
		robotCode:    c.cfg.RobotCode,
		onEvent:      c.handleStreamEvent,
	})
	if err := c.stream.Start(streamCtx); err != nil {
		cancel()
		c.cancel = nil
		c.MarkFailed("DingTalk stream start failed", err.Error(), channels.ChannelFailureKindUnknown, true)
		return fmt.Errorf("dingtalk: start stream: %w", err)
	}

	c.SetRunning(true)
	c.MarkHealthy("Connected")
	slog.Info("channel.dingtalk.started", "client_id", redact(c.cfg.ClientID))
	return nil
}

// Stop closes the WSS connection.
func (c *Channel) Stop(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.IsRunning() {
		return nil
	}
	if c.cancel != nil {
		c.cancel()
		c.cancel = nil
	}
	if c.stream != nil {
		_ = c.stream.Stop(ctx)
		c.stream = nil
	}
	c.SetRunning(false)
	c.MarkStopped("Stopped")
	slog.Info("channel.dingtalk.stopped")
	return nil
}

// Send delivers an outbound message. Routes to the appropriate REST endpoint
// based on whether ChatID indicates a group conversation.
//
// TODO(dingtalk): implement REST send. DingTalk has separate endpoints for
// 1:1 (oToMessages/batchSend) and group (groupMessages/send) — pick by
// msg.Metadata["peer_kind"] / chat ID prefix. Use chunking.go to split
// markdown >5000 chars per the platform limit.
func (c *Channel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return errors.New("dingtalk: channel not running")
	}
	return errors.New("dingtalk: send not yet implemented (skeleton)")
}

func (c *Channel) handleStreamEvent(ctx context.Context, evt streamEvent) error {
	// TODO(dingtalk): translate evt → bus.InboundMessage:
	//   - parse evt.Type to decide single chat vs group vs card callback
	//   - extract sender_id, chat_id, content, mentions
	//   - apply group_policy (mention_only) before publish
	//   - call c.HandleMessage(senderID, chatID, content, media, metadata, peerKind)
	slog.Debug("channel.dingtalk.event_received", "type", evt.Type, "raw_size", len(evt.Raw))
	return nil
}

// redact returns a short fingerprint of a secret-ish value for logs.
func redact(s string) string {
	if len(s) <= 6 {
		return "***"
	}
	return s[:3] + "***" + s[len(s)-3:]
}
