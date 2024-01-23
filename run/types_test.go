package run

import "context"

type MyTestType struct{}

func (m *MyTestType) Test1(_ string)                {}
func (m *MyTestType) Test2(_, _ string)             {}
func (m *MyTestType) Test3(_ string, _ int, _ bool) {}
func (m *MyTestType) Test4(_, _, _, _ string)       {}
func (m *MyTestType) Test5(_, _, _, _, _ string)    {}
func (m *MyTestType) Test6(_, _, _, _, _, _ string) {}

type MyTestTypeErr struct{}

func (m *MyTestTypeErr) Test1(_ string) error {
	return nil
}

func (m *MyTestTypeErr) Test2(_, _ string) error {
	return nil
}

func (m *MyTestTypeErr) Test3(_ string, _ int, _ bool) error {
	return nil
}

func (m *MyTestTypeErr) Test4(_, _, _, _ string) error {
	return nil
}

func (m *MyTestTypeErr) Test5(_, _, _, _, _ string) error {
	return nil
}

func (m *MyTestTypeErr) Test6(_, _, _, _, _, _ string) error {
	return nil
}

type MyTestTypeCtxErr struct{}

func (m *MyTestTypeCtxErr) Test1(_ context.Context, _ string) error {
	return nil
}

func (m *MyTestTypeCtxErr) Test2(_ context.Context, _, _ string) error {
	return nil
}

func (m *MyTestTypeCtxErr) Test3(_ context.Context, _ string, _ int, _ bool) error {
	return nil
}

func (m *MyTestTypeCtxErr) Test4(_ context.Context, _, _, _, _ string) error {
	return nil
}

func (m *MyTestTypeCtxErr) Test5(_ context.Context, _, _, _, _, _ string) error {
	return nil
}

func (m *MyTestTypeCtxErr) Test6(_ context.Context, _, _, _, _, _, _ string) error {
	return nil
}

type MyTestTypeCtx struct{}

func (m *MyTestTypeCtx) Test1(_ context.Context, _ string)                {}
func (m *MyTestTypeCtx) Test2(_ context.Context, _, _ string)             {}
func (m *MyTestTypeCtx) Test3(_ context.Context, _ string, _ int, _ bool) {}
func (m *MyTestTypeCtx) Test4(_ context.Context, _, _, _, _ string)       {}
func (m *MyTestTypeCtx) Test5(_ context.Context, _, _, _, _, _ string)    {}
func (m *MyTestTypeCtx) Test6(_ context.Context, _, _, _, _, _, _ string) {}

func myFunc() {}
func myFuncErr() error {
	return nil
}
func myFuncCtx(_ context.Context) {}
func myFuncCtxErr(_ context.Context) error {
	return nil
}

func test1(_ context.Context, _ string) error {
	return nil
}

func test2(_ context.Context, _, _ string) error {
	return nil
}

func test3(_ context.Context, _ string, _ int, _ bool) error {
	return nil
}

func test4(_ context.Context, _, _, _, _ string) error {
	return nil
}

func test5(_ context.Context, _, _, _, _, _ string) error {
	return nil
}

func test6(_ context.Context, _, _, _, _, _, _ string) error {
	return nil
}
