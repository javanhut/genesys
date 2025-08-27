package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/javanhut/genesys/pkg/provider"
	"github.com/spf13/cobra"
)

var (
	discoverProvider string
	discoverRegion   string
	discoverFormat   string
	discoverService  string
)

// NewDiscoverCommand creates the discover command
func NewDiscoverCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "discover",
		Aliases: []string{"list"},
		Short:   "List existing resources in your cloud account",
		Long: `List and discover existing cloud resources in your account.

Use this command to:
  • View all resources across services
  • Check for existing resources before creating new ones
  • Verify deployed resources after creation
  • Get resource details for management

Examples:
  genesys list resources              # List all resources
  genesys discover                    # Same as list (alias)
  genesys list --service storage      # List only storage resources (S3 buckets)
  genesys list --provider aws         # Use specific provider
  genesys list --region us-west-2     # Use specific region
  genesys list --output json          # JSON output format`,
		RunE: runDiscover,
	}

	cmd.Flags().StringVar(&discoverProvider, "provider", "aws", "Cloud provider (aws|gcp|azure)")
	cmd.Flags().StringVar(&discoverRegion, "region", "", "Cloud region")
	cmd.Flags().StringVarP(&discoverFormat, "output", "o", "human", "Output format (human|json)")
	cmd.Flags().StringVar(&discoverService, "service", "", "Specific service to discover (storage|compute|network|database|serverless)")

	return cmd
}

func runDiscover(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	fmt.Println("Discovering Resources")
	fmt.Println("=======================")
	fmt.Println()

	// Get the provider
	p, err := provider.Get(discoverProvider, map[string]string{
		"region": discoverRegion,
	})
	if err != nil {
		// For now, create a mock provider for testing
		fmt.Printf("Note: Using mock provider (real provider not yet implemented)\n\n")
		p = provider.NewMockProvider(discoverProvider, discoverRegion)
	}

	// Discover resources based on service filter
	discoveryResults := &DiscoveryResults{
		Provider: discoverProvider,
		Region:   discoverRegion,
		Services: make(map[string]*ServiceDiscovery),
	}

	startTime := time.Now()

	// Discover resources from each service
	if discoverService == "" || discoverService == "storage" {
		if err := discoverStorageResources(ctx, p, discoveryResults); err != nil {
			fmt.Printf("Warning: Failed to discover storage resources: %v\n", err)
		}
	}

	if discoverService == "" || discoverService == "compute" {
		if err := discoverComputeResources(ctx, p, discoveryResults); err != nil {
			fmt.Printf("Warning: Failed to discover compute resources: %v\n", err)
		}
	}

	if discoverService == "" || discoverService == "network" {
		if err := discoverNetworkResources(ctx, p, discoveryResults); err != nil {
			fmt.Printf("Warning: Failed to discover network resources: %v\n", err)
		}
	}

	if discoverService == "" || discoverService == "database" {
		if err := discoverDatabaseResources(ctx, p, discoveryResults); err != nil {
			fmt.Printf("Warning: Failed to discover database resources: %v\n", err)
		}
	}

	if discoverService == "" || discoverService == "serverless" {
		if err := discoverServerlessResources(ctx, p, discoveryResults); err != nil {
			fmt.Printf("Warning: Failed to discover serverless resources: %v\n", err)
		}
	}

	discoveryResults.Duration = time.Since(startTime)

	// Display results
	if discoverFormat == "json" {
		fmt.Println(discoveryResults.ToJSON())
	} else {
		fmt.Println(discoveryResults.ToHumanReadable())
	}

	return nil
}

func discoverStorageResources(ctx context.Context, p provider.Provider, results *DiscoveryResults) error {
	buckets, err := p.Storage().DiscoverBuckets(ctx)
	if err != nil {
		return err
	}

	results.Services["storage"] = &ServiceDiscovery{
		Name:      "Storage",
		Count:     len(buckets),
		Resources: make([]ResourceInfo, len(buckets)),
	}

	for i, bucket := range buckets {
		results.Services["storage"].Resources[i] = ResourceInfo{
			ID:           bucket.Name,
			Name:         bucket.Name,
			Type:         "bucket",
			Region:       bucket.Region,
			State:        "available",
			CreatedAt:    bucket.CreatedAt,
			Tags:         bucket.Tags,
			ProviderData: bucket.ProviderData,
		}
	}

	return nil
}

