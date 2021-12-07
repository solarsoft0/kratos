package template

import (
	"encoding/json"

	"github.com/ory/kratos/driver/config"
)

type TestEmailStub struct {
	c *config.Config
	m *TestEmailStubModel
}

type TestEmailStubModel struct {
	To      string
	Subject string
	Body    string
}

func NewTestEmailStub(c *config.Config, m *TestEmailStubModel) *TestEmailStub {
	return &TestEmailStub{c: c, m: m}
}

func (t *TestEmailStub) EmailRecipient() (string, error) {
	return t.m.To, nil
}

func (t *TestEmailStub) EmailSubject() (string, error) {
	return loadTextTemplate(t.c.CourierTemplatesRoot(), "test_stub/email.subject.gotmpl", "test_stub/email.subject*", t.m)
}

func (t *TestEmailStub) EmailBody() (string, error) {
	return loadHTMLTemplate(t.c.CourierTemplatesRoot(), "test_stub/email.body.gotmpl", "test_stub/email.body*", t.m)
}

func (t *TestEmailStub) EmailBodyPlaintext() (string, error) {
	return loadTextTemplate(t.c.CourierTemplatesRoot(), "test_stub/email.body.plaintext.gotmpl", "test_stub/email.body.plaintext*", t.m)
}

func (t *TestEmailStub) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.m)
}
