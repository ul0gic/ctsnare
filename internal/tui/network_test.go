package tui

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ul0gic/ctsnare/internal/domain"
)

// stubStore embeds domain.Store so it satisfies the interface while only the
// methods the network view exercises need real implementations. Calling an
// unimplemented method panics, which surfaces accidental dependencies in tests.
type stubStore struct {
	domain.Store
	clusters []domain.NetworkCluster
	err      error
}

func (s stubStore) NetworkClusters(_ context.Context) ([]domain.NetworkCluster, error) {
	return s.clusters, s.err
}

// runCmd executes a tea.Cmd and returns the message it produced (nil if none).
func runCmd(cmd tea.Cmd) tea.Msg {
	if cmd == nil {
		return nil
	}
	return cmd()
}

func TestNetworkModel_LoadAndDrill(t *testing.T) {
	clusters := []domain.NetworkCluster{
		{IP: "5.5.5.5", DomainCount: 3, LiveCount: 2, HostingProvider: "digitalocean", MaxScore: 9, SampleDomains: []string{"a.xyz", "b.xyz"}},
		{IP: "7.7.7.7", DomainCount: 2, LiveCount: 0, HostingProvider: "aws", MaxScore: 6},
	}
	m := NewNetworkModel(stubStore{clusters: clusters})

	// Init loads clusters; feeding the resulting message populates the table.
	msg := runCmd(m.Init())
	loaded, ok := msg.(ClustersLoadedMsg)
	if !ok {
		t.Fatalf("expected ClustersLoadedMsg from Init, got %T", msg)
	}
	m, _ = m.Update(loaded)
	if len(m.clusters) != 2 {
		t.Fatalf("expected 2 clusters loaded, got %d", len(m.clusters))
	}

	// Size the view so the table is interactive.
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	// Enter on the first cluster drills into its IP.
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	out := runCmd(cmd)
	drill, ok := out.(ShowClusterMsg)
	if !ok {
		t.Fatalf("expected ShowClusterMsg on enter, got %T", out)
	}
	if drill.IP != "5.5.5.5" {
		t.Errorf("expected drill into 5.5.5.5, got %s", drill.IP)
	}
}

func TestAppShowClusterDrillsIntoExplorer(t *testing.T) {
	app := NewApp(stubStore{}, nil, nil, nil, nil, "all")
	model, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app = asAppModel(t, model)

	model, _ = app.Update(ShowClusterMsg{IP: "203.0.113.7"})
	app = asAppModel(t, model)

	if app.activeView != viewExplorer {
		t.Fatalf("expected explorer view after cluster drill-in, got %d", app.activeView)
	}
	if app.explorer.filter.SharedIP != "203.0.113.7" {
		t.Errorf("expected explorer filtered by shared IP, got %q", app.explorer.filter.SharedIP)
	}
}

func TestNetworkModel_EmptyAndError(t *testing.T) {
	tests := []struct {
		name  string
		store stubStore
	}{
		{"empty result", stubStore{clusters: nil}},
		{"store error yields no clusters", stubStore{err: context.DeadlineExceeded}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewNetworkModel(tt.store)
			m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
			m, _ = m.Update(runCmd(m.Init()))
			if len(m.clusters) != 0 {
				t.Errorf("expected no clusters, got %d", len(m.clusters))
			}
			// View must render the empty state without panicking.
			if got := m.View(); got == "" {
				t.Error("expected non-empty view")
			}
		})
	}
}

func TestNetworkModel_NilStore(t *testing.T) {
	m := NewNetworkModel(nil)
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m, _ = m.Update(runCmd(m.Init()))
	if len(m.clusters) != 0 {
		t.Errorf("expected no clusters from nil store, got %d", len(m.clusters))
	}
}

func TestNetworkModel_SampleColumnResponsive(t *testing.T) {
	m := NewNetworkModel(stubStore{})

	// Narrow: no sample-domains column.
	m, _ = m.Update(tea.WindowSizeMsg{Width: networkSampleMinWidth - 1, Height: 24})
	if m.showSamp {
		t.Errorf("expected sample column hidden below %d cols", networkSampleMinWidth)
	}

	// Wide: sample-domains column shown.
	m, _ = m.Update(tea.WindowSizeMsg{Width: networkSampleMinWidth, Height: 24})
	if !m.showSamp {
		t.Errorf("expected sample column shown at %d cols", networkSampleMinWidth)
	}
}

func TestExplorerModel_IPColumnResponsive(t *testing.T) {
	m := NewExplorerModel(nil)

	m = m.resize(tea.WindowSizeMsg{Width: ipColumnMinWidth - 1, Height: 24})
	if m.showIP {
		t.Errorf("expected IP column hidden below %d cols", ipColumnMinWidth)
	}

	m = m.resize(tea.WindowSizeMsg{Width: ipColumnMinWidth, Height: 24})
	if !m.showIP {
		t.Errorf("expected IP column shown at %d cols", ipColumnMinWidth)
	}
}

func TestPrimaryIP(t *testing.T) {
	tests := []struct {
		name string
		ips  []string
		want string
	}{
		{"no IPs", nil, "—"},
		{"empty slice", []string{}, "—"},
		{"single IP", []string{"1.2.3.4"}, "1.2.3.4"},
		{"first of many", []string{"1.2.3.4", "5.6.7.8"}, "1.2.3.4"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := primaryIP(domain.Hit{ResolvedIPs: tt.ips}); got != tt.want {
				t.Errorf("primaryIP(%v) = %q, want %q", tt.ips, got, tt.want)
			}
		})
	}
}
