package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/cloudsigma/terraform-provider-cloudsigma/internal/provider/migration"
)

var (
	_ datasource.DataSource              = (*tagDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*tagDataSource)(nil)
)

// tagDataSource is the tag data source implementation.
type tagDataSource struct {
	client *cloudsigma.Client
}

// tagDataSourceModel maps the tag data source schema data.
type tagDataSourceModel struct {
	Filters     []migration.FilterModel `tfsdk:"filter"`
	ID          types.String            `tfsdk:"id"`
	Name        types.String            `tfsdk:"name"`
	ResourceURI types.String            `tfsdk:"resource_uri"`
	UUID        types.String            `tfsdk:"uuid"`
}

func NewTagDataSource() datasource.DataSource {
	return &tagDataSource{}
}

func (d *tagDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = "cloudsigma_tag"
}

func (d *tagDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: `
The tag data source provides information about an existing CloudSigma tag.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the tag.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the tag.",
				Computed:            true,
				Optional:            true,
			},
			"resource_uri": schema.StringAttribute{
				MarkdownDescription: "The unique resource identifier of the tag.",
				Computed:            true,
			},
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The unique universal identifier of the current tag, equal to ID.",
				Computed:            true,
				Optional:            true,
			},
		},
		Blocks: map[string]schema.Block{
			"filter": schema.SetNestedBlock{
				MarkdownDescription: "One or more name/value pairs to filter off of.",
				DeprecationMessage:  `Configure "uuid" or "name" instead. The "filter" block will be removed in a future version of the provider.`,
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "The name of the attribute to filter.",
							Optional:            true,
						},
						"values": schema.ListAttribute{
							MarkdownDescription: "The value of the attribute to filter.",
							ElementType:         types.StringType,
							Optional:            true,
						},
					},
				},
			},
		},
	}
}

func (d *tagDataSource) Configure(_ context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
	if request.ProviderData == nil {
		return
	}

	client, ok := request.ProviderData.(*cloudsigma.Client)
	if !ok {
		response.Diagnostics.AddError(
			"Unconfigured CloudSigma client",
			"Please report this issue to the provider developers.",
		)
		return
	}

	d.client = client
}

func (d *tagDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var data tagDataSourceModel

	// read state data into the model
	diags := request.Config.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	if data.Filters != nil && len(data.Filters) > 0 {
		// this logic belongs to deprecated filter block and should be removed after breaking change release
		tflog.Warn(ctx, "Using legacy filter block", map[string]interface{}{"filters_count": len(data.Filters)})

		tflog.Trace(ctx, "Getting tags")
		tags, _, err := d.client.Tags.List(ctx)
		if err != nil {
			response.Diagnostics.AddError("Unable to get tags", err.Error())
			return
		}
		tflog.Trace(ctx, "Got tags", map[string]interface{}{"tags_count": len(tags)})

		tflog.Debug(ctx, "Converting tags for filtering")
		tagsForFilters := migration.ToAnySlice(tags)
		tflog.Debug(ctx, "Converted tags for filtering", map[string]interface{}{"tags": tagsForFilters})
		filteredTags, diags := migration.ApplyFilter(data.Filters, tagsForFilters)
		if diags != nil {
			response.Diagnostics.Append(diags)
			return
		}

		if len(filteredTags) > 1 {
			response.Diagnostics.AddError(
				"Too many search results",
				fmt.Sprintf("Please refine your search to be more specific. Found %v tags.", len(filteredTags)),
			)
			return
		}
		if len(filteredTags) < 1 {
			response.Diagnostics.AddError("No search results", "Please refine your search.")
			return
		}

		tag := filteredTags[0].(cloudsigma.Tag)

		data.ID = types.StringValue(tag.UUID)
		data.Name = types.StringValue(tag.Name)
		data.ResourceURI = types.StringValue(tag.ResourceURI)
		data.UUID = types.StringValue(tag.UUID)
	} else {
		tagUUID := data.UUID.ValueString()
		tagName := data.Name.ValueString()

		if tagUUID == "" && tagName == "" {
			response.Diagnostics.AddError(
				"Missing required attributes",
				`The attribute "name" or "uuid" must be defined.`,
			)
			return
		}

		if tagUUID != "" {
			tflog.Trace(ctx, "Getting tag using UUID", map[string]interface{}{"tag_uuid": tagUUID})
			tag, _, err := d.client.Tags.Get(ctx, tagUUID)
			if err != nil {
				response.Diagnostics.AddError("Unable to get tag", err.Error())
				return
			}
			tflog.Trace(ctx, "Got tag", map[string]interface{}{"data": tag})

			// if name is defined check that it's equal
			if tagName != "" && tagName != tag.Name {
				response.Diagnostics.AddError(
					"Ambiguous search result",
					fmt.Sprintf("Specified and actual tag name are different. Expected '%s', got '%s'", tagName, tag.Name),
				)
				return
			}

			data.ID = types.StringValue(tag.UUID)
			data.Name = types.StringValue(tag.Name)
			data.ResourceURI = types.StringValue(tag.ResourceURI)
			data.UUID = types.StringValue(tag.UUID)
		} else {
			tflog.Trace(ctx, "Getting tags for filtering", map[string]interface{}{"tag_name": tagName})
			tags, _, err := d.client.Tags.List(ctx)
			if err != nil {
				response.Diagnostics.AddError("Unable to get tags", err.Error())
				return
			}
			tflog.Trace(ctx, "Got tags", map[string]interface{}{"data": tags})

			tagFound := false
			for _, tag := range tags {
				if tagName == tag.Name {
					data.ID = types.StringValue(tag.UUID)
					data.Name = types.StringValue(tag.Name)
					data.ResourceURI = types.StringValue(tag.ResourceURI)
					data.UUID = types.StringValue(tag.UUID)

					tagFound = true
					break
				}
			}

			if !tagFound {
				response.Diagnostics.AddError("No search results", "Please refine your search.")
				return
			}
		}
	}

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}
