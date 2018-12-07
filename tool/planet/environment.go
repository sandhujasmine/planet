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
	m := &environMonitor{
		ctx:    ctx,
		client: client,
		// update request channel services a request at a time
		updateCh:  make(chan map[string]string, 1),
		requestCh: make(chan map[string]string),
	}
	go m.monitorEnvironmentConfigMap()
	go m.updateEnvironmentLoop()
	go m.requestLoop()
	return nil
}

func (r environMonitor) monitorEnvironmentConfigMap() {
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
	controller.Run(r.ctx.Done())
}

func (r *environMonitor) add(obj interface{}) {
	r.updateEnvironmentConfigMap(nil, obj)
}

func (r *environMonitor) update(oldObj, newObj interface{}) {
	r.updateEnvironmentConfigMap(oldObj, newObj)
}

func (r *environMonitor) updateEnvironmentConfigMap(oldObj, newObj interface{}) {
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
		return
	}
	select {
	case <-r.ctx.Done():
		return
	case r.requestCh <- newConfigmap.Data:
		// Handling of updateCh must be snappy, will not drop
	}
}

// requestLoop accepts requests to update environment.
// If it's able to pass on the updated environment immediately, its job is done.
// Otherwise, it persists the value and tries to pass it on in a loop until successful
// or the context is cancelled.
// The purpose is maintaining the last update request and guaranteed delivery
// of the update
func (r *environMonitor) requestLoop() {
	var kvs map[string]string
	ticker := time.NewTicker(5 * time.Second)
	var tickerCh <-chan time.Time
	for {
		select {
		case <-r.ctx.Done():
			return
		case update := <-r.requestCh:
			kvs = update
			// Schedule the update
			tickerCh = ticker.C
		case <-tickerCh:
			select {
			case <-r.ctx.Done():
				return
			case r.updateCh <- kvs:
				kvs = nil
				tickerCh = nil
			default:
				// Try again
			}
		}
	}
}

// updateEnvironmentLoop runs the actual environment update loop.
// It is a blocking operation, which, upon receiving new environment,
// restarts all relevant services one by one, until all have
// been restarted
func (r *environMonitor) updateEnvironmentLoop() {
	for {
		select {
		case <-r.ctx.Done():
			return
		case kvs := <-r.updateCh:
			if err := updateEnvironment(r.ctx, kvs); err != nil {
				log.WithError(err).Warn("Failed to update environment.")
			}
		}
	}
}

func updateEnvironment(ctx context.Context, kvs map[string]string) error {
	log.WithField("kvs", kvs).Info("Update environment.")
	env, err := box.ReadEnvironment(ContainerEnvironmentFile)
	if err != nil {
		return trace.Wrap(err, "failed to read cluster environment file")
	}

	for k, v := range kvs {
		env.Upsert(k, v)
	}

	err = utils.SafeWriteFile(ContainerEnvironmentFile, env, SharedFileMask)
	if err != nil {
		return trace.Wrap(err, "failed to write cluster environment file")
	}

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
			log.Warnf("Failed to restart: %s (%v).", out, err)
		}
	}
	return nil
}

type environMonitor struct {
	ctx       context.Context
	client    *kube.Clientset
	updateCh  chan map[string]string
	requestCh chan map[string]string
}
