// Package probes periodically exercises registered AI providers,
// vector stores, and storage backends so Kit.Health carries live
// healthy/unhealthy state. Probe results feed
// ProviderInfo / StorageInfo / VectorStoreInfo.
// Probe sweeps are mounted under a context-aware module loop so hot-unmount can
// cancel and join in-flight probe work.
//
// Status: beta.
package probes
