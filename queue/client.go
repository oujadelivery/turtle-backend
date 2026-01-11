package queue

import "github.com/hibiken/asynq"

var Client = asynq.NewClient(asynq.RedisClientOpt{Addr: "localhost:6379"})
