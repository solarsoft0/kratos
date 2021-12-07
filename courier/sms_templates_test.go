package courier_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ory/kratos/courier"
	"github.com/ory/kratos/courier/template"
	"github.com/ory/kratos/internal"
)

func TestGetSmsTemplateType(t *testing.T) {
	for expectedType, tmpl := range map[courier.TemplateType]courier.SmsTemplate{
		courier.TypeSmsCode:  &template.SmsLogin{},
		courier.TypeTestStub: &template.TestSmsStub{},
	} {
		t.Run(fmt.Sprintf("case=%s", expectedType), func(t *testing.T) {
			actualType, err := courier.GetSmsTemplateType(tmpl)
			require.NoError(t, err)
			require.Equal(t, expectedType, actualType)

		})

	}
}

func TestNewSmsTemplateFromMessage(t *testing.T) {
	conf := internal.NewConfigurationWithDefaults(t)
	for tmplType, expectedTmpl := range map[courier.TemplateType]courier.SmsTemplate{
		courier.TypeSmsCode:  template.NewSmsLogin(conf, &template.SmsLoginModel{Code: "0000", Phone: "+12345678901"}),
		courier.TypeTestStub: template.NewTestSmsStub(conf, &template.TestSmsStubModel{Phone: "+12345678901", Body: "test body"}),
	} {
		t.Run(fmt.Sprintf("case=%s", tmplType), func(t *testing.T) {
			tmplData, err := json.Marshal(expectedTmpl)
			require.NoError(t, err)

			m := courier.Message{TemplateType: tmplType, TemplateData: tmplData}
			actualTmpl, err := courier.NewSmsTemplateFromMessage(conf, m)
			require.NoError(t, err)

			require.IsType(t, expectedTmpl, actualTmpl)

			expectedRecipient, err := expectedTmpl.SmsRecipientPhone()
			require.NoError(t, err)
			actualRecipient, err := actualTmpl.SmsRecipientPhone()
			require.NoError(t, err)
			require.Equal(t, expectedRecipient, actualRecipient)

			expectedBody, err := expectedTmpl.SmsBody()
			require.NoError(t, err)
			actualBody, err := actualTmpl.SmsBody()
			require.NoError(t, err)
			require.Equal(t, expectedBody, actualBody)

		})
	}
}
