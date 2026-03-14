package entadapter

import (
	"embed"

	"entgo.io/ent/entc"
	"entgo.io/ent/entc/gen"
)

//go:embed templates/*.tmpl
var entTemplatesFS embed.FS

type (
	Extension struct {
		entc.DefaultExtension
	}
	Annotation struct {
		ID        string
		ProtoName string
		IsEnum    bool
		IsList    bool
		IsMap     bool
		IsMessage bool
	}
)

func (Annotation) Name() string {
	return "Protostore"
}

func (*Extension) Annotations() []entc.Annotation {
	return []entc.Annotation{
		Annotation{},
	}
}

func (*Extension) Templates() []*gen.Template {
	return []*gen.Template{
		gen.MustParse(gen.NewTemplate("protostore").ParseFS(entTemplatesFS, "templates/*.tmpl")),
	}
}
