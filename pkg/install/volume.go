package install

import (
	v1 "k8s.io/api/core/v1"
)

func AgentVolume() v1.Volume {
	return v1.Volume{
		Name: AgentAnnotationVolumeName,
		VolumeSource: v1.VolumeSource{
			DownwardAPI: &v1.DownwardAPIVolumeSource{
				Items: []v1.DownwardAPIVolumeFile{{
					FieldRef: &v1.ObjectFieldSelector{
						FieldPath: "metadata.annotations",
					},
					Path: "annotations",
				}}}}}
}
