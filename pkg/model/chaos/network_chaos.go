package chaos

import (
	"fmt"
)

type NetworkChaos struct {
	APIVersion string      `yaml:"apiVersion"`
	Kind       string      `yaml:"kind"`
	Metadata   Metadata    `yaml:"metadata"`
	Spec       NetworkSpec `yaml:"spec"`
}

type NetworkSpec struct {
	Action    string   `yaml:"action"`
	Mode      string   `yaml:"mode"`
	Selector  Selector `yaml:"selector"`
	Direction string   `yaml:"direction"`
	Target    Target   `yaml:"target"`
}

type Target struct {
	Mode     string   `yaml:"mode"`
	Selector Selector `yaml:"selector"`
}

func MakeSymmetricPartition(sourcePod, targetPod, podNS, chaosNS string) NetworkChaos {
	return makePartition(sourcePod, targetPod, podNS, chaosNS, "both")
}

func MakeAsymmetricPartition(sourcePod, targetPod, podNS, chaosNS string) NetworkChaos {
	return makePartition(sourcePod, targetPod, podNS, chaosNS, "to")
}

func makePartition(sourcePod, targetPod, podNS, chaosNS, direction string) NetworkChaos {
	return NetworkChaos{
		APIVersion: "chaos-mesh.org/v1alpha1",
		Kind:       "NetworkChaos",
		Metadata: Metadata{
			Name:      fmt.Sprintf("%s-%s", sourcePod, targetPod),
			Namespace: chaosNS,
		},
		Spec: NetworkSpec{
			Action:    "partition",
			Mode:      "all",
			Direction: direction,
			Selector: Selector{
				Namespaces: []string{podNS},
				LabelSelectors: LabelSelectors{
					StatefulsetKubernetesIoPodName: sourcePod,
				},
			},
			Target: Target{
				Mode: "one",
				Selector: Selector{
					Namespaces: []string{podNS},
					LabelSelectors: LabelSelectors{
						StatefulsetKubernetesIoPodName: targetPod,
					},
				},
			},
		},
	}
}
