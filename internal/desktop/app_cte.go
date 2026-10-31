// CT-e methods of App: sync, XML import, status and reset, list and events,
// viewed marks, and the XML, DACTE and document exports.

package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/desktop/desktopapi"
	"github.com/vasfvitor/nanci/internal/syncstate"
)

func (a *App) PullCTe(ctx context.Context, input desktopapi.PullCTeInput) (desktopapi.PullCTeResult, error) {
	ctx, end, err := a.begin(ctx, true)
	if err != nil {
		return desktopapi.PullCTeResult{}, err
	}
	defer end()
	ctx, stop := a.guard.cancelable(ctx, syncKey{input.CNPJ, syncstate.SyncSourceCTe})
	defer stop()
	res, err := a.svc.cte.Pull(ctx, input.CNPJ)
	if err != nil {
		return desktopapi.PullCTeResult{}, err
	}
	return desktopapi.PullCTeResult{
		CompanyName:      res.CompanyName,
		CNPJ:             res.CNPJ,
		Status:           res.Status,
		StopReason:       res.StopReason,
		LastNSU:          res.LastNSU,
		MaxNSU:           res.MaxNSU,
		DocumentsSaved:   res.DocumentsSaved,
		EventsSaved:      res.EventsSaved,
		Errors:           res.Errors,
		NextAllowedAt:    res.NextAllowedAt,
		RequestsLastHour: res.RequestsLastHour,
		RequestBudget:    res.RequestBudget,
		Duration:         res.Duration,
	}, nil
}

// ImportCTeXML stores CT-e and CT-e event XML files the company holds,
// without contacting SEFAZ.
func (a *App) ImportCTeXML(ctx context.Context, input desktopapi.ImportXMLInput) (desktopapi.ImportXMLResult, error) {
	if err := validateImport(input); err != nil {
		return desktopapi.ImportXMLResult{}, err
	}
	ctx, end, err := a.begin(ctx, true)
	if err != nil {
		return desktopapi.ImportXMLResult{}, err
	}
	defer end()
	res, err := a.svc.cte.ImportXML(ctx, app.ImportXMLInput{CNPJ: input.CNPJ, Files: input.Files})
	if err != nil {
		return desktopapi.ImportXMLResult{}, err
	}
	return desktopapi.ImportXMLResultFrom(res), nil
}

func (a *App) StatusCTe(cnpj string) (desktopapi.CTeStatusResult, error) {
	if err := a.ready(); err != nil {
		return desktopapi.CTeStatusResult{}, err
	}
	res, err := a.svc.cte.Status(a.ctx, cnpj)
	if err != nil {
		return desktopapi.CTeStatusResult{}, err
	}
	return desktopapi.CTeStatusResult{
		CompanyName:       res.CompanyName,
		CNPJ:              res.CNPJ,
		UF:                res.UF,
		TpAmb:             res.TpAmb,
		LastNSU:           res.LastNSU,
		MaxNSU:            res.MaxNSU,
		LastSyncAt:        res.LastSyncAt,
		LastRunStatus:     res.LastRunStatus,
		LastRunStopReason: res.LastRunStopReason,
		LastRunError:      res.LastRunError,
		InitialSyncDoneAt: res.InitialSyncDoneAt,
		NextAllowedAt:     res.NextAllowedAt,
		BlockedReason:     res.BlockedReason,
		RequestsLastHour:  res.RequestsLastHour,
		RequestBudget:     res.RequestBudget,
		IdleDays:          res.IdleDays,
		TotalTomador:      res.TotalTomador,
		TotalDestinatario: res.TotalDestinatario,
		TotalRemetente:    res.TotalRemetente,
		TotalOutros:       res.TotalOutros,
	}, nil
}

func (a *App) ListCTe(input desktopapi.ListCTeInput) ([]desktopapi.CTeRow, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	documents, err := a.svc.cte.ListDocuments(a.ctx, app.ListCTeInput{
		CNPJ:         input.CNPJ,
		Competence:   input.Competence,
		Situacao:     input.Situacao,
		Role:         input.Role,
		Modelo:       input.Modelo,
		EmitenteCNPJ: input.EmitenteCNPJ,
		TomadorCNPJ:  input.TomadorCNPJ,
		NFeChave:     input.NFeChave,
		ChavesAcesso: input.ChavesAcesso,
		OnlyUnread:   input.OnlyUnread,
		Limit:        input.Limit,
	})
	if err != nil {
		return nil, err
	}
	return desktopapi.CTeRows(documents), nil
}

// MarkCTeViewed marks the company's CT-e with the given chaves as viewed and
// returns how many were new.
func (a *App) MarkCTeViewed(input desktopapi.MarkViewedInput) (int, error) {
	if err := a.ready(); err != nil {
		return 0, err
	}
	return a.svc.cte.MarkViewed(a.ctx, input.CNPJ, input.ChavesAcesso)
}

