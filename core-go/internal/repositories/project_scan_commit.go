package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

// ProjectScanRepo provides an atomic write path for project scan results.
type ProjectScanRepo struct {
	db *sql.DB
}

func NewProjectScanRepo(db *sql.DB) *ProjectScanRepo {
	return &ProjectScanRepo{db: db}
}

// ProviderScanResult carries the detected state of one provider for CommitProjectScan.
// Warnings must have ScopeType=WarningScopeProjectProvider; scope_id is filled in by commit.
type ProviderScanResult struct {
	ProviderDefinitionID int64
	DetectedPath         *string
	SkillsPath           *string
	DetectionStatus      domain.DetectionStatus
	Installs             []InstallScanResult
	Warnings             []domain.Warning
}

// InstallScanResult carries one observed filesystem entry for CommitProjectScan.
// Warning (if set) must have ScopeType=WarningScopeInstall; scope_id is filled in by commit.
type InstallScanResult struct {
	SkillID                   *int64
	SkillName                 string
	InstallMode               domain.InstallMode
	InstallStatus             domain.InstallStatus
	ProjectSkillPath          string
	SourceSkillPath           *string
	SymlinkTargetPath         *string
	InstalledFromHostFolderID *int64
	Warning                   *domain.Warning
}

// CommitProjectScan persists a full project scan atomically:
//  1. Clears active warnings across project/project_provider/install scopes.
//  2. Upserts project_providers; marks absent providers missing.
//  3. For each provider: upserts installs; marks absent installs missing (no hard delete);
//     inserts provider and install-scoped warnings with the correct scope ids.
//  4. Inserts project-scoped warnings.
//  5. Updates projects.last_scanned_at and status=active.
func (r *ProjectScanRepo) CommitProjectScan(
	ctx context.Context,
	projectID int64,
	providers []ProviderScanResult,
	projectWarnings []domain.Warning,
	now time.Time,
) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin project scan tx: %w", err)
	}
	defer tx.Rollback()

	nowStr := now.UTC().Format(time.RFC3339)

	// 1. Clear all active warnings for this project across all scopes.
	if err := clearProjectWarnings(ctx, tx, projectID, nowStr); err != nil {
		return fmt.Errorf("clear warnings: %w", err)
	}

	// 2. Upsert providers and collect their db ids; mark absent ones missing.
	seenDefIDs := make([]int64, 0, len(providers))
	providerDBIDs := make([]int64, len(providers))

	for i, p := range providers {
		ppID, err := upsertProjectProvider(ctx, tx, projectID, p, nowStr)
		if err != nil {
			return fmt.Errorf("upsert provider %d: %w", p.ProviderDefinitionID, err)
		}
		providerDBIDs[i] = ppID
		seenDefIDs = append(seenDefIDs, p.ProviderDefinitionID)
	}

	if err := markAbsentProvidersMissing(ctx, tx, projectID, seenDefIDs, nowStr); err != nil {
		return fmt.Errorf("mark absent providers: %w", err)
	}

	// Cascade: installs under absent providers also become missing.
	if err := cascadeInstallsMissingForAbsentProviders(ctx, tx, projectID, seenDefIDs, nowStr); err != nil {
		return fmt.Errorf("cascade installs missing for absent providers: %w", err)
	}

	// 3. For each provider, reconcile installs and insert warnings.
	for i, p := range providers {
		ppID := providerDBIDs[i]

		seenPaths := make([]string, 0, len(p.Installs))
		for _, inst := range p.Installs {
			installID, err := upsertInstall(ctx, tx, ppID, inst, nowStr)
			if err != nil {
				return fmt.Errorf("upsert install %q: %w", inst.ProjectSkillPath, err)
			}
			seenPaths = append(seenPaths, inst.ProjectSkillPath)

			if inst.Warning != nil {
				w := *inst.Warning
				w.ScopeID = &installID
				if err := insertWarningTx(ctx, tx, w); err != nil {
					return fmt.Errorf("insert install warning: %w", err)
				}
			}
		}

		if err := markAbsentInstallsMissing(ctx, tx, ppID, seenPaths, nowStr); err != nil {
			return fmt.Errorf("mark absent installs for provider %d: %w", ppID, err)
		}

		for _, w := range p.Warnings {
			w.ScopeID = &ppID
			if err := insertWarningTx(ctx, tx, w); err != nil {
				return fmt.Errorf("insert provider warning: %w", err)
			}
		}
	}

	// 4. Insert project-scoped warnings.
	for _, w := range projectWarnings {
		id := int64(projectID)
		w.ScopeID = &id
		if err := insertWarningTx(ctx, tx, w); err != nil {
			return fmt.Errorf("insert project warning: %w", err)
		}
	}

	// 5. Update project status and last_scanned_at.
	if _, err := tx.ExecContext(ctx,
		`UPDATE projects SET status='active', last_scanned_at=?,
		 updated_at=strftime('%Y-%m-%dT%H:%M:%SZ','now') WHERE id=?`,
		nowStr, projectID,
	); err != nil {
		return fmt.Errorf("update project: %w", err)
	}

	return tx.Commit()
}

