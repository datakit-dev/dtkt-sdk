package protoschema

import (
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strings"

	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	"github.com/invopop/jsonschema"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	bufValidateName  = validate.File_buf_validate_validate_proto.FullName()
	timestampName    = new(timestamppb.Timestamp).ProtoReflect().Descriptor().FullName()
	durationName     = new(durationpb.Duration).ProtoReflect().Descriptor().FullName()
	wellKnownImports = map[protoreflect.FullName]string{
		bufValidateName: "buf/validate/validate.proto",
		timestampName:   "google/protobuf/timestamp.proto",
		durationName:    "google/protobuf/duration.proto",
	}
)

// ParserOptions configures the JSON Schema to protobuf parser.
type ParserOptions struct {
	// PackageName is the protobuf package name for the generated descriptor (required).
	PackageName string
	// MessageName is an explicit message name override for the root message (optional).
	MessageName string
	// Syntax specifies the protobuf syntax ("proto3" or "proto2"), defaults to "proto3".
	Syntax string
	// UseJSONNames configures the parser to preserve JSON property names as json_name in fields.
	UseJSONNames bool
	// EnableValidation adds protovalidate field options based on JSON Schema constraints.
	EnableValidation bool
	// Resolver is used to resolve references found in provided JSON Schema and to
	// register new enum and message types.
	Resolver ParserResolver
}

type ParserResolver interface {
	protodesc.Resolver
	RegisterEnum(protoreflect.EnumType) error
	RegisterMessage(protoreflect.MessageType) error
}

// Parser converts JSON Schema to protobuf descriptors.
type Parser struct {
	opts         ParserOptions
	messages     map[string]*descriptorpb.DescriptorProto
	enums        map[string]*descriptorpb.EnumDescriptorProto
	messageNames map[string]bool // Track used message names to avoid collisions

	imports []string
}

// NewParser creates a new JSON Schema parser with the given options.
func NewParser(opts ParserOptions) *Parser {
	if opts.Syntax == "" {
		opts.Syntax = "proto3"
	}
	if opts.PackageName == "" {
		opts.PackageName = "generated"
	}
	// if opts.Resolver == nil {
	// dynamicpb.NewTypes()
	// opts.Resolver = &protoregistry.Files{}
	// }
	return &Parser{
		opts:         opts,
		messages:     make(map[string]*descriptorpb.DescriptorProto),
		enums:        make(map[string]*descriptorpb.EnumDescriptorProto),
		messageNames: make(map[string]bool),
	}
}

// ParseMap converts a JSON Schema of map[string]any to a FileDescriptorProto.
func (p *Parser) ParseMap(schemaMap map[string]any) (*descriptorpb.FileDescriptorProto, error) {
	b, err := json.Marshal(schemaMap)
	if err != nil {
		return nil, err
	}

	return p.Parse(b)
}

// Parse converts JSON Schema bytes to a FileDescriptorProto.
func (p *Parser) Parse(schemaJSON []byte) (*descriptorpb.FileDescriptorProto, error) {
	schema, err := p.unmarshalSchema(schemaJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON schema: %w", err)
	}

	// Process $defs first if present
	if schema.Definitions != nil {
		for defName, defSchema := range schema.Definitions {
			if defSchema != nil {
				_, err := p.parseMessage(SanitizeMessageName(defName), defSchema)
				if err != nil {
					return nil, fmt.Errorf("failed to parse $def %q: %w", defName, err)
				}
			}
		}
	}

	// Extract message name from title
	msgName := p.getMessageName(schema)
	msgDesc, err := p.parseMessage(msgName, schema)
	if err != nil {
		return nil, err
	}

	fdp := &descriptorpb.FileDescriptorProto{
		Name:        new(msgName + ".proto"),
		Package:     new(p.opts.PackageName),
		Syntax:      new(p.opts.Syntax),
		MessageType: []*descriptorpb.DescriptorProto{msgDesc},
	}

	// Add any nested/referenced messages (excluding the root message)
	for name, msg := range p.messages {
		if name != msgName {
			fdp.MessageType = append(fdp.MessageType, msg)
		}
	}

	// Add any enums
	for _, enum := range p.enums {
		fdp.EnumType = append(fdp.EnumType, enum)
	}

	for _, dep := range util.SliceSet(p.imports) {
		if dep != "" && !slices.Contains(fdp.Dependency, dep) {
			fdp.Dependency = append(fdp.Dependency, dep)
		}
	}

	return fdp, nil
}

func (p *Parser) ParseMessageType(schemaJSON []byte) (protoreflect.MessageType, error) {
	desc, err := p.ParseMessage(schemaJSON)
	if err != nil {
		return nil, err
	}

	return dynamicpb.NewMessageType(desc), nil
}

func (p *Parser) ParseMessageTypeMap(schemaMap map[string]any) (protoreflect.MessageType, error) {
	desc, err := p.ParseMessageMap(schemaMap)
	if err != nil {
		return nil, err
	}

	return dynamicpb.NewMessageType(desc), nil
}

