// NFS-e methods of App: sync from the ADN, sync state and reset, direct
// query, document list and events, viewed marks and the DANFSe, XML and
// document exports.

package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/desktop/desktopapi"
	nsync "github.com/vasfvitor/nanci/internal/sync"
	"github.com/vasfvitor/nanci/internal/syncstate"
)

func (a *App) Pull(ctx context.Context, input desktopapi.PullInput) (desktopapi.PullResult, error) {
	if err := a.ready(); err != nil {
		return desktopapi.PullResult{}, err
	}
	ctx, cancel := a.callContext(ctx)
	defer cancel()
	done, err := a.trackPull()
	if err != nil {
		return desktopapi.PullResult{}, err
	}
	defer done()
	res, err := a.svc.sync.Pull(ctx, nsync.PullInput{
		CNPJ:   input.CNPJ,
		Mode:   input.Mode,
		Source: syncstate.SyncSourceNFSe,
	})
	if err != nil {
		return desktopapi.PullResult{}, err
	}
	return desktopapi.PullResult{
		CompanyName:              res.CompanyName,
		CNPJ:                     res.CNPJ,
		CredentialLabel:          res.CredentialLabel,
		CredentialCNPJ:           res.CredentialCNPJ,
		ConsultationBasis:        res.ConsultationBasis,
		Status:                   res.Status,
		StopReason:               res.StopReason,
		LastProcessedNSU:         res.LastProcessedNSU,
		LastFoundNSU:             res.LastFoundNSU,
		EmptyStreak:              res.EmptyStreak,
		DocumentsFound:           res.DocumentsFound,
		EventsFound:              res.EventsFound,
		DocumentsSaved:           res.DocumentsSaved,
		EventsSaved:              res.EventsSaved,
		DocumentsSkippedByPolicy: res.DocumentsSkippedByPolicy,
		EventsSkippedByPolicy:    res.EventsSkippedByPolicy,
		Errors:                   res.Errors,
		Duration:                 res.Duration,
	}, nil
}

func (a *App) ResetSyncState(input desktopapi.ResetSyncInput) error {
	if err := a.ready(); err != nil {
		return err
	}
	done, err := a.track()
	if err != nil {
		return err
	}
	defer done()
	return a.svc.sync.ResetSyncState(a.ctx, nsync.ResetSyncInput{
		CNPJ:   input.CompanyCNPJ,
		Source: syncstate.SyncSourceNFSe,
	})
}

func (a *App) QueryNFSeEvents(ctx context.Context, input desktopapi.QueryNFSeInput) (string, error) {
	if err := a.ready(); err != nil {
		return "", err
	}
	ctx, cancel := a.callContext(ctx)
	defer cancel()
	done, err := a.track()
	if err != nil {
		return "", err
	}
	defer done()
	return a.svc.query.QueryNFSeEvents(ctx, app.QueryNFSeInput{
		CNPJ:        input.CompanyCNPJ,
		ChaveAcesso: input.ChaveAcesso,
	})
}

func (a *App) ListDocuments(input desktopapi.ListInput) ([]desktopapi.DocumentRow, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	documents, err := a.svc.documents.ListDocuments(a.ctx, app.ListInput{
		CNPJ:       input.CNPJ,
		Competence: input.Competence,
		Direction:  input.Direction,
		OnlyUnread: input.OnlyUnread,
	})
	if err != nil {
		return nil, err
	}
	return desktopapi.DocumentRows(documents), nil
}

func (a *App) ListEventsForDocument(documentID string) ([]desktopapi.DocumentEvent, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	events, err := a.svc.documents.ListEventsForDocument(a.ctx, documentID)
	if err != nil {
		return nil, err
	}
	return desktopapi.DocumentEvents(events), nil
}

func (a *App) Status(cnpj string) (desktopapi.StatusResult, error) {
	if err := a.ready(); err != nil {
		return desktopapi.StatusResult{}, err
	}
	res, err := a.svc.sync.Status(a.ctx, cnpj)
	if err != nil {
		return desktopapi.StatusResult{}, err
	}
	return desktopapi.StatusResult{
		CompanyName:        res.CompanyName,
		CNPJ:               res.CNPJ,
		Environment:        res.Environment,
		ConsultationCNPJ:   res.ConsultationCNPJ,
		CredentialCNPJ:     res.CredentialCNPJ,
		CredentialNotAfter: res.CredentialNotAfter,
		LastProcessedNSU:   res.LastProcessedNSU,
		LastFoundNSU:       res.LastFoundNSU,
		LastSyncAt:         res.LastSyncAt,
		LastRunStatus:      res.LastRunStatus,
		LastRunStopReason:  res.LastRunStopReason,
		TotalEmitidas:      res.TotalEmitidas,
		TotalTomadas:       res.TotalTomadas,
	}, nil
}

