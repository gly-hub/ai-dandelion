package logic

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	aiagent "github.com/team-dandelion/ai-dandelion/proto/ai-agent"
	"github.com/team-dandelion/ai-dandelion/toolbox/authctx"
)

const (
	skillManifestFile = "manifest.json"
	claudeSkillDir    = ".claude/skills"
	systemSkillOwner  = "_system"
)

type SkillLogic struct {
	storageDir string
}

type skillManifest struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Source      string `json:"source"`
	Enabled     bool   `json:"enabled"`
	UpdatedAt   int64  `json:"updatedAt"`
}

func NewSkillLogic(storageDir string) *SkillLogic {
	return &SkillLogic{storageDir: storageDir}
}

func (s *SkillLogic) ListSkills(ctx context.Context, req *aiagent.ListSkillsReq) ([]*aiagent.AgentSkill, error) {
	userID, err := userIDFromContextOrRequest(ctx, req.GetUserId())
	if err != nil {
		return nil, err
	}
	skillDir := s.userClaudeSkillDir(userID)
	entries, err := os.ReadDir(skillDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	skills := make([]*aiagent.AgentSkill, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		manifest, err := s.readSkillManifestFromDir(filepath.Join(skillDir, entry.Name()))
		if err != nil {
			continue
		}
		skills = append(skills, skillManifestToProto(manifest))
	}
	sort.Slice(skills, func(i, j int) bool {
		return skills[i].GetUpdatedAt() > skills[j].GetUpdatedAt()
	})
	return skills, nil
}

func (s *SkillLogic) ImportSkillPackage(ctx context.Context, req *aiagent.ImportSkillPackageReq) (*aiagent.AgentSkill, error) {
	userID, err := userIDFromContextOrRequest(ctx, req.GetUserId())
	if err != nil {
		return nil, err
	}
	if !isZipFileName(req.GetFileName()) {
		return nil, errors.New("only zip skill packages are supported")
	}
	if len(req.GetData()) == 0 {
		return nil, errors.New("skill package is required")
	}

	reader, err := zip.NewReader(bytes.NewReader(req.GetData()), int64(len(req.GetData())))
	if err != nil {
		return nil, errors.New("invalid zip skill package")
	}
	skillFile := findSkillMarkdown(reader.File)
	if skillFile == nil {
		return nil, errors.New("SKILL.md is required in skill package")
	}
	markdown, err := readZipFileText(skillFile)
	if err != nil {
		return nil, err
	}
	meta, err := parseSkillMarkdown(markdown)
	if err != nil {
		return nil, err
	}

	now := time.Now().UnixMilli()
	manifest := skillManifest{
		ID:          normalizeSkillID(meta.ID),
		Name:        meta.Name,
		Description: meta.Description,
		Source:      "package:" + filepath.Base(req.GetFileName()),
		Enabled:     true,
		UpdatedAt:   now,
	}
	if manifest.ID == "" {
		manifest.ID = normalizeSkillID(manifest.Name)
	}
	if manifest.ID == "" {
		return nil, errors.New("SKILL.md name is invalid")
	}

	targetDir := filepath.Join(s.userClaudeSkillDir(userID), manifest.ID)
	if err := os.RemoveAll(targetDir); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return nil, err
	}
	if err := extractSkillZip(reader.File, skillFile, targetDir); err != nil {
		return nil, err
	}
	if err := writeSkillManifest(filepath.Join(targetDir, skillManifestFile), manifest); err != nil {
		return nil, err
	}
	return skillManifestToProto(manifest), nil
}

