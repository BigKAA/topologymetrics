package dephealth

import (
	"testing"
	"time"
)

func TestValidateName(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{"postgres-main", false},
		{"redis-cache", false},
		{"a", false},
		{"kafka-0", false},
		{"my-service-123", false},

		// Невалидные
		{"", true},                       // пустое
		{"A", true},                      // заглавные
		{"0abc", true},                   // начинается с цифры
		{"-abc", true},                   // начинается с дефиса
		{"abc_def", true},                // подчёркивание
		{"abc def", true},                // пробел
		{"abc.def", true},                // точка
		{string(make([]byte, 64)), true}, // слишком длинное
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateName(tt.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateName(%q) error = %v, wantErr %v", tt.name, err, tt.wantErr)
			}
		})
	}
}

func TestValidateLabelName(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{"role", false},
		{"shard", false},
		{"vhost", false},
		{"env", false},
		{"_private", false},
		{"my_label_123", false},

		// Зарезервированные
		{"name", true},
		{"dependency", true},
		{"type", true},
		{"host", true},
		{"port", true},
		{"critical", true},

		// Невалидные
		{"0invalid", true}, // начинается с цифры
		{"invalid-", true}, // содержит дефис
		{"my label", true}, // содержит пробел
		{"my.label", true}, // содержит точку
		{"", true},         // пустое
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLabelName(tt.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateLabelName(%q) error = %v, wantErr %v", tt.name, err, tt.wantErr)
			}
		})
	}
}

func TestValidateLabels(t *testing.T) {
	// Валидные.
	if err := ValidateLabels(map[string]string{"role": "primary", "shard": "01"}); err != nil {
		t.Errorf("ValidateLabels(valid) = %v", err)
	}

	// Пустые — валидно.
	if err := ValidateLabels(nil); err != nil {
		t.Errorf("ValidateLabels(nil) = %v", err)
	}

	// Зарезервированная метка.
	if err := ValidateLabels(map[string]string{"dependency": "bad"}); err == nil {
		t.Error("ожидали ошибку для зарезервированной метки")
	}

	// Невалидное имя.
	if err := ValidateLabels(map[string]string{"0bad": "val"}); err == nil {
		t.Error("ожидали ошибку для невалидного имени метки")
	}
}

func TestBoolToYesNo(t *testing.T) {
	if got := BoolToYesNo(true); got != "yes" {
		t.Errorf("BoolToYesNo(true) = %q, want yes", got)
	}
	if got := BoolToYesNo(false); got != "no" {
		t.Errorf("BoolToYesNo(false) = %q, want no", got)
	}
}

