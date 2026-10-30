package main

import (
	"errors"
	"sync"
	"sync/atomic"
)

var (
	// errUpdateInstalling refuses an operation once an update started: the
	// restart into the new version would cut it short.
	errUpdateInstalling = errors.New("o Nanci está instalando uma atualização e vai reiniciar")
	// errUpdateDuringSync keeps the restart from cutting a sync short.
	errUpdateDuringSync = errors.New("aguarde a sincronização terminar para atualizar")
	// errUpdateBusy keeps the restart from cutting short an export, a query,
	// a reset or an event sent to SEFAZ.
	errUpdateBusy = errors.New("aguarde a operação em andamento terminar para atualizar")
)

// closeGuard tracks the long operations in flight. The syncs (pulls) keep
// the window open until the user confirms; every tracked operation holds
// off an update. Pull, PullNFe, PullCTe and the other tracked methods block
// until their work ends, so the counts are the operations in flight.
type closeGuard struct {
	pulls atomic.Int32 // syncs, XML imports and fetches by chave
	allow atomic.Bool

	mu         sync.Mutex
	ops        int  // every tracked operation, pulls included
	installing bool // an update is downloading or restarting
}

// start counts an operation until the returned func runs. It refuses once an
// update is installing. The check and the count happen under the same lock
// as beginInstall's, so an operation and an update can never both start.
func (g *closeGuard) start(pull bool) (done func(), err error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.installing {
		return nil, errUpdateInstalling
	}
	g.ops++
	if pull {
		g.pulls.Add(1)
	}
	return func() {
		g.mu.Lock()
		defer g.mu.Unlock()
		g.ops--
		if pull {
			g.pulls.Add(-1)
		}
	}, nil
}

// beginInstall marks an update as installing when no tracked operation
// runs; from then on start refuses. A running sync gets errUpdateDuringSync,
// which the frontend reports as a sync in progress.
func (g *closeGuard) beginInstall() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	switch {
	case g.installing:
		return errUpdateInstalling
	case g.pulls.Load() > 0:
		return errUpdateDuringSync
	case g.ops > 0:
		return errUpdateBusy
	}
	g.installing = true
	return nil
}

// endInstall lets operations start again after an update failed.
func (g *closeGuard) endInstall() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.installing = false
}

// track counts a long operation (an export, a query, a reset, an event sent
// to SEFAZ) until the returned func runs.
func (a *App) track() (done func(), err error) {
	return a.guard.start(false)
}

// trackPull counts a sync until the returned func runs. A sync also keeps the
// window open on close.
func (a *App) trackPull() (done func(), err error) {
	return a.guard.start(true)
}

// beforeClose runs for every close: the title-bar X, Ctrl+Q and the menu
// "Sair" reach it through Options.ShouldQuit, the OS close through the
// window's WindowClosing hook (main.go). Returning true keeps the window
// open. During a sync it asks the frontend to confirm through the
// app:before-close event, which carries the number of syncs in flight.
// ShouldQuit runs on the UI thread, so the event is sent from a goroutine
// and nothing here waits.
func (a *App) beforeClose() bool {
	if a.guard.allow.Load() {
		return false
	}
	pulls := a.guard.pulls.Load()
	if pulls == 0 {
		return false
	}
	go a.events.Emit(eventBeforeClose, int(pulls))
	return true
}

// ConfirmQuit closes the app after the user chose to leave during a sync.
func (a *App) ConfirmQuit() {
	a.guard.allow.Store(true)
	a.quit()
}
