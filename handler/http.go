package handler

import (
	"goplay/matchmaker"

	"net/http"

	"github.com/gin-gonic/gin"
)

type HttpHandler struct {
	matchmaker matchmaker.Matchmaker
}

func NewHttpHandler(matchmaker matchmaker.Matchmaker) *HttpHandler {
	return &HttpHandler{
		matchmaker: matchmaker,
	}
}

type AddGroupReq struct {
	ID        string
	PlayerIDs []int
}

func (h *HttpHandler) AddGroup(c *gin.Context) {
	var req AddGroupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	found := make(chan string)
	cancelled := make(chan bool)
	err := h.matchmaker.AddGroup(c.Request.Context(), req.ID, req.PlayerIDs, found, cancelled)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	for {
		select {
		case serverId := <-found:
			c.JSON(http.StatusOK, serverId)
			return
		case <-cancelled:
			c.Status(http.StatusOK)
			return
		}
	}
}

type RemoveGroupReq struct {
	ID string
}

func (h *HttpHandler) RemoveGroup(c *gin.Context) {
	var req RemoveGroupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.matchmaker.RemoveGroup(req.ID)

	c.Status(http.StatusOK)
}

type SetPlayerReadyReq struct {
	PlayerId int
}

func (h *HttpHandler) SetPlayerReady(c *gin.Context) {
	var req SetPlayerReadyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.matchmaker.SetPlayerReady(req.PlayerId)

	c.Status(http.StatusOK)
}
