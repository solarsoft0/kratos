package identity

import (
	"fmt"
	"strings"
	"sync"

	"github.com/ory/go-convenience/stringslice"
	"github.com/ory/jsonschema/v3"
	"github.com/ory/x/sqlxx"

	"github.com/ory/kratos/schema"
)

type SchemaExtensionCredentials struct {
	i *Identity
	v []string
	l sync.Mutex
}

func NewSchemaExtensionCredentials(i *Identity) *SchemaExtensionCredentials {
	return &SchemaExtensionCredentials{i: i}
}

func (r *SchemaExtensionCredentials) Run(_ jsonschema.ValidationContext, s schema.ExtensionConfig, value interface{}) error {
	r.l.Lock()
	defer r.l.Unlock()
	if s.Credentials.Password.Identifier {
		r.setCredentials(value, CredentialsTypePassword)
	} else if s.Credentials.Sms.Identifier {
		r.setCredentials(value, CredentialsTypeSMS)
	}
	return nil
}

func (r *SchemaExtensionCredentials) setCredentials(value interface{}, t CredentialsType) {
	cred, ok := r.i.GetCredentials(t)
	if !ok {
		cred = &Credentials{
			Type:        t,
			Identifiers: []string{},
			Config:      sqlxx.JSONRawMessage{},
		}
	}

	r.v = stringslice.Unique(append(r.v, strings.ToLower(fmt.Sprintf("%s", value))))
	cred.Identifiers = r.v
	r.i.SetCredentials(t, *cred)
}

func (r *SchemaExtensionCredentials) Finish() error {
	return nil
}
