package parser

// xmlTagOverrides supplies XML element names for struct fields that lack xml struct tags.
//
// Some libvirtxml types use custom MarshalXML/UnmarshalXML to dispatch between element
// names that encoding/xml cannot express as struct tags (e.g. a slice interleaving <rule>
// and <filterref> elements). The reflector skips untagged fields; this table injects the
// missing name so the reflector can generate a model and schema for those fields.
//
// For mixed-content slices like NWFilter.Entries the tag value is synthetic: it only
// exists to prevent the field from being skipped. The actual XML round-trip is handled
// by the parent's custom marshaler, not generated code.
//
// Key: "StructName.FieldName", Value: XML element name.
var xmlTagOverrides = map[string]string{
	"NWFilter.Entries": "entry",    // synthetic; real dispatch in NWFilter.MarshalXML
	"NWFilterEntry.Rule": "rule",   // <rule> element
	"NWFilterEntry.Ref":  "filterref", // <filterref> element
}
