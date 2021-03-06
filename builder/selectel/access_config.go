package selectel

import (
	"crypto/tls"
	"fmt"
	"os"

	"crypto/x509"
	"io/ioutil"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/packer/template/interpolate"
)

// AccessConfig is for common configuration related to openstack access
type AccessConfig struct {
	Username         string `mapstructure:"username"`
	UserID           string `mapstructure:"user_id"`
	Password         string `mapstructure:"password"`
	IdentityEndpoint string `mapstructure:"identity_endpoint"`
	TenantID         string `mapstructure:"tenant_id"`
	TenantName       string `mapstructure:"tenant_name"`
	DomainID         string `mapstructure:"domain_id"`
	DomainName       string `mapstructure:"domain_name"`
	Insecure         bool   `mapstructure:"insecure"`
	Region           string `mapstructure:"region"`
	EndpointType     string `mapstructure:"endpoint_type"`
	CACertFile       string `mapstructure:"cacert"`
	ClientCertFile   string `mapstructure:"cert"`
	ClientKeyFile    string `mapstructure:"key"`

	osClient *gophercloud.ProviderClient
}

func (c *AccessConfig) Prepare(ctx *interpolate.Context) []error {
	if c.EndpointType != "internal" && c.EndpointType != "internalURL" &&
		c.EndpointType != "admin" && c.EndpointType != "adminURL" &&
		c.EndpointType != "public" && c.EndpointType != "publicURL" &&
		c.EndpointType != "" {
		return []error{fmt.Errorf("Invalid endpoint type provided")}
	}

	if c.Region == "" {
		c.Region = os.Getenv("OS_REGION_NAME")
	}

	// Legacy RackSpace stuff. We're keeping this around to keep things BC.
	if c.Password == "" {
		c.Password = os.Getenv("SDK_PASSWORD")
	}
	if c.Region == "" {
		c.Region = os.Getenv("SDK_REGION")
	}
	if c.TenantName == "" {
		c.TenantName = os.Getenv("SDK_PROJECT")
	}
	if c.Username == "" {
		c.Username = os.Getenv("SDK_USERNAME")
	}
	if c.CACertFile == "" {
		c.CACertFile = os.Getenv("OS_CACERT")
	}
	if c.ClientCertFile == "" {
		c.ClientCertFile = os.Getenv("OS_CERT")
	}
	if c.ClientKeyFile == "" {
		c.ClientKeyFile = os.Getenv("OS_KEY")
	}

	// Get as much as possible from the end
	ao, _ := openstack.AuthOptionsFromEnv()

	// Make sure we reauth as needed
	ao.AllowReauth = true

	// Override values if we have them in our config
	overrides := []struct {
		From, To *string
	}{
		{&c.Username, &ao.Username},
		{&c.UserID, &ao.UserID},
		{&c.Password, &ao.Password},
		{&c.IdentityEndpoint, &ao.IdentityEndpoint},
		{&c.TenantID, &ao.TenantID},
		{&c.TenantName, &ao.TenantName},
		{&c.DomainID, &ao.DomainID},
		{&c.DomainName, &ao.DomainName},
	}
	for _, s := range overrides {
		if *s.From != "" {
			*s.To = *s.From
		}
	}

	// Build the client itself
	client, err := openstack.NewClient(ao.IdentityEndpoint)
	if err != nil {
		return []error{err}
	}

	tls_config := &tls.Config{}

	if c.CACertFile != "" {
		caCert, err := ioutil.ReadFile(c.CACertFile)
		if err != nil {
			return []error{err}
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		tls_config.RootCAs = caCertPool
	}

	// If we have insecure set, then create a custom HTTP client that
	// ignores SSL errors.
	if c.Insecure {
		tls_config.InsecureSkipVerify = true
	}

	if c.ClientCertFile != "" && c.ClientKeyFile != "" {
		cert, err := tls.LoadX509KeyPair(c.ClientCertFile, c.ClientKeyFile)
		if err != nil {
			return []error{err}
		}

		tls_config.Certificates = []tls.Certificate{cert}
	}

	transport := cleanhttp.DefaultTransport()
	transport.TLSClientConfig = tls_config
	client.HTTPClient.Transport = transport

	// Auth
	err = openstack.Authenticate(client, ao)
	if err != nil {
		return []error{err}
	}

	c.osClient = client
	return nil
}

func (c *AccessConfig) computeV2Client() (*gophercloud.ServiceClient, error) {
	return openstack.NewComputeV2(c.osClient, gophercloud.EndpointOpts{
		Region:       c.Region,
		Availability: c.getEndpointType(),
	})
}

func (c *AccessConfig) imageV2Client() (*gophercloud.ServiceClient, error) {
	return openstack.NewImageServiceV2(c.osClient, gophercloud.EndpointOpts{
		Region:       c.Region,
		Availability: c.getEndpointType(),
	})
}

func (c *AccessConfig) blockStorageV2Client() (*gophercloud.ServiceClient, error) {
	return openstack.NewBlockStorageV2(c.osClient, gophercloud.EndpointOpts{
		Region:       c.Region,
		Availability: c.getEndpointType(),
	})
}

func (c *AccessConfig) getEndpointType() gophercloud.Availability {
	if c.EndpointType == "internal" || c.EndpointType == "internalURL" {
		return gophercloud.AvailabilityInternal
	}
	if c.EndpointType == "admin" || c.EndpointType == "adminURL" {
		return gophercloud.AvailabilityAdmin
	}
	return gophercloud.AvailabilityPublic
}
