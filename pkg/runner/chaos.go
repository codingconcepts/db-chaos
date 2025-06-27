package runner

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"sort"
	"time"

	"github.com/codingconcepts/db-chaos/pkg/model/chaos"
	"github.com/codingconcepts/db-chaos/pkg/repo"
	"github.com/fatih/color"
	"github.com/samber/lo"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var (
	yellow = color.RGB(255, 240, 133).SprintFunc()
)

type ChaosRunner struct {
	repo           repo.Repo
	ns             string
	chaosNS        string
	downDuration   time.Duration
	readyTimeout   time.Duration
	notify         chan<- string
	kubeRestConfig *rest.Config
	kubeClient     *kubernetes.Clientset
	kubeClientDyn  *dynamic.DynamicClient
}

func NewChaosRunner(repo repo.Repo, ns, chaosNS string, downDuration, readyTimeout time.Duration, notify chan<- string) (*ChaosRunner, error) {
	restConfig, kubeClient, dynClient, err := createKubernetesClient()
	if err != nil {
		return nil, fmt.Errorf("creating kubernetes client: %w", err)
	}

	return &ChaosRunner{
		repo:           repo,
		ns:             ns,
		chaosNS:        chaosNS,
		downDuration:   downDuration,
		readyTimeout:   readyTimeout,
		notify:         notify,
		kubeRestConfig: restConfig,
		kubeClient:     kubeClient,
		kubeClientDyn:  dynClient,
	}, nil
}

func (r *ChaosRunner) Run() error {
	pods, err := r.getPods()
	if err != nil {
		return fmt.Errorf("fetching pods: %w", err)
	}

	log.Printf("[%s] pods: %v", yellow("chaos"), pods)

	log.Printf("[%s] Running Pod Failures", yellow("chaos"))
	if err = r.PodChaos(pods, "pod-failure"); err != nil {
		return fmt.Errorf("running pod-failure: %w", err)
	}

	log.Printf("[%s] Running Pod Kills", yellow("chaos"))
	if err = r.PodChaos(pods, "pod-kill"); err != nil {
		return fmt.Errorf("running pod-kill: %w", err)
	}

	log.Printf("[%s] Running Symmetric Network Partitions", yellow("chaos"))
	if err = r.NetworkChaos(pods, "both"); err != nil {
		return fmt.Errorf("running symmetric partition: %w", err)
	}

	log.Printf("[%s] Running Asymmetric Network Partitions", yellow("chaos"))
	if err = r.NetworkChaos(pods, "to"); err != nil {
		return fmt.Errorf("running asymmetric partition: %w", err)
	}

	return nil
}

func (r *ChaosRunner) PodChaos(pods []string, action string) error {
	r.notify <- fmt.Sprintf("pod-chaos-%s", action)
	defer func() { r.notify <- "" }()

	for _, pod := range pods {
		if err := r.waitForReady(); err != nil {
			return fmt.Errorf("waiting for ready: %w", err)
		}

		chaos := chaos.MakePodChaos(pod, r.ns, r.chaosNS, action, r.downDuration)

		delete, err := applyExperiment(r.kubeRestConfig, r.kubeClientDyn, &chaos)
		if err != nil {
			return fmt.Errorf("applying experiment: %w", err)
		}
		log.Printf("[%s] applied chaos: %s", yellow("chaos"), chaos.Metadata.Name)

		time.Sleep(r.downDuration)

		if err := delete(); err != nil {
			return fmt.Errorf("applying experiment: %w", err)
		}
		log.Printf("[%s] deleted chaos: %s", yellow("chaos"), chaos.Metadata.Name)
	}

	return nil
}

func (r *ChaosRunner) NetworkChaos(pods []string, action string) error {
	r.notify <- fmt.Sprintf("network-chaos-%s", action)
	defer func() { r.notify <- "" }()

	partitionPairs := getPairCombinationsSlices(pods)

	for _, pair := range partitionPairs {
		if err := r.waitForReady(); err != nil {
			return fmt.Errorf("waiting for ready: %w", err)
		}

		chaos := chaos.MakePartition(pair[0], pair[1], r.ns, r.chaosNS, action)

		delete, err := applyExperiment(r.kubeRestConfig, r.kubeClientDyn, &chaos)
		if err != nil {
			return fmt.Errorf("applying experiment: %w", err)
		}
		log.Printf("[%s] applied chaos: %s", yellow("chaos"), chaos.Metadata.Name)

		time.Sleep(r.downDuration)

		if err := delete(); err != nil {
			return fmt.Errorf("applying experiment: %w", err)
		}
		log.Printf("[%s] deleted chaos: %s", yellow("chaos"), chaos.Metadata.Name)
	}

	return nil
}

func getPairCombinationsSlices(strings []string) [][2]string {
	var pairs [][2]string

	for i := range strings {
		for j := range strings {
			if i != j {
				pairs = append(pairs, [2]string{strings[i], strings[j]})
			}
		}
	}

	return pairs
}

func (r *ChaosRunner) waitForReady() error {
	timeout := time.Tick(r.readyTimeout)
	check := time.Tick(time.Second * 5)

	for {
		select {
		case <-check:
			ready, err := r.repo.IsReady()
			if err != nil {
				log.Printf("[%s] error checking readiness: %v", yellow("chaos"), err)
				continue
			}

			if ready {
				log.Printf("[%s] database ready for next experiment", yellow("chaos"))
				return nil
			} else {
				log.Printf("[%s] waiting for database to be ready...", yellow("chaos"))
				continue
			}

		case <-timeout:
			return fmt.Errorf("timeout")
		}
	}
}

func createKubernetesClient() (*rest.Config, *kubernetes.Clientset, *dynamic.DynamicClient, error) {
	var config *rest.Config
	var err error

	// Try to use in-cluster config and fallback to regular kube config.
	config, err = rest.InClusterConfig()
	if err != nil {
		var kubeconfig string
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = filepath.Join(home, ".kube", "config")
		}

		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("building config: %w", err)
		}
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("creating client: %w", err)
	}

	dyn, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("creating client: %w", err)
	}

	return config, client, dyn, nil
}

func (r *ChaosRunner) getPods() ([]string, error) {
	timeout, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	pods, err := r.kubeClient.CoreV1().Pods(r.ns).List(timeout, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %v", err)
	}

	podNames := lo.Map(pods.Items, func(p v1.Pod, _ int) string {
		return p.Name
	})

	sort.Strings(podNames)
	return podNames, nil
}
