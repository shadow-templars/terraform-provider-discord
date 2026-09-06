package datasource

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*LocalImageDataSource)(nil)

type LocalImageDataSource struct{}

type LocalImageDataSourceModel struct {
	File    types.String `tfsdk:"file"`
	DataURI types.String `tfsdk:"data_uri"`
}

func NewLocalImageDataSource() datasource.DataSource {
	return &LocalImageDataSource{}
}

func (d *LocalImageDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_local_image"
}

func (d *LocalImageDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads a local image file and encodes it as a data URI for use with Discord's API.",
		Attributes: map[string]schema.Attribute{
			"file": schema.StringAttribute{
				Required:    true,
				Description: "The path to the local image file.",
			},
			"data_uri": schema.StringAttribute{
				Computed:    true,
				Description: "The base64-encoded data URI of the image (e.g., `data:image/png;base64,...`).",
			},
		},
	}
}

func (d *LocalImageDataSource) Configure(_ context.Context, _ datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
}

func (d *LocalImageDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data LocalImageDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	filePath := data.File.ValueString()

	content, err := os.ReadFile(filePath)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Image File",
			fmt.Sprintf("Could not read file %q: %s", filePath, err.Error()),
		)
		return
	}

	// Detect MIME type from content.
	mimeType := http.DetectContentType(content)

	// Support common image extensions as fallback.
	switch filepath.Ext(filePath) {
	case ".gif":
		mimeType = "image/gif"
	case ".png":
		mimeType = "image/png"
	case ".jpg", ".jpeg":
		mimeType = "image/jpeg"
	case ".webp":
		mimeType = "image/webp"
	}

	encoded := base64.StdEncoding.EncodeToString(content)
	dataURI := fmt.Sprintf("data:%s;base64,%s", mimeType, encoded)

	data.DataURI = types.StringValue(dataURI)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
