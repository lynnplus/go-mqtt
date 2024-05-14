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
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

type MessageHandler func(ctx Context)

type HandlerCancelFunc func()

// Router is an interface that routes Publish messages based on topic
type Router interface {
	Route(ctx Context)
}

// DefaultRouter is a structure that implements the Router interface
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

// Register registers a MessageHandler associated with topicFilter in the router
func (d *DefaultRouter) Register(topicFilter string, handler MessageHandler) {
	d.RegisterCancelable(topicFilter, handler)
}

// RegisterCancelable registers a MessageHandler associated with topicFilter and returns a HandlerCancelFunc.
// Calling HandlerCancelFunc can cancel the registration independently.
//
// Different from UnRegister, UnRegister will cancel all Message Handlers associated with topic Filter,
// but Handler Cancel Func only deletes itself.
func (d *DefaultRouter) RegisterCancelable(topicFilter string, handler MessageHandler) HandlerCancelFunc {
	data := &RouteData{
		handler: handler,
		rid:     d.idCounter.Add(1),
	}
	d.mutex.Lock()
	cancel := d.trie.Insert(topicFilter, data)
	d.mutex.Unlock()
	return func() {
		d.mutex.Lock()
		cancel()
		d.mutex.Unlock()
	}
}

// UnRegister receives a topicFilter and unregisters all MessageHandlers associated with the topicFilter in the router.
func (d *DefaultRouter) UnRegister(topicFilter string) {
	d.mutex.Lock()
	d.trie.Remove(topicFilter)
	d.mutex.Unlock()
}

func (d *DefaultRouter) NewRouteGroup() *RouteGroup {
	return &RouteGroup{DefaultRouter: d}
}

// RouteGroup is a structure that extends DefaultRouter and integrates the function of batch unregister.
// Created by NewRouteGroup of DefaultRouter
type RouteGroup struct {
	*DefaultRouter
	cancels []HandlerCancelFunc
	locked  atomic.Bool
}

// Register is a wrapper function for RegisterCancelable
func (r *RouteGroup) Register(topicFilter string, handler MessageHandler) {
	r.RegisterCancelable(topicFilter, handler)
}

// RegisterCancelable registers a MessageHandler associated with topicFilter with the router,
// and internally records a HandlerCancelFunc that can be used in Cancel.
func (r *RouteGroup) RegisterCancelable(topicFilter string, handler MessageHandler) HandlerCancelFunc {
	f := r.DefaultRouter.RegisterCancelable(topicFilter, handler)

	for !r.locked.CompareAndSwap(false, true) {
		runtime.Gosched()
	}
	r.cancels = append(r.cancels, f)
	r.locked.Store(false)
	return f
}

// Cancel unregisters all MessageHandlers registered on this RouteGroup
func (r *RouteGroup) Cancel() {
	for !r.locked.CompareAndSwap(false, true) {
		runtime.Gosched()
	}
	cache := make([]HandlerCancelFunc, len(r.cancels))
	copy(cache, r.cancels)
	r.cancels = r.cancels[:0]
	r.locked.Store(false)

	for _, cancelFunc := range cache {
		cancelFunc()
	}
}

func (r *RouteGroup) NewRouteGroup() *RouteGroup {
	return &RouteGroup{DefaultRouter: r.DefaultRouter}
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
