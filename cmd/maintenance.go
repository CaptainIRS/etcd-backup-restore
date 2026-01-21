/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors

SPDX-License-Identifier: Apache-2.0
*/

package cmd

import (
	"context"

	"github.com/gardener/etcd-backup-restore/pkg/maintenance"
	"github.com/go-logr/logr"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	runtimelog "sigs.k8s.io/controller-runtime/pkg/log"
)

// NewMaintenanceCommand returns the parent command for live etcd maintenance operations.
func NewMaintenanceCommand(ctx context.Context) *cobra.Command {
	maintenanceCmd := &cobra.Command{
		Use:   "maintenance",
		Short: "perform live etcd maintenance operations",
		Long:  "Maintenance subcommands perform live etcd cluster maintenance such as compaction and defragmentation.",
		Run: func(cmd *cobra.Command, _ []string) {
			// Show help when invoked without a subcommand.
			_ = cmd.Help()
		},
	}

	maintenanceCmd.AddCommand(NewMaintenanceCompactCommand(ctx))
	maintenanceCmd.AddCommand(NewMaintenanceDefragCommand(ctx))

	return maintenanceCmd
}

func NewMaintenanceCompactCommand(ctx context.Context) *cobra.Command {
	opts := newMaintenanceCompactOptions()

	cmd := &cobra.Command{
		Use:   "compact",
		Short: "compact a live etcd cluster by removing old revisions",
		Long:  "Performs an etcd maintenance compaction against a live cluster, compacting up to the current revision.",
		Run: func(_ *cobra.Command, _ []string) {
			logger := logrus.NewEntry(logrus.New())
			runtimelog.SetLogger(logr.New(runtimelog.NullLogSink{}))

			if err := opts.validate(); err != nil {
				logger.Fatalf("failed to validate options: %v", err)
				return
			}

			if err := maintenance.Compact(ctx, opts.etcdConnectionConfig, logger); err != nil {
				logger.Fatalf("failed to compact etcd: %v", err)
				return
			}
		},
	}

	opts.addFlags(cmd.Flags())
	return cmd
}

func NewMaintenanceDefragCommand(ctx context.Context) *cobra.Command {
	opts := newMaintenanceDefragOptions()

	cmd := &cobra.Command{
		Use:   "defrag",
		Short: "defragment a live etcd cluster to reclaim backend storage space",
		Long:  "Performs a rolling defragmentation across etcd members of a live cluster, arranging to defragment followers first and leader last (best-effort).",
		Run: func(_ *cobra.Command, _ []string) {
			logger := logrus.NewEntry(logrus.New())
			runtimelog.SetLogger(logr.New(runtimelog.NullLogSink{}))

			if err := opts.validate(); err != nil {
				logger.Fatalf("failed to validate options: %v", err)
				return
			}

			if err := maintenance.Defrag(ctx, opts.etcdConnectionConfig, logger); err != nil {
				logger.Fatalf("failed to defragment etcd cluster: %v", err)
				return
			}
		},
	}

	opts.addFlags(cmd.Flags())
	return cmd
}
