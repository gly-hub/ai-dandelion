package generatedapp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gly-hub/ai-dandelion/func-operation/internal/dao"
	"github.com/gly-hub/ai-dandelion/func-operation/internal/model"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

var generatedAppUUIDPattern = regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`)
var capabilityNamePattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_.-]{0,127}$`)
var artifactSHA256Pattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

var ErrAppNotFound = errors.New("generated app not found")
var ErrAppNotReady = errors.New("generated app not ready")

const defaultMaxModuleBytes = 16 << 20
const defaultMaxMemoryPages = 1024 // 64 MiB

type InvokeResult struct {
	AppID         string `json:"appId"`
	Version       string `json:"version"`
	Export        string `json:"export"`
	Result        uint64 `json:"result"`
	Response      string `json:"response,omitempty"`
	Duration      string `json:"duration"`
	Runtime       string `json:"runtime"`
	ModuleLen     int    `json:"moduleLen"`
	BackendSource string `json:"backendSource"`
	BackendModule string `json:"backendModule"`
}

type ArtifactSnapshot struct {
	AppID        string
	SHA256       string
	ManifestJSON string
}

type ArtifactReconcileResult struct {
	RemovedOrphans int
	RemovedStaging int
}

type CapabilityCallRequest struct {
	AppID      string         `json:"appId"`
	Capability string         `json:"capability"`
	Params     map[string]any `json:"params,omitempty"`
}

type CapabilityBroker interface {
	CallCapability(context.Context, string, CapabilityCallRequest) (any, error)
}

type ExternalAPICallRequest struct {
	APIKey  string         `json:"apiKey"`
	Query   map[string]any `json:"query,omitempty"`
	Headers map[string]any `json:"headers,omitempty"`
	Body    any            `json:"body,omitempty"`
}

type ExternalAPIExecutor interface {
	CallExternalAPI(context.Context, string, ExternalAPICallRequest) (any, error)
}

type Service struct {
	mu                  sync.RWMutex
	rootDir             string
	artifactStore       ArtifactStore
	store               *dao.GeneratedApp
	dataStore           *dao.GeneratedApp
	runtime             wazero.Runtime
	compiled            map[string]wazero.CompiledModule
	apps                map[string]model.GeneratedApp
	approvedArtifacts   map[string]string
	activeDirs          map[string]string
	draftHashes         map[string]string
	draftLoadMu         sync.Mutex
	results             map[uint64]resultRecord
	nextID              uint64
	nextInvocationID    uint64
	invokeTimeout       time.Duration
	maxResultBytes      int
	maxModuleBytes      int
	capabilityBroker    CapabilityBroker
	externalAPIExecutor ExternalAPIExecutor
	draftRuntime        bool
}

type Option func(*Service)

func WithInvokeTimeout(timeout time.Duration) Option {
	return func(s *Service) {
		s.invokeTimeout = timeout
	}
}

func WithMaxResultBytes(limit int) Option {
	return func(s *Service) {
		if limit > 0 {
			s.maxResultBytes = limit
		}
	}
}

// WithMaxModuleBytes limits untrusted WASM artifacts before compilation.
func WithMaxModuleBytes(limit int) Option {
	return func(s *Service) {
		if limit > 0 {
			s.maxModuleBytes = limit
		}
	}
}

func WithArtifactStore(store ArtifactStore) Option {
	return func(s *Service) {
		if store != nil {
			s.artifactStore = store
		}
	}
}

// WithDataStore separates runtime business data from application metadata.
func WithDataStore(store *dao.GeneratedApp) Option {
	return func(s *Service) {
		if store != nil {
			s.dataStore = store
		}
	}
}

// WithDraftRuntime creates an editor-only runtime. It loads the mutable
// function workspace and must never be used by published application routes.
func WithDraftRuntime() Option {
	return func(s *Service) { s.draftRuntime = true }
}

type resultRecord struct {
	appID        string
	invocationID uint64
	data         []byte
}

type manifest struct {
	SchemaVersion  string   `json:"schemaVersion,omitempty"`
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Version        string   `json:"version"`
	Description    string   `json:"description"`
	Export         string   `json:"export"`
	Actions        []string `json:"actions"`
	FrontendEntry  string   `json:"frontendEntry"`
	FrontendFile   string   `json:"frontendFile"`
	BackendSource  string   `json:"backendSource"`
	BackendModule  string   `json:"backendModule"`
	DataModelHints []string `json:"dataModelHints,omitempty"`
	Tables         []struct {
		Name string `json:"name"`
		DDL  string `json:"ddl"`
	} `json:"tables"`
	DataModels   []dao.DataModel          `json:"dataModels"`
	Relations    []dao.DataRelation       `json:"relations"`
	Queries      []dao.DataQuery          `json:"queries"`
	Provides     []CapabilityProvide      `json:"provides,omitempty"`
	Consumes     []CapabilityConsume      `json:"consumes,omitempty"`
	ConfigKeys   []string                 `json:"configKeys,omitempty"`
	ExternalAPIs []ExternalAPIDeclaration `json:"externalApis,omitempty"`
}

type ExternalAPIDeclaration struct {
	APIKey string `json:"apiKey"`
}

type CapabilityProvide struct {
	Name  string `json:"name"`
	Query string `json:"query"`
}
type CapabilityConsume struct {
	AppID      string `json:"appId"`
	Capability string `json:"capability"`
}

func (s *Service) SetCapabilityBroker(broker CapabilityBroker) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.capabilityBroker = broker
}

func (s *Service) SetExternalAPIExecutor(executor ExternalAPIExecutor) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.externalAPIExecutor = executor
}

