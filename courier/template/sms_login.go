package template

import (
	"encoding/json"

	"github.com/ory/kratos/driver/config"
)

type (
	SmsLogin struct {
		c *config.Config
		m *SmsLoginModel
	}
	SmsLoginModel struct {
		Code  string
		Phone string
	}
)

func NewSmsLogin(c *config.Config, m *SmsLoginModel) *SmsLogin {
	return &SmsLogin{c: c, m: m}
}

func (t *SmsLogin) SmsRecipientPhone() (string, error) {
	return t.m.Phone, nil
}

func (t *SmsLogin) SmsBody() (string, error) {
	return loadTextTemplate(t.c.CourierTemplatesRoot(), "login/sms.body.gotmpl", "login/sms.body*", t.m)
}

func (t *SmsLogin) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.m)
}
