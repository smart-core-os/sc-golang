package masks

import (
	"github.com/mennanov/fmutils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type FieldUpdater struct {
	writableFields      *fieldmaskpb.FieldMask
	updateMask          *fieldmaskpb.FieldMask
	updateMaskFieldName string

	intersectionMask *fieldmaskpb.FieldMask
}

func NewFieldUpdater(opts ...FieldUpdaterOption) *FieldUpdater {
	f := &FieldUpdater{}
	for _, opt := range DefaultFieldUpdateOptions {
		opt(f)
	}
	for _, opt := range opts {
		opt(f)
	}
	return f
}

func (f *FieldUpdater) Validate(m proto.Message) error {
	if f.updateMask != nil {
		// is the update mask valid?
		if !f.updateMask.IsValid(m) {
			return status.Errorf(codes.InvalidArgument, "%v mentions unknown fields", f.updateMaskFieldName)
		}

		// are fields mentioned in the update mask actually writable?
		if f.writableFields != nil {
			common := f.fullMask()
			if len(common.Paths) != len(f.updateMask.Paths) {
				return status.Errorf(codes.InvalidArgument, "%v mentions read-only fields", f.updateMaskFieldName)
			}
		}
	}

	return nil
}

// Merge copies the values in src into dst based on the configured field masks.
func (f *FieldUpdater) Merge(dst, src proto.Message) {
	mask := f.fullMask()
	if mask == nil {
		// no mask, make dst look like src exactly
		proto.Reset(dst)
	} else if len(mask.GetPaths()) == 0 {
		// non-nil mask with no paths => no changes
		return
	}

	nestedMask := fmutils.NestedMaskFromPaths(mask.GetPaths())
	nestedMask.Filter(src)
	proto.Merge(dst, src)

	// if a field mentioned by the mask is nil, we should clear it
	pruneEmpty(dst, src, nestedMask)

	return
}

func pruneEmpty(dst, src proto.Message, mask fmutils.NestedMask) {
	dstPr := dst.ProtoReflect()
	srcPr := src.ProtoReflect()
	dstPr.Range(func(d protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		fieldMask, ok := mask[string(d.Name())]
		if !ok {
			return true
		}
		if !srcPr.Has(d) {
			dstPr.Clear(d)
			return true
		}
		if d.Kind() == protoreflect.MessageKind {
			pruneEmpty(dstPr.Get(d).Message().Interface(), srcPr.Get(d).Message().Interface(), fieldMask)
		}
		return true
	})
}

func (f *FieldUpdater) fullMask() *fieldmaskpb.FieldMask {
	if f.intersectionMask == nil {
		var nonNilMasks []*fieldmaskpb.FieldMask
		if f.writableFields != nil {
			nonNilMasks = append(nonNilMasks, f.writableFields)
		}
		if f.updateMask != nil {
			nonNilMasks = append(nonNilMasks, f.updateMask)
		}

		switch len(nonNilMasks) {
		case 0:
			return nil
		case 1:
			f.intersectionMask = nonNilMasks[0]
		case 2:
			f.intersectionMask = fieldmaskpb.Intersect(nonNilMasks[0], nonNilMasks[1])
		default:
			f.intersectionMask = fieldmaskpb.Intersect(nonNilMasks[0], nonNilMasks[1], nonNilMasks[2:]...)
		}
	}
	return f.intersectionMask
}

type FieldUpdaterOption func(*FieldUpdater)

var DefaultFieldUpdateOptions = []FieldUpdaterOption{
	WithUpdateMaskFieldName("update_mask"),
}

// emptyFieldUpdaterOption is used when existing options decide they don't have anything to do
var emptyFieldUpdaterOption FieldUpdaterOption = func(_ *FieldUpdater) {
}

func WithWritableFields(writableFields *fieldmaskpb.FieldMask) FieldUpdaterOption {
	if writableFields == nil {
		return emptyFieldUpdaterOption
	}
	return func(updater *FieldUpdater) {
		updater.writableFields = writableFields
	}
}

func WithUpdateMask(updateMask *fieldmaskpb.FieldMask) FieldUpdaterOption {
	if updateMask == nil {
		return emptyFieldUpdaterOption
	}
	return func(updater *FieldUpdater) {
		updater.updateMask = updateMask
	}
}

func WithUpdateMaskFieldName(name string) FieldUpdaterOption {
	return func(updater *FieldUpdater) {
		updater.updateMaskFieldName = name
	}
}
