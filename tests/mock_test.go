package tests

import (
	"webtyp.com/model"
	"webtyp.com/router"
)

type mockCaller struct {
	lastOp   router.Operation
	lastArgs model.Encodable
	onCall   func(op router.Operation, args model.Encodable, into model.Decodable) error
}

func (m *mockCaller) Call(op router.Operation, args model.Encodable, into model.Decodable, done func(err error)) {
	m.lastOp = op
	m.lastArgs = args
	if m.onCall != nil {
		err := m.onCall(op, args, into)
		done(err)
	} else {
		done(nil)
	}
}

func (m *mockCaller) Dispatch(op router.Operation, args model.Encodable) {
	m.lastOp = op
	m.lastArgs = args
}
