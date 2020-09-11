package masks

import (
	"github.com/google/go-cmp/cmp"
	fieldMaskUtils "github.com/mennanov/fieldmask-utils"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"testing"
)

func TestValidWritableMask(t *testing.T) {
	type args struct {
		writableFields *fieldmaskpb.FieldMask
		mask           *fieldmaskpb.FieldMask
		message        proto.Message
	}
	tests := []struct {
		name    string
		args    args
		want    fieldMaskUtils.Mask
		wantErr bool
	}{
		{name: "no mask", args: args{message: ts()}, want: nil},
		{name: "no mask with writable", args: args{writableFields: fm(), message: ts()}, want: nil},
		{name: "all writable", args: args{mask: fm("seconds"), message: ts()}, want: m("Seconds")},
		{name: "none writable", args: args{writableFields: fm(), mask: fm("seconds"), message: ts()}, wantErr: true},
		{name: "empty writable", args: args{writableFields: fm(""), mask: fm("seconds"), message: ts()}, wantErr: true},
		{name: "some writable", args: args{writableFields: fm("seconds"), mask: fm("seconds"), message: ts()}, want: m("Seconds")},
		{name: "not writable", args: args{writableFields: fm("seconds"), mask: fm("nanos"), message: ts()}, wantErr: true},
		{name: "some not writable", args: args{writableFields: fm("seconds"), mask: fm("seconds", "nanos"), message: ts()}, wantErr: true},
		{name: "choose writable", args: args{writableFields: fm("seconds", "nanos"), mask: fm("seconds"), message: ts()}, want: m("Seconds")},
		{name: "choose some writable", args: args{writableFields: fm("seconds", "nanos"), mask: fm("seconds", "other"), message: ts()}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidWritableMask(tt.args.writableFields, tt.args.mask, tt.args.message)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidWritableMask() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(tt.want, got, protocmp.Transform()); diff != "" {
				t.Errorf("ValidWritableMask() (-want,+got)\n%v", diff)
			}
		})
	}
}

func ts() *timestamppb.Timestamp {
	return &timestamppb.Timestamp{}
}

func fm(paths ...string) *fieldmaskpb.FieldMask {
	return &fieldmaskpb.FieldMask{Paths: paths}
}

func m(paths ...string) fieldMaskUtils.Mask {
	mask, _ := fieldMaskUtils.MaskFromPaths(paths, func(s string) string {
		return s
	})
	return mask
}
