package cli

import (
	"encoding/json"
	"errors"
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

// progress prints a progress line such as "Gerando arquivo ZIP..." in text
// mode. With --json it prints nothing, so stdout holds only the result.
func (e CommandEnv) progress(w io.Writer, line string) {
	if e.jsonOutput() {
		return
	}
	_, _ = fmt.Fprintln(w, line)
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
// the code of app.ErrorCode ("" when the error has none). It is written on one
// line, like the JSON log records before it, so stderr is JSON Lines.
type errorEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// errorCode is app.ErrorCode, plus invalid_input for a usage error.
func errorCode(err error) string {
	var usage usageError
	if errors.As(err, &usage) {
		return app.ErrorCodeInvalidInput
	}
	return app.ErrorCode(err)
}

// reportError writes err to w: as a JSON envelope with --json, or as the
// message line otherwise.
func reportError(w io.Writer, err error, asJSON bool) {
	if !asJSON {
		_, _ = fmt.Fprintln(w, err)
		return
	}
	var envelope errorEnvelope
	envelope.Error.Code = errorCode(err)
	envelope.Error.Message = err.Error()
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if jsonErr := enc.Encode(envelope); jsonErr != nil {
		_, _ = fmt.Fprintln(w, err)
	}
}
