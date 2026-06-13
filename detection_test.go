package s3

// Unit tests for provider detection and config validation.
//
// Sections:
//   - service-type detection from endpoint / region / access-key (PKG-15)
//   - client config validation
//
// Provider detection is tested here at the Go level with ClientConfig values
// directly, instead of from Starlark scripts holding literal credentials:
// credentials are host-injected (PKG-15), so access-key-pattern detection is a
// host-config concern, not a script concern.

import "testing"

func TestDetectServiceType(t *testing.T) {
	cases := []struct {
		name string
		cfg  ClientConfig
		want string
	}{
		{"aws endpoint", ClientConfig{ServiceType: "auto", Endpoint: "https://s3.amazonaws.com"}, ProviderAWS},
		{"aws AKIA key", ClientConfig{ServiceType: "auto", AccessKey: "AKIAIOSFODNN7EXAMPLE"}, ProviderAWS},
		{"aws ASIA key", ClientConfig{ServiceType: "auto", AccessKey: "ASIAIOSFODNN7EXAMPLE"}, ProviderAWS},
		{"minio default key", ClientConfig{ServiceType: "auto", AccessKey: "minioadmin"}, ProviderMinIO},
		{"minio localhost", ClientConfig{ServiceType: "auto", Endpoint: "http://localhost:9000/"}, ProviderMinIO},
		{"cloudflare r2 endpoint", ClientConfig{ServiceType: "auto", Endpoint: "https://abc.r2.cloudflarestorage.com"}, ProviderCloudflare},
		{"digitalocean endpoint", ClientConfig{ServiceType: "auto", Endpoint: "https://nyc3.digitaloceanspaces.com"}, ProviderDigitalOcean},
		{"wasabi endpoint", ClientConfig{ServiceType: "auto", Endpoint: "https://s3.wasabisys.com"}, ProviderWasabi},
		{"alibaba endpoint", ClientConfig{ServiceType: "auto", Endpoint: "https://oss-cn-hangzhou.aliyuncs.com"}, ProviderAlibaba},
	}
	for _, c := range cases {
		got := c.cfg.detectServiceType()
		if got != c.want {
			t.Errorf("%s: detectServiceType() = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestClientConfigValidateNormalizes(t *testing.T) {
	// Validate fills defaults and resolves an empty/auto service type.
	c := ClientConfig{ServiceType: "", Endpoint: "https://s3.amazonaws.com"}
	if err := c.Validate(); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
	if c.ServiceType != ProviderAWS {
		t.Errorf("ServiceType = %q, want %q (auto-detected)", c.ServiceType, ProviderAWS)
	}
	if c.Timeout != 30 {
		t.Errorf("Timeout default = %d, want 30", c.Timeout)
	}
	if c.PartSize != 5*1024*1024 {
		t.Errorf("PartSize default = %d, want 5MiB", c.PartSize)
	}
	if c.Concurrency != 3 {
		t.Errorf("Concurrency default = %d, want 3", c.Concurrency)
	}
	if c.UserAgent == "" {
		t.Error("UserAgent should default to a non-empty value")
	}
}
