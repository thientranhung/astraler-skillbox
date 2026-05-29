package repositories

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

// ---- migration tests ----

func TestMigration000012_TablesExist(t *testing.T) {
	db := NewTestDB(t)
	for _, tbl := range []string{"provider_plugin_layer_scans", "provider_plugin_entries", "provider_plugin_marketplaces"} {
		var name string
		err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, tbl).Scan(&name)
		if err != nil {
			t.Errorf("table %q missing: %v", tbl, err)
		}
	}
}

func TestMigration000012_DatabaseVersion(t *testing.T) {
	db := NewTestDB(t)
	var v int
	if err := db.QueryRow(`SELECT database_version FROM app_settings WHERE id=1`).Scan(&v); err != nil {
		t.Fatalf("database_version query: %v", err)
	}
	if v != 22 {
		t.Errorf("database_version: got %d want 22", v)
	}
}

func TestMigration000013_ScanWarningsColumn(t *testing.T) {
	db := NewTestDB(t)
	var v int
	if err := db.QueryRow(`SELECT database_version FROM app_settings WHERE id=1`).Scan(&v); err != nil {
		t.Fatalf("database_version query: %v", err)
	}
	if v != 22 {
		t.Errorf("database_version: got %d want 22", v)
	}

	// Column must exist with default value of '[]'
	var provID int64
	if err := db.QueryRow(`SELECT id FROM provider_definitions WHERE key='claude'`).Scan(&provID); err != nil {
		t.Fatalf("claude not found: %v", err)
	}
	res, err := db.Exec(`INSERT INTO provider_plugin_layer_scans (provider_definition_id, project_id, settings_layer, scan_status, settings_file_path) VALUES (?, NULL, 'user', 'ok', '/tmp/f')`, provID)
	if err != nil {
		t.Fatalf("insert without scan_warnings: %v", err)
	}
	id, _ := res.LastInsertId()
	var w string
	if err := db.QueryRow(`SELECT scan_warnings FROM provider_plugin_layer_scans WHERE id=?`, id).Scan(&w); err != nil {
		t.Fatalf("select scan_warnings: %v", err)
	}
	if w != "[]" {
		t.Errorf("scan_warnings default: got %q want []", w)
	}
}

func TestMigration000012_CheckConstraint_UserLayerMustHaveNullProjectID(t *testing.T) {
	db := NewTestDB(t)
	var provID int64
	if err := db.QueryRow(`SELECT id FROM provider_definitions WHERE key='claude'`).Scan(&provID); err != nil {
		t.Fatalf("claude not found: %v", err)
	}
	// user layer with non-null project_id should fail
	_, err := db.Exec(`
		INSERT INTO provider_plugin_layer_scans (provider_definition_id, project_id, settings_layer, scan_status, settings_file_path)
		VALUES (?, 1, 'user', 'ok', '/tmp/f')`, provID)
	if err == nil {
		t.Error("expected CHECK constraint error: user layer with non-null project_id")
	}
}

func TestMigration000012_CheckConstraint_ProjectLayerMustHaveNonNullProjectID(t *testing.T) {
	db := NewTestDB(t)
	var provID int64
	if err := db.QueryRow(`SELECT id FROM provider_definitions WHERE key='claude'`).Scan(&provID); err != nil {
		t.Fatalf("claude not found: %v", err)
	}
	// project layer with null project_id should fail
	_, err := db.Exec(`
		INSERT INTO provider_plugin_layer_scans (provider_definition_id, project_id, settings_layer, scan_status, settings_file_path)
		VALUES (?, NULL, 'project', 'ok', '/tmp/f')`, provID)
	if err == nil {
		t.Error("expected CHECK constraint error: project layer with null project_id")
	}
}

