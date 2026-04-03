package ssa

import (
	"fmt"
	"math"

	"github.com/glyphlang/glyph/pkg/ast"
)

// Builder converts AST to SSA form using the Braun et al. algorithm
// for on-the-fly SSA construction.
type Builder struct {
	fn       *Func
	curBlock *Block
	// Variable tracking: map from variable name to current SSA value per block
	varDefs map[*Block]map[string]*Value
}

// NewBuilder creates a new SSA builder.
func NewBuilder() *Builder {
	return &Builder{
		varDefs: make(map[*Block]map[string]*Value),
	}
}

// BuildRoute converts a route AST node into an SSA function.
func (b *Builder) BuildRoute(route *ast.Route) (*Func, error) {
	f := NewFunc(fmt.Sprintf("%s %s", route.Method, route.Path), SourceRoute)
	b.fn = f
	b.curBlock = f.Entry

	// Inject path parameters as pre-defined variables
	// (they'll be set by the runtime before execution)

	if err := b.buildStatements(route.Body); err != nil {
		return nil, err
	}

	// Ensure the function has a terminator
	if b.curBlock != nil && b.curBlock.Terminator() == nil {
		v := b.emitConst(TypeNull, 0, "")
		b.emitTerminator(OpHalt, v)
	}

	return f, nil
}

// BuildFunction converts a function AST node into an SSA function.
func (b *Builder) BuildFunction(fn *ast.Function) (*Func, error) {
	f := NewFunc(fn.Name, SourceFunction)
	b.fn = f
	b.curBlock = f.Entry

	// Record parameter names
	for _, p := range fn.Params {
		f.Params = append(f.Params, p.Name)
	}

	if err := b.buildStatements(fn.Body); err != nil {
		return nil, err
	}

	if b.curBlock != nil && b.curBlock.Terminator() == nil {
		v := b.emitConst(TypeNull, 0, "")
		b.emitTerminator(OpReturn, v)
	}

	return f, nil
}

// BuildProgram converts an entire module into an SSA program.
func (b *Builder) BuildProgram(module *ast.Module) (*Program, error) {
	prog := &Program{}
	for _, item := range module.Items {
		switch it := item.(type) {
		case *ast.Route:
			builder := NewBuilder()
			f, err := builder.BuildRoute(it)
			if err != nil {
				return nil, fmt.Errorf("route %s %s: %w", it.Method, it.Path, err)
			}
			prog.Funcs = append(prog.Funcs, f)
		case *ast.Function:
			builder := NewBuilder()
			f, err := builder.BuildFunction(it)
			if err != nil {
				return nil, fmt.Errorf("function %s: %w", it.Name, err)
			}
			prog.Funcs = append(prog.Funcs, f)
		}
	}
	return prog, nil
}

// --- Statement compilation ---

func (b *Builder) buildStatements(stmts []ast.Statement) error {
	for _, stmt := range stmts {
		if b.curBlock == nil {
			break // unreachable code after terminator
		}
		if err := b.buildStatement(stmt); err != nil {
			return err
		}
	}
	return nil
}

func (b *Builder) buildStatement(stmt ast.Statement) error {
	switch s := stmt.(type) {
	case ast.AssignStatement, *ast.AssignStatement:
		t, v := unwrapAssign(s)
		return b.buildAssign(t, v)
	case ast.ReassignStatement, *ast.ReassignStatement:
		t, v := unwrapAssign(s)
		return b.buildAssign(t, v)
	case ast.ReturnStatement:
		return b.buildReturn(s)
	case *ast.ReturnStatement:
		return b.buildReturn(*s)
	case ast.IfStatement:
		return b.buildIf(s)
	case *ast.IfStatement:
		return b.buildIf(*s)
	case ast.WhileStatement:
		return b.buildWhile(s)
	case *ast.WhileStatement:
		return b.buildWhile(*s)
	case ast.ForStatement:
		return b.buildFor(s)
	case *ast.ForStatement:
		return b.buildFor(*s)
	case ast.ExpressionStatement:
		_, err := b.buildExpr(s.Expr)
		return err
	case *ast.ExpressionStatement:
		_, err := b.buildExpr(s.Expr)
		return err
	case ast.BreakStatement, *ast.BreakStatement:
		return nil
	case ast.ContinueStatement, *ast.ContinueStatement:
		return nil
	case ast.IndexAssignStatement:
		return b.buildIndexAssign(s)
	case *ast.IndexAssignStatement:
		return b.buildIndexAssign(*s)
	default:
		// Unsupported statements become opaque calls
		return nil
	}
}

