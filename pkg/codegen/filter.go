package codegen

import "github.com/getkin/kin-openapi/openapi3"

func filterDocument(doc *openapi3.T, cfg *Configuration) (*openapi3.T, error) {
	filterOperationsByTag(doc, cfg)
	filterOperationsByOperationID(doc, cfg)

	if !cfg.SkipPrune {
		pruneUnusedComponents(doc)
	}

	return doc, nil
}

func sliceToMap(items []string) map[string]bool {
	m := make(map[string]bool, len(items))
	for _, item := range items {
		m[item] = true
	}
	return m
}

func filterOperationsByTag(swagger *openapi3.T, opts *Configuration) {
	if len(opts.Filter.Exclude.Tags) > 0 {
		operationsWithTags(swagger.Paths, sliceToMap(opts.Filter.Exclude.Tags), true)
	}
	if len(opts.Filter.Include.Tags) > 0 {
		operationsWithTags(swagger.Paths, sliceToMap(opts.Filter.Include.Tags), false)
	}
}

func operationsWithTags(paths *openapi3.Paths, tags map[string]bool, exclude bool) {
	if paths == nil {
		return
	}

	for _, pathItem := range paths.Map() {
		ops := pathItem.Operations()
		names := make([]string, 0, len(ops))
		for name, op := range ops {
			if operationHasTag(op, tags) == exclude {
				names = append(names, name)
			}
		}
		for _, name := range names {
			pathItem.SetOperation(name, nil)
		}
	}
}

// operationHasTag returns true if the operation is tagged with any of tags
func operationHasTag(op *openapi3.Operation, tags map[string]bool) bool {
	if op == nil {
		return false
	}
	for _, hasTag := range op.Tags {
		if tags[hasTag] {
			return true
		}
	}
	return false
}

func filterOperationsByOperationID(swagger *openapi3.T, opts *Configuration) {
	if len(opts.Filter.Exclude.OperationIDs) > 0 {
		operationsWithOperationIDs(swagger.Paths, sliceToMap(opts.Filter.Exclude.OperationIDs), true)
	}
	if len(opts.Filter.Include.OperationIDs) > 0 {
		operationsWithOperationIDs(swagger.Paths, sliceToMap(opts.Filter.Include.OperationIDs), false)
	}
}

func operationsWithOperationIDs(paths *openapi3.Paths, operationIDs map[string]bool, exclude bool) {
	if paths == nil {
		return
	}

	for _, pathItem := range paths.Map() {
		ops := pathItem.Operations()
		names := make([]string, 0, len(ops))
		for name, op := range ops {
			if operationIDs[op.OperationID] == exclude {
				names = append(names, name)
			}
		}
		for _, name := range names {
			pathItem.SetOperation(name, nil)
		}
	}
}
