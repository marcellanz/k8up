package cfg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Configuration_ValidateSyntax(t *testing.T) {
	type assertConfig = func(t *testing.T, c *Configuration)
	tests := map[string]struct {
		givenConfig       *Configuration
		assertConfig      assertConfig
		expectErr         bool
		containErrMessage string
	}{
		"GivenOperatorNamespace_WhenEmptyValue_ThenExpectError": {
			givenConfig: &Configuration{
				OperatorNamespace: "",
			},
			expectErr:         true,
			containErrMessage: "operator namespace",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := tt.givenConfig.ValidateSyntax()
			if tt.expectErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.containErrMessage)
				return
			}
			assert.NoError(t, err)
			tt.assertConfig(t, tt.givenConfig)
		})
	}
}

func Test_Configuration_DefaultConfig(t *testing.T) {
	c := NewDefaultConfig()
	err := c.ValidateSyntax()
	require.NoError(t, err)
}
