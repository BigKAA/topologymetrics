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
	validDep := Dependency{
		Name:     "postgres-main",
		Type:     TypePostgres,
		Critical: true,
		Endpoints: []Endpoint{
			{Host: "pg.svc", Port: "5432"},
		},
		Config: DefaultCheckConfig(),
	}

	if err := validDep.Validate(); err != nil {
		t.Fatalf("valid dependency: Validate() = %v", err)
	}

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
				Endpoints: []Endpoint{{Host: "host", Port: "80"}},
				Config:    DefaultCheckConfig(),
			},
			wantErr: true,
		},
		{
			name: "no endpoints",
			dep: Dependency{
				Name:      "test",
				Type:      TypeRedis,
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
				Endpoints: []Endpoint{{Host: "redis", Port: ""}},
				Config:    DefaultCheckConfig(),
			},
			wantErr: true,
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
