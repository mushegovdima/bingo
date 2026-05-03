package notifications

import (
	"fmt"
	"reflect"
	"strings"
)

// ArgsOf extracts all leaf field values from n as a flat map of
// "Prefix.FieldName" → "value". The key format matches the placeholder
// syntax used in template bodies, so Substitute can apply them directly.
// Named struct fields are recursed with a "FieldName." prefix;
// anonymous (embedded) struct fields are recursed without adding a prefix.
func ArgsOf(n Notification) map[string]string {
	t := reflect.TypeOf(n)
	v := reflect.ValueOf(n)
	if t.Kind() == reflect.Pointer {
		t, v = t.Elem(), v.Elem()
	}
	args := make(map[string]string)
	collectArgs(t, v, "", args)
	return args
}

func collectArgs(t reflect.Type, v reflect.Value, prefix string, args map[string]string) {
	for i := range t.NumField() {
		f := t.Field(i)
		fv := v.Field(i)
		if f.Type.Kind() == reflect.Struct {
			if f.Anonymous {
				collectArgs(f.Type, fv, prefix, args)
			} else {
				collectArgs(f.Type, fv, prefix+f.Name+".", args)
			}
			continue
		}
		if _, ok := f.Tag.Lookup("label"); !ok {
			continue
		}
		args[prefix+f.Name] = fmt.Sprintf("%v", fv.Interface())
	}
}

// Substitute replaces all {{Key}} placeholders in body with the corresponding
// values from args. It is the single substitution entry point used by the worker.
func Substitute(body string, args map[string]string) string {
	pairs := make([]string, 0, len(args)*2)
	for k, val := range args {
		pairs = append(pairs, "{{"+k+"}}", val)
	}
	return strings.NewReplacer(pairs...).Replace(body)
}

// UserArgs extracts the NotificationUser fields as a flat args map
// with the "User." prefix, e.g. {"User.Name": "Alice", "User.Username": "alice"}.
// Use this to merge with notification args before calling Substitute.
func (u *NotificationUser) UserArgs() map[string]string {
	t := reflect.TypeOf(*u)
	v := reflect.ValueOf(*u)
	args := make(map[string]string, t.NumField())
	for i := range t.NumField() {
		args["User."+t.Field(i).Name] = fmt.Sprintf("%v", v.Field(i).Interface())
	}
	return args
}

// SubstituteUser replaces all {{User.FieldName}} placeholders in body with the
// corresponding field values of u, using reflection.
func (u *NotificationUser) SubstituteUser(body string) string {
	t := reflect.TypeOf(*u)
	v := reflect.ValueOf(*u)
	for i := range t.NumField() {
		body = strings.ReplaceAll(body,
			"{{User."+t.Field(i).Name+"}}",
			fmt.Sprintf("%v", v.Field(i).Interface()),
		)
	}
	return body
}

// VarsOf reflects over a notification struct and collects TemplateVar entries
// for every leaf field with a `label:` tag. Named struct fields are recursed
// with a "FieldName." prefix; anonymous (embedded) struct fields are recursed
// without adding a prefix.
func VarsOf(n Notification) []TemplateVar {
	t := reflect.TypeOf(n)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return collectVars(t, "")
}

func collectVars(t reflect.Type, prefix string) []TemplateVar {
	var vars []TemplateVar
	for i := range t.NumField() {
		f := t.Field(i)
		if f.Type.Kind() == reflect.Struct {
			var nested []TemplateVar
			if f.Anonymous {
				nested = collectVars(f.Type, prefix)
			} else {
				nested = collectVars(f.Type, prefix+f.Name+".")
			}
			vars = append(vars, nested...)
			continue
		}
		label, ok := f.Tag.Lookup("label")
		if !ok {
			continue
		}
		vars = append(vars, TemplateVar{Key: prefix + f.Name, Label: label})
	}
	return vars
}
