package sync

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/vasfvitor/nanci/internal/dfe"
	"github.com/vasfvitor/nanci/internal/syncstate"
)

// sourceLockTTL is how long a source lease lasts without renewal. The holder
// renews it every sourceLockTTL/3, so a crashed process frees the source
// within one TTL. Tests shorten it.
var sourceLockTTL = 60 * time.Second

// lockTimeFormat has a fixed width, so the lease times compare as text.
const lockTimeFormat = "2006-01-02T15:04:05.000Z"

// sourceLock is a lease row of source_locks.
type sourceLock struct {
	owner  string
	holder string
}

// newLockOwner returns a random token naming one reservation.
func newLockOwner() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

// AcquireSourceLock takes the lease of the company's source for owner,
// unless another owner holds a lease that has not expired. It returns whether
// it took it and, when not, the holder of the live lease.
func (r *Store) AcquireSourceLock(ctx context.Context, companyID dfe.CompanyID, source syncstate.SyncSource, owner, holder string) (bool, string, error) {
	now := time.Now().UTC()
	nowText := now.Format(lockTimeFormat)
	expires := now.Add(sourceLockTTL).Format(lockTimeFormat)
	if _, err := r.db.ExecContext(ctx, `
		INSERT INTO source_locks (company_id, source, owner, holder, pid, acquired_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (company_id, source) DO UPDATE SET
			owner = excluded.owner,
			holder = excluded.holder,
			pid = excluded.pid,
			acquired_at = excluded.acquired_at,
			expires_at = excluded.expires_at
		WHERE source_locks.expires_at < ?
	`, string(companyID), string(source), owner, holder, os.Getpid(), nowText, expires, nowText); err != nil {
		return false, "", fmt.Errorf("reservar origem: %w", err)
	}
	var current sourceLock
	err := r.db.QueryRowContext(ctx, `
		SELECT owner, holder FROM source_locks WHERE company_id = ? AND source = ?
	`, string(companyID), string(source)).Scan(&current.owner, &current.holder)
	if errors.Is(err, sql.ErrNoRows) {
		// Released between the two statements: try once more.
		return r.AcquireSourceLock(ctx, companyID, source, owner, holder)
	}
	if err != nil {
		return false, "", fmt.Errorf("ler reserva da origem: %w", err)
	}
	return current.owner == owner, current.holder, nil
}

// RenewSourceLock extends owner's lease of the company's source by
// sourceLockTTL. It reports false when owner no longer holds it.
func (r *Store) RenewSourceLock(ctx context.Context, companyID dfe.CompanyID, source syncstate.SyncSource, owner string) (bool, error) {
	res, err := r.db.ExecContext(ctx, `
		UPDATE source_locks SET expires_at = ? WHERE company_id = ? AND source = ? AND owner = ?
	`, time.Now().UTC().Add(sourceLockTTL).Format(lockTimeFormat), string(companyID), string(source), owner)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n == 1, err
}

// ReleaseSourceLock drops owner's lease of the company's source.
func (r *Store) ReleaseSourceLock(ctx context.Context, companyID dfe.CompanyID, source syncstate.SyncSource, owner string) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM source_locks WHERE company_id = ? AND source = ? AND owner = ?
	`, string(companyID), string(source), owner)
	return err
}

// holdSourceLock takes the lease and renews it until the returned stop func
// runs, which also releases it. When another process holds the source it
// returns ErrSyncRunning naming that process.
func (m *Manager) holdSourceLock(ctx context.Context, companyID dfe.CompanyID, source syncstate.SyncSource) (func(), error) {
	owner, err := newLockOwner()
	if err != nil {
		return nil, fmt.Errorf("reservar origem: %w", err)
	}
	holder := m.Holder
	if holder == "" {
		holder = "nanci"
	}
	acquired, current, err := m.SyncRepo.AcquireSourceLock(ctx, companyID, source, owner, holder)
	if err != nil {
		return nil, err
	}
	if !acquired {
		return nil, fmt.Errorf("%w (%s) em outro processo do nanci (%s)", ErrSyncRunning, sourceLabel(source), current)
	}

	// The lease outlives a cancel of ctx until the release func runs.
	leaseCtx := context.WithoutCancel(ctx)
	done := make(chan struct{})
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		ticker := time.NewTicker(sourceLockTTL / 3)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				renewCtx, cancel := context.WithTimeout(leaseCtx, 5*time.Second)
				held, err := m.SyncRepo.RenewSourceLock(renewCtx, companyID, source, owner)
				cancel()
				if err != nil || !held {
					m.Log.Warn("Não foi possível renovar a reserva da origem",
						"source", string(source), "held", held, "error", err)
				}
			}
		}
	}()

	return func() {
		close(done)
		<-stopped
		releaseCtx, cancel := context.WithTimeout(leaseCtx, 5*time.Second)
		defer cancel()
		if err := m.SyncRepo.ReleaseSourceLock(releaseCtx, companyID, source, owner); err != nil {
			m.Log.Warn("Não foi possível liberar a reserva da origem", "source", string(source), "error", err)
		}
	}, nil
}
