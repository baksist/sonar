package smtpx

// Meta contains SMTP-specific event metadata.
type Meta struct {
	Session Data  `json:"session"`
	Email   Email `json:"email"`
}

// Recipients returns email recipients addresses. Addresses from the "To"
// header are preferred, but if it is missing or unparsable, envelope
// "RCPT TO" addresses are used instead.
func (m *Meta) Recipients() []string {
	if len(m.Email.To) > 0 {
		res := make([]string, 0, len(m.Email.To))
		for _, addr := range m.Email.To {
			res = append(res, addr.Address)
		}
		return res
	}

	return m.Session.RcptTo
}
