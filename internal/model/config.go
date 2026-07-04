package model

import (
	"fmt"
	"regexp"
	"time"
)

var idPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

var validLogLevels = map[string]bool{
	"DEBUG": true,
	"INFO":  true,
	"WARN":  true,
	"ERROR": true,
}

type Config struct {
	ID        string    `json:"id" db:"id"`
	Host      string    `json:"host" db:"host"`
	Port      int       `json:"port" db:"port"`
	AppName   string    `json:"app_name" db:"app_name"`
	LogLevel  string    `json:"log_level" db:"log_level"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type UpsertConfigRequest struct {
	ID       string `json:"id"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	AppName  string `json:"app_name"`
	LogLevel string `json:"log_level"`
}

func (r UpsertConfigRequest) Validate() []string {
	var errs []string

	if r.ID == "" {
		errs = append(errs, "id is required")
	} else if len(r.ID) > 255 {
		errs = append(errs, "id must be at most 255 characters")
	} else if !idPattern.MatchString(r.ID) {
		errs = append(errs, "id must contain only alphanumeric characters, hyphens, and underscores")
	}

	if r.Host == "" {
		errs = append(errs, "host is required")
	}

	if r.Port < 1 || r.Port > 65535 {
		errs = append(errs, fmt.Sprintf("port must be between 1 and 65535, got %d", r.Port))
	}

	if r.AppName == "" {
		errs = append(errs, "app_name is required")
	}

	if !validLogLevels[r.LogLevel] {
		errs = append(errs, "log_level must be one of: DEBUG, INFO, WARN, ERROR")
	}

	return errs
}
