package store_test

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"testing"

	"github.com/vasfvitor/nanci/internal/store"
)

// frozenMigrations is the SHA-256 of every migration file. goose records the
// version it applied and never runs a file again, so editing one leaves
// existing databases with the old schema while new ones get the new (a
// dev database once lost cte_documents.masked_keys that way).
var frozenMigrations = map[string]string{
	"001_v2_schema.sql":                             "cce11d17459071db73a33134610e58977fef6c7587c23bc30b3eef848e8e5083",
	"002_drop_environment.sql":                      "9cf0f8200232547ab9136f4e4ccb38c200a0b60b0febdaebd22f2d6629ba63ff",
	"003_sync_state.sql":                            "1fbd01a9ab2fd94d188593ad20f756ad5d64efbf60f8769fec83695a36c54079",
	"004_incremental_export.sql":                    "94bf8255a11e45be77c133a5211730560ca94d3c0de45e8d2f3cb896da2e631c",
	"005_initial_sync_policy.sql":                   "414ca0697473afa94a268bf7d1ec2d437d0a1588b76c6e41e47bda91a840cdcd",
	"006_schema_cleanup.sql":                        "b31d1ebff3a7e83413ddc29e0463dadf79f9207328c2d1d9146f018eedcd28e3",
	"007_sync_source.sql":                           "97db5c761ac577c9ec06b130f75b537df53e5e9327eff57a3d4ebe94f395d06c",
	"008_nfe.sql":                                   "5a4c73d8404844d1d09787d8b268491fe69b1f13cd9b9e460ba280a1acb89026",
	"009_company_uf.sql":                            "13e494604a11fb29be8892a45e4add96782f75867c6a650924a2af14cdb680a1",
	"010_sync_item_failures.sql":                    "1cd169a42d398650b0a77b29cfda6eb913eb474a3b5cd397812b1e222a190f5f",
	"011_nfe_manifestations_tp_amb.sql":             "820f7bf38a0473fc5311a9a9af17d40f6e6eb51299cfcdee05e08bd90b518077",
	"012_company_nfe_documents_document_index.sql":  "aa10e138afa97134e7e6f9def737743e813e62d046a54d2872d468c34fa8aeea",
	"013_manifestacoes_and_sync_requests_check.sql": "e864830e249a0c87467e11193816f9748a9ae5b4ffc680fce0dc6d42897f4663",
	"014_drop_initial_sync_mirror.sql":              "85ae74e2e11a92379a5a94c9dd2e0176fd0520a3048e4834052824f94ebd5065",
	"015_cte.sql":                                   "2851755e9c49a9fb3baf371b86f9a6a38b90686170c0f57647086a1386e9cf24",
	"016_nfe_tp_amb.sql":                            "c2593c5b4aaf2bb597f1b34219fb866c523902b5a0558ae55b79f47d94166bd7",
	"017_cte_viewed_at.sql":                         "8022f7e220ac538f8b2b3b039b05b1b774637fbdd1c71064e7af75ddac141286",
	"018_export_marks_event_count.sql":              "7569d40be4e906552a9ed7fd1536ecce6430a1927fdf105238366f44f7ebca7b",
	"019_sync_sources_by_environment.sql":           "e22a0bed0f9ba2e0bb5d632f8308c069e9cceacd545d5c1ffce1c366a53f0f19",
	"020_nfse_chave_without_prefix.sql":             "d57eaf84545fe48d08dfa8cbd88cb094389c5d775ea7cb25ef02d1ea6e1c655a",
	"021_export_kinds.sql":                          "aec53a4d997dab93836ef091ae885c1799ecaf6f1fac567aa4f9b09532706ddd",
	"022_sync_run_errors.sql":                       "22efe5bf83c03c0cee55180c252a46024b280451ee5c803460272bb3b8591c5b",
}

func TestMigrationsAreFrozen(t *testing.T) {
	migrations, err := store.Migrations()
	if err != nil {
		t.Fatal(err)
	}
	files, err := fs.Glob(migrations, "*.sql")
	if err != nil {
		t.Fatal(err)
	}

	present := map[string]bool{}
	for _, name := range files {
		present[name] = true
		data, err := fs.ReadFile(migrations, name)
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(data)
		got := hex.EncodeToString(sum[:])
		want, known := frozenMigrations[name]
		switch {
		case !known:
			t.Errorf("new migration %s: append %q: %q, to frozenMigrations", name, name, got)
		case got != want:
			t.Errorf("%s changed after it was applied: write the change as a new migration instead of editing this one", name)
		}
	}
	for name := range frozenMigrations {
		if !present[name] {
			t.Errorf("%s was removed or renamed: databases that applied it keep its version; restore it", name)
		}
	}
}
