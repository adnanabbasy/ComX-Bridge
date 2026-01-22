// Package ai provides AI powered features.
package ai

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"text/template"
)

// TemplateGenerator implements CodeGenerator using Go templates.
type TemplateGenerator struct{}

// NewTemplateGenerator creates a new template generator.
func NewTemplateGenerator() *TemplateGenerator {
	return &TemplateGenerator{}
}

const structTemplate = `type {{.Name}} struct {
{{range .Fields}}	{{.Name}} {{.Type}} // Offset: {{.Offset}}, Length: {{.Length}}
{{end}}}`

func (g *TemplateGenerator) GenerateParser(ctx context.Context, structure *PacketStructure, lang string) (*GeneratedCode, error) {
	if lang == "lua" {
		return g.generateLuaParser(structure)
	}
	if lang != "go" {
		return nil, fmt.Errorf("unsupported language: %s", lang)
	}

	// Transform PacketStructure to template data
	type FieldData struct {
		Name   string
		Type   string
		Offset int
		Length int
	}
	type StructData struct {
		Name   string
		Fields []FieldData
	}

	data := StructData{
		Name: "GeneratedPacket",
	}

	for i, f := range structure.Fields {
		fieldName := fmt.Sprintf("Field%d", i)
		if f.Name != "" {
			fieldName = strings.Title(f.Name)
		}

		fieldType := "[]byte"
		if f.Type != "" {
			fieldType = f.Type
		}

		data.Fields = append(data.Fields, FieldData{
			Name:   fieldName,
			Type:   fieldType,
			Offset: f.Offset,
			Length: f.Length,
		})
	}

	tmpl, err := template.New("struct").Parse(structTemplate)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}

	return &GeneratedCode{
		Language: "go",
		Files: []GeneratedFile{
			{
				Path:    "packet.go",
				Content: buf.String(),
				Type:    "source",
			},
		},
	}, nil
}

const luaParserTemplate = `
function Parse(data)
	local packet = {}
	local offset = 0
	
	{{range .Fields}}
	-- {{.Name}} ({{.Length}} bytes)
	packet["{{.Name}}"] = string.sub(data, {{.Offset}} + 1, {{.Offset}} + {{.Length}})
	{{end}}
	
	return packet
end
`

func (g *TemplateGenerator) generateLuaParser(structure *PacketStructure) (*GeneratedCode, error) {
	tmpl, err := template.New("lua").Parse(luaParserTemplate)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, structure); err != nil {
		return nil, err
	}

	return &GeneratedCode{
		Language: "lua",
		Files: []GeneratedFile{
			{
				Path:    "parser.lua",
				Content: buf.String(),
				Type:    "source",
			},
		},
	}, nil
}

func (g *TemplateGenerator) GenerateProtocol(ctx context.Context, analysis *ProtocolAnalysis, lang string) (*GeneratedCode, error) {
	return nil, fmt.Errorf("not implemented")
}

func (g *TemplateGenerator) GeneratePlugin(ctx context.Context, spec *PluginSpec, lang string) (*GeneratedCode, error) {
	return nil, fmt.Errorf("not implemented")
}

func (g *TemplateGenerator) SupportedLanguages() []string {
	return []string{"go"}
}