func (s *Service) RunDeclaredCapability(ctx context.Context, appID, capability string, params map[string]any) (dao.DataListResult, error) {
	item, err := s.activeManifestForApp(appID)
	if err != nil {
		return dao.DataListResult{}, err
	}
	for _, provide := range item.Provides {
		if strings.TrimSpace(provide.Name) == strings.TrimSpace(capability) {
			return s.dataStore.DataRunQuery(ctx, appID, item.DataModels, item.Relations, item.Queries, dao.DataRunQueryRequest{Query: strings.TrimSpace(provide.Query), Params: params})
		}
	}
	return dao.DataListResult{}, errors.New("target capability is not declared")
}

// AllowsConfigKeys verifies that a function can only resolve the public
// configuration values it declared in its immutable manifest.
func (s *Service) AllowsConfigKeys(appID string, keys []string) bool {
	item, err := s.activeManifestForApp(strings.TrimSpace(appID))
	if err != nil {
		return false
	}
	allowed := make(map[string]struct{}, len(item.ConfigKeys))
	for _, key := range item.ConfigKeys {
		allowed[strings.TrimSpace(key)] = struct{}{}
	}
	for _, key := range keys {
		if _, ok := allowed[strings.TrimSpace(key)]; !ok {
			return false
		}
	}
	return len(keys) > 0
}

// DeclaresAction checks the manifest that is currently loaded by this runtime.
// Published runtimes therefore consult immutable release content, never a
// mutable draft workspace.
func (s *Service) DeclaresAction(appID, action string) bool {
	item, err := s.activeManifestForApp(strings.TrimSpace(appID))
	if err != nil {
		return false
	}
	for _, declared := range item.Actions {
		if strings.TrimSpace(declared) == strings.TrimSpace(action) {
			return true
		}
	}
	return false
}

func (s *Service) RootDir() string {
	if s == nil {
		return ""
	}
	return s.rootDir
}

func NewService(ctx context.Context, rootDir string, store *dao.GeneratedApp, options ...Option) (*Service, error) {
	resolvedRoot, err := resolveRootDir(rootDir)
	if err != nil {
		return nil, err
	}
	if store == nil {
		return nil, errors.New("generated app store is required")
	}
	service := &Service{
		rootDir:           resolvedRoot,
		artifactStore:     NewLocalArtifactStore(resolvedRoot),
		store:             store,
		dataStore:         store,
		runtime:           wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfig().WithMemoryLimitPages(defaultMaxMemoryPages).WithCloseOnContextDone(true)),
		compiled:          make(map[string]wazero.CompiledModule),
		apps:              make(map[string]model.GeneratedApp),
		approvedArtifacts: make(map[string]string),
		activeDirs:        make(map[string]string),
		draftHashes:       make(map[string]string),
		results:           make(map[uint64]resultRecord),
		nextID:            1,
		nextInvocationID:  1,
		invokeTimeout:     5 * time.Second,
		maxResultBytes:    1 << 20,
		maxModuleBytes:    defaultMaxModuleBytes,
	}
	for _, option := range options {
		if option != nil {
			option(service)
		}
	}
	wasi_snapshot_preview1.MustInstantiate(ctx, service.runtime)
	if err := service.instantiatePlatformHost(ctx); err != nil {
		_ = service.Close(ctx)
		return nil, err
	}
	return service, nil
}

// LoadApprovedArtifacts is used at boot and after a publish. It only loads
// artifacts whose immutable hash has been recorded in a published release.
func (s *Service) LoadApprovedArtifacts(ctx context.Context, approvals map[string]string) error {
	approved := make(map[string]string, len(approvals))
	for appID, artifactSHA := range approvals {
		appID = strings.TrimSpace(appID)
		artifactSHA = strings.TrimSpace(artifactSHA)
		if !generatedAppUUIDPattern.MatchString(appID) || !artifactSHA256Pattern.MatchString(artifactSHA) {
			return errors.New("invalid published artifact approval")
		}
		approved[appID] = artifactSHA
	}
	nextApps := make(map[string]model.GeneratedApp, len(approved))
	nextCompiled := make(map[string]wazero.CompiledModule, len(approved))
	nextDirs := make(map[string]string, len(approved))
	for appID, hash := range approved {
		dir, err := s.artifactStore.Materialize(ctx, hash)
		if err != nil {
			return fmt.Errorf("materialize published artifact %q: %w", appID, err)
		}
		snapshot, err := s.publishedArtifactSnapshotAt(dir)
		if err != nil {
			return fmt.Errorf("validate published artifact %q: %w", appID, err)
		}
		if snapshot.AppID != appID || snapshot.SHA256 != hash {
			return fmt.Errorf("published artifact %q does not match its approved release", appID)
		}
		app, compiled, err := s.loadPublishedApp(ctx, dir)
		if err != nil {
			return fmt.Errorf("load published artifact %q: %w", appID, err)
		}
		nextApps[appID], nextCompiled[appID], nextDirs[appID] = app, compiled, dir
	}
	s.mu.Lock()
	s.approvedArtifacts = approved
	s.apps, s.compiled, s.activeDirs = nextApps, nextCompiled, nextDirs
	s.mu.Unlock()
	return nil
}

// LoadDraftApp validates and loads the current workspace source into the
// editor-only runtime. Draft code is deliberately not promoted to the
// artifact store and does not create a release record.
func (s *Service) LoadDraftApp(ctx context.Context, appID string) (model.GeneratedApp, error) {
	if s == nil || !s.draftRuntime {
		return model.GeneratedApp{}, errors.New("draft runtime is not configured")
	}
	appID = strings.TrimSpace(appID)
	if !generatedAppUUIDPattern.MatchString(appID) {
		return model.GeneratedApp{}, ErrAppNotFound
	}
	s.draftLoadMu.Lock()
	defer s.draftLoadMu.Unlock()
	fingerprint, err := s.draftSourceFingerprint(appID)
	if err != nil {
		return model.GeneratedApp{}, err
	}
	s.mu.RLock()
	cached, cachedOK := s.apps[appID]
	cachedHash := s.draftHashes[appID]
	compiled := s.compiled[appID]
	s.mu.RUnlock()
	if cachedOK && compiled != nil && cachedHash == fingerprint {
		return cached, nil
	}
	snapshot, err := s.ArtifactSnapshot(appID)
	if err != nil {
		return model.GeneratedApp{}, err
	}
	app, compiled, err := s.loadApp(ctx, filepath.Join(s.rootDir, appID))
	if err != nil {
		return model.GeneratedApp{}, err
	}
	s.mu.Lock()
	s.apps[appID] = app
	s.compiled[appID] = compiled
	s.activeDirs[appID] = filepath.Join(s.rootDir, appID)
	s.draftHashes[appID] = snapshot.SHA256
	s.mu.Unlock()
	return app, nil
}

