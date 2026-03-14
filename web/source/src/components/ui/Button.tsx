import React from 'react';
import { FiLoader } from 'react-icons/fi';
import './Button.css';

interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'primary' | 'secondary' | 'danger' | 'ghost';
  size?: 'sm' | 'md' | 'lg';
  icon?: React.ReactNode;
  loading?: boolean;
}

export const Button: React.FC<ButtonProps> = ({
  children,
  className = '',
  variant = 'primary',
  size = 'md',
  icon,
  loading = false,
  disabled,
  ...props
}) => {
  const baseClass = 'btn';
  const variantClass = `btn-${variant}`;
  const sizeClass = `btn-${size}`;
  const combinedClassName = `${baseClass} ${variantClass} ${sizeClass} ${className}`.trim();

  return (
    <button
      className={combinedClassName}
      disabled={disabled || loading}
      {...props}
    >
      {loading ? <FiLoader className="animate-spin mr-2" /> : (icon && <span className="btn-icon">{icon}</span>)}
      {children}
    </button>
  );
};
