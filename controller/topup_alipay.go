package controller

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/smartwalle/alipay/v3"
	"github.com/thanhpk/randstr"
)

type AlipayPayRequest struct {
	Amount int64 `json:"amount"`
}

type AlipayPayResult struct {
	TradeNo  string `json:"trade_no"`
	QRCode   string `json:"qr_code"`
	ExpireAt int64  `json:"expire_at,omitempty"`
}

func RequestAlipayPay(c *gin.Context) {
	if !requirePaymentCompliance(c) {
		return
	}
	if !isAlipayTopUpEnabled() {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "支付宝支付未启用"})
		return
	}

	var req AlipayPayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "参数错误"})
		return
	}
	if req.Amount < getAlipayMinTopup() {
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("充值数量不能小于 %d", getAlipayMinTopup()), "data": 10})
		return
	}
	if req.Amount > 10000 {
		c.JSON(http.StatusOK, gin.H{"message": "充值数量不能大于 10000", "data": 10})
		return
	}

	id := c.GetInt("id")
	user, err := model.GetUserById(id, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	chargedMoney := getAlipayPayMoney(float64(req.Amount), user.Group)
	if chargedMoney < 0.01 {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "充值金额过低"})
		return
	}

	tradeNo := fmt.Sprintf("ALIUSR%dNO%s%d", id, randstr.String(6), time.Now().Unix())
	client, err := service.GetAlipayClient()
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Alipay client init failed: %v", err))
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "支付宝未正确配置"})
		return
	}

	notifyURL := service.GetCallbackAddress() + "/api/alipay/notify"
	var p = alipay.TradePreCreate{}
	p.NotifyURL = notifyURL
	p.Subject = fmt.Sprintf("%s 充值", common.SystemName)
	p.OutTradeNo = tradeNo
	p.TotalAmount = formatAlipayMoney(chargedMoney)
	p.ProductCode = "FACE_TO_FACE_PAYMENT"
	p.TimeoutExpress = "15m"

	rsp, err := client.TradePreCreate(c.Request.Context(), p)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf(
			"Alipay TradePreCreate failed user_id=%d trade_no=%s notify_url=%s error=%q",
			id, tradeNo, notifyURL, err.Error(),
		))
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "拉起支付失败"})
		return
	}
	// smartwalle/alipay returns err=nil on HTTP OK even when Alipay business fails.
	if rsp == nil || rsp.IsFailure() || strings.TrimSpace(rsp.QRCode) == "" {
		code, msg, subCode, subMsg := "", "", "", ""
		if rsp != nil {
			code, msg, subCode, subMsg = string(rsp.Code), rsp.Msg, rsp.SubCode, rsp.SubMsg
		}
		logger.LogError(c.Request.Context(), fmt.Sprintf(
			"Alipay TradePreCreate rejected user_id=%d trade_no=%s notify_url=%s code=%s msg=%q sub_code=%s sub_msg=%q qr_empty=%v",
			id, tradeNo, notifyURL, code, msg, subCode, subMsg,
			rsp == nil || strings.TrimSpace(rsp.QRCode) == "",
		))
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "拉起支付失败"})
		return
	}

	topUp := &model.TopUp{
		UserId:          id,
		Amount:          req.Amount,
		Money:           chargedMoney,
		TradeNo:         tradeNo,
		PaymentMethod:   model.PaymentMethodAlipay,
		PaymentProvider: model.PaymentProviderAlipay,
		CreateTime:      time.Now().Unix(),
		Status:          common.TopUpStatusPending,
	}
	if err := topUp.Insert(); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Alipay create order failed user_id=%d trade_no=%s error=%q", id, tradeNo, err.Error()))
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "创建订单失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "success",
		"data": AlipayPayResult{
			TradeNo:  tradeNo,
			QRCode:   rsp.QRCode,
			ExpireAt: time.Now().Add(15 * time.Minute).Unix(),
		},
	})
}

