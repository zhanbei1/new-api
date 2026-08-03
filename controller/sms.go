package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/sms"
	"github.com/gin-gonic/gin"
)

type SMSTestRequest struct {
	Phone string `json:"phone"`
}

// TestSMS sends a test verification SMS using the currently configured SMS provider.
func TestSMS(c *gin.Context) {
	var req SMSTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	phone := model.NormalizePhone(req.Phone)
	if !model.IsValidPhone(phone) {
		common.ApiErrorMsg(c, "手机号格式不正确")
		return
	}
	if !sms.IsConfigured() {
		common.ApiErrorMsg(c, "SMS 服务未配置完整")
		return
	}
	code := common.GenerateNumericVerificationCode(6)
	if err := sms.SendVerificationCode(c.Request.Context(), phone, code); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"phone": phone,
	})
}

type SMSSendRequest struct {
	Phone     string `json:"phone"`
	Turnstile string `json:"turnstile"`
}

func sendSMSCode(c *gin.Context, purpose string, requireExistingUser bool) {
	if !common.PhoneVerificationEnabled {
		common.ApiErrorMsg(c, "手机号验证未启用")
		return
	}
	var req SMSSendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	phone := model.NormalizePhone(req.Phone)
	if !model.IsValidPhone(phone) {
		common.ApiErrorMsg(c, "手机号格式不正确")
		return
	}
	if !sms.IsConfigured() {
		common.ApiErrorMsg(c, "SMS 服务未配置")
		return
	}

	if requireExistingUser {
		if !model.IsPhoneAlreadyTaken(phone) {
			common.ApiSuccess(c, nil)
			return
		}
	} else {
		if model.IsPhoneAlreadyTaken(phone) {
			common.ApiErrorMsg(c, "该手机号已被占用")
			return
		}
	}

	code := common.GenerateNumericVerificationCode(6)
	common.RegisterVerificationCodeWithKey(phone, code, purpose)
	if err := sms.SendVerificationCode(c.Request.Context(), phone, code); err != nil {
		common.DeleteKey(phone, purpose)
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}

// SendSMSVerification sends a registration/bind verification code.
func SendSMSVerification(c *gin.Context) {
	sendSMSCode(c, common.PhoneVerificationPurpose, false)
}

// SendSMSLoginCode sends a login verification code.
func SendSMSLoginCode(c *gin.Context) {
	sendSMSCode(c, common.PhoneLoginPurpose, true)
}

type phoneBindRequest struct {
	Phone string `json:"phone"`
	Code  string `json:"code"`
}

func PhoneBind(c *gin.Context) {
	if !common.PhoneVerificationEnabled {
		common.ApiErrorMsg(c, "手机号验证未启用")
		return
	}
	var req phoneBindRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, errors.New("invalid request body"))
		return
	}
	phone := model.NormalizePhone(req.Phone)
	code := strings.TrimSpace(req.Code)
	if !model.IsValidPhone(phone) || code == "" {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if !common.VerifyCodeWithKey(phone, code, common.PhoneVerificationPurpose) {
		common.ApiErrorI18n(c, i18n.MsgUserVerificationCodeError)
		return
	}
	user := model.User{
		Id: c.GetInt("id"),
	}
	if user.Id == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "not authenticated"})
		return
	}
	if err := user.FillUserById(); err != nil {
		common.ApiError(c, err)
		return
	}
	if model.IsPhoneAlreadyTaken(phone) && user.Phone != phone {
		common.ApiErrorMsg(c, "该手机号已被占用")
		return
	}
	user.Phone = phone
	if err := user.Update(false); err != nil {
		common.ApiError(c, err)
		return
	}
	common.DeleteKey(phone, common.PhoneVerificationPurpose)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}

type SMSLoginRequest struct {
	Phone            string `json:"phone"`
	VerificationCode string `json:"verification_code"`
}

func LoginSMS(c *gin.Context) {
	if !common.PhoneVerificationEnabled {
		common.ApiErrorMsg(c, "手机号验证未启用")
		return
	}
	var req SMSLoginRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	phone := model.NormalizePhone(req.Phone)
	code := strings.TrimSpace(req.VerificationCode)
	if !model.IsValidPhone(phone) || code == "" {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	if !common.VerifyCodeWithKey(phone, code, common.PhoneLoginPurpose) {
		common.ApiErrorI18n(c, i18n.MsgUserVerificationCodeError)
		return
	}
	user, err := model.GetUserByPhone(phone)
	if err != nil || user.Status != common.UserStatusEnabled {
		common.ApiErrorI18n(c, i18n.MsgUserUsernameOrPasswordError)
		return
	}

	twoFAEnabled, err := model.IsTwoFAEnabled(user.Id)
	if err != nil {
		common.SysLog(fmt.Sprintf("SMS login failed to load 2FA status for user %d: %v", user.Id, err))
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	if twoFAEnabled {
		expiresAt := time.Now().Add(5 * time.Minute)
		payload, err := common.Marshal(twoFALoginFlowPayload{AuthVersion: user.AuthVersion})
		if err != nil {
			common.ApiError(c, err)
			return
		}
		flowToken, _, err := model.CreateAuthFlow(model.AuthFlowCreate{
			Purpose:   model.AuthFlowPurposeTwoFALogin,
			UserId:    user.Id,
			Payload:   string(payload),
			ExpiresAt: expiresAt,
		})
		if err != nil {
			common.ApiError(c, err)
			return
		}
		common.DeleteKey(phone, common.PhoneLoginPurpose)
		c.JSON(http.StatusOK, gin.H{
			"message": i18n.T(c, i18n.MsgUserRequire2FA),
			"success": true,
			"data": map[string]interface{}{
				"require_2fa": true,
				"flow_token":  flowToken,
				"expires_at":  expiresAt.Unix(),
			},
		})
		return
	}

	common.DeleteKey(phone, common.PhoneLoginPurpose)
	setupLogin(user, c)
}
