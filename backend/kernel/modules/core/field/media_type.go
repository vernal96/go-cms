package field

import "fmt"

type mediaType struct{}

func (mediaType) Code() TypeCode { return TypeMedia }
func (mediaType) Compile(ctx CompileContext, options any) (ValueType, error) {
	config, err := DecodeOptions[MediaOptions](options)
	if err != nil {
		return nil, err
	}
	return withList(mediaValue{}, config.Multiple, config.MinItems, config.MaxItems, false)
}

type mediaValue struct{}

func (mediaValue) StorageKind() StorageKind { return StorageReference }
func (mediaValue) Multiple() bool           { return false }
func (mediaValue) ReferenceTarget() string  { return ReferenceMedia }
func (mediaValue) Normalize(value any) (any, error) {
	id, ok := normalizeInteger(value)
	if !ok || id <= 0 {
		return nil, fmt.Errorf("expected positive media id, got %T", value)
	}
	return id, nil
}
func (mediaValue) Empty(any) bool     { return false }
func (mediaValue) Validate(any) error { return nil }
func (mediaValue) Rules() []string    { return nil }
func (mediaValue) Example() any       { return int64(1) }
