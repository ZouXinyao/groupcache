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

// Package singleflight provides a duplicate function call suppression
// mechanism.
// 当一个函数进行调用时（比如发起了一个请求），第一个请求正在执行，这时发过来一个相同的请求，
// 需要保证第一次请求的返回结果和第二次的一样。
package singleflight

import "sync"

// call is an in-flight or completed Do call
// 相当于一个已经完成的请求或者一个正在执行的请求。
// 完成和正在执行取决于wg
type call struct {
	wg  sync.WaitGroup	// 阻塞请求，等待goroutine其他完成
	val interface{}		// 请求的返回结果
	err error
}

// Group represents a class of work and forms a namespace in which
// units of work can be executed with duplicate suppression.
type Group struct {
	mu sync.Mutex       // protects m; 自带map不是并发安全，所以需要锁来保证原子性
	m  map[string]*call // lazily initialized; 保存处理的请求，用map可以判断两个请求是否一样。
}

// Do executes and returns the results of the given function, making
// sure that only one execution is in-flight for a given key at a
// time. If a duplicate comes in, the duplicate caller waits for the
// original to complete and receives the same results.
// 保证当前key只有一个请求正在执行，其他相同key的请求等在这次处理完，然后直接返回这次处理的结果。
func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[string]*call)
	}
	// 如果map中存在这个key了，代表有goroutine正在处理该请求，等待处理结束直接返回就行了。
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		c.wg.Wait()
		return c.val, c.err
	}
	// 该goroutine是第一个处理这个key的请求的。
	c := new(call)
	c.wg.Add(1)
	g.m[key] = c
	g.mu.Unlock()

	c.val, c.err = fn()
	// 到此为止请求已经处理完成了，接下来可以允许其他相同key的请求返回结果了。
	c.wg.Done()

	// 我这个goroutine负责把map中的请求删除，因为这一时刻的请求已经处理完成，
	// 如果不删，下次这个key的请求结果可能会变，如果返回这次的请求结果，会产生错误。
	// 同样保证m的原子性，需要锁。
	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()

	return c.val, c.err
}
