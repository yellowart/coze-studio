/*
 * Copyright 2025 coze-dev Authors
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

package redis

import (
	"context"
	"errors"
	"fmt"

	goredis "github.com/redis/go-redis/v9"

	"github.com/coze-dev/coze-studio/backend/infra/contract/eventbus"
	"github.com/coze-dev/coze-studio/backend/pkg/lang/conv"
	"github.com/coze-dev/coze-studio/backend/pkg/lang/signal"
	"github.com/coze-dev/coze-studio/backend/pkg/logs"
	"github.com/coze-dev/coze-studio/backend/pkg/safego"
)

func RegisterConsumer(nameServer, topic, group string, consumerHandler eventbus.ConsumerHandler, opts ...eventbus.ConsumerOpt) error {
	if nameServer == "" {
		return fmt.Errorf("name server is empty")
	}
	if topic == "" {
		return fmt.Errorf("topic is empty")
	}

	if group == "" {
		return fmt.Errorf("group is empty")
	}

	if consumerHandler == nil {
		return fmt.Errorf("consumer handler is nil")
	}

	redisOpts, err := goredis.ParseURL(nameServer)
	if err != nil {
		return err
	}
	cli := goredis.NewClient(redisOpts)
	if err := cli.Ping(context.Background()).Err(); err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	safego.Go(context.Background(), func() {
		consumer := &consumerImpl{
			Topic:   topic,
			Group:   group,
			Handler: consumerHandler,
		}
		for {
			data, err := cli.LPop(ctx, topic).Result()
			if err != nil {
				if errors.Is(err, context.Canceled) {
					break
				}
				logs.Errorf("pop msg failed, topic: %s , err: %v\n", topic, err)
				continue
			}
			err = consumer.HandleMessage(ctx, data)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					break
				}
				logs.Errorf("handle msg failed, topic: %s , err: %v\n", topic, err)
				break
			}
		}
	})

	safego.Go(context.Background(), func() {
		signal.WaitExit()
		cancel()
		cli.Close()
	})

	return nil
}

// Customize the Handler to handle each message received
type consumerImpl struct {
	Topic   string
	Group   string
	Handler eventbus.ConsumerHandler
}

func (h *consumerImpl) HandleMessage(ctx context.Context, data string) error {
	msg := &eventbus.Message{
		Topic: h.Topic,
		Group: h.Group,
		Body:  []byte(data),
	}

	logs.Debugf("[Subscribe] receive msg : %v \n", conv.DebugJsonToStr(msg))
	err := h.Handler.HandleMessage(ctx, msg)
	if err != nil {
		logs.Errorf("[Subscribe] handle msg failed, topic : %s , group : %s, err: %v \n", msg.Topic, msg.Group, err)
		return err
	}

	logs.Debugf("subscribe callback: %v \n", conv.DebugJsonToStr(msg))

	return nil
}
