package app

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Mikhalevich/tgbot"

	"github.com/Mikhalevich/filestorage/cmd/bot/internal/app/handler"
	"github.com/Mikhalevich/filestorage/cmd/bot/internal/config"
	"github.com/Mikhalevich/filestorage/internal/infra/tracing"
)

// Start runs telegram bot saving received files into fileSvc until ctx is canceled.
func Start(
	ctx context.Context,
	cfg config.Bot,
	fileSvc handler.FileService,
) error {
	opts := []tgbot.Option{
		tgbot.WithNewTracerFn(func() tgbot.Tracer {
			return newTracer(tracing.StdTracer())
		}),
		tgbot.WithReadTimeout(cfg.ReadTimeout),
		tgbot.WithWriteTimeout(cfg.WriteTimeout),
		tgbot.WithShutdownTimeout(cfg.ShutdownTimeout),
	}

	if cfg.WebhookToken != "" {
		opts = append(opts, tgbot.WithWebHookToken(cfg.WebhookToken))
	}

	bot, err := tgbot.New(cfg.Token, opts...)
	if err != nil {
		return fmt.Errorf("create bot: %w", err)
	}

	bot.AddDefaultHandler(handler.New(
		fileSvc,
		&http.Client{Timeout: cfg.DownloadTimeout},
	).Upload)

	if err := bot.Start(ctx); err != nil {
		return fmt.Errorf("start bot: %w", err)
	}

	return nil
}
