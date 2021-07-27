package sms

//go:generate mockgen -destination=mocks/mock_persistence.go -package=mocks github.com/ory/kratos/selfservice/strategy/sms CodePersister

import (
	"context"
	"github.com/gofrs/uuid"
	"time"
)

type CodePersister interface {
	CreateSmsCode(ctx context.Context, smsCode *Code) error

	// FindSmsCode selects code by login flow id and expiration date/time.
	FindSmsCode(ctx context.Context, flowId uuid.UUID, expiresAfter time.Time) (*Code, error)
}

type CodePersistenceProvider interface {
	CodePersister() CodePersister
}
