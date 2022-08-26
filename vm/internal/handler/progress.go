package handler

import (
	"github.com/labstack/echo"
	"net/http"
	"sync"
)

type ProgressCache struct {
	sync.RWMutex
	m map[string]string // map of volumes and actions, e.g. m["vol-1"] = "export"
}

// ActionsInProgress retrieves the current action (i.e. export, import, save or load) that is running for every volume.
func (h *Handler) ActionsInProgress(ctx echo.Context) error {
	return ctx.JSON(http.StatusOK, h.ProgressCache.m)
}
