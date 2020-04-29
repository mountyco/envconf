package envconf

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
)

func Load(v interface{}) error {
	if reflect.TypeOf(v).Kind() != reflect.Ptr {
		return fmt.Errorf("interface should be of pointer kind")
	}

	orig := reflect.ValueOf(v).Elem()
	return load(orig)
}

func load(orig reflect.Value) error {
	switch orig.Kind() {
	case reflect.Struct:
		ot := reflect.TypeOf(orig.Interface())

		for i := 0; i < ot.NumField(); i++ {
			f := ot.Field(i)
			tagVal, fieldGotEnvTag := f.Tag.Lookup("env")

			if f.Type.Kind() == reflect.Ptr {
				if fieldGotEnvTag {
					return fmt.Errorf("field %s, is of kind %s, must not contain any env tag", f.Name, f.Type.Kind().String())
				}
				err := Load(orig.Field(i).Interface())
				if err != nil {
					return err
				}
				continue
			}

			if f.Type.Kind() == reflect.Struct {
				if fieldGotEnvTag {
					return fmt.Errorf("field %s, is of kind %s, must not contain any env tag", f.Name, f.Type.Kind().String())
				}
				err := load(orig.Field(i))
				if err != nil {
					return err
				}
				continue
			}

			if !fieldGotEnvTag {
				continue
			}
			ok := fieldKindValidForEnv(f)
			if !ok {
				return fmt.Errorf("field %s, is of kind %s, must not contain any env tag", f.Name, f.Type.Kind().String())
			}

			// Chek if any default value is set
			defaulVal, ok := f.Tag.Lookup("envdefault")
			if !ok {
				defaulVal = ""
			}
			err := setVar(orig.Field(i), f.Type.Kind(), tagVal, defaulVal)
			if err != nil {
				return fmt.Errorf("env var %s could not parsed into field %s of kind %s", tagVal, f.Name, f.Type.Kind())
			}
		}
	}
	return nil
}

func fieldKindValidForEnv(f reflect.StructField) bool {
	invalidKind := []reflect.Kind{
		reflect.Array,
		reflect.Chan,
		reflect.Func,
		reflect.Interface,
		reflect.Map,
		reflect.Slice,
		reflect.UnsafePointer,
		reflect.Ptr,
		reflect.Struct,
	}

	for _, k := range invalidKind {
		if f.Type.Kind() == k {
			return false
		}
	}

	return true
}

func setVar(v reflect.Value, vkind reflect.Kind, varKey string, defaulVal string) error {
	envVar, ok := os.LookupEnv(varKey)
	if !ok {
		if defaulVal != "" {
			envVar = defaulVal
		} else {
			return nil
		}
	}

	switch vkind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		integer, err := strconv.ParseInt(envVar, 10, 64)
		if err != nil {
			return err
		}
		v.SetInt(integer)

	case reflect.Float32, reflect.Float64:
		float, err := strconv.ParseFloat(envVar, 64)
		if err != nil {
			return err
		}
		v.SetFloat(float)

	case reflect.Bool:
		if envVar == "true" {
			v.SetBool(true)
		} else if envVar == "false" {
			v.SetBool(false)
		} else {
			return fmt.Errorf("could not parse env var %s into bool", varKey)
		}

	case reflect.String:
		v.SetString(envVar)
	}

	return nil
}
