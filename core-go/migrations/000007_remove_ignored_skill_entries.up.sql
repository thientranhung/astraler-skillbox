-- Remove filesystem metadata entries that older scans may have persisted as skills.
DELETE FROM warnings
 WHERE scope_type = 'global_install'
   AND scope_id IN (
       SELECT id
        FROM global_installs
        WHERE skill_name = '.DS_Store'
           OR global_skill_path LIKE '%/.DS_Store'
           OR skill_id IN (
               SELECT id
                 FROM skills
                WHERE name = '.DS_Store'
                   OR relative_path = '.DS_Store'
                   OR absolute_path LIKE '%/.DS_Store'
           )
   );

DELETE FROM warnings
 WHERE scope_type = 'install'
   AND scope_id IN (
       SELECT id
        FROM installs
        WHERE skill_name = '.DS_Store'
           OR project_skill_path LIKE '%/.DS_Store'
           OR skill_id IN (
               SELECT id
                 FROM skills
                WHERE name = '.DS_Store'
                   OR relative_path = '.DS_Store'
                   OR absolute_path LIKE '%/.DS_Store'
           )
   );

DELETE FROM warnings
 WHERE scope_type = 'skill'
   AND scope_id IN (
       SELECT id
         FROM skills
        WHERE name = '.DS_Store'
           OR relative_path = '.DS_Store'
           OR absolute_path LIKE '%/.DS_Store'
   );

DELETE FROM global_installs
 WHERE skill_name = '.DS_Store'
    OR global_skill_path LIKE '%/.DS_Store'
    OR skill_id IN (
        SELECT id
          FROM skills
         WHERE name = '.DS_Store'
            OR relative_path = '.DS_Store'
            OR absolute_path LIKE '%/.DS_Store'
    );

DELETE FROM installs
 WHERE skill_name = '.DS_Store'
    OR project_skill_path LIKE '%/.DS_Store'
    OR skill_id IN (
        SELECT id
          FROM skills
         WHERE name = '.DS_Store'
            OR relative_path = '.DS_Store'
            OR absolute_path LIKE '%/.DS_Store'
    );

DELETE FROM skills
 WHERE name = '.DS_Store'
    OR relative_path = '.DS_Store'
    OR absolute_path LIKE '%/.DS_Store';

UPDATE app_settings
   SET database_version = 7, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE id = 1;
