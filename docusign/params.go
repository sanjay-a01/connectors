package docusign

import (
	"context"
	"errors"
	"net/http"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/paramsbuilder"
	"golang.org/x/oauth2"
)

type Option func(params *parameters)

type parameters struct {
	paramsbuilder.Client
}

func (p parameters) FromOptions(opts ...Option) (*parameters, error) {
	params := &p
	for _, opt := range opts {
		opt(params)
	}

	return params, params.ValidateParams()
}

func (p parameters) ValidateParams() error {
	return errors.Join(
		p.Client.ValidateParams(),
	)
}

// WithClient sets the http client to use for the connector. Saves some boilerplate.
func WithClient(ctx context.Context, client *http.Client,
	config *oauth2.Config, token *oauth2.Token, opts ...common.OAuthOption,
) Option {
	return func(params *parameters) {
		params.WithClient(ctx, client, config, token, opts...)
	}
}

func WithAuthenticatedClient(client common.AuthenticatedHTTPClient) Option {
	return func(params *parameters) {
		params.WithAuthenticatedClient(client)
	}
}
