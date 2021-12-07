package template

import (
	"encoding/json"

	"github.com/ory/kratos/driver/config"
)

type TestSmsStub struct {
	c *config.Config
	m *TestSmsStubModel
}

type TestSmsStubModel struct {
	Phone string
	Body  string
}

func NewTestSmsStub(c *config.Config, m *TestSmsStubModel) *TestSmsStub {
	return &TestSmsStub{c: c, m: m}
}

func (t *TestSmsStub) SmsRecipientPhone() (string, error) {
	return t.m.Phone, nil
}

func (t *TestSmsStub) SmsBody() (string, error) {
	return loadTextTemplate(t.c.CourierTemplatesRoot(), "test_stub/sms.body.gotmpl", "test_stub/sms.body*", t.m)
}

func (t *TestSmsStub) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.m)
}
