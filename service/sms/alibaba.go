package sms

import (
	"context"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/dysmsapi"
)

type alibabaProvider struct {
	client   *dysmsapi.Client
	signName string
}

func newAlibabaProvider() (Provider, error) {
	accessKeyId := strings.TrimSpace(common.SMSClientId)
	accessKeySecret := strings.TrimSpace(common.SMSClientSecret)
	signName := strings.TrimSpace(common.SMSSignName)
	if accessKeyId == "" || accessKeySecret == "" {
		return nil, fmt.Errorf("Alibaba Cloud SMS client credentials are not configured")
	}
	if signName == "" {
		return nil, fmt.Errorf("Alibaba Cloud SMS sign name is not configured")
	}
	client, err := dysmsapi.NewClientWithAccessKey("cn-hangzhou", accessKeyId, accessKeySecret)
	if err != nil {
		return nil, fmt.Errorf("create Alibaba Cloud SMS client: %w", err)
	}
	return &alibabaProvider{client: client, signName: signName}, nil
}

func (p *alibabaProvider) Send(ctx context.Context, phone, templateCode string, params map[string]string) error {
	_ = ctx
	phone = strings.TrimSpace(phone)
	templateCode = strings.TrimSpace(templateCode)
	if phone == "" {
		return fmt.Errorf("phone number is required")
	}
	if templateCode == "" {
		return fmt.Errorf("SMS template code is required")
	}

	req := dysmsapi.CreateSendSmsRequest()
	req.Scheme = "https"
	req.PhoneNumbers = phone
	req.SignName = p.signName
	req.TemplateCode = templateCode
	if len(params) > 0 {
		paramBytes, err := common.Marshal(params)
		if err != nil {
			return fmt.Errorf("marshal SMS template params: %w", err)
		}
		req.TemplateParam = string(paramBytes)
	}

	resp, err := p.client.SendSms(req)
	if err != nil {
		return fmt.Errorf("send SMS via Alibaba Cloud: %w", err)
	}
	if resp == nil {
		return fmt.Errorf("empty SMS response from Alibaba Cloud")
	}
	if !strings.EqualFold(resp.Code, "OK") {
		return fmt.Errorf("Alibaba Cloud SMS error: %s (%s)", resp.Message, resp.Code)
	}
	return nil
}
