package events

import (
	"github.com/axelarnetwork/utils/test/rand"
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
)

func TestQueryBuilder(t *testing.T) {
	t.Run("build valid tx event query with module and attributes", func(t *testing.T) {
		q := NewTxEventQuery("testType").MatchModule("testModule").MatchAction("testAction").
			MatchAttributes(
				sdk.Attribute{Key: "attr1", Value: "attrValue1"},
				sdk.Attribute{Key: "attr2", Value: "attrValue2"},
				sdk.Attribute{Key: "attr3", Value: "attrValue3"},
				sdk.Attribute{Key: "attr4", Value: "attrValue4"}).Build()

		assert.Equal(t, "tm.event='Tx' AND testType.module='testModule' AND testType.action='testAction' "+
			"AND testType.attr1='attrValue1' AND testType.attr2='attrValue2' "+
			"AND testType.attr3='attrValue3' AND testType.attr4='attrValue4'", q.String())
	})
	t.Run("build valid tx event query with attribute set", func(t *testing.T) {
		key := "attr1"
		eventType := "testType"
		module := "testModule"
		attributeValues := []string{}
		maxCases := int64(20)
		for i := 0; i < int(rand.I64Between(1, maxCases)); i++ {
			attributeValues = append(attributeValues, "attrValue"+strconv.Itoa(i))
		}

		valueSet := NewAttributeValueSet(key, attributeValues...)
		q := QueryTxEventByValueSets(eventType, module, valueSet)

		validCases := []Event{}
		for i := 1; i < int(rand.I64Between(2, int64(len(attributeValues)))); i++ {
			validCases = append(validCases, Event{Type: eventType,
				Attributes: map[string]string{
					"module":     module,
					key:          "attrValue" + strconv.Itoa(i),
					rand.Str(10): rand.Str(10),
				},
				Height: 0,
			})
		}

		invalidCases := []Event{}
		for i := 1; i < int(rand.I64Between(2, maxCases)); i++ {
			invalidCases = append(invalidCases, Event{Type: eventType,
				Attributes: map[string]string{
					"module":     module,
					key:          rand.Str(10),
					rand.Str(10): rand.Str(10),
				},
				Height: 0,
			})
		}

		for _, event := range validCases {
			assert.True(t, q.Predicate(event))
		}

		for _, event := range invalidCases {
			assert.False(t, q.Predicate(event))
		}

		// empty value set
		q = QueryTxEventByValueSets(eventType, module, NewAttributeValueSet(""))
		for _, event := range append(validCases, invalidCases...) {
			assert.False(t, q.Predicate(event))
		}
	})
	t.Run("build valid block header event query with module and custom key/value matches", func(t *testing.T) {
		q := NewBlockHeaderEventQuery("testType").MatchModule("testModule").Match("someKey", "someValue").Build()
		assert.Equal(t, "tm.event='NewBlockHeader' AND testType.module='testModule' AND someKey='someValue'", q.String())
	})
}
