package message

import (
	"fmt"
	"net"
	"strings"

	"github.com/nt0xa/sonar/internal/database"
	"github.com/nt0xa/sonar/internal/modules"
	"github.com/nt0xa/sonar/internal/templates"
)

// Build creates a Markdown-formatted Mattermost message from a notification.
func Build(n *modules.Notification, body string) (string, error) {
	host, _, err := net.SplitHostPort(n.Event.RemoteAddr)
	if err != nil {
		return "", fmt.Errorf("failed to split host port: %w", err)
	}

	var sb strings.Builder

	// Header
	fmt.Fprintf(&sb, "#### %s [%s] %s\n",
		headerEmoji(n.Event.Protocol),
		n.Payload.Name,
		strings.ToUpper(n.Event.Protocol),
	)

	// IP and time
	fmt.Fprintf(&sb, "**IP**: %s | **Time**: %s\n",
		host,
		n.Event.ReceivedAt.Format("02 Jan 2006 15:04:05 MST"),
	)

	// GeoIP
	if geoip := n.Event.Meta.GeoIP; geoip != nil {
		location := fmt.Sprintf("%s %s", templates.FlagEmoji(geoip.Country.ISOCode), geoip.Country.Name)
		if geoip.City != "" {
			location += ", " + geoip.City
		}
		fmt.Fprintf(&sb, "**Location**: %s | **Org**: %s (AS%d)\n",
			location,
			geoip.ASN.Org,
			geoip.ASN.Number,
		)
	}

	// SMTP metadata
	if n.Event.Meta.SMTP != nil {
		email := n.Event.Meta.SMTP.Email
		if len(email.From) > 0 {
			addrs := make([]string, 0, len(email.From))
			for _, f := range email.From {
				addrs = append(addrs, f.Address)
			}
			fmt.Fprintf(&sb, "**From**: %s\n", strings.Join(addrs, ", "))
		}
		if email.Subject != "" {
			fmt.Fprintf(&sb, "**Subject**: %s\n", email.Subject)
		}
	}

	// Body
	if body != "" {
		fmt.Fprintf(&sb, "\n```\n%s\n```\n", body)
	}

	return sb.String(), nil
}

func headerEmoji(protocol string) string {
	switch database.ProtoToCategory(protocol) {
	case database.ProtoCategoryDNS:
		return ":mag:"
	case database.ProtoCategoryFTP:
		return ":file_folder:"
	case database.ProtoCategorySMTP:
		return ":email:"
	case database.ProtoCategoryHTTP:
		return ":globe_with_meridians:"
	default:
		return ":bell:"
	}
}