// unwrapAssign extracts (target, value) from either value or pointer form.
func unwrapAssign(s ast.Statement) (string, ast.Expr) {
	switch a := s.(type) {
	case ast.AssignStatement:
		return a.Target, a.Value
	case *ast.AssignStatement:
		return a.Target, a.Value
	case ast.ReassignStatement:
		return a.Target, a.Value
	case *ast.ReassignStatement:
		return a.Target, a.Value
	}
	return "", nil
}

func (b *Builder) buildAssign(name string, expr ast.Expr) error {
	val, err := b.buildExpr(expr)
	if err != nil {
		return err
	}
	b.writeVar(name, val)
	v := b.emit(OpStoreVar, TypeVoid, val)
	v.AuxStr = name
	return nil
}

func (b *Builder) buildReturn(s ast.ReturnStatement) error {
	if s.Value == nil {
		v := b.emitConst(TypeNull, 0, "")
		b.emitTerminator(OpReturn, v)
		return nil
	}
	val, err := b.buildExpr(s.Value)
	if err != nil {
		return err
	}
	b.emitTerminator(OpReturn, val)
	return nil
}

func (b *Builder) buildIf(s ast.IfStatement) error {
	cond, err := b.buildExpr(s.Condition)
	if err != nil {
		return err
	}

	thenBlock := b.fn.NewBlock("if.then")
	elseBlock := b.fn.NewBlock("if.else")
	mergeBlock := b.fn.NewBlock("if.merge")

	// Emit conditional branch
	b.emitIf(cond, thenBlock, elseBlock)

	// Then branch
	b.switchTo(thenBlock)
	if err := b.buildStatements(s.ThenBlock); err != nil {
		return err
	}
	if b.curBlock != nil && b.curBlock.Terminator() == nil {
		b.emitJump(mergeBlock)
	}

	// Else branch
	b.switchTo(elseBlock)
	if err := b.buildStatements(s.ElseBlock); err != nil {
		return err
	}
	if b.curBlock != nil && b.curBlock.Terminator() == nil {
		b.emitJump(mergeBlock)
	}

	b.switchTo(mergeBlock)
	return nil
}

func (b *Builder) buildWhile(s ast.WhileStatement) error {
	condBlock := b.fn.NewBlock("while.cond")
	bodyBlock := b.fn.NewBlock("while.body")
	exitBlock := b.fn.NewBlock("while.exit")

	b.emitJump(condBlock)

	// Condition
	b.switchTo(condBlock)
	cond, err := b.buildExpr(s.Condition)
	if err != nil {
		return err
	}
	b.emitIf(cond, bodyBlock, exitBlock)

	// Body
	b.switchTo(bodyBlock)
	if err := b.buildStatements(s.Body); err != nil {
		return err
	}
	if b.curBlock != nil && b.curBlock.Terminator() == nil {
		b.emitJump(condBlock)
	}

	b.switchTo(exitBlock)
	return nil
}

func (b *Builder) buildFor(s ast.ForStatement) error {
	// For loops desugar to: iter = GetIter(iterable); while IterHasNext(iter) { val = IterNext(iter); body }
	iterable, err := b.buildExpr(s.Iterable)
	if err != nil {
		return err
	}

	iter := b.emit(OpGetIter, TypeAny, iterable)

	condBlock := b.fn.NewBlock("for.cond")
	bodyBlock := b.fn.NewBlock("for.body")
	exitBlock := b.fn.NewBlock("for.exit")

	b.emitJump(condBlock)

	// Condition
	b.switchTo(condBlock)
	hasNext := b.emit(OpIterHasNext, TypeBool, iter)
	b.emitIf(hasNext, bodyBlock, exitBlock)

	// Body
	b.switchTo(bodyBlock)
	next := b.emit(OpIterNext, TypeAny, iter)
	b.writeVar(s.ValueVar, next)
	sv := b.emit(OpStoreVar, TypeVoid, next)
	sv.AuxStr = s.ValueVar

	if err := b.buildStatements(s.Body); err != nil {
		return err
	}
	if b.curBlock != nil && b.curBlock.Terminator() == nil {
		b.emitJump(condBlock)
	}

	b.switchTo(exitBlock)
	return nil
}

