package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/vasfvitor/nanci/internal/app"
)

// jsonOutput reports whether the command runs with --json: one JSON value on
// stdout instead of text, no progress lines, and errors as a JSON envelope on
// stderr. It is the mode for scripts and AI agents.
func (e CommandEnv) jsonOutput() bool {
	return e.JSON != nil && *e.JSON
}

// printJSON writes v as indented JSON, one value per command run.
func printJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("escrever JSON: %w", err)
	}
	return nil
}

// errorEnvelope is how an error reaches stderr with --json: the message and
// the code of app.ErrorCode ("" when the error has none).
type errorEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// reportError writes err to w: as a JSON envelope with --json, or as the
// message line otherwise.
func reportError(w io.Writer, err error, asJSON bool) {
	if !asJSON {
		_, _ = fmt.Fprintln(w, err)
		return
	}
	var envelope errorEnvelope
	envelope.Error.Code = app.ErrorCode(err)
	envelope.Error.Message = err.Error()
	if jsonErr := printJSON(w, envelope); jsonErr != nil {
		_, _ = fmt.Fprintln(w, err)
	}
}
