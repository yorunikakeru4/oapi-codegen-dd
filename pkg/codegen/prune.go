package codegen

import (
	"fmt"
	"iter"
	"slices"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3high "github.com/pb33f/libopenapi/datamodel/high/v3"
)

func pruneSchema(doc libopenapi.Document) (libopenapi.Document, error) {
	for {
		model, errs := doc.BuildV3Model()
		if len(errs) > 0 {
			return nil, errs[0]
		}

		refs := findOperationRefs(&model.Model)
		countRemoved := removeOrphanedComponents(&model.Model, refs)

		_, doc, _, errs = doc.RenderAndReload()
		if errs != nil {
			return nil, fmt.Errorf("error reloading document: %w", errs[0])
		}

		if countRemoved < 1 {
			return doc, nil
		}
	}
}

func removeOrphanedComponents(model *v3high.Document, refs []string) int {
	if model.Components == nil {
		return 0
	}

	countRemoved := 0

	for _, key := range getComponentKeys(model.Components.Schemas.KeysFromOldest()) {
		ref := fmt.Sprintf("#/components/schemas/%s", key)
		if !slices.Contains(refs, ref) {
			countRemoved++
			model.Components.Schemas.Delete(key)
		}
	}

	for _, key := range getComponentKeys(model.Components.Parameters.KeysFromOldest()) {
		ref := fmt.Sprintf("#/components/parameters/%s", key)
		if !slices.Contains(refs, ref) {
			countRemoved++
			model.Components.Parameters.Delete(key)
		}
	}

	for _, key := range getComponentKeys(model.Components.RequestBodies.KeysFromOldest()) {
		ref := fmt.Sprintf("#/components/requestBodies/%s", key)
		if !slices.Contains(refs, ref) {
			countRemoved++
			model.Components.RequestBodies.Delete(key)
		}
	}

	for _, key := range getComponentKeys(model.Components.Responses.KeysFromOldest()) {
		ref := fmt.Sprintf("#/components/responses/%s", key)
		if !slices.Contains(refs, ref) {
			countRemoved++
			model.Components.Responses.Delete(key)
		}
	}

	for _, key := range getComponentKeys(model.Components.Headers.KeysFromOldest()) {
		ref := fmt.Sprintf("#/components/headers/%s", key)
		if !slices.Contains(refs, ref) {
			countRemoved++
			model.Components.Headers.Delete(key)
		}
	}

	for _, key := range getComponentKeys(model.Components.Examples.KeysFromOldest()) {
		ref := fmt.Sprintf("#/components/examples/%s", key)
		if !slices.Contains(refs, ref) {
			countRemoved++
			model.Components.Examples.Delete(key)
		}
	}

	for _, key := range getComponentKeys(model.Components.Links.KeysFromOldest()) {
		ref := fmt.Sprintf("#/components/links/%s", key)
		if !slices.Contains(refs, ref) {
			countRemoved++
			model.Components.Links.Delete(key)
		}
	}

	for _, key := range getComponentKeys(model.Components.Callbacks.KeysFromOldest()) {
		ref := fmt.Sprintf("#/components/callbacks/%s", key)
		if !slices.Contains(refs, ref) {
			countRemoved++
			model.Components.Callbacks.Delete(key)
		}
	}

	return countRemoved
}

