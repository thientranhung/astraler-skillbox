-- 000018_add_pi_provider.down.sql
-- Remove pi provider.

DELETE FROM provider_path_candidates
 WHERE provider_definition_id = (SELECT id FROM provider_definitions WHERE key = 'pi');

DELETE FROM provider_path_overrides
 WHERE provider_definition_id = (SELECT id FROM provider_definitions WHERE key = 'pi');

DELETE FROM provider_user_settings
 WHERE provider_definition_id = (SELECT id FROM provider_definitions WHERE key = 'pi');

DELETE FROM provider_definitions
 WHERE key = 'pi';

UPDATE app_settings
   SET database_version = 17, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE id = 1;
