package dephealth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// Sentinel errors for the scheduler.
var (
	ErrAlreadyStarted = errors.New("scheduler already started")
	ErrNotStarted     = errors.New("scheduler not started")
)

// endpointState holds the health check state for a specific endpoint.
type endpointState struct {
	mu                   sync.Mutex
	healthy              *bool // nil = UNKNOWN (before first check)
	consecutiveFailures  int
	consecutiveSuccesses int

	// Fields for HealthDetails() API.
	lastStatus    StatusCategory
	lastDetail    string
	lastLatency   time.Duration
	lastCheckedAt time.Time

	// Static fields set at state creation time.
	depName  string
	depType  DependencyType
	host     string
	port     string
	critical bool
	labels   map[string]string
}

// Scheduler manages periodic execution of health checks.
type Scheduler struct {
	deps    []scheduledDep
	metrics *MetricsExporter
	logger  *slog.Logger

	states  map[string]*endpointState // key: "name:host:port"
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	started bool
	stopped bool
	mu      sync.Mutex
}

// scheduledDep contains a dependency with its associated checker.
type scheduledDep struct {
	dep     Dependency
	checker HealthChecker
}

// SchedulerOption is a functional option for Scheduler.
type SchedulerOption func(*schedulerConfig)

type schedulerConfig struct {
	logger *slog.Logger
}

// WithSchedulerLogger sets the logger for the scheduler.
func WithSchedulerLogger(l *slog.Logger) SchedulerOption {
	return func(c *schedulerConfig) {
		c.logger = l
	}
}

// NewScheduler creates a new scheduler.
// metrics is the metrics exporter used for recording health check results.
func NewScheduler(metrics *MetricsExporter, opts ...SchedulerOption) *Scheduler {
	cfg := schedulerConfig{
		logger: slog.Default(),
	}
	for _, o := range opts {
		o(&cfg)
	}

	return &Scheduler{
		metrics: metrics,
		logger:  cfg.logger,
	}
}

// Start launches periodic health checks for all registered dependencies.
// Each endpoint of each dependency is checked in a separate goroutine.
// Calling Start more than once returns an error.
func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return ErrAlreadyStarted
	}
	s.started = true

	ctx, s.cancel = context.WithCancel(ctx)

	s.states = make(map[string]*endpointState)
	for _, sd := range s.deps {
		critical := sd.dep.Critical != nil && *sd.dep.Critical
		for _, ep := range sd.dep.Endpoints {
			key := sd.dep.Name + ":" + ep.Host + ":" + ep.Port
			labels := make(map[string]string, len(ep.Labels))
			for k, v := range ep.Labels {
				labels[k] = v
			}
			st := &endpointState{
				lastStatus: StatusUnknown,
				lastDetail: "unknown",
				depName:    sd.dep.Name,
				depType:    sd.dep.Type,
				host:       ep.Host,
				port:       ep.Port,
				critical:   critical,
				labels:     labels,
			}
			s.states[key] = st
			s.wg.Add(1)
			go s.runEndpointLoop(ctx, sd.dep, ep, sd.checker, st)
		}
	}

	return nil
}

// Stop stops the scheduler and waits for all goroutines to finish.
// Repeated calls are no-op.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	if s.stopped || !s.started {
		s.mu.Unlock()
		return
	}
	s.stopped = true
	cancel := s.cancel
	s.mu.Unlock()

	cancel()
	s.wg.Wait()
}

// Health returns the current health state of all endpoints.
// Key is "dependency:host:port", value is true (healthy) / false (unhealthy).
// Endpoints in UNKNOWN state (before first check) are not included in the result.
func (s *Scheduler) Health() map[string]bool {
	s.mu.Lock()
	states := s.states
	s.mu.Unlock()

	if states == nil {
		return nil
	}

	result := make(map[string]bool, len(states))
	for key, st := range states {
		st.mu.Lock()
		if st.healthy != nil {
			result[key] = *st.healthy
		}
		st.mu.Unlock()
	}
	return result
}

// HealthDetails returns the detailed health state of all endpoints.
// Key is "dependency:host:port". Unlike Health(), UNKNOWN endpoints
// (before first check) are included with Status="unknown" and Healthy=nil.
// Returns nil before Start() is called.
func (s *Scheduler) HealthDetails() map[string]EndpointStatus {
	s.mu.Lock()
	states := s.states
	s.mu.Unlock()

	if states == nil {
		return nil
	}

	result := make(map[string]EndpointStatus, len(states))
	for key, st := range states {
		st.mu.Lock()
		es := EndpointStatus{
			Healthy:       copyBoolPtr(st.healthy),
			Status:        st.lastStatus,
			Detail:        st.lastDetail,
			Latency:       st.lastLatency,
			Type:          st.depType,
			Name:          st.depName,
			Host:          st.host,
			Port:          st.port,
			Critical:      st.critical,
			LastCheckedAt: st.lastCheckedAt,
			Labels:        copyStringMap(st.labels),
		}
		st.mu.Unlock()
		result[key] = es
	}
	return result
}

// copyBoolPtr returns a copy of a *bool pointer.
func copyBoolPtr(b *bool) *bool {
	if b == nil {
		return nil
	}
	v := *b
	return &v
}