// CommitProjectTerminal handles terminal scan states (missing/unreadable):
// clears only project-scoped warnings, optionally inserts a new project warning,
// and updates projects.status. Does NOT touch project_providers or installs.
func (r *ProjectScanRepo) CommitProjectTerminal(
	ctx context.Context,
	projectID int64,
	status domain.ProjectStatus,
	warning *domain.Warning,
	now time.Time,
) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin terminal tx: %w", err)
	}
	defer tx.Rollback()

	nowStr := now.UTC().Format(time.RFC3339)

	// Clear project-scoped warnings only.
	if _, err := tx.ExecContext(ctx,
		`UPDATE warnings SET is_resolved=1,
		 resolved_at=strftime('%Y-%m-%dT%H:%M:%SZ','now'),
		 updated_at=strftime('%Y-%m-%dT%H:%M:%SZ','now')
		 WHERE scope_type='project' AND scope_id=? AND is_resolved=0`,
		projectID,
	); err != nil {
		return fmt.Errorf("clear project warnings: %w", err)
	}

	if warning != nil {
		w := *warning
		id := int64(projectID)
		w.ScopeID = &id
		if err := insertWarningTx(ctx, tx, w); err != nil {
			return fmt.Errorf("insert terminal warning: %w", err)
		}
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE projects SET status=?, last_scanned_at=?,
		 updated_at=strftime('%Y-%m-%dT%H:%M:%SZ','now') WHERE id=?`,
		string(status), nowStr, projectID,
	); err != nil {
		return fmt.Errorf("update project status: %w", err)
	}

	return tx.Commit()
}

// --- helpers ---

func clearProjectWarnings(ctx context.Context, tx *sql.Tx, projectID int64, nowStr string) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE warnings SET is_resolved=1,
		 resolved_at=strftime('%Y-%m-%dT%H:%M:%SZ','now'),
		 updated_at=strftime('%Y-%m-%dT%H:%M:%SZ','now')
		WHERE is_resolved=0 AND (
			(scope_type='project' AND scope_id=?)
			OR (scope_type='project_provider' AND scope_id IN (
				SELECT id FROM project_providers WHERE project_id=?
			))
			OR (scope_type='install' AND scope_id IN (
				SELECT i.id FROM installs i
				JOIN project_providers pp ON pp.id=i.project_provider_id
				WHERE pp.project_id=?
			))
		)`, projectID, projectID, projectID)
	return err
}