func RequestAlipayAmount(c *gin.Context) {
	var req AlipayPayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "参数错误"})
		return
	}
	if req.Amount < getAlipayMinTopup() {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": fmt.Sprintf("充值数量不能小于 %d", getAlipayMinTopup())})
		return
	}
	id := c.GetInt("id")
	group, err := model.GetUserGroup(id, true)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "获取用户分组失败"})
		return
	}
	payMoney := getAlipayPayMoney(float64(req.Amount), group)
	if payMoney <= 0.01 {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "充值金额过低"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "success", "data": strconv.FormatFloat(payMoney, 'f', 2, 64)})
}

func GetTopUpStatus(c *gin.Context) {
	tradeNo := strings.TrimSpace(c.Query("trade_no"))
	if tradeNo == "" {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	userId := c.GetInt("id")
	topUp := model.GetTopUpByTradeNo(tradeNo)
	if topUp == nil || topUp.UserId != userId {
		common.ApiErrorMsg(c, "订单不存在")
		return
	}
	common.ApiSuccess(c, gin.H{
		"trade_no": topUp.TradeNo,
		"status":   topUp.Status,
		"amount":   topUp.Amount,
		"money":    topUp.Money,
	})
}

func AlipayNotify(c *gin.Context) {
	if err := c.Request.ParseForm(); err != nil {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("Alipay notify parse form failed: %v", err))
		c.String(http.StatusBadRequest, "fail")
		return
	}
	client, err := service.GetAlipayClient()
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Alipay notify client error: %v", err))
		c.String(http.StatusOK, "fail")
		return
	}
	noti, err := client.DecodeNotification(c.Request.Context(), c.Request.Form)
	if err != nil {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("Alipay notify verify failed: %v", err))
		c.String(http.StatusOK, "fail")
		return
	}
	if noti.TradeStatus != alipay.TradeStatusSuccess && noti.TradeStatus != alipay.TradeStatusFinished {
		alipay.ACKNotification(c.Writer)
		return
	}

	tradeNo := noti.OutTradeNo
	LockOrder(tradeNo)
	defer UnlockOrder(tradeNo)

	// Try subscription order first, then top-up.
	subOrder := model.GetSubscriptionOrderByTradeNo(tradeNo)
	if subOrder != nil {
		payload, _ := common.Marshal(noti)
		if err := model.CompleteSubscriptionOrder(tradeNo, string(payload), model.PaymentProviderAlipay, model.PaymentMethodAlipay); err != nil {
			logger.LogError(c.Request.Context(), fmt.Sprintf("Alipay subscription complete failed trade_no=%s error=%q", tradeNo, err.Error()))
			c.String(http.StatusOK, "fail")
			return
		}
		alipay.ACKNotification(c.Writer)
		return
	}

	if err := model.RechargeAlipay(tradeNo, c.ClientIP()); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Alipay recharge failed trade_no=%s error=%q", tradeNo, err.Error()))
		topUp := model.GetTopUpByTradeNo(tradeNo)
		if topUp != nil && topUp.Status == common.TopUpStatusSuccess {
			alipay.ACKNotification(c.Writer)
			return
		}
		c.String(http.StatusOK, "fail")
		return
	}
	alipay.ACKNotification(c.Writer)
}

func getAlipayPayMoney(amount float64, group string) float64 {
	originalAmount := amount
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		amount = amount / common.QuotaPerUnit
	}
	topupGroupRatio := common.GetTopupGroupRatio(group)
	if topupGroupRatio == 0 {
		topupGroupRatio = 1
	}
	discount := 1.0
	if ds, ok := operation_setting.GetPaymentSetting().AmountDiscount[int(originalAmount)]; ok {
		if ds > 0 {
			discount = ds
		}
	}
	return amount * setting.AlipayUnitPrice * topupGroupRatio * discount
}

func getAlipayMinTopup() int64 {
	minTopup := setting.AlipayMinTopUp
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		minTopup = minTopup * int(common.QuotaPerUnit)
	}
	return int64(minTopup)
}

func formatAlipayMoney(money float64) string {
	d := decimal.NewFromFloat(money)
	return d.StringFixed(2)
}
