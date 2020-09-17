package masks

import (
	"github.com/iancoleman/strcase"
	fieldMaskUtils "github.com/mennanov/fieldmask-utils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

// ValidFields will verify that all fields are applicable to the given message type and return an error if not.
func ValidFields(mask *fieldmaskpb.FieldMask, message proto.Message) (*fieldmaskpb.FieldMask, error) {
	if !mask.IsValid(message) {
		return nil, status.Errorf(codes.InvalidArgument, "unknown mask paths")
	}
	return mask, nil
}

// ValidMask will convert the given FieldMask into a Mask for use with your types. It will verify that all fields are
// applicable to the given message type and return an error if not.
func ValidMask(mask *fieldmaskpb.FieldMask, message proto.Message) (fieldMaskUtils.Mask, error) {
	if mask == nil || len(mask.GetPaths()) == 0 {
		return nil, nil
	}

	if _, err := ValidFields(mask, message); err != nil {
		return nil, err
	}

	return fieldMaskUtils.MaskFromProtoFieldMask(mask, strcase.ToCamel)
}

// ValidWritableFields will check the validity of the given mask against writable fields. A nil writableFields means
// all fields are writable, a writableFields with no paths means no fields are writable. If any fields are in mask
// that aren't covered by writableFields then an error is returned. Any fields in the union of writableFields and mask
// that aren't valid fields in message will return an error.
func ValidWritableFields(writableFields, mask *fieldmaskpb.FieldMask, message proto.Message) (*fieldmaskpb.FieldMask, error) {
	if writableFields == nil {
		return ValidFields(mask, message)
	}

	fields := writableFields
	if mask != nil {
		if !mask.IsValid(message) {
			return nil, status.Errorf(codes.InvalidArgument, "unknown mask paths")
		}
		fields = fieldmaskpb.Intersect(fields, mask)
		if len(mask.GetPaths()) > len(fields.GetPaths()) {
			return nil, status.Errorf(codes.InvalidArgument, "non-writable mask paths")
		}
	}

	return fields, nil
}

// ValidWritableMask will convert the given FieldMask into a Mask for use with your types. A nil writableFields means
// all fields are writable, a writableFields with no paths means no fields are writable. If any fields are in mask
// that aren't covered by writableFields then an error is returned. Any fields in the union of writableFields and mask
// that aren't valid fields in message will return an error.
func ValidWritableMask(writableFields, mask *fieldmaskpb.FieldMask, message proto.Message) (fieldMaskUtils.Mask, error) {
	if writableFields == nil {
		return ValidMask(mask, message)
	}

	fields, err := ValidWritableFields(writableFields, mask, message)
	if err != nil {
		return nil, err
	}

	if len(fields.GetPaths()) == 0 {
		return nil, nil
	}

	return fieldMaskUtils.MaskFromProtoFieldMask(fields, strcase.ToCamel)
}
