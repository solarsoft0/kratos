package courier

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"

	"github.com/gofrs/uuid"

	"github.com/ory/kratos/driver/config"
)

type smsClient struct {
	*http.Client
	Host string
}

func newSMS(c *config.Config) *smsClient {
	return &smsClient{
		Client: &http.Client{},
		Host:   c.CourierSMSHost().String(),
	}

}

func (c *courierImpl) QueueSMS(ctx context.Context, t SmsTemplate) (uuid.UUID, error) {
	recipient, err := t.SmsRecipientPhone()
	if err != nil {
		return uuid.Nil, err
	}

	templateType, err := GetSmsTemplateType(t)
	if err != nil {
		return uuid.Nil, err
	}

	templateData, err := json.Marshal(t)
	if err != nil {
		return uuid.Nil, err
	}

	message := &Message{
		Status:       MessageStatusQueued,
		Type:         MessageTypePhone,
		Recipient:    recipient,
		TemplateType: templateType,
		TemplateData: templateData,
	}
	if err := c.deps.CourierPersister().AddMessage(ctx, message); err != nil {
		return uuid.Nil, err
	}

	return message.ID, nil
}

func (c *courierImpl) dispatchSMS(ctx context.Context, msg Message) error {
	from := c.deps.Config(ctx).CourierSMSFrom()

	tmpl, err := NewSmsTemplateFromMessage(c.deps.Config(ctx), msg)
	if err != nil {
		return err
	}
	body, err := tmpl.SmsBody()
	if err != nil {
		return err
	}

	v := url.Values{}
	v.Set("To", msg.Recipient)
	v.Set("From", from)
	v.Set("Body", body)

	res, err := c.smsClient.PostForm(c.smsClient.Host, v)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return errors.New(http.StatusText(res.StatusCode))
	}

	return nil
}
