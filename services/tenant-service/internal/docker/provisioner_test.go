package docker

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/docker/docker/api/types/container"
	dockernetwork "github.com/docker/docker/api/types/network"

	"slate/services/tenant-service/internal/models"
)

// fakeDockerAPI records calls so tests can assert on them without Docker.
type fakeDockerAPI struct {
	mu         sync.Mutex
	containers map[string]*fakeContainer // id → container
	networks   map[string]string         // id → name
	failOn     string                    // service name that should fail to start
	failKind   string                    // "create" | "start"
}

type fakeContainer struct {
	ID       string
	Name     string
	Labels   map[string]string
	Env      []string
	State    string
	Networks []string
}

func newFakeDockerAPI() *fakeDockerAPI {
	return &fakeDockerAPI{
		containers: map[string]*fakeContainer{},
		networks:   map[string]string{},
	}
}

func (f *fakeDockerAPI) ContainerCreate(_ context.Context, cfg *container.Config, _ *container.HostConfig, netCfg *dockernetwork.NetworkingConfig, _ *ocispec, name string) (containerCreateResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failKind == "create" && strings.Contains(name, f.failOn) {
		return containerCreateResponse{}, fmt.Errorf("simulated create failure for %s", name)
	}
	id := "cid-" + name
	nets := []string{}
	if netCfg != nil {
		for n := range netCfg.EndpointsConfig {
			nets = append(nets, n)
		}
	}
	f.containers[id] = &fakeContainer{
		ID: id, Name: name, Labels: cfg.Labels, Env: cfg.Env,
		State: "created", Networks: nets,
	}
	return containerCreateResponse{ID: id}, nil
}

func (f *fakeDockerAPI) ContainerStart(_ context.Context, id string, _ container.StartOptions) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	c, ok := f.containers[id]
	if !ok {
		return fmt.Errorf("container not found")
	}
	if f.failKind == "start" && strings.Contains(c.Name, f.failOn) {
		return fmt.Errorf("simulated start failure for %s", c.Name)
	}
	c.State = "running"
	return nil
}

func (f *fakeDockerAPI) ContainerStop(_ context.Context, id string, _ container.StopOptions) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if c, ok := f.containers[id]; ok {
		c.State = "exited"
	}
	return nil
}

func (f *fakeDockerAPI) ContainerRemove(_ context.Context, id string, _ container.RemoveOptions) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.containers, id)
	return nil
}

func (f *fakeDockerAPI) ContainerList(_ context.Context, opts container.ListOptions) ([]dockerContainerSummary, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	wantLabels := map[string]string{}
	opts.Filters.WalkValues("label", func(v string) error {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) == 2 {
			wantLabels[parts[0]] = parts[1]
		}
		return nil
	})
	out := make([]dockerContainerSummary, 0, len(f.containers))
	for _, c := range f.containers {
		matches := true
		for k, v := range wantLabels {
			if c.Labels[k] != v {
				matches = false
				break
			}
		}
		if !matches {
			continue
		}
		out = append(out, dockerContainerSummary{
			ID: c.ID, Names: []string{"/" + c.Name}, State: c.State, Labels: c.Labels,
		})
	}
	return out, nil
}

func (f *fakeDockerAPI) NetworkCreate(_ context.Context, name string, _ dockernetwork.CreateOptions) (dockerNetworkCreateResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id := "net-" + name
	f.networks[id] = name
	return dockerNetworkCreateResponse{ID: id}, nil
}
func (f *fakeDockerAPI) NetworkConnect(context.Context, string, string, *dockernetwork.EndpointSettings) error {
	return nil
}
func (f *fakeDockerAPI) NetworkRemove(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.networks, id)
	return nil
}
func (f *fakeDockerAPI) NetworkList(_ context.Context, opts dockernetwork.ListOptions) ([]dockerNetworkResource, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	want := ""
	opts.Filters.WalkValues("name", func(v string) error {
		want = v
		return nil
	})
	out := []dockerNetworkResource{}
	for id, name := range f.networks {
		if want != "" && name != want {
			continue
		}
		out = append(out, dockerNetworkResource{ID: id, Name: name})
	}
	return out, nil
}
func (f *fakeDockerAPI) Close() error { return nil }

func testTenant() *models.TenantV2 {
	return &models.TenantV2{ID: "t-1", Slug: "eastfield", Name: "Eastfield"}
}

func TestProvision_CreatesAllEightServices(t *testing.T) {
	api := newFakeDockerAPI()
	p := NewWithDockerAPI(api, Env{RedisURL: "redis:6379", KafkaBrokers: "kafka:9092", MinioEndpoint: "minio:9000", DBDSN: "dsn", PlatformPublicKey: "pk", AnthropicAPIKey: "ak", AITokenBudget: "1000"})

	ids, err := p.Provision(context.Background(), testTenant())
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 8 {
		t.Fatalf("expected 8 containers, got %d", len(ids))
	}

	// Verify each expected service was created with the correct labels+env.
	services := []string{}
	for _, c := range api.containers {
		services = append(services, c.Labels["service"])
	}
	sort.Strings(services)
	want := []string{"ai", "assignment", "content", "course", "discussion", "scheduling", "user-auth", "video"}
	if strings.Join(services, ",") != strings.Join(want, ",") {
		t.Fatalf("expected %v, got %v", want, services)
	}

	// Tenant network was created.
	if len(api.networks) != 1 {
		t.Fatalf("expected exactly 1 tenant network, got %d", len(api.networks))
	}
	foundName := ""
	for _, n := range api.networks {
		foundName = n
	}
	if foundName != "tenant-eastfield-network" {
		t.Fatalf("expected network tenant-eastfield-network, got %s", foundName)
	}
}

