package interpreter

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	. "github.com/glyphlang/glyph/pkg/ast"
	gpuPkg "github.com/glyphlang/glyph/pkg/gpu"
	vmPkg "github.com/glyphlang/glyph/pkg/vm"
	"github.com/google/uuid"
)

// builtinFunc is the signature for all builtin function implementations.
type builtinFunc func(i *Interpreter, args []Expr, env *Environment) (interface{}, error)

// builtinFuncs is the dispatch table mapping builtin function names to their implementations.
// Initialized in init() to avoid initialization cycle with evaluateFunctionCall.
var builtinFuncs map[string]builtinFunc

func init() {
	builtinFuncs = map[string]builtinFunc{
		"time.now":    builtinTimeNow,
		"now":         builtinNow,
		"Ok":          builtinOk,
		"Err":         builtinErr,
		"upper":       builtinUpper,
		"lower":       builtinLower,
		"trim":        builtinTrim,
		"split":       builtinSplit,
		"join":        builtinJoin,
		"contains":    builtinContains,
		"replace":     builtinReplace,
		"substring":   builtinSubstring,
		"length":      builtinLength,
		"spawn":       builtinSpawn,
		"mutate":      builtinMutate,
		"startsWith":  builtinStartsWith,
		"endsWith":    builtinEndsWith,
		"indexOf":     builtinIndexOf,
 		"repeat":      builtinRepeat,
		
		"flatten":     builtinFlatten,
		"range":       builtinRange,
		"charAt":      builtinCharAt,
		"charCodeAt":  builtinCharCodeAt,
		"charAtCode":  builtinCharCodeAt,
		"parseInt":    builtinParseInt,
		"parseFloat":  builtinParseFloat,
		"toString":    builtinToString,
		"abs":         builtinAbs,
		"min":         builtinMin,
		"max":         builtinMax,
		"randomInt":   builtinRandomInt,
		"generateId":  builtinGenerateId,
		"append":      builtinAppend,
		"set":         builtinSet,
		"remove":      builtinRemove,
		"keys":        builtinKeys,
		"map":         builtinMap,
		"filter":      builtinFilter,
		"reduce":      builtinReduce,
		"find":        builtinFind,
		"some":        builtinSome,
		"every":       builtinEvery,
		"sort":        builtinSort,
		"reverse":     builtinReverse,
		"flat":        builtinFlat,
		"slice":       builtinSlice,
		"print":       builtinPrint,
		"typeOf":      builtinTypeOf,
		"shr":         builtinShr,
		"band":        builtinBand,
		"intToBytes4": builtinIntToBytes4,
		"intToBytes8": builtinIntToBytes8,
		"writeFile":   builtinWriteFile,
		"readFile":    builtinReadFile,
		"toInt":       builtinToInt,
		"text":        builtinText,
		"html":        builtinHTML,
		"blob":        builtinBlob,
		"redirect":    builtinRedirect,
		"vmExec":      builtinVMExec,
		"gpuExec":     builtinGPUExec,
		"telemetry":   builtinTelemetry,
		"args":        builtinArgs,
		"exists":      builtinExists,
		"__mutator":   builtinMutator,
		"__mitosis":   builtinMitosis,
		"eval_source": builtinEvalSource,
	}
}

func builtinTelemetry(i *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("telemetry() expects 2 arguments, got %d", len(args))
	}
	slotArg, err := i.EvaluateExpression(args[0], env)
	if err != nil {
		return nil, err
	}
	valArg, err := i.EvaluateExpression(args[1], env)
	if err != nil {
		return nil, err
	}

	slot, ok1 := slotArg.(int64)
	val, ok2 := valArg.(int64)
	if !ok1 || !ok2 {
		return nil, fmt.Errorf("telemetry() expects integer arguments, got %T and %T", slotArg, valArg)
	}

	// For interpreted mode, we just print or potentially update a local mock
	if os.Getenv("GLYPH_DEBUG") == "true" {
		fmt.Printf("[TELEMETRY] Slot %d = %d\n", slot, val)
	}

	return nil, nil
}

func builtinTimeNow(_ *Interpreter, _ []Expr, _ *Environment) (interface{}, error) {
	return time.Now().Unix(), nil
}

func builtinNow(_ *Interpreter, _ []Expr, _ *Environment) (interface{}, error) {
	return time.Now().Unix(), nil
}

func builtinOk(i *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("Ok() expects 1 argument, got %d", len(args))
	}
	val, err := i.EvaluateExpression(args[0], env)
	if err != nil {
		return nil, err
	}
	return NewOk(val), nil
}

func builtinErr(i *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("Err() expects 1 argument, got %d", len(args))
	}
	val, err := i.EvaluateExpression(args[0], env)
	if err != nil {
		return nil, err
	}
	return NewErr(val), nil
}

func builtinUpper(i *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	// Convert string to uppercase
	if len(args) != 1 {
		return nil, fmt.Errorf("upper() expects 1 argument, got %d", len(args))
	}
	arg, err := i.EvaluateExpression(args[0], env)
	if err != nil {
		return nil, err
	}
	str, ok := arg.(string)
	if !ok {
		return nil, fmt.Errorf("upper() expects a string argument, got %T", arg)
	}
	return strings.ToUpper(str), nil
}

func builtinLower(i *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	// Convert string to lowercase
	if len(args) != 1 {
		return nil, fmt.Errorf("lower() expects 1 argument, got %d", len(args))
	}
	arg, err := i.EvaluateExpression(args[0], env)
	if err != nil {
		return nil, err
	}
	str, ok := arg.(string)
	if !ok {
		return nil, fmt.Errorf("lower() expects a string argument, got %T", arg)
	}
	return strings.ToLower(str), nil
}

func builtinTrim(i *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	// Remove leading/trailing whitespace
	if len(args) != 1 {
		return nil, fmt.Errorf("trim() expects 1 argument, got %d", len(args))
	}
	arg, err := i.EvaluateExpression(args[0], env)
	if err != nil {
		return nil, err
	}
	str, ok := arg.(string)
	if !ok {
		return nil, fmt.Errorf("trim() expects a string argument, got %T", arg)
	}
	return strings.TrimSpace(str), nil
}

func builtinSplit(i *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	// Split string into array
	if len(args) != 2 {
		return nil, fmt.Errorf("split() expects 2 arguments, got %d", len(args))
	}
	strArg, err := i.EvaluateExpression(args[0], env)
	if err != nil {
		return nil, err
	}
	str, ok := strArg.(string)
	if !ok {
		return nil, fmt.Errorf("split() expects first argument to be a string, got %T", strArg)
	}
	delimArg, err := i.EvaluateExpression(args[1], env)
	if err != nil {
		return nil, err
	}
	delim, ok := delimArg.(string)
	if !ok {
		return nil, fmt.Errorf("split() expects second argument to be a string, got %T", delimArg)
	}
	parts := strings.Split(str, delim)
	result := make([]interface{}, len(parts))
	for idx, part := range parts {
		result[idx] = part
	}
	return result, nil
}

