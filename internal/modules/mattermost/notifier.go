package mattermost

import (
	"context"
	"fmt"
	"unicode/utf8"

	mattermost "github.com/mattermost/mattermost/server/public/model"

	"github.com/nt0xa/sonar/internal/database"
	"github.com/nt0xa/sonar/internal/modules"
	"github.com/nt0xa/sonar/internal/modules/mattermost/message"
)

func (mm *Mattermost) Name() string { return "mattermost" }

func (mm *Mattermost) Notify(ctx context.Context, n *modules.Notification) error {
	if n.User.MattermostID == nil {
		return fmt.Errorf("user %d has no mattermost id", n.User.ID)
	}
	userID := *n.User.MattermostID

	channelID, err := mm.resolveChannelID(ctx, userID)
	if err != nil {
		return err
	}

	isSMTP := database.ProtoToCategory(n.Event.Protocol) == database.ProtoCategorySMTP

	var body string
	if isSMTP && n.Event.Meta.SMTP != nil {
		body = n.Event.Meta.SMTP.Email.Text
	} else {
		body = string(n.Event.RW)
	}

	var rootPost *mattermost.Post

	if utf8.ValidString(body) {
		msg, err := message.Build(n, body)
		if err != nil {
			return fmt.Errorf("failed to build message: %w", err)
		}

		rootPost, _, err = mm.client.CreatePost(ctx, &mattermost.Post{
			ChannelId: channelID,
			Message:   msg,
		})
		if err != nil {
			return fmt.Errorf("failed to post message: %w", err)
		}
	} else {
		// Binary body — send header only, attach raw bytes as file
		msg, err := message.Build(n, "")
		if err != nil {
			return fmt.Errorf("failed to build message: %w", err)
		}

		filename := fmt.Sprintf("log-%s-%s.bin",
			n.Payload.Name,
			n.Event.ReceivedAt.Format("15-04-05_02-Jan-2006"),
		)

		fileID, err := mm.uploadFile(ctx, channelID, []byte(n.Event.RW), filename)
		if err != nil {
			return err
		}

		rootPost, _, err = mm.client.CreatePost(ctx, &mattermost.Post{
			ChannelId: channelID,
			Message:   msg,
			FileIds:   mattermost.StringArray{fileID},
		})
		if err != nil {
			return fmt.Errorf("failed to post message: %w", err)
		}
	}

	// For SMTP events, attach .eml and raw session as thread replies
	if isSMTP && n.Event.Meta.SMTP != nil {
		ts := n.Event.ReceivedAt.Format("15-04-05_02-Jan-2006")

		if data := n.Event.Meta.SMTP.Session.Data; len(data) > 0 {
			fileID, err := mm.uploadFile(ctx, channelID, []byte(data),
				fmt.Sprintf("email-%s-%s.eml", n.Payload.Name, ts))
			if err != nil {
				return err
			}
			if _, _, err := mm.client.CreatePost(ctx, &mattermost.Post{
				ChannelId: channelID,
				RootId:    rootPost.Id,
				FileIds:   mattermost.StringArray{fileID},
			}); err != nil {
				return fmt.Errorf("failed to post eml attachment: %w", err)
			}
		}

		if len(n.Event.RW) > 0 {
			fileID, err := mm.uploadFile(ctx, channelID, n.Event.RW,
				fmt.Sprintf("smtp-%s-%s.txt", n.Payload.Name, ts))
			if err != nil {
				return err
			}
			if _, _, err := mm.client.CreatePost(ctx, &mattermost.Post{
				ChannelId: channelID,
				RootId:    rootPost.Id,
				FileIds:   mattermost.StringArray{fileID},
			}); err != nil {
				return fmt.Errorf("failed to post smtp txt attachment: %w", err)
			}
		}
	}

	return nil
}

func (mm *Mattermost) uploadFile(ctx context.Context, channelID string, data []byte, filename string) (string, error) {
	resp, _, err := mm.client.UploadFile(ctx, data, channelID, filename)
	if err != nil {
		return "", fmt.Errorf("failed to upload file %q: %w", filename, err)
	}
	if len(resp.FileInfos) == 0 {
		return "", fmt.Errorf("upload of %q returned no file info", filename)
	}
	return resp.FileInfos[0].Id, nil
}
