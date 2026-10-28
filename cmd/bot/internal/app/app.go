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
	bot, err := tgbot.New(
		cfg.Token,
		tgbot.WithWorkers(cfg.Workers),
		tgbot.WithNewTracerFn(func() tgbot.Tracer {
			return newTracer(tracing.StdTracer())
		}),
		tgbot.WithWebHookToken(cfg.WebhookToken),
		tgbot.WithReadTimeout(cfg.ReadTimeout),
		tgbot.WithWriteTimeout(cfg.WriteTimeout),
		tgbot.WithShutdownTimeout(cfg.ShutdownTimeout))
	if err != nil {
		return fmt.Errorf("create bot: %w", err)
	}

	hndlr := handler.New(
		fileSvc,
		&http.Client{},
	)

	routes(bot, hndlr)

	if err := bot.Start(ctx); err != nil {
		return fmt.Errorf("start bot: %w", err)
	}

	return nil
}

func routes(bot *tgbot.TGBot, hndlr *handler.Handler) {
	bot.AddDefaultHandler(hndlr.Upload)
}
