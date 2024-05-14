/*
 * Copyright (c) 2024 Lynn <lynnplus90@gmail.com>
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package mqtt

import (
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

type MessageHandler func(ctx Context)

type Router interface {
	Route(ctx Context)
}

type DefaultRouter struct {
	trie      *RouteNode
	idCounter atomic.Uint64
	mutex     sync.RWMutex
}

func NewDefaultRouter() *DefaultRouter {
	return &DefaultRouter{
		trie: &RouteNode{},
	}
}

// Register 向路由中添加一个对topicFilter的消息处理器，并返回一个处理器的注册id，
func (d *DefaultRouter) Register(topicFilter string, handler MessageHandler) uint64 {
	data := &RouteData{
		handler: handler,
		rid:     d.idCounter.Add(1),
	}
	d.mutex.Lock()
	d.trie.Insert(topicFilter, data)
	d.mutex.Unlock()
	return data.rid
}

// UnRegister
func (d *DefaultRouter) UnRegister(topicFilter string) {
	d.mutex.Lock()
	d.trie.Remove(topicFilter)
	d.mutex.Unlock()
}

func (d *DefaultRouter) UnRegisterHandler(rid uint64) {

}

func (d *DefaultRouter) NewHandlerGroup(rid uint64) {

}

func (d *DefaultRouter) Route(ctx Context) {
	d.mutex.RLock()
	rs := d.trie.Search(ctx.Topic())
	d.mutex.RUnlock()

	go func() {
		sort.SliceStable(rs, func(i, j int) bool {
			return rs[i].rid < rs[j].rid
		})
		for _, temp := range rs {
			if temp == nil || temp.handler == nil {
				continue
			}
			temp.handler(ctx)
		}
	}()
}

type RouteData struct {
	handler MessageHandler
	rid     uint64
}

type RouteNode struct {
	name     string
	children map[string]*RouteNode
	data     []*RouteData
}

func (r *RouteNode) Insert(route string, data *RouteData) func() {
	rs := strings.Split(route, "/")
	node := r

	var ok bool
	for _, s := range rs {
		if node.children == nil {
			node.children = make(map[string]*RouteNode)
		}
		if _, ok = node.children[s]; !ok {
			d := &RouteNode{
				name: s,
				data: make([]*RouteData, 0),
			}
			node.children[s] = d
			node = d
		} else {
			node = node.children[s]
		}
	}
	node.data = append(node.data, data)
	return func() {
		if node != nil {
			node.removeData(data.rid)
		}
	}
}

func (r *RouteNode) Remove(route string) {
	if route == "" || route == "/" {
		return
	}
	rs := strings.Split(route, "/")
	r.removeMatched(rs)
	return
}

func (r *RouteNode) removeMatched(rs []string) {
	if len(rs) == 0 {
		r.data = nil
		return
	}
	if len(r.children) == 0 {
		return
	}

	node := r.children[rs[0]]
	if node == nil {
		return
	}
	node.removeMatched(rs[1:])

	if len(node.data) == 0 && len(node.children) == 0 {
		delete(r.children, rs[0])
	}
}

func (r *RouteNode) removeData(rid uint64) {
	if len(r.data) == 0 {
		return
	}
	tgt := r.data[:0]
	for _, d := range r.data {
		if d.rid == rid {
			continue
		}
		tgt = append(tgt, d)
	}
	r.data = tgt
}

func (r *RouteNode) Search(route string) []*RouteData {
	rs := strings.Split(route, "/")
	ns := make([]*RouteData, 0, 10)
	r.match(rs, &ns)
	return ns
}

func (r *RouteNode) match(ts []string, ns *[]*RouteData) {
	if len(r.children) == 0 {
		if len(ts) == 0 && len(r.data) > 0 {
			r.fillData(ns)
			return
		}
		return
	}

	if n, ok := r.children["#"]; ok {
		n.match(nil, ns)
	}

	if len(ts) == 0 {
		r.fillData(ns)
		return
	}

	if n, ok := r.children["+"]; ok {
		n.match(ts[1:], ns)
	}

	if n, ok := r.children[ts[0]]; ok {
		n.match(ts[1:], ns)
	}
}

func (r *RouteNode) fillData(ns *[]*RouteData) {
	if len(r.data) == 0 {
		return
	}
	*ns = append(*ns, r.data...)
}