func (s *SkillLogic) UpdateSkill(ctx context.Context, req *aiagent.UpdateSkillReq) (*aiagent.AgentSkill, error) {
	userID, err := userIDFromContextOrRequest(ctx, req.GetUserId())
	if err != nil {
		return nil, err
	}
	input := req.GetSkill()
	if input == nil {
		return nil, errors.New("skill is required")
	}
	skillID := normalizeSkillID(input.GetId())
	if skillID == "" {
		return nil, errors.New("skill id is required")
	}
	name := strings.TrimSpace(input.GetName())
	if name == "" {
		return nil, errors.New("skill name is required")
	}

	manifestPath := filepath.Join(s.userClaudeSkillDir(userID), skillID, skillManifestFile)
	manifest, err := readSkillManifest(manifestPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, errors.New("skill not found")
		}
		return nil, err
	}
	manifest.ID = skillID
	manifest.Name = name
	manifest.Description = strings.TrimSpace(input.GetDescription())
	manifest.UpdatedAt = time.Now().UnixMilli()
	if manifest.Source == "" {
		manifest.Source = strings.TrimSpace(input.GetSource())
	}
	if manifest.Source == "" {
		manifest.Source = "personal"
	}
	manifest.Enabled = true

	if err := writeSkillManifest(manifestPath, manifest); err != nil {
		return nil, err
	}
	return skillManifestToProto(manifest), nil
}

func (s *SkillLogic) DeleteSkill(ctx context.Context, req *aiagent.DeleteSkillReq) (string, error) {
	userID, err := userIDFromContextOrRequest(ctx, req.GetUserId())
	if err != nil {
		return "", err
	}
	skillID := normalizeSkillID(req.GetId())
	if skillID == "" {
		return "", errors.New("skill id is required")
	}
	return skillID, os.RemoveAll(filepath.Join(s.userClaudeSkillDir(userID), skillID))
}

func (s *SkillLogic) UserSkillDir(userID string) string {
	userID = normalizePathSegment(userID)
	if userID == "" {
		return ""
	}
	return s.userDir(userID)
}

func (s *SkillLogic) ResolveSkillDirs(userID string, ids []string) ([]string, error) {
	ids = uniqueNormalizedSkillIDs(ids)
	if s == nil || len(ids) == 0 {
		return nil, nil
	}
	owners := []string{normalizePathSegment(userID), systemSkillOwner}
	dirs := make([]string, 0, len(owners))
	for _, owner := range uniqueStrings(owners) {
		if owner == "" {
			continue
		}
		if s.ownerHasAnySkill(owner, ids) {
			dirs = append(dirs, s.userDir(owner))
		}
	}
	return uniqueStrings(dirs), nil
}

func (s *SkillLogic) ResolveLinkedSkillDirs(ids []string) ([]string, error) {
	ids = uniqueNormalizedSkillIDs(ids)
	if s == nil || len(ids) == 0 {
		return nil, nil
	}
	needed := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		needed[id] = struct{}{}
	}
	root := s.storageRoot()
	entries, err := os.ReadDir(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	dirs := make([]string, 0, len(ids))
	seen := make(map[string]struct{}, len(ids))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillRoot := filepath.Join(root, entry.Name(), claudeSkillDir)
		for id := range needed {
			if _, ok := seen[id]; ok {
				continue
			}
			dir := filepath.Join(skillRoot, id)
			manifest, err := s.readSkillManifestFromDir(dir)
			if err != nil || !manifest.Enabled {
				continue
			}
			if normalizeSkillID(manifest.ID) != id {
				continue
			}
			dirs = append(dirs, s.userDir(entry.Name()))
			seen[id] = struct{}{}
		}
	}
	return uniqueStrings(dirs), nil
}

func (s *SkillLogic) ownerHasAnySkill(owner string, ids []string) bool {
	owner = normalizePathSegment(owner)
	if owner == "" || len(ids) == 0 {
		return false
	}
	for _, id := range ids {
		dir := filepath.Join(s.userClaudeSkillDir(owner), id)
		manifest, err := s.readSkillManifestFromDir(dir)
		if err != nil || !manifest.Enabled {
			continue
		}
		if normalizeSkillID(manifest.ID) == id {
			return true
		}
	}
	return false
}

