package commands

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/vasfvitor/nanci/internal/adn"
	"github.com/vasfvitor/nanci/internal/files"
	"github.com/vasfvitor/nanci/internal/foundation/cert"
	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/vasfvitor/nanci/internal/service/sync"
)

var pullCNPJ string

var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Sincroniza documentos fiscais da API ADN",
	Run: func(cmd *cobra.Command, args []string) {
		app, err := initApp()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao inicializar: %v\n", err)
			os.Exit(1)
		}
		defer app.Close()

		if err := cnpj.Validate(pullCNPJ); err != nil {
			fmt.Fprintf(os.Stderr, "CNPJ inválido: %v\n", err)
			os.Exit(1)
		}

		cleanedCNPJ := cnpj.Clean(pullCNPJ)
		ctx := cmd.Context()

		// 1. Get Company
		company, err := app.Store.GetCompany(ctx, cleanedCNPJ)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao buscar empresa: %v\n", err)
			os.Exit(1)
		}
		if company == nil {
			fmt.Fprintf(os.Stderr, "Empresa não encontrada para o CNPJ %s\n", cnpj.Format(cleanedCNPJ))
			os.Exit(1)
		}

		fmt.Printf("Iniciando sincronização para %s (%s)\n", company.Name, cnpj.Format(company.CNPJ))

		// 2. Get Certificate Password
		credProvider := &CLICredentialProvider{}
		pass, err := credProvider.GetCertPassword(ctx, company.CertPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro na senha do certificado: %v\n", err)
			os.Exit(1)
		}

		// 3. Load Certificate
		fmt.Println("Carregando certificado...")
		tlsCert, err := cert.LoadPKCS12(company.CertPath, string(pass))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao carregar certificado: %v\n", err)
			os.Exit(1)
		}

		// 4. Setup API Client
		httpClient := adn.NewHTTPClient(tlsCert)
		apiClient, err := adn.NewClient(httpClient, company.Environment)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao configurar API ADN: %v\n", err)
			os.Exit(1)
		}

		// 5. Setup FileWriter
		fileWriter := files.NewWriter(app.DataDir)

		// 6. Setup SyncService
		syncService := syncservice.NewSyncService(app.Store, apiClient, fileWriter)

		// 7. Define Progress Callback
		progress := func(event nfse.ProgressEvent) {
			if event.Message != "" {
				fmt.Printf("[INFO] %s\n", event.Message)
			} else {
				fmt.Printf("\rNSU: %d / %d | Lote: %d | Total Encontrado: %d | Erros: %d",
					event.CurrentNSU, event.MaxNSU, event.DocsInBatch, event.DocsFound, event.Errors)
			}
		}

		// 8. Run Sync
		fmt.Println("Conectando à API ADN...")
		startTime := time.Now()
		
		err = syncService.Sync(ctx, company, progress)
		fmt.Println() // Nova linha após a barra de progresso (se houver)

		if err != nil {
			fmt.Fprintf(os.Stderr, "Sincronização falhou: %v\n", err)
			os.Exit(1)
		}

		duration := time.Since(startTime)
		fmt.Printf("Sincronização concluída com sucesso em %v!\n", duration)
	},
}

func init() {
	rootCmd.AddCommand(pullCmd)
	pullCmd.Flags().StringVarP(&pullCNPJ, "cnpj", "c", "", "CNPJ da empresa para sincronizar")
	pullCmd.MarkFlagRequired("cnpj")
}
