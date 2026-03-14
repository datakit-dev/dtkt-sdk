//go:build ignore

package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/lib/protoschema"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	"github.com/fivetran/go-fivetran/connections"
	"github.com/fivetran/go-fivetran/destinations"
	"github.com/jhump/protoreflect/v2/protoprint"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

var types = map[string]typeConfig{
	// "ConnectionConfig": {
	// 	value: new(connections.ConnectionConfig).Request(),
	// },
	"ConnectionDetails": {
		value: new(connections.DetailsResponseDataCommon),
	},
	"DestinationConfig": {
		value: new(destinations.DestinationConfig).Request(),
	},
	"DestinationDetails": {
		value: new(destinations.DestinationDetailsBase),
	},
}

type typeConfig struct {
	parseOpts protoschema.ParserOptions
	value     any
}

func main() {
	if len(os.Args) < 2 || os.Args[1] == "" {
		panic("proto dir required")
	}

	var (
		protoDir = os.Args[1]
		printer  = protoprint.Printer{
			SortElements: true,
		}
		files []protoreflect.FileDescriptor
	)

	for name, config := range types {
		schema, err := common.NewJSONSchema(config.value)
		if err != nil {
			log.Fatal(err)
		}

		config.parseOpts.PackageName = "dtkt.lib.fivetran.v1"
		config.parseOpts.MessageName = name
		config.parseOpts.UseJSONNames = true
		// EnableValidation: true,
		// UseJSONNames:     true,

		fileProto, err := protoschema.NewParser(config.parseOpts).Parse(schema.Bytes())
		if err != nil {
			log.Fatal(err)
		}

		fileProto.Name = new(util.ToSnakeCase(name) + ".proto")
		if fileProto.Options == nil {
			fileProto.Options = new(descriptorpb.FileOptions)
		}

		fileProto.Options.GoPackage = new("github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/lib/fivetran/v1;fivetranv1")

		file, err := protodesc.NewFile(fileProto, protoregistry.GlobalFiles)
		if err != nil {
			log.Fatal(err)
		}

		files = append(files, file)
	}

	err := printer.PrintProtosToFileSystem(files, filepath.Join(protoDir, "lib/fivetran/v1"))
	if err != nil {
		log.Fatal(err)
	}
}
