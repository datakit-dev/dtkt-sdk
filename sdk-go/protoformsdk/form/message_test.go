package form_test

import (
	"testing"

	sharedv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/shared/v1beta1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/protoformsdk/form"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestMessage(t *testing.T) {
	pkg := &sharedv1beta1.Package{
		Identity: &sharedv1beta1.Package_Identity{
			Name:    "FooBar",
			Version: "0.1.0",
		},
		Description: "foobar",
		Build:       &sharedv1beta1.Package_BuildConfig{},
	}

	msg, _ := form.NewMessage(pkg.ProtoReflect())
	if msg.String() != msg.StringOf(pkg.ProtoReflect()) {
		t.Fatalf("expected to be equal: %s != %s", msg.String(), msg.StringOf(pkg.ProtoReflect()))
	}

	if !proto.Equal(pkg, msg.Get().Interface()) {
		t.Fatalf("expected to be equal: %s != %s", pkg, msg.Get().Interface())
	}

	out, err := msg.Parse(msg.String())
	if err != nil {
		t.Fatalf("failed to parse message: %s: %s", msg.String(), err)
	} else if !proto.Equal(out.Interface(), msg.Get().Interface()) {
		t.Fatalf("expected to be equal: %s != %s", out.Interface(), msg.Get().Interface())
	}

	pkg2 := proto.Clone(pkg)
	msg.Set(pkg2.ProtoReflect())
	if !proto.Equal(pkg, msg.Get().Interface()) {
		t.Fatalf("expected to be equal: %s != %s", pkg, msg.Get().Interface())
	}

	pkg.Icon = "https://foo.bar/"
	pkg.Type = sharedv1beta1.PackageType_PACKAGE_TYPE_GO

	// err = msg.Validate(pkg.ProtoReflect())
	// if err != nil {
	// 	t.Fatal(err)
	// }

	for _, field := range msg.FieldGroup().GetFields() {
		switch field.Type.Descriptor().Name() {
		case "type":
			if scalar, ok := field.IsScalar(); ok {
				if selec, ok := scalar.Element().IsSelect(); ok {
					selec.GetOptions().Range(func(key string, val any) bool {
						if err := scalar.SetAny(key); err != nil {
							t.Fatal(err)
						}
						return true
					})
				} else {
					t.Fatalf("expected type field to be select element, got: %T", scalar.Element().Type)
				}

				if value, ok := scalar.ScalarType.(*form.Scalar[protoreflect.EnumNumber]); ok {
					t.Logf("type: %s", value)
				} else {
					t.Fatalf("expected type field to be enum")
				}
			} else {
				t.Fatalf("expected type scalar field")
			}
		case "platforms":
			if field == nil {
				t.Fatalf("expected platforms field, got nil")
			}
			if list, ok := field.IsList(); ok {
				if value, ok := list.ListType.(*form.List[*form.Message]); ok {
					m1, _ := form.NewMessage((&sharedv1beta1.Platform{
						Os:   sharedv1beta1.OS_OS_LINUX,
						Arch: sharedv1beta1.Arch_ARCH_X86,
					}).ProtoReflect())
					m2, _ := form.NewMessage((&sharedv1beta1.Platform{
						Os:   sharedv1beta1.OS_OS_WINDOWS,
						Arch: sharedv1beta1.Arch_ARCH_X86,
					}).ProtoReflect())
					value.Set([]*form.Message{m1, m2})
				} else {
					t.Fatalf("expected platforms field to be message list")
				}

				t.Logf("platforms: %s", list)

				if list, ok := list.IsMessage(); ok {
					msgs, err := list.Parse(list.String())
					if err != nil {
						t.Fatal(err)
					} else if len(list.GetItems()) != len(msgs) {
						t.Fatalf("expected platforms len: %d, got: %d", len(list.GetItems()), len(msgs))
					} else {
						for _, msg := range msgs {
							t.Log(msg.Get().Interface())
						}
					}
				}
			} else {
				t.Fatalf("expected platforms list field")
			}
		case "services":
			if field == nil {
				t.Fatalf("expected services field, got nil")
			}
			if list, ok := field.IsList(); ok {
				if value, ok := list.ListType.(*form.List[string]); ok {
					value.Set([]string{"bar"})

					t.Logf("services: %s", value)
				} else {
					t.Fatalf("expected services field to be string list")
				}
			} else {
				t.Fatalf("expected services list field")
			}
		case "description":
			if scalar, ok := field.IsScalar(); ok {
				if value, ok := scalar.ScalarType.(*form.Scalar[string]); ok {
					t.Logf("description: %s", value)
					value.Set("bar baz!")

					if value.Get() != pkg.GetDescription() {
						t.Fatalf("expected to be equal: %s != %s", value.Get(), pkg.GetDescription())
					}

					pkg.Description = "hello world"

					if value.Get() != pkg.GetDescription() {
						t.Fatalf("expected to be equal: %s != %s", value.Get(), pkg.GetDescription())
					}

					if value.Get() != value.GetAny() {
						t.Fatalf("expected to be equal: %s != %s", value.Get(), value.GetAny())
					}

					if err := value.SetAny(1234); err != nil {
						t.Fatal(err)
					}
				} else {
					t.Fatalf("expected description scalar field to be string")
				}
			} else {
				t.Fatalf("expected description scalar field")
			}
		case "identity":
			if ident, ok := field.IsMessage(); ok {
				t.Logf("identity: %s", ident)

				pkg.Identity.Name = "BarBaz"

				t.Logf("identity: %s", ident)

				if !proto.Equal(ident.Get().Interface(), pkg.Identity) {
					t.Fatalf("expected to be equal: %s != %s", ident.Get().Interface(), pkg.Identity)
				}

				ident.Set((&sharedv1beta1.Package_Identity{
					Name:    "FooBar",
					Version: "0.2.0",
				}).ProtoReflect())

				t.Logf("identity: %s", ident)

				if !proto.Equal(ident.Get().Interface(), pkg.Identity) {
					t.Fatalf("expected to be equal: %s != %s", ident.Get().Interface(), pkg.Identity)
				}

				pkg.ProtoReflect().Clear(field.Type.Descriptor())

				if msg.Get().Has(ident.Descriptor()) {
					t.Fatalf("expected identity to be cleared, got: %s", ident.Get().Interface())
				}
			}
		}
	}
}
