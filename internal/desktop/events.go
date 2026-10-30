package main

import (
	"github.com/wailsapp/wails/v3/pkg/application"

	nsync "github.com/vasfvitor/nanci/internal/sync"
)

// Names of the events the backend sends to the frontend.
const (
	eventBackendLog          = "backend-log"
	eventBeforeClose         = "app:before-close"
	eventRequestCertPassword = "request-cert-password" // #nosec G101 -- an event name, not a credential.
	eventNotifySuccess       = "notify-success"
	eventNotifyError         = "notify-error"
)

// init registers each event with its payload type. The binding generator
// reads these calls to type the payloads in the frontend, and Wails cancels
// an emit whose data is not exactly the registered type (a pointer does not
// match a value). The generator only sees calls with constant arguments made
// directly in init.
func init() {
	application.RegisterEvent[desktopLogEvent](eventBackendLog)
	application.RegisterEvent[int](eventBeforeClose) // number of syncs in flight
	application.RegisterEvent[nsync.CertPasswordRequest](eventRequestCertPassword)
	application.RegisterEvent[string](eventNotifySuccess)
	application.RegisterEvent[string](eventNotifyError)
}
