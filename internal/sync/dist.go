package sync

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/vasfvitor/nanci/internal/sefaz"
)

// distWaitAfterStop is how long SEFAZ wants us to wait after catching up
// (cStat 137, or ultNSU = maxNSU) and after cStat 656, in the NF-e and CT-e
// distributions.
const distWaitAfterStop = time.Hour

// requestsPerHour is the hourly request budget per CNPJ of source; 0 means
// unlimited.
func requestsPerHour(source nfse.SyncSource) int {
	switch source {
	case nfse.SyncSourceNFe:
		// The SEFAZ limit; going over it gets cStat 656 and an hour of
		// blocking.
		return 20
	case nfse.SyncSourceCTe:
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
func distBatch(ctx context.Context, log *slog.Logger, resp sefaz.DistResult, cursor int64, isEvent func(schema string) bool, label string) (Batch, error) {
	batch := Batch{
		UltNSU:     resp.UltNSU,
		MaxNSU:     resp.MaxNSU,
		NextCursor: max(cursor, resp.UltNSU),
	}
	waitUntil := time.Now().UTC().Add(distWaitAfterStop)

	switch resp.CStat {
	case sefaz.CStatDocumentoLocalizado:
		batch.Items = make([]Item, 0, len(resp.Docs))
		for _, doc := range resp.Docs {
			batch.Items = append(batch.Items, Item{
				NSU:     doc.NSU,
				Schema:  doc.Schema,
				Payload: doc.Content,
				IsEvent: isEvent(doc.Schema),
			})
		}
		if resp.UltNSU >= resp.MaxNSU {
			batch.Done = true
			batch.StopReason = nfse.SyncStopReasonCaughtUp
			batch.WaitUntil = &waitUntil
		}
	case sefaz.CStatNenhumDocumento:
		batch.Done = true
		batch.StopReason = nfse.SyncStopReasonCaughtUp
		batch.WaitUntil = &waitUntil
	case sefaz.CStatConsumoIndevido:
		batch.Done = true
		batch.StopReason = nfse.SyncStopReasonConsumoIndevido
		batch.WaitUntil = &waitUntil
		log.WarnContext(ctx, "SEFAZ bloqueou a consulta de "+label+" por consumo indevido",
			slog.Int64("ult_nsu", resp.UltNSU),
			slog.Time("next_allowed_at", waitUntil))
	default:
		return Batch{}, &sefaz.RejectionError{CStat: resp.CStat, XMotivo: resp.XMotivo}
	}
	return batch, nil
}

// checkTpAmb returns the tpAmb to store for an NF-e or CT-e item of a pull
// that queried pullTpAmb. It is the only place that decides it: the parsers
// only read it from the XML. The XML wins, with a warning when it names
// the other environment. A resumo carries no tpAmb and takes the pull's, and
// so does an XML with a missing or invalid one.
func checkTpAmb(xmlTpAmb, pullTpAmb string, warnings *[]string) string {
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
