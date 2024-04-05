package server

import (
	"context"
	"github.com/gin-gonic/gin"
	"log/slog"
	"net/http"
)

type ctrlWorker struct {
	manager *Manager
	logger  *slog.Logger
}

func NewCtrlWorker(_ context.Context, logger *slog.Logger) (Worker, error) {
	return &ctrlWorker{manager: NewManager(logger), logger: logger}, nil
}

func (w *ctrlWorker) Mount(r gin.IRouter) {
	r.GET("/wsx", w.serveWS)
}

func (w *ctrlWorker) Tag() string {
	return "ctrl"
}

func (w *ctrlWorker) Run(ctx context.Context) error {
	w.manager.Run(ctx)
	return nil
}
func (w *ctrlWorker) serveWS(c *gin.Context) {
	sess, err := w.manager.Accept(c.Writer, c.Request)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
	}
	w.logger.Info("accept session ", "id", sess.conn.ID)
}
