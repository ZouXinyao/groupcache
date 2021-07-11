/*
Copyright 2012 Google Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package groupcache

import (
	"bytes"
	"errors"
	"io"
	"strings"
)

// A ByteView holds an immutable view of bytes.
// Internally it wraps either a []byte or a string,
// but that detail is invisible to callers.
// 不可变的字节视图。相当于存了一个字节数组。

// A ByteView is meant to be used as a value type, not
// a pointer (like a time.Time).
// 为什么要用值类型？因为这个是视图，当然要是一个副本，不影响原数据
type ByteView struct {
	// If b is non-nil, b is used, else s is used.
	b []byte	// 优先级更高，不是nil就用b
	s string	// b是nil，就用s
}

// Len returns the view's length.
func (v ByteView) Len() int {
	if v.b != nil {
		return len(v.b)
	}
	return len(v.s)
}

// ByteSlice returns a copy of the data as a byte slice.
// 返回了一个字节数组的副本，
func (v ByteView) ByteSlice() []byte {
	if v.b != nil {
		return cloneBytes(v.b)
	}
	return []byte(v.s)
}

// String returns the data as a string, making a copy if necessary.
// 返回一个字符串的副本
func (v ByteView) String() string {
	if v.b != nil {
		return string(v.b)
	}
	return v.s
}

// At returns the byte at index i.
// 返回索引为i的字符
func (v ByteView) At(i int) byte {
	if v.b != nil {
		return v.b[i]
	}
	return v.s[i]
}

// Slice slices the view between the provided from and to indices.
// 返回一个索引从from到to的ByteView视图。
func (v ByteView) Slice(from, to int) ByteView {
	if v.b != nil {
		return ByteView{b: v.b[from:to]}
	}
	return ByteView{s: v.s[from:to]}
}

// SliceFrom slices the view from the provided index until the end.
// 返回一个索引从from到结尾的ByteView视图。
func (v ByteView) SliceFrom(from int) ByteView {
	if v.b != nil {
		return ByteView{b: v.b[from:]}
	}
	return ByteView{s: v.s[from:]}
}

// Copy copies b into dest and returns the number of bytes copied.
// 将视图中的数据copy到dest中，返回长度。
func (v ByteView) Copy(dest []byte) int {
	if v.b != nil {
		return copy(dest, v.b)
	}
	return copy(dest, v.s)
}

// Equal returns whether the bytes in b are the same as the bytes in
// b2.
// 比较两个视图是否相等
func (v ByteView) Equal(b2 ByteView) bool {
	if b2.b == nil {
		return v.EqualString(b2.s)
	}
	return v.EqualBytes(b2.b)
}

// EqualString returns whether the bytes in b are the same as the bytes
// in s.
// 比较输入的字符串s是否与v中的字符串或字节数组相等。
func (v ByteView) EqualString(s string) bool {
	if v.b == nil {
		return v.s == s
	}
	l := v.Len()
	if len(s) != l {
		return false
	}
	for i, bi := range v.b {
		if bi != s[i] {
			return false
		}
	}
	return true
}

// EqualBytes returns whether the bytes in b are the same as the bytes
// in b2.
// 比较输入的字符字节数组b2是否与v中的字符串或字节数组相等。
func (v ByteView) EqualBytes(b2 []byte) bool {
	if v.b != nil {
		return bytes.Equal(v.b, b2)
	}
	l := v.Len()
	if len(b2) != l {
		return false
	}
	for i, bi := range b2 {
		if bi != v.s[i] {
			return false
		}
	}
	return true
}

// Reader returns an io.ReadSeeker for the bytes in v.
// io.ReadSeeker支持任意位置读取的IO接口
func (v ByteView) Reader() io.ReadSeeker {
	if v.b != nil {
		return bytes.NewReader(v.b)
	}
	return strings.NewReader(v.s)
}

// ReadAt implements io.ReaderAt on the bytes in v.
// 实现io.ReaderAt接口，可以实现任意位置的读取。
func (v ByteView) ReadAt(p []byte, off int64) (n int, err error) {
	if off < 0 {
		return 0, errors.New("view: invalid offset")
	}
	if off >= int64(v.Len()) {
		return 0, io.EOF
	}
	n = v.SliceFrom(int(off)).Copy(p)
	if n < len(p) {
		err = io.EOF
	}
	return
}

// WriteTo implements io.WriterTo on the bytes in v.
// 将输入写入到w，直到没有数据需要写入。
func (v ByteView) WriteTo(w io.Writer) (n int64, err error) {
	var m int
	if v.b != nil {
		m, err = w.Write(v.b)
	} else {
		m, err = io.WriteString(w, v.s)
	}
	if err == nil && m < v.Len() {
		err = io.ErrShortWrite
	}
	n = int64(m)
	return
}
