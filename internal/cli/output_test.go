package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
)

func TestReportErrorAsJSONCarriesTheCode(t *testing.T) {
	// A malformed chave fails before the app opens, as an agent would see it.
	calls, err := runWithoutApp(t, "--json", "nfe", "buscar", "-c", nfeTestCNPJ, "--chave", "123")
	if err == nil || calls != 0 {
		t.Fatalf("err = %v, AppFactory calls = %d; want an error before the app", err, calls)
	}

	var out bytes.Buffer
	reportError(&out, err, true)
	var envelope errorEnvelope
	if jsonErr := json.Unmarshal(out.Bytes(), &envelope); jsonErr != nil {
		t.Fatalf("stderr is not one JSON value: %v\n%s", jsonErr, out.String())
	}
	if envelope.Error.Code != "invalid_input" || envelope.Error.Message != err.Error() {
		t.Errorf("envelope = %+v, want invalid_input with the message", envelope)
	}
}

func TestReportErrorAsText(t *testing.T) {
	var out bytes.Buffer
	reportError(&out, errors.New("empresa não encontrada"), false)
	if got := out.String(); got != "empresa não encontrada\n" {
		t.Errorf("text = %q", got)
	}
}

// internalJSONKeys are fields --json must never print: hashes, row IDs and
// paths of the local store.
var internalJSONKeys = []string{"RawHash", "ResumoRawHash", "RelationID", "DocumentID", "CompanyID", "XMLPath", "CertPath", "CredentialCertPath"}

// decodeJSONOutput decodes the whole of stdout as one JSON value into v.
// json.Unmarshal rejects anything after the value, so a progress line or a
// second value fails the test.
func decodeJSONOutput(t *testing.T, out *bytes.Buffer, v any) {
	t.Helper()
	if err := json.Unmarshal(out.Bytes(), v); err != nil {
		t.Fatalf("stdout is not one JSON value: %v\n%s", err, out.String())
	}
}

// assertNoJSONKeys fails when an object anywhere in v has one of keys.
func assertNoJSONKeys(t *testing.T, v any, keys ...string) {
	t.Helper()
	switch value := v.(type) {
	case map[string]any:
		for _, key := range keys {
			if _, ok := value[key]; ok {
				t.Errorf("JSON output has the internal field %q", key)
			}
		}
		for _, child := range value {
			assertNoJSONKeys(t, child, keys...)
		}
	case []any:
		for _, child := range value {
			assertNoJSONKeys(t, child, keys...)
		}
	}
}
