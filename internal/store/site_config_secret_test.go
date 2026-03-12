package store

import (
	"encoding/base64"
	"encoding/hex"
	"strings"
	"testing"

	"YoBFF/internal/config"
)

// TestLoadSiteConfigEncryptionKey 验证密钥读取支持空值、raw/hex/base64 三种格式。
func TestLoadSiteConfigEncryptionKey(t *testing.T) {
	t.Run("empty env", func(t *testing.T) {
		t.Setenv("SITE_CONFIG_ENCRYPTION_KEY", "")
		got, ok := loadSiteConfigEncryptionKey()
		if ok || got != nil {
			t.Fatalf("读取空密钥应返回 ok=false: ok=%v len=%d", ok, len(got))
		}
	})

	t.Run("raw 32 chars", func(t *testing.T) {
		want := "12345678901234567890123456789012"
		t.Setenv("SITE_CONFIG_ENCRYPTION_KEY", want)
		got, ok := loadSiteConfigEncryptionKey()
		if !ok {
			t.Fatalf("读取 raw 32 字符密钥应成功")
		}
		if string(got) != want {
			t.Fatalf("密钥不匹配: got=%q", string(got))
		}
	})

	t.Run("hex 64 chars", func(t *testing.T) {
		raw := make([]byte, 32)
		for idx := range raw {
			raw[idx] = byte(idx)
		}
		hexText := hex.EncodeToString(raw)
		t.Setenv("SITE_CONFIG_ENCRYPTION_KEY", hexText)
		got, ok := loadSiteConfigEncryptionKey()
		if !ok {
			t.Fatalf("读取 hex 64 字符密钥应成功")
		}
		if hex.EncodeToString(got) != hexText {
			t.Fatalf("密钥不匹配: got=%q want=%q", hex.EncodeToString(got), hexText)
		}
	})

	t.Run("base64 32 bytes", func(t *testing.T) {
		raw := make([]byte, 32)
		for idx := range raw {
			raw[idx] = byte(100 + idx)
		}
		base64Text := base64.StdEncoding.EncodeToString(raw)
		t.Setenv("SITE_CONFIG_ENCRYPTION_KEY", base64Text)
		got, ok := loadSiteConfigEncryptionKey()
		if !ok {
			t.Fatalf("读取 base64 密钥应成功")
		}
		if base64.StdEncoding.EncodeToString(got) != base64Text {
			t.Fatalf("密钥不匹配")
		}
	})
}

// TestEncryptAndDecryptSiteConfigSecret 验证加解密基本闭环与非密文透传行为。
func TestEncryptAndDecryptSiteConfigSecret(t *testing.T) {
	t.Setenv("SITE_CONFIG_ENCRYPTION_KEY", "12345678901234567890123456789012")

	encrypted, err := encryptSiteConfigSecret("plain-text")
	if err != nil {
		t.Fatalf("加密失败: %v", err)
	}
	if !strings.HasPrefix(encrypted, siteConfigSecretPrefix) {
		t.Fatalf("密文前缀不匹配: %q", encrypted)
	}

	decrypted, err := decryptSiteConfigSecret(encrypted)
	if err != nil {
		t.Fatalf("解密失败: %v", err)
	}
	if decrypted != "plain-text" {
		t.Fatalf("解密结果不匹配: %q", decrypted)
	}

	// 对非本系统格式的输入，当前实现要求已配置密钥，但会原样返回值。
	plain, err := decryptSiteConfigSecret("not-encrypted")
	if err != nil {
		t.Fatalf("非密文解密不应失败: %v", err)
	}
	if plain != "not-encrypted" {
		t.Fatalf("非密文应原样返回: %q", plain)
	}
}

// TestDecryptSiteConfigSecret_InvalidCiphertext 验证非法密文会返回错误而非静默成功。
func TestDecryptSiteConfigSecret_InvalidCiphertext(t *testing.T) {
	t.Setenv("SITE_CONFIG_ENCRYPTION_KEY", "12345678901234567890123456789012")

	value := siteConfigSecretPrefix + base64.StdEncoding.EncodeToString([]byte("abc"))
	if _, err := decryptSiteConfigSecret(value); err == nil {
		t.Fatalf("非法密文应解密失败")
	}
}

