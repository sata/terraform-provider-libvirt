package provider

import (
	"context"
	"fmt"
	"strings"

	golibvirt "github.com/digitalocean/go-libvirt"
	"github.com/dmacvicar/terraform-provider-libvirt/v2/internal/generated"
	libvirtclient "github.com/dmacvicar/terraform-provider-libvirt/v2/internal/libvirt"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"libvirt.org/go/libvirtxml"
)

// NWFilterResource implements the libvirt_nwfilter resource.
type NWFilterResource struct {
	client *libvirtclient.Client
}

// NWFilterResourceModel extends the generated model with provider-managed fields.
type NWFilterResourceModel struct {
	generated.NWFilterModel
	ID types.String `tfsdk:"id"`
}

func NewNWFilterResource() resource.Resource {
	return &NWFilterResource{}
}

func (r *NWFilterResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_nwfilter"
}

func (r *NWFilterResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = generated.NWFilterSchema(map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Description: "Network filter identifier (UUID)",
			Computed:    true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
	})
}

func (r *NWFilterResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*libvirtclient.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *libvirt.Client, got: %T", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *NWFilterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model NWFilterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	filterXML, err := generated.NWFilterToXML(ctx, &model.NWFilterModel)
	if err != nil {
		resp.Diagnostics.AddError("Model to XML Conversion Failed",
			fmt.Sprintf("Failed to convert model to XML: %s", err))
		return
	}

	xmlDoc, err := filterXML.Marshal()
	if err != nil {
		resp.Diagnostics.AddError("XML Marshaling Failed",
			fmt.Sprintf("Failed to marshal nwfilter XML: %s", err))
		return
	}

	tflog.Debug(ctx, "Generated nwfilter XML", map[string]any{"xml": xmlDoc})

	filter, err := r.client.Libvirt().NwfilterDefineXML(xmlDoc)
	if err != nil {
		resp.Diagnostics.AddError("NWFilter Creation Failed",
			fmt.Sprintf("Failed to define nwfilter: %s", err))
		return
	}

	uuidStr := libvirtclient.UUIDString(filter.UUID)
	model.ID = types.StringValue(uuidStr)

	tflog.Info(ctx, "Created nwfilter", map[string]any{
		"uuid": uuidStr,
		"name": model.Name.ValueString(),
	})

	planModel := model.NWFilterModel
	if err := r.readNWFilter(ctx, &model, filter, &planModel); err != nil {
		resp.Diagnostics.AddError("NWFilter Read Failed",
			fmt.Sprintf("NWFilter created but failed to read back: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *NWFilterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model NWFilterResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	filter, err := r.client.LookupNWFilterByUUID(model.ID.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "NWFilter not found") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("NWFilter Lookup Failed",
			fmt.Sprintf("Failed to find nwfilter: %s", err))
		return
	}

	if err := r.readNWFilter(ctx, &model, filter, &model.NWFilterModel); err != nil {
		resp.Diagnostics.AddError("NWFilter Read Failed",
			fmt.Sprintf("Failed to read nwfilter: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *NWFilterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var model NWFilterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state NWFilterResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	model.ID = state.ID

	filterXML, err := generated.NWFilterToXML(ctx, &model.NWFilterModel)
	if err != nil {
		resp.Diagnostics.AddError("Model to XML Conversion Failed",
			fmt.Sprintf("Failed to convert model to XML: %s", err))
		return
	}

	// Preserve UUID so libvirt replaces the existing definition.
	filterXML.UUID = state.UUID.ValueString()

	xmlDoc, err := filterXML.Marshal()
	if err != nil {
		resp.Diagnostics.AddError("XML Marshaling Failed",
			fmt.Sprintf("Failed to marshal nwfilter XML: %s", err))
		return
	}

	tflog.Debug(ctx, "Updating nwfilter XML", map[string]any{"xml": xmlDoc})

	filter, err := r.client.Libvirt().NwfilterDefineXML(xmlDoc)
	if err != nil {
		resp.Diagnostics.AddError("NWFilter Update Failed",
			fmt.Sprintf("Failed to redefine nwfilter: %s", err))
		return
	}

	planModel := model.NWFilterModel
	if err := r.readNWFilter(ctx, &model, filter, &planModel); err != nil {
		resp.Diagnostics.AddError("NWFilter Read Failed",
			fmt.Sprintf("NWFilter updated but failed to read back: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *NWFilterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model NWFilterResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	filter, err := r.client.LookupNWFilterByUUID(model.ID.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "NWFilter not found") {
			return
		}
		resp.Diagnostics.AddError("NWFilter Lookup Failed",
			fmt.Sprintf("Failed to find nwfilter for deletion: %s", err))
		return
	}

	if err := r.client.Libvirt().NwfilterUndefine(filter); err != nil {
		resp.Diagnostics.AddError("NWFilter Delete Failed",
			fmt.Sprintf("Failed to undefine nwfilter: %s", err))
		return
	}

	tflog.Info(ctx, "Deleted nwfilter", map[string]any{
		"uuid": model.ID.ValueString(),
	})
}

func (r *NWFilterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import by name: look up by name, set id from UUID, then normal Read takes over.
	filter, err := r.client.Libvirt().NwfilterLookupByName(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("NWFilter Import Failed",
			fmt.Sprintf("Failed to find nwfilter %q: %s", req.ID, err))
		return
	}

	uuidStr := libvirtclient.UUIDString(filter.UUID)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), uuidStr)...)
}

// readNWFilter reads nwfilter state from libvirt and populates the model.
// plan is nil during import (populate everything); non-nil otherwise (preserve user intent).
func (r *NWFilterResource) readNWFilter(ctx context.Context, model *NWFilterResourceModel, filter golibvirt.Nwfilter, plan *generated.NWFilterModel) error {
	xmlDoc, err := r.client.Libvirt().NwfilterGetXMLDesc(filter, 0)
	if err != nil {
		return fmt.Errorf("failed to get nwfilter XML: %w", err)
	}

	var filterXML libvirtxml.NWFilter
	if err := filterXML.Unmarshal(xmlDoc); err != nil {
		return fmt.Errorf("failed to parse nwfilter XML: %w", err)
	}

	filterModel, err := generated.NWFilterFromXML(ctx, &filterXML, plan)
	if err != nil {
		return fmt.Errorf("failed to convert XML to model: %w", err)
	}

	model.NWFilterModel = *filterModel
	return nil
}
