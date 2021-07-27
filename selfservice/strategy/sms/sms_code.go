package sms

import (
	"context"
	"github.com/gofrs/uuid"
	"github.com/ory/kratos/corp"
	"time"
)

type Code struct {
	ID        uuid.UUID `json:"-" faker:"-" db:"id"`
	Phone     string    `json:"-" faker:"phone_number" db:"phone"`
	Code      string    `json:"-" db:"code"`
	FlowId    uuid.UUID `json:"-" faker:"-" db:"flow_id"`
	ExpiresAt time.Time `json:"-" faker:"time_type" db:"expires_at"`

	// CreatedAt is a helper struct field for gobuffalo.pop.
	CreatedAt time.Time `json:"-" faker:"-" db:"created_at"`
	// UpdatedAt is a helper struct field for gobuffalo.pop.
	UpdatedAt time.Time `json:"-" faker:"-" db:"updated_at"`
}

func (m Code) TableName(ctx context.Context) string {
	return corp.ContextualizeTableName(ctx, "sms_codes")
}

func (m *Code) GetID() uuid.UUID {
	return m.ID
}
