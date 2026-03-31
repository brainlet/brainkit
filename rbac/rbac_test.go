package rbac

import "testing"

func TestTopicFilter_Allows(t *testing.T) {
	tests := []struct {
		name   string
		filter TopicFilter
		topic  string
		want   bool
	}{
		{"wildcard allow", TopicFilter{Allow: []string{"*"}}, "anything", true},
		{"exact allow", TopicFilter{Allow: []string{"foo.bar"}}, "foo.bar", true},
		{"glob allow", TopicFilter{Allow: []string{"events.*"}}, "events.user.created", true},
		{"no match", TopicFilter{Allow: []string{"events.*"}}, "incoming.message", false},
		{"deny wins", TopicFilter{Allow: []string{"*"}, Deny: []string{"secrets.*"}}, "secrets.get", false},
		{"deny specific allow rest", TopicFilter{Allow: []string{"*"}, Deny: []string{"kit.deploy"}}, "tools.call", true},
		{"empty filter", TopicFilter{}, "anything", false},
		{"multiple allow", TopicFilter{Allow: []string{"incoming.*", "events.*"}}, "events.foo", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.filter.Allows(tt.topic)
			if got != tt.want {
				t.Errorf("Allows(%q) = %v, want %v", tt.topic, got, tt.want)
			}
		})
	}
}

func TestCommandPermissions_AllowsCommand(t *testing.T) {
	tests := []struct {
		name string
		perm CommandPermissions
		cmd  string
		want bool
	}{
		{"wildcard", CommandPermissions{Allow: []string{"*"}}, "anything", true},
		{"exact", CommandPermissions{Allow: []string{"tools.call", "fs.read"}}, "tools.call", true},
		{"not in list", CommandPermissions{Allow: []string{"tools.call"}}, "kit.deploy", false},
		{"deny overrides", CommandPermissions{Allow: []string{"*"}, Deny: []string{"kit.deploy"}}, "kit.deploy", false},
		{"empty", CommandPermissions{}, "anything", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.perm.AllowsCommand(tt.cmd)
			if got != tt.want {
				t.Errorf("AllowsCommand(%q) = %v, want %v", tt.cmd, got, tt.want)
			}
		})
	}
}

func TestManager_RoleForSource(t *testing.T) {
	m := NewManager(nil, "service")

	// Default role
	role := m.RoleForSource("unknown.ts")
	if role.Name != "service" {
		t.Fatalf("expected 'service', got %q", role.Name)
	}

	// Assign admin
	m.Assign("brainling.ts", "admin")
	role = m.RoleForSource("brainling.ts")
	if role.Name != "admin" {
		t.Fatalf("expected 'admin', got %q", role.Name)
	}

	// Revoke → back to default
	m.Revoke("brainling.ts")
	role = m.RoleForSource("brainling.ts")
	if role.Name != "service" {
		t.Fatalf("expected 'service' after revoke, got %q", role.Name)
	}
}

func TestManager_ListAssignments(t *testing.T) {
	m := NewManager(nil, "service")
	m.Assign("a.ts", "admin")
	m.Assign("b.ts", "observer")

	assignments := m.ListAssignments()
	if len(assignments) != 2 {
		t.Fatalf("expected 2 assignments, got %d", len(assignments))
	}
}

func TestManager_ListRoles(t *testing.T) {
	m := NewManager(nil, "service")
	roles := m.ListRoles()
	if len(roles) != 4 {
		t.Fatalf("expected 4 preset roles, got %d", len(roles))
	}
}

func TestManager_CustomRole(t *testing.T) {
	custom := map[string]Role{
		"restricted": {
			Bus: BusPermissions{
				Publish: TopicFilter{Allow: []string{"logs.*"}},
			},
			Commands: CommandPermissions{Allow: []string{"tools.list"}},
		},
	}
	m := NewManager(custom, "restricted")

	role := m.RoleForSource("any.ts")
	if role.Name != "restricted" {
		t.Fatalf("expected 'restricted', got %q", role.Name)
	}
	if !role.Bus.Publish.Allows("logs.info") {
		t.Fatal("expected logs.* publish allowed")
	}
	if role.Bus.Publish.Allows("incoming.message") {
		t.Fatal("expected incoming.* publish denied")
	}
}

func TestManager_AssignInvalidRole(t *testing.T) {
	m := NewManager(nil, "service")
	err := m.Assign("x.ts", "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent role")
	}
}

func TestIsOwnMailbox(t *testing.T) {
	tests := []struct {
		source string
		topic  string
		want   bool
	}{
		{"chat.ts", "ts.chat.greet", true},
		{"chat.ts", "ts.chat.ask", true},
		{"chat.ts", "ts.other.ask", false},
		{"nested/svc.ts", "ts.nested.svc.rpc", true},
		{"chat.ts", "incoming.message", false},
		{"", "ts.chat.greet", false},
	}

	for _, tt := range tests {
		got := IsOwnMailbox(tt.source, tt.topic)
		if got != tt.want {
			t.Errorf("IsOwnMailbox(%q, %q) = %v, want %v", tt.source, tt.topic, got, tt.want)
		}
	}
}

func TestPresets_AdminCanDoEverything(t *testing.T) {
	if !RoleAdmin.Bus.Publish.Allows("anything") {
		t.Fatal("admin should publish to anything")
	}
	if !RoleAdmin.Bus.Subscribe.Allows("anything") {
		t.Fatal("admin should subscribe to anything")
	}
	if !RoleAdmin.Commands.AllowsCommand("kit.deploy") {
		t.Fatal("admin should call any command")
	}
	if !RoleAdmin.Registration.Tools || !RoleAdmin.Registration.Agents {
		t.Fatal("admin should register tools and agents")
	}
}

func TestPresets_ObserverCantPublish(t *testing.T) {
	if RoleObserver.Bus.Publish.Allows("anything") {
		t.Fatal("observer should not publish")
	}
	if !RoleObserver.Bus.Subscribe.Allows("events.foo") {
		t.Fatal("observer should subscribe to anything")
	}
	if RoleObserver.Commands.AllowsCommand("kit.deploy") {
		t.Fatal("observer should not call kit.deploy")
	}
	if RoleObserver.Commands.AllowsCommand("tools.list") == false {
		t.Fatal("observer should call tools.list")
	}
}
