package domain

import "time"

type Tenant struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	APIKey             string    `json:"-"`
	RateLimitPerMinute int       `json:"rate_limit_per_minute"`
	MaxRetries         int       `json:"max_retries"`
	RetryBaseMs        int       `json:"retry_base_ms"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type CreateTenantInput struct {
	Name               string `json:"name"`
	RateLimitPerMinute int    `json:"rate_limit_per_minute"`
	MaxRetries         int    `json:"max_retries"`
	RetryBaseMs        int    `json:"retry_base_ms"`
}

type UpdateTenantInput struct {
	Name               *string `json:"name"`
	RateLimitPerMinute *int    `json:"rate_limit_per_minute"`
	MaxRetries         *int    `json:"max_retries"`
	RetryBaseMs        *int    `json:"retry_base_ms"`
}

func (i CreateTenantInput) Validate() []ValidationError {
	var errs []ValidationError
	if i.Name == "" {
		errs = append(errs, ValidationError{Field: "name", Message: "is required"})
	} else if len(i.Name) > 255 {
		errs = append(errs, ValidationError{Field: "name", Message: "must be 255 characters or less"})
	}
	if i.RateLimitPerMinute < 0 {
		errs = append(errs, ValidationError{Field: "rate_limit_per_minute", Message: "must be non-negative"})
	}
	if i.MaxRetries < 0 || i.MaxRetries > 10 {
		errs = append(errs, ValidationError{Field: "max_retries", Message: "must be between 0 and 10"})
	}
	if i.RetryBaseMs != 0 && i.RetryBaseMs < 100 {
		errs = append(errs, ValidationError{Field: "retry_base_ms", Message: "must be at least 100ms"})
	}
	return errs
}
