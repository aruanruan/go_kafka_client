/**
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 * 
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package go_kafka_client

import (
	"github.com/jimlawless/cfg"
	"reflect"
	"math/rand"
	"sync"
	"container/ring"
	"hash/fnv"
	log "github.com/cihub/seelog"
	"fmt"
	"time"
)

var Logger, _ = log.LoggerFromConfigAsFile("seelog.xml")

func Trace(contextName interface{}, message interface{}) {
	Logger.Trace(fmt.Sprintf("[%s] %s", contextName, message))
}
func Tracef(contextName interface{}, message interface{}, params ...interface{}) {
	Logger.Tracef(fmt.Sprintf("[%s] %s", contextName, message), params...)
}

func Debug(contextName interface{}, message interface{}) {
	Logger.Debug(fmt.Sprintf("[%s] %s", contextName, message))
}
func Debugf(contextName interface{}, message interface{}, params ...interface{}) {
	Logger.Debugf(fmt.Sprintf("[%s] %s", contextName, message), params...)
}

func Info(contextName interface{}, message interface{}) {
	Logger.Info(fmt.Sprintf("[%s] %s", contextName, message))
}
func Infof(contextName interface{}, message interface{}, params ...interface{}) {
	Logger.Infof(fmt.Sprintf("[%s] %s", contextName, message), params...)
}

func Warn(contextName interface{}, message interface{}) {
	Logger.Warn(fmt.Sprintf("[%s] %s", contextName, message))
}
func Warnf(contextName interface{}, message interface{}, params ...interface{}) {
	Logger.Warnf(fmt.Sprintf("[%s] %s", contextName, message), params...)
}

func Error(contextName interface{}, message interface{}) {
	Logger.Error(fmt.Sprintf("[%s] %s", contextName, message))
}
func Errorf(contextName interface{}, message interface{}, params ...interface{}) {
	Logger.Errorf(fmt.Sprintf("[%s] %s", contextName, message), params...)
}

func Critical(contextName interface{}, message interface{}) {
	Logger.Critical(fmt.Sprintf("[%s] %s", contextName, message))
}
func Criticalf(contextName interface{}, message interface{}, params ...interface{}) {
	Logger.Criticalf(fmt.Sprintf("[%s] %s", contextName, message), params...)
}

func LoadConfiguration(path string) (map[string]string, error) {
	cfgMap := make(map[string]string)
	err := cfg.Load(path, cfgMap)

	return cfgMap, err
}

func InLock(lock *sync.Mutex, fun func()) {
	lock.Lock()
	defer lock.Unlock()

	fun()
}

func ShuffleArray(src interface{}, dest interface{}) {
	rSrc := reflect.ValueOf(src).Elem()
	rDest := reflect.ValueOf(dest).Elem()

	perm := rand.Perm(rSrc.Len())
	for i, v := range perm {
		rDest.Index(v).Set(rSrc.Index(i))
	}
}

func CircularIterator(src interface{}) *ring.Ring {
	arr := reflect.ValueOf(src).Elem()
	circle := ring.New(arr.Len())
	for i := 0; i < arr.Len(); i++ {
		circle.Value = arr.Index(i).Interface()
		circle = circle.Next()
	}

	return circle
}

func Position(haystack interface {}, needle interface {}) int {
	rSrc := reflect.ValueOf(haystack).Elem()
	for position := 0; position < rSrc.Len(); position++ {
		if (reflect.DeepEqual(rSrc.Index(position).Interface(), needle)) {
			return position
		}
	}

	return -1
}

func Hash(s string) int32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return int32(h.Sum32())
}

func RedirectChannelsTo(inputChannels interface{}, outputChannel interface{}) chan bool {
	killChannel, _ := RedirectChannelsToWithTimeout(inputChannels, outputChannel, 0 * time.Second)
	return killChannel
}

func RedirectChannelsToWithTimeout(inputChannels interface{}, outputChannel interface{}, timeout time.Duration) (chan bool, <-chan time.Time) {
	input := reflect.ValueOf(inputChannels)
	var timeoutInputChannel <-chan time.Time
	if timeout.Seconds() == 0 {
		timeoutInputChannel = nil
	} else {
		timeoutInputChannel = time.After(timeout)
	}
	output := reflect.ValueOf(outputChannel)
	timeoutOutputChannel := make(chan time.Time)
	killChannel := make(chan bool)

	if input.Kind() != reflect.Array && input.Kind() != reflect.Slice {
		panic("Incorrect input channels type")
	}

	if output.Kind() != reflect.Chan {
		panic("Incorrect output channel type")
	}


	cases := make([]reflect.SelectCase, input.Len())
	for i := 0; i < input.Len(); i++ {
		if input.Index(i).Kind() == reflect.Ptr {
			cases[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(input.Index(i).Elem().Interface())}
		} else {
			cases[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(input.Index(i).Interface())}
		}
	}
	cases = append(cases, reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(killChannel)})
	if timeoutInputChannel != nil {
		cases = append(cases, reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(timeoutInputChannel)})
	}

	go func() {
		for {
			remaining := len(cases)
			for remaining > 0 {
				chosen, value, ok := reflect.Select(cases)
				if !ok {
					// The chosen channel has been closed, so zero out the channel to disable the case
					cases[chosen].Chan = reflect.ValueOf(nil)
					remaining -= 1
					continue
				}

				if cases[chosen].Chan.Interface() == killChannel {
					return
				}

				if cases[chosen].Chan.Interface() == timeoutInputChannel {
					timeoutOutputChannel <- value.Interface().(time.Time)
				}

				output.Send(value)
			}
		}
	}()

	return killChannel, timeoutOutputChannel
}
