// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package defragmentor

import (
	"context"
	"time"

	"github.com/gardener/etcd-backup-restore/pkg/maintenance"
	brtypes "github.com/gardener/etcd-backup-restore/pkg/types"

	cron "github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
)

// CallbackFunc is type decalration for callback function for defragmentor
type CallbackFunc func(ctx context.Context, isFinal bool) (*brtypes.Snapshot, error)

// defragmentorJob implement the cron.Job for etcd defragmentation.
type defragmentorJob struct {
	ctx                  context.Context
	etcdConnectionConfig *brtypes.EtcdConnectionConfig
	logger               *logrus.Entry
	callback             CallbackFunc
}

// NewDefragmentorJob returns the new defragmentor job.
func NewDefragmentorJob(ctx context.Context, etcdConnectionConfig *brtypes.EtcdConnectionConfig, logger *logrus.Entry, callback CallbackFunc) cron.Job {
	return &defragmentorJob{
		ctx:                  ctx,
		etcdConnectionConfig: etcdConnectionConfig,
		logger:               logger.WithField("job", "defragmentor"),
		callback:             callback,
	}
}

func (d *defragmentorJob) Run() {
	ticker := time.NewTicker(brtypes.DefragRetryPeriod)
	defer ticker.Stop()

waitLoop:
	for {
		select {
		case <-d.ctx.Done():
			return
		case <-ticker.C:
			// Attempt defragmentation using consolidated maintenance logic with per-run client creation.
			onSuccess := func(ctx context.Context) error {
				if d.callback != nil {
					if _, err := d.callback(ctx, false); err != nil {
						d.logger.Warnf("defragmentation callback failed with error: %v", err)
						return err
					}
				}
				return nil
			}

			d.logger.Infof("Starting defragmentation attempt...")
			if err := maintenance.DefragmentCluster(d.ctx, d.etcdConnectionConfig, d.logger, onSuccess); err != nil {
				d.logger.Warnf("failed to defragment data with error: %v", err)
				continue
			}

			// Success; exit wait loop.
			break waitLoop
		}
	}
}

// DefragDataPeriodically defragments the data directory of each etcd member.
func DefragDataPeriodically(ctx context.Context, etcdConnectionConfig *brtypes.EtcdConnectionConfig, defragmentationSchedule cron.Schedule, callback CallbackFunc, logger *logrus.Entry) {
	defragmentorJob := NewDefragmentorJob(ctx, etcdConnectionConfig, logger, callback)
	// TODO: Sync logrus logger to cron logger
	jobRunner := cron.New(cron.WithChain(cron.SkipIfStillRunning(cron.DefaultLogger)))
	jobRunner.Schedule(defragmentationSchedule, defragmentorJob)

	jobRunner.Start()

	<-ctx.Done()
	logger.Info("Closing defragmentor.")
	jobRunnerCtx := jobRunner.Stop()
	<-jobRunnerCtx.Done()
}
