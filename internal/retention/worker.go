package retention

import (
	"context"
	"time"

	"knit/internal/audit"
	"knit/internal/config"
	"knit/internal/storage"
)

type Worker struct {
	cfg       config.Config
	store     storage.Store
	artifacts *storage.ArtifactStore
	audit     *audit.Logger
}

func NewWorker(cfg config.Config, store storage.Store, artifacts *storage.ArtifactStore, auditLogger *audit.Logger) *Worker {
	return &Worker{cfg: cfg, store: store, artifacts: artifacts, audit: auditLogger}
}

func (w *Worker) Run(ctx context.Context) {
	if !w.cfg.PurgeScheduleEnabled {
		return
	}
	interval := w.cfg.PurgeInterval
	if interval <= 0 {
		interval = 30 * time.Minute
	}

	w.RunOnce()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.RunOnce()
		}
	}
}

func (w *Worker) RunOnce() {
	now := time.Now().UTC()
	purgeDetails := map[string]any{}

	if w.cfg.StructuredRetention > 0 {
		cutoff := now.Add(-w.cfg.StructuredRetention)
		if n, err := w.store.PurgeSessionsOlderThan(cutoff); err == nil {
			purgeDetails["sessions_deleted"] = n
		}
		if n, err := w.store.PurgeCanonicalPackagesOlderThan(cutoff); err == nil {
			purgeDetails["packages_deleted"] = n
		}
		if n, err := w.store.PurgeSubmissionsOlderThan(cutoff); err == nil {
			purgeDetails["submissions_deleted"] = n
		}
	}
	if w.cfg.ScreenshotRetention > 0 {
		cutoff := now.Add(-w.cfg.ScreenshotRetention)
		if n, err := w.artifacts.PurgeOlderThan("screenshot", cutoff); err == nil {
			purgeDetails["screenshots_deleted"] = n
		}
	}
	if w.cfg.AudioRetention > 0 {
		cutoff := now.Add(-w.cfg.AudioRetention)
		if n, err := w.artifacts.PurgeOlderThan("audio", cutoff); err == nil {
			purgeDetails["audio_deleted"] = n
		}
	}
	if w.cfg.VideoRetention > 0 {
		cutoff := now.Add(-w.cfg.VideoRetention)
		if n, err := w.artifacts.PurgeOlderThan("clip", cutoff); err == nil {
			purgeDetails["clips_deleted"] = n
		}
	}
	if w.cfg.ArtifactMaxFiles > 0 {
		if n, err := w.artifacts.PruneToLimit(w.cfg.ArtifactMaxFiles); err == nil {
			purgeDetails["artifacts_pruned"] = n
		}
	}
	if len(purgeDetails) > 0 {
		_ = w.audit.Write(audit.Event{Type: "retention_purge", Details: purgeDetails})
	}
}
