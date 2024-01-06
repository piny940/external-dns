/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cloudflaretunnel

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	cloudflare "github.com/cloudflare/cloudflare-go"
	log "github.com/sirupsen/logrus"

	"sigs.k8s.io/external-dns/endpoint"
	"sigs.k8s.io/external-dns/plan"
	"sigs.k8s.io/external-dns/provider"
)

type cloudFlareDNS interface {
	GetTunnelConfiguration(ctx context.Context, rc *cloudflare.ResourceContainer, tunnelID string) (cloudflare.TunnelConfigurationResult, error)
	UpdateTunnelConfiguration(ctx context.Context, rc *cloudflare.ResourceContainer, rp cloudflare.TunnelConfigurationParams) (cloudflare.TunnelConfigurationResult, error)
}

type accountService struct {
	service *cloudflare.API
}

func (a accountService) GetTunnelConfiguration(ctx context.Context, rc *cloudflare.ResourceContainer, tunnelID string) (cloudflare.TunnelConfigurationResult, error) {
	return a.service.GetTunnelConfiguration(ctx, rc, tunnelID)
}

func (a accountService) UpdateTunnelConfiguration(ctx context.Context, rc *cloudflare.ResourceContainer, cp cloudflare.TunnelConfigurationParams) (cloudflare.TunnelConfigurationResult, error) {
	return a.service.UpdateTunnelConfiguration(ctx, rc, cp)
}

// CloudFlareProvider is an implementation of Provider for CloudFlare DNS.
type CloudFlareProvider struct {
	provider.BaseProvider
	Client cloudFlareDNS
	// only consider hosted zones managing domains ending in this suffix
	domainFilter      endpoint.DomainFilter
	accountIdentifier string
	tunnelID          string
}

// NewCloudFlareProvider initializes a new CloudFlare DNS based Provider.
func NewCloudFlareProvider(domainFilter endpoint.DomainFilter, zoneIDFilter provider.ZoneIDFilter, dnsRecordsPerPage int) (*CloudFlareProvider, error) {
	// initialize via chosen auth method and returns new API object
	var (
		config *cloudflare.API
		err    error
	)
	if os.Getenv("CF_API_TOKEN") != "" {
		token := os.Getenv("CF_API_TOKEN")
		if strings.HasPrefix(token, "file:") {
			tokenBytes, err := os.ReadFile(strings.TrimPrefix(token, "file:"))
			if err != nil {
				return nil, fmt.Errorf("failed to read CF_API_TOKEN from file: %w", err)
			}
			token = string(tokenBytes)
		}
		config, err = cloudflare.NewWithAPIToken(token)
	} else {
		config, err = cloudflare.New(os.Getenv("CF_API_KEY"), os.Getenv("CF_API_EMAIL"))
	}
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cloudflare provider: %v", err)
	}
	tunnelID := os.Getenv("CF_TUNNEL_ID")
	if tunnelID == "" {
		return nil, fmt.Errorf("tunnelID is empty")
	}
	accountID := os.Getenv("CF_ACCOUNT_ID")
	if accountID == "" {
		return nil, fmt.Errorf("accountID is empty")
	}
	provider := &CloudFlareProvider{
		// Client: config,
		Client:            accountService{service: config},
		domainFilter:      domainFilter,
		tunnelID:          tunnelID,
		accountIdentifier: accountID,
	}
	return provider, nil
}

// Records returns the list of records.
func (p *CloudFlareProvider) Records(ctx context.Context) ([]*endpoint.Endpoint, error) {
	resourceContainer := cloudflare.AccountIdentifier(p.accountIdentifier)
	result, err := p.Client.GetTunnelConfiguration(ctx, resourceContainer, p.tunnelID)
	if err != nil {
		return nil, fmt.Errorf("Failed to get tunnel configuration: %v", err)
	}
	records := make([]*endpoint.Endpoint, len(result.Config.Ingress))
	for _, rule := range result.Config.Ingress {
		target, err := extractTarget(rule.Service)
		if err != nil {
			continue
		}
		records = append(records, endpoint.NewEndpoint(
			rule.Hostname,
			"A",
			rule.Service,
			target,
		))
	}
	return records, nil
}

// ApplyChanges applies a given set of changes in a given zone.
func (p *CloudFlareProvider) ApplyChanges(ctx context.Context, changes *plan.Changes) error {
	// return early if there is nothing to change
	if len(changes.Create) == 0 && len(changes.UpdateNew) == 0 && len(changes.Delete) == 0 {
		log.Info("All records are already up to date")
		return nil
	}
	resourceContainer := cloudflare.AccountIdentifier(p.accountIdentifier)
	oldConf, err := p.Client.GetTunnelConfiguration(ctx, resourceContainer, p.tunnelID)
	if err != nil {
		log.Errorf("Failed to get tunnel configuration: %v", err)
		return err
	}

	ingresses := make(map[string]cloudflare.UnvalidatedIngressRule)
	for _, ingress := range oldConf.Config.Ingress {
		ingresses[ingress.Hostname] = ingress
	}

	for _, change := range changes.Create {
		ingresses[change.DNSName] = newIngressRule(change)
	}
	for _, change := range changes.UpdateNew {
		ingresses[change.DNSName] = newIngressRule(change)
	}
	for _, change := range changes.Delete {
		delete(ingresses, change.DNSName)
	}

	newIngress := make([]cloudflare.UnvalidatedIngressRule, len(ingresses))
	for _, v := range ingresses {
		newIngress = append(newIngress, v)
	}
	newConf := cloudflare.TunnelConfigurationParams{
		TunnelID: p.tunnelID,
		Config: cloudflare.TunnelConfiguration{
			Ingress:       newIngress,
			OriginRequest: oldConf.Config.OriginRequest,
			WarpRouting:   oldConf.Config.WarpRouting,
		},
	}
	_, err = p.Client.UpdateTunnelConfiguration(ctx, resourceContainer, newConf)
	if err != nil {
		log.Errorf("Unable to update tunnel configuration: %v", err)
	}

	return nil
}

func includesHost(ingress []cloudflare.UnvalidatedIngressRule, hostname string) bool {
	for _, item := range ingress {
		if item.Hostname == hostname {
			return true
		}
	}
	return false
}

func newIngressRule(e *endpoint.Endpoint) cloudflare.UnvalidatedIngressRule {
	return cloudflare.UnvalidatedIngressRule{
		Hostname: e.DNSName,
		Path:     "/",
		Service:  fmt.Sprintf("https://%v:443", e.Targets[0]),
		OriginRequest: &cloudflare.OriginRequestConfig{
			Http2Origin: boolPtr(true),
			NoTLSVerify: boolPtr(true),
		},
	}
}

func extractTarget(cfService string) (string, error) {
	pattern := `(([a-zA-Z0-9]+\.)+[a-zA-Z0-9]+)|localhost`
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", err
	}
	match := re.FindString(cfService)
	if match == "" {
		return "", fmt.Errorf("there is no match. regexp: %s, cfService: %s", pattern, cfService)
	}
	return match, nil
}

// AdjustEndpoints modifies the endpoints as needed by the specific provider
func (p *CloudFlareProvider) AdjustEndpoints(endpoints []*endpoint.Endpoint) ([]*endpoint.Endpoint, error) {
	return endpoints, nil
}

// boolPtr is used as a helper function to return a pointer to a boolean
// Needed because some parameters require a pointer.
func boolPtr(b bool) *bool {
	return &b
}