// ParseMessage converts a JSON Schema to a protoreflect.MessageDescriptor.
// This provides ergonomic access to the message descriptor without needing to
// manually navigate the FileDescriptorProto.
//
// Example:
//
//	parser := protoschema.NewParser(protoschema.ParserOptions{PackageName: "example"})
//	md, err := parser.ParseMessage(schemaJSON)
//	if err != nil {
//		panic(err)
//	}
//	// Use md directly
//	msg := dynamicpb.NewMessage(md)
func (p *Parser) ParseMessage(schemaJSON []byte) (protoreflect.MessageDescriptor, error) {
	fdp, err := p.Parse(schemaJSON)
	if err != nil {
		return nil, err
	}

	// Build the file descriptor
	fd, err := protodesc.NewFile(fdp, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create file descriptor: %w", err)
	}

	// Return the first (main) message descriptor
	if fd.Messages().Len() == 0 {
		return nil, fmt.Errorf("no message descriptor found")
	}

	return fd.Messages().Get(0), nil
}

// ParseMessageMap converts a JSON Schema map to a protoreflect.MessageDescriptor.
// This is a convenience wrapper around ParseMessage for map inputs.
func (p *Parser) ParseMessageMap(schemaMap map[string]any) (protoreflect.MessageDescriptor, error) {
	b, err := json.Marshal(schemaMap)
	if err != nil {
		return nil, err
	}
	return p.ParseMessage(b)
}

// unmarshalSchema unmarshals JSON into a jsonschema.Schema while preserving
// custom extensions (x-* properties) in the Extras field.
func (p *Parser) unmarshalSchema(data []byte) (*jsonschema.Schema, error) {
	// First unmarshal into a map to capture all properties including custom ones
	var rawMap map[string]any
	if err := json.Unmarshal(data, &rawMap); err != nil {
		return nil, err
	}

	// Now unmarshal into the schema struct
	var schema jsonschema.Schema
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, err
	}

	// Populate Extras recursively
	p.populateExtras(&schema, rawMap)

	return &schema, nil
}

// populateExtras recursively populates Extras fields for a schema and its nested schemas.
func (p *Parser) populateExtras(schema *jsonschema.Schema, rawMap map[string]any) {
	// Populate Extras for the current schema
	schema.Extras = p.extractExtras(rawMap)

	// Recursively process nested schemas in properties
	if schema.Properties != nil {
		if propData, ok := rawMap["properties"].(map[string]any); ok {
			for pair := schema.Properties.Oldest(); pair != nil; pair = pair.Next() {
				if propMap, ok := propData[pair.Key].(map[string]any); ok {
					p.populateExtras(pair.Value, propMap)
				}
			}
		}
	}

	// Process items
	if schema.Items != nil {
		if itemsData, ok := rawMap["items"].(map[string]any); ok {
			p.populateExtras(schema.Items, itemsData)
		}
	}

	// Process additionalProperties
	if schema.AdditionalProperties != nil {
		if addlProps, ok := rawMap["additionalProperties"].(map[string]any); ok {
			p.populateExtras(schema.AdditionalProperties, addlProps)
		}
	}

	// Process AnyOf
	if len(schema.AnyOf) > 0 {
		if anyOfData, ok := rawMap["anyOf"].([]any); ok {
			for i, anyOfSchema := range schema.AnyOf {
				if i < len(anyOfData) {
					if anyOfMap, ok := anyOfData[i].(map[string]any); ok {
						p.populateExtras(anyOfSchema, anyOfMap)
					}
				}
			}
		}
	}

	// Process OneOf
	if len(schema.OneOf) > 0 {
		if oneOfData, ok := rawMap["oneOf"].([]any); ok {
			for i, oneOfSchema := range schema.OneOf {
				if i < len(oneOfData) {
					if oneOfMap, ok := oneOfData[i].(map[string]any); ok {
						p.populateExtras(oneOfSchema, oneOfMap)
					}
				}
			}
		}
	}

	// Process AllOf
	if len(schema.AllOf) > 0 {
		if allOfData, ok := rawMap["allOf"].([]any); ok {
			for i, allOfSchema := range schema.AllOf {
				if i < len(allOfData) {
					if allOfMap, ok := allOfData[i].(map[string]any); ok {
						p.populateExtras(allOfSchema, allOfMap)
					}
				}
			}
		}
	}

	// Process definitions
	if schema.Definitions != nil {
		if defs, ok := rawMap["$defs"].(map[string]any); ok {
			for defName, defSchema := range schema.Definitions {
				if defMap, ok := defs[defName].(map[string]any); ok {
					p.populateExtras(defSchema, defMap)
				}
			}
		}
	}
}

// extractExtras extracts custom properties (those starting with 'x-') from a raw map.
func (p *Parser) extractExtras(rawMap map[string]any) map[string]any {
	extras := make(map[string]any)
	for key, val := range rawMap {
		// Include properties starting with "x-" as extensions
		if len(key) > 2 && key[:2] == "x-" {
			extras[key] = val
		}
	}
	if len(extras) == 0 {
		return nil
	}
	return extras
}

