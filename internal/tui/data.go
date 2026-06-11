package tui

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/cengebretson/orc/internal/config"
	"github.com/cengebretson/orc/internal/featurelist"
	"github.com/cengebretson/orc/internal/health"
	"github.com/cengebretson/orc/internal/state"
	"github.com/cengebretson/orc/internal/workers"
	tea "github.com/charmbracelet/bubbletea"
)

func loadData(root string) tea.Cmd {
	return func() tea.Msg {
		features := collectFeatures(root)
		report := health.Run(root)

		// build workflow chains from workflows.yaml
		workflowCfg, _ := config.Load(root)
		var chains []workflowChain
		allStages := map[string]bool{}
		for _, wfName := range workflowCfg.Names() {
			stages := workflowCfg.StageNames(wfName)
			var steps []routeStep
			inThisChain := map[string]bool{}
			for _, stageName := range stages {
				sc, _ := workflowCfg.StageConfig(wfName, stageName)
				steps = append(steps, routeStep{name: stageName, advance: sc.Advance, workerID: sc.Worker})
				inThisChain[stageName] = true
				allStages[stageName] = true
			}
			// loop stages — derived from Loop blocks on pipeline stages
			var loops []repairLoop
			var repairs []repairStep
			for _, sc := range workflowCfg.Stages(wfName) {
				if sc.Loop == nil || !inThisChain[sc.Name] {
					continue
				}
				loops = append(loops, repairLoop{name: sc.Loop.Via, target: sc.Name})
				repairs = append(repairs, repairStep{
					name:       sc.Loop.Via,
					workerID:   sc.Loop.Worker,
					repairs:    sc.Name,
					maxRetries: sc.Loop.Max,
				})
			}
			chains = append(chains, workflowChain{name: wfName, steps: steps, loops: loops, repairSteps: repairs})
		}
		// fallback: flat list of all stage files
		if len(chains) == 0 {
			stagesDir := filepath.Join(root, "stages")
			entries, _ := os.ReadDir(stagesDir)
			var steps []routeStep
			for _, e := range entries {
				if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
					steps = append(steps, routeStep{name: strings.TrimSuffix(e.Name(), ".md")})
				}
			}
			chains = []workflowChain{{name: "", steps: steps}}
		}
		// collect all stage names for display
		var wfNames []string
		for stageName := range allStages {
			wfNames = append(wfNames, stageName)
		}

		// worker names
		allWorkers, _ := workers.Load(filepath.Join(root, "workers"))
		var workerNames []string
		for _, w := range allWorkers {
			name := w.Name
			if name == "" {
				name = w.ID
			}
			workerNames = append(workerNames, name)
		}

		var repos []config.Repo
		if cfg, err := config.Load(root); err == nil {
			repos = cfg.Repos
		}

		// section items for navigable file viewer
		si := map[string][]sectionItem{}

		// workflows: one entry per workflow chain; path="" signals detail view
		for _, c := range chains {
			si["workflows"] = append(si["workflows"], sectionItem{label: c.name, path: ""})
		}

		// workers: actual .md files in workers/
		workersDir := filepath.Join(root, "workers")
		if entries, err := filepath.Glob(filepath.Join(workersDir, "*.md")); err == nil {
			for _, p := range entries {
				base := filepath.Base(p)
				if base == "_template.md" {
					continue
				}
				id := strings.TrimSuffix(base, ".md")
				label := id
				for _, w := range allWorkers {
					if w.ID == id && w.Name != "" {
						label = w.Name
						break
					}
				}
				si["workers"] = append(si["workers"], sectionItem{label: label, path: p})
			}
		}

		// routes: ROUTER.md as a single item
		routerPath := filepath.Join(root, "ROUTER.md")
		if _, err := os.Stat(routerPath); err == nil {
			si["routes"] = []sectionItem{{label: "ROUTER.md", path: routerPath}}
		}

		return dataMsg{
			features:        features,
			healthItems:     report.Results,
			workflowNames:   wfNames,
			workerNames:     workerNames,
			allWorkers:      allWorkers,
			workflows:       chains,
			repos:           repos,
			sectionItems:    si,
			refreshInterval: workflowCfg.TuiRefreshInterval(),
			quotes:          workflowCfg.Settings.Quotes,
		}
	}
}

func collectFeatures(root string) []*featureRow {
	features, _ := featurelist.Collect(root, featurelist.Options{IncludeArchived: true})
	rows := make([]*featureRow, 0, len(features))
	for _, f := range features {
		if f.LoadError != nil {
			continue
		}
		rows = append(rows, &featureRow{
			s:              f.State,
			featureDir:     f.FeatureDir,
			workflow:       f.Workflow,
			stageLoopLabel: f.StageLoopLabel,
			workerName:     f.WorkerName,
			tmuxLive:       f.TmuxLive,
			hasIssues:      f.HasIssues,
		})
	}
	return rows
}

func buildFileList(featureDir string, s *state.State) []detailFile {
	candidates := []detailFile{
		{"TICKET.md", filepath.Join(featureDir, "TICKET.md")},
		{"SPEC.md", filepath.Join(featureDir, "SPEC.md")},
		{"PLAN.md", filepath.Join(featureDir, "PLAN.md")},
		{"DECISIONS.md", filepath.Join(featureDir, "DECISIONS.md")},
		{"impl/QA_HANDOFF.md", filepath.Join(featureDir, "impl", "QA_HANDOFF.md")},
		{"impl/REVIEW.md", filepath.Join(featureDir, "impl", "REVIEW.md")},
		{"impl/PR.md", filepath.Join(featureDir, "impl", "PR.md")},
		{"qa/QA_PLAN.md", filepath.Join(featureDir, "qa", "QA_PLAN.md")},
		{"qa/QA_RESULT.md", filepath.Join(featureDir, "qa", "QA_RESULT.md")},
	}
	core := map[string]bool{"TICKET.md": true, "SPEC.md": true, "PLAN.md": true}
	var out []detailFile
	for _, f := range candidates {
		if fileExists(f.path) || core[f.label] {
			out = append(out, f)
		}
	}
	return out
}
