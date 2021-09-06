package events

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
)

func TestQueryBuilder(t *testing.T) {
	t.Run("build valid tx event query with module and attributes", func(t *testing.T) {
		q := MatchTxEvent("testType").MatchModule("testModule").MatchAction("testAction").
			MatchAttributes(
				sdk.Attribute{Key: "attr1", Value: "attrValue1"},
				sdk.Attribute{Key: "attr2", Value: "attrValue2"},
				sdk.Attribute{Key: "attr3", Value: "attrValue3"},
				sdk.Attribute{Key: "attr4", Value: "attrValue4"}).Build()

		assert.Equal(t, "tm.event='Tx' AND testType.module='testModule' AND testType.action='testAction' "+
			"AND testType.attr1='attrValue1' AND testType.attr2='attrValue2' "+
			"AND testType.attr3='attrValue3' AND testType.attr4='attrValue4'", q.String())
	})
	t.Run("build valid block header event query with module and custom key/value matches", func(t *testing.T) {
		q := MatchBlockHeaderEvent("testType").MatchModule("testModule").Match("someKey", "someValue").Build()
		assert.Equal(t, "tm.event='NewBlockHeader' AND testType.module='testModule' AND someKey='someValue'", q.String())
	})
}