// FrontendBundle returns the complete allowed ESM graph in one response. The
// caller already has an authenticated function/app authorization boundary.
func (s *Service) FrontendBundle(appID string) (version, entry string, modules map[string]string, err error) {
	appID = strings.TrimSpace(appID)
	s.mu.RLock()
	app, ok := s.apps[appID]
	appDir := s.activeDirs[appID]
	version = s.approvedArtifacts[appID]
	if s.draftRuntime {
		version = s.draftHashes[appID]
	}
	s.mu.RUnlock()
	if !ok || appDir == "" {
		return "", "", nil, ErrAppNotFound
	}
	entry = frontendFileFromEntry(app.FrontendEntry)
	files, err := artifactFiles(appDir, app.BackendModule)
	if err != nil {
		return "", "", nil, err
	}
	modules = make(map[string]string)
	for _, file := range files {
		if file == "manifest.json" || file == app.BackendModule || filepath.Ext(file) != ".js" {
			continue
		}
		// Keep the artifact-relative path. Entry modules commonly import
		// "./frontend/<module>.js", which resolves from the app root URL.
		key := file
		contents, readErr := os.ReadFile(filepath.Join(appDir, filepath.FromSlash(file)))
		if readErr != nil {
			return "", "", nil, readErr
		}
		modules[key] = string(contents)
	}
	if _, ok := modules[entry]; !ok {
		return "", "", nil, ErrAppNotReady
	}
	return version, entry, modules, nil
}

func (s *Service) draftSourceFingerprint(appID string) (string, error) {
	dir := filepath.Join(s.rootDir, appID)
	item, err := loadManifestFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return "", err
	}
	app, err := appFromManifest(item)
	if err != nil {
		return "", err
	}
	files, err := artifactFiles(dir, app.BackendModule)
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	for _, file := range files {
		contents, readErr := os.ReadFile(filepath.Join(dir, filepath.FromSlash(file)))
		if readErr != nil {
			return "", readErr
		}
		_, _ = hash.Write([]byte(file))
		_, _ = hash.Write([]byte{0})
		_, _ = hash.Write(contents)
		_, _ = hash.Write([]byte{0})
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// ArtifactSnapshot validates a candidate artifact and fingerprints every file
// executable by the browser or Wazero runtime.
func (s *Service) ArtifactSnapshot(appID string) (ArtifactSnapshot, error) {
	appID = strings.TrimSpace(appID)
	if !generatedAppUUIDPattern.MatchString(appID) {
		return ArtifactSnapshot{}, errors.New("invalid app id")
	}
	return s.artifactSnapshotAt(filepath.Join(s.rootDir, appID))
}

func (s *Service) artifactSnapshotAt(appDir string) (ArtifactSnapshot, error) {
	item, err := loadManifestFile(filepath.Join(appDir, "manifest.json"))
	if err != nil {
		return ArtifactSnapshot{}, err
	}
	if len(item.Tables) > 0 {
		return ArtifactSnapshot{}, errors.New("legacy manifest DDL is not supported; declare dataModels instead")
	}
	if err := validateCapabilities(item); err != nil {
		return ArtifactSnapshot{}, err
	}
	app, err := s.validateAppForManifest(context.Background(), appDir, item, true)
	if err != nil {
		return ArtifactSnapshot{}, err
	}
	files, err := artifactFiles(appDir, app.BackendModule)
	if err != nil {
		return ArtifactSnapshot{}, err
	}
	hash := sha256.New()
	for _, file := range files {
		contents, readErr := os.ReadFile(filepath.Join(appDir, filepath.FromSlash(file)))
		if readErr != nil {
			return ArtifactSnapshot{}, readErr
		}
		_, _ = hash.Write([]byte(file))
		_, _ = hash.Write([]byte{0})
		_, _ = hash.Write(contents)
		_, _ = hash.Write([]byte{0})
	}
	raw, err := os.ReadFile(filepath.Join(appDir, "manifest.json"))
	if err != nil {
		return ArtifactSnapshot{}, err
	}
	return ArtifactSnapshot{AppID: app.UUID, SHA256: hex.EncodeToString(hash.Sum(nil)), ManifestJSON: string(raw)}, nil
}

// PublishedArtifactSnapshot verifies a promoted release without normalizing or
// writing files. Published artifacts are immutable runtime input.
func (s *Service) PublishedArtifactSnapshot(appID, artifactSHA string) (ArtifactSnapshot, error) {
	appID = strings.TrimSpace(appID)
	artifactSHA = strings.TrimSpace(artifactSHA)
	if !generatedAppUUIDPattern.MatchString(appID) || !artifactSHA256Pattern.MatchString(artifactSHA) {
		return ArtifactSnapshot{}, errors.New("invalid published artifact reference")
	}
	dir, err := s.artifactStore.Materialize(context.Background(), artifactSHA)
	if err != nil {
		return ArtifactSnapshot{}, err
	}
	snapshot, err := s.publishedArtifactSnapshotAt(dir)
	if err != nil {
		return ArtifactSnapshot{}, err
	}
	if snapshot.AppID != appID || snapshot.SHA256 != artifactSHA {
		return ArtifactSnapshot{}, errors.New("published artifact does not match its release")
	}
	return snapshot, nil
}

func (s *Service) publishedArtifactSnapshotAt(appDir string) (ArtifactSnapshot, error) {
	item, raw, err := loadImmutableManifest(filepath.Join(appDir, "manifest.json"))
	if err != nil {
		return ArtifactSnapshot{}, err
	}
	if len(item.Tables) > 0 {
		return ArtifactSnapshot{}, errors.New("legacy manifest DDL is not supported; declare dataModels instead")
	}
	if err := validateCapabilities(item); err != nil {
		return ArtifactSnapshot{}, err
	}
	app, err := s.validateAppForManifest(context.Background(), appDir, item, false)
	if err != nil {
		return ArtifactSnapshot{}, err
	}
	files, err := artifactFiles(appDir, app.BackendModule)
	if err != nil {
		return ArtifactSnapshot{}, err
	}
	hash := sha256.New()
	for _, file := range files {
		contents, readErr := os.ReadFile(filepath.Join(appDir, filepath.FromSlash(file)))
		if readErr != nil {
			return ArtifactSnapshot{}, readErr
		}
		_, _ = hash.Write([]byte(file))
		_, _ = hash.Write([]byte{0})
		_, _ = hash.Write(contents)
		_, _ = hash.Write([]byte{0})
	}
	return ArtifactSnapshot{AppID: app.UUID, SHA256: hex.EncodeToString(hash.Sum(nil)), ManifestJSON: string(raw)}, nil
}

// PromoteArtifact validates a draft and promotes its executable files through
// the configured artifact store.
func (s *Service) PromoteArtifact(appID, expectedSHA string) error {
	snapshot, err := s.ArtifactSnapshot(appID)
	if err != nil {
		return err
	}
	if snapshot.SHA256 != strings.TrimSpace(expectedSHA) {
		return errors.New("artifact changed before promotion")
	}
	bundle, err := s.artifactBundle(appID, snapshot)
	if err != nil {
		return err
	}
	if err := s.artifactStore.Promote(context.Background(), bundle); err != nil {
		return err
	}
	_, err = s.PublishedArtifactSnapshot(appID, snapshot.SHA256)
	return err
}

func (s *Service) artifactBundle(appID string, snapshot ArtifactSnapshot) (ArtifactBundle, error) {
	source := filepath.Join(s.rootDir, appID)
	item, err := loadManifestFile(filepath.Join(source, "manifest.json"))
	if err != nil {
		return ArtifactBundle{}, err
	}
	app, err := appFromManifest(item)
	if err != nil {
		return ArtifactBundle{}, err
	}
	files, err := artifactFiles(source, app.BackendModule)
	if err != nil {
		return ArtifactBundle{}, err
	}
	bundle := ArtifactBundle{SHA256: snapshot.SHA256, Files: make(map[string][]byte, len(files))}
	for _, file := range files {
		contents, err := os.ReadFile(filepath.Join(source, filepath.FromSlash(file)))
		if err != nil {
			return ArtifactBundle{}, err
		}
		if file == "manifest.json" {
			contents = []byte(snapshot.ManifestJSON)
		}
		bundle.Files[file] = contents
	}
	return bundle, nil
}

func sealArtifactDir(root string) error {
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return errors.New("artifact cannot contain symlinks")
		}
		if entry.IsDir() {
			return os.Chmod(path, 0o555)
		}
		return os.Chmod(path, 0o444)
	})
}

