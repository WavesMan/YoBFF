import { translateMessage } from '../ErrorDrawerUtils';
import { describe, it, expect } from 'vitest';

describe('ErrorDrawerUtils', () => {
  it('translates by code', () => {
    expect(translateMessage('Some error', 'invalid_captcha')).toBe('验证码错误');
  });

  it('translates by message content', () => {
    expect(translateMessage('This is an internal error occurred')).toBe('服务器内部错误');
  });

  it('returns original message if no match', () => {
    expect(translateMessage('Unknown error')).toBe('Unknown error');
  });

  it('prioritizes code over message', () => {
    expect(translateMessage('internal error', 'invalid_username_or_password')).toBe('用户名或密码错误');
  });
});
