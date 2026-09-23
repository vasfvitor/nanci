package sync

import (
	"errors"
	"fmt"
	"time"

	"github.com/vasfvitor/nanci/internal/nfse"
)

// ErrSourceBlocked matches every *BlockedError through errors.Is.
var ErrSourceBlocked = errors.New("consulta bloqueada temporariamente")

// ErrSyncRunning means a pull of the same company and source is already
// running in this process.
var ErrSyncRunning = errors.New("sincronização já em andamento para esta empresa")

// BlockedError says the source must not be queried for the company before
// Until. Reason is the stop reason that set the wait.
type BlockedError struct {
	Source nfse.SyncSource
	Until  time.Time
	Reason nfse.SyncStopReason
}

func (e *BlockedError) Error() string {
	return fmt.Sprintf("consulta %s bloqueada até %s (%s)", sourceLabel(e.Source), e.Until.Local().Format("02/01/2006 15:04"), e.Reason)
}

// Is makes errors.Is(err, ErrSourceBlocked) true for a *BlockedError.
func (e *BlockedError) Is(target error) bool {
	return target == ErrSourceBlocked
}

// checkBlocked returns a *BlockedError when the source state forbids a query at now.
func checkBlocked(source nfse.SyncSource, state SourceState, now time.Time) error {
	if state.BlockedUntil == nil || !now.Before(*state.BlockedUntil) {
		return nil
	}
	return &BlockedError{Source: source, Until: *state.BlockedUntil, Reason: state.BlockedReason}
}

func sourceLabel(source nfse.SyncSource) string {
	switch source {
	case nfse.SyncSourceNFSe:
		return "NFS-e"
	case nfse.SyncSourceNFe:
		return "NF-e"
	default:
		return string(source)
	}
}