func discoverComputeResources(ctx context.Context, p provider.Provider, results *DiscoveryResults) error {
	instances, err := p.Compute().DiscoverInstances(ctx)
	if err != nil {
		return err
	}

	results.Services["compute"] = &ServiceDiscovery{
		Name:      "Compute",
		Count:     len(instances),
		Resources: make([]ResourceInfo, len(instances)),
	}

	for i, instance := range instances {
		results.Services["compute"].Resources[i] = ResourceInfo{
			ID:           instance.ID,
			Name:         instance.Name,
			Type:         "instance",
			Size:         string(instance.Type),
			State:        instance.State,
			PublicIP:     instance.PublicIP,
			PrivateIP:    instance.PrivateIP,
			Tags:         instance.Tags,
			ProviderData: instance.ProviderData,
		}
	}

	return nil
}

func discoverNetworkResources(ctx context.Context, p provider.Provider, results *DiscoveryResults) error {
	networks, err := p.Network().DiscoverNetworks(ctx)
	if err != nil {
		return err
	}

	results.Services["network"] = &ServiceDiscovery{
		Name:      "Network",
		Count:     len(networks),
		Resources: make([]ResourceInfo, len(networks)),
	}

	for i, network := range networks {
		results.Services["network"].Resources[i] = ResourceInfo{
			ID:    network.ID,
			Name:  network.Name,
			Type:  "vpc",
			CIDR:  network.CIDR,
			State: "available",

			Subnets:      len(network.Subnets),
			Tags:         network.Tags,
			ProviderData: network.ProviderData,
		}
	}

	return nil
}

func discoverDatabaseResources(ctx context.Context, p provider.Provider, results *DiscoveryResults) error {
	databases, err := p.Database().DiscoverDatabases(ctx)
	if err != nil {
		return err
	}

	results.Services["database"] = &ServiceDiscovery{
		Name:      "Database",
		Count:     len(databases),
		Resources: make([]ResourceInfo, len(databases)),
	}

	for i, database := range databases {
		results.Services["database"].Resources[i] = ResourceInfo{
			ID:      database.ID,
			Name:    database.Name,
			Type:    "database",
			Engine:  database.Engine,
			Version: database.Version,
			Size:    string(database.Size),

			Endpoint:     database.Endpoint,
			Port:         fmt.Sprintf("%d", database.Port),
			Tags:         database.Tags,
			ProviderData: database.ProviderData,
		}
	}

	return nil
}

func discoverServerlessResources(ctx context.Context, p provider.Provider, results *DiscoveryResults) error {
	functions, err := p.Serverless().DiscoverFunctions(ctx)
	if err != nil {
		return err
	}

	results.Services["serverless"] = &ServiceDiscovery{
		Name:      "Serverless",
		Count:     len(functions),
		Resources: make([]ResourceInfo, len(functions)),
	}

	for i, function := range functions {
		results.Services["serverless"].Resources[i] = ResourceInfo{
			ID:      function.ID,
			Name:    function.Name,
			Type:    "function",
			Runtime: function.Runtime,
			Memory:  function.Memory,
			Timeout: function.Timeout,

			Tags:         function.Tags,
			ProviderData: function.ProviderData,
		}
	}

	return nil
}

// DiscoveryResults holds the results of a discovery operation
type DiscoveryResults struct {
	Provider string                       `json:"provider"`
	Region   string                       `json:"region"`
	Services map[string]*ServiceDiscovery `json:"services"`
	Duration time.Duration                `json:"duration"`
}

// ServiceDiscovery holds discovery results for a specific service
type ServiceDiscovery struct {
	Name      string         `json:"name"`
	Count     int            `json:"count"`
	Resources []ResourceInfo `json:"resources"`
}

// ResourceInfo holds information about a discovered resource
type ResourceInfo struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Type         string                 `json:"type"`
	Region       string                 `json:"region,omitempty"`
	Size         string                 `json:"size,omitempty"`
	State        string                 `json:"state,omitempty"`
	PublicIP     string                 `json:"public_ip,omitempty"`
	PrivateIP    string                 `json:"private_ip,omitempty"`
	CIDR         string                 `json:"cidr,omitempty"`
	PublicAccess bool                   `json:"public_access,omitempty"`
	Subnets      int                    `json:"subnets,omitempty"`
	Engine       string                 `json:"engine,omitempty"`
	Version      string                 `json:"version,omitempty"`
	Endpoint     string                 `json:"endpoint,omitempty"`
	Port         string                 `json:"port,omitempty"`
	Runtime      string                 `json:"runtime,omitempty"`
	Memory       int                    `json:"memory,omitempty"`
	Timeout      int                    `json:"timeout,omitempty"`
	CreatedAt    time.Time              `json:"created_at,omitempty"`
	Tags         map[string]string      `json:"tags,omitempty"`
	ProviderData map[string]interface{} `json:"provider_data,omitempty"`
}

