package sms_test

import (
	"context"
	"github.com/benbjohnson/clock"
	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"github.com/ory/kratos/driver/config"
	"github.com/ory/kratos/internal"
	"github.com/ory/kratos/selfservice/flow"
	"github.com/ory/kratos/selfservice/strategy/sms"
	smsMock "github.com/ory/kratos/selfservice/strategy/sms/mocks"
	"github.com/pkg/errors"
	"log"
	"testing"
	"time"
)

type testContext struct {
	context    context.Context
	controller *gomock.Controller
	config     *config.Config
}

func TestAuthenticationService_SendCode(t *testing.T) {
	tc := testContext{
		context.Background(),
		gomock.NewController(t),
		internal.NewConfigurationWithDefaults(t),
	}

	tests := []struct {
		name    string
		service sms.SmsAuthenticationService
		flow    sms.Flow
		phone   string
		wantErr bool
	}{
		{"error if flow is not active",
			tc.NewSmsAuthenticationService(
				tc.repoNoCalls(),
				tc.notifierNoCalls(),
				clock.NewMock(),
				tc.smsCode("0000"),
			),
			tc.invalidFlow(),
			"000000",
			true,
		},
		{"send code when not sent before",
			tc.NewSmsAuthenticationService(
				tc.repoNoCodeCreateCode(),
				tc.notifier("1234"),
				clock.NewMock(),
				tc.smsCode("0000"),
			),
			tc.validFlow(),
			"1234",
			false,
		},
		{"block resending if timeout not expired",
			tc.NewSmsAuthenticationService(
				tc.repoActiveCode("555", newTime("2021-07-10T12:00:00Z")),
				tc.notifierNoCalls(),
				fixClock("2021-07-10T12:00:00Z"),
				tc.smsCode("0000"),
			),
			tc.validFlow(),
			"555",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.service.SendCode(tc.context, tt.flow, tt.phone); (err != nil) != tt.wantErr {
				t.Errorf("SendCode() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAuthenticationService_VerifyCode(t *testing.T) {
	tc := testContext{
		context.Background(),
		gomock.NewController(t),
		internal.NewConfigurationWithDefaults(t),
	}

	tests := []struct {
		name    string
		service sms.SmsAuthenticationService
		flow    sms.Flow
		code    string
		wantErr bool
	}{
		{"error if flow is not active",
			tc.NewSmsAuthenticationService(
				tc.repoNoCalls(),
				tc.notifierNoCalls(),
				clock.NewMock(),
				tc.smsCode("0000"),
			),
			tc.invalidFlow(),
			"0000",
			true,
		},
		{"code not found or expired",
			tc.NewSmsAuthenticationService(
				tc.repoNoCode(),
				tc.notifierNoCalls(),
				clock.NewMock(),
				tc.smsCode("0000"),
			),
			tc.validFlow(),
			"1234",
			true,
		},
		{"code didn't match",
			tc.NewSmsAuthenticationService(
				tc.repoActiveCode("0000", newTime("2021-07-10T12:00:00Z")),
				tc.notifierNoCalls(),
				fixClock("2021-07-10T12:00:00Z"),
				tc.smsCode("0000"),
			),
			tc.validFlow(),
			"1234",
			true,
		},
		{"code match",
			tc.NewSmsAuthenticationService(
				tc.repoActiveCode("0000", newTime("2021-07-10T12:00:00Z")),
				tc.notifierNoCalls(),
				fixClock("2021-07-10T12:00:00Z"),
				tc.smsCode("0000"),
			),
			tc.validFlow(),
			"0000",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.service.VerifyCode(tc.context, tt.flow, tt.code); (err != nil) != tt.wantErr {
				t.Errorf("VerifyCode() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func newTime(s string) time.Time {
	tm, err := time.Parse(time.RFC3339, s)
	if err != nil {
		log.Fatal(err)
	}
	return tm
}

func fixClock(t string) clock.Clock {
	c := clock.NewMock()
	c.Set(newTime(t))
	return c
}

func (tc *testContext) invalidFlow() sms.Flow {
	m := smsMock.NewMockFlow(tc.controller)
	m.EXPECT().Valid().Return(errors.WithStack(flow.NewFlowExpiredError(time.Now())))
	return m
}

func (tc *testContext) validFlow() sms.Flow {
	m := smsMock.NewMockFlow(tc.controller)
	m.EXPECT().Valid().Return(nil)
	m.EXPECT().GetID().MinTimes(1).Return(uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"))
	return m
}

func (tc *testContext) repoNoCode() sms.CodePersister {
	m := smsMock.NewMockCodePersister(tc.controller)
	m.EXPECT().FindSmsCode(tc.context, gomock.Any(), gomock.Any()).Return(nil, nil)
	return m
}

func (tc *testContext) repoNoCodeCreateCode() sms.CodePersister {
	m := smsMock.NewMockCodePersister(tc.controller)
	m.EXPECT().FindSmsCode(tc.context, gomock.Any(), gomock.Any()).Return(nil, nil)
	m.EXPECT().CreateSmsCode(tc.context, gomock.Any())
	return m
}

func (tc *testContext) repoActiveCode(code string, t time.Time) sms.CodePersister {
	m := smsMock.NewMockCodePersister(tc.controller)
	m.EXPECT().FindSmsCode(tc.context, gomock.Any(), t).Return(
		&sms.Code{
			Phone: "11111",
			Code:  code,
		},
		nil,
	)
	return m
}

func (tc *testContext) repoNoCalls() sms.CodePersister {
	m := smsMock.NewMockCodePersister(tc.controller)
	return m
}

func (tc *testContext) notifierNoCalls() sms.NotificationClient {
	return smsMock.NewMockNotificationClient(tc.controller)
}

func (tc *testContext) notifier(phone string) sms.NotificationClient {
	m := smsMock.NewMockNotificationClient(tc.controller)
	m.EXPECT().Send(tc.context, phone, gomock.Any())
	return m
}

func (tc *testContext) lifespan(s string) *config.Config {
	tc.config.MustSet(config.SmsLifespan, s)
	return tc.config
}

type randomCodeGeneratorStub struct {
	code string
}

//goland:noinspection GoUnusedParameter
func (s *randomCodeGeneratorStub) Generate(max int) string {
	return s.code
}

func (tc *testContext) NewSmsAuthenticationService(
	codePersister sms.CodePersister,
	notificationClient sms.NotificationClient,
	clock clock.Clock,
	randomCodeGenerator sms.RandomCodeGenerator,
) sms.SmsAuthenticationService {

	return sms.NewSmsAuthenticationService(&dependencies{
		tc.config,
		codePersister,
		notificationClient,
		clock,
		randomCodeGenerator,
	})
}

func (tc *testContext) smsCode(code string) sms.RandomCodeGenerator {
	return &randomCodeGeneratorStub{code: code}
}

type dependencies struct {
	config              *config.Config
	codePersister       sms.CodePersister
	notificationClient  sms.NotificationClient
	clock               clock.Clock
	randomCodeGenerator sms.RandomCodeGenerator
}

func (d *dependencies) Clock() clock.Clock {
	return d.clock
}

func (d *dependencies) CodePersister() sms.CodePersister {
	return d.codePersister
}

func (d *dependencies) SmsNotificationClient() sms.NotificationClient {
	return d.notificationClient
}

//goland:noinspection GoUnusedParameter
func (d *dependencies) Config(ctx context.Context) *config.Config { return d.config }

func (d *dependencies) RandomCodeGenerator() sms.RandomCodeGenerator {
	return d.randomCodeGenerator
}
