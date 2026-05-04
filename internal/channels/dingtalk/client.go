package dingtalk

import (
	"context"
	"errors"
	"log/slog"
)

// streamEvent is the minimal envelope passed to the channel's event handler.
// Once the SDK is wired, replace Raw with a parsed payload struct (or keep
// Raw and parse per-topic in handler.go).
type streamEvent struct {
	Type string
	Raw  []byte
}

type streamClientDeps struct {
	clientID     string
	clientSecret string
	robotCode    string
	onEvent      func(ctx context.Context, evt streamEvent) error
}

// streamClient wraps the DingTalk Stream SDK. The current implementation is a
// stub — Start logs a clear "not implemented" warning and returns success so
// the rest of the channel pipeline (registration, health, RPC surface) is
// exercisable end-to-end without a live DingTalk app.
//
// TODO(dingtalk): replace stub with real SDK. Approximate shape:
//
//	import dts "github.com/open-dingtalk/dingtalk-stream-sdk-go/client"
//	import "github.com/open-dingtalk/dingtalk-stream-sdk-go/payload"
//
//	c.client = dts.NewStreamClient(
//	    dts.WithAppCredential(dts.NewAppCredentialConfig(deps.clientID, deps.clientSecret)),
//	    dts.WithUserAgent(dts.NewDingtalkGoSDKUserAgent()),
//	    dts.WithSubscription(payload.SubscriptionTypeKCallback, "/v1.0/im/bot/messages/get", c.onCallback),
//	)
//	return c.client.Start(ctx)
type streamClient struct {
	deps    streamClientDeps
	started bool
}

func newStreamClient(deps streamClientDeps) *streamClient {
	return &streamClient{deps: deps}
}

func (c *streamClient) Start(ctx context.Context) error {
	if c.deps.onEvent == nil {
		return errors.New("dingtalk: stream client onEvent handler missing")
	}
	c.started = true
	slog.Warn("channel.dingtalk.stream_stub_active",
		"msg", "DingTalk Stream SDK not yet wired — channel is registered but will not receive events. See internal/channels/dingtalk/client.go TODO.")
	return nil
}

func (c *streamClient) Stop(ctx context.Context) error {
	if !c.started {
		return nil
	}
	c.started = false
	// TODO(dingtalk): c.client.Close() once SDK is wired.
	return nil
}
