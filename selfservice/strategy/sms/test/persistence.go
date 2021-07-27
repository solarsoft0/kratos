package test

import (
	"context"
	"github.com/bxcodec/faker/v3"
	"github.com/gofrs/uuid"
	"github.com/ory/kratos/persistence"
	"github.com/ory/kratos/selfservice/strategy/sms"
	"github.com/ory/kratos/x"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

//goland:noinspection GoNameStartsWithPackageName
func TestCodePersister(ctx context.Context, p persistence.Persister) func(t *testing.T) {
	var clearids = func(r *sms.Code) {
		r.ID = uuid.UUID{}
	}

	return func(t *testing.T) {

		var newCode = func(t *testing.T) *sms.Code {
			var r sms.Code
			require.NoError(t, faker.FakeData(&r))
			clearids(&r)
			return &r
		}

		t.Run("case=should create and fetch a code", func(t *testing.T) {
			expected := newCode(t)
			err := p.CreateSmsCode(ctx, expected)
			require.NoError(t, err)

			actual, err := p.FindSmsCode(ctx, expected.FlowId, expected.ExpiresAt.Add(-time.Minute))
			require.NoError(t, err)

			assert.NotNil(t, actual)
			assert.EqualValues(t, expected.ID, actual.ID)
			assert.EqualValues(t, expected.Phone, actual.Phone)
			x.AssertEqualTime(t, expected.ExpiresAt, actual.ExpiresAt)
		})
	}
}