func makeArtifactDirWritable(root string) error {
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		if entry.IsDir() {
			return os.Chmod(path, 0o755)
		}
		return os.Chmod(path, 0o644)
	})
}

func (s *Service) ReconcileArtifacts(ctx context.Context, referenced map[string]struct{}, staleStagingAfter time.Duration) (ArtifactReconcileResult, error) {
	if s == nil || s.artifactStore == nil {
		return ArtifactReconcileResult{}, errors.New("artifact store is not configured")
	}
	return s.artifactStore.Reconcile(ctx, referenced, staleStagingAfter)
}

// ReconcileLocalArtifacts remains as the local-store compatibility entry point.
func (s *Service) ReconcileLocalArtifacts(referenced map[string]struct{}, staleStagingAfter time.Duration) (ArtifactReconcileResult, error) {
	return s.ReconcileArtifacts(context.Background(), referenced, staleStagingAfter)
}

func validateCapabilities(item manifest) error {
	queries := make(map[string]struct{}, len(item.Queries))
	for _, query := range item.Queries {
		queries[strings.TrimSpace(query.Name)] = struct{}{}
	}
	provided := make(map[string]struct{}, len(item.Provides))
	for _, provide := range item.Provides {
		name, query := strings.TrimSpace(provide.Name), strings.TrimSpace(provide.Query)
		if !capabilityNamePattern.MatchString(name) || query == "" {
			return errors.New("invalid provided capability")
		}
		if _, ok := queries[query]; !ok {
			return fmt.Errorf("provided capability %q references undeclared query %q", name, query)
		}
		if _, ok := provided[name]; ok {
			return fmt.Errorf("duplicate provided capability %q", name)
		}
		provided[name] = struct{}{}
	}
	consumed := make(map[string]struct{}, len(item.Consumes))
	for _, consume := range item.Consumes {
		appID, name := strings.TrimSpace(consume.AppID), strings.TrimSpace(consume.Capability)
		if !generatedAppUUIDPattern.MatchString(appID) || appID == item.ID || !capabilityNamePattern.MatchString(name) {
			return errors.New("invalid consumed capability")
		}
		key := appID + "\x00" + name
		if _, ok := consumed[key]; ok {
			return fmt.Errorf("duplicate consumed capability %q", name)
		}
		consumed[key] = struct{}{}
	}
	return nil
}