func builtinJoin(i *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	// Join array into string
	if len(args) != 2 {
		return nil, fmt.Errorf("join() expects 2 arguments, got %d", len(args))
	}
	arrArg, err := i.EvaluateExpression(args[0], env)
	if err != nil {
		return nil, err
	}
	arr, ok := arrArg.([]interface{})
	if !ok {
		return nil, fmt.Errorf("join() expects first argument to be an array, got %T", arrArg)
	}
	delimArg, err := i.EvaluateExpression(args[1], env)
	if err != nil {
		return nil, err
	}
	delim, ok := delimArg.(string)
	if !ok {
		return nil, fmt.Errorf("join() expects second argument to be a string, got %T", delimArg)
	}
	strParts := make([]string, len(arr))
	for idx, elem := range arr {
		strParts[idx] = fmt.Sprintf("%v", elem)
	}
	return strings.Join(strParts, delim), nil
}

func builtinContains(i *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	// Check if string contains substring
	if len(args) != 2 {
		return nil, fmt.Errorf("contains() expects 2 arguments, got %d", len(args))
	}
	strArg, err := i.EvaluateExpression(args[0], env)
	if err != nil {
		return nil, err
	}
	str, ok := strArg.(string)
	if !ok {
		return nil, fmt.Errorf("contains() expects first argument to be a string, got %T", strArg)
	}
	substrArg, err := i.EvaluateExpression(args[1], env)
	if err != nil {
		return nil, err
	}
	substr, ok := substrArg.(string)
	if !ok {
		return nil, fmt.Errorf("contains() expects second argument to be a string, got %T", substrArg)
	}
	return strings.Contains(str, substr), nil
}

func builtinReplace(i *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	// Replace occurrences in string
	if len(args) != 3 {
		return nil, fmt.Errorf("replace() expects 3 arguments, got %d", len(args))
	}
	strArg, err := i.EvaluateExpression(args[0], env)
	if err != nil {
		return nil, err
	}
	str, ok := strArg.(string)
	if !ok {
		return nil, fmt.Errorf("replace() expects first argument to be a string, got %T", strArg)
	}
	oldArg, err := i.EvaluateExpression(args[1], env)
	if err != nil {
		return nil, err
	}
	old, ok := oldArg.(string)
	if !ok {
		return nil, fmt.Errorf("replace() expects second argument to be a string, got %T", oldArg)
	}
	newArg, err := i.EvaluateExpression(args[2], env)
	if err != nil {
		return nil, err
	}
	new, ok := newArg.(string)
	if !ok {
		return nil, fmt.Errorf("replace() expects third argument to be a string, got %T", newArg)
	}
	return strings.ReplaceAll(str, old, new), nil
}

func builtinSubstring(i *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	// Get substring
	if len(args) != 3 {
		return nil, fmt.Errorf("substring() expects 3 arguments, got %d", len(args))
	}
	strArg, err := i.EvaluateExpression(args[0], env)
	if err != nil {
		return nil, err
	}
	str, ok := strArg.(string)
	if !ok {
		return nil, fmt.Errorf("substring() expects first argument to be a string, got %T", strArg)
	}
	startArg, err := i.EvaluateExpression(args[1], env)
	if err != nil {
		return nil, err
	}
	start, ok := startArg.(int64)
	if !ok {
		// Try to convert int to int64
		if startInt, ok := startArg.(int); ok {
			start = int64(startInt)
		} else {
			return nil, fmt.Errorf("substring() expects second argument to be an integer, got %T", startArg)
		}
	}
	endArg, err := i.EvaluateExpression(args[2], env)
	if err != nil {
		return nil, err
	}
	end, ok := endArg.(int64)
	if !ok {
		// Try to convert int to int64
		if endInt, ok := endArg.(int); ok {
			end = int64(endInt)
		} else {
			return nil, fmt.Errorf("substring() expects third argument to be an integer, got %T", endArg)
		}
	}
	if start < 0 || end < 0 {
		return nil, fmt.Errorf("substring() indices must be non-negative")
	}
	if start > end {
		return nil, fmt.Errorf("substring() start index must be less than or equal to end index")
	}
	runes := []rune(str)
	if int(start) > len(runes) {
		return nil, fmt.Errorf("substring() start index out of bounds: %d (length %d)", start, len(runes))
	}
	if int(end) > len(runes) {
		return nil, fmt.Errorf("substring() end index out of bounds: %d (length %d)", end, len(runes))
	}
	return string(runes[start:end]), nil
}

func builtinLength(i *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	// Get length of string or array
	if len(args) != 1 {
		return nil, fmt.Errorf("length() expects 1 argument, got %d", len(args))
	}
	arg, err := i.EvaluateExpression(args[0], env)
	if err != nil {
		return nil, err
	}
	switch v := arg.(type) {
	case string:
		return int64(len([]rune(v))), nil
	case []interface{}:
		return int64(len(v)), nil
	case map[string]interface{}:
		return int64(len(v)), nil
	default:
		return nil, fmt.Errorf("length() expects a string, array, or object argument, got %T", arg)
	}
}

func builtinStartsWith(i *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	// Check if string starts with prefix
	if len(args) != 2 {
		return nil, fmt.Errorf("startsWith() expects 2 arguments, got %d", len(args))
	}
	strArg, err := i.EvaluateExpression(args[0], env)
	if err != nil {
		return nil, err
	}
	str, ok := strArg.(string)
	if !ok {
		return nil, fmt.Errorf("startsWith() expects first argument to be a string, got %T", strArg)
	}
	prefixArg, err := i.EvaluateExpression(args[1], env)
	if err != nil {
		return nil, err
	}
	prefix, ok := prefixArg.(string)
	if !ok {
		return nil, fmt.Errorf("startsWith() expects second argument to be a string, got %T", prefixArg)
	}
	return strings.HasPrefix(str, prefix), nil
}

func builtinEndsWith(i *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	// Check if string ends with suffix
	if len(args) != 2 {
		return nil, fmt.Errorf("endsWith() expects 2 arguments, got %d", len(args))
	}
	strArg, err := i.EvaluateExpression(args[0], env)
	if err != nil {
		return nil, err
	}
	str, ok := strArg.(string)
	if !ok {
		return nil, fmt.Errorf("endsWith() expects first argument to be a string, got %T", strArg)
	}
	suffixArg, err := i.EvaluateExpression(args[1], env)
	if err != nil {
		return nil, err
	}
	suffix, ok := suffixArg.(string)
	if !ok {
		return nil, fmt.Errorf("endsWith() expects second argument to be a string, got %T", suffixArg)
	}
	return strings.HasSuffix(str, suffix), nil
}

