package sync

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/vasfvitor/nanci/internal/dfe"
	"github.com/vasfvitor/nanci/internal/store"
	"github.com/vasfvitor/nanci/internal/syncstate"
)

// NSUGapCause says how the distribution moved the cursor past NSUs it did
// not deliver.
type NSUGapCause string

const (
	// NSUGapConsumoIndevido is the jump of a cStat 656: SEFAZ answers with
	// the ultNSU of the CNPJ's shared sequence, which is ahead of nanci's
	// cursor when another program consumed the NSUs in between.
	NSUGapConsumoIndevido NSUGapCause = "consumo_indevido"
	// NSUGapBatchHole is an NSU inside a page's range that came without a
	// document.
	NSUGapBatchHole NSUGapCause = "batch_hole"
)

// NSURange is the inclusive range of NSUs First..Last the distribution
// skipped, and why.
type NSURange struct {
	First int64
	Last  int64
	Cause NSUGapCause
}

// Count is how many NSUs the range holds.
func (r NSURange) Count() int64 {
	return r.Last - r.First + 1
}

// NSUGap is an open gap: Next is the first NSU recovery has not asked for
// yet.
type NSUGap struct {
	ID         int64
	First      int64
	Last       int64
	Next       int64
	Cause      NSUGapCause
	Recovered  int
	Empty      int
	DetectedAt time.Time
}

// Missing is how many NSUs of the gap recovery still has to ask for.
func (g NSUGap) Missing() int64 {
	return g.Last - g.Next + 1
}

// RecordNSUGaps stores the ranges as open gaps of the source in env. A range
// already recorded with the same first NSU keeps the wider end and opens
// again if it grew, so detecting the same jump twice (a crash before the
// checkpoint) records it once.
func (r *Store) RecordNSUGaps(ctx context.Context, companyID dfe.CompanyID, source syncstate.SyncSource, env dfe.Environment, gaps []NSURange) error {
	if len(gaps) == 0 {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	now := time.Now().UTC().Format(time.RFC3339)
	for _, gap := range gaps {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO sync_nsu_gaps (company_id, source, environment, first_nsu, last_nsu, next_nsu, cause, detected_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT (company_id, source, environment, first_nsu) DO UPDATE SET
				status = CASE WHEN excluded.last_nsu > last_nsu THEN 'open' ELSE status END,
				finished_at = CASE WHEN excluded.last_nsu > last_nsu THEN NULL ELSE finished_at END,
				last_nsu = MAX(last_nsu, excluded.last_nsu)
		`, string(companyID), string(source), string(env), gap.First, gap.Last, gap.First, string(gap.Cause), now); err != nil {
			return fmt.Errorf("record NSU gap %d-%d: %w", gap.First, gap.Last, err)
		}
	}
	return tx.Commit()
}

// OpenNSUGaps lists the source's open gaps in env, oldest NSU first.
func (r *Store) OpenNSUGaps(ctx context.Context, companyID dfe.CompanyID, source syncstate.SyncSource, env dfe.Environment) ([]NSUGap, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, first_nsu, last_nsu, next_nsu, cause, recovered, empty, detected_at
		FROM sync_nsu_gaps
		WHERE company_id = ? AND source = ? AND environment = ? AND status = 'open'
		ORDER BY first_nsu
	`, string(companyID), string(source), string(env))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var gaps []NSUGap
	for rows.Next() {
		var gap NSUGap
		var cause, detectedAt string
		if err := rows.Scan(&gap.ID, &gap.First, &gap.Last, &gap.Next, &cause, &gap.Recovered, &gap.Empty, &detectedAt); err != nil {
			return nil, err
		}
		gap.Cause = NSUGapCause(cause)
		if parsed := store.ParseNullableTime(sql.NullString{String: detectedAt, Valid: true}); parsed != nil {
			gap.DetectedAt = *parsed
		}
		gaps = append(gaps, gap)
	}
	return gaps, rows.Err()
}

// MissingNSUs counts the NSUs of the source's open gaps in env detected at or
// after since; older gaps are past the distribution window and recovery
// closes them without asking.
func (r *Store) MissingNSUs(ctx context.Context, companyID dfe.CompanyID, source syncstate.SyncSource, env dfe.Environment, since time.Time) (int64, error) {
	var missing int64
	err := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(last_nsu - next_nsu + 1), 0)
		FROM sync_nsu_gaps
		WHERE company_id = ? AND source = ? AND environment = ? AND status = 'open' AND detected_at >= ?
	`, string(companyID), string(source), string(env), since.UTC().Format(time.RFC3339)).Scan(&missing)
	return missing, err
}

// advanceNSUGap moves the gap past nsu and counts the answer, closing the gap
// when nsu was its last. It runs on tx so a recovered document and the move
// commit together.
func advanceNSUGap(ctx context.Context, tx executor, gap NSUGap, nsu int64, recovered bool) error {
	recoveredDelta, emptyDelta := 0, 1
	if recovered {
		recoveredDelta, emptyDelta = 1, 0
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := tx.ExecContext(ctx, `
		UPDATE sync_nsu_gaps
		SET
			next_nsu = ? + 1,
			recovered = recovered + ?,
			empty = empty + ?,
			status = CASE WHEN ? >= last_nsu THEN 'done' ELSE status END,
			finished_at = CASE WHEN ? >= last_nsu THEN ? ELSE finished_at END
		WHERE id = ?
	`, nsu, recoveredDelta, emptyDelta, nsu, nsu, now, gap.ID)
	return err
}

// AdvanceNSUGap records that nsu of gap came back without a document.
func (r *Store) AdvanceNSUGap(ctx context.Context, gap NSUGap, nsu int64) error {
	return advanceNSUGap(ctx, r.db, gap, nsu, false)
}

// ApplyAndAdvanceNSUGap returns a CommitFunc that stores an item recovered at
// nsu and moves gap past it in the same transaction.
func (r *Store) ApplyAndAdvanceNSUGap(gap NSUGap, nsu int64) CommitFunc {
	return func(ctx context.Context, write func(tx *sql.Tx) (ItemOutcome, error)) (ItemOutcome, error) {
		return r.Apply(ctx, func(tx *sql.Tx) (ItemOutcome, error) {
			outcome, err := write(tx)
			if err != nil {
				return ItemOutcome{}, err
			}
			if err := advanceNSUGap(ctx, tx, gap, nsu, true); err != nil {
				return ItemOutcome{}, fmt.Errorf("advance NSU gap: %w", err)
			}
			return outcome, nil
		})
	}
}

// ExpireNSUGap closes a gap whose NSUs are past the distribution window.
func (r *Store) ExpireNSUGap(ctx context.Context, gap NSUGap) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE sync_nsu_gaps SET status = 'expired', finished_at = ? WHERE id = ?
	`, time.Now().UTC().Format(time.RFC3339), gap.ID)
	return err
}
