package workspacectx

import (
	"fmt"
	"path/filepath"

	"github.com/cengebretson/orc/internal/config"
	"github.com/cengebretson/orc/internal/workers"
)

type Context struct {
	Config    *config.Config
	Workers   []*workers.Worker
	WorkerIDs []string
}

func Load(root string) (*Context, error) {
	cfg, err := config.Load(root)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	allWorkers, err := workers.Load(filepath.Join(root, "workers"))
	if err != nil {
		return nil, fmt.Errorf("loading workers: %w", err)
	}

	return &Context{
		Config:    cfg,
		Workers:   allWorkers,
		WorkerIDs: WorkerIDs(allWorkers),
	}, nil
}

func LoadValidated(root string) (*Context, config.ValidationErrors, error) {
	ctx, err := Load(root)
	if err != nil {
		return nil, nil, err
	}
	errs := config.Validate(ctx.Config, ctx.WorkerIDs)
	if len(errs) > 0 {
		return ctx, errs, nil
	}
	return ctx, nil, nil
}

func WorkerIDs(allWorkers []*workers.Worker) []string {
	ids := make([]string, 0, len(allWorkers))
	for _, worker := range allWorkers {
		ids = append(ids, worker.ID)
	}
	return ids
}
