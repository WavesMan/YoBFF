import React, { useState, useRef, useEffect } from 'react';
import './DropdownMenu.css';

interface DropdownItem {
  label: string;
  onClick: () => void;
  icon?: React.ReactNode;
  danger?: boolean;
}

interface DropdownMenuProps {
  trigger: React.ReactNode;
  items: DropdownItem[];
  align?: 'left' | 'right';
}

export const DropdownMenu: React.FC<DropdownMenuProps> = ({
  trigger,
  items,
  align = 'right',
}) => {
  const [isOpen, setIsOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    };

    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside);
    }

    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
    };
  }, [isOpen]);

  const handleToggle = () => {
    setIsOpen(!isOpen);
  };

  return (
    <div className="dropdown" ref={dropdownRef}>
      <div className="dropdown-trigger" onClick={handleToggle}>
        {trigger}
      </div>
      <div 
        className={`dropdown-menu ${isOpen ? 'open' : ''}`}
        style={{ [align]: 0, right: align === 'right' ? 0 : 'auto', left: align === 'left' ? 0 : 'auto' }}
      >
        {items.map((item, index) => (
          <button
            key={index}
            className={`dropdown-item ${item.danger ? 'danger' : ''}`}
            onClick={() => {
              item.onClick();
              setIsOpen(false);
            }}
          >
            {item.icon && <span className="dropdown-icon">{item.icon}</span>}
            {item.label}
          </button>
        ))}
      </div>
    </div>
  );
};
