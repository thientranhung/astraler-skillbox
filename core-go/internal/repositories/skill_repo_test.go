package repositories

import (
	"context"
	"testing"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

func seedHost(t *testing.T, repo *SkillHostFolderRepo) int64 {
	t.Helper()
	id, _, err := repo.UpsertAndActivate(context.Background(), "host", "/tmp/host", "/tmp/host/.agents/skills")
	if err != nil {
		t.Fatalf("seedHost: %v", err)
	}
	return id
}

func TestSkillRepo_UpsertMany_And_List(t *testing.T) {
	db := NewTestDB(t)
	skillRepo := NewSkillRepo(db)
	hostRepo := NewSkillHostFolderRepo(db)
	ctx := context.Background()

	hostID := seedHost(t, hostRepo)

	skills := []domain.Skill{
		{Name: "skill-a", RelativePath: ".agents/skills/skill-a", AbsolutePath: "/tmp/host/.agents/skills/skill-a", Status: domain.SkillStatusAvailable},
		{Name: "skill-b", RelativePath: ".agents/skills/skill-b", AbsolutePath: "/tmp/host/.agents/skills/skill-b", Status: domain.SkillStatusAvailable},
	}
	if err := skillRepo.UpsertMany(ctx, hostID, skills); err != nil {
		t.Fatalf("UpsertMany: %v", err)
	}

	listed, err := skillRepo.ListByHost(ctx, hostID)
	if err != nil {
		t.Fatalf("ListByHost: %v", err)
	}
	if len(listed) != 2 {
		t.Fatalf("expected 2 skills, got %d", len(listed))
	}
}

func TestSkillRepo_UpsertMany_Updates(t *testing.T) {
	db := NewTestDB(t)
	skillRepo := NewSkillRepo(db)
	hostRepo := NewSkillHostFolderRepo(db)
	ctx := context.Background()
	hostID := seedHost(t, hostRepo)

	skill := domain.Skill{Name: "skill-a", RelativePath: ".agents/skills/skill-a", AbsolutePath: "/tmp/host/.agents/skills/skill-a", Status: domain.SkillStatusAvailable}
	_ = skillRepo.UpsertMany(ctx, hostID, []domain.Skill{skill})

	skill.Status = domain.SkillStatusLocalModified
	_ = skillRepo.UpsertMany(ctx, hostID, []domain.Skill{skill})

	listed, _ := skillRepo.ListByHost(ctx, hostID)
	if len(listed) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(listed))
	}
	if listed[0].Status != domain.SkillStatusLocalModified {
		t.Errorf("status not updated: %q", listed[0].Status)
	}
}

func TestSkillRepo_CountByHost(t *testing.T) {
	db := NewTestDB(t)
	skillRepo := NewSkillRepo(db)
	hostRepo := NewSkillHostFolderRepo(db)
	ctx := context.Background()

	host1ID := seedHost(t, hostRepo)
	host2ID, _, err := hostRepo.UpsertAndActivate(ctx, "host2", "/tmp/host2", "/tmp/host2/.agents/skills")
	if err != nil {
		t.Fatalf("seedHost2: %v", err)
	}

	skills1 := []domain.Skill{
		{Name: "a", RelativePath: ".agents/skills/a", AbsolutePath: "/tmp/host/.agents/skills/a", Status: domain.SkillStatusAvailable},
		{Name: "b", RelativePath: ".agents/skills/b", AbsolutePath: "/tmp/host/.agents/skills/b", Status: domain.SkillStatusAvailable},
	}
	if err := skillRepo.UpsertMany(ctx, host1ID, skills1); err != nil {
		t.Fatalf("UpsertMany host1: %v", err)
	}

	skills2 := []domain.Skill{
		{Name: "x", RelativePath: ".agents/skills/x", AbsolutePath: "/tmp/host2/.agents/skills/x", Status: domain.SkillStatusAvailable},
		{Name: "y", RelativePath: ".agents/skills/y", AbsolutePath: "/tmp/host2/.agents/skills/y", Status: domain.SkillStatusAvailable},
		{Name: "z", RelativePath: ".agents/skills/z", AbsolutePath: "/tmp/host2/.agents/skills/z", Status: domain.SkillStatusAvailable},
	}
	if err := skillRepo.UpsertMany(ctx, host2ID, skills2); err != nil {
		t.Fatalf("UpsertMany host2: %v", err)
	}

	count1, err := skillRepo.CountByHost(ctx, host1ID)
	if err != nil {
		t.Fatalf("CountByHost host1: %v", err)
	}
	if count1 != 2 {
		t.Errorf("host1 count: got %d want 2", count1)
	}

	count2, err := skillRepo.CountByHost(ctx, host2ID)
	if err != nil {
		t.Fatalf("CountByHost host2: %v", err)
	}
	if count2 != 3 {
		t.Errorf("host2 count: got %d want 3", count2)
	}
}

