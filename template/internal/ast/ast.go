/*
Copyright 2017 WALLIX

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ast

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/wallix/awless/template/env"
	"github.com/wallix/awless/template/params"
)

var (
	_ WithHoles = (*CommandNode)(nil)
	_ WithHoles = (*ValueNode)(nil)
)

type Node interface {
	clone() Node
	String() string
}

type AST struct {
	Statements []*Statement

	// state to build the AST
	stmtBuilder *statementBuilder
}

type Statement struct {
	Node
}

type DeclarationNode struct {
	Ident string
	Expr  ExpressionNode
}

type ExpressionNode interface {
	Node
	Result() interface{}
	Err() error
}

type Hole struct {
	Name       string
	ParamPaths []string
	IsOptional bool
}

type WithHoles interface {
	ProcessHoles(fills map[string]interface{}) (processed map[string]interface{})
	GetHoles() map[string]*Hole
}

type Command interface {
	ParamsSpec() params.Spec
	Run(env.Running, map[string]interface{}) (interface{}, error)
}

func (c *CommandNode) Result() interface{} { return c.CmdResult }
func (c *CommandNode) Err() error          { return c.CmdErr }

func (c *CommandNode) Keys() (keys []string) {
	for k := range c.Params {
		keys = append(keys, k)
	}
	return
}

func (c *CommandNode) String() string {
	var all []string

	for k, v := range c.ParamNodes {
		all = append(all, fmt.Sprintf("%s=%v", k, v))
	}

	sort.Strings(all)

	var buff bytes.Buffer

	fmt.Fprintf(&buff, "%s %s", c.Action, c.Entity)

	if len(all) > 0 {
		fmt.Fprintf(&buff, " %s", strings.Join(all, " "))
	}

	return buff.String()
}

func (c *CommandNode) clone() Node {
	cmd := &CommandNode{
		Command: c.Command,
		Action:  c.Action, Entity: c.Entity,
		Params:     make(map[string]CompositeValue),
		ParamNodes: make(map[string]interface{}),
	}

	for k, v := range c.Params {
		cmd.Params[k] = v.Clone()
	}
	for k, v := range c.ParamNodes {
		cmd.ParamNodes[k] = v
	}
	return cmd
}

func (c *CommandNode) ProcessHoles(fills map[string]interface{}) map[string]interface{} {
	processed := make(map[string]interface{})

	for _, param := range c.Params {
		if withHoles, ok := param.(WithHoles); ok {
			paramProcessed := withHoles.ProcessHoles(fills)
			for k, v := range paramProcessed {
				processed[k] = v
			}
		}
	}

	for paramKey, param := range c.ParamNodes {
		if hole, ok := param.(HoleNode); ok {
			for k, v := range fills {
				if k == hole.key {
					c.ParamNodes[paramKey] = v
					processed[k] = v
				}
			}
		}

		if list, ok := param.(ListNode); ok {
			var new []interface{}
			for _, e := range list.arr {
				newElem := e
				if hole, isHole := e.(HoleNode); isHole {
					for k, v := range fills {
						if k == hole.key {
							newElem = v
							processed[k] = v
						}
					}
				}
				new = append(new, newElem)
			}
			list.arr = new
		}
	}

	return processed
}

func (c *CommandNode) GetHoles() map[string]*Hole {
	holes := make(map[string]*Hole)
	for paramKey, param := range c.Params {
		if withHoles, ok := param.(WithHoles); ok {
			for k, v := range withHoles.GetHoles() {
				if _, ok := holes[k]; !ok {
					holes[k] = v
				}
				holes[k].ParamPaths = append(holes[k].ParamPaths, strings.Join([]string{c.Action, c.Entity, paramKey}, "."))
			}

		}
	}
	return holes
}

func (c *CommandNode) ProcessRefs(refs map[string]interface{}) {
	for _, param := range c.Params {
		if withRef, ok := param.(WithRefs); ok {
			withRef.ProcessRefs(refs)
		}
	}

	for paramKey, param := range c.ParamNodes {
		if ref, ok := param.(RefNode); ok {
			for k, v := range refs {
				if k == ref.key {
					c.ParamNodes[paramKey] = v
				}
			}
		}

		if list, ok := param.(ListNode); ok {
			var new []interface{}
			for _, e := range list.arr {
				newElem := e
				if ref, isRef := e.(RefNode); isRef {
					for k, v := range refs {
						if k == ref.key {
							newElem = v
						}
					}
				}
				new = append(new, newElem)
			}
			list.arr = new
		}
	}
}

func (c *CommandNode) GetRefs() (refs []string) {
	for _, param := range c.Params {
		if withRef, ok := param.(WithRefs); ok {
			refs = append(refs, withRef.GetRefs()...)
		}
	}
	return
}

func (c *CommandNode) ReplaceRef(key string, value CompositeValue) {
	for k, param := range c.Params {
		if withRef, ok := param.(WithRefs); ok {
			if withRef.IsRef(key) {
				c.Params[k] = value
			} else {
				withRef.ReplaceRef(key, value)
			}
		}
	}
}

func (c *CommandNode) IsRef(key string) bool {
	return false
}

func (c *CommandNode) ToDriverParams() map[string]interface{} {
	params := make(map[string]interface{})
	for k, v := range c.ParamNodes {
		switch node := v.(type) {
		case InterfaceNode:
			params[k] = node.i
		case RefNode, HoleNode, AliasNode:
		default:
			params[k] = node
		}
	}
	return params
}

func (c *CommandNode) ToDriverParamsExcludingRefs() map[string]interface{} {
	params := make(map[string]interface{})
	for k, v := range c.Params {
		if _, ok := v.(WithRefs); ok {
			continue
		}
		if v.Value() != nil {
			params[k] = v.Value()
		}
	}
	return params
}

func (c *CommandNode) ToFillerParams() map[string]interface{} {
	params := make(map[string]interface{})
	fn := func(k string, v interface{}) interface{} {
		switch vv := v.(type) {
		case InterfaceNode:
			return vv.i
		case AliasNode:
			return v
		}
		return nil
	}

	for k, v := range c.ParamNodes {
		i := fn(k, v)
		if i != nil {
			params[k] = i
			continue
		}
		switch vv := v.(type) {
		case ListNode:
			var arr []interface{}
			for _, a := range vv.arr {
				arr = append(arr, fn(k, a))
			}
			params[k] = arr
		}
	}
	return params
}

type ValueNode struct {
	Value CompositeValue
}

func (n *ValueNode) clone() Node {
	return &ValueNode{
		Value: n.Value.Clone(),
	}
}

func (n *ValueNode) String() string {
	return n.Value.String()
}

func (n *ValueNode) Result() interface{} { return n.Value }
func (n *ValueNode) Err() error          { return nil }

func (n *ValueNode) IsResolved() bool {
	if withHoles, ok := n.Value.(WithHoles); ok {
		return len(withHoles.GetHoles()) == 0
	}
	return true
}

func (n *ValueNode) ProcessHoles(fills map[string]interface{}) map[string]interface{} {
	if withHoles, ok := n.Value.(WithHoles); ok {
		return withHoles.ProcessHoles(fills)
	}
	return make(map[string]interface{})
}

func (n *ValueNode) ProcessRefs(refs map[string]interface{}) {
	if withRef, ok := n.Value.(WithRefs); ok {
		withRef.ProcessRefs(refs)
	}
}

func (n *ValueNode) GetRefs() (refs []string) {
	if withRef, ok := n.Value.(WithRefs); ok {
		refs = append(refs, withRef.GetRefs()...)
	}
	return
}

func (n *ValueNode) ReplaceRef(key string, value CompositeValue) {
	if withRef, ok := n.Value.(WithRefs); ok {
		if withRef.IsRef(key) {
			n.Value = value
		} else {
			withRef.ReplaceRef(key, value)
		}
	}
}

func (n *ValueNode) IsRef(key string) bool {
	return false
}

func (n *ValueNode) GetHoles() map[string]*Hole {
	if withHoles, ok := n.Value.(WithHoles); ok {
		return withHoles.GetHoles()
	}
	return make(map[string]*Hole)
}

func (s *Statement) Clone() *Statement {
	newStat := &Statement{}
	newStat.Node = s.Node.clone()

	return newStat
}

func (a *AST) String() string {
	var all []string
	for _, stat := range a.Statements {
		all = append(all, stat.String())
	}
	return strings.Join(all, "\n")
}

func (n *DeclarationNode) clone() Node {
	decl := &DeclarationNode{
		Ident: n.Ident,
	}
	if n.Expr != nil {
		decl.Expr = n.Expr.clone().(ExpressionNode)
	}
	return decl
}

func (n *DeclarationNode) String() string {
	return fmt.Sprintf("%s = %s", n.Ident, n.Expr)
}

func printParamValue(i interface{}) string {
	switch ii := i.(type) {
	case nil:
		return ""
	case []string:
		return "[" + strings.Join(ii, ",") + "]"
	case []interface{}:
		var strs []string
		for _, val := range ii {
			strs = append(strs, fmt.Sprint(val))
		}
		return "[" + strings.Join(strs, ",") + "]"
	case string:
		return quoteStringIfNeeded(ii)
	default:
		return fmt.Sprintf("%v", i)
	}
}

func (a *AST) Clone() *AST {
	clone := &AST{}
	for _, stat := range a.Statements {
		clone.Statements = append(clone.Statements, stat.Clone())
	}
	return clone
}
