package courier

import (
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/ory/kratos/courier/template"
	"github.com/ory/kratos/driver/config"
)

const (
	TypeSmsCode TemplateType = "sms_code"
)

type SmsTemplate interface {
	json.Marshaler
	SmsBody() (string, error)
	SmsRecipientPhone() (string, error)
}

func GetSmsTemplateType(t SmsTemplate) (TemplateType, error) {
	switch t.(type) {
	case *template.SmsLogin:
		return TypeSmsCode, nil
	case *template.TestSmsStub:
		return TypeTestStub, nil
	default:
		return "", errors.Errorf("unexpected template type")
	}
}

func NewSmsTemplateFromMessage(c *config.Config, m Message) (SmsTemplate, error) {
	switch m.TemplateType {
	case TypeSmsCode:
		var t template.SmsLoginModel
		if err := json.Unmarshal(m.TemplateData, &t); err != nil {
			return nil, err
		}
		return template.NewSmsLogin(c, &t), nil
	case TypeTestStub:
		var t template.TestSmsStubModel
		if err := json.Unmarshal(m.TemplateData, &t); err != nil {
			return nil, err
		}
		return template.NewTestSmsStub(c, &t), nil
	default:
		return nil, errors.Errorf("received unexpected message template type: %s", m.TemplateType)
	}
}
