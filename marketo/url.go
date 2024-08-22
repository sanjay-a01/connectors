package marketo

import (
	"fmt"
	"strings"
	"time"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/urlbuilder"
)

var restAPIPrefix string = "rest" //nolint:gochecknoglobals

func (c *Connector) getURL(params common.ReadParams) (*urlbuilder.URL, error) {
	// If NextPage is set, then we're reading the next page of results.
	// The NextPage URL has all the necessary parameters.
	if len(params.NextPage) > 0 {
		return urlbuilder.New(params.NextPage.String())
	}

	link, err := c.getAPIURL(params.ObjectName)
	if err != nil {
		return nil, err
	}

	// This affects  a very few number of objects.
	// Leads, Deleted Leads, Lead Changes,
	if !params.Since.IsZero() {
		time := params.Since.Format(time.RFC3339)
		fmtTime := fmt.Sprintf("%v", time)
		link.WithQueryParam("sinceDatetime", fmtTime)
	}

	return link, nil
}

func updateURLWithID(url *urlbuilder.URL, id string) (*urlbuilder.URL, error) {
	s := removeJSONSuffix(url.String())

	url, err := urlbuilder.New(s, id)
	if err != nil {
		return nil, err
	}

	s = addJSONSuffix(url.String())

	return urlbuilder.New(s)
}

func removeJSONSuffix(s string) string {
	return strings.TrimSuffix(s, ".json")
}

func addJSONSuffix(s string) string {
	return s + ".json"
}
