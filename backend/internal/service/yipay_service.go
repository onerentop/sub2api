package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/imroc/req/v3"
)

// YiPayService 易支付服务
// 负责与易支付平台的交互：生成支付链接、验证回调签名
type YiPayService struct {
	config         *config.PaymentConfig
	client         *req.Client
	settingService *SettingService
}

// NewYiPayService 创建易支付服务实例
func NewYiPayService(cfg *config.Config) *YiPayService {
	return &YiPayService{
		config: &cfg.Payment,
		client: req.C().SetTimeout(30 * time.Second),
	}
}

// SetSettingService 注入 SettingService 以支持动态配置
func (s *YiPayService) SetSettingService(settingService *SettingService) {
	s.settingService = settingService
}

// getConfig 获取支付配置（优先动态配置，回退静态配置）
func (s *YiPayService) getConfig(ctx context.Context) *config.PaymentConfig {
	if s.settingService != nil {
		cfg, err := s.settingService.GetPaymentConfig(ctx)
		if err == nil && cfg != nil {
			return cfg
		}
	}
	return s.config
}

// PaymentType 支付类型
type PaymentType string

const (
	PaymentTypeAlipay PaymentType = "alipay" // 支付宝
	PaymentTypeWechat PaymentType = "wxpay"  // 微信支付
)

// CreatePaymentRequest 创建支付请求参数
type CreatePaymentRequest struct {
	OrderNo     string      // 商户订单号
	Amount      float64     // 订单金额（元）
	ProductName string      // 商品名称
	PaymentType PaymentType // 支付类型
	ReturnURL   string      // 同步跳转地址（可选，默认使用配置）
	NotifyURL   string      // 异步通知地址（可选，默认使用配置）
}

// CreatePaymentResponse 创建支付响应
type CreatePaymentResponse struct {
	PaymentURL string // 支付跳转链接
	TradeNo    string // 易支付订单号（如果有）
}

// CallbackData 回调数据
type CallbackData struct {
	TradeNo       string  // 易支付订单号
	OutTradeNo    string  // 商户订单号
	Type          string  // 支付类型
	Name          string  // 商品名称
	Money         float64 // 金额
	TradeStatus   string  // 交易状态
	Sign          string  // 签名
	SignType      string  // 签名类型
	RawData       map[string]string
}

// IsEnabled 是否启用支付功能（使用 background context，适用于无 context 的检查场景）
func (s *YiPayService) IsEnabled() bool {
	return s.getConfig(context.Background()).Enabled
}

// IsEnabledWithContext 是否启用支付功能（使用提供的 context）
func (s *YiPayService) IsEnabledWithContext(ctx context.Context) bool {
	return s.getConfig(ctx).Enabled
}

// CreatePayment 创建支付链接
func (s *YiPayService) CreatePayment(ctx context.Context, req *CreatePaymentRequest) (*CreatePaymentResponse, error) {
	cfg := s.getConfig(ctx)
	if !cfg.Enabled {
		return nil, fmt.Errorf("payment is not enabled")
	}

	yiPayCfg := cfg.YiPay

	// 构建请求参数
	params := map[string]string{
		"pid":          yiPayCfg.PID,
		"type":         string(req.PaymentType),
		"out_trade_no": req.OrderNo,
		"notify_url":   s.getNotifyURLWithConfig(req.NotifyURL, &yiPayCfg),
		"return_url":   s.getReturnURLWithConfig(req.ReturnURL, &yiPayCfg),
		"name":         req.ProductName,
		"money":        fmt.Sprintf("%.2f", req.Amount),
	}

	// 计算签名
	sign := s.generateSignWithKey(params, yiPayCfg.Key)
	params["sign"] = sign
	params["sign_type"] = "MD5"

	// 构建支付链接
	paymentURL := s.buildPaymentURLWithConfig(params, yiPayCfg.APIURL)

	return &CreatePaymentResponse{
		PaymentURL: paymentURL,
	}, nil
}

// VerifyCallback 验证回调签名
func (s *YiPayService) VerifyCallback(data map[string]string) (*CallbackData, error) {
	cfg := s.getConfig(context.Background())
	if !cfg.Enabled {
		return nil, fmt.Errorf("payment is not enabled")
	}

	// 获取签名
	sign := data["sign"]
	if sign == "" {
		return nil, ErrPaymentSignInvalid
	}

	// 复制 data 用于签名验证（去掉 sign 和 sign_type）
	signParams := make(map[string]string)
	for k, v := range data {
		if k != "sign" && k != "sign_type" {
			signParams[k] = v
		}
	}

	// 验证签名
	expectedSign := s.generateSignWithKey(signParams, cfg.YiPay.Key)
	if !strings.EqualFold(sign, expectedSign) {
		return nil, ErrPaymentSignInvalid
	}

	// 解析金额
	var money float64
	if _, err := fmt.Sscanf(data["money"], "%f", &money); err != nil {
		return nil, fmt.Errorf("invalid money format: %w", err)
	}

	return &CallbackData{
		TradeNo:     data["trade_no"],
		OutTradeNo:  data["out_trade_no"],
		Type:        data["type"],
		Name:        data["name"],
		Money:       money,
		TradeStatus: data["trade_status"],
		Sign:        sign,
		SignType:    data["sign_type"],
		RawData:     data,
	}, nil
}