// parseMessage converts a JSON Schema object to a DescriptorProto.
func (p *Parser) parseMessage(name string, schema *jsonschema.Schema) (*descriptorpb.DescriptorProto, error) {
	// Check if we've already parsed this message
	if existing, ok := p.messages[name]; ok {
		return existing, nil
	}

	msg := &descriptorpb.DescriptorProto{
		Name: new(name),
	}

	// Register message immediately to handle circular references
	p.messages[name] = msg
	p.messageNames[name] = true

	if schema.Properties == nil || schema.Properties.Len() == 0 {
		// TODO: handle scalars using wrapperspb types
		// No properties, return empty message
		return msg, nil
	}

	// Get required fields
	required := schema.Required

	// Sort property names for stable field numbers
	propNames := make([]string, 0, schema.Properties.Len())
	for pair := schema.Properties.Oldest(); pair != nil; pair = pair.Next() {
		propNames = append(propNames, pair.Key)
	}
	sort.Strings(propNames)

	// Parse each field
	fieldNum := int32(1)
	for _, propName := range propNames {
		propSchema, _ := schema.Properties.Get(propName)
		if propSchema == nil {
			continue
		}

		field, err := p.parseField(propName, propSchema, fieldNum, required, msg)
		if err != nil {
			return nil, fmt.Errorf("failed to parse field %q: %w", propName, err)
		}

		msg.Field = append(msg.Field, field)
		fieldNum++
	}

	return msg, nil
}

// parseField converts a JSON Schema property to a FieldDescriptorProto.
// parentMsg is the message this field belongs to (for nested map entry messages).
func (p *Parser) parseField(name string, schema *jsonschema.Schema, num int32, required []string, parentMsg *descriptorpb.DescriptorProto) (*descriptorpb.FieldDescriptorProto, error) {
	field := &descriptorpb.FieldDescriptorProto{
		Name:   new(SanitizeFieldName(name)),
		Number: new(num),
	}

	// Set JSON name if using JSON names
	if p.opts.UseJSONNames {
		field.JsonName = new(name)
	}

	// Check for nullable types (anyOf/oneOf with null)
	if nullable := p.extractNullableSchema(schema); nullable != nil {
		// Parse the non-null schema and mark as optional
		field, err := p.parseField(name, nullable, num, required, parentMsg)
		if err != nil {
			return nil, err
		}
		// Ensure it's marked as optional
		if p.opts.Syntax == "proto3" {
			field.Proto3Optional = new(true)
		}
		return field, nil
	}

	// Check for $ref first
	if schema.Ref != "" {
		return p.parseRefField(name, schema.Ref, num, required)
	}

	// Check for enum
	// if len(schema.Enum) > 0 {
	// 	return p.parseEnumField(name, schema, num, required)
	// }

	// Check for well-known types based on format
	if wktField := p.tryParseWellKnownType(name, schema, num, required); wktField != nil {
		return wktField, nil
	}

	// Determine type from schema
	schemaType := p.getSchemaType(schema)
	switch schemaType {
	case "boolean":
		field.Type = new(descriptorpb.FieldDescriptorProto_TYPE_BOOL)
	case "string":
		field.Type = new(p.inferStringType(schema))
	case "integer":
		field.Type = new(p.inferIntegerType(schema))
	case "number":
		field.Type = new(p.inferNumberType(schema))
	case "array":
		return p.parseArrayField(name, schema, num, required, parentMsg)
	case "object":
		// Check if this is a map type (has additionalProperties but no properties)
		if p.isMapType(schema) {
			return p.parseMapField(name, schema, num, parentMsg)
		}
		return p.parseNestedObjectField(name, schema, num, required)
	default:
		return nil, fmt.Errorf("unsupported schema type: %s", schemaType)
	}

	// Set optional label for proto3 if not required
	if p.opts.Syntax == "proto3" && !slices.Contains(required, name) {
		field.Proto3Optional = new(true)
	}

	// Add validation constraints if enabled
	if p.opts.EnableValidation {
		p.addValidationConstraints(field, schema, required, name)
	}

	return field, nil
}

// getMessageName extracts a message name from the schema.
func (p *Parser) getMessageName(schema *jsonschema.Schema) string {
	if p.opts.MessageName != "" {
		return SanitizeMessageName(p.opts.MessageName)
	}
	if schema.Title != "" {
		return SanitizeMessageName(schema.Title)
	}
	return "GeneratedMessage"
}

// getSchemaType returns the type of the JSON Schema.
func (p *Parser) getSchemaType(schema *jsonschema.Schema) string {
	return schema.Type
}

// inferStringType determines the appropriate protobuf string type.
func (p *Parser) inferStringType(schema *jsonschema.Schema) descriptorpb.FieldDescriptorProto_Type {
	// Check for bytes format
	if schema.Format == "byte" || schema.Format == "binary" {
		return descriptorpb.FieldDescriptorProto_TYPE_BYTES
	}
	return descriptorpb.FieldDescriptorProto_TYPE_STRING
}