func builtinIndexOf(i *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	// Find first occurrence of substring
	if len(args) != 2 {
		return nil, fmt.Errorf("indexOf() expects 2 arguments, got %d", len(args))
	}
	strArg, err := i.EvaluateExpression(args[0], env)
	if err != nil {
		return nil, err
	}
	str, ok := strArg.(string)
	if !ok {
		return nil, fmt.Errorf("indexOf() expects first argument to be a string, got %T", strArg)
	}
	substrArg, err := i.EvaluateExpression(args[1], env)
	if err != nil {
		return nil, err
	}
	substr, ok := substrArg.(string)
	if !ok {
		return nil, fmt.Errorf("indexOf() expects second argument to be a string, got %T", substrArg)
	}
	byteIndex := strings.Index(str, substr)
	if byteIndex < 0 {
		return int64(-1), nil
	}
	// Convert byte offset to rune offset for Unicode consistency
	runeIndex := len([]rune(str[:byteIndex]))
	return int64(runeIndex), nil
}

func builtinCharAt(i *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	// Get character at index
	if len(args) != 2 {
		return nil, fmt.Errorf("charAt() expects 2 arguments, got %d", len(args))
	}
	strArg, err := i.EvaluateExpression(args[0], env)
	if err != nil {
		return nil, err
	}
	str, ok := strArg.(string)
	if !ok {
		return nil, fmt.Errorf("charAt() expects first argument to be a string, got %T", strArg)
	}
	indexArg, err := i.EvaluateExpression(args[1], env)
	if err != nil {
		return nil, err
	}
	index, ok := indexArg.(int64)
	if !ok {
		return nil, fmt.Errorf("charAt() expects second argument to be an integer, got %T", indexArg)
	}
	runes := []rune(str)
	if index < 0 || int(index) >= len(runes) {
		return nil, fmt.Errorf("charAt() index out of bounds: %d", index)
	}
	return string(runes[index]), nil
}

func builtinCharCodeAt(i *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	// Get character code at index
	if len(args) != 2 {
		return nil, fmt.Errorf("charCodeAt() expects 2 arguments, got %d", len(args))
	}
	strArg, err := i.EvaluateExpression(args[0], env)
	if err != nil {
		return nil, err
	}
	str, ok := strArg.(string)
	if !ok {
		return nil, fmt.Errorf("charCodeAt() expects first argument to be a string, got %T", strArg)
	}
	indexArg, err := i.EvaluateExpression(args[1], env)
	if err != nil {
		return nil, err
	}
	index, ok := indexArg.(int64)
	if !ok {
		return nil, fmt.Errorf("charCodeAt() expects second argument to be an integer, got %T", indexArg)
	}
	runes := []rune(str)
	if index < 0 || int(index) >= len(runes) {
		return nil, fmt.Errorf("charCodeAt() index out of bounds: %d", index)
	}
	return int64(runes[index]), nil
}

func builtinParseInt(i *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	// Parse string to integer
	if len(args) != 1 {
		return nil, fmt.Errorf("parseInt() expects 1 argument, got %d", len(args))
	}
	arg, err := i.EvaluateExpression(args[0], env)
	if err != nil {
		return nil, err
	}
	str, ok := arg.(string)
	if !ok {
		return nil, fmt.Errorf("parseInt() expects a string argument, got %T", arg)
	}
	str = strings.TrimSpace(str)
	result, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parseInt() failed to parse '%s': %v", str, err)
	}
	return result, nil
}

func builtinParseFloat(i *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	// Parse string to float
	if len(args) != 1 {
		return nil, fmt.Errorf("parseFloat() expects 1 argument, got %d", len(args))
	}
	arg, err := i.EvaluateExpression(args[0], env)
	if err != nil {
		return nil, err
	}
	str, ok := arg.(string)
	if !ok {
		return nil, fmt.Errorf("parseFloat() expects a string argument, got %T", arg)
	}
	str = strings.TrimSpace(str)
	result, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return nil, fmt.Errorf("parseFloat() failed to parse '%s': %v", str, err)
	}
	return result, nil
}

func builtinToString(i *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	// Convert value to string
	if len(args) != 1 {
		return nil, fmt.Errorf("toString() expects 1 argument, got %d", len(args))
	}
	arg, err := i.EvaluateExpression(args[0], env)
	if err != nil {
		return nil, err
	}
	return fmt.Sprintf("%v", arg), nil
}

func builtinAbs(i *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	// Absolute value
	if len(args) != 1 {
		return nil, fmt.Errorf("abs() expects 1 argument, got %d", len(args))
	}
	arg, err := i.EvaluateExpression(args[0], env)
	if err != nil {
		return nil, err
	}
	switch v := arg.(type) {
	case int64:
		if v == math.MinInt64 {
			return nil, fmt.Errorf("abs() overflow: cannot negate minimum int64 value")
		}
		if v < 0 {
			return -v, nil
		}
		return v, nil
	case float64:
		return math.Abs(v), nil
	default:
		return nil, fmt.Errorf("abs() expects a numeric argument, got %T", arg)
	}
}

func builtinMin(i *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	// Minimum of two values
	if len(args) != 2 {
		return nil, fmt.Errorf("min() expects 2 arguments, got %d", len(args))
	}
	leftArg, err := i.EvaluateExpression(args[0], env)
	if err != nil {
		return nil, err
	}
	rightArg, err := i.EvaluateExpression(args[1], env)
	if err != nil {
		return nil, err
	}
	switch l := leftArg.(type) {
	case int64:
		r, ok := rightArg.(int64)
		if !ok {
			return nil, fmt.Errorf("min() arguments must be same type")
		}
		if l < r {
			return l, nil
		}
		return r, nil
	case float64:
		r, ok := rightArg.(float64)
		if !ok {
			return nil, fmt.Errorf("min() arguments must be same type")
		}
		if l < r {
			return l, nil
		}
		return r, nil
	default:
		return nil, fmt.Errorf("min() expects numeric arguments, got %T", leftArg)
	}
}

func builtinMax(i *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	// Maximum of two values
	if len(args) != 2 {
		return nil, fmt.Errorf("max() expects 2 arguments, got %d", len(args))
	}
	leftArg, err := i.EvaluateExpression(args[0], env)
	if err != nil {
		return nil, err
	}
	rightArg, err := i.EvaluateExpression(args[1], env)
	if err != nil {
		return nil, err
	}
	switch l := leftArg.(type) {
	case int64:
		r, ok := rightArg.(int64)
		if !ok {
			return nil, fmt.Errorf("max() arguments must be same type")
		}
		if l > r {
			return l, nil
		}
		return r, nil
	case float64:
		r, ok := rightArg.(float64)
		if !ok {
			return nil, fmt.Errorf("max() arguments must be same type")
		}
		if l > r {
			return l, nil
		}
		return r, nil
	default:
		return nil, fmt.Errorf("max() expects numeric arguments, got %T", leftArg)
	}
}

func builtinRandomInt(i *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("randomInt() expects 2 arguments (min, max), got %d", len(args))
	}
	minArg, err := i.EvaluateExpression(args[0], env)
	if err != nil {
		return nil, err
	}
	maxArg, err := i.EvaluateExpression(args[1], env)
	if err != nil {
		return nil, err
	}
	minVal, ok := minArg.(int64)
	if !ok {
		return nil, fmt.Errorf("randomInt() expects integer arguments, got %T for min", minArg)
	}
	maxVal, ok := maxArg.(int64)
	if !ok {
		return nil, fmt.Errorf("randomInt() expects integer arguments, got %T for max", maxArg)
	}
	if minVal > maxVal {
		return nil, fmt.Errorf("randomInt() requires min <= max, got min=%d, max=%d", minVal, maxVal)
	}
	// #nosec G404 -- non-cryptographic PRNG intentional for general-purpose scripting use
	return minVal + rand.Int63n(maxVal-minVal+1), nil
}