func TestMigration000012_UniqueIndex_UserLayer(t *testing.T) {
	db := NewTestDB(t)
	var provID int64
	if err := db.QueryRow(`SELECT id FROM provider_definitions WHERE key='claude'`).Scan(&provID); err != nil {
		t.Fatalf("claude not found: %v", err)
	}
	_, err := db.Exec(`INSERT INTO provider_plugin_layer_scans (provider_definition_id, project_id, settings_layer, scan_status, settings_file_path) VALUES (?, NULL, 'user', 'ok', '/tmp/f')`, provID)
	if err != nil {
		t.Fatalf("first insert: %v", err)
	}
	_, err = db.Exec(`INSERT INTO provider_plugin_layer_scans (provider_definition_id, project_id, settings_layer, scan_status, settings_file_path) VALUES (?, NULL, 'user', 'missing', '/tmp/f')`, provID)
	if err == nil {
		t.Error("expected UNIQUE index violation on second user-layer insert for same provider")
	}
}

func TestMigration000012_FKCascadeOnProviderDelete(t *testing.T) {
	db := NewTestDB(t)
	var provID int64
	if err := db.QueryRow(`SELECT id FROM provider_definitions WHERE key='claude'`).Scan(&provID); err != nil {
		t.Fatalf("claude not found: %v", err)
	}
	_, err := db.Exec(`INSERT INTO provider_plugin_layer_scans (provider_definition_id, project_id, settings_layer, scan_status, settings_file_path) VALUES (?, NULL, 'user', 'ok', '/tmp/f')`, provID)
	if err != nil {
		t.Fatalf("insert scan: %v", err)
	}
	// FK to nonexistent provider_definition should fail
	_, err = db.Exec(`INSERT INTO provider_plugin_layer_scans (provider_definition_id, project_id, settings_layer, scan_status, settings_file_path) VALUES (99999, NULL, 'user', 'ok', '/tmp/f')`)
	if err == nil {
		t.Error("expected FK constraint error for nonexistent provider_definition_id")
	}
}

