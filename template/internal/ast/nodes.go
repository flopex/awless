package ast

import (
	"fmt"
	"strings"
)

type CmdNode struct {
	action, entity string
	ParamNodes     map[string]interface{}
}

func (c CmdNode) String() string {
	var paramsStr []string
	for k, p := range c.ParamNodes {
		paramsStr = append(paramsStr, fmt.Sprintf("%s:%s", k, p))
	}
	return fmt.Sprintf("action:%s, entity:%s, params[%s]", c.action, c.entity, strings.Join(paramsStr, ","))
}

type CompiledCommand struct {
	Params map[string]interface{}
	Refs   map[string]string
}

type RefNode struct {
	key string
}

func (n RefNode) String() string {
	return "$" + n.key
}

type AliasNode struct {
	key string
}

func (n AliasNode) String() string {
	return "@" + n.key
}

type HoleNode struct {
	key string
}

func (n HoleNode) String() string {
	return "{" + n.key + "}"
}

type InterfaceNode struct {
	i interface{}
}
