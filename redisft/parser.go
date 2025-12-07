package redisft

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

func fillStructsFromFTSearch(ctx context.Context, rdb RedisClient, index, query string, destSlicePtr interface{}) error {
	rv := reflect.ValueOf(destSlicePtr)
	if rv.Kind() != reflect.Ptr || rv.Elem().Kind() != reflect.Slice {
		return errors.New("destSlicePtr must be pointer to a slice")
	}
	sliceVal := rv.Elem()
	elemType := sliceVal.Type().Elem()

	res, err := rdb.Do(ctx,
		"FT.SEARCH", index, query,
		"LIMIT", "0", "100",
	).Result()
	if err != nil {
		return err
	}
	rows, ok := res.([]interface{})
	if !ok {
		return errors.New("FT.SEARCH: invalid response format")
	}
	if len(rows) < 2 {
		return nil
	}

	for i := 1; i < len(rows); i += 2 {
		fieldsArr, ok := rows[i+1].([]interface{})
		if !ok {
			return fmt.Errorf("invalid fields array at %d", i+1)
		}
		m := make(map[string]interface{}, len(fieldsArr)/2)
		for j := 0; j < len(fieldsArr); j += 2 {
			key, _ := fieldsArr[j].(string)
			m[key] = fieldsArr[j+1]
		}

		newElem := reflect.New(elemType).Elem()
		if err := fillStruct(newElem, m); err != nil {
			return err
		}
		sliceVal.Set(reflect.Append(sliceVal, newElem))
	}
	return nil
}

func structToMap(data interface{}) (map[string]interface{}, error) {
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil, errors.New("input not struct")
	}

	timeType := reflect.TypeOf(time.Time{})

	fields := make(map[string]interface{})
	for i := 0; i < v.NumField(); i++ {
		fieldVal := v.Field(i)
		if !fieldVal.CanInterface() {
			continue
		}

		fieldType := v.Type().Field(i)
		key := strings.ToLower(fieldType.Name)

		value := fieldVal.Interface()
		if reflect.DeepEqual(value, reflect.Zero(fieldVal.Type()).Interface()) {
			continue
		}

		if fieldVal.Type() == timeType {
			t := value.(time.Time)
			fields[key] = t.Unix()
		} else {
			fields[key] = value
		}
	}
	return fields, nil
}

func getStructName(data interface{}) string {
	t := reflect.TypeOf(data)

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Name()
}

func generateIndexQuery(input any) []any {
	v := reflect.ValueOf(input)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		panic("input not struct")
	}
	t := v.Type()
	name := strings.ToLower(t.Name())

	args := []any{"idx:" + name, "ON", "HASH", "PREFIX", 1, name + ":", "SCHEMA"}
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if tag := sf.Tag.Get("redis"); tag != "" {
			args = append(args, strings.ToLower(sf.Name))
			for _, tok := range strings.Fields(tag) {
				args = append(args, strings.ToUpper(tok))
			}
		}
	}

	return args
}

func fillStruct(v reflect.Value, m map[string]interface{}) error {
	timeType := reflect.TypeOf(time.Time{})

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		structField := v.Type().Field(i)
		if !field.CanSet() {
			continue
		}

		key := strings.ToLower(structField.Name)
		val, exists := m[key]
		if !exists {
			continue
		}

		if field.Type() == timeType {
			switch t := val.(type) {
			case string:
				if parsed, err := time.Parse(time.RFC3339, t); err == nil {
					field.Set(reflect.ValueOf(parsed))
				} else if sec, err2 := strconv.ParseInt(t, 10, 64); err2 == nil {
					field.Set(reflect.ValueOf(time.Unix(sec, 0)))
				} else {
					return fmt.Errorf("time parse %q: %v / %v", key, err, err2)
				}
			case float64:
				field.Set(reflect.ValueOf(time.Unix(int64(t), 0)))
			case int64:
				field.Set(reflect.ValueOf(time.Unix(t, 0)))
			case time.Time:
				field.Set(reflect.ValueOf(t))
			default:
				return fmt.Errorf("unsupported time type %T for %q", val, key)
			}
			continue
		}

		switch field.Kind() {
		case reflect.String:
			if s, ok := val.(string); ok {
				field.SetString(s)
			}
		case reflect.Bool:
			var b bool
			switch t := val.(type) {
			case string:
				b, _ = strconv.ParseBool(t)
			case bool:
				b = t
			}
			field.SetBool(b)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			var n int64
			switch t := val.(type) {
			case string:
				n, _ = strconv.ParseInt(t, 10, 64)
			case float64:
				n = int64(t)
			case int64:
				n = t
			}
			field.SetInt(n)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			var n uint64
			switch t := val.(type) {
			case string:
				n, _ = strconv.ParseUint(t, 10, 64)
			case float64:
				n = uint64(t)
			case int64:
				n = uint64(t)
			}
			field.SetUint(n)
		case reflect.Float32, reflect.Float64:
			var f float64
			switch t := val.(type) {
			case string:
				f, _ = strconv.ParseFloat(t, 64)
			case float64:
				f = t
			}
			field.SetFloat(f)
		default:
			valRv := reflect.ValueOf(val)
			if valRv.Type().ConvertibleTo(field.Type()) {
				field.Set(valRv.Convert(field.Type()))
			}
		}
	}
	return nil
}

func fillStructFromSlice(A interface{}, arr []interface{}) error {
	v := reflect.ValueOf(A)
	if v.Kind() != reflect.Ptr {
		return fmt.Errorf("must be a pointer to a struct or slice")
	}

	elem := v.Elem()
	switch elem.Kind() {
	case reflect.Struct:
		if len(arr) == 0 {
			return nil
		}
		m, ok := arr[0].(map[string]interface{})
		if !ok {
			return fmt.Errorf("the first element of arr must be of type map[string]interface{}")
		}
		return fillStruct(elem, m)

	case reflect.Slice:
		elemType := elem.Type().Elem()

		newSlice := reflect.MakeSlice(elem.Type(), 0, len(arr))
		for _, item := range arr {
			m, ok := item.(map[string]interface{})
			if !ok {
				return fmt.Errorf("all elements inside arr must be of type map[string]interface{}")
			}

			newElem := reflect.New(elemType).Elem()
			if err := fillStruct(newElem, m); err != nil {
				return err
			}
			newSlice = reflect.Append(newSlice, newElem)
		}

		elem.Set(newSlice)
		return nil

	default:
		return errors.New("A must be a pointer to a struct or slice")
	}
}
