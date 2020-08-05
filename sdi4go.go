package sdi4go

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/libs4go/errors"
)

// errors .
var (
	ErrName                 = errors.New("object name duplicate")
	ErrNotFound             = errors.New("inject object name not found")
	ErrSingletonConstructor = errors.New("bind function must only provide one option of Singleton or Contructor")
	ErrFactoryType          = errors.New("factory type must be func()(Type,error)")
	ErrSingletonType        = errors.New("singleton type must be struct ptr")
	ErrInjectObject         = errors.New("inject object must be struct ptr")
	ErrInjectField          = errors.New("inject object field must public and must be struct ptr or interface")
	ErrBindType             = errors.New("bind type mismatch inject target field")
	ErrSliceType            = errors.New("CreateAll expect slice ptr which elem type must be interface or struct ptr")
	ErrObjectPtr            = errors.New("Create func expect struct ptr or interface")
)

// Debug debug flag , open debug sdi4go will print debug message into console
var Debug = false

func debug(fmtstr string, args ...interface{}) {
	if Debug {
		println(fmt.Sprintf(fmtstr, args...))
	}
}

// Factory .
type Factory interface{}

type objectRegister struct {
	name      string
	factory   Factory
	objType   reflect.Type
	ptrType   reflect.Type
	singleton interface{}
}

func (register *objectRegister) checkOptions() error {
	if register.factory == nil {

		if register.singleton == nil {
			return ErrSingletonConstructor
		}

		register.objType = reflect.TypeOf(register.singleton)

		if register.objType.Kind() != reflect.Ptr {
			return ErrSingletonType
		}

		register.ptrType = register.objType

		register.objType = register.objType.Elem()

		if register.objType.Kind() != reflect.Struct {
			return ErrSingletonType
		}

		return nil
	}

	factoryType := reflect.TypeOf(register.factory)

	if factoryType.Kind() != reflect.Func {
		return ErrFactoryType
	}

	if factoryType.NumIn() != 0 {
		return ErrFactoryType
	}

	if factoryType.NumOut() != 2 {
		return ErrFactoryType
	}

	if factoryType.Out(0).Kind() != reflect.Ptr || factoryType.Out(0).Elem().Kind() != reflect.Struct {
		return ErrFactoryType
	}

	errorInterface := reflect.TypeOf((*error)(nil)).Elem()

	if !factoryType.Out(1).Implements(errorInterface) {
		return ErrFactoryType
	}

	register.ptrType = factoryType.Out(0).Elem()
	register.objType = register.ptrType.Elem()

	return nil
}

func (register *objectRegister) create() (interface{}, error) {
	if register.singleton != nil {
		return register.singleton, nil
	}

	if register.factory == nil {
		return nil, ErrSingletonConstructor
	}

	results := reflect.ValueOf(register.factory).Call(nil)

	if results[1].IsNil() {
		return results[0].Interface(), nil
	}

	return nil, results[1].Interface().(error)
}

// Option .
type Option func(register *objectRegister)

// Singleton bind a singleton class
func Singleton(object interface{}) Option {
	return func(register *objectRegister) {
		register.singleton = object
	}
}

// Constructor bind class with object factory
func Constructor(f Factory) Option {
	return func(register *objectRegister) {
		register.factory = f
	}
}

// Injector the golang inject objects sdi4go factory
type Injector interface {
	// bind named object creator
	Bind(name string, options ...Option) error
	// Create object with name
	Create(name string, objectPtr interface{}) error
	// inject object fields with tag `inject:"object name"`
	Inject(object interface{}) error
	// CreateAll create special type objects slice
	// this function will search all bind class which must the provide slice type
	CreateAll(objectSlice interface{}) error
}

type sdi4goImpl struct {
	registers map[string]*objectRegister
}

// New create a new golang struct sdi4go
func New() Injector {
	return &sdi4goImpl{
		registers: make(map[string]*objectRegister),
	}
}

func (impl *sdi4goImpl) Bind(name string, options ...Option) error {

	if _, ok := impl.registers[name]; ok {
		return ErrName
	}

	register := &objectRegister{
		name: name,
	}

	for _, opt := range options {
		opt(register)
	}

	if err := register.checkOptions(); err != nil {
		return err
	}

	impl.registers[name] = register

	return nil
}

func (impl *sdi4goImpl) Inject(object interface{}) error {
	injector, err := newInjector(object)

	if err != nil {
		return err
	}

	return injector.Inject(impl)
}

