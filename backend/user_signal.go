// Generated by: main
// TypeWriter: signal
// Directive: +gen on User

package main

import (
	"sync"
)

// Generated from Signal (https://github.com/jackc/signal)
// Copyright 2015 Jack Christensen
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// The primary type that represents a signal
type UserSignal struct {
	listeners [](chan User)
	mutex     sync.Mutex
}

// Add channel c to the signal to receive messages from this Signal
func (s *UserSignal) Add(c chan User) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.listeners = append(s.listeners, c)
}

// Remove channel c from the signal
func (s *UserSignal) Remove(c chan User) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for i, l := range s.listeners {
		if c == l {
			s.listeners[i] = s.listeners[len(s.listeners)-1]
			s.listeners = s.listeners[:len(s.listeners)-1]
			return
		}
	}
}

// Dispatch synchronously sends msg to all channels that have been added to this signal.
func (s *UserSignal) Dispatch(msg User) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, l := range s.listeners {
		l <- msg
	}
}
