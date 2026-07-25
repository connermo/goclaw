//go:build sqlite || sqliteonly

package sqlitestore

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/nextlevelbuilder/goclaw/internal/store"
)

// newStaleRecoveryStore returns a store over a fresh schema with one tenant.
func newStaleRecoveryStore(t *testing.T) (*SQLiteTracingStore, uuid.UUID) {
	t.Helper()
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}
	tenantID := uuid.New()
	if _, err := db.Exec(`INSERT INTO tenants (id, name, slug, status) VALUES (?, 'T', 't-stale-recovery', 'active')`, tenantID.String()); err != nil {
		t.Fatalf("insert tenant: %v", err)
	}
	return NewSQLiteTracingStore(db), tenantID
}

// insertRunningTrace creates a running trace with an explicit start_time and
// last_activity_at (pass a zero time for "never heartbeated").
func insertRunningTrace(t *testing.T, s *SQLiteTracingStore, tenantID uuid.UUID, startTime time.Time, lastActivity time.Time) uuid.UUID {
	t.Helper()
	id := uuid.New()
	// Bind time.Time exactly like CreateTrace does — the driver's own encoding
	// is what the recovery query compares against, so a hand-formatted string
	// would not be comparable.
	var activity any
	if !lastActivity.IsZero() {
		activity = lastActivity.UTC()
	}
	_, err := s.db.Exec(
		`INSERT INTO traces (id, tenant_id, name, status, start_time, last_activity_at)
		 VALUES (?, ?, 'run', 'running', ?, ?)`,
		id.String(), tenantID.String(), startTime.UTC(), activity)
	if err != nil {
		t.Fatalf("insert trace: %v", err)
	}
	return id
}

func traceStatus(t *testing.T, s *SQLiteTracingStore, id uuid.UUID) string {
	t.Helper()
	var status string
	if err := s.db.QueryRow(`SELECT status FROM traces WHERE id = ?`, id.String()).Scan(&status); err != nil {
		t.Fatalf("read status: %v", err)
	}
	return status
}

// A run that started hours ago but is still heartbeating is alive, not stale.
// This is the case the old start_time-based sweep got wrong, and the reason
// recovery stayed disabled.
func TestRecoverStaleRunningTraces_KeepsHeartbeatingLongRun(t *testing.T) {
	s, tenantID := newStaleRecoveryStore(t)
	now := time.Now().UTC()

	longRun := insertRunningTrace(t, s, tenantID, now.Add(-3*time.Hour), now.Add(-2*time.Second))

	cutoff := now.Add(-10 * time.Minute)
	recovered, err := s.RecoverStaleRunningTraces(context.Background(), cutoff)
	if err != nil {
		t.Fatalf("RecoverStaleRunningTraces: %v", err)
	}
	if recovered != 0 {
		t.Fatalf("recovered = %d, want 0 — a heartbeating run must never be swept", recovered)
	}
	if got := traceStatus(t, s, longRun); got != store.TraceStatusRunning {
		t.Fatalf("status = %q, want %q", got, store.TraceStatusRunning)
	}
}

// A trace whose owner died stops heartbeating and must be recovered.
func TestRecoverStaleRunningTraces_SweepsAbandonedTrace(t *testing.T) {
	s, tenantID := newStaleRecoveryStore(t)
	now := time.Now().UTC()

	orphan := insertRunningTrace(t, s, tenantID, now.Add(-30*time.Minute), now.Add(-20*time.Minute))

	recovered, err := s.RecoverStaleRunningTraces(context.Background(), now.Add(-10*time.Minute))
	if err != nil {
		t.Fatalf("RecoverStaleRunningTraces: %v", err)
	}
	if recovered != 1 {
		t.Fatalf("recovered = %d, want 1", recovered)
	}
	if got := traceStatus(t, s, orphan); got != store.TraceStatusError {
		t.Fatalf("status = %q, want %q", got, store.TraceStatusError)
	}
}