// CandidateApp validates a draft artifact and persists only its metadata. It
// deliberately does not add the module to the executable runtime map; that is
// reserved for LoadApprovedArtifacts after release publication.
func (s *Service) CandidateApp(ctx context.Context, appID string) (*model.GeneratedApp, error) {
	if s == nil || s.store == nil {
		return nil, errors.New("generated app runtime is not configured")
	}
	if _, err := s.ArtifactSnapshot(appID); err != nil {
		return nil, err
	}
	_, item, err := s.manifestPathForApp(appID)
	if err != nil {
		return nil, err
	}
	app, err := appFromManifest(item)
	if err != nil {
		return nil, err
	}
	if err := s.store.Upsert(ctx, &app); err != nil {
		return nil, err
	}
	return &app, nil
}

func artifactFiles(appDir string, backendModule string) ([]string, error) {
	files := []string{"manifest.json", filepath.ToSlash(backendModule), "frontend.js"}
	frontendDir := filepath.Join(appDir, "frontend")
	if err := filepath.WalkDir(frontendDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return errors.New("artifact cannot contain symlinks")
		}
		if entry.IsDir() {
			return nil
		}
		if filepath.Ext(entry.Name()) != ".js" {
			return errors.New("frontend artifact must be a JavaScript module")
		}
		rel, relErr := filepath.Rel(appDir, path)
		if relErr != nil {
			return relErr
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	}); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func (s *Service) ListApps(ctx context.Context) ([]model.GeneratedApp, error) {
	apps, err := s.store.List(ctx)
	if err != nil {
		return nil, err
	}
	sort.Slice(apps, func(left, right int) bool {
		return apps[left].UUID < apps[right].UUID
	})
	return apps, nil
}

// HasApprovedArtifact confirms that the running runtime has already loaded the
// immutable release expected by the database. Request paths must use this
// instead of recompiling the same WASM module just to revalidate it.
func (s *Service) HasApprovedArtifact(appID, artifactSHA string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, appLoaded := s.apps[appID]
	return appLoaded && s.compiled[appID] != nil && s.activeDirs[appID] != "" && s.approvedArtifacts[appID] == artifactSHA
}

func (s *Service) Invoke(ctx context.Context, appID string, payload json.RawMessage) (InvokeResult, error) {
	s.mu.RLock()
	app, ok := s.apps[appID]
	compiled := s.compiled[appID]
	s.mu.RUnlock()
	if !ok || compiled == nil {
		return InvokeResult{}, ErrAppNotFound
	}

	if s.invokeTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.invokeTimeout)
		defer cancel()
	}

	invocationID := s.beginInvocation()
	callCtx := context.WithValue(ctx, appIDContextKey{}, appID)
	callCtx = context.WithValue(callCtx, invocationIDContextKey{}, invocationID)
	defer s.cleanupInvocationResults(invocationID)

	start := time.Now()
	module, err := s.runtime.InstantiateModule(ctx, compiled, wazero.NewModuleConfig().WithStartFunctions())
	if err != nil {
		return InvokeResult{}, fmt.Errorf("instantiate generated app %q: %w", appID, err)
	}
	defer module.Close(ctx)

	if initialize := module.ExportedFunction("_initialize"); initialize != nil {
		if _, err := initialize.Call(ctx); err != nil {
			return InvokeResult{}, fmt.Errorf("initialize generated app %q: %w", appID, err)
		}
	}

	handle := module.ExportedFunction(app.Export)
	if handle == nil {
		return InvokeResult{}, fmt.Errorf("generated app %q missing export %q", appID, app.Export)
	}
	results, err := s.callAppHandle(callCtx, module, handle, payload)
	if err != nil {
		return InvokeResult{}, fmt.Errorf("call generated app %q export %q: %w", appID, app.Export, err)
	}
	if len(results) == 0 {
		return InvokeResult{}, fmt.Errorf("generated app %q export %q returned no result", appID, app.Export)
	}
	response, err := s.resultPayload(callCtx, results[0])
	if err != nil {
		return InvokeResult{}, fmt.Errorf("read generated app %q result: %w", appID, err)
	}

	return InvokeResult{
		AppID:         app.UUID,
		Version:       app.Version,
		Export:        app.Export,
		Result:        results[0],
		Response:      response,
		Duration:      time.Since(start).String(),
		Runtime:       "wazero",
		ModuleLen:     moduleSize(s.activeDir(appID), app),
		BackendSource: app.BackendSource,
		BackendModule: app.BackendModule,
	}, nil
}

func (s *Service) FrontendCode(appID string, requestedPath string) (string, error) {
	s.mu.RLock()
	app, ok := s.apps[appID]
	appDir := s.activeDirs[appID]
	s.mu.RUnlock()
	if !ok || appDir == "" {
		return "", ErrAppNotFound
	}
	fileName := frontendFileFromEntry(app.FrontendEntry)
	if strings.TrimSpace(requestedPath) != "" {
		fileName = requestedPath
	}
	path, err := safeFrontendFilePath(appDir, fileName)
	if err != nil {
		return "", err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read frontend code %q: %w", path, err)
	}
	return string(content), nil
}

func (s *Service) Close(ctx context.Context) error {
	if s == nil || s.runtime == nil {
		return nil
	}
	return s.runtime.Close(ctx)
}

