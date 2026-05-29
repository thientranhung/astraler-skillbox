package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

type UpdateCheckCacheRepo struct {
	db *sql.DB
}

func NewUpdateCheckCacheRepo(db *sql.DB) *UpdateCheckCacheRepo {
	return &UpdateCheckCacheRepo{db: db}
}

func (r *UpdateCheckCacheRepo) Upsert(ctx context.Context, e domain.UpdateCheckCacheEntry) error {
	var ua *int
	if e.UpdateAvailable != nil {
		v := 0
		if *e.UpdateAvailable {
			v = 1
		}
		ua = &v
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO plugin_update_check_cache
		    (provider_key, plugin_name, marketplace_name, source_url, source_ref,
		     installed_sha, installed_version, remote_sha, remote_latest_tag,
		     update_available, checked_at, error)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(provider_key, plugin_name, marketplace_name) DO UPDATE SET
		    source_url=excluded.source_url,
		    source_ref=excluded.source_ref,
		    installed_sha=excluded.installed_sha,
		    installed_version=excluded.installed_version,
		    remote_sha=excluded.remote_sha,
		    remote_latest_tag=excluded.remote_latest_tag,
		    update_available=excluded.update_available,
		    checked_at=excluded.checked_at,
		    error=excluded.error`,
		e.ProviderKey, e.PluginName, e.MarketplaceName,
		e.SourceURL, e.SourceRef,
		e.InstalledSHA, e.InstalledVersion,
		e.RemoteSHA, e.RemoteLatestTag,
		ua, e.CheckedAt.UTC().Format(time.RFC3339), nullableString(e.Error))
	if err != nil {
		return fmt.Errorf("update_check_cache upsert: %w", err)
	}
	return nil
}

func (r *UpdateCheckCacheRepo) GetAll(ctx context.Context) ([]domain.UpdateCheckCacheEntry, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT provider_key, plugin_name, marketplace_name, source_url, source_ref,
		       installed_sha, installed_version, remote_sha, remote_latest_tag,
		       update_available, checked_at, error
		  FROM plugin_update_check_cache`)
	if err != nil {
		return nil, fmt.Errorf("update_check_cache list: %w", err)
	}
	defer rows.Close()

	var out []domain.UpdateCheckCacheEntry
	for rows.Next() {
		e, err := scanCacheEntry(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *UpdateCheckCacheRepo) GetByPlugin(ctx context.Context, providerKey, pluginName, marketplaceName string) (*domain.UpdateCheckCacheEntry, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT provider_key, plugin_name, marketplace_name, source_url, source_ref,
		       installed_sha, installed_version, remote_sha, remote_latest_tag,
		       update_available, checked_at, error
		  FROM plugin_update_check_cache
		 WHERE provider_key = ? AND plugin_name = ? AND marketplace_name = ?`,
		providerKey, pluginName, marketplaceName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, nil
	}
	e, err := scanCacheEntry(rows)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func scanCacheEntry(rows *sql.Rows) (domain.UpdateCheckCacheEntry, error) {
	var e domain.UpdateCheckCacheEntry
	var ua sql.NullInt64
	var checkedAt string
	var errStr sql.NullString
	if err := rows.Scan(
		&e.ProviderKey, &e.PluginName, &e.MarketplaceName,
		&e.SourceURL, &e.SourceRef,
		&e.InstalledSHA, &e.InstalledVersion,
		&e.RemoteSHA, &e.RemoteLatestTag,
		&ua, &checkedAt, &errStr,
	); err != nil {
		return e, err
	}
	if ua.Valid {
		b := ua.Int64 == 1
		e.UpdateAvailable = &b
	}
	e.CheckedAt, _ = time.Parse(time.RFC3339, checkedAt)
	if errStr.Valid {
		e.Error = errStr.String
	}
	return e, nil
}

func nullableString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
