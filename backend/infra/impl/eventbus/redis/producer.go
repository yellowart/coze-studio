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
	"fmt"

	goredis "github.com/redis/go-redis/v9"

	"github.com/coze-dev/coze-studio/backend/infra/contract/eventbus"
	"github.com/coze-dev/coze-studio/backend/pkg/lang/signal"
	"github.com/coze-dev/coze-studio/backend/pkg/safego"
)

type producerImpl struct {
	nameServer string
	topic      string
	cli        *goredis.Client
}

func NewProducer(nameServer, topic, group string) (eventbus.Producer, error) {
	if nameServer == "" {
		return nil, fmt.Errorf("name server is empty")
	}

	if topic == "" {
		return nil, fmt.Errorf("topic is empty")
	}

	opts, err := goredis.ParseURL(nameServer)
	if err != nil {
		return nil, err
	}
	cli := goredis.NewClient(opts)
	safego.Go(context.Background(), func() {
		signal.WaitExit()
		cli.Close()
	})

	return &producerImpl{
		nameServer: nameServer,
		topic:      topic,
		cli:        cli,
	}, nil
}

func (r *producerImpl) Send(ctx context.Context, body []byte, opts ...eventbus.SendOpt) error {
	err := r.cli.RPush(ctx, r.topic, body).Err()
	if err != nil {
		return fmt.Errorf("[producerImpl] send message failed: %w", err)
	}
	return err
}

func (r *producerImpl) BatchSend(ctx context.Context, bodyArr [][]byte, opts ...eventbus.SendOpt) error {
	option := eventbus.SendOption{}
	for _, opt := range opts {
		opt(&option)
	}

	err := r.cli.RPush(ctx, r.topic, bodyArr).Err()
	if err != nil {
		return fmt.Errorf("[producerImpl] batch send messages failed: %w", err)
	}
	return nil
}
