import React, { createContext, useContext, useState, useCallback, useRef } from 'react';
import { createPortal } from 'react-dom';
import { FiCheckCircle, FiAlertCircle, FiAlertTriangle, FiInfo, FiX } from 'react-icons/fi';
import './Toast.css';

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
}

interface ToastContextType {
  toast: (message: string, type?: ToastType, options?: ToastOptions) => void;
  success: (message: string, options?: ToastOptions) => void;
  error: (message: string, options?: ToastOptions) => void;
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

  const addToast = useCallback((message: string, type: ToastType = 'info', options: ToastOptions = {}) => {
    const id = Math.random().toString(36).slice(2, 11);
    const duration = options.duration || 3000;

    const newToast: ToastItem = {
      id,
      type,
      message,
      title: options.title,
      duration,
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
  const error = useCallback((message: string, options?: ToastOptions) => addToast(message, 'error', options), [addToast]);
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
                <div className="toast-message">{t.message}</div>
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
        </div>,
        document.body
      )}
    </ToastContext.Provider>
  );
};
