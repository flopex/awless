package ast

import (
	"errors"
	"fmt"
	"strings"
)

func VerifyRefs(tree *AST) error {
	var errs []string
	add := func(err string) {
		errs = append(errs, err)
	}

	newCtx := new(visitContext)
	tree.visitRefs(func(ctx *visitContext, parent interface{}, node RefNode) {
		if !contains(ctx.declaredVariables, node.key) {
			add(fmt.Sprintf("using reference '$%s' but '%[1]s' is undefined in template", node.key))
		}
	}, newCtx)

	allDeclaredVariables := newCtx.declaredVariables

	for i, declared := range allDeclaredVariables {
		if contains(allDeclaredVariables[:i], declared) {
			add(fmt.Sprintf("using reference '$%s' but '%[1]s' has already been assigned in template", declared))
		}
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}

	return nil
}

func ProcessRefs(tree *AST, fillers map[string]interface{}) {
	tree.visitRefs(func(ctx *visitContext, parent interface{}, node RefNode) {
		var done bool
		var val interface{}
		for k, v := range fillers {
			if k == node.key {
				done = true
				val = v
			}
		}

		if done {
			switch p := parent.(type) {
			case ListNode:
				p.arr[ctx.listIndex] = val
			case *CommandNode:
				p.ParamNodes[ctx.key] = val
			case *RightExpressionNode:
				p.i = val
			}
		}
	})
}

func RemoveOptionalHoles(tree *AST) {
	tree.visitHoles(func(ctx *visitContext, parent interface{}, node HoleNode) {
		if node.IsOptional() {
			switch p := parent.(type) {
			case ListNode:
				p.arr = append(p.arr[:ctx.listIndex], p.arr[ctx.listIndex+1:]...)
			case *CommandNode:
				delete(p.ParamNodes, ctx.key)
			case *RightExpressionNode:
				p.i = nil
			}
		}
	})
}

func CollectHoles(tree *AST) (holes []HoleNode) {
	tree.visitHoles(func(ctx *visitContext, parent interface{}, node HoleNode) {
		holes = append(holes, node)
	})
	return
}

func ProcessHoles(tree *AST, fillers map[string]interface{}) map[string]interface{} {
	processed := make(map[string]interface{})
	tree.visitHoles(func(ctx *visitContext, parent interface{}, node HoleNode) {
		var done bool
		var val interface{}
		for k, v := range fillers {
			if k == node.key {
				done = true
				val = v
				switch vv := v.(type) {
				case AliasNode, RefNode, HoleNode, ConcatenationNode:
					processed[k] = fmt.Sprint(v)
				case ListNode:
					var arr []interface{}
					for _, a := range vv.arr {
						switch e := a.(type) {
						case AliasNode, RefNode, HoleNode:
							arr = append(arr, fmt.Sprint(e))
						default:
							arr = append(arr, e)
						}
					}
					processed[k] = arr
				default:
					processed[k] = v
				}
			}
		}

		if done {
			switch p := parent.(type) {
			case ConcatenationNode:
				p.arr[ctx.concatItemIndex] = val
			case ListNode:
				p.arr[ctx.listIndex] = val
			case *CommandNode:
				p.ParamNodes[ctx.key] = val
			case *RightExpressionNode:
				p.i = val
			}
		}
	})

	return processed
}

func CollectAliases(tree *AST) (aliases []AliasNode) {
	tree.visitAliases(func(ctx *visitContext, parent interface{}, node AliasNode) {
		aliases = append(aliases, node)
	})
	return
}

func ProcessAliases(tree *AST, aliasFunc func(action, entity string, key string) func(string) (string, bool)) {
	tree.visitAliases(func(ctx *visitContext, parent interface{}, node AliasNode) {
		resolv, hasResolv := aliasFunc(ctx.action, ctx.entity, ctx.key)(node.key)
		if hasResolv {
			switch p := parent.(type) {
			case ListNode:
				p.arr[ctx.listIndex] = resolv
			case ConcatenationNode:
				p.arr[ctx.concatItemIndex] = resolv
			case *CommandNode:
				p.ParamNodes[ctx.key] = resolv
			case *RightExpressionNode:
				p.i = resolv
			}
		}
	})
}

type visitContext struct {
	action, entity, key string
	declaredVariables   []string
	listIndex           int
	concatItemIndex     int
}

func (a *AST) visitRefs(visit func(*visitContext, interface{}, RefNode), contexts ...*visitContext) {
	var ctx *visitContext
	if len(contexts) > 0 {
		ctx = contexts[0]
	} else {
		ctx = new(visitContext)
	}
	for _, sts := range a.Statements {
		switch st := sts.Node.(type) {
		case *CommandNode:
			ctx.action, ctx.entity = st.Action, st.Entity
			for pKey, pNode := range st.ParamNodes {
				ctx.key = pKey
				switch p := pNode.(type) {
				case RefNode:
					visit(ctx, st, p)
				case ListNode:
					for i, el := range p.arr {
						ctx.listIndex = i
						if ref, ok := el.(RefNode); ok {
							visit(ctx, p, ref)
						}
					}
				}
			}
		case *DeclarationNode:
			expr := st.Expr
			switch node := expr.(type) {
			case *CommandNode:
				ctx.action, ctx.entity = node.Action, node.Entity
				for pKey, pNode := range node.ParamNodes {
					ctx.key = pKey
					switch p := pNode.(type) {
					case RefNode:
						visit(ctx, node, p)
					case ListNode:
						for i, el := range p.arr {
							ctx.listIndex = i
							if ref, ok := el.(RefNode); ok {
								visit(ctx, p, ref)
							}
						}
					}
				}
			case *RightExpressionNode:
				ctx.key = st.Ident
				switch right := node.i.(type) {
				case RefNode:
					visit(ctx, node, right)
				case ListNode:
					for i, el := range right.arr {
						ctx.listIndex = i
						if ref, ok := el.(RefNode); ok {
							visit(ctx, right, ref)
						}
					}
				}
			}
			ctx.declaredVariables = append(ctx.declaredVariables, st.Ident)
		}
	}
}