func (b *Builder) buildIndexAssign(s ast.IndexAssignStatement) error {
	val, err := b.buildExpr(s.Value)
	if err != nil {
		return err
	}

	switch target := s.Target.(type) {
	case ast.ArrayIndexExpr:
		arr, err := b.buildExpr(target.Array)
		if err != nil {
			return err
		}
		idx, err := b.buildExpr(target.Index)
		if err != nil {
			return err
		}
		b.emit(OpSetIndex, TypeVoid, arr, idx, val)
	case ast.FieldAccessExpr:
		obj, err := b.buildExpr(target.Object)
		if err != nil {
			return err
		}
		v := b.emit(OpSetField, TypeVoid, obj, val)
		v.AuxStr = target.Field
	}
	return nil
}

// --- Expression compilation ---

func (b *Builder) buildExpr(expr ast.Expr) (*Value, error) {
	switch e := expr.(type) {
	case ast.LiteralExpr:
		return b.buildLiteral(e.Value), nil
	case ast.VariableExpr:
		return b.readVar(e.Name), nil
	case ast.BinaryOpExpr:
		return b.buildBinOp(e)
	case ast.UnaryOpExpr:
		return b.buildUnaryOp(e)
	case ast.FunctionCallExpr:
		return b.buildCall(e)
	case ast.ObjectExpr:
		return b.buildObject(e)
	case ast.ArrayExpr:
		return b.buildArray(e)
	case ast.FieldAccessExpr:
		return b.buildFieldAccess(e)
	case ast.ArrayIndexExpr:
		return b.buildArrayIndex(e)
	case ast.TryExpression:
		inner, err := b.buildExpr(e.Expr)
		if err != nil {
			return nil, err
		}
		return b.emit(OpTry, TypeAny, inner), nil
	case ast.PipeExpr:
		return b.buildPipe(e)
	case ast.LambdaExpr:
		// Lambdas are complex — represent as opaque for now
		return b.emitConst(TypeFunction, 0, ""), nil
	default:
		return b.emitConst(TypeAny, 0, ""), nil
	}
}

func (b *Builder) buildLiteral(lit ast.Literal) *Value {
	switch l := lit.(type) {
	case ast.IntLiteral:
		return b.emitConst(TypeInt, l.Value, "")
	case ast.FloatLiteral:
		return b.emitConst(TypeFloat, int64(math.Float64bits(l.Value)), "")
	case ast.StringLiteral:
		return b.emitConst(TypeString, 0, l.Value)
	case ast.BoolLiteral:
		aux := int64(0)
		if l.Value {
			aux = 1
		}
		return b.emitConst(TypeBool, aux, "")
	case ast.NullLiteral:
		return b.emitConst(TypeNull, 0, "")
	default:
		return b.emitConst(TypeNull, 0, "")
	}
}

