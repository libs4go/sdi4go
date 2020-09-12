package sdi4go

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func init() {
	Debug = true
}

type testA struct {
	V int
}

func (a *testA) SayHello() {

}

type testB struct {
	A  *testA `inject:"a1"`
	A1 *testA `inject:"a2"`
}

type Hello interface {
	SayHello()
}

type Property struct {
	name  string
	value string
}

func TestInjectorProperty(t *testing.T) {
	sdi4go := New()

	p := &Property{name: "test", value: "name"}

	require.NoError(t, sdi4go.Bind("p", Singleton(p)))

	var p1 Property

	require.NoError(t, sdi4go.Create("p", &p1))

	require.Equal(t, p1.name, p.name)

	require.Equal(t, p1.value, p.value)
}
func TestInjectorGet(t *testing.T) {

	sdi4go := New()

	a11 := &testA{1}
	a22 := &testA{2}

	require.NotEqual(t, a11, a22)

	require.NoError(t, sdi4go.Bind("a1", Singleton(a11)))
	require.NoError(t, sdi4go.Bind("a2", Singleton(a22)))

	var sayHello []Hello

	require.NoError(t, sdi4go.CreateAll(&sayHello))

	require.Equal(t, 2, len(sayHello))

	var sayHello2 Hello

	require.NoError(t, sdi4go.Create("a1", &sayHello2))

	require.NotNil(t, sayHello2)

	var a1 *testA

	require.NoError(t, sdi4go.Create("a1", &a1))

	require.NotNil(t, a1)

	require.Equal(t, a11, a1)

	var a2 *testA

	require.NoError(t, sdi4go.Create("a2", &a2))

	require.NotNil(t, a2)

	require.Equal(t, a22, a2)

	require.NotEqual(t, a1, a2)

	require.True(t, sdi4go.Create("test", &a1) == ErrNotFound)

	a := make([]*testA, 0)

	require.NoError(t, sdi4go.CreateAll(&a))

	require.Equal(t, 2, len(a))

	b := &testB{}

	require.NoError(t, sdi4go.Inject(b))

	require.Equal(t, b.A, a1)
	require.Equal(t, b.A1, a2)

}
