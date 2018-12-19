package main

import (
	"context"
	"fmt"

	"github.com/gravitational/planet/lib/box"
	"github.com/gravitational/planet/lib/constants"
	"github.com/gravitational/planet/lib/utils"

	etcdconf "github.com/gravitational/coordinate/config"
	"github.com/gravitational/trace"
	log "github.com/sirupsen/logrus"
)

func updateEnvironment(kvs map[string]string) error {
	log.WithField("kvs", kvs).Info("Update environment.")

	env, err := box.ReadEnvironment(ContainerEnvironmentFile)
	if err != nil {
		return trace.Wrap(err, "failed to read cluster environment file")
	}

	for k, v := range kvs {
		env.Upsert(k, v)
	}

	err = utils.SafeWriteFile(ContainerEnvironmentFile, env, constants.SharedReadMask)
	if err != nil {
		return trace.Wrap(err, "failed to write cluster environment file")
	}

	leaderAddr, err := getCurrentLeader(env)
	if err != nil {
		return trace.Wrap(err)
	}

	isLeader := env.Get(EnvPublicIP) == leaderAddr
	services := append(commonServices, kubeServices...)
	if isLeader {
		services = append(services, kubeMasterServices...)
	}
	return restartServices(context.Background(), services)
}

func restartServices(ctx context.Context, services []string) error {
	for _, service := range services {
		logger := log.WithField("service", service)
		active, out, err := utils.ServiceIsActive(ctx, service)
		if err != nil {
			logger.WithFields(log.Fields{
				log.ErrorKey: err,
				"output":     string(out),
			}).Warn("Failed to check status.")
			continue
		}
		if !active {
			continue
		}
		logger.Info("Will restart.")
		out, err = utils.ServiceCtl(ctx, "restart", service, utils.Blocking(true))
		if err != nil {
			logger.WithFields(log.Fields{
				log.ErrorKey: err,
				"output":     string(out),
			}).Warn("Failed to restart.")
		}
	}
	return nil
}

func getCurrentLeader(env box.EnvVars) (addr string, err error) {
	config := &etcdconf.Config{
		Endpoints: []string{DefaultEtcdEndpoints},
		CAFile:    "/var/state/root.cert",
		CertFile:  "/var/state/etcd.cert",
		KeyFile:   "/var/state/etcd.key",
	}
	key := fmt.Sprintf("/planet/cluster/%v/master", env.Get(EnvClusterID))
	return getLeader(context.TODO(), key, config)
}

var commonServices = []string{
	"serf.service",
	"flanneld.service",
	"docker.service",
	"etcd.service",
	"planet-agent.service",
}

var kubeServices = []string{
	"kube-kubelet.service",
	"kube-proxy.service",
}

var kubeMasterServices = []string{
	"kube-apiserver.service",
	"kube-controller-manager.service",
	"kube-scheduler.service",
}
