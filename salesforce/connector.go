package salesforce

import (
	"fmt"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/interpreter"
	"github.com/amp-labs/connectors/common/paramsbuilder"
	"github.com/amp-labs/connectors/common/urlbuilder"
	"github.com/amp-labs/connectors/providers"
)

const (
	apiVersion                 = "59.0"
	versionPrefix              = "v"
	version                    = versionPrefix + apiVersion
	restAPISuffix              = "/services/data/" + version
	uriSobjects                = restAPISuffix + "/sobjects"
	uriToolingEventRelayConfig = "tooling/sobjects/EventRelayConfig"
)

// Connector is a Salesforce connector.
type Connector struct {
	BaseURL   string
	Client    *common.JSONHTTPClient
	XMLClient *common.XMLHTTPClient
}

func APIVersionSOAP() string {
	return apiVersion
}

// NewConnector returns a new Salesforce connector.
func NewConnector(opts ...Option) (conn *Connector, outErr error) {
	defer common.PanicRecovery(func(cause error) {
		outErr = cause
		conn = nil
	})

	params, err := paramsbuilder.Apply(parameters{}, opts)
	if err != nil {
		return nil, err
	}

	httpClient := params.Client.Caller
	conn = &Connector{
		Client: &common.JSONHTTPClient{
			HTTPClient: httpClient,
			ErrorPostProcessor: common.ErrorPostProcessor{
				Process: handleError,
			},
		},
		XMLClient: &common.XMLHTTPClient{
			HTTPClient: httpClient,
			ErrorPostProcessor: common.ErrorPostProcessor{
				Process: handleError,
			},
		},
	}

	providerInfo, err := providers.ReadInfo(conn.Provider(), &params.Workspace)
	if err != nil {
		return nil, err
	}

	conn.setBaseURL(providerInfo.BaseURL)
	conn.Client.HTTPClient.ErrorHandler = interpreter.ErrorHandler{
		JSON: &interpreter.DirectFaultyResponder{Callback: conn.interpretJSONError},
		XML:  &interpreter.DirectFaultyResponder{Callback: conn.interpretXMLError},
	}.Handle

	return conn, nil
}

// Provider returns the connector provider.
func (c *Connector) Provider() providers.Provider {
	return providers.Salesforce
}

// String returns a string representation of the connector, which is useful for logging / debugging.
func (c *Connector) String() string {
	return fmt.Sprintf("%s.Connector", c.Provider())
}

func (c *Connector) getRestApiURL(paths ...string) (*urlbuilder.URL, error) {
	parts := append([]string{
		restAPISuffix, // scope URLs to API version
	}, paths...)

	return urlbuilder.New(c.BaseURL, parts...)
}

func (c *Connector) getDomainURL(paths ...string) (*urlbuilder.URL, error) {
	return urlbuilder.New(c.BaseURL, paths...)
}

func (c *Connector) getSoapURL() (*urlbuilder.URL, error) {
	return urlbuilder.New(c.BaseURL, "services/Soap/m", APIVersionSOAP())
}

// nolint: lll
// https://developer.salesforce.com/docs/atlas.en-us.api_tooling.meta/api_tooling/tooling_api_objects_eventrelayconfig.htm?q=EventRelayConfig
func (c *Connector) getURIPartEventRelayConfig(paths ...string) (*urlbuilder.URL, error) {
	return urlbuilder.New(uriToolingEventRelayConfig, paths...)
}

func (c *Connector) getURIPartSobjectsDescribe(objectName string) (*urlbuilder.URL, error) {
	return urlbuilder.New(uriSobjects, objectName, "describe")
}

func (c *Connector) setBaseURL(newURL string) {
	c.BaseURL = newURL
	c.Client.HTTPClient.Base = newURL
}
