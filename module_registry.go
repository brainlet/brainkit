package brainkit

import bkmodule "github.com/brainlet/brainkit/module"

// ModuleContext is what assembly layers hand to a factory before a Kit exists.
type ModuleContext = bkmodule.BuildContext

// ModuleFactory constructs a hot-mountable module from a config section.
type ModuleFactory = bkmodule.Factory

// ModuleDescriptor is optional metadata a factory can expose for CLI/docs.
type ModuleDescriptor = bkmodule.Descriptor

// ModuleMessageDescriptor describes one command/event/subscription topic in a
// module manifest.
type ModuleMessageDescriptor = bkmodule.MessageDescriptor

// ModuleCapabilityDescriptor describes one module host capability in a module
// manifest.
type ModuleCapabilityDescriptor = bkmodule.CapabilityDescriptor

// ModuleDescriber is an optional interface a ModuleFactory can implement.
type ModuleDescriber = bkmodule.Describer

// ModuleFactoryFunc adapts a plain function to ModuleFactory.
type ModuleFactoryFunc = bkmodule.FactoryFunc

// RegisterModule adds a factory to the global module registry.
func RegisterModule(name string, factory ModuleFactory) { bkmodule.Register(name, factory) }

// LookupModuleFactory returns the factory registered under name.
func LookupModuleFactory(name string) (ModuleFactory, bool) { return bkmodule.Lookup(name) }

// RegisteredModuleNames returns registered module names in stable order.
func RegisteredModuleNames() []string { return bkmodule.Names() }

// RegisteredModules returns descriptors for every registered module.
func RegisteredModules() []ModuleDescriptor { return bkmodule.Descriptors() }