func builtinGenerateId(_ *Interpreter, args []Expr, _ *Environment) (interface{}, error) {
	if len(args) != 0 {
		return nil, fmt.Errorf("generateId() expects 0 arguments, got %d", len(args))
	}
	return uuid.New().String(), nil
}

func builtinAppend(i *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("append() expects 2 arguments (array, item), got %d", len(args))
	}
	arrArg, err := i.EvaluateExpression(args[0], env)
	if err != nil {
		return nil, err
	}
	arr, ok := arrArg.([]interface{})
	if !ok {
		return nil, fmt.Errorf("append() expects first argument to be an array, got %T", arrArg)
	}
	item, err := i.EvaluateExpression(args[1], env)
	if err != nil {
		return nil, err
	}
	return append(arr, item), nil
}

func builtinSet(i *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	if len(args) != 3 {
		return nil, fmt.Errorf("set() expects 3 arguments (object, key, value), got %d", len(args))
	}
	objArg, err := i.EvaluateExpression(args[0], env)
	if err != nil {
		return nil, err
	}
	obj, ok := objArg.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("set() expects first argument to be an object, got %T", objArg)
	}
	keyArg, err := i.EvaluateExpression(args[1], env)
	if err != nil {
		return nil, err
	}
	key, ok := keyArg.(string)
	if !ok {
		return nil, fmt.Errorf("set() expects second argument to be a string key, got %T", keyArg)
	}
	value, err := i.EvaluateExpression(args[2], env)
	if err != nil {
		return nil, err
	}
	obj[key] = value
	return obj, nil
}

func builtinRemove(i *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("remove() expects 2 arguments (object, key), got %d", len(args))
	}
	objArg, err := i.EvaluateExpression(args[0], env)
	if err != nil {
		return nil, err
	}
	obj, ok := objArg.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("remove() expects first argument to be an object, got %T", objArg)
	}
	keyArg, err := i.EvaluateExpression(args[1], env)
	if err != nil {
		return nil, err
	}
	key, ok := keyArg.(string)
	if !ok {
		return nil, fmt.Errorf("remove() expects second argument to be a string key, got %T", keyArg)
	}
	delete(obj, key)
	return obj, nil
}

func builtinKeys(i *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("keys() expects 1 argument (object), got %d", len(args))
	}
	objArg, err := i.EvaluateExpression(args[0], env)
	if err != nil {
		return nil, err
	}
	obj, ok := objArg.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("keys() expects an object argument, got %T", objArg)
	}
	keys := make([]interface{}, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	return keys, nil
}

// callCallable invokes a callable (LambdaClosure or Function) with the given arguments.
func (i *Interpreter) callCallable(fn interface{}, args []interface{}) (interface{}, error) {
	switch f := fn.(type) {
	case *LambdaClosure:
		return i.callLambdaClosure(f, args)
	case Function:
		fnEnv := NewFunctionEnvironment(NewEnvironment())
		for idx, param := range f.Params {
			if idx < len(args) {
				fnEnv.Define(param.Name, args[idx])
			}
		}
		result, err := i.executeStatements(f.Body, fnEnv)
		if err != nil {
			if val, isReturn := unwrapReturn(err); isReturn {
				return val, nil
			}
			return nil, err
		}
		return result, nil
	case *Function:
		return i.callCallable(*f, args)
	default:
		return nil, fmt.Errorf("expected a function, got %T", fn)
	}
}

func builtinMap(i *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("map() expects 2 arguments (array, function), got %d", len(args))
	}
	arrArg, err := i.EvaluateExpression(args[0], env)
	if err != nil {
		return nil, err
	}
	arr, ok := arrArg.([]interface{})
	if !ok {
		return nil, fmt.Errorf("map() expects first argument to be an array, got %T", arrArg)
	}
	fnArg, err := i.EvaluateExpression(args[1], env)
	if err != nil {
		return nil, err
	}
	result := make([]interface{}, len(arr))
	for idx, elem := range arr {
		val, err := i.callCallable(fnArg, []interface{}{elem})
		if err != nil {
			return nil, fmt.Errorf("map() callback error at index %d: %v", idx, err)
		}
		result[idx] = val
	}
	return result, nil
}

func builtinFilter(i *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("filter() expects 2 arguments (array, function), got %d", len(args))
	}
	arrArg, err := i.EvaluateExpression(args[0], env)
	if err != nil {
		return nil, err
	}
	arr, ok := arrArg.([]interface{})
	if !ok {
		return nil, fmt.Errorf("filter() expects first argument to be an array, got %T", arrArg)
	}
	fnArg, err := i.EvaluateExpression(args[1], env)
	if err != nil {
		return nil, err
	}
	result := make([]interface{}, 0)
	for idx, elem := range arr {
		val, err := i.callCallable(fnArg, []interface{}{elem})
		if err != nil {
			return nil, fmt.Errorf("filter() callback error at index %d: %v", idx, err)
		}
		if truthy, ok := val.(bool); ok && truthy {
			result = append(result, elem)
		}
	}
	return result, nil
}

func builtinReduce(i *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	if len(args) != 3 {
		return nil, fmt.Errorf("reduce() expects 3 arguments (array, function, initial), got %d", len(args))
	}
	arrArg, err := i.EvaluateExpression(args[0], env)
	if err != nil {
		return nil, err
	}
	arr, ok := arrArg.([]interface{})
	if !ok {
		return nil, fmt.Errorf("reduce() expects first argument to be an array, got %T", arrArg)
	}
	fnArg, err := i.EvaluateExpression(args[1], env)
	if err != nil {
		return nil, err
	}
	acc, err := i.EvaluateExpression(args[2], env)
	if err != nil {
		return nil, err
	}
	for idx, elem := range arr {
		acc, err = i.callCallable(fnArg, []interface{}{acc, elem})
		if err != nil {
			return nil, fmt.Errorf("reduce() callback error at index %d: %v", idx, err)
		}
	}
	return acc, nil
}

func builtinFind(i *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("find() expects 2 arguments (array, function), got %d", len(args))
	}
	arrArg, err := i.EvaluateExpression(args[0], env)
	if err != nil {
		return nil, err
	}
	arr, ok := arrArg.([]interface{})
	if !ok {
		return nil, fmt.Errorf("find() expects first argument to be an array, got %T", arrArg)
	}
	fnArg, err := i.EvaluateExpression(args[1], env)
	if err != nil {
		return nil, err
	}
	for idx, elem := range arr {
		val, err := i.callCallable(fnArg, []interface{}{elem})
		if err != nil {
			return nil, fmt.Errorf("find() callback error at index %d: %v", idx, err)
		}
		if truthy, ok := val.(bool); ok && truthy {
			return elem, nil
		}
	}
	return nil, nil
}

