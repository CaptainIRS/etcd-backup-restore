/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors

SPDX-License-Identifier: Apache-2.0
*/

package maintenance_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/gardener/etcd-backup-restore/test/utils"

	"github.com/sirupsen/logrus"
	"go.etcd.io/etcd/embed"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	outputDir          = "../../test/output"
	etcdDir            = outputDir + "/maintenance.etcd"
	etcdDialTimeout    = time.Second * 30
	embeddedEtcdPortNo = "9090"
)

var (
	testCtx   = context.Background()
	logger    = logrus.New().WithField("suite", "maintenance")
	etcd      *embed.Etcd
	endpoints []string
	err       error
)

func TestMaintenance(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Maintenance Suite")
}

var _ = SynchronizedBeforeSuite(func() []byte {
	logger.Logger.Out = GinkgoWriter
	err = os.RemoveAll(outputDir)
	Expect(err).ShouldNot(HaveOccurred())

	etcd, err = utils.StartEmbeddedEtcd(testCtx, etcdDir, logger, utils.DefaultEtcdName, embeddedEtcdPortNo)
	Expect(err).ShouldNot(HaveOccurred())
	endpoints = []string{etcd.Clients[0].Addr().String()}
	logger.Infof("endpoints: %s", endpoints)
	return nil
}, func(_ []byte) {})

var _ = SynchronizedAfterSuite(func() {}, func() {
	etcd.Server.Stop()
	etcd.Close()
})
