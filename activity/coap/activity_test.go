package coap

import (
	"testing"

	"github.com/project-flogo/core/activity"
	"github.com/stretchr/testify/assert"
)

func TestRegister(t *testing.T) {

	ref := activity.GetRef(&CoAPActivity{})
	act := activity.Get(ref)

	assert.NotNil(t, act)
}
