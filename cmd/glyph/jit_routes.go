package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/glyphlang/glyph/pkg/ast"
	"github.com/glyphlang/glyph/pkg/jit"
)

// JITRouteManager manages JIT compilation and tier progression for routes.
// It wraps the JIT compiler to provide live recompilation of hot routes.
type JITRouteManager struct {
	jitCompiler *jit.JITCompiler
	routes      map[string]*ast.Route // route name → AST for recompilation
	mu          sync.RWMutex
}

// NewJITRouteManager creates a JIT route manager with default settings.
func NewJITRouteManager() *JITRouteManager {
	return &JITRouteManager{
		jitCompiler: jit.NewJITCompiler(),
		routes:      make(map[string]*ast.Route),
	}
}

// CompileRoute compiles a route through the JIT pipeline with tier-aware optimization.
// Returns bytecode at the appropriate optimization level.
func (m *JITRouteManager) CompileRoute(route *ast.Route) ([]byte, error) {
	name := fmt.Sprintf("%s %s", route.Method, route.Path)

	m.mu.Lock()
	m.routes[name] = route
	m.mu.Unlock()

	return m.jitCompiler.CompileRoute(name, route)
}

// RecordExecution records a route execution and returns upgraded bytecode if
// the route was recompiled to a higher tier, or nil if no recompilation occurred.
func (m *JITRouteManager) RecordExecution(routeName string, executionTime time.Duration) []byte {
	m.jitCompiler.RecordExecution(routeName, executionTime)

	// Check if the route should be recompiled
	m.mu.RLock()
	route, exists := m.routes[routeName]
	m.mu.RUnlock()

	if !exists {
		return nil
	}

	// Try to get recompiled bytecode (will return cached if no recompile needed)
	unit, ok := m.jitCompiler.GetUnit(routeName)
	if !ok {
		return nil
	}

	// Use CompileRoute which internally checks shouldRecompile
	bytecode, err := m.jitCompiler.CompileRoute(routeName, route)
	if err != nil {
		return nil
	}

	// Check if tier actually changed (bytecode was upgraded)
	newUnit, ok := m.jitCompiler.GetUnit(routeName)
	if !ok || newUnit.Tier == unit.Tier {
		return nil
	}

	printInfo(fmt.Sprintf("JIT tier upgrade: %s → %s (%s)",
		tierName(unit.Tier), tierName(newUnit.Tier), routeName))

	return bytecode
}

// Stats returns JIT statistics.
func (m *JITRouteManager) Stats() jit.JITStats {
	return m.jitCompiler.GetStats()
}

// HotPaths returns currently hot routes.
func (m *JITRouteManager) HotPaths() []string {
	return m.jitCompiler.GetHotPaths()
}

// DetailedStats returns comprehensive JIT metrics.
func (m *JITRouteManager) DetailedStats() map[string]interface{} {
	return m.jitCompiler.GetDetailedStats()
}

func tierName(tier jit.OptimizationTier) string {
	switch tier {
	case jit.TierInterpreted:
		return "interpreted"
	case jit.TierBaseline:
		return "baseline"
	case jit.TierOptimized:
		return "optimized"
	case jit.TierHighlyOptimized:
		return "aggressive"
	default:
		return "unknown"
	}
}