func (a *AST) visitHoles(visit func(*visitContext, interface{}, HoleNode)) {
	ctx := new(visitContext)
	for _, sts := range a.Statements {
		switch st := sts.Node.(type) {
		case *CommandNode:
			ctx.action, ctx.entity = st.Action, st.Entity
			for pKey, pNode := range st.ParamNodes {
				ctx.key = pKey
				switch p := pNode.(type) {
				case HoleNode:
					visit(ctx, st, p)
				case ListNode:
					for i, el := range p.arr {
						ctx.listIndex = i
						if hole, ok := el.(HoleNode); ok {
							visit(ctx, p, hole)
						}
					}
				case ConcatenationNode:
					for i, el := range p.arr {
						ctx.concatItemIndex = i
						if hole, ok := el.(HoleNode); ok {
							visit(ctx, p, hole)
						}
					}
				}
			}
		case *DeclarationNode:
			expr := st.Expr
			switch node := expr.(type) {
			case *CommandNode:
				ctx.action, ctx.entity = node.Action, node.Entity
				for pKey, pNode := range node.ParamNodes {
					ctx.key = pKey
					switch p := pNode.(type) {
					case HoleNode:
						visit(ctx, node, p)
					case ListNode:
						for i, el := range p.arr {
							ctx.listIndex = i
							if hole, ok := el.(HoleNode); ok {
								visit(ctx, p, hole)
							}
						}
					case ConcatenationNode:
						for i, el := range p.arr {
							ctx.concatItemIndex = i
							if hole, ok := el.(HoleNode); ok {
								visit(ctx, p, hole)
							}
						}
					}
				}
			case *RightExpressionNode:
				ctx.key = st.Ident
				switch right := node.i.(type) {
				case HoleNode:
					visit(ctx, node, right)
				case ListNode:
					for i, el := range right.arr {
						ctx.listIndex = i
						if hole, ok := el.(HoleNode); ok {
							visit(ctx, right, hole)
						}
					}
				case ConcatenationNode:
					for i, el := range right.arr {
						ctx.concatItemIndex = i
						if hole, ok := el.(HoleNode); ok {
							visit(ctx, right, hole)
						}
					}
				}
			}
		}
	}
}

func (a *AST) visitAliases(visit func(ctx *visitContext, parent interface{}, node AliasNode)) {
	ctx := new(visitContext)
	for _, sts := range a.Statements {
		switch st := sts.Node.(type) {
		case *CommandNode:
			ctx.action, ctx.entity = st.Action, st.Entity
			for pKey, pNode := range st.ParamNodes {
				ctx.key = pKey
				switch p := pNode.(type) {
				case AliasNode:
					visit(ctx, st, p)
				case ListNode:
					for i, el := range p.arr {
						ctx.listIndex = i
						if alias, ok := el.(AliasNode); ok {
							visit(ctx, p, alias)
						}
					}
				case ConcatenationNode:
					for i, el := range p.arr {
						ctx.concatItemIndex = i
						if alias, ok := el.(AliasNode); ok {
							visit(ctx, p, alias)
						}
					}
				}
			}
		case *DeclarationNode:
			expr := st.Expr
			switch node := expr.(type) {
			case *CommandNode:
				for pKey, pNode := range node.ParamNodes {
					ctx.key = pKey
					switch p := pNode.(type) {
					case AliasNode:
						ctx.key = p.key
						visit(ctx, node, p)
					case ListNode:
						for i, el := range p.arr {
							ctx.listIndex = i
							if alias, ok := el.(AliasNode); ok {
								ctx.key = alias.key
								visit(ctx, p, alias)
							}
						}
					case ConcatenationNode:
						for i, el := range p.arr {
							ctx.concatItemIndex = i
							if alias, ok := el.(AliasNode); ok {
								visit(ctx, p, alias)
							}
						}
					}
				}
			case *RightExpressionNode:
				ctx.key = st.Ident
				switch right := node.i.(type) {
				case AliasNode:
					visit(ctx, node, right)
				case ListNode:
					for i, el := range right.arr {
						ctx.listIndex = i
						if alias, ok := el.(AliasNode); ok {
							visit(ctx, right, alias)
						}
					}
				case ConcatenationNode:
					for i, el := range right.arr {
						ctx.concatItemIndex = i
						if alias, ok := el.(AliasNode); ok {
							visit(ctx, right, alias)
						}
					}
				}
			}
		}
	}
}

func contains(arr []string, s string) bool {
	for _, v := range arr {
		if v == s {
			return true
		}
	}
	return false
}
