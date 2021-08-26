package types

import (
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/libs/pubsub"
	"github.com/tendermint/tendermint/libs/pubsub/query"
	tm "github.com/tendermint/tendermint/types"
)

func eventKey(eventType, attrKey string) string {
	return fmt.Sprintf("%s.%s", eventType, attrKey)
}

var EventKeyAction = eventKey(sdk.EventTypeMessage, sdk.AttributeKeyAction)
var EventKeyModule = eventKey(sdk.EventTypeMessage, sdk.AttributeKeyModule)

func QueryKeyValue(key, value string) pubsub.Query {
	return query.MustParse(fmt.Sprintf("%s='%s'", key, value))
}

func MakeModuleTxQuery(eventType string, module string) pubsub.Query {
	qString := fmt.Sprintf("%s='%s' AND %s.%s='%s'",
		tm.EventTypeKey, tm.EventTx, eventType, sdk.AttributeKeyModule, module)
	return query.MustParse(qString)
}

func MakeTxEventQuery(eventType string, attributes ...Param) pubsub.Query {
	qString := fmt.Sprintf("%s='%s'", tm.EventTypeKey, tm.EventTx)
	params := []string{qString}
	for _, attr := range attributes {
		params = append(params, fmt.Sprintf("%s.%s='%s'", eventType, attr.Key, attr.Value))
	}
	qString = strings.Join(params, " AND ")

	return query.MustParse(qString)
}

func MakeEventQuery(eventType string, attributes ...Param) pubsub.Query {
	params := []string{}
	for _, attr := range attributes {
		params = append(params, fmt.Sprintf("%s.%s='%s'", eventType, attr.Key, attr.Value))
	}

	return query.MustParse(strings.Join(params, " AND "))
}

func MsgQueryAction(action string) pubsub.Query {
	return query.MustParse(
		fmt.Sprintf("%s='%s'", EventKeyAction, action))
}

func EventQueryTxFor(txHash string) pubsub.Query {
	return query.MustParse(fmt.Sprintf("%s='%s' AND %s='%X'", tm.EventTypeKey, tm.EventTx, tm.TxHashKey, txHash))
}
