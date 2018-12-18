package main

import (
	"context"

	"github.com/gravitational/planet/lib/box"
	"github.com/gravitational/planet/lib/constants"
	"github.com/gravitational/planet/lib/utils"

	"github.com/gravitational/trace"
	log "github.com/sirupsen/logrus"
)

func updateEnvironment(kvs map[string]string) error {
	log.WithField("kvs", kvs).Info("Update environment.")
	// Create a backup of the current environment
	err := utils.CopyFileWithPerms(ContainerEnvironmentFileBackup, ContainerEnvironmentFile, constants.SharedReadMask)
	if err != nil {
		return trace.Wrap(err)
	}

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

	return restartServices(context.Background())
}

func restartServices(ctx context.Context) error {
	// FIXME: validate whether this node is a leader or merely rely on service status?
	for _, service := range []string{
		"serf.service",
		"flanneld.service",
		"docker.service",
		"etcd.service",
		"kube-apiserver.service",
		"kube-kubelet.service",
		"kube-proxy.service",
		"kube-controller-manager.service",
		"kube-scheduler.service",
		"planet-agent.service",
	} {
		log := log.WithField("service", service)
		active, out, err := utils.ServiceIsActive(ctx, service)
		if err != nil {
			log.Warnf("Failed to check status: %s (%v).", out, err)
			continue
		}
		if !active {
			continue
		}
		log.Info("Will restart.")
		out, err = utils.ServiceCtl(ctx, "restart", service, utils.Blocking(true))
		if err != nil {
			log.WithError(err).Warnf("Failed to restart: %s.", out)
		}
	}
	return nil
}
