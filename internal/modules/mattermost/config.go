package mattermost

import validation "github.com/go-ozzo/ozzo-validation/v4"

type Config struct {
	Admin string `koanf:"admin"`
	URL   string `koanf:"url"`
	Token string `koanf:"token"`
}

func (c Config) Validate() error {
	return validation.ValidateStruct(&c,
		validation.Field(&c.Admin, validation.Required),
		validation.Field(&c.URL, validation.Required),
		validation.Field(&c.Token, validation.Required),
	)
}