func (s *Service) callAppHandle(ctx context.Context, module api.Module, handle api.Function, payload json.RawMessage) ([]uint64, error) {
	paramTypes := handle.Definition().ParamTypes()
	if len(paramTypes) == 0 {
		return handle.Call(ctx)
	}
	if len(paramTypes) != 2 || paramTypes[0] != api.ValueTypeI32 || paramTypes[1] != api.ValueTypeI32 {
		return nil, fmt.Errorf("unsupported handle signature params=%v", paramTypes)
	}
	alloc := module.ExportedFunction("alloc")
	if alloc == nil {
		return nil, errors.New("payload-aware wasm app must export alloc(size uint32) uint32")
	}
	request := []byte(payload)
	if len(request) == 0 || string(request) == "null" {
		request = []byte("{}")
	}
	allocResults, err := alloc.Call(ctx, uint64(len(request)))
	if err != nil {
		return nil, fmt.Errorf("allocate request memory: %w", err)
	}
	if len(allocResults) == 0 {
		return nil, errors.New("alloc returned no pointer")
	}
	ptr := uint32(allocResults[0])
	if !module.Memory().Write(ptr, request) {
		return nil, errors.New("write request memory failed")
	}
	return handle.Call(ctx, uint64(ptr), uint64(len(request)))
}

func (s *Service) loadApp(ctx context.Context, dir string) (model.GeneratedApp, wazero.CompiledModule, error) {
	item, err := loadManifestFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return model.GeneratedApp{}, nil, fmt.Errorf("%w: manifest is missing for %q", ErrAppNotReady, dir)
		}
		return model.GeneratedApp{}, nil, fmt.Errorf("decode generated app manifest %q: %w", dir, err)
	}
	return s.loadAppForManifest(ctx, dir, item, true)
}

func (s *Service) loadPublishedApp(ctx context.Context, dir string) (model.GeneratedApp, wazero.CompiledModule, error) {
	item, _, err := loadImmutableManifest(filepath.Join(dir, "manifest.json"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return model.GeneratedApp{}, nil, fmt.Errorf("%w: manifest is missing for %q", ErrAppNotReady, dir)
		}
		return model.GeneratedApp{}, nil, fmt.Errorf("decode published app manifest %q: %w", dir, err)
	}
	return s.loadAppForManifest(ctx, dir, item, false)
}

func (s *Service) loadAppForManifest(ctx context.Context, dir string, item manifest, requireDirID bool) (model.GeneratedApp, wazero.CompiledModule, error) {
	app, moduleBytes, err := s.appModuleForManifest(dir, item, requireDirID)
	if err != nil {
		return model.GeneratedApp{}, nil, err
	}
	compiled, err := s.runtime.CompileModule(ctx, moduleBytes)
	if err != nil {
		return model.GeneratedApp{}, nil, fmt.Errorf("compile generated app %q: %w", app.UUID, err)
	}
	return app, compiled, nil
}

// validateAppForManifest compiles using an isolated runtime. Wazero caches
// compiled code by the module content hash; closing a validation compilation in
// the execution runtime could evict the active module used by Invoke.
func (s *Service) validateAppForManifest(ctx context.Context, dir string, item manifest, requireDirID bool) (model.GeneratedApp, error) {
	app, moduleBytes, err := s.appModuleForManifest(dir, item, requireDirID)
	if err != nil {
		return model.GeneratedApp{}, err
	}
	validator := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfig().WithMemoryLimitPages(defaultMaxMemoryPages).WithCloseOnContextDone(true))
	defer func() { _ = validator.Close(ctx) }()
	compiled, err := validator.CompileModule(ctx, moduleBytes)
	if err != nil {
		return model.GeneratedApp{}, fmt.Errorf("compile generated app %q: %w", app.UUID, err)
	}
	defer func() { _ = compiled.Close(ctx) }()
	return app, nil
}

func (s *Service) appModuleForManifest(dir string, item manifest, requireDirID bool) (model.GeneratedApp, []byte, error) {
	app, err := appFromManifest(item)
	if err != nil {
		return model.GeneratedApp{}, nil, fmt.Errorf("invalid generated app manifest %q: %w", dir, err)
	}
	if folderID := filepath.Base(dir); requireDirID && folderID != app.UUID {
		return model.GeneratedApp{}, nil, fmt.Errorf("generated app folder %q must match manifest id %q", folderID, app.UUID)
	}
	if len(item.Tables) > 0 {
		return model.GeneratedApp{}, nil, errors.New("legacy manifest DDL is not supported; declare dataModels instead")
	}
	modulePath := filepath.Join(dir, app.BackendModule)
	if info, statErr := os.Lstat(modulePath); statErr != nil {
		return model.GeneratedApp{}, nil, statErr
	} else if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return model.GeneratedApp{}, nil, errors.New("backend module must be a regular file")
	}
	moduleBytes, err := readWASMModule(modulePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return model.GeneratedApp{}, nil, fmt.Errorf("%w: backend module is missing for %q", ErrAppNotReady, dir)
		}
		return model.GeneratedApp{}, nil, err
	}
	if s.maxModuleBytes > 0 && len(moduleBytes) > s.maxModuleBytes {
		return model.GeneratedApp{}, nil, fmt.Errorf("generated app %q module exceeds the size limit", app.UUID)
	}
	return app, moduleBytes, nil
}

func (s *Service) ApplyDeclaredDataModels(ctx context.Context, appID string) error {
	if s == nil || s.draftRuntime {
		return errors.New("published data model runtime is not configured")
	}
	return s.applyDeclaredDataModels(ctx, appID)
}

// PrepareDraftDataModels initializes the isolated debug database for a draft.
func (s *Service) PrepareDraftDataModels(ctx context.Context, appID string) error {
	if s == nil || !s.draftRuntime {
		return errors.New("draft data model runtime is not configured")
	}
	return s.applyDeclaredDataModels(ctx, appID)
}

func (s *Service) applyDeclaredDataModels(ctx context.Context, appID string) error {
	if s == nil || s.dataStore == nil {
		return errors.New("generated app runtime is not configured")
	}
	_, item, err := s.manifestPathForApp(appID)
	if err != nil {
		return err
	}
	if len(item.Tables) > 0 {
		return errors.New("legacy manifest DDL is not supported; declare dataModels instead")
	}
	return s.dataStore.ApplyDataModels(ctx, appID, item.DataModels)
}

