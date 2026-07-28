package controller

import (
	"errors"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type createTicketRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

type ticketReplyRequest struct {
	Content  string `json:"content"`
	ParentId *int   `json:"parent_id"`
}

type updateTicketStatusRequest struct {
	Status string `json:"status"`
}

func CreateTicket(c *gin.Context) {
	var req createTicketRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	ticket, err := model.CreateTicket(c.GetInt("id"), req.Title, req.Content)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, ticket)
}

func GetUserTickets(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	status := strings.TrimSpace(c.Query("status"))
	tickets, total, err := model.GetUserTickets(c.GetInt("id"), status, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(tickets)
	common.ApiSuccess(c, pageInfo)
}

func GetUserTicket(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	detail, err := model.GetTicketDetail(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			common.ApiErrorMsg(c, "ticket not found")
			return
		}
		common.ApiError(c, err)
		return
	}
	if detail.UserId != c.GetInt("id") {
		common.ApiErrorMsg(c, "ticket not found")
		return
	}
	common.ApiSuccess(c, detail)
}

func CreateUserTicketReply(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	ticket, err := model.GetTicketById(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			common.ApiErrorMsg(c, "ticket not found")
			return
		}
		common.ApiError(c, err)
		return
	}
	if ticket.UserId != c.GetInt("id") {
		common.ApiErrorMsg(c, "ticket not found")
		return
	}
	var req ticketReplyRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	reply, err := model.AddTicketReply(id, c.GetInt("id"), false, req.Content, req.ParentId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, reply)
}

func CloseUserTicket(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	ticket, err := model.GetTicketById(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			common.ApiErrorMsg(c, "ticket not found")
			return
		}
		common.ApiError(c, err)
		return
	}
	if ticket.UserId != c.GetInt("id") {
		common.ApiErrorMsg(c, "ticket not found")
		return
	}
	if err := model.CloseTicket(id); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func DeleteUserTicket(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	ticket, err := model.GetTicketById(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			common.ApiErrorMsg(c, "ticket not found")
			return
		}
		common.ApiError(c, err)
		return
	}
	if ticket.UserId != c.GetInt("id") {
		common.ApiErrorMsg(c, "ticket not found")
		return
	}
	if err := model.DeleteTicketCascade(id); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func GetAllTickets(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	status := strings.TrimSpace(c.Query("status"))
	keyword := strings.TrimSpace(c.Query("keyword"))
	userId, _ := strconv.Atoi(c.Query("user_id"))
	tickets, total, err := model.GetAllTickets(status, userId, keyword, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(tickets)
	common.ApiSuccess(c, pageInfo)
}

func GetAdminTicket(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	detail, err := model.GetTicketDetail(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			common.ApiErrorMsg(c, "ticket not found")
			return
		}
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, detail)
}

func CreateAdminTicketReply(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	if _, err := model.GetTicketById(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			common.ApiErrorMsg(c, "ticket not found")
			return
		}
		common.ApiError(c, err)
		return
	}
	var req ticketReplyRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	reply, err := model.AddTicketReply(id, c.GetInt("id"), true, req.Content, req.ParentId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, reply)
}

func CloseAdminTicket(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	if _, err := model.GetTicketById(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			common.ApiErrorMsg(c, "ticket not found")
			return
		}
		common.ApiError(c, err)
		return
	}
	if err := model.CloseTicket(id); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func UpdateAdminTicket(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	if _, err := model.GetTicketById(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			common.ApiErrorMsg(c, "ticket not found")
			return
		}
		common.ApiError(c, err)
		return
	}
	var req updateTicketStatusRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	if err := model.UpdateTicketStatus(id, strings.TrimSpace(req.Status)); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func DeleteAdminTicket(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	if err := model.DeleteTicketCascade(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			common.ApiErrorMsg(c, "ticket not found")
			return
		}
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}
