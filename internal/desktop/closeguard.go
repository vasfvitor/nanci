package main

import (
	"context"
	"errors"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vasfvitor/nanci/internal/syncstate"
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

// shutdownWait bounds how long ServiceShutdown waits for the tracked
// operations before it closes the database.
var shutdownWait = 10 * time.Second

// closeGuard tracks the long operations in flight. The syncs (pulls) keep
// the window open until the user confirms; every tracked operation holds
// off an update and the closing of the database at shutdown. Pull, PullNFe,
// PullCTe and the other tracked methods block until their work ends, so the
// counts are the operations in flight.
type closeGuard struct {
	allow atomic.Bool

	mu         sync.Mutex
	ops        int  // every tracked operation, pulls included
	pulls      int  // syncs, XML imports and fetches by chave
	installing bool // an update is downloading or restarting
	// cancels holds the cancel func of each Pull, PullNFe and PullCTe in
	// flight, for CancelSync. A second sync of the same pair is refused by
	// the core, but it registers before, so a key may hold two for a moment.
	cancels map[syncKey][]*context.CancelFunc
}

// syncKey names a sync by company and source, as stores/companySync.ts keys
// it in the frontend.
type syncKey struct {
	cnpj   string
	source syncstate.SyncSource
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
		g.pulls++
	}
	return func() {
		g.mu.Lock()
		defer g.mu.Unlock()
		g.ops--
		if pull {
			g.pulls--
		}
	}, nil
}

// beginInstall marks an update as installing when no tracked operation
// runs; from then on start refuses. A running sync gets errUpdateDuringSync,
// any other operation errUpdateBusy.
func (g *closeGuard) beginInstall() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	switch {
	case g.installing:
		return errUpdateInstalling
	case g.pulls > 0:
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

// waitIdle returns when no tracked operation runs or when limit passes.
func (g *closeGuard) waitIdle(limit time.Duration) {
	deadline := time.Now().Add(limit)
	for {
		g.mu.Lock()
		ops := g.ops
		g.mu.Unlock()
		if ops == 0 || time.Now().After(deadline) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// begin starts a long bound method: it checks that the core is ready,
// counts the method in the guard (pull for a sync, an XML import or a fetch
// by chave) and returns the context to pass to the core (callContext). end
// cancels that context and releases the guard; defer it. The methods without
// a context parameter pass a.ctx.
func (a *App) begin(ctx context.Context, pull bool) (_ context.Context, end func(), err error) {
	if err := a.ready(); err != nil {
		return nil, nil, err
	}
	done, err := a.guard.start(pull)
	if err != nil {
		return nil, nil, err
	}
	ctx, cancel := a.callContext(ctx)
	return ctx, func() {
		cancel()
		done()
	}, nil
}

// cancelable returns a context CancelSync can cancel by company and source.
// stop forgets it and cancels it; defer it.
func (g *closeGuard) cancelable(ctx context.Context, key syncKey) (_ context.Context, stop func()) {
	ctx, cancel := context.WithCancel(ctx)
	entry := &cancel
	g.mu.Lock()
	if g.cancels == nil {
		g.cancels = make(map[syncKey][]*context.CancelFunc)
	}
	g.cancels[key] = append(g.cancels[key], entry)
	g.mu.Unlock()
	return ctx, func() {
		g.mu.Lock()
		g.cancels[key] = slices.DeleteFunc(g.cancels[key], func(e *context.CancelFunc) bool { return e == entry })
		if len(g.cancels[key]) == 0 {
			delete(g.cancels, key)
		}
		g.mu.Unlock()
		cancel()
	}
}

// CancelSync stops the company's sync of source ("nfse", "nfe" or "cte"),
// if one runs. The Pull call itself returns, with the canceled code, only
// once the core has stopped and recorded the run, so the frontend keeps the
// source busy exactly as long as the backend does.
func (a *App) CancelSync(cnpj string, source string) {
	a.guard.mu.Lock()
	defer a.guard.mu.Unlock()
	for _, cancel := range a.guard.cancels[syncKey{cnpj, syncstate.SyncSource(source)}] {
		(*cancel)()
	}
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
	a.guard.mu.Lock()
	pulls := a.guard.pulls
	a.guard.mu.Unlock()
	if pulls == 0 {
		return false
	}
	go a.events.Emit(eventBeforeClose, pulls)
	return true
}

// ConfirmQuit closes the app after the user chose to leave during a sync.
func (a *App) ConfirmQuit() {
	a.guard.allow.Store(true)
	a.quit()
}