// ToHumanReadable formats the discovery results for human consumption
func (dr *DiscoveryResults) ToHumanReadable() string {
	var output strings.Builder

	output.WriteString(fmt.Sprintf("Provider: %s", strings.ToUpper(dr.Provider)))
	if dr.Region != "" {
		output.WriteString(fmt.Sprintf(" (%s)", dr.Region))
	}
	output.WriteString("\n")
	output.WriteString(fmt.Sprintf("Discovery completed in %v\n", dr.Duration.Round(time.Millisecond)))
	output.WriteString("\n")

	totalResources := 0
	for _, service := range dr.Services {
		totalResources += service.Count
	}

	if totalResources == 0 {
		output.WriteString("No resources found.\n")
		return output.String()
	}

	// Display each service's resources
	for _, service := range dr.Services {
		if service.Count == 0 {
			continue
		}

		output.WriteString(fmt.Sprintf("%s Resources (%d):\n", service.Name, service.Count))

		for _, resource := range service.Resources {
			output.WriteString(dr.formatResource(resource))
		}
		output.WriteString("\n")
	}

	// Summary
	output.WriteString("Summary:\n")
	output.WriteString(fmt.Sprintf("  Total resources found: %d\n", totalResources))

	// Service breakdown
	for _, service := range dr.Services {
		if service.Count > 0 {
			output.WriteString(fmt.Sprintf("  %s: %d\n", service.Name, service.Count))
		}
	}

	output.WriteString("\nTip: Use 'genesys adopt <resource-id>' to manage these resources with Genesys\n")

	return output.String()
}

// formatResource formats a single resource for display
func (dr *DiscoveryResults) formatResource(resource ResourceInfo) string {
	var output strings.Builder
	output.WriteString(fmt.Sprintf("  📦 %s", resource.Name))

	// Add type-specific information
	switch resource.Type {
	case "bucket":
		if resource.Region != "" {
			output.WriteString(fmt.Sprintf(" (%s)", resource.Region))
		}
		if !resource.CreatedAt.IsZero() {
			output.WriteString(fmt.Sprintf(" - Created: %s", resource.CreatedAt.Format("2006-01-02")))
		}
	case "instance":
		if resource.Type != "" {
			output.WriteString(fmt.Sprintf(" (%s)", resource.Type))
		}
		if resource.State != "" {
			output.WriteString(fmt.Sprintf(" - %s", resource.State))
		}
		if resource.PublicIP != "" {
			output.WriteString(fmt.Sprintf(" - %s", resource.PublicIP))
		}
	case "vpc":
		if resource.CIDR != "" {
			output.WriteString(fmt.Sprintf(" (%s)", resource.CIDR))
		}
		if resource.Subnets > 0 {
			output.WriteString(fmt.Sprintf(" - %d subnets", resource.Subnets))
		}
	case "database":
		if resource.Engine != "" {
			output.WriteString(fmt.Sprintf(" (%s", resource.Engine))
			if resource.Version != "" {
				output.WriteString(fmt.Sprintf(" %s", resource.Version))
			}
			if resource.Size != "" {
				output.WriteString(fmt.Sprintf(", %s", string(resource.Size)))
			}
			output.WriteString(")")
		}
		if resource.State != "" {
			output.WriteString(fmt.Sprintf(" - %s", resource.State))
		}
	case "function":
		if resource.Runtime != "" {
			output.WriteString(fmt.Sprintf(" (%s", resource.Runtime))
			if resource.Memory > 0 {
				output.WriteString(fmt.Sprintf(", %dMB", resource.Memory))
			}
			output.WriteString(")")
		}
		if resource.State != "" {
			output.WriteString(fmt.Sprintf(" - %s", resource.State))
		}
	}

	output.WriteString("\n")
	return output.String()
}

// ToJSON formats the discovery results as JSON
func (dr *DiscoveryResults) ToJSON() string {
	// Simple JSON formatting - in a real implementation, you'd use json.Marshal
	return fmt.Sprintf(`{
  "provider": "%s",
  "region": "%s",
  "duration": "%v",
  "services": {
%s
  }
}`, dr.Provider, dr.Region, dr.Duration, dr.formatServicesJSON())
}

func (dr *DiscoveryResults) formatServicesJSON() string {
	var services []string
	for name, service := range dr.Services {
		services = append(services, fmt.Sprintf(`    "%s": {
      "name": "%s",
      "count": %d,
      "resources": [%s]
    }`, name, service.Name, service.Count, dr.formatResourcesJSON(service.Resources)))
	}
	return strings.Join(services, ",\n")
}

func (dr *DiscoveryResults) formatResourcesJSON(resources []ResourceInfo) string {
	var resourceStrs []string
	for _, resource := range resources {
		resourceStrs = append(resourceStrs, fmt.Sprintf(`{
        "id": "%s",
        "name": "%s",
        "type": "%s"
      }`, resource.ID, resource.Name, resource.Type))
	}
	return strings.Join(resourceStrs, ",")
}
