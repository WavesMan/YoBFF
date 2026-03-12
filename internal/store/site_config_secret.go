package store

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"

	"YoBFF/internal/config"
)

const siteConfigSecretPrefix = "enc:v1:"

// loadSiteConfigEncryptionKey 从环境变量读取站点配置加密密钥，用于 SecretKey 的静态加解密。
// 约束：仅接受 32 字节密钥，支持 hex/base64/原始 32 字符三种输入格式。
func loadSiteConfigEncryptionKey() ([]byte, bool) {
	raw := strings.TrimSpace(config.EnvOrDefault("SITE_CONFIG_ENCRYPTION_KEY", ""))
	if raw == "" {
		return nil, false
	}
	if len(raw) == 64 {
		if decoded, err := hex.DecodeString(raw); err == nil && len(decoded) == 32 {
			return decoded, true
		}
	}
	if decoded, err := base64.StdEncoding.DecodeString(raw); err == nil && len(decoded) == 32 {
		return decoded, true
	}
	if len(raw) == 32 {
		return []byte(raw), true
	}
	return nil, false
}

// encryptSiteConfigSecret 将站点配置中的 SecretKey 加密为可存储的密文格式，避免明文落库。
// 异常：当未配置加密密钥或加密失败时返回错误，调用方应中止保存以避免明文写入。
func encryptSiteConfigSecret(plain string) (string, error) {
	key, ok := loadSiteConfigEncryptionKey()
	if !ok {
		return "", errors.New("站点配置加密密钥未配置")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nil, nonce, []byte(plain), nil)
	buf := make([]byte, 0, len(nonce)+len(ciphertext))
	buf = append(buf, nonce...)
	buf = append(buf, ciphertext...)
	return siteConfigSecretPrefix + base64.StdEncoding.EncodeToString(buf), nil
}

// decryptSiteConfigSecret 解密站点配置中的 SecretKey 密文，供内部任务调用第三方 API 使用。
// 说明：对非本系统密文格式的输入直接原样返回，便于历史数据逐步迁移。
func decryptSiteConfigSecret(value string) (string, error) {
	key, ok := loadSiteConfigEncryptionKey()
	if !ok {
		return "", errors.New("站点配置加密密钥未配置")
	}
	if !strings.HasPrefix(value, siteConfigSecretPrefix) {
		return value, nil
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(value, siteConfigSecretPrefix))
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(raw) < nonceSize {
		return "", errors.New("密文格式非法")
	}
	nonce := raw[:nonceSize]
	ciphertext := raw[nonceSize:]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

// applyDecryptedCDNSecrets 将站点配置中已加密的 SecretKey 解密为明文，供内部执行使用。
// 异常：解密失败时清空对应 SecretKey，确保不会把不可用密文当作凭证继续传递。
func applyDecryptedCDNSecrets(cfg config.Config) config.Config {
	if cfg.Security.CDNProviderSettings == nil {
		return cfg
	}
	next := cfg
	next.Security.CDNProviderSettings = make(map[string]config.CDNProviderSetting, len(cfg.Security.CDNProviderSettings))
	for provider, settings := range cfg.Security.CDNProviderSettings {
		secret := strings.TrimSpace(settings.SecretKey)
		if strings.HasPrefix(secret, siteConfigSecretPrefix) {
			plain, err := decryptSiteConfigSecret(secret)
			if err != nil {
				settings.SecretKey = ""
			} else {
				settings.SecretKey = plain
			}
		}
		next.Security.CDNProviderSettings[provider] = settings
	}
	return next
}

// mergeAndEncryptCDNSecrets 合并并加密入参中的 SecretKey，实现“只写不回显”语义。
// 规则：当入参 SecretKey 为空时保留已有密文；当入参包含明文 SecretKey 时加密后写入。
func mergeAndEncryptCDNSecrets(existing config.Config, incoming config.Config) (config.Config, error) {
	if incoming.Security.CDNProviderSettings == nil {
		return incoming, nil
	}
	next := incoming
	next.Security.CDNProviderSettings = make(map[string]config.CDNProviderSetting, len(incoming.Security.CDNProviderSettings))
	for provider, settings := range incoming.Security.CDNProviderSettings {
		secret := strings.TrimSpace(settings.SecretKey)
		if secret == "" {
			if existing.Security.CDNProviderSettings != nil {
				if previous, ok := existing.Security.CDNProviderSettings[provider]; ok {
					settings.SecretKey = previous.SecretKey
				}
			}
		} else if !strings.HasPrefix(secret, siteConfigSecretPrefix) {
			encrypted, err := encryptSiteConfigSecret(secret)
			if err != nil {
				return config.Config{}, err
			}
			settings.SecretKey = encrypted
		}
		next.Security.CDNProviderSettings[provider] = settings
	}
	return next, nil
}
