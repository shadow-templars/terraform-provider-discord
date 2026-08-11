package datasource

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*ColorDataSource)(nil)

type ColorDataSource struct{}

type ColorDataSourceModel struct {
	Hex types.String `tfsdk:"hex"`
	Dec types.Int64  `tfsdk:"dec"`
}

func NewColorDataSource() datasource.DataSource {
	return &ColorDataSource{}
}

func (d *ColorDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_color"
}

func (d *ColorDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Converts a hex color code to a decimal integer for use in Discord role colors.",
		Attributes: map[string]schema.Attribute{
			"hex": schema.StringAttribute{
				Required:    true,
				Description: "The hex color code (e.g., `#5b2c8e` or `5b2c8e`).",
			},
			"dec": schema.Int64Attribute{
				Computed:    true,
				Description: "The decimal representation of the color.",
			},
		},
	}
}

func (d *ColorDataSource) Configure(_ context.Context, _ datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	// No client needed — this is a pure computation.
}

func (d *ColorDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ColorDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hex := strings.TrimPrefix(data.Hex.ValueString(), "#")
	dec, err := strconv.ParseInt(hex, 16, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Hex Color",
			fmt.Sprintf("Could not parse hex color %q: %s", data.Hex.ValueString(), err.Error()),
		)
		return
	}

	data.Dec = types.Int64Value(dec)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