type DataFormSummary struct {
	Name       string
	Label      string
	FieldCount int32
	RowCount   int64
	TableName  string
}

func (s *Service) ListDataForms(ctx context.Context, appID string) ([]DataFormSummary, error) {
	if s == nil || s.dataStore == nil {
		return nil, errors.New("generated app runtime is not configured")
	}
	item, err := s.manifestForApp(appID)
	if err != nil {
		return nil, err
	}
	summaries, err := s.dataStore.ListDataModelSummaries(ctx, appID, item.DataModels)
	if err != nil {
		return nil, err
	}
	out := make([]DataFormSummary, 0, len(summaries))
	for _, summary := range summaries {
		out = append(out, DataFormSummary{
			Name:       summary.Model.Name,
			Label:      firstNonEmpty(strings.TrimSpace(summary.Model.Label), summary.Model.Name),
			FieldCount: int32(len(summary.Model.Fields)),
			RowCount:   summary.RowCount,
			TableName:  summary.TableName,
		})
	}
	return out, nil
}

func (s *Service) DeleteDataForm(ctx context.Context, appID string, formName string) ([]DataFormSummary, error) {
	if s == nil || s.dataStore == nil {
		return nil, errors.New("generated app runtime is not configured")
	}
	formName = strings.TrimSpace(formName)
	if formName == "" {
		return nil, errors.New("data form name is required")
	}
	appDir, item, err := s.manifestPathForApp(appID)
	if err != nil {
		return nil, err
	}
	index := -1
	var target dao.DataModel
	for i, model := range item.DataModels {
		if model.Name == formName {
			index = i
			target = model
			break
		}
	}
	if index < 0 {
		return nil, fmt.Errorf("data form %q is not declared", formName)
	}
	if err := s.dataStore.DropDataModel(ctx, appID, target); err != nil {
		return nil, err
	}
	item.DataModels = append(item.DataModels[:index], item.DataModels[index+1:]...)
	item.Relations = filterRelationsForDeletedModel(item.Relations, formName)
	item.Queries = filterQueriesForDeletedModel(item.Queries, formName)
	raw, err := marshalManifest(item)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(appDir, "manifest.json"), raw, 0o644); err != nil {
		return nil, fmt.Errorf("write generated app manifest %q: %w", appID, err)
	}
	return s.ListDataForms(ctx, appID)
}

func (s *Service) manifestPathForApp(appID string) (string, manifest, error) {
	appID = strings.TrimSpace(appID)
	if !generatedAppUUIDPattern.MatchString(appID) {
		return "", manifest{}, ErrAppNotFound
	}
	appDir := filepath.Join(s.rootDir, appID)
	raw, err := os.ReadFile(filepath.Join(appDir, "manifest.json"))
	if err != nil {
		return "", manifest{}, fmt.Errorf("read generated app manifest %q: %w", appID, err)
	}
	item, _, _, err := decodeManifestWithDir(appDir, raw)
	if err != nil {
		return "", manifest{}, fmt.Errorf("decode generated app manifest %q: %w", appID, err)
	}
	if item.ID != appID {
		return "", manifest{}, errors.New("generated app manifest id does not match path")
	}
	return appDir, item, nil
}

func filterRelationsForDeletedModel(relations []dao.DataRelation, modelName string) []dao.DataRelation {
	out := make([]dao.DataRelation, 0, len(relations))
	for _, relation := range relations {
		fromModel, _, fromErr := splitQualifiedName(relation.From)
		toModel, _, toErr := splitQualifiedName(relation.To)
		if fromErr == nil && fromModel == modelName {
			continue
		}
		if toErr == nil && toModel == modelName {
			continue
		}
		out = append(out, relation)
	}
	return out
}

func filterQueriesForDeletedModel(queries []dao.DataQuery, modelName string) []dao.DataQuery {
	out := make([]dao.DataQuery, 0, len(queries))
	for _, query := range queries {
		if query.From == modelName {
			continue
		}
		if queryReferencesModel(query, modelName) {
			continue
		}
		out = append(out, query)
	}
	return out
}

func queryReferencesModel(query dao.DataQuery, modelName string) bool {
	prefix := modelName + "."
	for _, field := range query.Select {
		if strings.HasPrefix(field, prefix) {
			return true
		}
	}
	for _, condition := range query.Where {
		if strings.HasPrefix(condition.Field, prefix) {
			return true
		}
	}
	for _, order := range query.OrderBy {
		if strings.HasPrefix(order.Field, prefix) {
			return true
		}
	}
	return false
}

func splitQualifiedName(value string) (string, string, error) {
	parts := strings.Split(strings.TrimSpace(value), ".")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid qualified name %q", value)
	}
	return parts[0], parts[1], nil
}

func (s *Service) dataModelsForApp(appID string) ([]dao.DataModel, error) {
	item, err := s.activeManifestForApp(appID)
	if err != nil {
		return nil, err
	}
	return item.DataModels, nil
}

func (s *Service) dataRelationsForApp(appID string) ([]dao.DataRelation, error) {
	item, err := s.activeManifestForApp(appID)
	if err != nil {
		return nil, err
	}
	return item.Relations, nil
}

func (s *Service) dataQueriesForApp(appID string) ([]dao.DataQuery, error) {
	item, err := s.activeManifestForApp(appID)
	if err != nil {
		return nil, err
	}
	return item.Queries, nil
}

func (s *Service) manifestForApp(appID string) (manifest, error) {
	_, item, err := s.manifestPathForApp(appID)
	return item, err
}

func (s *Service) activeDir(appID string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.activeDirs[appID]
}

