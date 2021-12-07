package driver

import (
	"github.com/ory/kratos/selfservice/strategy/sms"
)

func (m *RegistryDefault) SmsAuthenticationService() sms.AuthenticationService {
	if m.selfserviceSmsAuthenticationService == nil {
		m.selfserviceSmsAuthenticationService = sms.NewSmsAuthenticationService(m)
	}

	return m.selfserviceSmsAuthenticationService
}

func (m *RegistryDefault) RandomCodeGenerator() sms.RandomCodeGenerator {
	if m.selfserviceSmsRandomCodeGenerator == nil {
		m.selfserviceSmsRandomCodeGenerator = sms.NewRandomCodeGenerator()
	}

	return m.selfserviceSmsRandomCodeGenerator
}
