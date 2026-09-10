package testutils

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithReturns_PhantomIsNotSerialised(t *testing.T) {
	got, err := json.Marshal(&WithReturns{Name: "n"})
	assert.NoError(t, err)
	assert.Equal(t, `{"Name":"n"}`, string(got), "f.Returns is a phantom, it must not appear in JSON")

	var back WithReturns
	err = json.Unmarshal([]byte(`{"Name":"n","Returns":{}}`), &back)
	assert.NoError(t, err)
	assert.Equal(t, WithReturns{Name: "n"}, back, "a stray Returns key is ignored")
}