func builtinSome(i *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("some() expects 2 arguments (array, function), got %d", len(args))
	}
	arrArg, err := i.EvaluateExpression(args[0], env)
	if err != nil {
		return nil, err
	}
	arr, ok := arrArg.([]interface{})
	if !ok {
		return nil, fmt.Errorf("some() expects first argument to be an array, got %T", arrArg)
	}
	fnArg, err := i.EvaluateExpression(args[1], env)
	if err != nil {
		return nil, err
	}
	for idx, elem := range arr {
		val, err := i.callCallable(fnArg, []interface{}{elem})
		if err != nil {
			return nil, fmt.Errorf("some() callback error at index %d: %v", idx, err)
		}
		if truthy, ok := val.(bool); ok && truthy {
			return true, nil
		}
	}
	return false, nil
}

func builtinEvery(i *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("every() expects 2 arguments (array, function), got %d", len(args))
	}
	arrArg, err := i.EvaluateExpression(args[0], env)
	if err != nil {
		return nil, err
	}
	arr, ok := arrArg.([]interface{})
	if !ok {
		return nil, fmt.Errorf("every() expects first argument to be an array, got %T", arrArg)
	}
	fnArg, err := i.EvaluateExpression(args[1], env)
	if err != nil {
		return nil, err
	}
	for idx, elem := range arr {
		val, err := i.callCallable(fnArg, []interface{}{elem})
		if err != nil {
			return nil, fmt.Errorf("every() callback error at index %d: %v", idx, err)
		}
		if truthy, ok := val.(bool); !ok || !truthy {
			return false, nil
		}
	}
	return true, nil
}

func builtinSort(i *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("sort() expects 1-2 arguments (array[, comparator]), got %d", len(args))
	}
	arrArg, err := i.EvaluateExpression(args[0], env)
	if err != nil {
		return nil, err
	}
	arr, ok := arrArg.([]interface{})
	if !ok {
		return nil, fmt.Errorf("sort() expects first argument to be an array, got %T", arrArg)
	}
	result := make([]interface{}, len(arr))
	copy(result, arr)

	if len(args) == 2 {
		fnArg, err := i.EvaluateExpression(args[1], env)
		if err != nil {
			return nil, err
		}
		var sortErr error
		sort.SliceStable(result, func(a, b int) bool {
			if sortErr != nil {
				return false
			}
			val, err := i.callCallable(fnArg, []interface{}{result[a], result[b]})
			if err != nil {
				sortErr = err
				return false
			}
			switch v := val.(type) {
			case int64:
				return v < 0
			case float64:
				return v < 0
			case bool:
				return v
			default:
				sortErr = fmt.Errorf("sort() comparator must return a number or boolean, got %T", val)
				return false
			}
		})
		if sortErr != nil {
			return nil, sortErr
		}
	} else {
		var sortErr error
		sort.SliceStable(result, func(a, b int) bool {
			if sortErr != nil {
				return false
			}
			return defaultLess(result[a], result[b], &sortErr)
		})
		if sortErr != nil {
			return nil, sortErr
		}
	}
	return result, nil
}

func defaultLess(a, b interface{}, errOut *error) bool {
	switch av := a.(type) {
	case int64:
		if bv, ok := b.(int64); ok {
			return av < bv
		}
	case float64:
		if bv, ok := b.(float64); ok {
			return av < bv
		}
	case string:
		if bv, ok := b.(string); ok {
			return av < bv
		}
	}
	*errOut = fmt.Errorf("sort() cannot compare %T and %T", a, b)
	return false
}

func builtinReverse(i *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("reverse() expects 1 argument, got %d", len(args))
	}
	val, err := i.EvaluateExpression(args[0], env)
	if err != nil {
		return nil, err
	}
	switch v := val.(type) {
	case string:
		runes := []rune(v)
		for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
			runes[i], runes[j] = runes[j], runes[i]
		}
		return string(runes), nil
	case []interface{}:
		result := make([]interface{}, len(v))
		for idx, elem := range v {
			result[len(v)-1-idx] = elem
		}
		return result, nil
	default:
		return nil, fmt.Errorf("reverse() expects string or array, got %T", val)
	}
}
func builtinFlat(i *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("flat() expects 1 argument (array), got %d", len(args))
	}
	arrArg, err := i.EvaluateExpression(args[0], env)
	if err != nil {
		return nil, err
	}
	arr, ok := arrArg.([]interface{})
	if !ok {
		return nil, fmt.Errorf("flat() expects an array argument, got %T", arrArg)
	}
	result := make([]interface{}, 0)
	for _, elem := range arr {
		if inner, ok := elem.([]interface{}); ok {
			result = append(result, inner...)
		} else {
			result = append(result, elem)
		}
	}
	return result, nil
}

func builtinSlice(i *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	if len(args) != 3 {
		return nil, fmt.Errorf("slice() expects 3 arguments (array, start, end), got %d", len(args))
	}
	arrArg, err := i.EvaluateExpression(args[0], env)
	if err != nil {
		return nil, err
	}
	arr, ok := arrArg.([]interface{})
	if !ok {
		return nil, fmt.Errorf("slice() expects first argument to be an array, got %T", arrArg)
	}
	startArg, err := i.EvaluateExpression(args[1], env)
	if err != nil {
		return nil, err
	}
	start, ok := startArg.(int64)
	if !ok {
		return nil, fmt.Errorf("slice() expects second argument to be an integer, got %T", startArg)
	}
	endArg, err := i.EvaluateExpression(args[2], env)
	if err != nil {
		return nil, err
	}
	end, ok := endArg.(int64)
	if !ok {
		return nil, fmt.Errorf("slice() expects third argument to be an integer, got %T", endArg)
	}
	if start < 0 {
		start = 0
	}
	if end > int64(len(arr)) {
		end = int64(len(arr))
	}
	if start > end {
		return make([]interface{}, 0), nil
	}
	result := make([]interface{}, end-start)
	copy(result, arr[start:end])
	return result, nil
}

func builtinPrint(i *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	for _, arg := range args {
		val, err := i.EvaluateExpression(arg, env)
		if err != nil {
			return nil, err
		}
		fmt.Printf("%v ", val)
	}
	fmt.Println()
	return nil, nil
}

func builtinTypeOf(i *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("typeOf() expects 1 argument, got %d", len(args))
	}
	val, err := i.EvaluateExpression(args[0], env)
	if err != nil {
		return nil, err
	}
	if val == nil {
		return "null", nil
	}
	switch v := val.(type) {
	case int64:
		return "int", nil
	case float64:
		return "float", nil
	case string:
		return "str", nil
	case bool:
		return "bool", nil
	case []interface{}:
		return "array", nil
	case map[string]interface{}:
		return "object", nil
	case *ResultValue:
		return "result", nil
	case *Future:
		return "future", nil
	default:
		return fmt.Sprintf("%T", v), nil
	}
}

