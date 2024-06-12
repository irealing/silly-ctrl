package silly_ctrl

import (
	"context"
)

type Worker interface {
	Tag() string
	Run(ctx context.Context) error
}

type WorkerCreator func(ctx context.Context) (Worker, error)
