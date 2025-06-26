package chaos

import (
	"fmt"
	"time"
)

type DiskChaos struct {
	APIVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Metadata   Metadata `yaml:"metadata"`
	Spec       DiskSpec `yaml:"spec"`
}

type DiskSpec struct {
	Action     string   `yaml:"action"`
	Mode       string   `yaml:"mode"`
	Selector   Selector `yaml:"selector"`
	VolumePath string   `yaml:"volumePath"`
	Path       string   `yaml:"path"`
	Delay      string   `yaml:"delay"`
	Percent    int      `yaml:"percent"`
	Duration   string   `yaml:"duration"`
}

func MakeDiskLatency(pod, podNS, chaosNS, volume, path string, percent int, latency time.Duration) DiskChaos {
	return makeDiskChaos(pod, podNS, chaosNS, "latency", volume, path, percent)
}

func makeDiskChaos(pod, podNS, chaosNS, action, volume, path string, percent int) DiskChaos {
	return DiskChaos{
		APIVersion: "chaos-mesh.org/v1alpha1",
		Kind:       "IOChaos",
		Metadata: Metadata{
			Name:      fmt.Sprintf("%s-%s", pod, action),
			Namespace: chaosNS,
		},
		Spec: DiskSpec{
			Action: action,
			Mode:   "one",
			Selector: Selector{
				Namespaces: []string{podNS},
				LabelSelectors: LabelSelectors{
					StatefulsetKubernetesIoPodName: pod,
				},
			},
			VolumePath: volume,
			Path:       path,
			Percent:    percent,
		},
	}
}
