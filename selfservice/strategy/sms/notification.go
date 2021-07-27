package sms

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/ory/herodot"
	"github.com/ory/kratos/driver/config"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	"log"
	"net/http"
)

//go:generate mockgen -destination=mocks/mock_notification.go -package=mocks github.com/ory/kratos/selfservice/strategy/sms NotificationClient

type NotificationClient interface {
	Send(ctx context.Context, phone string, code string) error
}

type notificationClientDependencies interface {
	config.Provider
}

type notificationClientImpl struct {
	r notificationClientDependencies
}

type NotificationClientProvider interface {
	SmsNotificationClient() NotificationClient
}

func NewNotificationClient(r notificationClientDependencies) NotificationClient {
	return &notificationClientImpl{r}
}

type Notification struct {
	Phone                    string `json:"phone"`
	Body                     string `json:"body"`
	IncludeVerificationToken bool   `json:"include_verification_token"`
}

func (c *notificationClientImpl) Send(ctx context.Context, phone string, code string) error {
	client := &http.Client{}
	//TODO change params names and add Twilio params if set in config
	body := &Notification{
		Phone:                    phone,
		Body:                     "Your passcode is: " + code,
		IncludeVerificationToken: false,
	}
	payloadBuf := new(bytes.Buffer)
	if err := json.NewEncoder(payloadBuf).Encode(body); err != nil {
		return err
	}
	req, err := http.NewRequest("POST", c.r.Config(ctx).SelfServiceSmsSenderUrl().String(), payloadBuf)
	if err != nil {
		return errors.WithStack(herodot.ErrInternalServerError.WithReasonf("%s", err))
	}
	//req.SetBasicAuth(accountSid, authToken)
	req.Header.Set("Accept", "application/json")
	//req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Content-Type", "application/json")

	r, err := client.Do(req)
	if err != nil {
		return errors.WithStack(herodot.ErrInternalServerError.WithReasonf("%s", err))
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(r.Body)

	if r.StatusCode >= 300 {
		bodyBytes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Fatal(err)
		}
		return errors.WithStack(herodot.ErrInternalServerError.WithReasonf(
			"Call to sms sending service returned code %d and body: %s",
			r.StatusCode, string(bodyBytes)),
		)
	}

	return nil
}