func findOperationRefs(model *v3high.Document) []string {
	refSet := make(map[string]struct{})

	for _, pathItem := range model.Paths.PathItems.FromOldest() {
		for _, op := range pathItem.GetOperations().FromOldest() {
			if op.RequestBody != nil {
				ref := op.RequestBody.GoLow().GetReference()
				if ref != "" {
					refSet[ref] = struct{}{}
				}
				for _, mediaType := range op.RequestBody.Content.FromOldest() {
					medRef := mediaType.Schema.GetReference()
					if medRef != "" {
						refSet[medRef] = struct{}{}
					}
					if mediaType.Schema != nil {
						collectSchemaRefs(mediaType.Schema.Schema(), refSet)
					}
				}
			}

			for _, param := range op.Parameters {
				if param == nil {
					continue
				}
				ref := param.GoLow().GetReference()
				if ref != "" {
					refSet[ref] = struct{}{}
				}
				for _, mediaType := range param.Content.FromOldest() {
					if mediaType.Schema != nil {
						collectSchemaRefs(mediaType.Schema.Schema(), refSet)
					}
				}
			}

			if op.Responses.Default != nil {
				ref := op.Responses.Default.GoLow().GetReference()
				if ref != "" {
					refSet[ref] = struct{}{}
				}
				for _, mediaType := range op.Responses.Default.Content.FromOldest() {
					if mediaType.Schema != nil {
						mRef := mediaType.Schema.GoLow().GetReference()
						if mRef != "" {
							refSet[mRef] = struct{}{}
						}
						collectSchemaRefs(mediaType.Schema.Schema(), refSet)
					}
				}
			}

			for _, resp := range op.Responses.Codes.FromOldest() {
				if resp == nil {
					continue
				}
				ref := resp.GoLow().GetReference()
				if ref != "" {
					refSet[ref] = struct{}{}
				}

				for _, mediaType := range resp.Content.FromOldest() {
					if mediaType.Schema != nil {
						mRef := mediaType.Schema.GoLow().GetReference()
						if mRef != "" {
							refSet[mRef] = struct{}{}
						}
						collectSchemaRefs(mediaType.Schema.Schema(), refSet)
					}
				}
			}
		}
	}

	refs := make([]string, 0, len(refSet))
	for r := range refSet {
		refs = append(refs, r)
	}
	slices.Sort(refs)
	return refs
}

func collectSchemaRefs(schema *base.Schema, refSet map[string]struct{}) {
	visited := make(map[*base.Schema]struct{})
	collectSchemaRefsInternal(schema, refSet, visited)
}

func collectSchemaRefsInternal(schema *base.Schema, refSet map[string]struct{}, visited map[*base.Schema]struct{}) {
	if schema == nil {
		return
	}

	// stop if already visited
	if _, ok := visited[schema]; ok {
		return
	}
	visited[schema] = struct{}{}

	ref := schema.GoLow().GetReference()
	if ref != "" {
		refSet[ref] = struct{}{}
		return
	}

	if schema.Properties != nil {
		for _, prop := range schema.Properties.FromOldest() {
			pRef := prop.GoLow().GetReference()
			if pRef != "" {
				refSet[pRef] = struct{}{}
			}
			collectSchemaRefsInternal(prop.Schema(), refSet, visited)
		}
	}

	items := schema.Items
	if items != nil && items.IsA() && items.A != nil {
		iRef := items.A.GoLow().GetReference()
		if iRef != "" {
			refSet[iRef] = struct{}{}
		}
		collectSchemaRefsInternal(items.A.Schema(), refSet, visited)
	}

	if schema.AdditionalProperties != nil && schema.AdditionalProperties.IsA() && schema.AdditionalProperties.A != nil {
		aRef := schema.AdditionalProperties.A.GoLow().GetReference()
		if aRef != "" {
			refSet[aRef] = struct{}{}
		}
		collectSchemaRefsInternal(schema.AdditionalProperties.A.Schema(), refSet, visited)
	}

	for _, schemaProxies := range [][]*base.SchemaProxy{schema.AllOf, schema.OneOf, schema.AnyOf, {schema.Not}} {
		for _, schemaProxy := range schemaProxies {
			if schemaProxy == nil {
				continue
			}
			sRef := schemaProxy.GoLow().GetReference()
			if sRef != "" {
				refSet[sRef] = struct{}{}
			}
			if schemaProxy.Schema() != nil {
				sRef := schemaProxy.Schema().GoLow().GetReference()
				if sRef != "" {
					refSet[sRef] = struct{}{}
				}
			}
			collectSchemaRefsInternal(schemaProxy.Schema(), refSet, visited)
		}
	}

	extensions := extractExtensions(schema.Extensions)
	for _, extValue := range extensions {
		switch extValue.(type) {
		case []any:
			for _, v := range extValue.([]any) {
				switch m := v.(type) {
				case keyValue[string, string]:
					if m.key == "$ref" {
						refSet[m.value] = struct{}{}
					}
				}
			}
		}
	}
}

func getComponentKeys(component iter.Seq[string]) []string {
	keys := make([]string, 0)
	for k := range component {
		keys = append(keys, k)
	}
	return keys
}