func upsertProjectProvider(ctx context.Context, tx *sql.Tx, projectID int64, p ProviderScanResult, nowStr string) (int64, error) {
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO project_providers
		  (project_id, provider_definition_id, detected_path, skills_path, detection_status, last_scanned_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(project_id, provider_definition_id) DO UPDATE SET
		  detected_path=excluded.detected_path,
		  skills_path=excluded.skills_path,
		  detection_status=excluded.detection_status,
		  last_scanned_at=excluded.last_scanned_at,
		  updated_at=strftime('%Y-%m-%dT%H:%M:%SZ','now')`,
		projectID, p.ProviderDefinitionID,
		nullableStr(p.DetectedPath), nullableStr(p.SkillsPath),
		string(p.DetectionStatus), nowStr,
	); err != nil {
		return 0, err
	}

	var id int64
	err := tx.QueryRowContext(ctx,
		`SELECT id FROM project_providers WHERE project_id=? AND provider_definition_id=?`,
		projectID, p.ProviderDefinitionID).Scan(&id)
	return id, err
}

func markAbsentProvidersMissing(ctx context.Context, tx *sql.Tx, projectID int64, seenDefIDs []int64, nowStr string) error {
	if len(seenDefIDs) == 0 {
		_, err := tx.ExecContext(ctx,
			`UPDATE project_providers SET detection_status='missing',
			 detected_path=NULL,
			 skills_path=NULL,
			 last_scanned_at=?,
			 updated_at=strftime('%Y-%m-%dT%H:%M:%SZ','now')
			 WHERE project_id=?`, nowStr, projectID)
		return err
	}
	ph := strings.Repeat("?,", len(seenDefIDs))
	ph = ph[:len(ph)-1]
	args := make([]interface{}, 0, 1+len(seenDefIDs))
	args = append(args, projectID)
	for _, id := range seenDefIDs {
		args = append(args, id)
	}
	_, err := tx.ExecContext(ctx,
		"UPDATE project_providers SET detection_status='missing',"+
			" detected_path=NULL,"+
			" skills_path=NULL,"+
			" last_scanned_at=?,"+
			" updated_at=strftime('%Y-%m-%dT%H:%M:%SZ','now')"+
			" WHERE project_id=? AND provider_definition_id NOT IN ("+ph+")",
		append([]interface{}{nowStr}, args...)...)
	return err
}

// cascadeInstallsMissingForAbsentProviders marks all installs as missing for
// providers that were not seen in the current scan (i.e., those just marked missing).
func cascadeInstallsMissingForAbsentProviders(ctx context.Context, tx *sql.Tx, projectID int64, seenDefIDs []int64, nowStr string) error {
	if len(seenDefIDs) == 0 {
		_, err := tx.ExecContext(ctx, `
			UPDATE installs SET install_status='missing',
			 updated_at=strftime('%Y-%m-%dT%H:%M:%SZ','now')
			 WHERE project_provider_id IN (
			     SELECT id FROM project_providers WHERE project_id=?
			 )`, projectID)
		return err
	}
	ph := strings.Repeat("?,", len(seenDefIDs))
	ph = ph[:len(ph)-1]
	args := make([]interface{}, 0, 1+len(seenDefIDs))
	args = append(args, projectID)
	for _, id := range seenDefIDs {
		args = append(args, id)
	}
	_, err := tx.ExecContext(ctx,
		"UPDATE installs SET install_status='missing',"+
			" updated_at=strftime('%Y-%m-%dT%H:%M:%SZ','now')"+
			" WHERE project_provider_id IN ("+
			"     SELECT id FROM project_providers WHERE project_id=? AND provider_definition_id NOT IN ("+ph+")"+
			")",
		args...)
	return err
}

func upsertInstall(ctx context.Context, tx *sql.Tx, ppID int64, inst InstallScanResult, nowStr string) (int64, error) {
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO installs
		  (project_provider_id, skill_id, skill_name, install_mode, install_status,
		   project_skill_path, source_skill_path, symlink_target_path,
		   installed_from_host_folder_id, last_scanned_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(project_provider_id, project_skill_path) DO UPDATE SET
		  skill_id=excluded.skill_id,
		  skill_name=excluded.skill_name,
		  install_mode=excluded.install_mode,
		  install_status=excluded.install_status,
		  source_skill_path=excluded.source_skill_path,
		  symlink_target_path=excluded.symlink_target_path,
		  installed_from_host_folder_id=excluded.installed_from_host_folder_id,
		  last_scanned_at=excluded.last_scanned_at,
		  updated_at=strftime('%Y-%m-%dT%H:%M:%SZ','now')`,
		ppID, ptrToSQL(inst.SkillID), inst.SkillName,
		string(inst.InstallMode), string(inst.InstallStatus),
		inst.ProjectSkillPath,
		nullableStr(inst.SourceSkillPath), nullableStr(inst.SymlinkTargetPath),
		ptrToSQL(inst.InstalledFromHostFolderID), nowStr,
	); err != nil {
		return 0, err
	}

	var id int64
	err := tx.QueryRowContext(ctx,
		`SELECT id FROM installs WHERE project_provider_id=? AND project_skill_path=?`,
		ppID, inst.ProjectSkillPath).Scan(&id)
	return id, err
}

func markAbsentInstallsMissing(ctx context.Context, tx *sql.Tx, ppID int64, seenPaths []string, nowStr string) error {
	if len(seenPaths) == 0 {
		_, err := tx.ExecContext(ctx,
			`UPDATE installs SET install_status='missing',
			 updated_at=strftime('%Y-%m-%dT%H:%M:%SZ','now')
			 WHERE project_provider_id=?`, ppID)
		return err
	}
	ph := strings.Repeat("?,", len(seenPaths))
	ph = ph[:len(ph)-1]
	args := make([]interface{}, 0, 1+len(seenPaths))
	args = append(args, ppID)
	for _, p := range seenPaths {
		args = append(args, p)
	}
	_, err := tx.ExecContext(ctx,
		"UPDATE installs SET install_status='missing',"+
			" updated_at=strftime('%Y-%m-%dT%H:%M:%SZ','now')"+
			" WHERE project_provider_id=? AND project_skill_path NOT IN ("+ph+")",
		args...)
	return err
}

func insertWarningTx(ctx context.Context, tx *sql.Tx, w domain.Warning) error {
	_, err := tx.ExecContext(ctx,
		`INSERT INTO warnings
		  (scope_type, scope_id, severity, code, message, action_key, source_operation_id, is_resolved)
		 VALUES (?, ?, ?, ?, ?, ?, ?, 0)`,
		string(w.ScopeType), ptrToSQL(w.ScopeID),
		string(w.Severity), w.Code, w.Message,
		nullableStrFromPtr(w.ActionKey), ptrToSQL(w.SourceOperationID),
	)
	return err
}

func nullableStr(s *string) interface{} {
	if s == nil {
		return nil
	}
	return *s
}

func nullableStrFromPtr(s *string) interface{} {
	if s == nil {
		return nil
	}
	return *s
}
