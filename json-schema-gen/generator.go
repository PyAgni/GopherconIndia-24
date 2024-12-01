package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
	"text/template"
)

// SchemaField represents a single field in the JSON schema
type SchemaField struct {
	Name        string
	Type        string
	Required    bool
	MinLength   *int
	Format      string
	Description string
}

// SchemaType represents the complete schema structure
type SchemaType struct {
	TypeName string
	Fields   []SchemaField
}

var schemaTemplate = template.Must(template.New("schema").Parse(`{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "title": "{{.TypeName}}",
    "type": "object",
    "properties": {
        {{- range $i, $field := .Fields}}
        {{if $i}},{{end}}
        "{{$field.Name}}": {
            "type": "{{$field.Type}}"
            {{- if $field.Required}},
            "required": true
            {{- end}}
            {{- if $field.MinLength}},
            "minLength": {{$field.MinLength}}
            {{- end}}
            {{- if $field.Format}},
            "format": "{{$field.Format}}"
            {{- end}}
            {{- if $field.Description}},
            "description": "{{$field.Description}}"
            {{- end}}
        }
        {{- end}}
    }
}`))

func main() {
	// Parse command line flags
	typeName := flag.String("type", "", "type name to generate schema for")
	flag.Parse()

	if *typeName == "" {
		fmt.Println("Please specify type name using -type flag")
		os.Exit(1)
	}

	// Get the input file from remaining args (provided by go:generate)
	if len(flag.Args()) < 1 {
		fmt.Println("Please specify an input file")
		os.Exit(1)
	}
	inputFile := flag.Args()[0]

	// Parse the Go source file
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, inputFile, nil, parser.ParseComments)
	if err != nil {
		fmt.Printf("Error parsing file: %v\n", err)
		os.Exit(1)
	}

	// Find and process the requested type
	var schema *SchemaType
	ast.Inspect(node, func(n ast.Node) bool {
		// check if the node represents a type declaration
		typeSpec, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}

		// continue if this is not the type we're looking for
		if typeSpec.Name.Name != *typeName {
			return true
		}

		// ensure this is a struct
		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			return true
		}

		schema = parseStruct(typeSpec.Name.Name, structType)
		return false
	})

	if schema == nil {
		fmt.Printf("Type %s not found in %s\n", *typeName, inputFile)
		os.Exit(1)
	}

	// Generate the schema file
	if err := generateSchemaFile(schema); err != nil {
		fmt.Printf("Error generating schema: %v\n", err)
		os.Exit(1)
	}
}

func parseStruct(typeName string, structType *ast.StructType) *SchemaType {
	schema := &SchemaType{
		TypeName: typeName,
		Fields:   make([]SchemaField, 0),
	}

	for _, field := range structType.Fields.List {
		if field.Tag == nil {
			continue
		}

		// Parse field tags
		tags := parseFieldTags(field.Tag.Value)
		if tags["json"] == "-" {
			continue
		}

		schemaField := SchemaField{
			Name:     tags["json"],
			Type:     getFieldType(field.Type),
			Required: strings.Contains(tags["schema"], "required"),
		}

		// Parse additional schema tags
		if strings.Contains(tags["schema"], "minLength=") {
			minLengthStr := strings.SplitN(strings.SplitN(tags["schema"], "minLength=", 2)[1], ",", 2)[0]
			if val, err := strconv.Atoi(minLengthStr); err == nil {
				schemaField.MinLength = &val
			}
		}

		if strings.Contains(tags["schema"], "format=") {
			format := strings.Split(tags["schema"], "format=")[1]
			format = strings.Split(format, ",")[0]
			schemaField.Format = format
		}

		schema.Fields = append(schema.Fields, schemaField)
	}

	return schema
}

func getFieldType(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		switch t.Name {
		case "string":
			return "string"
		case "int", "int32", "int64":
			return "integer"
		case "float32", "float64":
			return "number"
		case "bool":
			return "boolean"
		}
	}
	return "string"
}

func parseFieldTags(tag string) map[string]string {
	tags := make(map[string]string)
	tag = strings.Trim(tag, "`")

	for _, t := range strings.Split(tag, " ") {
		parts := strings.Split(t, ":")
		if len(parts) != 2 {
			continue
		}
		tags[parts[0]] = strings.Trim(parts[1], "\"")
	}
	return tags
}

func generateSchemaFile(schema *SchemaType) error {
	outputFile := strings.ToLower(schema.TypeName) + ".schema.json"

	var output bytes.Buffer
	if err := schemaTemplate.Execute(&output, schema); err != nil {
		return fmt.Errorf("template execution failed: %v", err)
	}

	fmt.Println("Schema file generated successfully\n", "FileName: ", outputFile)
	return os.WriteFile(outputFile, output.Bytes(), 0644)
}
