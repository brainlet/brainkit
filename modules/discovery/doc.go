// Package discovery provides peer presence + a Provider abstraction
// for cross-kit networking. Static and bus-mode providers ship here;
// modules/topology consumes the Provider to expose the
// peers.list / peers.resolve bus commands and WithCallTo routing.
//
// The Module's lifecycle is presence-only — Init registers the local
// Kit with the provider, Close tears it down. Users that want the
// bus surface must also wire modules/topology.
package discovery
