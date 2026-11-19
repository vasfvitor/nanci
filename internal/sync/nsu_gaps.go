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

// NSUGapWindow is how long the distribution keeps a document: 90 days for
// the NF-e (NT 2014.002) and 3 months for the CT-e (NT 2015.002). A gap whose
// NSUs may date from longer ago holds nothing consNSU can still return.
const NSUGapWindow = 90 * 24 * time.Hour

// NSURange is the inclusive range of NSUs First..Last the distribution
// skipped, and why.
type NSURange struct {
	First int64
	Last  int64
	Cause NSUGapCause
}

// NSUGap is an open gap: Next is the first NSU recovery has not asked for
// yet. Since is the earliest time its NSUs may date from; the distribution
// window counts from it.
type NSUGap struct {
	ID    int64
	First int64
	Last  int64
	Next  int64
	Cause NSUGapCause
	Since time.Time
}

// RecordNSUGaps stores the ranges as open gaps of the source in env. since
// is the earliest time their NSUs may date from: when the distribution last
// answered before it skipped them. A range keeps only the NSUs no recorded
// gap holds yet, open or finished, so detecting the same jump twice (a crash
// before the checkpoint) or an overlapping one never asks for an NSU twice.
func (r *Store) RecordNSUGaps(ctx context.Context, companyID dfe.CompanyID, source syncstate.SyncSource, env dfe.Environment, since time.Time, gaps []NSURange) error {
	if len(gaps) == 0 {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	now := time.Now().UTC().Format(time.RFC3339)
	sinceText := since.UTC().Format(time.RFC3339)
	for _, gap := range gaps {
		held, err := heldNSURanges(ctx, tx, companyID, source, env, gap)
		if err != nil {
			return fmt.Errorf("read NSU gaps around %d-%d: %w", gap.First, gap.Last, err)
		}
		for _, part := range subtractRanges(gap, held) {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO sync_nsu_gaps (company_id, source, environment, first_nsu, last_nsu, next_nsu, cause, nsus_since, detected_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, string(companyID), string(source), string(env), part.First, part.Last, part.First, string(part.Cause), sinceText, now); err != nil {
				return fmt.Errorf("record NSU gap %d-%d: %w", part.First, part.Last, err)
			}
		}
	}
	return tx.Commit()
}

// heldNSURanges lists, oldest first, the recorded gaps that overlap gap.
func heldNSURanges(ctx context.Context, tx *sql.Tx, companyID dfe.CompanyID, source syncstate.SyncSource, env dfe.Environment, gap NSURange) ([]NSURange, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT first_nsu, last_nsu
		FROM sync_nsu_gaps
		WHERE company_id = ? AND source = ? AND environment = ? AND first_nsu <= ? AND last_nsu >= ?
		ORDER BY first_nsu
	`, string(companyID), string(source), string(env), gap.Last, gap.First)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var held []NSURange
	for rows.Next() {
		var r NSURange
		if err := rows.Scan(&r.First, &r.Last); err != nil {
			return nil, err
		}
		held = append(held, r)
	}
	return held, rows.Err()
}

// subtractRanges returns the parts of gap outside every range of held, which
// is sorted by First.
func subtractRanges(gap NSURange, held []NSURange) []NSURange {
	var parts []NSURange
	from := gap.First
	for _, h := range held {
		if h.First > from {
			parts = append(parts, NSURange{First: from, Last: min(h.First-1, gap.Last), Cause: gap.Cause})
		}
		from = max(from, h.Last+1)
		if from > gap.Last {
			return parts
		}
	}
	return append(parts, NSURange{First: from, Last: gap.Last, Cause: gap.Cause})
}

// OpenNSUGaps lists the source's open gaps in env, oldest NSU first.
func (r *Store) OpenNSUGaps(ctx context.Context, companyID dfe.CompanyID, source syncstate.SyncSource, env dfe.Environment) ([]NSUGap, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, first_nsu, last_nsu, next_nsu, cause, nsus_since
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
		var cause, since string
		if err := rows.Scan(&gap.ID, &gap.First, &gap.Last, &gap.Next, &cause, &since); err != nil {
			return nil, err
		}
		gap.Cause = NSUGapCause(cause)
		if parsed := store.ParseNullableTime(sql.NullString{String: since, Valid: true}); parsed != nil {
			gap.Since = *parsed
		}
		gaps = append(gaps, gap)
	}
	return gaps, rows.Err()
}

// windowStart is the oldest nsus_since still inside the distribution window.
func windowStart() string {
	return time.Now().Add(-NSUGapWindow).UTC().Format(time.RFC3339)
}

// MissingNSUs counts the NSUs recovery can still ask for: those of the
// source's open gaps in env inside the distribution window.
func (r *Store) MissingNSUs(ctx context.Context, companyID dfe.CompanyID, source syncstate.SyncSource, env dfe.Environment) (int64, error) {
	var missing int64
	err := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(last_nsu - next_nsu + 1), 0)
		FROM sync_nsu_gaps
		WHERE company_id = ? AND source = ? AND environment = ? AND status = 'open' AND nsus_since >= ?
	`, string(companyID), string(source), string(env), windowStart()).Scan(&missing)
	return missing, err
}

