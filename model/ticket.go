package model

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	TicketStatusOpen    = "open"
	TicketStatusReplied = "replied"
	TicketStatusClosed  = "closed"
)

type Ticket struct {
	Id        int    `json:"id"`
	UserId    int    `json:"user_id" gorm:"index;not null"`
	Title     string `json:"title" gorm:"type:varchar(200);not null"`
	Content   string `json:"content" gorm:"type:text;not null"`
	Status    string `json:"status" gorm:"type:varchar(16);index;not null"`
	CreatedAt int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt int64  `json:"updated_at" gorm:"bigint"`
}

type TicketReply struct {
	Id        int    `json:"id"`
	TicketId  int    `json:"ticket_id" gorm:"index;not null"`
	UserId    int    `json:"user_id" gorm:"index;not null"`
	IsAdmin   bool   `json:"is_admin"`
	Content   string `json:"content" gorm:"type:text;not null"`
	ParentId  *int   `json:"parent_id"`
	CreatedAt int64  `json:"created_at" gorm:"bigint"`
}

type TicketDetail struct {
	Ticket
	Replies []TicketReply `json:"replies"`
}

func (Ticket) TableName() string {
	return "tickets"
}

func (TicketReply) TableName() string {
	return "ticket_replies"
}

func CreateTicket(userId int, title, content string) (*Ticket, error) {
	title = strings.TrimSpace(title)
	content = strings.TrimSpace(content)
	if title == "" || content == "" {
		return nil, errors.New("title and content are required")
	}
	if len(title) > 200 {
		return nil, errors.New("title is too long")
	}
	now := common.GetTimestamp()
	ticket := &Ticket{
		UserId:    userId,
		Title:     title,
		Content:   content,
		Status:    TicketStatusOpen,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := DB.Create(ticket).Error; err != nil {
		return nil, err
	}
	return ticket, nil
}

func GetTicketById(id int) (*Ticket, error) {
	var ticket Ticket
	err := DB.Where("id = ?", id).First(&ticket).Error
	if err != nil {
		return nil, err
	}
	return &ticket, nil
}

func GetTicketDetail(id int) (*TicketDetail, error) {
	ticket, err := GetTicketById(id)
	if err != nil {
		return nil, err
	}
	replies, err := GetTicketReplies(id)
	if err != nil {
		return nil, err
	}
	return &TicketDetail{Ticket: *ticket, Replies: replies}, nil
}

func GetTicketReplies(ticketId int) ([]TicketReply, error) {
	var replies []TicketReply
	err := DB.Where("ticket_id = ?", ticketId).Order("id ASC").Find(&replies).Error
	return replies, err
}

func GetUserTickets(userId int, status string, startIdx, pageSize int) ([]*Ticket, int64, error) {
	query := DB.Model(&Ticket{}).Where("user_id = ?", userId)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var tickets []*Ticket
	err := query.Order("updated_at DESC").Offset(startIdx).Limit(pageSize).Find(&tickets).Error
	return tickets, total, err
}

func GetAllTickets(status string, userId int, keyword string, startIdx, pageSize int) ([]*Ticket, int64, error) {
	query := DB.Model(&Ticket{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if userId > 0 {
		query = query.Where("user_id = ?", userId)
	}
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("title LIKE ? OR content LIKE ?", like, like)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var tickets []*Ticket
	err := query.Order("updated_at DESC").Offset(startIdx).Limit(pageSize).Find(&tickets).Error
	return tickets, total, err
}

func CloseTicket(id int) error {
	now := common.GetTimestamp()
	return DB.Model(&Ticket{}).Where("id = ?", id).Updates(map[string]any{
		"status":     TicketStatusClosed,
		"updated_at": now,
	}).Error
}

func UpdateTicketStatus(id int, status string) error {
	switch status {
	case TicketStatusOpen, TicketStatusReplied, TicketStatusClosed:
	default:
		return errors.New("invalid ticket status")
	}
	now := common.GetTimestamp()
	return DB.Model(&Ticket{}).Where("id = ?", id).Updates(map[string]any{
		"status":     status,
		"updated_at": now,
	}).Error
}

func DeleteTicketCascade(id int) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("ticket_id = ?", id).Delete(&TicketReply{}).Error; err != nil {
			return err
		}
		result := tx.Where("id = ?", id).Delete(&Ticket{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

func AddTicketReply(ticketId, userId int, isAdmin bool, content string, parentId *int) (*TicketReply, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, errors.New("content is required")
	}
	ticket, err := GetTicketById(ticketId)
	if err != nil {
		return nil, err
	}
	if ticket.Status == TicketStatusClosed {
		return nil, errors.New("ticket is closed")
	}
	if parentId != nil {
		var parent TicketReply
		if err := DB.Where("id = ? AND ticket_id = ?", *parentId, ticketId).First(&parent).Error; err != nil {
			return nil, errors.New("parent reply not found")
		}
	}
	reply := &TicketReply{
		TicketId:  ticketId,
		UserId:    userId,
		IsAdmin:   isAdmin,
		Content:   content,
		ParentId:  parentId,
		CreatedAt: common.GetTimestamp(),
	}
	err = DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(reply).Error; err != nil {
			return err
		}
		newStatus := TicketStatusOpen
		if isAdmin {
			newStatus = TicketStatusReplied
		}
		return tx.Model(&Ticket{}).Where("id = ?", ticketId).Updates(map[string]any{
			"status":     newStatus,
			"updated_at": common.GetTimestamp(),
		}).Error
	})
	if err != nil {
		return nil, err
	}
	return reply, nil
}
