package main

import (
	"encoding/json"
	"errors"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/desktop/desktopapi"
)

// formatError builds the payload every error a bound method returns reaches
// the frontend with. platform/wails/client/base.ts reads Code into
// WailsClientError.code. A blocked source error already names the time the
// source may be queried again. context.Canceled means the call ended before
// the core did: the frontend cancelled it (CancelSync for a sync), the window
// closed or the app shut down (callContext).
func formatError(err error) desktopapi.ErrorPayload {
	payload := desktopapi.ErrorPayload{Message: err.Error()}
	switch {
	case errors.Is(err, errUpdateDuringSync), errors.Is(err, errUpdateBusy):
		payload.Code = "update_busy"
	case errors.Is(err, errNotStarted):
		payload.Code = "startup_failed"
	default:
		// The codes of the core are the ones the CLI's --json gives too.
		payload.Code = app.ErrorCode(err)
	}
	return payload
}

// marshalError is the MarshalError of the App service options. Wails puts the JSON it
// returns in the cause field of the rejected call, next to message and kind.
func marshalError(err error) []byte {
	data, jsonErr := json.Marshal(formatError(err))
	if jsonErr != nil {
		return nil
	}
	return data
}
