package httpserver

import (
	"encoding/json"
	"net/http"
	"sync/atomic"
)

type Readiness interface {
	SetDependency(name string, ok bool)
	SetShuttingDown()
	ReadyResponse() (status int, payload []byte)
}

type readiness struct {
	shuttingDown atomic.Bool
	deps         atomic.Value // stores map[string]bool
}

func NewReadiness() Readiness {
	r := &readiness{}
	r.deps.Store(map[string]bool{
		"db":  false,
		"s3":  false,
		"app": true,
	})
	return r
}

func (r *readiness) SetDependency(name string, ok bool) {
	cur := r.deps.Load().(map[string]bool)
	next := make(map[string]bool, len(cur)+1)
	for k, v := range cur {
		next[k] = v
	}
	next[name] = ok
	r.deps.Store(next)
}

func (r *readiness) SetShuttingDown() {
	r.shuttingDown.Store(true)
}

func (r *readiness) ReadyResponse() (int, []byte) {
	if r.shuttingDown.Load() {
		b, _ := json.Marshal(map[string]any{
			"ready": false,
			"deps":  r.deps.Load(),
			"state": "shutting_down",
		})
		return http.StatusServiceUnavailable, b
	}

	deps := r.deps.Load().(map[string]bool)
	ready := true
	for _, ok := range deps {
		if !ok {
			ready = false
			break
		}
	}

	code := http.StatusOK
	if !ready {
		code = http.StatusServiceUnavailable
	}

	b, _ := json.Marshal(map[string]any{
		"ready": ready,
		"deps":  deps,
	})
	return code, b
}