func (b *Builder) buildBinOp(e ast.BinaryOpExpr) (*Value, error) {
	left, err := b.buildExpr(e.Left)
	if err != nil {
		return nil, err
	}
	right, err := b.buildExpr(e.Right)
	if err != nil {
		return nil, err
	}

	// Determine typed operation based on operand types
	// For now, use the left operand's type to choose the op.
	// The type system can refine this later.
	lt := left.Type
	if lt == TypeAny {
		lt = right.Type
	}

	switch e.Op {
	case ast.Add:
		switch lt {
		case TypeFloat:
			return b.emit(OpAddFloat, TypeFloat, left, right), nil
		case TypeString:
			return b.emit(OpAddStr, TypeString, left, right), nil
		default:
			return b.emit(OpAddInt, TypeInt, left, right), nil
		}
	case ast.Sub:
		if lt == TypeFloat {
			return b.emit(OpSubFloat, TypeFloat, left, right), nil
		}
		return b.emit(OpSubInt, TypeInt, left, right), nil
	case ast.Mul:
		if lt == TypeFloat {
			return b.emit(OpMulFloat, TypeFloat, left, right), nil
		}
		return b.emit(OpMulInt, TypeInt, left, right), nil
	case ast.Div:
		if lt == TypeFloat {
			return b.emit(OpDivFloat, TypeFloat, left, right), nil
		}
		return b.emit(OpDivInt, TypeInt, left, right), nil
	case ast.Mod:
		return b.emit(OpModInt, TypeInt, left, right), nil
	case ast.Eq:
		return b.emitCmp(OpEqInt, OpEqFloat, OpEqStr, OpEqBool, lt, left, right), nil
	case ast.Ne:
		return b.emitCmp(OpNeInt, OpNeFloat, OpNeStr, OpNeBool, lt, left, right), nil
	case ast.Lt:
		if lt == TypeFloat {
			return b.emit(OpLtFloat, TypeBool, left, right), nil
		}
		return b.emit(OpLtInt, TypeBool, left, right), nil
	case ast.Le:
		if lt == TypeFloat {
			return b.emit(OpLeFloat, TypeBool, left, right), nil
		}
		return b.emit(OpLeInt, TypeBool, left, right), nil
	case ast.Gt:
		if lt == TypeFloat {
			return b.emit(OpGtFloat, TypeBool, left, right), nil
		}
		return b.emit(OpGtInt, TypeBool, left, right), nil
	case ast.Ge:
		if lt == TypeFloat {
			return b.emit(OpGeFloat, TypeBool, left, right), nil
		}
		return b.emit(OpGeInt, TypeBool, left, right), nil
	case ast.And:
		return b.emit(OpAnd, TypeBool, left, right), nil
	case ast.Or:
		return b.emit(OpOr, TypeBool, left, right), nil
	default:
		return b.emit(OpAddInt, TypeInt, left, right), nil
	}
}

func (b *Builder) emitCmp(intOp, floatOp, strOp, boolOp Op, lt Type, left, right *Value) *Value {
	switch lt {
	case TypeFloat:
		return b.emit(floatOp, TypeBool, left, right)
	case TypeString:
		return b.emit(strOp, TypeBool, left, right)
	case TypeBool:
		return b.emit(boolOp, TypeBool, left, right)
	default:
		return b.emit(intOp, TypeBool, left, right)
	}
}

func (b *Builder) buildUnaryOp(e ast.UnaryOpExpr) (*Value, error) {
	operand, err := b.buildExpr(e.Right)
	if err != nil {
		return nil, err
	}

	switch e.Op {
	case ast.Neg:
		if operand.Type == TypeFloat {
			return b.emit(OpNegFloat, TypeFloat, operand), nil
		}
		return b.emit(OpNegInt, TypeInt, operand), nil
	case ast.Not:
		return b.emit(OpNot, TypeBool, operand), nil
	default:
		return operand, nil
	}
}
func (b *Builder) buildCall(e ast.FunctionCallExpr) (*Value, error) {
	// Special handle builtins that map to SSA ops
        if e.Name == "spawn" && len(e.Args) == 1 {
                offset, err := b.buildExpr(e.Args[0])
                if err != nil {
                        return nil, err
                }
                return b.emit(OpMitosis, TypeInt, offset), nil
        }

        if e.Name == "mutate" && len(e.Args) == 2 {
                val, err := b.buildExpr(e.Args[0])
                if err != nil {
                        return nil, err
                }
                offset, err := b.buildExpr(e.Args[1])
                if err != nil {
                        return nil, err
                }
                return b.emit(OpMutator, TypeVoid, val, offset), nil
        }
	if e.Name == "telemetry" && len(e.Args) == 2 {
		slot, err := b.buildExpr(e.Args[0])
		if err != nil {
			return nil, err
		}
		val, err := b.buildExpr(e.Args[1])
		if err != nil {
			return nil, err
		}
		return b.emit(OpTelemetry, TypeVoid, slot, val), nil
	}

	args := make([]*Value, len(e.Args))
	for i, arg := range e.Args {
		v, err := b.buildExpr(arg)
		if err != nil {
			return nil, err
		}
		args[i] = v
	}
	v := b.emit(OpCall, TypeAny, args...)
	v.AuxStr = e.Name
	return v, nil
}