func TestSkillRepo_CountProjectsPerSkillByHost(t *testing.T) {
	db := NewTestDB(t)
	skillRepo := NewSkillRepo(db)
	hostRepo := NewSkillHostFolderRepo(db)
	projRepo := NewProjectRepo(db)
	ctx := context.Background()

	hostID := seedHost(t, hostRepo)

	// Seed 2 skills.
	_ = skillRepo.UpsertMany(ctx, hostID, []domain.Skill{
		{Name: "skill1", RelativePath: ".agents/skills/skill1", AbsolutePath: "/tmp/host/.agents/skills/skill1", Status: domain.SkillStatusAvailable},
		{Name: "skill2", RelativePath: ".agents/skills/skill2", AbsolutePath: "/tmp/host/.agents/skills/skill2", Status: domain.SkillStatusAvailable},
	})
	skills, _ := skillRepo.ListByHost(ctx, hostID)
	var skill1ID, skill2ID int64
	for _, s := range skills {
		if s.Name == "skill1" {
			skill1ID = s.ID
		} else {
			skill2ID = s.ID
		}
	}

	defID := getGenericAgentsDefID(t, db)

	// Active project A: two installs of skill1 (same skill, different project_providers would mean different pp rows
	// but here same provider — still counts project once).
	pidA := seedProject(t, projRepo, "proj-a", "/tmp/proj-a")
	ppA := seedProjectProvider(t, db, pidA, defID)

	for i, path := range []string{"/tmp/proj-a/.agents/skills/skill1", "/tmp/proj-a/.agents/skills/skill1-alias"} {
		_ = i
		_, err := db.ExecContext(ctx,
			`INSERT INTO installs (project_provider_id, skill_id, skill_name, install_mode, install_status, project_skill_path)
			 VALUES (?, ?, 'skill1', 'symlink', 'current', ?)`, ppA, skill1ID, path)
		if err != nil {
			t.Fatalf("insert install A: %v", err)
		}
	}

	// Removed project B: install of skill1 → must be excluded.
	pidB := seedProject(t, projRepo, "proj-b", "/tmp/proj-b")
	_, err := db.ExecContext(ctx, `UPDATE projects SET status='removed' WHERE id=?`, pidB)
	if err != nil {
		t.Fatalf("mark removed: %v", err)
	}
	ppB := seedProjectProvider(t, db, pidB, defID)
	_, err = db.ExecContext(ctx,
		`INSERT INTO installs (project_provider_id, skill_id, skill_name, install_mode, install_status, project_skill_path)
		 VALUES (?, ?, 'skill1', 'symlink', 'current', '/tmp/proj-b/.agents/skills/skill1')`, ppB, skill1ID)
	if err != nil {
		t.Fatalf("insert install B: %v", err)
	}

	// skill2 has no installs.

	counts, err := skillRepo.CountProjectsPerSkillByHost(ctx, hostID)
	if err != nil {
		t.Fatalf("CountProjectsPerSkillByHost: %v", err)
	}

	if got := counts[skill1ID]; got != 1 {
		t.Errorf("skill1 count: got %d want 1 (two installs in same project, removed excluded)", got)
	}
	if got := counts[skill2ID]; got != 0 {
		t.Errorf("skill2 count: got %d want 0", got)
	}
}

func TestSkillRepo_GetByID(t *testing.T) {
	db := NewTestDB(t)
	skillRepo := NewSkillRepo(db)
	hostRepo := NewSkillHostFolderRepo(db)
	ctx := context.Background()
	hostID := seedHost(t, hostRepo)

	_ = skillRepo.UpsertMany(ctx, hostID, []domain.Skill{
		{Name: "skill-x", RelativePath: ".agents/skills/skill-x", AbsolutePath: "/tmp/host/.agents/skills/skill-x", Status: domain.SkillStatusAvailable},
	})
	listed, _ := skillRepo.ListByHost(ctx, hostID)
	id := listed[0].ID

	skill, err := skillRepo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if skill == nil {
		t.Fatal("expected skill, got nil")
	}
	if skill.Name != "skill-x" {
		t.Errorf("name: got %q want skill-x", skill.Name)
	}

	missing, err := skillRepo.GetByID(ctx, 99999)
	if err != nil {
		t.Fatalf("GetByID unknown: %v", err)
	}
	if missing != nil {
		t.Errorf("expected nil for unknown id, got %+v", missing)
	}
}

