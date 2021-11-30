package courier

import (
	"context"
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

func (c *Courier) QueueSMS(ctx context.Context, t EmailTemplate) (uuid.UUID, error) {
	message := &Message{
		Status: MessageStatusQueued,
		Type:   MessageTypePhone,
	}
	if err := c.deps.CourierPersister().AddMessage(ctx, message); err != nil {
		return uuid.Nil, err
	}

	return message.ID, nil
}

func (c *Courier) dispatchSMS(ctx context.Context, msg Message) error {
	from := c.deps.Config(ctx).CourierSMSFrom()

	v := url.Values{}
	v.Set("To", msg.Recipient)
	v.Set("From", from)
	v.Set("Body", msg.Body)

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
