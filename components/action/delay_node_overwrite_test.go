/*
 * Copyright 2026 The RuleGo Authors.
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

package action

import (
	"testing"
	"time"

	"github.com/rulego/rulego/api/types"
	"github.com/rulego/rulego/test"
	"github.com/rulego/rulego/test/assert"
)

// In overwrite mode OnMsg reads LastPendingMsgId first and only then takes the
// lock to replace the pending message. If the delay timer delivers that message
// in between, the entry is already gone and LastPendingMsgId already cleared. The
// replacement used to be written back under the dead id anyway: no timer exists
// for it, so the message was never delivered, and the entry stayed in PendingMsgs
// for good, using up a slot of maxPendingMsgs.
//
// The test sets up the state such a message observes (an id that is no longer
// pending) and checks that the message starts a new delay cycle instead.
func TestDelayNodeOverwriteStalePendingId(t *testing.T) {
	node, err := test.CreateAndInitNode("delay", types.Configuration{
		"delayMs":   "100",
		"overwrite": true,
	}, Registry)
	assert.Nil(t, err)
	delayNode := node.(*DelayNode)
	delayNode.LastPendingMsgId.Store("already-delivered-msg-id")

	done := make(chan types.RuleMsg, 1)
	test.NodeOnMsg(t, node, []test.Msg{{
		MetaData: types.BuildMetadata(nil),
		MsgType:  "TEST_MSG_TYPE",
		Data:     "NEW",
	}}, func(msg types.RuleMsg, relationType string, err error) {
		assert.Nil(t, err)
		assert.Equal(t, types.Success, relationType)
		done <- msg
	})

	select {
	case msg := <-done:
		assert.Equal(t, "NEW", msg.GetData())
	case <-time.After(2 * time.Second):
		delayNode.mu.Lock()
		leaked := len(delayNode.PendingMsgs)
		delayNode.mu.Unlock()
		t.Fatalf("message was never delivered; %d entries left in PendingMsgs", leaked)
	}

	delayNode.mu.Lock()
	defer delayNode.mu.Unlock()
	assert.Equal(t, 0, len(delayNode.PendingMsgs))
	assert.Equal(t, "", delayNode.LastPendingMsgId.Load().(string))
}

// The normal overwrite behaviour must not change: while a message is pending,
// a newer one replaces it, and only the newest is delivered.
func TestDelayNodeOverwriteReplacesPending(t *testing.T) {
	node, err := test.CreateAndInitNode("delay", types.Configuration{
		"delayMs":   "300",
		"overwrite": true,
	}, Registry)
	assert.Nil(t, err)

	delivered := make(chan string, 4)
	test.NodeOnMsg(t, node, []test.Msg{
		{MetaData: types.BuildMetadata(nil), MsgType: "TEST_MSG_TYPE", Data: "FIRST", AfterSleep: 50 * time.Millisecond},
		{MetaData: types.BuildMetadata(nil), MsgType: "TEST_MSG_TYPE", Data: "SECOND"},
	}, func(msg types.RuleMsg, relationType string, err error) {
		assert.Nil(t, err)
		assert.Equal(t, types.Success, relationType)
		delivered <- msg.GetData()
	})

	select {
	case data := <-delivered:
		assert.Equal(t, "SECOND", data)
	case <-time.After(2 * time.Second):
		t.Fatal("no message was delivered")
	}
	select {
	case data := <-delivered:
		t.Fatalf("unexpected second delivery: %s", data)
	case <-time.After(600 * time.Millisecond):
	}
}