func TestSkillRepo_ProjectsUsingSkill(t *testing.T) {
	db := NewTestDB(t)
	skillRepo := NewSkillRepo(db)
	hostRepo := NewSkillHostFolderRepo(db)
	projRepo := NewProjectRepo(db)
	ctx := context.Background()

	hostID := seedHost(t, hostRepo)
	_ = skillRepo.UpsertMany(ctx, hostID, []domain.Skill{
		{Name: "skill1", RelativePath: ".agents/skills/skill1", AbsolutePath: "/tmp/host/.agents/skills/skill1", Status: domain.SkillStatusAvailable},
	})
	skills, _ := skillRepo.ListByHost(ctx, hostID)
	skill1ID := skills[0].ID

	defID := getGenericAgentsDefID(t, db)

	// Active project A — one install.
	pidA := seedProject(t, projRepo, "proj-alpha", "/tmp/proj-alpha")
	ppA := seedProjectProvider(t, db, pidA, defID)
	_, err := db.ExecContext(ctx,
		`INSERT INTO installs (project_provider_id, skill_id, skill_name, install_mode, install_status, project_skill_path)
		 VALUES (?, ?, 'skill1', 'symlink', 'current', '/tmp/proj-alpha/.agents/skills/skill1')`, ppA, skill1ID)
	if err != nil {
		t.Fatalf("insert install: %v", err)
	}

	// Removed project B — must be excluded.
	pidB := seedProject(t, projRepo, "proj-beta", "/tmp/proj-beta")
	_, err = db.ExecContext(ctx, `UPDATE projects SET status='removed' WHERE id=?`, pidB)
	if err != nil {
		t.Fatalf("mark removed: %v", err)
	}
	ppB := seedProjectProvider(t, db, pidB, defID)
	_, err = db.ExecContext(ctx,
		`INSERT INTO installs (project_provider_id, skill_id, skill_name, install_mode, install_status, project_skill_path)
		 VALUES (?, ?, 'skill1', 'symlink', 'current', '/tmp/proj-beta/.agents/skills/skill1')`, ppB, skill1ID)
	if err != nil {
		t.Fatalf("insert removed install: %v", err)
	}

	rows, err := skillRepo.ProjectsUsingSkill(ctx, skill1ID)
	if err != nil {
		t.Fatalf("ProjectsUsingSkill: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row (removed excluded), got %d", len(rows))
	}
	row := rows[0]
	if row.ProjectName != "proj-alpha" {
		t.Errorf("ProjectName: got %q want proj-alpha", row.ProjectName)
	}
	if row.ProviderKey != "generic_agents" {
		t.Errorf("ProviderKey: got %q want generic_agents", row.ProviderKey)
	}
	if row.Mode != "symlink" {
		t.Errorf("Mode: got %q want symlink", row.Mode)
	}
	if row.Status != "current" {
		t.Errorf("Status: got %q want current", row.Status)
	}
	if row.ProjectSkillPath != "/tmp/proj-alpha/.agents/skills/skill1" {
		t.Errorf("ProjectSkillPath: got %q", row.ProjectSkillPath)
	}
}

func TestSkillRepo_MarkMissing(t *testing.T) {
	db := NewTestDB(t)
	skillRepo := NewSkillRepo(db)
	hostRepo := NewSkillHostFolderRepo(db)
	ctx := context.Background()
	hostID := seedHost(t, hostRepo)

	skills := []domain.Skill{
		{Name: "a", RelativePath: ".agents/skills/a", AbsolutePath: "/tmp/a", Status: domain.SkillStatusAvailable},
		{Name: "b", RelativePath: ".agents/skills/b", AbsolutePath: "/tmp/b", Status: domain.SkillStatusAvailable},
	}
	_ = skillRepo.UpsertMany(ctx, hostID, skills)

	listed, _ := skillRepo.ListByHost(ctx, hostID)
	// Keep only the first skill's ID as "present".
	if err := skillRepo.MarkMissing(ctx, hostID, []int64{listed[0].ID}); err != nil {
		t.Fatalf("MarkMissing: %v", err)
	}

	after, _ := skillRepo.ListByHost(ctx, hostID)
	for _, s := range after {
		if s.Name == "a" && s.Status != domain.SkillStatusAvailable {
			t.Errorf("skill-a should remain available, got %q", s.Status)
		}
		if s.Name == "b" && s.Status != domain.SkillStatusMissing {
			t.Errorf("skill-b should be missing, got %q", s.Status)
		}
	}
}
