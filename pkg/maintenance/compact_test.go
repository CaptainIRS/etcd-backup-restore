package maintenance_test

import (
	"context"
	"time"

	"github.com/gardener/etcd-backup-restore/pkg/etcdutil"
	"github.com/gardener/etcd-backup-restore/pkg/maintenance"
	brtypes "github.com/gardener/etcd-backup-restore/pkg/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/mvcc"
)

var _ = Describe("Maintenance Compact", func() {
	var (
		cfg *brtypes.EtcdConnectionConfig
	)

	BeforeEach(func() {
		cfg = brtypes.NewEtcdConnectionConfig()
		cfg.Endpoints = endpoints
		cfg.ConnectionTimeout.Duration = 30 * time.Second
	})

	It("should compact to current revision and make earlier revisions unavailable", func() {
		// Create a KV client to read current revision and verify compact effect.
		clientFactory := etcdutil.NewFactory(*cfg)
		clientKV, err := clientFactory.NewKV()
		Expect(err).ShouldNot(HaveOccurred())
		defer clientKV.Close()

		// Read current revision.
		ctx, cancel := context.WithTimeout(testCtx, etcdDialTimeout)
		getResp, err := clientKV.Get(ctx, "maintenance-compact-rev-probe")
		cancel()
		Expect(err).ShouldNot(HaveOccurred())
		rev := getResp.Header.GetRevision()

		// Ensure we have a valid earlier revision to probe after compaction.
		// If rev <= 1, write something to bump the revision, then re-read.
		if rev <= 1 {
			ctx, cancel = context.WithTimeout(testCtx, etcdDialTimeout)
			_, err = clientKV.Put(ctx, "maintenance-compact-bump", "1")
			cancel()
			Expect(err).ShouldNot(HaveOccurred())

			ctx, cancel = context.WithTimeout(testCtx, etcdDialTimeout)
			getResp, err = clientKV.Get(ctx, "maintenance-compact-rev-probe")
			cancel()
			Expect(err).ShouldNot(HaveOccurred())
			rev = getResp.Header.GetRevision()
			Expect(rev).Should(BeNumerically(">=", 2))
		}

		// Perform compaction to current revision.
		err = maintenance.Compact(testCtx, cfg, logger)
		Expect(err).ShouldNot(HaveOccurred())

		// Accessing an older revision should fail with ErrCompacted.
		ctx, cancel = context.WithTimeout(testCtx, etcdDialTimeout)
		_, err = clientKV.Get(ctx, "maintenance-compact-rev-probe", clientv3.WithRev(rev-1))
		cancel()
		Expect(err).Should(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(mvcc.ErrCompacted.Error()))
	})

	It("should be a no-op when compaction was already performed up to current revision", func() {
		// First compaction
		err := maintenance.Compact(testCtx, cfg, logger)
		Expect(err).ShouldNot(HaveOccurred())

		// Second compaction should be a no-op (handled gracefully even if etcd returns ErrCompacted)
		err = maintenance.Compact(testCtx, cfg, logger)
		Expect(err).ShouldNot(HaveOccurred())
	})
})