// IsTradeSuccess 检查交易是否成功
func (c *CallbackData) IsTradeSuccess() bool {
	return c.TradeStatus == "TRADE_SUCCESS"
}

// generateSign 生成签名（使用静态配置，保留用于兼容）
// 易支付签名规则：按参数名 ASCII 排序，拼接成 key=value& 格式，最后拼接 key，做 MD5
func (s *YiPayService) generateSign(params map[string]string) string {
	return s.generateSignWithKey(params, s.config.YiPay.Key)
}

// generateSignWithKey 生成签名（使用指定的 key）
func (s *YiPayService) generateSignWithKey(params map[string]string, key string) string {
	// 过滤空值并获取 key 列表
	var keys []string
	for k, v := range params {
		if v != "" {
			keys = append(keys, k)
		}
	}

	// 按字母顺序排序
	sort.Strings(keys)

	// 拼接字符串
	var builder strings.Builder
	for i, k := range keys {
		if i > 0 {
			builder.WriteString("&")
		}
		builder.WriteString(k)
		builder.WriteString("=")
		builder.WriteString(params[k])
	}

	// 拼接 key
	builder.WriteString(key)

	// 计算 MD5
	hash := md5.Sum([]byte(builder.String()))
	return hex.EncodeToString(hash[:])
}

// buildPaymentURL 构建支付链接（使用静态配置）
func (s *YiPayService) buildPaymentURL(params map[string]string) string {
	return s.buildPaymentURLWithConfig(params, s.config.YiPay.APIURL)
}

// buildPaymentURLWithConfig 构建支付链接（使用指定 API URL）
func (s *YiPayService) buildPaymentURLWithConfig(params map[string]string, apiURL string) string {
	baseURL := strings.TrimSuffix(apiURL, "/")
	if !strings.HasSuffix(baseURL, "/submit.php") {
		baseURL += "/submit.php"
	}

	// 构建 query string
	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}

	return baseURL + "?" + values.Encode()
}

// getNotifyURL 获取回调通知 URL（使用静态配置）
func (s *YiPayService) getNotifyURL(override string) string {
	if override != "" {
		return override
	}
	return s.config.YiPay.NotifyURL
}

// getNotifyURLWithConfig 获取回调通知 URL（使用指定配置）
func (s *YiPayService) getNotifyURLWithConfig(override string, yiPayCfg *config.YiPayConfig) string {
	if override != "" {
		return override
	}
	return yiPayCfg.NotifyURL
}

// getReturnURL 获取同步跳转 URL（使用静态配置）
func (s *YiPayService) getReturnURL(override string) string {
	if override != "" {
		return override
	}
	return s.config.YiPay.ReturnURL
}

// getReturnURLWithConfig 获取同步跳转 URL（使用指定配置）
func (s *YiPayService) getReturnURLWithConfig(override string, yiPayCfg *config.YiPayConfig) string {
	if override != "" {
		return override
	}
	return yiPayCfg.ReturnURL
}

// QueryOrder 查询订单状态（可选功能，部分易支付支持）
func (s *YiPayService) QueryOrder(ctx context.Context, orderNo string) (*CallbackData, error) {
	cfg := s.getConfig(ctx)
	if !cfg.Enabled {
		return nil, fmt.Errorf("payment is not enabled")
	}

	yiPayCfg := cfg.YiPay

	// 构建查询参数
	params := map[string]string{
		"act":          "order",
		"pid":          yiPayCfg.PID,
		"key":          yiPayCfg.Key,
		"out_trade_no": orderNo,
	}

	// 构建查询 URL
	baseURL := strings.TrimSuffix(yiPayCfg.APIURL, "/")
	queryURL := baseURL + "/api.php"

	// 发送请求
	var result map[string]any
	resp, err := s.client.R().
		SetContext(ctx).
		SetQueryParams(params).
		SetSuccessResult(&result).
		Get(queryURL)

	if err != nil {
		return nil, fmt.Errorf("query order failed: %w", err)
	}

	if !resp.IsSuccessState() {
		return nil, fmt.Errorf("query order failed: status %d", resp.StatusCode)
	}

	// 检查返回码
	if code, ok := result["code"].(float64); ok && code != 1 {
		msg, _ := result["msg"].(string)
		return nil, fmt.Errorf("query order failed: %s", msg)
	}

	// 解析结果
	var money float64
	if moneyStr, ok := result["money"].(string); ok {
		fmt.Sscanf(moneyStr, "%f", &money)
	}

	tradeStatus := "UNKNOWN"
	if status, ok := result["status"].(float64); ok {
		if status == 1 {
			tradeStatus = "TRADE_SUCCESS"
		}
	}

	return &CallbackData{
		TradeNo:     getString(result, "trade_no"),
		OutTradeNo:  getString(result, "out_trade_no"),
		Type:        getString(result, "type"),
		Name:        getString(result, "name"),
		Money:       money,
		TradeStatus: tradeStatus,
	}, nil
}

// getString 从 map 中安全获取字符串
func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
