/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors

SPDX-License-Identifier: Apache-2.0
*/

package maintenance

import (
	"context"
	"fmt"

	"github.com/gardener/etcd-backup-restore/pkg/etcdutil"
	"github.com/gardener/etcd-backup-restore/pkg/miscellaneous"
	brtypes "github.com/gardener/etcd-backup-restore/pkg/types"

	"github.com/sirupsen/logrus"
)

// DefragmentCluster performs a rolling defragmentation across all etcd members of a live cluster.
// It always discovers endpoints from the cluster, requires the cluster to be healthy,
// and executes best-effort follower-first then leader-last ordering. If onSuccess is provided,
// it will be invoked after a successful defragmentation.
func DefragmentCluster(ctx context.Context, cfg *brtypes.EtcdConnectionConfig, logger *logrus.Entry, onSuccess func(context.Context) error) error {
	if cfg == nil {
		return fmt.Errorf("nil EtcdConnectionConfig")
	}

	factory := etcdutil.NewFactory(*cfg)

	maintenance, err := factory.NewMaintenance()
	if err != nil {
		return fmt.Errorf("failed to create etcd maintenance client: %w", err)
	}
	defer maintenance.Close()

	cluster, err := factory.NewCluster()
	if err != nil {
		return fmt.Errorf("failed to create etcd cluster client: %w", err)
	}
	defer cluster.Close()

	// Discover endpoints from current cluster membership
	endpoints, err := miscellaneous.GetAllEtcdEndpoints(ctx, cluster, cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to discover etcd endpoints: %w", err)
	}
	if len(endpoints) == 0 {
		return fmt.Errorf("no etcd endpoints discovered")
	}
	logger.Infof("Discovered etcd endpoints: %v", endpoints)

	// Ensure full cluster health before defragmentation
	isHealthy, err := miscellaneous.IsEtcdClusterHealthy(ctx, maintenance, cfg, endpoints, logger)
	if err != nil {
		return fmt.Errorf("failed to check etcd cluster health: %w", err)
	}
	if !isHealthy {
		return fmt.Errorf("etcd cluster is not healthy; aborting defragmentation")
	}

	// Log planned best-effort order (followers first, then leader)
	if leaderEndpoints, followerEndpoints, err := etcdutil.GetEtcdEndPointsSorted(ctx, maintenance, cluster, endpoints, logger); err != nil {
		logger.Warnf("failed to compute defragmentation order (will proceed anyway): %v", err)
	} else {
		logger.Infof("Planned defragmentation order (best-effort): followers=%v, then leader=%v", followerEndpoints, leaderEndpoints)
	}

	// Perform defragmentation with follower-first then leader-last ordering (best-effort)
	if err := etcdutil.DefragmentData(ctx, maintenance, cluster, endpoints, cfg.DefragTimeout.Duration, logger); err != nil {
		return fmt.Errorf("failed to defragment etcd cluster: %w", err)
	}

	// Invoke optional post-success callback
	if onSuccess != nil {
		if err := onSuccess(ctx); err != nil {
			logger.Warnf("post-defragmentation onSuccess callback failed: %v", err)
		}
	}

	logger.Info("Successfully defragmented etcd cluster.")
	return nil
}

// Defrag is a thin wrapper around DefragmentCluster without a post-success callback.
func Defrag(ctx context.Context, cfg *brtypes.EtcdConnectionConfig, logger *logrus.Entry) error {
	return DefragmentCluster(ctx, cfg, logger, nil)
}
