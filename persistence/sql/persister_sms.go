package sql

import (
	"context"
	"github.com/gofrs/uuid"
	"github.com/ory/kratos/selfservice/strategy/sms"
	"github.com/ory/x/sqlcon"
	"time"
)

var _ sms.CodePersister = new(Persister)

func (p *Persister) CreateSmsCode(ctx context.Context, smsCode *sms.Code) error {
	return p.GetConnection(ctx).Create(smsCode)
}

func (p *Persister) FindSmsCode(ctx context.Context, flowId uuid.UUID, expiresAfter time.Time) (*sms.Code, error) {
	var r []sms.Code
	if err := p.GetConnection(ctx).Where("flow_id = ? AND expires_at > ?", flowId, expiresAfter).All(&r); err != nil {
		return nil, sqlcon.HandleError(err)
	}
	if len(r) > 0 {
		return &r[0], nil
	}
	return nil, nil
}
