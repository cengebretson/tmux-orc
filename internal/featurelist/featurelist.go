package featurelist

import (
	"os"
	"path/filepath"

	"github.com/cengebretson/orc/internal/config"
	"github.com/cengebretson/orc/internal/state"
	"github.com/cengebretson/orc/internal/tmux"
	"github.com/cengebretson/orc/internal/workers"
)

type Feature struct {
	State      *state.State
	FeatureDir string
	Archived   bool
	WorkerID   string
	WorkerName string
	TmuxLive   bool
	LoadError  error
}

type Options struct {
	IncludeArchived bool
	TmuxAvailable   func() bool
	ListSessions    func() []string
}

func Collect(root string, opts Options) ([]*Feature, error) {
	if opts.TmuxAvailable == nil {
		opts.TmuxAvailable = tmux.Available
	}
	if opts.ListSessions == nil {
		opts.ListSessions = tmux.ListSessions
	}

	cfg, _ := config.Load(root)
	allWorkers, _ := workers.Load(filepath.Join(root, "workers"))
	activeSessions := map[string]bool{}
	if opts.TmuxAvailable() {
		for _, name := range opts.ListSessions() {
			activeSessions[name] = true
		}
	}

	featuresDir := filepath.Join(root, "features")
	var out []*Feature
	if err := collectDir(root, featuresDir, false, cfg, allWorkers, activeSessions, &out); err != nil {
		return nil, err
	}
	if opts.IncludeArchived {
		if err := collectDir(root, filepath.Join(featuresDir, "_archive"), true, cfg, allWorkers, activeSessions, &out); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func collectDir(root, dir string, archived bool, cfg *config.Config, allWorkers []*workers.Worker, activeSessions map[string]bool, out *[]*Feature) error {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	for _, e := range entries {
		if !e.IsDir() || e.Name() == "_template" || e.Name() == "_archive" {
			continue
		}
		featureDir := filepath.Join(dir, e.Name())
		s, err := state.Load(featureDir)
		if err != nil {
			*out = append(*out, &Feature{
				FeatureDir: featureDir,
				Archived:   archived,
				LoadError:  err,
			})
			continue
		}

		workerID := resolveWorkerID(cfg, s)
		*out = append(*out, &Feature{
			State:      s,
			FeatureDir: featureDir,
			Archived:   archived,
			WorkerID:   workerID,
			WorkerName: resolveWorkerName(allWorkers, workerID),
			TmuxLive:   s.Runtime.Tmux != nil && activeSessions[s.Runtime.Tmux.Session],
		})
	}
	return nil
}

func resolveWorkerID(cfg *config.Config, s *state.State) string {
	if s.Stage.Worker != "" {
		return s.Stage.Worker
	}
	if cfg == nil {
		return ""
	}
	workflow := s.Workflow
	if workflow == "" {
		workflow = cfg.DefaultWorkflow()
	}
	if sc, ok := cfg.StageConfig(workflow, s.Stage.Name); ok {
		return sc.Worker
	}
	return ""
}

func resolveWorkerName(allWorkers []*workers.Worker, workerID string) string {
	if workerID == "" {
		return ""
	}
	if w := workers.FindByID(allWorkers, workerID); w != nil {
		return w.Name
	}
	return workerID
}
