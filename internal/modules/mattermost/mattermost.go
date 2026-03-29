package mattermost

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	mattermost "github.com/mattermost/mattermost/server/public/model"

	"github.com/nt0xa/sonar/internal/actions"
	"github.com/nt0xa/sonar/internal/actionsdb"
	"github.com/nt0xa/sonar/internal/cmd"
	"github.com/nt0xa/sonar/internal/database"
	"github.com/nt0xa/sonar/internal/templates"
	"github.com/nt0xa/sonar/pkg/telemetry"
)

type Mattermost struct {
	client    *mattermost.Client4
	botUserID string
	db        *database.DB
	tel       telemetry.Telemetry
	log       *slog.Logger
	cmd       *cmd.Command
	actions   actions.Actions
	tmpl      *templates.Templates
	url       string
}

func New(cfg *Config, db *database.DB, log *slog.Logger, tel telemetry.Telemetry, actions actions.Actions, domain string) (*Mattermost, error) {
	client := mattermost.NewAPIv4Client(cfg.URL)
	client.SetToken(cfg.Token)

	ctx := context.Background()
	me, _, err := client.GetMe(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("mattermost auth test failed: %w", err)
	}

	tmpl := templates.New(domain, templates.Default(
		templates.HTMLEscape(false),
		templates.Markup(
			templates.Bold("**", "**"),
			templates.CodeBlock("```", "```"),
			templates.CodeInline("`", "`"),
		),
	))

	mm := &Mattermost{
		client:    client,
		botUserID: me.Id,
		db:        db,
		tel:       tel,
		log:       log,
		cmd:       cmd.New(actions),
		actions:   actions,
		tmpl:      tmpl,
		url:       cfg.URL,
	}
	return mm, nil
}

func (mm *Mattermost) Start() error {
	wsURL := strings.Replace(mm.url, "http", "ws", 1)

	wsClient, _ := mattermost.NewWebSocketClient4(wsURL, mm.client.AuthToken)

	go func() {
		for evt := range wsClient.EventChannel {
			switch evt.EventType() {
			case mattermost.WebsocketEventPosted:
				data, _ := evt.GetData()["post"].(string)
				var post mattermost.Post
				if err := json.Unmarshal([]byte(data), &post); err != nil {
					mm.log.Error("error decoding post", "err", err)
					continue
				}
				if post.UserId == mm.botUserID {
					continue
				}
				mm.processCommand(context.TODO(), &post)
			}
		}
	}()

	mm.log.Info("Starting Mattermost WebSocket client")
	wsClient.Listen()

	if wsClient.ListenError != nil {
		return fmt.Errorf("mattermost websocket error: %w", wsClient.ListenError)
	}

	return nil
}

func (mm *Mattermost) processCommand(ctx context.Context, post *mattermost.Post) {

	var uid string
	mmChannel, _, err := mm.client.GetChannel(context.Background(), post.ChannelId)
	switch mmChannel.Type {
	case mattermost.ChannelTypeDirect:
		uid = post.UserId
	default:
		uid = post.ChannelId
	}
	mm.log.Info("channel", "ch", mmChannel.Name)

	chatUser, err := mm.db.UsersGetByMattermostID(ctx, uid)
	if err != nil {
		chatUser = nil
		if mmChannel.Name == "town-square" {
			return
		}
	}

	ctx = actionsdb.SetUser(ctx, chatUser)
	ctx = actionsdb.SetSource(ctx, "mattermost")

	reply := ""

	stdout, stderr, err := mm.cmd.ParseAndExec(ctx, post.Message,
		func(ctx context.Context, res actions.Result) error {
			content, err := mm.tmpl.RenderResult(res)
			if err != nil {
				return err
			}
			reply = content
			return nil
		},
	)

	if err != nil {
		reply = err.Error()
	}
	if stdout != "" {
		reply = stdout
	}
	if stderr != "" {
		reply = stderr
	}

	if reply == "" {
		return
	}

	if _, _, err := mm.client.CreatePost(ctx, &mattermost.Post{
		ChannelId: post.ChannelId,
		Message:   reply,
	}); err != nil {
		mm.log.Error("Failed to send Mattermost command reply", "err", err)
	}
}

func (mm *Mattermost) resolveChannelID(ctx context.Context, mattermostID string) (string, error) {
	ch, _, err := mm.client.GetChannel(ctx, mattermostID)
	if err == nil {
		switch ch.Type {
		case mattermost.ChannelTypeOpen, mattermost.ChannelTypePrivate:
			return ch.Id, nil
		}
	}
	// Fall back to creating a DM with the user ID.
	dm, _, err := mm.client.CreateDirectChannel(ctx, mm.botUserID, mattermostID)
	if err != nil {
		return "", fmt.Errorf("failed to resolve notification channel for %q: %w", mattermostID, err)
	}
	return dm.Id, nil
}
