package domain

import "time"

type Endpoint struct {
    ID          string    `json:"id"`
    TenantID    string    `json:"tenant_id"`
    URL         string    `json:"url"`
    Description string    `json:"description,omitempty"`
    EventTypes  []string  `json:"event_types"`
    Secret      string    `json:"-"`
    Active      bool      `json:"active"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

type CreateEndpointInput struct {
    URL         string   `json:"url"`
    Description string   `json:"description"`
    EventTypes  []string `json:"event_types"`
    Active      *bool    `json:"active"`
}

type UpdateEndpointInput struct {
    URL         *string  `json:"url"`
    Description *string  `json:"description"`
    EventTypes  []string `json:"event_types"`
    Active      *bool    `json:"active"`
}

func (i CreateEndpointInput) Validate() []ValidationError {
    var errs []ValidationError
    if i.URL == "" {
        errs = append(errs, ValidationError{Field: "url", Message: "is required"})
    } else if len(i.URL) > 2048 {
        errs = append(errs, ValidationError{Field: "url", Message: "must be 2048 characters or less"})
    }
    if len(i.Description) > 1000 {
        errs = append(errs, ValidationError{Field: "description", Message: "must be 1000 characters or less"})
    }
    for _, et := range i.EventTypes {
        if et == "" {
            errs = append(errs, ValidationError{Field: "event_types", Message: "must not contain empty strings"})
            break
        }
    }
    return errs
}