// inferIntegerType determines the appropriate protobuf integer type.
func (p *Parser) inferIntegerType(schema *jsonschema.Schema) descriptorpb.FieldDescriptorProto_Type {
	// Check x-dtkt-format first (preserves exact Go type)
	if dtkFormat, ok := p.getExtension(schema, "x-dtkt-format").(string); ok {
		switch dtkFormat {
		case "int32":
			return descriptorpb.FieldDescriptorProto_TYPE_INT32
		case "int", "int64":
			return descriptorpb.FieldDescriptorProto_TYPE_INT64
		case "int8", "int16":
			// Map smaller ints to int32 for protobuf compatibility
			return descriptorpb.FieldDescriptorProto_TYPE_INT32
		case "uint32":
			return descriptorpb.FieldDescriptorProto_TYPE_UINT32
		case "uint64":
			return descriptorpb.FieldDescriptorProto_TYPE_UINT64
		case "uint8", "uint16":
			// Map smaller uints to uint32 for protobuf compatibility
			return descriptorpb.FieldDescriptorProto_TYPE_UINT32
		}
	}

	// Check standard format hint
	if schema.Format != "" {
		switch schema.Format {
		case "int32":
			return descriptorpb.FieldDescriptorProto_TYPE_INT32
		case "int64":
			return descriptorpb.FieldDescriptorProto_TYPE_INT64
		case "uint32":
			return descriptorpb.FieldDescriptorProto_TYPE_UINT32
		case "uint64":
			return descriptorpb.FieldDescriptorProto_TYPE_UINT64
		}
	}

	// Check minimum to determine signed vs unsigned
	if schema.Minimum != "" {
		min, _ := schema.Minimum.Float64()
		if min >= 0 {
			// Check maximum to determine size
			if schema.Maximum != "" {
				max, _ := schema.Maximum.Float64()
				if max <= 4294967295 { // 2^32 - 1
					return descriptorpb.FieldDescriptorProto_TYPE_UINT32
				}
				return descriptorpb.FieldDescriptorProto_TYPE_UINT64
			}
			return descriptorpb.FieldDescriptorProto_TYPE_UINT32
		}
	}

	// Default to signed int32
	return descriptorpb.FieldDescriptorProto_TYPE_INT32
}

// inferNumberType determines the appropriate protobuf floating-point type.
func (p *Parser) inferNumberType(schema *jsonschema.Schema) descriptorpb.FieldDescriptorProto_Type {
	// Check x-dtkt-format first (preserves exact Go type)
	if dtkFormat, ok := p.getExtension(schema, "x-dtkt-format").(string); ok {
		switch dtkFormat {
		case "float32":
			return descriptorpb.FieldDescriptorProto_TYPE_FLOAT
		case "float64":
			return descriptorpb.FieldDescriptorProto_TYPE_DOUBLE
		}
	}

	// Check standard format hint
	if schema.Format == "float" {
		return descriptorpb.FieldDescriptorProto_TYPE_FLOAT
	}

	// Default to double
	return descriptorpb.FieldDescriptorProto_TYPE_DOUBLE
}

// getExtension retrieves a custom extension field from the schema.
// Extensions like x-dtkt-format preserve additional type information.
func (p *Parser) getExtension(schema *jsonschema.Schema, key string) any {
	if schema.Extras != nil {
		if val, ok := schema.Extras[key]; ok {
			return val
		}
	}
	return nil
}

// SanitizeFieldName converts a JSON property name to a valid protobuf field name.
func SanitizeFieldName(name string) string {
	// Replace invalid characters with underscores
	name = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			return r
		}
		return '_'
	}, name)

	// Ensure it doesn't start with a number
	if len(name) > 0 && name[0] >= '0' && name[0] <= '9' {
		name = "_" + name
	}

	return name
}

// SanitizeMessageName converts a title to a valid protobuf message name.
func SanitizeMessageName(name string) string {
	// Remove spaces and invalid characters, capitalize each word
	words := strings.FieldsFunc(name, func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9')
	})

	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}

	name = strings.Join(words, "")
	if name == "" {
		return "Message"
	}

	return name
}

// parseArrayField handles array types (repeated fields)
func (p *Parser) parseArrayField(name string, schema *jsonschema.Schema, num int32, required []string, parentMsg *descriptorpb.DescriptorProto) (*descriptorpb.FieldDescriptorProto, error) {
	if schema.Items == nil {
		return nil, fmt.Errorf("array field %q missing items definition", name)
	}

	// Create a field for the item type
	itemField, err := p.parseField(name, schema.Items, num, required, parentMsg)
	if err != nil {
		return nil, err
	}

	// Mark as repeated
	itemField.Label = new(descriptorpb.FieldDescriptorProto_LABEL_REPEATED)
	// Arrays are never optional
	itemField.Proto3Optional = nil

	return itemField, nil
}

// parseNestedObjectField handles nested object types
func (p *Parser) parseNestedObjectField(name string, schema *jsonschema.Schema, num int32, required []string) (*descriptorpb.FieldDescriptorProto, error) {
	// Generate a name for the nested message
	nestedName := SanitizeMessageName(name)

	// Ensure unique name
	if p.messageNames[nestedName] {
		// Add suffix to make it unique
		for i := 1; ; i++ {
			testName := fmt.Sprintf("%s%d", nestedName, i)
			if !p.messageNames[testName] {
				nestedName = testName
				break
			}
		}
	}

	// Parse the nested message
	_, err := p.parseMessage(nestedName, schema)
	if err != nil {
		return nil, err
	}

	field := &descriptorpb.FieldDescriptorProto{
		Name:     new(SanitizeFieldName(name)),
		Number:   new(num),
		Type:     new(descriptorpb.FieldDescriptorProto_TYPE_MESSAGE),
		TypeName: new("." + p.opts.PackageName + "." + nestedName),
	}

	if p.opts.UseJSONNames {
		field.JsonName = new(name)
	}

	if p.opts.Syntax == "proto3" && !slices.Contains(required, name) {
		field.Proto3Optional = new(true)
	}

	return field, nil
}

