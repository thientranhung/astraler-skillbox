package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

// ProviderPluginRepo manages provider_plugin_layer_scans, provider_plugin_entries,
// and provider_plugin_marketplaces tables.
type ProviderPluginRepo struct {
	db *sql.DB
}

func NewProviderPluginRepo(db *sql.DB) *ProviderPluginRepo {
	return &ProviderPluginRepo{db: db}
}

// CommitLayerScan atomically upserts a layer scan row and delete-replaces its child entries/marketplaces.
// On any non-ok scan status, stale child rows are still cleared (no stale enabled state after read failure).
func (r *ProviderPluginRepo) CommitLayerScan(
	ctx context.Context,
	scan *domain.PluginLayerScan,
	entries []domain.PluginEntry,
	marketplaces []domain.PluginMarketplace,
) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	scanID, err := upsertPluginLayerScan(ctx, tx, scan)
	if err != nil {
		return fmt.Errorf("upsert layer scan: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM provider_plugin_entries WHERE layer_scan_id = ?`, scanID); err != nil {
		return fmt.Errorf("delete entries: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM provider_plugin_marketplaces WHERE layer_scan_id = ?`, scanID); err != nil {
		return fmt.Errorf("delete marketplaces: %w", err)
	}

	for _, e := range entries {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO provider_plugin_entries (layer_scan_id, plugin_name, marketplace_name, declaration, version) VALUES (?, ?, ?, ?, ?)`,
			scanID, e.PluginName, e.MarketplaceName, string(e.Declaration), e.Version,
		); err != nil {
			return fmt.Errorf("insert entry: %w", err)
		}
	}
	for _, m := range marketplaces {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO provider_plugin_marketplaces (layer_scan_id, marketplace_name, source_type, source_summary) VALUES (?, ?, ?, ?)`,
			scanID, m.MarketplaceName, m.SourceType, m.SourceSummary,
		); err != nil {
			return fmt.Errorf("insert marketplace: %w", err)
		}
	}

	return tx.Commit()
}

// ListLayerScansForProvider returns all layer scan rows for a provider definition, ordered by id.
func (r *ProviderPluginRepo) ListLayerScansForProvider(ctx context.Context, provDefID int64) ([]domain.PluginLayerScan, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, provider_definition_id, project_id, settings_layer, scan_status, settings_file_path, last_scanned_at, source_operation_id, scan_warnings
		   FROM provider_plugin_layer_scans
		  WHERE provider_definition_id = ?
		  ORDER BY id`,
		provDefID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPluginLayerRows(rows)
}

// ListEntriesForScan returns all plugin entry rows for a given layer scan, ordered by id.
func (r *ProviderPluginRepo) ListEntriesForScan(ctx context.Context, layerScanID int64) ([]domain.PluginEntry, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, layer_scan_id, plugin_name, marketplace_name, declaration, version
		   FROM provider_plugin_entries WHERE layer_scan_id = ? ORDER BY id`,
		layerScanID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []domain.PluginEntry
	for rows.Next() {
		var e domain.PluginEntry
		var decl string
		var version sql.NullString
		if err := rows.Scan(&e.ID, &e.LayerScanID, &e.PluginName, &e.MarketplaceName, &decl, &version); err != nil {
			return nil, err
		}
		e.Declaration = domain.PluginDeclaration(decl)
		if version.Valid {
			e.Version = &version.String
		}
		result = append(result, e)
	}
	return result, rows.Err()
}

