package stripe

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/stripe/stripe-go/v81"
	stripeclient "github.com/stripe/stripe-go/v81/client"
	"github.com/stripe/stripe-go/v81/webhook"
)

const APIVersion = "2024-04-10"

type Config struct {
	PublicKey     string
	SecretKey     string
	WebhookSecret string
}

type User struct {
	UserId int64
	Email  string
}
type NotifyResult struct {
	EventType string
	OrderNo   string
	TradeNo   string
	Method    string
	UserId    int64
	Amount    int64
}
type Order struct {
	OrderNo   string
	Subscribe string
	Amount    int64
	Currency  string
	Payment   string
}

type Client struct {
	Config
	api *stripeclient.API
}

type PaymentSheet struct {
	ClientSecret   string
	EphemeralKey   string
	Customer       string
	PublishableKey string
	TradeNo        string
}

func NewClient(config Config) *Client {
	return &Client{
		Config: config,
		api:    stripeclient.New(config.SecretKey, nil),
	}
}

func (c *Client) CreatePaymentSheet(order *Order, user *User) (*PaymentSheet, error) {
	if order == nil || user == nil || order.OrderNo == "" || order.Amount <= 0 {
		return nil, fmt.Errorf("Stripe payment order and user are required")
	}
	// Create a new Stripe customer if it does not exist
	customerDataRes, err := c.SearchStripeCustomer(user)
	if err != nil {
		return nil, err
	}
	if customerDataRes == nil {
		customerDataRes, err = c.CreateCustomer(user)
		if err != nil {
			return nil, err
		}
	}
	// Create Ephemeral Key
	ekParams := &stripe.EphemeralKeyParams{
		Customer:      stripe.String(customerDataRes.ID),
		StripeVersion: stripe.String(APIVersion),
	}
	ek, err := c.api.EphemeralKeys.New(ekParams)
	if err != nil {
		return nil, err
	}
	// Create Payment Intent
	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(order.Amount),
		Customer: stripe.String(customerDataRes.ID),
		Currency: stripe.String(order.Currency),
		PaymentMethodTypes: []*string{
			stripe.String(order.Payment),
		},
		Metadata: map[string]string{
			"order_no":  order.OrderNo,
			"user_id":   strconv.FormatInt(user.UserId, 10),
			"subscribe": order.Subscribe,
		},
	}
	params.SetIdempotencyKey(stripeIdempotencyKey("order", order.OrderNo))
	result, err := c.api.PaymentIntents.New(params)
	if err != nil {
		return nil, err
	}
	return &PaymentSheet{
		ClientSecret:   result.ClientSecret,
		EphemeralKey:   ek.Secret,
		Customer:       customerDataRes.ID,
		PublishableKey: c.PublicKey,
		TradeNo:        result.ID,
	}, nil
}

// SearchStripeCustomer  Search for a Stripe customer by email or user ID
func (c *Client) SearchStripeCustomer(user *User) (*stripe.Customer, error) {
	params := &stripe.CustomerSearchParams{}
	if user.UserId > 0 {
		params.SearchParams.Query = fmt.Sprintf("metadata['user_id']:'%d'", user.UserId)
	} else if user.Email != "" {
		params.SearchParams.Query = fmt.Sprintf("email:'%s'", stripeSearchLiteral(user.Email))
	} else {
		return nil, fmt.Errorf("Stripe customer identity is required")
	}
	result := c.api.Customers.Search(params)
	if result.Err() != nil {
		logger.Errorw("[Stripe] Search customer failed", logger.Field("error", result.Err().Error()))
		return nil, result.Err()
	}

	if len(result.CustomerSearchResult().Data) != 0 {
		return result.CustomerSearchResult().Data[0], nil
	}
	return nil, nil
}

// CreateCustomer Create a new Stripe customer
func (c *Client) CreateCustomer(user *User) (*stripe.Customer, error) {
	customerData := &stripe.CustomerParams{}
	if user.Email != "" {
		customerData.Email = &user.Email
	}
	customerData.AddMetadata("user_id", strconv.FormatInt(user.UserId, 10))
	customerData.SetIdempotencyKey(stripeIdempotencyKey("customer", stripeCustomerIdentity(user)))
	return c.api.Customers.New(customerData)
}

// QueryOrderStatus Query the status of the order
func (c *Client) QueryOrderStatus(orderNo string) (bool, error) {
	intent, err := c.api.PaymentIntents.Get(orderNo, nil)
	if err != nil {
		return false, err
	}
	return intent.Status == "succeeded", err
}

// ParseNotify
func (c *Client) ParseNotify(payload []byte, signature string) (*NotifyResult, error) {
	event, err := webhook.ConstructEventWithOptions(payload, signature, c.Config.WebhookSecret, webhook.ConstructEventOptions{
		IgnoreAPIVersionMismatch: true,
	})
	if err != nil {
		return nil, err
	}
	var paymentIntent stripe.PaymentIntent
	err = json.Unmarshal(event.Data.Raw, &paymentIntent)
	if err != nil {
		logger.Error("Failed to unmarshal payment intent", logger.Field("error", err.Error()))
		return nil, err
	}
	orderNo := paymentIntent.Metadata["order_no"]
	userId := paymentIntent.Metadata["user_id"]
	var method string
	if paymentIntent.PaymentMethod != nil && paymentIntent.PaymentMethod.ID != "" {
		logger.Debugw("[Stripe] Payment method received", logger.Field("payment_method", paymentIntent.PaymentMethod.ID))
		result, err := c.RetrievePaymentMethod(paymentIntent.PaymentMethod.ID)
		if err != nil {
			logger.Error("[stripe] Payment callback query payment method error", logger.Field("errors", err.Error()))
		}
		if result != nil {
			method = string(result.Type)
		}
	}
	// userId string 转 int64
	uid, _ := strconv.ParseInt(userId, 10, 64)
	return &NotifyResult{
		EventType: string(event.Type),
		OrderNo:   orderNo,
		TradeNo:   paymentIntent.ID,
		UserId:    uid,
		Method:    method,
		Amount:    paymentIntent.Amount,
	}, nil
}

// RetrievePaymentMethod 查询支付方式
func (c *Client) RetrievePaymentMethod(id string) (*stripe.PaymentMethod, error) {
	return c.api.PaymentMethods.Get(id, nil)
}

// CreateWebhookEndpoint 创建 webhook endpoint
func (c *Client) CreateWebhookEndpoint(url string) (*stripe.WebhookEndpoint, error) {
	params := &stripe.WebhookEndpointParams{
		URL: stripe.String(url),
		EnabledEvents: []*string{
			stripe.String("payment_intent.succeeded"),
			stripe.String("payment_intent.payment_failed"),
		},
	}
	return c.api.WebhookEndpoints.New(params)
}

func stripeCustomerIdentity(user *User) string {
	if user.UserId > 0 {
		return strconv.FormatInt(user.UserId, 10)
	}
	return user.Email
}

func stripeIdempotencyKey(kind, value string) string {
	digest := sha256.Sum256([]byte(value))
	return fmt.Sprintf("alpha-%s-%x", kind, digest)
}

func stripeSearchLiteral(value string) string {
	return strings.NewReplacer(`\`, `\\`, `'`, `\'`).Replace(value)
}
