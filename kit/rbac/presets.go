package rbac

// Four built-in role presets. The Go developer customizes or creates new roles.

var RoleAdmin = Role{
	Name: "admin",
	Bus: BusPermissions{
		Publish:   TopicFilter{Allow: []string{"*"}},
		Subscribe: TopicFilter{Allow: []string{"*"}},
		Emit:      TopicFilter{Allow: []string{"*"}},
	},
	Commands:     CommandPermissions{Allow: []string{"*"}},
	Registration: RegistrationPermissions{Tools: true, Agents: true},
}

var RoleService = Role{
	Name: "service",
	Bus: BusPermissions{
		Publish:   TopicFilter{Allow: []string{"incoming.*", "events.*"}},
		Subscribe: TopicFilter{Allow: []string{"*.reply.*"}},
		Emit:      TopicFilter{Allow: []string{"events.*"}},
	},
	Commands: CommandPermissions{Allow: []string{
		"tools.call", "tools.list", "tools.resolve",
		"fs.read", "fs.write", "fs.list", "fs.stat",
		"secrets.get",
	}},
	Registration: RegistrationPermissions{Tools: true, Agents: false},
}

var RoleGateway = Role{
	Name: "gateway",
	Bus: BusPermissions{
		Publish:   TopicFilter{Allow: []string{"incoming.*", "gateway.*"}},
		Subscribe: TopicFilter{Allow: []string{"*.reply.*"}},
		Emit:      TopicFilter{Allow: []string{"gateway.*"}},
	},
	Commands:     CommandPermissions{Allow: []string{}},
	Registration: RegistrationPermissions{},
}

var RoleObserver = Role{
	Name: "observer",
	Bus: BusPermissions{
		Subscribe: TopicFilter{Allow: []string{"*"}},
	},
	Commands: CommandPermissions{Allow: []string{
		"tools.list", "kit.list", "registry.list", "registry.has",
	}},
	Registration: RegistrationPermissions{},
}
