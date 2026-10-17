// NF-e methods of App: sync, XML import, fetch by chave, status and reset,
// list and events, viewed marks, ciência and manifestação, and the XML,
// DANFE and document exports.

package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/desktop/desktopapi"
)

func (a *App) PullNFe(input desktopapi.PullNFeInput) (desktopapi.PullNFeResult, error) {
	if err := a.ready(); err != nil {
		return desktopapi.PullNFeResult{}, err
	}
	defer a.trackPull()()
	res, err := a.svc.nfe.Pull(a.ctx, input.CNPJ)
	if err != nil {
		return desktopapi.PullNFeResult{}, err
	}
	return desktopapi.PullNFeResult{
		CompanyName:      res.CompanyName,
		CNPJ:             res.CNPJ,
		Status:           res.Status,
		StopReason:       res.StopReason,
		LastNSU:          res.LastNSU,
		MaxNSU:           res.MaxNSU,
		CompletasSaved:   res.CompletasSaved,
		ResumosSaved:     res.ResumosSaved,
		EventsSaved:      res.EventsSaved,
		Errors:           res.Errors,
		NextAllowedAt:    res.NextAllowedAt,
		RequestsLastHour: res.RequestsLastHour,
		RequestBudget:    res.RequestBudget,
		Duration:         res.Duration,
	}, nil
}

// ImportNFeXML stores NF-e and NF-e event XML files the company holds,
// without contacting SEFAZ. It counts as a sync for the close guard, since
// it writes the company's documents.
func (a *App) ImportNFeXML(input desktopapi.ImportXMLInput) (desktopapi.ImportXMLResult, error) {
	if err := a.ready(); err != nil {
		return desktopapi.ImportXMLResult{}, err
	}
	if err := validateImport(input); err != nil {
		return desktopapi.ImportXMLResult{}, err
	}
	defer a.trackPull()()
	res, err := a.svc.nfe.ImportXML(a.ctx, app.ImportXMLInput{CNPJ: input.CNPJ, Files: input.Files})
	if err != nil {
		return desktopapi.ImportXMLResult{}, err
	}
	return desktopapi.ImportXMLResultFrom(res), nil
}

// FetchNFeByChave asks SEFAZ for one NF-e by its chave. It spends one
// request of the company's NF-e hourly budget.
func (a *App) FetchNFeByChave(input desktopapi.FetchByChaveInput) (desktopapi.FetchByChaveResult, error) {
	if err := a.ready(); err != nil {
		return desktopapi.FetchByChaveResult{}, err
	}
	if input.CNPJ == "" {
		return desktopapi.FetchByChaveResult{}, errors.New("empresa não especificada")
	}
	if input.ChaveAcesso == "" {
		return desktopapi.FetchByChaveResult{}, errors.New("chave de acesso não especificada")
	}
	defer a.trackPull()()
	res, err := a.svc.nfe.FetchByChave(a.ctx, app.FetchByChaveInput{CNPJ: input.CNPJ, ChaveAcesso: input.ChaveAcesso})
	if err != nil {
		return desktopapi.FetchByChaveResult{}, err
	}
	return desktopapi.FetchByChaveResultFrom(res), nil
}

func (a *App) StatusNFe(cnpj string) (desktopapi.NFeStatusResult, error) {
	if err := a.ready(); err != nil {
		return desktopapi.NFeStatusResult{}, err
	}
	res, err := a.svc.nfe.Status(a.ctx, cnpj)
	if err != nil {
		return desktopapi.NFeStatusResult{}, err
	}
	return desktopapi.NFeStatusResult{
		CompanyName:       res.CompanyName,
		CNPJ:              res.CNPJ,
		UF:                res.UF,
		TpAmb:             res.TpAmb,
		LastNSU:           res.LastNSU,
		MaxNSU:            res.MaxNSU,
		LastSyncAt:        res.LastSyncAt,
		LastRunStatus:     res.LastRunStatus,
		LastRunStopReason: res.LastRunStopReason,
		InitialSyncDoneAt: res.InitialSyncDoneAt,
		NextAllowedAt:     res.NextAllowedAt,
		BlockedReason:     res.BlockedReason,
		RequestsLastHour:  res.RequestsLastHour,
		RequestBudget:     res.RequestBudget,
		IdleDays:          res.IdleDays,
		TotalDestinatario: res.TotalDestinatario,
		TotalEmitente:     res.TotalEmitente,
		TotalOutros:       res.TotalOutros,
		TotalResumos:      res.TotalResumos,
		TotalCompletas:    res.TotalCompletas,
		PendingCiencia:    res.PendingCiencia,
		PendingConclusiva: res.PendingConclusiva,
		CienciaOverdue:    res.CienciaOverdue,
	}, nil
}