// parseRefField handles $ref references
func (p *Parser) parseRefField(name string, ref string, num int32, required []string) (*descriptorpb.FieldDescriptorProto, error) {
	// Extract the referenced type name from the $ref
	// Assuming format like "#/$defs/Customer"
	refName := ""
	if strings.HasPrefix(ref, "#/$defs/") {
		refName = strings.TrimPrefix(ref, "#/$defs/")
	} else if strings.HasPrefix(ref, "#/definitions/") {
		refName = strings.TrimPrefix(ref, "#/definitions/")
	} else {
		return nil, fmt.Errorf("unsupported $ref format: %s", ref)
	}

	refName = SanitizeMessageName(refName)

	// Create field referencing the message
	field := &descriptorpb.FieldDescriptorProto{
		Name:     new(SanitizeFieldName(name)),
		Number:   new(num),
		Type:     new(descriptorpb.FieldDescriptorProto_TYPE_MESSAGE),
		TypeName: new("." + p.opts.PackageName + "." + refName),
	}

	if p.opts.UseJSONNames {
		field.JsonName = new(name)
	}

	if p.opts.Syntax == "proto3" && !slices.Contains(required, name) {
		field.Proto3Optional = new(true)
	}

	return field, nil
}

// tryParseWellKnownType attempts to parse a field as a well-known protobuf type.
// Returns nil if the field is not a well-known type.
func (p *Parser) tryParseWellKnownType(name string, schema *jsonschema.Schema, num int32, required []string) *descriptorpb.FieldDescriptorProto {
	schemaType := p.getSchemaType(schema)
	if schemaType != "string" {
		return nil
	}

	if schema.Format == "" {
		return nil
	}

	var typeName string
	switch schema.Format {
	case "duration":
		p.imports = util.SliceSet(append(p.imports, wellKnownImports[durationName]))
		typeName = string(durationName)
	case "date-time":
		p.imports = util.SliceSet(append(p.imports, wellKnownImports[timestampName]))
		typeName = string(timestampName)
	default:
		return nil
	}

	field := &descriptorpb.FieldDescriptorProto{
		Name:     new(SanitizeFieldName(name)),
		Number:   new(num),
		Type:     new(descriptorpb.FieldDescriptorProto_TYPE_MESSAGE),
		TypeName: new("." + typeName),
	}

	if p.opts.UseJSONNames {
		field.JsonName = new(name)
	}

	if p.opts.Syntax == "proto3" && !slices.Contains(required, name) {
		field.Proto3Optional = new(true)
	}

	return field
}

// extractNullableSchema detects anyOf/oneOf with null and extracts the non-null schema.
// Returns nil if not a nullable pattern.
func (p *Parser) extractNullableSchema(schema *jsonschema.Schema) *jsonschema.Schema {
	// Check anyOf pattern: {"anyOf": [{"type": "null"}, {actual schema}]}
	if len(schema.AnyOf) == 2 {
		for _, item := range schema.AnyOf {
			if item.Type == "null" {
				continue
			}
			// This is the non-null schema
			return item
		}
	}

	// Check oneOf pattern
	if len(schema.OneOf) == 2 {
		for _, item := range schema.OneOf {
			if item.Type == "null" {
				continue
			}
			return item
		}
	}

	return nil
}

// isMapType checks if an object schema represents a map type.
// A map type in JSON Schema is an object with additionalProperties but either
// no properties or only explicitly marked as a map container.
func (p *Parser) isMapType(schema *jsonschema.Schema) bool {
	return schema.Type == "object" &&
		(schema.Properties == nil || schema.Properties.Len() == 0) &&
		schema.AdditionalProperties != nil
}