func (impl *sdi4goImpl) Create(name string, objectPtr interface{}) error {

	objType := reflect.TypeOf(objectPtr)

	if reflect.TypeOf(objType).Kind() != reflect.Ptr {
		return ErrObjectPtr
	}

	objType = objType.Elem()

	if register, ok := impl.registers[name]; ok {

		if objType.Kind() == reflect.Interface {
			if !register.ptrType.Implements(objType) {
				return ErrBindType
			}
		} else if objType.Kind() == reflect.Ptr && objType.Elem().Kind() == reflect.Struct {
			if register.objType != objType.Elem() {
				debug("create type with name %s is type %s, but the bind target is %s", name, register.objType, objType)
				return ErrBindType
			}
		} else {
			return ErrObjectPtr
		}

		val, err := register.create()

		if err != nil {
			return err
		}

		reflect.ValueOf(objectPtr).Elem().Set(reflect.ValueOf(val))

		println(fmt.Sprintf("found %s val %p", name, val))

		return nil
	}

	println("not found: " + name)

	return ErrNotFound
}

func (impl *sdi4goImpl) CreateAll(objectSlice interface{}) error {

	sliceType := reflect.TypeOf(objectSlice)

	debug("CreateAll input type: %s", sliceType.String())

	if sliceType.Kind() != reflect.Ptr {
		return ErrSliceType
	}

	sliceType = sliceType.Elem()

	if sliceType.Kind() != reflect.Slice {
		return ErrSliceType
	}

	elemType := sliceType.Elem()

	var storaged []interface{}

	if elemType.Kind() == reflect.Interface {

		debug("CreateAll slice element is interface %s", elemType)

		for _, register := range impl.registers {
			debug("check register(%s) with type %s", register.name, register.objType.String())

			if register.ptrType.Implements(elemType) {

				debug("register(%s) with type %s implement %s", register.name, register.objType.String(), elemType.String())

				val, err := register.create()

				if err != nil {
					return err
				}

				storaged = append(storaged, val)
			}
		}

	} else if elemType.Kind() == reflect.Ptr && elemType.Elem().Kind() == reflect.Struct {
		for _, register := range impl.registers {
			if register.objType == elemType.Elem() {
				val, err := register.create()

				if err != nil {
					return err
				}

				storaged = append(storaged, val)
			}
		}
	} else {
		return ErrSliceType
	}

	if len(storaged) > 0 {
		sliceValue := reflect.MakeSlice(reflect.SliceOf(elemType), len(storaged), len(storaged))

		for i := 0; i < len(storaged); i++ {
			sliceValue.Index(i).Set(reflect.ValueOf(storaged[i]))
		}

		reflect.ValueOf(objectSlice).Elem().Set(sliceValue)
	}

	return nil
}

type injector struct {
	fields []*injectField
}

type injectField struct {
	tag   string
	field reflect.Value
}

func newInjector(object interface{}) (*injector, error) {
	injector := &injector{}

	objType := reflect.TypeOf(object)

	if objType.Kind() != reflect.Ptr || objType.Elem().Kind() != reflect.Struct {
		return nil, ErrInjectObject
	}

	objType = objType.Elem()

	objValue := reflect.ValueOf(object).Elem()

	for i := 0; i < objType.NumField(); i++ {

		field := objType.Field(i)

		tagStr, ok := field.Tag.Lookup("inject")

		if !ok {
			continue
		}

		debug("find type %s inject field %s", objType, field.Name)

		if field.Type.Kind() != reflect.Interface &&
			(field.Type.Kind() == reflect.Ptr && field.Type.Elem().Kind() != reflect.Struct) {
			debug("inject field %s must be struct ptr or interface", field.Name)

			return nil, ErrInjectField
		}

		if strings.ToTitle(field.Name[:1]) != field.Name[:1] {
			debug("inject field %s must be public", field.Name)
			return nil, ErrInjectField
		}

		injector.fields = append(injector.fields, &injectField{
			tag:   tagStr,
			field: objValue.Field(i),
		})
	}

	return injector, nil
}

func (impl *injector) Inject(sdi4go *sdi4goImpl) error {
	for _, field := range impl.fields {

		register, ok := sdi4go.registers[field.tag]

		if !ok {
			return errors.Wrap(ErrNotFound, "can't find service %s", field.tag)
		}

		if field.field.Type().Kind() == reflect.Interface {
			if !register.ptrType.Implements(field.field.Type()) {
				return errors.Wrap(ErrBindType, "service %s can't cast %v to %v", field.tag, register.ptrType, field.field.Type())
			}
		} else {
			if field.field.Type().Elem() != register.objType {
				return errors.Wrap(ErrBindType, "service %s type error", field.tag)
			}
		}

		err := sdi4go.Create(field.tag, field.field.Addr().Interface())

		if err != nil {
			return err
		}
	}

	return nil
}
