package logic

import (
	"archive/zip"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	aiagent "github.com/gly-hub/ai-dandelion/proto/ai-agent"
)

func TestSkillLogicImportListAndDeleteIsUserScoped(t *testing.T) {
	storageRoot := t.TempDir()
	logic := NewSkillLogic(storageRoot)
	ctx := context.Background()
	zipData := buildSkillZip(t, map[string]string{
		"demo/SKILL.md":          "---\nname: demo-skill\ndescription: 测试技能\n---\n",
		"__MACOSX/demo/SKILL.md": "",
	})

	skill, err := logic.ImportSkillPackage(ctx, &aiagent.ImportSkillPackageReq{
		UserId:   "user-a",
		FileName: "demo.zip",
		Data:     zipData,
	})
	if err != nil {
		t.Fatalf("import skill: %v", err)
	}
	if skill.GetId() != "demo-skill" || skill.GetDescription() != "测试技能" {
		t.Fatalf("unexpected imported skill: %#v", skill)
	}
	skillDir := filepath.Join(storageRoot, "user-a", ".claude", "skills", "demo-skill")
	if _, err := os.Stat(filepath.Join(skillDir, "SKILL.md")); err != nil {
		t.Fatalf("expected skill files under claude skill dir: %v", err)
	}
	if _, err := os.Stat(filepath.Join(skillDir, "demo")); !os.IsNotExist(err) {
		t.Fatalf("expected import to strip package root dir, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(skillDir, "__MACOSX")); !os.IsNotExist(err) {
		t.Fatalf("expected import to skip macOS metadata dir, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(skillDir, skillManifestFile)); err != nil {
		t.Fatalf("expected skill manifest under claude skill dir: %v", err)
	}

	userASkills, err := logic.ListSkills(ctx, &aiagent.ListSkillsReq{UserId: "user-a"})
	if err != nil {
		t.Fatalf("list user a skills: %v", err)
	}
	if len(userASkills) != 1 || userASkills[0].GetId() != "demo-skill" {
		t.Fatalf("unexpected user a skills: %#v", userASkills)
	}

	userBSkills, err := logic.ListSkills(ctx, &aiagent.ListSkillsReq{UserId: "user-b"})
	if err != nil {
		t.Fatalf("list user b skills: %v", err)
	}
	if len(userBSkills) != 0 {
		t.Fatalf("expected user b to be isolated, got %#v", userBSkills)
	}

	deletedID, err := logic.DeleteSkill(ctx, &aiagent.DeleteSkillReq{UserId: "user-a", Id: "demo-skill"})
	if err != nil {
		t.Fatalf("delete skill: %v", err)
	}
	if deletedID != "demo-skill" {
		t.Fatalf("unexpected deleted id: %q", deletedID)
	}
	userASkills, err = logic.ListSkills(ctx, &aiagent.ListSkillsReq{UserId: "user-a"})
	if err != nil {
		t.Fatalf("list user a skills after delete: %v", err)
	}
	if len(userASkills) != 0 {
		t.Fatalf("expected deleted skill to disappear, got %#v", userASkills)
	}
	if _, err := os.Stat(skillDir); !os.IsNotExist(err) {
		t.Fatalf("expected skill dir to be removed, stat err=%v", err)
	}
}

func TestSkillLogicRejectsPackageWithoutSkillMarkdown(t *testing.T) {
	logic := NewSkillLogic(t.TempDir())
	_, err := logic.ImportSkillPackage(context.Background(), &aiagent.ImportSkillPackageReq{
		UserId:   "user-a",
		FileName: "demo.zip",
		Data: buildSkillZip(t, map[string]string{
			"README.md": "# Demo\n",
		}),
	})
	if err == nil {
		t.Fatalf("expected missing SKILL.md error")
	}
}

func TestSkillLogicResolvesSystemSkillWithoutManifest(t *testing.T) {
	storageRoot := t.TempDir()
	logic := NewSkillLogic(storageRoot)
	skillDir := filepath.Join(storageRoot, systemSkillOwner, ".claude", "skills", "generated-app-builder")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("create system skill dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: generated-app-builder\ndescription: 生成应用\n---\n"), 0o644); err != nil {
		t.Fatalf("write system skill: %v", err)
	}

	dirs, err := logic.ResolveSkillDirs("user-a", []string{"generated-app-builder"})
	if err != nil {
		t.Fatalf("resolve skill dirs: %v", err)
	}
	if len(dirs) != 1 || dirs[0] != filepath.Join(storageRoot, systemSkillOwner) {
		t.Fatalf("unexpected skill dirs: %#v", dirs)
	}
}

func buildSkillZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for name, content := range files {
		file, err := writer.Create(name)
		if err != nil {
			t.Fatalf("create zip entry: %v", err)
		}
		if _, err := file.Write([]byte(content)); err != nil {
			t.Fatalf("write zip entry: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return buffer.Bytes()
}
