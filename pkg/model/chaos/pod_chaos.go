package chaos

import (
	"fmt"
	"time"
)

type PodChaos struct {
	APIVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Metadata   Metadata `yaml:"metadata"`
	Spec       PodSpec  `yaml:"spec"`
}

type Metadata struct {
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace"`
}

type PodSpec struct {
	Action   string        `yaml:"action"`
	Mode     string        `yaml:"mode"`
	Duration time.Duration `yaml:"duration"`
	Selector Selector      `yaml:"selector"`
}

type Selector struct {
	Namespaces     []string       `yaml:"namespaces"`
	LabelSelectors LabelSelectors `yaml:"labelSelectors"`
}

type LabelSelectors struct {
	StatefulsetKubernetesIoPodName string `yaml:"statefulset.kubernetes.io/pod-name"`
}

func MakePodChaos(pod, podNS, chaosNS, action string, duration time.Duration) PodChaos {
	return PodChaos{
		APIVersion: "chaos-mesh.org/v1alpha1",
		Kind:       "PodChaos",
		Metadata: Metadata{
			Name:      fmt.Sprintf("%s-%s", pod, action),
			Namespace: chaosNS,
		},
		Spec: PodSpec{
			Action:   action,
			Mode:     "one",
			Duration: duration,
			Selector: Selector{
				Namespaces: []string{podNS},
				LabelSelectors: LabelSelectors{
					StatefulsetKubernetesIoPodName: pod,
				},
			},
		},
	}
}
