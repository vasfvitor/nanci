package main

import (
	"errors"
	"fmt"

	"github.com/vasfvitor/nanci/internal/app"
)

// desktopError tags the errors the frontend branches on. The prefix is parsed
// by platform/wails/client.ts into WailsClientError.code; the rest of the
// message is the original error. A blocked source error already names the
// time the source may be queried again.
func desktopError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, app.ErrOperationCanceled):
		return fmt.Errorf("ERR_CANCELED: %w", err)
	case errors.Is(err, app.ErrSourceBlocked):
		return fmt.Errorf("ERR_SEFAZ_BLOCKED: %w", err)
	case errors.Is(err, app.ErrSyncRunning):
		return fmt.Errorf("ERR_SYNC_RUNNING: %w", err)
	default:
		return err
	}
}
