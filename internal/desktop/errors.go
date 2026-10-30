package main

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/desktop/desktopapi"
)

// formatError builds the payload every error a bound method returns reaches
// the frontend with. platform/wails/client/base.ts reads Code into
// WailsClientError.code. A blocked source error already names the time the
// source may be queried again. context.Canceled means the call ended before
// the core did: the frontend cancelled it, the window closed or the app shut
// down (callContext).
func formatError(err error) desktopapi.ErrorPayload {
	payload := desktopapi.ErrorPayload{Message: err.Error()}
	switch {
	case errors.Is(err, app.ErrOperationCanceled), errors.Is(err, context.Canceled):
		payload.Code = "canceled"
	case errors.Is(err, app.ErrSourceBlocked):
		payload.Code = "sefaz_blocked"
	case errors.Is(err, app.ErrSyncRunning), errors.Is(err, errUpdateDuringSync):
		payload.Code = "sync_running"
	case errors.Is(err, errUpdatesDisabled):
		payload.Code = "update_unavailable"
	case errors.Is(err, errNotStarted):
		payload.Code = "startup_failed"
	}
	return payload
}

// marshalError is the Wails Options.MarshalError hook. Wails puts the JSON it
// returns in the cause field of the rejected call, next to message and kind.
func marshalError(err error) []byte {
	data, jsonErr := json.Marshal(formatError(err))
	if jsonErr != nil {
		return nil
	}
	return data
}
