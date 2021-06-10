package resource

import (
	"context"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/datawire/ambassador/pkg/kates"
	"github.com/datawire/dlib/dlog"
	"github.com/telepresenceio/telepresence/v2/pkg/client"
	"github.com/telepresenceio/telepresence/v2/pkg/install"
)

const managerLicenseName = "systema-license"

type tmDeployment struct {
	found *kates.Deployment
}

var TrafficManagerDeployment Instance = &tmDeployment{}

func (ri *tmDeployment) deployment(ctx context.Context) *kates.Deployment {
	dep := new(kates.Deployment)
	dep.TypeMeta = kates.TypeMeta{
		Kind: "Deployment",
	}
	sc := getScope(ctx)
	dep.ObjectMeta = kates.ObjectMeta{
		Namespace: sc.namespace,
		Name:      install.ManagerAppName,
		Labels:    sc.tmSelector,
	}
	return dep
}

func (ri *tmDeployment) desiredDeployment(ctx context.Context, addLicense bool) *kates.Deployment {
	replicas := int32(1)

	sc := getScope(ctx)
	var containerEnv = []corev1.EnvVar{
		{Name: "LOG_LEVEL", Value: "info"},
		{Name: "SYSTEMA_HOST", Value: sc.env.SystemAHost},
		{Name: "SYSTEMA_PORT", Value: sc.env.SystemAPort},
		{Name: "CLUSTER_ID", Value: sc.clusterID},
		{Name: "TELEPRESENCE_REGISTRY", Value: sc.env.Registry},

		// Manager needs to know its own namespace so that it can propagate that when
		// to agents when injecting them
		{
			Name: "MANAGER_NAMESPACE",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		},
	}
	if sc.env.AgentImage != "" {
		containerEnv = append(containerEnv, corev1.EnvVar{Name: "TELEPRESENCE_AGENT_IMAGE", Value: sc.env.AgentImage})
	}

	// If addLicense is true, we mount the secret as a volume into the traffic-manager
	// and then we mount that volume to a path in the container that the traffic-manager
	// knows about and can read from.
	var volumes []corev1.Volume
	var volumeMounts []corev1.VolumeMount
	if addLicense {
		volumes = append(volumes, corev1.Volume{
			Name: "license",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: managerLicenseName,
				},
			},
		})
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "license",
			ReadOnly:  true,
			MountPath: "/home/telepresence/",
		})
	}

	optional := true
	volumes = append(volumes, corev1.Volume{
		Name: "tls",
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: install.AgentInjectorTLSName,
				Optional:   &optional,
			},
		},
	})
	volumeMounts = append(volumeMounts, corev1.VolumeMount{
		Name:      "tls",
		ReadOnly:  true,
		MountPath: "/var/run/secrets/tls",
	})

	dep := ri.deployment(ctx)
	dep.Spec = appsv1.DeploymentSpec{
		Replicas: &replicas,
		Selector: &metav1.LabelSelector{
			MatchLabels: sc.tmSelector,
		},
		Template: kates.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: sc.tmSelector,
			},
			Spec: corev1.PodSpec{
				Volumes: volumes,
				Containers: []corev1.Container{
					{
						Name:  install.ManagerAppName,
						Image: ri.imageName(ctx),
						Env:   containerEnv,
						Ports: []corev1.ContainerPort{
							{
								Name:          "api",
								ContainerPort: install.ManagerPortHTTP,
							},
							{
								Name:          "https",
								ContainerPort: install.ManagerPortHTTPS,
							},
						},
						VolumeMounts: volumeMounts,
					}},
				ServiceAccountName: install.ManagerAppName,
			},
		},
	}
	return dep
}

func (ri *tmDeployment) hasLicense(ctx context.Context) (bool, error) {
	return exists(ctx, &kates.Secret{
		TypeMeta:   kates.TypeMeta{Kind: "Secret"},
		ObjectMeta: kates.ObjectMeta{Name: managerLicenseName, Namespace: getScope(ctx).namespace},
	})
}

func (ri *tmDeployment) imageName(ctx context.Context) string {
	return fmt.Sprintf("%s/tel2:%s", getScope(ctx).env.Registry, strings.TrimPrefix(client.Version(), "v"))
}

func (ri *tmDeployment) Create(ctx context.Context) error {
	hasLicense, err := ri.hasLicense(ctx)
	if err != nil {
		return err
	}
	return create(ctx, ri.desiredDeployment(ctx, hasLicense))
}

func (ri *tmDeployment) Exists(ctx context.Context) (bool, error) {
	found, err := find(ctx, ri.deployment(ctx))
	if err != nil {
		return false, err
	}
	if found == nil {
		return false, nil
	}
	ri.found = found.(*kates.Deployment)
	return true, nil
}

func (ri *tmDeployment) Delete(ctx context.Context) error {
	return remove(ctx, ri.deployment(ctx))
}

func (ri *tmDeployment) Update(ctx context.Context) error {
	if ri.found == nil {
		return nil
	}

	imageName := ri.imageName(ctx)
	hasLicense, err := ri.hasLicense(ctx)
	if err != nil {
		return err
	}
	currentPodSpec := &ri.found.Spec.Template.Spec

	hasLicenseVolume := false
	for _, v := range currentPodSpec.Volumes {
		if v.Name == "license" {
			hasLicenseVolume = true
			break
		}
	}
	if hasLicense == hasLicenseVolume {
		cns := currentPodSpec.Containers
		for i := range cns {
			cn := &cns[i]
			if cn.Image == imageName {
				dlog.Infof(ctx, "%s is up-to-date. Image: %s, License %t", logName(ri.found), imageName, hasLicense)
				return nil
			}
		}
	}

	dep := ri.desiredDeployment(ctx, hasLicense)
	dep.ResourceVersion = ri.found.ResourceVersion
	dlog.Infof(ctx, "Updating %s. Image: %s, License %t", logName(dep), imageName, hasLicense)
	if err = getScope(ctx).client.Update(ctx, dep, dep); err != nil {
		return fmt.Errorf("failed to update %s: %w", logName(dep), err)
	}
	return waitForDeployApply(ctx, dep)
}