func (s *Service) activeManifestForApp(appID string) (manifest, error) {
	appID = strings.TrimSpace(appID)
	if !generatedAppUUIDPattern.MatchString(appID) {
		return manifest{}, ErrAppNotFound
	}
	appDir := s.activeDir(appID)
	if appDir == "" {
		return manifest{}, ErrAppNotFound
	}
	var item manifest
	var err error
	if s.draftRuntime {
		item, err = loadManifestFile(filepath.Join(appDir, "manifest.json"))
	} else {
		item, _, err = loadImmutableManifest(filepath.Join(appDir, "manifest.json"))
	}
	if err != nil {
		return manifest{}, fmt.Errorf("read published app manifest %q: %w", appID, err)
	}
	if item.ID != appID {
		return manifest{}, errors.New("published app manifest id does not match release")
	}
	return item, nil
}

func loadImmutableManifest(path string) (manifest, []byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return manifest{}, nil, err
	}
	item, normalizedRaw, changed, err := decodeManifestWithDir(filepath.Dir(path), raw)
	if err != nil {
		return manifest{}, nil, err
	}
	if changed {
		return manifest{}, nil, errors.New("published manifest is not normalized")
	}
	return item, normalizedRaw, nil
}

func loadManifestFile(path string) (manifest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return manifest{}, err
	}
	item, normalizedRaw, changed, err := decodeManifestWithDir(filepath.Dir(path), raw)
	if err != nil {
		return manifest{}, err
	}
	if changed {
		if err := os.WriteFile(path, normalizedRaw, 0o644); err != nil {
			return manifest{}, fmt.Errorf("normalize generated app manifest %q: %w", path, err)
		}
	}
	return item, nil
}

func appFromManifest(item manifest) (model.GeneratedApp, error) {
	appID := strings.TrimSpace(item.ID)
	if appID == "" {
		return model.GeneratedApp{}, errors.New("id is required")
	}
	if strings.ContainsAny(appID, `/\`) {
		return model.GeneratedApp{}, errors.New("id cannot contain path separators")
	}
	if !generatedAppUUIDPattern.MatchString(appID) {
		return model.GeneratedApp{}, errors.New("id must be a lowercase UUID")
	}
	export := firstNonEmpty(strings.TrimSpace(item.Export), "handle")
	backendSource := firstNonEmpty(strings.TrimSpace(item.BackendSource), "backend")
	backendModule := firstNonEmpty(strings.TrimSpace(item.BackendModule), "backend.wasm")
	now := time.Now().UnixMicro()
	return model.GeneratedApp{
		UUID:          appID,
		Name:          firstNonEmpty(strings.TrimSpace(item.Name), appID),
		Version:       strings.TrimSpace(item.Version),
		Description:   strings.TrimSpace(item.Description),
		Export:        export,
		FrontendEntry: fmt.Sprintf("/func-operation/generated-apps/%s/frontend.js", appID),
		BackendSource: backendSource,
		BackendModule: backendModule,
		CreatedAt:     now,
		UpdatedAt:     now,
	}, nil
}

func resolveRootDir(rootDir string) (string, error) {
	rootDir = strings.TrimSpace(rootDir)
	if rootDir == "" {
		return "", errors.New("generated app root is required")
	}
	if filepath.IsAbs(rootDir) {
		return rootDir, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}
	for {
		candidate := filepath.Join(cwd, rootDir)
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate, nil
		}
		parent := filepath.Dir(cwd)
		if parent == cwd {
			break
		}
		cwd = parent
	}
	return rootDir, nil
}

func readWASMModule(path string) ([]byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read wasm module %q: %w", path, err)
	}
	if filepath.Ext(path) == ".wasm" {
		return raw, nil
	}
	decoded, err := hex.DecodeString(strings.TrimSpace(string(raw)))
	if err != nil {
		return nil, fmt.Errorf("decode wasm module %q: %w", path, err)
	}
	return decoded, nil
}

func safeAppFilePath(rootDir string, appID string, fileName string) string {
	return filepath.Join(rootDir, filepath.Clean(appID), filepath.Clean(fileName))
}

func safeAppFrontendFilePath(rootDir string, appID string, fileName string) (string, error) {
	cleanAppID := filepath.Clean(appID)
	if cleanAppID == "." || cleanAppID == "" || strings.Contains(cleanAppID, string(filepath.Separator)) {
		return "", errors.New("invalid app id")
	}
	return safeFrontendFilePath(filepath.Join(rootDir, cleanAppID), fileName)
}

func safeFrontendFilePath(appDir string, fileName string) (string, error) {
	cleanFile := filepath.ToSlash(filepath.Clean(strings.TrimPrefix(strings.TrimSpace(fileName), "/")))
	if cleanFile == "." || cleanFile == "" {
		cleanFile = frontendFileFromEntry("")
	}
	if cleanFile == "frontend.js" {
		return filepath.Join(appDir, cleanFile), nil
	}
	if !strings.HasPrefix(cleanFile, "frontend/") || strings.Contains(cleanFile, "../") {
		return "", errors.New("frontend file path must be frontend.js or under frontend/")
	}
	if filepath.Ext(cleanFile) != ".js" {
		return "", errors.New("frontend module must be a .js file")
	}
	return filepath.Join(appDir, filepath.FromSlash(cleanFile)), nil
}

func frontendFileFromEntry(entry string) string {
	fileName := filepath.Base(strings.TrimSpace(entry))
	if fileName == "." || fileName == string(filepath.Separator) || fileName == "" {
		return "frontend.js"
	}
	return fileName
}

func moduleSize(rootDir string, app model.GeneratedApp) int {
	raw, err := os.ReadFile(safeAppFilePath(rootDir, app.UUID, app.BackendModule))
	if err != nil {
		return 0
	}
	if filepath.Ext(app.BackendModule) == ".wasm" {
		return len(raw)
	}
	decoded, err := hex.DecodeString(strings.TrimSpace(string(raw)))
	if err != nil {
		return 0
	}
	return len(decoded)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
