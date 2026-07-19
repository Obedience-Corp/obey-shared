// Package brand provides shared Obedience Corp terminal brand primitives.
//
// The package is intentionally renderer-neutral. Consumers resolve semantic
// palette roles from their own product mode and runtime capabilities, then
// choose how to render those roles. The companion brand/lipgloss package
// adapts the palette and logo frames for Charm-based TUIs.
//
// Motion policy: use Capabilities.AllowMotion and FrameForCapabilities /
// FlameFrameFor so reduced motion, plain, and CI environments freeze
// decorative frames. ColorDepth is advisory for adapters; Resolve does not
// rewrite semantic hex tokens by depth (termenv/Lip Gloss own approximation).
package brand
