package tests

import "webtyp.com/fmt"

// testIDGen is the composition-root double for model.IDGenerator
type testIDGen struct{ n int }

func (g *testIDGen) NewID() string {
	g.n++
	return "test-id-" + fmt.Convert(g.n).String()
}
