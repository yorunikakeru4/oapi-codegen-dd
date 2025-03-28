package codegen

import "errors"

var (
	ErrOperationNameEmpty                        = errors.New("operation name cannot be an empty string")
	ErrRequestPathEmpty                          = errors.New("request path cannot be an empty string")
	ErrMergingSchemasWithDifferentUniqueItems    = errors.New("merging two schemas with different UniqueItems")
	ErrMergingSchemasWithDifferentExclusiveMin   = errors.New("merging two schemas with different ExclusiveMin")
	ErrMergingSchemasWithDifferentExclusiveMax   = errors.New("merging two schemas with different ExclusiveMax")
	ErrMergingSchemasWithDifferentNullable       = errors.New("merging two schemas with different Nullable")
	ErrMergingSchemasWithDifferentReadOnly       = errors.New("merging two schemas with different ReadOnly")
	ErrMergingSchemasWithDifferentWriteOnly      = errors.New("merging two schemas with different WriteOnly")
	ErrTransitiveMergingAllOfSchema1             = errors.New("error transitive merging AllOf on schema 1")
	ErrTransitiveMergingAllOfSchema2             = errors.New("error transitive merging AllOf on schema 2")
	ErrMergingSchemasWithDifferentDefaults       = errors.New("merging two sets of defaults is undefined")
	ErrMergingSchemasWithDifferentFormats        = errors.New("can not merge incompatible formats")
	ErrMergingSchemasWithDifferentDiscriminators = errors.New("merging two schemas with discriminators is not supported")
	ErrMergingSchemasWithAdditionalProperties    = errors.New("merging two schemas with additional properties, this is unhandled")
	ErrAmbiguousDiscriminatorMapping             = errors.New("ambiguous discriminator.mapping: please replace inlined object with $ref")
	ErrDiscriminatorNotAllMapped                 = errors.New("discriminator: not all schemas were mapped")
	ErrEmptySchema                               = errors.New("empty schema")
)
