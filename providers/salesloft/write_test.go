package salesloft

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amp-labs/connectors"
	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/interpreter"
	"github.com/amp-labs/connectors/test/utils/mockutils"
	"github.com/amp-labs/connectors/test/utils/mockutils/mockserver"
	"github.com/amp-labs/connectors/test/utils/testroutines"
	"github.com/amp-labs/connectors/test/utils/testutils"
)

func TestWrite(t *testing.T) { // nolint:funlen,cyclop
	t.Parallel()

	listSchema := testutils.DataFromFile(t, "write-signals-error.json")
	createAccountRes := testutils.DataFromFile(t, "write-create-account.json")
	createTaskRes := testutils.DataFromFile(t, "write-create-task.json")

	tests := []testroutines.Write{
		{
			Name:         "Write object must be included",
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingObjects},
		},
		{
			Name:         "Mime response header expected",
			Input:        common.WriteParams{ObjectName: "signals"},
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{interpreter.ErrMissingContentType},
		},
		{
			Name:  "Correct error message is understood from JSON response",
			Input: common.WriteParams{ObjectName: "signals", RecordId: "22165"},
			Server: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnprocessableEntity)
				_, _ = w.Write(listSchema)
			})),
			ExpectedErrs: []error{
				common.ErrBadRequest,
				errors.New("no Signal Registration found for integration id 5167 and given type"), // nolint:goerr113
			},
		},
		{
			Name:  "Write must act as a Create",
			Input: common.WriteParams{ObjectName: "signals"},
			Server: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				mockutils.RespondToMethod(w, r, "POST", func() {
					w.WriteHeader(http.StatusOK)
				})
			})),
			Expected:     &common.WriteResult{Success: true},
			ExpectedErrs: nil,
		},
		{
			Name:  "Write must act as an Update",
			Input: common.WriteParams{ObjectName: "signals", RecordId: "22165"},
			Server: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				mockutils.RespondToMethod(w, r, "PUT", func() {
					w.WriteHeader(http.StatusOK)
				})
			})),
			Expected:     &common.WriteResult{Success: true},
			ExpectedErrs: nil,
		},
		{
			Name:  "Valid creation of account",
			Input: common.WriteParams{ObjectName: "accounts"},
			Server: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				mockutils.RespondToMethod(w, r, "POST", func() {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write(createAccountRes)
				})
			})),
			Comparator: func(serverURL string, actual, expected *common.WriteResult) bool {
				return mockutils.WriteResultComparator.SubsetData(actual, expected)
			},
			Expected: &common.WriteResult{
				Success:  true,
				RecordId: "1",
				Errors:   nil,
				Data: map[string]any{
					"id":          1.0,
					"name":        "Hogwarts School of Witchcraft and Wizardry",
					"description": "British school of magic for students",
					"country":     "Scotland",
					"counts":      map[string]any{"people": 15.0},
				},
			},
			ExpectedErrs: nil,
		},
		{
			Name:  "Valid creation of a task",
			Input: common.WriteParams{ObjectName: "tasks"},
			Server: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				mockutils.RespondToMethod(w, r, "POST", func() {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write(createTaskRes)
				})
			})),
			Comparator: func(serverURL string, actual, expected *common.WriteResult) bool {
				return mockutils.WriteResultComparator.SubsetData(actual, expected)
			},
			Expected: &common.WriteResult{
				Success:  true,
				RecordId: "175204275",
				Errors:   nil,
				Data: map[string]any{
					"subject":       "call me maybe",
					"current_state": "scheduled",
					"task_type":     "call",
				},
			},
			ExpectedErrs: nil,
		},
		{
			Name:  "Valid update of Saved List View",
			Input: common.WriteParams{ObjectName: "saved_list_views", RecordId: "22463"},
			Server: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				mockutils.RespondToMethod(w, r, "PUT", func() {
					w.WriteHeader(http.StatusOK)
					mockutils.WriteBody(w, `{"data":{"id":22463,"view":"companies",
							"name":"Hierarchy overview","view_params":{},"is_default":false,"shared":false}}`)
				})
			})),
			Comparator: func(serverURL string, actual, expected *common.WriteResult) bool {
				return mockutils.WriteResultComparator.SubsetData(actual, expected)
			},
			Expected: &common.WriteResult{
				Success:  true,
				RecordId: "22463",
				Errors:   nil,
				Data: map[string]any{
					"id":   22463.0,
					"name": "Hierarchy overview",
					"view": "companies",
				},
			},
			ExpectedErrs: nil,
		},
	}

	for _, tt := range tests {
		// nolint:varnamelen
		tt := tt // rebind, omit loop side effects for parallel goroutine
		t.Run(tt.Name, func(t *testing.T) {
			t.Parallel()

			tt.Run(t, func() (connectors.WriteConnector, error) {
				return constructTestConnector(tt.Server.URL)
			})
		})
	}
}