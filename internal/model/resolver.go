package model

import (
	"regexp"
	"strings"
)

// Resolution holds the result of model name resolution.
type Resolution struct {
	InternalID   string // ID to send to Kiro (CodeWhisperer format or passthrough)
	ExternalName string // Normalized name to return to clients
	Source       string // "static", "passthrough"
	Original     string // What the client sent
}

// Resolver performs smart model name resolution.
// It normalizes client model names and maps them to CodeWhisperer IDs.
// Unknown models are passed through to Kiro (gateway, not gatekeeper).
type Resolver struct {
	forwardMap  map[string]string
	reverseMap  map[string]string
	modelList   []ModelInfo
}

// NewResolver creates a resolver with the default model maps.
func NewResolver() *Resolver {
	return &Resolver{
		forwardMap:  ForwardModelMap,
		reverseMap:  ReverseModelMap,
		modelList:   SupportedModels(),
	}
}

// Resolve takes a client-provided model name and returns a Resolution.
// The pipeline: normalize → static lookup → passthrough.
func (r *Resolver) Resolve(clientModel string) Resolution {
	normalized := NormalizeModelName(clientModel)

	// Check static forward map with normalized name
	if internalID, ok := r.forwardMap[normalized]; ok {
		return Resolution{
			InternalID:   internalID,
			ExternalName: normalized,
			Source:       "static",
			Original:     clientModel,
		}
	}

	// Check with original name (for versioned names like "claude-3-5-sonnet-20241022")
	if internalID, ok := r.forwardMap[clientModel]; ok {
		externalName := normalized
		if externalName == clientModel {
			// If normalization didn't change it, use the map's reverse lookup
			if rev, ok := r.reverseMap[internalID]; ok {
				externalName = rev
			}
		}
		return Resolution{
			InternalID:   internalID,
			ExternalName: externalName,
			Source:       "static",
			Original:     clientModel,
		}
	}

	// Passthrough: let Kiro decide
	return Resolution{
		InternalID:   normalized,
		ExternalName: normalized,
		Source:       "passthrough",
		Original:     clientModel,
	}
}

// ReverseResolve maps a CodeWhisperer ID back to a public Anthropic name.
func (r *Resolver) ReverseResolve(kiroModel string) string {
	if name, ok := r.reverseMap[kiroModel]; ok {
		return name
	}
	return kiroModel
}

// ListModels returns the public model list for the /v1/models endpoint.
func (r *Resolver) ListModels() []ModelInfo {
	return r.modelList
}

// --- Normalization patterns ---

var (
	// claude-{family}-{major}-{minor}(-{date|latest})
	// e.g. claude-sonnet-4-5, claude-sonnet-4-5-20250929, claude-haiku-4-5-latest
	patStandard = regexp.MustCompile(
		`^(claude-(?:haiku|sonnet|opus)-\d+)-(\d{1,2})(?:-(?:\d{8}|latest))?$`,
	)

	// claude-{family}-{major}(-{date})
	// e.g. claude-sonnet-4, claude-sonnet-4-20250514
	patNoMinor = regexp.MustCompile(
		`^(claude-(?:haiku|sonnet|opus)-\d+)(?:-\d{8})?$`,
	)

	// claude-{major}-{minor}-{family}(-{date|latest})
	// e.g. claude-3-7-sonnet, claude-3-5-sonnet-20241022
	patLegacy = regexp.MustCompile(
		`^(claude)-(\d+)-(\d+)-(haiku|sonnet|opus)(?:-(?:\d{8}|latest))?$`,
	)

	// claude-{family}-{major}.{minor}-{date}
	// e.g. claude-sonnet-4.5-20250514
	patDotWithDate = regexp.MustCompile(
		`^(claude-(?:haiku|sonnet|opus)-\d+\.\d+)-\d{8}$`,
	)

	// claude-{major}.{minor}-{family}-{suffix}
	// e.g. claude-4.5-opus-high
	patInverted = regexp.MustCompile(
		`^claude-(\d+)\.(\d+)-(haiku|sonnet|opus)-.+$`,
	)
)

// NormalizeModelName converts various client model name formats into a canonical form.
//
// Transformations:
//   - claude-sonnet-4-5            → claude-sonnet-4.5
//   - claude-sonnet-4-5-20250929   → claude-sonnet-4.5
//   - claude-sonnet-4-5-latest     → claude-sonnet-4.5
//   - claude-sonnet-4-20250514     → claude-sonnet-4
//   - claude-3-7-sonnet            → claude-3.7-sonnet
//   - claude-3-5-sonnet-20241022   → claude-3.5-sonnet
//   - claude-sonnet-4.5-20250514   → claude-sonnet-4.5
//   - claude-4.5-opus-high         → claude-opus-4.5
func NormalizeModelName(name string) string {
	if name == "" {
		return name
	}
	lower := strings.ToLower(name)

	// Pattern 1: Standard — claude-{family}-{major}-{minor}(-{suffix})?
	if m := patStandard.FindStringSubmatch(lower); m != nil {
		return m[1] + "." + m[2] // e.g. claude-sonnet-4.5
	}

	// Pattern 2: No minor — claude-{family}-{major}(-{date})?
	if m := patNoMinor.FindStringSubmatch(lower); m != nil {
		return m[1] // e.g. claude-sonnet-4
	}

	// Pattern 3: Legacy — claude-{major}-{minor}-{family}(-{suffix})?
	if m := patLegacy.FindStringSubmatch(lower); m != nil {
		return m[1] + "-" + m[2] + "." + m[3] + "-" + m[4] // e.g. claude-3.7-sonnet
	}

	// Pattern 4: Dot with date — claude-{family}-{major}.{minor}-{date}
	if m := patDotWithDate.FindStringSubmatch(lower); m != nil {
		return m[1] // e.g. claude-sonnet-4.5
	}

	// Pattern 5: Inverted with suffix — claude-{major}.{minor}-{family}-{suffix}
	if m := patInverted.FindStringSubmatch(lower); m != nil {
		return "claude-" + m[3] + "-" + m[1] + "." + m[2] // e.g. claude-opus-4.5
	}

	// No transformation needed
	return lower
}