func builtinShr(i *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("shr() expects 2 arguments (value, bits), got %d", len(args))
	}
	val, err := i.EvaluateExpression(args[0], env)
	if err != nil {
		return nil, err
	}
	bits, err := i.EvaluateExpression(args[1], env)
	if err != nil {
		return nil, err
	}
	valInt, ok1 := val.(int64)
	if !ok1 {
		return nil, fmt.Errorf("shr() expects first argument to be int, got %T", val)
	}
	bitsInt, ok2 := bits.(int64)
	if !ok2 {
		return nil, fmt.Errorf("shr() expects second argument to be int, got %T", bits)
	}
	return valInt >> uint(bitsInt), nil
}

func builtinBand(i *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("band() expects 2 arguments, got %d", len(args))
	}
	val, err := i.EvaluateExpression(args[0], env)
	if err != nil {
		return nil, err
	}
	mask, err := i.EvaluateExpression(args[1], env)
	if err != nil {
		return nil, err
	}
	valInt, ok1 := val.(int64)
	if !ok1 {
		return nil, fmt.Errorf("band() expects int arguments, got %T", val)
	}
	maskInt, ok2 := mask.(int64)
	if !ok2 {
		return nil, fmt.Errorf("band() expects int arguments, got %T", mask)
	}
	return valInt & maskInt, nil
}

func builtinIntToBytes4(i *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("intToBytes4() expects 1 argument, got %d", len(args))
	}
	val, err := i.EvaluateExpression(args[0], env)
	if err != nil {
		return nil, err
	}
	n, ok := val.(int64)
	if !ok {
		return nil, fmt.Errorf("intToBytes4() expects int, got %T", val)
	}
	return []interface{}{
		int64(n & 0xFF),
		int64((n >> 8) & 0xFF),
		int64((n >> 16) & 0xFF),
		int64((n >> 24) & 0xFF),
	}, nil
}

func builtinIntToBytes8(i *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("intToBytes8() expects 1 argument, got %d", len(args))
	}
	val, err := i.EvaluateExpression(args[0], env)
	if err != nil {
		return nil, err
	}
	n, ok := val.(int64)
	if !ok {
		return nil, fmt.Errorf("intToBytes8() expects int, got %T", val)
	}
	return []interface{}{
		int64(n & 0xFF),
		int64((n >> 8) & 0xFF),
		int64((n >> 16) & 0xFF),
		int64((n >> 24) & 0xFF),
		int64((n >> 32) & 0xFF),
		int64((n >> 40) & 0xFF),
		int64((n >> 48) & 0xFF),
		int64((n >> 56) & 0xFF),
	}, nil
}

func builtinWriteFile(i *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("writeFile() expects 2 arguments (path, data), got %d", len(args))
	}
	pathVal, err := i.EvaluateExpression(args[0], env)
	if err != nil {
		return nil, err
	}
	path, ok := pathVal.(string)
	if !ok {
		return nil, fmt.Errorf("writeFile() expects first argument to be a string path, got %T", pathVal)
	}

	dataVal, err := i.EvaluateExpression(args[1], env)
	if err != nil {
		return nil, err
	}

	var bytes []byte
	switch v := dataVal.(type) {
	case string:
		bytes = []byte(v)
	case []interface{}:
		bytes = make([]byte, len(v))
		for idx, item := range v {
			if n, ok := item.(int64); ok {
				bytes[idx] = byte(n)
			} else {
				return nil, fmt.Errorf("writeFile() expects data array to contain integers, got %T at index %d", item, idx)
			}
		}
	default:
		return nil, fmt.Errorf("writeFile() expects second argument to be a string or array of bytes, got %T", dataVal)
	}

	if err := os.WriteFile(path, bytes, 0644); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	return nil, nil
}

func builtinReadFile(i *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("readFile() expects 1 argument (path), got %d", len(args))
	}
	pathVal, err := i.EvaluateExpression(args[0], env)
	if err != nil {
		return nil, err
	}
	path, ok := pathVal.(string)
	if !ok {
		return nil, fmt.Errorf("readFile() expects argument to be a string path, got %T", pathVal)
	}

	bytes, err := os.ReadFile(path)
	if err != nil {
		return "", nil // Return empty string if file doesn't exist
	}

	return string(bytes), nil
}

func builtinToInt(i *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("toInt() expects 1 argument, got %d", len(args))
	}
	val, err := i.EvaluateExpression(args[0], env)
	if err != nil {
		return nil, err
	}

	switch v := val.(type) {
	case int64:
		return v, nil
	case int:
		return int64(v), nil
	case float64:
		return int64(v), nil
	case string:
		var result int64
		_, err := fmt.Sscanf(v, "%d", &result)
		if err != nil {
			return int64(0), nil
		}
		return result, nil
	case bool:
		if v {
			return int64(1), nil
		}
		return int64(0), nil
	default:
		return int64(0), nil
	}
}

func builtinText(interp *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("text() requires 1-2 arguments: text(body) or text(body, statusCode)")
	}
	bodyVal, err := interp.EvaluateExpression(args[0], env)
	if err != nil {
		return nil, err
	}
	bodyStr, ok := bodyVal.(string)
	if !ok {
		return nil, fmt.Errorf("text() first argument must be a string, got %T", bodyVal)
	}
	statusCode := 200
	if len(args) == 2 {
		codeVal, err := interp.EvaluateExpression(args[1], env)
		if err != nil {
			return nil, err
		}
		switch v := codeVal.(type) {
		case int64:
			statusCode = int(v)
		case int:
			statusCode = v
		case float64:
			statusCode = int(v)
		default:
			return nil, fmt.Errorf("text() second argument must be an integer status code, got %T", codeVal)
		}
	}
	return &TextResponse{Body: bodyStr, StatusCode: statusCode}, nil
}

func builtinHTML(interp *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("html() requires 1-2 arguments: html(body) or html(body, statusCode)")
	}
	bodyVal, err := interp.EvaluateExpression(args[0], env)
	if err != nil {
		return nil, err
	}
	bodyStr, ok := bodyVal.(string)
	if !ok {
		return nil, fmt.Errorf("html() first argument must be a string, got %T", bodyVal)
	}
	statusCode := 200
	if len(args) == 2 {
		codeVal, err := interp.EvaluateExpression(args[1], env)
		if err != nil {
			return nil, err
		}
		switch v := codeVal.(type) {
		case int64:
			statusCode = int(v)
		case int:
			statusCode = v
		case float64:
			statusCode = int(v)
		default:
			return nil, fmt.Errorf("html() second argument must be an integer status code, got %T", codeVal)
		}
	}
	return &HTMLResponse{Body: bodyStr, StatusCode: statusCode}, nil
}

