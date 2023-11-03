package captcha

import (
	captcha2 "github.com/gly-hub/toolbox/captcha"
	"image"
	"image/color"
)

var (
	Config  *config
	captcha *captcha2.Captcha
	storage *Storage
)

type config struct {
	Captcha captchaConfig `json:"captcha" yaml:"captcha"`
}

type captchaConfig struct {
	Length       int      `json:"length" yaml:"length"`
	StrType      int      `json:"str_type" yaml:"strType"`
	Size         Size     `json:"size" yaml:"size"`
	FrontColors  []Color  `json:"front_colors" yaml:"frontColors"`
	BkgColors    []Color  `json:"bkg_colors" yaml:"bkgColors"`
	FontPath     []string `json:"font_path" yaml:"fontPath"`
	DisturbLevel string   `json:"disturb_level" yaml:"disturbLevel"`
	Expire       int      `json:"expire" yaml:"expire"`
}

type Size struct {
	X int `json:"x" yaml:"x"`
	Y int `json:"y" yaml:"y"`
}

type Color struct {
	R uint8 `json:"r" yaml:"r"`
	G uint8 `json:"g" yaml:"g"`
	B uint8 `json:"b" yaml:"b"`
	A uint8 `json:"a" yaml:"a"`
}

func Plug() *Plugin {
	return &Plugin{}
}

type Plugin struct {
}

func (p *Plugin) Config() interface{} {
	Config = &config{}
	return Config
}

func (p *Plugin) InitPlugin() error {
	customOpt := captcha2.Option{
		Size:         &image.Point{X: Config.Captcha.Size.X, Y: Config.Captcha.Size.Y},
		FrontColors:  nil,
		BkgColors:    nil,
		FontPath:     nil,
		DisturbLevel: 0,
	}

	var frontColors []color.Color
	for _, v := range Config.Captcha.FrontColors {
		frontColors = append(frontColors, color.RGBA{R: v.R, G: v.G, B: v.B, A: v.A})
	}
	customOpt.FrontColors = frontColors

	var bkgColors []color.Color
	for _, v := range Config.Captcha.BkgColors {
		bkgColors = append(bkgColors, color.RGBA{R: v.R, G: v.G, B: v.B, A: v.A})
	}
	customOpt.BkgColors = bkgColors

	customOpt.FontPath = Config.Captcha.FontPath

	switch Config.Captcha.DisturbLevel {
	case "normal":
		customOpt.DisturbLevel = captcha2.NORMAL
	case "medium":
		customOpt.DisturbLevel = captcha2.MEDIUM
	case "high":
		customOpt.DisturbLevel = captcha2.HIGH
	}

	if Config.Captcha.Length == 0 {
		Config.Captcha.Length = 4
	}

	if Config.Captcha.Expire == 0 {
		Config.Captcha.Expire = 300
	}

	captcha = captcha2.New(customOpt)
	storage = InitStorage()
	return nil
}

func Create(key string) (*captcha2.Image, string, error) {
	img, code := captcha.Create(Config.Captcha.Length, captcha2.StrType(Config.Captcha.StrType))
	if err := storage.Set(key, code); err != nil {
		return nil, "", err
	}

	return img, code, nil
}

func Verify(key string, code string) bool {
	value, err := storage.Get(key, true)
	if err != nil {
		return false
	}
	if value == code {
		return true
	}
	return false
}
