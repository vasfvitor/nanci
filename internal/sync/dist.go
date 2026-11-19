package sync

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/vasfvitor/nanci/internal/sefaz"
	"github.com/vasfvitor/nanci/internal/syncstate"
)

// distWaitAfterStop is how long SEFAZ wants us to wait after catching up
// (cStat 137, or ultNSU = maxNSU) and after cStat 656, in the NF-e and CT-e
// distributions.
const distWaitAfterStop = time.Hour

// DistIdleWarningDays is how many days an NF-e or CT-e distribution may go
// without a distNSU before the status warns. The Ambiente Nacional only
// generates NSUs for CNPJ roots that called distNSU in the last 60 days
// (NT 2014.002 v1.10, known only from third-party reports), and the
// documents of a longer pause never reach the queue; 45 days leaves two
// weeks to run a pull.
const DistIdleWarningDays = 45

// DistIdle returns how many whole days have passed between lastQueryAt, the
// last time the SEFAZ answered a distNSU, and now, when that reached
// DistIdleWarningDays; it returns 0 otherwise, and for a distribution never
// queried (lastQueryAt nil).
func DistIdle(lastQueryAt *time.Time, now time.Time) int {
	if lastQueryAt == nil {
		return 0
	}
	days := int(now.Sub(*lastQueryAt) / (24 * time.Hour))
	if days < DistIdleWarningDays {
		return 0
	}
	return days
}

// requestsPerHour is the hourly request budget per CNPJ of source; 0 means
// unlimited.
func requestsPerHour(source syncstate.SyncSource) int {
	switch source {
	case syncstate.SyncSourceNFe:
		// The SEFAZ limit; going over it gets cStat 656 and an hour of
		// blocking. The official cap covers consNSU+consChNFe; counting
		// distNSU here is the app's conservative choice.
		return 20
	case syncstate.SyncSourceCTe:
		// The CT-e technical note does not publish a limit; this follows the
		// NF-e one.
		return 20
	default:
		return 0
	}
}

// distBatch turns a NFeDistribuicaoDFe or CTeDistribuicaoDFe answer to a
// query at cursor into the loop's batch and stop rules:
//
//   - 138: items; ultNSU >= maxNSU means caught up.
//   - 137: nothing new; caught up.
//   - 656: consumo indevido; stop and wait.
//
// Caught up and 656 both ask the loop to wait an hour before the next query.
// The next cursor is ultNSU, but never behind cursor. isEvent tells which
// schemas are events; label names the distribution in the log.
//
// The NSUs the cursor moves past without a document are the batch's gaps:
// the Ambiente Nacional generates NSUs without holes, and the NT tells the
// client to ask for each missing one with consNSU (NT 2014.002 v1.40, §3.6).
// A 138 page with holes and a 656 whose ultNSU is ahead of the cursor
// (another program consumed the CNPJ's sequence) have them. A walk from
// cursor 0 has none: the distribution starts it at the oldest NSU still in
// the window, and the ones before it are gone. A 137 has none either, since
// SEFAZ found nothing to deliver.
func distBatch(ctx context.Context, log *slog.Logger, resp sefaz.DistResult, cursor int64, isEvent func(schema string) bool, label string) (Batch, error) {
	batch := Batch{
		UltNSU:     resp.UltNSU,
		MaxNSU:     resp.MaxNSU,
		NextCursor: max(cursor, resp.UltNSU),
	}
	waitUntil := time.Now().UTC().Add(distWaitAfterStop)

	switch resp.CStat {
	case sefaz.CStatDocumentoLocalizado:
		batch.Items = distItems(resp, isEvent)
		if cursor > 0 {
			delivered := make([]int64, 0, len(resp.Docs))
			for _, doc := range resp.Docs {
				delivered = append(delivered, doc.NSU)
			}
			batch.Gaps = nsuGaps(cursor, batch.NextCursor, delivered, NSUGapBatchHole)
		}
		if resp.UltNSU >= resp.MaxNSU {
			batch.Done = true
			batch.StopReason = syncstate.SyncStopReasonCaughtUp
			batch.WaitUntil = &waitUntil
		}
	case sefaz.CStatNenhumDocumento:
		batch.Done = true
		batch.StopReason = syncstate.SyncStopReasonCaughtUp
		batch.WaitUntil = &waitUntil
	case sefaz.CStatConsumoIndevido:
		batch.Done = true
		batch.StopReason = syncstate.SyncStopReasonConsumoIndevido
		batch.WaitUntil = &waitUntil
		if cursor > 0 {
			batch.Gaps = nsuGaps(cursor, batch.NextCursor, nil, NSUGapConsumoIndevido)
		}
		log.WarnContext(ctx, "SEFAZ bloqueou a consulta de "+label+" por consumo indevido",
			slog.Int64("ult_nsu", resp.UltNSU),
			slog.Time("next_allowed_at", waitUntil))
	default:
		return Batch{}, &sefaz.RejectionError{CStat: resp.CStat, XMotivo: resp.XMotivo}
	}
	return batch, nil
}

// distItems turns the docZips of a distribution answer into loop items.
func distItems(resp sefaz.DistResult, isEvent func(schema string) bool) []Item {
	items := make([]Item, 0, len(resp.Docs))
	for _, doc := range resp.Docs {
		items = append(items, Item{
			NSU:     doc.NSU,
			Schema:  doc.Schema,
			Payload: doc.Content,
			IsEvent: isEvent(doc.Schema),
		})
	}
	return items
}

// CheckTpAmb returns the tpAmb to store for an NF-e or CT-e item of a pull
// that queried pullTpAmb, or of an XML stored outside the distribution, where
// pullTpAmb is the company's environment. It is the only place that decides
// it: the parsers only read it from the XML. The XML wins, with a warning when it names
// the other environment. A resumo carries no tpAmb and takes the pull's, and
// so does an XML with a missing or invalid one.
func CheckTpAmb(xmlTpAmb, pullTpAmb string, warnings *[]string) string {
	switch xmlTpAmb {
	case "":
		return pullTpAmb
	case pullTpAmb:
		return xmlTpAmb
	case sefaz.TpAmbProducao, sefaz.TpAmbHomologacao:
		*warnings = append(*warnings, fmt.Sprintf("tpAmb %s differs from the queried tpAmb %s; kept the XML's", xmlTpAmb, pullTpAmb))
		return xmlTpAmb
	default:
		*warnings = append(*warnings, fmt.Sprintf("invalid tpAmb %q; using the queried tpAmb %s", xmlTpAmb, pullTpAmb))
		return pullTpAmb
	}
}

// nsuGaps lists the NSUs in (cursor, next] that are not in delivered, as
// ranges with cause. The distribution answers a distNSU at cursor with the
// documents after it and the new ultNSU (next); every NSU in between that
// came without a document never reaches nanci by distNSU again.
func nsuGaps(cursor, next int64, delivered []int64, cause NSUGapCause) []NSURange {
	if next <= cursor {
		return nil
	}
	inRange := make([]int64, 0, len(delivered))
	for _, nsu := range delivered {
		if nsu > cursor && nsu <= next {
			inRange = append(inRange, nsu)
		}
	}
	slices.Sort(inRange)
	inRange = slices.Compact(inRange)

	var gaps []NSURange
	from := cursor + 1
	for _, nsu := range inRange {
		if nsu > from {
			gaps = append(gaps, NSURange{First: from, Last: nsu - 1, Cause: cause})
		}
		from = nsu + 1
	}
	if from <= next {
		gaps = append(gaps, NSURange{First: from, Last: next, Cause: cause})
	}
	return gaps
}
