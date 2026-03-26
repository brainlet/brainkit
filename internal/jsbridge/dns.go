package jsbridge

import (
	"context"
	"fmt"
	"net"

	quickjs "github.com/buke/quickjs-go"
)

// DNSPolyfill provides dns.lookup and dns.promises for hostname resolution.
// pg uses dns.lookup() to resolve hostnames before TCP connect.
// MongoDB uses dns.promises.resolveSrv() for mongodb+srv:// URLs.
type DNSPolyfill struct {
	bridge *Bridge
}

// DNS creates a DNS polyfill.
func DNS() *DNSPolyfill { return &DNSPolyfill{} }

func (p *DNSPolyfill) Name() string { return "dns" }

func (p *DNSPolyfill) SetBridge(b *Bridge) { p.bridge = b }

func (p *DNSPolyfill) Setup(ctx *quickjs.Context) error {
	polyfill := p

	// __go_dns_lookup(hostname) → JSON { address, family }
	ctx.Globals().Set("__go_dns_lookup", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return qctx.ThrowError(fmt.Errorf("dns.lookup: hostname required"))
		}
		hostname := args[0].String()

		addrs, err := net.LookupHost(hostname)
		if err != nil {
			return qctx.ThrowError(fmt.Errorf("dns.lookup %s: %w", hostname, err))
		}
		if len(addrs) == 0 {
			return qctx.ThrowError(fmt.Errorf("dns.lookup %s: no addresses found", hostname))
		}

		addr := addrs[0]
		family := 4
		if ip := net.ParseIP(addr); ip != nil && ip.To4() == nil {
			family = 6
		}

		return qctx.NewString(fmt.Sprintf(`{"address":"%s","family":%d}`, addr, family))
	}))

	// __go_dns_lookup_async(hostname) → Promise<JSON { address, family }>
	ctx.Globals().Set("__go_dns_lookup_async", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return qctx.ThrowError(fmt.Errorf("dns.lookup: hostname required"))
		}
		hostname := args[0].String()

		return qctx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
			polyfill.bridge.Go(func(goCtx context.Context) {
				addrs, err := net.LookupHost(hostname)
				if err != nil || len(addrs) == 0 {
					errMsg := "no addresses found"
					if err != nil {
						errMsg = err.Error()
					}
					qctx.Schedule(func(qctx *quickjs.Context) {
						errVal := qctx.NewError(fmt.Errorf("dns.lookup %s: %s", hostname, errMsg))
						defer errVal.Free()
						reject(errVal)
					})
					return
				}

				addr := addrs[0]
				family := 4
				if ip := net.ParseIP(addr); ip != nil && ip.To4() == nil {
					family = 6
				}

				qctx.Schedule(func(qctx *quickjs.Context) {
					resolve(qctx.NewString(fmt.Sprintf(`{"address":"%s","family":%d}`, addr, family)))
				})
			})
		})
	}))

	return evalJS(ctx, dnsJS)
}

const dnsJS = `
(function() {
  "use strict";

  var _lookup = __go_dns_lookup;
  var _lookupAsync = __go_dns_lookup_async;

  globalThis.dns = {
    lookup: function(hostname, optionsOrCb, cb) {
      if (typeof optionsOrCb === "function") { cb = optionsOrCb; }
      if (typeof cb !== "function") { cb = function() {}; }
      try {
        var result = JSON.parse(_lookup(hostname));
        cb(null, result.address, result.family);
      } catch(e) {
        cb(e);
      }
    },
    resolve4: function(hostname, cb) {
      try {
        var result = JSON.parse(_lookup(hostname));
        if (typeof cb === "function") cb(null, [result.address]);
      } catch(e) {
        if (typeof cb === "function") cb(e);
      }
    },
    Resolver: class Resolver {
      resolve(hostname, rrtype, cb) { if (typeof cb === "function") cb(new Error("dns.Resolver not available")); }
    },
    promises: {
      lookup: async function(hostname) {
        var raw = await _lookupAsync(hostname);
        return JSON.parse(raw);
      },
      resolve4: async function(hostname) {
        var raw = await _lookupAsync(hostname);
        var result = JSON.parse(raw);
        return [result.address];
      },
      resolveSrv: async function() { return []; },
      resolveCname: async function() { return []; },
      resolvePtr: async function() { return []; },
    },
    ADDRCONFIG: 0,
    V4MAPPED: 0,
    NODATA: "ENODATA",
    FORMERR: "EFORMERR",
    SERVFAIL: "ESERVFAIL",
    NOTFOUND: "ENOTFOUND",
    TIMEOUT: "ETIMEOUT",
  };
})();
`