func (a *App) ListNFe(input desktopapi.ListNFeInput) ([]desktopapi.NFeRow, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	documents, err := a.svc.nfe.ListDocuments(a.ctx, app.NFeListInput{
		CNPJ:         input.CNPJ,
		Competence:   input.Competence,
		Situacao:     input.Situacao,
		Completeness: input.Completeness,
		Role:         input.Role,
		Manifestacao: input.Manifestacao,
		EmitenteCNPJ: input.EmitenteCNPJ,
		ChavesAcesso: input.ChavesAcesso,
		OnlyUnread:   input.OnlyUnread,
	})
	if err != nil {
		return nil, err
	}
	return desktopapi.NFeRows(documents), nil
}

// MarkNFeViewed marks the company's NF-e with the given chaves as viewed and
// returns how many were new.
func (a *App) MarkNFeViewed(input desktopapi.MarkViewedInput) (int, error) {
	if err := a.ready(); err != nil {
		return 0, err
	}
	return a.svc.nfe.MarkViewed(a.ctx, input.CNPJ, input.ChavesAcesso)
}

func (a *App) ListNFeEvents(input desktopapi.NFeKeyInput) ([]desktopapi.NFeEvent, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	events, err := a.svc.nfe.ListEvents(a.ctx, input.CNPJ, input.ChaveAcesso)
	if err != nil {
		return nil, err
	}
	return desktopapi.NFeEvents(events), nil
}

func (a *App) ListNFePendingManifestacoes(input desktopapi.NFePendingInput) ([]desktopapi.NFePendingRow, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	pending, err := a.svc.nfe.ListPendingManifestacoes(a.ctx, app.NFePendingInput{
		CNPJ:          input.CNPJ,
		DueWithinDays: input.DueWithinDays,
	})
	if err != nil {
		return nil, err
	}
	return desktopapi.NFePendingRows(pending), nil
}

// PlanNFeCiencia lists which NF-e RegisterNFeCiencia would send and which it would
// skip. It sends nothing and asks for no password.
func (a *App) PlanNFeCiencia(input desktopapi.RegisterNFeCienciaInput) (desktopapi.NFeCienciaPlan, error) {
	if err := a.ready(); err != nil {
		return desktopapi.NFeCienciaPlan{}, err
	}
	plan, err := a.svc.nfe.PlanCiencia(a.ctx, app.NFeCienciaInput{
		CNPJ:         input.CNPJ,
		ChavesAcesso: input.ChavesAcesso,
	})
	if err != nil {
		return desktopapi.NFeCienciaPlan{}, err
	}
	return desktopapi.NFeCienciaPlanFrom(plan), nil
}

// RegisterNFeCiencia sends Ciência da Operação for the eligible NF-e. Failures
// after sending started are reported per chave in the result, not as an error.
func (a *App) RegisterNFeCiencia(input desktopapi.RegisterNFeCienciaInput) (desktopapi.NFeEventBatchResult, error) {
	if err := a.ready(); err != nil {
		return desktopapi.NFeEventBatchResult{}, err
	}
	summary, err := a.svc.nfe.RegisterCiencia(a.ctx, app.NFeCienciaInput{
		CNPJ:         input.CNPJ,
		ChavesAcesso: input.ChavesAcesso,
	})
	if err != nil {
		return desktopapi.NFeEventBatchResult{}, err
	}
	return desktopapi.NFeEventResults(summary), nil
}

func (a *App) RegisterNFeManifestacao(input desktopapi.RegisterNFeManifestacaoInput) (desktopapi.NFeEventResult, error) {
	if err := a.ready(); err != nil {
		return desktopapi.NFeEventResult{}, err
	}
	outcome, err := a.svc.nfe.RegisterManifestacao(a.ctx, app.NFeManifestacaoInput{
		CNPJ:          input.CNPJ,
		ChaveAcesso:   input.ChaveAcesso,
		Tipo:          input.Tipo,
		Justificativa: input.Justificativa,
	})
	if err != nil {
		return desktopapi.NFeEventResult{}, err
	}
	return desktopapi.NFeEventResultFrom(outcome), nil
}