// parseMapField handles map types (represented as additionalProperties in JSON Schema)
// parentMsg is the message this map field belongs to - the entry message will be nested inside it.
func (p *Parser) parseMapField(name string, schema *jsonschema.Schema, num int32, parentMsg *descriptorpb.DescriptorProto) (*descriptorpb.FieldDescriptorProto, error) {
	if schema.AdditionalProperties == nil {
		return nil, fmt.Errorf("map field %q has invalid additionalProperties", name)
	}

	// Generate a name for the map entry message (proto3 convention)
	entryName := SanitizeMessageName(name) + "Entry"

	// Ensure unique name within the parent message's nested types
	existingNames := make(map[string]bool)
	for _, nested := range parentMsg.NestedType {
		existingNames[nested.GetName()] = true
	}

	if existingNames[entryName] {
		for i := 1; ; i++ {
			testName := fmt.Sprintf("%s%d", entryName, i)
			if !existingNames[testName] {
				entryName = testName
				break
			}
		}
	}

	// Create the map entry message
	// In proto3, maps are represented as: map<key_type, value_type>
	// Which is syntactic sugar for: repeated MapFieldEntry map_field = N;
	// Where MapFieldEntry is: message MapFieldEntry { key_type key = 1; value_type value = 2; }

	entryMsg := &descriptorpb.DescriptorProto{
		Name: new(entryName),
		Options: &descriptorpb.MessageOptions{
			MapEntry: new(true),
		},
	}

	// Key field (always string for JSON Schema maps)
	keyField := &descriptorpb.FieldDescriptorProto{
		Name:   new("key"),
		Number: new(int32(1)),
		Type:   new(descriptorpb.FieldDescriptorProto_TYPE_STRING),
		Label:  new(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
	}

	// Value field - parse from additionalProperties schema
	valueField, err := p.parseField("value", schema.AdditionalProperties, 2, []string{}, entryMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to parse map value type: %w", err)
	}
	// Map values are always optional in the entry message
	valueField.Label = new(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL)
	valueField.Proto3Optional = nil
	// Preserve the field name as "value" for map entries
	valueField.Name = new("value")

	entryMsg.Field = []*descriptorpb.FieldDescriptorProto{keyField, valueField}

	// Add the entry message as a nested type of the parent message
	parentMsg.NestedType = append(parentMsg.NestedType, entryMsg)

	// Get parent message name for the type reference
	parentMsgName := parentMsg.GetName()

	// Create the map field (repeated entry message)
	field := &descriptorpb.FieldDescriptorProto{
		Name:     new(SanitizeFieldName(name)),
		Number:   new(num),
		Type:     new(descriptorpb.FieldDescriptorProto_TYPE_MESSAGE),
		TypeName: new("." + p.opts.PackageName + "." + parentMsgName + "." + entryName),
		Label:    new(descriptorpb.FieldDescriptorProto_LABEL_REPEATED),
	}

	if p.opts.UseJSONNames {
		field.JsonName = new(name)
	}

	// Maps are never optional (they're repeated)
	field.Proto3Optional = nil

	return field, nil
}

// addValidationConstraints adds protovalidate field options based on JSON Schema constraints.
func (p *Parser) addValidationConstraints(field *descriptorpb.FieldDescriptorProto, schema *jsonschema.Schema, required []string, name string) {
	rules := &validate.FieldRules{}
	hasConstraints := false

	// Check if field is required
	if slices.Contains(required, name) {
		rules.Required = new(true)
		hasConstraints = true
	}

	// Get the field type for type-specific constraints
	fieldType := field.GetType()

	switch fieldType {
	case descriptorpb.FieldDescriptorProto_TYPE_STRING:
		if stringRules := p.extractStringRules(schema); stringRules != nil {
			rules.Type = &validate.FieldRules_String_{String_: stringRules}
			hasConstraints = true
		}
	case descriptorpb.FieldDescriptorProto_TYPE_INT32, descriptorpb.FieldDescriptorProto_TYPE_SINT32, descriptorpb.FieldDescriptorProto_TYPE_SFIXED32:
		if int32Rules := p.extractInt32Rules(schema); int32Rules != nil {
			rules.Type = &validate.FieldRules_Int32{Int32: int32Rules}
			hasConstraints = true
		}
	case descriptorpb.FieldDescriptorProto_TYPE_INT64, descriptorpb.FieldDescriptorProto_TYPE_SINT64, descriptorpb.FieldDescriptorProto_TYPE_SFIXED64:
		if int64Rules := p.extractInt64Rules(schema); int64Rules != nil {
			rules.Type = &validate.FieldRules_Int64{Int64: int64Rules}
			hasConstraints = true
		}
	case descriptorpb.FieldDescriptorProto_TYPE_UINT32, descriptorpb.FieldDescriptorProto_TYPE_FIXED32:
		if uint32Rules := p.extractUint32Rules(schema); uint32Rules != nil {
			rules.Type = &validate.FieldRules_Uint32{Uint32: uint32Rules}
			hasConstraints = true
		}
	case descriptorpb.FieldDescriptorProto_TYPE_UINT64, descriptorpb.FieldDescriptorProto_TYPE_FIXED64:
		if uint64Rules := p.extractUint64Rules(schema); uint64Rules != nil {
			rules.Type = &validate.FieldRules_Uint64{Uint64: uint64Rules}
			hasConstraints = true
		}
	case descriptorpb.FieldDescriptorProto_TYPE_FLOAT:
		if floatRules := p.extractFloatRules(schema); floatRules != nil {
			rules.Type = &validate.FieldRules_Float{Float: floatRules}
			hasConstraints = true
		}
	case descriptorpb.FieldDescriptorProto_TYPE_DOUBLE:
		if doubleRules := p.extractDoubleRules(schema); doubleRules != nil {
			rules.Type = &validate.FieldRules_Double{Double: doubleRules}
			hasConstraints = true
		}
	case descriptorpb.FieldDescriptorProto_TYPE_BYTES:
		if bytesRules := p.extractBytesRules(schema); bytesRules != nil {
			rules.Type = &validate.FieldRules_Bytes{Bytes: bytesRules}
			hasConstraints = true
		}
	}

	// Apply constraints if any were found
	if hasConstraints {
		p.setFieldOptions(field, rules)
		p.imports = util.SliceSet(append(p.imports, wellKnownImports[bufValidateName]))
	}
}

// setFieldOptions sets the protovalidate field options on a field descriptor.
func (p *Parser) setFieldOptions(field *descriptorpb.FieldDescriptorProto, rules *validate.FieldRules) {
	if field.Options == nil {
		field.Options = &descriptorpb.FieldOptions{}
	}

	// Marshal the rules and set as extension
	ruleBytes, err := proto.Marshal(rules)
	if err != nil {
		return // Skip on error
	}

	proto.SetExtension(field.Options, validate.E_Field, rules)
	_ = ruleBytes // Avoid unused variable
}

// extractStringRules extracts string validation rules from JSON Schema.
func (p *Parser) extractStringRules(schema *jsonschema.Schema) *validate.StringRules {
	rules := &validate.StringRules{}
	hasRules := false

	// MinLength
	if schema.MinLength != nil {
		rules.MinLen = schema.MinLength
		hasRules = true
	}

	// MaxLength
	if schema.MaxLength != nil {
		rules.MaxLen = schema.MaxLength
		hasRules = true
	}

	// Pattern
	if schema.Pattern != "" {
		rules.Pattern = &schema.Pattern
		hasRules = true
	}

	// Const
	if constVal, ok := schema.Const.(string); ok {
		rules.Const = &constVal
		hasRules = true
	}

	// Enum becomes In
	if len(schema.Enum) > 0 {
		strVals := make([]string, 0, len(schema.Enum))
		for _, val := range schema.Enum {
			if strVal, ok := val.(string); ok {
				strVals = append(strVals, strVal)
			}
		}
		if len(strVals) > 0 {
			rules.In = strVals
			hasRules = true
		}
	}

	// Format-based well-known patterns
	if schema.Format != "" {
		switch schema.Format {
		case "email":
			rules.WellKnown = &validate.StringRules_Email{Email: true}
			hasRules = true
		case "hostname":
			rules.WellKnown = &validate.StringRules_Hostname{Hostname: true}
			hasRules = true
		case "ipv4":
			rules.WellKnown = &validate.StringRules_Ipv4{Ipv4: true}
			hasRules = true
		case "ipv6":
			rules.WellKnown = &validate.StringRules_Ipv6{Ipv6: true}
			hasRules = true
		case "uri":
			rules.WellKnown = &validate.StringRules_Uri{Uri: true}
			hasRules = true
		case "uri-reference":
			rules.WellKnown = &validate.StringRules_UriRef{UriRef: true}
			hasRules = true
		case "uuid":
			rules.WellKnown = &validate.StringRules_Uuid{Uuid: true}
			hasRules = true
		}
	}

	if !hasRules {
		return nil
	}
	return rules
}

// extractInt32Rules extracts int32 validation rules from JSON Schema.
func (p *Parser) extractInt32Rules(schema *jsonschema.Schema) *validate.Int32Rules {
	rules := &validate.Int32Rules{}
	hasRules := false

	if constVal, ok := schema.Const.(float64); ok {
		val := int32(constVal)
		rules.Const = &val
		hasRules = true
	}

	if schema.Minimum != "" {
		if min, err := schema.Minimum.Float64(); err == nil {
			val := int32(min)
			rules.GreaterThan = &validate.Int32Rules_Gte{
				Gte: val,
			}
			hasRules = true
		}
	}

	if schema.Maximum != "" {
		if max, err := schema.Maximum.Float64(); err == nil {
			val := int32(max)
			rules.LessThan = &validate.Int32Rules_Lte{
				Lte: val,
			}
			hasRules = true
		}
	}

	if schema.ExclusiveMinimum != "" {
		if exclusiveMin, err := schema.ExclusiveMinimum.Float64(); err == nil {
			val := int32(exclusiveMin)
			rules.GreaterThan = &validate.Int32Rules_Gt{
				Gt: val,
			}
			hasRules = true
		}
	}

	if schema.ExclusiveMaximum != "" {
		if exclusiveMax, err := schema.ExclusiveMaximum.Float64(); err == nil {
			val := int32(exclusiveMax)
			rules.LessThan = &validate.Int32Rules_Lt{
				Lt: val,
			}
			hasRules = true
		}
	}

	if !hasRules {
		return nil
	}
	return rules
}

// extractInt64Rules extracts int64 validation rules from JSON Schema.
func (p *Parser) extractInt64Rules(schema *jsonschema.Schema) *validate.Int64Rules {
	rules := &validate.Int64Rules{}
	hasRules := false

	if constVal, ok := schema.Const.(float64); ok {
		val := int64(constVal)
		rules.Const = &val
		hasRules = true
	}

	if schema.Minimum != "" {
		if min, err := schema.Minimum.Float64(); err == nil {
			val := int64(min)
			rules.GreaterThan = &validate.Int64Rules_Gte{
				Gte: val,
			}
			hasRules = true
		}
	}

	if schema.Maximum != "" {
		if max, err := schema.Maximum.Float64(); err == nil {
			val := int64(max)
			rules.LessThan = &validate.Int64Rules_Lte{
				Lte: val,
			}
			hasRules = true
		}
	}

	if !hasRules {
		return nil
	}
	return rules
}

// extractUint32Rules extracts uint32 validation rules from JSON Schema.
func (p *Parser) extractUint32Rules(schema *jsonschema.Schema) *validate.UInt32Rules {
	rules := &validate.UInt32Rules{}
	hasRules := false

	if constVal, ok := schema.Const.(float64); ok {
		val := uint32(constVal)
		rules.Const = &val
		hasRules = true
	}

	if schema.Minimum != "" {
		if min, err := schema.Minimum.Float64(); err == nil {
			val := uint32(min)
			rules.GreaterThan = &validate.UInt32Rules_Gte{
				Gte: val,
			}
			hasRules = true
		}
	}

	if schema.Maximum != "" {
		if max, err := schema.Maximum.Float64(); err == nil {
			val := uint32(max)
			rules.LessThan = &validate.UInt32Rules_Lte{
				Lte: val,
			}
			hasRules = true
		}
	}

	if !hasRules {
		return nil
	}
	return rules
}

// extractUint64Rules extracts uint64 validation rules from JSON Schema.
func (p *Parser) extractUint64Rules(schema *jsonschema.Schema) *validate.UInt64Rules {
	rules := &validate.UInt64Rules{}
	hasRules := false

	if constVal, ok := schema.Const.(float64); ok {
		val := uint64(constVal)
		rules.Const = &val
		hasRules = true
	}

	if schema.Minimum != "" {
		if min, err := schema.Minimum.Float64(); err == nil {
			val := uint64(min)
			rules.GreaterThan = &validate.UInt64Rules_Gte{
				Gte: val,
			}
			hasRules = true
		}
	}

	if schema.Maximum != "" {
		if max, err := schema.Maximum.Float64(); err == nil {
			val := uint64(max)
			rules.LessThan = &validate.UInt64Rules_Lte{
				Lte: val,
			}
			hasRules = true
		}
	}

	if !hasRules {
		return nil
	}
	return rules
}

// extractFloatRules extracts float validation rules from JSON Schema.
func (p *Parser) extractFloatRules(schema *jsonschema.Schema) *validate.FloatRules {
	rules := &validate.FloatRules{}
	hasRules := false

	if constVal, ok := schema.Const.(float64); ok {
		val := float32(constVal)
		rules.Const = &val
		hasRules = true
	}

	if schema.Minimum != "" {
		if min, err := schema.Minimum.Float64(); err == nil {
			val := float32(min)
			rules.GreaterThan = &validate.FloatRules_Gte{
				Gte: val,
			}
			hasRules = true
		}
	}

	if schema.Maximum != "" {
		if max, err := schema.Maximum.Float64(); err == nil {
			val := float32(max)
			rules.LessThan = &validate.FloatRules_Lte{
				Lte: val,
			}
			hasRules = true
		}
	}

	if !hasRules {
		return nil
	}
	return rules
}

// extractDoubleRules extracts double validation rules from JSON Schema.
func (p *Parser) extractDoubleRules(schema *jsonschema.Schema) *validate.DoubleRules {
	rules := &validate.DoubleRules{}
	hasRules := false

	if constVal, ok := schema.Const.(float64); ok {
		rules.Const = &constVal
		hasRules = true
	}

	if schema.Minimum != "" {
		if min, err := schema.Minimum.Float64(); err == nil {
			rules.GreaterThan = &validate.DoubleRules_Gte{
				Gte: min,
			}
			hasRules = true
		}
	}

	if schema.Maximum != "" {
		if max, err := schema.Maximum.Float64(); err == nil {
			rules.LessThan = &validate.DoubleRules_Lte{
				Lte: max,
			}
			hasRules = true
		}
	}

	if !hasRules {
		return nil
	}
	return rules
}

// extractBytesRules extracts bytes validation rules from JSON Schema.
func (p *Parser) extractBytesRules(schema *jsonschema.Schema) *validate.BytesRules {
	rules := &validate.BytesRules{}
	hasRules := false

	// Bytes are base64 encoded in JSON, so minLength/maxLength refer to encoded length
	// We need to approximate the byte length
	if schema.MinLength != nil {
		// Base64 decoding: ~3/4 of encoded length
		byteLen := uint64(float64(*schema.MinLength) * 0.75)
		rules.MinLen = &byteLen
		hasRules = true
	}

	if schema.MaxLength != nil {
		byteLen := uint64(float64(*schema.MaxLength) * 0.75)
		rules.MaxLen = &byteLen
		hasRules = true
	}

	if !hasRules {
		return nil
	}
	return rules
}

// parseEnumField handles string fields with enum constraints
// func (p *Parser) parseEnumField(name string, schema *jsonschema.Schema, num int32, required []string) (*descriptorpb.FieldDescriptorProto, error) {
// 	if len(schema.Enum) == 0 {
// 		return nil, fmt.Errorf("enum field %q has invalid enum values", name)
// 	}

// 	// Generate enum name from field name
// 	enumName := SanitizeMessageName(name)

// 	// Ensure unique name
// 	if _, exists := p.enums[enumName]; exists {
// 		for i := 1; ; i++ {
// 			testName := fmt.Sprintf("%s%d", enumName, i)
// 			if _, exists := p.enums[testName]; !exists {
// 				enumName = testName
// 				break
// 			}
// 		}
// 	}

// 	// Create enum descriptor
// 	enumDesc := &descriptorpb.EnumDescriptorProto{
// 		Name: new(enumName),
// 	}

// 	// Add UNSPECIFIED value as first value (proto3 convention)
// 	enumDesc.Value = append(enumDesc.Value, &descriptorpb.EnumValueDescriptorProto{
// 		Name:   new(strings.ToUpper(enumName) + "_UNSPECIFIED"),
// 		Number: new(int32(0)),
// 	})

// 	// Add enum values
// 	for i, val := range schema.Enum {
// 		if strVal, ok := val.(string); ok {
// 			valueName := strings.ToUpper(enumName) + "_" + strings.ToUpper(SanitizeFieldName(strVal))
// 			enumDesc.Value = append(enumDesc.Value, &descriptorpb.EnumValueDescriptorProto{
// 				Name:   new(valueName),
// 				Number: new(int32(i + 1)),
// 			})
// 		}
// 	}

// 	p.enums[enumName] = enumDesc

// 	// Create field referencing the enum
// 	field := &descriptorpb.FieldDescriptorProto{
// 		Name:     new(SanitizeFieldName(name)),
// 		Number:   new(num),
// 		Type:     new(descriptorpb.FieldDescriptorProto_TYPE_ENUM),
// 		TypeName: new("." + p.opts.PackageName + "." + enumName),
// 	}

// 	if p.opts.UseJSONNames {
// 		field.JsonName = new(name)
// 	}

// 	if p.opts.Syntax == "proto3" && !slices.Contains(required, name) {
// 		field.Proto3Optional = new(true)
// 	}

// 	return field, nil
// }
