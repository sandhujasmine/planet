package main

import (
	"context"
	"time"

	"github.com/gravitational/planet/lib/constants"
	"github.com/gravitational/planet/lib/utils"
	"github.com/gravitational/satellite/cmd"
	"github.com/gravitational/trace"

	log "github.com/sirupsen/logrus"
	api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	kube "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

func runEnvironmentLoop(ctx context.Context) error {
	// FIXME: use the appropriate kubeconfig
	client, err := cmd.GetKubeClientFromPath(constants.CoreDNSConfigPath)
	if err != nil {
		return trace.Wrap(err)
	}
	m := environMonitor{client: client}
	go m.monitorEnvironmentConfigMap(ctx)
	return nil
}

func (r environMonitor) monitorEnvironmentConfigMap(ctx context.Context) {
	_, controller := cache.NewInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				options.FieldSelector = fields.OneTermEqualSelector(
					"metadata.name",
					constants.EnvironmentConfigMapName,
				).String()
				return r.client.CoreV1().ConfigMaps(metav1.NamespaceSystem).List(options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				options.FieldSelector = fields.OneTermEqualSelector(
					"metadata.name",
					constants.EnvironmentConfigMapName,
				).String()
				return r.client.CoreV1().ConfigMaps(metav1.NamespaceSystem).Watch(options)
			},
		},
		&api.ConfigMap{},
		15*time.Minute,
		cache.ResourceEventHandlerFuncs{
			AddFunc:    r.add,
			UpdateFunc: r.update,
		},
	)
	controller.Run(ctx.Done())
}

func (r environMonitor) add(obj interface{}) {
}

func (r environMonitor) update(oldObj, newObj interface{}) {
	switch configmap := newObj.(type) {
	case *api.ConfigMap:
		if err := updateEnvironment(configmap.Data); err != nil {
			log.WithError(err).Warn("Failed to update environment.")
		}
	}
}

func updateEnvironment(kvs map[string]string) error {
	log.WithField("kvs", kvs).Info("Update environment.")
	// TODO: read and update the list of environment variables

	var environ string
	err := utils.SafeWriteFile(ContainerEnvironmentFile, []byte(environ), SharedFileMask)
	if err != nil {
		return trace.Wrap(err)
	}

	// TODO: restart all affected systemd units

	return nil
}

type environMonitor struct {
	client *kube.Clientset
}