func (a *App) ExportNFeXML(input desktopapi.ExportNFeXMLInput) (desktopapi.ExportResult, error) {
	if err := a.ready(); err != nil {
		return desktopapi.ExportResult{}, err
	}
	if input.OutPath == "" {
		return desktopapi.ExportResult{}, fmt.Errorf("caminho de saída não especificado")
	}

	err := a.svc.nfe.ExportXML(a.ctx, app.NFeExportXMLInput{
		CNPJ:        input.CNPJ,
		ChaveAcesso: input.ChaveAcesso,
		OutPath:     input.OutPath,
	})
	if err != nil {
		return desktopapi.ExportResult{}, err
	}
	return desktopapi.ExportResult{OutPath: input.OutPath, Format: "xml", ExportedCount: 1}, nil
}

func (a *App) ExportDANFE(input desktopapi.ExportDANFEInput) (desktopapi.ExportResult, error) {
	if err := a.ready(); err != nil {
		return desktopapi.ExportResult{}, err
	}
	if input.OutPath == "" {
		return desktopapi.ExportResult{}, fmt.Errorf("caminho de saída não especificado")
	}

	err := a.svc.nfe.ExportDANFE(a.ctx, app.NFeExportDANFEInput{
		CNPJ:        input.CNPJ,
		ChaveAcesso: input.ChaveAcesso,
		OutPath:     input.OutPath,
	})
	if err != nil {
		return desktopapi.ExportResult{}, err
	}
	return desktopapi.ExportResult{OutPath: input.OutPath, Format: "danfe", ExportedCount: 1}, nil
}

func (a *App) ExportNFeDocuments(input desktopapi.ExportNFeDocumentsInput) (desktopapi.NFeExportResult, error) {
	if err := a.ready(); err != nil {
		return desktopapi.NFeExportResult{}, err
	}
	format := strings.ToLower(strings.TrimSpace(input.Format))
	if input.OutPath == "" {
		return desktopapi.NFeExportResult{}, fmt.Errorf("caminho de saída não especificado")
	}

	exportInput := app.NFeExportInput{
		CNPJ:           input.CNPJ,
		Competence:     input.Competence,
		Role:           input.Role,
		ChavesAcesso:   input.ChavesAcesso,
		IncludeResumos: input.IncludeResumos,
		Incremental:    input.Incremental,
		OutPath:        input.OutPath,
	}

	var res app.NFeExportResult
	var err error
	switch format {
	case "csv":
		res, err = a.svc.nfe.ExportCSV(a.ctx, exportInput)
	case "xlsx":
		res, err = a.svc.nfe.ExportXLSX(a.ctx, exportInput)
	case "zip":
		res, err = a.svc.nfe.ExportXMLZip(a.ctx, exportInput)
	case "danfe-zip":
		res, err = a.svc.nfe.ExportDANFEZip(a.ctx, exportInput)
	default:
		return desktopapi.NFeExportResult{}, fmt.Errorf("formato de exportação desconhecido: %s", format)
	}
	if err != nil {
		return desktopapi.NFeExportResult{}, err
	}
	return desktopapi.NFeExportResult{
		ExportResult:   desktopapi.ExportResult(res.ExportResult),
		SkippedResumos: res.SkippedResumos,
	}, nil
}

// ResetNFe removes the company's NF-e and resets its NF-e sync, which lets the
// company environment change again.
func (a *App) ResetNFe(cnpj string) (desktopapi.NFeResetResult, error) {
	if err := a.ready(); err != nil {
		return desktopapi.NFeResetResult{}, err
	}
	res, err := a.svc.nfe.Reset(a.ctx, cnpj)
	if err != nil {
		return desktopapi.NFeResetResult{}, err
	}
	return desktopapi.NFeResetResult{
		CompanyName:       res.CompanyName,
		CNPJ:              res.CNPJ,
		CompanyDocuments:  res.CompanyDocuments,
		Documents:         res.Documents,
		Events:            res.Events,
		ExportMarks:       res.ExportMarks,
		ManifestacoesKept: res.ManifestacoesKept,
	}, nil
}
