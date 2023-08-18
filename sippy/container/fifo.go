// Package sippy_container Copyright (c) 2015 Andrii Pylypenko. All rights reserved.
//
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without modification,
// are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice, this
// list of conditions and the following disclaimer.
//
// 2. Redistributions in binary form must reproduce the above copyright notice,
// this list of conditions and the following disclaimer in the documentation and/or
// other materials provided with the distribution.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR
// ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON
// ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
package sippy_container

type FifoNode struct {
	next  *FifoNode
	Value interface{}
}

type Fifo interface {
	Put(interface{})
	Get() *FifoNode
	IsEmpty() bool
}

type fifo struct {
	first *FifoNode
	last  *FifoNode
}

func NewFifo(first, last *FifoNode) Fifo {
	return &fifo{first: first, last: last}
}

func (f *fifo) Put(v interface{}) {
	node := &FifoNode{next: nil, Value: v}
	if f.last != nil {
		f.last.next = node
		f.last = node
	} else {
		f.first = node
		f.last = node
	}
}

func (f *fifo) Get() *FifoNode {
	node := f.first
	if node != nil {
		f.first = node.next
		node.next = nil
	}
	if f.first == nil {
		f.last = nil
	}
	return node
}

func (f *fifo) IsEmpty() bool {
	return f.first == nil
}