// copyStringMap returns a shallow copy of a string map.
func copyStringMap(m map[string]string) map[string]string {
	if m == nil {
		return map[string]string{}
	}
	cp := make(map[string]string, len(m))
	for k, v := range m {
		cp[k] = v
	}
	return cp
}

// runEndpointLoop is the main check loop for a single endpoint.
func (s *Scheduler) runEndpointLoop(ctx context.Context, dep Dependency, ep Endpoint, checker HealthChecker, state *endpointState) {
	defer s.wg.Done()

	logAttrs := []slog.Attr{
		slog.String("dependency", dep.Name),
		slog.String("type", string(dep.Type)),
		slog.String("host", ep.Host),
		slog.String("port", ep.Port),
	}

	// initialDelay.
	if dep.Config.InitialDelay > 0 {
		select {
		case <-ctx.Done():
			return
		case <-time.After(dep.Config.InitialDelay):
		}
	}

	// First check.
	s.executeCheck(ctx, dep, ep, checker, state, logAttrs, true)

	// Periodic checks.
	ticker := time.NewTicker(dep.Config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.executeCheck(ctx, dep, ep, checker, state, logAttrs, false)
		}
	}
}

// executeCheck performs a single health check and updates state and metrics.
func (s *Scheduler) executeCheck(
	ctx context.Context,
	dep Dependency,
	ep Endpoint,
	checker HealthChecker,
	state *endpointState,
	logAttrs []slog.Attr,
	isFirst bool,
) {
	// Skip check if context is already cancelled.
	if ctx.Err() != nil {
		return
	}

	// Create a context with timeout for the check.
	checkCtx, checkCancel := context.WithTimeout(ctx, dep.Config.Timeout)
	defer checkCancel()

	start := time.Now()
	checkErr := s.safeCheck(checkCtx, checker, ep)
	duration := time.Since(start)

	// Record latency always (both on success and failure).
	s.metrics.ObserveLatency(dep, ep, duration)

	// Classify the check result for status metrics.
	result := classifyError(checkErr)
	s.metrics.SetStatus(dep, ep, result.Category)
	s.metrics.SetStatusDetail(dep, ep, result.Detail)

	state.mu.Lock()
	defer state.mu.Unlock()

	// Store classification results for HealthDetails() API.
	state.lastStatus = result.Category
	state.lastDetail = result.Detail
	state.lastLatency = duration
	state.lastCheckedAt = time.Now()

	if isFirst {
		// First check: set state immediately without threshold logic.
		healthy := checkErr == nil
		state.healthy = &healthy
		if healthy {
			state.consecutiveSuccesses = 1
			state.consecutiveFailures = 0
		} else {
			state.consecutiveFailures = 1
			state.consecutiveSuccesses = 0
		}
		if healthy {
			s.metrics.SetHealth(dep, ep, 1)
		} else {
			s.metrics.SetHealth(dep, ep, 0)
			s.logger.LogAttrs(ctx, slog.LevelWarn, "dephealth: check failed",
				append(logAttrs, slog.String("error", checkErr.Error()))...)
		}
		return
	}

	if checkErr != nil {
		// Failed check.
		state.consecutiveSuccesses = 0
		state.consecutiveFailures++

		s.logger.LogAttrs(ctx, slog.LevelWarn, "dephealth: check failed",
			append(logAttrs, slog.String("error", checkErr.Error()))...)

		if state.healthy != nil && *state.healthy &&
			state.consecutiveFailures >= dep.Config.FailureThreshold {
			// Transition HEALTHY -> UNHEALTHY.
			healthy := false
			state.healthy = &healthy
			s.metrics.SetHealth(dep, ep, 0)
			s.logger.LogAttrs(ctx, slog.LevelError, "dephealth: dependency unhealthy",
				append(logAttrs, slog.Int("consecutive_failures", state.consecutiveFailures))...)
		} else if state.healthy != nil && !*state.healthy {
			// Already unhealthy — update metric (it is already 0).
			s.metrics.SetHealth(dep, ep, 0)
		}
	} else {
		// Successful check.
		state.consecutiveFailures = 0
		state.consecutiveSuccesses++

		if state.healthy != nil && !*state.healthy &&
			state.consecutiveSuccesses >= dep.Config.SuccessThreshold {
			// Transition UNHEALTHY -> HEALTHY.
			healthy := true
			state.healthy = &healthy
			s.metrics.SetHealth(dep, ep, 1)
			s.logger.LogAttrs(ctx, slog.LevelInfo, "dephealth: dependency recovered", logAttrs...)
		} else if state.healthy != nil && *state.healthy {
			// Already healthy — update metric (it is already 1).
			s.metrics.SetHealth(dep, ep, 1)
		}
	}
}

// safeCheck calls checker.Check with panic recovery.
func (s *Scheduler) safeCheck(ctx context.Context, checker HealthChecker, ep Endpoint) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic in health checker: %v", r)
			s.logger.Error("dephealth: panic in health checker",
				"endpoint", ep.Host+":"+ep.Port,
				"panic", r,
			)
		}
	}()
	return checker.Check(ctx, ep)
}