func TestCheckConfigValidate(t *testing.T) {
	valid := DefaultCheckConfig()
	if err := valid.Validate(); err != nil {
		t.Fatalf("DefaultCheckConfig().Validate() = %v", err)
	}

	tests := []struct {
		name    string
		config  CheckConfig
		wantErr bool
	}{
		{
			name:    "default is valid",
			config:  DefaultCheckConfig(),
			wantErr: false,
		},
		{
			name: "timeout >= interval",
			config: CheckConfig{
				Interval:         5 * time.Second,
				Timeout:          5 * time.Second,
				InitialDelay:     0,
				FailureThreshold: 1,
				SuccessThreshold: 1,
			},
			wantErr: true,
		},
		{
			name: "interval too small",
			config: CheckConfig{
				Interval:         500 * time.Millisecond,
				Timeout:          100 * time.Millisecond,
				InitialDelay:     0,
				FailureThreshold: 1,
				SuccessThreshold: 1,
			},
			wantErr: true,
		},
		{
			name: "failure threshold too high",
			config: CheckConfig{
				Interval:         15 * time.Second,
				Timeout:          5 * time.Second,
				InitialDelay:     0,
				FailureThreshold: 11,
				SuccessThreshold: 1,
			},
			wantErr: true,
		},
		{
			name: "custom valid config",
			config: CheckConfig{
				Interval:         30 * time.Second,
				Timeout:          10 * time.Second,
				InitialDelay:     0,
				FailureThreshold: 3,
				SuccessThreshold: 2,
			},
			wantErr: false,
		},
		{
			name: "timeout too small",
			config: CheckConfig{
				Interval:         15 * time.Second,
				Timeout:          10 * time.Millisecond,
				InitialDelay:     0,
				FailureThreshold: 1,
				SuccessThreshold: 1,
			},
			wantErr: true,
		},
		{
			name: "initial delay too high",
			config: CheckConfig{
				Interval:         15 * time.Second,
				Timeout:          5 * time.Second,
				InitialDelay:     6 * time.Minute,
				FailureThreshold: 1,
				SuccessThreshold: 1,
			},
			wantErr: true,
		},
		{
			name: "success threshold too high",
			config: CheckConfig{
				Interval:         15 * time.Second,
				Timeout:          5 * time.Second,
				InitialDelay:     0,
				FailureThreshold: 1,
				SuccessThreshold: 11,
			},
			wantErr: true,
		},
		{
			name: "interval too high",
			config: CheckConfig{
				Interval:         11 * time.Minute,
				Timeout:          5 * time.Second,
				InitialDelay:     0,
				FailureThreshold: 1,
				SuccessThreshold: 1,
			},
			wantErr: true,
		},
		{
			name: "timeout too high",
			config: CheckConfig{
				Interval:         15 * time.Second,
				Timeout:          31 * time.Second,
				InitialDelay:     0,
				FailureThreshold: 1,
				SuccessThreshold: 1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDependencyValidate(t *testing.T) {
	crit := true
	validDep := Dependency{
		Name:     "postgres-main",
		Type:     TypePostgres,
		Critical: &crit,
		Endpoints: []Endpoint{
			{Host: "pg.svc", Port: "5432"},
		},
		Config: DefaultCheckConfig(),
	}

	if err := validDep.Validate(); err != nil {
		t.Fatalf("valid dependency: Validate() = %v", err)
	}

	critFalse := false
	tests := []struct {
		name    string
		dep     Dependency
		wantErr bool
	}{
		{
			name:    "valid",
			dep:     validDep,
			wantErr: false,
		},
		{
			name: "invalid name",
			dep: Dependency{
				Name:      "INVALID",
				Type:      TypeRedis,
				Critical:  &critFalse,
				Endpoints: []Endpoint{{Host: "redis", Port: "6379"}},
				Config:    DefaultCheckConfig(),
			},
			wantErr: true,
		},
		{
			name: "unknown type",
			dep: Dependency{
				Name:      "test",
				Type:      "unknown",
				Critical:  &critFalse,
				Endpoints: []Endpoint{{Host: "host", Port: "80"}},
				Config:    DefaultCheckConfig(),
			},
			wantErr: true,
		},
		{
			name: "missing critical",
			dep: Dependency{
				Name:      "test",
				Type:      TypeRedis,
				Critical:  nil,
				Endpoints: []Endpoint{{Host: "redis", Port: "6379"}},
				Config:    DefaultCheckConfig(),
			},
			wantErr: true,
		},
		{
			name: "no endpoints",
			dep: Dependency{
				Name:      "test",
				Type:      TypeRedis,
				Critical:  &critFalse,
				Endpoints: nil,
				Config:    DefaultCheckConfig(),
			},
			wantErr: true,
		},
		{
			name: "empty host",
			dep: Dependency{
				Name:      "test",
				Type:      TypeRedis,
				Critical:  &critFalse,
				Endpoints: []Endpoint{{Host: "", Port: "6379"}},
				Config:    DefaultCheckConfig(),
			},
			wantErr: true,
		},
		{
			name: "empty port",
			dep: Dependency{
				Name:      "test",
				Type:      TypeRedis,
				Critical:  &critFalse,
				Endpoints: []Endpoint{{Host: "redis", Port: ""}},
				Config:    DefaultCheckConfig(),
			},
			wantErr: true,
		},
		{
			name: "invalid endpoint label",
			dep: Dependency{
				Name:     "test",
				Type:     TypeRedis,
				Critical: &critFalse,
				Endpoints: []Endpoint{{
					Host:   "redis",
					Port:   "6379",
					Labels: map[string]string{"name": "bad"},
				}},
				Config: DefaultCheckConfig(),
			},
			wantErr: true,
		},
		{
			name: "valid endpoint labels",
			dep: Dependency{
				Name:     "test",
				Type:     TypeRedis,
				Critical: &critFalse,
				Endpoints: []Endpoint{{
					Host:   "redis",
					Port:   "6379",
					Labels: map[string]string{"role": "primary", "shard": "01"},
				}},
				Config: DefaultCheckConfig(),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.dep.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