func (s *SkillLogic) readSkillManifestFromDir(dir string) (skillManifest, error) {
	manifest, err := readSkillManifest(filepath.Join(dir, skillManifestFile))
	if err == nil {
		return manifest, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return skillManifest{}, err
	}
	markdown, err := os.ReadFile(filepath.Join(dir, "SKILL.md"))
	if err != nil {
		return skillManifest{}, err
	}
	meta, err := parseSkillMarkdown(string(markdown))
	if err != nil {
		return skillManifest{}, err
	}
	id := normalizeSkillID(meta.ID)
	if id == "" {
		id = normalizeSkillID(filepath.Base(dir))
	}
	if id == "" {
		id = normalizeSkillID(meta.Name)
	}
	if id == "" {
		return skillManifest{}, errors.New("skill id is required")
	}
	return skillManifest{
		ID:          id,
		Name:        firstNonEmpty(meta.Name, id),
		Description: meta.Description,
		Source:      "filesystem:" + dir,
		Enabled:     true,
		UpdatedAt:   fileModTime(filepath.Join(dir, "SKILL.md")),
	}, nil
}

func (s *SkillLogic) userDir(userID string) string {
	return filepath.Join(s.storageRoot(), normalizePathSegment(userID))
}

func (s *SkillLogic) userClaudeSkillDir(userID string) string {
	return filepath.Join(s.userDir(userID), claudeSkillDir)
}

func (s *SkillLogic) storageRoot() string {
	if strings.TrimSpace(s.storageDir) != "" {
		return filepath.Clean(s.storageDir)
	}
	return filepath.Join("data", "skills")
}

type parsedSkillMarkdown struct {
	ID          string
	Name        string
	Description string
}

func parseSkillMarkdown(markdown string) (parsedSkillMarkdown, error) {
	frontmatter := extractFrontmatter(markdown)
	name := readYAMLString(frontmatter, "name")
	if name == "" {
		name = readMarkdownTitle(markdown)
	}
	if name == "" {
		return parsedSkillMarkdown{}, errors.New("SKILL.md name is required")
	}
	return parsedSkillMarkdown{
		ID:          readYAMLString(frontmatter, "id"),
		Name:        name,
		Description: firstNonEmpty(readYAMLString(frontmatter, "description"), readYAMLString(frontmatter, "short-description"), readYAMLString(frontmatter, "short_description")),
	}, nil
}

func findSkillMarkdown(files []*zip.File) *zip.File {
	for _, file := range files {
		name := strings.TrimPrefix(filepath.ToSlash(file.Name), "/")
		if strings.HasPrefix(name, "__MACOSX/") || strings.Contains(name, "/__MACOSX/") {
			continue
		}
		if strings.EqualFold(filepath.Base(file.Name), "SKILL.md") {
			return file
		}
	}
	return nil
}

func readZipFileText(file *zip.File) (string, error) {
	rc, err := file.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()
	data, err := io.ReadAll(io.LimitReader(rc, 1024*1024))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func extractSkillZip(files []*zip.File, skillFile *zip.File, targetDir string) error {
	root := skillPackageRoot(skillFile)
	for _, file := range files {
		name, ok := cleanSkillZipEntry(file.Name, root)
		if !ok {
			continue
		}
		if name == "." || strings.HasPrefix(name, "..") || filepath.IsAbs(name) {
			return fmt.Errorf("invalid zip entry path: %s", file.Name)
		}
		targetPath := filepath.Join(targetDir, name)
		if !strings.HasPrefix(targetPath, targetDir+string(os.PathSeparator)) && targetPath != targetDir {
			return fmt.Errorf("invalid zip entry path: %s", file.Name)
		}
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return err
		}
		if err := extractZipFile(file, targetPath); err != nil {
			return err
		}
	}
	return nil
}

func skillPackageRoot(skillFile *zip.File) string {
	if skillFile == nil {
		return ""
	}
	dir := filepath.ToSlash(filepath.Dir(filepath.Clean(skillFile.Name)))
	if dir == "." {
		return ""
	}
	return strings.TrimSuffix(dir, "/") + "/"
}

