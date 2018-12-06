package main

import (
	"context"
	"reflect"
	"time"

	"github.com/gravitational/planet/lib/box"
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
	client, err := cmd.GetKubeClientFromPath(constants.SchedulerConfigPath)
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
	updateEnvironmentConfigMap(nil, obj)
}

func (r environMonitor) update(oldObj, newObj interface{}) {
	updateEnvironmentConfigMap(oldObj, newObj)
}

func updateEnvironmentConfigMap(oldObj, newObj interface{}) {
	oldConfigmap, ok := oldObj.(*api.ConfigMap)
	if !ok && oldObj != nil {
		log.Warnf("Unexpected resource type %T.", oldObj)
	}
	newConfigmap, ok := newObj.(*api.ConfigMap)
	if !ok {
		log.Warnf("Unexpected resource type %T.", newObj)
	}
	if oldObj != nil && reflect.DeepEqual(oldConfigmap.Data, newConfigmap.Data) {
		// Ignore idempotent update
		// TODO: check the container environment to avoid restarting services
		// on restart of the agent service
		return
	}
	if err := updateEnvironment(context.TODO(), newConfigmap.Data); err != nil {
		log.WithError(err).Warn("Failed to update environment.")
	}
}

func updateEnvironment(ctx context.Context, kvs map[string]string) error {
	log.WithField("kvs", kvs).Info("Update environment.")

	env, err := box.ReadEnvironment(ContainerEnvironmentFile)
	if err != nil {
		return trace.Wrap(err)
	}

	for k, v := range kvs {
		env.Upsert(k, v)
	}

	err = utils.SafeWriteFile(ContainerEnvironmentFile, env, SharedFileMask)
	if err != nil {
		return trace.Wrap(err)
	}

	for _, service := range []string{
		"etcd.service",
		"kube-apiserver.service",
		"kube-kubelet.service",
		"kube-proxy.service",
		"kube-controller-manager.service",
		"kube-scheduler.service",
		"flanneld.service",
		"serf.service",
		"docker.service",
	} {
		active, out, err := utils.ServiceIsActive(ctx, service)
		if err != nil {
			log.Warnf("Failed to check status of %q: %s (%v).", service, out, err)
			continue
		}
		if !active {
			continue
		}
		log.WithField("service", service).Info("Will restart.")
		out, err = utils.ServiceCtl(ctx, "restart", service, utils.Blocking(true))
		if err != nil {
			log.Warnf("Failed to restart %q: %s (%v).", service, out, err)
		}
	}

	return nil
}

type environMonitor struct {
	client *kube.Clientset
}
