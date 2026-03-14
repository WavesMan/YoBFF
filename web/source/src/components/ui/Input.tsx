import React from 'react';
import './Input.css';

interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  label?: string;
  error?: string;
  icon?: React.ReactNode;
  layout?: 'vertical' | 'horizontal';
}

export const Input: React.FC<InputProps> = ({
  label,
  error,
  icon,
  className = '',
  layout = 'vertical',
  id,
  ...props
}) => {
  const generatedId = React.useId();
  const inputId = id || props.name || generatedId;
  
  return (
    <div className={`input-wrapper ${layout} ${className}`}>
      {label && (
        <label htmlFor={inputId} className="input-label">
          {label}
        </label>
      )}
      <div className="input-container">
        {icon && <span className="input-icon">{icon}</span>}
        <input
          id={inputId}
          className={`input-field ${icon ? 'has-icon' : ''} ${error ? 'has-error' : ''}`}
          {...props}
        />
      </div>
      {error && <span className="input-error">{error}</span>}
    </div>
  );
};