func cleanSkillZipEntry(name string, root string) (string, bool) {
	name = strings.TrimPrefix(filepath.ToSlash(name), "/")
	if name == "" || name == "." {
		return "", false
	}
	if strings.HasPrefix(name, "__MACOSX/") || strings.Contains(name, "/__MACOSX/") {
		return "", false
	}
	if filepath.Base(name) == ".DS_Store" {
		return "", false
	}
	if root != "" {
		if !strings.HasPrefix(name, root) {
			return "", false
		}
		name = strings.TrimPrefix(name, root)
	}
	name = filepath.Clean(name)
	if name == "." {
		return "", false
	}
	return name, true
}

func extractZipFile(file *zip.File, targetPath string) error {
	rc, err := file.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	out, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, file.Mode())
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, rc)
	return err
}

func readSkillManifest(path string) (skillManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return skillManifest{}, err
	}
	var manifest skillManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return skillManifest{}, err
	}
	return manifest, nil
}

func writeSkillManifest(path string, manifest skillManifest) error {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func fileModTime(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.ModTime().UnixMilli()
}

func skillManifestToProto(manifest skillManifest) *aiagent.AgentSkill {
	return &aiagent.AgentSkill{
		Id:          manifest.ID,
		Name:        manifest.Name,
		Description: manifest.Description,
		Source:      manifest.Source,
		Enabled:     manifest.Enabled,
		UpdatedAt:   manifest.UpdatedAt,
	}
}

func requireUserID(userID string) (string, error) {
	userID = normalizePathSegment(userID)
	if userID == "" {
		return "", errors.New("userId is required")
	}
	return userID, nil
}

func userIDFromContextOrRequest(ctx context.Context, fallback string) (string, error) {
	if userID, err := authctx.RequireUserID(ctx); err == nil {
		return requireUserID(userID)
	}
	return requireUserID(fallback)
}

func isZipFileName(name string) bool {
	return strings.EqualFold(filepath.Ext(name), ".zip")
}

func extractFrontmatter(markdown string) string {
	markdown = strings.TrimPrefix(markdown, "\uFEFF")
	if !strings.HasPrefix(markdown, "---") {
		return ""
	}
	end := strings.Index(markdown[3:], "\n---")
	if end < 0 {
		return ""
	}
	return markdown[3 : 3+end]
}

func readYAMLString(source string, key string) string {
	if source == "" {
		return ""
	}
	re := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(key) + `\s*:\s*(.+?)\s*$`)
	match := re.FindStringSubmatch(source)
	if len(match) < 2 {
		return ""
	}
	value := strings.TrimSpace(match[1])
	value = strings.Trim(value, `"'`)
	if hash := strings.Index(value, " #"); hash >= 0 {
		value = strings.TrimSpace(value[:hash])
	}
	if value == "|" || value == ">" {
		return ""
	}
	return value
}

func readMarkdownTitle(markdown string) string {
	for _, line := range strings.Split(markdown, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
	}
	return ""
}

func normalizeSkillID(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ToLower(value)
	replacer := strings.NewReplacer(" ", "-", "\t", "-", "\n", "-")
	value = replacer.Replace(value)
	allowed := regexp.MustCompile(`[^a-z0-9_.:/\-\p{Han}]`)
	value = allowed.ReplaceAllString(value, "")
	value = strings.Trim(value, "-")
	return value
}

func uniqueNormalizedSkillIDs(values []string) []string {
	next := make([]string, 0, len(values))
	seen := make(map[string]struct{})
	for _, value := range values {
		id := normalizeSkillID(value)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		next = append(next, id)
	}
	return next
}

func normalizePathSegment(value string) string {
	value = normalizeSkillID(value)
	value = strings.ReplaceAll(value, "/", "-")
	value = strings.ReplaceAll(value, ":", "-")
	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func uniqueStrings(values []string) []string {
	next := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		next = append(next, value)
	}
	return next
}
