package dephealth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// Sentinel-ошибки планировщика.
var (
	ErrAlreadyStarted = errors.New("scheduler already started")
	ErrNotStarted     = errors.New("scheduler not started")
)

// endpointState хранит состояние проверки конкретного endpoint.
type endpointState struct {
	mu                   sync.Mutex
	healthy              *bool // nil = UNKNOWN (до первой проверки)
	consecutiveFailures  int
	consecutiveSuccesses int
}

// Scheduler управляет периодическим запуском проверок здоровья.
type Scheduler struct {
	deps    []scheduledDep
	metrics *MetricsExporter
	logger  *slog.Logger

	states  map[string]*endpointState // ключ: "name:host:port"
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	started bool
	stopped bool
	mu      sync.Mutex
}

// scheduledDep содержит зависимость с привязанным чекером.
type scheduledDep struct {
	dep     Dependency
	checker HealthChecker
}

// SchedulerOption — функциональная опция для Scheduler.
type SchedulerOption func(*schedulerConfig)

type schedulerConfig struct {
	logger *slog.Logger
}

// WithSchedulerLogger задаёт логгер для планировщика.
func WithSchedulerLogger(l *slog.Logger) SchedulerOption {
	return func(c *schedulerConfig) {
		c.logger = l
	}
}

// NewScheduler создаёт новый планировщик.
// deps — пары зависимость+чекер, metrics — экспортёр метрик.
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

// Add добавляет зависимость с чекером в планировщик.
// Должна вызываться до Start.
func (s *Scheduler) Add(dep Dependency, checker HealthChecker) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return ErrAlreadyStarted
	}

	if err := dep.Validate(); err != nil {
		return fmt.Errorf("invalid dependency %q: %w", dep.Name, err)
	}

	s.deps = append(s.deps, scheduledDep{dep: dep, checker: checker})
	return nil
}

// Start запускает периодические проверки для всех зарегистрированных зависимостей.
// Каждый endpoint каждой зависимости проверяется в отдельной горутине.
// Вызов Start более одного раза возвращает ошибку.
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
		for _, ep := range sd.dep.Endpoints {
			key := sd.dep.Name + ":" + ep.Host + ":" + ep.Port
			st := &endpointState{}
			s.states[key] = st
			s.wg.Add(1)
			go s.runEndpointLoop(ctx, sd.dep, ep, sd.checker, st)
		}
	}

	return nil
}

// Stop останавливает планировщик и ожидает завершения всех горутин.
// Повторный вызов — no-op.
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

// Health возвращает текущее состояние всех endpoint-ов.
// Ключ — "dependency:host:port", значение — true (healthy) / false (unhealthy).
// Endpoint-ы в состоянии UNKNOWN (до первой проверки) не включаются в результат.
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

// runEndpointLoop — основной цикл проверки одного endpoint.
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

	// Первая проверка.
	s.executeCheck(ctx, dep, ep, checker, state, logAttrs, true)

	// Периодические проверки.
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

// executeCheck выполняет одну проверку и обновляет состояние и метрики.
func (s *Scheduler) executeCheck(
	ctx context.Context,
	dep Dependency,
	ep Endpoint,
	checker HealthChecker,
	state *endpointState,
	logAttrs []slog.Attr,
	isFirst bool,
) {
	// Не запускаем проверку, если контекст уже отменён.
	if ctx.Err() != nil {
		return
	}

	// Создаём контекст с таймаутом для проверки.
	checkCtx, checkCancel := context.WithTimeout(ctx, dep.Config.Timeout)
	defer checkCancel()

	start := time.Now()
	checkErr := s.safeCheck(checkCtx, checker, ep)
	duration := time.Since(start)

	// Записываем латентность всегда (и при успехе, и при ошибке).
	s.metrics.ObserveLatency(dep, ep, duration)

	state.mu.Lock()
	defer state.mu.Unlock()

	if isFirst {
		// Первая проверка: сразу устанавливаем состояние без учёта порогов.
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
		// Неудачная проверка.
		state.consecutiveSuccesses = 0
		state.consecutiveFailures++

		s.logger.LogAttrs(ctx, slog.LevelWarn, "dephealth: check failed",
			append(logAttrs, slog.String("error", checkErr.Error()))...)

		if state.healthy != nil && *state.healthy &&
			state.consecutiveFailures >= dep.Config.FailureThreshold {
			// Переход HEALTHY → UNHEALTHY.
			healthy := false
			state.healthy = &healthy
			s.metrics.SetHealth(dep, ep, 0)
			s.logger.LogAttrs(ctx, slog.LevelError, "dephealth: dependency unhealthy",
				append(logAttrs, slog.Int("consecutive_failures", state.consecutiveFailures))...)
		} else if state.healthy != nil && !*state.healthy {
			// Уже unhealthy — обновляем метрику (она и так 0).
			s.metrics.SetHealth(dep, ep, 0)
		}
	} else {
		// Успешная проверка.
		state.consecutiveFailures = 0
		state.consecutiveSuccesses++

		if state.healthy != nil && !*state.healthy &&
			state.consecutiveSuccesses >= dep.Config.SuccessThreshold {
			// Переход UNHEALTHY → HEALTHY.
			healthy := true
			state.healthy = &healthy
			s.metrics.SetHealth(dep, ep, 1)
			s.logger.LogAttrs(ctx, slog.LevelInfo, "dephealth: dependency recovered", logAttrs...)
		} else if state.healthy != nil && *state.healthy {
			// Уже healthy — обновляем метрику (она и так 1).
			s.metrics.SetHealth(dep, ep, 1)
		}
	}
}

// safeCheck вызывает checker.Check с перехватом паники.
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
