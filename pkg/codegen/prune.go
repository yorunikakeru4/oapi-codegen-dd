package codegen

import (
	"fmt"
	"slices"

	"github.com/pb33f/libopenapi"
	v3high "github.com/pb33f/libopenapi/datamodel/high/v3"
)

func pruneSchema(doc libopenapi.Document) (libopenapi.Document, error) {
	for {
		model, errs := doc.BuildV3Model()
		if len(errs) > 0 {
			return nil, errs[0]
		}

		refs := findComponentRefs(&model.Model)
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

	for key := range model.Components.Schemas.KeysFromOldest() {
		ref := fmt.Sprintf("#/components/schemas/%s", key)
		if !slices.Contains(refs, ref) {
			countRemoved++
			model.Components.Schemas.Delete(key)
		}
	}

	for key := range model.Components.Parameters.KeysFromOldest() {
		ref := fmt.Sprintf("#/components/parameters/%s", key)
		if !slices.Contains(refs, ref) {
			countRemoved++
			model.Components.Parameters.Delete(key)
		}
	}

	for key := range model.Components.RequestBodies.KeysFromOldest() {
		ref := fmt.Sprintf("#/components/requestBodies/%s", key)
		if !slices.Contains(refs, ref) {
			countRemoved++
			model.Components.RequestBodies.Delete(key)
		}
	}

	for key := range model.Components.Responses.KeysFromOldest() {
		ref := fmt.Sprintf("#/components/responses/%s", key)
		if !slices.Contains(refs, ref) {
			countRemoved++
			model.Components.Responses.Delete(key)
		}
	}

	for key := range model.Components.Headers.KeysFromOldest() {
		ref := fmt.Sprintf("#/components/headers/%s", key)
		if !slices.Contains(refs, ref) {
			countRemoved++
			model.Components.Headers.Delete(key)
		}
	}

	for key := range model.Components.Examples.KeysFromOldest() {
		ref := fmt.Sprintf("#/components/examples/%s", key)
		if !slices.Contains(refs, ref) {
			countRemoved++
			model.Components.Examples.Delete(key)
		}
	}

	for key := range model.Components.Links.KeysFromOldest() {
		ref := fmt.Sprintf("#/components/links/%s", key)
		if !slices.Contains(refs, ref) {
			countRemoved++
			model.Components.Links.Delete(key)
		}
	}

	for key := range model.Components.Callbacks.KeysFromOldest() {
		ref := fmt.Sprintf("#/components/callbacks/%s", key)
		if !slices.Contains(refs, ref) {
			countRemoved++
			model.Components.Callbacks.Delete(key)
		}
	}

	return countRemoved
}

func findComponentRefs(model *v3high.Document) []string {
	var refs []string

	index := model.Index
	for _, ref := range index.GetRawReferencesSequenced() {
		refs = append(refs, ref.Definition)
	}
	return refs
}
