// Package topology provides cross-kit routing ergonomics for
// brainkit runtimes.
//
// A Kit's namespace identifies it on the bus. Peer names are the
// developer-friendly shorthand a user writes in code:
//
//	resp, err := brainkit.Call(kit, ctx, req, brainkit.WithCallTo("analytics"))
//
// Without the topology module, "analytics" is treated as a raw
// namespace. With the module, the string is first resolved to a
// namespace by consulting the module's static peer list (and, when
// configured, a discovery.Provider).
//
// The module registers two bus commands:
//
//	peers.list     → PeersListResp { Peers, Namespaces }
//	peers.resolve  → PeersResolveResp { Namespace }
//
// These are the canonical query surface for "who is in this cluster"
// — the discovery module supplies the mechanism, topology supplies
// the routing semantics.
//
// Example — static peers only:
//
//	kit, _ := brainkit.New(brainkit.Config{
//	    Namespace: "orchestrator",
//	    Transport: brainkit.NATS(url),
//	    Modules: []brainkit.Module{
//	        topology.NewModule(topology.Config{
//	            Peers: []topology.Peer{
//	                {Name: "analytics", Namespace: "analytics-prod"},
//	                {Name: "ingest",    Namespace: "ingest-prod"},
//	            },
//	        }),
//	    },
//	})
//
// Example — combined with bus discovery:
//
//	bus := discovery.NewBus(discovery.BusConfig{Transport: kit.PresenceTransport()})
//	topology.NewModule(topology.Config{Discovery: bus})
package topology
