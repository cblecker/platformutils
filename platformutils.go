// Package platformutils is a Go library of utility functions for working with
// the PlatformStatus of OpenShift clusters
package platformutils // import "github.com/cblecker/platformutils"

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/yaml"

	configv1 "github.com/openshift/api/config/v1"
	corev1 "k8s.io/api/core/v1"
)

type installConfig struct {
	Platform struct {
		AWS struct {
			Region string `json:"region"`
		} `json:"aws"`
	} `json:"platform"`
}

// PlatformClient is a kubernetes client that is capable of retrieving the items
// necessary to determine the platform status
type PlatformClient struct {
	Client  client.Client
	Context context.Context
}

// NewClient provides a new PlatformClient
func NewClient(ctx context.Context) (PlatformClient, error) {
	var err error
	c := PlatformClient{
		Context: ctx,
	}
	scheme := runtime.NewScheme()

	// Set up platform status client
	cfg, err := config.GetConfig()
	if err != nil {
		return c, err
	}

	// Add OpenShift config apis to scheme
	if err := configv1.Install(scheme); err != nil {
		return c, err
	}

	// Add Core apis to scheme
	if err := corev1.AddToScheme(scheme); err != nil {
		return c, err
	}

	// Create client
	c.Client, err = client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		return c, err
	}
	return c, nil
}

// GetInfrastructureStatus fetches the InfrastructureStatus for the cluster
func (c *PlatformClient) GetInfrastructureStatus() (*configv1.InfrastructureStatus, error) {
	var err error

	// Retrieve the cluster infrastructure config.
	infra := &configv1.Infrastructure{}
	err = c.Client.Get(c.Context, types.NamespacedName{Name: "cluster"}, infra)
	if err != nil {
		return nil, err
	}

	return &infra.Status, nil
}

// GetPlatformStatus provides a backwards-compatible way to look up the
// PlatformStatus configuration for a cluster, if provided with an
// InfrastructureStatus. OCP clusters that were originally installed prior to
// version 4.2 on AWS expose the region config only through the install-config.
// Non-AWS clusters and all installed on 4.2 or later expose this information
// via the infrastructure custom resource.
func (c *PlatformClient) GetPlatformStatus(infraStatus *configv1.InfrastructureStatus) (*configv1.PlatformStatus, error) {
	if status := infraStatus.PlatformStatus; status != nil {
		// Only AWS needs backwards compatibility with install-config
		if status.Type != configv1.AWSPlatformType {
			return status, nil
		}

		// Check whether the cluster config is already migrated
		if status.AWS != nil && len(status.AWS.Region) > 0 {
			return status, nil
		}
	}

	// Otherwise build a platform status from the deprecated install-config
	clusterConfigName := types.NamespacedName{Namespace: "kube-system", Name: "cluster-config-v1"}
	clusterConfig := &corev1.ConfigMap{}
	if err := c.Client.Get(c.Context, clusterConfigName, clusterConfig); err != nil {
		return nil, fmt.Errorf("failed to get configmap %s: %v", clusterConfigName, err)
	}
	data, ok := clusterConfig.Data["install-config"]
	if !ok {
		return nil, fmt.Errorf("missing install-config in configmap")
	}
	var ic installConfig
	if err := yaml.Unmarshal([]byte(data), &ic); err != nil {
		return nil, fmt.Errorf("invalid install-config: %v\njson:\n%s", err, data)
	}
	return &configv1.PlatformStatus{
		//lint:ignore SA1019 ignore deprecation, as this function is specifically designed for backwards compatibility
		//nolint:staticcheck // ref https://github.com/golangci/golangci-lint/issues/741
		Type: infraStatus.Platform,
		AWS: &configv1.AWSPlatformStatus{
			Region: ic.Platform.AWS.Region,
		},
	}, nil
}

// IsPlatformSupported checks if specified platform is in a slice of supported
// platforms
func IsPlatformSupported(platform configv1.PlatformType, supportedPlatforms []configv1.PlatformType) bool {
	for _, p := range supportedPlatforms {
		if p == platform {
			return true
		}
	}
	return false
}
