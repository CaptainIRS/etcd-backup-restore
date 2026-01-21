/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors

SPDX-License-Identifier: Apache-2.0
*/

package maintenance_test

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gardener/etcd-backup-restore/pkg/etcdutil"
	"github.com/gardener/etcd-backup-restore/pkg/maintenance"
	brtypes "github.com/gardener/etcd-backup-restore/pkg/types"
	"github.com/gardener/etcd-backup-restore/test/utils"

	"go.etcd.io/etcd/clientv3"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Maintenance Defragmentation", func() {
	var (
		etcdConnectionConfig *brtypes.EtcdConnectionConfig
		keyPrefix            = "/maintenance/defrag/key-"
		valuePrefix          = "val"
	)

	BeforeEach(func() {
		etcdConnectionConfig = brtypes.NewEtcdConnectionConfig()
		etcdConnectionConfig.Endpoints = endpoints
		etcdConnectionConfig.ConnectionTimeout.Duration = 30 * time.Second
		etcdConnectionConfig.SnapshotTimeout.Duration = 30 * time.Second
		etcdConnectionConfig.DefragTimeout.Duration = 30 * time.Second
	})

	Context("Defragmentation", func() {
		BeforeEach(func() {
			// Create fragmentation by writing many keys and then deleting half.
			now := time.Now().Unix()
			clientFactory := etcdutil.NewFactory(*etcdConnectionConfig)
			clientKV, err := clientFactory.NewKV()
			Expect(err).ShouldNot(HaveOccurred())
			defer clientKV.Close()

			logger.Infof("etcdConnectionConfig %v, Endpoint %v", etcdConnectionConfig, endpoints)

			for index := 0; index <= 1000; index++ {
				ctx, cancel := context.WithTimeout(testCtx, etcdConnectionConfig.ConnectionTimeout.Duration)
				_, err = clientKV.Put(ctx, fmt.Sprintf("%s%d%d", keyPrefix, now, index), valuePrefix)
				cancel()
				Expect(err).ShouldNot(HaveOccurred())
			}
			for index := 0; index <= 500; index++ {
				ctx, cancel := context.WithTimeout(testCtx, etcdConnectionConfig.ConnectionTimeout.Duration)
				_, err = clientKV.Delete(ctx, fmt.Sprintf("%s%d%d", keyPrefix, now, index))
				cancel()
				Expect(err).ShouldNot(HaveOccurred())
			}
		})

		It("should defragment cluster and reduce size of DB within time", func() {
			clientFactory := etcdutil.NewFactory(*etcdConnectionConfig)

			clientMaintenance, err := clientFactory.NewMaintenance()
			Expect(err).ShouldNot(HaveOccurred())
			defer clientMaintenance.Close()

			clientKV, err := clientFactory.NewKV()
			Expect(err).ShouldNot(HaveOccurred())
			defer clientKV.Close()

			ctx, cancel := context.WithTimeout(testCtx, etcdDialTimeout)
			oldStatus, err := clientMaintenance.Status(ctx, endpoints[0])
			cancel()
			Expect(err).ShouldNot(HaveOccurred())
			oldDBSize := oldStatus.DbSize
			oldRevision := oldStatus.Header.GetRevision()

			// Compact to the current revision to maximize impact of defrag.
			_, err = clientKV.Compact(testCtx, oldRevision, clientv3.WithCompactPhysical())
			Expect(err).ShouldNot(HaveOccurred())

			// Run defragmentation across the cluster.
			err = maintenance.DefragmentCluster(testCtx, etcdConnectionConfig, logger, nil)
			Expect(err).ShouldNot(HaveOccurred())

			ctx, cancel = context.WithTimeout(testCtx, etcdDialTimeout)
			newStatus, err := clientMaintenance.Status(ctx, endpoints[0])
			cancel()
			Expect(err).ShouldNot(HaveOccurred())

			Expect(newStatus.DbSize).Should(BeNumerically("<", oldDBSize))
			Expect(newStatus.Header.GetRevision()).Should(BeNumerically("==", oldRevision))
		})

		It("should keep size of DB same in case of timeout", func() {
			// Set a very short defrag timeout so that defragmentation fails.
			etcdConnectionConfig.DefragTimeout.Duration = time.Microsecond
			clientFactory := etcdutil.NewFactory(*etcdConnectionConfig)

			clientMaintenance, err := clientFactory.NewMaintenance()
			Expect(err).ShouldNot(HaveOccurred())
			defer clientMaintenance.Close()

			ctx, cancel := context.WithTimeout(testCtx, etcdDialTimeout)
			oldStatus, err := clientMaintenance.Status(ctx, endpoints[0])
			cancel()
			Expect(err).ShouldNot(HaveOccurred())
			oldDBSize := oldStatus.DbSize
			oldRevision := oldStatus.Header.GetRevision()

			// Defragmentation is expected to error due to timeout.
			err = maintenance.DefragmentCluster(testCtx, etcdConnectionConfig, logger, nil)
			Expect(err).Should(HaveOccurred())

			ctx, cancel = context.WithTimeout(testCtx, etcdDialTimeout)
			newStatus, err := clientMaintenance.Status(ctx, endpoints[0])
			cancel()
			Expect(err).ShouldNot(HaveOccurred())

			Expect(newStatus.Header.GetRevision()).Should(BeNumerically("==", oldRevision))
			Expect(newStatus.DbSize).Should(Equal(oldDBSize))
		})

		It("should defragment cluster and invoke onSuccess callback", func() {
			// Populate ETCD with additional data to ensure there's something to defrag.
			populatorCtx, cancelPopulator := context.WithTimeout(testCtx, 5*time.Second)
			defer cancelPopulator()
			resp := &utils.EtcdDataPopulationResponse{}
			wg := &sync.WaitGroup{}
			wg.Add(1)
			go utils.PopulateEtcdWithWaitGroup(populatorCtx, wg, logger, endpoints, "", "", resp)
			Expect(resp.Err).ShouldNot(HaveOccurred())
			wg.Wait()

			clientFactory := etcdutil.NewFactory(*etcdConnectionConfig)
			clientMaintenance, err := clientFactory.NewMaintenance()
			Expect(err).ShouldNot(HaveOccurred())
			defer clientMaintenance.Close()

			statusReqCtx, cancelStatusReq := context.WithTimeout(testCtx, etcdDialTimeout)
			oldStatus, err := clientMaintenance.Status(statusReqCtx, endpoints[0])
			cancelStatusReq()
			Expect(err).ShouldNot(HaveOccurred())
			oldDBSize := oldStatus.DbSize
			oldRevision := oldStatus.Header.GetRevision()

			defragCount := 0
			err = maintenance.DefragmentCluster(testCtx, etcdConnectionConfig, logger, func(_ context.Context) error {
				defragCount++
				return nil
			})
			Expect(err).ShouldNot(HaveOccurred())

			statusReqCtx, cancelStatusReq = context.WithTimeout(testCtx, etcdDialTimeout)
			newStatus, err := clientMaintenance.Status(statusReqCtx, endpoints[0])
			cancelStatusReq()
			Expect(err).ShouldNot(HaveOccurred())

			Expect(defragCount).Should(Equal(1))
			Expect(newStatus.DbSize).Should(BeNumerically("<", oldDBSize))
			Expect(newStatus.Header.GetRevision()).Should(BeNumerically("==", oldRevision))
		})
	})
})
