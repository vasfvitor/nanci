package main

import (
	"errors"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/desktop/desktopapi"
)

// formatError is the Wails ErrorFormatter: every error a bound method returns
// reaches the frontend as the payload built here. platform/wails/client.ts
// reads Code into WailsClientError.code. A blocked source error already names
// the time the source may be queried again.
func formatError(err error) any {
	payload := desktopapi.ErrorPayload{Message: err.Error()}
	switch {
	case errors.Is(err, app.ErrOperationCanceled):
		payload.Code = "canceled"
	case errors.Is(err, app.ErrSourceBlocked):
		payload.Code = "sefaz_blocked"
	case errors.Is(err, app.ErrSyncRunning):
		payload.Code = "sync_running"
	}
	return payload
}
