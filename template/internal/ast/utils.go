package ast

func ProcessHoles(tree *AST, fillers map[string]interface{}) map[string]interface{} {
	processed := make(map[string]interface{})
	tree.visitHoles(func(ctx visitContext, parent interface{}, node HoleNode) {
		var done bool
		var val interface{}
		for k, v := range fillers {
			if k == node.key {
				done = true
				val = v
				processed[k] = v
			}
		}

		if done {
			switch p := parent.(type) {
			case ListNode:

			case *CommandNode:
				p.ParamNodes[ctx.key] = val
			case *RightExpressionNode:
				p.i = val
			}
		}
	})

	return processed
}

func ProcessAliases(tree *AST, aliasFunc func(action, entity string, key string) func(string) (string, bool)) {
	tree.visitAliases(func(ctx visitContext, parent interface{}, node AliasNode) {
		resolv, hasResolv := aliasFunc(ctx.action, ctx.entity, ctx.key)(node.key)
		if hasResolv {
			switch par := parent.(type) {
			case ListNode:

			case *CommandNode:
				par.ParamNodes[ctx.key] = resolv
			case *RightExpressionNode:
				par.i = resolv
			}
		}
	})
}

type visitContext struct {
	action, entity, key string
}

func (a *AST) visitHoles(visit func(visitContext, interface{}, HoleNode)) {
	ctx := visitContext{}
	for _, sts := range a.Statements {
		switch node := sts.Node.(type) {
		case *CommandNode:
			ctx.action, ctx.entity = node.Action, node.Entity
			for pKey, pNode := range node.ParamNodes {
				ctx.key = pKey
				switch p := pNode.(type) {
				case HoleNode:
					visit(ctx, node, p)
				case ListNode:
					for _, el := range p.arr {
						if hole, ok := el.(HoleNode); ok {
							visit(ctx, p, hole)
						}
					}
				}
			}
		case *DeclarationNode:
			expr := sts.Node.(*DeclarationNode).Expr
			switch node := expr.(type) {
			case *CommandNode:
				ctx.action, ctx.entity = node.Action, node.Entity
				for pKey, pNode := range node.ParamNodes {
					ctx.key = pKey
					switch p := pNode.(type) {
					case HoleNode:
						visit(ctx, node, p)
					case ListNode:
						for _, el := range p.arr {
							if hole, ok := el.(HoleNode); ok {
								visit(ctx, p, hole)
							}
						}
					}
				}
			case *RightExpressionNode:
				if hole, ok := node.i.(HoleNode); ok {
					visit(ctx, node, hole)
				}
			}
		}
	}
}

func (a *AST) visitAliases(visit func(ctx visitContext, parent interface{}, node AliasNode)) {
	ctx := visitContext{}
	for _, sts := range a.Statements {
		switch node := sts.Node.(type) {
		case *CommandNode:
			ctx.action, ctx.entity = node.Action, node.Entity
			for pKey, pNode := range node.ParamNodes {
				ctx.key = pKey
				switch p := pNode.(type) {
				case AliasNode:
					visit(ctx, node, p)
				case ListNode:
					for _, el := range p.arr {
						if alias, ok := el.(AliasNode); ok {
							visit(ctx, p, alias)
						}
					}
				}
			}
		case *DeclarationNode:
			expr := sts.Node.(*DeclarationNode).Expr
			switch node := expr.(type) {
			case *CommandNode:
				for pKey, pNode := range node.ParamNodes {
					ctx.key = pKey
					switch p := pNode.(type) {
					case AliasNode:
						ctx.key = p.key
						visit(ctx, node, p)
					case ListNode:
						for _, el := range p.arr {
							if alias, ok := el.(AliasNode); ok {
								ctx.key = alias.key
								visit(ctx, p, alias)
							}
						}
					}
				}
			case *RightExpressionNode:
				if alias, ok := node.i.(AliasNode); ok {
					visit(ctx, node, alias)
				}
			}
		}
	}
}
