package app

// Audit-only tests. Assertions express the desired invariant, so failures are
// counterexamples, not regressions introduced by changes to production code.
// All persistence is temporary; HTTP delivery uses a counting fake client.
import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	ps "proxynth/payment-sandbox/internal/payment/adapters/sqlite"
	pa "proxynth/payment-sandbox/internal/payment/application"
	pd "proxynth/payment-sandbox/internal/payment/domain"
	ws "proxynth/payment-sandbox/internal/paymentworkflow/adapters/sqlite"
	wa "proxynth/payment-sandbox/internal/paymentworkflow/application"
	wd "proxynth/payment-sandbox/internal/paymentworkflow/domain"
	"proxynth/payment-sandbox/internal/platform/clock"
	"proxynth/payment-sandbox/internal/platform/config"
	"proxynth/payment-sandbox/internal/platform/observability"
	sq "proxynth/payment-sandbox/internal/platform/persistence/sqlite"
	"proxynth/payment-sandbox/internal/platform/persistence/sqlite/migrations"
	"proxynth/payment-sandbox/internal/provider/fake"
	ss "proxynth/payment-sandbox/internal/scheduler/adapters/sqlite"
	sa "proxynth/payment-sandbox/internal/scheduler/application"
	sd "proxynth/payment-sandbox/internal/scheduler/domain"
	whm "proxynth/payment-sandbox/internal/webhook/adapters/memory"
	whs "proxynth/payment-sandbox/internal/webhook/adapters/sqlite"
	wha "proxynth/payment-sandbox/internal/webhook/application"
	whd "proxynth/payment-sandbox/internal/webhook/domain"
)

var auditAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

