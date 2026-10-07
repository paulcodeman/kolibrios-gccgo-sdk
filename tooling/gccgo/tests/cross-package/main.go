package main

import (
	"reflect"
	"regression/consumer"
	"regression/owner"
)

var boxes []owner.Box[int]

type intAlias = int

func hash() int { return 13 }

func main() {
	nested := consumer.Nested()
	if nested.Get() != 37 || len(nested.Slice()) != 1 || nested.Slice()[0] != 37 {
		panic("private generic named fields in imported constructor")
	}
	embedded := consumer.NewEmbedded()
	if embedded.Get() != 19 || embedded.Box.Get() != 19 {
		panic("embedded generic pointer field or promoted method")
	}
	var promoted interface{ Get() int } = embedded
	if promoted.Get() != 19 || consumer.NewEmbeddedValue().Get() != 23 {
		panic("embedded generic method set")
	}
	field := reflect.TypeOf(embedded).Field(0)
	if field.Name != "Box" || !field.Anonymous || field.PkgPath != "" || field.Tag.Get("json") != "embedded" {
		panic("embedded generic field reflection")
	}
	genericEmbedded := owner.MakeEmbedded(29)
	if genericEmbedded.Get() != 29 || genericEmbedded.Box.Get() != 29 {
		panic("generic type embedding another generic type")
	}
	privateEmbedded := owner.MakePrivateEmbedded(31)
	if privateEmbedded.Get() != 31 {
		panic("promoted method through private embedded generic field")
	}
	privateField := reflect.TypeOf(privateEmbedded).Field(0)
	if privateField.Name != "privateBox" || !privateField.Anonymous || privateField.PkgPath != "regression/owner" {
		panic("private embedded generic field visibility")
	}
	pointed := 17
	if owner.PointerRoundTrip(&pointed) != &pointed {
		panic("private bridge unsafe pointer signature")
	}
	if owner.Legacy(18).Value != 18 {
		panic("public keyed literal exposed an ordinary package private type")
	}
	if owner.Buffer(1).String() != "buffer" {
		panic("zero literal exposed a private imported field type")
	}
	var b owner.Box[int] = consumer.New().Box
	boxes = append(boxes, b)
	if b.Get() != 7 {
		panic("cross-package assignment")
	}
	dynamic, ok := consumer.Interface().(owner.Box[int])
	if !ok || dynamic.Get() != 11 {
		panic("cross-package type assertion")
	}
	if owner.Add(5) != 8 {
		panic("private constant in generic function")
	}
	if owner.Touch(5) != 7 || owner.Stored() != 6 {
		panic("private field, function and variable identity")
	}
	if owner.Read(owner.Pointer(8)) != 8 {
		panic("private struct literal address")
	}
	if owner.Variadic(5) != 10 {
		panic("private variadic function signature")
	}
	if owner.Digest("a") != 3826002220 || hash() != 13 {
		panic("generated import names collide")
	}
	var alias owner.Box[intAlias] = b
	if alias.Get() != 7 {
		panic("alias specialization identity")
	}
	reflected := reflect.TypeOf(consumer.Interface())
	if reflected.Name() != "Box[int]" || reflected.PkgPath() != "regression/owner" || reflected.String() != "owner.Box[int]" {
		panic("generic reflection identity")
	}
	tagged := owner.Make(struct {
		Value int `json:"a"`
	}{Value: 1})
	if _, ok := any(tagged).(owner.Box[struct {
		Value int `json:"b"`
	}]); ok {
		panic("struct tags erased from generic identity")
	}
	println("PASS: generic identity, aliases, tags, reflection, methods and private state across packages")
}
