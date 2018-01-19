package ast

func ProcessRefs(tree *AST, fillers map[string]interface{}) {
	tree.visitRefs(func(ctx visitContext, parent interface{}, node RefNode) {
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

			case *CommandNode:
				p.ParamNodes[ctx.key] = val
			case *RightExpressionNode:
				p.i = val
			}
		}
	})
}

func RemoveOptionalHoles(tree *AST) {
	tree.visitHoles(func(ctx visitContext, parent interface{}, node HoleNode) {
		if node.IsOptional() {
			switch p := parent.(type) {
			case ListNode:

			case *CommandNode:
				delete(p.ParamNodes, ctx.key)
			case *RightExpressionNode:
				p.i = nil
			}
		}
	})
}

func CollectHoles(tree *AST) (holes []HoleNode) {
	tree.visitHoles(func(ctx visitContext, parent interface{}, node HoleNode) {
		holes = append(holes, node)
	})
	return
}

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

func CollectAliases(tree *AST) (aliases []AliasNode) {
	tree.visitAliases(func(ctx visitContext, parent interface{}, node AliasNode) {
		aliases = append(aliases, node)
	})
	return
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

func (a *AST) visitRefs(visit func(visitContext, interface{}, RefNode)) {
	ctx := visitContext{}
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
					for _, el := range p.arr {
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
						for _, el := range p.arr {
							if ref, ok := el.(RefNode); ok {
								visit(ctx, p, ref)
							}
						}
					}
				}
			case *RightExpressionNode:
				if ref, ok := node.i.(RefNode); ok {
					ctx.key = st.Ident
					visit(ctx, node, ref)
				}
			}
		}
	}
}

func (a *AST) visitHoles(visit func(visitContext, interface{}, HoleNode)) {
	ctx := visitContext{}
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
					for _, el := range p.arr {
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
						for _, el := range p.arr {
							if hole, ok := el.(HoleNode); ok {
								visit(ctx, p, hole)
							}
						}
					}
				}
			case *RightExpressionNode:
				if hole, ok := node.i.(HoleNode); ok {
					ctx.key = st.Ident
					visit(ctx, node, hole)
				}
			}
		}
	}
}

func (a *AST) visitAliases(visit func(ctx visitContext, parent interface{}, node AliasNode)) {
	ctx := visitContext{}
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
					for _, el := range p.arr {
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
					ctx.key = st.Ident
					visit(ctx, node, alias)
				}
			}
		}
	}
}
