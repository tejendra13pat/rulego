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

// A delay of zero (or less) means "do not delay": the message is passed on
// immediately. That already works when the value comes from a ${...} template;
// a literal "0" used to be rejected with "no delay time configured" because the
// pre-parsed value was only used when it was greater than zero.
func TestDelayNodeNoDelay(t *testing.T) {
	tests := []struct {
		name     string
		delayMs  string
		metadata map[string]string
	}{
		{name: "literal zero", delayMs: "0"},
		{name: "literal negative", delayMs: "-1"},
		{name: "template resolving to zero", delayMs: "${metadata.delay}", metadata: map[string]string{"delay": "0"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := test.CreateAndInitNode("delay", types.Configuration{"delayMs": tt.delayMs}, Registry)
			assert.Nil(t, err)

			done := make(chan struct{})
			test.NodeOnMsg(t, node, []test.Msg{{
				MetaData: types.BuildMetadata(tt.metadata),
				MsgType:  "TEST_MSG_TYPE",
				Data:     "AA",
			}}, func(msg types.RuleMsg, relationType string, err error) {
				assert.Nil(t, err)
				assert.Equal(t, types.Success, relationType)
				assert.Equal(t, "AA", msg.GetData())
				close(done)
			})
			select {
			case <-done:
			case <-time.After(2 * time.Second):
				t.Fatal("message was not passed on")
			}
		})
	}
}
