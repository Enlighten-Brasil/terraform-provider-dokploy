package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/ahmedali6/terraform-provider-dokploy/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &LibsqlResource{}
var _ resource.ResourceWithImportState = &LibsqlResource{}

func NewLibsqlResource() resource.Resource {
	return &LibsqlResource{}
}

type LibsqlResource struct {
	client *client.DokployClient
}

type LibsqlResourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	AppNamePrefix     types.String `tfsdk:"app_name_prefix"`
	AppName           types.String `tfsdk:"app_name"`
	Description       types.String `tfsdk:"description"`
	DatabaseUser      types.String `tfsdk:"database_user"`
	DatabasePassword  types.String `tfsdk:"database_password"`
	SqldNode          types.String `tfsdk:"sqld_node"`
	SqldPrimaryUrl    types.String `tfsdk:"sqld_primary_url"`
	DockerImage       types.String `tfsdk:"docker_image"`
	EnvironmentID     types.String `tfsdk:"environment_id"`
	ApplicationStatus types.String `tfsdk:"application_status"`
	Replicas          types.Int64  `tfsdk:"replicas"`
	ServerID          types.String `tfsdk:"server_id"`
	DeployOnCreate    types.Bool   `tfsdk:"deploy_on_create"`
}

func (r *LibsqlResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_libsql"
}

func (r *LibsqlResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a LibSQL (sqld) database instance in Dokploy (>= 0.29.0).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the LibSQL instance.",
			},
			"app_name_prefix": schema.StringAttribute{
				Required:    true,
				Description: "Application name prefix; Dokploy appends a random suffix.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"app_name": schema.StringAttribute{
				Computed:    true,
				Description: "Full application name (prefix + server suffix).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "Description of the instance.",
			},
			"database_user": schema.StringAttribute{
				Optional:    true,
				Description: "Database user name.",
			},
			"database_password": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "Password for the database user.",
			},
			"sqld_node": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("primary"),
				Description: "sqld node role: primary or replica.",
			},
			"sqld_primary_url": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "Primary URL for replica nodes (empty for primary).",
			},
			"docker_image": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Docker image for the instance.",
			},
			"environment_id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the environment to deploy the LibSQL instance in.",
			},
			"application_status": schema.StringAttribute{
				Computed:    true,
				Description: "Current status of the instance: idle, running, done, error.",
			},
			"replicas": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Number of replicas.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"server_id": schema.StringAttribute{
				Optional:    true,
				Description: "ID of the server to deploy the LibSQL instance on.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"deploy_on_create": schema.BoolAttribute{
				Optional:    true,
				Description: "Trigger a deployment after creating the instance (libsql.deploy).",
			},
		},
	}
}

func (r *LibsqlResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*client.DokployClient)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Type", fmt.Sprintf("Expected *client.DokployClient, got: %T", req.ProviderData))
		return
	}
	r.client = client
}

func (r *LibsqlResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan LibsqlResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	libsql := client.Libsql{
		Name:             plan.Name.ValueString(),
		AppName:          plan.AppNamePrefix.ValueString(),
		Description:      plan.Description.ValueString(),
		DatabaseUser:     plan.DatabaseUser.ValueString(),
		DatabasePassword: plan.DatabasePassword.ValueString(),
		SqldNode:         plan.SqldNode.ValueString(),
		SqldPrimaryUrl:   plan.SqldPrimaryUrl.ValueString(),
		DockerImage:      plan.DockerImage.ValueString(),
		EnvironmentID:    plan.EnvironmentID.ValueString(),
		ServerID:         plan.ServerID.ValueString(),
	}

	created, err := r.client.CreateLibsql(libsql)
	if err != nil {
		resp.Diagnostics.AddError("Error creating LibSQL instance", err.Error())
		return
	}

	// Preserve the configured app_name prefix (server appends a suffix).
	prefix := plan.AppNamePrefix
	r.mapLibsqlToState(&plan, created)
	plan.AppNamePrefix = prefix

	// Deploy if requested
	if !plan.DeployOnCreate.IsNull() && plan.DeployOnCreate.ValueBool() {
		if err := r.client.DeployLibsql(plan.ID.ValueString()); err != nil {
			resp.Diagnostics.AddWarning("Deployment Trigger Failed", fmt.Sprintf("Instance created but deployment failed to trigger: %s", err.Error()))
		}
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *LibsqlResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state LibsqlResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	libsql, err := r.client.GetLibsql(state.ID.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "Not Found") || strings.Contains(err.Error(), "404") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading LibSQL instance", err.Error())
		return
	}

	prefix := state.AppNamePrefix
	r.mapLibsqlToState(&state, libsql)
	if !prefix.IsNull() && !prefix.IsUnknown() {
		state.AppNamePrefix = prefix
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

func (r *LibsqlResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan LibsqlResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	update := client.Libsql{
		LibsqlID:         plan.ID.ValueString(),
		Name:             plan.Name.ValueString(),
		Description:      plan.Description.ValueString(),
		DatabaseUser:     plan.DatabaseUser.ValueString(),
		DatabasePassword: plan.DatabasePassword.ValueString(),
		SqldNode:         plan.SqldNode.ValueString(),
		SqldPrimaryUrl:   plan.SqldPrimaryUrl.ValueString(),
		DockerImage:      plan.DockerImage.ValueString(),
	}

	updated, err := r.client.UpdateLibsql(update)
	if err != nil {
		resp.Diagnostics.AddError("Error updating LibSQL instance", err.Error())
		return
	}

	prefix := plan.AppNamePrefix
	r.mapLibsqlToState(&plan, updated)
	plan.AppNamePrefix = prefix

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *LibsqlResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state LibsqlResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteLibsql(state.ID.ValueString())
	if err != nil {
		errStr := strings.ToLower(err.Error())
		if strings.Contains(errStr, "not found") || strings.Contains(errStr, "not_found") || strings.Contains(errStr, "404") {
			return
		}
		resp.Diagnostics.AddError("Error deleting LibSQL instance", err.Error())
		return
	}
}

func (r *LibsqlResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *LibsqlResource) mapLibsqlToState(state *LibsqlResourceModel, libsql *client.Libsql) {
	state.ID = types.StringValue(libsql.LibsqlID)
	state.Name = types.StringValue(libsql.Name)
	state.AppName = types.StringValue(libsql.AppName)
	state.Description = types.StringValue(libsql.Description)
	state.DatabaseUser = types.StringValue(libsql.DatabaseUser)
	state.SqldNode = types.StringValue(libsql.SqldNode)
	state.SqldPrimaryUrl = types.StringValue(libsql.SqldPrimaryUrl)
	if libsql.DockerImage != "" {
		state.DockerImage = types.StringValue(libsql.DockerImage)
	}
	state.EnvironmentID = types.StringValue(libsql.EnvironmentID)
	state.ApplicationStatus = types.StringValue(libsql.ApplicationStatus)
	if libsql.Replicas > 0 {
		state.Replicas = types.Int64Value(int64(libsql.Replicas))
	}
	if libsql.ServerID != "" {
		state.ServerID = types.StringValue(libsql.ServerID)
	}
}