// ExpireNSUGaps closes the source's open gaps in env whose NSUs are past the
// distribution window and returns how many NSUs they still held.
func (r *Store) ExpireNSUGaps(ctx context.Context, companyID dfe.CompanyID, source syncstate.SyncSource, env dfe.Environment) (int64, error) {
	return r.closeNSUGaps(ctx, companyID, source, env, "expired", windowStart())
}

// DiscardNSUGaps closes every open gap of the source in env at the user's
// request, so recovery never asks for those NSUs, and returns how many NSUs
// they still held.
func (r *Store) DiscardNSUGaps(ctx context.Context, companyID dfe.CompanyID, source syncstate.SyncSource, env dfe.Environment) (int64, error) {
	// Every nsus_since is before the far future: all open gaps.
	return r.closeNSUGaps(ctx, companyID, source, env, "discarded", "9999-12-31T23:59:59Z")
}

// OpenNSUs counts the NSUs of every open gap of the source in env, inside the
// window or not: what DiscardNSUGaps would close.
func (r *Store) OpenNSUs(ctx context.Context, companyID dfe.CompanyID, source syncstate.SyncSource, env dfe.Environment) (int64, error) {
	var open int64
	err := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(last_nsu - next_nsu + 1), 0)
		FROM sync_nsu_gaps
		WHERE company_id = ? AND source = ? AND environment = ? AND status = 'open'
	`, string(companyID), string(source), string(env)).Scan(&open)
	return open, err
}

// closeNSUGaps sets status on the source's open gaps in env whose NSUs date
// from before before, and returns how many NSUs they still held.
func (r *Store) closeNSUGaps(ctx context.Context, companyID dfe.CompanyID, source syncstate.SyncSource, env dfe.Environment, status, before string) (int64, error) {
	rows, err := r.db.QueryContext(ctx, `
		UPDATE sync_nsu_gaps SET status = ?, finished_at = ?
		WHERE company_id = ? AND source = ? AND environment = ? AND status = 'open' AND nsus_since < ?
		RETURNING last_nsu - next_nsu + 1
	`, status, time.Now().UTC().Format(time.RFC3339), string(companyID), string(source), string(env), before)
	if err != nil {
		return 0, err
	}
	defer func() { _ = rows.Close() }()

	var expired int64
	for rows.Next() {
		var n int64
		if err := rows.Scan(&n); err != nil {
			return 0, err
		}
		expired += n
	}
	return expired, rows.Err()
}

// advanceNSUGap moves the gap past nsu and counts the answer as recovered (a
// document stored) or empty (nothing stored), closing the gap when nsu was
// its last. It runs on tx so a recovered document and the move commit
// together.
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

// AdvanceNSUGap records that nsu of gap came back without a document the
// recovery could store.
func (r *Store) AdvanceNSUGap(ctx context.Context, gap NSUGap, nsu int64) error {
	return advanceNSUGap(ctx, r.db, gap, nsu, false)
}

// ApplyAndAdvanceNSUGap returns a CommitFunc that stores an item recovered at
// nsu and moves gap past it in the same transaction. An item the source
// skipped by policy, such as an event of an NF-e the company does not have,
// counts as empty.
func (r *Store) ApplyAndAdvanceNSUGap(gap NSUGap, nsu int64) CommitFunc {
	return func(ctx context.Context, write func(tx *sql.Tx) (ItemOutcome, error)) (ItemOutcome, error) {
		return r.Apply(ctx, func(tx *sql.Tx) (ItemOutcome, error) {
			outcome, err := write(tx)
			if err != nil {
				return ItemOutcome{}, err
			}
			if err := advanceNSUGap(ctx, tx, gap, nsu, !outcome.SkippedByPolicy); err != nil {
				return ItemOutcome{}, fmt.Errorf("advance NSU gap: %w", err)
			}
			return outcome, nil
		})
	}
}