// ListMarketplacesForScan returns all marketplace rows for a given layer scan, ordered by id.
func (r *ProviderPluginRepo) ListMarketplacesForScan(ctx context.Context, layerScanID int64) ([]domain.PluginMarketplace, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, layer_scan_id, marketplace_name, source_type, source_summary
		   FROM provider_plugin_marketplaces WHERE layer_scan_id = ? ORDER BY id`,
		layerScanID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []domain.PluginMarketplace
	for rows.Next() {
		var m domain.PluginMarketplace
		if err := rows.Scan(&m.ID, &m.LayerScanID, &m.MarketplaceName, &m.SourceType, &m.SourceSummary); err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

// upsertPluginLayerScan selects an existing row (by provider+project+layer) and updates it,
// or inserts a new one. Returns the row ID.
func upsertPluginLayerScan(ctx context.Context, tx *sql.Tx, scan *domain.PluginLayerScan) (int64, error) {
	scannedAt := scan.LastScannedAt.UTC().Format(time.RFC3339)
	var opID interface{}
	if scan.SourceOperationID != nil {
		opID = *scan.SourceOperationID
	}
	warningsJSON, err := marshalWarnings(scan.Warnings)
	if err != nil {
		return 0, fmt.Errorf("marshal warnings: %w", err)
	}

	var existingID int64
	var queryErr error
	if scan.ProjectID == nil {
		queryErr = tx.QueryRowContext(ctx,
			`SELECT id FROM provider_plugin_layer_scans WHERE provider_definition_id = ? AND project_id IS NULL AND settings_layer = ?`,
			scan.ProviderDefinitionID, string(scan.SettingsLayer),
		).Scan(&existingID)
	} else {
		queryErr = tx.QueryRowContext(ctx,
			`SELECT id FROM provider_plugin_layer_scans WHERE provider_definition_id = ? AND project_id = ? AND settings_layer = ?`,
			scan.ProviderDefinitionID, *scan.ProjectID, string(scan.SettingsLayer),
		).Scan(&existingID)
	}

	if queryErr == sql.ErrNoRows {
		var res sql.Result
		var insertErr error
		if scan.ProjectID == nil {
			res, insertErr = tx.ExecContext(ctx,
				`INSERT INTO provider_plugin_layer_scans (provider_definition_id, project_id, settings_layer, scan_status, settings_file_path, last_scanned_at, source_operation_id, scan_warnings)
				 VALUES (?, NULL, ?, ?, ?, ?, ?, ?)`,
				scan.ProviderDefinitionID, string(scan.SettingsLayer), string(scan.ScanStatus), scan.SettingsFilePath, scannedAt, opID, warningsJSON)
		} else {
			res, insertErr = tx.ExecContext(ctx,
				`INSERT INTO provider_plugin_layer_scans (provider_definition_id, project_id, settings_layer, scan_status, settings_file_path, last_scanned_at, source_operation_id, scan_warnings)
				 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
				scan.ProviderDefinitionID, *scan.ProjectID, string(scan.SettingsLayer), string(scan.ScanStatus), scan.SettingsFilePath, scannedAt, opID, warningsJSON)
		}
		if insertErr != nil {
			return 0, insertErr
		}
		return res.LastInsertId()
	}
	if queryErr != nil {
		return 0, queryErr
	}

	_, err = tx.ExecContext(ctx,
		`UPDATE provider_plugin_layer_scans SET scan_status = ?, settings_file_path = ?, last_scanned_at = ?, source_operation_id = ?, scan_warnings = ? WHERE id = ?`,
		string(scan.ScanStatus), scan.SettingsFilePath, scannedAt, opID, warningsJSON, existingID)
	return existingID, err
}

func marshalWarnings(warnings []string) (string, error) {
	if len(warnings) == 0 {
		return "[]", nil
	}
	b, err := json.Marshal(warnings)
	if err != nil {
		return "[]", err
	}
	return string(b), nil
}

func scanPluginLayerRows(rows *sql.Rows) ([]domain.PluginLayerScan, error) {
	var result []domain.PluginLayerScan
	for rows.Next() {
		var s domain.PluginLayerScan
		var projID sql.NullInt64
		var opID sql.NullInt64
		var scannedAt string
		var warningsJSON string
		if err := rows.Scan(
			&s.ID, &s.ProviderDefinitionID, &projID,
			(*string)(&s.SettingsLayer), (*string)(&s.ScanStatus),
			&s.SettingsFilePath, &scannedAt, &opID, &warningsJSON,
		); err != nil {
			return nil, err
		}
		if projID.Valid {
			s.ProjectID = &projID.Int64
		}
		if opID.Valid {
			s.SourceOperationID = &opID.Int64
		}
		t, _ := time.Parse(time.RFC3339, scannedAt)
		s.LastScannedAt = t
		if warningsJSON != "" && warningsJSON != "[]" {
			_ = json.Unmarshal([]byte(warningsJSON), &s.Warnings)
		}
		result = append(result, s)
	}
	return result, rows.Err()
}