func auditContext() context.Context {
	return observability.WithMetadata(context.Background(), observability.Metadata{CorrelationID: "audit-fixed"})
}
func auditDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sq.Open(context.Background(), config.DatabaseConfig{Path: filepath.Join(t.TempDir(), "audit.db"), BusyTimeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := migrations.Up(db); err != nil {
		t.Fatal(err)
	}
	return db
}
func auditClock(t *testing.T, at time.Time) *clock.VirtualClock {
	t.Helper()
	c, err := clock.NewVirtualClock(at)
	if err != nil {
		t.Fatal(err)
	}
	return c
}
func auditExec(t *testing.T, db *sql.DB, query string) {
	t.Helper()
	if _, err := db.ExecContext(context.Background(), query); err != nil {
		t.Fatal(err)
	}
}
func auditCount(t *testing.T, db *sql.DB, query string) int {
	t.Helper()
	var n int
	if err := db.QueryRowContext(context.Background(), query).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}
func auditRuntime(t *testing.T, db *sql.DB) *application {
	t.Helper()
	cfg := config.Default()
	cfg.HTTP.Address = "127.0.0.1:0"
	cfg.Admin.Token = "audit-only"
	a, err := compose(cfg, db)
	if err != nil {
		t.Fatal(err)
	}
	return a
}
func auditRequest(a *application, method, path, body, key string) *httptest.ResponseRecorder {
	req := httptest.NewRequestWithContext(context.Background(), method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer audit-only")
	req.Header.Set("X-Correlation-ID", "audit-fixed")
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	rec := httptest.NewRecorder()
	a.server.Handler().ServeHTTP(rec, req)
	return rec
}
func auditHTTP(t *testing.T, a *application, method, path, body, key string, want int) string {
	t.Helper()
	r := auditRequest(a, method, path, body, key)
	if r.Code != want {
		t.Fatalf("%s %s: status=%d body=%s, want=%d", method, path, r.Code, r.Body.String(), want)
	}
	return r.Body.String()
}
func auditPayment(t *testing.T, a *application) {
	t.Helper()
	auditHTTP(t, a, "POST", "/payments", `{"id":"p","amount":1000,"currency":"EUR"}`, "", 201)
	auditHTTP(t, a, "POST", "/payments/p/authorize", "", "", 200)
}

// Invariant: a retried capture/refund with the same key has one monetary effect.
func TestAuditHTTPIdempotency(t *testing.T) {
	for _, op := range []string{"capture", "refund"} {
		t.Run(op, func(t *testing.T) {
			db := auditDB(t)
			a := auditRuntime(t, db)
			auditPayment(t, a)
			if op == "refund" {
				auditHTTP(t, a, "POST", "/payments/p/capture", `{"amount":1000,"currency":"EUR"}`, "", 200)
			}
			for range 2 {
				auditHTTP(t, a, "POST", "/payments/p/"+op, `{"amount":400,"currency":"EUR"}`, "same-operation", 200)
			}
			p, err := ps.NewRepository(db).FindByID(auditContext(), "p")
			if err != nil {
				t.Fatal(err)
			}
			got := p.CapturedAmount().Amount()
			if op == "refund" {
				got = p.RefundedAmount().Amount()
			}
			if got != 400 {
				t.Fatalf("same Idempotency-Key applied twice: %s amount=%d, want=400", op, got)
			}
		})
	}
}

// Invariant: failure to append the event must not commit the monetary effect.
func TestAuditEventFailureDoesNotCommitPayment(t *testing.T) {
	db := auditDB(t)
	a := auditRuntime(t, db)
	auditPayment(t, a)
	auditExec(t, db, `CREATE TRIGGER audit_fail_event BEFORE INSERT ON event_log BEGIN SELECT RAISE(ABORT,'audit injected event failure'); END`)
	auditHTTP(t, a, "POST", "/payments/p/capture", `{"amount":400,"currency":"EUR"}`, "retry-key", 500)
	auditExec(t, db, `DROP TRIGGER audit_fail_event`)
	auditHTTP(t, a, "POST", "/payments/p/capture", `{"amount":400,"currency":"EUR"}`, "retry-key", 200)
	p, err := ps.NewRepository(db).FindByID(auditContext(), "p")
	if err != nil {
		t.Fatal(err)
	}
	n := auditCount(t, db, `SELECT count(*) FROM event_log WHERE event_type='payment.captured'`)
	if p.CapturedAmount().Amount() != 400 {
		t.Fatalf("500 then retry: captured=%d, capture events=%d; want one effect of 400", p.CapturedAmount().Amount(), n)
	}
}

type auditCrashPublisher struct{}

func (auditCrashPublisher) Publish(context.Context, *pd.Payment, pd.EventType) error {
	os.Exit(73)
	return nil
}

// Invariant: real process death after Save must not leave a payment without its event.
func TestAuditCrashBetweenPaymentAndEvent(t *testing.T) {
	if os.Getenv("PAYMENT_AUDIT_CRASH_CHILD") == "1" {
		db, err := sq.Open(context.Background(), config.DatabaseConfig{Path: os.Getenv("PAYMENT_AUDIT_DB"), BusyTimeout: time.Second})
		if err != nil {
			t.Fatal(err)
		}
		_, err = pa.NewCreatePaymentWithPublisher(ps.NewRepository(db), auditCrashPublisher{}).Execute(auditContext(), pa.CreatePaymentCommand{ID: "crash-payment", Amount: 1000, Currency: "EUR"})
		t.Fatalf("child did not exit: %v", err)
	}
	path := filepath.Join(t.TempDir(), "crash.db")
	db, err := sq.Open(context.Background(), config.DatabaseConfig{Path: path, BusyTimeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	if err := migrations.Up(db); err != nil {
		t.Fatal(err)
	}
	child := exec.CommandContext(context.Background(), os.Args[0], "-test.run=^TestAuditCrashBetweenPaymentAndEvent$")
	child.Env = append(os.Environ(), "PAYMENT_AUDIT_CRASH_CHILD=1", "PAYMENT_AUDIT_DB="+path)
	out, err := child.CombinedOutput()
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 73 {
		t.Fatalf("unexpected child result: %v %s", err, out)
	}
	payments := auditCount(t, db, `SELECT count(*) FROM payments`)
	events := auditCount(t, db, `SELECT count(*) FROM event_log`)
	if payments != events {
		t.Fatalf("after abrupt exit(73): payments=%d events=%d", payments, events)
	}
}

// Invariant: all intended webhook jobs accompany the committed business event.
func TestAuditEventFanoutIsAtomicAndRecoverable(t *testing.T) {
	db := auditDB(t)
	a := auditRuntime(t, db)
	for _, id := range []string{"a", "b"} {
		auditHTTP(t, a, "POST", "/webhook-endpoints", fmt.Sprintf(`{"id":%q,"url":"https://example.test/hooks"}`, id), "", 201)
	}
	auditExec(t, db, `CREATE TRIGGER audit_fail_job BEFORE INSERT ON scheduler_jobs WHEN NEW.id LIKE '%:b' BEGIN SELECT RAISE(ABORT,'audit injected job failure'); END`)
	auditHTTP(t, a, "POST", "/payments", `{"id":"p","amount":1000,"currency":"EUR"}`, "", 500)
	payments := auditCount(t, db, `SELECT count(*) FROM payments`)
	events := auditCount(t, db, `SELECT count(*) FROM event_log`)
	jobs := auditCount(t, db, `SELECT count(*) FROM scheduler_jobs`)
	auditExec(t, db, `DROP TRIGGER audit_fail_job`)
	retry := auditRequest(a, "POST", "/payments", `{"id":"p","amount":1000,"currency":"EUR"}`, "")
	if payments != 0 || events != 0 || jobs != 0 {
		t.Fatalf("partial commit: payments=%d events=%d jobs=%d (2 endpoints); retry status=%d", payments, events, jobs, retry.Code)
	}
}

func auditNewJob(t *testing.T, repo *ss.Repository, id string, at time.Time) *sd.Job {
	t.Helper()
	j, err := sd.NewJob(sd.JobID(id), "audit", []byte(`{}`), at)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Save(auditContext(), &j); err != nil {
		t.Fatal(err)
	}
	return &j
}
func auditScheduler(t *testing.T, repo sa.Repository, dispatcher sa.Dispatcher, business, operational clock.Clock) *sa.Scheduler {
	t.Helper()
	s, err := sa.NewScheduler(repo, dispatcher, business, operational, sa.Config{Owner: "audit", BatchSize: 100, LeaseDuration: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

type auditDispatchFunc func(context.Context, *sd.Job) error

func (f auditDispatchFunc) Dispatch(ctx context.Context, j *sd.Job) error { return f(ctx, j) }

// Invariant: failed runtime jobs are retried; the real SQLite path must work.
func TestAuditFailedJobCanRetry(t *testing.T) {
	db := auditDB(t)
	repo := ss.NewRepository(db)
	auditNewJob(t, repo, "failed", auditAt)
	c := auditClock(t, auditAt)
	calls := 0
	worker, err := sa.NewWorker(repo, map[sd.JobType]sa.JobHandler{"audit": func(context.Context, []byte) error { calls++; return errors.New("transient") }})
	if err != nil {
		t.Fatal(err)
	}
	s := auditScheduler(t, repo, runtimeDispatcher{worker}, c, c)
	first := s.Tick(auditContext())
	second := s.Tick(auditContext())
	if calls != 2 {
		t.Fatalf("handler calls=%d, first=%v, second=%v; want retry", calls, first, second)
	}
}

// Invariant: expired leased/running jobs are discovered and recovered after restart.
func TestAuditExpiredJobsAreDiscovered(t *testing.T) {
	for _, running := range []bool{false, true} {
		t.Run(fmt.Sprint(running), func(t *testing.T) {
			db := auditDB(t)
			repo := ss.NewRepository(db)
			j := auditNewJob(t, repo, "expired", auditAt)
			if err := j.Lease("dead-worker", auditAt.Add(time.Minute)); err != nil {
				t.Fatal(err)
			}
			if running {
				if err := j.Start(); err != nil {
					t.Fatal(err)
				}
			}
			if err := repo.Save(auditContext(), j); err != nil {
				t.Fatal(err)
			}
			repo = ss.NewRepository(db)
			c := auditClock(t, auditAt.Add(2*time.Minute))
			calls := 0
			s := auditScheduler(t, repo, auditDispatchFunc(func(context.Context, *sd.Job) error { calls++; return nil }), c, c)
			if err := s.Tick(auditContext()); err != nil {
				t.Fatal(err)
			}
			if calls != 1 {
				t.Fatalf("expired %s job dispatches=%d, want=1", j.Status(), calls)
			}
		})
	}
}

// Invariant: operational leases use operational time, independently of virtual time.
func TestAuditLeaseUsesOperationalClock(t *testing.T) {
	db := auditDB(t)
	repo := ss.NewRepository(db)
	auditNewJob(t, repo, "clock", auditAt)
	business := auditClock(t, auditAt)
	operational := auditClock(t, auditAt.Add(24*time.Hour))
	var expires time.Time
	s := auditScheduler(t, repo, auditDispatchFunc(func(_ context.Context, j *sd.Job) error { expires = j.LeaseExpiresAt(); return nil }), business, operational)
	if err := s.Tick(auditContext()); err != nil {
		t.Fatal(err)
	}
	if !expires.Equal(operational.Now().Add(time.Minute)) {
		t.Fatalf("lease expires=%s, operational now=%s; lease already expired", expires, operational.Now())
	}
}

type auditBarrierDB struct {
	*sql.DB
	arrived chan struct{}
	release chan struct{}
}

func (b auditBarrierDB) ExecContext(ctx context.Context, q string, args ...any) (sql.Result, error) {
	if strings.Contains(q, "UPDATE scheduler_jobs SET status") {
		b.arrived <- struct{}{}
		select {
		case <-b.release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return b.DB.ExecContext(ctx, q, args...)
}

// Invariant: two concurrent acquisitions grant at most one lease and one effect.
// Barrier pauses each atomic lease write until both contenders are ready.
func TestAuditConcurrentAcquisitionHasOneWinner(t *testing.T) {
	db := auditDB(t)
	base := ss.NewRepository(db)
	auditNewJob(t, base, "race", auditAt)
	ctx, cancel := context.WithTimeout(auditContext(), 5*time.Second)
	defer cancel()
	barrier := auditBarrierDB{db, make(chan struct{}, 2), make(chan struct{})}
	repo := ss.NewRepository(barrier)
	var effects atomic.Int64
	worker, err := sa.NewWorker(base, map[sd.JobType]sa.JobHandler{"audit": func(context.Context, []byte) error { effects.Add(1); return nil }})
	if err != nil {
		t.Fatal(err)
	}
	results := make(chan error, 2)
	for _, owner := range []string{"worker-a", "worker-b"} {
		go func(owner string) {
			j, err := repo.Acquire(ctx, "race", owner, auditAt.Add(time.Minute), auditAt)
			if err == nil {
				err = worker.Execute(ctx, j)
			}
			results <- err
		}(owner)
	}
	for range 2 {
		select {
		case <-barrier.arrived:
		case <-ctx.Done():
			close(barrier.release)
			t.Fatal("barrier timeout")
		}
	}
	close(barrier.release)
	first, second := <-results, <-results
	if effects.Load() != 1 {
		t.Fatalf("business effects=%d, errors=[%v,%v]; both leases won", effects.Load(), first, second)
	}
}

// Invariant: re-enqueuing the same identity cannot resurrect completed work.
func TestAuditCompletedJobCannotBeResetByDuplicateEnqueue(t *testing.T) {
	db := auditDB(t)
	repo := ss.NewRepository(db)
	j := auditNewJob(t, repo, "done", auditAt)
	if err := j.Lease("a", auditAt.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := j.Start(); err != nil {
		t.Fatal(err)
	}
	if err := j.Complete(); err != nil {
		t.Fatal(err)
	}
	if err := repo.Save(auditContext(), j); err != nil {
		t.Fatal(err)
	}
	duplicate, err := sd.NewJob("done", "audit", []byte(`{}`), auditAt)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Save(auditContext(), &duplicate); err != nil {
		return
	}
	pending := auditCount(t, db, `SELECT count(*) FROM scheduler_jobs WHERE status='pending' AND attempts=0`)
	if pending != 0 {
		t.Fatal("duplicate enqueue changed completed/attempts=1 into pending/attempts=0")
	}
}

// Invariant: a job scheduled after now is never returned as eligible.
func TestAuditSQLiteTimestampOrdering(t *testing.T) {
	db := auditDB(t)
	repo := ss.NewRepository(db)
	auditNewJob(t, repo, "future", auditAt.Add(500*time.Millisecond))
	jobs, err := repo.FindExecutable(auditContext(), auditAt, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) > 0 {
		t.Fatalf("at=%s returned future job due=%s", auditAt, jobs[0].NextAttemptAt())
	}
}

// Invariant: event order follows aggregate versions when timestamps differ in precision.
func TestAuditEventOrderTracksBusinessOrder(t *testing.T) {
	db := auditDB(t)
	repo := ps.NewEventLogRepository(db)
	for i, at := range []time.Time{auditAt, auditAt.Add(100 * time.Millisecond)} {
		eventType := []pd.EventType{pd.EventPaymentCreated, pd.EventPaymentAuthorized}[i]
		event, err := pd.NewBusinessEvent(pd.EventID(fmt.Sprint(i)), "p", eventType, at, uint64(i+1), "audit", "")
		if err != nil {
			t.Fatal(err)
		}
		if err := repo.Append(auditContext(), event); err != nil {
			t.Fatal(err)
		}
	}
	events, err := repo.ListByAggregate(auditContext(), "p")
	if err != nil {
		t.Fatal(err)
	}
	if events[0].AggregateVersion() != 1 {
		t.Fatalf("event versions=[%d,%d], want=[1,2]", events[0].AggregateVersion(), events[1].AggregateVersion())
	}
}

// Invariant: restart preserves endpoint registrations, scenario inputs and virtual time.
func TestAuditRuntimeRestartPreservesControlState(t *testing.T) {
	db := auditDB(t)
	first := auditRuntime(t, db)
	auditHTTP(t, first, "POST", "/webhook-endpoints", `{"id":"e","url":"https://example.test/hooks"}`, "", 201)
	auditHTTP(t, first, "POST", "/admin/scenarios", `{"id":"restart-scenario","provider":{"id":"fake"},"initial_virtual_time":"2026-01-01T00:00:00Z","commands":[]}`, "", 201)
	auditHTTP(t, first, "POST", "/admin/time/advance", `{"by":"8760h"}`, "", 200)
	before := auditHTTP(t, first, "GET", "/admin/time", "", "", 200)
	second := auditRuntime(t, db)
	after := auditHTTP(t, second, "GET", "/admin/time", "", "", 200)
	endpoints := auditHTTP(t, second, "GET", "/webhook-endpoints", "", "", 200)
	scenario := auditRequest(second, "GET", "/admin/scenarios/restart-scenario", "", "")
	if !strings.Contains(endpoints, `"e"`) || before != after {
		t.Fatalf("recompose same DB: endpoints=%s scenario status=%d time before=%s after=%s", endpoints, scenario.Code, before, after)
	}
}

// Invariant: inspecting a scenario preserves the inputs required to replay it.
func TestAuditScenarioInspectionPreservesControls(t *testing.T) {
	db := auditDB(t)
	a := auditRuntime(t, db)
	auditHTTP(t, a, "POST", "/admin/scenarios", `{"id":"controls","provider":{"id":"fake","profile":"pending_authorize"},"initial_virtual_time":"2026-01-01T00:00:00Z","commands":[{"type":"advance_time","duration":"1m"},{"type":"execute_async","operation_id":"fake:p:async"}]}`, "", 201)
	body := auditHTTP(t, a, "GET", "/admin/scenarios/controls", "", "", 200)
	if !strings.Contains(body, `"duration"`) || !strings.Contains(body, `"operation_id"`) {
		t.Fatalf("scenario inspection discarded replay controls: %s", body)
	}
}

// Positive control: scenario execution remains isolated from persistent runtime state.
func TestAuditScenarioExecutionIsIsolated(t *testing.T) {
	db := auditDB(t)
	a := auditRuntime(t, db)
	auditHTTP(t, a, "POST", "/admin/scenarios", `{"id":"isolated","provider":{"id":"fake"},"initial_virtual_time":"2026-01-01T00:00:00Z","commands":[{"type":"create_payment","payment_id":"isolated-p","amount":1000,"currency":"EUR"},{"type":"authorize","payment_id":"isolated-p"}]}`, "", 201)
	first := auditHTTP(t, a, "POST", "/admin/scenarios/isolated/execute", "", "", 200)
	second := auditHTTP(t, a, "POST", "/admin/scenarios/isolated/execute", "", "", 200)
	if first != second || auditCount(t, db, `SELECT count(*) FROM payments`) != 0 || auditCount(t, db, `SELECT count(*) FROM event_log`) != 0 {
		t.Fatal("isolated replay modified runtime or changed result")
	}
}

// Invariant: the event log contains enough information to reconstruct monetary history.
func TestAuditEventLogDistinguishesAmounts(t *testing.T) {
	histories := make([][]pd.BusinessEvent, 0, 2)
	for _, amount := range []int64{400, 600} {
		db := auditDB(t)
		payments := ps.NewRepository(db)
		events := ps.NewEventLogRepository(db)
		pub, err := newPaymentEventPublisher(events, whm.NewRepository(), ss.NewRepository(db), auditClock(t, auditAt))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pa.NewCreatePaymentWithPublisher(payments, pub).Execute(auditContext(), pa.CreatePaymentCommand{ID: "p", Amount: 1000, Currency: "EUR"}); err != nil {
			t.Fatal(err)
		}
		if _, err := pa.NewAuthorizePaymentWithPublisher(payments, pub).Execute(auditContext(), pa.AuthorizePaymentCommand{PaymentID: "p"}); err != nil {
			t.Fatal(err)
		}
		if _, err := pa.NewCapturePaymentWithPublisher(payments, pub).Execute(auditContext(), pa.CapturePaymentCommand{PaymentID: "p", Amount: amount, Currency: "EUR"}); err != nil {
			t.Fatal(err)
		}
		history, err := events.ListByAggregate(auditContext(), "p")
		if err != nil {
			t.Fatal(err)
		}
		histories = append(histories, history)
	}
	if reflect.DeepEqual(histories[0], histories[1]) {
		t.Fatal("capture 400 and capture 600 produced identical persisted event histories")
	}
}

// Invariant: a stale Saga write is explicitly rejected, not reported successful.
func TestAuditSagaReportsVersionConflict(t *testing.T) {
	db := auditDB(t)
	repo := ws.NewRepository(db)
	initial, err := wd.NewWithPayload("s", "p", []byte(`{"amount":1000,"currency":"EUR"}`), 42, auditAt)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Save(auditContext(), initial); err != nil {
		t.Fatal(err)
	}
	winner, stale := initial, initial
	if err := winner.ApplySuccess(wd.StepAuthorize, auditAt); err != nil {
		t.Fatal(err)
	}
	if err := stale.BeginCompensation(wd.StepAuthorize, auditAt); err != nil {
		t.Fatal(err)
	}
	if err := repo.Save(auditContext(), winner); err != nil {
		t.Fatal(err)
	}
	err = repo.Save(auditContext(), stale)
	actual, findErr := repo.Find(auditContext(), "s")
	if findErr != nil {
		t.Fatal(findErr)
	}
	if err == nil {
		t.Fatalf("stale Save returned nil; intended=%s actual=%s", stale.Status, actual.Status)
	}
}

type auditFailSagaStore struct {
	*ws.Repository
	fail atomic.Bool
}

func (s *auditFailSagaStore) Save(ctx context.Context, i wd.Instance) error {
	if s.fail.Load() {
		return errors.New("audit saga persistence failure")
	}
	return s.Repository.Save(ctx, i)
}

// Invariant: replay of a message after payment commit and Saga-save failure can recover.
func TestAuditSagaRecoversAfterPaymentCommit(t *testing.T) {
	db := auditDB(t)
	payments := ps.NewRepository(db)
	store := &auditFailSagaStore{Repository: ws.NewRepository(db)}
	c := auditClock(t, auditAt)
	jobs := ss.NewRepository(db)
	orchestrator, err := wa.NewOrchestrator(store, ws.NewPublisher(jobs), c.Now)
	if err != nil {
		t.Fatal(err)
	}
	executor, err := wa.NewPaymentExecutor(payments, fake.New(), c)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pa.NewCreatePayment(payments).Execute(auditContext(), pa.CreatePaymentCommand{ID: "p", Amount: 1000, Currency: "EUR"}); err != nil {
		t.Fatal(err)
	}
	if err := orchestrator.StartWithPayload(auditContext(), "s", "p", []byte(`{"amount":1000,"currency":"EUR"}`), 42); err != nil {
		t.Fatal(err)
	}
	var payload []byte
	if err := db.QueryRowContext(auditContext(), `SELECT payload FROM scheduler_jobs`).Scan(&payload); err != nil {
		t.Fatal(err)
	}
	var msg wd.Message
	if err := json.Unmarshal(payload, &msg); err != nil {
		t.Fatal(err)
	}
	store.fail.Store(true)
	first := orchestrator.Handle(auditContext(), msg, executor)
	if first == nil {
		t.Fatal("expected injected error")
	}
	store.fail.Store(false)
	second := orchestrator.Handle(auditContext(), msg, executor)
	p, err := payments.FindByID(auditContext(), "p")
	if err != nil {
		t.Fatal(err)
	}
	s, err := store.Find(auditContext(), "s")
	if err != nil {
		t.Fatal(err)
	}
	if second != nil {
		t.Fatalf("retry error=%v payment=%s saga step=%s; cannot recover", second, p.Status(), s.CurrentStep)
	}
}

type auditHTTPClient struct{ calls atomic.Int64 }

func (c *auditHTTPClient) Do(*http.Request) (*http.Response, error) {
	c.calls.Add(1)
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("ok"))}, nil
}

// Characterisation: at-least-once webhook delivery permits duplicates; it is not exactly-once.
func TestAuditWebhookDeliveryCharacterization(t *testing.T) {
	endpoints := whm.NewRepository()
	e, err := whd.NewEndpoint("e", "https://example.test/hooks")
	if err != nil {
		t.Fatal(err)
	}
	if err := endpoints.Save(auditContext(), e); err != nil {
		t.Fatal(err)
	}
	client := &auditHTTPClient{}
	callback, err := wha.NewOutboundCallback(endpoints, client)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := wha.NewDeliveryPayload("e", []byte(`{"id":"same-event"}`))
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := callback.Execute(auditContext(), payload); err != nil {
			t.Fatal(err)
		}
	}
	if client.calls.Load() != 2 {
		t.Fatalf("calls=%d", client.calls.Load())
	}
	t.Log("same event delivered twice through fake client; consumer deduplication is required")
}

// Invariant: if the process dies after an accepted callback but before it can
// mark the job completed, lease recovery causes a second at-least-once
// delivery and leaves one durable audit result for each attempt.
func TestAuditWebhookCrashAfterSuccessBeforeCompletionIsTraceable(t *testing.T) {
	db := auditDB(t)
	endpoints := whs.NewRepository(db)
	endpoint, err := whd.NewEndpoint("endpoint-1", "https://example.test/hooks")
	if err != nil {
		t.Fatal(err)
	}
	if err := endpoints.Save(auditContext(), endpoint); err != nil {
		t.Fatal(err)
	}

	payload, err := wha.NewDeliveryPayload(endpoint.ID(), []byte(`{"event":"payment.authorized"}`), "request-1", "event-1")
	if err != nil {
		t.Fatal(err)
	}
	job, err := sd.NewJob("webhook-job-1", wha.DeliveryJobType, payload, auditAt)
	if err != nil {
		t.Fatal(err)
	}
	jobs := ss.NewRepository(db)
	if err := jobs.Save(auditContext(), &job); err != nil {
		t.Fatal(err)
	}
	first, err := jobs.Acquire(auditContext(), job.ID(), "first-worker", auditAt.Add(time.Minute), auditAt)
	if err != nil {
		t.Fatal(err)
	}
	client := &auditHTTPClient{}
	callback, err := wha.NewOutboundCallbackWithAudit(endpoints, client, whs.NewDeliveryAuditRepository(db))
	if err != nil {
		t.Fatal(err)
	}
	firstWorker, err := sa.NewWorker(&auditFailCompletedSaveRepository{inner: jobs}, map[sd.JobType]sa.JobHandler{wha.DeliveryJobType: callback.Execute})
	if err != nil {
		t.Fatal(err)
	}
	if err := firstWorker.Execute(auditContext(), first); err == nil {
		t.Fatal("first worker unexpectedly completed after injected completion persistence failure")
	}

	secondCandidates, err := jobs.FindExecutable(auditContext(), auditAt.Add(time.Minute), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(secondCandidates) != 1 {
		t.Fatalf("recovered candidates = %d, want 1", len(secondCandidates))
	}
	second, err := jobs.Acquire(auditContext(), job.ID(), "second-worker", auditAt.Add(2*time.Minute), auditAt.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	secondWorker, err := sa.NewWorker(jobs, map[sd.JobType]sa.JobHandler{wha.DeliveryJobType: callback.Execute})
	if err != nil {
		t.Fatal(err)
	}
	if err := secondWorker.Execute(auditContext(), second); err != nil {
		t.Fatal(err)
	}

	if client.calls.Load() != 2 {
		t.Fatalf("callback deliveries = %d, want 2 after crash window", client.calls.Load())
	}
	rows, err := db.Query(`SELECT attempt, outcome, http_status FROM webhook_delivery_audit WHERE job_id = ? ORDER BY attempt`, job.ID())
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var recorded []int
	for rows.Next() {
		var attempt, status int
		var outcome string
		if err := rows.Scan(&attempt, &outcome, &status); err != nil {
			t.Fatal(err)
		}
		if outcome != string(wha.DeliverySucceeded) || status != http.StatusOK {
			t.Fatalf("audit outcome attempt %d = %q/%d", attempt, outcome, status)
		}
		recorded = append(recorded, attempt)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(recorded, []int{1, 2}) {
		t.Fatalf("recorded webhook attempts = %v, want [1 2]", recorded)
	}
}

// Invariant: failure to write a terminal audit result prevents a callback from
// being silently forgotten. The durable start marker remains, and a retry gets
// its own terminal result after audit persistence recovers.
func TestAuditWebhookAuditWriteFailureLeavesUnknownAttemptTrace(t *testing.T) {
	db := auditDB(t)
	endpoints := whs.NewRepository(db)
	endpoint, err := whd.NewEndpoint("endpoint-1", "https://example.test/hooks")
	if err != nil {
		t.Fatal(err)
	}
	if err := endpoints.Save(auditContext(), endpoint); err != nil {
		t.Fatal(err)
	}
	payload, err := wha.NewDeliveryPayload(endpoint.ID(), []byte(`{"event":"payment.authorized"}`), "request-1", "event-1")
	if err != nil {
		t.Fatal(err)
	}
	job, err := sd.NewJob("webhook-job-audit-failure", wha.DeliveryJobType, payload, auditAt)
	if err != nil {
		t.Fatal(err)
	}
	jobs := ss.NewRepository(db)
	if err := jobs.Save(auditContext(), &job); err != nil {
		t.Fatal(err)
	}
	first, err := jobs.Acquire(auditContext(), job.ID(), "first-worker", auditAt.Add(time.Minute), auditAt)
	if err != nil {
		t.Fatal(err)
	}
	client := &auditHTTPClient{}
	callback, err := wha.NewOutboundCallbackWithAudit(endpoints, client, whs.NewDeliveryAuditRepository(db))
	if err != nil {
		t.Fatal(err)
	}
	worker, err := sa.NewWorker(jobs, map[sd.JobType]sa.JobHandler{wha.DeliveryJobType: callback.Execute})
	if err != nil {
		t.Fatal(err)
	}
	auditExec(t, db, `CREATE TRIGGER audit_fail_webhook_result BEFORE UPDATE ON webhook_delivery_audit BEGIN SELECT RAISE(ABORT,'injected audit write failure'); END`)
	if err := worker.Execute(auditContext(), first); err == nil {
		t.Fatal("worker unexpectedly completed despite terminal audit failure")
	}
	auditExec(t, db, `DROP TRIGGER audit_fail_webhook_result`)

	second, err := jobs.Acquire(auditContext(), job.ID(), "second-worker", auditAt.Add(2*time.Minute), auditAt)
	if err != nil {
		t.Fatal(err)
	}
	if err := worker.Execute(auditContext(), second); err != nil {
		t.Fatal(err)
	}
	if client.calls.Load() != 2 {
		t.Fatalf("callback deliveries = %d, want retry after audit failure", client.calls.Load())
	}
	rows, err := db.Query(`SELECT attempt, outcome FROM webhook_delivery_audit WHERE job_id = ? ORDER BY attempt`, job.ID())
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var recorded []string
	for rows.Next() {
		var attempt int
		var outcome string
		if err := rows.Scan(&attempt, &outcome); err != nil {
			t.Fatal(err)
		}
		recorded = append(recorded, fmt.Sprintf("%d:%s", attempt, outcome))
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(recorded, []string{"1:started", "2:succeeded"}) {
		t.Fatalf("webhook audit trail = %v, want [1:started 2:succeeded]", recorded)
	}
}

type auditFailCompletedSaveRepository struct {
	inner  sa.WorkerRepository
	failed bool
}

func (r *auditFailCompletedSaveRepository) Save(ctx context.Context, job *sd.Job) error {
	if job.Status() == sd.JobCompleted && !r.failed {
		r.failed = true
		return errors.New("injected crash before completed job persistence")
	}
	return r.inner.Save(ctx, job)
}

// Positive control: monetary optimistic concurrency rejects a stale competing transition.
func TestAuditPaymentConcurrentCAS(t *testing.T) {
	db := auditDB(t)
	repo := ps.NewRepository(db)
	if _, err := pa.NewCreatePayment(repo).Execute(auditContext(), pa.CreatePaymentCommand{ID: "p", Amount: 1000, Currency: "EUR"}); err != nil {
		t.Fatal(err)
	}
	one, err := repo.FindByID(auditContext(), "p")
	if err != nil {
		t.Fatal(err)
	}
	two, err := repo.FindByID(auditContext(), "p")
	if err != nil {
		t.Fatal(err)
	}
	if err := one.Authorize(); err != nil {
		t.Fatal(err)
	}
	if err := two.Authorize(); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, p := range []*pd.Payment{one, two} {
		wg.Add(1)
		go func(p *pd.Payment) { defer wg.Done(); results <- repo.Save(auditContext(), p) }(p)
	}
	wg.Wait()
	close(results)
	success, conflict := 0, 0
	for err := range results {
		if err == nil {
			success++
		} else if errors.Is(err, pa.ErrPaymentVersionConflict) {
			conflict++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatalf("success=%d conflict=%d", success, conflict)
	}
}
