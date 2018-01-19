package ast

import (
	"errors"
	"fmt"
	"strings"
)

var _ ExpressionNode = (*RightExpressionNode)(nil)

type RightExpressionNode struct {
	i interface{}
}

func (n *RightExpressionNode) Node() interface{} {
	return n.i
}

func (n *RightExpressionNode) Result() interface{} {
	switch v := n.i.(type) {
	case InterfaceNode:
		return v.i
	default:
		return nil
	}
}

func (n *RightExpressionNode) Err() error {
	switch n.i.(type) {
	case InterfaceNode:
		return nil
	default:
		return errors.New("right expr node is not an interface node")
	}
}

func (n *RightExpressionNode) String() string {
	return fmt.Sprint(n.i)
}

func (n *RightExpressionNode) clone() Node {
	return &RightExpressionNode{
		i: n.i,
	}
}

type CompiledCommand struct {
	Params map[string]interface{}
	Refs   map[string]string
}

type CommandNode struct {
	Command
	CmdResult interface{}
	CmdErr    error

	Action, Entity string
	Params         map[string]CompositeValue
	ParamNodes     map[string]interface{}
}

type RefNode struct {
	key string
}

func (n RefNode) clone() Node {
	return n
}

func (n RefNode) String() string {
	return "$" + n.key
}

type AliasNode struct {
	key string
}

func NewAliasNode(s string) AliasNode {
	return AliasNode{key: s}
}

func (n AliasNode) clone() Node {
	return n
}

func (n AliasNode) String() string {
	return "@" + n.key
}

type HoleNode struct {
	key      string
	optional bool
}

func NewHoleNode(s string) HoleNode {
	return HoleNode{key: s}
}

func NewOptionalHoleNode(s string) HoleNode {
	return HoleNode{key: s, optional: true}
}

func (n HoleNode) IsOptional() bool {
	return n.optional
}

func (n HoleNode) String() string {
	return "{" + n.key + "}"
}

type ListNode struct {
	arr []interface{}
}

func (n ListNode) String() string {
	var a []string
	for _, e := range n.arr {
		a = append(a, fmt.Sprint(e))
	}
	return "[" + strings.Join(a, ",") + "]"
}

func (n ListNode) clone() Node {
	return n
}

type InterfaceNode struct {
	i interface{}
}

func (n InterfaceNode) String() string {
	switch v := n.i.(type) {
	case []string:
		return "[" + strings.Join(v, ",") + "]"
	case string:
		return quoteStringIfNeeded(v)
	default:
		return fmt.Sprint(v)
	}
}

func (n InterfaceNode) clone() Node {
	return n
}
