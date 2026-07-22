package runtime

import (
	"os"
	"strings"
)

// Role enumerates the runtime role of a goclaw process.
//
// In single-process deployments (compose, desktop, dev) the role is RoleAll
// and every subsystem starts. In Kubernetes the API and worker concerns are
// split into separate Deployments via GOCLAW_ROLE so that long-running
// singletons (channel pollers, cron loop, consolidation workers) only run on
// the worker pod while HTTP/WS handlers scale horizontally on the api pods.
type Role string

const (
	RoleAll    Role = "all"
	RoleAPI    Role = "api"
	RoleWorker Role = "worker"
)

const envVar = "GOCLAW_ROLE"

// Current returns the role configured via GOCLAW_ROLE. Unknown values fall
// back to RoleAll so that misconfiguration never silently disables work.
func Current() Role {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(envVar))) {
	case string(RoleAPI):
		return RoleAPI
	case string(RoleWorker):
		return RoleWorker
	default:
		return RoleAll
	}
}

// IsAPI reports whether HTTP/WS listeners and request handlers should start.
func IsAPI() bool {
	r := Current()
	return r == RoleAPI || r == RoleAll
}

// IsWorker reports whether background singletons (channels, cron,
// consolidation, task ticker) should start.
func IsWorker() bool {
	r := Current()
	return r == RoleWorker || r == RoleAll
}
