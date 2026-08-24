package subst

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

var token = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

// Expand replaces ${VAR} tokens using vars, then the process environment.
func Expand(s string, vars map[string]string) string {
	return token.ReplaceAllStringFunc(s, func(m string) string {
		name := m[2 : len(m)-1]
		if v, ok := vars[name]; ok {
			return v
		}
		if v, ok := os.LookupEnv(name); ok {
			return v
		}
		return m
	})
}

// ExpandAll expands a slice of strings.
func ExpandAll(in []string, vars map[string]string) []string {
	out := make([]string, len(in))
	for i, s := range in {
		out[i] = Expand(s, vars)
	}
	return out
}

// Merge maps with later maps winning.
func Merge(maps ...map[string]string) map[string]string {
	out := make(map[string]string)
	for _, m := range maps {
		for k, v := range m {
			out[k] = v
		}
	}
	return out
}

// RequireExpanded reports leftover ${VAR} tokens.
func RequireExpanded(label string, values []string) error {
	for _, v := range values {
		if token.MatchString(v) {
			return fmt.Errorf("%s contains unresolved variable %s", label, strings.TrimSpace(v))
		}
	}
	return nil
}
