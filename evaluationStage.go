package govaluate

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

const (
	logicalErrorFormat    string = "Value '%v' cannot be used with the logical operator '%v', it is not a bool"
	modifierErrorFormat   string = "Value '%v' cannot be used with the modifier '%v', it is not a number"
	comparatorErrorFormat string = "Value '%v' cannot be used with the comparator '%v', it is not a number"
	ternaryErrorFormat    string = "Value '%v' cannot be used with the ternary operator '%v', it is not a bool"
	prefixErrorFormat     string = "Value '%v' cannot be used with the prefix '%v'"
)

type evaluationOperator func(left interface{}, right interface{}, parameters Parameters) (interface{}, error)
type stageTypeCheck func(value interface{}) bool
type stageCombinedTypeCheck func(left interface{}, right interface{}) bool

type evaluationStage struct {
	symbol OperatorSymbol

	leftStage, rightStage *evaluationStage

	// the operation that will be used to evaluate this stage (such as adding [left] to [right] and return the result)
	operator evaluationOperator

	// ensures that both left and right values are appropriate for this stage. Returns an error if they aren't operable.
	leftTypeCheck  stageTypeCheck
	rightTypeCheck stageTypeCheck

	// if specified, will override whatever is used in "leftTypeCheck" and "rightTypeCheck".
	// primarily used for specific operators that don't care which side a given type is on, but still requires one side to be of a given type
	// (like string concat)
	typeCheck stageCombinedTypeCheck

	// regardless of which type check is used, this string format will be used as the error message for type errors
	typeErrorFormat string
}

var (
	_true  = interface{}(true)
	_false = interface{}(false)
)

func (this *evaluationStage) swapWith(other *evaluationStage) {

	temp := *other
	other.setToNonStage(*this)
	this.setToNonStage(temp)
}

func (this *evaluationStage) setToNonStage(other evaluationStage) {

	this.symbol = other.symbol
	this.operator = other.operator
	this.leftTypeCheck = other.leftTypeCheck
	this.rightTypeCheck = other.rightTypeCheck
	this.typeCheck = other.typeCheck
	this.typeErrorFormat = other.typeErrorFormat
}

func (this *evaluationStage) isShortCircuitable() bool {

	switch this.symbol {
	case AND:
		fallthrough
	case OR:
		fallthrough
	case TERNARY_TRUE:
		fallthrough
	case TERNARY_FALSE:
		fallthrough
	case COALESCE:
		return true
	}

	return false
}

