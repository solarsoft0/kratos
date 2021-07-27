package sms

import (
	"github.com/ory/kratos/driver/config"
	"github.com/ory/kratos/identity"
	"github.com/ory/kratos/session"
	"github.com/ory/kratos/ui/node"
	"github.com/ory/kratos/x"
	"github.com/ory/x/decoderx"
)

type strategyDependencies interface {
	config.Provider
	x.CSRFTokenGeneratorProvider
	session.ManagementProvider
	identity.PrivilegedPoolProvider
	identity.ManagementProvider
	identity.ValidationProvider
	x.LoggingProvider
	SmsAuthenticationServiceProvider
}

type Strategy struct {
	d  strategyDependencies
	hd *decoderx.HTTP
}

func NewStrategy(d strategyDependencies) *Strategy {
	return &Strategy{
		d: d,
	}
}

func (s *Strategy) ID() identity.CredentialsType {
	return identity.CredentialsTypeSMS
}

func (s *Strategy) NodeGroup() node.Group {

	return ""
}

func (s *Strategy) RegisterLoginRoutes(*x.RouterPublic) {

}
