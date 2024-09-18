package main

import (
	"log/slog"

	"github.com/amp-labs/connectors/providers/zendesksupport/metadata"
	"github.com/amp-labs/connectors/providers/zendesksupport/openapi"
	"github.com/amp-labs/connectors/tools/fileconv/api3"
	"github.com/amp-labs/connectors/tools/scrapper"
)

var (
	ignoreEndpoints = []string{ // nolint:gochecknoglobals
		// Wild rules.
		"/api/lotus/*",
		"*/create_many",
		"*/update_many",
		"*/destroy_many",
		"*/reorder",
		"*/count",
		"*/create_or_update",
		"*/show_many",
		"/api/v2/incremental/*",
		"/api/v2/autocomplete/*",
		"*/autocomplete",
		"*/active",
		"*/export",
		"*/definitions",
		"*/assignable",
		// Complex Path with multiple slashes.
		"/api/v2/channels/twitter/tickets",
		"/api/v2/suspended_tickets/attachments",
		"/api/v2/dynamic_content/items",
		"/api/v2/slas/policies",
		"/api/v2/macros/*",
		"/api/v2/object_layouts/essentials_cards",
		"/api/v2/locales/public",
		"/api/v2/views/compact",
		"/api/v2/locales/agent",
		"/api/v2/group_slas/policies",
		"/api/v2/slas/policies",
		"/api/v2/routing/requirements/fulfilled",
		// Resources with search.
		"/api/v2/users/search",
		"/api/v2/requests/search",
		"/api/v2/organizations/search",
		"/api/v2/automations/search",
		"/api/v2/views/search",
		"/api/v2/triggers/search",
		// Not applicable.
		"/api/v2/channels/voice/tickets", // only POST method for create.
		"/api/v2/imports/tickets",        // only POST method for create.
		"/api/v2/custom_objects/limits/object_limit",
		"/api/v2/users/me/session/renew",
		"/api/v2/locales/current",
		"/api/v2/locales/detect_best_locale",
		"/api/v2/brands/check_host_mapping",
		"/api/v2/views/count_many",
		"/api/v2/accounts/available",
		"/api/v2/users/me",
		"/api/v2/custom_objects/limits/record_limit",
		"/api/v2/account/settings",
	}
	displayNameOverride = map[string]string{ // nolint:gochecknoglobals
		"search":               "Search Results",
		"trigger_categories":   "Trigger Categories",
		"satisfaction_reasons": "Satisfaction Rating Reasons",
		"ticket_audits":        "Ticket Audits",
	}
	objectNameToResponseField = map[string]string{ // nolint:gochecknoglobals
		"ticket_audits":        "audits",
		"search":               "results", // This is "/api/v2/search"
		"satisfaction_reasons": "reasons",
	}
)

func main() {
	explorer, err := openapi.FileManager.GetExplorer(
		api3.WithDisplayNamePostProcessors(
			api3.CamelCaseToSpaceSeparated,
			api3.CapitalizeFirstLetterEveryWord,
		),
	)
	must(err)

	objects, err := explorer.GetBasicReadObjects(
		ignoreEndpoints, nil, displayNameOverride, IsResponseFieldAppropriate,
	)
	must(err)

	schemas := scrapper.NewObjectMetadataResult()

	for _, object := range objects {
		if object.Problem != nil {
			slog.Error("schema not extracted",
				"objectName", object.ObjectName,
				"error", object.Problem,
			)
		}

		for _, field := range object.Fields {
			schemas.Add(object.ObjectName, object.DisplayName, field, nil)
		}
	}

	must(metadata.FileManager.SaveSchemas(schemas))

	slog.Info("Completed.")
}

func IsResponseFieldAppropriate(objectName, fieldName string) bool {
	if responseFieldName, ok := objectNameToResponseField[objectName]; ok {
		return fieldName == responseFieldName
	}

	return api3.IdenticalObjectCheck(objectName, fieldName)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
