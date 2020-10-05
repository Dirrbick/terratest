/*

This file implements an Azure client factory that automatically handles setting up Base URI
values for sovereign cloud support. Note the list of clients below is not initially exhaustive;
rather, additional clients will me added as-needed.

*/

package azure

import (
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/containerservice/mgmt/2019-11-01/containerservice"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-06-01/subscriptions"
	autorestAzure "github.com/Azure/go-autorest/autorest/azure"
)

const (
	// AzureEnvironmentEnvName is the name of the Azure environment to use. Set to one of the following:
	//
	// "AzureChinaCloud":        ChinaCloud
	// "AzureGermanCloud":       GermanCloud
	// "AzurePublicCloud":       PublicCloud
	// "AzureUSGovernmentCloud": USGovernmentCloud
	// "AzureStackCloud":		 Azure stack
	AzureEnvironmentEnvName = "AZURE_ENVIRONMENT"
)

// ClientType describes the type of client a module can create.
type ClientType int

const (
	// SubscriptionsClientType represents a SubscriptionClient
	SubscriptionsClientType ClientType = iota

	// VirtualMachinesClientType represents a VirtualMachinesClient
	VirtualMachinesClientType

	// ManagedClustersClientType represents a ManagedClustersClient
	ManagedClustersClientType
)

// ClientFactory describes the methods available on client factory implementatoins
type ClientFactory interface {
	// GetClientE returns a client instance based on the ClientType passed, or optionally an error.
	GetClientE(clientType ClientType, subscriptionID string) (interface{}, error)
}

// multiEnvClientFactory is used to coordinate handing out properly configured Azure SDK clients
// that are properly setup for use with Public or Sovereign clouds (depending on configuration)
type multiEnvClientFactory struct{}

// NewClientFactory returns a new multi-environment client factory
func NewClientFactory() ClientFactory {
	return &multiEnvClientFactory{}
}

// GetClientE returns a client instance based on the ClientType passed, or optionally an error.
func (factory *multiEnvClientFactory) GetClientE(clientType ClientType, subscriptionID string) (interface{}, error) {
	// Validate Azure subscription ID
	subscriptionID, err := getTargetAzureSubscription(subscriptionID)
	if err != nil {
		return nil, err
	}

	// Lookup environment URI
	baseURI, err := factory.getEnvironmentBaseURI()
	if err != nil {
		return nil, err
	}

	// Create correct client based on type passed
	switch clientType {
	case SubscriptionsClientType:
		return subscriptions.NewClientWithBaseURI(baseURI), nil
	case VirtualMachinesClientType:
		return compute.NewVirtualMachinesClientWithBaseURI(baseURI, subscriptionID), nil
	case ManagedClustersClientType:
		return containerservice.NewManagedClustersClientWithBaseURI(baseURI, subscriptionID), nil
	}

	// If nothing matched, this is an error
	return nil, fmt.Errorf("Unknown client type %s", clientType)
}

// getDefaultEnvironmentName returns either a configured Azure environment name, or the public default
func (factory *multiEnvClientFactory) getDefaultEnvironmentName() string {
	envName, exists := os.LookupEnv(AzureEnvironmentEnvName)

	if !exists || envName == "" {
		envName = autorestAzure.PublicCloud.Name
	}

	return envName
}

// getEnvironmentBaseUri returns the ARM management URI for the configured Azure environment.
func (factory *multiEnvClientFactory) getEnvironmentBaseURI() (string, error) {
	envName := factory.getDefaultEnvironmentName()
	env, err := autorestAzure.EnvironmentFromName(envName)
	if err != nil {
		return "", err
	}
	return env.ResourceManagerEndpoint, nil
}

// getKeyVaultURISuffix returns the proper KeyVault URI suffix for the configured Azure environment.
func (factory *multiEnvClientFactory) getKeyVaultURISuffix() (string, error) {
	envName := factory.getDefaultEnvironmentName()
	env, err := autorestAzure.EnvironmentFromName(envName)
	if err != nil {
		return "", err
	}
	return env.KeyVaultDNSSuffix, nil
}

// getStorageURISuffix returns the proper storage URI suffix for the configured Azure environment
func (factory *multiEnvClientFactory) getStorageURISuffix() (string, error) {
	envName := factory.getDefaultEnvironmentName()
	env, err := autorestAzure.EnvironmentFromName(envName)
	if err != nil {
		return "", err
	}
	return env.StorageEndpointSuffix, nil
}
