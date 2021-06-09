package install

import (
	"path/filepath"
	"strconv"
	"strings"

	v1 "k8s.io/api/core/v1"

	"github.com/datawire/ambassador/pkg/kates"
)

const envPrefix = "TEL_APP_"

func AgentContainer(name, imageName string, appContainer *v1.Container, port v1.ContainerPort, appPort int, managerNs string) v1.Container {
	return v1.Container{
		Name:         AgentContainerName,
		Image:        imageName,
		Args:         []string{"agent"},
		Ports:        []v1.ContainerPort{port},
		Env:          agentEnvironment(name, appContainer, appPort, managerNs),
		EnvFrom:      agentEnvFrom(appContainer.EnvFrom),
		VolumeMounts: agentVolumeMounts(appContainer.VolumeMounts),
		ReadinessProbe: &v1.Probe{
			Handler: v1.Handler{
				Exec: &v1.ExecAction{
					Command: []string{"/bin/stat", "/tmp/agent/ready"},
				},
			},
		}}
}

func agentEnvironment(agentName string, appContainer *kates.Container, appPort int, managerNS string) []v1.EnvVar {
	appEnv := appEnvironment(appContainer)
	env := make([]v1.EnvVar, len(appEnv), len(appEnv)+7)
	copy(env, appEnv)
	env = append(env,
		v1.EnvVar{
			Name:  "LOG_LEVEL",
			Value: "debug",
		},
		v1.EnvVar{
			Name:  "AGENT_NAME",
			Value: agentName,
		},
		v1.EnvVar{
			Name: "AGENT_NAMESPACE",
			ValueFrom: &v1.EnvVarSource{
				FieldRef: &v1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		},
		v1.EnvVar{
			Name: "AGENT_POD_IP",
			ValueFrom: &v1.EnvVarSource{
				FieldRef: &v1.ObjectFieldSelector{
					FieldPath: "status.podIP",
				},
			},
		},
		v1.EnvVar{
			Name:  "APP_PORT",
			Value: strconv.Itoa(appPort),
		})
	if len(appContainer.VolumeMounts) > 0 {
		env = append(env, v1.EnvVar{
			Name:  "APP_MOUNTS",
			Value: TelAppMountPoint,
		})

		// Have the agent propagate the mount-points as TELEPRESENCE_MOUNTS to make it easy for the
		// local app to create symlinks.
		mounts := make([]string, len(appContainer.VolumeMounts))
		for i := range appContainer.VolumeMounts {
			mounts[i] = appContainer.VolumeMounts[i].MountPath
		}
		env = append(env, v1.EnvVar{
			Name:  envPrefix + "TELEPRESENCE_MOUNTS",
			Value: strings.Join(mounts, ":"),
		})
	}
	env = append(env, v1.EnvVar{
		Name:  "MANAGER_HOST",
		Value: ManagerAppName + "." + managerNS,
	})
	return env
}

func appEnvironment(appContainer *kates.Container) []v1.EnvVar {
	envCopy := make([]v1.EnvVar, len(appContainer.Env)+1)
	for i, ev := range appContainer.Env {
		ev.Name = envPrefix + ev.Name
		envCopy[i] = ev
	}
	envCopy[len(appContainer.Env)] = v1.EnvVar{
		Name:  "TELEPRESENCE_CONTAINER",
		Value: appContainer.Name,
	}
	return envCopy
}

func agentEnvFrom(appEF []v1.EnvFromSource) []v1.EnvFromSource {
	if ln := len(appEF); ln > 0 {
		agentEF := make([]v1.EnvFromSource, ln)
		for i, appE := range appEF {
			appE.Prefix = envPrefix + appE.Prefix
			agentEF[i] = appE
		}
		return agentEF
	}
	return appEF
}

func agentVolumeMounts(mounts []v1.VolumeMount) []v1.VolumeMount {
	agentMounts := make([]v1.VolumeMount, len(mounts)+1)
	for i, mount := range mounts {
		// Keep the ServiceAccount mount unaltered or a new one will be generated
		if mount.MountPath != "/var/run/secrets/kubernetes.io/serviceaccount" {
			mount.MountPath = filepath.Join(TelAppMountPoint, mount.MountPath)
		}
		agentMounts[i] = mount
	}
	agentMounts[len(mounts)] = v1.VolumeMount{
		Name:      AgentAnnotationVolumeName,
		MountPath: "/tel_pod_info",
	}
	return agentMounts
}

const maxPortNameLen = 15

// HiddenPortName prefixes the given name with "tm-" and truncates it to 15 characters. If
// the ordinal is greater than zero, the last two digits are reserved for the hexadecimal
// representation of that ordinal.
func HiddenPortName(name string, ordinal int) string {
	// New name must be max 15 characters long
	hiddenName := "tm-" + name
	if len(hiddenName) > maxPortNameLen {
		if ordinal > 0 {
			hiddenName = hiddenName[:maxPortNameLen-2] + strconv.FormatInt(int64(ordinal), 16) // we don't expect more than 256 ports
		} else {
			hiddenName = hiddenName[:maxPortNameLen]
		}
	}
	return hiddenName
}
