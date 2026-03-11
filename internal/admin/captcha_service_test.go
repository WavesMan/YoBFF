package admin

import "testing"

func TestBase64CaptchaService_GenerateAndVerify(t *testing.T) {
	serviceValue, err := newBase64CaptchaService()
	if err != nil {
		t.Fatalf("初始化验证码服务失败: %v", err)
	}
	service, ok := serviceValue.(*base64CaptchaService)
	if !ok {
		t.Fatalf("验证码服务类型不匹配")
	}

	cases := []struct {
		name   string
		preset string
	}{
		{name: "empty preset fallback", preset: ""},
		{name: "easy preset", preset: "easy"},
		{name: "unknown preset fallback", preset: "unknown"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			id, imageBase64, generateErr := service.Generate(tt.preset)
			if generateErr != nil {
				t.Fatalf("生成验证码失败: %v", generateErr)
			}
			if id == "" || imageBase64 == "" {
				t.Fatalf("验证码结果为空: id=%q image=%q", id, imageBase64)
			}
			if service.Verify(id, "wrong-code") {
				t.Fatalf("错误验证码不应校验通过")
			}
		})
	}
}