func (b *Builder) buildObject(e ast.ObjectExpr) (*Value, error) {
	args := make([]*Value, len(e.Fields))
	names := make([]string, len(e.Fields))
	for i, f := range e.Fields {
		v, err := b.buildExpr(f.Value)
		if err != nil {
			return nil, err
		}
		args[i] = v
		names[i] = f.Key
	}
	v := b.emit(OpBuildObj, TypeObject, args...)
	v.AuxStr = joinNames(names)
	return v, nil
}

func (b *Builder) buildArray(e ast.ArrayExpr) (*Value, error) {
	args := make([]*Value, len(e.Elements))
	for i, el := range e.Elements {
		v, err := b.buildExpr(el)
		if err != nil {
			return nil, err
		}
		args[i] = v
	}
	return b.emit(OpBuildArray, TypeArray, args...), nil
}

func (b *Builder) buildFieldAccess(e ast.FieldAccessExpr) (*Value, error) {
	obj, err := b.buildExpr(e.Object)
	if err != nil {
		return nil, err
	}
	v := b.emit(OpGetField, TypeAny, obj)
	v.AuxStr = e.Field
	return v, nil
}

func (b *Builder) buildArrayIndex(e ast.ArrayIndexExpr) (*Value, error) {
	arr, err := b.buildExpr(e.Array)
	if err != nil {
		return nil, err
	}
	idx, err := b.buildExpr(e.Index)
	if err != nil {
		return nil, err
	}
	return b.emit(OpGetIndex, TypeAny, arr, idx), nil
}

func (b *Builder) buildPipe(e ast.PipeExpr) (*Value, error) {
	left, err := b.buildExpr(e.Left)
	if err != nil {
		return nil, err
	}
	// Pipe: left |> fn becomes fn(left)
	if call, ok := e.Right.(ast.FunctionCallExpr); ok {
		args := make([]*Value, 1+len(call.Args))
		args[0] = left
		for i, arg := range call.Args {
			v, err := b.buildExpr(arg)
			if err != nil {
				return nil, err
			}
			args[i+1] = v
		}
		v := b.emit(OpCall, TypeAny, args...)
		v.AuxStr = call.Name
		return v, nil
	}
	// Fallback: just evaluate right side
	return b.buildExpr(e.Right)
}

// --- Helpers ---

func (b *Builder) emit(op Op, typ Type, args ...*Value) *Value {
	return b.fn.NewValue(b.curBlock, op, typ, args...)
}

func (b *Builder) emitConst(typ Type, auxInt int64, auxStr string) *Value {
	v := b.fn.NewValue(b.curBlock, OpConst, typ)
	v.AuxInt = auxInt
	v.AuxStr = auxStr
	return v
}

func (b *Builder) emitTerminator(op Op, args ...*Value) {
	v := b.fn.NewValue(b.curBlock, op, TypeVoid, args...)
	_ = v
	b.curBlock = nil // block is now terminated
}

func (b *Builder) emitIf(cond *Value, thenBlk, elseBlk *Block) {
	v := b.fn.NewValue(b.curBlock, OpIf, TypeVoid, cond)
	_ = v
	b.curBlock.Succs = []*Block{thenBlk, elseBlk}
	thenBlk.Preds = append(thenBlk.Preds, b.curBlock)
	elseBlk.Preds = append(elseBlk.Preds, b.curBlock)
	b.curBlock = nil
}

func (b *Builder) emitJump(target *Block) {
	b.fn.NewValue(b.curBlock, OpJump, TypeVoid)
	b.curBlock.Succs = []*Block{target}
	target.Preds = append(target.Preds, b.curBlock)
	b.curBlock = nil
}

func (b *Builder) switchTo(blk *Block) {
	b.curBlock = blk
}

func (b *Builder) writeVar(name string, val *Value) {
	if b.varDefs[b.curBlock] == nil {
		b.varDefs[b.curBlock] = make(map[string]*Value)
	}
	b.varDefs[b.curBlock][name] = val
}

func (b *Builder) readVar(name string) *Value {
	// Check current block first
	if defs, ok := b.varDefs[b.curBlock]; ok {
		if v, ok := defs[name]; ok {
			return v
		}
	}
	// Not found — emit a LoadVar (the runtime will resolve it)
	v := b.emit(OpLoadVar, TypeAny)
	v.AuxStr = name
	return v
}

func joinNames(names []string) string {
	result := ""
	for i, n := range names {
		if i > 0 {
			result += ","
		}
		result += n
	}
	return result
}