func (a *App) ListCTeEvents(input desktopapi.CTeKeyInput) ([]desktopapi.CTeEvent, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	events, err := a.svc.cte.ListEvents(a.ctx, input.CNPJ, input.ChaveAcesso)
	if err != nil {
		return nil, err
	}
	return desktopapi.CTeEvents(events), nil
}

func (a *App) ExportCTeXML(ctx context.Context, input desktopapi.ExportCTeXMLInput) (desktopapi.ExportResult, error) {
	if input.OutPath == "" {
		return desktopapi.ExportResult{}, fmt.Errorf("caminho de saída não especificado")
	}
	ctx, end, err := a.begin(ctx, false)
	if err != nil {
		return desktopapi.ExportResult{}, err
	}
	defer end()

	err = a.svc.cte.ExportXML(ctx, app.CTeExportXMLInput{
		CNPJ:        input.CNPJ,
		ChaveAcesso: input.ChaveAcesso,
		OutPath:     input.OutPath,
	})
	if err != nil {
		return desktopapi.ExportResult{}, err
	}
	return desktopapi.ExportResult{OutPath: input.OutPath, Format: "xml", ExportedCount: 1}, nil
}

func (a *App) ExportDACTE(ctx context.Context, input desktopapi.ExportDACTEInput) (desktopapi.ExportResult, error) {
	if input.OutPath == "" {
		return desktopapi.ExportResult{}, fmt.Errorf("caminho de saída não especificado")
	}
	ctx, end, err := a.begin(ctx, false)
	if err != nil {
		return desktopapi.ExportResult{}, err
	}
	defer end()

	err = a.svc.cte.ExportDACTE(ctx, app.CTeExportDACTEInput{
		CNPJ:        input.CNPJ,
		ChaveAcesso: input.ChaveAcesso,
		OutPath:     input.OutPath,
	})
	if err != nil {
		return desktopapi.ExportResult{}, err
	}
	return desktopapi.ExportResult{OutPath: input.OutPath, Format: "dacte", ExportedCount: 1}, nil
}

func (a *App) ExportCTeDocuments(ctx context.Context, input desktopapi.ExportCTeDocumentsInput) (desktopapi.CTeExportResult, error) {
	if input.OutPath == "" {
		return desktopapi.CTeExportResult{}, fmt.Errorf("caminho de saída não especificado")
	}
	ctx, end, err := a.begin(ctx, false)
	if err != nil {
		return desktopapi.CTeExportResult{}, err
	}
	defer end()
	format := strings.ToLower(strings.TrimSpace(input.Format))

	exportInput := app.CTeExportInput{
		CNPJ:         input.CNPJ,
		Competence:   input.Competence,
		Role:         input.Role,
		ChavesAcesso: input.ChavesAcesso,
		Incremental:  input.Incremental,
		OutPath:      input.OutPath,
	}

	var res app.CTeExportResult
	switch format {
	case "csv":
		res, err = a.svc.cte.ExportCSV(ctx, exportInput)
	case "xlsx":
		res, err = a.svc.cte.ExportXLSX(ctx, exportInput)
	case "zip":
		res, err = a.svc.cte.ExportXMLZip(ctx, exportInput)
	case "dacte-zip":
		res, err = a.svc.cte.ExportDACTEZip(ctx, exportInput)
	default:
		return desktopapi.CTeExportResult{}, fmt.Errorf("formato de exportação desconhecido: %s", format)
	}
	if err != nil {
		return desktopapi.CTeExportResult{}, err
	}
	return desktopapi.CTeExportResultFrom(res), nil
}

// PreviewResetCTe returns what ResetCTe would remove, changing nothing.
func (a *App) PreviewResetCTe(cnpj string) (desktopapi.CTeResetResult, error) {
	ctx, end, err := a.begin(a.ctx, false)
	if err != nil {
		return desktopapi.CTeResetResult{}, err
	}
	defer end()
	res, err := a.svc.cte.PreviewReset(ctx, cnpj)
	if err != nil {
		return desktopapi.CTeResetResult{}, err
	}
	return desktopapi.CTeResetResultFrom(res), nil
}

// ResetCTe removes the company's CT-e and resets its CT-e sync, so the next
// pull starts over from NSU 0.
func (a *App) ResetCTe(cnpj string) (desktopapi.CTeResetResult, error) {
	ctx, end, err := a.begin(a.ctx, false)
	if err != nil {
		return desktopapi.CTeResetResult{}, err
	}
	defer end()
	res, err := a.svc.cte.Reset(ctx, cnpj)
	if err != nil {
		return desktopapi.CTeResetResult{}, err
	}
	return desktopapi.CTeResetResultFrom(res), nil
}
