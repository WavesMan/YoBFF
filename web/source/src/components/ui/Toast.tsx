import React, { createContext, useContext, useState, useCallback, useRef } from 'react';
import { createPortal } from 'react-dom';
import { FiCheckCircle, FiAlertCircle, FiAlertTriangle, FiInfo, FiX } from 'react-icons/fi';
import './Toast.css';
import { ErrorDrawer } from './ErrorDrawer';
import { translateMessage, type ErrorLogItem } from './ErrorDrawerUtils';
import { RequestError } from '../../admin/api';

export type ToastType = 'success' | 'error' | 'warning' | 'info';

export interface ToastOptions {
  title?: string;
  duration?: number; // ms
}

interface ToastItem {
  id: string;
  type: ToastType;
  message: string;
  title?: string;
  duration: number;
  exiting?: boolean;
  errorCode?: string;
  requestId?: string;
}

interface ToastContextType {
  toast: (message: string, type?: ToastType, options?: ToastOptions) => void;
  success: (message: string, options?: ToastOptions) => void;
  error: (content: string | Error | RequestError, options?: ToastOptions) => void;
  warning: (message: string, options?: ToastOptions) => void;
  info: (message: string, options?: ToastOptions) => void;
}

const ToastContext = createContext<ToastContextType | undefined>(undefined);

// eslint-disable-next-line react-refresh/only-export-components
export const useToast = () => {
  const context = useContext(ToastContext);
  if (!context) {
    throw new Error('useToast must be used within a ToastProvider');
  }
  return context;
};

export const ToastProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [toasts, setToasts] = useState<ToastItem[]>([]);
  const [errorLog, setErrorLog] = useState<ErrorLogItem[]>([]);
  const timeoutsRef = useRef<Record<string, ReturnType<typeof setTimeout>>>({});

  const removeToast = useCallback((id: string) => {
    setToasts((prev) => prev.map(t => t.id === id ? { ...t, exiting: true } : t));
    
    // Wait for animation to finish before removing from DOM
    setTimeout(() => {
      setToasts((prev) => prev.filter((t) => t.id !== id));
      if (timeoutsRef.current[id]) {
        clearTimeout(timeoutsRef.current[id]);
        delete timeoutsRef.current[id];
      }
    }, 300);
  }, []);

  const addToast = useCallback((message: string, type: ToastType = 'info', options: ToastOptions & { errorCode?: string; requestId?: string } = {}) => {
    const id = Math.random().toString(36).slice(2, 11);
    const duration = options.duration || 3000;

    const newToast: ToastItem = {
      id,
      type,
      message,
      title: options.title,
      duration,
      errorCode: options.errorCode,
      requestId: options.requestId,
    };

    setToasts((prev) => [...prev, newToast]);

    if (duration > 0) {
      timeoutsRef.current[id] = setTimeout(() => {
        removeToast(id);
      }, duration);
    }
  }, [removeToast]);

  const toast = useCallback((message: string, type: ToastType = 'info', options?: ToastOptions) => {
    addToast(message, type, options);
  }, [addToast]);

  const success = useCallback((message: string, options?: ToastOptions) => addToast(message, 'success', options), [addToast]);
  
  const error = useCallback((content: string | Error | RequestError, options?: ToastOptions) => {
    let message = '';
    let errorCode: string | undefined;
    let requestId: string | undefined;

    if (content instanceof RequestError) {
      message = content.message;
      errorCode = content.error_code;
      requestId = content.request_id;
    } else if (content instanceof Error) {
      message = content.message;
    } else {
      message = String(content);
    }

    // 尝试从字符串中解析 [error_code=...] 和 [request_id=...]
    if (!errorCode) {
      const codeMatch = message.match(/\[error_code=([^\]]+)\]/);
      if (codeMatch) {
        errorCode = codeMatch[1];
        // 可选：从显示的消息中移除这些标记，保持界面整洁？
        // 用户需求似乎是格式化显示，所以这里可以保留或移除，ErrorDrawer 会单独展示
        // 这里我们保留 message 原样用于 Toast 显示（或简化之），但提取出 metadata 存入 Log
      }
    }
    if (!requestId) {
      const idMatch = message.match(/\[request_id=([^\]]+)\]/);
      if (idMatch) {
        requestId = idMatch[1];
      }
    }

    // 添加到 Toast 显示
    // 错误类型的 Toast 建议显示时间稍短，或者保持一致，反正会进入历史
    addToast(message, 'error', { ...options, errorCode, requestId });

    // 添加到错误日志
    const logItem: ErrorLogItem = {
      id: Math.random().toString(36).slice(2, 11),
      message,
      errorCode,
      requestId,
      timestamp: Date.now(),
    };
    setErrorLog((prev) => [logItem, ...prev]);

  }, [addToast]);

  const warning = useCallback((message: string, options?: ToastOptions) => addToast(message, 'warning', options), [addToast]);
  const info = useCallback((message: string, options?: ToastOptions) => addToast(message, 'info', options), [addToast]);

  const getIcon = (type: ToastType) => {
    switch (type) {
      case 'success': return <FiCheckCircle />;
      case 'error': return <FiAlertCircle />;
      case 'warning': return <FiAlertTriangle />;
      case 'info': return <FiInfo />;
      default: return <FiInfo />;
    }
  };

  return (
    <ToastContext.Provider value={{ toast, success, error, warning, info }}>
      {children}
      {typeof document !== 'undefined' && createPortal(
        <>
          <div className="toast-container">
            {toasts.map((t) => (
              <div 
                key={t.id} 
                className={`toast toast-${t.type} ${t.exiting ? 'exiting' : ''}`}
                role="alert"
              >
                <div className="toast-icon">{getIcon(t.type)}</div>
                <div className="toast-content">
                  {t.title && <div className="toast-title">{t.title}</div>}
                  <div className="toast-message">
                    {t.type === 'error' && t.errorCode ? (
                      <>
                         <span className="toast-error-code">[{t.errorCode}]</span> {translateMessage(t.message, t.errorCode)}
                      </>
                    ) : (
                      t.message
                    )}
                  </div>
                  {t.type === 'error' && t.requestId && (
                    <div className="toast-request-id">[request_id={t.requestId}]</div>
                  )}
                </div>
                <button 
                  className="toast-close" 
                  onClick={() => removeToast(t.id)}
                  aria-label="Close"
                >
                  <FiX size={16} />
                </button>
              </div>
            ))}
          </div>
          
          {/* 错误日志抽屉 */}
          <ErrorDrawer errors={errorLog} onClear={() => setErrorLog([])} />
        </>,
        document.body
      )}
    </ToastContext.Provider>
  );
};
