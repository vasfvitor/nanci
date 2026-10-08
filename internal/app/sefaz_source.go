package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/vasfvitor/nanci/internal/company"
	"github.com/vasfvitor/nanci/internal/sefaz"
	"github.com/vasfvitor/nanci/internal/sync"
	"github.com/vasfvitor/nanci/internal/syncstate"
)

// SefazSourceStatus is the part of the NF-e and CT-e status that comes from
// the company and the sync tables, the same for both SEFAZ sources.
type SefazSourceStatus struct {
	CompanyName       string
	CNPJ              string
	UF                string
	TpAmb             string // "1" produção, "2" homologação
	LastNSU           int64
	MaxNSU            *int64 // nil when unknown
	LastSyncAt        *time.Time
	LastRunStatus     string
	LastRunStopReason string
	InitialSyncDoneAt *time.Time
	NextAllowedAt     *time.Time // set while SEFAZ must not be queried
	BlockedReason     string     // caught_up | consumo_indevido | rate_budget; empty when not blocked
	RequestsLastHour  int
	RequestBudget     int
	// IdleDays is the number of days since the last successful distribution
	// query, set only when it reached sync.DistIdleWarningDays; 0 otherwise.
	// The Ambiente Nacional stops generating NSUs for the CNPJ root after 60
	// days without a distNSU, and a pull resets the count.
	IdleDays int
}

// loadSefazSourceStatus reads the company's cursor, last run, initial sync
// and request limits for one SEFAZ source, and returns them with the
// company. now dates the idle warning. It never contacts SEFAZ.
func loadSefazSourceStatus(ctx context.Context, companies *company.Store, syncRepo *sync.Store, manager *sync.Manager, cnpj string, source syncstate.SyncSource, now time.Time) (SefazSourceStatus, *company.Company, error) {
	comp, err := company.LookupByCNPJ(ctx, companies, cnpj)
	if err != nil {
		return SefazSourceStatus{}, nil, err
	}
	tpAmb, err := environmentTpAmb(comp)
	if err != nil {
		return SefazSourceStatus{}, nil, err
	}
	status := SefazSourceStatus{
		CompanyName: comp.Name,
		CNPJ:        comp.CNPJ,
		UF:          comp.UF,
		TpAmb:       tpAmb,
	}

	snapshot, err := syncRepo.LatestSyncSnapshot(ctx, comp.ID, source, comp.Environment, comp.CNPJ)
	if err != nil {
		return SefazSourceStatus{}, nil, fmt.Errorf("carregar snapshot de sincronização: %w", err)
	}
	if snapshot.State != nil {
		status.LastNSU = snapshot.State.LastProcessedNSU
		status.MaxNSU = snapshot.State.MaxNSU
		status.LastSyncAt = snapshot.State.LastSuccessAt
		status.IdleDays = sync.DistIdle(snapshot.State.LastSuccessAt, now)
	}
	if snapshot.Run != nil {
		status.LastRunStatus = string(snapshot.Run.Status)
		status.LastRunStopReason = string(snapshot.Run.StopReason)
		if snapshot.Run.FinishedAt != nil {
			status.LastSyncAt = snapshot.Run.FinishedAt
		}
	}

	sourceState, err := syncRepo.SourceState(ctx, comp.ID, source, comp.Environment)
	if err != nil {
		return SefazSourceStatus{}, nil, fmt.Errorf("carregar estado da origem: %w", err)
	}
	status.InitialSyncDoneAt = sourceState.InitialSyncDoneAt
	limits, err := manager.SourceLimits(ctx, comp, source)
	if err != nil {
		return SefazSourceStatus{}, nil, err
	}
	status.NextAllowedAt = limits.NextAllowedAt
	status.BlockedReason = string(limits.BlockedReason)
	status.RequestsLastHour = limits.RequestsLastHour
	status.RequestBudget = limits.RequestBudget
	return status, comp, nil
}

// sefazConnection names what a connection test checks.
type sefazConnection struct {
	purpose string // shown in the password prompt
	target  string // "SEFAZ" or "SEFAZ (CT-e)" in the result texts
	// check is the method expression sefazClient.CheckTLS or CheckTLSCTe.
	check func(client sefazClient, ctx context.Context) error
}

// testSefazConnection loads the certificate and opens a TLS connection with
// conn.check. It sends no request and spends no hourly budget; SEFAZ only
// checks the client certificate on a real query.
func testSefazConnection(ctx context.Context, log *slog.Logger, companies *company.Store, certs *sync.CertificateLoader, cnpj string, conn sefazConnection) (ConnectionTestResult, error) {
	var result ConnectionTestResult
	comp, err := company.LookupByCNPJ(ctx, companies, cnpj)
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
	if err := conn.check(client, ctx); err != nil {
		result.StatusExplanation = fmt.Sprintf("Falha na conexão TLS com a %s: %v", conn.target, err)
		return result, nil
	}
	result.EndpointReached = true
	result.StatusExplanation = fmt.Sprintf("Conexão TLS com a %s estabelecida. Nenhuma consulta foi enviada, para não gastar o limite de consultas por hora; a SEFAZ só valida o certificado na primeira consulta.", conn.target)
	return result, nil
}
