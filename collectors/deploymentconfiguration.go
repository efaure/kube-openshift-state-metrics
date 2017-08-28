/*
Copyright 2016 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package collectors

import (
	"context"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
//	"k8s.io/client-go/kubernetes"
	occlient "github.com/openshift/origin/pkg/client"
//	"k8s.io/client-go/pkg/api/v1"

	deployv1 "github.com/openshift/origin/pkg/deploy/apis/apps/v1"
//	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

var (
	descDeploymentConfigurationCreated = prometheus.NewDesc(
		"kube_deploymentconfiguration_created",
		"Unix creation timestamp",
		[]string{"namespace", "replicationcontroller"}, nil,
	)

	descDeploymentConfigurationStatusReadyReplicas = prometheus.NewDesc(
		"kube_deploymentconfiguration_status_ready_replicas",
		"The number of ready replicas per ReplicationController.",
		[]string{"namespace", "replicationcontroller"}, nil,
	)
	descDeploymentConfigurationStatusAvailableReplicas = prometheus.NewDesc(
		"kube_deploymentconfiguration_status_available_replicas",
		"The number of available replicas per ReplicationController.",
		[]string{"namespace", "replicationcontroller"}, nil,
	)


)

//type DeploymentConfigurationLister func() ([]v1.ReplicationController, error)
type DeploymentConfigurationLister func() ([]deployv1.DeploymentConfig, error)

func (l DeploymentConfigurationLister) List() ([]deployv1.DeploymentConfig, error) {
	return l()
}

func RegisterDeploymentConfigurationCollector(registry prometheus.Registerer, ocClient *occlient.Client, namespace string) {


	client := ocClient.RESTClient
	rclw := cache.NewListWatchFromClient(client, "deploymentconfiguration", namespace, nil)
	rcinf := cache.NewSharedInformer(rclw, &deployv1.DeploymentConfig{}, resyncPeriod)

	deploymentConfigurationLister := DeploymentConfigurationLister(func() (rcs []deployv1.DeploymentConfig, err error) {
		for _, c := range rcinf.GetStore().List() {
			rcs = append(rcs, *(c.(*deployv1.DeploymentConfig)))
		}
		return rcs, nil
	})

	registry.MustRegister(&deploymentconfigurationCollector{store: deploymentConfigurationLister})
	go rcinf.Run(context.Background().Done())
}

type deploymentconfigurationStore interface {
	List() (deploymentconfigurations []deployv1.DeploymentConfig, err error)
}

type deploymentconfigurationCollector struct {
	store deploymentconfigurationStore
}

// Describe implements the prometheus.Collector interface.
func (dc *deploymentconfigurationCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- descDeploymentConfigurationCreated
	ch <- descDeploymentConfigurationStatusReadyReplicas
	ch <- descDeploymentConfigurationStatusAvailableReplicas
}


// Collect implements the prometheus.Collector interface.
func (dc *deploymentconfigurationCollector) Collect(ch chan<- prometheus.Metric) {
	dpls, err := dc.store.List()
	if err != nil {
		glog.Errorf("listing deploymentconfiguration failed: %s", err)
		return
	}
	for _, d := range dpls {
		dc.collectDeploymentConfiguration(ch, d)
	}
}

func (dc *deploymentconfigurationCollector) collectDeploymentConfiguration(ch chan<- prometheus.Metric, d deployv1.DeploymentConfig) {
	addGauge := func(desc *prometheus.Desc, v float64, lv ...string) {
		lv = append([]string{d.Namespace, d.Name}, lv...)
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, lv...)
	}
	if !d.CreationTimestamp.IsZero() {
		addGauge(descDeploymentConfigurationCreated, float64(d.CreationTimestamp.Unix()))
	}
	addGauge(descDeploymentConfigurationStatusReadyReplicas, float64(d.Status.ReadyReplicas))
	addGauge(descDeploymentConfigurationStatusAvailableReplicas, float64(d.Status.AvailableReplicas))
}
