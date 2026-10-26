package channel

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestOptionalString(t *testing.T) {
	if got := optionalString(""); !got.IsNull() {
		t.Errorf(`optionalString("") = %v, want null`, got)
	}
	if got := optionalString("rules"); !got.Equal(types.StringValue("rules")) {
		t.Errorf(`optionalString("rules") = %v, want "rules"`, got)
	}
}