func (a *App) ExportDANFSe(ctx context.Context, input desktopapi.ExportDANFSeInput) (desktopapi.ExportResult, error) {
	if err := a.ready(); err != nil {
		return desktopapi.ExportResult{}, err
	}
	ctx, cancel := a.callContext(ctx)
	defer cancel()
	done, err := a.track()
	if err != nil {
		return desktopapi.ExportResult{}, err
	}
	defer done()
	if input.OutPath == "" {
		return desktopapi.ExportResult{}, fmt.Errorf("caminho de saída não especificado")
	}

	err = a.svc.exports.ExportDANFSe(ctx, app.ExportDANFSeInput{
		CNPJ:        input.CNPJ,
		ChaveAcesso: input.ChaveAcesso,
		OutPath:     input.OutPath,
	})
	if err != nil {
		return desktopapi.ExportResult{}, err
	}
	return desktopapi.ExportResult{OutPath: input.OutPath, Format: "danfse", ExportedCount: 1}, nil
}

func (a *App) ExportXML(ctx context.Context, input desktopapi.ExportXMLInput) (desktopapi.ExportResult, error) {
	if err := a.ready(); err != nil {
		return desktopapi.ExportResult{}, err
	}
	ctx, cancel := a.callContext(ctx)
	defer cancel()
	done, err := a.track()
	if err != nil {
		return desktopapi.ExportResult{}, err
	}
	defer done()
	if input.OutPath == "" {
		return desktopapi.ExportResult{}, fmt.Errorf("caminho de saída não especificado")
	}

	err = a.svc.exports.ExportXML(ctx, app.ExportXMLInput{
		CNPJ:        input.CNPJ,
		ChaveAcesso: input.ChaveAcesso,
		OutPath:     input.OutPath,
	})
	if err != nil {
		return desktopapi.ExportResult{}, err
	}
	return desktopapi.ExportResult{OutPath: input.OutPath, Format: "xml", ExportedCount: 1}, nil
}

func (a *App) ExportDocuments(ctx context.Context, input desktopapi.ExportDocumentsInput) (desktopapi.ExportResult, error) {
	if err := a.ready(); err != nil {
		return desktopapi.ExportResult{}, err
	}
	ctx, cancel := a.callContext(ctx)
	defer cancel()
	done, err := a.track()
	if err != nil {
		return desktopapi.ExportResult{}, err
	}
	defer done()
	format := strings.ToLower(strings.TrimSpace(input.Format))
	if input.OutPath == "" {
		return desktopapi.ExportResult{}, fmt.Errorf("caminho de saída não especificado")
	}

	exportInput := app.ExportInput{
		CNPJ:         input.CNPJ,
		Competence:   input.Competence,
		Direction:    input.Direction,
		OutPath:      input.OutPath,
		Incremental:  input.Incremental,
		ChavesAcesso: input.ChavesAcesso,
	}

	var res app.ExportResult
	switch format {
	case "csv":
		res, err = a.svc.exports.ExportCSV(ctx, exportInput)
	case "xlsx":
		res, err = a.svc.exports.ExportXLSX(ctx, exportInput)
	case "zip":
		res, err = a.svc.exports.ExportZIP(ctx, exportInput)
	case "danfse-zip":
		res, err = a.svc.exports.ExportDANFSeZIP(ctx, exportInput)
	default:
		return desktopapi.ExportResult{}, fmt.Errorf("formato de exportação desconhecido: %s", format)
	}

	if err != nil {
		return desktopapi.ExportResult{}, err
	}

	return desktopapi.ExportResult(res), nil
}

// MarkDocumentsViewed marks the company's NFS-e with the given chaves as
// viewed and returns how many were new.
func (a *App) MarkDocumentsViewed(input desktopapi.MarkViewedInput) (int, error) {
	if err := a.ready(); err != nil {
		return 0, err
	}
	return a.svc.documents.MarkDocumentsViewed(a.ctx, input.CNPJ, input.ChavesAcesso)
}
