package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/chaitin/MonkeyCode/taskflow/internal/runner"
	"github.com/chaitin/MonkeyCode/taskflow/internal/store"
)

type PortForwardHandler struct {
	store   *store.RedisStore
	manager *runner.Manager
}

func NewPortForwardHandler(s *store.RedisStore, m *runner.Manager) *PortForwardHandler {
	return &PortForwardHandler{store: s, manager: m}
}

type CreatePortForwardRequest struct {
	VMID          string `json:"vm_id"`
	ContainerID   string `json:"container_id"`
	HostPort      int    `json:"host_port"`
	ContainerPort int    `json:"container_port"`
	Protocol      string `json:"protocol"`
}

type UpdatePortForwardRequest struct {
	ID       string `json:"id"`
	VMID     string `json:"vm_id"`
	HostPort int    `json:"host_port"`
}

type ClosePortForwardRequest struct {
	ID   string `json:"id"`
	VMID string `json:"vm_id"`
}

func (h *PortForwardHandler) List(c echo.Context) error {
	vmID := c.QueryParam("id")

	if vmID != "" {
		forwardIDs, err := h.store.GetVMForwardIDs(c.Request().Context(), vmID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		infos := make([]*store.PortForward, 0)
		for _, fid := range forwardIDs {
			fwd, err := h.store.GetPortForward(c.Request().Context(), fid)
			if err != nil {
				continue
			}
			infos = append(infos, fwd)
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"code": 0,
			"data": infos,
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"code": 0,
		"data": []*store.PortForward{},
	})
}

func (h *PortForwardHandler) Create(c echo.Context) error {
	var req CreatePortForwardRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if req.VMID == "" || req.ContainerID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "vm_id and container_id required")
	}

	conn, ok := h.manager.GetRunner(req.VMID)
	if !ok {
		return echo.NewHTTPError(http.StatusNotFound, "runner not found")
	}

	cmd := map[string]interface{}{
		"type": "portforward_create",
		"data": map[string]interface{}{
			"forward_id":     req.VMID,
			"container_id":   req.ContainerID,
			"host_port":      req.HostPort,
			"container_port": req.ContainerPort,
			"protocol":       req.Protocol,
		},
	}

	cmdBytes, _ := json.Marshal(cmd)
	select {
	case conn.Stream <- cmdBytes:
	default:
		return echo.NewHTTPError(http.StatusServiceUnavailable, "runner stream full")
	}

	info := &store.PortForward{
		ID:            fmt.Sprintf("%s-%d", req.VMID, req.HostPort),
		VMID:          req.VMID,
		HostPort:      req.HostPort,
		ContainerPort: req.ContainerPort,
		Protocol:      req.Protocol,
		Status:        "created",
	}

	if err := h.store.SetPortForward(c.Request().Context(), info); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if err := h.store.AddVMForwardID(c.Request().Context(), req.VMID, info.ID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"code": 0,
		"data": info,
	})
}

func (h *PortForwardHandler) Update(c echo.Context) error {
	var req UpdatePortForwardRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if req.ID == "" || req.VMID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "id and vm_id required")
	}

	info, err := h.store.GetPortForward(c.Request().Context(), req.ID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "port forward not found")
	}

	info.HostPort = req.HostPort

	if err := h.store.SetPortForward(c.Request().Context(), info); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"code": 0,
		"data": info,
	})
}

func (h *PortForwardHandler) Close(c echo.Context) error {
	var req ClosePortForwardRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if req.ID == "" || req.VMID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "id and vm_id required")
	}

	conn, ok := h.manager.GetRunner(req.VMID)
	if !ok {
		return echo.NewHTTPError(http.StatusNotFound, "runner not found")
	}

	cmd := map[string]interface{}{
		"type": "portforward_close",
		"data": map[string]interface{}{
			"forward_id": req.ID,
		},
	}

	cmdBytes, _ := json.Marshal(cmd)
	select {
	case conn.Stream <- cmdBytes:
	default:
		return echo.NewHTTPError(http.StatusServiceUnavailable, "runner stream full")
	}

	if err := h.store.DeletePortForward(c.Request().Context(), req.ID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if err := h.store.RemoveVMForwardID(c.Request().Context(), req.VMID, req.ID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"code":    0,
		"message": "closed",
	})
}
