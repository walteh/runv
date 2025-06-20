// adapted from github.com/kei2100/protoc-gen-go-log-valuer/plugin/protoc-gen-go-log-valuer/main.go
// commit: 996332d07459c1ffde02cdce8ac8d6b232580fc1
// license: Apache-2.0

package main

import (
	"fmt"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

const (
	fmtPkg     = protogen.GoImportPath("fmt")
	slogPkg    = protogen.GoImportPath("log/slog")
	strconvPkg = protogen.GoImportPath("strconv")
)

func main() {
	protogen.Options{}.Run(func(plugin *protogen.Plugin) error {
		plugin.SupportedEditionsMaximum = descriptorpb.Edition_EDITION_MAX
		plugin.SupportedEditionsMinimum = descriptorpb.Edition_EDITION_PROTO3
		plugin.SupportedFeatures = uint64(
			pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL |
				pluginpb.CodeGeneratorResponse_FEATURE_SUPPORTS_EDITIONS)
		for _, file := range plugin.FilesByPath {
			if !file.Generate {
				continue
			}
			generateFile(plugin, file)
		}
		return nil
	})
}

func generateFile(gen *protogen.Plugin, file *protogen.File) *protogen.GeneratedFile {
	if len(file.Messages) == 0 {
		return nil
	}
	filename := fmt.Sprintf("%s_slog.pb.go", file.GeneratedFilenamePrefix)
	g := gen.NewGeneratedFile(filename, file.GoImportPath)
	g.P("// Code generated by protoc-gen-slog-valuer. DO NOT EDIT.")
	g.P("//")
	g.P("// source: ", file.Desc.Path())
	g.P()
	g.P("package ", file.GoPackageName)
	g.P()
	for _, m := range file.Messages {
		generateMessage(g, m)
	}
	return g
}

func generateMessage(g *protogen.GeneratedFile, m *protogen.Message) {
	ident := g.QualifiedGoIdent(m.GoIdent)
	g.P("func (x *", ident, ") LogValue() ", g.QualifiedGoIdent(slogPkg.Ident("Value")), " {")
	g.P("if x == nil {")
	g.P("return ", g.QualifiedGoIdent(slogPkg.Ident("AnyValue(nil)")))
	g.P("}")

	g.P("attrs := make([]", g.QualifiedGoIdent(slogPkg.Ident("Attr")), ", 0, ", len(m.Fields), ")")

	// Process oneofs first to avoid duplicating fields
	oneofFields := make(map[*protogen.Oneof]bool)
	for _, f := range m.Fields {
		if f.Oneof != nil {
			oneofFields[f.Oneof] = true
		}
	}

	// Process regular fields
	for _, f := range m.Fields {
		if f.Oneof != nil {
			// Skip oneof fields here, they're handled separately
			continue
		}

		if debugRedact(f.Desc.Options().(*descriptorpb.FieldOptions)) {
			g.P("attrs = append(attrs, ", g.QualifiedGoIdent(slogPkg.Ident(`String("`)), f.Desc.Name(), `", "[REDACTED]"))`)
		} else if f.Desc.IsList() {
			generateListField(g, f)
		} else if f.Desc.IsMap() {
			generateMapField(g, f)
		} else {
			handleExplicitPresence(g, f, generatePrimitiveField)
		}
	}

	// Process oneofs
	for oneof := range oneofFields {
		generateOneofField(g, oneof)
	}

	g.P("return ", g.QualifiedGoIdent(slogPkg.Ident("GroupValue(attrs...)")))
	g.P("}")
	g.P()

	for _, submsg := range m.Messages {
		if submsg.Desc.IsMapEntry() {
			continue
		}
		generateMessage(g, submsg)
	}
}

func generateOneofField(g *protogen.GeneratedFile, oneof *protogen.Oneof) {
	// For each oneof field, generate a switch statement to handle the different types
	oneofName := oneof.GoName

	g.P("// Handle oneof field: ", oneofName)
	g.P("switch x.Which", oneofName, "() {")

	for _, f := range oneof.Fields {
		fieldName := f.Desc.Name()
		parentMsg := f.Parent.GoIdent
		g.P("case ", g.QualifiedGoIdent(parentMsg), "_", f.GoName, "_case:")

		if debugRedact(f.Desc.Options().(*descriptorpb.FieldOptions)) {
			g.P("attrs = append(attrs, ", g.QualifiedGoIdent(slogPkg.Ident(`String("`)), fieldName, `", "[REDACTED]"))`)
		} else {
			switch f.Desc.Kind() {
			case protoreflect.BoolKind:
				g.P("attrs = append(attrs, ", g.QualifiedGoIdent(slogPkg.Ident(`Bool("`)), fieldName, `", x.Get`, f.GoName, "()))")
			case protoreflect.BytesKind:
				g.P("attrs = append(attrs, ", g.QualifiedGoIdent(slogPkg.Ident(`Any("`)), fieldName, `", x.Get`, f.GoName, "()))")
			case protoreflect.DoubleKind:
				g.P("attrs = append(attrs, ", g.QualifiedGoIdent(slogPkg.Ident(`Float64("`)), fieldName, `", x.Get`, f.GoName, "()))")
			case protoreflect.EnumKind:
				g.P("attrs = append(attrs, ", g.QualifiedGoIdent(slogPkg.Ident(`String("`)), fieldName, `", x.Get`, f.GoName, "().String()))")
			case protoreflect.Fixed32Kind, protoreflect.Uint32Kind:
				g.P("attrs = append(attrs, ", g.QualifiedGoIdent(slogPkg.Ident(`Uint64("`)), fieldName, `", uint64(x.Get`, f.GoName, "())))")
			case protoreflect.Fixed64Kind, protoreflect.Uint64Kind:
				g.P("attrs = append(attrs, ", g.QualifiedGoIdent(slogPkg.Ident(`Uint64("`)), fieldName, `", x.Get`, f.GoName, "()))")
			case protoreflect.FloatKind:
				g.P("_fmt_", fieldName, " := ", g.QualifiedGoIdent(strconvPkg.Ident("FormatFloat(float64(x.Get")), f.GoName, "()), 'f', -1, 32)")
				g.P("_", fieldName, ", _ := ", g.QualifiedGoIdent(strconvPkg.Ident("ParseFloat(")), "_fmt_", fieldName, ", 64)")
				g.P("attrs = append(attrs, ", g.QualifiedGoIdent(slogPkg.Ident(`Float64("`)), fieldName, `", _`, fieldName, "))")
			case protoreflect.Int32Kind, protoreflect.Sfixed32Kind, protoreflect.Sint32Kind:
				g.P("attrs = append(attrs, ", g.QualifiedGoIdent(slogPkg.Ident(`Int64("`)), fieldName, `", int64(x.Get`, f.GoName, "())))")
			case protoreflect.Int64Kind, protoreflect.Sfixed64Kind, protoreflect.Sint64Kind:
				g.P("attrs = append(attrs, ", g.QualifiedGoIdent(slogPkg.Ident(`Int64("`)), fieldName, `", x.Get`, f.GoName, "()))")
			case protoreflect.GroupKind:
				g.P("attrs = append(attrs, ", g.QualifiedGoIdent(slogPkg.Ident(`Any("`)), fieldName, `", x.Get`, f.GoName, "()))")
			case protoreflect.MessageKind:
				g.P("if msgValue, ok := interface{}(x.Get", f.GoName, "()).(", g.QualifiedGoIdent(slogPkg.Ident("LogValuer")), "); ok {")
				g.P("attrs = append(attrs, ", g.QualifiedGoIdent(slogPkg.Ident(`Attr{Key: "`)), fieldName, `", Value: msgValue.LogValue()})`)
				g.P("} else {")
				g.P("attrs = append(attrs, ", g.QualifiedGoIdent(slogPkg.Ident(`Any("`)), fieldName, `", x.Get`, f.GoName, "()))")
				g.P("}")
			case protoreflect.StringKind:
				g.P("attrs = append(attrs, ", g.QualifiedGoIdent(slogPkg.Ident(`String("`)), fieldName, `", x.Get`, f.GoName, "()))")
			default:
				g.P("attrs = append(attrs, ", g.QualifiedGoIdent(slogPkg.Ident(`Any("`)), fieldName, `", x.Get`, f.GoName, "()))")
			}
		}
	}
	g.P("}")
}

func handleExplicitPresence(g *protogen.GeneratedFile, f *protogen.Field, generateFunc func(*protogen.GeneratedFile, *protogen.Field)) {
	// Omit the fields that are defined as `Explicit Presence` and the value is not present.
	// https://protobuf.dev/programming-guides/field_presence/#presence-in-proto3-apis
	switch {
	case f.Desc.HasOptionalKeyword():
		// Handle optional fields
		g.P("if x.Has", f.GoName, "() {")
		defer g.P("}")
	case f.Desc.Kind() == protoreflect.MessageKind || f.Desc.Kind() == protoreflect.GroupKind:
		// Handle message fields
		g.P("if x.Get", f.GoName, "() != nil {")
		defer g.P("}")
	}
	generateFunc(g, f)
}

func generateListField(g *protogen.GeneratedFile, f *protogen.Field) {
	// Use getters for field access to support Opaque API
	fname := f.Desc.Name()
	getterName := "Get" + f.GoName + "()"

	g.P("if len(x.", getterName, ") != 0 {")
	// len(x.FieldName) > 0
	attrs := fmt.Sprintf("attrs%d", f.Desc.Index())
	g.P(attrs, " := make([]", g.QualifiedGoIdent(slogPkg.Ident("Attr")), ", 0, len(x.", getterName, "))")
	g.P("for i, v := range x.", getterName, " {")
	switch f.Desc.Kind() {
	case protoreflect.BoolKind:
		g.P(attrs, " = append(", attrs, ", ", g.QualifiedGoIdent(slogPkg.Ident(`Bool(`)), fmtPkg.Ident("Sprintf"), `("%d", i), v))`)
	case protoreflect.BytesKind:
		g.P(attrs, " = append(", attrs, ", ", g.QualifiedGoIdent(slogPkg.Ident(`Any(`)), fmtPkg.Ident("Sprintf"), `("%d", i), v))`)
	case protoreflect.DoubleKind:
		g.P(attrs, " = append(", attrs, ", ", g.QualifiedGoIdent(slogPkg.Ident(`Float64(`)), fmtPkg.Ident("Sprintf"), `("%d", i), v))`)
	case protoreflect.EnumKind:
		g.P(attrs, " = append(", attrs, ", ", g.QualifiedGoIdent(slogPkg.Ident(`String(`)), fmtPkg.Ident("Sprintf"), `("%d", i), v.String()))`)
	case protoreflect.Fixed32Kind, protoreflect.Uint32Kind:
		g.P(attrs, " = append(", attrs, ", ", g.QualifiedGoIdent(slogPkg.Ident(`Uint64(`)), fmtPkg.Ident("Sprintf"), `("%d", i), uint64(v)))`)
	case protoreflect.Fixed64Kind, protoreflect.Uint64Kind:
		g.P(attrs, " = append(", attrs, ", ", g.QualifiedGoIdent(slogPkg.Ident(`Uint64(`)), fmtPkg.Ident("Sprintf"), `("%d", i), v))`)
	case protoreflect.FloatKind:
		g.P("_fmt_", fname, " := ", g.QualifiedGoIdent(strconvPkg.Ident("FormatFloat(float64(v), 'f', -1, 32)")))
		g.P("_", fname, ", _ := ", g.QualifiedGoIdent(strconvPkg.Ident("ParseFloat(")), "_fmt_", fname, ", 64)")
		g.P(attrs, " = append(", attrs, ", ", g.QualifiedGoIdent(slogPkg.Ident(`Float64(`)), fmtPkg.Ident("Sprintf"), `("%d", i), float64(_`, fname, `)))`)
	case protoreflect.Int32Kind, protoreflect.Sfixed32Kind, protoreflect.Sint32Kind:
		g.P(attrs, " = append(", attrs, ", ", g.QualifiedGoIdent(slogPkg.Ident(`Int64(`)), fmtPkg.Ident("Sprintf"), `("%d", i), int64(v)))`)
	case protoreflect.Int64Kind, protoreflect.Sfixed64Kind, protoreflect.Sint64Kind:
		g.P(attrs, " = append(", attrs, ", ", g.QualifiedGoIdent(slogPkg.Ident(`Int64(`)), fmtPkg.Ident("Sprintf"), `("%d", i), v))`)
	case protoreflect.GroupKind:
		g.P(attrs, " = append(", attrs, ", ", g.QualifiedGoIdent(slogPkg.Ident(`Any(`)), fmtPkg.Ident("Sprintf"), `("%d", i), v))`)
	case protoreflect.MessageKind:
		g.P("if v, ok := interface{}(v).(", g.QualifiedGoIdent(slogPkg.Ident("LogValuer")), "); ok {")
		g.P(attrs, " = append(", attrs, ", ", g.QualifiedGoIdent(slogPkg.Ident(`Attr{Key: `)), fmtPkg.Ident("Sprintf"), `("%d", i), Value: v.LogValue()})`)
		g.P("} else {")
		g.P(attrs, " = append(", attrs, ", ", g.QualifiedGoIdent(slogPkg.Ident(`Any(`)), fmtPkg.Ident("Sprintf"), `("%d", i), v))`)
		g.P("}")
	case protoreflect.StringKind:
		g.P(attrs, " = append(", attrs, ", ", g.QualifiedGoIdent(slogPkg.Ident(`String(`)), fmtPkg.Ident("Sprintf"), `("%d", i), v))`)
	default:
		g.P(attrs, " = append(", attrs, ", ", g.QualifiedGoIdent(slogPkg.Ident(`Any(`)), fmtPkg.Ident("Sprintf"), `("%d", i), v))`)
	}
	g.P("}")
	g.P("attrs = append(attrs, ", g.QualifiedGoIdent(slogPkg.Ident(`Any("`)), fname, `", `, attrs, "))")
	g.P("}")
}

func generateMapField(g *protogen.GeneratedFile, f *protogen.Field) {
	// Use getters for field access to support Opaque API
	fname := f.Desc.Name()
	getterName := "Get" + f.GoName + "()"

	g.P("if len(x.", getterName, ") != 0 {")
	// len(x.FieldName) > 0
	attrs := fmt.Sprintf("attrs%d", f.Desc.Index())
	g.P(attrs, " := make([]", g.QualifiedGoIdent(slogPkg.Ident("Attr")), ", 0, len(x.", getterName, "))")
	g.P("for k, v := range x.", getterName, " {")
	switch f.Desc.MapValue().Kind() {
	case protoreflect.BoolKind:
		g.P(attrs, " = append(", attrs, ", ", g.QualifiedGoIdent(slogPkg.Ident(`Bool(`)), fmtPkg.Ident("Sprintf"), `("%v", k), v))`)
	case protoreflect.BytesKind:
		g.P(attrs, " = append(", attrs, ", ", g.QualifiedGoIdent(slogPkg.Ident(`Any(`)), fmtPkg.Ident("Sprintf"), `("%v", k), v))`)
	case protoreflect.DoubleKind:
		g.P(attrs, " = append(", attrs, ", ", g.QualifiedGoIdent(slogPkg.Ident(`Float64(`)), fmtPkg.Ident("Sprintf"), `("%v", k), v))`)
	case protoreflect.EnumKind:
		g.P(attrs, " = append(", attrs, ", ", g.QualifiedGoIdent(slogPkg.Ident(`String(`)), fmtPkg.Ident("Sprintf"), `("%v", k), v.String()))`)
	case protoreflect.Fixed32Kind, protoreflect.Uint32Kind:
		g.P(attrs, " = append(", attrs, ", ", g.QualifiedGoIdent(slogPkg.Ident(`Uint64(`)), fmtPkg.Ident("Sprintf"), `("%v", k), uint64(v)))`)
	case protoreflect.Fixed64Kind, protoreflect.Uint64Kind:
		g.P(attrs, " = append(", attrs, ", ", g.QualifiedGoIdent(slogPkg.Ident(`Uint64(`)), fmtPkg.Ident("Sprintf"), `("%v", k), v))`)
	case protoreflect.FloatKind:
		g.P("_fmt_", fname, " := ", g.QualifiedGoIdent(strconvPkg.Ident("FormatFloat(float64(v), 'f', -1, 32)")))
		g.P("_", fname, ", _ := ", g.QualifiedGoIdent(strconvPkg.Ident("ParseFloat(")), "_fmt_", fname, ", 64)")
		g.P(attrs, " = append(", attrs, ", ", g.QualifiedGoIdent(slogPkg.Ident(`Float64(`)), fmtPkg.Ident("Sprintf"), `("%v", k), float64(_`, fname, `)))`)
	case protoreflect.Int32Kind, protoreflect.Sfixed32Kind, protoreflect.Sint32Kind:
		g.P(attrs, " = append(", attrs, ", ", g.QualifiedGoIdent(slogPkg.Ident(`Int64(`)), fmtPkg.Ident("Sprintf"), `("%v", k), int64(v)))`)
	case protoreflect.Int64Kind, protoreflect.Sfixed64Kind, protoreflect.Sint64Kind:
		g.P(attrs, " = append(", attrs, ", ", g.QualifiedGoIdent(slogPkg.Ident(`Int64(`)), fmtPkg.Ident("Sprintf"), `("%v", k), v))`)
	case protoreflect.GroupKind:
		g.P(attrs, " = append(", attrs, ", ", g.QualifiedGoIdent(slogPkg.Ident(`Any(`)), fmtPkg.Ident("Sprintf"), `("%v", k), v))`)
	case protoreflect.MessageKind:
		g.P("if vv, ok := interface{}(v).(", g.QualifiedGoIdent(slogPkg.Ident("LogValuer")), "); ok {")
		g.P(attrs, " = append(", attrs, ", ", g.QualifiedGoIdent(slogPkg.Ident(`Attr{Key: `)), fmtPkg.Ident("Sprintf"), `("%v", k), Value: vv.LogValue()})`)
		g.P("} else {")
		g.P(attrs, " = append(", attrs, ", ", g.QualifiedGoIdent(slogPkg.Ident(`Any(`)), fmtPkg.Ident("Sprintf"), `("%v", k), v))`)
		g.P("}")
	case protoreflect.StringKind:
		g.P(attrs, " = append(", attrs, ", ", g.QualifiedGoIdent(slogPkg.Ident(`String(`)), fmtPkg.Ident("Sprintf"), `("%v", k), v))`)
	default:
		g.P(attrs, " = append(", attrs, ", ", g.QualifiedGoIdent(slogPkg.Ident(`Any(`)), fmtPkg.Ident("Sprintf"), `("%v", k), v))`)
	}
	g.P("}")
	g.P("attrs = append(attrs, ", g.QualifiedGoIdent(slogPkg.Ident(`Any("`)), fname, `", `, attrs, "))")
	g.P("}")
}

func generatePrimitiveField(g *protogen.GeneratedFile, f *protogen.Field) {
	fname := f.Desc.Name()
	// Use getters for field access to support Opaque API
	getterName := "Get" + f.GoName + "()"

	switch f.Desc.Kind() {
	case protoreflect.BoolKind:
		g.P("attrs = append(attrs, ", g.QualifiedGoIdent(slogPkg.Ident(`Bool("`)), fname, `", x.`, getterName, "))")
	case protoreflect.BytesKind:
		g.P("attrs = append(attrs, ", g.QualifiedGoIdent(slogPkg.Ident(`Any("`)), fname, `", x.`, getterName, "))")
	case protoreflect.DoubleKind:
		g.P("attrs = append(attrs, ", g.QualifiedGoIdent(slogPkg.Ident(`Float64("`)), fname, `", x.`, getterName, "))")
	case protoreflect.EnumKind:
		g.P("attrs = append(attrs, ", g.QualifiedGoIdent(slogPkg.Ident(`String("`)), fname, `", x.`, getterName, ".String()))")
	case protoreflect.Fixed32Kind, protoreflect.Uint32Kind:
		g.P("attrs = append(attrs, ", g.QualifiedGoIdent(slogPkg.Ident(`Uint64("`)), fname, `", uint64(x.`, getterName, ")))")
	case protoreflect.Fixed64Kind, protoreflect.Uint64Kind:
		g.P("attrs = append(attrs, ", g.QualifiedGoIdent(slogPkg.Ident(`Uint64("`)), fname, `", x.`, getterName, "))")
	case protoreflect.FloatKind:
		g.P("_fmt_", fname, " := ", g.QualifiedGoIdent(strconvPkg.Ident("FormatFloat(float64(x.")), getterName, "), 'f', -1, 32)")
		g.P("_", fname, ", _ := ", g.QualifiedGoIdent(strconvPkg.Ident("ParseFloat(")), "_fmt_", fname, ", 64)")
		g.P("attrs = append(attrs, ", g.QualifiedGoIdent(slogPkg.Ident(`Float64("`)), fname, `", _`, fname, "))")
	case protoreflect.Int32Kind, protoreflect.Sfixed32Kind, protoreflect.Sint32Kind:
		g.P("attrs = append(attrs, ", g.QualifiedGoIdent(slogPkg.Ident(`Int64("`)), fname, `", int64(x.`, getterName, ")))")
	case protoreflect.Int64Kind, protoreflect.Sfixed64Kind, protoreflect.Sint64Kind:
		g.P("attrs = append(attrs, ", g.QualifiedGoIdent(slogPkg.Ident(`Int64("`)), fname, `", x.`, getterName, "))")
	case protoreflect.GroupKind:
		g.P("attrs = append(attrs, ", g.QualifiedGoIdent(slogPkg.Ident(`Any("`)), fname, `", x.`, getterName, "))")
	case protoreflect.MessageKind:
		g.P("if v, ok := interface{}(x.", getterName, ").(", g.QualifiedGoIdent(slogPkg.Ident("LogValuer")), "); ok {")
		g.P("attrs = append(attrs, ", g.QualifiedGoIdent(slogPkg.Ident(`Attr{Key: "`)), fname, `", Value: v.LogValue()})`)
		g.P("} else {")
		g.P("attrs = append(attrs, ", g.QualifiedGoIdent(slogPkg.Ident(`Any("`)), fname, `", x.`, getterName, "))")
		g.P("}")
	case protoreflect.StringKind:
		g.P("attrs = append(attrs, ", g.QualifiedGoIdent(slogPkg.Ident(`String("`)), fname, `", x.`, getterName, "))")
	default:
		g.P("attrs = append(attrs, ", g.QualifiedGoIdent(slogPkg.Ident(`Any("`)), fname, `", x.`, getterName, "))")
	}
}

func debugRedact(opts *descriptorpb.FieldOptions) bool {
	return opts.GetDebugRedact()
}
