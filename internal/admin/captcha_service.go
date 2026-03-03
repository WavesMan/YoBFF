package admin

import "github.com/mojocn/base64Captcha"

type captchaService interface {
	Generate(preset string) (id string, imageBase64 string, err error)
	Verify(id string, code string) bool
}

type base64CaptchaService struct {
	store    base64Captcha.Store
	captchas map[string]*base64Captcha.Captcha
}

// newBase64CaptchaService 创建基于内存存储的验证码服务实现。
// 参数：无。
// 返回：captchaService 实例与错误。
// 异常：创建失败时返回错误。
func newBase64CaptchaService() (captchaService, error) {
	store := base64Captcha.DefaultMemStore

	easyDriver := base64Captcha.NewDriverString(
		50,
		180,
		0,
		0,
		4,
		"23456789",
		nil,
		base64Captcha.DefaultEmbeddedFonts,
		[]string{"RitaSmith.ttf"},
	).ConvertFonts()

	defaultDriver := base64Captcha.NewDriverString(
		40,
		150,
		3,
		0,
		4,
		"234567890abcdefghjkmnpqrstuvwxyz",
		nil,
		base64Captcha.DefaultEmbeddedFonts,
		[]string{"RitaSmith.ttf"},
	).ConvertFonts()

	return &base64CaptchaService{
		store: store,
		captchas: map[string]*base64Captcha.Captcha{
			"default": base64Captcha.NewCaptcha(defaultDriver, store),
			"easy":    base64Captcha.NewCaptcha(easyDriver, store),
		},
	}, nil
}

// Generate 生成验证码并返回 ID 与图片 Base64。
// 参数：preset 为验证码预设，支持 easy 与 default。
// 返回：id 为验证码标识，imageBase64 为图片内容。
// 异常：底层生成失败时返回错误。
func (s *base64CaptchaService) Generate(preset string) (string, string, error) {
	if preset == "" {
		preset = "easy"
	}
	captcha := s.captchas[preset]
	if captcha == nil {
		captcha = s.captchas["easy"]
	}
	id, imageBase64, _, err := captcha.Generate()
	if err != nil {
		return "", "", err
	}
	return id, imageBase64, nil
}

// Verify 校验验证码并在成功时销毁答案，避免重放。
// 参数：id 为验证码标识，code 为用户输入。
// 返回：true 表示校验通过。
// 异常：无。
func (s *base64CaptchaService) Verify(id string, code string) bool {
	return s.store.Verify(id, code, true)
}