func TestMigration000012_EntriesFK_Cascade(t *testing.T) {
	db := NewTestDB(t)
	var provID int64
	if err := db.QueryRow(`SELECT id FROM provider_definitions WHERE key='claude'`).Scan(&provID); err != nil {
		t.Fatalf("claude not found: %v", err)
	}
	res, err := db.Exec(`INSERT INTO provider_plugin_layer_scans (provider_definition_id, project_id, settings_layer, scan_status, settings_file_path) VALUES (?, NULL, 'user', 'ok', '/tmp/f')`, provID)
	if err != nil {
		t.Fatalf("insert scan: %v", err)
	}
	scanID, _ := res.LastInsertId()
	_, err = db.Exec(`INSERT INTO provider_plugin_entries (layer_scan_id, plugin_name, marketplace_name, declaration) VALUES (?, 'plugin', 'npm', 'enabled')`, scanID)
	if err != nil {
		t.Fatalf("insert entry: %v", err)
	}
	// Delete scan → entry must cascade
	if _, err := db.Exec(`DELETE FROM provider_plugin_layer_scans WHERE id=?`, scanID); err != nil {
		t.Fatalf("delete scan: %v", err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM provider_plugin_entries WHERE layer_scan_id=?`, scanID).Scan(&count); err != nil {
		t.Fatalf("count entries: %v", err)
	}
	if count != 0 {
		t.Errorf("expected cascade delete, got %d orphan entries", count)
	}
}

// ---- repo CRUD tests ----

func TestProviderPluginRepo_CommitLayerScan_Insert(t *testing.T) {
	db := NewTestDB(t)
	r := NewProviderPluginRepo(db)
	ctx := context.Background()

	var provID int64
	if err := db.QueryRow(`SELECT id FROM provider_definitions WHERE key='claude'`).Scan(&provID); err != nil {
		t.Fatalf("claude not found: %v", err)
	}

	scan := &domain.PluginLayerScan{
		ProviderDefinitionID: provID,
		ProjectID:            nil,
		SettingsLayer:        domain.PluginLayerUser,
		ScanStatus:           domain.PluginLayerScanOK,
		SettingsFilePath:     "/home/user/.claude/settings.json",
		LastScannedAt:        time.Now().UTC(),
	}
	entries := []domain.PluginEntry{
		{PluginName: "plugin-a", MarketplaceName: "npm", Declaration: domain.PluginDeclarationEnabled},
		{PluginName: "plugin-b", MarketplaceName: "npm", Declaration: domain.PluginDeclarationDisabled},
	}
	mps := []domain.PluginMarketplace{
		{MarketplaceName: "npm", SourceType: "github", SourceSummary: "anthropics/plugins"},
	}

	if err := r.CommitLayerScan(ctx, scan, entries, mps); err != nil {
		t.Fatalf("CommitLayerScan: %v", err)
	}

	scans, err := r.ListLayerScansForProvider(ctx, provID)
	if err != nil {
		t.Fatalf("ListLayerScansForProvider: %v", err)
	}
	if len(scans) != 1 {
		t.Fatalf("scans: got %d want 1", len(scans))
	}
	if scans[0].ScanStatus != domain.PluginLayerScanOK {
		t.Errorf("scan status: got %q want ok", scans[0].ScanStatus)
	}

	gotEntries, err := r.ListEntriesForScan(ctx, scans[0].ID)
	if err != nil {
		t.Fatalf("ListEntriesForScan: %v", err)
	}
	if len(gotEntries) != 2 {
		t.Errorf("entries: got %d want 2", len(gotEntries))
	}

	gotMPs, err := r.ListMarketplacesForScan(ctx, scans[0].ID)
	if err != nil {
		t.Fatalf("ListMarketplacesForScan: %v", err)
	}
	if len(gotMPs) != 1 {
		t.Errorf("marketplaces: got %d want 1", len(gotMPs))
	}
}

func TestProviderPluginRepo_CommitLayerScan_Upsert(t *testing.T) {
	db := NewTestDB(t)
	r := NewProviderPluginRepo(db)
	ctx := context.Background()

	var provID int64
	if err := db.QueryRow(`SELECT id FROM provider_definitions WHERE key='claude'`).Scan(&provID); err != nil {
		t.Fatalf("claude not found: %v", err)
	}

	scan := &domain.PluginLayerScan{
		ProviderDefinitionID: provID,
		SettingsLayer:        domain.PluginLayerUser,
		ScanStatus:           domain.PluginLayerScanOK,
		SettingsFilePath:     "/tmp/s.json",
		LastScannedAt:        time.Now().UTC(),
	}
	entries1 := []domain.PluginEntry{
		{PluginName: "old-plugin", MarketplaceName: "npm", Declaration: domain.PluginDeclarationEnabled},
	}
	if err := r.CommitLayerScan(ctx, scan, entries1, nil); err != nil {
		t.Fatalf("first commit: %v", err)
	}

	// Second commit: same layer, different status and entries
	scan.ScanStatus = domain.PluginLayerScanMalformed
	entries2 := []domain.PluginEntry{} // empty on malformed
	if err := r.CommitLayerScan(ctx, scan, entries2, nil); err != nil {
		t.Fatalf("second commit: %v", err)
	}

	scans, _ := r.ListLayerScansForProvider(ctx, provID)
	if len(scans) != 1 {
		t.Fatalf("expected 1 scan row after upsert, got %d", len(scans))
	}
	if scans[0].ScanStatus != domain.PluginLayerScanMalformed {
		t.Errorf("scan status after upsert: got %q want malformed", scans[0].ScanStatus)
	}

	// Stale entries must be cleared
	entries, _ := r.ListEntriesForScan(ctx, scans[0].ID)
	if len(entries) != 0 {
		t.Errorf("expected 0 entries after malformed scan, got %d (stale state persisted)", len(entries))
	}
}

func TestProviderPluginRepo_CommitLayerScan_WarningsPersisted(t *testing.T) {
	db := NewTestDB(t)
	r := NewProviderPluginRepo(db)
	ctx := context.Background()

	var provID int64
	if err := db.QueryRow(`SELECT id FROM provider_definitions WHERE key='claude'`).Scan(&provID); err != nil {
		t.Fatalf("claude not found: %v", err)
	}

	scan := &domain.PluginLayerScan{
		ProviderDefinitionID: provID,
		SettingsLayer:        domain.PluginLayerUser,
		ScanStatus:           domain.PluginLayerScanOK,
		SettingsFilePath:     "/tmp/s.json",
		LastScannedAt:        time.Now().UTC(),
		Warnings: []string{
			"enabledPlugins truncated at 1000 entries",
			"skipped enabledPlugins entry: key format must be name@marketplace",
		},
	}
	if err := r.CommitLayerScan(ctx, scan, nil, nil); err != nil {
		t.Fatalf("CommitLayerScan: %v", err)
	}

	scans, err := r.ListLayerScansForProvider(ctx, provID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(scans) != 1 {
		t.Fatalf("scans: got %d want 1", len(scans))
	}
	if len(scans[0].Warnings) != 2 {
		t.Fatalf("warnings: got %d want 2", len(scans[0].Warnings))
	}
	if scans[0].Warnings[0] != "enabledPlugins truncated at 1000 entries" {
		t.Errorf("warning[0]: got %q", scans[0].Warnings[0])
	}
	// Verify no raw key content leaks through
	for _, w := range scans[0].Warnings {
		if strings.Contains(w, "bad") || strings.Contains(w, "%q") {
			t.Errorf("warning contains raw key content: %q", w)
		}
	}
}

func TestProviderPluginRepo_CommitLayerScan_EmptyWarningsRoundTrip(t *testing.T) {
	db := NewTestDB(t)
	r := NewProviderPluginRepo(db)
	ctx := context.Background()

	var provID int64
	if err := db.QueryRow(`SELECT id FROM provider_definitions WHERE key='claude'`).Scan(&provID); err != nil {
		t.Fatalf("claude not found: %v", err)
	}

	scan := &domain.PluginLayerScan{
		ProviderDefinitionID: provID,
		SettingsLayer:        domain.PluginLayerUser,
		ScanStatus:           domain.PluginLayerScanMissing,
		SettingsFilePath:     "/tmp/s.json",
		LastScannedAt:        time.Now().UTC(),
		Warnings:             nil,
	}
	if err := r.CommitLayerScan(ctx, scan, nil, nil); err != nil {
		t.Fatalf("CommitLayerScan: %v", err)
	}

	scans, _ := r.ListLayerScansForProvider(ctx, provID)
	if scans[0].Warnings != nil {
		t.Errorf("nil warnings should round-trip as nil, got %v", scans[0].Warnings)
	}
}

func TestProviderPluginRepo_CommitLayerScan_DeleteReplace(t *testing.T) {
	db := NewTestDB(t)
	r := NewProviderPluginRepo(db)
	ctx := context.Background()

	var provID int64
	if err := db.QueryRow(`SELECT id FROM provider_definitions WHERE key='claude'`).Scan(&provID); err != nil {
		t.Fatalf("claude not found: %v", err)
	}

	scan := &domain.PluginLayerScan{
		ProviderDefinitionID: provID,
		SettingsLayer:        domain.PluginLayerUser,
		ScanStatus:           domain.PluginLayerScanOK,
		SettingsFilePath:     "/tmp/s.json",
		LastScannedAt:        time.Now().UTC(),
	}
	// First commit: 3 plugins
	if err := r.CommitLayerScan(ctx, scan, []domain.PluginEntry{
		{PluginName: "a", MarketplaceName: "npm", Declaration: domain.PluginDeclarationEnabled},
		{PluginName: "b", MarketplaceName: "npm", Declaration: domain.PluginDeclarationEnabled},
		{PluginName: "c", MarketplaceName: "npm", Declaration: domain.PluginDeclarationEnabled},
	}, nil); err != nil {
		t.Fatalf("first commit: %v", err)
	}

	scans, _ := r.ListLayerScansForProvider(ctx, provID)
	scanID := scans[0].ID

	// Second commit: only 1 plugin (c removed)
	if err := r.CommitLayerScan(ctx, scan, []domain.PluginEntry{
		{PluginName: "a", MarketplaceName: "npm", Declaration: domain.PluginDeclarationEnabled},
	}, nil); err != nil {
		t.Fatalf("second commit: %v", err)
	}

	entries, _ := r.ListEntriesForScan(ctx, scanID)
	if len(entries) != 1 {
		t.Errorf("entries after delete-replace: got %d want 1", len(entries))
	}
	if entries[0].PluginName != "a" {
		t.Errorf("remaining entry: got %q want a", entries[0].PluginName)
	}
}


func TestMigration000021_VersionColumn(t *testing.T) {
	db := NewTestDB(t)
	// Verify the version column was added by migration 000021
	var v int
	if err := db.QueryRow(`SELECT database_version FROM app_settings WHERE id=1`).Scan(&v); err != nil {
		t.Fatalf("database_version query: %v", err)
	}
	if v != 22 {
		t.Errorf("database_version: got %d want 22", v)
	}

	// Column must exist and accept NULL
	var provID int64
	if err := db.QueryRow(`SELECT id FROM provider_definitions WHERE key='claude'`).Scan(&provID); err != nil {
		t.Fatalf("claude not found: %v", err)
	}
	res, err := db.Exec(`INSERT INTO provider_plugin_layer_scans (provider_definition_id, project_id, settings_layer, scan_status, settings_file_path) VALUES (?, NULL, 'user', 'ok', '/tmp/f')`, provID)
	if err != nil {
		t.Fatalf("insert layer scan: %v", err)
	}
	scanID, _ := res.LastInsertId()

	_, err = db.Exec(`INSERT INTO provider_plugin_entries (layer_scan_id, plugin_name, marketplace_name, declaration, version) VALUES (?, 'plug', 'mkt', 'enabled', NULL)`, scanID)
	if err != nil {
		t.Fatalf("insert entry with NULL version: %v", err)
	}
	_, err = db.Exec(`INSERT INTO provider_plugin_entries (layer_scan_id, plugin_name, marketplace_name, declaration, version) VALUES (?, 'plug2', 'mkt', 'enabled', '1.0.0')`, scanID)
	if err != nil {
		t.Fatalf("insert entry with version: %v", err)
	}
}

func TestProviderPluginRepo_VersionRoundTrip(t *testing.T) {
	db := NewTestDB(t)
	r := NewProviderPluginRepo(db)
	ctx := context.Background()

	var provID int64
	if err := db.QueryRow(`SELECT id FROM provider_definitions WHERE key='claude'`).Scan(&provID); err != nil {
		t.Fatalf("claude not found: %v", err)
	}

	scan := &domain.PluginLayerScan{
		ProviderDefinitionID: provID,
		SettingsLayer:        domain.PluginLayerUser,
		ScanStatus:           domain.PluginLayerScanOK,
		SettingsFilePath:     "/tmp/s.json",
		LastScannedAt:        time.Now().UTC(),
	}
	v100 := "1.0.0"
	vUnk := "unknown"
	entries := []domain.PluginEntry{
		{PluginName: "has-version", MarketplaceName: "mkt", Declaration: domain.PluginDeclarationEnabled, Version: &v100},
		{PluginName: "unknown-version", MarketplaceName: "mkt", Declaration: domain.PluginDeclarationEnabled, Version: &vUnk},
		{PluginName: "nil-version", MarketplaceName: "mkt", Declaration: domain.PluginDeclarationEnabled, Version: nil},
	}
	if err := r.CommitLayerScan(ctx, scan, entries, nil); err != nil {
		t.Fatalf("CommitLayerScan: %v", err)
	}

	scans, _ := r.ListLayerScansForProvider(ctx, provID)
	got, err := r.ListEntriesForScan(ctx, scans[0].ID)
	if err != nil {
		t.Fatalf("ListEntriesForScan: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("entries len = %d, want 3", len(got))
	}
	if got[0].Version == nil || *got[0].Version != "1.0.0" {
		t.Errorf("has-version: got %v, want 1.0.0", got[0].Version)
	}
	if got[1].Version == nil || *got[1].Version != "unknown" {
		t.Errorf("unknown-version: got %v, want 'unknown'", got[1].Version)
	}
	if got[2].Version != nil {
		t.Errorf("nil-version: got %v, want nil", got[2].Version)
	}
}
