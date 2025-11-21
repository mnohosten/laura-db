package query

import (
	"fmt"
	"reflect"
	"regexp"
)

// Operator represents a query operator
type Operator string

const (
	// Comparison operators
	OpEqual              Operator = "$eq"
	OpNotEqual           Operator = "$ne"
	OpGreaterThan        Operator = "$gt"
	OpGreaterThanOrEqual Operator = "$gte"
	OpLessThan           Operator = "$lt"
	OpLessThanOrEqual    Operator = "$lte"
	OpIn                 Operator = "$in"
	OpNotIn              Operator = "$nin"

	// Logical operators
	OpAnd Operator = "$and"
	OpOr  Operator = "$or"
	OpNot Operator = "$not"

	// Element operators
	OpExists Operator = "$exists"
	OpType   Operator = "$type"

	// Evaluation operators
	OpRegex Operator = "$regex"
	OpMod   Operator = "$mod"

	// Array operators
	OpAll      Operator = "$all"
	OpElemMatch Operator = "$elemMatch"
	OpSize     Operator = "$size"
)

// EvaluateOperator evaluates an operator expression
func EvaluateOperator(op Operator, fieldValue interface{}, operatorValue interface{}) (bool, error) {
	switch op {
	case OpEqual:
		return evaluateEqual(fieldValue, operatorValue), nil
	case OpNotEqual:
		return !evaluateEqual(fieldValue, operatorValue), nil
	case OpGreaterThan:
		return evaluateGreaterThan(fieldValue, operatorValue), nil
	case OpGreaterThanOrEqual:
		return evaluateGreaterThanOrEqual(fieldValue, operatorValue), nil
	case OpLessThan:
		return evaluateLessThan(fieldValue, operatorValue), nil
	case OpLessThanOrEqual:
		return evaluateLessThanOrEqual(fieldValue, operatorValue), nil
	case OpIn:
		return evaluateIn(fieldValue, operatorValue), nil
	case OpNotIn:
		return !evaluateIn(fieldValue, operatorValue), nil
	case OpExists:
		exists := fieldValue != nil
		if boolVal, ok := operatorValue.(bool); ok {
			return exists == boolVal, nil
		}
		return false, fmt.Errorf("$exists requires boolean value")
	case OpRegex:
		return evaluateRegex(fieldValue, operatorValue)
	case OpSize:
		return evaluateSize(fieldValue, operatorValue), nil
	case OpElemMatch:
		return evaluateElemMatch(fieldValue, operatorValue)
	default:
		return false, fmt.Errorf("unsupported operator: %s", op)
	}
}

// evaluateEqual checks if two values are equal
func evaluateEqual(a, b interface{}) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Try direct comparison first
	if reflect.DeepEqual(a, b) {
		return true
	}

	// Handle numeric comparisons across types
	aVal, aOk := toFloat64(a)
	bVal, bOk := toFloat64(b)
	if aOk && bOk {
		return aVal == bVal
	}

	return false
}

// evaluateGreaterThan checks if a > b
func evaluateGreaterThan(a, b interface{}) bool {
	aVal, aOk := toFloat64(a)
	bVal, bOk := toFloat64(b)
	if aOk && bOk {
		return aVal > bVal
	}

	// String comparison
	aStr, aOk := a.(string)
	bStr, bOk := b.(string)
	if aOk && bOk {
		return aStr > bStr
	}

	return false
}

// evaluateGreaterThanOrEqual checks if a >= b
func evaluateGreaterThanOrEqual(a, b interface{}) bool {
	return evaluateGreaterThan(a, b) || evaluateEqual(a, b)
}

// evaluateLessThan checks if a < b
func evaluateLessThan(a, b interface{}) bool {
	aVal, aOk := toFloat64(a)
	bVal, bOk := toFloat64(b)
	if aOk && bOk {
		return aVal < bVal
	}

	// String comparison
	aStr, aOk := a.(string)
	bStr, bOk := b.(string)
	if aOk && bOk {
		return aStr < bStr
	}

	return false
}

// evaluateLessThanOrEqual checks if a <= b
func evaluateLessThanOrEqual(a, b interface{}) bool {
	return evaluateLessThan(a, b) || evaluateEqual(a, b)
}

// evaluateIn checks if value is in the array
func evaluateIn(value interface{}, array interface{}) bool {
	arrVal := reflect.ValueOf(array)
	if arrVal.Kind() != reflect.Slice && arrVal.Kind() != reflect.Array {
		return false
	}

	for i := 0; i < arrVal.Len(); i++ {
		if evaluateEqual(value, arrVal.Index(i).Interface()) {
			return true
		}
	}

	return false
}

// evaluateRegex matches a regex pattern
func evaluateRegex(value interface{}, pattern interface{}) (bool, error) {
	str, ok := value.(string)
	if !ok {
		return false, nil
	}

	patternStr, ok := pattern.(string)
	if !ok {
		return false, fmt.Errorf("regex pattern must be a string")
	}

	matched, err := regexp.MatchString(patternStr, str)
	if err != nil {
		return false, fmt.Errorf("invalid regex pattern: %w", err)
	}

	return matched, nil
}

// evaluateSize checks array size
func evaluateSize(value interface{}, size interface{}) bool {
	arrVal := reflect.ValueOf(value)
	if arrVal.Kind() != reflect.Slice && arrVal.Kind() != reflect.Array {
		return false
	}

	expectedSize, ok := toInt64(size)
	if !ok {
		return false
	}

	return int64(arrVal.Len()) == expectedSize
}

// toFloat64 converts a value to float64
func toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int32:
		return float64(val), true
	case int64:
		return float64(val), true
	case uint:
		return float64(val), true
	case uint32:
		return float64(val), true
	case uint64:
		return float64(val), true
	default:
		return 0, false
	}
}

// toInt64 converts a value to int64
func toInt64(v interface{}) (int64, bool) {
	switch val := v.(type) {
	case int:
		return int64(val), true
	case int32:
		return int64(val), true
	case int64:
		return val, true
	case uint:
		return int64(val), true
	case uint32:
		return int64(val), true
	case float64:
		return int64(val), true
	default:
		return 0, false
	}
}

// evaluateElemMatch checks if array contains element matching all conditions
func evaluateElemMatch(value interface{}, conditions interface{}) (bool, error) {
	// Value must be an array
	arrVal := reflect.ValueOf(value)
	if arrVal.Kind() != reflect.Slice && arrVal.Kind() != reflect.Array {
		return false, nil
	}

	// Conditions must be a map of operators
	condMap, ok := conditions.(map[string]interface{})
	if !ok {
		return false, fmt.Errorf("$elemMatch requires an object with conditions")
	}

	// Check each array element
	for i := 0; i < arrVal.Len(); i++ {
		element := arrVal.Index(i).Interface()
		matchesAll := true

		// Element must match ALL conditions
		for opStr, opValue := range condMap {
			op := Operator(opStr)

			// Evaluate the operator against this element
			matches, err := EvaluateOperator(op, element, opValue)
			if err != nil {
				return false, err
			}

			if !matches {
				matchesAll = false
				break
			}
		}

		// If this element matches all conditions, return true
		if matchesAll {
			return true, nil
		}
	}

	// No element matched all conditions
	return false, nil
}
