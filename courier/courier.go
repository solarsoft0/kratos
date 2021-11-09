package courier

import (
	"context"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/pkg/errors"

	"github.com/ory/herodot"
	"github.com/ory/kratos/driver/config"
	"github.com/ory/kratos/x"
	gomail "github.com/ory/mail/v3"
)

type (
	Dependencies interface {
		PersistenceProvider
		x.LoggingProvider
		config.Provider
	}

	Courier struct {
		smsClient  *smsClient
		smtpDialer *gomail.Dialer
		deps       Dependencies
	}

	Provider interface {
		Courier(ctx context.Context) *Courier
	}
)

func NewCourier(d Dependencies, c *config.Config) *Courier {
	return &Courier{
		smsClient:  newSMS(c),
		smtpDialer: newSMTP(c),
		deps:       d,
	}
}

func (c *Courier) Work(ctx context.Context) error {
	errChan := make(chan error)
	defer close(errChan)

	go c.watchMessages(ctx, errChan)

	select {
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.Canceled) {
			return nil
		}
		return ctx.Err()
	case err := <-errChan:
		return err
	}
}

func (c *Courier) watchMessages(ctx context.Context, errChan chan error) {
	for {
		if err := backoff.Retry(func() error {
			return c.DispatchQueue(ctx)
		}, backoff.NewExponentialBackOff()); err != nil {
			errChan <- err
			return
		}
		time.Sleep(time.Second)
	}
}

func (c *Courier) DispatchMessage(ctx context.Context, msg Message) error {
	switch msg.Type {
	case MessageTypeEmail:
		if err := c.dispatchEmail(ctx, msg); err != nil {
			return err
		}
	case MessageTypePhone:
		if err := c.dispatchSMS(ctx, msg); err != nil {
			return err
		}
	default:
		return errors.New("received unexpected message type")
	}

	if err := c.deps.CourierPersister().SetMessageStatus(ctx, msg.ID, MessageStatusSent); err != nil {
		c.deps.Logger().
			WithError(err).
			WithField("message_id", msg.ID).
			Error(`Unable to set the message status to "sent".`)
		return err
	}

	c.deps.Logger().
		WithField("message_id", msg.ID).
		WithField("message_type", msg.Type).
		WithField("message_template_type", msg.TemplateType).
		WithField("message_subject", msg.Subject).
		Debug("Courier sent out message.")

	return errors.Errorf("received unexpected message type: %d", msg.Type)
}

func (c *Courier) DispatchQueue(ctx context.Context) error {
	if len(c.smtpDialer.Host) == 0 {
		return errors.WithStack(herodot.ErrInternalServerError.WithReasonf("Courier tried to deliver an email but courier.smtp_url is not set!"))
	}
	if len(c.smsClient.Host) == 0 {
		return errors.WithStack(herodot.ErrInternalServerError.WithReasonf("Courier tried to deliver a sms but courier.sms.host is not set!"))
	}

	messages, err := c.deps.CourierPersister().NextMessages(ctx, 10)
	if err != nil {
		if errors.Is(err, ErrQueueEmpty) {
			return nil
		}
		return err
	}

	for k := range messages {
		var msg = messages[k]
		if err := c.DispatchMessage(ctx, msg); err != nil {
			for _, replace := range messages[k:] {
				if err := c.deps.CourierPersister().SetMessageStatus(ctx, replace.ID, MessageStatusQueued); err != nil {
					c.deps.Logger().
						WithError(err).
						WithField("message_id", replace.ID).
						Error(`Unable to reset the failed message's status to "queued".`)
				}
			}

			return err
		}
	}

	return nil
}
