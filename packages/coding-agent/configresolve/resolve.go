// Package configresolve provides config value resolution matching TS resolve-config-value.ts.
package configresolve

import (
	"os"
	"strings"
)

// Value represents a resolved configuration value.
type Value struct {
	Value  string
	Source string // "env", "project", "global", "default"
	IsSet  bool
}

// Resolve resolves a config value from multiple sources in priority order.
func Resolve(envVar, projectVal, globalVal, defaultVal string) Value {
	if v := os.Getenv(envVar); v != "" {
		return Value{Value: v, Source: "env", IsSet: true}
	}
	if projectVal != "" {
		return Value{Value: projectVal, Source: "project", IsSet: true}
	}
	if globalVal != "" {
		return Value{Value: globalVal, Source: "global", IsSet: true}
	}
	return Value{Value: defaultVal, Source: "default", IsSet: defaultVal != ""}
}

// ResolveBool resolves a boolean config value.
func ResolveBool(envVar string, projectVal, globalVal *bool, defaultVal bool) bool {
	if v := os.Getenv(envVar); v != "" {
		return strings.EqualFold(v, "true") || v == "1"
	}
	if projectVal != nil {
		return *projectVal
	}
	if globalVal != nil {
		return *globalVal
	}
	return defaultVal
}

// ResolveInt resolves an int config value.
func ResolveInt(envVar string, projectVal, globalVal *int, defaultVal int) int {
	if v := os.Getenv(envVar); v != "" {
		var result int
		for _, c := range v {
			if c >= '0' && c <= '9' {
				result = result*10 + int(c-'0')
			} else {
				return defaultVal
			}
		}
		return result
	}
	if projectVal != nil {
		return *projectVal
	}
	if globalVal != nil {
		return *globalVal
	}
	return defaultVal
}
