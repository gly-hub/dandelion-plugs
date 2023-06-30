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
	Size         Size     `json:"size" yaml:"size"`
	FrontColors  []Color  `json:"front_colors" yaml:"frontColors"`
	BkgColors    []Color  `json:"bkg_colors" yaml:"bkgColors"`
	FontPath     []string `json:"font_path" yaml:"fontPath"`
	DisturbLevel string   `json:"disturb_level" yaml:"disturbLevel"`
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

type Plugin struct {
}

func (p *Plugin) Config() interface{} {
	Config = &config{}
	return Config
}

func (p *Plugin) InitPlugin() error {
	customOpt := captcha2.Option{
		Size:         &image.Point{X: Config.Size.X, Y: Config.Size.Y},
		FrontColors:  nil,
		BkgColors:    nil,
		FontPath:     nil,
		DisturbLevel: 0,
	}

	var frontColors []color.Color
	for _, v := range Config.FrontColors {
		frontColors = append(frontColors, color.RGBA{R: v.R, G: v.G, B: v.B, A: v.A})
	}
	customOpt.FrontColors = frontColors

	var bkgColors []color.Color
	for _, v := range Config.BkgColors {
		bkgColors = append(bkgColors, color.RGBA{R: v.R, G: v.G, B: v.B, A: v.A})
	}
	customOpt.BkgColors = bkgColors

	customOpt.FontPath = Config.FontPath

	switch Config.DisturbLevel {
	case "normal":
		customOpt.DisturbLevel = captcha2.NORMAL
	case "medium":
		customOpt.DisturbLevel = captcha2.MEDIUM
	case "high":
		customOpt.DisturbLevel = captcha2.HIGH
	}

	captcha = captcha2.New(customOpt)
	storage = InitStorage()
	return nil
}

func Create(key string, num int, t captcha2.StrType) (*captcha2.Image, error) {
	img, code := captcha.Create(num, t)
	if err := storage.Set(key, code); err != nil {
		return nil, err
	}
	return img, nil
}
