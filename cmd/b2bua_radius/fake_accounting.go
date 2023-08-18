package main

type fakeAccounting struct {
}

func NewFakeAccounting() *fakeAccounting {
	return &fakeAccounting{}
}

/*
class FakeAccounting(object):
    def __init__(s, *args):
        pass

    def conn(s, *args):
        pass

    def disc(s, *args):
        pass
*/
