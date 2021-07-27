package sms

//go:generate mockgen -destination=mocks/mock_service.go -package=mocks github.com/ory/kratos/selfservice/strategy/sms Flow

import (
	"context"
	"github.com/gofrs/uuid"
	"github.com/ory/kratos/driver/clock"
	"github.com/ory/kratos/driver/config"
	"github.com/pkg/errors"
	"time"
)

type Flow interface {
	GetID() uuid.UUID
	Valid() error
}

type SmsAuthenticationService interface {
	SendCode(ctx context.Context, flow Flow, phone string) error
	VerifyCode(ctx context.Context, flow Flow, code string) error
}

type dependencies interface {
	config.Provider
	clock.Provider
	CodePersistenceProvider
	NotificationClientProvider
	RandomCodeGeneratorProvider
}

type AuthenticationService struct {
	r dependencies
}

type SmsAuthenticationServiceProvider interface {
	SmsAuthenticationService() *AuthenticationService
}

func NewSmsAuthenticationService(r dependencies) *AuthenticationService {
	return &AuthenticationService{r}
}

// SendCode
// Sends a new code to the user in an SMS message.
// Returns error if the code was already sent and is not expired yet.
func (s *AuthenticationService) SendCode(ctx context.Context, flow Flow, phone string) error {
	if err := flow.Valid(); err != nil {
		return err
	}
	code, err := s.r.CodePersister().FindSmsCode(ctx, flow.GetID(), s.r.Clock().Now())
	if err != nil {
		return err
	}
	if code != nil {
		return errors.New("active code found, will not resend until it expires")
	}

	codeValue := s.r.RandomCodeGenerator().Generate(4)
	if err := s.r.SmsNotificationClient().Send(ctx, phone, codeValue); err != nil {
		return err
	}
	if err := s.r.CodePersister().CreateSmsCode(ctx, &Code{
		FlowId:    flow.GetID(),
		Phone:     phone,
		Code:      codeValue,
		ExpiresAt: s.r.Clock().Now().Add(time.Minute), //TODO Read from config
	}); err != nil {
		return err
	}
	return nil
}

// VerifyCode
// Verifies SMS code by looking up in db.
func (s *AuthenticationService) VerifyCode(ctx context.Context, flow Flow, code string) error {
	if err := flow.Valid(); err != nil {
		return err
	}
	expectedCode, err := s.r.CodePersister().FindSmsCode(ctx, flow.GetID(), s.r.Clock().Now())
	if err != nil {
		return err
	}
	if expectedCode == nil {
		return errors.New("active code not found")
	} else if expectedCode.Code != code {
		return errors.WithStack(NewInvalidSmsCodeError())
	}

	return nil
}
