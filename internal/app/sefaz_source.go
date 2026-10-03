package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/vasfvitor/nanci/internal/company"
	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/vasfvitor/nanci/internal/sefaz"
	"github.com/vasfvitor/nanci/internal/sync"
)

// sefazSourceStatus is the part of the NF-e and CT-e status that comes from
// the sync tables, the same for both SEFAZ sources.
type sefazSourceStatus struct {
	comp              *nfse.Company
	tpAmb             string // "1" produção, "2" homologação
	lastNSU           int64
	maxNSU            *int64 // nil when unknown
	lastSyncAt        *time.Time
	lastRunStatus     string
	lastRunStopReason string
	initialSyncDoneAt *time.Time
	nextAllowedAt     *time.Time // set while SEFAZ must not be queried
	blockedReason     string     // empty when not blocked
	requestsLastHour  int
	requestBudget     int
}

// loadSefazSourceStatus reads the company's cursor, last run, initial sync
// and request limits for one SEFAZ source. It never contacts SEFAZ.
func loadSefazSourceStatus(ctx context.Context, companies *company.Store, syncRepo *sync.Store, manager *sync.Manager, cnpj string, source nfse.SyncSource) (sefazSourceStatus, error) {
	comp, err := lookupCompanyByCNPJ(ctx, companies, cnpj)
	if err != nil {
		return sefazSourceStatus{}, err
	}
	tpAmb, err := environmentTpAmb(comp)
	if err != nil {
		return sefazSourceStatus{}, err
	}
	status := sefazSourceStatus{comp: comp, tpAmb: tpAmb}

	snapshot, err := syncRepo.LatestSyncSnapshot(ctx, comp.ID, source, comp.Environment, comp.CNPJ)
	if err != nil {
		return sefazSourceStatus{}, fmt.Errorf("carregar snapshot de sincronização: %w", err)
	}
	if snapshot.State != nil {
		status.lastNSU = snapshot.State.LastProcessedNSU
		status.maxNSU = snapshot.State.MaxNSU
		status.lastSyncAt = snapshot.State.LastSuccessAt
	}
	if snapshot.Run != nil {
		status.lastRunStatus = string(snapshot.Run.Status)
		status.lastRunStopReason = string(snapshot.Run.StopReason)
		if snapshot.Run.FinishedAt != nil {
			status.lastSyncAt = snapshot.Run.FinishedAt
		}
	}

	sourceState, err := syncRepo.SourceState(ctx, comp.ID, source, comp.Environment)
	if err != nil {
		return sefazSourceStatus{}, fmt.Errorf("carregar estado da origem: %w", err)
	}
	status.initialSyncDoneAt = sourceState.InitialSyncDoneAt
	limits, err := manager.SourceLimits(ctx, comp, source)
	if err != nil {
		return sefazSourceStatus{}, err
	}
	status.nextAllowedAt = limits.NextAllowedAt
	status.blockedReason = string(limits.BlockedReason)
	status.requestsLastHour = limits.RequestsLastHour
	status.requestBudget = limits.RequestBudget
	return status, nil
}

// sefazConnection names what a connection test checks.
type sefazConnection struct {
	purpose string // shown in the password prompt
	target  string // "SEFAZ" or "SEFAZ (CT-e)" in the result texts
	check   func(ctx context.Context, client sefazClient) error
}

// testSefazConnection loads the certificate and opens a TLS connection with
// conn.check. It sends no request, so it does not use the hourly budget;
// SEFAZ only checks the client certificate on a real query.
func testSefazConnection(ctx context.Context, log *slog.Logger, companies *company.Store, certs *sync.CertificateLoader, cnpj string, conn sefazConnection) (ConnectionTestResult, error) {
	var result ConnectionTestResult
	comp, err := lookupCompanyByCNPJ(ctx, companies, cnpj)
	if err != nil {
		return result, err
	}

	loaded, err := certs.LoadForCompany(ctx, comp, conn.purpose)
	if err != nil {
		if errors.Is(err, ErrOperationCanceled) {
			return result, err
		}
		result.StatusExplanation = fmt.Sprintf("Erro ao carregar certificado/senha: %v", err)
		return result, nil
	}
	result.CertLoaded = true
	result.CertSubject = loaded.Credential.SubjectName
	if loaded.Credential.NotAfter != nil {
		result.CertExpiration = loaded.Credential.NotAfter.Format("02/01/2006 15:04:05")
	}

	client, err := newSEFAZClient(sefaz.ClientConfig{
		Environment: comp.Environment,
		Certificate: &loaded.TLS,
		Log:         log,
	})
	if err != nil {
		result.StatusExplanation = fmt.Sprintf("Erro ao configurar cliente SEFAZ: %v", err)
		return result, nil
	}
	if err := conn.check(ctx, client); err != nil {
		result.StatusExplanation = fmt.Sprintf("Falha na conexão TLS com a %s: %v", conn.target, err)
		return result, nil
	}
	result.EndpointReached = true
	result.StatusExplanation = fmt.Sprintf("Conexão TLS com a %s estabelecida. Nenhuma consulta foi enviada, para não gastar o limite de consultas por hora; a SEFAZ só valida o certificado na primeira consulta.", conn.target)
	return result, nil
}