func noopStageRight(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {
	return right, nil
}

func addStage(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {

	// string concat if either are strings
	if isString(left) || isString(right) {
		return fmt.Sprintf("%v%v", left, right), nil
	}

	return left.(float64) + right.(float64), nil
}
func subtractStage(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {
	return left.(float64) - right.(float64), nil
}
func multiplyStage(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {
	return left.(float64) * right.(float64), nil
}
func divideStage(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {
	return left.(float64) / right.(float64), nil
}
func exponentStage(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {
	return math.Pow(left.(float64), right.(float64)), nil
}
func modulusStage(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {
	return math.Mod(left.(float64), right.(float64)), nil
}
func gteStage(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {
	var err error
	left, right, err = _uniformDataType(left, right)
	if err != nil {
		return nil, err
	}
	if isString(left) && isString(right) {
		return boolIface(left.(string) >= right.(string)), nil
	}
	return boolIface(left.(float64) >= right.(float64)), nil
}
func gtStage(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {
	var err error
	left, right, err = _uniformDataType(left, right)
	if err != nil {
		return nil, err
	}
	if isString(left) && isString(right) {
		return boolIface(left.(string) > right.(string)), nil
	}
	return boolIface(left.(float64) > right.(float64)), nil
}
func lteStage(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {
	var err error
	left, right, err = _uniformDataType(left, right)
	if err != nil {
		return nil, err
	}
	if isString(left) && isString(right) {
		return boolIface(left.(string) <= right.(string)), nil
	}
	return boolIface(left.(float64) <= right.(float64)), nil
}
func ltStage(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {
	var err error
	left, right, err = _uniformDataType(left, right)
	if err != nil {
		return nil, err
	}
	if isString(left) && isString(right) {
		return boolIface(left.(string) < right.(string)), nil
	}
	return boolIface(left.(float64) < right.(float64)), nil
}

// 尝试统一化数据类型，比如 left="123", right=123 时，将left转换为123
// @modified yorkershi
func _uniformDataType(left, right any) (rLeft any, rRight any, err error) {
	//1. 如果已经同时为 string 或 float64 类型，直接返回
	if (isString(left) && isString(right)) || (isFloat64(left) && isFloat64(right)) {
		return left, right, nil
	}

	//2. 如果同时为数值类型，则统一转换为 float64 返回
	if v1, ok1 := _tryNumericToFloat64(left); ok1 {
		if v2, ok2 := _tryNumericToFloat64(right); ok2 {
			return v1, v2, nil
		}
	}

	//3. 如果一边为string, 另一边为数值类型，则先尝试将string转换为float64，如果不能转换，则将数值类型转换为string
	if isString(left) {
		if v, ok := _tryNumericToFloat64(right); ok {
			rLeft, rRight = _assignType(left.(string), v)
			return rLeft, rRight, nil
		}
	}
	if isString(right) {
		if v, ok := _tryNumericToFloat64(left); ok {
			rRight, rLeft = _assignType(right.(string), v)
			return rLeft, rRight, nil
		}
	}

	//4. 如果一条为bool类型，另一边为string类型时，尝试将bool类型转换为string类型返回
	if isBool(left) && isString(right) {
		return strconv.FormatBool(left.(bool)), right, nil
	} else if isString(left) && isBool(right) {
		return left, strconv.FormatBool(right.(bool)), nil
	}

	return left, right, nil
}

// 是否所以传入的数据均为数值类型
func _tryNumericToFloat64(data any) (f float64, isSuccess bool) {
	if v, ok := data.(int); ok {
		return float64(v), true
	}
	if v, ok := data.(uint); ok {
		return float64(v), true
	}
	if v, ok := data.(int8); ok {
		return float64(v), true
	}
	if v, ok := data.(uint8); ok {
		return float64(v), true
	}
	if v, ok := data.(int16); ok {
		return float64(v), true
	}
	if v, ok := data.(uint16); ok {
		return float64(v), true
	}
	if v, ok := data.(int32); ok {
		return float64(v), true
	}
	if v, ok := data.(uint32); ok {
		return float64(v), true
	}
	if v, ok := data.(int64); ok {
		return float64(v), true
	}
	if v, ok := data.(uint64); ok {
		return float64(v), true
	}
	if v, ok := data.(float32); ok {
		return float64(v), true
	}
	if v, ok := data.(float64); ok {
		return v, true
	}
	return 0, false

}

// 类型对齐
// 则先尝试将string转换为float64，如果不能转换，则将数值类型转换为string
// 返回相同的数据类型
func _assignType(s string, f float64) (any, any) {
	if sf, err := strconv.ParseFloat(s, 64); err == nil {
		//成功转换时
		return sf, f
	} else {
		//无法转换时
		return s, fmt.Sprintf("%v", f)
	}
}

func equalStage(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {
	//兼容 "True" == "true", "False" == "false" 的形式
	//modified by yorkershi
	if leftv := _getBoolStr(left); len(leftv) > 0 {
		if rightv := _getBoolStr(right); len(rightv) > 0 {
			left = leftv
			right = rightv
		}
	}

	//弱类型兼容
	var err error
	left, right, err = _uniformDataType(left, right)
	if err != nil {
		return nil, err
	}

	return boolIface(reflect.DeepEqual(left, right)), nil
}
func _getBoolStr(data any) string {
	if v, ok := data.(string); ok {
		v = strings.ToLower(v)
		if v == "true" || v == "false" {
			return v
		}
	}
	return ""
}
func notEqualStage(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {
	//新增弱类型兼容
	//modified by yorkershi
	var err error
	left, right, err = _uniformDataType(left, right)
	if err != nil {
		return nil, err
	}
	return boolIface(!reflect.DeepEqual(left, right)), nil
}
func andStage(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {
	return boolIface(left.(bool) && right.(bool)), nil
}
func orStage(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {
	return boolIface(left.(bool) || right.(bool)), nil
}
func negateStage(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {
	return -right.(float64), nil
}
func invertStage(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {
	return boolIface(!right.(bool)), nil
}
func bitwiseNotStage(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {
	return float64(^int64(right.(float64))), nil
}
func ternaryIfStage(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {
	if left.(bool) {
		return right, nil
	}
	return nil, nil
}
func ternaryElseStage(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {
	if left != nil {
		return left, nil
	}
	return right, nil
}

func regexStage(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {

	var pattern *regexp.Regexp
	var err error

	switch right.(type) {
	case string:
		pattern, err = regexp.Compile(right.(string))
		if err != nil {
			return nil, errors.New(fmt.Sprintf("Unable to compile regexp pattern '%v': %v", right, err))
		}
	case *regexp.Regexp:
		pattern = right.(*regexp.Regexp)
	}

	return pattern.Match([]byte(left.(string))), nil
}

func notRegexStage(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {

	ret, err := regexStage(left, right, parameters)
	if err != nil {
		return nil, err
	}

	return !(ret.(bool)), nil
}

func bitwiseOrStage(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {
	return float64(int64(left.(float64)) | int64(right.(float64))), nil
}
func bitwiseAndStage(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {
	return float64(int64(left.(float64)) & int64(right.(float64))), nil
}
func bitwiseXORStage(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {
	return float64(int64(left.(float64)) ^ int64(right.(float64))), nil
}
func leftShiftStage(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {
	return float64(uint64(left.(float64)) << uint64(right.(float64))), nil
}
func rightShiftStage(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {
	return float64(uint64(left.(float64)) >> uint64(right.(float64))), nil
}

func makeParameterStage(parameterName string) evaluationOperator {

	return func(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {
		value, err := parameters.Get(parameterName)
		if err != nil {
			return nil, err
		}

		return value, nil
	}
}

func makeLiteralStage(literal interface{}) evaluationOperator {
	return func(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {
		return literal, nil
	}
}

func makeFunctionStage(function ExpressionFunction) evaluationOperator {
	return func(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {
		if right == nil {
			return function()
		}

		needSingleParameter := false
		for k, v := range customFunctions {
			if reflect.ValueOf(v).Pointer() == reflect.ValueOf(function).Pointer() {
				if _, ok := singleParamFuncNames[k]; ok {
					needSingleParameter = true
				}
			}
		}

		switch right.(type) {
		case []any:
			if needSingleParameter {
				return function(right)
			} else {
				args := right.([]any)
				if len(args) > 0 && fmt.Sprintf("%v", args[0]) == CustomFunctionFirstParamPlaceholder {
					//移除第一个参数
					tmp := make([]any, 0)
					for i, v := range args {
						if i == 0 {
							continue
						}
						tmp = append(tmp, v)
					}
					args = tmp
				}
				return function(args...)
			}
		default:
			return function(right)
		}
	}
}

func typeConvertParam(p reflect.Value, t reflect.Type) (ret reflect.Value, err error) {
	defer func() {
		if r := recover(); r != nil {
			errorMsg := fmt.Sprintf("Argument type conversion failed: failed to convert '%s' to '%s'", p.Kind().String(), t.Kind().String())
			err = errors.New(errorMsg)
			ret = p
		}
	}()

	return p.Convert(t), nil
}

func typeConvertParams(method reflect.Value, params []reflect.Value) ([]reflect.Value, error) {

	methodType := method.Type()
	numIn := methodType.NumIn()
	numParams := len(params)

	if numIn != numParams {
		if numIn > numParams {
			return nil, fmt.Errorf("Too few arguments to parameter call: got %d arguments, expected %d", len(params), numIn)
		}
		return nil, fmt.Errorf("Too many arguments to parameter call: got %d arguments, expected %d", len(params), numIn)
	}

	for i := 0; i < numIn; i++ {
		t := methodType.In(i)
		p := params[i]
		pt := p.Type()

		if t.Kind() != pt.Kind() {
			np, err := typeConvertParam(p, t)
			if err != nil {
				return nil, err
			}
			params[i] = np
		}
	}

	return params, nil
}

func makeAccessorStage(pair []string) evaluationOperator {

	reconstructed := strings.Join(pair, ".")

	return func(left interface{}, right interface{}, parameters Parameters) (ret interface{}, err error) {

		var params []reflect.Value

		value, err := parameters.Get(pair[0])
		if err != nil {
			return nil, err
		}

		// while this library generally tries to handle panic-inducing cases on its own,
		// accessors are a sticky case which have a lot of possible ways to fail.
		// therefore every call to an accessor sets up a defer that tries to recover from panics, converting them to errors.
		defer func() {
			if r := recover(); r != nil {
				errorMsg := fmt.Sprintf("Failed to access '%s': %v", reconstructed, r.(string))
				err = errors.New(errorMsg)
				ret = nil
			}
		}()

		for i := 1; i < len(pair); i++ {

			coreValue := reflect.ValueOf(value)

			var corePtrVal reflect.Value

			// if this is a pointer, resolve it.
			if coreValue.Kind() == reflect.Ptr {
				corePtrVal = coreValue
				coreValue = coreValue.Elem()
			}

			if coreValue.Kind() != reflect.Struct {
				return nil, errors.New("Unable to access '" + pair[i] + "', '" + pair[i-1] + "' is not a struct")
			}

			field := coreValue.FieldByName(pair[i])
			if field != (reflect.Value{}) {
				value = field.Interface()
				continue
			}

			method := coreValue.MethodByName(pair[i])
			if method == (reflect.Value{}) {
				if corePtrVal.IsValid() {
					method = corePtrVal.MethodByName(pair[i])
				}
				if method == (reflect.Value{}) {
					return nil, errors.New("No method or field '" + pair[i] + "' present on parameter '" + pair[i-1] + "'")
				}
			}

			switch right.(type) {
			case []interface{}:

				givenParams := right.([]interface{})
				params = make([]reflect.Value, len(givenParams))
				for idx, _ := range givenParams {
					params[idx] = reflect.ValueOf(givenParams[idx])
				}

			default:

				if right == nil {
					params = []reflect.Value{}
					break
				}

				params = []reflect.Value{reflect.ValueOf(right.(interface{}))}
			}

			params, err = typeConvertParams(method, params)

			if err != nil {
				return nil, errors.New("Method call failed - '" + pair[0] + "." + pair[1] + "': " + err.Error())
			}

			returned := method.Call(params)
			retLength := len(returned)

			if retLength == 0 {
				return nil, errors.New("Method call '" + pair[i-1] + "." + pair[i] + "' did not return any values.")
			}

			if retLength == 1 {

				value = returned[0].Interface()
				continue
			}

			if retLength == 2 {

				errIface := returned[1].Interface()
				err, validType := errIface.(error)

				if validType && errIface != nil {
					return returned[0].Interface(), err
				}

				value = returned[0].Interface()
				continue
			}

			return nil, errors.New("Method call '" + pair[0] + "." + pair[1] + "' did not return either one value, or a value and an error. Cannot interpret meaning.")
		}

		value = castToFloat64(value)
		return value, nil
	}
}

func separatorStage(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {

	var ret []interface{}

	switch left.(type) {
	case []interface{}:
		//note by: yorkershi
		//如果自定义函数传入的第一个参数是一个list，这里的处理逻辑会将list中的元素打散后传给自定义函数，这是不符合预期的！
		//因此，自定义函数的第一个参数要确保不是一个列表
		ret = append(left.([]interface{}), right)
	default:
		ret = []interface{}{left, right}
	}

	return ret, nil
}

func inStage(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {
	for _, value := range right.([]interface{}) {
		//统一转换为string再进行比较
		if fmt.Sprintf("%v", left) == fmt.Sprintf("%v", value) {
			return true, nil
		}
	}
	return false, nil
}

//

func isString(value interface{}) bool {

	switch value.(type) {
	case string:
		return true
	}
	return false
}

func isRegexOrString(value interface{}) bool {

	switch value.(type) {
	case string:
		return true
	case *regexp.Regexp:
		return true
	}
	return false
}

func isBool(value interface{}) bool {
	switch value.(type) {
	case bool:
		return true
	}
	return false
}

func isFloat64(value interface{}) bool {
	switch value.(type) {
	case float64:
		return true
	}
	return false
}

/*
Addition usually means between numbers, but can also mean string concat.
String concat needs one (or both) of the sides to be a string.
*/
func additionTypeCheck(left interface{}, right interface{}) bool {
	if isFloat64(left) && isFloat64(right) {
		return true
	}
	if !isString(left) && !isString(right) {
		return false
	}
	return true
}

/*
Comparison can either be between numbers, or lexicographic between two strings,
but never between the two.
*/
func comparatorTypeCheck(left interface{}, right interface{}) bool {
	if isFloat64(left) && isFloat64(right) {
		return true
	}
	if isString(left) && isString(right) {
		return true
	}

	//针对类似 left为 "123", right为 123 的情况，字符串可以正常转换为 float64时，也允许通过检测
	if isFloat64(left) && isString(right) {
		if _, err := strconv.ParseFloat(fmt.Sprintf("%v", right), 64); err == nil {
			return true
		}
	}
	if isFloat64(right) && isString(left) {
		if _, err := strconv.ParseFloat(fmt.Sprintf("%v", left), 64); err == nil {
			return true
		}
	}

	return false
}

func isArray(value interface{}) bool {
	switch value.(type) {
	case []interface{}:
		return true
	}
	return false
}

/*
Converting a boolean to an interface{} requires an allocation.
We can use interned bools to avoid this cost.
*/
func boolIface(b bool) interface{} {
	if b {
		return _true
	}
	return _false
}