// Rows predating the column have no heartbeat; they fall back to start_time so
// old orphans still get cleaned up.
func TestRecoverStaleRunningTraces_NullHeartbeatFallsBackToStartTime(t *testing.T) {
	s, tenantID := newStaleRecoveryStore(t)
	now := time.Now().UTC()

	old := insertRunningTrace(t, s, tenantID, now.Add(-2*time.Hour), time.Time{})
	fresh := insertRunningTrace(t, s, tenantID, now.Add(-1*time.Minute), time.Time{})

	recovered, err := s.RecoverStaleRunningTraces(context.Background(), now.Add(-10*time.Minute))
	if err != nil {
		t.Fatalf("RecoverStaleRunningTraces: %v", err)
	}
	if recovered != 1 {
		t.Fatalf("recovered = %d, want 1", recovered)
	}
	if got := traceStatus(t, s, old); got != store.TraceStatusError {
		t.Fatalf("old trace status = %q, want %q", got, store.TraceStatusError)
	}
	if got := traceStatus(t, s, fresh); got != store.TraceStatusRunning {
		t.Fatalf("fresh trace status = %q, want %q", got, store.TraceStatusRunning)
	}
}

// TouchTracesActivity is what keeps a live run out of the sweep: a trace that
// would otherwise be stale survives once its owner heartbeats.
func TestTouchTracesActivityRescuesTraceFromSweep(t *testing.T) {
	s, tenantID := newStaleRecoveryStore(t)
	now := time.Now().UTC()

	id := insertRunningTrace(t, s, tenantID, now.Add(-2*time.Hour), now.Add(-30*time.Minute))

	if err := s.TouchTracesActivity(context.Background(), []uuid.UUID{id}, now); err != nil {
		t.Fatalf("TouchTracesActivity: %v", err)
	}

	recovered, err := s.RecoverStaleRunningTraces(context.Background(), now.Add(-10*time.Minute))
	if err != nil {
		t.Fatalf("RecoverStaleRunningTraces: %v", err)
	}
	if recovered != 0 {
		t.Fatalf("recovered = %d, want 0 after heartbeat", recovered)
	}
	if got := traceStatus(t, s, id); got != store.TraceStatusRunning {
		t.Fatalf("status = %q, want %q", got, store.TraceStatusRunning)
	}
}

// Open spans belonging to a healthy trace must survive a sweep that recovers a
// different, abandoned trace.
func TestRecoverStaleRunningTraces_LeavesSpansOfHealthyTraces(t *testing.T) {
	s, tenantID := newStaleRecoveryStore(t)
	now := time.Now().UTC()

	orphan := insertRunningTrace(t, s, tenantID, now.Add(-30*time.Minute), now.Add(-20*time.Minute))
	healthy := insertRunningTrace(t, s, tenantID, now.Add(-30*time.Minute), now.Add(-2*time.Second))

	insertRunningSpan := func(traceID uuid.UUID) uuid.UUID {
		spanID := uuid.New()
		_, err := s.db.Exec(
			`INSERT INTO spans (id, trace_id, tenant_id, name, span_type, status, start_time)
			 VALUES (?, ?, ?, 'llm', 'llm_call', 'running', ?)`,
			spanID.String(), traceID.String(), tenantID.String(),
			now.Add(-25*time.Minute).UTC())
		if err != nil {
			t.Fatalf("insert span: %v", err)
		}
		return spanID
	}
	orphanSpan := insertRunningSpan(orphan)
	healthySpan := insertRunningSpan(healthy)

	if _, err := s.RecoverStaleRunningTraces(context.Background(), now.Add(-10*time.Minute)); err != nil {
		t.Fatalf("RecoverStaleRunningTraces: %v", err)
	}

	spanStatus := func(id uuid.UUID) string {
		var status string
		if err := s.db.QueryRow(`SELECT status FROM spans WHERE id = ?`, id.String()).Scan(&status); err != nil {
			t.Fatalf("read span status: %v", err)
		}
		return status
	}
	if got := spanStatus(orphanSpan); got != store.SpanStatusError {
		t.Fatalf("orphan span status = %q, want %q", got, store.SpanStatusError)
	}
	// The old implementation swept every running span by start_time, which
	// would close this one too even though its trace is alive.
	if got := spanStatus(healthySpan); got != store.TraceStatusRunning {
		t.Fatalf("healthy span status = %q, want %q — spans must be scoped to recovered traces", got, store.TraceStatusRunning)
	}
}
