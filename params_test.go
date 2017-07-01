package task

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseCall(t *testing.T) {
	tests := []struct {
		source string
		call   Call
	}{
		{
			"^without-params",
			Call{Name: "without-params", Params: Params{}},
		},
		{
			"^with-params PARAM=value",
			Call{Name: "with-params", Params: Params{"PARAM": "value"}},
		},
		{
			"^empty-value PARAM= FOO=bar",
			Call{Name: "empty-value", Params: Params{"PARAM": "", "FOO": "bar"}},
		},
		{
			"^with-equal-sign PARAM=foo=bar",
			Call{Name: "with-equal-sign", Params: Params{"PARAM": "foo=bar"}},
		},
		// FIXME: need to handle if a param have a space in it
		// {
		// 	`^with-space param="foo bar baz"`,
		// 	Call{Name: "with-space", Params: Params{"param": "foo bar baz"}},
		// },
	}

	for _, test := range tests {
		call, err := ParseCall(test.source)
		assert.NoError(t, err)
		assert.Equal(t, test.call, *call)
	}
}
