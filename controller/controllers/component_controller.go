/*

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

package controllers

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	js "github.com/dop251/goja"
	"github.com/go-logr/logr"
	"github.com/kapp-staging/kapp/controller/lib/files"
	"github.com/kapp-staging/kapp/controller/utils"
	"github.com/kapp-staging/kapp/controller/vm"
	"github.com/xeipuuv/gojsonschema"
	appsV1 "k8s.io/api/apps/v1"
	batchV1 "k8s.io/api/batch/v1"
	batchV1Beta1 "k8s.io/api/batch/v1beta1"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"

	corev1alpha1 "github.com/kapp-staging/kapp/controller/api/v1alpha1"
)

// ComponentReconciler reconciles a Component object
type ComponentReconciler struct {
	client.Client
	Reader client.Reader
	Log    logr.Logger
	Scheme *runtime.Scheme
}

type ComponentReconcilerTask struct {
	*ComponentReconciler
	Log logr.Logger

	// The following fields will be filled by calling SetupAttributes() function
	ctx         context.Context
	component   *corev1alpha1.Component
	application *corev1alpha1.Application

	// related resources
	service        *coreV1.Service
	cronJob        *batchV1Beta1.CronJob
	deployment     *appsV1.Deployment
	daemonSet      *appsV1.DaemonSet
	statefulSet    *appsV1.StatefulSet
	pluginBindings *corev1alpha1.ComponentPluginBindingList
}

// +kubebuilder:rbac:groups=core.kapp.dev,resources=components,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core.kapp.dev,resources=components/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core.kapp.dev,resources=componentplugins,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core.kapp.dev,resources=componentplugins/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=extensions,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=extensions,resources=daemonsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,resources=cronjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete

func (r *ComponentReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	task := &ComponentReconcilerTask{
		ComponentReconciler: r,
		ctx:                 context.Background(),
		Log:                 r.Log.WithValues("application", req.NamespacedName),
	}

	task.Log.Info("=========== start reconciling ===========")
	defer task.Log.Info("=========== reconciling done ===========")

	return ctrl.Result{}, task.Run(req)
}

func (r *ComponentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(&appsV1.Deployment{}, ownerKey, func(rawObj runtime.Object) []string {
		deployment := rawObj.(*appsV1.Deployment)
		owner := metaV1.GetControllerOf(deployment)

		if owner == nil {
			return nil
		}

		if owner.APIVersion != apiGVStr || owner.Kind != "Application" {
			return nil
		}

		return []string{owner.Name}
	}); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(&batchV1Beta1.CronJob{}, ownerKey, func(rawObj runtime.Object) []string {
		cronjob := rawObj.(*batchV1Beta1.CronJob)
		owner := metaV1.GetControllerOf(cronjob)

		if owner == nil {
			return nil
		}

		if owner.APIVersion != apiGVStr || owner.Kind != "Application" {
			return nil
		}

		return []string{owner.Name}
	}); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(&coreV1.Service{}, ownerKey, func(rawObj runtime.Object) []string {
		// grab the job object, extract the owner...
		service := rawObj.(*coreV1.Service)
		owner := metaV1.GetControllerOf(service)

		if owner == nil {
			return nil
		}

		if owner.APIVersion != apiGVStr || owner.Kind != "Application" {
			return nil
		}

		return []string{owner.Name}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1alpha1.Component{}).
		Owns(&appsV1.Deployment{}).
		Owns(&batchV1Beta1.CronJob{}).
		Owns(&appsV1.DaemonSet{}).
		Owns(&appsV1.StatefulSet{}).
		Owns(&coreV1.Service{}).
		Complete(r)
}
func (r *ComponentReconcilerTask) Run(req ctrl.Request) error {
	r.ctx = context.Background()
	r.Log = r.Log.WithValues("component", req.NamespacedName)

	r.Log.Info("=========== start reconciling ===========")
	defer r.Log.Info("=========== reconciling done ===========")

	if err := r.SetupAttributes(req); err != nil {
		return client.IgnoreNotFound(err)
	}

	//if err := r.FixComponentDefaultValues(); err != nil {
	//	r.Log.Error(err, "Fix component default values error.")
	//	return ctrl.Result{}, err
	//}

	if err := r.LoadResources(); err != nil {
		return err
	}

	if err := r.HandleDelete(); err != nil {
		return err
	}

	if !r.component.ObjectMeta.DeletionTimestamp.IsZero() {
		return nil
	}

	if err := r.ReconcileService(); err != nil {
		return err
	}

	if err := r.ReconcileWorkload(); err != nil {
		return err
	}

	return nil
}

func (r *ComponentReconcilerTask) GetLabels() map[string]string {
	return map[string]string{
		"kapp-application": r.application.Name,
		"kapp-component":   r.component.Name,
	}
}

func (r *ComponentReconcilerTask) FixComponentDefaultValues() (err error) {
	if r.component == nil {
		return nil
	}

	if r.component.Spec.WorkloadType == "" {
		r.component.Spec.WorkloadType = corev1alpha1.WorkloadTypeServer
	}

	if r.component.Spec.DnsPolicy == "" {
		r.component.Spec.DnsPolicy = coreV1.DNSClusterFirst
	}

	if r.component.Spec.RestartPolicy == "" {
		r.component.Spec.RestartPolicy = coreV1.RestartPolicyAlways
	}

	if r.component.Spec.TerminationGracePeriodSeconds == nil {
		x := int64(30)
		r.component.Spec.TerminationGracePeriodSeconds = &x
	}

	if r.component.Spec.RestartStrategy == "" {
		r.component.Spec.RestartStrategy = appsV1.RollingUpdateDeploymentStrategyType
	}

	for i := range r.component.Spec.Env {
		if r.component.Spec.Env[i].Type == "" {
			r.component.Spec.Env[i].Type = corev1alpha1.EnvVarTypeStatic
		}
	}

	return r.Update(r.ctx, r.component)
}

func (r *ComponentReconcilerTask) ReconcileService() (err error) {
	labels := r.GetLabels()

	if !r.application.Spec.IsActive {
		if r.service != nil {
			return r.Delete(r.ctx, r.service)
		}

		return nil
	}

	if len(r.component.Spec.Ports) > 0 {
		newService := false

		if r.service == nil {
			newService = true
			r.service = &coreV1.Service{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      r.component.Name,
					Namespace: r.component.Namespace,
					Labels:    labels,
				},
				Spec: coreV1.ServiceSpec{
					Selector: labels,
				},
			}
		}

		var ps []coreV1.ServicePort
		for _, port := range r.component.Spec.Ports {
			// if service port is missing, set it same as containerPort
			if port.ServicePort == 0 && port.ContainerPort != 0 {
				port.ServicePort = port.ContainerPort
			}

			sp := coreV1.ServicePort{
				Name:       port.Name,
				TargetPort: intstr.FromInt(int(port.ContainerPort)),
				Port:       int32(port.ServicePort),
			}

			if port.Protocol != "" {
				sp.Protocol = port.Protocol
			}

			ps = append(ps, sp)
		}

		r.service.Spec.Ports = ps

		// TODO service ComponentPlugin call

		if newService {
			//if err := ctrl.SetControllerReference(app, service, r.Scheme); err != nil {
			//	return err
			//}

			if err := r.Create(r.ctx, r.service); err != nil {
				r.Log.Error(err, "unable to create Service for Component")
				return err
			}
		} else {
			if err := r.Update(r.ctx, r.service); err != nil {
				r.Log.Error(err, "unable to update Service for Component")
				return err
			}
		}
	}

	if r.service != nil && len(r.component.Spec.Ports) == 0 {
		if err := r.Delete(r.ctx, r.service); err != nil {
			r.Log.Error(err, "unable to delete Service for Application Component")
			return err
		}
	}

	return r.LoadService()
}

func (r *ComponentReconcilerTask) ReconcileWorkload() (err error) {

	if !r.application.Spec.IsActive {
		if r.deployment != nil {
			if err := r.Delete(r.ctx, r.deployment); err != nil {
				return err
			}
		}
		if r.cronJob != nil {
			if err := r.Delete(r.ctx, r.cronJob); err != nil {
				return err
			}
		}
		if r.daemonSet != nil {
			if err := r.Delete(r.ctx, r.daemonSet); err != nil {
				return err
			}
		}
		if r.statefulSet != nil {
			if err := r.Delete(r.ctx, r.statefulSet); err != nil {
				return err
			}
		}

		return
	}

	if err := r.reconcileDirectConfigs(); err != nil {
		return err
	}

	if err := r.reconcilePermission(); err != nil {
		return err
	}

	template, err := r.GetPodTemplate()

	if err != nil {
		return err
	}

	switch r.component.Spec.WorkloadType {
	case corev1alpha1.WorkloadTypeServer, "":
		return r.ReconcileDeployment(template)
	case corev1alpha1.WorkloadTypeCronjob:
		return r.ReconcileCronJob(template)
	case corev1alpha1.WorkloadTypeDaemonSet:
		return r.ReconcileDaemonSet(template)
	case corev1alpha1.WorkloadTypeStatefulSet:
		return r.ReconcileStatefulSet(template)
	default:
		return fmt.Errorf("unknown workload type: %s", string(r.component.Spec.WorkloadType))
	}
}

func (r *ComponentReconcilerTask) ReconcileDeployment(podTemplateSpec *coreV1.PodTemplateSpec) (err error) {
	app := r.application
	component := r.component
	log := r.Log
	ctx := r.ctx
	deployment := r.deployment
	isNewDeployment := false
	labelMap := r.GetLabels()

	if deployment == nil {
		isNewDeployment = true

		deployment = &appsV1.Deployment{
			ObjectMeta: metaV1.ObjectMeta{
				Labels:      labelMap,
				Annotations: make(map[string]string),
				Name:        component.Name,
				Namespace:   app.Name,
			},
			Spec: appsV1.DeploymentSpec{
				Template: *podTemplateSpec,
				Selector: &metaV1.LabelSelector{
					MatchLabels: labelMap,
				},
			},
		}
	} else {
		deployment.Spec.Template = *podTemplateSpec
	}

	// TODO consider to move to plugin
	if component.Spec.Replicas != nil {
		deployment.Spec.Replicas = component.Spec.Replicas
	} else {
		deployment.Spec.Replicas = nil
	}

	//if len(component.Ports) > 0 {
	//	var ports []coreV1.ContainerPort
	//	for _, p := range component.Ports {
	//		ports = append(ports, coreV1.ContainerPort{
	//			Name:          p.Name,
	//			ContainerPort: int32(p.ContainerPort),
	//			Protocol:      p.Protocol,
	//		})
	//	}
	//}

	// apply plugins
	//for _, pluginDef := range component.Spec.Plugins {
	//	plugin := corev1alpha1.GetPlugin(pluginDef)
	//
	//	switch p := plugin.(type) {
	//	case *corev1alpha1.PluginManualScaler:
	//		p.Operate(deployment)
	//	}
	//}

	if err := ctrl.SetControllerReference(component, deployment, r.Scheme); err != nil {
		log.Error(err, "unable to set owner for deployment")
		return err
	}

	if err := r.runPlugins(ComponentPluginMethodBeforeDeploymentSave, component, deployment, deployment); err != nil {
		log.Error(err, "run before deployment save error.")
		return err
	}

	if isNewDeployment {
		if err := r.Create(ctx, deployment); err != nil {
			log.Error(err, "unable to create Deployment for Application")
			return err
		}

		log.Info("create Deployment " + deployment.Name)
	} else {
		if err := r.Update(ctx, deployment); err != nil {
			log.Error(err, "unable to update Deployment for Application")
			return err
		}

		log.Info("update Deployment " + deployment.Name)
	}

	// apply plugins
	//for _, pluginDef := range app.Spec.Components[0].Plugins {
	//	plugin := corev1alpha1.GetPlugin(pluginDef)
	//
	//	switch p := plugin.(type) {
	//	case *corev1alpha1.PluginManualScaler:
	//		p.Operate(deployment)
	//	}
	//}

	return nil
}

func (r *ComponentReconcilerTask) ReconcileDaemonSet(podTemplateSpec *coreV1.PodTemplateSpec) error {
	log := r.Log
	labelMap := r.GetLabels()

	daemonSet := r.daemonSet
	isNewDs := false

	if daemonSet == nil {
		isNewDs = true

		daemonSet = &appsV1.DaemonSet{
			ObjectMeta: metaV1.ObjectMeta{
				Labels:      labelMap,
				Annotations: make(map[string]string),
				Name:        r.component.Name,
				Namespace:   r.component.Namespace,
			},
			Spec: appsV1.DaemonSetSpec{
				Template: *podTemplateSpec,
				Selector: &metaV1.LabelSelector{
					MatchLabels: labelMap,
				},
			},
		}
	} else {
		daemonSet.Spec.Template = *podTemplateSpec
	}

	if isNewDs {
		if err := ctrl.SetControllerReference(r.component, daemonSet, r.Scheme); err != nil {
			log.Error(err, "unable to set owner for daemonSet")
			return err
		}

		if err := r.Create(r.ctx, daemonSet); err != nil {
			log.Error(err, "unable to create daemonSet for Component")
			return err
		}

		log.Info("create daemonSet " + daemonSet.Name)
	} else {
		if err := r.Update(r.ctx, daemonSet); err != nil {
			log.Error(err, "unable to update daemonSet for Component")
			return err
		}

		log.Info("update daemonSet " + daemonSet.Name)
	}

	return nil
}

func (r *ComponentReconcilerTask) ReconcileCronJob(podTemplateSpec *coreV1.PodTemplateSpec) (err error) {
	app := r.application
	log := r.Log
	ctx := r.ctx
	cj := r.cronJob
	component := r.component
	labelMap := r.GetLabels()

	// restartPolicy
	if podTemplateSpec.Spec.RestartPolicy == coreV1.RestartPolicyAlways ||
		podTemplateSpec.Spec.RestartPolicy == "" {

		podTemplateSpec.Spec.RestartPolicy = coreV1.RestartPolicyOnFailure

	}

	successJobHistoryLimit := int32(3)
	failJobHistoryLimit := int32(5)

	desiredCJSpec := batchV1Beta1.CronJobSpec{
		Schedule: component.Spec.Schedule,
		JobTemplate: batchV1Beta1.JobTemplateSpec{
			Spec: batchV1.JobSpec{
				Template: *podTemplateSpec,
			},
		},
		SuccessfulJobsHistoryLimit: &successJobHistoryLimit,
		FailedJobsHistoryLimit:     &failJobHistoryLimit,
	}

	var isNewCJ bool
	if cj == nil {
		isNewCJ = true

		cj = &batchV1Beta1.CronJob{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      component.Name,
				Namespace: app.Name,
				Labels:    labelMap,
			},
			Spec: desiredCJSpec,
		}
	} else {
		cj.Spec = desiredCJSpec
	}

	if isNewCJ {
		if err := ctrl.SetControllerReference(component, cj, r.Scheme); err != nil {
			log.Error(err, "unable to set owner for cronJob")
			return err
		}

		if err := r.Create(ctx, cj); err != nil {
			log.Error(err, "unable to create CronJob for Component")
			return err
		}

		log.Info("create CronJob " + cj.Name)
	} else {
		if err := r.Update(ctx, cj); err != nil {
			log.Error(err, "unable to update CronJob for Component")
			return err
		}

		log.Info("update CronJob:" + cj.Name)
	}

	return nil
}

func (r *ComponentReconcilerTask) ReconcileStatefulSet(spec *coreV1.PodTemplateSpec) error {

	log := r.Log
	labelMap := r.GetLabels()

	sts := r.statefulSet

	isNewSts := false
	if sts == nil {
		isNewSts = true

		sts = &appsV1.StatefulSet{
			ObjectMeta: metaV1.ObjectMeta{
				Labels:      labelMap,
				Annotations: make(map[string]string),
				Name:        r.component.Name,
				Namespace:   r.component.Namespace,
			},
			Spec: appsV1.StatefulSetSpec{
				Template: *spec,
				Selector: &metaV1.LabelSelector{
					MatchLabels: labelMap,
				},
			},
		}
	} else {
		sts.Spec.Template = *spec
	}

	if isNewSts {
		if err := ctrl.SetControllerReference(r.component, sts, r.Scheme); err != nil {
			log.Error(err, "unable to set owner for sts")
			return err
		}

		if err := r.Create(r.ctx, sts); err != nil {
			log.Error(err, "unable to create sts for Component")
			return err
		}

		log.Info("create sts " + sts.Name)
	} else {
		if err := r.Update(r.ctx, sts); err != nil {
			log.Error(err, "unable to update sts for Component")
			return err
		}

		log.Info("update sts " + sts.Name)
	}

	return nil
}

func (r *ComponentReconcilerTask) GetPodTemplate() (template *coreV1.PodTemplateSpec, err error) {
	component := r.component

	labels := r.GetLabels()
	labels["app"] = component.Name
	labels["version"] = "v1" // TODO

	template = &coreV1.PodTemplateSpec{
		ObjectMeta: metaV1.ObjectMeta{
			Labels: labels,
		},
		Spec: coreV1.PodSpec{
			Containers: []coreV1.Container{
				{
					Name:    component.Name,
					Image:   component.Spec.Image,
					Env:     []coreV1.EnvVar{},
					Command: component.Spec.Command,
					Args:    component.Spec.Args,
					Resources: coreV1.ResourceRequirements{
						Requests: make(map[coreV1.ResourceName]resource.Quantity),
						Limits:   make(map[coreV1.ResourceName]resource.Quantity),
					},
					ReadinessProbe: component.Spec.ReadinessProbe,
					LivenessProbe:  component.Spec.LivenessProbe,
				},
			},
		},
	}

	//decide affinity
	if affinity, exist := r.decideAffinity(); exist {
		template.Spec.Affinity = affinity
	}

	if component.Spec.RunnerPermission != nil {
		template.Spec.ServiceAccountName = r.getNameForPermission()
	}

	mainContainer := &template.Spec.Containers[0]

	// resources
	if component.Spec.CPU != nil && !component.Spec.CPU.IsZero() {
		mainContainer.Resources.Requests[coreV1.ResourceCPU] = *component.Spec.CPU
		mainContainer.Resources.Limits[coreV1.ResourceCPU] = *component.Spec.CPU
	}

	if component.Spec.Memory != nil && !component.Spec.Memory.IsZero() {
		mainContainer.Resources.Limits[coreV1.ResourceMemory] = *component.Spec.Memory
		mainContainer.Resources.Limits[coreV1.ResourceMemory] = *component.Spec.Memory
	}

	// set image secret
	if r.application.Spec.ImagePullSecretName != "" {
		secs := []coreV1.LocalObjectReference{
			{Name: r.application.Spec.ImagePullSecretName},
		}
		template.Spec.ImagePullSecrets = secs
	}

	// apply envs
	var envs []coreV1.EnvVar
	for _, env := range component.Spec.Env {
		var value string

		if env.Type == "" || env.Type == corev1alpha1.EnvVarTypeStatic {
			value = env.Value
		} else if env.Type == corev1alpha1.EnvVarTypeExternal {
			value, err = r.FindShareEnvValue(env.Value)

			//  if the env can't be found in sharedEnv, ignore it
			if err != nil {
				continue
			}
		} else if env.Type == corev1alpha1.EnvVarTypeLinked {
			value, err = r.getValueOfLinkedEnv(env)

			if err != nil {
				return nil, err
			}
		}

		envs = append(envs, coreV1.EnvVar{
			Name:  env.Name,
			Value: value,
		})
	}

	mainContainer.Env = envs

	// Volumes
	// add volumes & volumesMounts
	var volumes []coreV1.Volume
	var volumeMounts []coreV1.VolumeMount
	for i, disk := range component.Spec.Volumes {
		volumeSource := coreV1.VolumeSource{}

		// TODO generate this name at api level
		pvcName := getPVCName(component.Name, disk.Path)

		if disk.Type == corev1alpha1.VolumeTypePersistentVolumeClaim {
			var pvc *coreV1.PersistentVolumeClaim

			if disk.PersistentVolumeClaimName != "" {
				pvcName = disk.PersistentVolumeClaimName
			}

			pvcFetched, err := r.getPVC(pvcName)

			if err != nil {
				return nil, err
			}

			if pvcFetched != nil {
				pvc = pvcFetched
			} else {
				pvc = &coreV1.PersistentVolumeClaim{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      pvcName,
						Namespace: r.component.Namespace,
						Labels:    r.GetLabels(),
					},
					Spec: coreV1.PersistentVolumeClaimSpec{
						AccessModes: []coreV1.PersistentVolumeAccessMode{coreV1.ReadWriteOnce},
						Resources: coreV1.ResourceRequirements{
							Requests: coreV1.ResourceList{
								coreV1.ResourceStorage: disk.Size,
							},
						},
						StorageClassName: disk.StorageClassName,
					},
				}

				if err := r.Create(r.ctx, pvc); err != nil {
					return nil, fmt.Errorf("fail to create PVC: %s, %s", pvc.Name, err)
				}

				component.Spec.Volumes[i].PersistentVolumeClaimName = pvcName
			}

			volumeSource.PersistentVolumeClaim = &coreV1.PersistentVolumeClaimVolumeSource{
				ClaimName: pvcName,
			}

		} else if disk.Type == corev1alpha1.VolumeTypeTemporaryDisk {
			volumeSource.EmptyDir = &coreV1.EmptyDirVolumeSource{
				Medium: coreV1.StorageMediumDefault,
			}
		} else if disk.Type == corev1alpha1.VolumeTypeTemporaryMemory {
			volumeSource.EmptyDir = &coreV1.EmptyDirVolumeSource{
				Medium: coreV1.StorageMediumMemory,
			}
		} else {
			// TODO wrong disk type
		}

		// save pvc name into applications
		if err := r.Update(r.ctx, r.component); err != nil {
			return nil, fmt.Errorf("fail to save PVC name: %s, %s", pvcName, err)
		}

		volumes = append(volumes, coreV1.Volume{
			Name:         pvcName,
			VolumeSource: volumeSource,
		})

		volumeMounts = append(volumeMounts, coreV1.VolumeMount{
			Name:      pvcName,
			MountPath: disk.Path,
		})
	}

	if component.Spec.Configs != nil || component.Spec.DirectConfigs != nil {
		r.parseComponentConfigs(component, &volumes, &volumeMounts)
	}

	if len(volumes) > 0 {
		template.Spec.Volumes = volumes
		mainContainer.VolumeMounts = volumeMounts
	}
	err = r.runPlugins(ComponentPluginMethodAfterPodTemplateGeneration, component, template, template)

	if err != nil {
		r.Log.Error(err, "run "+ComponentPluginMethodAfterPodTemplateGeneration+" save plugin error")
		return nil, err
	}

	return template, nil

}

func getPVCName(componentName, diskPath string) string {
	return fmt.Sprintf("%s-%x", componentName, md5.Sum([]byte(diskPath)))
}

func (r *ComponentReconcilerTask) FindShareEnvValue(name string) (string, error) {
	for _, env := range r.application.Spec.SharedEnv {
		if env.Name != name {
			continue
		}

		if env.Type == corev1alpha1.EnvVarTypeLinked {
			return r.getValueOfLinkedEnv(env)
		} else if env.Type == "" || env.Type == corev1alpha1.EnvVarTypeStatic {
			return env.Value, nil
		}

	}

	return "", fmt.Errorf("fail to find value for shareEnv: %s", name)
}

func (r *ComponentReconcilerTask) getValueOfLinkedEnv(env corev1alpha1.EnvVar) (string, error) {
	if env.Value == "" {
		return env.Value, nil
	}

	parts := strings.Split(env.Value, "/")
	if len(parts) != 2 {
		return "", fmt.Errorf("wrong componentPort config %s, format error", env.Value)
	}

	var service coreV1.Service
	err := r.Reader.Get(r.ctx, types.NamespacedName{
		Namespace: r.component.Namespace,
		Name:      parts[0],
	}, &service)

	if err != nil {
		return "", fmt.Errorf("wrong componentPort config %s, service not exist", env.Value)
	}

	var port int32
	for _, p := range service.Spec.Ports {
		if p.Name == parts[1] {
			port = p.Port
		}
	}

	if port == 0 {
		return "", fmt.Errorf("wrong componentPort config %s, port not exist", env.Value)
	}

	//svc.ns:port
	value := fmt.Sprintf("%s.%s:%d", service.Name, r.component.Namespace, port)

	//<prefix>value<suffix>
	return fmt.Sprintf("%s%s%s", env.Prefix, value, env.Suffix), nil
}

func (r *ComponentReconcilerTask) initPluginRuntime(component *corev1alpha1.Component) *js.Runtime {
	rt := vm.InitRuntime()

	rt.Set("getApplicationName", func(call js.FunctionCall) js.Value {
		return rt.ToValue(r.application.Name)
	})

	rt.Set("getCurrentComponent", func(call js.FunctionCall) js.Value {
		bts, _ := json.Marshal(component)
		var res map[string]interface{}
		_ = json.Unmarshal(bts, &res)
		return rt.ToValue(res)
	})

	return rt
}

func (r *ComponentReconcilerTask) runPlugins(methodName string, component *corev1alpha1.Component, desc interface{}, args ...interface{}) (err error) {
	if r.pluginBindings == nil {
		var bindings corev1alpha1.ComponentPluginBindingList
		if err := r.Reader.List(r.ctx, &bindings, client.InNamespace(r.component.Namespace)); err != nil {
			r.Log.Error(err, "get plugin bindings error")
			return err
		}

		r.pluginBindings = &bindings
	}

	if r.pluginBindings == nil {
		return nil
	}

	for _, binding := range r.pluginBindings.Items {
		if binding.DeletionTimestamp != nil || binding.Spec.IsDisabled {
			continue
		}

		if binding.Spec.ComponentName != "" && binding.Spec.ComponentName != component.Name {
			continue
		}

		pluginProgram, config, err := findPluginAndValidateConfigNew(&binding, methodName, component)

		if err != nil {
			return err
		}

		if pluginProgram == nil {
			continue
		}

		rt := r.initPluginRuntime(component)

		r.insertBuildInPluginImpls(rt, binding.Spec.PluginName, methodName, component, desc, args)

		// TODO refactor this filter
		if pluginProgram.Methods[ComponentPluginMethodComponentFilter] {
			shouldExecute := new(bool)

			err := vm.RunMethod(
				rt,
				pluginProgram.Program,
				ComponentPluginMethodComponentFilter,
				config,
				shouldExecute,
				component,
			)

			if err != nil {
				return err
			}

			if !*shouldExecute {
				continue
			}
		}

		err = vm.RunMethod(
			rt,
			pluginProgram.Program,
			methodName,
			config,
			desc,
			args...,
		)

		if err != nil {
			r.Log.Error(err, fmt.Sprintf("Run plugin error. methodName: %s, componentName: %s, pluginName: %s", methodName, component.Name, binding.Spec.PluginName))
			return err
		}
	}

	return nil
}

func findPluginAndValidateConfigNew(pluginBinding *corev1alpha1.ComponentPluginBinding, methodName string, component *corev1alpha1.Component) (*ComponentPluginProgram, []byte, error) {
	pluginProgram := componentPluginsCache.Get(pluginBinding.Spec.PluginName)

	if pluginProgram == nil {
		return nil, nil, fmt.Errorf("Can't find plugin %s in cache.", pluginBinding.Spec.PluginName)
	}

	if !pluginProgram.Methods[methodName] {
		return nil, nil, nil
	}

	workloadType := component.Spec.WorkloadType

	if workloadType == "" {
		// TODO are we safe to remove this fallback value?
		workloadType = corev1alpha1.WorkloadTypeServer
	}

	if !pluginProgram.AvailableForAllWorkloadTypes && !pluginProgram.AvailableWorkloadTypes[workloadType] {
		return nil, nil, nil
	}

	if pluginProgram.ConfigSchema != nil {
		if pluginBinding.Spec.Config == nil {
			return nil, nil, fmt.Errorf("ComponentPlugin %s require configuration.", pluginBinding.Spec.PluginName)
		}

		pluginConfig := gojsonschema.NewStringLoader(string(pluginBinding.Spec.Config.Raw))
		res, err := pluginProgram.ConfigSchema.Validate(pluginConfig)

		if err != nil {
			return nil, nil, err
		}

		if !res.Valid() {
			return nil, nil, fmt.Errorf(res.Errors()[0].String())
		}

		return pluginProgram, pluginBinding.Spec.Config.Raw, nil
	}

	return pluginProgram, nil, nil
}

func (r *ComponentReconcilerTask) parseComponentConfigs(component *corev1alpha1.Component, volumes *[]coreV1.Volume, volumeMounts *[]coreV1.VolumeMount) {
	var configMap coreV1.ConfigMap

	err := r.Client.Get(r.ctx, types.NamespacedName{
		Name:      files.KAPP_CONFIG_MAP_NAME,
		Namespace: r.component.Namespace,
	}, &configMap)

	if err != nil {
		r.Log.Error(err, "can't get files config-map. Skip configs.")
		return
	}

	// key is mount dir, values is the files
	mountPaths := make(map[string]map[string]bool)

	for _, config := range component.Spec.Configs {
		mountPath := config.MountPath

		for _, path := range config.Paths {
			root, err := files.GetFileItemTree(&configMap, path)

			if err != nil {
				r.Log.Error(err, fmt.Sprintf("can't find file item at %s", path))
				continue
			}

			files.ResolveMountPaths(mountPaths, mountPath, root)
		}
	}

	for mountPath, rawFileNamesMap := range mountPaths {
		name := fmt.Sprintf("configs-%x", md5.Sum([]byte(mountPath)))
		items := make([]coreV1.KeyToPath, 0, len(rawFileNamesMap))

		for itemRawFileName := range rawFileNamesMap {

			items = append(items, coreV1.KeyToPath{
				Path: files.GetFileNameFromRawPath(itemRawFileName),
				Key:  files.EncodeFilePath(itemRawFileName),
			})
		}

		volume := coreV1.Volume{
			Name: name,
			VolumeSource: coreV1.VolumeSource{
				ConfigMap: &coreV1.ConfigMapVolumeSource{
					LocalObjectReference: coreV1.LocalObjectReference{
						Name: files.KAPP_CONFIG_MAP_NAME,
					},
					Items: items,
				},
			},
		}

		volumeMount := coreV1.VolumeMount{
			Name:      name,
			MountPath: mountPath,
		}

		*volumes = append(*volumes, volume)
		*volumeMounts = append(*volumeMounts, volumeMount)
	}

	// directConfigs
	for i, directConfig := range component.Spec.DirectConfigs {
		r.Log.Info("direct", "i:", i)

		path := getPathOfDirectConfig(component.Name, i)

		name := fmt.Sprintf("direct-config-%s-%d", component.Name, i)

		vol := coreV1.Volume{
			Name: name,
			VolumeSource: coreV1.VolumeSource{
				ConfigMap: &coreV1.ConfigMapVolumeSource{
					LocalObjectReference: coreV1.LocalObjectReference{
						Name: files.KAPP_CONFIG_MAP_NAME,
					},
					Items: []coreV1.KeyToPath{
						{
							Key:  files.EncodeFilePath(path),
							Path: "adhoc-name",
						},
					},
				},
			},
		}

		volMount := coreV1.VolumeMount{
			Name:      name,
			MountPath: directConfig.MountFilePath,
			SubPath:   "adhoc-name",
		}

		*volumes = append(*volumes, vol)
		*volumeMounts = append(*volumeMounts, volMount)
	}
}

func (r *ComponentReconcilerTask) getPVC(pvcName string) (*coreV1.PersistentVolumeClaim, error) {
	pvcList := coreV1.PersistentVolumeClaimList{}

	err := r.List(
		r.ctx,
		&pvcList,
		client.InNamespace(r.component.Namespace),
	)
	if err != nil {
		return nil, err
	}

	for _, item := range pvcList.Items {
		if item.Name == pvcName {
			return &item, nil
		}
	}

	return nil, nil
}

func (r *ComponentReconcilerTask) decideAffinity() (*coreV1.Affinity, bool) {
	component := &r.component.Spec

	var nodeSelectorTerms []coreV1.NodeSelectorTerm
	for label, v := range component.NodeSelectorLabels {
		nodeSelectorTerms = append(nodeSelectorTerms, coreV1.NodeSelectorTerm{
			MatchExpressions: []coreV1.NodeSelectorRequirement{
				{
					Key:      label,
					Operator: coreV1.NodeSelectorOpIn,
					Values:   []string{v},
				},
			},
		})
	}

	var nodeAffinity *coreV1.NodeAffinity
	if len(nodeSelectorTerms) > 0 {
		nodeAffinity = &coreV1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &coreV1.NodeSelector{
				NodeSelectorTerms: nodeSelectorTerms,
			},
		}
	}

	labelsOfThisComponent := r.GetLabels()

	var podAffinity *coreV1.PodAffinity
	if component.PodAffinityType == corev1alpha1.PodAffinityTypePreferGather {
		// same
		podAffinity = &coreV1.PodAffinity{
			PreferredDuringSchedulingIgnoredDuringExecution: []coreV1.WeightedPodAffinityTerm{
				{
					Weight: 1,
					PodAffinityTerm: coreV1.PodAffinityTerm{
						TopologyKey: "kubernetes.io/hostname",
						LabelSelector: &metaV1.LabelSelector{
							MatchLabels: labelsOfThisComponent,
						},
					},
				},
			},
		}
	}

	var podAntiAffinity *coreV1.PodAntiAffinity
	if component.PodAffinityType == corev1alpha1.PodAffinityTypePreferFanout {
		podAntiAffinity = &coreV1.PodAntiAffinity{
			PreferredDuringSchedulingIgnoredDuringExecution: []coreV1.WeightedPodAffinityTerm{
				{
					Weight: 1,
					PodAffinityTerm: coreV1.PodAffinityTerm{
						TopologyKey: "kubernetes.io/hostname",
						LabelSelector: &metaV1.LabelSelector{
							MatchLabels: labelsOfThisComponent,
						},
					},
				},
			},
		}
	}

	if nodeAffinity == nil && podAffinity == nil && podAntiAffinity == nil {
		return nil, false
	}

	return &coreV1.Affinity{
		NodeAffinity:    nodeAffinity,
		PodAffinity:     podAffinity,
		PodAntiAffinity: podAntiAffinity,
	}, true
}

func (r *ComponentReconcilerTask) HandleDelete() (err error) {
	r.Log.Info("handleDelete for component")

	if r.component.ObjectMeta.DeletionTimestamp.IsZero() {
		if !utils.ContainsString(r.component.ObjectMeta.Finalizers, finalizerName) {
			r.component.ObjectMeta.Finalizers = append(r.component.ObjectMeta.Finalizers, finalizerName)
			if err := r.Update(context.Background(), r.component); err != nil {
				return err
			}
			r.Log.Info("add finalizer", r.component.Namespace, r.component.Name)
		}
	} else {
		if utils.ContainsString(r.component.ObjectMeta.Finalizers, finalizerName) {
			r.Log.Info("has finalizer", r.component.Namespace, r.component.Name)

			// TODO remove resources
			if err := r.DeleteResources(); err != nil {
				r.Log.Error(err, "fail when DeleteResources")
				return err
			}

			r.Log.Info("rm finalizer for component...")
			r.component.ObjectMeta.Finalizers = utils.RemoveString(r.component.ObjectMeta.Finalizers, finalizerName)
			if err := r.Update(r.ctx, r.component); err != nil {
				r.Log.Error(err, "fail when update component")
				return err
			}
		}
	}

	return nil
}

func (r *ComponentReconcilerTask) SetupAttributes(req ctrl.Request) (err error) {
	var component corev1alpha1.Component
	err = r.Reader.Get(r.ctx, req.NamespacedName, &component)

	if err != nil {
		return err
	}
	r.component = &component

	var app corev1alpha1.Application
	err = r.Reader.Get(r.ctx, types.NamespacedName{
		Namespace: "",
		Name:      component.Namespace,
	}, &app)

	if err != nil {
		return err
	}
	r.application = &app
	return nil
}

func (r *ComponentReconcilerTask) DeleteResources() (err error) {
	if r.service != nil {
		if err := r.Client.Delete(r.ctx, r.service); err != nil {
			r.Log.Error(err, "Delete service error")
			return err
		}
	}

	if r.deployment != nil {
		if err := r.DeleteItem(r.deployment); err != nil {
			return err
		}
	}

	if r.daemonSet != nil {
		if err := r.DeleteItem(r.daemonSet); err != nil {
			return err
		}
	}

	if r.cronJob != nil {
		if err := r.DeleteItem(r.cronJob); err != nil {
			return err
		}
	}

	if r.statefulSet != nil {
		if err := r.DeleteItem(r.statefulSet); err != nil {
			return err
		}
	}

	var bindingList corev1alpha1.ComponentPluginBindingList

	if err := r.Reader.List(r.ctx, &bindingList, client.MatchingLabels{
		"kapp-component": r.component.Name,
	}); err != nil {
		r.Log.Error(err, "get plugin binding list error.")
		return err
	}

	for _, binding := range bindingList.Items {
		if err := r.Delete(r.ctx, &binding); err != nil {
			r.Log.Error(err, "Delete plugin binding error.")
		}
	}

	return nil
}

func (r *ComponentReconcilerTask) DeleteItem(obj runtime.Object) (err error) {
	if err := r.Client.Delete(r.ctx, obj); err != nil {
		gvk := obj.GetObjectKind().GroupVersionKind()
		r.Log.Error(err, fmt.Sprintf(" delete item error. Group: %s, Version: %s, Kind: %s", gvk.Group, gvk.Version, gvk.Kind))
		return err
	}

	return nil
}

func (r *ComponentReconcilerTask) LoadResources() (err error) {
	if err := r.LoadService(); err != nil {
		return err
	}

	switch r.component.Spec.WorkloadType {
	case corev1alpha1.WorkloadTypeServer, "":
		return r.LoadDeployment()
	case corev1alpha1.WorkloadTypeCronjob:
		return r.LoadCronJob()
	case corev1alpha1.WorkloadTypeDaemonSet:
		return r.LoadDaemonSet()
	case corev1alpha1.WorkloadTypeStatefulSet:
		return r.LoadStatefulSet()
	default:
		r.Log.Info("see unknown workloadType:", "type:", r.component.Spec.WorkloadType)
	}

	return nil
}

func (r *ComponentReconcilerTask) LoadService() error {
	var service coreV1.Service

	if err := r.Reader.Get(
		r.ctx,
		types.NamespacedName{
			Namespace: r.component.Namespace,
			Name:      r.component.Name,
		},
		&service,
	); err != nil {
		return client.IgnoreNotFound(err)
	}

	r.service = &service

	return nil
}

func (r *ComponentReconcilerTask) LoadDeployment() error {
	var deploy appsV1.Deployment
	err := r.LoadItem(&deploy)
	if err != nil {
		return client.IgnoreNotFound(err)
	}
	r.deployment = &deploy
	return nil
}

func (r *ComponentReconcilerTask) LoadCronJob() error {
	var cornJob batchV1Beta1.CronJob
	err := r.LoadItem(&cornJob)
	if err != nil {
		return client.IgnoreNotFound(err)
	}
	r.cronJob = &cornJob
	return nil
}

func (r *ComponentReconcilerTask) LoadDaemonSet() error {
	var daemonSet appsV1.DaemonSet
	err := r.LoadItem(&daemonSet)
	if err != nil {
		return client.IgnoreNotFound(err)
	}
	r.daemonSet = &daemonSet
	return nil
}

func (r *ComponentReconcilerTask) LoadStatefulSet() error {
	var statefulSet appsV1.StatefulSet
	err := r.LoadItem(&statefulSet)
	if err != nil {
		return client.IgnoreNotFound(err)
	}
	r.statefulSet = &statefulSet
	return nil
}

func (r *ComponentReconcilerTask) LoadItem(dest runtime.Object) (err error) {
	if err := r.Reader.Get(
		r.ctx,
		types.NamespacedName{
			Namespace: r.component.Namespace,
			Name:      r.component.Name,
		},
		dest,
	); err != nil {
		return err
	}

	return nil
}

func (r *ComponentReconcilerTask) insertBuildInPluginImpls(rt *js.Runtime, pluginName, methodName string, component *corev1alpha1.Component, desc interface{}, args []interface{}) {
	// TODO
}
