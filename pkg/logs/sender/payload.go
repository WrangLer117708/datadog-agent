// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package sender

import (
	"time"

	"github.com/DataDog/datadog-agent/pkg/logs/message"
)

type Payload interface {
	AddMessage(message *message.Message) bool
	Clear()
	GetContent() []byte
	GetMessages() []*message.Message
	IsFull() bool
	IsEmpty() bool
	IsReady() <-chan time.Time
}
