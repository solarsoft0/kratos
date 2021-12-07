package courier

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	"github.com/ory/kratos/driver/config"
	gomail "github.com/ory/mail/v3"
)

func newSMTP(c *config.Config) *gomail.Dialer {
	uri := c.CourierSMTPURL()
	password, _ := uri.User.Password()
	port, _ := strconv.ParseInt(uri.Port(), 10, 0)

	dialer := &gomail.Dialer{
		Host:     uri.Hostname(),
		Port:     int(port),
		Username: uri.User.Username(),
		Password: password,

		Timeout:      time.Second * 10,
		RetryFailure: true,
	}

	sslSkipVerify, _ := strconv.ParseBool(uri.Query().Get("skip_ssl_verify"))

	// SMTP schemes
	// smtp: smtp clear text (with uri parameter) or with StartTLS (enforced by default)
	// smtps: smtp with implicit TLS (recommended way in 2021 to avoid StartTLS downgrade attacks
	//    and defaulting to fully-encrypted protocols https://datatracker.ietf.org/doc/html/rfc8314)
	switch uri.Scheme {
	case "smtp":
		// Enforcing StartTLS by default for security best practices (config review, etc.)
		skipStartTLS, _ := strconv.ParseBool(uri.Query().Get("disable_starttls"))
		if !skipStartTLS {
			// #nosec G402 This is ok (and required!) because it is configurable and disabled by default.
			dialer.TLSConfig = &tls.Config{InsecureSkipVerify: sslSkipVerify, ServerName: uri.Hostname()}
			// Enforcing StartTLS
			dialer.StartTLSPolicy = gomail.MandatoryStartTLS
		}
	case "smtps":
		// #nosec G402 This is ok (and required!) because it is configurable and disabled by default.
		dialer.TLSConfig = &tls.Config{InsecureSkipVerify: sslSkipVerify, ServerName: uri.Hostname()}
		dialer.SSL = true
	}

	return dialer
}
func (c *courierImpl) SmtpDialer() *gomail.Dialer {
	return c.smtpDialer
}

func (c *courierImpl) QueueEmail(ctx context.Context, t EmailTemplate) (uuid.UUID, error) {
	recipient, err := t.EmailRecipient()
	if err != nil {
		return uuid.Nil, err
	}

	subject, err := t.EmailSubject()
	if err != nil {
		return uuid.Nil, err
	}

	bodyPlaintext, err := t.EmailBodyPlaintext()
	if err != nil {
		return uuid.Nil, err
	}

	templateType, err := GetEmailTemplateType(t)
	if err != nil {
		return uuid.Nil, err
	}

	templateData, err := json.Marshal(t)
	if err != nil {
		return uuid.Nil, err
	}

	message := &Message{
		Status:       MessageStatusQueued,
		Type:         MessageTypeEmail,
		Recipient:    recipient,
		Body:         bodyPlaintext,
		Subject:      subject,
		TemplateType: templateType,
		TemplateData: templateData,
	}
	if err := c.deps.CourierPersister().AddMessage(ctx, message); err != nil {
		return uuid.Nil, err
	}

	return message.ID, nil
}

func (c *courierImpl) dispatchEmail(ctx context.Context, msg Message) error {
	from := c.deps.Config(ctx).CourierSMTPFrom()
	fromName := c.deps.Config(ctx).CourierSMTPFromName()
	gm := gomail.NewMessage()
	if fromName == "" {
		gm.SetHeader("From", from)
	} else {
		gm.SetAddressHeader("From", from, fromName)
	}

	gm.SetHeader("To", msg.Recipient)
	gm.SetHeader("Subject", msg.Subject)

	headers := c.deps.Config(ctx).CourierSMTPHeaders()
	for k, v := range headers {
		gm.SetHeader(k, v)
	}

	gm.SetBody("text/plain", msg.Body)

	tmpl, err := NewEmailTemplateFromMessage(c.deps.Config(ctx), msg)
	if err != nil {
		c.deps.Logger().
			WithError(err).
			WithField("message_id", msg.ID).
			Error(`Unable to get email template from message.`)
	} else {
		htmlBody, err := tmpl.EmailBody()
		if err != nil {
			c.deps.Logger().
				WithError(err).
				WithField("message_id", msg.ID).
				Error(`Unable to get email body from template.`)
		} else {
			gm.AddAlternative("text/html", htmlBody)
		}
	}

	if err := c.smtpDialer.DialAndSend(ctx, gm); err != nil {
		c.deps.Logger().
			WithError(err).
			WithField("smtp_server", fmt.Sprintf("%s:%d", c.smtpDialer.Host, c.smtpDialer.Port)).
			WithField("smtp_ssl_enabled", c.smtpDialer.SSL).
			// WithField("email_to", msg.Recipient).
			WithField("message_from", from).
			Error("Unable to send email using SMTP connection.")
		return errors.WithStack(err)
	}

	return nil
}
