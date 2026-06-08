package rpc

import (
	"context"

	"github.com/gotd/td/tg"

	"telesrv/internal/compat/tdesktop"
)

// registerPayments 注册第一阶段 TDesktop 启动所需 payments.* RPC 兼容响应。
func (r *Router) registerPayments(d *tg.ServerDispatcher) {
	d.OnPaymentsGetStarsStatus(func(ctx context.Context, req *tg.PaymentsGetStarsStatusRequest) (*tg.PaymentsStarsStatus, error) {
		balance := tg.StarsAmountClass(&tg.StarsAmount{})
		if req.GetTon() {
			balance = &tg.StarsTonAmount{}
		}
		return &tg.PaymentsStarsStatus{
			Balance: balance,
			Chats:   []tg.ChatClass{},
			Users:   []tg.UserClass{},
		}, nil
	})
	d.OnPaymentsGetStarGiftActiveAuctions(func(ctx context.Context, hash int64) (tg.PaymentsStarGiftActiveAuctionsClass, error) {
		return tdesktop.StarGiftActiveAuctions(), nil
	})
	d.OnPaymentsGetSavedStarGifts(func(ctx context.Context, req *tg.PaymentsGetSavedStarGiftsRequest) (*tg.PaymentsSavedStarGifts, error) {
		return tdesktop.SavedStarGifts(), nil
	})
	d.OnPaymentsGetSavedStarGift(func(ctx context.Context, stargift []tg.InputSavedStarGiftClass) (*tg.PaymentsSavedStarGifts, error) {
		return tdesktop.SavedStarGifts(), nil
	})
}
