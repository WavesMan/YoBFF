export interface ErrorLogItem {
    id: string;
    message: string;
    errorCode?: string;
    requestId?: string;
    timestamp: number;
}

export const ERROR_TRANSLATIONS: Record<string, string> = {
    'invalid_captcha': '验证码错误',
    'invalid_username_or_password': '用户名或密码错误',
    'token_expired': '登录已过期',
    'not_found': '资源未找到',
    'internal_error': '服务器内部错误',
    'unauthorized': '未授权访问',
    'forbidden': '禁止访问',
    'bad_request': '请求参数错误',
};

export const translateMessage = (msg: string, code?: string): string => {
    if (code && ERROR_TRANSLATIONS[code]) {
        return ERROR_TRANSLATIONS[code];
    }
    
    for (const [key, value] of Object.entries(ERROR_TRANSLATIONS)) {
        if (msg.toLowerCase().includes(key.replace(/_/g, ' '))) {
            return value;
        }
    }

    return msg;
};
