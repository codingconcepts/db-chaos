package runner

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"

	yamlenc "gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

func applyExperiment(kubeRestConfig *rest.Config, dynClient *dynamic.DynamicClient, exp any) (func() error, error) {
	yamlBytes, err := yamlenc.Marshal(exp)
	if err != nil {
		return nil, fmt.Errorf("marshalling object yaml: %w", err)
	}

	discoClient := discovery.NewDiscoveryClientForConfigOrDie(kubeRestConfig)
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(discoClient))

	// Decode YAML to unstructured for use against the dynamic API.
	decUnstructured := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	obj := &unstructured.Unstructured{}
	_, gvk, err := decUnstructured.Decode(yamlBytes, nil, obj)
	if err != nil {
		return nil, fmt.Errorf("failed to decode YAML: %v", err)
	}

	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to get REST mapping: %v", err)
	}

	var dr dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		namespace := obj.GetNamespace()
		if namespace == "" {
			namespace = "default"
		}
		dr = dynClient.Resource(mapping.Resource).Namespace(namespace)
	} else {
		dr = dynClient.Resource(mapping.Resource)
	}

	_, err = dr.Create(context.Background(), obj, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %v", err)
	}

	delete := func() error {
		name := obj.GetName()
		return dr.Delete(context.Background(), name, metav1.DeleteOptions{})
	}

	return delete, nil
}

func deleteExperiment(kubeRestConfig *rest.Config, dynClient *dynamic.DynamicClient, exp any) error {
	yamlBytes, err := yamlenc.Marshal(exp)
	if err != nil {
		return fmt.Errorf("marshalling object yaml: %w", err)
	}

	// Create RESTMapper to discover GVR
	discoClient := discovery.NewDiscoveryClientForConfigOrDie(kubeRestConfig)
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(discoClient))

	// Decode YAML to Unstructured
	decUnstructured := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	obj := &unstructured.Unstructured{}
	_, gvk, err := decUnstructured.Decode(yamlBytes, nil, obj)
	if err != nil {
		return fmt.Errorf("failed to decode YAML: %v", err)
	}

	// Map GVK to GVR
	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return fmt.Errorf("failed to get REST mapping: %v", err)
	}

	// Get the resource interface
	var dr dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		namespace := obj.GetNamespace()
		dr = dynClient.Resource(mapping.Resource).Namespace(namespace)
	} else {
		dr = dynClient.Resource(mapping.Resource)
	}

	name := obj.GetName()
	return dr.Delete(context.Background(), name, metav1.DeleteOptions{})
}
