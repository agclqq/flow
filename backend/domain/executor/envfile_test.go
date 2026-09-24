package executor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseEnvFile(t *testing.T) {
	data := `# comment
d=1
export e=2
f=

invalid-line
`
	env := ParseEnvFile([]byte(data))
	assert.Equal(t, "1", env["d"])
	assert.Equal(t, "2", env["e"])
	assert.Equal(t, "", env["f"])
	assert.NotContains(t, env, "invalid-line")
}