func builtinBlob(interp *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	if len(args) < 2 || len(args) > 3 {
		return nil, fmt.Errorf("blob() requires 2-3 arguments: blob(data, contentType) or blob(data, contentType, statusCode)")
	}
	dataVal, err := interp.EvaluateExpression(args[0], env)
	if err != nil {
		return nil, err
	}
	// Accept string data and convert to []byte
	var data []byte
	switch v := dataVal.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	default:
		return nil, fmt.Errorf("blob() first argument must be a string or bytes, got %T", dataVal)
	}

	ctVal, err := interp.EvaluateExpression(args[1], env)
	if err != nil {
		return nil, err
	}
	contentType, ok := ctVal.(string)
	if !ok {
		return nil, fmt.Errorf("blob() second argument must be a content type string, got %T", ctVal)
	}

	statusCode := 200
	if len(args) == 3 {
		codeVal, err := interp.EvaluateExpression(args[2], env)
		if err != nil {
			return nil, err
		}
		switch v := codeVal.(type) {
		case int64:
			statusCode = int(v)
		case int:
			statusCode = v
		case float64:
			statusCode = int(v)
		default:
			return nil, fmt.Errorf("blob() third argument must be an integer status code, got %T", codeVal)
		}
	}
	return &BlobResponse{Data: data, ContentType: contentType, StatusCode: statusCode}, nil
}

func builtinRedirect(interp *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("redirect() requires 1-2 arguments: redirect(url) or redirect(url, statusCode)")
	}

	urlVal, err := interp.EvaluateExpression(args[0], env)
	if err != nil {
		return nil, err
	}
	urlStr, ok := urlVal.(string)
	if !ok {
		return nil, fmt.Errorf("redirect() first argument must be a string URL, got %T", urlVal)
	}

	statusCode := 302 // default
	if len(args) == 2 {
		codeVal, err := interp.EvaluateExpression(args[1], env)
		if err != nil {
			return nil, err
		}
		switch v := codeVal.(type) {
		case int64:
			statusCode = int(v)
		case int:
			statusCode = v
		case float64:
			statusCode = int(v)
		default:
			return nil, fmt.Errorf("redirect() second argument must be an integer status code, got %T", codeVal)
		}
	}

	if err := ValidateRedirect(urlStr, statusCode); err != nil {
		return nil, err
	}

	return &RedirectResponse{URL: urlStr, StatusCode: statusCode}, nil
}

// builtinVMExec executes bytecode through the Go VM.
// Takes an array of ints (bytecode bytes) and returns the VM result.
func builtinVMExec(interp *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("vmExec() expects 1 argument (bytecode array), got %d", len(args))
	}
	val, err := interp.EvaluateExpression(args[0], env)
	if err != nil {
		return nil, err
	}

	arr, ok := val.([]interface{})
	if !ok {
		return nil, fmt.Errorf("vmExec() expects an array of integers, got %T", val)
	}

	bytes := make([]byte, len(arr))
	for idx, item := range arr {
		n, ok := item.(int64)
		if !ok {
			return nil, fmt.Errorf("vmExec() expects integers in bytecode array, got %T at index %d", item, idx)
		}
		bytes[idx] = byte(n)
	}

	vmInstance := vmPkg.NewVM()
	result, err := vmInstance.Execute(bytes)
	if err != nil {
		return nil, fmt.Errorf("VM execution failed: %w", err)
	}

	return vmValueToInterface(result), nil
}

// builtinGPUExec executes bytecode on the GPU compute backend.
// Takes an array of ints (bytecode bytes) and optional VM count.
// Returns the result(s) from GPU execution.
func builtinGPUExec(interp *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("gpuExec() expects 1-2 arguments (bytecode array, [numVMs]), got %d", len(args))
	}
	val, err := interp.EvaluateExpression(args[0], env)
	if err != nil {
		return nil, err
	}

	arr, ok := val.([]interface{})
	if !ok {
		return nil, fmt.Errorf("gpuExec() expects an array of integers, got %T", val)
	}

	bytecodeBytes := make([]byte, len(arr))
	for idx, item := range arr {
		n, ok := item.(int64)
		if !ok {
			return nil, fmt.Errorf("gpuExec() expects integers in bytecode array, got %T at index %d", item, idx)
		}
		bytecodeBytes[idx] = byte(n)
	}


	dispatcher := gpuPkg.NewMitosisVM(4096)
	results, err := dispatcher.ExecuteWithMitosis(bytecodeBytes)
	if err != nil {
		return nil, fmt.Errorf("GPU execution failed: %w", err)
	}

	if len(results) == 1 {
		r := results[0].Result
		if r.Error != nil {
			return nil, fmt.Errorf("GPU VM error: %w", r.Error)
		}
		return gpuResultToInterface(r), nil
	}

	// Multiple VMs: return array of results
	out := make([]interface{}, len(results))
	for i, tr := range results {
		r := tr.Result
		if r.Error != nil {
			out[i] = map[string]interface{}{"error": r.Error.Error(), "steps": r.Steps}
		} else {
			out[i] = gpuResultToInterface(r)
		}
	}
	return out, nil
}

func builtinArgs(i *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	result := make([]interface{}, len(os.Args))
	for idx, arg := range os.Args {
		result[idx] = arg
	}
	return result, nil
}

func builtinExists(i *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("exists() expects 1 argument (path), got %d", len(args))
	}
	pathVal, err := i.EvaluateExpression(args[0], env)
	if err != nil {
		return nil, err
	}
	path, ok := pathVal.(string)
	if !ok {
		return nil, fmt.Errorf("exists() expects argument to be a string path, got %T", pathVal)
	}
	_, err = os.Stat(path)
	if err == nil {
		return true, nil
	}
	return false, nil
}

func gpuResultToInterface(r gpuPkg.Result) interface{} {
	fmt.Printf("[DEBUG] gpuResultToInterface: Tag=%d, IntVal=%d, Error=%v\n", r.Tag, r.IntVal, r.Error)
	switch r.Tag {
	case gpuPkg.TagInt:
		return r.IntVal
	case gpuPkg.TagFloat:
		return r.FloatVal
	case gpuPkg.TagBool:
		return r.BoolVal
	default:
		return nil
	}
}

// vmValueToInterface converts a VM value to a Go interface{} for the interpreter.
func vmValueToInterface(val vmPkg.Value) interface{} {
	if val == nil {
		return nil
	}
	switch v := val.(type) {
	case vmPkg.IntValue:
		return v.Val
	case vmPkg.FloatValue:
		return v.Val
	case vmPkg.BoolValue:
		return v.Val
	case vmPkg.StringValue:
		return v.Val
	case vmPkg.NullValue:
		return nil
	case vmPkg.ArrayValue:
		result := make([]interface{}, len(v.Val))
		for i, elem := range v.Val {
			result[i] = vmValueToInterface(elem)
		}
		return result
	case vmPkg.ObjectValue:
		result := make(map[string]interface{})
		for k, elem := range v.Val {
			result[k] = vmValueToInterface(elem)
		}
		return result
	default:
		return fmt.Sprintf("%v", val)
	}
}

func builtinRepeat(i *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("repeat() expects 2 arguments, got %d", len(args))
	}
	strArg, err := i.EvaluateExpression(args[0], env)
	if err != nil {
		return nil, err
	}
	countArg, err := i.EvaluateExpression(args[1], env)
	if err != nil {
		return nil, err
	}
	s, ok1 := strArg.(string)
	n, ok2 := countArg.(int64)
	if !ok1 || !ok2 {
		return nil, fmt.Errorf("type error in repeat()")
	}
	return strings.Repeat(s, int(n)), nil
}

