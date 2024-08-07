package testroutines

import (
	"errors"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/go-test/deep"
)

// suiteOutline describes major components that are used to test any Connector methods.
// It is universal and generic `V` value represents the data type of the expected output.
type suiteOutline[V any] struct {
	// Name of the test suite.
	Name string
	// Mock Server which connector will call.
	Server *httptest.Server
	// Custom Comparator of how expected output agrees with actual output.
	Comparator func(serverURL string, actual, expected *V) bool
	// Expected return value.
	Expected *V
	// ExpectedErrs is a list of errors that must be present in error output.
	ExpectedErrs []error
}

// Validate checks if supplied input conforms to the test intention.
func (o suiteOutline[V]) Validate(t *testing.T, err error, output *V) {
	defer o.Server.Close()

	// performs validation of error output using described test suite outline.
	o.checkError(t, err)
	// performs validation of data output using described test suite outline.
	o.checkValue(t, output)
}

func (o suiteOutline[V]) checkError(t *testing.T, err error) {
	if err != nil {
		if len(o.ExpectedErrs) == 0 {
			t.Fatalf("%s: expected no errors, got: (%v)", o.Name, err)
		}
	} else {
		// check that missing error is what is expected
		if len(o.ExpectedErrs) != 0 {
			t.Fatalf("%s: expected errors (%v), but got nothing", o.Name, o.ExpectedErrs)
		}
	}

	// check every error
	for _, expectedErr := range o.ExpectedErrs {
		if !errors.Is(err, expectedErr) && !strings.Contains(err.Error(), expectedErr.Error()) {
			t.Fatalf("%s: expected Error: (%v), got: (%v)", o.Name, expectedErr, err)
		}
	}
}

func (o suiteOutline[V]) checkValue(t *testing.T, output *V) {
	// compare desired output
	var ok bool
	if o.Comparator == nil {
		// default comparison is concerned about all fields
		ok = reflect.DeepEqual(output, o.Expected)
	} else {
		ok = o.Comparator(o.Server.URL, output, o.Expected)
	}

	if !ok {
		diff := deep.Equal(output, o.Expected)
		t.Fatalf("%s:, \nexpected: (%v), \ngot: (%v), \ndiff: (%v)", o.Name, o.Expected, output, diff)
	}
}
