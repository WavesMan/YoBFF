import React, { useEffect, useRef } from 'react';
import { createPortal } from 'react-dom';
import { FiX } from 'react-icons/fi';
import './Drawer.css';

interface DrawerProps {
  isOpen: boolean;
  onClose: () => void;
  title: React.ReactNode;
  headerExtra?: React.ReactNode;
  children: React.ReactNode;
  footer?: React.ReactNode;
  width?: number | string;
  bodyClassName?: string;
}

export const Drawer: React.FC<DrawerProps> = ({
  isOpen,
  onClose,
  title,
  headerExtra,
  children,
  footer,
  width,
  bodyClassName,
}) => {
  const drawerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose();
      }
    };

    if (isOpen) {
      document.addEventListener('keydown', handleEscape);
      document.body.style.overflow = 'hidden';
    }

    return () => {
      document.removeEventListener('keydown', handleEscape);
      document.body.style.overflow = '';
    };
  }, [isOpen, onClose]);

  const handleOverlayClick = (e: React.MouseEvent) => {
    // Only close if clicking the overlay itself, not the drawer
    if (e.target === e.currentTarget) {
      onClose();
    }
  };

  const titleId = React.useId();

  if (!isOpen && typeof document === 'undefined') return null;

  const content = (
    <div 
      className={`drawer-overlay ${isOpen ? 'open' : ''}`} 
      onClick={handleOverlayClick}
    >
      <div 
        className="drawer" 
        ref={drawerRef} 
        style={width ? { maxWidth: width, width: '100%' } : {}}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
      >
        <div className="drawer-header">
          <div className="flex items-center gap-4">
            <button className="drawer-close" onClick={onClose} aria-label="Close">
              <FiX size={20} />
            </button>
            <div className="drawer-title" id={titleId}>{title}</div>
          </div>
          {headerExtra && <div className="drawer-header-extra">{headerExtra}</div>}
        </div>
        <div className={`drawer-content ${bodyClassName || ''}`}>{children}</div>
        {footer && <div className="drawer-footer">{footer}</div>}
      </div>
    </div>
  );

  return createPortal(content, document.body);
};