// TestApplyDecryptedCDNSecrets 验证解密后用于内部执行且失败时清空敏感字段。
func TestApplyDecryptedCDNSecrets(t *testing.T) {
	t.Setenv("SITE_CONFIG_ENCRYPTION_KEY", "12345678901234567890123456789012")

	encrypted, err := encryptSiteConfigSecret("secret")
	if err != nil {
		t.Fatalf("加密失败: %v", err)
	}

	cfg := config.Config{
		Security: config.SecurityConfig{
			CDNProviderSettings: map[string]config.CDNProviderSetting{
				"cloudflare": {SecretKey: encrypted},
				"aliyun":     {SecretKey: siteConfigSecretPrefix + "not-base64"},
			},
		},
	}

	got := applyDecryptedCDNSecrets(cfg)
	if got.Security.CDNProviderSettings["cloudflare"].SecretKey != "secret" {
		t.Fatalf("cloudflare 解密结果错误: %q", got.Security.CDNProviderSettings["cloudflare"].SecretKey)
	}
	if got.Security.CDNProviderSettings["aliyun"].SecretKey != "" {
		t.Fatalf("解密失败应清空 SecretKey: %q", got.Security.CDNProviderSettings["aliyun"].SecretKey)
	}

	// 原配置不应被修改
	if cfg.Security.CDNProviderSettings["cloudflare"].SecretKey != encrypted {
		t.Fatalf("原配置被意外修改")
	}
}

// TestMergeAndEncryptCDNSecrets 验证“只写不回显”语义与密文/明文输入处理规则。
func TestMergeAndEncryptCDNSecrets(t *testing.T) {
	t.Setenv("SITE_CONFIG_ENCRYPTION_KEY", "12345678901234567890123456789012")

	existingEncrypted, err := encryptSiteConfigSecret("existing")
	if err != nil {
		t.Fatalf("加密失败: %v", err)
	}

	existing := config.Config{
		Security: config.SecurityConfig{
			CDNProviderSettings: map[string]config.CDNProviderSetting{
				"cloudflare": {SecretKey: existingEncrypted},
			},
		},
	}

	t.Run("nil incoming settings", func(t *testing.T) {
		incoming := config.Config{}
		got, err := mergeAndEncryptCDNSecrets(existing, incoming)
		if err != nil {
			t.Fatalf("不应报错: %v", err)
		}
		if got.Security.CDNProviderSettings != nil {
			t.Fatalf("入参为 nil map 时应原样返回")
		}
	})

	t.Run("empty secret preserves existing", func(t *testing.T) {
		incoming := config.Config{
			Security: config.SecurityConfig{
				CDNProviderSettings: map[string]config.CDNProviderSetting{
					"cloudflare": {SecretKey: ""},
				},
			},
		}
		got, err := mergeAndEncryptCDNSecrets(existing, incoming)
		if err != nil {
			t.Fatalf("合并失败: %v", err)
		}
		if got.Security.CDNProviderSettings["cloudflare"].SecretKey != existingEncrypted {
			t.Fatalf("应保留已有密文")
		}
	})

	t.Run("plain secret gets encrypted", func(t *testing.T) {
		incoming := config.Config{
			Security: config.SecurityConfig{
				CDNProviderSettings: map[string]config.CDNProviderSetting{
					"cloudflare": {SecretKey: "plain"},
				},
			},
		}
		got, err := mergeAndEncryptCDNSecrets(existing, incoming)
		if err != nil {
			t.Fatalf("合并失败: %v", err)
		}
		secret := got.Security.CDNProviderSettings["cloudflare"].SecretKey
		if !strings.HasPrefix(secret, siteConfigSecretPrefix) {
			t.Fatalf("加密结果应包含前缀: %q", secret)
		}
		if secret == "plain" {
			t.Fatalf("密文不应等于明文")
		}
	})

	t.Run("encrypted secret keeps as-is", func(t *testing.T) {
		incoming := config.Config{
			Security: config.SecurityConfig{
				CDNProviderSettings: map[string]config.CDNProviderSetting{
					"cloudflare": {SecretKey: existingEncrypted},
				},
			},
		}
		got, err := mergeAndEncryptCDNSecrets(existing, incoming)
		if err != nil {
			t.Fatalf("合并失败: %v", err)
		}
		if got.Security.CDNProviderSettings["cloudflare"].SecretKey != existingEncrypted {
			t.Fatalf("已是密文时应保持不变")
		}
	})
}
