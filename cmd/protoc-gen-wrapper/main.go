// protoc-gen-wrapper is a protoc plugin that generates a wrapper for each service that exists in the proto files.
//
// This outputs to the {out}/pkg/trait/{trait name} package, it is intended to target the root of this project.
package main

import (
	_ "embed"
	"fmt"
	"path/filepath"
	"strings"
	"text/template"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/pluginpb"
)

//go:embed wrapper.go.gotxt
var serviceTmplStr string
var serviceTmpl *template.Template

func main() {
	var err error
	serviceTmpl, err = template.New("service").Parse(serviceTmplStr)
	if err != nil {
		panic(err)
	}

	opts := protogen.Options{}
	opts.Run(func(plugin *protogen.Plugin) error {
		for _, file := range plugin.Files {
			if !file.Generate {
				continue
			}
			if err := generateFile(plugin, file); err != nil {
				return err
			}
		}
		plugin.SupportedFeatures = uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL)
		return nil
	})
}

func generateFile(plugin *protogen.Plugin, file *protogen.File) error {
	pkg := filepath.Base(filepath.Dir(file.GeneratedFilenamePrefix))
	// looked like something/something/traits/something, use something more sensible
	if pkg == "traits" {
		pkg = filepath.Base(file.GeneratedFilenamePrefix)
	}
	pkg = strings.ReplaceAll(pkg, "_", "")

	for _, service := range file.Services {
		name := trimPrefixIgnoreCase(service.GoName, pkg)
		filename := fmt.Sprintf("pkg/trait/%s/%s_wrap.pb.go", pkg, strings.ToLower(name))

		g := plugin.NewGeneratedFile(filename, protogen.GoImportPath(fmt.Sprintf("github.com/smart-core-os/sc-golang/pkg/trait/%s", pkg)))
		model := newServiceModel(g, service, file, pkg, name)
		err := serviceTmpl.Execute(g, model)
		if err != nil {
			return err
		}
	}
	return nil
}

func newServiceModel(g *protogen.GeneratedFile, service *protogen.Service, file *protogen.File, pkg, underlying string) ServiceModel {
	model := ServiceModel{
		Service: service,
		QualifiedServerName: g.QualifiedGoIdent(protogen.GoIdent{
			GoName:       service.GoName + "Server",
			GoImportPath: file.GoImportPath,
		}),
		QualifiedClientName: g.QualifiedGoIdent(protogen.GoIdent{
			GoName:       service.GoName + "Client",
			GoImportPath: file.GoImportPath,
		}),
		QualifiedUnimplementedServerName: g.QualifiedGoIdent(protogen.GoIdent{
			GoName:       "Unimplemented" + service.GoName + "Server",
			GoImportPath: file.GoImportPath,
		}),
		QualifiedRegisterService: g.QualifiedGoIdent(protogen.GoIdent{
			GoName:       "Register" + service.GoName + "Server",
			GoImportPath: file.GoImportPath,
		}),
		PackageName: pkg,
		Wrapper:     ident(underlying + "Wrapper"),
		Underlying:  ident(underlying),
	}

	for _, method := range service.Methods {
		// make sure the correct imports are available
		g.QualifiedGoIdent(protogen.GoIdent{GoImportPath: "context"})
		g.QualifiedGoIdent(protogen.GoIdent{GoImportPath: "google.golang.org/grpc"})
		if method.Desc.IsStreamingServer() {
			g.QualifiedGoIdent(protogen.GoIdent{GoImportPath: "github.com/smart-core-os/sc-golang/pkg/wrap"})
		}

		model.Methods = append(model.Methods, ServiceMethod{
			Method:        method,
			GoNamePrivate: strings.ToLower(method.GoName[:1]) + method.GoName[1:],
			Streaming:     method.Desc.IsStreamingServer(),
			QualifiedServerStream: g.QualifiedGoIdent(protogen.GoIdent{
				GoName:       fmt.Sprintf("%s_%sServer", service.GoName, method.GoName),
				GoImportPath: file.GoImportPath,
			}),
			QualifiedClientStream: g.QualifiedGoIdent(protogen.GoIdent{
				GoName:       fmt.Sprintf("%s_%sClient", service.GoName, method.GoName),
				GoImportPath: file.GoImportPath,
			}),
			GoInput:           method.Input.GoIdent.GoName,
			QualifiedGoInput:  g.QualifiedGoIdent(method.Input.GoIdent),
			GoOutput:          method.Output.GoIdent.GoName,
			QualifiedGoOutput: g.QualifiedGoIdent(method.Output.GoIdent),
		})
	}
	return model
}

func trimPrefixIgnoreCase(s, prefix string) string {
	ls, lp := strings.ToLower(s), strings.ToLower(prefix)
	ls = strings.TrimPrefix(ls, lp)
	return s[len(s)-len(ls):]
}

type ServiceModel struct {
	*protogen.Service

	PackageName string
	Wrapper     Ident
	Underlying  Ident

	QualifiedServerName              string
	QualifiedClientName              string
	QualifiedUnimplementedServerName string
	QualifiedRegisterService         string

	Methods []ServiceMethod
}

type ServiceMethod struct {
	*protogen.Method

	GoNamePrivate string

	Streaming             bool
	QualifiedServerStream string
	QualifiedClientStream string

	GoInput           string
	QualifiedGoInput  string
	GoOutput          string
	QualifiedGoOutput string
}

type Ident struct {
	Exported string
	Private  string
}

func ident(n string) Ident {
	first, rest := n[:1], n[1:]
	return Ident{
		Exported: strings.ToUpper(first) + rest,
		Private:  strings.ToLower(first) + rest,
	}
}
