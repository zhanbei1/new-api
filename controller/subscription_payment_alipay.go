package controller

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/smartwalle/alipay/v3"
	"github.com/thanhpk/randstr"
)

type SubscriptionAlipayPayRequest struct {
	PlanId int `json:"plan_id"`
}

func SubscriptionRequestAlipayPay(c *gin.Context) {
	if !requirePaymentCompliance(c) {
		return
	}
	if !isAlipayTopUpEnabled() {
		common.ApiErrorMsg(c, "支付宝支付未启用")
		return
	}

	var req SubscriptionAlipayPayRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.PlanId <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}

	plan, err := model.GetSubscriptionPlanById(req.PlanId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if !plan.Enabled {
		common.ApiErrorMsg(c, "套餐未启用")
		return
	}
	if plan.AllowAlipay == nil || !*plan.AllowAlipay {
		common.ApiErrorMsg(c, "该套餐未开启支付宝支付")
		return
	}
	if plan.PriceAmount < 0.01 {
		common.ApiErrorMsg(c, "套餐金额过低")
		return
	}

	userId := c.GetInt("id")
	if plan.MaxPurchasePerUser > 0 {
		count, err := model.CountUserSubscriptionsByPlan(userId, plan.Id)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		if count >= int64(plan.MaxPurchasePerUser) {
			common.ApiErrorMsg(c, "已达到该套餐购买上限")
			return
		}
	}

	tradeNo := fmt.Sprintf("SUBALI%dNO%s%d", userId, randstr.String(6), time.Now().Unix())
	client, err := service.GetAlipayClient()
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Alipay subscription client init failed: %v", err))
		common.ApiErrorMsg(c, "支付宝未正确配置")
		return
	}

	notifyURL := service.GetCallbackAddress() + "/api/alipay/notify"
	var p = alipay.TradePreCreate{}
	p.NotifyURL = notifyURL
	p.Subject = fmt.Sprintf("SUB:%s", plan.Title)
	p.OutTradeNo = tradeNo
	p.TotalAmount = strconv.FormatFloat(plan.PriceAmount, 'f', 2, 64)
	p.ProductCode = "FACE_TO_FACE_PAYMENT"
	p.TimeoutExpress = "15m"

	rsp, err := client.TradePreCreate(c.Request.Context(), p)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf(
			"Alipay subscription TradePreCreate failed user_id=%d plan_id=%d trade_no=%s notify_url=%s error=%q",
			userId, plan.Id, tradeNo, notifyURL, err.Error(),
		))
		common.ApiErrorMsg(c, "拉起支付失败")
		return
	}
	// smartwalle/alipay returns err=nil on HTTP OK even when Alipay business fails;
	// empty qr_code + SubCode/SubMsg is the usual failure shape (previously silent — no log).
	if rsp == nil || rsp.IsFailure() || strings.TrimSpace(rsp.QRCode) == "" {
		code, msg, subCode, subMsg := "", "", "", ""
		if rsp != nil {
			code, msg, subCode, subMsg = string(rsp.Code), rsp.Msg, rsp.SubCode, rsp.SubMsg
		}
		logger.LogError(c.Request.Context(), fmt.Sprintf(
			"Alipay subscription TradePreCreate rejected user_id=%d plan_id=%d trade_no=%s notify_url=%s code=%s msg=%q sub_code=%s sub_msg=%q qr_empty=%v",
			userId, plan.Id, tradeNo, notifyURL, code, msg, subCode, subMsg,
			rsp == nil || strings.TrimSpace(rsp.QRCode) == "",
		))
		common.ApiErrorMsg(c, "拉起支付失败")
		return
	}

	order := &model.SubscriptionOrder{
		UserId:          userId,
		PlanId:          plan.Id,
		Money:           plan.PriceAmount,
		TradeNo:         tradeNo,
		PaymentMethod:   model.PaymentMethodAlipay,
		PaymentProvider: model.PaymentProviderAlipay,
		Status:          common.TopUpStatusPending,
		CreateTime:      common.GetTimestamp(),
	}
	if err := order.Insert(); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Alipay subscription create order failed: %v", err))
		common.ApiErrorMsg(c, "创建订单失败")
		return
	}

	common.ApiSuccess(c, AlipayPayResult{
		TradeNo:  tradeNo,
		QRCode:   rsp.QRCode,
		ExpireAt: time.Now().Add(15 * time.Minute).Unix(),
	})
}

func GetSubscriptionOrderStatus(c *gin.Context) {
	tradeNo := strings.TrimSpace(c.Query("trade_no"))
	if tradeNo == "" {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	userId := c.GetInt("id")
	order := model.GetSubscriptionOrderByTradeNo(tradeNo)
	if order == nil || order.UserId != userId {
		common.ApiErrorMsg(c, "订单不存在")
		return
	}
	common.ApiSuccess(c, gin.H{
		"trade_no": order.TradeNo,
		"status":   order.Status,
		"plan_id":  order.PlanId,
		"money":    order.Money,
	})
}
