package model

import "encoding/json"

// JSONSchemas generates JSON Schema (draft 2020-12) documents describing this
// model's agent and context files, so editors can autocomplete and validate them.
// Reference them with "$schema": "./.jsonschema/agent.json" (or context.json).
func JSONSchemas(m *Model) (map[string][]byte, error) {
	scalar := func(k Kind) map[string]any {
		switch k {
		case KindInt:
			return map[string]any{"type": "integer"}
		case KindFloat:
			return map[string]any{"type": "number"}
		case KindBool:
			return map[string]any{"type": "boolean"}
		}
		return map[string]any{"type": "string"}
	}
	fieldSchema := func(f *Field) map[string]any {
		s := scalar(f.Kind)
		if f.Description != "" {
			s["description"] = f.Description
		}
		if f.Min != nil {
			s["minimum"] = *f.Min
		}
		if f.Max != nil {
			s["maximum"] = *f.Max
		}
		if len(f.Enum) > 0 {
			s["enum"] = f.Enum
		}
		return s
	}
	dist := map[string]any{
		"type":     "object",
		"required": []string{"dist"},
		"properties": map[string]any{
			"dist":        map[string]any{"enum": []string{"uniform", "normal", "bernoulli", "choice"}},
			"min":         map[string]any{"type": "number"},
			"max":         map[string]any{"type": "number"},
			"mean":        map[string]any{"type": "number"},
			"std":         map[string]any{"type": "number", "minimum": 0},
			"p":           map[string]any{"type": "number", "minimum": 0, "maximum": 1},
			"values":      map[string]any{"type": "array"},
			"weights":     map[string]any{"type": "array", "items": map[string]any{"type": "number"}},
			"description": map[string]any{"type": "string"},
		},
		"additionalProperties": false,
	}

	var variants []any
	for _, t := range m.Types {
		attrs := map[string]any{}
		for _, f := range t.Attrs {
			attrs[f.Name] = map[string]any{"anyOf": []any{fieldSchema(f), map[string]any{"$ref": "#/$defs/distribution"}}}
		}
		props := map[string]any{
			"$schema":     map[string]any{"type": "string"},
			"description": map[string]any{"type": "string"},
			"type":        map[string]any{"const": t.Name},
			"name":        map[string]any{"type": "string"},
			"count":       map[string]any{"type": "integer", "minimum": 1},
			"attributes":  map[string]any{"type": "object", "properties": attrs, "additionalProperties": false},
		}
		if len(m.Relations) > 0 {
			props["links"] = map[string]any{"type": "array", "items": map[string]any{
				"type": "object", "required": []string{"relation", "to"},
				"properties": map[string]any{
					"relation": map[string]any{"enum": relationNames(m)},
					"to":       map[string]any{"type": "string"},
				},
				"additionalProperties": false,
			}}
		}
		if len(m.GroupKinds) > 0 {
			groups := map[string]any{}
			for _, g := range m.GroupKinds {
				groups[g.Name] = map[string]any{"anyOf": []any{map[string]any{"type": "string"}, map[string]any{"$ref": "#/$defs/distribution"}}}
			}
			props["groups"] = map[string]any{"type": "object", "properties": groups, "additionalProperties": false}
		}
		if t.Spatial {
			coord := map[string]any{"anyOf": []any{map[string]any{"type": "number"}, map[string]any{"$ref": "#/$defs/distribution"}}}
			props["position"] = map[string]any{"type": "object", "properties": map[string]any{"x": coord, "y": coord}, "additionalProperties": false}
		}
		v := map[string]any{"type": "object", "required": []string{"type"}, "properties": props, "additionalProperties": false}
		if t.Description != "" {
			v["description"] = t.Description
		}
		variants = append(variants, v)
	}
	agent := map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"title":   m.Name + " agent file",
		"oneOf":   variants,
		"$defs":   map[string]any{"distribution": dist},
	}

	ctxProps := map[string]any{
		"$schema":     map[string]any{"type": "string"},
		"description": map[string]any{"type": "string"},
	}
	for _, f := range m.Context {
		if f.IsMap {
			s := map[string]any{"type": "object", "additionalProperties": fieldSchema(f)}
			if f.Description != "" {
				s["description"] = f.Description
			}
			ctxProps[f.Name] = s
			continue
		}
		ctxProps[f.Name] = fieldSchema(f)
	}
	context := map[string]any{
		"$schema":              "https://json-schema.org/draft/2020-12/schema",
		"title":                m.Name + " context file",
		"type":                 "object",
		"properties":           ctxProps,
		"additionalProperties": false,
	}

	out := map[string][]byte{}
	for name, doc := range map[string]any{"agent.json": agent, "context.json": context} {
		data, err := json.MarshalIndent(doc, "", "  ")
		if err != nil {
			return nil, err
		}
		out[name] = append(data, '\n')
	}
	return out, nil
}

func relationNames(m *Model) []string {
	names := make([]string, 0, len(m.Relations))
	for _, r := range m.Relations {
		names = append(names, r.Name)
	}
	return names
}