func builtinFlatten(i *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("flatten() expects 1 argument")
	}
	arrArg, err := i.EvaluateExpression(args[0], env)
	if err != nil {
		return nil, err
	}
	arr, ok := arrArg.([]interface{})
	if !ok {
		return nil, fmt.Errorf("flatten() expects array")
	}
	var res []interface{}
	for _, item := range arr {
		if sub, ok := item.([]interface{}); ok {
			res = append(res, sub...)
		} else {
			res = append(res, item)
		}
	}
	return res, nil
}

func builtinRange(i *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("range() expects 1 or 2 arguments")
	}
	start := int64(0)
	var stop int64
	arg0, err := i.EvaluateExpression(args[0], env)
	if err != nil { return nil, err }
	if len(args) == 1 {
		stop = arg0.(int64)
	} else {
		start = arg0.(int64)
		arg1, err := i.EvaluateExpression(args[1], env)
		if err != nil { return nil, err }
		stop = arg1.(int64)
	}
	res := make([]interface{}, 0)
	for val := start; val < stop; val++ {
		res = append(res, val)
	}
	return res, nil
}

// builtinMutator implements __mutator(value, offset) — the self-modification primitive.
// In interpreted mode, this records a mutation that can be read back via the mutation table.
// Returns the value that was written (acting as a passthrough for chaining).
func builtinMutator(i *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("__mutator() expects 2 arguments (value, offset), got %d", len(args))
	}
	valArg, err := i.EvaluateExpression(args[0], env)
	if err != nil {
		return nil, err
	}
	offsetArg, err := i.EvaluateExpression(args[1], env)
	if err != nil {
		return nil, err
	}

	val, ok1 := valArg.(int64)
	offset, ok2 := offsetArg.(int64)
	if !ok1 || !ok2 {
		return nil, fmt.Errorf("__mutator() expects integer arguments, got %T and %T", valArg, offsetArg)
	}

	// Store the mutation in the global environment so it can be read later.
	// Walk up to the global scope so mutations are visible across function calls.
	globalEnv := env
	for globalEnv.parent != nil {
		globalEnv = globalEnv.parent
	}
	mutationsRaw, _ := globalEnv.Get("__mutations")
	var mutations map[int64]int64
	if mutationsRaw != nil {
		mutations, _ = mutationsRaw.(map[int64]int64)
	}
	if mutations == nil {
		mutations = make(map[int64]int64)
	}
	mutations[offset] = val
	globalEnv.Define("__mutations", mutations)

	if os.Getenv("GLYPH_DEBUG") == "true" {
		fmt.Printf("[MUTATOR] Wrote value %d at offset %d\n", val, offset)
	}
	return val, nil
}

// builtinMitosis implements __mitosis(spatial_offset) — the thread-fork primitive.
// In interpreted mode, this always returns true (parent path), since the interpreter
// runs single-threaded. The child concept is simulated by the mutation table.
func builtinMitosis(i *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("__mitosis() expects at least 1 argument, got %d", len(args))
	}
	_, err := i.EvaluateExpression(args[0], env)
	if err != nil {
		return nil, err
	}

	// Always return true = parent path (interpreter is single-threaded)
	return true, nil
}

func builtinSpawn(i *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("spawn() takes exactly 1 argument (spatial offset), got %d", len(args))
	}
	return true, nil
}

func builtinMutate(i *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("mutate() takes exactly 2 arguments (value, offset), got %d", len(args))
	}
	return true, nil
}

// builtinEvalSource implements eval_source(source_string) — the dynamic
// evaluation primitive for self-hosting. It parses and executes a GlyphLang
// source string, returning a map { value, error }.
//
// The source can be:
//   - A bare expression: "1 + 2" → evaluates and returns the result
//   - A function call: "print(42)" → wraps in a command body, executes, returns nil
//   - Full .glyph module: "! f() { > 5 } $ result = f()" → loads and executes
//
// Nested eval is supported: the inner interpreter also has eval_source
// registered as a builtin, so eval_source("eval_source(\"...\")") works.
func builtinEvalSource(i *Interpreter, args []Expr, env *Environment) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("eval_source() expects at least 1 argument, got %d", len(args))
	}

	srcArg, err := i.EvaluateExpression(args[0], env)
	if err != nil {
		return nil, err
	}
	src, ok := srcArg.(string)
	if !ok {
		return nil, fmt.Errorf("eval_source() expects a string argument, got %T", srcArg)
	}

	// We need a ParseFunc to parse the source. Use the interpreter's
	// moduleResolver ParseFunc if available, otherwise fall back to a
	// direct import of the parser package.
	parseFunc := i.moduleResolver.ParseFunc
	if parseFunc == nil {
		return map[string]interface{}{"value": nil, "error": "no parser available (moduleResolver.ParseFunc not set)"}, nil
	}

	// Try parsing the source as-is (valid .glyph module)
	mod, parseErr := parseFunc(src)
	if parseErr != nil {
		// Source isn't valid top-level .glyph. Try wrapping strategies:

		// Strategy 1: Wrap as a command body with a result assignment
		// (for bare expressions like "1 + 2" or function calls like "print(42)")
		wrapped1 := "@ command __eval__ {\n  $ __eval_result__ = " + src + "\n}"
		mod, parseErr = parseFunc(wrapped1)
		if parseErr != nil {
			// Strategy 2: Wrap as a plain command body (for mixed source with
			// function defs + statements like "! f() {...} $ x = f()")
			wrapped2 := "@ command __eval__ {\n  " + src + "\n}"
			mod, parseErr = parseFunc(wrapped2)
			if parseErr != nil {
				return map[string]interface{}{"value": nil, "error": fmt.Sprintf("parse error: %v", parseErr)}, nil
			}
		}
	}

	// Create a child interpreter for the evaluated source. It inherits
	// the ParseFunc so nested eval_source calls work.
	childInterp := NewInterpreter()
	childInterp.moduleResolver.ParseFunc = parseFunc

	// Load the module into the child interpreter
	if loadErr := childInterp.LoadModule(*mod); loadErr != nil {
		return map[string]interface{}{"value": nil, "error": fmt.Sprintf("load error: %v", loadErr)}, nil
	}

	// Check if there's a "__eval__" command (from wrapped expression source)
	var result interface{}
	if cmd, ok := childInterp.GetCommand("__eval__"); ok {
		var execErr error
		result, execErr = childInterp.ExecuteCommand(&cmd, nil)
		if execErr != nil {
			return map[string]interface{}{"value": nil, "error": fmt.Sprintf("exec error: %v", execErr)}, nil
		}
		// If the command set a __eval_result__ variable, use that as the value
		if evalResult, err := childInterp.globalEnv.Get("__eval_result__"); err == nil {
			result = evalResult
		}
	} else {
		// For full module source, check for a "result" variable in global env
		if val, err := childInterp.globalEnv.Get("result"); err == nil {
			result = val
		}
	}

	return map[string]interface{}{"value": result, "error": ""}, nil
}
