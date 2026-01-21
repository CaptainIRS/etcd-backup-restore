/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors

SPDX-License-Identifier: Apache-2.0
*/

package maintenance

import (
	"context"
	"fmt"
	"strings"

	"github.com/gardener/etcd-backup-restore/pkg/etcdutil"
	brtypes "github.com/gardener/etcd-backup-restore/pkg/types"

	"github.com/sirupsen/logrus"
	"go.etcd.io/etcd/mvcc"
)

// Compact performs an etcd maintenance compaction on a live cluster up to the current revision.
// It determines the current revision using a lightweight GET and then runs a logical compaction.
// Best used for reclaiming storage from old revisions without changing snapshot backups.
func Compact(ctx context.Context, cfg *brtypes.EtcdConnectionConfig, logger *logrus.Entry) error {
	if cfg == nil {
		return fmt.Errorf("nil EtcdConnectionConfig")
	}

	factory := etcdutil.NewFactory(*cfg)

	kv, err := factory.NewKV()
	if err != nil {
		return fmt.Errorf("failed to create etcd KV client: %w", err)
	}
	defer kv.Close()

	// Determine current revision via a lightweight GET
	revCtx, cancel := context.WithTimeout(ctx, cfg.ConnectionTimeout.Duration)
	resp, err := kv.Get(revCtx, "foo") // key is arbitrary; just need the header revision
	cancel()
	if err != nil {
		return fmt.Errorf("failed to obtain current revision from etcd: %w", err)
	}
	revision := resp.Header.GetRevision()
	logger.Infof("Compacting etcd to current revision: %d", revision)

	// Logical compaction up to current revision
	cmpCtx, cancel := context.WithTimeout(ctx, cfg.ConnectionTimeout.Duration)
	_, err = kv.Compact(cmpCtx, revision)
	cancel()
	if err != nil {
		// Handle already-compacted case as a no-op
		if strings.Contains(err.Error(), mvcc.ErrCompacted.Error()) {
			logger.Warnf("Compaction no-op: %v", err)
			return nil
		}
		return fmt.Errorf("failed to compact etcd: %w", err)
	}

	logger.Info("Successfully compacted etcd to current revision.")
	return nil
}
