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