func TestProvision_StampsRequiredEnv(t *testing.T) {
	api := newFakeDockerAPI()
	p := NewWithDockerAPI(api, Env{
		RedisURL:          "redis:6379",
		KafkaBrokers:      "kafka:9092",
		MinioEndpoint:     "minio:9000",
		DBDSN:             "postgres://dsn",
		PlatformPublicKey: "PLATFORM_KEY",
		AnthropicAPIKey:   "sk-ant-123",
		AITokenBudget:     "500000",
	})
	if _, err := p.Provision(context.Background(), testTenant()); err != nil {
		t.Fatal(err)
	}

	containers := map[string]*fakeContainer{}
	for _, c := range api.containers {
		containers[c.Labels["service"]] = c
	}

	mustHave := func(envs []string, key, val string) {
		t.Helper()
		want := key + "=" + val
		for _, e := range envs {
			if e == want {
				return
			}
		}
		t.Fatalf("env missing %s in %v", want, envs)
	}

	// Every service gets the base env.
	for svc, c := range containers {
		mustHave(c.Env, "TENANT_ID", "t-1")
		mustHave(c.Env, "TENANT_SLUG", "eastfield")
		mustHave(c.Env, "DB_SCHEMA", "tenant_eastfield")
		mustHave(c.Env, "REDIS_URL", "redis:6379")
		mustHave(c.Env, "KAFKA_BROKERS", "kafka:9092")
		mustHave(c.Env, "MINIO_ENDPOINT", "minio:9000")
		mustHave(c.Env, "DB_DSN", "postgres://dsn")
		mustHave(c.Env, "SERVICE_NAME", svc)
	}

	// user-auth + ai both receive PLATFORM_PUBLIC_KEY.
	for _, svc := range []string{"user-auth", "ai"} {
		mustHave(containers[svc].Env, "PLATFORM_PUBLIC_KEY", "PLATFORM_KEY")
	}
	// ai additionally gets ANTHROPIC_API_KEY + TENANT_AI_TOKEN_BUDGET.
	mustHave(containers["ai"].Env, "ANTHROPIC_API_KEY", "sk-ant-123")
	mustHave(containers["ai"].Env, "TENANT_AI_TOKEN_BUDGET", "500000")

	// course (no platform key required) must NOT have PLATFORM_PUBLIC_KEY.
	for _, e := range containers["course"].Env {
		if strings.HasPrefix(e, "PLATFORM_PUBLIC_KEY=") {
			t.Fatalf("course container should not receive PLATFORM_PUBLIC_KEY")
		}
	}
}

func TestProvision_RollsBackOnFailure(t *testing.T) {
	api := newFakeDockerAPI()
	api.failOn = "discussion"
	api.failKind = "start"
	p := NewWithDockerAPI(api, Env{})

	if _, err := p.Provision(context.Background(), testTenant()); err == nil {
		t.Fatal("expected provisioning to fail")
	} else if !strings.Contains(err.Error(), "discussion") {
		t.Fatalf("expected error to mention the failing service, got %v", err)
	}

	// After rollback the fake should contain zero containers and zero networks.
	if len(api.containers) != 0 {
		t.Fatalf("expected rollback to clear containers, have %d", len(api.containers))
	}
	if len(api.networks) != 0 {
		t.Fatalf("expected rollback to clear network, have %d", len(api.networks))
	}
}

func TestDeprovision_RemovesContainersAndNetwork(t *testing.T) {
	api := newFakeDockerAPI()
	p := NewWithDockerAPI(api, Env{})
	if _, err := p.Provision(context.Background(), testTenant()); err != nil {
		t.Fatal(err)
	}

	if err := p.Deprovision(context.Background(), "t-1"); err != nil {
		t.Fatal(err)
	}
	if len(api.containers) != 0 {
		t.Fatalf("expected deprovision to remove all containers")
	}
	if len(api.networks) != 0 {
		t.Fatalf("expected deprovision to remove tenant network")
	}
}

func TestListTenantContainers_ReturnsInventory(t *testing.T) {
	api := newFakeDockerAPI()
	p := NewWithDockerAPI(api, Env{})
	if _, err := p.Provision(context.Background(), testTenant()); err != nil {
		t.Fatal(err)
	}
	infos, err := p.ListTenantContainers(context.Background(), "t-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(infos) != 8 {
		t.Fatalf("expected 8 ContainerInfo entries, got %d", len(infos))
	}
	seen := map[string]bool{}
	for _, info := range infos {
		seen[info.Service] = true
	}
	for _, spec := range ServiceSpecs {
		if !seen[spec.Name] {
			t.Fatalf("expected ListTenantContainers to include %s", spec.Name)
		}
	}
}

// Sanity: the exported Provisioner interface is actually satisfied by
// TenantProvisioner so main.go can accept it.
var _ Provisioner = (*TenantProvisioner)(nil)

// Helper — ensure errors package is referenced.
var _ = errors.New
